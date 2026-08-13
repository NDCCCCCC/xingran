package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		router.Use(MultiAuth(fakeSvc, fakeLogger))
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
		router.Use(MultiAuth(fakeSvc, fakeLogger))
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
		router.Use(MultiAuth(fakeSvc, fakeLogger))
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

// --- D-02 构造函数证据 (SC#2 / AUTH-02) ---

// TestConstructorsCallable_D02 真实实例化 services.NewUsageLogger(db) 与 services.NewRateLimiter()，
// 并把返回值喂给 MultiAuth 第 2 参 / RateLimitByScope 第 1 参。
// 编译通过即证明 4 中间件签名自洽且构造函数可装配 (Pitfall 4: 用 services.NewRateLimiter 无参版)。
func TestConstructorsCallable_D02(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := services.NewUsageLogger(db)
	assert.NotNil(t, logger)
	_ = MultiAuth(&fakeAPIKeyService{}, logger) // 编译通过即证明 MultiAuth 第 2 参接受 services.UsageLogger

	rl := services.NewRateLimiter()
	assert.NotNil(t, rl)
	_ = RateLimitByScope(rl) // 编译通过即证明 RateLimitByScope 第 1 参接受 *services.RateLimiter
}
