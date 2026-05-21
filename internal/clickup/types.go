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
	TeamID      string    `json:"team_id"`
	List        ListRef   `json:"list"`
}

type filteredTasksResponse struct {
	Tasks    []Task `json:"tasks"`
	LastPage bool   `json:"last_page"`
}

type listResponse struct {
	ID       string   `json:"id"`
	Statuses []Status `json:"statuses"`
}

type taskUpdateBody struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type Comment struct {
	ID          string          `json:"id"`
	CommentText string          `json:"comment_text"`
	Comment     []CommentBlock  `json:"comment"`
	User        CommentUser     `json:"user"`
	Date        string          `json:"date"`
	ReplyCount  json.Number     `json:"reply_count"`
}

// CommentBlock is one segment of a ClickUp comment body. ClickUp returns a
// heterogeneous array — plain text spans have Type="" (or absent), mentions
// have Type="tag", inline images have Type="image" with an Image payload.
// Unknown types are still preserved when we round-trip the raw JSON.
type CommentBlock struct {
	Type  string          `json:"type,omitempty"`
	Text  string          `json:"text,omitempty"`
	Image *CommentImage   `json:"image,omitempty"`
	User  *CommentUser    `json:"user,omitempty"`
}

type CommentImage struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	Title           string `json:"title,omitempty"`
	Extension       string `json:"extension,omitempty"`
	URL             string `json:"url,omitempty"`
	ThumbnailSmall  string `json:"thumbnail_small,omitempty"`
	ThumbnailMedium string `json:"thumbnail_medium,omitempty"`
	ThumbnailLarge  string `json:"thumbnail_large,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
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
