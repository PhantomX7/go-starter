package adminrole_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"

	"github.com/PhantomX7/athleton/pkg/constants/permissions"
)

type adminRolePayload struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// TestAdminRoleCRUDLifecycleOverHTTP drives the admin-role module through the
// real router: create (with Casbin permission sync), paginated list, get by
// id, partial PATCH (omitted fields preserved, permission set replaced), and
// delete with Casbin policy cleanup. This is the scaffold's canonical CRUD
// walkthrough now that the post POC module is gone.
func TestAdminRoleCRUDLifecycleOverHTTP(t *testing.T) {
	app := harness.New(t)
	tokens := app.LoginAs(t, harness.RootUsername, harness.TestPassword)

	// Create a role; the service must mirror the grants into Casbin.
	rec := app.Request(t, http.MethodPost, "/api/v1/admin/admin-role", map[string]any{
		"name":        "Support",
		"description": "support staff",
		"permissions": []string{permissions.LogRead.String()},
	}, tokens.AccessToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var created adminRolePayload
	harness.DecodeData(t, harness.DecodeEnvelope(t, rec), &created)
	require.NotZero(t, created.ID)
	require.Equal(t, "Support", created.Name)
	require.Equal(t, []string{permissions.LogRead.String()}, created.Permissions)
	require.Equal(t, []string{permissions.LogRead.String()}, app.Casbin.GetRolePermissions(created.ID))

	// Invalid permission strings are rejected up front.
	rec = app.Request(t, http.MethodPost, "/api/v1/admin/admin-role", map[string]any{
		"name":        "Broken",
		"permissions": []string{"not:a-permission"},
	}, tokens.AccessToken)
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	// List includes the seeded fixture role and the new one.
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/admin-role?limit=1", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.NotNil(t, env.Meta)
	require.Equal(t, int64(2), env.Meta.Total)
	var page []adminRolePayload
	require.NoError(t, json.Unmarshal(env.Data, &page))
	require.Len(t, page, 1)

	// Get by id.
	rec = app.Request(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/admin-role/%d", created.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var fetched adminRolePayload
	harness.DecodeData(t, harness.DecodeEnvelope(t, rec), &fetched)
	require.Equal(t, "Support", fetched.Name)

	// PATCH with only a permission change: the omitted name is preserved and
	// the Casbin grant set is replaced, not appended.
	rec = app.Request(t, http.MethodPatch, fmt.Sprintf("/api/v1/admin/admin-role/%d", created.ID), map[string]any{
		"permissions": []string{permissions.UserRead.String()},
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var updated adminRolePayload
	harness.DecodeData(t, harness.DecodeEnvelope(t, rec), &updated)
	require.Equal(t, "Support", updated.Name, "omitted field must keep its value")
	require.Equal(t, []string{permissions.UserRead.String()}, app.Casbin.GetRolePermissions(created.ID))

	// Delete removes the role and its Casbin policies; a later GET is 404.
	rec = app.Request(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/admin-role/%d", created.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Empty(t, app.Casbin.GetRolePermissions(created.ID), "casbin grants must be cleaned up on delete")

	rec = app.Request(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/admin-role/%d", created.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	require.False(t, harness.DecodeEnvelope(t, rec).Status)
}
