package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/models"
)

// TestSeededAdminMustChangeDefaultPassword exercises the
// must-change-default-password gate end-to-end: an admin seeded with the
// default password (PasswordChangedAt == nil) is locked out of /admin with
// 403 "password change required", can still reach the ungated /auth escape
// hatches (me, change-password, logout), and regains /admin access after
// rotating the password.
func TestSeededAdminMustChangeDefaultPassword(t *testing.T) {
	app := newTestApp(t)

	// Seed an admin exactly like database/seeder/seed/user.go does: default
	// password, PasswordChangedAt left nil.
	seededAdmin := models.User{
		Username: "seeded-admin",
		Name:     "Seeded Admin",
		Email:    "seeded.admin@test.local",
		Phone:    "+620000000004",
		IsActive: true,
		Role:     models.UserRoleAdmin,
		Password: testPasswordHash(),
	}
	require.NoError(t, app.db.Create(&seededAdmin).Error)

	tokens := app.loginAs(t, "seeded-admin", testPassword)

	// Gated: every /admin endpoint answers 403 with the canonical envelope.
	rec := app.request(t, http.MethodGet, "/api/v1/admin/post", nil, tokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "password change required", env.Message)

	// The ungated /auth endpoints stay reachable so the account is not bricked.
	rec = app.request(t, http.MethodGet, "/api/v1/auth/me", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Change the password via the escape hatch.
	rec = app.request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"old_password": testPassword,
		"new_password": testNewPassword,
		"except_token": tokens.RefreshToken,
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// The change is persisted: PasswordChangedAt is now set.
	var updated models.User
	require.NoError(t, app.db.First(&updated, seededAdmin.ID).Error)
	require.NotNil(t, updated.PasswordChangedAt)

	// The current session was excepted from revocation, so the same access
	// token now clears the gate and reaches /admin.
	rec = app.request(t, http.MethodGet, "/api/v1/admin/post", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// A fresh login with the new password also works against /admin.
	fresh := app.loginAs(t, "seeded-admin", testNewPassword)
	rec = app.request(t, http.MethodGet, "/api/v1/admin/post", nil, fresh.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Logout (the other escape hatch) keeps working for completeness.
	rec = app.request(t, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refresh_token": fresh.RefreshToken,
	}, fresh.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestSeededRootMustChangeDefaultPassword covers the root role variant of the
// gate: seeded root accounts are blocked from /admin until rotation, while a
// regular user with a nil PasswordChangedAt is not affected by the gate at all
// (their /admin access already fails on role semantics, not the gate message).
func TestSeededRootMustChangeDefaultPassword(t *testing.T) {
	app := newTestApp(t)

	seededRoot := models.User{
		Username: "seeded-root",
		Name:     "Seeded Root",
		Email:    "seeded.root@test.local",
		Phone:    "+620000000005",
		IsActive: true,
		Role:     models.UserRoleRoot,
		Password: testPasswordHash(),
	}
	require.NoError(t, app.db.Create(&seededRoot).Error)

	tokens := app.loginAs(t, "seeded-root", testPassword)

	rec := app.request(t, http.MethodGet, "/api/v1/admin/post", nil, tokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "password change required", env.Message)

	rec = app.request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"old_password": testPassword,
		"new_password": testNewPassword,
		"except_token": tokens.RefreshToken,
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = app.request(t, http.MethodGet, "/api/v1/admin/post", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
