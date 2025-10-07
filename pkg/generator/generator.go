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
}

// prepareModuleData converts the module name to different cases
func (g *ModuleGenerator) prepareModuleData(moduleName string) ModuleData {
	// Convert to snake_case (package name)
	snakeCase := g.toSnakeCase(moduleName)
	
	// Convert to camelCase (variable names)
	camelCase := g.toCamelCase(moduleName)
	
	// Convert to PascalCase (type names)
	pascalCase := g.toPascalCase(moduleName)
	
	// Convert to lowercase (for some contexts)
	lowerCase := strings.ToLower(strings.ReplaceAll(moduleName, "_", ""))
	
	return ModuleData{
		SnakeCase:  snakeCase,
		CamelCase:  camelCase,
		PascalCase: pascalCase,
		LowerCase:  lowerCase,
	}
}

// createDirectoryStructure creates the necessary directories for the module
func (g *ModuleGenerator) createDirectoryStructure(moduleName string) error {
	basePath := filepath.Join(g.modulesPath, moduleName)
	
	directories := []string{
		basePath,
		filepath.Join(basePath, "controller"),
		filepath.Join(basePath, "service"),
		filepath.Join(basePath, "repository"),
		filepath.Join(basePath, "dto"),
	}
	
	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0755); err != nil {
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
	if err := g.generateFile(filepath.Join(basePath, "repository", "repository.go"), repositoryTemplate, data); err != nil {
		return err
	}
	
	// Generate DTO
	if err := g.generateFile(filepath.Join(basePath, "dto", "dto.go"), dtoTemplate, data); err != nil {
		return err
	}
	
	return nil
}

// generateFile creates a file from a template
func (g *ModuleGenerator) generateFile(filePath string, templateContent string, data ModuleData) error {
	tmpl, err := template.New("file").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()
	
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	
	return nil
}

// Helper functions for case conversion
func (g *ModuleGenerator) toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func (g *ModuleGenerator) toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	if len(words) == 0 {
		return s
	}
	
	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		result += strings.Title(strings.ToLower(words[i]))
	}
	return result
}

func (g *ModuleGenerator) toPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	var result strings.Builder
	for _, word := range words {
		result.WriteString(strings.Title(strings.ToLower(word)))
	}
	return result.String()
}

// Public helper functions for external use
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func ToCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	if len(words) == 0 {
		return s
	}
	
	result := strings.ToLower(words[0])
	for i := 1; i < len(words); i++ {
		result += strings.Title(strings.ToLower(words[i]))
	}
	return result
}

func ToPascalCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	var result strings.Builder
	for _, word := range words {
		result.WriteString(strings.Title(strings.ToLower(word)))
	}
	return result.String()
}

func ToLowerCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}