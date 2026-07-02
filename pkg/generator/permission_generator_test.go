package generator

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTestPermissionsRegistry creates a minimal permissions.go in dir,
// mirroring the marker structure of pkg/constants/permissions/permissions.go
// that GeneratePermissions relies on.
func writeTestPermissionsRegistry(t *testing.T, dir string) string {
	t.Helper()

	registryPath := filepath.Join(dir, "permissions.go")
	initial := `// Package permissions defines the permission registry used by authorization checks.
package permissions

// Permission represents a single permission string (format: "resource:action")
type Permission string

func (p Permission) String() string {
	return string(p)
}

// Standard Actions (common across resources)
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionManage = "manage"
)

// Resources
const (
	ResourceUser = "user"
)

// ============================================================================
// USER PERMISSIONS
// ============================================================================
const (
	UserRead Permission = "user:read"
)

// ============================================================================
// PERMISSION REGISTRY
// ============================================================================

// PermissionInfo contains metadata about a permission
type PermissionInfo struct {
	Permission  Permission
	Resource    string
	Action      string
	Description string
}

// AllPermissions maps resources to their available permissions with descriptions
var AllPermissions = map[string][]PermissionInfo{
	ResourceUser: {
		{UserRead, ResourceUser, ActionRead, "View users"},
	},
}
`
	require.NoError(t, os.WriteFile(registryPath, []byte(initial), 0644))
	return registryPath
}

func TestGeneratePermissionsAddsResourceConstsAndRegistryEntry(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	registryPath := writeTestPermissionsRegistry(t, tempDir)

	gen := NewPermissionGenerator(registryPath)
	require.NoError(t, gen.GeneratePermissions("InventoryItem"))

	raw, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	content := string(raw)

	// Resource constant lands inside the Resources const block. gofmt aligns
	// const blocks, so whitespace around "=" is flexible.
	require.Regexp(t, regexp.MustCompile(`ResourceInventoryItem\s*= "inventory_item"`), content)

	// CRUD permission constants exist with the resource:action format.
	require.Regexp(t, regexp.MustCompile(`InventoryItemCreate\s+Permission = "inventory_item:create"`), content)
	require.Regexp(t, regexp.MustCompile(`InventoryItemRead\s+Permission = "inventory_item:read"`), content)
	require.Regexp(t, regexp.MustCompile(`InventoryItemUpdate\s+Permission = "inventory_item:update"`), content)
	require.Regexp(t, regexp.MustCompile(`InventoryItemDelete\s+Permission = "inventory_item:delete"`), content)

	// AllPermissions gains a registry entry so the permissions are assignable.
	require.Contains(t, content, "ResourceInventoryItem: {")
	require.Contains(t, content, `{InventoryItemCreate, ResourceInventoryItem, ActionCreate, "Create inventory items"}`)
	require.Contains(t, content, `{InventoryItemRead, ResourceInventoryItem, ActionRead, "View inventory items"}`)
	require.Contains(t, content, `{InventoryItemUpdate, ResourceInventoryItem, ActionUpdate, "Update inventory items"}`)
	require.Contains(t, content, `{InventoryItemDelete, ResourceInventoryItem, ActionDelete, "Delete inventory items"}`)

	// Existing entries survive untouched.
	require.Regexp(t, regexp.MustCompile(`ResourceUser\s*= "user"`), content)
	require.Contains(t, content, `{UserRead, ResourceUser, ActionRead, "View users"}`)

	// The updated file must still be valid Go.
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, registryPath, raw, parser.AllErrors)
	require.NoError(t, err)
}

func TestGeneratePermissionsIsIdempotent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	registryPath := writeTestPermissionsRegistry(t, tempDir)

	gen := NewPermissionGenerator(registryPath)
	require.NoError(t, gen.GeneratePermissions("InventoryItem"))

	afterFirst, err := os.ReadFile(registryPath)
	require.NoError(t, err)

	// A second run for the same module must not duplicate anything.
	require.NoError(t, gen.GeneratePermissions("inventory_item"))

	afterSecond, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	require.Equal(t, string(afterFirst), string(afterSecond))
	require.Len(t, regexp.MustCompile(`ResourceInventoryItem\s*= "inventory_item"`).FindAllString(string(afterSecond), -1), 1)
	require.Equal(t, 1, strings.Count(string(afterSecond), "ResourceInventoryItem: {"))
}

func TestGeneratePermissionsFailsOnMissingMarkers(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	registryPath := filepath.Join(tempDir, "permissions.go")
	require.NoError(t, os.WriteFile(registryPath, []byte("package permissions\n"), 0644))

	gen := NewPermissionGenerator(registryPath)
	err := gen.GeneratePermissions("InventoryItem")

	require.Error(t, err)
}

func TestGeneratePermissionsFailsOnMissingRegistryFile(t *testing.T) {
	t.Parallel()

	gen := NewPermissionGenerator(filepath.Join(t.TempDir(), "does_not_exist.go"))
	err := gen.GeneratePermissions("InventoryItem")

	require.Error(t, err)
}
