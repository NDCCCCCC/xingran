package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// --- D-01 手写 fake (无 gomock 框架, 符合 TESTING.md 既有约定) ---

// fakeAPIKeyService 实现 system.APIKeyService 全 9 方法 (apikey_service.go:29-39)。
// 仅 ValidateAPIKey 给真逻辑 (MultiAuth 路径唯一被调方法), 其余 8 方法 panic。
//
// Pitfall 2 防御: 必须实现全部 9 方法, 否则 `MultiAuth(fakeSvc, ...)` 编译报
// `cannot use fakeSvc as type system.APIKeyService`。
type fakeAPIKeyService struct {
	validKey    *models.APIKey // 预置: 有效 key 时返回
	validateErr error          // 预置: 无效 key 时返回 (触发 401 路径)
}

func (f *fakeAPIKeyService) ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error) {
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	return f.validKey, nil
}

func (f *fakeAPIKeyService) CreateAPIKey(ctx context.Context, userID string, req *requests.CreateAPIKeyRequest) (*string, error) {
	panic("not used in MultiAuth path")
}
func (f *fakeAPIKeyService) ListAPIKeys(ctx context.Context, userID string, params requests.ListAPIKeysParams) (*system.PageResult, error) {
	panic("not used in MultiAuth path")
}
func (f *fakeAPIKeyService) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	panic("not used in MultiAuth path")
}
func (f *fakeAPIKeyService) UpdateAPIKey(ctx context.Context, id string, req *requests.UpdateAPIKeyRequest) error {
	panic("not used in MultiAuth path")
}
func (f *fakeAPIKeyService) DeleteAPIKey(ctx context.Context, id string) error {
	panic("not used in MultiAuth path")
}
func (f *fakeAPIKeyService) ToggleAPIKeyStatus(ctx context.Context, id string) error {
	panic("not used in MultiAuth path")
}
func (f *fakeAPIKeyService) ListUsageLogs(ctx context.Context, params system.ListUsageLogsParams) (*system.UsageLogsPageResult, error) {
	panic("not used in MultiAuth path")
}
func (f *fakeAPIKeyService) GetUsageLogSummary(ctx context.Context, apiKeyID string) (*system.UsageSummary, error) {
	panic("not used in MultiAuth path")
}

// fakeUsageLogger 实现 services.UsageLogger 单方法接口 (usage_logger.go:12-17)。
//
// done channel 用于同步: MultiAuth 的使用日志是 fire-and-forget 异步 goroutine
// (apikey.go:62-74), 测试断言 logged 前必须等待 goroutine 完成, 否则在
// `go test ./...` 高并行负载下偶发失败 (flaky)。生产代码 goroutine 仍在请求
// 结束后访问 gin.Context 属 P59 数据竞态范围 (P1-2/P2-b), 本测试不覆盖 -race 行为。
type fakeUsageLogger struct {
	logged bool
	done   chan struct{}
}

func newFakeUsageLogger() *fakeUsageLogger {
	return &fakeUsageLogger{done: make(chan struct{})}
}

func (f *fakeUsageLogger) LogUsage(ctx context.Context, req *services.LogUsageRequest) error {
	f.logged = true
	close(f.done)
	return nil
}

// waitForLog 等待异步 LogUsage 完成, 消除 fire-and-forget goroutine 带来的竞态/flaky。
// channel close 建立 happens-before: LogUsage 内 `logged=true` 对断言可见。
func (f *fakeUsageLogger) waitForLog(t *testing.T) {
	t.Helper()
	select {
	case <-f.done:
	case <-time.After(2 * time.Second):
		t.Fatal("MultiAuth 未在 2s 内调用 usage logger (异步日志 goroutine 未触发)")
	}
}

// --- helpers ---

// setupUsageLoggerTestDB 复制自 internal/services/usage_logger_test.go:30-57 (VERIFIED)。
//
// 关键设计点:
//   - 每测试独立文件 DB (os.TempDir + 唯一名), 替代共享内存 DB 防 SQLite 写锁
//   - busy_timeout=5000 让写锁排队而非立即报错
//   - 不用 t.TempDir: LogUsage 的 fire-and-forget goroutine 测试结束后仍写文件,
//     t.TempDir 自动 cleanup 会 mark test failed
func setupUsageLoggerTestDB(t *testing.T) *gorm.DB {
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_usage_%d_%d.db", time.Now().UnixNano(), os.Getpid()))
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_api_key_usage_logs (
			id TEXT PRIMARY KEY,
			api_key_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			method TEXT,
			path TEXT,
			status_code INTEGER,
			client_ip TEXT,
			user_agent TEXT,
			duration INTEGER,
			success BOOLEAN,
			created_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	return db
}

// hex64 返回 64 位十六进制字符串。
// 满足 isValidKeyFormat (apikey.go:86-106): rec_ + 64 hex = 68 字符。
// Pitfall 3 防御: 格式错会在 ValidateAPIKey 前被拦截走 401,
// 导致路径①/② 实际返回 401 而非 200/403。
func hex64() string {
	return strings.Repeat("0123456789abcdef", 4) // 64 chars
}

// --- 集成测试 (SC#1 + SC#3) ---

// TestMultiAuthIntegration 三路径集成测试: 200 / 403 / 401 全覆盖。
// 用手写 fake/stub + 真实 gin.Engine + httptest 驱动 MultiAuth 中间件。
func TestMultiAuthIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("有效key+正确scope_通过并写入context", func(t *testing.T) {
		uid := "11111111-2222-3333-4444-555555555555"
		fakeSvc := &fakeAPIKeyService{
			validKey: &models.APIKey{
				BaseModel: models.BaseModel{ID: "ak-test-id"},
				Name:      "test-key",
				UserID:    &uid,
				Scopes:    []string{"read"},
				IsActive:  true,
			},
		}
		fakeLogger := newFakeUsageLogger()

		router := gin.New()
		router.Use(MultiAuth(fakeSvc, fakeLogger, nil, nil))
		router.GET("/ping", RequireScope("read"), func(c *gin.Context) {
			// SC#1: 断言 setUserContextForAPIKey 写入的 4 个 context 键 (AUTH-01 修复证据)
			assert.NotEmpty(t, c.GetString("user_id"), "user_id 应非空")
			assert.Equal(t, "ak-test-id", c.GetString("api_key_id"), "api_key_id 应为预置 ID")
			assert.Equal(t, []string{"read"}, c.MustGet("scopes"), "scopes 应为预置 []string")
			assert.Equal(t, "api_key", c.GetString("auth_type"), "auth_type 应为字面量 api_key")
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/ping", nil)
		req.Header.Set("X-API-Key", "rec_"+hex64())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		fakeLogger.waitForLog(t) // 等待异步日志 goroutine 完成, 消除 flaky
		assert.Equal(t, 200, w.Code, "有效 key + 正确 scope 应 200")
		assert.True(t, fakeLogger.logged, "MultiAuth 应异步记录使用日志")
	})

	t.Run("有效key+缺失scope_403", func(t *testing.T) {
		fakeSvc := &fakeAPIKeyService{
			validKey: &models.APIKey{
				BaseModel: models.BaseModel{ID: "ak-test-id"},
				Name:      "test-key",
				Scopes:    []string{"read"},
				IsActive:  true,
			},
		}
		fakeLogger := newFakeUsageLogger()

		router := gin.New()
		router.Use(MultiAuth(fakeSvc, fakeLogger, nil, nil))
		router.GET("/write-only", RequireScope("write"), func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/write-only", nil)
		req.Header.Set("X-API-Key", "rec_"+hex64())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		fakeLogger.waitForLog(t) // 有效 key 路径仍触发异步日志 goroutine, 等待其完成避免跨子测试竞态
		assert.Equal(t, 403, w.Code, "有效 key + 缺失 scope 应 403 (response.ErrForbidden.HTTPStatus)")
	})

	t.Run("无效key_401", func(t *testing.T) {
		fakeSvc := &fakeAPIKeyService{
			validateErr: errors.New("密钥不存在或已禁用"),
		}
		fakeLogger := newFakeUsageLogger()

		router := gin.New()
		router.Use(MultiAuth(fakeSvc, fakeLogger, nil, nil))
		router.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		req := httptest.NewRequest("GET", "/ping", nil)
		req.Header.Set("X-API-Key", "rec_"+hex64())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code, "无效 key 应 401 (response.ErrUnauthorized.HTTPStatus)")
	})
}

// --- WR-03 回归锚: InheritPerms=true + permSvc/db 未注入 → 401 fail-closed(非 panic) ---

// TestMultiAuth_InheritPermsNilService WR-03(Phase 61 review):
// MultiAuth(fakeSvc, logger, nil, nil) 且 key 开 InheritPerms=true 时,
// 修复前在 permSvc.GetUserPermissions 处 nil 接口解引用 panic;
// 修复后防御性 401 "用户权限加载失败"(D-09 fail-closed),handler 不可达。
func TestMultiAuth_InheritPermsNilService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uid := "11111111-2222-3333-4444-555555555555"
	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel:    models.BaseModel{ID: "ak-nil-permsvc"},
			Name:         "nil-permsvc-key",
			UserID:       &uid, // UserID 非 nil — 单独隔离 permSvc/db nil 路径
			Scopes:       []string{"read"},
			InheritPerms: true, // 关键: 打开继承权限,触发 permSvc 调用路径
			IsActive:     true,
		},
	}
	fakeLogger := newFakeUsageLogger()

	router := gin.New()
	router.Use(gin.Recovery()) // 防御: 若修复回退,panic 被 recovery 吞为 500 而非 crash 测试进程
	router.Use(MultiAuth(fakeSvc, fakeLogger, nil, nil))
	router.GET("/probe", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	req.Header.Set("X-API-Key", "rec_"+hex64())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code,
		"InheritPerms=true 但 permSvc/db 为 nil 应 401 fail-closed (WR-03), 不得 panic/500")
	assert.Contains(t, w.Body.String(), "用户权限加载失败")
}

// --- QUAL-01 / D-12 集成测试: 限流响应头跨 gin.Engine + 中间件链路实证 ---

// TestRateLimitHeadersInResponse 用真实 gin.Engine + 真实 services.NewRateLimiter +
// MultiAuth→RateLimitByScope 完整链路,断言 X-RateLimit-* 响应头是可被 strconv.Atoi
// 反解析的数字字面量 (D-11 修复 P2-a: string(rune(int)) → strconv.Itoa)。
//
// 与 apikey_test.go:TestRateLimitHeaderEncoding 的分工: 后者锁编码函数语义(纯单测),
// 本测试锁「中间件真的把该值写进了响应头」(跨 HTTP 边界的端到端证据)。
//
// Pitfall 4 防御: rateLimiter 必须传真实 services.NewRateLimiter() —— 传 nil 会在
// rateLimiter.Check 处 nil-pointer panic。
func TestRateLimitHeadersInResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel: models.BaseModel{ID: "ak-ratelimit"},
			Name:      "rl-key",
			Scopes:    []string{"read"}, // read 档: PerMinute=30 (rate_limiter.go:48)
			IsActive:  true,
		},
	}
	fakeLogger := newFakeUsageLogger()
	rl := services.NewRateLimiter(nil) // Pitfall 4: 不能传 nil *RateLimiter;Phase 61 起 NewRateLimiter 接收 provider,nil provider → static 兜底

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, fakeLogger, nil, nil))
	router.Use(RateLimitByScope(rl, "list")) // Phase 61 / D-11: 新增 action 参数
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	req.Header.Set("X-API-Key", "rec_"+hex64())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fakeLogger.waitForLog(t) // 等待异步使用日志 goroutine, 消除跨测试竞态
	assert.Equal(t, 200, w.Code, "首个请求未触限流, 应 200")

	limitHeader := w.Header().Get("X-RateLimit-Limit")
	remainingHeader := w.Header().Get("X-RateLimit-Remaining")
	resetHeader := w.Header().Get("X-RateLimit-Reset")

	// 核心断言 (SC#4): 限流头可被标准工具反解析
	n, err := strconv.Atoi(limitHeader)
	assert.NoError(t, err, "X-RateLimit-Limit 必须是数字字符串, 实际=%q", limitHeader)
	assert.Greater(t, n, 0, "X-RateLimit-Limit 应为正整数")

	n2, err := strconv.Atoi(remainingHeader)
	assert.NoError(t, err, "X-RateLimit-Remaining 必须是数字字符串, 实际=%q", remainingHeader)
	assert.GreaterOrEqual(t, n2, 0, "X-RateLimit-Remaining 应为非负整数")
	assert.Less(t, n2, n, "消耗 1 次配额后 Remaining 应小于 Limit")

	// 防御性断言: P2-a 的 string(rune(100))=="d" 不得再出现
	assert.NotEqual(t, "d", limitHeader, "P2-a 回归: 限流头不得是 rune 字面量 \"d\"")
	assert.NotEqual(t, "c", remainingHeader, "P2-a 回归: 限流头不得是 rune 字面量 \"c\"")

	// Reset 头本就是 RFC3339 字符串 (D-11 明确不动), 顺带确认未被误改
	_, resetErr := time.Parse(time.RFC3339, resetHeader)
	assert.NoError(t, resetErr, "X-RateLimit-Reset 应保持 RFC3339 时间字符串, 实际=%q", resetHeader)
}

// --- D-02 构造函数证据 (SC#2 / AUTH-02) ---

// TestConstructorsCallable_D02 真实实例化 services.NewUsageLogger(db) 与 services.NewRateLimiter()，
// 并把返回值喂给 MultiAuth 第 2 参 / RateLimitByScope 第 1 参。
// 编译通过即证明 4 中间件签名自洽且构造函数可装配 (Pitfall 4: 用 services.NewRateLimiter 无参版)。
func TestConstructorsCallable_D02(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := services.NewUsageLogger(db)
	assert.NotNil(t, logger)
	_ = MultiAuth(&fakeAPIKeyService{}, logger, nil, nil) // 编译通过即证明 MultiAuth 第 2 参接受 services.UsageLogger

	rl := services.NewRateLimiter(nil)
	assert.NotNil(t, rl)
	_ = RateLimitByScope(rl, "list") // 编译通过即证明 RateLimitByScope 第 1 参接受 *services.RateLimiter, 第 2 参接受 action (Phase 61 / D-11)
}

// --- Phase 59 Plan 02: SC#1 / SC#2 DB 行实证 (Wave 2 回归锚) ---

// waitForUsageLog 用 require.Eventually 轮询 DB 行数, 替代既有 time.Sleep flaky 反模式。
// 形态镜像 RESEARCH.md §异步写入可测试性机制 — 同形副本落本文件因 Go 测试包隔离。
func waitForUsageLog(t *testing.T, db *gorm.DB, apiKeyID string, want int64) {
	t.Helper()
	require.Eventually(t, func() bool {
		var count int64
		db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", apiKeyID).Count(&count)
		return count >= want
	}, 2*time.Second, 10*time.Millisecond,
		"usage log for key=%s not persisted within 2s", apiKeyID)
}

// TestMultiAuthUsageLogTiming SC#1: 2xx 请求后日志行 StatusCode=200 / Duration>0 / Success=true。
// 真实 services.NewUsageLogger + 真实 gin.Engine + 真实 sqlite DB 行断言 (非 fake)。
// D-03 落实: 用既有 setupUsageLoggerTestDB (per-test 独立文件 DB) 复用, 不另起 AutoMigrate。
func TestMultiAuthUsageLogTiming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupUsageLoggerTestDB(t)
	realLogger := services.NewUsageLogger(db) // 真实 logger,非 fake
	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel: models.BaseModel{ID: "ak-sc1"},
			Name:      "sc1-key",
			Scopes:    []string{"read"},
			IsActive:  true,
		},
	}

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, realLogger, nil, nil))
	router.GET("/ok", RequireScope("read"), func(c *gin.Context) {
		// 微小 sleep 确保 Duration 字段 > 0ms (time.Since().Milliseconds() 取整)
		time.Sleep(time.Millisecond)
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	req.Header.Set("X-API-Key", "rec_"+hex64())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "2xx 请求应 200")

	// 等待异步落库 (替代 time.Sleep flaky)
	waitForUsageLog(t, db, "ak-sc1", 1)

	// DB 行实证 (SC#1)
	var log models.APIKeyUsageLog
	require.NoError(t, db.Where("api_key_id = ?", "ak-sc1").First(&log).Error)
	assert.Equal(t, 200, log.StatusCode)
	assert.Greater(t, log.Duration, 0)
	assert.True(t, log.Success) // D-01: 2xx → Success=true
}

// TestMultiAuthUsageLogFailure SC#2: 下游 RequireScope→403 → StatusCode=403 / Success=false。
// Pitfall 3 规避: 用 RequireScope→403 走下游 c.Next() (非 pre-auth 401, pre-auth 在 c.Next() 前 abort 不写行)。
func TestMultiAuthUsageLogFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := setupUsageLoggerTestDB(t)
	realLogger := services.NewUsageLogger(db)
	fakeSvc := &fakeAPIKeyService{
		validKey: &models.APIKey{
			BaseModel: models.BaseModel{ID: "ak-fail"},
			Name:      "fail-key",
			Scopes:    []string{"read"}, // 仅 read scope
			IsActive:  true,
		},
	}

	router := gin.New()
	router.Use(MultiAuth(fakeSvc, realLogger, nil, nil))
	// key 仅 read scope, RequireScope("write") 在 c.Next() 下游 abort 403
	router.GET("/write-only", RequireScope("write"), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/write-only", nil)
	req.Header.Set("X-API-Key", "rec_"+hex64())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "下游 RequireScope 失败应 403")

	// 等待异步落库
	waitForUsageLog(t, db, "ak-fail", 1)

	// DB 行实证 (SC#2)
	var log models.APIKeyUsageLog
	require.NoError(t, db.Where("api_key_id = ?", "ak-fail").First(&log).Error)
	assert.Equal(t, 403, log.StatusCode)
	assert.False(t, log.Success) // D-01: 4xx → Success=false
}
