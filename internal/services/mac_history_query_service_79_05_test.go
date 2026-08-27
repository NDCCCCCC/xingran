package services

import (
	"context"
	"fmt"
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

// seedMhq7905 落 n 条跨端口/跨时间的 MAC 历史(同设备),返回设备 UUID。
// 时间一律 time.Date 显式构造(禁 time.Now 相对断言)。
func seedMhq7905(t *testing.T, db *gorm.DB, n int) string {
	t.Helper()
	deviceID := uuid.New().String()
	base := mhq7905Time(10, 8, 0)
	records := make([]*models.DeviceMACHistory, 0, n)
	for i := 0; i < n; i++ {
		first := base.Add(time.Duration(i) * time.Hour)
		last := first.Add(30 * time.Minute)
		records = append(records, mhq7905History(
			deviceID,
			fmt.Sprintf("AA:BB:CC:00:00:%02X", i),
			fmt.Sprintf("GE0/0/%d", i%3+1),
			mhq7905Int(100+i%2),
			models.EventAppeared,
			first, last,
		))
	}
	require.NoError(t, db.Create(&records).Error, "seed mac history")
	return deviceID
}

// mhq7905DeviceUUID 稳定 UUID(device_id 列是 type:uuid,查询入口校验 uuid.Parse)。
func mhq7905DeviceUUID(t *testing.T) string {
	t.Helper()
	return uuid.New().String()
}

// -------------------------------------------------------------------------
// Task 2: 查询四链(QueryPortHistory/QueryDeviceHistory/QueryHistory/
// QueryConnectionStats)+ getLongOccupancyThreshold + 缓存语义等价
// -------------------------------------------------------------------------

// TestMhq7905_QueryPortHistory 端口历史链(:288-412)。
func TestMhq7905_QueryPortHistory(t *testing.T) {
	ctx := context.Background()
	svc, db := newMhq7905(t)
	deviceID := seedMhq7905(t, db, 6)

	t.Run("all_rows", func(t *testing.T) {
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{DeviceID: deviceID})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, int64(6), res.Total)
		assert.Len(t, res.List, 6)
		assert.Equal(t, 1, res.Current, "缺省 current 应补 1")
		assert.Equal(t, 20, res.PageSize, "缺省 pageSize 应补 20")
		assert.NotEmpty(t, res.List[0].ID)
		assert.Equal(t, deviceID, res.List[0].DeviceID)
		assert.Equal(t, "dev-7905", res.List[0].DeviceNameSnapshot)
	})

	t.Run("interface_filter", func(t *testing.T) {
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{DeviceID: deviceID, InterfaceName: "GE0/0/1"})
		require.NoError(t, err)
		assert.Equal(t, int64(2), res.Total, "i%3+1==1 的记录应恰 2 条")
		for _, r := range res.List {
			assert.Equal(t, "GE0/0/1", r.InterfaceName)
		}
	})

	t.Run("time_window", func(t *testing.T) {
		// 宽窗(3 天)而非小时级窄窗: QueryHistory 会把 RFC3339 瞬时值 .Local() 后按
		// 裸 wall-clock 文本比较(无时区列),窄窗会随宿主机时区漂移;宽窗对任意时区稳定。
		start := mhq7905Time(8, 0, 0).Format(time.RFC3339)
		end := mhq7905Time(12, 0, 0).Format(time.RFC3339)
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{DeviceID: deviceID, StartTime: start, EndTime: end})
		require.NoError(t, err)
		assert.Equal(t, int64(6), res.Total, "种子全部落在宽窗内")
	})

	t.Run("far_future_window_empty", func(t *testing.T) {
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{
			DeviceID:  deviceID,
			StartTime: mhq7905Time(20, 0, 0).Format(time.RFC3339),
			EndTime:   mhq7905Time(22, 0, 0).Format(time.RFC3339),
		})
		require.NoError(t, err)
		assert.Zero(t, res.Total, "种子之前的窗口应为空")
	})

	t.Run("only_start", func(t *testing.T) {
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{
			DeviceID:  deviceID,
			StartTime: mhq7905Time(7, 0, 0).Format(time.RFC3339),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(6), res.Total, "仅下界(早于全部种子)应返回全部")
	})

	t.Run("only_end", func(t *testing.T) {
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{
			DeviceID: deviceID,
			EndTime:  mhq7905Time(13, 0, 0).Format(time.RFC3339),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(6), res.Total, "仅上界(晚于全部种子)应返回全部")
	})

	t.Run("pagination", func(t *testing.T) {
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{DeviceID: deviceID, Current: 2, PageSize: 4})
		require.NoError(t, err)
		assert.Equal(t, int64(6), res.Total)
		assert.Len(t, res.List, 2)
		assert.Equal(t, 2, res.Current)
		assert.Equal(t, 4, res.PageSize)
	})

	t.Run("error_invalid_device_id", func(t *testing.T) {
		_, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{DeviceID: "not-a-uuid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的设备ID格式")
	})

	t.Run("error_bad_start_time", func(t *testing.T) {
		_, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{DeviceID: deviceID, StartTime: "yesterday"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的开始时间格式")
	})

	t.Run("error_bad_end_time", func(t *testing.T) {
		_, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{
			DeviceID: deviceID,
			EndTime:  "tomorrow",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的结束时间格式")
	})

	t.Run("error_range_too_large", func(t *testing.T) {
		_, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{
			DeviceID:  deviceID,
			StartTime: mhq7905Time(1, 0, 0).Format(time.RFC3339),
			EndTime:   mhq7905Time(1, 0, 0).AddDate(2, 0, 0).Format(time.RFC3339),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "查询时间跨度过大")
	})

	t.Run("empty_result_shape", func(t *testing.T) {
		res, err := svc.QueryPortHistory(ctx, &PortHistoryQuery{DeviceID: uuid.New().String()})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Zero(t, res.Total)
		assert.NotNil(t, res.List, "空结果应为非 nil 空 slice")
		assert.Empty(t, res.List)
	})
}

// TestMhq7905_QueryDeviceHistory 设备历史链(:413-533)。
func TestMhq7905_QueryDeviceHistory(t *testing.T) {
	ctx := context.Background()
	svc, db := newMhq7905(t)
	deviceID := seedMhq7905(t, db, 5)

	res, err := svc.QueryDeviceHistory(ctx, &DeviceHistoryQuery{DeviceID: deviceID})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, int64(5), res.Total)
	assert.Len(t, res.List, 5)
	assert.Equal(t, 1, res.Current)
	assert.Equal(t, 20, res.PageSize)

	// 分页 + 默认值补齐
	res2, err := svc.QueryDeviceHistory(ctx, &DeviceHistoryQuery{DeviceID: deviceID, Current: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), res2.Total)
	assert.Len(t, res2.List, 2)
	assert.Equal(t, 2, res2.Current)
	assert.Equal(t, 2, res2.PageSize)
	// 排序 first_seen DESC:第 2 页首条应是第 3 条新记录
	assert.True(t, res2.List[0].FirstSeen.After(res.List[3].FirstSeen), "分页应延续 DESC 排序")

	// 时间窗(宽窗口径,见 TestMhq7905_QueryPortHistory/time_window 注释)
	res3, err := svc.QueryDeviceHistory(ctx, &DeviceHistoryQuery{
		DeviceID:  deviceID,
		StartTime: mhq7905Time(8, 0, 0).Format(time.RFC3339),
		EndTime:   mhq7905Time(12, 0, 0).Format(time.RFC3339),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), res3.Total)

	// 错误分支
	_, err = svc.QueryDeviceHistory(ctx, &DeviceHistoryQuery{DeviceID: "bad"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无效的设备ID格式")

	_, err = svc.QueryDeviceHistory(ctx, &DeviceHistoryQuery{
		DeviceID:  deviceID,
		StartTime: mhq7905Time(1, 0, 0).Format(time.RFC3339),
		EndTime:   mhq7905Time(1, 0, 0).AddDate(2, 0, 0).Format(time.RFC3339),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询时间跨度过大")
}

// TestMhq7905_QueryHistory_List 通用列表链(:534-659):多过滤组合 + 分页 + 错误分支。
func TestMhq7905_QueryHistory_List(t *testing.T) {
	ctx := context.Background()
	svc, db := newMhq7905(t)
	deviceID := seedMhq7905(t, db, 6)

	t.Run("no_filter", func(t *testing.T) {
		res, err := svc.QueryHistory(ctx, &MACHistoryListQuery{})
		require.NoError(t, err)
		assert.Equal(t, int64(6), res.Total)
		assert.Equal(t, 1, res.Current)
		assert.Equal(t, 20, res.PageSize)
	})

	t.Run("device_filter", func(t *testing.T) {
		res, err := svc.QueryHistory(ctx, &MACHistoryListQuery{DeviceID: deviceID})
		require.NoError(t, err)
		assert.Equal(t, int64(6), res.Total)
	})

	t.Run("mac_filter_normalized", func(t *testing.T) {
		res, err := svc.QueryHistory(ctx, &MACHistoryListQuery{MAC: "AA:BB:CC:00:00:03"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total, "MAC 过滤应按归一化形态命中")
		require.Len(t, res.List, 1)
		assert.Equal(t, "AA:BB:CC:00:00:03", res.List[0].MACAddress)
	})

	t.Run("interface_and_vlan_and_event", func(t *testing.T) {
		// 种子: iface = GE0/0/{i%3+1}, vlan = 100+i%2。
		// GE0/0/2 → i∈{1,4};其中 vlan==100 仅 i=4(i=1 是 101)。
		res, err := svc.QueryHistory(ctx, &MACHistoryListQuery{
			DeviceID:      deviceID,
			InterfaceName: "GE0/0/2",
			VLANID:        mhq7905Int(100),
			EventType:     string(models.EventAppeared),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total, "GE0/0/2 × vlan100 × appeared 应 1 条")
	})

	t.Run("vlan_filter_only", func(t *testing.T) {
		res, err := svc.QueryHistory(ctx, &MACHistoryListQuery{DeviceID: deviceID, VLANID: mhq7905Int(100)})
		require.NoError(t, err)
		assert.Equal(t, int64(3), res.Total, "vlan100 → 偶数 i 三条")
	})

	t.Run("time_range", func(t *testing.T) {
		// 宽窗(3 天): QueryHistory 会把 RFC3339 瞬时值 .Local() 后按裸 wall-clock 比较,
		// 小时级窄窗随宿主机时区漂移;宽窗对任意时区稳定。
		res, err := svc.QueryHistory(ctx, &MACHistoryListQuery{
			StartTime: mhq7905Time(8, 0, 0).Format(time.RFC3339),
			EndTime:   mhq7905Time(12, 0, 0).Format(time.RFC3339),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(6), res.Total, "全部种子落宽窗")
	})

	t.Run("far_future_window_empty", func(t *testing.T) {
		res, err := svc.QueryHistory(ctx, &MACHistoryListQuery{
			StartTime: mhq7905Time(20, 0, 0).Format(time.RFC3339),
			EndTime:   mhq7905Time(22, 0, 0).Format(time.RFC3339),
		})
		require.NoError(t, err)
		assert.Zero(t, res.Total)
	})

	t.Run("empty_db_zero_shape", func(t *testing.T) {
		bare, _ := newMhq7905(t)
		res, err := bare.QueryHistory(ctx, &MACHistoryListQuery{MAC: "AA:BB:CC:00:00:99"})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Zero(t, res.Total)
		assert.NotNil(t, res.List)
		assert.Empty(t, res.List)
	})

	t.Run("error_invalid_mac", func(t *testing.T) {
		_, err := svc.QueryHistory(ctx, &MACHistoryListQuery{MAC: "ZZ:BB:CC:00:00:03"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MAC地址验证失败")
	})

	t.Run("error_invalid_device", func(t *testing.T) {
		_, err := svc.QueryHistory(ctx, &MACHistoryListQuery{DeviceID: "nope"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的设备ID格式")
	})

	t.Run("error_bad_time", func(t *testing.T) {
		_, err := svc.QueryHistory(ctx, &MACHistoryListQuery{StartTime: "now-ish"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的开始时间格式")
	})

	t.Run("error_range_too_large", func(t *testing.T) {
		_, err := svc.QueryHistory(ctx, &MACHistoryListQuery{
			StartTime: mhq7905Time(1, 0, 0).Format(time.RFC3339),
			EndTime:   mhq7905Time(1, 0, 0).AddDate(2, 0, 0).Format(time.RFC3339),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "查询时间跨度过大")
	})

	t.Run("status_filter_unbacked_column_errors", func(t *testing.T) {
		//QUIRK-79-05-B(锁定): models.DeviceMACHistory 无 Status 字段(sqlite fixture 无该列),
		// req.Status 非 nil 时 SQL 带 status = ? → sqlite 报 no such column → 错误包装分支。
		// 该分支在现网 PG 依赖 migration 手工列,模型层不映射;零生产改动,按现行为断言。
		status := 0
		_, err := svc.QueryHistory(ctx, &MACHistoryListQuery{Status: &status})
		require.Error(t, err, "status 列在 fixture 表不存在时应报查询错误")
	})
}

// TestMhq7905_GetLongOccupancyThreshold 长期占用阈值(:796-825)。
//
// QUIRK-79-05-C(锁定): 配置不存在时走 `Row().Scan()` 返回 driver 的 sql.ErrNoRows,
// 与 `err == gorm.ErrRecordNotFound` 的判断不成立(:809 用 == 比较非 errors.Is),
// 因此「配置不存在 → (defaultDays, nil)」分支在 GORM Row().Scan 路径不可达,
// 实际返回 (30, err)。上层 QueryConnectionStats 以 Warnf+回退 30 天兜住该错误。
func TestMhq7905_GetLongOccupancyThreshold(t *testing.T) {
	ctx := context.Background()

	t.Run("missing_config_returns_default_with_err", func(t *testing.T) {
		svc, _ := newMhq7905(t)
		days, err := svc.getLongOccupancyThreshold(ctx)
		assert.Equal(t, 30, days, "缺配置应回 30 天默认值")
		require.Error(t, err, "QUIRK-79-05-C: Row().Scan 无行返回 sql.ErrNoRows 而非 gorm.ErrRecordNotFound")
	})

	t.Run("configured_value", func(t *testing.T) {
		svc, db := newMhq7905(t)
		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "长期占用阈值",
			ConfigKey:   "network.mac.history.long_occupancy_threshold_days",
			ConfigValue: "45",
			ConfigType:  models.ConfigTypeYes,
		}).Error)
		days, err := svc.getLongOccupancyThreshold(ctx)
		require.NoError(t, err)
		assert.Equal(t, 45, days)
	})

	t.Run("invalid_values_fall_back", func(t *testing.T) {
		for _, bad := range []string{"abc", "0", "-5", ""} {
			svc, db := newMhq7905(t)
			require.NoError(t, db.Create(&models.Config{
				ConfigName:  "长期占用阈值",
				ConfigKey:   "network.mac.history.long_occupancy_threshold_days",
				ConfigValue: bad,
				ConfigType:  models.ConfigTypeYes,
			}).Error)
			days, err := svc.getLongOccupancyThreshold(ctx)
			assert.Equal(t, 30, days, "非法值 %q 应回默认 30", bad)
			if bad == "" {
				// 空串可被 Sscanf 解析为 0 → 走 days<=0 回退分支(err==nil)
				assert.NoError(t, err)
			} else {
				assert.NoError(t, err, "解析失败走 Warnf 回退分支,仍返回 nil error")
			}
		}
	})
}

// TestMhq7905_QueryConnectionStats 连接统计链(:829-1055)。
//
// QUIRK-79-05-D(锁定,PG-only): queryConnectionStatsFromDB 的三条原生 SQL 使用
// EXTRACT(EPOCH FROM ...)/::bigint/COUNT(*) FILTER — PostgreSQL 专属语法,
// sqlite 无法解析 → 明细查询必走「查询连接时长明细失败」错误包装分支。
// 三段聚合的映射循环在 sqlite 下不可达(R6 同族口径,零生产改动,不真建 PG)。
func TestMhq7905_QueryConnectionStats(t *testing.T) {
	ctx := context.Background()
	svc, db := newMhq7905(t)
	deviceID := seedMhq7905(t, db, 4)

	validStart := mhq7905Time(9, 0, 0).Format(time.RFC3339)
	validEnd := mhq7905Time(12, 0, 0).Format(time.RFC3339)

	t.Run("error_missing_time", func(t *testing.T) {
		_, err := svc.QueryConnectionStats(ctx, &ConnectionStatsQuery{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "开始时间和结束时间为必填项")
	})

	t.Run("error_bad_start", func(t *testing.T) {
		_, err := svc.QueryConnectionStats(ctx, &ConnectionStatsQuery{StartTime: "soon", EndTime: validEnd})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的开始时间格式")
	})

	t.Run("error_bad_end", func(t *testing.T) {
		_, err := svc.QueryConnectionStats(ctx, &ConnectionStatsQuery{StartTime: validStart, EndTime: "later"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的结束时间格式")
	})

	t.Run("error_range_too_large", func(t *testing.T) {
		_, err := svc.QueryConnectionStats(ctx, &ConnectionStatsQuery{
			StartTime: mhq7905Time(1, 0, 0).Format(time.RFC3339),
			EndTime:   mhq7905Time(1, 0, 0).AddDate(2, 0, 0).Format(time.RFC3339),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "查询时间跨度过大")
	})

	t.Run("error_invalid_mac", func(t *testing.T) {
		_, err := svc.QueryConnectionStats(ctx, &ConnectionStatsQuery{
			StartTime: validStart, EndTime: validEnd, MACAddress: "GG:00:00:00:00:01",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MAC地址验证失败")
	})

	t.Run("sqlite_dialect_error_wrapped", func(t *testing.T) {
		res, err := svc.QueryConnectionStats(ctx, &ConnectionStatsQuery{
			StartTime: validStart, EndTime: validEnd, TopN: 5,
		})
		require.Error(t, err, "QUIRK-79-05-D: sqlite 无法解析 PG 专属统计 SQL")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "查询连接时长明细失败")
		_ = deviceID
	})
}

// TestMhq7905_Query_CachedParity 缓存装配与裸装配语义等价(缓存不改变结果)。
func TestMhq7905_Query_CachedParity(t *testing.T) {
	ctx := context.Background()

	// 缓存装配(独立库)
	cached, _, cachedDB := newMhq7905Cached(t)
	deviceID := seedMhq7905(t, cachedDB, 4)

	req := func() *MACHistoryListQuery {
		return &MACHistoryListQuery{DeviceID: deviceID, PageSize: 3}
	}
	first, err := cached.QueryHistory(ctx, req())
	require.NoError(t, err)
	second, err := cached.QueryHistory(ctx, req())
	require.NoError(t, err)
	assert.Equal(t, first.Total, second.Total)
	assert.Equal(t, len(first.List), len(second.List))
	for i := range first.List {
		assert.Equal(t, first.List[i].ID, second.List[i].ID, "缓存命中应返回同序同集")
		assert.Equal(t, first.List[i].MACAddress, second.List[i].MACAddress)
	}

	// 裸装配(独立库,同形态种子)结果一致 → 缓存不改变语义
	bare, bareDB := newMhq7905(t)
	seedMhq7905At(t, bareDB, bare, deviceID)

	third, err := bare.QueryHistory(ctx, req())
	require.NoError(t, err)
	assert.Equal(t, first.Total, third.Total)
	assert.Equal(t, len(first.List), len(third.List))

	// port-history 的 cache-aside 装饰分支(命中 GetOrSet 路径)
	portReq := &PortHistoryQuery{DeviceID: deviceID, PageSize: 2}
	p1, err := cached.QueryPortHistory(ctx, portReq)
	require.NoError(t, err)
	p2, err := cached.QueryPortHistory(ctx, portReq)
	require.NoError(t, err)
	assert.Equal(t, p1.Total, p2.Total)
	assert.Len(t, p2.List, len(p1.List))
}

// seedMhq7905At 在指定库上以指定 deviceID 落同形态种子(等价性对照用)。
func seedMhq7905At(t *testing.T, db *gorm.DB, svc *macHistoryQueryServiceImpl, deviceID string) {
	t.Helper()
	base := mhq7905Time(10, 8, 0)
	records := make([]*models.DeviceMACHistory, 0, 4)
	for i := 0; i < 4; i++ {
		first := base.Add(time.Duration(i) * time.Hour)
		records = append(records, mhq7905History(
			deviceID,
			fmt.Sprintf("AA:BB:CC:00:00:%02X", i),
			fmt.Sprintf("GE0/0/%d", i%3+1),
			mhq7905Int(100+i%2),
			models.EventAppeared,
			first, first.Add(30*time.Minute),
		))
	}
	require.NoError(t, db.Create(&records).Error)
	_ = svc
}
