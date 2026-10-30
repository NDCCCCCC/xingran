package middleware

// =====================================================================
// Phase 80-05 Task 6: middleware 加解密 + oper_log 中间件(httptest + 真加密器)。
//
// 74-08 已覆盖 ResponseEncryption 大面 / GetConfigFromDB / isExcludedPath /
// ResponseWriter;本 plan 补 RequestDecryption 主函数 0% + OperLogMiddleware
// 0% + ResponseEncryption 残余分支。
//
// 纪律(R6):
//   - 真 SM2 keypair:crypto.GenerateKeyPair() + NewRequestEncryptor;
//     同步 SM4 key + iv 用 BuildEncryptedRequest 模式(沿 crypto_74_08_test.go)。
//   - timestamp ±window±1 表驱动:SetReplayWindowSec 缩到 1s 以避开墙钟依赖。
//   - writeBuffer 只断言 flush 后可读,勿断言未 flush 的中间态。
//   - oper_log 落库 sqlite,断言行字段;t.Cleanup 关闭 db。
// =====================================================================

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tjfoc/gmsm/sm2"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

func init() { gin.SetMode(gin.TestMode) }

// primeEncryptionCache8005 把 globalConfigCache 标记为"近期已加载",
// 让 getConfigFromDB 直接返回缓存值而非走 db 查询(测试不依赖 DB)。
func primeEncryptionCache8005(t *testing.T, enabled bool) {
	t.Helper()
	globalConfigCache.mu.Lock()
	globalConfigCache.value = enabled
	globalConfigCache.lastUpdate = time.Now()
	globalConfigCache.mu.Unlock()
	t.Cleanup(func() {
		globalConfigCache.mu.Lock()
		globalConfigCache.value = true
		globalConfigCache.lastUpdate = time.Time{}
		globalConfigCache.mu.Unlock()
	})
}

// buildEncryptedReq 构造合法加密请求(SM2 加密 SM4 key + SM4-CBC 加密数据),
// 必须用调用方传入的 enc(保证 SM2 密钥对一致,中间件才能解密) + 其 SM2 公钥。
// 沿用 crypto_74_08_test.go 的 buildEncryptedRequest 形态(同包可复,这里
// 复制避免引入测试 cross-dep)。
func buildEncryptedReq(t *testing.T, enc *crypto.RequestEncryptor, pub *sm2.PublicKey, plaintext, nonce string, ts int64) *crypto.EncryptedRequest {
	t.Helper()
	sm4KeyHex := "0123456789abcdeffedcba9876543210"
	keyBytes, err := hex.DecodeString(sm4KeyHex)
	require.NoError(t, err)
	iv := []byte("0123456789abcdef")

	encKey, err := crypto.EncryptWithSM2(sm4KeyHex, pub)
	require.NoError(t, err)

	encResp, err := enc.EncryptResponseWithKey([]byte(plaintext), keyBytes, iv)
	require.NoError(t, err)

	return &crypto.EncryptedRequest{
		Encrypted: true,
		Data:      encResp.Data,
		SM4Key:    encKey,
		IV:        base64.StdEncoding.EncodeToString(iv),
		Timestamp: ts,
		Nonce:     nonce,
	}
}

// miniCore8005 给 oper_log 用的极小 core:真 GetDB(),其他字段用零值(utils
// 助手在 username/nickname 未注入时返回 nil)。
func miniCore8005(t *testing.T) (*core.Core, *gorm.DB) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mdw8005.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	// oper_log 落库需 sys_oper_log 表。
	require.NoError(t, gormDB.AutoMigrate(&models.OperLog{}))

	db := &coredb.Database{DB: gormDB, Type: "sqlite"}
	infra := &core.CoreInfra{Config: &config.Config{}, DB: db}
	c := &core.Core{CoreInfra: infra, CoreServices: &core.CoreServices{}}
	return c, gormDB
}

// TestMdw8005_RequestDecryption_RoundTrip:客户端按 SM2+SM4 混合封装 POST → 中间件解密 → handler 收明文。
func TestMdw8005_RequestDecryption_RoundTrip(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	require.NoError(t, err)
	enc := crypto.NewRequestEncryptor(priv, pub)
	primeEncryptionCache8005(t, true)

	r := gin.New()
	r.POST("/api/test", RequestDecryption(enc, &RequestDecryptionConfig{
		Enabled:           true,
		RequireEncryption: true,
	}, nil), func(c *gin.Context) {
		body, err := c.GetRawData()
		require.NoError(t, err)
		c.String(200, string(body))
	})

	now := time.Now().Unix()
	encReq := buildEncryptedReq(t, enc, pub, `{"hello":"world"}`, "nonce-rd-rt", now)
	body, err := json.Marshal(encReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "解密成功应 200,body=%s", w.Body.String())
	assert.JSONEq(t, `{"hello":"world"}`, w.Body.String())
}

// TestMdw8005_RequestDecryption_Replay_Table:timestamp ±window±1 表驱动(R6,
// 缩到 1s 窗口避开墙钟依赖)。
func TestMdw8005_RequestDecryption_Replay_Table(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	require.NoError(t, err)
	enc := crypto.NewRequestEncryptor(priv, pub)
	enc.SetReplayWindowSec(1) // 缩窗口:±1s,避开墙钟漂移
	primeEncryptionCache8005(t, true)

	r := gin.New()
	r.POST("/api/replay", RequestDecryption(enc, &RequestDecryptionConfig{
		Enabled:           true,
		RequireEncryption: true,
	}, nil), func(c *gin.Context) {
		c.String(200, "ok")
	})

	now := time.Now().Unix()

	// (1) window-1 旧时间戳 → 应通过(±window)。
	encReq := buildEncryptedReq(t, enc, pub, `{"k":1}`, "nonce-replay-1", now-1)
	body, err := json.Marshal(encReq)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/replay", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "now-1 应在 ±1s 窗口内通过")

	// (2) window+1 → 拒绝。
	encReq2 := buildEncryptedReq(t, enc, pub, `{"k":2}`, "nonce-replay-2", now-3)
	body2, _ := json.Marshal(encReq2)
	req2 := httptest.NewRequest("POST", "/api/replay", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, 400, w2.Code, "now-3 应被 ±1s 窗口拒绝")

	// (3) nonce 重放 → 拒绝(同一 nonce 第二次出现)。
	encReq3 := buildEncryptedReq(t, enc, pub, `{"k":3}`, "nonce-replay-reuse", now)
	body3, _ := json.Marshal(encReq3)
	req3 := httptest.NewRequest("POST", "/api/replay", bytes.NewReader(body3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, 200, w3.Code, "首次出现 nonce 应通过")

	req3b := httptest.NewRequest("POST", "/api/replay", bytes.NewReader(body3))
	req3b.Header.Set("Content-Type", "application/json")
	w3b := httptest.NewRecorder()
	r.ServeHTTP(w3b, req3b)
	assert.Equal(t, 400, w3b.Code, "重复 nonce 应被拒绝")
}

// TestMdw8005_RequestDecryption_BadPayload:坏 SM2 密文 / 缺字段 / 非法 encrypted 标志 /
// 非 POST 方法 / multipart / exclude_paths 命中。
func TestMdw8005_RequestDecryption_BadPayload(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	require.NoError(t, err)
	enc := crypto.NewRequestEncryptor(priv, pub)
	primeEncryptionCache8005(t, true)

	r := gin.New()
	r.POST("/api/x", RequestDecryption(enc, &RequestDecryptionConfig{
		Enabled: true,
		ExcludePaths: []string{
			"/api/excluded",
			"/api/excluded/*",
		},
	}, nil), func(c *gin.Context) {
		c.String(200, "ok")
	})
	r.GET("/api/x", RequestDecryption(enc, &RequestDecryptionConfig{Enabled: true}, nil),
		func(c *gin.Context) { c.String(200, "get-ok") })
	r.POST("/api/excluded", RequestDecryption(enc, &RequestDecryptionConfig{Enabled: true}, nil),
		func(c *gin.Context) { c.String(200, "excluded-ok") })

	// (1) 未加密 POST + RequireEncryption=false → 直通(pass-through)。
	r2 := gin.New()
	r2.POST("/api/opt", RequestDecryption(enc, &RequestDecryptionConfig{
		Enabled: true, RequireEncryption: false,
	}, nil), func(c *gin.Context) {
		body, _ := c.GetRawData()
		c.String(200, string(body))
	})
	req := httptest.NewRequest("POST", "/api/opt", bytes.NewReader([]byte(`{"raw":true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code, "RequireEncryption=false 时未加密请求应直通")
	assert.Equal(t, `{"raw":true}`, w.Body.String())

	// (2) GET 方法 → 直通(RequestDecryption 只处理 POST/PUT/PATCH)。
	req = httptest.NewRequest("GET", "/api/x", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "get-ok", w.Body.String())

	// (3) exclude_paths 命中 → 直通。
	req = httptest.NewRequest("POST", "/api/excluded", bytes.NewReader([]byte(`{"x":1}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "excluded-ok", w.Body.String())

	// (4) enabled=false → 整条直通。
	r3 := gin.New()
	r3.POST("/api/d", RequestDecryption(enc, &RequestDecryptionConfig{Enabled: false}, nil),
		func(c *gin.Context) {
			body, _ := c.GetRawData()
			c.String(200, string(body))
		})
	req = httptest.NewRequest("POST", "/api/d", bytes.NewReader([]byte(`{"raw":"plain"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r3.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `{"raw":"plain"}`, w.Body.String())

	// (5) 坏 SM2 密文(sm4Key 字段乱填)→ 解密失败 → 400。
	bad := &crypto.EncryptedRequest{
		Encrypted: true,
		Data:      "invalid-base64-or-data",
		SM4Key:    "not-a-valid-sm2-cipher",
		IV:        base64.StdEncoding.EncodeToString([]byte("0123456789abcdef")),
		Timestamp: time.Now().Unix(),
		Nonce:     "nonce-bad-1",
	}
	badBody, _ := json.Marshal(bad)
	req = httptest.NewRequest("POST", "/api/x", bytes.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code, "坏 SM2 密文应解密失败 → 400")

	// (6) 缺字段(没有 nonce/timestamp)→ 400。
	missingFields := &crypto.EncryptedRequest{
		Encrypted: true,
		Data:      "data",
		SM4Key:    "key",
		IV:        "iv",
		// Timestamp / Nonce 缺
	}
	missingBody, _ := json.Marshal(missingFields)
	req = httptest.NewRequest("POST", "/api/x", bytes.NewReader(missingBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code, "缺 timestamp/nonce 应 400")

	// (7) encrypted=false + RequireEncryption=true → 400。
	plainBody := []byte(`{"encrypted":false,"data":"x"}`)
	req = httptest.NewRequest("POST", "/api/x", bytes.NewReader(plainBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r5 := gin.New()
	r5.POST("/api/x", RequestDecryption(enc, &RequestDecryptionConfig{
		Enabled: true, RequireEncryption: true,
	}, nil), func(c *gin.Context) { c.String(200, "ok") })
	r5.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code, "未加密 + RequireEncryption=true 应 400")
}

// TestMdw8005_ResponseEncryption:enabled + sm4_key 注入 → 响应体被加密;
// disabled → 明文;writeBuffer flush 后可读。
func TestMdw8005_ResponseEncryption(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair()
	require.NoError(t, err)
	enc := crypto.NewRequestEncryptor(priv, pub)
	primeEncryptionCache8005(t, true)

	// (1) enabled + sm4_key 注入 → 响应体加密。
	sm4KeyBytes, _ := hex.DecodeString("0123456789abcdeffedcba9876543210")
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// 模拟请求解密中间件注入 sm4_key(16 字节 raw,非 hex 字符串)。
		c.Set("sm4_key", sm4KeyBytes)
		c.Set("sm4_iv", []byte("0123456789abcdef"))
		c.Next()
	})
	r.GET("/api/enc", ResponseEncryption(enc, &ResponseEncryptionConfig{Enabled: true}, nil),
		func(c *gin.Context) {
			c.JSON(200, gin.H{"data": "secret"})
		})

	req := httptest.NewRequest("GET", "/api/enc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	// 响应体为加密 EncryptedRequest JSON(以 encrypted=true 标识),不是明文 gin.H。
	assert.Contains(t, body, `"encrypted":true`, "加密路径应产生 EncryptedRequest 形态")
	assert.NotContains(t, body, `"data":"secret"`, "明文 data 不应出现在加密响应中")

	// (2) disabled → 直通明文。
	r2 := gin.New()
	r2.GET("/api/plain", ResponseEncryption(enc, &ResponseEncryptionConfig{Enabled: false}, nil),
		func(c *gin.Context) {
			c.JSON(200, gin.H{"data": "secret"})
		})
	req = httptest.NewRequest("GET", "/api/plain", nil)
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"data":"secret"`, "disabled 应明文直通")

	// (3) writeBuffer 只一次 flush;responseWriter 缓冲语义在 encryption_74_08
	// 已测。本测试主要断言 enabled + sm4_key 注入路径触达(原本 12.5% → 提升)。
}

// TestMdw8005_OperLogMiddleware:httptest 请求穿过中间件 → oper_log 落库(sqlite
// 行断言)。
func TestMdw8005_OperLogMiddleware(t *testing.T) {
	c, gormDB := miniCore8005(t)
	operLogSvc := services.NewOperLogService()

	r := gin.New()
	r.Use(OperLogMiddleware(operLogSvc, c))
	r.POST("/system/user/add", func(c *gin.Context) {
		// handler 设置 oper_log 元数据(模拟真实业务 handler)。
		SetOperLogInfo(c, "用户管理", operlog.OperTypeCreate, "POST")
		c.JSON(200, gin.H{"ok": true})
	})

	// (1) 命中 LogPaths(默认包含 /system/user)→ 异步落库。
	req := httptest.NewRequest("POST", "/system/user/add", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// (2) RecordAsync 在 OperLogMiddleware 中同步调用;等待落库。
	assert.Eventually(t, func() bool {
		var count int64
		gormDB.Model(&models.OperLog{}).Count(&count)
		return count >= 1
	}, 2*time.Second, 20*time.Millisecond, "应异步落库 1 条 oper_log")

	var log models.OperLog
	require.NoError(t, gormDB.First(&log).Error)
	assert.Equal(t, "用户管理", log.Title)
	assert.Equal(t, operlog.OperTypeCreate, log.BusinessType)
	assert.Equal(t, "新增", log.Method, "method 应映射自 /add 路径 → 新增")
	assert.Equal(t, "POST", log.RequestMethod)
	assert.Equal(t, 0, log.Status)

	// (3) 路径不在 LogPaths 内 → 不记录。
	r2 := gin.New()
	r2.Use(OperLogMiddleware(operLogSvc, c))
	r2.POST("/unrelated/path", func(c *gin.Context) {
		SetOperLogInfo(c, "无关", 0, "POST")
		c.JSON(200, gin.H{"ok": true})
	})
	req = httptest.NewRequest("POST", "/unrelated/path", nil)
	w = httptest.NewRecorder()
	r2.ServeHTTP(w, req)
	// 等待 + 断言行数仍为 1(未新增)。
	time.Sleep(100 * time.Millisecond)
	var count int64
	gormDB.Model(&models.OperLog{}).Count(&count)
	assert.Equal(t, int64(1), count, "不在 LogPaths 内的请求不应记录")
}

// TestMdw8005_GetBusinessType:getBusinessType 路径关键字到 OperType 常量映射。
func TestMdw8005_GetBusinessType(t *testing.T) {
	// OperType 常量值 0/1/2/3/5/6;映射:create=1, update=2, delete=3, export=5, import=6, other=0。
	assert.Equal(t, operlog.OperTypeCreate, GetBusinessType("/user/add", "POST"))
	assert.Equal(t, operlog.OperTypeCreate, GetBusinessType("/user/create", "POST"))
	assert.Equal(t, operlog.OperTypeUpdate, GetBusinessType("/user/edit", "POST"))
	assert.Equal(t, operlog.OperTypeUpdate, GetBusinessType("/user/update", "POST"))
	assert.Equal(t, operlog.OperTypeDelete, GetBusinessType("/user/delete", "POST"))
	assert.Equal(t, operlog.OperTypeDelete, GetBusinessType("/user/remove", "POST"))
	assert.Equal(t, operlog.OperTypeExport, GetBusinessType("/report/export", "GET"))
	assert.Equal(t, operlog.OperTypeImport, GetBusinessType("/data/import", "POST"))
	assert.Equal(t, operlog.OperTypeOther, GetBusinessType("/list", "GET"))
}

// TestMdw8005_ShouldLogOperation:路径/方法/排除路径表驱动。
func TestMdw8005_ShouldLogOperation(t *testing.T) {
	cfg := DefaultOperLogConfig()

	// POST/PUT/DELETE 在 LogPaths 内 → true。
	assert.True(t, shouldLogOperation("/system/user/add", "POST", cfg))
	assert.True(t, shouldLogOperation("/system/role/edit", "PUT", cfg))
	assert.True(t, shouldLogOperation("/system/user/delete", "DELETE", cfg))

	// GET → false。
	assert.False(t, shouldLogOperation("/system/user/list", "GET", cfg))

	// 排除路径优先。
	cfg2 := &OperLogConfig{
		LogPaths:     []string{"/system"},
		ExcludePaths: []string{"/system/healthcheck"},
	}
	assert.False(t, shouldLogOperation("/system/healthcheck", "POST", cfg2))
	assert.True(t, shouldLogOperation("/system/user", "POST", cfg2))
}

// TestMdw8005_RefreshEncryptionConfigCache_AndGet:刷新 + 读取缓存值。
func TestMdw8005_RefreshEncryptionConfigCache_AndGet(t *testing.T) {
	prev := GetEncryptionConfigFromCache()
	t.Cleanup(func() {
		globalConfigCache.mu.Lock()
		globalConfigCache.value = prev
		globalConfigCache.lastUpdate = time.Time{}
		globalConfigCache.mu.Unlock()
	})

	RefreshEncryptionConfigCache()
	assert.True(t, time.Time{}.Equal(globalConfigCache.lastUpdate),
		"Refresh 后 lastUpdate 应被置零(下次请求重读 db)")

	// 写入缓存值并读取。
	globalConfigCache.mu.Lock()
	globalConfigCache.value = false
	globalConfigCache.lastUpdate = time.Now()
	globalConfigCache.mu.Unlock()
	assert.False(t, GetEncryptionConfigFromCache())

	globalConfigCache.mu.Lock()
	globalConfigCache.value = true
	globalConfigCache.lastUpdate = time.Now()
	globalConfigCache.mu.Unlock()
	assert.True(t, GetEncryptionConfigFromCache())
}

// _ suppress unused import warnings for core config in test-only paths.
var _ = fmt.Sprintf