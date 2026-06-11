package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// ModelGenerator handles the generation of model files
type ModelGenerator struct {
	modelsPath string
	force      bool
}

// NewModelGenerator creates a new instance of ModelGenerator
func NewModelGenerator(modelsPath string, force bool) *ModelGenerator {
	return &ModelGenerator{
		modelsPath: modelsPath,
		force:      force,
	}
}

// GenerateModel creates a model file for the given module
func (g *ModelGenerator) GenerateModel(moduleName string) error {
	moduleData := prepareModuleData(moduleName)

	modelPath := filepath.Join(g.modelsPath, fmt.Sprintf("%s.go", moduleData.SnakeCase))

	if !g.force {
		if _, err := os.Stat(modelPath); err == nil {
			return fmt.Errorf("model file %s already exists; pass -force to overwrite it", modelPath)
		}
	}

	if err := writeGoFile(modelPath, modelTemplate, moduleData); err != nil {
		return fmt.Errorf("failed to generate model file: %w", err)
	}

	fmt.Printf("Model '%s' generated successfully!\n", moduleData.PascalCase)
	return nil
}

// prepareModuleData is a helper function to prepare module data
func prepareModuleData(moduleName string) ModuleData {
	converter := NewCaseConverter()
	return converter.ConvertModuleData(moduleName)
}

// modelTemplate defines the model file template. It mirrors the shape of the
// existing models (e.g. internal/models/post.go): gorm.Model for ID and
// timestamps, plus a ToResponse mapping to the module's response DTO.
const modelTemplate = `package models

import (
	"github.com/PhantomX7/athleton/internal/dto"

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
