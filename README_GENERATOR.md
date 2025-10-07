# Module Generator

This project includes a code generator that creates module templates similar to the existing 'post' module. The generator follows specific naming conventions and creates a complete module structure with controllers, services, and repositories.

## Naming Conventions

The generator follows these naming conventions:
- **snake_case** for package names (e.g., `user_profile`)
- **camelCase** for variable names (e.g., `userProfile`)
- **PascalCase** for class/type names (e.g., `UserProfile`)

## Features

The generator creates:
1. Complete module structure including controllers, services, and repositories
2. Proper import/export statements
3. Basic CRUD operation templates
4. Consistent code style throughout the generated files
5. Optional model generation

## Usage

### Using the CLI Tool

#### Generate a module only:
```bash
go run ./cmd/generate/main.go -name ModuleName
```

#### Generate a module with model:
```bash
go run ./cmd/generate/main.go -name ModuleName -model
```

#### Show help:
```bash
go run ./cmd/generate/main.go -help
```

### Using Makefile Commands

#### Generate a module only:
```bash
make generate-module name=ModuleName
```

#### Generate a module with model:
```bash
make generate-full name=ModuleName
```

#### Show generator help:
```bash
make generate-help
```

## Examples

### Generate a "User" module:
```bash
# Using CLI
go run ./cmd/generate/main.go -name User

# Using Makefile
make generate-module name=User
```

This will create:
- `internal/modules/user/` directory structure
- Controller with CRUD operations
- Service layer with business logic
- Repository interface and implementation
- DTOs for requests and responses
- Module registration file

### Generate a "UserProfile" module with model:
```bash
# Using CLI
go run ./cmd/generate/main.go -name UserProfile -model

# Using Makefile
make generate-full name=UserProfile
```

This will create everything above plus:
- `internal/models/user_profile.go` model file

## Generated Structure

For a module named "User", the generator creates:

```
internal/
├── modules/
│   └── user/
│       ├── module.go
│       ├── controller/
│       │   └── controller.go
│       ├── service/
│       │   └── service.go
│       ├── repository/
│       │   └── repository.go
│       └── dto/
│           └── dto.go
└── models/
    └── user.go (if -model flag is used)
```

## Generated Files Content

### Controller
- Index (GET /users) - List users with pagination
- Create (POST /users) - Create a new user
- Update (PUT /users/:id) - Update an existing user
- Delete (DELETE /users/:id) - Delete a user
- FindById (GET /users/:id) - Get a specific user

### Service
- Business logic layer
- Validation and processing
- Repository interaction
- Pagination handling

### Repository
- Data access layer
- CRUD operations
- Database interaction using generic repository pattern

### DTOs
- Request DTOs for Create and Update operations
- Response DTO for API responses
- Pagination configuration

### Model (optional)
- Database model with GORM tags
- Basic fields (Name, Description, IsActive)
- ToResponse method for DTO conversion

## Customization

After generation, you can customize the generated files:
1. Add specific fields to the model
2. Modify validation rules in DTOs
3. Add business logic to services
4. Customize API endpoints in controllers
5. Add specific database queries to repositories

## Integration

To integrate a generated module into your application:
1. Add the module to your main application's dependency injection
2. Register the routes in your router
3. Run database migrations if you added new models
4. Update API documentation

Example integration in `cmd/main.go`:
```go
fx.Options(
    // ... existing modules
    user.Module, // Add your generated module
),
```