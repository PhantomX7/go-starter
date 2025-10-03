package dto

import (
	"time"

	"github.com/PhantomX7/go-starter/pkg/pagination"
)

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

func NewPostPagination(conditions map[string][]string) *pagination.Pagination{
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("title", pagination.FilterConfig{
			Field: "title",
			Type:  pagination.FilterTypeString,
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDate,
		}).
		AddSort("id", pagination.SortConfig{Field: "id", Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Field: "created_at", Allowed: true})
		
	return pagination.NewPagination(conditions, filterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit: 300,
		DefaultOrder: "id asc",
	})
}

