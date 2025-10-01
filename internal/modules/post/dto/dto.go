package dto

import "time"

type PostCreateRequest struct {
	Title   string `json:"title" form:"title" binding:"required"`
	Content string `json:"content" form:"content" binding:"required"`
}

type PostUpdateRequest struct {
	Title   string `json:"title" form:"title"`
	Content string `json:"content" form:"content"`
}

type PostResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
