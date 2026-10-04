package api

import "time"

type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type registerResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type loginResponse struct {
	Token string `json:"token"`
}

type meResponse struct {
	UserID string `json:"user_id"`
}

type noteRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
