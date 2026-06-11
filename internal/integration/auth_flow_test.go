package integration_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/models"
)

// TestAuthLoginRefreshLogoutFlow walks the complete session lifecycle through
// real HTTP requests:
//
//	login -> protected endpoint -> refresh rotation (old refresh and old
//	access both die) -> logout (new access dies via jti session binding).
func TestAuthLoginRefreshLogoutFlow(t *testing.T) {
	app := newTestApp(t)

	// Login returns access + refresh tokens.
	first := app.loginAs(t, rootUsername, testPassword)

	// The access token works on a protected admin endpoint.
	rec := app.request(t, http.MethodGet, "/api/v1/admin/post", nil, first.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Refresh rotates the token pair.
	rec = app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": first.RefreshToken,
	}, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var second tokenPair
	decodeData(t, decodeEnvelope(t, rec), &second)
	require.NotEmpty(t, second.AccessToken)
	require.NotEmpty(t, second.RefreshToken)
	require.NotEqual(t, first.AccessToken, second.AccessToken)
	require.NotEqual(t, first.RefreshToken, second.RefreshToken)

	// The old refresh token was revoked by rotation and is rejected on reuse.
	rec = app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": first.RefreshToken,
	}, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.False(t, decodeEnvelope(t, rec).Status)

	// The old access token carried the revoked session's jti, so it is dead
	// too. gin-jwt reports an authorizer denial (valid signature, revoked
	// session) as 403 rather than 401.
	rec = app.request(t, http.MethodGet, "/api/v1/admin/post", nil, first.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	require.False(t, decodeEnvelope(t, rec).Status)

	// The rotated access token works.
	rec = app.request(t, http.MethodGet, "/api/v1/admin/post", nil, second.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Logout revokes the session.
	rec = app.request(t, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refresh_token": second.RefreshToken,
	}, second.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// The access token minted for that session is now rejected (jti binding;
	// 403 from the authorizer, see above).
	rec = app.request(t, http.MethodGet, "/api/v1/admin/post", nil, second.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// And the logged-out refresh token cannot be used to mint new tokens.
	rec = app.request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": second.RefreshToken,
	}, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

// TestRegisterIssuesTokensAndRejectsDuplicateEmail covers public registration
// end-to-end, including the custom `unique=users.email` validator that
// bootstrap.SetupServer registers against the real database.
func TestRegisterIssuesTokensAndRejectsDuplicateEmail(t *testing.T) {
	app := newTestApp(t)

	payload := map[string]string{
		"name":          "New User",
		"business_name": "New Business",
		"email":         "new.user@test.local",
		"phone":         "+620000000099",
		"password":      "register-pass-1",
	}

	rec := app.request(t, http.MethodPost, "/api/v1/auth/register", payload, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env := decodeEnvelope(t, rec)
	require.True(t, env.Status)

	var tokens tokenPair
	decodeData(t, env, &tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)

	// The user row exists with the expected role.
	var user models.User
	require.NoError(t, app.db.Where("email = ?", "new.user@test.local").First(&user).Error)
	require.Equal(t, models.UserRoleUser, user.Role)
	require.True(t, user.IsActive)

	// Registering the same email again fails the unique validator with the
	// structured validation envelope.
	rec = app.request(t, http.MethodPost, "/api/v1/auth/register", payload, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	env = decodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "Validation failed", env.Message)

	var validation struct {
		TotalErrors int               `json:"total_errors"`
		Fields      map[string]string `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(env.Error, &validation))
	require.Contains(t, validation.Fields, "email")
}
