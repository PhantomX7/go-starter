# Authentication Implementation Guide

## Table of Contents
1. [Overview](#overview)
2. [Authentication Architecture](#authentication-architecture)
3. [User Roles & Permissions](#user-roles--permissions)
4. [Authentication Flow](#authentication-flow)
5. [JWT Token System](#jwt-token-system)
6. [API Endpoints](#api-endpoints)
7. [Middleware Implementation](#middleware-implementation)
8. [Security Features](#security-features)
9. [Database Schema](#database-schema)
10. [Code Examples](#code-examples)
11. [Best Practices](#best-practices)
12. [Troubleshooting](#troubleshooting)

---

## Overview

The authentication system in athleton provides secure user authentication using JSON Web Tokens (JWT) with refresh token rotation, role-based access control (RBAC), and comprehensive security measures.

### Key Features
- **JWT-based Authentication**: Stateless token-based authentication
- **Refresh Token Rotation**: Automatic refresh token rotation on each use
- **Role-Based Access Control**: Multiple user roles with fine-grained permissions
- **Password Security**: bcrypt hashing with timing attack prevention
- **Token Revocation**: Support for token revocation on logout and password changes
- **Session Management**: Database-backed refresh token tracking
- **Multi-Device Support**: Users can have multiple active sessions

### Technologies Used
- **gin-jwt**: JWT middleware for Gin framework
- **golang-jwt/jwt**: JWT implementation for Go
- **bcrypt**: Password hashing
- **UUID**: Unique token generation
- **GORM**: Database operations for refresh tokens

---

## Authentication Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     HTTP Request                              │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              Middleware Chain                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Request ID │  │     CORS     │  │   Timeout    │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Auth JWT   │  │    Role      │  │   Logger     │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   Controller Layer                            │
│  - Request validation (DTOs)                                  │
│  - Business logic coordination                                │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Service Layer                              │
│  - Authentication logic                                      │
│  - Token generation & validation                              │
│  - Password management                                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                  Repository Layer                              │
│  - User data access                                           │
│  - Refresh token management                                   │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Database                                   │
│  - users table                                                │
│  - refresh_tokens table                                       │
└─────────────────────────────────────────────────────────────┘
```

### Module Structure

```
internal/modules/auth/
├── module.go                    # FX module definition
├── controller/
│   └── controller.go            # HTTP handlers
│       - Register()
│       - GetMe()
│       - Refresh()
│       - ChangePassword()
│       - Logout()
├── service/
│   └── service.go               # Business logic
│       - Register()
│       - GetMe()
│       - Refresh()
│       - ChangePassword()
│       - Logout()
└── jwt/
    └── jwt.go                   # JWT middleware
        - Middleware configuration
        - Token generation
        - Token validation
        - Refresh token management
```

---

## User Roles & Permissions

### Available Roles

| Role | Description | Access Level |
|------|-------------|--------------|
| `root` | Super administrator | Full system access |
| `admin` | Administrator | Administrative access |
| `writer` | Content creator | Blog and media access |
| `reseller` | Reseller partner | Limited access (future) |
| `user` | Regular user | Public access only |

### Role-Based Route Access

#### Admin & Root Routes
- `/api/v1/admin/banner/*` - Banner management
- `/api/v1/admin/brand/*` - Brand management
- `/api/v1/admin/category/*` - Category management
- `/api/v1/admin/config/*` - Configuration management
- `/api/v1/admin/product/*` - Product management
- `/api/v1/admin/spec-definition/*` - Specification management
- `/api/v1/admin/statistic/*` - Statistics
- `/api/v1/admin/user/*` - User management

#### Admin, Root, and Writer Routes
- `/api/v1/admin/blog/*` - Blog management
- `/api/v1/admin/media/upload` - Media upload

#### Public Routes (Optional Auth)
- `/api/v1/public/product/*` - Product browsing
- `/api/v1/public/search/*` - Search functionality

#### Public Routes (No Auth)
- `/api/v1/public/banner/*` - Banner viewing
- `/api/v1/public/blog/*` - Blog reading
- `/api/v1/public/brand/*` - Brand viewing
- `/api/v1/public/category/*` - Category viewing
- `/api/v1/public/config/*` - Configuration viewing
- `/api/v1/public/pc-build/*` - PC build configuration

### Role Definition

```go
type UserRole string

const (
    UserRoleUser     UserRole = "user"
    UserRoleAdmin    UserRole = "admin"
    UserRoleWriter   UserRole = "writer"
    UserRoleRoot     UserRole = "root"
    UserRoleReseller UserRole = "reseller"
)

func (u UserRole) ToString() string {
    return string(u)
}
```

---

## Authentication Flow

### 1. User Registration

**Endpoint:** `POST /api/v1/auth/register`

**Request:**
```json
{
  "name": "John Doe",
  "business_name": "Tech Store",
  "email": "john@example.com",
  "phone": "+6281234567890",
  "password": "securePassword123"
}
```

**Flow:**
1. Controller validates request using `RegisterRequest` DTO
2. Service normalizes input (lowercase email, trim whitespace)
3. Password is hashed using bcrypt (cost factor: 12)
4. User is created in database with `role: "user"` and `is_active: true`
5. Access token and refresh token are generated
6. Tokens are returned to client

**Response:**
```json
{
  "message": "register success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "550e8400-e29b-41d4-a716-446655440000-...",
    "token_type": "Bearer"
  }
}
```

### 2. User Login

**Endpoint:** `POST /api/v1/auth/login`

**Request:**
```json
{
  "username": "john@example.com",
  "password": "securePassword123"
}
```

**Flow:**
1. gin-jwt middleware intercepts request
2. `authenticator()` callback validates credentials
3. User is found by email or username
4. Password is verified using bcrypt
5. User must be active (`is_active: true`)
6. Access token is generated by gin-jwt
7. Refresh token is generated and stored in database
8. Both tokens are returned to client

**Response:**
```json
{
  "message": "login success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "550e8400-e29b-41d4-a716-446655440000-...",
    "token_type": "Bearer"
  }
}
```

### 3. Access Protected Resource

**Endpoint:** `GET /api/v1/auth/me` (example)

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Flow:**
1. `RequireAuth()` middleware validates access token
2. Token is extracted from `Authorization` header
3. Token signature is verified using JWT secret
4. User ID and role are extracted from token claims
5. `authorizer()` callback validates:
   - User exists in database
   - User is active
   - User has at least one valid refresh token
6. User context is set in request context
7. Request proceeds to controller
8. Controller retrieves user from context
9. User data is returned

**Response:**
```json
{
  "message": "get me success",
  "data": {
    "id": 1,
    "name": "John Doe",
    "business_name": "Tech Store",
    "username": "john@example.com",
    "email": "john@example.com",
    "phone": "+6281234567890",
    "is_active": true,
    "role": "user",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

### 4. Token Refresh

**Endpoint:** `POST /api/v1/auth/refresh`

**Request:**
```json
{
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000-..."
}
```

**Flow:**
1. Controller validates request
2. Service finds refresh token in database
3. Validates token is not expired and not revoked
4. Retrieves user from database
5. Validates user is active
6. Old refresh token is revoked
7. New access token and refresh token are generated
8. New tokens are returned to client

**Response:**
```json
{
  "message": "refresh success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "a7e8f001-f29b-41d4-a716-446655440001-...",
    "token_type": "Bearer"
  }
}
```

### 5. Password Change

**Endpoint:** `POST /api/v1/auth/change-password`

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Request:**
```json
{
  "old_password": "securePassword123",
  "new_password": "newSecurePassword456",
  "except_token": "550e8400-e29b-41d4-a716-446655440000-..."
}
```

**Flow:**
1. Middleware validates access token and sets user context
2. Controller validates request
3. Service retrieves user from context
4. Old password is verified using bcrypt
5. New password is hashed using bcrypt
6. Password is updated in database
7. All user's refresh tokens are revoked EXCEPT the one specified
8. Success response is returned

**Response:**
```json
{
  "message": "password changed successfully",
  "data": null
}
```

### 6. Logout

**Endpoint:** `POST /api/v1/auth/logout`

**Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Request:**
```json
{
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000-..."
}
```

**Flow:**
1. Middleware validates access token and sets user context
2. Controller validates request
3. Service verifies refresh token belongs to user
4. Refresh token is revoked in database
5. Success response is returned
6. Client should discard both access and refresh tokens

**Response:**
```json
{
  "message": "logout successful",
  "data": null
}
```

---

## JWT Token System

### Access Token

**Purpose:** Short-lived token for API authentication

**Configuration:**
```go
Timeout: cfg.JWT.Expiration  // Typically 15-30 minutes
SigningAlgorithm: "HS256"
```

**Payload:**
```json
{
  "user_id": 123,
  "role": "admin",
  "sub": 123,
  "iss": "athleton",
  "exp": 1704067200,
  "iat": 1704063600
}
```

**Usage:**
- Sent in `Authorization: Bearer <token>` header
- Validated by `RequireAuth()` middleware
- Stateless - no database lookup required for validation

### Refresh Token

**Purpose:** Long-lived token for obtaining new access tokens

**Configuration:**
```go
MaxRefresh: cfg.JWT.RefreshExpiration  // Typically 7-30 days
```

**Format:** UUID-based string with hyphen separator
```
550e8400-e29b-41d4-a716-446655440000-550e8400-e29b-41d4-a716-446655440001
```

**Storage:** Database table `refresh_tokens`

**Usage:**
- Stored securely on client (e.g., httpOnly cookie or secure storage)
- Sent to `/auth/refresh` endpoint to get new tokens
- Validated against database on each use
- Revoked on logout and password change

### Token Rotation

Refresh token rotation is implemented to enhance security:

1. **On Refresh:** Each time a refresh token is used, a new one is generated and the old one is revoked
2. **Prevents Replay Attacks:** Even if an old token is stolen, it can only be used once
3. **Detection of Compromise:** If a refresh token is used twice, it indicates potential token theft

### Token Lifecycle

```
┌──────────────┐
│ Registration │
│    or Login  │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│ Generate Tokens  │
│ - Access Token   │ (15-30 min)
│ - Refresh Token  │ (7-30 days)
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│ Access Protected│
│   Resources      │◄─────────────┐
└──────────────────┘              │
       │                          │
       ▼                          │
┌──────────────────┐              │
│ Access Token     │              │
│   Expires?       │──Yes─────────┤
└──────┬───────────┘              │
       │ No                       │
       ▼                          │
┌──────────────────┐              │
│ Refresh Tokens   │              │
└──────┬───────────┘              │
       │                          │
       ▼                          │
┌──────────────────┐              │
│ Use Refresh Token│              │
└──────┬───────────┘              │
       │                          │
       ▼                          │
┌──────────────────┐              │
│ Validate & Rotate│              │
│ - Old Revoke     │              │
│ - New Generate   │              │
└──────┬───────────┘              │
       │                          │
       └──────────────────────────┘
```

---

## API Endpoints

### Authentication Endpoints

#### Public Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/auth/register` | Register new user | No |
| POST | `/api/v1/auth/login` | Login user | No |
| POST | `/api/v1/auth/refresh` | Refresh access token | No |

#### Authenticated Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/auth/me` | Get current user profile | Yes |
| POST | `/api/v1/auth/change-password` | Change password | Yes |
| POST | `/api/v1/auth/logout` | Logout user | Yes |

### Endpoint Details

#### Register
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "name": "string (required)",
  "business_name": "string (required)",
  "email": "string (required, unique)",
  "phone": "string (required)",
  "password": "string (required, min=8)"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "string (required)",
  "password": "string (required, min=8)"
}
```

#### Refresh
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "string (required)"
}
```

#### Get Me
```http
GET /api/v1/auth/me
Authorization: Bearer <access_token>
```

#### Change Password
```http
POST /api/v1/auth/change-password
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "old_password": "string (required)",
  "new_password": "string (required, min=8)",
  "except_token": "string (required)"
}
```

#### Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "refresh_token": "string (required)"
}
```

---

## Middleware Implementation

### Middleware Types

#### 1. RequireAuth
Enforces authentication for protected routes.

```go
func (m *Middleware) RequireAuth() gin.HandlerFunc {
    return m.authJWT.Middleware.MiddlewareFunc()
}
```

**Usage:**
```go
api := route.Group("/api/v1")
authRoute := api.Group("/auth", middleware.RequireAuth())
{
    authRoute.GET("/me", authController.GetMe)
}
```

#### 2. OptionalAuth
Allows requests with or without valid authentication.

```go
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        if userID, role, ok := parseTokenFromHeader(c); ok {
            setContextValues(c, userID, role)
        }
        c.Next()
    }
}
```

**Usage:**
```go
publicApi := api.Group("/public")
productRoute := publicApi.Group("/product", middleware.OptionalAuth())
{
    productRoute.GET("", productController.Index)
}
```

#### 3. RequireRole
Validates if authenticated user has required role.

```go
func (m *Middleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        values, err := utils.ValuesFromContext(c.Request.Context())
        if err != nil {
            c.JSON(http.StatusUnauthorized, response.BuildResponseFailed("unauthorized"))
            c.Abort()
            return
        }

        if !slices.Contains(allowedRoles, values.Role) {
            c.JSON(http.StatusForbidden, response.BuildResponseFailed("insufficient permissions"))
            c.Abort()
            return
        }

        c.Next()
    }
}
```

**Usage:**
```go
adminApi := api.Group("/admin", middleware.RequireAuth())
blogRoute := adminApi.Group("/blog", middleware.RequireRole("admin", "root", "writer"))
{
    blogRoute.POST("", blogController.Create)
}
```

### JWT Middleware Configuration

The JWT middleware is configured with the following settings:

```go
ginjwt.New(&ginjwt.GinJWTMiddleware{
    Realm:            cfg.App.Name,
    Key:              []byte(cfg.JWT.Secret),
    Timeout:          cfg.JWT.Expiration,           // Access token TTL
    MaxRefresh:       cfg.JWT.RefreshExpiration,     // Refresh token TTL
    IdentityKey:      "user_id",
    SigningAlgorithm: "HS256",
    TokenLookup:      "header: Authorization",
    TokenHeadName:    "Bearer",
    TimeFunc:         time.Now,
    SendCookie:       false,
    SecureCookie:     cfg.IsProduction(),
    CookieHTTPOnly:   true,
    CookieSameSite:   http.SameSiteStrictMode,
    
    // Callbacks
    PayloadFunc:     a.payloadFunc,
    IdentityHandler: a.identityHandler,
    Authenticator:   a.authenticator,
    Authorizer:      a.authorizer,
    Unauthorized:    a.unauthorized,
    LoginResponse:   a.loginResponse,
    LogoutResponse:  a.logoutResponse,
})
```

### Context Values

After successful authentication, the following values are set in the request context:

```go
type ContextValues struct {
    UserID    uint
    Role      string
    RequestID string
}
```

Access in controller/service:
```go
values, err := utils.ValuesFromContext(ctx)
if err != nil {
    return nil, err
}
// values.UserID
// values.Role
// values.RequestID
```

---

## Security Features

### 1. Password Security

#### Bcrypt Hashing
- Cost factor: 12 (balances security and performance)
- Prevents rainbow table attacks
- Built-in salt

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
```

#### Timing Attack Prevention
Dummy hash comparison to prevent timing attacks:

```go
dummyHash := []byte("$2a$12$dummy.hash.to.prevent.timing.attacks")
bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
```

### 2. Token Security

#### Access Token
- Short lifespan (15-30 minutes)
- Stateless validation
- No database storage
- Contains user ID and role

#### Refresh Token
- Long lifespan (7-30 days)
- UUID-based format
- Database-backed
- Automatic rotation on use
- Revocation support

### 3. Input Validation

#### Request Validation
All requests are validated using struct tags:

```go
type RegisterRequest struct {
    Name         string `json:"name" binding:"required"`
    Email        string `json:"email" binding:"required,unique=users.email"`
    Phone        string `json:"phone" binding:"required"`
    Password     string `json:"password" binding:"required,min=8"`
}
```

#### Custom Validators
- `unique`: Validates field uniqueness in database
- `exist`: Validates reference existence
- `file`: Validates file uploads

### 4. SQL Injection Prevention

- GORM ORM with parameterized queries
- Input validation before database operations
- Prepared statements for raw SQL

### 5. Session Management

#### Multi-Device Support
Users can have multiple active sessions:
- Each device gets its own refresh token
- Password change revokes all tokens except one
- Logout revokes only the specific token

#### Token Cleanup
- Old tokens are marked as revoked
- Tokens are expired based on `expires_at` timestamp
- Future: Periodic cleanup job to delete expired/revoked tokens

### 6. CORS Configuration

Configurable CORS settings:
```go
c.Use(cors.New(cors.Config{
    AllowOrigins:     cfg.CORS.AllowedOrigins,
    AllowMethods:     cfg.CORS.AllowedMethods,
    AllowHeaders:     cfg.CORS.AllowedHeaders,
    ExposeHeaders:    cfg.CORS.ExposedHeaders,
    AllowCredentials: cfg.CORS.AllowCredentials,
    MaxAge:           cfg.CORS.MaxAge,
}))
```

### 7. Rate Limiting

- Timeout middleware for request timeouts
- Connection limits at server level
- Future: Token bucket rate limiting per user

---

## Database Schema

### Users Table

```sql
CREATE TYPE user_role AS ENUM ('user', 'admin', 'writer', 'root', 'reseller');

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    business_name VARCHAR(255),
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    role user_role NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
```

### Refresh Tokens Table

```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL,
    token VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,
    CONSTRAINT fk_refresh_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

### Model Definitions

#### User Model
```go
type User struct {
    ID           uint     `json:"id" gorm:"primaryKey"`
    Username     string   `json:"username" gorm:"type:varchar(255);not null"`
    Name         string   `json:"name" gorm:"type:varchar(255);null"`
    BusinessName string   `json:"business_name" gorm:"type:varchar(255);null"`
    Email        string   `json:"email" gorm:"type:varchar(255);not null"`
    Phone        string   `json:"phone" gorm:"type:varchar(255);not null"`
    IsActive     bool     `json:"is_active" gorm:"not null;default:true"`
    Role         UserRole `json:"role" gorm:"type:user_role;not null"`
    Password     string   `json:"-" gorm:"type:varchar(255);not null"`
    Timestamp
}
```

#### Refresh Token Model
```go
type RefreshToken struct {
    ID        uuid.UUID  `json:"id" gorm:"primary_key;not null"`
    UserID    uint       `json:"user_id" gorm:"type:bigint;not null"`
    Token     string     `json:"token" gorm:"not null"`
    ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
    CreatedAt time.Time  `json:"created_at" gorm:"not null"`
    UpdatedAt time.Time  `json:"updated_at" gorm:"not null"`
    RevokedAt *time.Time `json:"revoked_at,omitempty" gorm:"null;default:null"`

    User User `json:"user" gorm:"foreignKey:UserID"`
}
```

---

## Code Examples

### Registering a New User

```go
// Controller
func (c *authController) Register(ctx *gin.Context) {
    var req dto.RegisterRequest
    if err := ctx.ShouldBind(&req); err != nil {
        ctx.Error(err).SetType(gin.ErrorTypeBind)
        return
    }

    res, err := c.authService.Register(ctx.Request.Context(), &req)
    if err != nil {
        ctx.Error(err).SetType(gin.ErrorTypePublic)
        return
    }

    ctx.JSON(http.StatusOK, response.BuildResponseSuccess("register success", res))
}

// Service
func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
    // Normalize inputs
    req.Email = strings.ToLower(strings.TrimSpace(req.Email))
    req.Phone = strings.TrimSpace(req.Phone)

    // Create user model
    user := &models.User{Role: models.UserRoleUser, IsActive: true, Username: req.Email}
    if err := copier.Copy(&user, &req); err != nil {
        return nil, cerrors.NewInternalServerError("failed to process user data", err)
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), BcryptCost)
    if err != nil {
        return nil, cerrors.NewInternalServerError("failed to process password", err)
    }
    user.Password = string(hashedPassword)

    // Create user and tokens in transaction
    var authResponse *dto.AuthResponse
    err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
        if err := s.userRepo.Create(txCtx, user); err != nil {
            return err
        }
        authResponse, err = s.authJWT.GenerateTokensForUser(txCtx, user)
        return err
    })

    return authResponse, err
}
```

### Validating Credentials

```go
func (a *AuthJWT) validateCredentials(ctx context.Context, username, password string) (*models.User, error) {
    username = strings.TrimSpace(username)
    dummyHash := []byte("$2a$12$dummy.hash.to.prevent.timing.attacks")

    var user *models.User
    var err error

    // Find user by email or username
    if strings.Contains(username, "@") {
        user, err = a.userRepo.FindByEmail(ctx, strings.ToLower(username))
    } else {
        user, err = a.userRepo.FindByUsername(ctx, strings.ToLower(username))
    }

    if err != nil {
        bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
        return nil, err
    }

    if !user.IsActive {
        bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
        return nil, errors.New("inactive account")
    }

    // Verify password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
        return nil, err
    }

    return user, nil
}
```

### Generating Tokens

```go
func (a *AuthJWT) GenerateTokensForUser(ctx context.Context, user *models.User) (*dto.AuthResponse, error) {
    // Generate access token
    token, err := a.Middleware.TokenGenerator(ctx, user)
    if err != nil {
        return nil, cerrors.NewInternalServerError("failed to generate access token", err)
    }

    // Generate and store refresh token
    refreshToken, err := a.createRefreshToken(ctx, user.ID)
    if err != nil {
        return nil, cerrors.NewInternalServerError("failed to generate refresh token", err)
    }

    return &dto.AuthResponse{
        AccessToken:  token.AccessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
    }, nil
}

func (a *AuthJWT) createRefreshToken(ctx context.Context, userID uint) (string, error) {
    token := uuid.New().String() + "-" + uuid.New().String()

    err := a.refreshTokenRepo.Create(ctx, &models.RefreshToken{
        ID:        uuid.New(),
        UserID:    userID,
        Token:     token,
        ExpiresAt: time.Now().Add(config.Get().JWT.RefreshExpiration),
    })

    return token, err
}
```

### Refreshing Tokens

```go
func (a *AuthJWT) ValidateAndRotateRefreshToken(ctx context.Context, oldToken string) (*dto.AuthResponse, error) {
    // Find token in database
    tokenRecord, err := a.refreshTokenRepo.FindByToken(ctx, oldToken)
    if err != nil {
        return nil, cerrors.NewBadRequestError("invalid or expired refresh token")
    }

    // Get user
    user, err := a.userRepo.FindById(ctx, tokenRecord.UserID)
    if err != nil {
        return nil, err
    }

    if !user.IsActive {
        return nil, cerrors.NewBadRequestError("user account is inactive")
    }

    // Revoke old token
    _ = a.refreshTokenRepo.RevokeByToken(ctx, oldToken)

    // Generate new tokens
    return a.GenerateTokensForUser(ctx, user)
}
```

### Using Middleware in Routes

```go
func RegisterRoutes(route *gin.Engine, middleware *middlewares.Middleware, ...) {
    api := route.Group("/api/v1")
    
    // Public routes
    authRoute := api.Group("/auth")
    {
        authRoute.POST("/register", authController.Register)
        authRoute.POST("/login", middleware.LoginHandler())
        authRoute.POST("/refresh", authController.Refresh)

        // Protected routes
        authenticatedAuthRoute := authRoute.Group("", middleware.RequireAuth())
        {
            authenticatedAuthRoute.GET("/me", authController.GetMe)
            authenticatedAuthRoute.POST("/change-password", authController.ChangePassword)
            authenticatedAuthRoute.POST("/logout", authController.Logout)
        }
    }

    // Admin routes with role-based access
    adminApi := api.Group("/admin", middleware.RequireAuth())
    {
        blogRoute := adminApi.Group("/blog", middleware.RequireRole("admin", "root", "writer"))
        {
            blogRoute.POST("", blogController.Create)
            blogRoute.GET("", blogController.Index)
        }

        productRoute := adminApi.Group("/product", middleware.RequireRole("admin", "root"))
        {
            productRoute.POST("", productController.Create)
            productRoute.GET("", productController.Index)
        }
    }

    // Public routes with optional auth
    publicApi := api.Group("/public")
    {
        productRoute := publicApi.Group("/product", middleware.OptionalAuth())
        {
            productRoute.GET("", productController.Index)
            productRoute.GET("/:slug", productController.FindBySlug)
        }
    }
}
```

---

## Best Practices

### For Developers

1. **Always Use Middleware**
   - Protect all sensitive endpoints with `RequireAuth()`
   - Use `RequireRole()` for role-based access control
   - Use `OptionalAuth()` for endpoints that provide personalized content when authenticated

2. **Handle Authentication Errors**
   ```go
   values, err := utils.ValuesFromContext(ctx)
   if err != nil {
       // User not authenticated
       return nil, cerrors.NewUnauthorizedError("unauthorized")
   }
   ```

3. **Check User Status**
   ```go
   user, err := s.userRepo.FindById(ctx, values.UserID)
   if err != nil || !user.IsActive {
       return nil, cerrors.NewForbiddenError("user account is inactive")
   }
   ```

4. **Use Transactions for Multi-Step Operations**
   ```go
   err = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
       if err := s.userRepo.Create(txCtx, user); err != nil {
           return err
       }
       return s.refreshTokenRepo.Create(txCtx, refreshToken)
   })
   ```

5. **Log Important Events**
   ```go
   logger.Info("Login successful", zap.Uint("user_id", user.ID))
   logger.Warn("Failed login attempt", zap.String("username", username))
   ```

### For API Consumers

1. **Store Tokens Securely**
   - Store access tokens in memory
   - Store refresh tokens in secure storage (httpOnly cookies, secure storage APIs)
   - Never store tokens in localStorage (vulnerable to XSS)

2. **Handle Token Expiration**
   - Check for 401 responses
   - Use refresh token to get new access token
   - Redirect to login if refresh fails

3. **Implement Token Refresh Logic**
   ```javascript
   async function refreshToken() {
     const response = await fetch('/api/v1/auth/refresh', {
       method: 'POST',
       headers: { 'Content-Type': 'application/json' },
       body: JSON.stringify({ refresh_token: storedRefreshToken })
     });
     
     if (response.ok) {
       const { data } = await response.json();
       updateTokens(data.access_token, data.refresh_token);
     } else {
       // Redirect to login
       logout();
     }
   }
   ```

4. **Logout Properly**
   - Call `/api/v1/auth/logout` with refresh token
   - Clear all tokens from storage
   - Redirect to login page

5. **Handle Password Changes**
   - On password change, all tokens except the current one are revoked
   - User must re-login on other devices
   - Keep the current session active by passing `except_token`

### Security Recommendations

1. **Use HTTPS in Production**
   - Never send tokens over unencrypted connections
   - Configure CORS properly

2. **Set Appropriate Token Expiration**
   - Access token: 15-30 minutes
   - Refresh token: 7-30 days
   - Balance security and user experience

3. **Implement Token Cleanup**
   - Periodically clean up expired/revoked tokens
   - Consider using a background job

4. **Monitor Authentication Events**
   - Track failed login attempts
   - Monitor unusual activity patterns
   - Implement rate limiting

5. **Keep JWT Secret Secure**
   - Use strong, random secret
   - Rotate secret periodically
   - Never commit secret to version control

---

## Troubleshooting

### Common Issues

#### 1. "unauthorized" Error

**Cause:** Invalid or expired access token

**Solution:**
- Verify token is sent in `Authorization` header
- Check token format: `Bearer <token>`
- Refresh token if expired
- Re-login if refresh fails

#### 2. "insufficient permissions" Error

**Cause:** User role doesn't have required permissions

**Solution:**
- Check user role in `/api/v1/auth/me`
- Verify role has access to endpoint
- Contact administrator if role needs to be changed

#### 3. "user account is inactive" Error

**Cause:** User account is deactivated

**Solution:**
- Contact administrator to activate account
- Verify account status in database

#### 4. "current password is incorrect" Error

**Cause:** Wrong old password provided

**Solution:**
- Verify old password
- Reset password if forgotten

#### 5. "invalid or expired refresh token" Error

**Cause:** Refresh token is invalid, expired, or revoked

**Solution:**
- Verify refresh token is correct
- Check if token has expired
- Re-login to get new tokens

#### 6. Token Revoked After Password Change

**Cause:** All tokens except the one used for password change are revoked

**Solution:**
- This is expected behavior
- User must re-login on other devices
- Pass `except_token` to keep current session active

### Debugging Tips

1. **Enable Debug Logging**
   ```go
   logger.Debug("Authentication request", 
       zap.String("user_id", userID),
       zap.String("role", role))
   ```

2. **Check Token Contents**
   ```bash
   # Decode JWT to see claims
   echo "eyJhbGciOiJIUzI1NiIs..." | base64 -d | jq .
   ```

3. **Verify Database State**
   ```sql
   SELECT id, email, is_active, role FROM users WHERE email = 'user@example.com';
   SELECT * FROM refresh_tokens WHERE user_id = 1;
   ```

4. **Test Middleware**
   - Use curl or Postman to test endpoints
   - Verify headers are sent correctly
   - Check response status codes

5. **Monitor Logs**
   - Check application logs for errors
   - Look for authentication failures
   - Review security events

### Performance Considerations

1. **Database Queries**
   - User lookup is fast with indexed fields
   - Refresh token validation requires database query
   - Consider caching user data for frequently accessed endpoints

2. **Token Validation**
   - Access token validation is fast (no database query)
   - Refresh token validation requires database query
   - Bcrypt hash comparison is computationally expensive

3. **Connection Pooling**
   - Configure database connection pool size
   - Monitor connection usage
   - Adjust pool size based on load

---

## Future Enhancements

### Planned Features

1. **Two-Factor Authentication (2FA)**
   - TOTP-based 2FA
   - SMS-based 2FA
   - Recovery codes

2. **Social Login**
   - Google OAuth
   - Facebook Login
   - Apple Sign In

3. **Session Management UI**
   - View active sessions
   - Revoke specific sessions
   - Session history

4. **Password Recovery**
   - Email-based password reset
   - Security questions
   - Account recovery flow

5. **Rate Limiting**
   - Per-IP rate limiting
   - Per-user rate limiting
   - Brute force protection

6. **Advanced Security**
   - Device fingerprinting
   - Anomaly detection
   - IP-based restrictions

7. **Audit Logging**
   - Comprehensive audit trail
   - Security event logging
   - Compliance reporting

---

## Conclusion

This authentication system provides a robust, secure, and scalable solution for user authentication in the athleton application. The combination of JWT tokens, refresh token rotation, role-based access control, and comprehensive security measures ensures that user data and resources are protected while providing a smooth user experience.

The modular design allows for easy integration with other parts of the application, and the clear separation of concerns makes the codebase maintainable and testable. Future enhancements can be added without disrupting the existing authentication flow.

For questions or issues, please refer to the project documentation or contact the development team.
