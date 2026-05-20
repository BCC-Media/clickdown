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
}

type filteredTasksResponse struct {
	Tasks    []Task `json:"tasks"`
	LastPage bool   `json:"last_page"`
}

type taskUpdateBody struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
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
