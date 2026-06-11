// Package middlewares provides shared Gin middleware for the API.
package middlewares

import (
	"net/http"

	"github.com/PhantomX7/athleton/internal/models"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/response"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// passwordChangeRequiredMessage is returned when an admin/root account still
// uses a password it did not choose itself (the seeder's default).
const passwordChangeRequiredMessage = "password change required"

// RequirePasswordChanged blocks admin/root accounts that have never changed
// their password (User.PasswordChangedAt == nil) from operating with the
// seeder's default password. It must run AFTER RequireAuth, whose authorizer
// loads the user record and stores it in the gin context under
// authjwt.AuthUserKey. Regular (non-admin) users pass through: they chose
// their own password at registration and are never seeded with a default.
//
// The auth module's change-password and logout endpoints live under /auth
// (outside the gated /admin group), so a gated account can always rotate its
// password and is never bricked.
func (m *Middleware) RequirePasswordChanged() gin.HandlerFunc {
	return func(c *gin.Context) {
		values, err := utils.ValuesFromContext(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
			c.Abort()
			return
		}

		// Only admin/root accounts are seeded with a default password; the
		// gate does not apply to other roles.
		if !models.UserRole(values.Role).IsAdminType() {
			c.Next()
			return
		}

		user, ok := userFromContext(c)
		if !ok {
			// RequireAuth did not run (or did not load the user). Without the
			// record we cannot prove the password was changed — fail closed.
			logger.Warn("RequirePasswordChanged ran without a loaded user; check middleware ordering",
				zap.String("request_id", utils.GetRequestIDFromContext(c.Request.Context())),
				zap.Uint("user_id", values.UserID),
			)
			c.JSON(http.StatusForbidden, response.BuildResponseFailed(passwordChangeRequiredMessage))
			c.Abort()
			return
		}

		if user.PasswordChangedAt == nil {
			c.JSON(http.StatusForbidden, response.BuildResponseFailed(passwordChangeRequiredMessage))
			c.Abort()
			return
		}

		c.Next()
	}
}

// userFromContext reads the user record stored by the JWT authorizer.
func userFromContext(c *gin.Context) (*models.User, bool) {
	value, exists := c.Get(authjwt.AuthUserKey)
	if !exists {
		return nil, false
	}
	user, ok := value.(*models.User)
	return user, ok && user != nil
}
