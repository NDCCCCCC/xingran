package migrations

// Migration 259 数据侧证据：站点品牌 brand.* seed 幂等性与默认值。
//
// 覆盖：
//  1. 空库首次执行 → 6 个 brand.* 键全部落库，值 = 预期默认值；
//  2. 幂等：重复执行不新增行、不覆盖管理员已改的值；
//  3. IsSystem=1（防参数管理页误删）、ConfigType=N（可编辑）；
//  4. 不触碰非 brand.* 的既有配置行。
//
// 库用 AutoMigrate(&models.Config{}) 建真实表（不用手写 CREATE TABLE）。

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// freshSQLiteDBForMigrate259 建仅含 sys_config 的内存 sqlite 库。
func freshSQLiteDBForMigrate259(t *testing.T, name string) *gorm.DB {
	t.Helper()
	safe := strings.NewReplacer("/", "_", " ", "_", "=", "_").Replace(name)
	dsn := "file:" + safe + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Config{}))
	return db
}

// expectedBrand259 迁移应播种的 6 键与默认值
var expectedBrand259 = map[string]string{
	"brand.site_name":     "数字化平台",
	"brand.site_mark":     "数",
	"brand.logo_url":      "",
	"brand.login_tagline": "以国密算法为基座,融合自动化与 AI 调度,让每一次资源调度都可观测、可审计、可信赖。",
	"brand.copyright":     "© 2026 数字化平台",
	"brand.site_subtitle": "请使用您的账号登录以访问运维控制台",
}

func TestMigrate259_SeedsAllBrandKeys(t *testing.T) {
	db := freshSQLiteDBForMigrate259(t, "mig259_seed")
	require.NoError(t, Migrate259BrandConfigSeed(db))

	for key, want := range expectedBrand259 {
		var got models.Config
		require.NoError(t, db.Where("config_key = ?", key).First(&got).Error, "缺少 seed 键 %s", key)
		assert.Equal(t, want, got.ConfigValue, "键 %s 默认值不符", key)
		assert.Equal(t, models.ConfigIsSystemYes, got.IsSystem, "品牌配置必须 IsSystem=1 防误删")
		assert.Equal(t, models.ConfigTypeNo, got.ConfigType, "品牌配置必须 ConfigType=N 可编辑")
	}

	// 不增其它 brand 行
	var total int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key LIKE 'brand.%'").Count(&total).Error)
	assert.EqualValues(t, len(expectedBrand259), total)
}

func TestMigrate259_IdempotentAndPreservesEdits(t *testing.T) {
	db := freshSQLiteDBForMigrate259(t, "mig259_idem")
	require.NoError(t, Migrate259BrandConfigSeed(db))

	// 管理员改值
	require.NoError(t, db.Model(&models.Config{}).
		Where("config_key = ?", "brand.site_name").
		Update("config_value", "管理员改过的名字").Error)

	// 再跑一次：不得覆盖、不得新增
	require.NoError(t, Migrate259BrandConfigSeed(db))

	var got models.Config
	require.NoError(t, db.Where("config_key = ?", "brand.site_name").First(&got).Error)
	assert.Equal(t, "管理员改过的名字", got.ConfigValue, "重跑迁移不得覆盖管理员已改的值")

	var total int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key LIKE 'brand.%'").Count(&total).Error)
	assert.EqualValues(t, len(expectedBrand259), total, "重跑不得新增行")
}

func TestMigrate259_DoesNotTouchOtherConfigs(t *testing.T) {
	db := freshSQLiteDBForMigrate259(t, "mig259_other")
	require.NoError(t, db.Create(&models.Config{
		ConfigName:  "加密开关",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "true",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    models.ConfigIsSystemYes,
	}).Error)

	require.NoError(t, Migrate259BrandConfigSeed(db))

	var got models.Config
	require.NoError(t, db.Where("config_key = ?", "sys.request.encryption.enabled").First(&got).Error)
	assert.Equal(t, "true", got.ConfigValue, "迁移不得改动非 brand.* 配置")
}
