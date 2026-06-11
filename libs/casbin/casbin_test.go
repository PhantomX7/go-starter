package casbin_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	libcasbin "github.com/PhantomX7/athleton/libs/casbin"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	return db
}

func newClient(t *testing.T) libcasbin.Client {
	t.Helper()

	c, err := libcasbin.New(setupDB(t))
	require.NoError(t, err)
	require.NotNil(t, c)

	return c
}

func TestNewBuildsClientWithEnforcer(t *testing.T) {
	c := newClient(t)
	require.NotNil(t, c.GetEnforcer())
}

func TestAddRolePermissionsAndCheckExactMatch(t *testing.T) {
	c := newClient(t)

	require.NoError(t, c.AddRolePermissions(1, []string{"post:create", "post:read"}))

	allowed, err := c.CheckPermission(1, "post:create")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = c.CheckPermission(1, "post:read")
	require.NoError(t, err)
	require.True(t, allowed)

	// Action the role does not have.
	allowed, err = c.CheckPermission(1, "post:delete")
	require.NoError(t, err)
	require.False(t, allowed)

	// Resource the role does not have.
	allowed, err = c.CheckPermission(1, "user:create")
	require.NoError(t, err)
	require.False(t, allowed)

	// Another role has no permissions at all.
	allowed, err = c.CheckPermission(2, "post:create")
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestCheckPermissionManageWildcard(t *testing.T) {
	c := newClient(t)

	require.NoError(t, c.AddRolePermissions(7, []string{"product:manage"}))

	for _, action := range []string{"create", "read", "update", "delete", "manage"} {
		allowed, err := c.CheckPermission(7, "product:"+action)
		require.NoError(t, err)
		require.True(t, allowed, "manage should grant product:%s", action)
	}

	// Manage on one resource does not leak to another resource.
	allowed, err := c.CheckPermission(7, "order:create")
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestRemoveRolePermissions(t *testing.T) {
	c := newClient(t)

	require.NoError(t, c.AddRolePermissions(3, []string{"post:create", "post:delete"}))
	require.NoError(t, c.RemoveRolePermissions(3, []string{"post:create"}))

	allowed, err := c.CheckPermission(3, "post:create")
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = c.CheckPermission(3, "post:delete")
	require.NoError(t, err)
	require.True(t, allowed)

	require.Equal(t, []string{"post:delete"}, c.GetRolePermissions(3))
}

func TestSetRolePermissionsReplacesExisting(t *testing.T) {
	c := newClient(t)

	require.NoError(t, c.AddRolePermissions(4, []string{"post:create", "post:read"}))
	require.NoError(t, c.SetRolePermissions(4, []string{"user:update"}))

	require.Equal(t, []string{"user:update"}, c.GetRolePermissions(4))

	allowed, err := c.CheckPermission(4, "post:create")
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = c.CheckPermission(4, "user:update")
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestSetRolePermissionsWithEmptyListClearsRole(t *testing.T) {
	c := newClient(t)

	require.NoError(t, c.AddRolePermissions(5, []string{"post:create"}))
	require.NoError(t, c.SetRolePermissions(5, nil))

	require.Empty(t, c.GetRolePermissions(5))
}

func TestGetRolePermissionsListsAllInResourceActionFormat(t *testing.T) {
	c := newClient(t)

	perms := []string{"post:create", "post:read", "user:manage"}
	require.NoError(t, c.AddRolePermissions(6, perms))

	require.ElementsMatch(t, perms, c.GetRolePermissions(6))
	require.Empty(t, c.GetRolePermissions(99))
}

func TestDeleteRoleRemovesAllPermissions(t *testing.T) {
	c := newClient(t)

	require.NoError(t, c.AddRolePermissions(8, []string{"post:create", "user:read"}))
	require.NoError(t, c.AddRolePermissions(9, []string{"post:create"}))

	require.NoError(t, c.DeleteRole(8))

	require.Empty(t, c.GetRolePermissions(8))
	// Other roles are untouched.
	require.Equal(t, []string{"post:create"}, c.GetRolePermissions(9))
}

func TestInvalidPermissionFormatErrors(t *testing.T) {
	c := newClient(t)

	invalid := []string{"noseparator", ":action", "resource:", ""}

	for _, perm := range invalid {
		require.Error(t, c.AddRolePermissions(1, []string{perm}), "add %q", perm)
		require.Error(t, c.RemoveRolePermissions(1, []string{perm}), "remove %q", perm)

		allowed, err := c.CheckPermission(1, perm)
		require.Error(t, err, "check %q", perm)
		require.False(t, allowed)
	}

	// "resource:action:extra" splits on the first colon only and is accepted.
	require.NoError(t, c.AddRolePermissions(1, []string{"post:sub:action"}))
	allowed, err := c.CheckPermission(1, "post:sub:action")
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestCheckPermissionWithRoot(t *testing.T) {
	c := newClient(t)

	roleID := uint(10)
	require.NoError(t, c.AddRolePermissions(roleID, []string{"post:create"}))

	// Root bypasses everything, even unknown permissions.
	allowed, err := c.CheckPermissionWithRoot("root", nil, "anything:at-all")
	require.NoError(t, err)
	require.True(t, allowed)

	// Non-admin roles are denied.
	allowed, err = c.CheckPermissionWithRoot("user", &roleID, "post:create")
	require.NoError(t, err)
	require.False(t, allowed)

	// Admin without an admin role is denied.
	allowed, err = c.CheckPermissionWithRoot("admin", nil, "post:create")
	require.NoError(t, err)
	require.False(t, allowed)

	// Admin with a role delegates to CheckPermission.
	allowed, err = c.CheckPermissionWithRoot("admin", &roleID, "post:create")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = c.CheckPermissionWithRoot("admin", &roleID, "post:delete")
	require.NoError(t, err)
	require.False(t, allowed)
}

func TestPoliciesPersistAcrossClients(t *testing.T) {
	db := setupDB(t)

	first, err := libcasbin.New(db)
	require.NoError(t, err)
	require.NoError(t, first.AddRolePermissions(11, []string{"post:create"}))

	// A fresh client over the same DB loads the persisted policies.
	second, err := libcasbin.New(db)
	require.NoError(t, err)

	allowed, err := second.CheckPermission(11, "post:create")
	require.NoError(t, err)
	require.True(t, allowed)
	require.Equal(t, []string{"post:create"}, second.GetRolePermissions(11))
}

func TestParseRoleIDFromSubject(t *testing.T) {
	id, err := libcasbin.ParseRoleIDFromSubject("role:42")
	require.NoError(t, err)
	require.Equal(t, uint(42), id)

	_, err = libcasbin.ParseRoleIDFromSubject("user:42")
	require.Error(t, err)

	_, err = libcasbin.ParseRoleIDFromSubject("role:not-a-number")
	require.Error(t, err)

	_, err = libcasbin.ParseRoleIDFromSubject("42")
	require.Error(t, err)
}
