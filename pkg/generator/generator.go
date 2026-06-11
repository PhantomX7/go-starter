package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ModuleGenerator handles the generation of module templates
type ModuleGenerator struct {
	modulesPath  string
	registryPath string
	force        bool
}

// NewModuleGenerator creates a new instance of ModuleGenerator. When force is
// false, generating over an existing module directory is refused so hand-written
// code can never be silently overwritten.
func NewModuleGenerator(modulesPath string, force bool) *ModuleGenerator {
	return &ModuleGenerator{
		modulesPath:  modulesPath,
		registryPath: filepath.Join(modulesPath, "modules.go"),
		force:        force,
	}
}

// GenerateModule creates a complete module structure with the given name
func (g *ModuleGenerator) GenerateModule(moduleName string) error {
	// Convert module name to different cases
	moduleData := g.prepareModuleData(moduleName)

	basePath := filepath.Join(g.modulesPath, moduleData.SnakeCase)
	if !g.force {
		if _, err := os.Stat(basePath); err == nil {
			return fmt.Errorf("module directory %s already exists; pass -force to overwrite it", basePath)
		}
	}

	// Create module directory structure
	if err := g.createDirectoryStructure(moduleData.SnakeCase); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Generate all module files
	if err := g.generateModuleFiles(moduleData); err != nil {
		return fmt.Errorf("failed to generate module files: %w", err)
	}

	if err := g.addModuleToRegistry(moduleData); err != nil {
		return fmt.Errorf("failed to update module registry: %w", err)
	}

	fmt.Printf("Module '%s' generated successfully!\n", moduleData.PascalCase)
	return nil
}

// ModuleData holds the module name in different cases
type ModuleData struct {
	SnakeCase  string // e.g., "user_profile"
	CamelCase  string // e.g., "userProfile"
	PascalCase string // e.g., "UserProfile"
	LowerCase  string // e.g., "userprofile"
	KebabCase  string // e.g., "user-profile"
	TableName  string // GORM table name, e.g., "user_profiles"
}

// prepareModuleData converts the module name to different cases
func (g *ModuleGenerator) prepareModuleData(moduleName string) ModuleData {
	converter := NewCaseConverter()
	return converter.ConvertModuleData(moduleName)
}

// createDirectoryStructure creates the necessary directories for the module
func (g *ModuleGenerator) createDirectoryStructure(moduleName string) error {
	basePath := filepath.Join(g.modulesPath, moduleName)

	directories := []string{
		basePath,
		filepath.Join(basePath, "controller"),
		filepath.Join(basePath, "service"),
		filepath.Join(basePath, "repository"),
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateModuleFiles generates all the necessary files for the module
func (g *ModuleGenerator) generateModuleFiles(data ModuleData) error {
	basePath := filepath.Join(g.modulesPath, data.SnakeCase)

	// Generate module.go
	if err := g.generateFile(filepath.Join(basePath, "module.go"), moduleTemplate, data); err != nil {
		return err
	}

	// Generate controller
	if err := g.generateFile(filepath.Join(basePath, "controller", "controller.go"), controllerTemplate, data); err != nil {
		return err
	}

	if err := g.generateFile(filepath.Join(basePath, "routes.go"), routesTemplate, data); err != nil {
		return err
	}

	// Generate service
	if err := g.generateFile(filepath.Join(basePath, "service", "service.go"), serviceTemplate, data); err != nil {
		return err
	}

	// Generate repository
	if err := g.generateFile(filepath.Join(basePath, "repository", "repository.go"), repositoryTemplate, data); err != nil {
		return err
	}

	if err := g.generateFile(filepath.Join(basePath, "controller", "controller_test.go"), controllerTestTemplate, data); err != nil {
		return err
	}

	if err := g.generateFile(filepath.Join(basePath, "service", "service_test.go"), serviceTestTemplate, data); err != nil {
		return err
	}

	return g.generateFile(filepath.Join(basePath, "repository", "repository_test.go"), repositoryTestTemplate, data)
}

// generateFile creates a file from a template
func (g *ModuleGenerator) generateFile(filePath string, templateContent string, data ModuleData) error {
	return writeGoFile(filePath, templateContent, data)
}

// writeGoFile renders a template, runs the result through gofmt, and writes it
// to disk. Formatting also acts as a syntax check: a template that renders
// invalid Go fails here instead of at the next build.
func writeGoFile(filePath string, templateContent string, data ModuleData) error {
	tmpl, err := template.New("file").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("generated code for %s is not valid Go: %w", filePath, err)
	}

	// #nosec G304,G306 -- generator writes a caller-selected output file inside the workspace.
	if err := os.WriteFile(filePath, src, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

func (g *ModuleGenerator) addModuleToRegistry(data ModuleData) error {
	content, err := os.ReadFile(g.registryPath)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}

	// The package name always equals the directory name, so a plain import is
	// enough — no alias. The contains check uses just the quoted path so an
	// existing aliased import of the same module also counts as present.
	importPath := fmt.Sprintf("\"github.com/PhantomX7/athleton/internal/modules/%s\"", data.SnakeCase)
	importLine := "\t" + importPath + "\n"
	moduleLine := fmt.Sprintf("\t%s.Module,\n", data.SnakeCase)

	updated := string(content)
	if !strings.Contains(updated, importPath) {
		marker := "import (\n"
		index := strings.Index(updated, marker)
		if index == -1 {
			return fmt.Errorf("registry import block not found")
		}
		insertAt := index + len(marker)
		updated = updated[:insertAt] + importLine + updated[insertAt:]
	}

	if !strings.Contains(updated, moduleLine) {
		marker := "var Module = fx.Options(\n"
		index := strings.Index(updated, marker)
		if index == -1 {
			return fmt.Errorf("registry module block not found")
		}
		insertAt := index + len(marker)
		updated = updated[:insertAt] + moduleLine + updated[insertAt:]
	}

	if updated == string(content) {
		return nil
	}

	formatted, err := format.Source([]byte(updated))
	if err != nil {
		return fmt.Errorf("format registry: %w", err)
	}

	// #nosec G306,G703 -- registryPath is derived from the project's internal/modules directory.
	if err := os.WriteFile(g.registryPath, formatted, 0600); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}
