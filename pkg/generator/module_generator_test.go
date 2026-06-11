package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTestRegistry creates a minimal modules.go registry in dir, matching the
// shape GenerateModule expects to update.
func writeTestRegistry(t *testing.T, dir string) string {
	t.Helper()

	registryPath := filepath.Join(dir, "modules.go")
	initial := `package modules

import (
	"github.com/PhantomX7/athleton/internal/modules/auth"

	"go.uber.org/fx"
)

var Module = fx.Options(
	auth.Module,
)
`
	require.NoError(t, os.WriteFile(registryPath, []byte(initial), 0644))
	return registryPath
}

func TestGenerateModuleRefusesExistingDirWithoutForce(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	writeTestRegistry(t, tempDir)

	// Pre-existing module directory must block generation when force is off.
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "blog_post"), 0750))

	gen := NewModuleGenerator(tempDir, false)
	err := gen.GenerateModule("blog_post")

	require.Error(t, err)
	require.ErrorContains(t, err, "already exists")
	require.ErrorContains(t, err, "-force")

	// Nothing should have been generated inside the existing directory.
	entries, readErr := os.ReadDir(filepath.Join(tempDir, "blog_post"))
	require.NoError(t, readErr)
	require.Empty(t, entries)
}

func TestGenerateModuleWithForceOverwritesExistingDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	registryPath := writeTestRegistry(t, tempDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "blog_post"), 0750))

	gen := NewModuleGenerator(tempDir, true)
	require.NoError(t, gen.GenerateModule("blog_post"))

	requireGeneratedModuleFiles(t, filepath.Join(tempDir, "blog_post"))

	registry, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	require.Contains(t, string(registry), `"github.com/PhantomX7/athleton/internal/modules/blog_post"`)
	require.Contains(t, string(registry), "blog_post.Module,")
}

func TestGenerateModuleFreshDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	registryPath := writeTestRegistry(t, tempDir)

	gen := NewModuleGenerator(tempDir, false)
	// PascalCase input exercises the case conversion end to end.
	require.NoError(t, gen.GenerateModule("InventoryItem"))

	moduleDir := filepath.Join(tempDir, "inventory_item")
	requireGeneratedModuleFiles(t, moduleDir)

	// Generated module.go must reference the converted names. writeGoFile runs
	// every file through gofmt, so a successful generation also proves the
	// templates render syntactically valid Go.
	content, err := os.ReadFile(filepath.Join(moduleDir, "module.go"))
	require.NoError(t, err)
	require.Contains(t, string(content), "package inventory_item")

	registry, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(registry), `"github.com/PhantomX7/athleton/internal/modules/inventory_item"`))
	require.Equal(t, 1, strings.Count(string(registry), "inventory_item.Module,"))
}

// requireGeneratedModuleFiles asserts the full generated file layout exists
// and that every file is non-empty.
func requireGeneratedModuleFiles(t *testing.T, moduleDir string) {
	t.Helper()

	files := []string{
		"module.go",
		"routes.go",
		filepath.Join("controller", "controller.go"),
		filepath.Join("controller", "controller_test.go"),
		filepath.Join("service", "service.go"),
		filepath.Join("service", "service_test.go"),
		filepath.Join("repository", "repository.go"),
		filepath.Join("repository", "repository_test.go"),
	}

	for _, file := range files {
		info, err := os.Stat(filepath.Join(moduleDir, file))
		require.NoError(t, err, file)
		require.Greater(t, info.Size(), int64(0), file)
	}

	requireRealGeneratedTests(t, moduleDir)
}

// requireRealGeneratedTests asserts the generated test files contain real test
// functions rather than t.Skip placeholders.
func requireRealGeneratedTests(t *testing.T, moduleDir string) {
	t.Helper()

	testFiles := []string{
		filepath.Join("controller", "controller_test.go"),
		filepath.Join("service", "service_test.go"),
		filepath.Join("repository", "repository_test.go"),
	}

	for _, file := range testFiles {
		raw, err := os.ReadFile(filepath.Join(moduleDir, file))
		require.NoError(t, err, file)
		content := string(raw)

		require.Contains(t, content, "func Test", file)
		require.NotContains(t, content, "t.Skip", file)
		// Real tests exercise behavior through testify assertions.
		require.Contains(t, content, "require.", file)
	}
}
