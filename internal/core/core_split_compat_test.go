// Package core 提供 P2-A1 (Core god struct split) 的回归测试
//
// 背景:
//   - 内部 core.Core 是一个 god struct,拥有 18+ 字段(基础设施 + 业务服务混合)。
//   - 拆分目标:将基础设施字段(DB/Cache/JWT/Pwd/...) 移入 *CoreInfra,
//     业务服务字段(DeviceExecutor/NoticeHub/OperLogService/...) 移入 *CoreServices。
//   - 通过 struct embedding 保持向后兼容的字段访问语法
//     (core.DB, core.Cache, core.NoticeHub 等仍可直接访问 — Go 字段提升机制)。
//
// 验证:
//   - 通过字面量构造一个 *core.Core,使用新的 CoreInfra/CoreServices 字段语法
//   - 验证旧式字段访问 (core.DB, core.UserService 等) 仍能正常工作 (字段提升)
//   - 验证 fields 仍可读写 (即嵌入不会因为类型断言失败而无法设置)
package core

import (
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newTestCoreForSplitCompat 构造一个最小可用的 *core.Core,使用新的 CoreInfra/CoreServices 嵌入语法。
// 主要验证:
//  1. 通过 CoreInfra 设置 DB,通过 CoreServices 设置 NoticeHub
//  2. 通过字段提升 (core.DB, core.NoticeHub) 仍能访问这些字段
//  3. 通过 core.CoreInfra / core.CoreServices 也能直接访问 (因为 *CoreInfra 是指针字段)
func newTestCoreForSplitCompat(t *testing.T) *Core {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}

	noticeHub := websocket.NewNoticeHub()

	// 通过嵌入语法构造 Core
	c := &Core{
		CoreInfra: &CoreInfra{
			DB: &db.Database{DB: gormDB, Type: "sqlite"},
		},
		CoreServices: &CoreServices{
			NoticeHub: noticeHub,
		},
	}

	return c
}

// TestCoreSplit_BackwardCompat 验证 Core 拆分后旧式字段访问仍能工作
func TestCoreSplit_BackwardCompat(t *testing.T) {
	c := newTestCoreForSplitCompat(t)

	// 1) 验证嵌入式结构字段已正确赋值
	assert.NotNil(t, c.CoreInfra, "CoreInfra should be set via literal construction")
	assert.NotNil(t, c.CoreServices, "CoreServices should be set via literal construction")

	// 2) 验证字段提升:旧式访问语法仍可用
	assert.NotNil(t, c.DB, "field promotion: core.DB should be accessible")
	assert.NotNil(t, c.NoticeHub, "field promotion: core.NoticeHub should be accessible")

	// 3) 验证字段值是预期实例
	assert.Equal(t, "sqlite", c.DB.Type)
	assert.NotNil(t, c.NoticeHub)

	// 4) 验证 GetDB() 方法仍能正常工作 (这是 Core 上的方法,不会因字段提升而丢失)
	gormDB := c.GetDB()
	assert.NotNil(t, gormDB, "GetDB() method on Core should still work")
}

// TestCoreSplit_FieldPromotionMatchesCoreInfra 验证字段提升值与 CoreInfra / CoreServices 字段值是同一引用
func TestCoreSplit_FieldPromotionMatchesCoreInfra(t *testing.T) {
	c := newTestCoreForSplitCompat(t)

	// c.DB (字段提升) 必须指向 c.CoreInfra.DB (同一指针)
	assert.Same(t, c.CoreInfra.DB, c.DB,
		"c.DB (promoted) and c.CoreInfra.DB must be the same pointer")

	// c.NoticeHub (字段提升) 必须指向 c.CoreServices.NoticeHub (同一指针)
	assert.Same(t, c.CoreServices.NoticeHub, c.NoticeHub,
		"c.NoticeHub (promoted) and c.CoreServices.NoticeHub must be the same pointer")
}

// minimalTestConfig 返回一个能让 New() 成功执行的最小配置
// New() 内部会调用 security.NewJWTManager (要求 SecretKey ≥16 字节) 与
// initSM4Cipher (要求 SM4Key 为 base64 编码的 16 字节,空值会中止启动)。
func minimalTestConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			SecretKey:        "test-secret-key-32bytes-min!",
			AccessKeyExpire:  3600,
			RefreshKeyExpire: 86400,
			Issuer:           "test",
		},
		Security: config.SecurityConfig{
			// base64("test-key-16-byte") — 16 字节,非仓库默认值以避免 SECURITY WARNING 刷屏
			SM4Key: "dGVzdC1rZXktMTYtYnl0ZQ==",
		},
	}
}

// TestCoreSplit_NewConstructorPopulatesInfraAndServices 验证 New() 构造函数同时初始化 CoreInfra 和 CoreServices
func TestCoreSplit_NewConstructorPopulatesInfraAndServices(t *testing.T) {
	// 使用最小配置调用 New,验证 CoreInfra 和 CoreServices 都被实例化
	// 注:不调用 Init,只验证 New 的输出结构正确
	cfg := minimalTestConfig()

	c, err := New(cfg)
	assert.NoError(t, err, "New() should not return an error with minimal config")
	assert.NotNil(t, c)
	assert.NotNil(t, c.CoreInfra, "New() must initialize CoreInfra")
	assert.NotNil(t, c.CoreServices, "New() must initialize CoreServices")
	// Config 必须从 New 透传到 CoreInfra
	assert.Same(t, cfg, c.CoreInfra.Config,
		"New() must propagate config into CoreInfra.Config")
	// JWTManager 必须被实例化
	assert.NotNil(t, c.JWTManager, "New() must instantiate JWTManager in CoreInfra")
	// PwdManager 必须被实例化
	assert.NotNil(t, c.PwdManager, "New() must instantiate PwdManager in CoreInfra")
}
