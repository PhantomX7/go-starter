package middlewares_test

import (
	"testing"

	casbinv2 "github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/libs/casbin"
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

type mockCasbinClient struct {
	checkPermissionFn func(uint, string) (bool, error)
}

func (m *mockCasbinClient) GetEnforcer() *casbinv2.Enforcer { return nil }

func (m *mockCasbinClient) AddRolePermissions(uint, []string) error {
	panic("unexpected AddRolePermissions call")
}

func (m *mockCasbinClient) RemoveRolePermissions(uint, []string) error {
	panic("unexpected RemoveRolePermissions call")
}

func (m *mockCasbinClient) SetRolePermissions(uint, []string) error {
	panic("unexpected SetRolePermissions call")
}

func (m *mockCasbinClient) GetRolePermissions(uint) []string {
	panic("unexpected GetRolePermissions call")
}

func (m *mockCasbinClient) CheckPermission(roleID uint, permission string) (bool, error) {
	if m.checkPermissionFn == nil {
		panic("unexpected CheckPermission call")
	}
	return m.checkPermissionFn(roleID, permission)
}

func (m *mockCasbinClient) CheckPermissionWithRoot(string, *uint, string) (bool, error) {
	panic("unexpected CheckPermissionWithRoot call")
}

func (m *mockCasbinClient) DeleteRole(uint) error {
	panic("unexpected DeleteRole call")
}

var _ casbin.Client = (*mockCasbinClient)(nil)

// newMiddleware builds the middleware bundle without a JWT dependency; tests
// here never exercise RequireAuth/LoginHandler, which are gin-jwt passthroughs.
func newMiddleware(casbinClient casbin.Client) *middlewares.Middleware {
	return middlewares.NewMiddleware(nil, casbinClient)
}

// withContextValues stands in for the JWT middleware by injecting
// authenticated context values into the request context.
func withContextValues(values utils.ContextValues) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(utils.NewContextWithValues(c.Request.Context(), values))
		c.Next()
	}
}
