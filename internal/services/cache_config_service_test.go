package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupCacheConfigTestDB 创建 CacheConfigService 测试数据库 (SQLite 内存模式,
// Phase 59 D-03 模式复用;与 system/config_invalidation_test.go 同形副本,
// 因 Go 测试包隔离无法跨包引用)。
func setupCacheConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	dsn := "file:" + safeName + "?mode=memory&cache=private"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_config (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			config_name TEXT NOT NULL,
			config_key TEXT NOT NULL UNIQUE,
			config_value TEXT,
			config_type TEXT DEFAULT 'Y',
			is_system INTEGER DEFAULT 0,
			remark TEXT
		)
	`).Error
	require.NoError(t, err)

	return db
}

// TestCacheConfigService_RateLimitDefaults D-16/D-17: 12 个 rate_limit.* 配置键
// 自动写入 DB,默认值与既有 rate_limiter.go 硬编码一致
func TestCacheConfigService_RateLimitDefaults(t *testing.T) {
	db := setupCacheConfigTestDB(t)
	svc := NewCacheConfigService(db)
	require.NotNil(t, svc)

	// 默认值断言(D-17 与既有硬编码一致)
	assert.Equal(t, 30, svc.GetRateLimit(RateLimitReadPerMinute, 999))
	assert.Equal(t, 1500, svc.GetRateLimit(RateLimitWritePerHour, 999))
	assert.Equal(t, 20000, svc.GetRateLimit(RateLimitDefaultPerDay, 999))
	assert.Equal(t, 200, svc.GetRateLimit(RateLimitAdminPerMinute, 999))

	// DB 中应存在 12 条 rate_limit.* 记录(由 setDefaultsIfNeeded 自动 INSERT)
	var count int64
	require.NoError(t, db.Model(&models.Config{}).
		Where("config_key LIKE ?", "rate_limit.%").Count(&count).Error)
	assert.Equal(t, int64(12), count, "应自动写入 12 条 rate_limit.* 默认配置")
}

// TestCacheConfigService_ReloadRateLimit D-19: reload 后新阈值对新请求生效
func TestCacheConfigService_ReloadRateLimit(t *testing.T) {
	db := setupCacheConfigTestDB(t)
	svc := NewCacheConfigService(db)
	require.NotNil(t, svc)

	// 直接 UPDATE sys_config 表
	require.NoError(t, db.Model(&models.Config{}).
		Where("config_key = ?", RateLimitReadPerMinute).
		Update("config_value", "5").Error)

	// reload 前仍是旧值
	assert.Equal(t, 30, svc.GetRateLimit(RateLimitReadPerMinute, 999))

	// reload 后新阈值生效
	require.NoError(t, svc.ReloadConfig(context.Background()))
	assert.Equal(t, 5, svc.GetRateLimit(RateLimitReadPerMinute, 999),
		"reload 后新阈值应对新请求生效 (D-19)")
}

// TestCacheConfigService_RangeValidation D-16: 超出 Min/Max 范围的值被重置为默认值,
// DB 自动修复(沿用既有 cache.* range 校验模式)
func TestCacheConfigService_RangeValidation(t *testing.T) {
	db := setupCacheConfigTestDB(t)
	svc := NewCacheConfigService(db)
	require.NotNil(t, svc)

	// INSERT 一条超出 Max=10000 的值
	require.NoError(t, db.Model(&models.Config{}).
		Where("config_key = ?", RateLimitReadPerMinute).
		Update("config_value", "99999").Error)

	// 重新加载 → 触发 range 校验 → 重置为默认值 30
	require.NoError(t, svc.LoadConfigs(context.Background()))
	assert.Equal(t, 30, svc.GetRateLimit(RateLimitReadPerMinute, 999),
		"超出范围的值应被重置为默认值")

	// DB 中的值应被自动修复回 30
	var cfg models.Config
	require.NoError(t, db.Where("config_key = ?", RateLimitReadPerMinute).First(&cfg).Error)
	assert.Equal(t, "30", cfg.ConfigValue, "DB 中的非法值应被自动修复")
}

// TestCacheConfigService_CacheUnaffected D-15: 新增 rate_limit.* 加载不破坏
// 既有 cache.* 加载逻辑
func TestCacheConfigService_CacheUnaffected(t *testing.T) {
	db := setupCacheConfigTestDB(t)
	svc := NewCacheConfigService(db)
	require.NotNil(t, svc)

	// 既有 cache.* 默认值仍正确(30 分钟)
	assert.Equal(t, 30*time.Minute, svc.GetDuration(CacheConfigDeptTree),
		"既有 cache.* 配置不应受 rate_limit.* 新增影响")
}

// TestRateLimitProviderInterface D-18: *CacheConfigService 自动实现
// RateLimitProvider 接口(编译期断言)
func TestRateLimitProviderInterface(t *testing.T) {
	var _ RateLimitProvider = (*CacheConfigService)(nil)

	db := setupCacheConfigTestDB(t)
	var provider RateLimitProvider = NewCacheConfigService(db)
	assert.Equal(t, 5000, provider.GetRateLimit(RateLimitReadPerDay, 999),
		"通过接口读取限流配置应返回默认值")
	assert.Equal(t, 999, provider.GetRateLimit("rate_limit.nonexist.per_minute", 999),
		"未知 key 应返回调用方传入的 defaultValue")
}
