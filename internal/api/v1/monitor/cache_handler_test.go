package monitor

// =============================================================================
// CacheHandler 测试 (Phase 72 CORE-02)
// =============================================================================

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// ==================== Mock CacheService ====================

type mockCacheService struct {
	monitorServices.CacheService

	GetCacheListFunc       func(ctx context.Context, params monitorServices.CacheListParams) ([]models.CacheInfo, int64, error)
	GetCacheInfoFunc       func(ctx context.Context, key string) (*models.CacheInfo, error)
	OperateCacheFunc       func(ctx context.Context, params monitorServices.CacheOperateParams) (interface{}, error)
	BatchOperateCacheFunc  func(ctx context.Context, params monitorServices.CacheBatchOperateParams) (map[string]interface{}, error)
	ClearCacheFunc         func(ctx context.Context) error
	GetCacheStatsFunc      func(ctx context.Context, params monitorServices.CacheStatsParams) ([]models.CacheStats, int64, error)
	GetCacheMonitorFunc    func(ctx context.Context) (map[string]interface{}, error)
	ExportCacheFunc        func(ctx context.Context, params monitorServices.CacheExportParams) ([]models.CacheInfo, error)
	GetCacheConfigsFunc    func(ctx context.Context) (map[string]monitorServices.CacheConfigInfo, map[string]int, error)
	UpdateCacheConfigFunc  func(ctx context.Context, key string, value int) error
	ReloadCacheConfigsFunc func(ctx context.Context) error
}

func (m *mockCacheService) GetCacheList(ctx context.Context, params monitorServices.CacheListParams) ([]models.CacheInfo, int64, error) {
	if m.GetCacheListFunc != nil {
		return m.GetCacheListFunc(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockCacheService) GetCacheInfo(ctx context.Context, key string) (*models.CacheInfo, error) {
	if m.GetCacheInfoFunc != nil {
		return m.GetCacheInfoFunc(ctx, key)
	}
	return nil, nil
}
func (m *mockCacheService) OperateCache(ctx context.Context, params monitorServices.CacheOperateParams) (interface{}, error) {
	if m.OperateCacheFunc != nil {
		return m.OperateCacheFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockCacheService) BatchOperateCache(ctx context.Context, params monitorServices.CacheBatchOperateParams) (map[string]interface{}, error) {
	if m.BatchOperateCacheFunc != nil {
		return m.BatchOperateCacheFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockCacheService) ClearCache(ctx context.Context) error {
	if m.ClearCacheFunc != nil {
		return m.ClearCacheFunc(ctx)
	}
	return nil
}
func (m *mockCacheService) GetCacheStats(ctx context.Context, params monitorServices.CacheStatsParams) ([]models.CacheStats, int64, error) {
	if m.GetCacheStatsFunc != nil {
		return m.GetCacheStatsFunc(ctx, params)
	}
	return nil, 0, nil
}
func (m *mockCacheService) GetCacheMonitor(ctx context.Context) (map[string]interface{}, error) {
	if m.GetCacheMonitorFunc != nil {
		return m.GetCacheMonitorFunc(ctx)
	}
	return nil, nil
}
func (m *mockCacheService) ExportCache(ctx context.Context, params monitorServices.CacheExportParams) ([]models.CacheInfo, error) {
	if m.ExportCacheFunc != nil {
		return m.ExportCacheFunc(ctx, params)
	}
	return nil, nil
}
func (m *mockCacheService) GetCacheConfigs(ctx context.Context) (map[string]monitorServices.CacheConfigInfo, map[string]int, error) {
	if m.GetCacheConfigsFunc != nil {
		return m.GetCacheConfigsFunc(ctx)
	}
	return nil, nil, nil
}
func (m *mockCacheService) UpdateCacheConfig(ctx context.Context, key string, value int) error {
	if m.UpdateCacheConfigFunc != nil {
		return m.UpdateCacheConfigFunc(ctx, key, value)
	}
	return nil
}
func (m *mockCacheService) ReloadCacheConfigs(ctx context.Context) error {
	if m.ReloadCacheConfigsFunc != nil {
		return m.ReloadCacheConfigsFunc(ctx)
	}
	return nil
}

var _ monitorServices.CacheService = (*mockCacheService)(nil)

// ==================== Test Infrastructure ====================

func newTestCtx(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func parseResp(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

func setupCacheHandler(mock *mockCacheService) *CacheHandler {
	// core.DB = nil 让 operlog.Record / saveCacheOperationLog 走 nil 守卫
	return NewCacheHandler(mock, &core.Core{
		CoreInfra:    &core.CoreInfra{DB: nil},
		CoreServices: &core.CoreServices{},
	})
}

// ==================== Test Cases ====================

func TestCacheHandler_GetCacheList_Empty(t *testing.T) {
	mock := &mockCacheService{
		GetCacheListFunc: func(ctx context.Context, params monitorServices.CacheListParams) ([]models.CacheInfo, int64, error) {
			return []models.CacheInfo{}, 0, nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/list", map[string]interface{}{})
	h.GetCacheList(c)
	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResp(t, w.Body.Bytes())
	assert.EqualValues(t, 0, resp["code"])
}

func TestCacheHandler_GetCacheList_Error(t *testing.T) {
	mock := &mockCacheService{
		GetCacheListFunc: func(ctx context.Context, params monitorServices.CacheListParams) ([]models.CacheInfo, int64, error) {
			return nil, 0, errors.New("redis down")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/list", map[string]interface{}{})
	h.GetCacheList(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheInfo_EmptyKey(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("GET", "/", nil)
	c.Params = gin.Params{{Key: "key", Value: ""}}
	h.GetCacheInfo(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheInfo_NotFound(t *testing.T) {
	mock := &mockCacheService{
		GetCacheInfoFunc: func(ctx context.Context, key string) (*models.CacheInfo, error) {
			return nil, monitorServices.ErrCacheNotFound
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("GET", "/k", nil)
	c.Params = gin.Params{{Key: "key", Value: "missing"}}
	h.GetCacheInfo(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheInfo_Success(t *testing.T) {
	mock := &mockCacheService{
		GetCacheInfoFunc: func(ctx context.Context, key string) (*models.CacheInfo, error) {
			return &models.CacheInfo{Key: key, Value: "v", Size: 10}, nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("GET", "/k", nil)
	c.Params = gin.Params{{Key: "key", Value: "k1"}}
	h.GetCacheInfo(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_OperateCache_InvalidOp(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/op", map[string]interface{}{
		"key":       "k1",
		"operation": "invalid",
	})
	h.OperateCache(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCacheHandler_OperateCache_GetSuccess(t *testing.T) {
	mock := &mockCacheService{
		OperateCacheFunc: func(ctx context.Context, params monitorServices.CacheOperateParams) (interface{}, error) {
			return "cached-value", nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/op", map[string]interface{}{
		"key":       "k1",
		"operation": "get",
	})
	h.OperateCache(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_OperateCache_Error(t *testing.T) {
	mock := &mockCacheService{
		OperateCacheFunc: func(ctx context.Context, params monitorServices.CacheOperateParams) (interface{}, error) {
			return nil, errors.New("op fail")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/op", map[string]interface{}{
		"key":       "k1",
		"operation": "set",
		"value":     "v1",
	})
	h.OperateCache(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_BatchOperateCache_InvalidRequest(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/batch", map[string]interface{}{})
	h.BatchOperateCache(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCacheHandler_BatchOperateCache_EmptyKeys(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/batch", map[string]interface{}{
		"keys":      []string{},
		"operation": "get",
	})
	h.BatchOperateCache(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_BatchOperateCache_Success(t *testing.T) {
	mock := &mockCacheService{
		BatchOperateCacheFunc: func(ctx context.Context, params monitorServices.CacheBatchOperateParams) (map[string]interface{}, error) {
			return map[string]interface{}{"k1": "v1"}, nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/batch", map[string]interface{}{
		"keys":      []string{"k1", "k2"},
		"operation": "get",
	})
	h.BatchOperateCache(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_BatchOperateCache_Error(t *testing.T) {
	mock := &mockCacheService{
		BatchOperateCacheFunc: func(ctx context.Context, params monitorServices.CacheBatchOperateParams) (map[string]interface{}, error) {
			return nil, errors.New("batch fail")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/batch", map[string]interface{}{
		"keys":      []string{"k1"},
		"operation": "del",
	})
	h.BatchOperateCache(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_ClearCache_Success(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/clear", nil)
	h.ClearCache(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_ClearCache_Error(t *testing.T) {
	mock := &mockCacheService{
		ClearCacheFunc: func(ctx context.Context) error { return errors.New("clear fail") },
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/clear", nil)
	h.ClearCache(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheStats_Success(t *testing.T) {
	mock := &mockCacheService{
		GetCacheStatsFunc: func(ctx context.Context, params monitorServices.CacheStatsParams) ([]models.CacheStats, int64, error) {
			return []models.CacheStats{{CacheType: "l1", HitCount: 100}}, 1, nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/stats/list", map[string]interface{}{})
	h.GetCacheStats(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheStats_Error(t *testing.T) {
	mock := &mockCacheService{
		GetCacheStatsFunc: func(ctx context.Context, params monitorServices.CacheStatsParams) ([]models.CacheStats, int64, error) {
			return nil, 0, errors.New("stats fail")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/stats/list", map[string]interface{}{})
	h.GetCacheStats(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheMonitor_Success(t *testing.T) {
	mock := &mockCacheService{
		GetCacheMonitorFunc: func(ctx context.Context) (map[string]interface{}, error) {
			return map[string]interface{}{"l1_size": 100}, nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/monitor", nil)
	h.GetCacheMonitor(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheMonitor_Error(t *testing.T) {
	mock := &mockCacheService{
		GetCacheMonitorFunc: func(ctx context.Context) (map[string]interface{}, error) {
			return nil, errors.New("monitor fail")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/monitor", nil)
	h.GetCacheMonitor(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_ExportCache_Success(t *testing.T) {
	mock := &mockCacheService{
		ExportCacheFunc: func(ctx context.Context, params monitorServices.CacheExportParams) ([]models.CacheInfo, error) {
			return []models.CacheInfo{{Key: "k1"}}, nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/export", map[string]interface{}{})
	h.ExportCache(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_ExportCache_Error(t *testing.T) {
	mock := &mockCacheService{
		ExportCacheFunc: func(ctx context.Context, params monitorServices.CacheExportParams) ([]models.CacheInfo, error) {
			return nil, errors.New("export fail")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/export", map[string]interface{}{})
	h.ExportCache(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheConfigs_Success(t *testing.T) {
	mock := &mockCacheService{
		GetCacheConfigsFunc: func(ctx context.Context) (map[string]monitorServices.CacheConfigInfo, map[string]int, error) {
			return map[string]monitorServices.CacheConfigInfo{
				"cache.user.ttl": {Key: "cache.user.ttl", Name: "User TTL", Category: "user", Min: 1, Max: 60, Default: 30},
			}, map[string]int{"cache.user.ttl": 30}, nil
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("GET", "/config", nil)
	h.GetCacheConfigs(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_GetCacheConfigs_Error(t *testing.T) {
	mock := &mockCacheService{
		GetCacheConfigsFunc: func(ctx context.Context) (map[string]monitorServices.CacheConfigInfo, map[string]int, error) {
			return nil, nil, errors.New("config fail")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("GET", "/config", nil)
	h.GetCacheConfigs(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_UpdateCacheConfig_InvalidRequest(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("PUT", "/config", map[string]interface{}{})
	h.UpdateCacheConfig(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCacheHandler_UpdateCacheConfig_Success(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("PUT", "/config", map[string]interface{}{
		"key":   "cache.user.ttl",
		"value": 30,
	})
	h.UpdateCacheConfig(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_UpdateCacheConfig_InvalidKey(t *testing.T) {
	mock := &mockCacheService{
		UpdateCacheConfigFunc: func(ctx context.Context, key string, value int) error {
			return monitorServices.ErrInvalidConfigKey
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("PUT", "/config", map[string]interface{}{
		"key":   "bad.key",
		"value": 30,
	})
	h.UpdateCacheConfig(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_UpdateCacheConfig_Error(t *testing.T) {
	mock := &mockCacheService{
		UpdateCacheConfigFunc: func(ctx context.Context, key string, value int) error {
			return errors.New("update fail")
		},
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("PUT", "/config", map[string]interface{}{
		"key":   "k",
		"value": 30,
	})
	h.UpdateCacheConfig(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_ReloadCacheConfigs_Success(t *testing.T) {
	mock := &mockCacheService{}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/reload", nil)
	h.ReloadCacheConfigs(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCacheHandler_ReloadCacheConfigs_Error(t *testing.T) {
	mock := &mockCacheService{
		ReloadCacheConfigsFunc: func(ctx context.Context) error { return errors.New("reload fail") },
	}
	h := setupCacheHandler(mock)
	c, w := newTestCtx("POST", "/reload", nil)
	h.ReloadCacheConfigs(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_TestCacheEndpoint_CoreNil(t *testing.T) {
	mock := &mockCacheService{}
	h := NewCacheHandler(mock, &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtx("GET", "/test", nil)
	h.TestCacheEndpoint(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_DebugRawKeys_CoreNil(t *testing.T) {
	mock := &mockCacheService{}
	h := NewCacheHandler(mock, &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtx("POST", "/debug/raw", nil)
	h.DebugRawKeys(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_DebugL1Cache_CoreNil(t *testing.T) {
	mock := &mockCacheService{}
	h := NewCacheHandler(mock, &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtx("POST", "/debug/l1", nil)
	h.DebugL1Cache(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheHandler_SetPaginationDefaults(t *testing.T) {
	h := &CacheHandler{}
	c, p := h.setPaginationDefaults(0, 0)
	assert.Equal(t, 1, c)
	assert.Equal(t, 10, p)
	c2, p2 := h.setPaginationDefaults(5, 20)
	assert.Equal(t, 5, c2)
	assert.Equal(t, 20, p2)
}

func TestCacheHandler_WithCore(t *testing.T) {
	h := &CacheHandler{}
	result := h.WithCore(&core.Core{CoreInfra: &core.CoreInfra{}})
	assert.Same(t, h, result)
	assert.NotNil(t, result.core)
}

func TestCacheHandler_OperateCache_LogSavePath(t *testing.T) {
	_ = gorm.DB{}
	mock := &mockCacheService{
		OperateCacheFunc: func(ctx context.Context, params monitorServices.CacheOperateParams) (interface{}, error) {
			return "result-data", nil
		},
	}
	// core.DB = nil 让 saveCacheOperationLog 走 early return
	h := NewCacheHandler(mock, &core.Core{
		CoreInfra:    &core.CoreInfra{DB: nil},
		CoreServices: &core.CoreServices{},
	})
	c, w := newTestCtx("POST", "/op", map[string]interface{}{
		"key":       "k1",
		"operation": "get",
		"ttl":       60,
	})
	h.OperateCache(c)
	assert.Equal(t, http.StatusOK, w.Code)
}
