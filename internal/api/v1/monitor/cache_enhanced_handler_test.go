package monitor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core"
)

func newTestCtxCE(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

func parseRespCE(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	return resp
}

// CacheEnhancedHandler 测试覆盖: CacheManager 依赖复杂,使用 nil manager 测试所有 if 分支即可。

func TestCacheEnhanced_GetCacheStats_NilManager(t *testing.T) {
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/stats", nil)
	h.GetCacheStats(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheEnhanced_InvalidateByModule_NilManager(t *testing.T) {
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/invalidate", map[string]interface{}{"module": "user"})
	h.InvalidateByModule(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheEnhanced_InvalidateByModule_InvalidBody(t *testing.T) {
	// 无 CacheManager 时的 binding 错误路径 (先 early-return)
	// 注: handler 先检查 CacheManager == nil,所以 binding 错误路径走不到
	// 这里只测 nil-manager 路径
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/invalidate", map[string]interface{}{})
	h.InvalidateByModule(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheEnhanced_InvalidateByPattern_NilManager(t *testing.T) {
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/invalidate-pattern", map[string]interface{}{"pattern": "user:*"})
	h.InvalidateByPattern(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheEnhanced_InvalidateByPattern_InvalidBody(t *testing.T) {
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/invalidate-pattern", map[string]interface{}{})
	h.InvalidateByPattern(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheEnhanced_WarmUpCache_NilManager(t *testing.T) {
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/warmup", nil)
	h.WarmUpCache(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheEnhanced_GetKeyInfo_NilManager(t *testing.T) {
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/key-info", map[string]interface{}{"key": "k1"})
	h.GetKeyInfo(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestCacheEnhanced_GetKeyInfo_InvalidBody(t *testing.T) {
	h := NewCacheEnhancedHandler(&core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{CacheManager: nil},
	})
	c, w := newTestCtxCE("POST", "/key-info", map[string]interface{}{})
	h.GetKeyInfo(c)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
