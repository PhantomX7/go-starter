package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

// TestAdminEndpointsRejectMissingOrInvalidToken verifies the RequireAuth guard
// on the /admin surface returns 401 with the standard JSON envelope.
func TestAdminEndpointsRejectMissingOrInvalidToken(t *testing.T) {
	app := newTestApp(t)

	t.Run("missing token", func(t *testing.T) {
		rec := app.request(t, http.MethodGet, "/api/v1/admin/post", nil, "")
		require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
		env := decodeEnvelope(t, rec)
		require.False(t, env.Status)
		require.NotEmpty(t, env.Message)
	})

	t.Run("garbage token", func(t *testing.T) {
		rec := app.request(t, http.MethodGet, "/api/v1/admin/post", nil, "not-a-jwt")
		require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
		require.False(t, decodeEnvelope(t, rec).Status)
	})
}

// TestPermissionEnforcementOnGuardedEndpoint exercises RequirePermission via
// Casbin on GET /api/v1/admin/log (guarded by log:read): an admin whose role
// has no grant gets 403, gets 200 once the rule is seeded, and root bypasses
// the check entirely.
func TestPermissionEnforcementOnGuardedEndpoint(t *testing.T) {
	app := newTestApp(t)

	adminTokens := app.loginAs(t, adminUsername, testPassword)

	// Admin without the casbin grant is denied.
	rec := app.request(t, http.MethodGet, "/api/v1/admin/log", nil, adminTokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "insufficient permissions", env.Message)

	// Seed the casbin rule for the admin's role; same token now passes.
	require.NoError(t, app.casbinClient.AddRolePermissions(
		app.adminRole.ID, []string{permissions.LogRead.String()},
	))
	rec = app.request(t, http.MethodGet, "/api/v1/admin/log", nil, adminTokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.True(t, decodeEnvelope(t, rec).Status)

	// Root bypasses permission checks without any casbin rule.
	rootTokens := app.loginAs(t, rootUsername, testPassword)
	rec = app.request(t, http.MethodGet, "/api/v1/admin/log", nil, rootTokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestRolePermissionRevocationRemovesAccess covers the replace semantics the
// admin-role Update endpoint relies on (casbin SetRolePermissions): a role
// granted log:read reaches the guarded endpoint, and once its permission set is
// replaced with an unrelated grant the SAME access token is denied — proving a
// permission edit propagates to live sessions without re-login. The existing
// enforcement test only covers the grant direction; this covers revocation.
func TestRolePermissionRevocationRemovesAccess(t *testing.T) {
	app := newTestApp(t)

	adminTokens := app.loginAs(t, adminUsername, testPassword)

	// Grant log:read via the same replace call the Update endpoint issues.
	require.NoError(t, app.casbinClient.SetRolePermissions(
		app.adminRole.ID, []string{permissions.LogRead.String()},
	))
	rec := app.request(t, http.MethodGet, "/api/v1/admin/log", nil, adminTokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Replace the role's permissions with an unrelated grant: log:read is dropped.
	require.NoError(t, app.casbinClient.SetRolePermissions(
		app.adminRole.ID, []string{permissions.UserRead.String()},
	))
	rec = app.request(t, http.MethodGet, "/api/v1/admin/log", nil, adminTokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	require.Equal(t, "insufficient permissions", decodeEnvelope(t, rec).Message)
}

// TestNonAdminRoleDeniedOnGuardedEndpoint verifies that an authenticated user
// whose role is neither root nor admin is denied on a permission-guarded
// endpoint regardless of casbin state.
func TestNonAdminRoleDeniedOnGuardedEndpoint(t *testing.T) {
	app := newTestApp(t)

	memberTokens := app.loginAs(t, memberUsername, testPassword)

	rec := app.request(t, http.MethodGet, "/api/v1/admin/log", nil, memberTokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "insufficient permissions", env.Message)
}
