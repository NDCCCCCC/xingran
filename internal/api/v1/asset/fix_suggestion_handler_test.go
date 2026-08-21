package asset

// Phase 74 Plan 02 — FixSuggestionHandler unit tests (D-12 strict: test-only)
//
// 7 endpoints to cover:
//   - ListFixSuggestions   (读)
//   - GetByID              (读)
//   - Stats                (读, optional windowDays body)
//   - Accept               (写 → operlog.OperTypeUpdate, 状态机)
//   - Reject               (写 → operlog.OperTypeReject, reason ≥10 char)
//   - Apply                (写 → operlog.OperTypeUpdate, cache invalidate)
//   - Rollback             (写 → operlog.OperTypeReset, 7d 窗口)
//
// 关键 invariants:
//   - Accept/Reject/Apply/Rollback 都必须 inject user_id via gin context (auth middleware)
//   - 错误码映射: "该建议已被处理或不存在" → 409, "拒绝原因至少 10 字符" → 400
//   - D-12: response.Error int-arg quirk maps to 400 (Phase 74-01 SUMMARY Quirks #1)

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	assetSvc "github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// ============================================================================
// mockFixSuggestionService — D-08 mock pattern
// ============================================================================

type mockFixSuggestionService struct {
	ListFixSuggestionsFunc func(ctx context.Context, params *assetSvc.FixSuggestionListParams) (*base.PageResult, error)
	GetByIDFunc            func(ctx context.Context, id string) (*assetSvc.FixSuggestionDetail, error)
	StatsFunc              func(ctx context.Context, windowDays int) (*assetSvc.FixSuggestionStatsResponse, error)
	AcceptFunc             func(ctx context.Context, id, userID string) error
	RejectFunc             func(ctx context.Context, id, userID, reason string) error
	ApplyFunc              func(ctx context.Context, id, userID string) error
	RollbackFunc           func(ctx context.Context, id, userID, reason string) error
	GenerateFixSuggestionsFunc func(ctx context.Context) (int, error)
}

func (m *mockFixSuggestionService) ListFixSuggestions(ctx context.Context, params *assetSvc.FixSuggestionListParams) (*base.PageResult, error) {
	if m.ListFixSuggestionsFunc != nil {
		return m.ListFixSuggestionsFunc(ctx, params)
	}
	return nil, errNotImplemented
}

func (m *mockFixSuggestionService) GetByID(ctx context.Context, id string) (*assetSvc.FixSuggestionDetail, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errNotImplemented
}

func (m *mockFixSuggestionService) Stats(ctx context.Context, windowDays int) (*assetSvc.FixSuggestionStatsResponse, error) {
	if m.StatsFunc != nil {
		return m.StatsFunc(ctx, windowDays)
	}
	return nil, errNotImplemented
}

func (m *mockFixSuggestionService) Accept(ctx context.Context, id, userID string) error {
	if m.AcceptFunc != nil {
		return m.AcceptFunc(ctx, id, userID)
	}
	return errNotImplemented
}

func (m *mockFixSuggestionService) Reject(ctx context.Context, id, userID, reason string) error {
	if m.RejectFunc != nil {
		return m.RejectFunc(ctx, id, userID, reason)
	}
	return errNotImplemented
}

func (m *mockFixSuggestionService) Apply(ctx context.Context, id, userID string) error {
	if m.ApplyFunc != nil {
		return m.ApplyFunc(ctx, id, userID)
	}
	return errNotImplemented
}

func (m *mockFixSuggestionService) Rollback(ctx context.Context, id, userID, reason string) error {
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx, id, userID, reason)
	}
	return errNotImplemented
}

func (m *mockFixSuggestionService) GenerateFixSuggestions(ctx context.Context) (int, error) {
	if m.GenerateFixSuggestionsFunc != nil {
		return m.GenerateFixSuggestionsFunc(ctx)
	}
	return 0, errNotImplemented
}

// newFixSuggestionHandler constructs a *FixSuggestionHandler with optional core.
func newFixSuggestionHandler(svc assetSvc.FixSuggestionService, c *core.Core) *FixSuggestionHandler {
	h := NewFixSuggestionHandler(svc)
	if c != nil {
		h.WithCore(c)
	}
	return h
}

// ============================================================================
// Test 1: ListFixSuggestions — happy path
// ============================================================================

func TestFixSuggestionHandler_List_Success(t *testing.T) {
	svc := &mockFixSuggestionService{
		ListFixSuggestionsFunc: func(ctx context.Context, params *assetSvc.FixSuggestionListParams) (*base.PageResult, error) {
			assert.Equal(t, "pending", params.FixStatus)
			return &base.PageResult{
				List:     []map[string]interface{}{{"id": "sug-1"}},
				Total:    1,
				Current:  1,
				PageSize: 10,
			}, nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/list", handler: h.ListFixSuggestions}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/list", `{"fixStatus":"pending","current":1,"pageSize":10}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sug-1"`)
}

func TestFixSuggestionHandler_List_BindError(t *testing.T) {
	svc := &mockFixSuggestionService{}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/list", handler: h.ListFixSuggestions}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/list", `{not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFixSuggestionHandler_List_ServiceError(t *testing.T) {
	svc := &mockFixSuggestionService{
		ListFixSuggestionsFunc: func(ctx context.Context, params *assetSvc.FixSuggestionListParams) (*base.PageResult, error) {
			return nil, errors.New("svc fail")
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/list", handler: h.ListFixSuggestions}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/list", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 2: GetByID
// ============================================================================

func TestFixSuggestionHandler_GetByID_Success(t *testing.T) {
	expected := &assetSvc.FixSuggestionDetail{}
	svc := &mockFixSuggestionService{
		GetByIDFunc: func(ctx context.Context, id string) (*assetSvc.FixSuggestionDetail, error) {
			assert.Equal(t, "sug-1", id)
			return expected, nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/:id", handler: h.GetByID}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFixSuggestionHandler_GetByID_NotFound(t *testing.T) {
	svc := &mockFixSuggestionService{
		GetByIDFunc: func(ctx context.Context, id string) (*assetSvc.FixSuggestionDetail, error) {
			return nil, nil // service returns nil → handler maps to 404 (but quirk returns 400)
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/:id", handler: h.GetByID}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/missing", `{}`)
	// response.Error(c, http.StatusNotFound, ...) returns 400 due to int-arg quirk (Phase 74-01 SUMMARY Quirks #1)
	assert.Equal(t, http.StatusBadRequest, w.Code, "nil result → 400 via response.Error int-arg quirk")
}

func TestFixSuggestionHandler_GetByID_Error(t *testing.T) {
	svc := &mockFixSuggestionService{
		GetByIDFunc: func(ctx context.Context, id string) (*assetSvc.FixSuggestionDetail, error) {
			return nil, errors.New("svc fail")
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/:id", handler: h.GetByID}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/abc", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 3: Stats
// ============================================================================

func TestFixSuggestionHandler_Stats_WithWindow(t *testing.T) {
	svc := &mockFixSuggestionService{
		StatsFunc: func(ctx context.Context, windowDays int) (*assetSvc.FixSuggestionStatsResponse, error) {
			assert.Equal(t, 14, windowDays)
			return &assetSvc.FixSuggestionStatsResponse{
				WindowDays: 14,
				Pending:    10,
				PendingAll: 25,
				Accepted:   5,
				Rejected:   1,
				Applied:    3,
				MisFixRate: 0.05,
			}, nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/stats", handler: h.Stats}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/stats", `{"windowDays":14}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"pending":10`)
	assert.Contains(t, w.Body.String(), `"windowDays":14`)
}

func TestFixSuggestionHandler_Stats_DefaultWindow(t *testing.T) {
	svc := &mockFixSuggestionService{
		StatsFunc: func(ctx context.Context, windowDays int) (*assetSvc.FixSuggestionStatsResponse, error) {
			assert.Equal(t, 7, windowDays, "windowDays should default to 7 when omitted")
			return &assetSvc.FixSuggestionStatsResponse{WindowDays: 7}, nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/stats", handler: h.Stats}})

	// No body → ContentLength=0 → bind skipped → default applied
	w := httpDo(r, http.MethodPost, "/fix-suggestion/stats", ``)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFixSuggestionHandler_Stats_BindErrorFallback(t *testing.T) {
	// Per fix_suggestion_handler.go Stats handler — bind error returns 400 (NOT fallback).
	// This differs from Summary handler (statistics_handler.go) which swallows bind errors.
	svc := &mockFixSuggestionService{
		StatsFunc: func(ctx context.Context, windowDays int) (*assetSvc.FixSuggestionStatsResponse, error) {
			t.Fatalf("service should not be called on bind error (handler returns 400 first)")
			return nil, nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/stats", handler: h.Stats}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/stats", `{not json`)
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"Stats bind error returns 400 (response.Error int-arg quirk maps to 400)")
}

func TestFixSuggestionHandler_Stats_Error(t *testing.T) {
	svc := &mockFixSuggestionService{
		StatsFunc: func(ctx context.Context, windowDays int) (*assetSvc.FixSuggestionStatsResponse, error) {
			return nil, errors.New("stats failed")
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/fix-suggestion/stats", handler: h.Stats}})

	w := httpDo(r, http.MethodPost, "/fix-suggestion/stats", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 4: Accept
// ============================================================================

func TestFixSuggestionHandler_Accept_Success(t *testing.T) {
	called := false
	svc := &mockFixSuggestionService{
		AcceptFunc: func(ctx context.Context, id, userID string) error {
			called = true
			assert.Equal(t, "sug-1", id)
			assert.Equal(t, "user-1", userID)
			return nil
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/accept", h.Accept)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/accept", `{}`)
	assert.True(t, called, "service.Accept must be called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"sug-1"`)
	assert.Contains(t, w.Body.String(), `"acceptedBy":"user-1"`)
}

func TestFixSuggestionHandler_Accept_NoUserID(t *testing.T) {
	svc := &mockFixSuggestionService{
		AcceptFunc: func(ctx context.Context, id, userID string) error {
			t.Fatalf("service should not be called when user_id missing")
			return nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := gin.New()
	// No user_id middleware
	r.POST("/fix-suggestion/:id/accept", h.Accept)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/accept", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "response.Error int-arg quirk: intended 401 2192 400")
}

func TestFixSuggestionHandler_Accept_AlreadyProcessed(t *testing.T) {
	svc := &mockFixSuggestionService{
		AcceptFunc: func(ctx context.Context, id, userID string) error {
			return errors.New("该建议已被处理或不存在")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/accept", h.Accept)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/accept", `{}`)
	// handler maps "该建议已被处理或不存在" → 409, but response.Error quirk → 400.
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "已被处理")
}

func TestFixSuggestionHandler_Accept_OtherError(t *testing.T) {
	svc := &mockFixSuggestionService{
		AcceptFunc: func(ctx context.Context, id, userID string) error {
			return errors.New("some other error")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/accept", h.Accept)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/accept", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 5: Reject
// ============================================================================

func TestFixSuggestionHandler_Reject_Success(t *testing.T) {
	called := false
	svc := &mockFixSuggestionService{
		RejectFunc: func(ctx context.Context, id, userID, reason string) error {
			called = true
			assert.Equal(t, "sug-1", id)
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "this is a valid reason", reason)
			return nil
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/reject", h.Reject)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/reject", `{"rejectionReason":"this is a valid reason"}`)
	assert.True(t, called, "service.Reject must be called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"rejectedBy":"user-1"`)
}

func TestFixSuggestionHandler_Reject_BindError(t *testing.T) {
	svc := &mockFixSuggestionService{
		RejectFunc: func(ctx context.Context, id, userID, reason string) error {
			t.Fatalf("service should not be called on bind error")
			return nil
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/reject", h.Reject)

	// Missing required rejectionReason
	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/reject", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "missing rejectionReason → 400")
}

func TestFixSuggestionHandler_Reject_AlreadyProcessed(t *testing.T) {
	svc := &mockFixSuggestionService{
		RejectFunc: func(ctx context.Context, id, userID, reason string) error {
			return errors.New("该建议已被处理或不存在")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/reject", h.Reject)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/reject", `{"rejectionReason":"valid reason text"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFixSuggestionHandler_Reject_ShortReason(t *testing.T) {
	svc := &mockFixSuggestionService{
		RejectFunc: func(ctx context.Context, id, userID, reason string) error {
			return errors.New("拒绝原因至少 10 字符")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/reject", h.Reject)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/reject", `{"rejectionReason":"valid reason text"}`)
	// "拒绝原因至少 10 字符" → 400 (handler check)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "拒绝原因")
}

func TestFixSuggestionHandler_Reject_NoUserID(t *testing.T) {
	svc := &mockFixSuggestionService{
		RejectFunc: func(ctx context.Context, id, userID, reason string) error {
			t.Fatalf("service should not be called when user_id missing")
			return nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := gin.New()
	r.POST("/fix-suggestion/:id/reject", h.Reject)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/reject", `{"rejectionReason":"valid reason text"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "response.Error int-arg quirk: intended 401 2192 400")
}

// ============================================================================
// Test 6: Apply
// ============================================================================

func TestFixSuggestionHandler_Apply_Success(t *testing.T) {
	called := false
	svc := &mockFixSuggestionService{
		ApplyFunc: func(ctx context.Context, id, userID string) error {
			called = true
			assert.Equal(t, "sug-1", id)
			assert.Equal(t, "user-1", userID)
			return nil
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/apply", h.Apply)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/apply", `{}`)
	assert.True(t, called, "service.Apply must be called")
	// Apply path: service success → cache invalidation (DB scan may fail silently) → operlog → success
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFixSuggestionHandler_Apply_NotInAcceptedState(t *testing.T) {
	svc := &mockFixSuggestionService{
		ApplyFunc: func(ctx context.Context, id, userID string) error {
			return errors.New("该建议不存在或未处于 accepted 状态")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/apply", h.Apply)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/apply", `{}`)
	// 409 → response.Error quirk → 400
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFixSuggestionHandler_Apply_NoUserID(t *testing.T) {
	svc := &mockFixSuggestionService{
		ApplyFunc: func(ctx context.Context, id, userID string) error {
			t.Fatalf("service should not be called when user_id missing")
			return nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := gin.New()
	r.POST("/fix-suggestion/:id/apply", h.Apply)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/apply", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "response.Error int-arg quirk: intended 401 2192 400")
}

// ============================================================================
// Test 7: Rollback
// ============================================================================

func TestFixSuggestionHandler_Rollback_Success(t *testing.T) {
	called := false
	svc := &mockFixSuggestionService{
		RollbackFunc: func(ctx context.Context, id, userID, reason string) error {
			called = true
			assert.Equal(t, "sug-1", id)
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "wrong fix reverted", reason)
			return nil
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/rollback", h.Rollback)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/rollback", `{"rollbackReason":"wrong fix reverted"}`)
	assert.True(t, called, "service.Rollback must be called")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFixSuggestionHandler_Rollback_BindError(t *testing.T) {
	svc := &mockFixSuggestionService{
		RollbackFunc: func(ctx context.Context, id, userID, reason string) error {
			t.Fatalf("service should not be called on bind error")
			return nil
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/rollback", h.Rollback)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/rollback", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "missing rollbackReason → 400")
}

func TestFixSuggestionHandler_Rollback_NotInAppliedState(t *testing.T) {
	svc := &mockFixSuggestionService{
		RollbackFunc: func(ctx context.Context, id, userID, reason string) error {
			return errors.New("该建议不存在或未处于 applied 状态")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/rollback", h.Rollback)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/rollback", `{"rollbackReason":"wrong fix reverted"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFixSuggestionHandler_Rollback_WindowExpired(t *testing.T) {
	svc := &mockFixSuggestionService{
		RollbackFunc: func(ctx context.Context, id, userID, reason string) error {
			return errors.New("回滚窗口已过(7d),不允许回滚")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/rollback", h.Rollback)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/rollback", `{"rollbackReason":"wrong fix reverted"}`)
	// 400 (handler check on window expired)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "窗口")
}

func TestFixSuggestionHandler_Rollback_ShortReason(t *testing.T) {
	svc := &mockFixSuggestionService{
		RollbackFunc: func(ctx context.Context, id, userID, reason string) error {
			return errors.New("回滚原因至少 10 字符")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/rollback", h.Rollback)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/rollback", `{"rollbackReason":"valid reason text"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFixSuggestionHandler_Rollback_OtherError(t *testing.T) {
	svc := &mockFixSuggestionService{
		RollbackFunc: func(ctx context.Context, id, userID, reason string) error {
			return errors.New("some other error")
		},
	}
	c := newTestCore(t)
	h := newFixSuggestionHandler(svc, c)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() })
	r.POST("/fix-suggestion/:id/rollback", h.Rollback)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/rollback", `{"rollbackReason":"valid reason text"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFixSuggestionHandler_Rollback_NoUserID(t *testing.T) {
	svc := &mockFixSuggestionService{
		RollbackFunc: func(ctx context.Context, id, userID, reason string) error {
			t.Fatalf("service should not be called when user_id missing")
			return nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := gin.New()
	r.POST("/fix-suggestion/:id/rollback", h.Rollback)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/rollback", `{"rollbackReason":"valid reason text"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "response.Error int-arg quirk: intended 401 2192 400")
}

// ============================================================================
// Test 8: Lifecycle (New + WithCore)
// ============================================================================

func TestFixSuggestionHandler_NewHandler(t *testing.T) {
	svc := &mockFixSuggestionService{}
	h := NewFixSuggestionHandler(svc)
	require.NotNil(t, h)
	assert.NotNil(t, h.service)
}

func TestFixSuggestionHandler_WithCore_NilSafe(t *testing.T) {
	svc := &mockFixSuggestionService{}
	h := NewFixSuggestionHandler(svc)

	// Nil receiver returns nil per WithCore pattern.
	var nilH *FixSuggestionHandler
	result := nilH.WithCore(newTestCore(t))
	assert.Nil(t, result)

	// Non-nil receiver sets core and returns self.
	c := newTestCore(t)
	h2 := h.WithCore(c)
	assert.Same(t, h, h2)
	assert.Equal(t, c, h2.core)
}

// ============================================================================
// Test 9: getUserID — empty user_id branch
// ============================================================================

func TestFixSuggestionHandler_Accept_EmptyUserID(t *testing.T) {
	svc := &mockFixSuggestionService{
		AcceptFunc: func(ctx context.Context, id, userID string) error {
			t.Fatalf("service should not be called when user_id is empty")
			return nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := gin.New()
	// user_id is empty string (not absent)
	r.Use(func(c *gin.Context) { c.Set("user_id", ""); c.Next() })
	r.POST("/fix-suggestion/:id/accept", h.Accept)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/accept", `{}`)
	// Empty user_id → handler returns 401 "当前用户ID格式错误"
	assert.Equal(t, http.StatusBadRequest, w.Code, "response.Error int-arg quirk: intended 401 2192 400")
}

func TestFixSuggestionHandler_Accept_WrongUserIDType(t *testing.T) {
	svc := &mockFixSuggestionService{
		AcceptFunc: func(ctx context.Context, id, userID string) error {
			t.Fatalf("service should not be called when user_id wrong type")
			return nil
		},
	}
	h := newFixSuggestionHandler(svc, newTestCore(t))
	r := gin.New()
	// user_id is int, not string → type assertion fails
	r.Use(func(c *gin.Context) { c.Set("user_id", 123); c.Next() })
	r.POST("/fix-suggestion/:id/accept", h.Accept)

	w := httpDo(r, http.MethodPost, "/fix-suggestion/sug-1/accept", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "response.Error int-arg quirk: intended 401 2192 400")
}
