package services

// =====================================================================
// Phase 79-01 Task 2: CacheConfigService 配置路径测试(sqlite sys_config 驱动)
//
// 覆盖目标: cache_config_service.go 67.7% → ≥75%(基线 124 stmts / 40 unc,79-RESEARCH §2)。
//
// 关键纪律:
//   - helper newCcs7901 带 plan 后缀(R5);既有 setupCacheConfigTestDB 与 5 个
//     rate-limit 测试不动(D-79-06:只补未覆盖分支,不改既有断言)。
//   - fixture 用 t.TempDir() 文件库(research §4:防 file::memory:?cache=shared 串数据)。
//   - 默认值断言一律引用源码常量与 GetConfigInfo() 元数据,禁裸魔法数字。
//   - 禁 t.Parallel()。
// =====================================================================

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ccs7901SysConfigDDL 与既有 setupCacheConfigTestDB 同一形态(Phase 59 D-03 模式),
// 常量本地化以免同包重名(R5)。
const ccs7901SysConfigDDL = `
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
`

// newCcs7901 装配 CacheConfigService + sqlite(t.TempDir 文件库)+ 种子行。
// seed 在 NewCacheConfigService(构造器内 LoadConfigs)之前写入,构造即生效。
func newCcs7901(t *testing.T, seed map[string]string) (*CacheConfigService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ccs7901.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.Exec(ccs7901SysConfigDDL).Error)

	id := 0
	for k, v := range seed {
		id++
		require.NoError(t, db.Exec(
			`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system) VALUES (?, ?, ?, ?, 'Y', 1)`,
			"ccs7901-"+k, "ccs7901 seed "+k, k, v,
		).Error)
	}

	return NewCacheConfigService(db), db
}

// ccs7901ConfigValue 读回 DB 中某配置键的当前值(断言自动修复用)。
func ccs7901ConfigValue(t *testing.T, db *gorm.DB, key string) string {
	t.Helper()
	var cfg models.Config
	require.NoError(t, db.Where("config_key = ?", key).First(&cfg).Error)
	return cfg.ConfigValue
}

// TestCcs7901_LoadConfigs_EmptyBackfillsDefaults 空表 → LoadConfigs →
// setDefaultsIfNeeded 回填默认值(GetAllConfigs 非空 + 已知默认键非零值 + DB 落行)。
func TestCcs7901_LoadConfigs_EmptyBackfillsDefaults(t *testing.T) {
	ctx := context.Background()
	svc, db := newCcs7901(t, nil)

	// 内存 map 已回填:抽 2 个已知默认键(:266 默认清单)
	assert.Equal(t, 30*time.Minute, svc.GetDuration(CacheConfigDeptTree),
		"cache.dept.tree 默认 30 分钟")
	assert.Equal(t, 60*time.Minute, svc.GetDuration(CacheConfigDictType),
		"cache.dict.type 默认 60 分钟")

	// GetAllConfigs 非空且值与 GetDuration 同源
	all := svc.GetAllConfigs(ctx)
	assert.NotEmpty(t, all, "空表加载后 GetAllConfigs 应被默认值回填")
	assert.Equal(t, int(30), all[CacheConfigDeptTree])
	assert.Equal(t, int(60), all[CacheConfigDictType])

	// 默认值同时写库:cache.* 与 12 条 rate_limit.* 都落行
	var cacheCount, rateCount int64
	require.NoError(t, db.Model(&models.Config{}).Where("config_key LIKE ?", "cache.%").Count(&cacheCount).Error)
	require.NoError(t, db.Model(&models.Config{}).Where("config_key LIKE ?", "rate_limit.%").Count(&rateCount).Error)
	assert.Greater(t, cacheCount, int64(0), "cache.* 默认配置应写库")
	assert.Equal(t, int64(12), rateCount, "rate_limit.* 应写库 12 条(D-16)")

	// 限流默认与 GetRateLimit 同源(:340 rateLimitDefaults 清单)
	assert.Equal(t, 30, svc.GetRateLimit(RateLimitReadPerMinute, 999))
	assert.Equal(t, 20000, svc.GetRateLimit(RateLimitDefaultPerDay, 999))
}

// TestCcs7901_LoadConfigs_SeedOverridesDefault 预置 duration 键覆盖默认值:
// cache.dept.tree = 120 → GetDuration 返回 120 分钟(单位换算按 :229 minutes * time.Minute)。
func TestCcs7901_LoadConfigs_SeedOverridesDefault(t *testing.T) {
	svc, _ := newCcs7901(t, map[string]string{
		CacheConfigDeptTree: "120",
	})

	assert.Equal(t, 120*time.Minute, svc.GetDuration(CacheConfigDeptTree),
		"种子值必须覆盖默认值(120 分钟)")
	// 未种子的键仍走默认回填
	assert.Equal(t, 10*time.Minute, svc.GetDuration(CacheConfigUserList),
		"cache.user.list 默认 10 分钟不受种子影响")
}

// TestCcs7901_Duration_InvalidAndUnknown 非数字值 → 回退 GetConfigInfo 默认值并修库;
// 未知键 → GetDuration 内部默认 30 分钟 / GetDurationWithDefault 返回显式 default(:392 分支)。
func TestCcs7901_Duration_InvalidAndUnknown(t *testing.T) {
	svc, db := newCcs7901(t, map[string]string{
		CacheConfigDeptTree: "not-a-number",
	})

	// 解析失败 → info.Default=30(GetConfigInfo 元数据)回填内存并修复 DB
	deptInfo := svc.GetConfigInfo()[CacheConfigDeptTree]
	require.Equal(t, 30, deptInfo.Default, "源码元数据 cache.dept.tree 默认应为 30 分钟")
	assert.Equal(t, time.Duration(deptInfo.Default)*time.Minute, svc.GetDuration(CacheConfigDeptTree),
		"非数字值应回退 GetConfigInfo 默认值")
	assert.Equal(t, "30", ccs7901ConfigValue(t, db, CacheConfigDeptTree),
		"非数字值应在 DB 中被自动修复为默认值")

	// 未知键:GetDuration 内部默认 30 分钟(:388)
	assert.Equal(t, 30*time.Minute, svc.GetDuration("cache.unknown.probe"))

	// 未知键:GetDurationWithDefault 返回显式 default(:392 分支)
	assert.Equal(t, 7*time.Minute, svc.GetDurationWithDefault("cache.unknown.probe", 7*time.Minute))
}

// TestCcs7901_GetRateLimit_ClampTail 既有 TestCacheConfigService_RangeValidation 只盖
// 「高于上限」(read.per_minute=99999);此处补「低于下限」与另一键「高于上限」双尾支。
func TestCcs7901_GetRateLimit_ClampTail(t *testing.T) {
	svc, db := newCcs7901(t, map[string]string{
		RateLimitWritePerDay: "0",         // 低于 Min=1
		RateLimitAdminPerDay: "999999999", // 高于 Max=1000000
	})

	writeInfo := svc.GetConfigInfo()[RateLimitWritePerDay]
	adminInfo := svc.GetConfigInfo()[RateLimitAdminPerDay]
	require.Equal(t, 1, writeInfo.Min, "源码元数据 write.per_day 下限应为 1 次")
	require.Equal(t, 1000000, adminInfo.Max, "源码元数据 admin.per_day 上限应为 1000000 次")

	assert.Equal(t, writeInfo.Default, svc.GetRateLimit(RateLimitWritePerDay, 999),
		"低于下限的值应被重置为默认值")
	assert.Equal(t, adminInfo.Default, svc.GetRateLimit(RateLimitAdminPerDay, 999),
		"高于上限的值应被重置为默认值")

	// DB 自动修复(与既有 RangeValidation 同口径,不同键)
	assert.Equal(t, "15000", ccs7901ConfigValue(t, db, RateLimitWritePerDay))
	assert.Equal(t, "50000", ccs7901ConfigValue(t, db, RateLimitAdminPerDay))
}

// TestCcs7901_GetConfigInfo_NamesRemarks GetConfigInfo 元数据 + getConfigName /
// getConfigRemark 的已知键、rate_limit 单位分支与未知键回退。
func TestCcs7901_GetConfigInfo_NamesRemarks(t *testing.T) {
	svc, _ := newCcs7901(t, nil)

	info := svc.GetConfigInfo()
	assert.NotEmpty(t, info, "GetConfigInfo 应返回完整元数据 map")
	assert.GreaterOrEqual(t, len(info), 40, "cache.* 35 键 + rate_limit.* 12 键 元数据齐备")

	// 已知键:名称/说明非空,说明含默认值与范围
	assert.Equal(t, "部门树缓存时间", svc.getConfigName(CacheConfigDeptTree))
	deptRemark := svc.getConfigRemark(CacheConfigDeptTree)
	assert.Contains(t, deptRemark, "默认30分钟")
	assert.Contains(t, deptRemark, "范围5-120分钟")

	// rate_limit 分支:值语义为「次」非「分钟」(:909-911;Description 里的「每分钟」
	// 是频率描述,断言落在单位后缀上)
	assert.Equal(t, "读作用域每分钟限流", svc.getConfigName(RateLimitReadPerMinute))
	readRemark := svc.getConfigRemark(RateLimitReadPerMinute)
	assert.Contains(t, readRemark, "默认30次")
	assert.Contains(t, readRemark, "范围1-10000次")
	assert.NotContains(t, readRemark, "分钟，范围",
		"rate_limit.* 说明的取值/范围单位必须是「次」,不得是「分钟」")

	// 未知键:getConfigName 回退键本身;getConfigRemark 回退「缓存配置」
	assert.Equal(t, "cache.unknown.probe", svc.getConfigName("cache.unknown.probe"))
	assert.Equal(t, "缓存配置", svc.getConfigRemark("cache.unknown.probe"))
}

// TestCcs7901_ReloadConfig_PicksUpDBChange 测试中直接 UPDATE sys_config 行 →
// ReloadConfig 后 GetRateLimit 反映新值(区分 LoadConfigs 冷读与 Reload 热读两分支)。
func TestCcs7901_ReloadConfig_PicksUpDBChange(t *testing.T) {
	ctx := context.Background()
	svc, db := newCcs7901(t, nil)

	baseline := svc.GetRateLimit(RateLimitWritePerMinute, 999)
	require.Equal(t, 100, baseline, "构造器冷读应得默认值 100 次/分钟")

	// 直接改库,不经过 service
	require.NoError(t, db.Model(&models.Config{}).
		Where("config_key = ?", RateLimitWritePerMinute).
		Update("config_value", "77").Error)

	// 热载前仍是旧值(内存 map 未刷新)
	assert.Equal(t, 100, svc.GetRateLimit(RateLimitWritePerMinute, 999),
		"reload 前不应读到库中新值")

	// ReloadConfig 热读 → 新值生效
	require.NoError(t, svc.ReloadConfig(ctx))
	assert.Equal(t, 77, svc.GetRateLimit(RateLimitWritePerMinute, 999),
		"reload 后新阈值应对新请求生效")
}

// TestCcs7901_GetExpiration_Wired ccs 注入 DataCacheService(SetCacheConfig)→
// GetExpiration 返回配置值;未配置键 → default(收口 Task 1 的注入分支)。
func TestCcs7901_GetExpiration_Wired(t *testing.T) {
	ccs, _ := newCcs7901(t, nil)
	dcs, _ := newDcs7901(t)

	assert.Nil(t, dcs.cacheConfig, "注入前 cacheConfig 应为 nil")
	dcs.SetCacheConfig(ccs)
	assert.NotNil(t, dcs.cacheConfig, "SetCacheConfig 注入后 cacheConfig 应非 nil")

	// 已配置键:返回配置值(cache.dept.tree = 30 分钟)
	assert.Equal(t, 30*time.Minute, dcs.GetExpiration(CacheConfigDeptTree, 5*time.Minute),
		"注入后 GetExpiration 应走配置值而非 default")

	// 未配置键:GetDurationWithDefault 返回显式 default
	assert.Equal(t, 5*time.Minute, dcs.GetExpiration("cache.unknown.probe", 5*time.Minute),
		"未配置键应返回调用方 default")
}
