package auth_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/integration/harness"

	"github.com/PhantomX7/athleton/internal/models"
)

// TestLoginRejectsBadCredentials covers the login failure paths end-to-end:
// wrong password, unknown username, and a deactivated account must all be
// rejected with 401 and the standard envelope — and with the same message, so
// the endpoint does not leak which part of the credentials was wrong.
func TestLoginRejectsBadCredentials(t *testing.T) {
	app := harness.New(t)

	inactive := models.User{
		Username: "inactive",
		Name:     "Inactive User",
		Email:    "inactive@test.local",
		Phone:    "+620000000006",
		Role:     models.UserRoleUser,
		Password: harness.PasswordHash(),
	}
	require.NoError(t, app.DB.Create(&inactive).Error)
	// IsActive carries a default:true column tag, so GORM replaces a false
	// zero-value on insert; deactivate explicitly like the admin flow does.
	require.NoError(t, app.DB.Model(&inactive).Update("is_active", false).Error)

	cases := []struct {
		name     string
		username string
		password string
	}{
		{"wrong password", harness.MemberUsername, "not-the-password"},
		{"unknown username", "who-is-this", harness.TestPassword},
		{"inactive user with correct password", "inactive", harness.TestPassword},
	}

	var messages []string
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := app.Request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
				"username": tc.username,
				"password": tc.password,
			}, "")

			require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
			env := harness.DecodeEnvelope(t, rec)
			require.False(t, env.Status)
			require.NotEmpty(t, env.Message)
			messages = append(messages, env.Message)
		})
	}

	// Uniform message across failure modes: no username/password oracle.
	for i := 1; i < len(messages); i++ {
		require.Equal(t, messages[0], messages[i],
			"login failures must not reveal which part of the credentials was wrong")
	}
}

// TestRefreshRejectsGarbageToken — a syntactically invalid refresh token is a
// client error with the standard envelope, not a 500.
func TestRefreshRejectsGarbageToken(t *testing.T) {
	app := harness.New(t)

	rec := app.Request(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "not-a-real-token",
	}, "")

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.False(t, harness.DecodeEnvelope(t, rec).Status)
}

// TestChangePasswordRejectsWrongOldPassword exercises the guard end-to-end:
// the session survives and the password is unchanged.
func TestChangePasswordRejectsWrongOldPassword(t *testing.T) {
	app := harness.New(t)
	tokens := app.LoginAs(t, harness.MemberUsername, harness.TestPassword)

	rec := app.Request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"old_password": "wrong-old-password",
		"new_password": harness.TestNewPassword,
		"except_token": tokens.RefreshToken,
	}, tokens.AccessToken)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.False(t, harness.DecodeEnvelope(t, rec).Status)

	// The old password still works.
	app.LoginAs(t, harness.MemberUsername, harness.TestPassword)
}
