package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// DTOGenerator handles the generation of DTO files
type DTOGenerator struct {
	dtoPath string
}

// NewDTOGenerator creates a new instance of DTOGenerator
func NewDTOGenerator(dtoPath string) *DTOGenerator {
	return &DTOGenerator{
		dtoPath: dtoPath,
	}
}

// GenerateDTO creates a DTO file for the given module
func (g *DTOGenerator) GenerateDTO(moduleName string) error {
	moduleData := prepareModuleData(moduleName)

	dtoPath := filepath.Join(g.dtoPath, fmt.Sprintf("%s.go", moduleData.SnakeCase))

	if err := g.generateDTOFile(dtoPath, dtoGlobalTemplate, moduleData); err != nil {
		return fmt.Errorf("failed to generate DTO file: %w", err)
	}

	fmt.Printf("DTO '%s' generated successfully!\n", moduleData.PascalCase)
	return nil
}

// generateDTOFile creates a DTO file from a template
func (g *DTOGenerator) generateDTOFile(filePath string, templateContent string, data ModuleData) error {
	tmpl, err := template.New("dto").Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse DTO template: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create DTO file %s: %w", filePath, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute DTO template: %w", err)
	}

	return nil
}

// dtoGlobalTemplate defines the DTO file template for the global dto package
const dtoGlobalTemplate = `package dto

import (
	"time"
)

// {{.PascalCase}}CreateRequest defines the structure for creating a new {{.LowerCase}}
type {{.PascalCase}}CreateRequest struct {
	Name        string ` + "`json:\"name\" form:\"name\" binding:\"required\"`" + `
	Description string ` + "`json:\"description\" form:\"description\"`" + `
}

// {{.PascalCase}}UpdateRequest defines the structure for updating a {{.LowerCase}}
type {{.PascalCase}}UpdateRequest struct {
	Name        string ` + "`json:\"name\" form:\"name\"`" + `
	Description string ` + "`json:\"description\" form:\"description\"`" + `
}

// {{.PascalCase}}Response defines the structure for {{.LowerCase}} response
type {{.PascalCase}}Response struct {
	ID          uint      ` + "`json:\"id\"`" + `
	Name        string    ` + "`json:\"name\"`" + `
	Description string    ` + "`json:\"description\"`" + `
	CreatedAt   time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt   time.Time ` + "`json:\"updated_at\"`" + `
}
`
