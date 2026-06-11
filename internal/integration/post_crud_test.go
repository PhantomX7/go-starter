package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type postPayload struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TestPostCRUDLifecycleOverHTTP drives the post module through the real
// router: create, paginated list, get by id, partial PATCH (pointer DTO
// semantics preserve omitted fields), delete, and 404 after deletion.
func TestPostCRUDLifecycleOverHTTP(t *testing.T) {
	app := newTestApp(t)
	tokens := app.loginAs(t, rootUsername, testPassword)

	// Create two posts.
	rec := app.request(t, http.MethodPost, "/api/v1/admin/post", map[string]string{
		"name":        "First post",
		"description": "first description",
	}, tokens.AccessToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var first postPayload
	decodeData(t, decodeEnvelope(t, rec), &first)
	require.NotZero(t, first.ID)
	require.Equal(t, "First post", first.Name)
	require.Equal(t, "first description", first.Description)

	rec = app.request(t, http.MethodPost, "/api/v1/admin/post", map[string]string{
		"name": "Second post",
	}, tokens.AccessToken)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var second postPayload
	decodeData(t, decodeEnvelope(t, rec), &second)
	require.NotZero(t, second.ID)

	// List with pagination: limit=1, default order id desc.
	rec = app.request(t, http.MethodGet, "/api/v1/admin/post?limit=1", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	require.True(t, env.Status)
	require.NotNil(t, env.Meta)
	require.Equal(t, int64(2), env.Meta.Total)
	require.Equal(t, 1, env.Meta.Limit)
	require.Equal(t, 0, env.Meta.Offset)
	var page []postPayload
	require.NoError(t, json.Unmarshal(env.Data, &page))
	require.Len(t, page, 1)
	require.Equal(t, second.ID, page[0].ID, "default order is id desc")

	// Offset moves to the next page.
	rec = app.request(t, http.MethodGet, "/api/v1/admin/post?limit=1&offset=1", nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env = decodeEnvelope(t, rec)
	require.NoError(t, json.Unmarshal(env.Data, &page))
	require.Len(t, page, 1)
	require.Equal(t, first.ID, page[0].ID)

	// Get by id.
	rec = app.request(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/post/%d", first.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var fetched postPayload
	decodeData(t, decodeEnvelope(t, rec), &fetched)
	require.Equal(t, first.ID, fetched.ID)
	require.Equal(t, "First post", fetched.Name)

	// PATCH with only description: the omitted name must be preserved
	// (pointer-DTO semantics — nil pointers are skipped on copy).
	rec = app.request(t, http.MethodPatch, fmt.Sprintf("/api/v1/admin/post/%d", first.ID), map[string]string{
		"description": "updated description",
	}, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var updated postPayload
	decodeData(t, decodeEnvelope(t, rec), &updated)
	require.Equal(t, "First post", updated.Name, "omitted field must keep its value")
	require.Equal(t, "updated description", updated.Description)

	// Re-read to confirm persistence of the partial update.
	rec = app.request(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/post/%d", first.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	decodeData(t, decodeEnvelope(t, rec), &fetched)
	require.Equal(t, "First post", fetched.Name)
	require.Equal(t, "updated description", fetched.Description)

	// Delete, then a subsequent GET is a 404 with the failure envelope.
	rec = app.request(t, http.MethodDelete, fmt.Sprintf("/api/v1/admin/post/%d", first.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = app.request(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/post/%d", first.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	require.False(t, decodeEnvelope(t, rec).Status)

	// The other post is untouched.
	rec = app.request(t, http.MethodGet, fmt.Sprintf("/api/v1/admin/post/%d", second.ID), nil, tokens.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
