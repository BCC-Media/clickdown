package clickup

import "encoding/json"

type User struct {
	ID       json.Number `json:"id"`
	Username string      `json:"username"`
}

type meResponse struct {
	User User `json:"user"`
}

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type teamsResponse struct {
	Teams []Team `json:"teams"`
}

type Status struct {
	Status     string      `json:"status"`
	Color      string      `json:"color"`
	Type       string      `json:"type"`
	Orderindex json.Number `json:"orderindex"`
}

type Tag struct {
	Name string `json:"name"`
}

type Priority struct {
	ID         json.Number `json:"id"`
	Priority   string      `json:"priority"`
	Color      string      `json:"color"`
	Orderindex json.Number `json:"orderindex"`
}

type ListRef struct {
	ID string `json:"id"`
}

type Task struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TextContent string    `json:"text_content"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	Priority    *Priority `json:"priority"`
	Tags        []Tag     `json:"tags"`
	DateUpdated string    `json:"date_updated"`
	DueDate     string    `json:"due_date"`
	TeamID      string    `json:"team_id"`
	List        ListRef   `json:"list"`
}

type filteredTasksResponse struct {
	Tasks    []Task `json:"tasks"`
	LastPage bool   `json:"last_page"`
}

type listResponse struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Statuses []Status `json:"statuses"`
	TeamID   string   `json:"team_id"`
}

// List is a minimal representation of a ClickUp list, enough to back the
// create-task modal's list dropdown.
type List struct {
	ID     string
	Name   string
	TeamID string
}

type taskUpdateBody struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type taskCreateBody struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	Assignees   []int64  `json:"assignees,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type Comment struct {
	ID          string            `json:"id"`
	CommentText string            `json:"comment_text"`
	// Comment is the heterogeneous block array — text spans, @mentions,
	// images, emoticons, and anything else ClickUp adds later. We pass each
	// block through as raw JSON so unknown fields (e.g. text-formatting
	// attributes) survive round-trips to the DB and the web client.
	Comment     []json.RawMessage `json:"comment"`
	User        CommentUser       `json:"user"`
	Date        string            `json:"date"`
	ReplyCount  json.Number       `json:"reply_count"`
}

type CommentUser struct {
	ID       json.Number `json:"id"`
	Username string      `json:"username"`
}

type commentsResponse struct {
	Comments []Comment `json:"comments"`
}

type postCommentBody struct {
	CommentText string `json:"comment_text"`
	NotifyAll   bool   `json:"notify_all"`
}

type PostCommentResponse struct {
	ID   json.Number `json:"id"`
	Date json.Number `json:"date"`
}

type APIError struct {
	Status  int    `json:"-"`
	Err     string `json:"err"`
	ECode   string `json:"ECODE"`
	Message string `json:"-"`
}

func (e *APIError) Error() string {
	if e.Err != "" {
		return e.Err
	}
	return e.Message
}
