package user_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

// TestAdminCanCreateAdminUserEndToEnd — an admin-created account comes up as
// role=admin with the assigned admin role, and must rotate its creator-chosen
// password before reaching /admin (the must-change-default-password gate).
func TestAdminCanCreateAdminUserEndToEnd(t *testing.T) {
	app := harness.New(t)
	rootTokens := app.LoginAs(t, harness.RootUsername, harness.TestPassword)

	// Give the fixture role a grant so the new admin can reach /admin/log later.
	require.NoError(t, app.Casbin.AddRolePermissions(
		app.AdminRole.ID, []string{permissions.LogRead.String()},
	))

	// Root creates the admin account. A smuggled "role":"root" field must be
	// ignored — the DTO does not bind a role at all.
	rec := app.Request(t, http.MethodPost, "/api/v1/admin/user", map[string]any{
		"username":      "second-admin",
		"name":          "Second Admin",
		"email":         "second.admin@test.local",
		"phone":         "+620000000008",
		"password":      "initial-pass-123",
		"admin_role_id": app.AdminRole.ID,
		"role":          "root",
	}, rootTokens.AccessToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var created models.User
	require.NoError(t, app.DB.Where("username = ?", "second-admin").First(&created).Error)
	require.Equal(t, models.UserRoleAdmin, created.Role, "created account must be admin, never root")
	require.Equal(t, &app.AdminRole.ID, created.AdminRoleID)
	require.Nil(t, created.PasswordChangedAt, "creator-chosen password must count as unrotated")

	// The new admin can log in but is gated until rotating the password.
	tokens := app.LoginAs(t, "second-admin", "initial-pass-123")
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, tokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	require.Equal(t, "password change required", harness.DecodeEnvelope(t, rec).Message)

	// After rotation the granted endpoint opens up.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"old_password": "initial-pass-123",
		"new_password": harness.TestNewPassword,
		"except_token": tokens.RefreshToken,
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestRootAccountsCannotBeModifiedOverHTTP — no admin endpoint may touch a
// root account: update, role assignment, and password change all 403, and
// there is no delete endpoint at all.
func TestRootAccountsCannotBeModifiedOverHTTP(t *testing.T) {
	app := harness.New(t)
	rootTokens := app.LoginAs(t, harness.RootUsername, harness.TestPassword)
	rootID := harness.Itoa(app.RootUser.ID)

	rec := app.Request(t, http.MethodPatch, "/api/v1/admin/user/"+rootID, map[string]string{
		"name": "renamed-root",
	}, rootTokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	rec = app.Request(t, http.MethodPost, "/api/v1/admin/user/"+rootID+"/admin-role", map[string]any{
		"admin_role_id": app.AdminRole.ID,
	}, rootTokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	rec = app.Request(t, http.MethodPost, "/api/v1/admin/user/"+rootID+"/change-password", map[string]string{
		"new_password": "new-root-pass-123",
	}, rootTokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// No delete route exists on the user resource.
	rec = app.Request(t, http.MethodDelete, "/api/v1/admin/user/"+rootID, nil, rootTokens.AccessToken)
	require.Equal(t, http.StatusMethodNotAllowed, rec.Code, rec.Body.String())
}
