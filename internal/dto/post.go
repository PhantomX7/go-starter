package dto

import (
	"time"
)

// PostCreateRequest defines the structure for creating a new post
type PostCreateRequest struct {
	Name        string `json:"name" form:"name" binding:"required"`
	Description string `json:"description" form:"description"`
}

// PostUpdateRequest defines the structure for updating a post
type PostUpdateRequest struct {
	Name        string `json:"name" form:"name"`
	Description string `json:"description" form:"description"`
}

// PostResponse defines the structure for post response
type PostResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
