package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	authmodule "github.com/PhantomX7/athleton/internal/modules/auth"
	authcontroller "github.com/PhantomX7/athleton/internal/modules/auth/controller"
	authjwt "github.com/PhantomX7/athleton/internal/modules/auth/jwt"
	authservice "github.com/PhantomX7/athleton/internal/modules/auth/service"
	logmodule "github.com/PhantomX7/athleton/internal/modules/log"
	logcontroller "github.com/PhantomX7/athleton/internal/modules/log/controller"
	logrepository "github.com/PhantomX7/athleton/internal/modules/log/repository"
	logservice "github.com/PhantomX7/athleton/internal/modules/log/service"
	postmodule "github.com/PhantomX7/athleton/internal/modules/post"
	postcontroller "github.com/PhantomX7/athleton/internal/modules/post/controller"
	postrepository "github.com/PhantomX7/athleton/internal/modules/post/repository"
	postservice "github.com/PhantomX7/athleton/internal/modules/post/service"
	rtokenrepository "github.com/PhantomX7/athleton/internal/modules/refresh_token/repository"
	userrepository "github.com/PhantomX7/athleton/internal/modules/user/repository"
	"github.com/PhantomX7/athleton/internal/routes"
	"github.com/PhantomX7/athleton/libs/casbin"
	"github.com/PhantomX7/athleton/libs/transaction_manager"
	"github.com/PhantomX7/athleton/pkg/config"
	"github.com/PhantomX7/athleton/pkg/logger"
	pkgvalidator "github.com/PhantomX7/athleton/pkg/validator"
)

const (
	// testPassword is the plaintext password for every seeded user.
	testPassword = "integration-pass-1"
	// testNewPassword is used by the change-password flow.
	testNewPassword = "integration-pass-2"

	rootUsername   = "root"
	adminUsername  = "admin"
	memberUsername = "member"

	testMaxBodyBytes = int64(10 << 20) // mirrors production default (10 MiB)
)

// testPasswordHash is computed once per process. bcrypt.MinCost keeps seeding
// fast; login still exercises the real bcrypt comparison path.
var testPasswordHash = sync.OnceValue(func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(hash)
})

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
			MaxBodyBytes:   testMaxBodyBytes,
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

// testApp bundles the assembled application and its seeded fixtures.
type testApp struct {
	engine       *gin.Engine
	db           *gorm.DB
	casbinClient casbin.Client

	rootUser   models.User
	adminUser  models.User
	memberUser models.User
	adminRole  models.AdminRole
}

// newTestApp hand-wires the real application stack (no fx): in-memory SQLite,
// repositories -> services -> controllers -> route registrars, the production
// middleware bundle, and the engine built by bootstrap.SetupServer. The /api/v1
// groups are created exactly like routes.RegisterRoutes does.
func newTestApp(t *testing.T) *testApp {
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
		&models.Post{},
		&models.Config{},
	))

	userRepo := userrepository.NewUserRepository(db)
	refreshTokenRepo := rtokenrepository.NewRefreshTokenRepository(db)
	logRepo := logrepository.NewLogRepository(db)
	postRepo := postrepository.NewPostRepository(db)

	authJWT, err := authjwt.NewAuthJWT(cfg, userRepo, refreshTokenRepo, logRepo)
	require.NoError(t, err)

	casbinClient, err := casbin.New(db)
	require.NoError(t, err)

	mw := middlewares.NewMiddleware(authJWT, casbinClient)
	engine := bootstrap.SetupServer(cfg, mw, pkgvalidator.New(db), db)

	txManager := transaction_manager.NewTransactionManager(db)
	authService := authservice.NewAuthService(userRepo, logRepo, authJWT, casbinClient, txManager)
	postService := postservice.NewPostService(postRepo)
	logService := logservice.NewLogService(logRepo)

	// Mirror routes.RegisterRoutes: shared /api/v1 groups with the same
	// middleware stack (rate limiting before auth on /admin, then the
	// must-change-default-password gate).
	root := engine.Group("/api/v1")
	routeCtx := &routes.Context{
		Root:   root,
		Public: root.Group("/public"),
		Admin:  root.Group("/admin", mw.AdminRateLimiter(), mw.RequireAuth(), mw.RequirePasswordChanged()),
		MW:     mw,
	}
	authmodule.NewRoutes(authcontroller.NewAuthController(authService)).RegisterRoutes(routeCtx)
	postmodule.NewRoutes(postcontroller.NewPostController(postService)).RegisterRoutes(routeCtx)
	logmodule.NewRoutes(logcontroller.NewLogController(logService)).RegisterRoutes(routeCtx)

	app := &testApp{
		engine:       engine,
		db:           db,
		casbinClient: casbinClient,
	}
	app.seed(t)
	return app
}

// seed inserts a deterministic fixture set: an admin role plus one user per
// role (root, admin with the role assigned, plain user).
func (a *testApp) seed(t *testing.T) {
	t.Helper()

	a.adminRole = models.AdminRole{
		Name:        "Editor",
		Description: "integration test role",
		IsActive:    true,
	}
	require.NoError(t, a.db.Create(&a.adminRole).Error)

	hash := testPasswordHash()
	// The fixture users already "changed" their password so the
	// RequirePasswordChanged gate on /admin stays out of the way for the rest
	// of the suite; the gate itself is covered by password_gate_test.go.
	passwordChangedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	a.rootUser = models.User{
		Username:          rootUsername,
		Name:              "Root User",
		Email:             "root@test.local",
		Phone:             "+620000000001",
		IsActive:          true,
		Role:              models.UserRoleRoot,
		Password:          hash,
		PasswordChangedAt: &passwordChangedAt,
	}
	a.adminUser = models.User{
		Username:          adminUsername,
		Name:              "Admin User",
		Email:             "admin@test.local",
		Phone:             "+620000000002",
		IsActive:          true,
		Role:              models.UserRoleAdmin,
		AdminRoleID:       &a.adminRole.ID,
		Password:          hash,
		PasswordChangedAt: &passwordChangedAt,
	}
	a.memberUser = models.User{
		Username: memberUsername,
		Name:     "Member User",
		Email:    "member@test.local",
		Phone:    "+620000000003",
		IsActive: true,
		Role:     models.UserRoleUser,
		Password: hash,
	}
	require.NoError(t, a.db.Create(&a.rootUser).Error)
	require.NoError(t, a.db.Create(&a.adminUser).Error)
	require.NoError(t, a.db.Create(&a.memberUser).Error)
}

// envelope mirrors pkg/response.Response with raw payloads for re-decoding.
type envelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Error   json.RawMessage `json:"error"`
	Data    json.RawMessage `json:"data"`
	Meta    *struct {
		Limit  int   `json:"limit"`
		Offset int   `json:"offset"`
		Total  int64 `json:"total"`
	} `json:"meta"`
}

// tokenPair is the auth payload inside a login/refresh/register response.
type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// request performs an HTTP request against the assembled engine. body may be
// nil, a raw string/[]byte, or any JSON-marshalable value.
func (a *testApp) request(t *testing.T, method, path string, body any, token string) *httptest.ResponseRecorder {
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
	a.engine.ServeHTTP(rec, req)
	return rec
}

// decodeEnvelope parses the standard JSON response envelope.
func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), "body: %s", rec.Body.String())
	return env
}

// decodeData unmarshals the envelope's data payload into out.
func decodeData(t *testing.T, env envelope, out any) {
	t.Helper()
	require.NotEmpty(t, env.Data, "envelope has no data payload")
	require.NoError(t, json.Unmarshal(env.Data, out))
}

// loginAs authenticates the given seeded user via the real login endpoint and
// returns the issued token pair.
func (a *testApp) loginAs(t *testing.T, username, password string) tokenPair {
	t.Helper()

	rec := a.request(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, "")
	require.Equal(t, http.StatusOK, rec.Code, "login failed: %s", rec.Body.String())

	env := decodeEnvelope(t, rec)
	require.True(t, env.Status)

	var tokens tokenPair
	decodeData(t, env, &tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	require.Equal(t, "Bearer", tokens.TokenType)
	return tokens
}

// newRequestWithHeader builds a bare request plus recorder for cases that need
// a custom header on an otherwise body-less request.
func newRequestWithHeader(t *testing.T, method, path, header, value string) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	req.Header.Set(header, value)
	return req, httptest.NewRecorder()
}

// waitForAuditLog polls (with a deadline, no fixed sleeps for correctness)
// until an audit log row matching action and entityID appears.
func (a *testApp) waitForAuditLog(t *testing.T, action models.LogAction, entityID uint) models.Log {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		var row models.Log
		err := a.db.
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
