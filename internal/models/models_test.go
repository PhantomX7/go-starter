package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/PhantomX7/athleton/internal/dto"
	"github.com/PhantomX7/athleton/internal/models"
)

func uintPtr(v uint) *uint {
	return &v
}

func TestUserRoleToString(t *testing.T) {
	require.Equal(t, "user", models.UserRoleUser.ToString())
	require.Equal(t, "admin", models.UserRoleAdmin.ToString())
	require.Equal(t, "root", models.UserRoleRoot.ToString())
}

func TestUserRoleIsAdminType(t *testing.T) {
	require.False(t, models.UserRoleUser.IsAdminType())
	require.True(t, models.UserRoleAdmin.IsAdminType())
	require.True(t, models.UserRoleRoot.IsAdminType())
	require.False(t, models.UserRole("other").IsAdminType())
}

func TestUserToResponseMapsAllFields(t *testing.T) {
	createdAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	user := models.User{
		ID:           7,
		Username:     "kenichi",
		Name:         "Ken Ichi",
		BusinessName: "Aimo",
		Email:        "ken@example.com",
		Phone:        "08123456789",
		IsActive:     true,
		Role:         models.UserRoleAdmin,
		AdminRoleID:  uintPtr(3),
		Password:     "secret-hash",
		Timestamp:    models.Timestamp{CreatedAt: createdAt},
	}

	got := user.ToResponse()

	require.NotNil(t, got)
	require.Equal(t, uint(7), got.ID)
	require.Equal(t, "kenichi", got.Username)
	require.Equal(t, "Ken Ichi", got.Name)
	require.Equal(t, "Aimo", got.BusinessName)
	require.Equal(t, "ken@example.com", got.Email)
	require.Equal(t, "08123456789", got.Phone)
	require.True(t, got.IsActive)
	require.Equal(t, "admin", got.Role)
	require.NotNil(t, got.AdminRoleID)
	require.Equal(t, uint(3), *got.AdminRoleID)
	require.Equal(t, createdAt, got.CreatedAt)
	require.Nil(t, got.AdminRole)
}

func TestUserToResponseNilAdminRoleFields(t *testing.T) {
	user := models.User{ID: 1, Role: models.UserRoleUser}

	got := user.ToResponse()

	require.NotNil(t, got)
	require.Nil(t, got.AdminRoleID)
	require.Nil(t, got.AdminRole)
	require.Equal(t, "user", got.Role)
	require.False(t, got.IsActive)
}

func TestUserToResponseIncludesAdminRole(t *testing.T) {
	user := models.User{
		ID:          2,
		Role:        models.UserRoleAdmin,
		AdminRoleID: uintPtr(9),
		AdminRole: &models.AdminRole{
			ID:          9,
			Name:        "editor",
			Permissions: []string{"post:read", "post:write"},
		},
	}

	got := user.ToResponse()

	require.NotNil(t, got.AdminRole)
	require.Equal(t, uint(9), got.AdminRole.ID)
	require.Equal(t, "editor", got.AdminRole.Name)
	require.Equal(t, []string{"post:read", "post:write"}, got.AdminRole.Permissions)
}

func TestAdminRoleToResponseMapsAllFields(t *testing.T) {
	createdAt := time.Date(2025, 2, 3, 4, 5, 6, 0, time.UTC)
	updatedAt := time.Date(2025, 2, 4, 4, 5, 6, 0, time.UTC)
	role := models.AdminRole{
		ID:          11,
		Name:        "moderator",
		Description: "moderates content",
		IsActive:    true,
		Permissions: []string{"log:read"},
		Timestamp:   models.Timestamp{CreatedAt: createdAt, UpdatedAt: updatedAt},
	}

	got := role.ToResponse()

	require.NotNil(t, got)
	require.Equal(t, uint(11), got.ID)
	require.Equal(t, "moderator", got.Name)
	require.Equal(t, "moderates content", got.Description)
	require.True(t, got.IsActive)
	require.Equal(t, []string{"log:read"}, got.Permissions)
	require.Equal(t, createdAt, got.CreatedAt)
	require.Equal(t, updatedAt, got.UpdatedAt)
}

func TestAdminRoleToResponseNilPermissions(t *testing.T) {
	role := models.AdminRole{ID: 12, Name: "empty"}

	got := role.ToResponse()

	require.NotNil(t, got)
	require.Nil(t, got.Permissions)
	require.False(t, got.IsActive)
}

func TestPostToResponseMapsAllFields(t *testing.T) {
	createdAt := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	updatedAt := time.Date(2025, 3, 5, 5, 6, 7, 0, time.UTC)
	post := models.Post{
		Model: gorm.Model{
			ID:        21,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:        "Hello",
		Description: "World",
		IsActive:    true,
	}

	got := post.ToResponse()

	resp, ok := got.(dto.PostResponse)
	require.True(t, ok)
	require.Equal(t, uint(21), resp.ID)
	require.Equal(t, "Hello", resp.Name)
	require.Equal(t, "World", resp.Description)
	require.Equal(t, createdAt, resp.CreatedAt)
	require.Equal(t, updatedAt, resp.UpdatedAt)
}

func TestConfigKeyToString(t *testing.T) {
	require.Equal(t, "site_name", models.ConfigKey("site_name").ToString())
}

func TestConfigToResponseMapsAllFields(t *testing.T) {
	cfg := models.Config{
		Model: gorm.Model{ID: 31},
		Key:   "site_name",
		Value: "Athleton",
	}

	got := cfg.ToResponse()

	require.NotNil(t, got)
	require.Equal(t, uint(31), got.ID)
	require.Equal(t, "site_name", got.Key)
	require.Equal(t, "Athleton", got.Value)
}

func TestLogActionToString(t *testing.T) {
	require.Equal(t, "create", models.LogActionCreate.ToString())
	require.Equal(t, "change_password", models.LogActionChangePassword.ToString())
}

func TestLogToResponseMapsAllFields(t *testing.T) {
	createdAt := time.Date(2025, 4, 5, 6, 7, 8, 0, time.UTC)
	log := models.Log{
		ID:         41,
		UserID:     uintPtr(7),
		Action:     models.LogActionUpdate,
		EntityType: models.LogEntityTypeConfig,
		EntityID:   31,
		Message:    "updated config",
		Timestamp:  models.Timestamp{CreatedAt: createdAt},
	}

	got := log.ToResponse()

	require.Equal(t, uint(41), got.ID)
	require.NotNil(t, got.UserID)
	require.Equal(t, uint(7), *got.UserID)
	require.Equal(t, "update", got.Action)
	require.Equal(t, "config", got.EntityType)
	require.Equal(t, uint(31), got.EntityID)
	require.Equal(t, "updated config", got.Message)
	require.Equal(t, createdAt, got.CreatedAt)
}

func TestLogToResponseNilRelationships(t *testing.T) {
	log := models.Log{ID: 42, Action: models.LogActionDelete}

	got := log.ToResponse()

	require.Nil(t, got.UserID)
	require.Nil(t, got.User)
	require.Nil(t, got.AdminRole)
	require.Nil(t, got.Config)
	require.Nil(t, got.TargetUser)
}

func TestLogToResponseIncludesRelationships(t *testing.T) {
	log := models.Log{
		ID:         43,
		UserID:     uintPtr(7),
		Action:     models.LogActionCreate,
		EntityType: models.LogEntityTypeUser,
		EntityID:   8,
		User:       &models.User{ID: 7, Username: "actor", Role: models.UserRoleAdmin},
		AdminRole:  &models.AdminRole{ID: 9, Name: "editor"},
		Config:     &models.Config{Model: gorm.Model{ID: 31}, Key: "k", Value: "v"},
		TargetUser: &models.User{ID: 8, Username: "target", Role: models.UserRoleUser},
	}

	got := log.ToResponse()

	require.NotNil(t, got.User)
	require.Equal(t, uint(7), got.User.ID)
	require.Equal(t, "actor", got.User.Username)

	require.NotNil(t, got.AdminRole)
	require.Equal(t, uint(9), got.AdminRole.ID)
	require.Equal(t, "editor", got.AdminRole.Name)

	require.NotNil(t, got.Config)
	require.Equal(t, uint(31), got.Config.ID)
	require.Equal(t, "k", got.Config.Key)
	require.Equal(t, "v", got.Config.Value)

	require.NotNil(t, got.TargetUser)
	require.Equal(t, uint(8), got.TargetUser.ID)
	require.Equal(t, "target", got.TargetUser.Username)
}
