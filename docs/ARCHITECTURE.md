# athleton Project Architecture

## Table of Contents
1. [Overview](#overview)
2. [Project Structure](#project-structure)
3. [Directory Organization](#directory-organization)
4. [Core Components](#core-components)
5. [Design Patterns](#design-patterns)
6. [Technology Stack](#technology-stack)
7. [Data Flow](#data-flow)
8. [Module System](#module-system)

---

## Overview

athleton is a Go-based e-commerce backend API built with clean architecture principles, using Gin framework for HTTP routing, Uber FX for dependency injection, and GORM for database operations. The application follows a modular design pattern where each business domain is encapsulated in its own module.

### Key Features
- Clean architecture with separation of concerns
- Dependency injection using Uber FX
- JWT-based authentication with refresh token rotation
- Role-based access control (RBAC)
- RESTful API design
- Database migrations with Atlas
- S3 integration for media storage
- Full-text search with Bleve
- Comprehensive logging with Zap
- Transaction management
- Swagger API documentation

---

## Project Structure

```
athleton/
├── cmd/                          # Application entry points
│   ├── main.go                  # Main application entry point
│   └── generate/                # Code generation tools
├── database/                    # Database configuration
│   ├── main.go                  # Database setup
│   ├── migrations/              # Database migration files
│   ├── schema/                  # Database schema
│   └── seeder/                  # Data seeding scripts
├── docs/                        # Documentation
│   ├── docs.go                  # Swagger docs entry
│   ├── swagger.json             # Swagger specification
│   └── swagger.yaml             # Swagger YAML format
├── internal/                    # Private application code
│   ├── bootstrap/                # Application initialization
│   ├── dto/                     # Data Transfer Objects
│   ├── middlewares/              # HTTP middleware
│   ├── models/                   # Domain models
│   ├── modules/                  # Business logic modules
│   │   ├── auth/                # Authentication module
│   │   ├── banner/              # Banner management
│   │   ├── blog/                # Blog management
│   │   ├── brand/               # Brand management
│   │   ├── category/            # Category management
│   │   ├── config/              # Configuration management
│   │   ├── cron/                # Scheduled tasks
│   │   ├── media/               # Media handling
│   │   ├── pc_build/            # PC build management
│   │   ├── product/             # Product management
│   │   ├── refresh_token/       # Refresh token management
│   │   ├── search/              # Search functionality
│   │   ├── spec_definition/     # Specification definitions
│   │   ├── spec_definition_value/# Specification values
│   │   ├── statistic/           # Statistics
│   │   └── user/                # User management
│   └── routes/                   # HTTP route definitions
├── libs/                         # Shared libraries
│   ├── bleve/                   # Search engine wrapper
│   ├── s3/                      # S3 storage wrapper
│   └── transaction_manager/     # Transaction management
├── pkg/                          # Public packages
│   ├── config/                  # Configuration management
│   ├── errors/                  # Error handling
│   ├── generator/               # Code generators
│   ├── logger/                  # Logging utilities
│   ├── pagination/              # Pagination utilities
│   ├── repository/              # Base repository interface
│   ├── response/                # Response formatting
│   ├── utils/                   # Utility functions
│   └── validator/               # Custom validators
├── logs/                         # Log files
├── .env.example                 # Environment variables template
├── .gitignore                   # Git ignore rules
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── Makefile                     # Build automation
├── atlas.hcl                    # Atlas configuration
└── README.md                    # Project documentation
```

---

## Directory Organization

### `cmd/`
Contains the application entry points. `main.go` initializes the application, sets up dependencies using Uber FX, and starts the HTTP server.

### `database/`
- **migrations/**: SQL migration files for database schema changes using Atlas
- **schema/**: Current database schema representation
- **seeder/**: Initial data population scripts

### `internal/`
Private application code that cannot be imported by external projects.

#### `bootstrap/`
Application initialization logic:
- Configuration loading
- Logger setup
- Database connection
- Server initialization
- Cron job setup

#### `dto/`
Data Transfer Objects define the structure of HTTP request/response payloads. Each DTO includes validation tags.

#### `middlewares/`
HTTP middleware chain:
- Authentication (JWT validation)
- Authorization (role-based access)
- CORS handling
- Error handling
- Logging
- Request ID tracking
- Timeout management

#### `models/`
Domain models representing database entities with GORM tags. Each model includes timestamps and methods for conversion to DTOs.

#### `modules/`
Business logic organized by domain. Each module follows this structure:
```
module_name/
├── module.go           # FX module definition
├── controller/         # HTTP handlers
├── service/            # Business logic
└── repository/         # Data access layer
```

#### `routes/`
Route definitions organized by public, admin, and authenticated endpoints.

### `libs/`
Shared libraries with external integrations:
- **bleve**: Full-text search engine
- **s3**: AWS S3 or compatible storage
- **transaction_manager**: Database transaction management

### `pkg/`
Reusable packages that can be used across the project:
- **config**: Configuration management
- **errors**: Custom error types and handling
- **generator**: Code generation utilities
- **logger**: Logging abstraction
- **pagination**: Pagination helpers
- **repository**: Base repository interface
- **response**: Standardized API responses
- **utils**: Common utility functions
- **validator**: Custom validation logic

---

## Core Components

### 1. Dependency Injection (Uber FX)
The application uses Uber FX for dependency injection and lifecycle management:
```go
app := fx.New(
    fx.Provide(
        bootstrap.SetUpDatabase,
        middlewares.NewMiddleware,
        validator.New,
        bootstrap.SetupServer,
    ),
    libs.Module,
    auth.Module,
    // ... other modules
    fx.Invoke(
        routes.RegisterRoutes,
        bootstrap.StartServer,
    ),
)
```

### 2. Module System
Each business domain is a self-contained FX module:
- **module.go**: Defines module dependencies
- **Controller**: HTTP request handlers
- **Service**: Business logic
- **Repository**: Data access layer

### 3. Middleware Chain
Request processing follows this pipeline:
1. Request ID generation
2. CORS handling
3. Authentication/Authorization
4. Logging
5. Error handling
6. Timeout management

### 4. Transaction Management
The transaction manager ensures data consistency:
```go
err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
    // Multiple database operations
    return nil
})
```

### 5. Error Handling
Centralized error handling with custom error types:
- `BadRequestError` (400)
- `UnauthorizedError` (401)
- `ForbiddenError` (403)
- `NotFoundError` (404)
- `InternalServerError` (500)

---

## Design Patterns

### 1. Clean Architecture
Separation of concerns with distinct layers:
- **Controller**: HTTP handling
- **Service**: Business logic
- **Repository**: Data access

### 2. Repository Pattern
Abstract data access behind interfaces:
```go
type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    FindById(ctx context.Context, id uint) (*models.User, error)
    // ... other methods
}
```

### 3. Dependency Injection
All dependencies are provided and injected by Uber FX.

### 4. Middleware Pattern
Cross-cutting concerns handled by middleware chain.

### 5. Context Propagation
Request context carries user information, request ID, and other metadata throughout the request lifecycle.

### 6. DTO Pattern
Data Transfer Objects separate API contracts from domain models.

---

## Technology Stack

### Core
- **Go 1.x**: Programming language
- **Gin**: HTTP web framework
- **Uber FX**: Dependency injection
- **GORM**: ORM for database operations

### Database
- **PostgreSQL/MySQL**: Relational database
- **Atlas**: Database migration tool

### Authentication
- **JWT**: JSON Web Tokens (gin-jwt)
- **bcrypt**: Password hashing

### Storage & Search
- **AWS S3/MinIO**: Object storage
- **Bleve**: Full-text search engine

### Documentation
- **Swagger/OpenAPI**: API documentation (swaggo)

### Logging
- **Zap**: Structured logging

### Validation
- **Go validator**: Request validation
- **Custom validators**: Business-specific validation rules

### Utilities
- **UUID**: Unique identifier generation
- **Copier**: Object copying
- **Viper**: Configuration management

---

## Data Flow

### Request Lifecycle

1. **Request Reception**
   - HTTP request arrives at Gin router
   - Request ID middleware assigns unique ID
   - CORS middleware handles cross-origin requests

2. **Authentication**
   - JWT middleware validates token (if required)
   - User context is extracted and stored in request context

3. **Authorization**
   - Role middleware checks user permissions (if required)
   - Access granted or denied based on role

4. **Controller**
   - Request is validated and bound to DTO
   - Controller calls appropriate service method

5. **Service Layer**
   - Business logic is executed
   - Transactions are managed if needed
   - Repository methods are called for data access

6. **Repository Layer**
   - Database queries are executed
   - Data is mapped to domain models

7. **Response**
   - Service returns data to controller
   - Controller formats response using standardized response structure
   - Response is sent to client

### Authentication Flow

1. **Login**
   - Client sends credentials to `/auth/login`
   - gin-jwt middleware validates credentials
   - JWT access token and refresh token are generated
   - Tokens are returned to client

2. **Access Protected Resource**
   - Client includes `Authorization: Bearer <token>` header
   - JWT middleware validates access token
   - User context is set in request
   - Request proceeds to protected endpoint

3. **Token Refresh**
   - Client sends refresh token to `/auth/refresh`
   - Server validates refresh token in database
   - New access and refresh tokens are generated
   - Old refresh token is revoked
   - New tokens are returned

4. **Logout**
   - Client sends refresh token to `/auth/logout`
   - Server revokes refresh token in database
   - Client should discard tokens

---

## Module System

### Module Structure
Each module follows a consistent structure:

```go
// module.go
var Module = fx.Options(
    fx.Provide(
        controller.NewXController,
        service.NewXService,
    ),
)
```

### Module Components

#### Controller
- Handles HTTP requests and responses
- Validates input using DTOs
- Calls service methods
- Formats responses using standardized response builder

#### Service
- Contains business logic
- Coordinates multiple repository calls
- Manages transactions
- Performs data transformations

#### Repository
- Abstracts database operations
- Implements CRUD operations
- Handles complex queries
- Manages database connections

### Module Interactions
Modules interact through:
- Dependency injection
- Shared context values (user ID, role, request ID)
- Transaction manager for cross-module operations
- Event-driven architecture (future enhancement)

---

## Configuration

Configuration is managed through:
- Environment variables (.env file)
- Structured configuration object
- Environment-specific settings (development, production)

### Key Configuration Sections
- App: Application name, environment
- Database: Connection details
- JWT: Secret, expiration times
- S3: Storage credentials
- Server: Port, timeouts

---

## Security Considerations

1. **Authentication**
   - JWT-based stateless authentication
   - Refresh token rotation
   - Token revocation support

2. **Authorization**
   - Role-based access control (RBAC)
   - Middleware for endpoint protection
   - Context-based user information

3. **Password Security**
   - bcrypt hashing with cost factor 12
   - Timing attack prevention
   - Password complexity requirements

4. **Input Validation**
   - Request validation using struct tags
   - Custom validators for business rules
   - SQL injection prevention via GORM

5. **CORS**
   - Configurable CORS middleware
   - SameSite cookie settings

6. **Rate Limiting**
   - Timeout middleware
   - Request size limits

---

## Scalability

### Horizontal Scaling
- Stateless authentication (JWT)
- Session-less design
- External storage (S3)

### Vertical Scaling
- Connection pooling
- Efficient queries
- Caching strategies (future)

### Database Scaling
- Read replicas (future)
- Database sharding (future)
- Connection pool optimization

---

## Monitoring & Observability

### Logging
- Structured logging with Zap
- Request ID tracking
- Error logging with context

### Metrics (Future)
- Request/response times
- Error rates
- Database query performance

### Health Checks (Future)
- Database connectivity
- External service health
- Application status

---

## Testing Strategy

### Unit Tests
- Service layer logic
- Repository mocks
- Utility functions

### Integration Tests
- API endpoints
- Database operations
- External service integration

### E2E Tests
- Complete user flows
- Multi-module interactions

---

## Deployment

### Build Process
```bash
make build
```

### Docker Support
- Containerized application
- Environment-specific configurations
- Volume mounts for logs and data

### CI/CD (Future)
- Automated testing
- Automated deployment
- Rollback capabilities

---

## Future Enhancements

1. **Caching Layer**
   - Redis integration
   - Query result caching
   - Session caching

2. **Message Queue**
   - Async job processing
   - Email notifications
   - Event sourcing

3. **API Gateway**
   - Rate limiting
   - Request transformation
   - Service aggregation

4. **Microservices**
   - Service decomposition
   - Inter-service communication
   - Distributed tracing

5. **GraphQL**
   - GraphQL API support
   - Query optimization
   - Schema stitching

---

## Conclusion

This architecture provides a solid foundation for building scalable, maintainable, and secure e-commerce applications. The modular design allows for easy addition of new features, while the clean architecture principles ensure that the codebase remains organized and testable.

The separation of concerns, dependency injection, and standardized patterns make the application easy to understand and modify, while the use of modern Go libraries ensures high performance and developer productivity.
