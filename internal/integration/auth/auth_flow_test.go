package auth_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"

	"github.com/PhantomX7/athleton/internal/models"
)

// TestAuthLoginRefreshLogoutFlow walks the complete session lifecycle through
// real HTTP requests:
//
//	login -> protected endpoint -> refresh rotation (refresh token swapped
//	in place: the old refresh token dies, but access tokens minted for the
//	session stay valid until expiry) -> logout (all access dies via jti
//	session binding).
func TestAuthLoginRefreshLogoutFlow(t *testing.T) {
	app := harness.New(t)

	// Login returns access + refresh tokens.
	first := app.LoginAs(t, harness.RootUsername, harness.TestPassword)

	// The access token works on a protected admin endpoint.
	rec := app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, first.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Refresh swaps the refresh-token hash on the same session row. The new
	// access token reuses the session's jti, so it can be byte-identical to
	// the first one when minted within the same second â€” only the refresh
	// token is guaranteed to change.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": first.RefreshToken,
	}, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var second harness.TokenPair
	harness.DecodeData(t, harness.DecodeEnvelope(t, rec), &second)
	require.NotEmpty(t, second.AccessToken)
	require.NotEmpty(t, second.RefreshToken)
	require.NotEqual(t, first.RefreshToken, second.RefreshToken)

	// The old refresh token was invalidated by rotation and is rejected on
	// reuse.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": first.RefreshToken,
	}, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.False(t, harness.DecodeEnvelope(t, rec).Status)

	// Rotation keeps the session row (and its jti) alive, so access tokens
	// minted before the refresh keep working until they expire â€” parallel
	// requests no longer race the refresh.
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, first.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// The rotated access token works.
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, second.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Logout revokes the session.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/logout", map[string]string{
		"refresh_token": second.RefreshToken,
	}, second.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// Every access token minted for that session is now rejected (jti
	// binding; gin-jwt reports the authorizer denial as 403 rather than 401).
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, second.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = app.Request(t, http.MethodGet, "/api/v1/admin/log", nil, first.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// And the logged-out refresh token cannot be used to mint new tokens.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": second.RefreshToken,
	}, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

// TestRegisterIssuesTokensAndRejectsDuplicateEmail covers public registration
// end-to-end, including the custom `unique=users.email` validator that
// bootstrap.SetupServer registers against the real database.
func TestRegisterIssuesTokensAndRejectsDuplicateEmail(t *testing.T) {
	app := harness.New(t)

	payload := map[string]string{
		"name":          "New User",
		"business_name": "New Business",
		"email":         "new.user@test.local",
		"phone":         "+620000000099",
		"password":      "register-pass-1",
	}

	rec := app.Request(t, http.MethodPost, "/api/v1/auth/register", payload, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	env := harness.DecodeEnvelope(t, rec)
	require.True(t, env.Status)

	var tokens harness.TokenPair
	harness.DecodeData(t, env, &tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)

	// The user row exists with the expected role.
	var user models.User
	require.NoError(t, app.DB.Where("email = ?", "new.user@test.local").First(&user).Error)
	require.Equal(t, models.UserRoleUser, user.Role)
	require.True(t, user.IsActive)

	// Registering the same email again fails the unique validator with the
	// structured validation envelope.
	rec = app.Request(t, http.MethodPost, "/api/v1/auth/register", payload, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	env = harness.DecodeEnvelope(t, rec)
	require.False(t, env.Status)
	require.Equal(t, "Validation failed", env.Message)

	var validation struct {
		TotalErrors int               `json:"total_errors"`
		Fields      map[string]string `json:"fields"`
	}
	require.NoError(t, json.Unmarshal(env.Error, &validation))
	require.Contains(t, validation.Fields, "email")
}
