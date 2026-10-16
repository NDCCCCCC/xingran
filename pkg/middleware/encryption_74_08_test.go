package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =====================================================================
// 74-08 Batch C 续: request_decryption.go 配置缓存 + isExcludedPath +
// response_encryption.go responseWriter/中间件各分支。
// 注意: globalConfigCache 是进程级 30s TTL 缓存,用例间必须显式重置。
// =====================================================================

func resetGlobalConfigCache() {
	globalConfigCache.mu.Lock()
	defer globalConfigCache.mu.Unlock()
	globalConfigCache.lastUpdate = time.Time{}
	globalConfigCache.value = false
}

func setGlobalConfigCache(v bool) {
	globalConfigCache.mu.Lock()
	defer globalConfigCache.mu.Unlock()
	globalConfigCache.value = v
	globalConfigCache.lastUpdate = time.Now()
}

func newConfigDB(t *testing.T, value *string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:cfg_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_config (id INTEGER PRIMARY KEY AUTOINCREMENT, config_key TEXT, config_value TEXT)`).Error)
	if value != nil {
		require.NoError(t, db.Exec(`INSERT INTO sys_config (config_key, config_value) VALUES (?, ?)`, encryptionConfigKey, *value).Error)
	}
	return db
}

// ---------------- getConfigFromDB / 缓存 ----------------

func TestGetConfigFromDB(t *testing.T) {
	ctx := context.Background()

	// DB true → true 并写缓存
	resetGlobalConfigCache()
	db := newConfigDB(t, strPtr("true"))
	assert.True(t, getConfigFromDB(ctx, db, false))
	// 缓存命中(30s 内): 即便 DB 改为 false 仍返回缓存 true
	assert.True(t, getConfigFromDB(ctx, db, false), "TTL 内命中缓存")

	// DB false → false
	resetGlobalConfigCache()
	db2 := newConfigDB(t, strPtr("0"))
	assert.False(t, getConfigFromDB(ctx, db2, true))

	// 非法值 → fallback
	resetGlobalConfigCache()
	db3 := newConfigDB(t, strPtr("garbage"))
	assert.True(t, getConfigFromDB(ctx, db3, true), "非法值回退 fallback=true")
	assert.False(t, getConfigFromDB(ctx, db3, false), "非法值回退 fallback=false")

	// 无记录 → Pluck 不报错但 configValue 空 → 非法值 → fallback
	resetGlobalConfigCache()
	db4 := newConfigDB(t, nil)
	assert.True(t, getConfigFromDB(ctx, db4, true))

	// DB 查询失败(无表) + 无缓存 → fallback
	resetGlobalConfigCache()
	dbNoTable, err := gorm.Open(sqlite.Open("file:cfg_notable?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	assert.False(t, getConfigFromDB(ctx, dbNoTable, false))
	assert.True(t, getConfigFromDB(ctx, dbNoTable, true))

	// DB 查询失败 + 有缓存 → 用缓存值
	resetGlobalConfigCache()
	setGlobalConfigCache(true)
	assert.True(t, getConfigFromDB(ctx, dbNoTable, false), "DB 故障用缓存值")

	resetGlobalConfigCache()
}

func TestRefreshAndGetEncryptionConfigCache(t *testing.T) {
	// 未初始化 → 默认 true
	resetGlobalConfigCache()
	assert.True(t, GetEncryptionConfigFromCache(), "未初始化默认 true")

	setGlobalConfigCache(false)
	assert.False(t, GetEncryptionConfigFromCache())

	RefreshEncryptionConfigCache()
	assert.True(t, GetEncryptionConfigFromCache(), "刷新后回到未初始化默认")

	resetGlobalConfigCache()
}

// ---------------- isExcludedPath ----------------

func TestIsExcludedPath(t *testing.T) {
	patterns := []string{"/api/v1/upload/*", "/exact/path", "/api/v1/system/auth/*"}

	assert.True(t, isExcludedPath("/exact/path", patterns))
	assert.False(t, isExcludedPath("/exact/path2", patterns))

	assert.True(t, isExcludedPath("/api/v1/upload/avatar", patterns))
	assert.True(t, isExcludedPath("/api/v1/upload", patterns), "前缀完全匹配")
	assert.False(t, isExcludedPath("/api/v1/uploadx", patterns), "非 / 边界不匹配")

	assert.True(t, isExcludedPath("/api/v1/system/auth/login", patterns))
	assert.False(t, isExcludedPath("/api/v1/system/user", patterns))

	assert.False(t, isExcludedPath("/anything", nil))
}

// ---------------- responseWriter ----------------

func TestResponseWriterBuffering(t *testing.T) {
	w := httptest.NewRecorder()
	bw := &responseWriter{ResponseWriter: gin.ResponseWriter(nil), buffer: bytes.NewBufferString("")}
	_ = w

	// Write/WriteString 只进 buffer
	n, err := bw.Write([]byte("abc"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	n, err = bw.WriteString("def")
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "abcdef", bw.buffer.String())
}

func TestResponseWriter_WriteBufferOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	bw := &responseWriter{ResponseWriter: c.Writer, buffer: bytes.NewBufferString("payload")}

	bw.writeBuffer()
	assert.Equal(t, "payload", rec.Body.String())

	// 第二次调用原子性跳过
	bw.buffer.Reset()
	bw.buffer.WriteString("should-not-appear")
	bw.writeBuffer()
	assert.Equal(t, "payload", rec.Body.String(), "writeBuffer 只执行一次")
}

// ---------------- ResponseEncryption 中间件 ----------------

func TestResponseEncryption_DisabledPassthrough(t *testing.T) {
	resetGlobalConfigCache()
	defer resetGlobalConfigCache()
	setGlobalConfigCache(false) // 禁用

	r := gin.New()
	r.Use(ResponseEncryption(nil, &ResponseEncryptionConfig{Enabled: true}, nil))
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{"a": 1}) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"a":1`)
	assert.Empty(t, w.Header().Get("X-Response-Encrypted"))
}

func TestResponseEncryption_NoSM4KeyPassthrough(t *testing.T) {
	resetGlobalConfigCache()
	defer resetGlobalConfigCache()
	setGlobalConfigCache(true) // 启用但 ctx 无 sm4_key(Phase 40 修复路径)

	r := gin.New()
	r.Use(ResponseEncryption(nil, &ResponseEncryptionConfig{Enabled: true}, nil))
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{"a": 1}) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Contains(t, w.Body.String(), `"a":1`, "无 sm4_key → 原样返回")
}

func TestResponseEncryption_ExcludedPath(t *testing.T) {
	resetGlobalConfigCache()
	defer resetGlobalConfigCache()
	setGlobalConfigCache(true)

	r := gin.New()
	r.Use(ResponseEncryption(nil, &ResponseEncryptionConfig{Enabled: true, ExcludePaths: []string{"/x/*"}}, nil))
	r.GET("/x/y", func(c *gin.Context) {
		c.Set("sm4_key", []byte("0123456789abcdef"))
		c.JSON(200, gin.H{"a": 1})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x/y", nil))
	assert.Contains(t, w.Body.String(), `"a":1`, "排除路径原样返回")
}

func TestResponseEncryption_ErrorStatusNotEncrypted(t *testing.T) {
	resetGlobalConfigCache()
	defer resetGlobalConfigCache()
	setGlobalConfigCache(true)

	r := gin.New()
	r.Use(ResponseEncryption(nil, &ResponseEncryptionConfig{Enabled: true}, nil))
	r.GET("/x", func(c *gin.Context) {
		c.Set("sm4_key", []byte("0123456789abcdef"))
		c.Set("sm4_iv", []byte("0123456789abcdef"))
		c.JSON(500, gin.H{"error": "boom"})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "boom", "错误状态不加密")
}

func TestResponseEncryption_InvalidJSONPassthrough(t *testing.T) {
	resetGlobalConfigCache()
	defer resetGlobalConfigCache()
	setGlobalConfigCache(true)

	r := gin.New()
	r.Use(ResponseEncryption(nil, &ResponseEncryptionConfig{Enabled: true}, nil))
	r.GET("/x", func(c *gin.Context) {
		c.Set("sm4_key", []byte("0123456789abcdef"))
		c.Set("sm4_iv", []byte("0123456789abcdef"))
		c.Data(200, "application/json", []byte("not-json{"))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, "not-json{", w.Body.String(), "非法 JSON 不加密")
}

func TestResponseEncryption_AlreadyEncryptedPassthrough(t *testing.T) {
	resetGlobalConfigCache()
	defer resetGlobalConfigCache()
	setGlobalConfigCache(true)

	r := gin.New()
	r.Use(ResponseEncryption(nil, &ResponseEncryptionConfig{Enabled: true}, nil))
	r.GET("/x", func(c *gin.Context) {
		c.Set("sm4_key", []byte("0123456789abcdef"))
		c.Set("sm4_iv", []byte("0123456789abcdef"))
		c.JSON(200, gin.H{"encrypted": true, "data": "xxx"})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "xxx", body["data"], "已加密格式不重复加密")
}

// strPtr 辅助。
func strPtr(s string) *string { return &s }
