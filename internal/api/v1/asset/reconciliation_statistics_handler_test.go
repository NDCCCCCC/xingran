package asset

// Phase 74 Plan 02 — StatisticsHandler unit tests (D-12 strict: test-only)
//
// 6 endpoints to cover:
//   - Summary              (5 KPI cards, body days param)
//   - ByConflictType       (A-F 6 buckets)
//   - BySeverity           (low/medium/high/critical 4 buckets)
//   - HealthTrend          (PG only, days window)
//   - TopUnresolved        (Top N, limit param)
//   - ExceptionRuleStats   (R3接入后)

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	assetSvc "github.com/xingran-next/xingran-go-backend/internal/services/asset"
)

// ============================================================================
// mockReconciliationStatistics — D-08 mock pattern (Phase 73-01 reference)
// ============================================================================

type mockReconciliationStatistics struct {
	SummaryFunc           func(ctx context.Context, filter assetSvc.StatsFilter) (*assetSvc.SummaryResult, error)
	ByConflictTypeFunc    func(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error)
	BySeverityFunc        func(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error)
	HealthTrendFunc       func(ctx context.Context, filter assetSvc.StatsFilter) ([]assetSvc.TrendPoint, error)
	TopUnresolvedFunc     func(ctx context.Context, limit int) ([]assetSvc.ExceptionSummary, error)
	ExceptionRuleStatsFunc func(ctx context.Context) ([]assetSvc.RuleStats, error)
}

func (m *mockReconciliationStatistics) Summary(ctx context.Context, filter assetSvc.StatsFilter) (*assetSvc.SummaryResult, error) {
	if m.SummaryFunc != nil {
		return m.SummaryFunc(ctx, filter)
	}
	return nil, errNotImplemented
}

func (m *mockReconciliationStatistics) ByConflictType(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error) {
	if m.ByConflictTypeFunc != nil {
		return m.ByConflictTypeFunc(ctx, filter)
	}
	return nil, errNotImplemented
}

func (m *mockReconciliationStatistics) BySeverity(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error) {
	if m.BySeverityFunc != nil {
		return m.BySeverityFunc(ctx, filter)
	}
	return nil, errNotImplemented
}

func (m *mockReconciliationStatistics) HealthTrend(ctx context.Context, filter assetSvc.StatsFilter) ([]assetSvc.TrendPoint, error) {
	if m.HealthTrendFunc != nil {
		return m.HealthTrendFunc(ctx, filter)
	}
	return nil, errNotImplemented
}

func (m *mockReconciliationStatistics) TopUnresolved(ctx context.Context, limit int) ([]assetSvc.ExceptionSummary, error) {
	if m.TopUnresolvedFunc != nil {
		return m.TopUnresolvedFunc(ctx, limit)
	}
	return nil, errNotImplemented
}

func (m *mockReconciliationStatistics) ExceptionRuleStats(ctx context.Context) ([]assetSvc.RuleStats, error) {
	if m.ExceptionRuleStatsFunc != nil {
		return m.ExceptionRuleStatsFunc(ctx)
	}
	return nil, errNotImplemented
}

// newStatisticsHandler constructs a *StatisticsHandler with optional core.
func newStatisticsHandler(svc assetSvc.ReconciliationStatistics, c *core.Core) *StatisticsHandler {
	h := NewStatisticsHandler(svc)
	if c != nil {
		h.WithCore(c)
	}
	return h
}

// ============================================================================
// Test 1: Summary — happy path
// ============================================================================

func TestStatisticsHandler_Summary_Success(t *testing.T) {
	svc := &mockReconciliationStatistics{
		SummaryFunc: func(ctx context.Context, filter assetSvc.StatsFilter) (*assetSvc.SummaryResult, error) {
			assert.Equal(t, 7, filter.Days)
			return &assetSvc.SummaryResult{
				TotalAssets:      100,
				OpenExceptions:   15,
				CriticalOpen:     3,
				Last7dNew:        5,
				TopConflictType:  "B",
				TopConflictCount: 8,
			}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/summary", handler: h.Summary}})

	w := httpDo(r, http.MethodPost, "/statistics/summary", `{"days":7}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"totalAssets":100`)
	assert.Contains(t, w.Body.String(), `"topConflictType":"B"`)
}

func TestStatisticsHandler_Summary_BindErrorFallsBack(t *testing.T) {
	// Per reconciliation_statistics_handler.go:53 — bind error falls back to days=0.
	svc := &mockReconciliationStatistics{
		SummaryFunc: func(ctx context.Context, filter assetSvc.StatsFilter) (*assetSvc.SummaryResult, error) {
			assert.Equal(t, 0, filter.Days, "days should fall back to 0 on bind error")
			return &assetSvc.SummaryResult{}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/summary", handler: h.Summary}})

	w := httpDo(r, http.MethodPost, "/statistics/summary", `{not json`)
	assert.Equal(t, http.StatusOK, w.Code, "bind error → fallback to days=0 → still 200")
}

func TestStatisticsHandler_Summary_ServiceError(t *testing.T) {
	svc := &mockReconciliationStatistics{
		SummaryFunc: func(ctx context.Context, filter assetSvc.StatsFilter) (*assetSvc.SummaryResult, error) {
			return nil, errors.New("db down")
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/summary", handler: h.Summary}})

	w := httpDo(r, http.MethodPost, "/statistics/summary", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "response.Error int-arg quirk maps to 400")
}

// ============================================================================
// Test 2: ByConflictType
// ============================================================================

func TestStatisticsHandler_ByConflictType_Success(t *testing.T) {
	svc := &mockReconciliationStatistics{
		ByConflictTypeFunc: func(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error) {
			return map[string]int64{"A": 5, "B": 10, "C": 0, "D": 1, "E": 0, "F": 2}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/by-conflict-type", handler: h.ByConflictType}})

	w := httpDo(r, http.MethodPost, "/statistics/by-conflict-type", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"B":10`)
	assert.Contains(t, w.Body.String(), `"A":5`)
}

func TestStatisticsHandler_ByConflictType_Error(t *testing.T) {
	svc := &mockReconciliationStatistics{
		ByConflictTypeFunc: func(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error) {
			return nil, errors.New("query failed")
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/by-conflict-type", handler: h.ByConflictType}})

	w := httpDo(r, http.MethodPost, "/statistics/by-conflict-type", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 3: BySeverity
// ============================================================================

func TestStatisticsHandler_BySeverity_Success(t *testing.T) {
	svc := &mockReconciliationStatistics{
		BySeverityFunc: func(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error) {
			return map[string]int64{"low": 5, "medium": 3, "high": 2, "critical": 1}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/by-severity", handler: h.BySeverity}})

	w := httpDo(r, http.MethodPost, "/statistics/by-severity", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"critical":1`)
}

func TestStatisticsHandler_BySeverity_Error(t *testing.T) {
	svc := &mockReconciliationStatistics{
		BySeverityFunc: func(ctx context.Context, filter assetSvc.StatsFilter) (map[string]int64, error) {
			return nil, errors.New("severity query failed")
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/by-severity", handler: h.BySeverity}})

	w := httpDo(r, http.MethodPost, "/statistics/by-severity", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 4: HealthTrend
// ============================================================================

func TestStatisticsHandler_HealthTrend_Success(t *testing.T) {
	svc := &mockReconciliationStatistics{
		HealthTrendFunc: func(ctx context.Context, filter assetSvc.StatsFilter) ([]assetSvc.TrendPoint, error) {
			assert.Equal(t, 30, filter.Days)
			return []assetSvc.TrendPoint{
				{Date: "2026-08-01", OpenCount: 5, CriticalCount: 1, NewCount: 2},
				{Date: "2026-08-02", OpenCount: 4, CriticalCount: 0, NewCount: 1},
			}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/health-trend", handler: h.HealthTrend}})

	w := httpDo(r, http.MethodPost, "/statistics/health-trend", `{"days":30}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "2026-08-01")
	assert.Contains(t, w.Body.String(), `"criticalCount":1`)
}

func TestStatisticsHandler_HealthTrend_BindErrorFallback(t *testing.T) {
	svc := &mockReconciliationStatistics{
		HealthTrendFunc: func(ctx context.Context, filter assetSvc.StatsFilter) ([]assetSvc.TrendPoint, error) {
			assert.Equal(t, 0, filter.Days, "bind error → days=0 fallback")
			return []assetSvc.TrendPoint{}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/health-trend", handler: h.HealthTrend}})

	w := httpDo(r, http.MethodPost, "/statistics/health-trend", `{not json`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStatisticsHandler_HealthTrend_Error(t *testing.T) {
	svc := &mockReconciliationStatistics{
		HealthTrendFunc: func(ctx context.Context, filter assetSvc.StatsFilter) ([]assetSvc.TrendPoint, error) {
			return nil, errors.New("PG FILTER unsupported")
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/health-trend", handler: h.HealthTrend}})

	w := httpDo(r, http.MethodPost, "/statistics/health-trend", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 5: TopUnresolved
// ============================================================================

func TestStatisticsHandler_TopUnresolved_Success(t *testing.T) {
	svc := &mockReconciliationStatistics{
		TopUnresolvedFunc: func(ctx context.Context, limit int) ([]assetSvc.ExceptionSummary, error) {
			assert.Equal(t, 5, limit)
			return []assetSvc.ExceptionSummary{
				{ID: "exc-1", AssetCode: "AST001", ConflictType: "B", Severity: "critical", DetectedAt: time.Now(), DaysUnresolved: 12},
				{ID: "exc-2", AssetCode: "AST002", ConflictType: "A", Severity: "high", DetectedAt: time.Now(), DaysUnresolved: 7},
			}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/top-unresolved", handler: h.TopUnresolved}})

	w := httpDo(r, http.MethodPost, "/statistics/top-unresolved", `{"limit":5}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"AST001"`)
	assert.Contains(t, w.Body.String(), `"daysUnresolved":12`)
}

func TestStatisticsHandler_TopUnresolved_BindErrorFallback(t *testing.T) {
	svc := &mockReconciliationStatistics{
		TopUnresolvedFunc: func(ctx context.Context, limit int) ([]assetSvc.ExceptionSummary, error) {
			assert.Equal(t, 0, limit, "bind error → limit=0 fallback")
			return []assetSvc.ExceptionSummary{}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/top-unresolved", handler: h.TopUnresolved}})

	w := httpDo(r, http.MethodPost, "/statistics/top-unresolved", `{not json`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStatisticsHandler_TopUnresolved_Error(t *testing.T) {
	svc := &mockReconciliationStatistics{
		TopUnresolvedFunc: func(ctx context.Context, limit int) ([]assetSvc.ExceptionSummary, error) {
			return nil, errors.New("top unresolved failed")
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/top-unresolved", handler: h.TopUnresolved}})

	w := httpDo(r, http.MethodPost, "/statistics/top-unresolved", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 6: ExceptionRuleStats
// ============================================================================

func TestStatisticsHandler_ExceptionRuleStats_Success(t *testing.T) {
	svc := &mockReconciliationStatistics{
		ExceptionRuleStatsFunc: func(ctx context.Context) ([]assetSvc.RuleStats, error) {
			return []assetSvc.RuleStats{
				{RuleID: "rule-1", RuleName: "Office subnet", MatchedCount: 25},
				{RuleID: "rule-2", RuleName: "Dev subnet", MatchedCount: 8},
			}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/exception-rule-stats", handler: h.ExceptionRuleStats}})

	w := httpDo(r, http.MethodPost, "/statistics/exception-rule-stats", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"rule-1"`)
	assert.Contains(t, w.Body.String(), `"matchedCount":25`)
}

func TestStatisticsHandler_ExceptionRuleStats_EmptySlice(t *testing.T) {
	svc := &mockReconciliationStatistics{
		ExceptionRuleStatsFunc: func(ctx context.Context) ([]assetSvc.RuleStats, error) {
			// R1 default: R3 接入前返回空 slice
			return []assetSvc.RuleStats{}, nil
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/exception-rule-stats", handler: h.ExceptionRuleStats}})

	w := httpDo(r, http.MethodPost, "/statistics/exception-rule-stats", `{}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStatisticsHandler_ExceptionRuleStats_Error(t *testing.T) {
	svc := &mockReconciliationStatistics{
		ExceptionRuleStatsFunc: func(ctx context.Context) ([]assetSvc.RuleStats, error) {
			return nil, errors.New("rule stats failed")
		},
	}
	h := newStatisticsHandler(svc, newTestCore(t))
	r := mountRouter([]routeMount{{method: http.MethodPost, path: "/statistics/exception-rule-stats", handler: h.ExceptionRuleStats}})

	w := httpDo(r, http.MethodPost, "/statistics/exception-rule-stats", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================================
// Test 7: NewStatisticsHandler + WithCore lifecycle
// ============================================================================

func TestStatisticsHandler_NewHandler(t *testing.T) {
	svc := &mockReconciliationStatistics{}
	h := NewStatisticsHandler(svc)
	require.NotNil(t, h)
	assert.NotNil(t, h.service)
}

func TestStatisticsHandler_WithCore_NilSafe(t *testing.T) {
	svc := &mockReconciliationStatistics{}
	h := NewStatisticsHandler(svc)

	// Nil receiver returns nil per WithCore pattern.
	var nilH *StatisticsHandler
	result := nilH.WithCore(newTestCore(t))
	assert.Nil(t, result)

	// Non-nil receiver sets core and returns self.
	c := newTestCore(t)
	h2 := h.WithCore(c)
	assert.Same(t, h, h2)
	assert.Equal(t, c, h2.core)
}
