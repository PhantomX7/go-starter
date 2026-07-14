package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newHealthTestServer(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	server := gin.New()
	registerHealthRoutes(server, db)
	return server, db
}

func probe(t *testing.T, server *gin.Engine, path string) (int, map[string]any) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	server.ServeHTTP(rec, req)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return rec.Code, body
}

func TestLivezReportsProcessLiveness(t *testing.T) {
	server, _ := newHealthTestServer(t)

	code, body := probe(t, server, "/livez")

	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "ok", body["status"])
}

func TestHealthzReportsDatabaseCheck(t *testing.T) {
	server, _ := newHealthTestServer(t)

	code, body := probe(t, server, "/healthz")

	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "ok", body["status"])
	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok, "healthz must report per-dependency checks, unlike livez")
	require.Equal(t, "ok", checks["database"])
}

func TestHealthzDegradesWhenDatabaseIsDown(t *testing.T) {
	server, db := newHealthTestServer(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	code, body := probe(t, server, "/healthz")

	require.Equal(t, http.StatusServiceUnavailable, code)
	require.Equal(t, "degraded", body["status"])
	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "unavailable", checks["database"])

	// Liveness must be unaffected: the process is still alive, so a DB outage
	// must not get the pod restarted.
	liveCode, _ := probe(t, server, "/livez")
	require.Equal(t, http.StatusOK, liveCode)
}

func TestReadyzFailsWhenDatabaseIsDown(t *testing.T) {
	server, db := newHealthTestServer(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	code, body := probe(t, server, "/readyz")

	require.Equal(t, http.StatusServiceUnavailable, code)
	require.Equal(t, "unavailable", body["status"])
}
