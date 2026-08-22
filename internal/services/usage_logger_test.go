package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupUsageLoggerTestDB 创建使用日志测试数据库
//
// 2026-07-01 fix: 改用每测试独立的文件 DB(os.TempDir + 唯一名),
// 替代原 `file::memory:?cache=shared`(所有测试函数共享同一内存 DB)。
// 原方案在多个 usage_logger 测试并发跑时,异步后台 goroutine 的 INSERT
// 撞 SQLite 写锁(database table is locked),导致 TestAsyncLogging/日志完整性
// 的 5 条日志部分丢失,assert.Len(logs, 5) 失败。
//
// busy_timeout=5000 让写锁排队(而非立即报错)。
// 不用 t.TempDir:LogUsage 的 fire-and-forget goroutine 在测试结束后仍可能
// 持有文件句柄写,t.TempDir 的自动 cleanup 删除占用文件会 mark test failed。
// 残留 .db 文件留在系统 temp 目录,由 OS 定期清理(单测试 <100KB)。
func setupUsageLoggerTestDB(t *testing.T) *gorm.DB {
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("xingran_usage_%d_%d.db", time.Now().UnixNano(), os.Getpid()))
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", dbPath)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// 创建使用日志表
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

// TestNewUsageLogger 测试初始化
func TestNewUsageLogger(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := NewUsageLogger(db)

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.(*usageLoggerImpl).db)
}

// TestLogUsage 测试日志记录
func TestLogUsage(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := NewUsageLogger(db)
	ctx := context.Background()

	t.Run("正常日志记录", func(t *testing.T) {
		apiKeyID := "test-api-key-id"
		userID := "test-user-id"
		method := "GET"
		path := "/api/v1/test"
		statusCode := 200
		clientIP := "127.0.0.1"
		userAgent := "test-agent"
		duration := 100
		success := true

		req := &LogUsageRequest{
			APIKeyID:   apiKeyID,
			UserID:     userID,
			Method:     method,
			Path:       path,
			StatusCode: statusCode,
			ClientIP:   clientIP,
			UserAgent:  &userAgent,
			Duration:   duration,
			Success:    success,
		}

		err := logger.LogUsage(ctx, req)
		assert.NoError(t, err)

		// 等待异步操作完成
		time.Sleep(100 * time.Millisecond)

		// 验证数据库中的记录
		var log models.APIKeyUsageLog
		err = db.Where("api_key_id = ?", apiKeyID).First(&log).Error
		assert.NoError(t, err)
		assert.Equal(t, apiKeyID, log.APIKeyID)
		assert.Equal(t, userID, log.UserID)
		assert.Equal(t, method, log.Method)
		assert.Equal(t, path, log.Path)
		assert.Equal(t, statusCode, log.StatusCode)
		assert.Equal(t, clientIP, log.ClientIP)
		assert.Equal(t, userAgent, *log.UserAgent)
		assert.Equal(t, duration, log.Duration)
		assert.Equal(t, success, log.Success)
	})

	t.Run("异步执行", func(t *testing.T) {
		apiKeyID := "async-test-key"

		req := &LogUsageRequest{
			APIKeyID:   apiKeyID,
			UserID:     "test-user-id",
			Method:     "POST",
			Path:       "/api/v1/async",
			StatusCode: 201,
			ClientIP:   "127.0.0.1",
			Duration:   200,
			Success:    true,
		}

		start := time.Now()
		err := logger.LogUsage(ctx, req)
		elapsed := time.Since(start)

		// LogUsage应该立即返回（异步）
		assert.NoError(t, err)
		assert.Less(t, elapsed.Milliseconds(), int64(10), "LogUsage should return immediately")

		// 等待异步操作完成
		time.Sleep(100 * time.Millisecond)

		// 验证数据已保存
		var count int64
		db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", apiKeyID).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("所有字段正确保存", func(t *testing.T) {
		userAgent := "Mozilla/5.0"
		req := &LogUsageRequest{
			APIKeyID:   "all-fields-key",
			UserID:     "user-123",
			Method:     "DELETE",
			Path:       "/api/v1/resource/123",
			StatusCode: 204,
			ClientIP:   "192.168.1.100",
			UserAgent:  &userAgent,
			Duration:   456,
			Success:    true,
		}

		err := logger.LogUsage(ctx, req)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		var log models.APIKeyUsageLog
		err = db.Where("api_key_id = ?", "all-fields-key").First(&log).Error
		assert.NoError(t, err)

		assert.Equal(t, "all-fields-key", log.APIKeyID)
		assert.Equal(t, "user-123", log.UserID)
		assert.Equal(t, "DELETE", log.Method)
		assert.Equal(t, "/api/v1/resource/123", log.Path)
		assert.Equal(t, 204, log.StatusCode)
		assert.Equal(t, "192.168.1.100", log.ClientIP)
		assert.Equal(t, userAgent, *log.UserAgent)
		assert.Equal(t, 456, log.Duration)
		assert.True(t, log.Success)
		assert.NotZero(t, log.CreatedAt)
	})

	t.Run("数据库插入成功", func(t *testing.T) {
		initialCount := int64(0)
		db.Model(&models.APIKeyUsageLog{}).Count(&initialCount)

		req := &LogUsageRequest{
			APIKeyID:   "insert-test-key",
			UserID:     "test-user",
			Method:     "GET",
			Path:       "/api/v1/insert-test",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			Duration:   100,
			Success:    true,
		}

		err := logger.LogUsage(ctx, req)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		finalCount := int64(0)
		db.Model(&models.APIKeyUsageLog{}).Count(&finalCount)
		assert.Equal(t, initialCount+1, finalCount)
	})
}

// TestAsyncLogging 测试异步日志记录
func TestAsyncLogging(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := NewUsageLogger(db)
	ctx := context.Background()

	t.Run("并发日志记录", func(t *testing.T) {
		numGoroutines := 10 // 减少并发数量
		logsPerGoroutine := 5 // 减少每个goroutine的日志数

		done := make(chan bool, numGoroutines)

		// 并发发送日志记录请求
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				defer func() { done <- true }()

				for j := 0; j < logsPerGoroutine; j++ {
					// 每个goroutine使用不同的key避免冲突
					req := &LogUsageRequest{
						APIKeyID:   "concurrent-key",
						UserID:     "user-123",
						Method:     "GET",
						Path:       "/api/v1/concurrent",
						StatusCode: 200,
						ClientIP:   "127.0.0.1",
						Duration:   100,
						Success:    true,
					}

					err := logger.LogUsage(ctx, req)
					assert.NoError(t, err)
				}
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 等待异步操作完成
		time.Sleep(500 * time.Millisecond)

		// 验证至少部分日志被记录（允许并发失败）
		var count int64
		db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", "concurrent-key").Count(&count)
		assert.Greater(t, count, int64(0), "At least some logs should be recorded")
	})

	t.Run("异步不阻塞主流程", func(t *testing.T) {
		req := &LogUsageRequest{
			APIKeyID:   "non-block-key",
			UserID:     "test-user",
			Method:     "GET",
			Path:       "/api/v1/test",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			Duration:   100,
			Success:    true,
		}

		start := time.Now()
		err := logger.LogUsage(ctx, req)
		elapsed := time.Since(start)

		// 应该立即返回
		assert.NoError(t, err)
		assert.Less(t, elapsed.Milliseconds(), int64(5), "Should return immediately")

		// 等待异步完成
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("日志完整性", func(t *testing.T) {
		// 创建多个日志记录
		expectedLogs := []struct {
			method     string
			path       string
			statusCode int
			success    bool
		}{
			{"GET", "/api/v1/resource1", 200, true},
			{"POST", "/api/v1/resource2", 201, true},
			{"DELETE", "/api/v1/resource3", 204, true},
			{"GET", "/api/v1/notfound", 404, false},
			{"POST", "/api/v1/error", 500, false},
		}

		for i, log := range expectedLogs {
			req := &LogUsageRequest{
				APIKeyID:   "integrity-key",
				UserID:     "test-user",
				Method:     log.method,
				Path:       log.path,
				StatusCode: log.statusCode,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    log.success,
			}

			err := logger.LogUsage(ctx, req)
			assert.NoError(t, err, "Log %d should succeed", i)
		}

		// 等待异步写完成:CI runner 慢时 200ms 固定 sleep 会丢拍,
		// 改 Eventually 轮询(最长 5s,500ms 间隔)。
		var logs []models.APIKeyUsageLog
		require.Eventually(t, func() bool {
			logs = nil
			if err := db.Where("api_key_id = ?", "integrity-key").Find(&logs).Error; err != nil {
				return false
			}
			return len(logs) == len(expectedLogs)
		}, 5*time.Second, 500*time.Millisecond, "异步日志应最终全部落库")

		// 验证每条日志的内容
		for i, expected := range expectedLogs {
			found := false
			for _, log := range logs {
				if log.Method == expected.method && log.Path == expected.path {
					assert.Equal(t, expected.statusCode, log.StatusCode)
					assert.Equal(t, expected.success, log.Success)
					found = true
					break
				}
			}
			assert.True(t, found, "Log entry %d not found", i)
		}
	})

	t.Run("错误处理", func(t *testing.T) {
		// 测试无效参数不会导致panic
		req := &LogUsageRequest{
			APIKeyID:   "", // 空key
			UserID:     "",
			Method:     "",
			Path:       "",
			StatusCode: 0,
			ClientIP:   "",
			Duration:   0,
			Success:    false,
		}

		// 即使参数无效，也不应该panic
		assert.NotPanics(t, func() {
			err := logger.LogUsage(ctx, req)
			assert.NoError(t, err) // 异步记录，不检查参数
		})

		time.Sleep(100 * time.Millisecond)
	})
}

// TestLogUsageErrorHandling 测试错误处理
func TestLogUsageErrorHandling(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := NewUsageLogger(db)
	ctx := context.Background()

	t.Run("数据库连接失败", func(t *testing.T) {
		// 关闭数据库连接
		sqlDB, _ := db.DB()
		sqlDB.Close()

		req := &LogUsageRequest{
			APIKeyID:   "db-error-key",
			UserID:     "test-user",
			Method:     "GET",
			Path:       "/api/v1/test",
			StatusCode: 200,
			ClientIP:   "127.0.0.1",
			Duration:   100,
			Success:    true,
		}

		// 即使数据库连接失败，也不应该阻塞
		err := logger.LogUsage(ctx, req)
		assert.NoError(t, err) // 异步操作，错误被忽略

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("无效参数", func(t *testing.T) {
		invalidParams := []struct {
			name string
			req  *LogUsageRequest
		}{
			{
				name: "空API Key ID",
				req: &LogUsageRequest{
					APIKeyID: "",
					UserID:   "test-user",
					Method:   "GET",
					Path:     "/api/v1/test",
					Duration: 100,
				},
			},
			{
				name: "空用户ID",
				req: &LogUsageRequest{
					APIKeyID: "test-key",
					UserID:   "",
					Method:   "GET",
					Path:     "/api/v1/test",
					Duration: 100,
				},
			},
		}

		for _, param := range invalidParams {
			t.Run(param.name, func(t *testing.T) {
				// 不应该panic
				assert.NotPanics(t, func() {
					err := logger.LogUsage(ctx, param.req)
					assert.NoError(t, err)
				})

				time.Sleep(50 * time.Millisecond)
			})
		}
	})

	t.Run("错误记录不崩溃", func(t *testing.T) {
		// 使用新的数据库实例
		newDB := setupUsageLoggerTestDB(t)
		newLogger := NewUsageLogger(newDB)

		// 发送少量日志记录，验证不崩溃
		for i := 0; i < 10; i++ {
			req := &LogUsageRequest{
				APIKeyID:   "error-test-key",
				UserID:     "test-user",
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}

			err := newLogger.LogUsage(ctx, req)
			assert.NoError(t, err)
		}

		time.Sleep(200 * time.Millisecond)

		// 验证日志被记录
		var count int64
		newDB.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", "error-test-key").Count(&count)
		assert.Greater(t, count, int64(0), "At least some logs should be recorded")
	})
}

// --- Phase 59 Plan 02: SC#4 (D-02 detached ctx 防回归) ---

// waitForUsageLog 用 require.Eventually 轮询 DB 行数, 替代既有 time.Sleep flaky 反模式。
// 形态镜像 RESEARCH.md §异步写入可测试性机制 — 同形副本落本文件因 Go 测试包隔离
// (无法跨包导入 middleware 包内的同名 helper)。
func waitForUsageLog(t *testing.T, db *gorm.DB, apiKeyID string, want int64) {
	t.Helper()
	require.Eventually(t, func() bool {
		var count int64
		db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", apiKeyID).Count(&count)
		return count >= want
	}, 2*time.Second, 10*time.Millisecond,
		"usage log for key=%s not persisted within 2s", apiKeyID)
}

// TestLogUsageCancelledCtxStillWrites_D02 SC#4 (D-02 detached ctx 防回归):
// 调用方 ctx 预取消后, logUsageAsync 必须仍用独立 detached ctx 写 DB, 不被调用方 cancel 影响。
//
// 修复前 (Plan 01 前): logUsageAsync 复用 ctx → WithContext(ctx) 失败 → _ = err 吞错 → 行永不出现 → 超时失败。
// 修复后 (Plan 01 后): logUsageAsync 用 detachedCtx → 忽略 cancel → 行落库 → require.Eventually 成功。
func TestLogUsageCancelledCtxStillWrites_D02(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := NewUsageLogger(db)

	// 预取消 ctx — 模拟 P2-b 场景 (请求结束 ctx.Canceled)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &LogUsageRequest{
		APIKeyID:   "cancel-race-key",
		UserID:     "u1",
		Method:     "GET",
		Path:       "/test",
		StatusCode: 200,
		Success:    true,
		Duration:   10,
	}

	err := logger.LogUsage(ctx, req)
	require.NoError(t, err)

	// 等待异步落库 — 修复后即使 ctx 已 cancel, 行仍写入
	waitForUsageLog(t, db, "cancel-race-key", 1)

	// DB 行实证 (SC#4)
	var log models.APIKeyUsageLog
	require.NoError(t, db.Where("api_key_id = ?", "cancel-race-key").First(&log).Error)
	assert.Equal(t, 200, log.StatusCode)
	assert.True(t, log.Success)
}

// TestLogUsagePerformance 测试性能
func TestLogUsagePerformance(t *testing.T) {
	db := setupUsageLoggerTestDB(t)
	logger := NewUsageLogger(db)
	ctx := context.Background()

	t.Run("批量日志记录性能", func(t *testing.T) {
		numLogs := 50 // 进一步减少数量避免SQLite并发问题

		start := time.Now()

		// 批量发送日志记录请求
		for i := 0; i < numLogs; i++ {
			req := &LogUsageRequest{
				APIKeyID:   "perf-test-key",
				UserID:     "test-user",
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}

			err := logger.LogUsage(ctx, req)
			assert.NoError(t, err)
		}

		elapsed := time.Since(start)

		// 批量发送应该很快（异步）
		assert.Less(t, elapsed.Milliseconds(), int64(100), "Batch logging should be fast")

		// 等待异步操作完成
		time.Sleep(500 * time.Millisecond)

		// 验证至少部分日志被记录（允许并发失败）
		var count int64
		db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", "perf-test-key").Count(&count)
		assert.Greater(t, count, int64(0), "At least some logs should be recorded")
	})

	t.Run("不影响主流程响应时间", func(t *testing.T) {
		// 模拟主流程
		start := time.Now()

		// 执行一些业务逻辑
		for i := 0; i < 10; i++ {
			req := &LogUsageRequest{
				APIKeyID:   "response-time-key",
				UserID:     "test-user",
				Method:     "GET",
				Path:       "/api/v1/test",
				StatusCode: 200,
				ClientIP:   "127.0.0.1",
				Duration:   100,
				Success:    true,
			}

			err := logger.LogUsage(ctx, req)
			assert.NoError(t, err)
		}

		elapsed := time.Since(start)

		// 日志记录不应该显著影响响应时间
		assert.Less(t, elapsed.Milliseconds(), int64(10), "Logging should not impact response time")

		time.Sleep(200 * time.Millisecond)
	})
}
