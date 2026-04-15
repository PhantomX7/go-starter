package middlewares

import (
	"net/http"
	"slices"

	cerrors "github.com/PhantomX7/athleton/pkg/errors"
	"github.com/PhantomX7/athleton/pkg/utils"

	"github.com/gin-gonic/gin"
)

func (m *Middleware) HasRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get role from context
		contextValues, err := utils.ValuesFromContext(c.Request.Context())
		if err != nil {
			return
		}

		// Check if the user's role matches one of the allowed roles
		isAllowed := slices.Contains(allowedRoles, contextValues.Role)

		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, cerrors.NewForbiddenError("Forbidden"))
			return
		}

		c.Next()
	}
}
