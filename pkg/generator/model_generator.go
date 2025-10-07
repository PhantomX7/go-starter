package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// ModelGenerator handles the generation of model files
type ModelGenerator struct {
	modelsPath string
}

// NewModelGenerator creates a new instance of ModelGenerator
func NewModelGenerator(modelsPath string) *ModelGenerator {
	return &ModelGenerator{
		modelsPath: modelsPath,
	}
}

// GenerateModel creates a model file for the given module
func (g *ModelGenerator) GenerateModel(moduleName string) error {
	moduleData := prepareModuleData(moduleName)
	
	modelPath := filepath.Join(g.modelsPath, fmt.Sprintf("%s.go", moduleData.LowerCase))
	
	if err := g.generateModelFile(modelPath, modelTemplate, moduleData); err != nil {
		return fmt.Errorf("failed to generate model file: %w", err)
	}
	
	fmt.Printf("Model '%s' generated successfully!\n", moduleData.PascalCase)
	return nil
}

// generateModelFile creates a model file from a template
func (g *ModelGenerator) generateModelFile(filePath string, templateContent string, data ModuleData) error {
	tmpl, err := template.New("model").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse model template: %w", err)
	}
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create model file %s: %w", filePath, err)
	}
	defer file.Close()
	
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute model template: %w", err)
	}
	
	return nil
}

// prepareModuleData is a helper function to prepare module data
func prepareModuleData(moduleName string) ModuleData {
	gen := &ModuleGenerator{}
	return gen.prepareModuleData(moduleName)
}

// Helper function to convert module name to different cases (duplicated for standalone use)
func (g *ModelGenerator) prepareModuleData(moduleName string) ModuleData {
	gen := &ModuleGenerator{}
	return gen.prepareModuleData(moduleName)
}

// modelTemplate defines the model file template
const modelTemplate = `package models

import (
	"github.com/PhantomX7/go-starter/internal/modules/{{.SnakeCase}}/dto"
	"gorm.io/gorm"
)

// {{.PascalCase}} represents the {{.LowerCase}} entity
type {{.PascalCase}} struct {
	gorm.Model

	Name        string ` + "`gorm:\"type:varchar(255);not null\" json:\"name\"`" + `
	Description string ` + "`gorm:\"type:text\" json:\"description\"`" + `
	IsActive    bool   ` + "`gorm:\"default:true\" json:\"is_active\"`" + `
}

// ToResponse converts the {{.PascalCase}} model to a response DTO
func (m {{.PascalCase}}) ToResponse() any {
	return dto.{{.PascalCase}}Response{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
`