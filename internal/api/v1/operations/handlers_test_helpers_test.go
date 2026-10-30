package operations

// Shared test helpers for internal/api/v1/operations handler tests (Phase 74-01).
// Per D-08 mock pattern (Phase 73-01 duty_handler_test.go), each sub-package uses
// mock services with *Func fields; this file centralizes cross-handler utilities
// so individual test files stay focused on per-handler behavior.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// errNotImplemented is returned by mock services when the test didn't override the
// corresponding *Func field. Helps surface test gaps loudly.
var errNotImplemented = errors.New("not implemented")

// stubRecorder satisfies services.OperLogService for operlog.Record; tests that
// need to assert on recording should swap with a custom Recorder.
type stubRecorder struct{}

func (s *stubRecorder) RecordAsync(_ *gorm.DB, _ string, _ int, _, _, _ string,
	_, _, _ *string, _ *string, _, _, _ *string, _ int, _ int64) {
}
func (s *stubRecorder) RecordOperLog(_ context.Context, _ *gorm.DB, _ *models.OperLog) error {
	return nil
}
func (s *stubRecorder) RecordFromGinContext(_ *gin.Context, _ *gorm.DB, _ string, _ int, _ string) {
}

// newTestCore returns a *core.Core backed by an in-memory sqlite DB and a stub
// operlog recorder (so operlog.Record doesn't panic on nil svc).
func newTestCore(t *testing.T) *core.Core {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return &core.Core{
		CoreInfra: &core.CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
		CoreServices: &core.CoreServices{
			OperLogService: &stubRecorder{},
		},
	}
}

// httpDo is a generic request helper used by handler-specific helpers below.
func httpDo(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// mountRoutes registers a handler map by URL pattern. Keys can contain :id placeholders.
// This is used by handler tests that need a single router for all CRUD endpoints.
type routeMount struct {
	method  string
	path    string
	handler gin.HandlerFunc
}

func mountRouter(routes []routeMount) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	for _, rm := range routes {
		switch rm.method {
		case http.MethodGet:
			r.GET(rm.path, rm.handler)
		default:
			r.POST(rm.path, rm.handler)
		}
	}
	return r
}

// genericMockListPage builds a minimal PageResult for List endpoints.
func genericMockListPage(total int) *operations.PageResult {
	return &operations.PageResult{
		List:     []map[string]interface{}{},
		Total:    int64(total),
		Current:  1,
		PageSize: 10,
	}
}

// listRequestFromBody returns a PageRequest-like default to keep tests concise.
// Field names mirror requests.PaginationParams → services/base.BaseListRequest.
func listRequestFromBody(_ string) requests.PaginationParams {
	return requests.PaginationParams{
		BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 10},
	}
}
