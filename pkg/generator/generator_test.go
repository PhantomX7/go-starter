package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddModuleToRegistryIsIdempotent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	registryPath := filepath.Join(tempDir, "modules.go")
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

	generator := &ModuleGenerator{
		modulesPath:  tempDir,
		registryPath: registryPath,
	}
	data := ModuleData{SnakeCase: "inventory_item"}

	require.NoError(t, generator.addModuleToRegistry(data))
	require.NoError(t, generator.addModuleToRegistry(data))

	updated, err := os.ReadFile(registryPath)
	require.NoError(t, err)

	content := string(updated)
	require.Equal(t, 1, strings.Count(content, "\"github.com/PhantomX7/athleton/internal/modules/inventory_item\""))
	require.Equal(t, 1, strings.Count(content, "inventory_item.Module,"))
}
