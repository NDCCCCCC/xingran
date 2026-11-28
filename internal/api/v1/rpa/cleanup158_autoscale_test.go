package rpa

// Phase 158 / CLEANUP-01：自动扩缩容配置真持久化用例。
//
// 此前 GetAutoScaleConfig 直接返回硬编码默认值、UpdateAutoScaleConfig 丢弃请求体
// 只报一句「已更新」—— 配置无处可存、GET 恒等于常量。本用例锁定：
//   - POST 写入的值，GET 原样读回（读写一致，不是常量）
//   - 配置真落在 sys_config（key=sys.rpa.autoscale.config）→ 重启后仍生效
//   - 首次未配置时 GET 返回兜底默认值（不是假数据：写回后即一致）
//
// 真库：sqlite 文件库 + db.NewDatabase + 真实 AutoMigrate（与生产 sqlite 启动同入口）。

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// cleanup158Core 组装「真实 sqlite 库」的 core（无 AI provider 依赖）。
func cleanup158Core(t *testing.T) *core.Core {
	t.Helper()
	database, err := db.NewDatabase(&config.DatabaseConfig{
		Type: "sqlite",
		Path: filepath.Join(t.TempDir(), "cleanup158.db"),
	})
	require.NoError(t, err, "创建 sqlite 测试库失败")
	require.NoError(t, database.AutoMigrate(), "真实 AutoMigrate 必须成功")
	t.Cleanup(func() {
		if sqlDB, err := database.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return &core.Core{
		CoreInfra:    &core.CoreInfra{DB: database, Config: &config.Config{}},
		CoreServices: &core.CoreServices{},
	}
}

// TestCleanup01_AutoScaleConfig_NotConfigured_ReturnsDefaults 未配置时返回兜底默认值。
func TestCleanup01_AutoScaleConfig_NotConfigured_ReturnsDefaults(t *testing.T) {
	c := cleanup158Core(t)
	h := NewWorkerHandler(nil, c)

	ctx, w := newTestCtxWorker("GET", "/autoscale/config", nil)
	h.GetAutoScaleConfig(ctx)
	require.Equal(t, http.StatusOK, w.Code)

	var count int64
	require.NoError(t, c.GetDB().Model(&models.Config{}).
		Where("config_key = ?", "sys.rpa.autoscale.config").Count(&count).Error)
	assert.Equal(t, int64(0), count, "未配置时不得凭空造 sys_config 行")
}

// TestCleanup01_AutoScaleConfig_WriteThenRead_Persisted 写后读一致 + 真落库。
func TestCleanup01_AutoScaleConfig_WriteThenRead_Persisted(t *testing.T) {
	c := cleanup158Core(t)
	h := NewWorkerHandler(nil, c)

	want := AutoScaleConfig{
		Enabled:            true,
		ScaleUpThreshold:   33,
		ScaleDownThreshold: 7,
		MinConcurrency:     2,
		MaxConcurrency:     20,
		CheckInterval:      45,
	}

	ctx, w := newTestCtxWorker("POST", "/autoscale/config", want)
	h.UpdateAutoScaleConfig(ctx)
	require.Equal(t, http.StatusOK, w.Code)

	// 真落库（重启后仍生效的根据）
	var row models.Config
	require.NoError(t, c.GetDB().
		Where("config_key = ?", "sys.rpa.autoscale.config").
		First(&row).Error)
	assert.NotEmpty(t, row.ConfigValue)

	// 读回
	ctx2, w2 := newTestCtxWorker("GET", "/autoscale/config", nil)
	h.GetAutoScaleConfig(ctx2)
	require.Equal(t, http.StatusOK, w2.Code)

	got, err := h.loadAutoScaleConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want.Enabled, got.Enabled)
	assert.Equal(t, want.ScaleUpThreshold, got.ScaleUpThreshold)
	assert.Equal(t, want.ScaleDownThreshold, got.ScaleDownThreshold)
	assert.Equal(t, want.MinConcurrency, got.MinConcurrency)
	assert.Equal(t, want.MaxConcurrency, got.MaxConcurrency)
	assert.Equal(t, want.CheckInterval, got.CheckInterval)

	// 二次写入（读-改-写幂等，不新增行）
	want.MaxConcurrency = 30
	ctx3, w3 := newTestCtxWorker("POST", "/autoscale/config", want)
	h.UpdateAutoScaleConfig(ctx3)
	require.Equal(t, http.StatusOK, w3.Code)

	var count int64
	require.NoError(t, c.GetDB().Model(&models.Config{}).
		Where("config_key = ?", "sys.rpa.autoscale.config").Count(&count).Error)
	assert.Equal(t, int64(1), count, "重复写入不得产生多行")

	got2, err := h.loadAutoScaleConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 30, got2.MaxConcurrency)
}
