package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// -------------------------------------------------------------------------
// 79-05 Task 1..3: mac_history_query_service.go(全包 #1 缺口文件,389 stmts @8.0%)
//
// 装配口径:
//   - 裸装配 newMhq7905: sqlite(t.TempDir 文件库) + cache:nil(复刻既有 setupTestService
//     的 cache:nil 口径,但改用文件库避免 file::memory:?cache=shared 跨测试串库)。
//   - 缓存装配 newMhq7905Cached: 同库 + cache.NewMemoryCache(1000, 5min) 同时注入
//     cache 字段(GetVendor 路径)与 dataCache 字段(query cache-aside 路径)+
//     NewCacheConfigService(db) 作为 perfConfig 字段,把 perfCacheTTL / GetOrSet 装饰
//     分支一并打开(79-01 MemoryCache 装配约定;t.Cleanup 单次 Close)。
//
// 既有 mac_history_query_service_test.go 的 setupTestService / TestGetVendor 等勿动,
// 本文件 helper 全部带 7905 后缀(R5 撞名纪律)。
// -------------------------------------------------------------------------

// newMhq7905 裸装配(cache=nil)。AutoMigrate MACOUIVendor + MAC 历史主表 + sys_config。
func newMhq7905(t *testing.T) (*macHistoryQueryServiceImpl, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mhq7905.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.MACOUIVendor{},
		&models.DeviceMACHistory{},
		&models.Config{},
	), "auto migrate mac query models")
	return &macHistoryQueryServiceImpl{db: db, cache: nil}, db
}

// newMhq7905Cached 缓存装配:MemoryCache + DataCacheService + CacheConfigService。
func newMhq7905Cached(t *testing.T) (*macHistoryQueryServiceImpl, *cache.MemoryCache, *gorm.DB) {
	t.Helper()
	svc, db := newMhq7905(t)
	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { mem.Close() }) // 单次 Close(QUIRK-P1 已幂等,仍守纪律)
	svc.cache = mem
	svc.dataCache = NewDataCacheService(mem)
	svc.perfConfig = NewCacheConfigService(db)
	return svc, mem, db
}

// mhq7905Time 固定时刻构造(禁 time.Now 相对断言,防跨日 flake)。
func mhq7905Time(day, hour, minute int) time.Time {
	return time.Date(2026, 3, day, hour, minute, 0, 0, time.UTC)
}

// mhq7905History 构造一条 MAC 历史(未落库)。MAC/接口名传 canonical 形态,
// 避开 BeforeCreate 归一化差异(model hook:大写冒号 MAC + 大写短名接口)。
func mhq7905History(deviceID, mac, iface string, vlan *int, eventType models.MACEventType, first, last time.Time) *models.DeviceMACHistory {
	return &models.DeviceMACHistory{
		DeviceID:           deviceID,
		DeviceNameSnapshot: "dev-7905",
		MACAddress:         mac,
		InterfaceName:      iface,
		VLANID:             vlan,
		EventType:          eventType,
		FirstSeen:          first,
		LastSeen:           last,
		CollectedAt:        last,
	}
}

// mhq7905Int 便捷 *int。
func mhq7905Int(v int) *int { return &v }

// -------------------------------------------------------------------------
// TestMhq7905_PerfCacheTTL perfCacheTTL(:155-165)
//
// QUIRK-79-05-A(锁定,零生产改动): MACPerfConfigCacheTTLSeconds =
// "network.mac.perf.cache_ttl_seconds" 不命中 CacheConfigService.LoadConfigs 的
// 「cache.%」/「rate_limit.%」两个 LIKE 通道(cache_config_service.go:144/:151),
// 因此只要 perfConfig 非 nil,GetDuration 永远走「键不存在 → 30 分钟」兜底;
// 键名里的 seconds 语义(秒)与 GetDuration 的分钟语义互相矛盾,但生产装配读不到该键,
// 矛盾不可达。测试锁定现行为:nil → 5 分钟;非 nil → 30 分钟(与种子值无关)。
// -------------------------------------------------------------------------

func TestMhq7905_PerfCacheTTL(t *testing.T) {
	svc, db := newMhq7905(t)

	// perfConfig == nil → 默认 5 分钟
	assert.Equal(t, 5*time.Minute, svc.perfCacheTTL(), "perfConfig nil 应返回 5 分钟兜底")

	// 种入目标键(含非法值形态)→ 仍不受影响(键不在 LoadConfigs 的 LIKE 通道内)
	require.NoError(t, db.Create(&models.Config{
		ConfigName:  "MAC查询缓存TTL(秒)",
		ConfigKey:   MACPerfConfigCacheTTLSeconds,
		ConfigValue: "120",
		ConfigType:  models.ConfigTypeYes,
	}).Error)
	require.NoError(t, db.Create(&models.Config{
		ConfigName:  "MAC热力图TopN",
		ConfigKey:   MACPerfConfigHeatmapTopN,
		ConfigValue: "not-a-number",
		ConfigType:  models.ConfigTypeYes,
	}).Error)

	svc2, _, db2 := newMhq7905Cached(t)
	_ = db2
	assert.Equal(t, 30*time.Minute, svc2.perfCacheTTL(),
		"perfConfig 非 nil 时 GetDuration 对未加载键回 30 分钟兜底(QUIRK-79-05-A)")
}

// TestMhq7905_NewConstructors 两个构造器(:145-152)装配与接口断言。
func TestMhq7905_NewConstructors(t *testing.T) {
	_, db := newMhq7905(t)

	var iface MACHistoryQueryService = NewMACHistoryQueryService(db)
	impl, ok := iface.(*macHistoryQueryServiceImpl)
	require.True(t, ok, "NewMACHistoryQueryService 应返回私有实现")
	assert.Nil(t, impl.cache, "裸构造 cache 为 nil")
	assert.Nil(t, impl.dataCache, "裸构造 dataCache 为 nil")
	assert.Nil(t, impl.perfConfig, "裸构造 perfConfig 为 nil")
	assert.NotNil(t, impl.db)

	var iface2 MACHistoryQueryService = NewMACHistoryQueryServiceWithCache(db, NewDataCacheService(cache.NewMemoryCache(10, time.Minute)), nil)
	impl2, ok := iface2.(*macHistoryQueryServiceImpl)
	require.True(t, ok, "NewMACHistoryQueryServiceWithCache 应返回私有实现")
	assert.NotNil(t, impl2.dataCache)
	assert.Nil(t, impl2.perfConfig)

	// nil-db 构造本身不 panic(查询路径的 nil-db 行为不属本断言范围,复刻生产装配前提)
	assert.NotPanics(t, func() { _ = NewMACHistoryQueryService(nil) })
}

// TestMhq7905_GetVendor_CacheAndDB GetVendor(:242-285)缓存/DB 双路径。
// 缓存命中的可观察证据:删除 OUI 行后二次调用仍返回同 vendor,且 MemoryCache 里
// 能直查到 "mac:vendor:<OUI>" 键。
func TestMhq7905_GetVendor_CacheAndDB(t *testing.T) {
	ctx := context.Background()
	svc, mem, db := newMhq7905Cached(t)

	seed := &models.MACOUIVendor{OUIPrefix: "AABBCC", VendorName: "Cache Vendor 7905"}
	require.NoError(t, db.Create(seed).Error)

	// 1) 首调走 DB
	vendor, err := svc.GetVendor(ctx, "AA:BB:CC:DD:EE:FF")
	require.NoError(t, err)
	assert.Equal(t, "Cache Vendor 7905", vendor)

	// 缓存键形态直查
	cached, err := mem.Get(ctx, "mac:vendor:AABBCC")
	require.NoError(t, err, "GetVendor 命中后应写缓存键 mac:vendor:<OUI>")
	assert.Equal(t, "Cache Vendor 7905", cached)

	// 2) 删行后二次调用仍命中(可观察的缓存证据)
	require.NoError(t, db.Delete(seed).Error)
	var rowCount int64
	require.NoError(t, db.Model(&models.MACOUIVendor{}).Count(&rowCount).Error)
	assert.Zero(t, rowCount, "前置:OUI 行已删")
	vendor2, err := svc.GetVendor(ctx, "aabbccddee ff")
	require.NoError(t, err)
	assert.Equal(t, "Cache Vendor 7905", vendor2, "删行后仍应命中缓存")

	// 3) 未知 OUI → "Unknown Vendor" 且缓存(24h)
	vendor3, err := svc.GetVendor(ctx, "DD:EE:FF:11:22:33")
	require.NoError(t, err)
	assert.Equal(t, "Unknown Vendor", vendor3)
	cachedUnknown, err := mem.Get(ctx, "mac:vendor:DDEEFF")
	require.NoError(t, err)
	assert.Equal(t, "Unknown Vendor", cachedUnknown, "未知 OUI 也应缓存避免重复查库")

	// 4) 短 MAC(<6 位规范化后)→ "Unknown Vendor",不查库不缓存
	vendor4, err := svc.GetVendor(ctx, "AA:BB")
	require.NoError(t, err)
	assert.Equal(t, "Unknown Vendor", vendor4)
	_, cacheErr := mem.Get(ctx, "mac:vendor:AABB")
	assert.Error(t, cacheErr, "短 MAC 不应产生缓存键")

	// 5) 裸装配(cache=nil)同语义(走 DB 分支)
	bare, _ := newMhq7905(t)
	require.NoError(t, bare.db.Create(&models.MACOUIVendor{OUIPrefix: "112233", VendorName: "Bare Vendor 7905"}).Error)
	vendor5, err := bare.GetVendor(ctx, "11-22-33-44-55-66")
	require.NoError(t, err)
	assert.Equal(t, "Bare Vendor 7905", vendor5, "cache=nil 时应直接走 DB")
}

// TestMhq7905_ImportOUIData ImportOUIData(:202-239)。
// 数据源是相对路径 configs/oui-vendors.json(os.ReadFile 相对测试进程 cwd),
// 用 t.TempDir + t.Chdir 造源文件,不写仓库目录(威胁模型:禁污染 configs/)。
func TestMhq7905_ImportOUIData(t *testing.T) {
	ctx := context.Background()

	t.Run("happy_and_idempotent", func(t *testing.T) {
		svc, db := newMhq7905(t)
		tmp := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "configs"), 0o755))
		payload := `[
			{"oui_prefix":"AAB790","vendor_name":"Vendor7905A"},
			{"oui_prefix":"AAB791","vendor_name":"Vendor7905B"},
			{"oui_prefix":"AAB792","vendor_name":"Vendor7905C"}
		]`
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "configs", "oui-vendors.json"), []byte(payload), 0o644))
		t.Chdir(tmp)

		require.NoError(t, svc.ImportOUIData(ctx), "首次导入应成功")
		var count int64
		require.NoError(t, db.Table("sys_mac_oui_vendor").Count(&count).Error)
		assert.Equal(t, int64(3), count, "应插入 3 行")

		// 幂等:表已有数据 → 直接跳过,不重复插入
		require.NoError(t, svc.ImportOUIData(ctx))
		require.NoError(t, db.Table("sys_mac_oui_vendor").Count(&count).Error)
		assert.Equal(t, int64(3), count, "已有数据时二次导入应跳过")

		var vendorName string
		require.NoError(t, db.Table("sys_mac_oui_vendor").
			Where("oui_prefix = ?", "AAB791").Select("vendor_name").Row().Scan(&vendorName))
		assert.Equal(t, "Vendor7905B", vendorName)
	})

	t.Run("missing_file", func(t *testing.T) {
		svc, _ := newMhq7905(t)
		t.Chdir(t.TempDir()) // 无 configs/oui-vendors.json
		err := svc.ImportOUIData(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read OUI JSON")
	})

	t.Run("bad_json", func(t *testing.T) {
		svc, _ := newMhq7905(t)
		tmp := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "configs"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "configs", "oui-vendors.json"), []byte("not-json"), 0o644))
		t.Chdir(tmp)
		err := svc.ImportOUIData(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse OUI JSON")
	})
}

// mhq7905DeviceUUID 稳定 UUID(device_id 列是 type:uuid,查询入口校验 uuid.Parse)。
func mhq7905DeviceUUID(t *testing.T) string {
	t.Helper()
	return uuid.New().String()
}
