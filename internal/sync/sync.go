package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/bcc-media/clickdown/internal/clickup"
	"github.com/bcc-media/clickdown/internal/db"
	"github.com/bcc-media/clickdown/internal/db/gen"
)

const (
	settingUserID   = "clickup_user_id"
	settingUsername = "clickup_username"
	settingTeamIDs  = "clickup_team_ids"
	settingLastAt   = "last_sync_at"
)

type Status struct {
	Running      bool   `json:"running"`
	LastSyncAt   int64  `json:"last_sync_at"`
	LastError    string `json:"last_error"`
	LastDuration int64  `json:"last_duration_ms"`
}

type Worker struct {
	Store   *db.Store
	Client  *clickup.Client
	trigger chan struct{}

	mu       sync.Mutex
	status   Status
	interval time.Duration

	stopCh chan struct{}
}

func NewWorker(s *db.Store, c *clickup.Client, interval time.Duration) *Worker {
	return &Worker{
		Store:    s,
		Client:   c,
		trigger:  make(chan struct{}, 1),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (w *Worker) SetInterval(d time.Duration) {
	w.mu.Lock()
	w.interval = d
	w.mu.Unlock()
}

func (w *Worker) Interval() time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.interval
}

func (w *Worker) Status() Status {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

// Trigger requests a sync; if one is already queued, this is a no-op.
func (w *Worker) Trigger() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *Worker) Run(ctx context.Context) {
	w.Trigger() // initial sync on start
	timer := time.NewTimer(w.Interval())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.trigger:
			w.runOnce(ctx)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(w.Interval())
		case <-timer.C:
			w.runOnce(ctx)
			timer.Reset(w.Interval())
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	w.mu.Lock()
	w.status.Running = true
	w.status.LastError = ""
	w.mu.Unlock()

	start := time.Now()
	err := w.sync(ctx)
	end := time.Now()

	w.mu.Lock()
	w.status.Running = false
	w.status.LastSyncAt = end.UnixMilli()
	w.status.LastDuration = end.Sub(start).Milliseconds()
	if err != nil {
		w.status.LastError = err.Error()
	}
	w.mu.Unlock()

	_ = w.Store.Q.SetSetting(ctx, gen.SetSettingParams{Key: settingLastAt, Value: strconv.FormatInt(end.UnixMilli(), 10)})
}

func (w *Worker) sync(ctx context.Context) error {
	if w.Client.Token == "" {
		return errors.New("CLICKUP_API_TOKEN is not configured")
	}
	userID, err := w.ensureUserID(ctx)
	if err != nil {
		return fmt.Errorf("user id: %w", err)
	}
	teamIDs, err := w.ensureTeamIDs(ctx)
	if err != nil {
		return fmt.Errorf("team ids: %w", err)
	}
	if len(teamIDs) == 0 {
		return errors.New("no ClickUp teams available for this token")
	}

	var remote []clickup.Task
	for _, tid := range teamIDs {
		tasks, err := w.Client.TasksAssignedToMe(ctx, tid, userID)
		if err != nil {
			return fmt.Errorf("team %s tasks: %w", tid, err)
		}
		for i := range tasks {
			if tasks[i].TeamID == "" {
				tasks[i].TeamID = tid
			}
		}
		remote = append(remote, tasks...)
	}

	if err := w.reconcile(ctx, remote); err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}
	if err := w.pushDirty(ctx); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	if err := w.pushDirtyComments(ctx); err != nil {
		return fmt.Errorf("push comments: %w", err)
	}
	return nil
}

func (w *Worker) ensureUserID(ctx context.Context) (string, error) {
	v, err := w.Store.Q.GetSetting(ctx, settingUserID)
	if err == nil && v != "" {
		return v, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	u, err := w.Client.Me(ctx)
	if err != nil {
		return "", err
	}
	id := u.ID.String()
	if err := w.Store.Q.SetSetting(ctx, gen.SetSettingParams{Key: settingUserID, Value: id}); err != nil {
		return "", err
	}
	if u.Username != "" {
		if err := w.Store.Q.SetSetting(ctx, gen.SetSettingParams{Key: settingUsername, Value: u.Username}); err != nil {
			return "", err
		}
	}
	return id, nil
}

func (w *Worker) ensureTeamIDs(ctx context.Context) ([]string, error) {
	v, err := w.Store.Q.GetSetting(ctx, settingTeamIDs)
	if err == nil && v != "" {
		var ids []string
		if jerr := json.Unmarshal([]byte(v), &ids); jerr == nil && len(ids) > 0 {
			return ids, nil
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	teams, err := w.Client.Teams(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(teams))
	for _, t := range teams {
		ids = append(ids, t.ID)
	}
	raw, _ := json.Marshal(ids)
	if err := w.Store.Q.SetSetting(ctx, gen.SetSettingParams{Key: settingTeamIDs, Value: string(raw)}); err != nil {
		return nil, err
	}
	return ids, nil
}

func (w *Worker) reconcile(ctx context.Context, remote []clickup.Task) error {
	// Fetch each unique list's full status schema before opening the tx so
	// HTTP latency doesn't hold a SQLite write lock. The list endpoint is the
	// authoritative source for statuses — task-embedded statuses only surface
	// states that some currently-open task is assigned to, so terminal states
	// like "Closed" would otherwise never reach the local DB.
	listStatuses := make(map[string][]clickup.Status)
	for _, t := range remote {
		if t.List.ID == "" {
			continue
		}
		if _, ok := listStatuses[t.List.ID]; ok {
			continue
		}
		sts, err := w.Client.ListStatuses(ctx, t.List.ID)
		if err != nil {
			log.Printf("clickdown: fetch list %s statuses: %v", t.List.ID, err)
			continue
		}
		listStatuses[t.List.ID] = sts
	}

	tx, err := w.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := w.Store.Q.WithTx(tx)
	now := time.Now().UnixMilli()

	// Upsert statuses. List-schema statuses first (authoritative), then
	// task-embedded ones as a fallback for any task without a list_id.
	seen := map[string]bool{}
	upsertStatus := func(s clickup.Status) error {
		if s.Status == "" || seen[s.Status] {
			return nil
		}
		seen[s.Status] = true
		order, _ := s.Orderindex.Int64()
		return q.UpsertStatus(ctx, gen.UpsertStatusParams{
			Name:       s.Status,
			Color:      s.Color,
			Type:       s.Type,
			Orderindex: order,
		})
	}
	for _, sts := range listStatuses {
		for _, s := range sts {
			if err := upsertStatus(s); err != nil {
				return err
			}
		}
	}
	for _, t := range remote {
		if err := upsertStatus(t.Status); err != nil {
			return err
		}
	}

	keepIDs := make([]string, 0, len(remote))
	for _, rt := range remote {
		keepIDs = append(keepIDs, rt.ID)
		if err := w.upsertTask(ctx, q, rt, now); err != nil {
			return fmt.Errorf("upsert task %s: %w", rt.ID, err)
		}
	}

	if len(keepIDs) > 0 {
		if err := q.SoftDeleteMissingTasks(ctx, gen.SoftDeleteMissingTasksParams{
			DeletedAt: &now,
			Keep:      keepIDs,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (w *Worker) upsertTask(ctx context.Context, q *gen.Queries, rt clickup.Task, now int64) error {
	updated := parseDateMillis(rt.DateUpdated)
	priority := priorityID(rt.Priority)
	teamID := nullableString(rt.TeamID)

	existing, err := q.GetTaskByClickupID(ctx, rt.ID)
	if errors.Is(err, sql.ErrNoRows) {
		t, err := q.InsertTask(ctx, gen.InsertTaskParams{
			ClickupID:        rt.ID,
			Title:            rt.Name,
			Description:      descriptionText(rt),
			Status:           rt.Status.Status,
			Priority:         priority,
			TeamID:           teamID,
			ClickupUpdatedAt: updated,
			LocalUpdatedAt:   now,
		})
		if err != nil {
			return err
		}
		return w.replaceClickupTags(ctx, q, t.ID, rt.Tags)
	}
	if err != nil {
		return err
	}

	dirty, derr := q.GetTaskDirty(ctx, existing.ID)
	hasDirty := derr == nil
	if derr != nil && !errors.Is(derr, sql.ErrNoRows) {
		return derr
	}

	title := existing.Title
	desc := existing.Description
	status := existing.Status
	if !hasDirty || dirty.Title == 0 {
		title = rt.Name
	}
	if !hasDirty || dirty.Description == 0 {
		desc = descriptionText(rt)
	}
	if !hasDirty || dirty.Status == 0 {
		status = rt.Status.Status
	}

	if err := q.UpdateTaskFromRemote(ctx, gen.UpdateTaskFromRemoteParams{
		Title:            title,
		Description:      desc,
		Status:           status,
		Priority:         priority,
		TeamID:           teamID,
		ClickupUpdatedAt: updated,
		ID:               existing.ID,
	}); err != nil {
		return err
	}

	return w.replaceClickupTags(ctx, q, existing.ID, rt.Tags)
}

// replaceClickupTags reconciles ClickUp-origin tags on a task while preserving
// local-only tags already attached.
func (w *Worker) replaceClickupTags(ctx context.Context, q *gen.Queries, taskID int64, remote []clickup.Tag) error {
	existing, err := q.ListTagsForTask(ctx, taskID)
	if err != nil {
		return err
	}
	// Build set of local-only tags to preserve.
	preserve := make(map[string]bool)
	for _, e := range existing {
		if e.Origin == "local" {
			preserve[e.Name] = true
		}
	}
	if err := q.ClearTaskTags(ctx, taskID); err != nil {
		return err
	}
	// Reattach local tags.
	for name := range preserve {
		tag, err := q.InsertTag(ctx, gen.InsertTagParams{Name: name, Origin: "local"})
		if err != nil {
			return err
		}
		if err := q.AttachTaskTag(ctx, gen.AttachTaskTagParams{TaskID: taskID, TagID: tag.ID}); err != nil {
			return err
		}
	}
	// Attach ClickUp tags.
	for _, rt := range remote {
		if rt.Name == "" {
			continue
		}
		tag, err := q.InsertTag(ctx, gen.InsertTagParams{Name: rt.Name, Origin: "clickup"})
		if err != nil {
			return err
		}
		if err := q.AttachTaskTag(ctx, gen.AttachTaskTagParams{TaskID: taskID, TagID: tag.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) pushDirty(ctx context.Context) error {
	dirty, err := w.Store.Q.ListTaskDirty(ctx)
	if err != nil {
		return err
	}
	for _, d := range dirty {
		t, err := w.Store.Q.GetTask(ctx, d.TaskID)
		if err != nil {
			continue
		}
		req := clickup.UpdateTaskRequest{}
		if d.Title != 0 {
			req.Title = &t.Title
		}
		if d.Description != 0 {
			req.Description = &t.Description
		}
		if d.Status != 0 {
			req.Status = &t.Status
		}
		updated, err := w.Client.UpdateTask(ctx, t.ClickupID, req)
		if err != nil {
			return fmt.Errorf("push task %s: %w", t.ClickupID, err)
		}
		now := time.Now().UnixMilli()
		remoteUpdated := parseDateMillis(updated.DateUpdated)
		if err := w.Store.Q.MarkTaskPushed(ctx, gen.MarkTaskPushedParams{
			LastPushedAt:     &now,
			ClickupUpdatedAt: remoteUpdated,
			ID:               t.ID,
		}); err != nil {
			return err
		}
		if err := w.Store.Q.ClearTaskDirty(ctx, t.ID); err != nil {
			return err
		}
	}
	return nil
}

func parseDateMillis(s string) *int64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func priorityID(p *clickup.Priority) *int64 {
	if p == nil {
		return nil
	}
	v, err := p.ID.Int64()
	if err != nil {
		return nil
	}
	return &v
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func descriptionText(t clickup.Task) string {
	if t.TextContent != "" {
		return t.TextContent
	}
	return t.Description
}

// PullCommentsForTask fetches the latest page of comments for the given local
// task ID from ClickUp and reconciles them into the local DB. Locally-queued
// comments (clickup_id IS NULL) are left untouched so they survive a refresh.
// Threaded replies are pulled for any top-level comment whose reply_count > 0.
func (w *Worker) PullCommentsForTask(ctx context.Context, taskID int64) error {
	t, err := w.Store.Q.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	remote, err := w.Client.TaskComments(ctx, t.ClickupID)
	if err != nil {
		return err
	}

	type replyBatch struct {
		parentID string
		replies  []clickup.Comment
	}
	var replies []replyBatch
	for _, rc := range remote {
		n, _ := rc.ReplyCount.Int64()
		if rc.ID == "" || n <= 0 {
			continue
		}
		rs, err := w.Client.CommentReplies(ctx, rc.ID)
		if err != nil {
			// Best-effort: log and continue; the parent thread still renders.
			log.Printf("clickdown: pull replies for comment %s: %v", rc.ID, err)
			continue
		}
		replies = append(replies, replyBatch{parentID: rc.ID, replies: rs})
	}

	tx, err := w.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := w.Store.Q.WithTx(tx)
	now := time.Now().UnixMilli()

	keep := make([]*string, 0, len(remote))
	upsert := func(rc clickup.Comment, parent *string) error {
		cid := rc.ID
		keep = append(keep, &cid)
		date := parseDateMillis(rc.Date)
		authorID := rc.User.ID.String()
		_, gerr := q.GetCommentByClickupID(ctx, &cid)
		if errors.Is(gerr, sql.ErrNoRows) {
			_, err := q.InsertRemoteComment(ctx, gen.InsertRemoteCommentParams{
				ClickupID:       &cid,
				TaskID:          t.ID,
				AuthorID:        authorID,
				AuthorUsername:  rc.User.Username,
				Text:            rc.CommentText,
				ClickupDate:     date,
				LocalCreatedAt:  now,
				ParentClickupID: parent,
			})
			return err
		}
		if gerr != nil {
			return gerr
		}
		return q.UpdateRemoteComment(ctx, gen.UpdateRemoteCommentParams{
			AuthorID:       authorID,
			AuthorUsername: rc.User.Username,
			Text:           rc.CommentText,
			ClickupDate:    date,
			ClickupID:      &cid,
		})
	}

	for _, rc := range remote {
		if rc.ID == "" {
			continue
		}
		if err := upsert(rc, nil); err != nil {
			return err
		}
	}
	for _, batch := range replies {
		parent := batch.parentID
		for _, rc := range batch.replies {
			if rc.ID == "" {
				continue
			}
			if err := upsert(rc, &parent); err != nil {
				return err
			}
		}
	}

	if err := q.SoftDeleteMissingCommentsForTask(ctx, gen.SoftDeleteMissingCommentsForTaskParams{
		DeletedAt: &now,
		TaskID:    t.ID,
		Keep:      keep,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (w *Worker) pushDirtyComments(ctx context.Context) error {
	dirty, err := w.Store.Q.ListCommentsDirty(ctx)
	if err != nil {
		return err
	}
	for _, d := range dirty {
		t, err := w.Store.Q.GetTask(ctx, d.TaskID)
		if err != nil {
			continue
		}
		var resp clickup.PostCommentResponse
		if d.ParentClickupID != nil {
			resp, err = w.Client.PostReply(ctx, *d.ParentClickupID, d.Text)
			if err != nil {
				return fmt.Errorf("post reply parent %s: %w", *d.ParentClickupID, err)
			}
		} else {
			resp, err = w.Client.PostComment(ctx, t.ClickupID, d.Text)
			if err != nil {
				return fmt.Errorf("post comment task %s: %w", t.ClickupID, err)
			}
		}
		cid := resp.ID.String()
		date := parseDateMillis(resp.Date.String())
		if err := w.Store.Q.SetCommentClickupID(ctx, gen.SetCommentClickupIDParams{
			ClickupID:   &cid,
			ClickupDate: date,
			ID:          d.CommentID,
		}); err != nil {
			return err
		}
		if err := w.Store.Q.ClearCommentDirty(ctx, d.CommentID); err != nil {
			return err
		}
	}
	return nil
}
