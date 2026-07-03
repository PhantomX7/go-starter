package middlewares_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/libs/casbin"
	casbinmocks "github.com/PhantomX7/athleton/libs/casbin/mocks"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
	"github.com/PhantomX7/athleton/pkg/utils"
)

func setupLogger(t *testing.T) {
	t.Helper()

	prev := logger.Log
	logger.Log = zap.NewNop()
	t.Cleanup(func() {
		logger.Log = prev
	})
}

// newCasbinClient builds a generated casbin mock whose CheckPermissionWithRoot
// mirrors the real client's decision so the guards exercise the production
// authorization rule: root bypasses, non-admin (or an admin with no role) is
// denied, and admins fall through to checkPermissionFn. A nil checkPermissionFn
// leaves CheckPermission unmocked so it panics if unexpectedly called.
func newCasbinClient(checkPermissionFn func(uint, string) (bool, error)) *casbinmocks.ClientMock {
	mock := &casbinmocks.ClientMock{
		CheckPermissionFunc: checkPermissionFn,
	}
	mock.CheckPermissionWithRootFunc = func(userRole string, adminRoleID *uint, permission string) (bool, error) {
		if userRole == "root" {
			return true, nil
		}
		if userRole != "admin" || adminRoleID == nil {
			return false, nil
		}
		return mock.CheckPermission(*adminRoleID, permission)
	}
	return mock
}

// newMiddleware builds the middleware bundle without a JWT dependency; tests
// here never exercise RequireAuth/LoginHandler, which are gin-jwt passthroughs.
// The zero-value config leaves CORS in wildcard mode; tests that need an
// origin allowlist use newMiddlewareWithConfig instead.
func newMiddleware(casbinClient casbin.Client) *middlewares.Middleware {
	return newMiddlewareWithConfig(&config.Config{}, casbinClient)
}

// newMiddlewareWithConfig builds the middleware bundle with an explicit config
// for tests that exercise config-driven behavior (e.g. the CORS allowlist).
func newMiddlewareWithConfig(cfg *config.Config, casbinClient casbin.Client) *middlewares.Middleware {
	return middlewares.NewMiddleware(cfg, nil, casbinClient)
}

// withContextValues stands in for the JWT middleware by injecting
// authenticated context values into the request context.
func withContextValues(values utils.ContextValues) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(utils.NewContextWithValues(c.Request.Context(), values))
		c.Next()
	}
}
