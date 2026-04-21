package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// ModuleGenerator handles the generation of module templates
type ModuleGenerator struct {
	modulesPath string
}

// NewModuleGenerator creates a new instance of ModuleGenerator
func NewModuleGenerator(modulesPath string) *ModuleGenerator {
	return &ModuleGenerator{
		modulesPath: modulesPath,
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

	// Generate service
	if err := g.generateFile(filepath.Join(basePath, "service", "service.go"), serviceTemplate, data); err != nil {
		return err
	}

	// Generate repository
	return g.generateFile(filepath.Join(basePath, "repository", "repository.go"), repositoryTemplate, data)
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
