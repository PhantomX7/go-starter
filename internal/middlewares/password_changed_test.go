package middlewares_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/models"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/pkg/utils"
)

// withAuthenticatedUser stands in for the full JWT middleware: it injects the
// authenticated context values AND stores the loaded user record in the gin
// context the way the authorizer does. user may be nil to simulate a missing
// record (middleware misordering).
func withAuthenticatedUser(values utils.ContextValues, user *models.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(utils.NewContextWithValues(c.Request.Context(), values))
		if user != nil {
			c.Set(authjwt.AuthUserKey, user)
		}
		c.Next()
	}
}

func passwordChangedAt() *time.Time {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return &ts
}

func TestRequirePasswordChangedRejectsMissingContextValues(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)

	rec := serve(newAuthRouter(nil, nil, m.RequirePasswordChanged()))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequirePasswordChangedAllowsRegularUserWithNilTimestamp(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	identity := withAuthenticatedUser(
		utils.ContextValues{UserID: 7, Role: models.UserRoleUser.ToString()},
		&models.User{ID: 7, Role: models.UserRoleUser},
	)

	rec := serve(newAuthRouter(nil, identity, m.RequirePasswordChanged()))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePasswordChangedRejectsAdminWithNilTimestamp(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	identity := withAuthenticatedUser(
		utils.ContextValues{UserID: 7, Role: models.UserRoleAdmin.ToString()},
		&models.User{ID: 7, Role: models.UserRoleAdmin},
	)

	rec := serve(newAuthRouter(nil, identity, m.RequirePasswordChanged()))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "password change required")
}

func TestRequirePasswordChangedRejectsRootWithNilTimestamp(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	identity := withAuthenticatedUser(
		utils.ContextValues{UserID: 1, Role: models.UserRoleRoot.ToString()},
		&models.User{ID: 1, Role: models.UserRoleRoot},
	)

	rec := serve(newAuthRouter(nil, identity, m.RequirePasswordChanged()))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "password change required")
}

func TestRequirePasswordChangedAllowsAdminWithTimestampSet(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	identity := withAuthenticatedUser(
		utils.ContextValues{UserID: 7, Role: models.UserRoleAdmin.ToString()},
		&models.User{ID: 7, Role: models.UserRoleAdmin, PasswordChangedAt: passwordChangedAt()},
	)

	rec := serve(newAuthRouter(nil, identity, m.RequirePasswordChanged()))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePasswordChangedAllowsRootWithTimestampSet(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	identity := withAuthenticatedUser(
		utils.ContextValues{UserID: 1, Role: models.UserRoleRoot.ToString()},
		&models.User{ID: 1, Role: models.UserRoleRoot, PasswordChangedAt: passwordChangedAt()},
	)

	rec := serve(newAuthRouter(nil, identity, m.RequirePasswordChanged()))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePasswordChangedFailsClosedWhenUserNotLoaded(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	// Admin context values but no user record in the gin context: the gate
	// cannot prove the password was changed and must fail closed.
	identity := withAuthenticatedUser(
		utils.ContextValues{UserID: 7, Role: models.UserRoleAdmin.ToString()},
		nil,
	)

	rec := serve(newAuthRouter(nil, identity, m.RequirePasswordChanged()))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "password change required")
}
