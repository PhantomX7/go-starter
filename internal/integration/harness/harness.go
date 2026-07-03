// Package harness assembles the real application stack for integration tests:
// the Gin engine built by bootstrap.SetupServer, the full middleware chain,
// and real services/repositories against an in-memory SQLite database. Test
// packages under internal/integration/* import it as their shared fixture.
package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/PhantomX7/athleton/internal/bootstrap"
	"github.com/PhantomX7/athleton/internal/middlewares"
	"github.com/PhantomX7/athleton/internal/models"
	adminrolemodule "github.com/PhantomX7/athleton/internal/modules/admin_role"
	adminrolecontroller "github.com/PhantomX7/athleton/internal/modules/admin_role/controller"
	adminrolerepository "github.com/PhantomX7/athleton/internal/modules/admin_role/repository"
	adminroleservice "github.com/PhantomX7/athleton/internal/modules/admin_role/service"
	authmodule "github.com/PhantomX7/athleton/internal/modules/auth"
	authcontroller "github.com/PhantomX7/athleton/internal/modules/auth/controller"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	authservice "github.com/PhantomX7/athleton/internal/modules/auth/service"
	configmodule "github.com/PhantomX7/athleton/internal/modules/config"
	configcontroller "github.com/PhantomX7/athleton/internal/modules/config/controller"
	configrepository "github.com/PhantomX7/athleton/internal/modules/config/repository"
	configservice "github.com/PhantomX7/athleton/internal/modules/config/service"
	logmodule "github.com/PhantomX7/athleton/internal/modules/log"
	logcontroller "github.com/PhantomX7/athleton/internal/modules/log/controller"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	logservice "github.com/PhantomX7/athleton/internal/modules/log/service"
	rtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	usermodule "github.com/PhantomX7/athleton/internal/modules/user"
	usercontroller "github.com/PhantomX7/athleton/internal/modules/user/controller"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	userservice "github.com/PhantomX7/athleton/internal/modules/user/service"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
	pkgvalidator "github.com/PhantomX7/athleton/pkg/validator"
)

const (
	// TestPassword is the plaintext password for every seeded user.
	TestPassword = "integration-pass-1"
	// TestNewPassword is used by the change-password flows.
	TestNewPassword = "integration-pass-2"

	// RootUsername identifies the seeded root fixture account.
	RootUsername = "root"
	// AdminUsername identifies the seeded admin fixture account.
	AdminUsername = "admin"
	// MemberUsername identifies the seeded regular-user fixture account.
	MemberUsername = "member"

	// TestMaxBodyBytes mirrors the production default (10 MiB).
	TestMaxBodyBytes = int64(10 << 20)
)

// passwordHash is computed once per process. bcrypt.MinCost keeps seeding
// fast; login still exercises the real bcrypt comparison path.
var passwordHash = sync.OnceValue(func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte(TestPassword), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
})

// PasswordHash returns the shared bcrypt hash of TestPassword for tests that
// seed additional users.
func PasswordHash() string {
	return passwordHash()
}

// testConfig returns a complete, valid configuration equivalent to what
// config.Load would produce in development.
func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:           "localhost",
			Port:           8080,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    120 * time.Second,
			RequestTimeout: 30 * time.Second,
			MaxBodyBytes:   TestMaxBodyBytes,
		},
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Host:   "memory",
		},
		JWT: config.JWTConfig{
			Secret:            "integration-test-secret-key-0123456789abcdef",
			Expiration:        10 * time.Minute,
			RefreshExpiration: 72 * time.Hour,
			Issuer:            "athleton-integration",
		},
		App: config.AppConfig{
			Name:        "Athleton",
			Version:     "test",
			Environment: "development",
			Assets:      "./assets",
		},
		S3: config.S3Config{
			Bucket: "test-bucket",
			Region: "ap-southeast-1",
		},
		Admin: config.AdminConfig{
			DefaultPassword: "integration-admin-pass",
		},
		Log: config.LogConfig{
			Level:      "error",
			FilePath:   "logs/integration-test.log",
			MaxSize:    1,
			MaxBackups: 1,
			MaxAge:     1,
			Compress:   false,
			Console:    false,
		},
	}
}

// App bundles the assembled application and its seeded fixtures.
type App struct {
	Engine *gin.Engine
	DB     *gorm.DB
	Casbin casbin.Client

	RootUser   models.User
	AdminUser  models.User
	MemberUser models.User
	AdminRole  models.AdminRole
}

// New hand-wires the real application stack (no fx): in-memory SQLite,
// repositories -> services -> controllers -> route registrars, the production
// middleware bundle, and the engine built by bootstrap.SetupServer. The
// /api/v1 groups are created exactly like routes.RegisterRoutes does.
func New(t *testing.T) *App {
	t.Helper()

	gin.SetMode(gin.TestMode)
	logger.Log = zap.NewNop()
	cfg := testConfig()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	// Each ":memory:" connection gets its own database; force a single shared
	// connection so handlers, transactions, and async audit writers all see
	// the same data.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.AdminRole{},
		&models.RefreshToken{},
		&models.Log{},
		&models.Config{},
	))

	userRepo := userrepository.NewUserRepository(db)
	refreshTokenRepo := rtokenrepository.NewRefreshTokenRepository(db)
	logRepo := logrepository.NewLogRepository(db)
	adminRoleRepo := adminrolerepository.NewAdminRoleRepository(db)
	configRepo := configrepository.NewConfigRepository(db)

	txManager := transaction_manager.NewTransactionManager(db)

	authJWT, err := authjwt.NewAuthJWT(cfg, userRepo, refreshTokenRepo, logRepo, txManager)
	require.NoError(t, err)

	casbinClient, err := casbin.New(db)
	require.NoError(t, err)

	mw := middlewares.NewMiddleware(cfg, authJWT, casbinClient)
	engine := bootstrap.SetupServer(cfg, mw, pkgvalidator.New(db), db)

	authService := authservice.NewAuthService(userRepo, logRepo, authJWT, casbinClient, txManager)
	adminRoleService := adminroleservice.NewAdminRoleService(adminRoleRepo, logRepo, casbinClient, txManager)
	configService := configservice.NewConfigService(configRepo, logRepo)
	logService := logservice.NewLogService(logRepo)
	userService := userservice.NewUserService(userRepo, adminRoleRepo, refreshTokenRepo, logRepo, casbinClient, txManager, zap.NewNop())

	// Mirror routes.RegisterRoutes: shared /api/v1 groups with the same
	// middleware stack (rate limiting before auth on /admin, the admin role
	// boundary, then the must-change-default-password gate).
	root := engine.Group("/api/v1")
	routeCtx := &routes.Context{
		Root:   root,
		Public: root.Group("/public"),
		Admin: root.Group("/admin",
			mw.AdminRateLimiter(),
			mw.RequireAuth(),
			mw.RequireRole(models.UserRoleAdmin.ToString(), models.UserRoleRoot.ToString()),
			mw.RequirePasswordChanged(),
		),
		MW: mw,
	}
	// Unguarded probe route: proves the group-level RequireRole boundary
	// rejects plain users even when a route carries no permission guard.
	routeCtx.Admin.GET("/__probe", func(c *gin.Context) { c.Status(http.StatusOK) })

	authmodule.NewRoutes(authcontroller.NewAuthController(authService)).RegisterRoutes(routeCtx)
	usermodule.NewRoutes(usercontroller.NewUserController(userService)).RegisterRoutes(routeCtx)
	adminrolemodule.NewRoutes(adminrolecontroller.NewAdminRoleController(adminRoleService)).RegisterRoutes(routeCtx)
	configController := configcontroller.NewConfigController(configService)
	configmodule.NewAdminRoutes(configController).RegisterRoutes(routeCtx)
	configmodule.NewPublicRoutes(configController).RegisterRoutes(routeCtx)
	logmodule.NewRoutes(logcontroller.NewLogController(logService)).RegisterRoutes(routeCtx)

	app := &App{
		Engine: engine,
		DB:     db,
		Casbin: casbinClient,
	}
	app.seed(t)
	return app
}

// seed inserts a deterministic fixture set: an admin role plus one user per
// role (root, admin with the role assigned, plain user).
func (a *App) seed(t *testing.T) {
	t.Helper()

	a.AdminRole = models.AdminRole{
		Name:        "Editor",
		Description: "integration test role",
		IsActive:    true,
	}
	require.NoError(t, a.DB.Create(&a.AdminRole).Error)

	hash := passwordHash()
	// The fixture users already "changed" their password so the
	// RequirePasswordChanged gate on /admin stays out of the way for the rest
	// of the suite; the gate itself is covered by the auth package tests.
	passwordChangedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	a.RootUser = models.User{
		Username:          RootUsername,
		Name:              "Root User",
		Email:             "root@test.local",
		Phone:             "+620000000001",
		IsActive:          true,
		Role:              models.UserRoleRoot,
		Password:          hash,
		PasswordChangedAt: &passwordChangedAt,
	}
	a.AdminUser = models.User{
		Username:          AdminUsername,
		Name:              "Admin User",
		Email:             "admin@test.local",
		Phone:             "+620000000002",
		IsActive:          true,
		Role:              models.UserRoleAdmin,
		AdminRoleID:       &a.AdminRole.ID,
		Password:          hash,
		PasswordChangedAt: &passwordChangedAt,
	}
	a.MemberUser = models.User{
		Username: MemberUsername,
		Name:     "Member User",
		Email:    "member@test.local",
		Phone:    "+620000000003",
		IsActive: true,
		Role:     models.UserRoleUser,
		Password: hash,
	}
	require.NoError(t, a.DB.Create(&a.RootUser).Error)
	require.NoError(t, a.DB.Create(&a.AdminUser).Error)
	require.NoError(t, a.DB.Create(&a.MemberUser).Error)
}

// EnvelopeMeta is the pagination metadata inside a list response.
type EnvelopeMeta struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}

// Envelope mirrors pkg/response.Response with raw payloads for re-decoding.
type Envelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Error   json.RawMessage `json:"error"`
	Data    json.RawMessage `json:"data"`
	Meta    *EnvelopeMeta   `json:"meta"`
}

// TokenPair is the auth payload inside a login/refresh/register response.
type TokenPair struct {
	AccessToken        string `json:"access_token"`
	RefreshToken       string `json:"refresh_token"`
	TokenType          string `json:"token_type"`
	MustChangePassword bool   `json:"must_change_password"`
}

// Request performs an HTTP request against the assembled engine. body may be
// nil, a raw string/[]byte, or any JSON-marshalable value.
func (a *App) Request(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	switch b := body.(type) {
	case nil:
		// no body
	case string:
		reader = strings.NewReader(b)
	case []byte:
		reader = bytes.NewReader(b)
	default:
		buf, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(buf)
	}

	req := httptest.NewRequestWithContext(context.Background(), method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	a.Engine.ServeHTTP(rec, req)
	return rec
}

// DecodeEnvelope parses the standard JSON response envelope.
func DecodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) Envelope {
	t.Helper()
	var env Envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "body: %s", rec.Body.String())
	return env
}

// DecodeData unmarshals the envelope's data payload into out.
func DecodeData(t *testing.T, env Envelope, out any) {
	t.Helper()
	require.NotEmpty(t, env.Data, "envelope has no data payload")
	require.NoError(t, json.Unmarshal(env.Data, out))
}

// LoginAs authenticates the given seeded user via the real login endpoint and
// returns the issued token pair.
func (a *App) LoginAs(t *testing.T, username, password string) TokenPair {
	t.Helper()

	rec := a.Request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, "")
	require.Equal(t, http.StatusOK, rec.Code, "login failed: %s", rec.Body.String())

	env := DecodeEnvelope(t, rec)
	require.True(t, env.Status)

	var tokens TokenPair
	DecodeData(t, env, &tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	require.Equal(t, "Bearer", tokens.TokenType)
	return tokens
}

// NewRequestWithHeader builds a bare request plus recorder for cases that
// need a custom header on an otherwise body-less request.
func NewRequestWithHeader(t *testing.T, method, path, header, value string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	req.Header.Set(header, value)
	return req, httptest.NewRecorder()
}

// WaitForAuditLog polls (with a deadline, no fixed sleeps for correctness)
// until an audit log row matching action and entityID appears.
func (a *App) WaitForAuditLog(t *testing.T, action models.LogAction, entityID uint) models.Log {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var row models.Log
		err := a.DB.
			Where("action = ? AND entity_id = ?", string(action), entityID).
			First(&row).Error
		if err == nil {
			return row
		}
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		require.True(t, time.Now().Before(deadline),
			"timed out waiting for audit log action=%s entity_id=%d", action, entityID)
		time.Sleep(10 * time.Millisecond)
	}
}

// Itoa formats a uint for building URL paths.
func Itoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
