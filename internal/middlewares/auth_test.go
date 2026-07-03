package middlewares_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/PhantomX7/athleton/internal/models"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/pkg/constants/permissions"
	"github.com/PhantomX7/athleton/pkg/utils"
)

func newAuthRouter(casbinClient casbin.Client, identity gin.HandlerFunc, guard gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers := []gin.HandlerFunc{}
	if identity != nil {
		handlers = append(handlers, identity)
	}
	handlers = append(handlers, guard, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/test", handlers...)
	return r
}

func serve(r *gin.Engine) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))
	return rec
}

func adminValues(roleID uint) utils.ContextValues {
	return utils.ContextValues{
		UserID:      7,
		UserName:    "alice",
		Role:        models.UserRoleAdmin.ToString(),
		AdminRoleID: &roleID,
	}
}

func TestRequireRoleRejectsMissingContextValues(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)

	rec := serve(newAuthRouter(nil, nil, m.RequireRole("admin")))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireRoleRejectsDisallowedRole(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	identity := withContextValues(utils.ContextValues{UserID: 7, Role: "member"})

	rec := serve(newAuthRouter(nil, identity, m.RequireRole("admin", "root")))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)
	identity := withContextValues(utils.ContextValues{UserID: 7, Role: "admin"})

	rec := serve(newAuthRouter(nil, identity, m.RequireRole("admin", "root")))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermissionRejectsMissingContextValues(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(nil)

	rec := serve(newAuthRouter(nil, nil, m.RequirePermission(permissions.UserRead)))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequirePermissionBypassesChecksForRoot(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{} // panics if CheckPermission is called
	m := newMiddleware(casbinClient)
	identity := withContextValues(utils.ContextValues{UserID: 1, Role: models.UserRoleRoot.ToString()})

	rec := serve(newAuthRouter(casbinClient, identity, m.RequirePermission(permissions.UserRead)))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermissionRejectsNonAdminRoles(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(&mockCasbinClient{})
	identity := withContextValues(utils.ContextValues{UserID: 7, Role: "member"})

	rec := serve(newAuthRouter(nil, identity, m.RequirePermission(permissions.UserRead)))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermissionRejectsAdminWithoutAssignedRole(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(&mockCasbinClient{})
	identity := withContextValues(utils.ContextValues{UserID: 7, Role: models.UserRoleAdmin.ToString()})

	rec := serve(newAuthRouter(nil, identity, m.RequirePermission(permissions.UserRead)))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermissionAllowsGrantedPermission(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(roleID uint, permission string) (bool, error) {
			require.Equal(t, uint(3), roleID)
			require.Equal(t, permissions.UserRead.String(), permission)
			return true, nil
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)), m.RequirePermission(permissions.UserRead)))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequirePermissionRejectsDeniedPermission(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(uint, string) (bool, error) {
			return false, nil
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)), m.RequirePermission(permissions.UserRead)))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequirePermissionFailsClosedOnCasbinError(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(uint, string) (bool, error) {
			return false, errors.New("enforcer unavailable")
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)), m.RequirePermission(permissions.UserRead)))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRequireAnyPermissionAllowsWhenOneGranted(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(_ uint, permission string) (bool, error) {
			return permission == permissions.LogRead.String(), nil
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)),
		m.RequireAnyPermission(permissions.UserRead, permissions.LogRead)))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAnyPermissionRejectsWhenNoneGranted(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(uint, string) (bool, error) {
			return false, nil
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)),
		m.RequireAnyPermission(permissions.UserRead, permissions.LogRead)))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireAnyPermissionBypassesChecksForRoot(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(&mockCasbinClient{})
	identity := withContextValues(utils.ContextValues{UserID: 1, Role: models.UserRoleRoot.ToString()})

	rec := serve(newAuthRouter(nil, identity, m.RequireAnyPermission(permissions.UserRead)))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAllPermissionsAllowsWhenAllGranted(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(uint, string) (bool, error) {
			return true, nil
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)),
		m.RequireAllPermissions(permissions.UserRead, permissions.LogRead)))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAllPermissionsRejectsWhenOneMissing(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(_ uint, permission string) (bool, error) {
			return permission == permissions.UserRead.String(), nil
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)),
		m.RequireAllPermissions(permissions.UserRead, permissions.LogRead)))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireAnyPermissionFailsClosedWhenAllChecksError(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(uint, string) (bool, error) {
			return false, errors.New("enforcer unavailable")
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)),
		m.RequireAnyPermission(permissions.UserRead, permissions.LogRead)))

	// An infrastructure outage must surface as a 500, not masquerade as an
	// authorization denial.
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRequireAnyPermissionStillAllowsWhenOneErrorsButAnotherGrants(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(_ uint, permission string) (bool, error) {
			if permission == permissions.UserRead.String() {
				return false, errors.New("enforcer hiccup")
			}
			return true, nil
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)),
		m.RequireAnyPermission(permissions.UserRead, permissions.LogRead)))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAnyPermissionRejectsEmptyPermissionList(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(&mockCasbinClient{})

	rec := serve(newAuthRouter(nil, withContextValues(adminValues(3)), m.RequireAnyPermission()))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireAllPermissionsRejectsEmptyPermissionList(t *testing.T) {
	setupLogger(t)
	m := newMiddleware(&mockCasbinClient{})

	// Zero permissions must fail closed: a guard accidentally built from an
	// empty slice should deny, not wave everything through.
	rec := serve(newAuthRouter(nil, withContextValues(adminValues(3)), m.RequireAllPermissions()))

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireAllPermissionsFailsClosedOnCasbinError(t *testing.T) {
	setupLogger(t)
	casbinClient := &mockCasbinClient{
		checkPermissionFn: func(uint, string) (bool, error) {
			return false, errors.New("enforcer unavailable")
		},
	}
	m := newMiddleware(casbinClient)

	rec := serve(newAuthRouter(casbinClient, withContextValues(adminValues(3)),
		m.RequireAllPermissions(permissions.UserRead)))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
