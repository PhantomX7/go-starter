package dto

import "time"

type LogResponse struct {
	ID         uint      `json:"id"`
	UserID     *uint     `json:"user_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   uint      `json:"entity_id"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`

	// Relationships
	User *UserResponse `json:"user,omitempty"`

	// Polymorphic Relations.
	// Since the consumer might need specific fields, we can either:
	// 1. Return a generic map[string]interface{} for the entity
	// 2. Return specific DTOs if they exist.

	AdminRole  *AdminRoleResponse `json:"admin_role,omitempty"`
	Config     *ConfigResponse    `json:"config,omitempty"`
	TargetUser *UserResponse      `json:"target_user,omitempty"`
}
