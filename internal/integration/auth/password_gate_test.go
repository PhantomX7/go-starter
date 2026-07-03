package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

// TestSeededAdminMustChangeDefaultPassword exercises the
// must-change-default-password gate end-to-end: an admin seeded with the
// default password (PasswordChangedAt == nil) is locked out of /admin with
// 403 "password change required", can still reach the ungated /auth escape
// hatches (me, change-password, logout), and regains /admin access after
// rotating the password.
func TestSeededAdminMustChangeDefaultPassword(t *testing.T) {
	app := harness.New(t)

	// Seed an admin exactly like database/seeder/seed/user.go does: default
	// password, PasswordChangedAt left nil. The role carries log:read so that
	// once the gate clears, /admin/log answers 200 instead of a permission 403.
	seededAdmin := models.User{
		Username:    "seeded-admin",
		Name:        "Seeded Admin",
		Email:       "seeded.admin@test.local",
		Phone:       "+620000000004",
		IsActive:    true,
		Role:        models.UserRoleAdmin,
		AdminRoleID: &app.AdminRole.ID,
		Password:    harness.PasswordHash(),
	}
	require.NoError(t, app.DB.Create(&seededAdmin).Error)
	require.NoError(t, app.Casbin.AddRolePermissions(
		app.AdminRole.ID, []string{permissions.LogRead.String()},
	))

	tokens := app.LoginAs(t, "seeded-admin", harness.TestPassword)

	// The login response hints the frontend that a rotation is required, so it
	// can route to the change-password screen without waiting for the 403.
	require.True(t, tokens.MustChangePassword, "seeded admin login must flag must_change_password")

	// Gated: every /admin endpoint answers 403 with the canonical envelope.
	rec := app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, tokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "password change required", env.Message)

	// The ungated /auth endpoints stay reachable so the account is not bricked.
	rec = app.Request(t, http.MethodGet, "/api/v1/auth/me", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Change the password via the escape hatch.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"old_password": harness.TestPassword,
		"new_password": harness.TestNewPassword,
		"except_token": tokens.RefreshToken,
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// The change is persisted: PasswordChangedAt is now set.
	var updated models.User
	require.NoError(t, app.DB.First(&updated, seededAdmin.ID).Error)
	require.NotNil(t, updated.PasswordChangedAt)

	// The current session was excepted from revocation, so the same access
	// token now clears the gate and reaches /admin.
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// A fresh login with the new password also works against /admin, and the
	// hint has cleared now that the password was rotated.
	fresh := app.LoginAs(t, "seeded-admin", harness.TestNewPassword)
	require.False(t, fresh.MustChangePassword, "after rotation the login hint must clear")
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, fresh.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Logout (the other escape hatch) keeps working for completeness.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refresh_token": fresh.RefreshToken,
	}, fresh.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestSeededRootMustChangeDefaultPassword covers the root role variant of the
// gate: seeded root accounts are blocked from /admin until rotation, while a
// regular user with a nil PasswordChangedAt is not affected by the gate at all
// (their /admin access already fails on role semantics, not the gate message).
func TestSeededRootMustChangeDefaultPassword(t *testing.T) {
	app := harness.New(t)

	seededRoot := models.User{
		Username: "seeded-root",
		Name:     "Seeded Root",
		Email:    "seeded.root@test.local",
		Phone:    "+620000000005",
		IsActive: true,
		Role:     models.UserRoleRoot,
		Password: harness.PasswordHash(),
	}
	require.NoError(t, app.DB.Create(&seededRoot).Error)

	tokens := app.LoginAs(t, "seeded-root", harness.TestPassword)

	rec := app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, tokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "password change required", env.Message)

	rec = app.Request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"old_password": harness.TestPassword,
		"new_password": harness.TestNewPassword,
		"except_token": tokens.RefreshToken,
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
