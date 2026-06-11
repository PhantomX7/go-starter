package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/audit"
	"github.com/PhantomX7/athleton/internal/models"
)

// drainAudit waits for all in-flight background audit writes to land.
func drainAudit(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, audit.Drain(ctx))
}

// TestAdminLoginWritesAuditLog verifies that a successful admin login produces
// an audit log row. The login log is written by a detached goroutine (not
// tracked by audit.Drain), so the assertion polls with a deadline.
func TestAdminLoginWritesAuditLog(t *testing.T) {
	app := newTestApp(t)

	app.loginAs(t, adminUsername, testPassword)

	row := app.waitForAuditLog(t, models.LogActionLogin, app.adminUser.ID)
	require.Equal(t, models.LogEntityTypeUser, row.EntityType)
	require.NotNil(t, row.UserID)
	require.Equal(t, app.adminUser.ID, *row.UserID)
	require.Contains(t, row.Message, app.adminUser.Name)
}

// TestChangePasswordWritesAuditLogAndRevokesOtherSessions covers the admin
// change-password mutation end-to-end: the audit row is written through
// audit.Record (waited on via audit.Drain), every other session is revoked
// while the excepted one survives, and the new password becomes effective.
func TestChangePasswordWritesAuditLogAndRevokesOtherSessions(t *testing.T) {
	app := newTestApp(t)

	keep := app.loginAs(t, adminUsername, testPassword)
	other := app.loginAs(t, adminUsername, testPassword)

	rec := app.request(t, http.MethodPost, "/api/v1/auth/change-password", map[string]string{
		"old_password": testPassword,
		"new_password": testNewPassword,
		"except_token": keep.RefreshToken,
	}, keep.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// The audit write is async via audit.Record; Drain waits for it.
	drainAudit(t)
	row := app.waitForAuditLog(t, models.LogActionChangePassword, app.adminUser.ID)
	require.Equal(t, models.LogEntityTypeUser, row.EntityType)
	require.Contains(t, row.Message, "changed password")

	// The excepted session is still alive...
	rec = app.request(t, http.MethodGet, "/api/v1/auth/me", nil, keep.AccessToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	// ...while the other session's access token was killed via jti binding
	// (gin-jwt reports an authorizer denial as 403).
	rec = app.request(t, http.MethodGet, "/api/v1/auth/me", nil, other.AccessToken)
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	// The new password is effective immediately.
	app.loginAs(t, adminUsername, testNewPassword)
}
