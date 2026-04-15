package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/PhantomX7/athleton/pkg/generator"
)

func main() {
	var moduleName string
	var generateModel bool
	var generateDTO bool
	var help bool

	// Define command line flags
	flag.StringVar(&moduleName, "name", "", "Name of the module to generate (required)")
	flag.BoolVar(&generateModel, "model", false, "Also generate the corresponding model file")
	flag.BoolVar(&generateDTO, "dto", false, "Also generate the corresponding DTO file")
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

	// Generate the module
	moduleGen := generator.NewModuleGenerator(modulesPath)
	if err := moduleGen.GenerateModule(moduleName); err != nil {
		log.Fatalf("Error generating module: %v", err)
	}

	// Generate the model if requested
	if generateModel {
		modelGen := generator.NewModelGenerator(modelsPath)
		if err := modelGen.GenerateModel(moduleName); err != nil {
			log.Fatalf("Error generating model: %v", err)
		}
	}

	// Generate the DTO if requested
	if generateDTO {
		dtoGen := generator.NewDTOGenerator(dtoPath)
		if err := dtoGen.GenerateDTO(moduleName); err != nil {
			log.Fatalf("Error generating DTO: %v", err)
		}
	}

	fmt.Println("\n✅ Generation completed successfully!")

	// Get converted module data for display
	moduleData := converter.ConvertModuleData(moduleName)
	fmt.Printf("�� Module created at: %s\n", filepath.Join(modulesPath, moduleData.SnakeCase))

	if generateModel {
		fmt.Printf("�� Model created at: %s\n", filepath.Join(modelsPath, fmt.Sprintf("%s.go", moduleData.LowerCase)))
	}

	if generateDTO {
		fmt.Printf("📋 DTO created at: %s\n", filepath.Join(dtoPath, fmt.Sprintf("%s.go", moduleData.SnakeCase)))
	}

	fmt.Println("\n📝 Case conversions applied:")
	fmt.Printf("  • Package name: %s (snake_case)\n", moduleData.SnakeCase)
	fmt.Printf("  • Struct names: %s (PascalCase)\n", moduleData.PascalCase)
	fmt.Printf("  • Variables: %s (camelCase)\n", moduleData.CamelCase)

	fmt.Println("\n📝 Next steps:")
	fmt.Println("1. Add the module to your main application")
	fmt.Println("2. Register routes in your router")
	if generateModel {
		fmt.Println("3. Run database migrations if needed")
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
	fmt.Println("  -model          Also generate the corresponding model file")
	fmt.Println("  -dto            Also generate the corresponding DTO file")
	fmt.Println("  -help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/generate/main.go -name UserProfile")
	fmt.Println("  go run cmd/generate/main.go -name ProductCategory -model -dto")
	fmt.Println("  go run cmd/generate/main.go -name productCategory -model -dto")
	fmt.Println()
	fmt.Println("Naming Conventions:")
	fmt.Println("  - Package names: snake_case (e.g., user_profile)")
	fmt.Println("  - Variable names: camelCase (e.g., userProfile)")
	fmt.Println("  - Type names: PascalCase (e.g., UserProfile)")
	fmt.Println()
	fmt.Println("The generator automatically detects your input format and applies")
	fmt.Println("the appropriate case conversions for Go naming conventions.")
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
