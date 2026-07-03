package generator

import (
	"fmt"
	"go/format"
	"os"
	"regexp"
	"strings"
)

// PermissionGenerator registers a module's resource and CRUD permissions in
// the permission registry (pkg/constants/permissions/permissions.go) so the
// generated routes compile guarded and the permissions are assignable to
// roles immediately.
type PermissionGenerator struct {
	registryPath string
}

// NewPermissionGenerator creates a PermissionGenerator that updates the
// registry file at registryPath. Generation is idempotent: a resource that is
// already registered is left untouched.
func NewPermissionGenerator(registryPath string) *PermissionGenerator {
	return &PermissionGenerator{registryPath: registryPath}
}

// GeneratePermissions inserts the resource constant, a CRUD permission const
// block, and an AllPermissions entry for the given module.
func (g *PermissionGenerator) GeneratePermissions(moduleName string) error {
	converter := NewCaseConverter()
	data := converter.ConvertModuleData(moduleName)

	content, err := os.ReadFile(g.registryPath)
	if err != nil {
		return fmt.Errorf("read permissions registry: %w", err)
	}

	// Normalize line endings so the \n-based markers match a CRLF checkout.
	updated := strings.ReplaceAll(string(content), "\r\n", "\n")

	resourceConst := "Resource" + data.PascalCase

	// \b guards against prefix collisions (ResourceUser vs ResourceUserProfile).
	alreadyRegistered := regexp.MustCompile(`\b` + resourceConst + `\s*=`).MatchString(updated)
	if alreadyRegistered {
		return nil
	}

	updated, err = insertAfter(updated, "// Resources\nconst (\n",
		fmt.Sprintf("\t%s = %q\n", resourceConst, data.SnakeCase),
		"Resources const block")
	if err != nil {
		return err
	}

	registryBanner := "// ============================================================================\n// PERMISSION REGISTRY"
	updated, err = insertBefore(updated, registryBanner,
		g.permissionConstBlock(data), "permission registry banner")
	if err != nil {
		return err
	}

	updated, err = insertAfter(updated, "var AllPermissions = map[string][]PermissionInfo{\n",
		g.allPermissionsEntry(data), "AllPermissions map")
	if err != nil {
		return err
	}

	formatted, err := format.Source([]byte(updated))
	if err != nil {
		return fmt.Errorf("format permissions registry: %w", err)
	}

	// #nosec G306,G703 -- registryPath points at a source file inside the workspace.
	if err := os.WriteFile(g.registryPath, formatted, 0600); err != nil {
		return fmt.Errorf("write permissions registry: %w", err)
	}

	return nil
}

// permissionConstBlock renders the banner-framed const block declaring the
// module's CRUD permissions, matching the style of the hand-written entries.
func (g *PermissionGenerator) permissionConstBlock(data ModuleData) string {
	banner := strings.ToUpper(strings.ReplaceAll(data.SnakeCase, "_", " "))

	var b strings.Builder
	b.WriteString("// ============================================================================\n")
	fmt.Fprintf(&b, "// %s PERMISSIONS\n", banner)
	b.WriteString("// ============================================================================\n")
	b.WriteString("const (\n")
	for _, action := range crudActions {
		fmt.Fprintf(&b, "\t%s%s Permission = \"%s:%s\"\n", data.PascalCase, action.suffix, data.SnakeCase, action.name)
	}
	b.WriteString(")\n\n")
	return b.String()
}

// allPermissionsEntry renders the AllPermissions map entry that makes the new
// permissions valid and assignable to roles.
func (g *PermissionGenerator) allPermissionsEntry(data ModuleData) string {
	humanPlural := strings.ReplaceAll(data.TableName, "_", " ")

	var b strings.Builder
	fmt.Fprintf(&b, "\tResource%s: {\n", data.PascalCase)
	for _, action := range crudActions {
		fmt.Fprintf(&b, "\t\t{%s%s, Resource%s, Action%s, \"%s %s\"},\n",
			data.PascalCase, action.suffix, data.PascalCase, action.suffix, action.verb, humanPlural)
	}
	b.WriteString("\t},\n")
	return b.String()
}

// crudActions describes the standard permission set generated per resource.
var crudActions = []struct {
	suffix string // constant suffix and Action* constant name
	name   string // permission string action segment
	verb   string // human description verb
}{
	{"Create", "create", "Create"},
	{"Read", "read", "View"},
	{"Update", "update", "Update"},
	{"Delete", "delete", "Delete"},
}

// insertAfter inserts snippet immediately after the first occurrence of marker.
func insertAfter(content, marker, snippet, what string) (string, error) {
	index := strings.Index(content, marker)
	if index == -1 {
		return "", fmt.Errorf("permissions registry: %s not found", what)
	}
	insertAt := index + len(marker)
	return content[:insertAt] + snippet + content[insertAt:], nil
}

// insertBefore inserts snippet immediately before the first occurrence of marker.
func insertBefore(content, marker, snippet, what string) (string, error) {
	index := strings.Index(content, marker)
	if index == -1 {
		return "", fmt.Errorf("permissions registry: %s not found", what)
	}
	return content[:index] + snippet + content[index:], nil
}
