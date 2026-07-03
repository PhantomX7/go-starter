package integration_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/models"
)

type configPayload struct {
	ID       uint   `json:"id"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsPublic bool   `json:"is_public"`
}

// seedConfigs inserts one public and one private config row. IsPublic has a
// default:false column tag, so the public flag is set with an explicit Update.
func seedConfigs(t *testing.T, app *testApp) (public, private models.Config) {
	t.Helper()

	public = models.Config{Key: "site_name", Value: "Athleton"}
	private = models.Config{Key: "smtp_password", Value: "hunter2"}
	require.NoError(t, app.db.Create(&public).Error)
	require.NoError(t, app.db.Create(&private).Error)
	require.NoError(t, app.db.Model(&public).Update("is_public", true).Error)
	return public, private
}

// TestPublicConfigEndpointsExposeOnlyPublicRows — the unauthenticated
// /public/config surface must never leak rows that are not explicitly marked
// public (a config table naturally accumulates secrets).
func TestPublicConfigEndpointsExposeOnlyPublicRows(t *testing.T) {
	app := newTestApp(t)
	public, private := seedConfigs(t, app)

	// Listing shows only the public row.
	rec := app.request(t, http.MethodGet, "/api/v1/public/config", nil, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	var list []configPayload
	require.NoError(t, json.Unmarshal(env.Data, &list))
	require.Len(t, list, 1)
	require.Equal(t, public.Key, list[0].Key)

	// Fetch by key works for the public row.
	rec = app.request(t, http.MethodGet, "/api/v1/public/config/key/"+public.Key, nil, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// A private key is indistinguishable from a missing one: both 404.
	rec = app.request(t, http.MethodGet, "/api/v1/public/config/key/"+private.Key, nil, "")
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	rec = app.request(t, http.MethodGet, "/api/v1/public/config/key/does-not-exist", nil, "")
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

// TestAdminConfigEndpointsSeeEverythingAndToggleVisibility — admins (root
// here; it bypasses permission checks) see private rows and can flip
// is_public through the update endpoint.
func TestAdminConfigEndpointsSeeEverythingAndToggleVisibility(t *testing.T) {
	app := newTestApp(t)
	_, private := seedConfigs(t, app)
	tokens := app.loginAs(t, rootUsername, testPassword)

	// Admin listing includes both rows.
	rec := app.request(t, http.MethodGet, "/api/v1/admin/config", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	var list []configPayload
	require.NoError(t, json.Unmarshal(env.Data, &list))
	require.Len(t, list, 2)

	// Toggle the private row public via PATCH.
	rec = app.request(t, http.MethodPatch, "/api/v1/admin/config/"+itoa(private.ID), map[string]any{
		"value":     private.Value,
		"is_public": true,
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// It is now visible on the public surface.
	rec = app.request(t, http.MethodGet, "/api/v1/public/config/key/"+private.Key, nil, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestAdminGroupRejectsNonAdminRoles — the /admin group itself must reject
// role=user before any handler or per-route permission guard runs, so a
// future unguarded admin route is not reachable by ordinary accounts. The
// harness mounts an unguarded probe route for exactly this test.
func TestAdminGroupRejectsNonAdminRoles(t *testing.T) {
	app := newTestApp(t)

	memberTokens := app.loginAs(t, memberUsername, testPassword)
	rec := app.request(t, http.MethodGet, "/api/v1/admin/__probe", nil, memberTokens.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	require.False(t, decodeEnvelope(t, rec).Status)

	// Admin-type roles pass the group gate.
	rootTokens := app.loginAs(t, rootUsername, testPassword)
	rec = app.request(t, http.MethodGet, "/api/v1/admin/__probe", nil, rootTokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func itoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
