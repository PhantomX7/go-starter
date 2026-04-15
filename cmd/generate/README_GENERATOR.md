# Module Generator Documentation

## Overview

The Module Generator is a powerful tool that automatically creates complete Go module structures with proper case conventions. It intelligently detects your input format and applies the appropriate case conversions for Go naming conventions.

## Features

- **Automatic Case Detection**: Detects camelCase, PascalCase, snake_case, and kebab-case inputs
- **Smart Case Conversion**: Automatically converts to the correct cases for different contexts
- **Complete Module Structure**: Generates controller, service, repository, and DTO files
- **Model Generation**: Optional model file generation with GORM integration
- **DTO Generation**: Optional DTO file generation in the global DTO package
- **Validation**: Validates module names against Go naming conventions and reserved keywords

## Installation

The generator is included in your project. No additional installation required.

## Usage

### Basic Usage

```bash
go run cmd/generate/main.go -name <module_name>
```

### Generate with Model and DTO

```bash
go run cmd/generate/main.go -name <module_name> -model -dto
```

### Show Help

```bash
go run cmd/generate/main.go -help
```

## Command Line Options

| Option | Description |
|--------|-------------|
| `-name string` | Name of the module to generate (required) |
| `-model` | Also generate the corresponding model file |
| `-dto` | Also generate the corresponding DTO file |
| `-help` | Show help message |

## Case Conversions

The generator automatically applies the following case conversions:

| Input Format | Package Name | Struct Names | Variables | Example |
|--------------|--------------|--------------|-----------|---------|
| `productCategory` | `product_category` | `ProductCategory` | `productCategory` | camelCase → snake_case/PascalCase/camelCase |
| `ProductCategory` | `product_category` | `ProductCategory` | `productCategory` | PascalCase → snake_case/camelCase |
| `product_category` | `product_category` | `ProductCategory` | `productCategory` | snake_case → PascalCase/camelCase |
| `product-category` | `product_category` | `ProductCategory` | `productCategory` | kebab-case → snake_case/PascalCase/camelCase |

## Examples

### Example 1: camelCase Input

```bash
go run cmd/generate/main.go -name productCategory -model -dto
```

**Output:**
```
🔍 Detected input format: camelCase

Module 'ProductCategory' generated successfully!
Model 'ProductCategory' generated successfully!
DTO 'ProductCategory' generated successfully!

✅ Generation completed successfully!
📁 Module created at: internal/modules/product_category
📄 Model created at: internal/models/productcategory.go
📋 DTO created at: internal/dto/product_category.go

📝 Case conversions applied:
  • Package name: product_category (snake_case)
  • Struct names: ProductCategory (PascalCase)
  • Variables: productCategory (camelCase)
```

### Example 2: PascalCase Input

```bash
go run cmd/generate/main.go -name UserProfile -model -dto
```

**Output:**
```
🔍 Detected input format: PascalCase

📝 Case conversions applied:
  • Package name: user_profile (snake_case)
  • Struct names: UserProfile (PascalCase)
  • Variables: userProfile (camelCase)
```

### Example 3: snake_case Input

```bash
go run cmd/generate/main.go -name shopping_cart -model -dto
```

**Output:**
```
🔍 Detected input format: snake_case

📝 Case conversions applied:
  • Package name: shopping_cart (snake_case)
  • Struct names: ShoppingCart (PascalCase)
  • Variables: shoppingCart (camelCase)
```

## Generated Structure

When you generate a module with all options, the following structure is created:

```
internal/modules/{snake_case_name}/
├── module.go
├── controller/
│   └── controller.go
├── service/
│   └── service.go
├── repository/
│   └── repository.go
└── dto/
    └── dto.go

internal/models/
└── {lowercase_name}.go

internal/dto/
└── {snake_case_name}.go
```

## Generated Files

### Module File (`internal/modules/{name}/module.go`)

Defines the FX module with all providers:

```go
package {snake_case_name}

import (
    "github.com/PhantomX7/athleton/internal/modules/{snake_case_name}/controller"
    "github.com/PhantomX7/athleton/internal/modules/{snake_case_name}/repository"
    "github.com/PhantomX7/athleton/internal/modules/{snake_case_name}/service"
    "go.uber.org/fx"
)

var Module = fx.Options(
    fx.Provide(
        controller.New{PascalCase}Controller,
        service.New{PascalCase}Service,
        repository.New{PascalCase}Repository,
    ),
)
```

### Controller (`internal/modules/{name}/controller/controller.go`)

RESTful controller with CRUD operations and Swagger documentation.

### Service (`internal/modules/{name}/service/service.go`)

Business logic layer with interface-based design.

### Repository (`internal/modules/{name}/repository/repository.go`)

Data access layer using the generic repository pattern.

### Model (`internal/models/{lowercase}.go`)

GORM model with basic fields and ToResponse method.

### DTO (`internal/dto/{snake_case_name}.go`)

Data Transfer Objects for requests and responses.

## Validation

The generator validates module names to ensure:

- Not empty
- Contains only valid characters (letters, numbers, underscores)
- Starts with a letter
- Not a Go reserved keyword

## Integration Steps

After generating a module:

1. **Add the module to your main application**
   ```go
   import "github.com/PhantomX7/athleton/internal/modules/{name}"
   
   // In your FX application:
   fx.Options(
       // ... other modules
       {name}.Module,
   )
   ```

2. **Register routes in your router**
   ```go
   // Add routes for the new module
   ```

3. **Run database migrations if needed**
   ```bash
   # Create and run migrations for the new model
   ```

## Best Practices

1. **Use descriptive names**: Choose names that clearly describe the module's purpose
2. **Follow Go conventions**: The generator handles this automatically, but be aware of the patterns
3. **Keep modules focused**: Each module should handle a single business domain
4. **Use the DTO option**: Always generate DTOs for proper API contract management
5. **Generate models**: Use the model option for database entities

## Troubleshooting

### Common Issues

1. **Invalid module name**: Ensure the name follows Go naming conventions
2. **Permission errors**: Make sure you have write permissions to the project directories
3. **Go modules not found**: Ensure you're in the correct project directory with go.mod

### Error Messages

- `"module name cannot be empty"`: Provide a valid module name
- `"module name can only contain letters, numbers, and underscores"`: Use valid characters only
- `"module name cannot be a Go reserved keyword"`: Avoid Go keywords like `func`, `var`, etc.

## Advanced Usage

### Custom Templates

The generator uses Go templates defined in `pkg/generator/templates.go`. You can modify these templates to customize the generated code structure.

### Programmatic Usage

You can also use the generator programmatically:

```go
import "github.com/PhantomX7/athleton/pkg/generator"

// Create generators
moduleGen := generator.NewModuleGenerator("internal/modules")
modelGen := generator.NewModelGenerator("internal/models")
dtoGen := generator.NewDTOGenerator("internal/dto")

// Generate files
err := moduleGen.GenerateModule("productCategory")
err = modelGen.GenerateModel("productCategory")
err = dtoGen.GenerateDTO("productCategory")
```

## Contributing

To extend the generator:

1. Modify templates in `pkg/generator/templates.go`
2. Add new generators in `pkg/generator/`
3. Update the CLI in `cmd/generate/main.go`
4. Update this documentation

## License

This generator is part of the athleton project and follows the same license terms.
