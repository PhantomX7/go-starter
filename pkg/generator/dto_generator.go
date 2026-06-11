package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// DTOGenerator handles the generation of DTO files
type DTOGenerator struct {
	dtoPath string
	force   bool
}

// NewDTOGenerator creates a new instance of DTOGenerator
func NewDTOGenerator(dtoPath string, force bool) *DTOGenerator {
	return &DTOGenerator{
		dtoPath: dtoPath,
		force:   force,
	}
}

// GenerateDTO creates a DTO file for the given module
func (g *DTOGenerator) GenerateDTO(moduleName string) error {
	moduleData := prepareModuleData(moduleName)

	dtoPath := filepath.Join(g.dtoPath, fmt.Sprintf("%s.go", moduleData.SnakeCase))

	if !g.force {
		if _, err := os.Stat(dtoPath); err == nil {
			return fmt.Errorf("DTO file %s already exists; pass -force to overwrite it", dtoPath)
		}
	}

	if err := writeGoFile(dtoPath, dtoGlobalTemplate, moduleData); err != nil {
		return fmt.Errorf("failed to generate DTO file: %w", err)
	}

	fmt.Printf("DTO '%s' generated successfully!\n", moduleData.PascalCase)
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

// {{.PascalCase}}UpdateRequest defines the structure for updating a {{.LowerCase}}.
// Fields are pointers so PATCH can tell "omitted" apart from "set to zero value";
// copier skips nil pointers, so omitted fields keep their current value.
type {{.PascalCase}}UpdateRequest struct {
	Name        *string ` + "`json:\"name\" form:\"name\"`" + `
	Description *string ` + "`json:\"description\" form:\"description\"`" + `
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
