// Package main provides the Athleton module generator CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/PhantomX7/athleton/pkg/generator"
)

func main() {
	var moduleName string
	var generateModel bool
	var generateDTO bool
	var generatePermissions bool
	var force bool
	var help bool

	// Define command line flags. Model, DTO, and permission generation default
	// to on because the module templates reference dto.<X>CreateRequest,
	// models.<X>, and permissions.<X>Read — a module generated without them
	// does not compile.
	flag.StringVar(&moduleName, "name", "", "Name of the module to generate (required)")
	flag.BoolVar(&generateModel, "model", true, "Also generate the corresponding model file")
	flag.BoolVar(&generateDTO, "dto", true, "Also generate the corresponding DTO file")
	flag.BoolVar(&generatePermissions, "permissions", true, "Also register CRUD permissions for the module")
	flag.BoolVar(&force, "force", false, "Overwrite existing module/model/DTO files")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.Parse()

	// Show help if requested or no module name provided
	if help || moduleName == "" {
		showHelp()
		return
	}

	// Validate the module name
	converter := generator.NewCaseConverter()
	if err := converter.ValidateModuleName(moduleName); err != nil {
		log.Fatalf("Invalid module name: %v", err)
	}

	// Detect and display input format
	inputFormat := converter.DetectInputFormat(moduleName)
	fmt.Printf("🔍 Detected input format: %s\n", inputFormat)

	// Get the project root directory
	projectRoot, err := getProjectRoot()
	if err != nil {
		log.Fatalf("Error finding project root: %v", err)
	}

	// Set up paths
	modulesPath := filepath.Join(projectRoot, "internal", "modules")
	modelsPath := filepath.Join(projectRoot, "internal", "models")
	dtoPath := filepath.Join(projectRoot, "internal", "dto")
	permissionsPath := filepath.Join(projectRoot, "pkg", "constants", "permissions", "permissions.go")

	// Generate the module
	moduleGen := generator.NewModuleGenerator(modulesPath, force)
	if err := moduleGen.GenerateModule(moduleName); err != nil {
		log.Fatalf("Error generating module: %v", err)
	}

	// Register CRUD permissions so the guarded routes in the generated
	// registrar compile and the permissions are assignable to roles.
	if generatePermissions {
		permGen := generator.NewPermissionGenerator(permissionsPath)
		if err := permGen.GeneratePermissions(moduleName); err != nil {
			log.Fatalf("Error registering permissions: %v", err)
		}
	}

	// Generate the model if requested
	if generateModel {
		modelGen := generator.NewModelGenerator(modelsPath, force)
		if err := modelGen.GenerateModel(moduleName); err != nil {
			log.Fatalf("Error generating model: %v", err)
		}
	}

	// Generate the DTO if requested
	if generateDTO {
		dtoGen := generator.NewDTOGenerator(dtoPath, force)
		if err := dtoGen.GenerateDTO(moduleName); err != nil {
			log.Fatalf("Error generating DTO: %v", err)
		}
	}

	// The generated controller references typed field helpers from
	// internal/generated, which only exist after the GORM CLI runs over the new
	// model. Refresh them now so the project compiles immediately.
	if generateModel {
		fmt.Println("Refreshing GORM field helpers (go generate ./internal/models/...)")
		cmd := exec.CommandContext(context.Background(), "go", "generate", "./internal/models/...")
		cmd.Dir = projectRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("WARNING: failed to refresh field helpers: %v", err)
			log.Printf("Run `make gorm-gen` manually — the generated controller will not compile until you do.")
		}
	}

	fmt.Println("\n✅ Generation completed successfully!")

	// Get converted module data for display
	moduleData := converter.ConvertModuleData(moduleName)
	fmt.Printf("📦 Module created at: %s\n", filepath.Join(modulesPath, moduleData.SnakeCase))

	if generateModel {
		fmt.Printf("🗃️ Model created at: %s\n", filepath.Join(modelsPath, fmt.Sprintf("%s.go", moduleData.SnakeCase)))
	}

	if generateDTO {
		fmt.Printf("📋 DTO created at: %s\n", filepath.Join(dtoPath, fmt.Sprintf("%s.go", moduleData.SnakeCase)))
	}

	if generatePermissions {
		fmt.Printf("🔐 CRUD permissions registered in: %s (%s:create/read/update/delete)\n", permissionsPath, moduleData.SnakeCase)
	}

	fmt.Println("\n📝 Case conversions applied:")
	fmt.Printf("  • Package name: %s (snake_case)\n", moduleData.SnakeCase)
	fmt.Printf("  • Struct names: %s (PascalCase)\n", moduleData.PascalCase)
	fmt.Printf("  • Variables: %s (camelCase)\n", moduleData.CamelCase)

	fmt.Println("\n📝 Next steps:")
	fmt.Println("1. Fill in the generated service, repository, and controller logic")
	fmt.Println("2. Adjust the generated route registrar if the feature is not standard admin CRUD")
	if generatePermissions {
		fmt.Println("3. Assign the new permissions to admin roles (root bypasses permission checks)")
	}
	if generateModel {
		fmt.Println("4. Run database migrations if needed")
	}
}

// showHelp displays the help message
func showHelp() {
	fmt.Println("Module Generator - Generate Go module templates")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/generate/main.go -name <module_name> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -name string    Name of the module to generate (required)")
	fmt.Println("  -model          Also generate the corresponding model file (default true)")
	fmt.Println("  -dto            Also generate the corresponding DTO file (default true)")
	fmt.Println("  -permissions    Also register CRUD permissions for the module (default true;")
	fmt.Println("                  with -permissions=false the generated routes will not compile")
	fmt.Println("                  until you register or remove the permission guards yourself)")
	fmt.Println("  -force          Overwrite existing module/model/DTO files")
	fmt.Println("  -help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/generate/main.go -name UserProfile")
	fmt.Println("  go run cmd/generate/main.go -name ProductCategory")
	fmt.Println("  go run cmd/generate/main.go -name productCategory -model=false -dto=false")
	fmt.Println()
	fmt.Println("Naming Conventions:")
	fmt.Println("  - Package names: snake_case (e.g., user_profile)")
	fmt.Println("  - Variable names: camelCase (e.g., userProfile)")
	fmt.Println("  - Type names: PascalCase (e.g., UserProfile)")
	fmt.Println()
	fmt.Println("The generator automatically detects your input format and applies")
	fmt.Println("the appropriate case conversions for Go naming conventions.")
	fmt.Println("New modules are added to internal/modules/modules.go automatically.")
}

// getProjectRoot finds the project root directory by looking for go.mod
func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod file in any parent directory")
		}
		dir = parent
	}
}
