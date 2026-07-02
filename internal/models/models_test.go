package models_test

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

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

// TestAutoMigrateUniqueIndexes verifies the model index tags migrate cleanly
// and that unique indexes on soft-deleting tables are partial: a soft-deleted
// row must not block reuse of its value, while an active duplicate must fail.
func TestAutoMigrateUniqueIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.Config{},
		&models.Log{},
		&models.AdminRole{},
	))

	newUser := func(username, email string) *models.User {
		return &models.User{
			Username: username,
			Email:    email,
			Phone:    "0",
			Role:     models.UserRoleUser,
			Password: "hash",
		}
	}

	// Active duplicates are rejected.
	require.NoError(t, db.Create(newUser("dup", "dup@example.com")).Error)
	require.Error(t, db.Create(newUser("dup", "other@example.com")).Error, "duplicate active username must fail")
	require.Error(t, db.Create(newUser("other", "dup@example.com")).Error, "duplicate active email must fail")

	// Soft-deleted rows do not block reuse.
	require.NoError(t, db.Where("username = ?", "dup").Delete(&models.User{}).Error)
	require.NoError(t, db.Create(newUser("dup", "dup@example.com")).Error, "soft-deleted username/email must be reusable")

	// admin_roles.name behaves the same way.
	require.NoError(t, db.Create(&models.AdminRole{Name: "editor"}).Error)
	require.Error(t, db.Create(&models.AdminRole{Name: "editor"}).Error, "duplicate active role name must fail")
	require.NoError(t, db.Where("name = ?", "editor").Delete(&models.AdminRole{}).Error)
	require.NoError(t, db.Create(&models.AdminRole{Name: "editor"}).Error, "soft-deleted role name must be reusable")

	// configs.key behaves the same way.
	require.NoError(t, db.Create(&models.Config{Key: "site", Value: "a"}).Error)
	require.Error(t, db.Create(&models.Config{Key: "site", Value: "b"}).Error, "duplicate active config key must fail")
	require.NoError(t, db.Where("key = ?", "site").Delete(&models.Config{}).Error)
	require.NoError(t, db.Create(&models.Config{Key: "site", Value: "c"}).Error, "soft-deleted config key must be reusable")
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
}

func TestLogToResponseIncludesRelationships(t *testing.T) {
	log := models.Log{
		ID:         43,
		UserID:     uintPtr(7),
		Action:     models.LogActionCreate,
		EntityType: models.LogEntityTypeUser,
		EntityID:   8,
		User:       &models.User{ID: 7, Username: "actor", Role: models.UserRoleAdmin},
	}

	got := log.ToResponse()

	require.NotNil(t, got.User)
	require.Equal(t, uint(7), got.User.ID)
	require.Equal(t, "actor", got.User.Username)
}

// TestPolymorphicLogAssociationsMatchWriterEntityTypes verifies that the Logs
// associations on User/AdminRole/Config use the same entity-type discriminator
// the audit writers store (models.LogEntityType*); a mismatch makes every
// preload silently return zero rows.
func TestPolymorphicLogAssociationsMatchWriterEntityTypes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.AdminRole{},
		&models.Config{},
		&models.Log{},
	))

	user := &models.User{Username: "audited", Email: "audited@test.local", Role: models.UserRoleAdmin}
	require.NoError(t, db.Create(user).Error)
	role := &models.AdminRole{Name: "Audited Role"}
	require.NoError(t, db.Create(role).Error)
	cfg := &models.Config{Key: "audited_key", Value: "v"}
	require.NoError(t, db.Create(cfg).Error)

	// Write logs exactly like the audit writers do.
	for _, entry := range []*models.Log{
		{Action: models.LogActionUpdate, EntityType: models.LogEntityTypeUser, EntityID: user.ID},
		{Action: models.LogActionUpdate, EntityType: models.LogEntityTypeAdminRole, EntityID: role.ID},
		{Action: models.LogActionUpdate, EntityType: models.LogEntityTypeConfig, EntityID: cfg.ID},
	} {
		require.NoError(t, db.Create(entry).Error)
	}

	var gotUser models.User
	require.NoError(t, db.Preload("Logs").First(&gotUser, user.ID).Error)
	require.Len(t, gotUser.Logs, 1, "User.Logs preload must find logs written with LogEntityTypeUser")

	var gotRole models.AdminRole
	require.NoError(t, db.Preload("Logs").First(&gotRole, role.ID).Error)
	require.Len(t, gotRole.Logs, 1, "AdminRole.Logs preload must find logs written with LogEntityTypeAdminRole")

	var gotConfig models.Config
	require.NoError(t, db.Preload("Logs").First(&gotConfig, cfg.ID).Error)
	require.Len(t, gotConfig.Logs, 1, "Config.Logs preload must find logs written with LogEntityTypeConfig")
}
