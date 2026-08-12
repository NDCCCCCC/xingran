package system

import (
	"context"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupConfigTestDB 创建 ConfigService 测试数据库 (SQLite 内存模式)。
// 每个测试用独立 cache name 隔离 in-memory DB,避免 sys_config UNIQUE 冲突;
// 显式建表以避免 SQLite 与 PostgreSQL 在 gen_random_uuid() 等扩展函数上的不兼容。
func setupConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// t.Name() 包含 "/" (子测试), 直接用作 DSN 会触发 SQLite URI 解析错误;
	// 用 strings.ReplaceAll 规整后再加随机后缀避免极端情况重复。
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

// seedConfig 直接 INSERT 一条 sys_config 记录,绕过 GORM 的 UUID 自动填充,
// 避免 SQLite 在测试环境下对 uuid.New().String() 的额外开销。
func seedConfig(t *testing.T, db *gorm.DB, id, key, value string) {
	t.Helper()
	err := db.Exec(`
		INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system)
		VALUES (?, ?, ?, ?, 'Y', 0)
	`, id, key, key, value).Error
	require.NoError(t, err)
}

// withEncryptionCallback 临时替换包级 OnEncryptionConfigChanged 变量为给定回调,
// 测试结束自动恢复。返回 counter 的最终值通过闭包捕获。
func withEncryptionCallback(t *testing.T, cb func()) {
	t.Helper()
	original := OnEncryptionConfigChanged
	OnEncryptionConfigChanged = cb
	t.Cleanup(func() {
		OnEncryptionConfigChanged = original
	})
}

// TestConfigService_UpdateEncryptionFlag_InvalidatesMiddlewareCache 验证
// P1-B1 修复: 修改 sys.request.encryption.enabled 后,中间件加密缓存立即失效。
//
// 子测试:
//  1. encryption flag false→true: callback 触发 1 次,DB 值更新。
//  2. encryption flag true→false: callback 再触发 1 次(总计 2 次)。
//  3. 非 encryption flag (sys.user.init.password): callback 0 次,隔离性验证。
func TestConfigService_UpdateEncryptionFlag_InvalidatesMiddlewareCache(t *testing.T) {
	ctx := context.Background()

	// Part A: encryption false → true
	t.Run("encryption_false_to_true_invalidates_cache", func(t *testing.T) {
		db := setupConfigTestDB(t)
		service := NewConfigService(db)

		encryptionID := "11111111-1111-1111-1111-111111111111"
		seedConfig(t, db, encryptionID, "sys.request.encryption.enabled", "false")

		callCount := 0
		withEncryptionCallback(t, func() {
			callCount++
		})

		req := &requests.ConfigUpdateRequest{
			ID:          encryptionID,
			ConfigName:  "请求加密开关",
			ConfigValue: "true",
			ConfigType:  models.ConfigTypeYes,
		}

		err := service.Update(ctx, req)
		require.NoError(t, err)

		// 回调必须恰好触发 1 次
		assert.Equal(t, 1, callCount, "OnEncryptionConfigChanged 应在 encryption flag 更新时恰好触发 1 次")

		// DB 值必须已更新
		var stored models.Config
		err = db.Where("id = ?", encryptionID).First(&stored).Error
		require.NoError(t, err)
		assert.Equal(t, "true", stored.ConfigValue, "DB 中 encryption flag 应已更新为 true")
	})

	// Part B: encryption true → false
	t.Run("encryption_true_to_false_invalidates_cache", func(t *testing.T) {
		db := setupConfigTestDB(t)
		service := NewConfigService(db)

		encryptionID := "22222222-2222-2222-2222-222222222222"
		seedConfig(t, db, encryptionID, "sys.request.encryption.enabled", "true")

		callCount := 0
		withEncryptionCallback(t, func() {
			callCount++
		})

		req := &requests.ConfigUpdateRequest{
			ID:          encryptionID,
			ConfigName:  "请求加密开关",
			ConfigValue: "false",
			ConfigType:  models.ConfigTypeYes,
		}

		err := service.Update(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, 1, callCount, "OnEncryptionConfigChanged 应在 encryption flag 更新时触发 1 次")
		assert.Equal(t, "false", storedConfigValue(t, db, encryptionID))
	})

	// Part C: 非 encryption flag 修改不应触发 callback
	t.Run("non_encryption_flag_does_not_invalidate_cache", func(t *testing.T) {
		db := setupConfigTestDB(t)
		service := NewConfigService(db)

		passwordID := "33333333-3333-3333-3333-333333333333"
		seedConfig(t, db, passwordID, "sys.user.init.password", "123456")

		callCount := 0
		withEncryptionCallback(t, func() {
			callCount++
		})

		req := &requests.ConfigUpdateRequest{
			ID:          passwordID,
			ConfigName:  "用户初始密码",
			ConfigValue: "654321",
			ConfigType:  models.ConfigTypeYes,
		}

		err := service.Update(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, 0, callCount, "非 encryption flag 更新不应触发 OnEncryptionConfigChanged")
		assert.Equal(t, "654321", storedConfigValue(t, db, passwordID))
	})
}

// TestConfigService_UpdateEncryptionFlag_NilCallbackSafe 验证未注入 callback
// 的场景(测试或独立构建)不会因 nil 指针 panic — 静默跳过缓存失效即可。
func TestConfigService_UpdateEncryptionFlag_NilCallbackSafe(t *testing.T) {
	db := setupConfigTestDB(t)
	service := NewConfigService(db)
	ctx := context.Background()

	// 显式清空 callback
	withEncryptionCallback(t, nil)

	encryptionID := "44444444-4444-4444-4444-444444444444"
	seedConfig(t, db, encryptionID, "sys.request.encryption.enabled", "false")

	req := &requests.ConfigUpdateRequest{
		ID:          encryptionID,
		ConfigName:  "请求加密开关",
		ConfigValue: "true",
		ConfigType:  models.ConfigTypeYes,
	}

	// 不应 panic,也不应返回 error
	assert.NotPanics(t, func() {
		err := service.Update(ctx, req)
		assert.NoError(t, err)
	}, "callback 为 nil 时 Update 必须静默跳过缓存失效,不允许 panic")

	assert.Equal(t, "true", storedConfigValue(t, db, encryptionID))
}

// storedConfigValue 辅助函数: 从 DB 读取 config_value 列。
func storedConfigValue(t *testing.T, db *gorm.DB, id string) string {
	t.Helper()
	var row struct {
		ConfigValue string
	}
	err := db.Raw("SELECT config_value FROM sys_config WHERE id = ?", id).Scan(&row).Error
	require.NoError(t, err)
	return row.ConfigValue
}
