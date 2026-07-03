package user_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"

	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

type userPayload struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

// TestManagingAdminAccountsRequiresAdminUserGrants — user:read/user:update
// govern regular accounts only; admin and root accounts are invisible and
// untouchable without the stronger admin_user:* grants.
func TestManagingAdminAccountsRequiresAdminUserGrants(t *testing.T) {
	app := harness.New(t)

	// The fixture admin holds only the regular-user grants.
	require.NoError(t, app.Casbin.AddRolePermissions(app.AdminRole.ID, []string{
		permissions.UserRead.String(),
		permissions.UserUpdate.String(),
	}))
	tokens := app.LoginAs(t, harness.AdminUsername, harness.TestPassword)

	// The listing hides admin and root accounts.
	rec := app.Request(t, http.MethodGet, "/api/v1/admin/user", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	var list []userPayload
	require.NoError(t, json.Unmarshal(env.Data, &list))
	require.Len(t, list, 1, "only regular accounts may appear without admin_user:read")
	require.Equal(t, app.MemberUser.ID, list[0].ID)

	// Fetching an admin account directly is denied.
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/user/"+harness.Itoa(app.AdminUser.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// Regular accounts remain manageable with user:update alone...
	rec = app.Request(t, http.MethodPatch, "/api/v1/admin/user/"+harness.Itoa(app.MemberUser.ID), map[string]string{
		"name": "Renamed Member",
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// ...but modifying an admin account is denied.
	rec = app.Request(t, http.MethodPatch, "/api/v1/admin/user/"+harness.Itoa(app.AdminUser.ID), map[string]string{
		"name": "Renamed Admin",
	}, tokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// With the stronger grants the same caller sees and manages admins.
	require.NoError(t, app.Casbin.AddRolePermissions(app.AdminRole.ID, []string{
		permissions.AdminUserRead.String(),
		permissions.AdminUserUpdate.String(),
	}))

	rec = app.Request(t, http.MethodGet, "/api/v1/admin/user", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env = harness.DecodeEnvelope(t, rec)
	require.NoError(t, json.Unmarshal(env.Data, &list))
	require.Len(t, list, 3, "admin_user:read exposes admin and root accounts in the listing")

	rec = app.Request(t, http.MethodGet, "/api/v1/admin/user/"+harness.Itoa(app.AdminUser.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = app.Request(t, http.MethodPatch, "/api/v1/admin/user/"+harness.Itoa(app.AdminUser.ID), map[string]string{
		"name": "Renamed Admin",
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
