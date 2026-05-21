package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bcc-media/clickdown/internal/db/gen"
)

type tagDTO struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
}

type taskDTO struct {
	ID        int64    `json:"id"`
	ClickupID string   `json:"clickup_id"`
	Title     string   `json:"title"`
	Desc      string   `json:"desc"`
	Status    string   `json:"status"`
	Priority  *int64   `json:"priority"`
	Tags      []tagDTO `json:"tags"`
	UpdatedAt int64    `json:"updated_at"`
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tasks, err := s.Store.Q.ListTasks(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tagRows, err := s.Store.Q.ListTasksWithTags(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tagsByTask := map[int64][]tagDTO{}
	for _, tr := range tagRows {
		tagsByTask[tr.TaskID] = append(tagsByTask[tr.TaskID], tagDTO{Name: tr.TagName, Origin: tr.TagOrigin})
	}
	out := make([]taskDTO, 0, len(tasks))
	for _, t := range tasks {
		tags := tagsByTask[t.ID]
		if tags == nil {
			tags = []tagDTO{}
		}
		out = append(out, taskDTO{
			ID:        t.ID,
			ClickupID: t.ClickupID,
			Title:     t.Title,
			Desc:      t.Description,
			Status:    t.Status,
			Priority:  t.Priority,
			Tags:      tags,
			UpdatedAt: t.LocalUpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type patchTaskBody struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

func (s *Server) patchTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var body patchTaskBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	task, err := s.Store.Q.GetTask(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	now := time.Now().UnixMilli()
	dirtyTitle, dirtyDesc, dirtyStatus := int64(0), int64(0), int64(0)
	if body.Title != nil && *body.Title != task.Title {
		if err := s.Store.Q.PatchTaskTitle(ctx, gen.PatchTaskTitleParams{Title: *body.Title, LocalUpdatedAt: now, ID: id}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		dirtyTitle = 1
	}
	if body.Description != nil && *body.Description != task.Description {
		if err := s.Store.Q.PatchTaskDescription(ctx, gen.PatchTaskDescriptionParams{Description: *body.Description, LocalUpdatedAt: now, ID: id}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		dirtyDesc = 1
	}
	if body.Status != nil && *body.Status != task.Status {
		if err := s.Store.Q.PatchTaskStatus(ctx, gen.PatchTaskStatusParams{Status: *body.Status, LocalUpdatedAt: now, ID: id}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		dirtyStatus = 1
	}
	if dirtyTitle|dirtyDesc|dirtyStatus != 0 {
		if err := s.Store.Q.UpsertTaskDirty(ctx, gen.UpsertTaskDirtyParams{
			TaskID:      id,
			Title:       dirtyTitle,
			Description: dirtyDesc,
			Status:      dirtyStatus,
			QueuedAt:    now,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		s.Sync.Trigger()
	}
	s.serveOneTask(ctx, w, id)
}

func (s *Server) putTaskTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var tagNames []string
	if err := json.NewDecoder(r.Body).Decode(&tagNames); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Preserve origin of existing tags; new tags default to 'local'.
	existing, err := s.Store.Q.ListTagsForTask(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	originByName := map[string]string{}
	for _, e := range existing {
		originByName[e.Name] = e.Origin
	}
	tx, err := s.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()
	q := s.Store.Q.WithTx(tx)
	if err := q.ClearTaskTags(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, name := range tagNames {
		if name == "" {
			continue
		}
		origin := originByName[name]
		if origin == "" {
			origin = "local"
		}
		tag, err := q.InsertTag(ctx, gen.InsertTagParams{Name: name, Origin: origin})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if err := q.AttachTaskTag(ctx, gen.AttachTaskTagParams{TaskID: id, TagID: tag.ID}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.serveOneTask(ctx, w, id)
}

func (s *Server) listStatuses(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Q.ListStatuses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) listTags(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Q.ListTags(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) triggerSync(w http.ResponseWriter, r *http.Request) {
	s.Sync.Trigger()
	writeJSON(w, http.StatusAccepted, s.Sync.Status())
}

func (s *Server) syncStatus(w http.ResponseWriter, r *http.Request) {
	st := s.Sync.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"running":          st.Running,
		"last_sync_at":     st.LastSyncAt,
		"last_error":       st.LastError,
		"last_duration_ms": st.LastDuration,
		"interval_seconds": int64(s.Sync.Interval().Seconds()),
	})
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Q.ListSettings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := map[string]string{}
	for _, kv := range rows {
		out[kv.Key] = kv.Value
	}
	// Always include current sync interval (live, even if not yet persisted).
	out["sync_interval_seconds"] = strconv.FormatInt(int64(s.Sync.Interval().Seconds()), 10)
	writeJSON(w, http.StatusOK, out)
}

type patchSettingsBody struct {
	SyncIntervalSeconds *int64  `json:"sync_interval_seconds"`
	Theme               *string `json:"theme"`
	Accent              *string `json:"accent"`
}

func (s *Server) patchSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body patchSettingsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.SyncIntervalSeconds != nil && *body.SyncIntervalSeconds > 0 {
		secs := *body.SyncIntervalSeconds
		if secs < 30 {
			secs = 30
		}
		s.Sync.SetInterval(time.Duration(secs) * time.Second)
		if err := s.Store.Q.SetSetting(ctx, gen.SetSettingParams{Key: "sync_interval_seconds", Value: strconv.FormatInt(secs, 10)}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if body.Theme != nil {
		if err := s.Store.Q.SetSetting(ctx, gen.SetSettingParams{Key: "theme", Value: *body.Theme}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if body.Accent != nil {
		if err := s.Store.Q.SetSetting(ctx, gen.SetSettingParams{Key: "accent", Value: *body.Accent}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	s.getSettings(w, r)
}

type commentDTO struct {
	ID              int64             `json:"id"`
	ClickupID       *string           `json:"clickup_id"`
	TaskID          int64             `json:"task_id"`
	ParentClickupID *string           `json:"parent_clickup_id"`
	Author          string            `json:"author"`
	Text            string            `json:"text"`
	Blocks          []json.RawMessage `json:"blocks,omitempty"`
	CreatedAt       int64             `json:"created_at"`
	Pending         bool              `json:"pending"`
}

func commentToDTO(c gen.Comment) commentDTO {
	created := c.LocalCreatedAt
	if c.ClickupDate != nil {
		created = *c.ClickupDate
	}
	var blocks []json.RawMessage
	if c.BlocksJson != nil && *c.BlocksJson != "" {
		// Pass the stored array through untouched. If it's malformed for any
		// reason, drop the field rather than failing the response.
		_ = json.Unmarshal([]byte(*c.BlocksJson), &blocks)
	}
	return commentDTO{
		ID:              c.ID,
		ClickupID:       c.ClickupID,
		TaskID:          c.TaskID,
		ParentClickupID: c.ParentClickupID,
		Author:          c.AuthorUsername,
		Text:            c.Text,
		Blocks:          blocks,
		CreatedAt:       created,
		Pending:         c.ClickupID == nil,
	}
}

func (s *Server) listTaskComments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if r.URL.Query().Get("refresh") == "1" {
		if err := s.Sync.PullCommentsForTask(ctx, id); err != nil {
			// Best-effort: log and continue with whatever's already in the local DB.
			log.Printf("clickdown: pull comments task %d: %v", id, err)
		}
	}
	rows, err := s.Store.Q.ListCommentsForTask(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]commentDTO, 0, len(rows))
	for _, c := range rows {
		out = append(out, commentToDTO(c))
	}
	writeJSON(w, http.StatusOK, out)
}

type postCommentBody struct {
	Text            string  `json:"text"`
	ParentClickupID *string `json:"parent_clickup_id"`
}

func (s *Server) postTaskComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var body postCommentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, errors.New("text is required"))
		return
	}
	if _, err := s.Store.Q.GetTask(ctx, id); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	authorID, _ := s.Store.Q.GetSetting(ctx, "clickup_user_id")
	authorName, _ := s.Store.Q.GetSetting(ctx, "clickup_username")
	now := time.Now().UnixMilli()
	c, err := s.Store.Q.InsertLocalComment(ctx, gen.InsertLocalCommentParams{
		TaskID:          id,
		AuthorID:        authorID,
		AuthorUsername:  authorName,
		Text:            text,
		LocalCreatedAt:  now,
		ParentClickupID: body.ParentClickupID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.Store.Q.UpsertCommentDirty(ctx, gen.UpsertCommentDirtyParams{
		CommentID: c.ID,
		QueuedAt:  now,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.Sync.Trigger()
	writeJSON(w, http.StatusCreated, commentToDTO(c))
}

func (s *Server) serveOneTask(ctx context.Context, w http.ResponseWriter, id int64) {
	t, err := s.Store.Q.GetTask(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	tagRows, err := s.Store.Q.ListTagsForTask(ctx, t.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tags := make([]tagDTO, 0, len(tagRows))
	for _, tr := range tagRows {
		tags = append(tags, tagDTO{Name: tr.Name, Origin: tr.Origin})
	}
	writeJSON(w, http.StatusOK, taskDTO{
		ID:        t.ID,
		ClickupID: t.ClickupID,
		Title:     t.Title,
		Desc:      t.Description,
		Status:    t.Status,
		Priority:  t.Priority,
		Tags:      tags,
		UpdatedAt: t.LocalUpdatedAt,
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		code = http.StatusNotFound
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
