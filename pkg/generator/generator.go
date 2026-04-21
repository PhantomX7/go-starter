package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ModuleGenerator handles the generation of module templates
type ModuleGenerator struct {
	modulesPath  string
	registryPath string
}

// NewModuleGenerator creates a new instance of ModuleGenerator
func NewModuleGenerator(modulesPath string) *ModuleGenerator {
	return &ModuleGenerator{
		modulesPath:  modulesPath,
		registryPath: filepath.Join(modulesPath, "modules.go"),
	}
}

// GenerateModule creates a complete module structure with the given name
func (g *ModuleGenerator) GenerateModule(moduleName string) error {
	// Convert module name to different cases
	moduleData := g.prepareModuleData(moduleName)

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
	tmpl, err := template.New("file").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// #nosec G304 -- generator writes a caller-selected output file inside the workspace.
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func (g *ModuleGenerator) addModuleToRegistry(data ModuleData) error {
	content, err := os.ReadFile(g.registryPath)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}

	importLine := fmt.Sprintf("\t%s \"github.com/PhantomX7/athleton/internal/modules/%s\"\n", data.SnakeCase, data.SnakeCase)
	moduleLine := fmt.Sprintf("\t%s.Module,\n", data.SnakeCase)

	updated := string(content)
	if !strings.Contains(updated, importLine) {
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

	// #nosec G306,G703 -- registryPath is derived from the project's internal/modules directory.
	if err := os.WriteFile(g.registryPath, []byte(updated), 0600); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}

	return nil
}
