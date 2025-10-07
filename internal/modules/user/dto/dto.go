package dto

import (
	"time"

	"github.com/PhantomX7/go-starter/pkg/pagination"
)

// UserCreateRequest defines the structure for creating a new user
type UserCreateRequest struct {
	Name        string `json:"name" form:"name" binding:"required"`
	Description string `json:"description" form:"description"`
}

// UserUpdateRequest defines the structure for updating a user
type UserUpdateRequest struct {
	Name        string `json:"name" form:"name"`
	Description string `json:"description" form:"description"`
}

// UserResponse defines the structure for user response
type UserResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewUserPagination creates a new pagination instance for users
func NewUserPagination(conditions map[string][]string) *pagination.Pagination {
	filterDefinition := pagination.NewFilterDefinition().
		AddFilter("name", pagination.FilterConfig{
			Field:     "name",
			TableName: "users",
			Type:      pagination.FilterTypeString,
		}).
		AddFilter("created_at", pagination.FilterConfig{
			Field: "created_at",
			Type:  pagination.FilterTypeDate,
		}).
		AddSort("id", pagination.SortConfig{Field: "id", Allowed: true}).
		AddSort("name", pagination.SortConfig{Field: "name", Allowed: true}).
		AddSort("created_at", pagination.SortConfig{Field: "created_at", Allowed: true})

	return pagination.NewPagination(conditions, filterDefinition, pagination.PaginationOptions{
		DefaultLimit: 20,
		MaxLimit:     300,
		DefaultOrder: "id asc",
	})
}
