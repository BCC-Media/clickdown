package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultBaseURL = "https://api.clickup.com/api/v2"

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(token string) *Client {
	return &Client{
		BaseURL: defaultBaseURL,
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		ae := &APIError{Status: resp.StatusCode, Message: string(raw)}
		_ = json.Unmarshal(raw, ae)
		return nil, fmt.Errorf("clickup %s %s: %d %s", method, path, resp.StatusCode, ae.Error())
	}
	return resp, nil
}

func decode[T any](resp *http.Response) (T, error) {
	var v T
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(&v)
	return v, err
}

func (c *Client) Me(ctx context.Context) (User, error) {
	resp, err := c.do(ctx, http.MethodGet, "/user", nil, nil)
	if err != nil {
		return User{}, err
	}
	out, err := decode[meResponse](resp)
	if err != nil {
		return User{}, err
	}
	return out.User, nil
}

func (c *Client) Teams(ctx context.Context) ([]Team, error) {
	resp, err := c.do(ctx, http.MethodGet, "/team", nil, nil)
	if err != nil {
		return nil, err
	}
	out, err := decode[teamsResponse](resp)
	if err != nil {
		return nil, err
	}
	return out.Teams, nil
}

// TasksAssignedToMe pages through all tasks in the given team that are
// assigned to userID and are not closed.
func (c *Client) TasksAssignedToMe(ctx context.Context, teamID, userID string) ([]Task, error) {
	var all []Task
	for page := 0; ; page++ {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("include_closed", "false")
		q.Set("subtasks", "true")
		q.Add("assignees[]", userID)
		path := "/team/" + teamID + "/task"
		resp, err := c.do(ctx, http.MethodGet, path, q, nil)
		if err != nil {
			return nil, err
		}
		out, err := decode[filteredTasksResponse](resp)
		if err != nil {
			return nil, err
		}
		all = append(all, out.Tasks...)
		if out.LastPage || len(out.Tasks) == 0 {
			break
		}
		if page > 200 {
			return nil, fmt.Errorf("clickup pagination runaway (>200 pages)")
		}
	}
	return all, nil
}

type UpdateTaskRequest struct {
	Title       *string
	Description *string
	Status      *string
}

// ListStatuses fetches the status schema for the given list from ClickUp.
// Returns every status defined on the list, including terminal ones that no
// task may currently be assigned to.
func (c *Client) ListStatuses(ctx context.Context, listID string) ([]Status, error) {
	resp, err := c.do(ctx, http.MethodGet, "/list/"+listID, nil, nil)
	if err != nil {
		return nil, err
	}
	out, err := decode[listResponse](resp)
	if err != nil {
		return nil, err
	}
	return out.Statuses, nil
}

func (c *Client) UpdateTask(ctx context.Context, taskID string, u UpdateTaskRequest) (Task, error) {
	body := taskUpdateBody{Name: u.Title, Description: u.Description, Status: u.Status}
	resp, err := c.do(ctx, http.MethodPut, "/task/"+taskID, nil, body)
	if err != nil {
		return Task{}, err
	}
	return decode[Task](resp)
}

// TaskComments fetches the most recent page of comments for a task (ClickUp
// returns up to 25 newest-first; pagination via the `start` cursor is not
// implemented here).
func (c *Client) TaskComments(ctx context.Context, taskID string) ([]Comment, error) {
	resp, err := c.do(ctx, http.MethodGet, "/task/"+taskID+"/comment", nil, nil)
	if err != nil {
		return nil, err
	}
	out, err := decode[commentsResponse](resp)
	if err != nil {
		return nil, err
	}
	return out.Comments, nil
}

// PostComment creates a top-level comment on the task. notify_all is forced
// false to avoid surprising downstream notifications during triage.
func (c *Client) PostComment(ctx context.Context, taskID, text string) (PostCommentResponse, error) {
	resp, err := c.do(ctx, http.MethodPost, "/task/"+taskID+"/comment", nil, postCommentBody{CommentText: text})
	if err != nil {
		return PostCommentResponse{}, err
	}
	return decode[PostCommentResponse](resp)
}

