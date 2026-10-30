package asset

// Phase 74 Plan 02 — ReconciliationHandler unit tests (D-12 strict: test-only)
//
// Per D-08 mock pattern (Phase 73-01 duty_handler_test.go), this file uses
// mockXxxService structs with *Func fields. The shared helpers (newTestCore,
// stubRecorder, mountRouter, httpDo) live in this file so individual test
// files stay focused on per-handler behavior.
//
// Coverage target: ReconciliationHandler (5 endpoints — ListExceptions,
// GetExceptionByID, ResolveException, GetByWorkstation, Refresh).

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	assetSvc "github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// ============================================================================
// Shared test helpers (Phase 74-01 / 74-02 pattern)
// ============================================================================

// errNotImplemented is returned by mock services when the test didn't override
// the corresponding *Func field.
var errNotImplemented = errors.New("not implemented")

// stubRecorder satisfies services.OperLogService for operlog.Record; tests
// that need to assert on recording should swap with a custom Recorder.
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

// routeMount describes a single (method, path, handler) tuple.
type routeMount struct {
	method  string
	path    string
	handler gin.HandlerFunc
}

// mountRouter builds a test router with the given routes registered.
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

// newReconciliationHandler builds a handler with optional core dependency.
func newReconciliationHandler(svc assetSvc.ReconciliationService, c *core.Core) *ReconciliationHandler {
	h := NewReconciliationHandler(svc)
	if c != nil {
		h.WithCore(c)
	}
	return h
}

// ============================================================================
// mockReconciliationService — D-08 mock pattern (Phase 73-01 reference)
// ============================================================================

type mockReconciliationService struct {
	ListExceptionsFunc    func(ctx context.Context, params *assetSvc.ExceptionListParams) (*base.PageResult, error)
	GetByIDFunc           func(ctx context.Context, id string) (*models.SysDataReconciliation, error)
	ResolveExceptionFunc  func(ctx context.Context, id, userID string, note *string) error
	RefreshFunc           func(ctx context.Context) (int, int, int, int, error)
	GetByWorkstationFunc  func(ctx context.Context, wsID, window string) (*assetSvc.ByWorkstationResponse, error)
}

func (m *mockReconciliationService) ListExceptions(ctx context.Context, params *assetSvc.ExceptionListParams) (*base.PageResult, error) {
	if m.ListExceptionsFunc != nil {
		return m.ListExceptionsFunc(ctx, params)
	}
	return nil, errNotImplemented
}

func (m *mockReconciliationService) GetByID(ctx context.Context, id string) (*models.SysDataReconciliation, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}

func (m *mockReconciliationService) ResolveException(ctx context.Context, id, userID string, note *string) error {
	if m.ResolveExceptionFunc != nil {
		return m.ResolveExceptionFunc(ctx, id, userID, note)
	}
	return errNotImplemented
}

func (m *mockReconciliationService) Refresh(ctx context.Context) (int, int, int, int, error) {
	if m.RefreshFunc != nil {
		return m.RefreshFunc(ctx)
	}
	return 0, 0, 0, 0, errNotImplemented
}

func (m *mockReconciliationService) GetByWorkstation(ctx context.Context, wsID, window string) (*assetSvc.ByWorkstationResponse, error) {
	if m.GetByWorkstationFunc != nil {
		return m.GetByWorkstationFunc(ctx, wsID, window)
	}
	return nil, errNotImplemented
}

// ============================================================================
// Test 1: ReconciliationHandler.ListExceptions — happy path
// ============================================================================

func TestReconciliationHandler_ListExceptions_Success(t *testing.T) {
	svc := &mockReconciliationService{
		ListExceptionsFunc: func(ctx context.Context, params *assetSvc.ExceptionListParams) (*base.PageResult, error) {
			assert.Equal(t, "B", params.ConflictType)
			return &base.PageResult{
				List:     []map[string]interface{}{{"id": "x1"}},
				Total:    1,
				Current:  1,
				PageSize: 10,
			}, nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/list", handler: h.ListExceptions}})

	w := httpDo(r, http.MethodPost, "/exception/list", `{"conflictType":"B","current":1,"pageSize":10}`)
	assert.Equal(t, http.StatusOK, w.Code, "ListExceptions success should return 200")
	assert.Contains(t, w.Body.String(), `"x1"`)
}

func TestReconciliationHandler_ListExceptions_BindError(t *testing.T) {
	svc := &mockReconciliationService{}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/list", handler: h.ListExceptions}})

	// invalid JSON body — base.BaseListRequest is loose (only current/pageSize ints).
	// Sending truly malformed JSON to provoke binding failure.
	w := httpDo(r, http.MethodPost, "/exception/list", `{not json`)
	// handler does ShouldBindJSON first → returns 400 on bad json
	assert.Equal(t, http.StatusBadRequest, w.Code, "malformed JSON should return 400")
}

func TestReconciliationHandler_ListExceptions_ServiceError(t *testing.T) {
	svc := &mockReconciliationService{
		ListExceptionsFunc: func(ctx context.Context, params *assetSvc.ExceptionListParams) (*base.PageResult, error) {
			return nil, errors.New("db unavailable")
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/list", handler: h.ListExceptions}})

	w := httpDo(r, http.MethodPost, "/exception/list", `{}`)
	// response.Error(c, int, string) hardcodes HTTPStatus to 400 (response.go quirk),
	// not the int value passed in. See Phase 74-01 SUMMARY "Quirks Discovery" #1.
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"service error maps to 400 via response.Error(c, int, string) quirk (int arg ignored)")
	assert.Contains(t, w.Body.String(), "db unavailable")
}

// ============================================================================
// Test 2: ReconciliationHandler.GetExceptionByID
// ============================================================================

func TestReconciliationHandler_GetByID_Success(t *testing.T) {
	expected := &models.SysDataReconciliation{BaseModel: models.BaseModel{ID: "abc-123"}}
	svc := &mockReconciliationService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.SysDataReconciliation, error) {
			assert.Equal(t, "abc-123", id)
			return expected, nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/:id", handler: h.GetExceptionByID}})

	w := httpDo(r, http.MethodPost, "/exception/abc-123", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"abc-123"`)
}

func TestReconciliationHandler_GetByID_EmptyID(t *testing.T) {
	svc := &mockReconciliationService{}
	h := newReconciliationHandler(svc, newTestCore(t))
	// Use a route that does NOT have :id to simulate empty id branch
	// (g.Param("id") on a path without :id returns "")
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/", handler: h.GetExceptionByID}})

	w := httpDo(r, http.MethodPost, "/exception/", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"empty id should return 400 (handler rejects before service call)")
}

func TestReconciliationHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockReconciliationService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.SysDataReconciliation, error) {
			return nil, nil // service returns nil + nil → handler maps to 404
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/:id", handler: h.GetExceptionByID}})

	w := httpDo(r, http.MethodPost, "/exception/missing-id", `{}`)
	// response.Error(c, http.StatusNotFound, ...) returns 400 due to int-arg quirk
	assert.Equal(t, http.StatusBadRequest, w.Code, "nil result → 400 via response.Error int-arg quirk")
}

func TestReconciliationHandler_GetByID_ServiceError(t *testing.T) {
	svc := &mockReconciliationService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.SysDataReconciliation, error) {
			return nil, errors.New("db error")
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/:id", handler: h.GetExceptionByID}})

	w := httpDo(r, http.MethodPost, "/exception/foo", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "service error maps to 400 via response.Error quirk")
}

// ============================================================================
// Test 3: ReconciliationHandler.ResolveException
// ============================================================================

func TestReconciliationHandler_ResolveException_Success(t *testing.T) {
	called := false
	svc := &mockReconciliationService{
		ResolveExceptionFunc: func(ctx context.Context, id, userID string, note *string) error {
			called = true
			assert.Equal(t, "exc-1", id)
			assert.Equal(t, "user-1", userID)
			require.NotNil(t, note)
			assert.Equal(t, "fixed via test", *note)
			return nil
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := gin.New()
	// Inject user_id via middleware (auth would do this in production)
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})
	r.POST("/exception/:id/resolve", h.ResolveException)

	note := "fixed via test"
	w := httpDo(r, http.MethodPost, "/exception/exc-1/resolve", `{"resolutionNote":"`+note+`"}`)
	assert.True(t, called, "service.ResolveException must be invoked")
	// ResolveException returns 200 via response.Success (gin.H{...}).
	// DB query for workstation_id (reconciliation_normalized) fails silently via applogger.Warnf,
	// but operlog.Record path completes since stubRecorder no-ops.
	assert.Equal(t, http.StatusOK, w.Code, "success path should return 200")
}

func TestReconciliationHandler_ResolveException_NoBody(t *testing.T) {
	// Empty body (no resolutionNote) — handler should still work.
	called := false
	svc := &mockReconciliationService{
		ResolveExceptionFunc: func(ctx context.Context, id, userID string, note *string) error {
			called = true
			assert.Nil(t, note, "note should be nil when no body sent")
			return nil
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/exception/:id/resolve", h.ResolveException)

	// httptest.NewRequest with nil body → ContentLength 0, body branch skipped
	w := httpDo(r, http.MethodPost, "/exception/exc-1/resolve", ``)
	assert.True(t, called, "service must be called even with no body")
	// The invalidation path runs DB scan which may fail in sqlite (table missing)
	// — applogger.Warnf swallows that, response still returns 200.
	_ = w
}

func TestReconciliationHandler_ResolveException_NoUserID(t *testing.T) {
	svc := &mockReconciliationService{
		ResolveExceptionFunc: func(ctx context.Context, id, userID string, note *string) error {
			t.Fatalf("service should not be called when user_id missing")
			return nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := gin.New()
	// No user_id middleware → handler should respond 401
	r.POST("/exception/:id/resolve", h.ResolveException)

	w := httpDo(r, http.MethodPost, "/exception/exc-1/resolve", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "missing user_id 2192 400 via response.Error int-arg quirk")
}

func TestReconciliationHandler_ResolveException_AlreadyResolved(t *testing.T) {
	svc := &mockReconciliationService{
		ResolveExceptionFunc: func(ctx context.Context, id, userID string, note *string) error {
			return errors.New("该异常已标记为已解决")
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/exception/:id/resolve", h.ResolveException)

	w := httpDo(r, http.MethodPost, "/exception/exc-1/resolve", `{}`)
	// handler maps "该异常已标记为已解决" → 400 (see reconciliation_handler.go:155)
	// response.Error(c, int, string) quirk → int arg ignored, returns 400.
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "已标记为已解决")
}

func TestReconciliationHandler_ResolveException_NotExist(t *testing.T) {
	svc := &mockReconciliationService{
		ResolveExceptionFunc: func(ctx context.Context, id, userID string, note *string) error {
			return errors.New("异常不存在")
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/exception/:id/resolve", h.ResolveException)

	w := httpDo(r, http.MethodPost, "/exception/missing/resolve", `{}`)
	// response.Error(c, http.StatusNotFound, ...) returns 400 due to int-arg quirk
	assert.Equal(t, http.StatusBadRequest, w.Code, "异常不存在 → 400 via response.Error int-arg quirk")
}

func TestReconciliationHandler_ResolveException_OtherError(t *testing.T) {
	svc := &mockReconciliationService{
		ResolveExceptionFunc: func(ctx context.Context, id, userID string, note *string) error {
			return errors.New("some other db error")
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/exception/:id/resolve", h.ResolveException)

	w := httpDo(r, http.MethodPost, "/exception/exc-1/resolve", `{}`)
	// 500 → response.Error quirk maps to 400
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 4: ReconciliationHandler.GetByWorkstation
// ============================================================================

func TestReconciliationHandler_GetByWorkstation_Success(t *testing.T) {
	svc := &mockReconciliationService{
		GetByWorkstationFunc: func(ctx context.Context, wsID, window string) (*assetSvc.ByWorkstationResponse, error) {
			assert.Equal(t, "ws-1", wsID)
			assert.Equal(t, "7d", window, "default window should be 7d when omitted")
			return &assetSvc.ByWorkstationResponse{
				Workstation: assetSvc.WorkstationBrief{ID: "ws-1", Name: "Test WS"},
				HealthScore: assetSvc.HealthScore{Total: 5, Score: 80},
				Assets:      []assetSvc.AssetHealthItem{},
				Visible:     false,
			}, nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/by-workstation", handler: h.GetByWorkstation}})

	w := httpDo(r, http.MethodPost, "/by-workstation", `{"workstationId":"ws-1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	// Visible defaults to false (no perm middleware sets user_roles in context).
	assert.Contains(t, w.Body.String(), `"visible":false`)
}

func TestReconciliationHandler_GetByWorkstation_DefaultWindow(t *testing.T) {
	svc := &mockReconciliationService{
		GetByWorkstationFunc: func(ctx context.Context, wsID, window string) (*assetSvc.ByWorkstationResponse, error) {
			// window="" input should default to "7d"
			assert.Equal(t, "7d", window)
			return &assetSvc.ByWorkstationResponse{Assets: []assetSvc.AssetHealthItem{}}, nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/by-workstation", handler: h.GetByWorkstation}})

	w := httpDo(r, http.MethodPost, "/by-workstation", `{"workstationId":"ws-1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReconciliationHandler_GetByWorkstation_BindError(t *testing.T) {
	svc := &mockReconciliationService{}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/by-workstation", handler: h.GetByWorkstation}})

	// Missing required workstationId
	w := httpDo(r, http.MethodPost, "/by-workstation", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "missing workstationId returns 400")
}

func TestReconciliationHandler_GetByWorkstation_ServiceError(t *testing.T) {
	svc := &mockReconciliationService{
		GetByWorkstationFunc: func(ctx context.Context, wsID, window string) (*assetSvc.ByWorkstationResponse, error) {
			return nil, errors.New("svc fail")
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/by-workstation", handler: h.GetByWorkstation}})

	w := httpDo(r, http.MethodPost, "/by-workstation", `{"workstationId":"ws-1"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "service error maps to 400 via response.Error quirk")
}

// ============================================================================
// Test 5: ReconciliationHandler.Refresh
// ============================================================================

func TestReconciliationHandler_Refresh_Success(t *testing.T) {
	svc := &mockReconciliationService{
		RefreshFunc: func(ctx context.Context) (int, int, int, int, error) {
			return 10, 5, 2, 1, nil
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/refresh", handler: h.Refresh}})

	w := httpDo(r, http.MethodPost, "/refresh", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"inserted":10`)
	assert.Contains(t, w.Body.String(), `"skipped":5`)
	assert.Contains(t, w.Body.String(), `"skippedSilence":2`)
	assert.Contains(t, w.Body.String(), `"skippedThrottle":1`)
}

func TestReconciliationHandler_Refresh_Error(t *testing.T) {
	svc := &mockReconciliationService{
		RefreshFunc: func(ctx context.Context) (int, int, int, int, error) {
			return 0, 0, 0, 0, errors.New("mv refresh failed")
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/refresh", handler: h.Refresh}})

	w := httpDo(r, http.MethodPost, "/refresh", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "service error maps to 400 via response.Error quirk")
}

// ============================================================================
// Test 6: WithCore + NewReconciliationHandler nil-safety
// ============================================================================

func TestReconciliationHandler_NewReconciliationHandler(t *testing.T) {
	svc := &mockReconciliationService{}
	h := NewReconciliationHandler(svc)
	require.NotNil(t, h)
	assert.NotNil(t, h.service)
}

func TestReconciliationHandler_WithCore_NilSafe(t *testing.T) {
	svc := &mockReconciliationService{}
	h := NewReconciliationHandler(svc)
	// WithCore on nil receiver returns nil (per source code check)
	var nilH *ReconciliationHandler
	result := nilH.WithCore(newTestCore(t))
	assert.Nil(t, result, "WithCore on nil receiver should return nil")

	// WithCore on non-nil receiver sets core and returns self
	c := newTestCore(t)
	h2 := h.WithCore(c)
	assert.Same(t, h, h2, "WithCore should return the same handler pointer")
	assert.Equal(t, c, h2.core, "core should be set on the handler")
}

// ============================================================================
// Test 7: hasReconciliationPerm — middleware.HasUserPermission integration
// ============================================================================

func TestReconciliationHandler_HasReconciliationPerm_NoCore(t *testing.T) {
	// Without a core, HasUserPermission returns false (gracefully).
	h := &ReconciliationHandler{}
	r := gin.New()
	r.GET("/probe", func(c *gin.Context) {
		ok := h.hasReconciliationPerm(c)
		assert.False(t, ok, "without core, perm check should return false")
		c.Status(http.StatusOK)
	})
	w := httpDo(r, http.MethodGet, "/probe", ``)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Test 8: Module constants (operlog integration)
// ============================================================================

func TestReconciliationHandler_ModuleConstants(t *testing.T) {
	// Static check: constants used by operlog must exist in source.
	assert.Equal(t, "资产对账", ModuleReconciliation,
		"ModuleReconciliation must equal 资产对账 (D-16)")
	assert.Equal(t, "资产对账-例外规则", ModuleReconciliationExceptionRule)
	assert.Equal(t, "资产对账-修复建议", ModuleReconciliationFixSuggestion)
}

// ============================================================================
// Test 9: ResolveException — note pointer path (with note)
// ============================================================================

func TestReconciliationHandler_ResolveException_NotePointer(t *testing.T) {
	capturedNote := (*string)(nil)
	svc := &mockReconciliationService{
		ResolveExceptionFunc: func(ctx context.Context, id, userID string, note *string) error {
			capturedNote = note
			return nil
		},
	}
	c := newTestCore(t)
	h := newReconciliationHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "u1"); c.Next() })
	r.POST("/exception/:id/resolve", h.ResolveException)

	w := httpDo(r, http.MethodPost, "/exception/exc-1/resolve", `{"resolutionNote":"my note"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedNote, "note should be non-nil when body has resolutionNote")
	assert.Equal(t, "my note", *capturedNote)
}

// ============================================================================
// Test 10: ListExceptions — pagination defaults
// ============================================================================

func TestReconciliationHandler_ListExceptions_EmptyParams(t *testing.T) {
	svc := &mockReconciliationService{
		ListExceptionsFunc: func(ctx context.Context, params *assetSvc.ExceptionListParams) (*base.PageResult, error) {
			return &base.PageResult{List: []map[string]interface{}{}, Total: 0, Current: 1, PageSize: 10}, nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/list", handler: h.ListExceptions}})

	// Empty body — base.BaseListRequest defaults are used.
	w := httpDo(r, http.MethodPost, "/exception/list", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Test 11: GetByWorkstation — explicit window override
// ============================================================================

func TestReconciliationHandler_GetByWorkstation_ExplicitWindow(t *testing.T) {
	svc := &mockReconciliationService{
		GetByWorkstationFunc: func(ctx context.Context, wsID, window string) (*assetSvc.ByWorkstationResponse, error) {
			assert.Equal(t, "30d", window, "explicit window should be passed through")
			return &assetSvc.ByWorkstationResponse{Assets: []assetSvc.AssetHealthItem{}}, nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/by-workstation", handler: h.GetByWorkstation}})

	w := httpDo(r, http.MethodPost, "/by-workstation", `{"workstationId":"ws-1","window":"30d"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================================
// Test 12: GetExceptionByID returns data on service success
// ============================================================================

func TestReconciliationHandler_GetByID_ReturnsData(t *testing.T) {
	expectedTime := time.Now()
	expected := &models.SysDataReconciliation{
		BaseModel:    models.BaseModel{ID: "test-id"},
		ConflictType: "B",
		Severity:     "high",
		DetectedAt:   expectedTime,
	}
	svc := &mockReconciliationService{
		GetByIDFunc: func(ctx context.Context, id string) (*models.SysDataReconciliation, error) {
			return expected, nil
		},
	}
	h := newReconciliationHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/exception/:id", handler: h.GetExceptionByID}})

	w := httpDo(r, http.MethodPost, "/exception/test-id", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-id")
	assert.Contains(t, w.Body.String(), `"conflictType":"B"`)
}

// silence unused-import lint if base import only used in mocks
var _ = base.BaseListRequest{}
