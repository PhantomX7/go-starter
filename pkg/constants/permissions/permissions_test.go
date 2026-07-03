package permissions

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidPermission(t *testing.T) {
	t.Parallel()

	require.True(t, IsValidPermission(UserRead.String()))
	require.True(t, IsValidPermission(AdminRoleCreate.String()))
	require.False(t, IsValidPermission("nope:read"))
	require.False(t, IsValidPermission(""))
	require.False(t, IsValidPermission("user"))
}

// TestDeclaredProductPermissionsAreRegistered — every exported Permission
// constant must be present in AllPermissions: an unregistered constant can be
// used to guard a route but can never be granted to any role, producing
// endpoints only the root bypass can reach.
func TestDeclaredProductPermissionsAreRegistered(t *testing.T) {
	t.Parallel()

	for _, p := range []Permission{
		ProductExport,
		ProductExportAll,
		ProductImport,
		ProductValidateImport,
	} {
		require.True(t, IsValidPermission(p.String()),
			"declared permission %s is not registered in AllPermissions and can never be granted", p)
	}
}

// TestGetResourceActionsReturnsCopy — callers must not be able to mutate the
// internal registry through the returned slice.
func TestGetResourceActionsReturnsCopy(t *testing.T) {
	t.Parallel()

	actions := GetResourceActions(ResourceUser)
	require.NotEmpty(t, actions)

	original := actions[0]
	actions[0] = "mutated"

	require.Equal(t, original, GetResourceActions(ResourceUser)[0],
		"GetResourceActions must return a copy, not the registry's internal slice")
}

func TestGetResourceActionsUnknownResource(t *testing.T) {
	t.Parallel()

	require.Empty(t, GetResourceActions("does_not_exist"))
}

// TestGetAllPermissionsListIsSortedAndComplete — a deterministic order keeps
// seeding and diffing stable across process restarts.
func TestGetAllPermissionsListIsSortedAndComplete(t *testing.T) {
	t.Parallel()

	list := GetAllPermissionsList()
	require.NotEmpty(t, list)
	require.True(t, sort.StringsAreSorted(list), "permission list must be sorted for deterministic output")
	require.Contains(t, list, UserRead.String())
	require.Contains(t, list, LogRead.String())
}

func TestGetPermissionsForFrontendShape(t *testing.T) {
	t.Parallel()

	result := GetPermissionsForFrontend()
	require.Contains(t, result, ResourceUser)

	var found bool
	for _, entry := range result[ResourceUser] {
		if entry["permission"] == UserRead.String() {
			found = true
			require.Equal(t, "read", entry["action"])
			require.NotEmpty(t, entry["description"])
		}
	}
	require.True(t, found, "user:read must be exposed to the frontend")
}
