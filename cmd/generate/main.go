package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/PhantomX7/go-starter/pkg/generator"
)

func main() {
	var moduleName string
	var generateModel bool
	var help bool

	// Define command line flags
	flag.StringVar(&moduleName, "name", "", "Name of the module to generate (required)")
	flag.BoolVar(&generateModel, "model", false, "Also generate the corresponding model file")
	flag.BoolVar(&help, "help", false, "Show help message")
	flag.Parse()

	// Show help if requested or no module name provided
	if help || moduleName == "" {
		showHelp()
		return
	}

	// Get the project root directory
	projectRoot, err := getProjectRoot()
	if err != nil {
		log.Fatalf("Error finding project root: %v", err)
	}

	// Set up paths
	modulesPath := filepath.Join(projectRoot, "internal", "modules")
	modelsPath := filepath.Join(projectRoot, "internal", "models")

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

	fmt.Println("\n✅ Generation completed successfully!")
	fmt.Printf("📁 Module created at: %s\n", filepath.Join(modulesPath, generator.ToSnakeCase(moduleName)))
	
	if generateModel {
		fmt.Printf("📄 Model created at: %s\n", filepath.Join(modelsPath, fmt.Sprintf("%s.go", generator.ToLowerCase(moduleName))))
	}

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
	fmt.Println("  -help           Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run cmd/generate/main.go -name UserProfile")
	fmt.Println("  go run cmd/generate/main.go -name ProductCategory -model")
	fmt.Println()
	fmt.Println("Naming Conventions:")
	fmt.Println("  - Package names: snake_case (e.g., user_profile)")
	fmt.Println("  - Variable names: camelCase (e.g., userProfile)")
	fmt.Println("  - Type names: PascalCase (e.g., UserProfile)")
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