package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// -------------------------------------------------------------------------
// 79-05 Task 5: mac 尾部五文件
//   mac_history_service.go(209/131 covered)+ mac_history_partition.go(95/0)
//   + mac_history_matview_service.go(30/0)+ mac_history_heatmap_service.go(52/0)
//   + mac_perf_config_seed.go(11/0)
//
// R6 口径: PG-only 路径(partition DDL/matview REFRESH/heatmap MV 查询)一律
// 『字符串形态断言 + sqlite 跳过分支』,禁真建分区/物化视图。字符串断言靠
// pgFakeDialector7905(仅 Dialector.Name()=="postgres" + 捕获 SQL 的失败 ConnPool,
// 零真实 PG 交互)—— 它让 dialect 守卫放行到 DDL 构造行,再用 Exec 失败收尾,
// 既覆盖 DDL 字符串构造,又断言 SQL 形态,且绝不落任何真实 PG。
//
// 既有 mac_history_merge_transitions_test.go / mac_history_purge_test.go 勿动(只补 unc)。
// -------------------------------------------------------------------------

// newMhs7905 同一 sqlite 库装配四 service(heatmap 走 79-01 MemoryCache + CacheConfigService)。
func newMhs7905(t *testing.T) (*gorm.DB, MACHistoryService, PartitionService, MACHistoryMatViewService, MACHistoryHeatmapService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(strings.ReplaceAll(t.TempDir(), `\`, "/")+"/mhs7905.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.DeviceMACAddress{},
		&models.DeviceMACHistory{},
		&models.NetworkDevice{},
		&models.Config{},
	), "auto migrate mac tail models")

	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { mem.Close() })
	heatmap := NewMACHistoryHeatmapService(db, NewDataCacheService(mem), NewCacheConfigService(db))

	return db,
		NewMACHistoryService(db),
		NewPartitionService(db),
		NewMACHistoryMatViewService(db),
		heatmap
}

// -------------------------------------------------------------------------
// fake PG dialector(R6 口径的字符串断言基建)
// -------------------------------------------------------------------------

// newMhs7905PGFake 构造一个「方言 postgres、Exec/Query 必失败并捕获 SQL」的 gorm.DB。
// 用途: 放行 isPostgres/isPostgreSQL 守卫 → 覆盖 PG-only DDL/查询字符串构造分支,
// 并断言 SQL 形态(禁真建分区/物化视图,Exec 直接失败)。
func newMhs7905PGFake(t *testing.T) (*gorm.DB, *[]string) {
	t.Helper()
	captured := &[]string{}
	db, err := gorm.Open(mhs7905PGDialector{captured: captured}, &gorm.Config{
		DisableAutomaticPing: true,
	})
	require.NoError(t, err, "open fake pg dialector db")
	return db, captured
}

// mhs7905PGDialector 仅满足 gorm.Dialector 接口的假方言。
type mhs7905PGDialector struct {
	captured *[]string
}

func (d mhs7905PGDialector) Name() string { return "postgres" }

func (d mhs7905PGDialector) Initialize(db *gorm.DB) error {
	db.ConnPool = mhs7905PGPool{captured: d.captured}
	// gorm 的默认 callbacks(create/query/update/delete/row/raw)由各真实 dialector 的
	// Initialize 注册;假方言必须自补,否则 Exec/Scan 会在空 fns 上静默返回。
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}

func (d mhs7905PGDialector) Migrator(*gorm.DB) gorm.Migrator { return nil }

func (d mhs7905PGDialector) DataTypeOf(*schema.Field) string { return "text" }

func (d mhs7905PGDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "NULL"}
}

func (d mhs7905PGDialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ any) {
	writer.WriteByte('?')
}

func (d mhs7905PGDialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('"')
	for _, r := range str {
		if r == '"' {
			writer.WriteString(`""`)
			continue
		}
		_, _ = writer.WriteString(string(r))
	}
	writer.WriteByte('"')
}

func (d mhs7905PGDialector) Explain(sqlStr string, vars ...any) string {
	return gormlogger.ExplainSQL(sqlStr, nil, `"`, vars...)
}

// mhs7905PGPool 失败型 ConnPool:Exec/Query 一律报错,但先把 SQL 文本捕获下来供断言。
type mhs7905PGPool struct {
	captured *[]string
}

func (p mhs7905PGPool) ExecContext(ctx context.Context, query string, _ ...any) (sql.Result, error) {
	*p.captured = append(*p.captured, query)
	return nil, fmt.Errorf("mhs7905 fake pg: exec disabled (R6: 禁真执行 PG DDL)")
}

func (p mhs7905PGPool) QueryContext(_ context.Context, query string, _ ...any) (*sql.Rows, error) {
	*p.captured = append(*p.captured, query)
	return nil, fmt.Errorf("mhs7905 fake pg: query disabled")
}

// QueryRowContext 返回 nil:本 fixture 只用于 Raw().Scan(slice) 路径(QueryContext),
// 任何走 Row().Scan() 的调用方会 panic —— 勿在该 fake 上调用 Row() 形态 API。
func (p mhs7905PGPool) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

func (p mhs7905PGPool) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, fmt.Errorf("mhs7905 fake pg: prepare disabled")
}

// -------------------------------------------------------------------------
// mac_history_service.go 纯函数 + 变更记录
// -------------------------------------------------------------------------

// TestMhs7905_BuildMACStateMap_AndVlanEqual BuildMACStateMap(:100-124)与 vlanEqual(:271-279)。
func TestMhs7905_BuildMACStateMap_AndVlanEqual(t *testing.T) {
	_, svc, _, _, _ := newMhs7905(t)

	t.Run("vlan_equal_table", func(t *testing.T) {
		v100 := 100
		v200 := 200
		assert.True(t, vlanEqual(nil, nil))
		assert.False(t, vlanEqual(&v100, nil))
		assert.False(t, vlanEqual(nil, &v100))
		assert.True(t, vlanEqual(&v100, &v100))
		assert.False(t, vlanEqual(&v100, &v200))
	})

	t.Run("state_map_keys_and_normalization", func(t *testing.T) {
		v := 100
		list := []models.DeviceMACAddress{
			{MACAddress: "aabbccddeeff", InterfaceName: "gigabitethernet0/0/1", VLANID: &v},
			{MACAddress: "AA:BB:CC:DD:EE:F1", InterfaceName: "GE0/0/2", VLANID: nil},
		}
		state := svc.BuildMACStateMap(list)
		require.Len(t, state, 2, "两条不同键应各占一槽")
		for key, mac := range state {
			assert.Equal(t, key.MACAddress, mac.MACAddress, "BuildMACStateMap 应把归一化值写回条目")
			assert.Equal(t, key.InterfaceName, mac.InterfaceName)
			// 归一化口径: MAC 大写冒号 + 接口大写短名
			assert.Regexp(t, `^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`, key.MACAddress)
			assert.Equal(t, strings.ToUpper(key.InterfaceName), key.InterfaceName)
		}
	})

	t.Run("state_map_overwrite_same_key", func(t *testing.T) {
		// 同一 MAC/接口的两种拼写 → 归一化后同键,后写覆盖前写
		a := models.DeviceMACAddress{MACAddress: "AA:BB:CC:00:00:01", InterfaceName: "GE0/0/1"}
		b := models.DeviceMACAddress{MACAddress: "aabbcc000001", InterfaceName: "ge0/0/1"}
		state := svc.BuildMACStateMap([]models.DeviceMACAddress{a, b})
		assert.Len(t, state, 1, "归一化同键应合并")
		for _, mac := range state {
			assert.Equal(t, "AA:BB:CC:00:00:01", mac.MACAddress, "应保留后写条目(含归一化)")
		}
	})

	t.Run("empty_list", func(t *testing.T) {
		state := svc.BuildMACStateMap(nil)
		assert.NotNil(t, state)
		assert.Empty(t, state)
	})
}

// TestMhs7905_RecordMACChange RecordMACChange(:138-268)四事件形态 + 无变化不落行。
func TestMhs7905_RecordMACChange(t *testing.T) {
	ctx := context.Background()
	db, svc, _, _, _ := newMhs7905(t)

	device := &models.NetworkDevice{DeviceName: "dev-mhs-7905"}
	require.NoError(t, db.Create(device).Error)

	mkState := func(mac, iface string, vlan *int) map[MACEvent]*models.DeviceMACAddress {
		entry := &models.DeviceMACAddress{MACAddress: mac, InterfaceName: iface, VLANID: vlan}
		return map[MACEvent]*models.DeviceMACAddress{
			{MACAddress: mac, InterfaceName: iface, VLANID: vlan}: entry,
		}
	}

	t.Run("appeared_disappeared", func(t *testing.T) {
		oldState := mkState("AA:BB:CC:00:00:01", "GE0/0/1", mhq7905Int(100))
		newState := mkState("AA:BB:CC:00:00:02", "GE0/0/2", mhq7905Int(100))
		require.NoError(t, svc.RecordMACChange(ctx, device, oldState, newState))

		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ?", device.ID).Find(&rows).Error)
		require.Len(t, rows, 2, "1 appeared + 1 disappeared")
		byType := map[string]string{}
		for _, r := range rows {
			byType[string(r.EventType)] = r.MACAddress
			assert.Equal(t, "dev-mhs-7905", r.DeviceNameSnapshot)
		}
		assert.Equal(t, "AA:BB:CC:00:00:02", byType[string(models.EventAppeared)])
		assert.Equal(t, "AA:BB:CC:00:00:01", byType[string(models.EventDisappeared)])
	})

	t.Run("moved", func(t *testing.T) {
		oldState := mkState("AA:BB:CC:00:00:10", "GE0/0/1", mhq7905Int(100))
		newState := mkState("AA:BB:CC:00:00:10", "GE0/0/9", mhq7905Int(100))
		require.NoError(t, svc.RecordMACChange(ctx, device, oldState, newState))

		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ? AND event_type = ?", device.ID, models.EventMoved).Find(&rows).Error)
		require.Len(t, rows, 1)
		assert.Equal(t, "GE0/0/9", rows[0].InterfaceName, "moved 记录新接口")
		assert.Equal(t, mhq7905Int(100), rows[0].VLANID)
	})

	t.Run("vlan_changed", func(t *testing.T) {
		oldState := mkState("AA:BB:CC:00:00:20", "GE0/0/1", mhq7905Int(100))
		newState := mkState("AA:BB:CC:00:00:20", "GE0/0/1", mhq7905Int(300))
		require.NoError(t, svc.RecordMACChange(ctx, device, oldState, newState))

		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ? AND event_type = ?", device.ID, models.EventVLANChanged).Find(&rows).Error)
		require.Len(t, rows, 1)
		assert.Equal(t, 300, *rows[0].VLANID, "vlan_changed 记录新 VLAN")
	})

	t.Run("no_change_no_rows", func(t *testing.T) {
		var before int64
		require.NoError(t, db.Model(&models.DeviceMACHistory{}).Where("device_id = ?", device.ID).Count(&before).Error)
		state := mkState("AA:BB:CC:00:00:30", "GE0/0/1", mhq7905Int(100))
		require.NoError(t, svc.RecordMACChange(ctx, device, state, state))

		var after int64
		require.NoError(t, db.Model(&models.DeviceMACHistory{}).Where("device_id = ?", device.ID).Count(&after).Error)
		assert.Equal(t, before, after, "新旧一致不应落行")
	})

	t.Run("first_seen_device_initializes", func(t *testing.T) {
		brandNew := &models.NetworkDevice{
			DeviceName: "dev-mhs-new-7905",
			DeviceType: models.DeviceTypeSwitch,
			Vendor:     models.VendorHuawei,
			IPAddress:  fmt.Sprintf("10.179.%d.%d", mcl7905DeviceSeq/250+5, mcl7905DeviceSeq%250+11),
		}
		mcl7905DeviceSeq++
		require.NoError(t, db.Create(brandNew).Error)
		newState := mkState("AA:BB:CC:00:00:40", "GE0/0/1", nil)
		require.NoError(t, svc.RecordMACChange(ctx, brandNew, map[MACEvent]*models.DeviceMACAddress{}, newState))

		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ?", brandNew.ID).Find(&rows).Error)
		require.Len(t, rows, 1, "首见设备应记 appeared")
		assert.Equal(t, string(models.EventAppeared), string(rows[0].EventType))
	})
}

// -------------------------------------------------------------------------
// mac_history_service.go flapping/转换合并 + 清理
// -------------------------------------------------------------------------

// mhs7905SeedHistoryRow 落一条历史记录(显式时间)。
func mhs7905SeedHistoryRow(t *testing.T, db *gorm.DB, deviceID, mac, iface string, vlan *int, eventType models.MACEventType, first, last time.Time) *models.DeviceMACHistory {
	t.Helper()
	row := &models.DeviceMACHistory{
		DeviceID:           deviceID,
		DeviceNameSnapshot: "dev-mhs-7905",
		MACAddress:         mac,
		InterfaceName:      iface,
		VLANID:             vlan,
		EventType:          eventType,
		FirstSeen:          first,
		LastSeen:           last,
		CollectedAt:        last,
	}
	require.NoError(t, db.Create(row).Error)
	return row
}

// TestMhs7905_MergeFlapping MergeFlappingRecords(:306-424)+ CleanupAllDevicesFlapping(:438-470)。
func TestMhs7905_MergeFlapping(t *testing.T) {
	ctx := context.Background()
	db, svc, _, _, _ := newMhs7905(t)
	deviceID := "bbbbbbbb-1111-1111-1111-111111111111"

	// 同 (MAC,接口,VLAN) 内 3 条抖动:间隔 10 分钟 ≤ FlappingMergeWindow(2h) → 合并成 1 条。
	// VLANID 一律传 nil,原因见 QUIRK-79-05-H。
	base := mhq7905Time(10, 8, 0)
	mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:01:00", "GE0/0/1", nil, models.EventAppeared, base, base.Add(5*time.Minute))
	mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:01:00", "GE0/0/1", nil, models.EventDisappeared, base.Add(15*time.Minute), base.Add(20*time.Minute))
	mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:01:00", "GE0/0/1", nil, models.EventAppeared, base.Add(30*time.Minute), base.Add(35*time.Minute))

	t.Run("merges_within_window", func(t *testing.T) {
		require.NoError(t, svc.MergeFlappingRecords(ctx, deviceID))
		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ?", deviceID).Find(&rows).Error)
		require.Len(t, rows, 1, "3 条抖动应合并为 1 条")
		// 时间断言按 wall-clock 文本比较:DeviceMACHistory.AfterFind 会把读回的
		// timestamp 无时区值重塑为 Local(loc 变、墙钟不变),与 UTC 构造的种子
		// 直接比 instant 会差时区偏移。
		assert.Equal(t, base.Add(35*time.Minute).Format("2006-01-02 15:04:05"),
			rows[0].LastSeen.Format("2006-01-02 15:04:05"), "保留条 last_seen 应延伸到末条(墙钟)")
		assert.Equal(t, string(models.EventAppeared), string(rows[0].EventType), "保留首条事件类型")
	})

	t.Run("no_flapping_is_noop", func(t *testing.T) {
		var before int64
		require.NoError(t, db.Model(&models.DeviceMACHistory{}).Where("device_id = ?", deviceID).Count(&before).Error)
		require.NoError(t, svc.MergeFlappingRecords(ctx, deviceID))
		var after int64
		require.NoError(t, db.Model(&models.DeviceMACHistory{}).Where("device_id = ?", deviceID).Count(&after).Error)
		assert.Equal(t, before, after)
	})

	t.Run("cleanup_all_devices", func(t *testing.T) {
		// 第二台设备单条记录(不构成 flapping)→ 汇总成功设备数 2
		other := "bbbbbbbb-2222-2222-2222-222222222222"
		mhs7905SeedHistoryRow(t, db, other, "AA:BB:CC:00:02:00", "GE0/0/1", nil, models.EventAppeared, base, base)
		count, err := svc.CleanupAllDevicesFlapping(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "两台有历史记录的设备均应被处理")
	})

	t.Run("out_of_window_not_merged", func(t *testing.T) {
		gapDevice := "bbbbbbbb-3333-3333-3333-333333333333"
		mhs7905SeedHistoryRow(t, db, gapDevice, "AA:BB:CC:00:03:00", "GE0/0/1", nil, models.EventAppeared, base, base)
		mhs7905SeedHistoryRow(t, db, gapDevice, "AA:BB:CC:00:03:00", "GE0/0/1", nil, models.EventAppeared, base.Add(3*time.Hour), base.Add(3*time.Hour))
		require.NoError(t, svc.MergeFlappingRecords(ctx, gapDevice))
		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ?", gapDevice).Find(&rows).Error)
		assert.Len(t, rows, 2, "超出 2h 窗口不应合并")
	})

	t.Run("non_nil_vlan_never_groups", func(t *testing.T) {
		//QUIRK-79-05-H(锁定,零生产改动): MergeFlappingRecords 的组键用
		// fmt.Sprintf("%v", hist.VLANID) — VLANID 是 *int,%v 打印的是指针地址。
		// 每行读回都是独立指针 → 非 nil VLAN 的行永远各成一组,合并永不触发。
		// 生产上 vlan_changed/moved 之后同 VLAN 的 flapping 不会被该算法折叠。
		// 修复属生产改动(应解引用),先立项;此处按现行为断言证据化。
		vlanDevice := "bbbbbbbb-4444-4444-4444-444444444444"
		mhs7905SeedHistoryRow(t, db, vlanDevice, "AA:BB:CC:00:07:00", "GE0/0/1", mhq7905Int(100), models.EventAppeared, base, base)
		mhs7905SeedHistoryRow(t, db, vlanDevice, "AA:BB:CC:00:07:00", "GE0/0/1", mhq7905Int(100), models.EventDisappeared, base.Add(10*time.Minute), base.Add(10*time.Minute))
		require.NoError(t, svc.MergeFlappingRecords(ctx, vlanDevice))
		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ?", vlanDevice).Find(&rows).Error)
		assert.Len(t, rows, 2, "非 nil VLAN 指针 → 各成一组 → 不合并(QUIRK-79-05-H)")
	})
}

// TestMhs7905_MergeByTransitions MergeByTransitions(:496-518)+ mergeTransitionsForDevice(:523-587)。
func TestMhs7905_MergeByTransitions(t *testing.T) {
	ctx := context.Background()
	db, svc, _, _, _ := newMhs7905(t)

	t.Run("keeps_position_transitions_only", func(t *testing.T) {
		deviceID := "cccccccc-1111-1111-1111-111111111111"
		base := mhq7905Time(10, 8, 0)
		// 同位置纯 flapping(应删): appeared → disappeared → appeared 同接口同 VLAN
		mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:04:00", "GE0/0/1", mhq7905Int(100), models.EventAppeared, base, base)
		mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:04:00", "GE0/0/1", mhq7905Int(100), models.EventDisappeared, base.Add(10*time.Minute), base.Add(10*time.Minute))
		mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:04:00", "GE0/0/1", mhq7905Int(100), models.EventAppeared, base.Add(20*time.Minute), base.Add(20*time.Minute))
		// 真实转换(应保留): moved 到新接口
		mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:04:00", "GE0/0/9", mhq7905Int(100), models.EventMoved, base.Add(30*time.Minute), base.Add(30*time.Minute))
		// 另一 MAC 单条(应保留)
		mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:04:99", "GE0/0/1", mhq7905Int(100), models.EventAppeared, base, base)

		deleted, err := svc.MergeByTransitions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted, "同位置 2 条 flapping 应删除,首条 + moved + 另一 MAC 保留")

		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ?", deviceID).Find(&rows).Error)
		require.Len(t, rows, 3)
	})

	t.Run("empty_db_returns_zero", func(t *testing.T) {
		empty, _, _, _, _ := newMhs7905(t)
		emptySvc := NewMACHistoryService(empty)
		deleted, err := emptySvc.MergeByTransitions(ctx)
		require.NoError(t, err)
		assert.Zero(t, deleted, "空库应返回 0")
	})

	t.Run("vlan_changed_deleted_by_design", func(t *testing.T) {
		deviceID := "cccccccc-2222-2222-2222-222222222222"
		base := mhq7905Time(10, 8, 0)
		mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:05:00", "GE0/0/1", mhq7905Int(100), models.EventAppeared, base, base)
		mhs7905SeedHistoryRow(t, db, deviceID, "AA:BB:CC:00:05:00", "GE0/0/1", mhq7905Int(200), models.EventVLANChanged, base.Add(10*time.Minute), base.Add(10*time.Minute))

		deleted, err := svc.MergeByTransitions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted, "严格口径: vlan_changed 不算位置变化 → 删除")

		var rows []models.DeviceMACHistory
		require.NoError(t, db.Where("device_id = ?", deviceID).Find(&rows).Error)
		require.Len(t, rows, 1)
	})
}

// TestMhs7905_PurgeMeaninglessRecords_NothingToDelete 补既有 purge 测试未覆盖的
// 「非 dry-run + 无可删行 → 提前返回」分支(:673-676)(既有 purge 测试勿动)。
func TestMhs7905_PurgeMeaninglessRecords_NothingToDelete(t *testing.T) {
	ctx := context.Background()
	db, svc, _, _, _ := newMhs7905(t)

	// 只有 moved 事件 → 无可删行
	base := mhq7905Time(10, 8, 0)
	mhs7905SeedHistoryRow(t, db, "dddddddd-1111-1111-1111-111111111111", "AA:BB:CC:00:06:00", "GE0/0/1", nil, models.EventMoved, base, base)

	deleted, backupTable, err := svc.PurgeMeaninglessRecords(ctx, false)
	require.NoError(t, err)
	assert.Zero(t, deleted, "无可删行应返回 0")
	assert.Empty(t, backupTable, "无可删行不应建备份表")

	// 确认没有残留备份表
	var tableCount int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'sys_device_mac_history_purge_backup%'").Scan(&tableCount).Error)
	assert.Zero(t, tableCount)
}

// -------------------------------------------------------------------------
// mac_history_partition.go
// -------------------------------------------------------------------------

// TestMhp7905_Partition_SqliteSkip sqlite 分支(:59-62/:110-114/:177-180)零 DDL 副作用。
func TestMhp7905_Partition_SqliteSkip(t *testing.T) {
	ctx := context.Background()
	db, _, partition, _, _ := newMhs7905(t)

	impl, ok := partition.(*partitionServiceImpl)
	require.True(t, ok)
	assert.False(t, impl.isPostgres(), "sqlite 方言 isPostgres 应为 false")

	require.NoError(t, partition.CreateMonthlyPartition(ctx, 2026, 3), "sqlite 应直接跳过创建分区")
	require.NoError(t, partition.EnsurePartitionsExist(ctx, 2), "sqlite 应跳过分区检查")
	require.NoError(t, partition.DropExpiredPartitions(ctx), "sqlite 应跳过分区清理")

	// 零 DDL 副作用: sqlite_master 不应出现任何分区表
	var tableCount int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'sys_device_mac_history_%'").Scan(&tableCount).Error)
	assert.Zero(t, tableCount, "sqlite 下不应产生任何分区表")
}

// TestMhp7905_GetRetentionDays GetRetentionDays(:149-171)默认/覆盖/下限钳制/非法回退。
func TestMhp7905_GetRetentionDays(t *testing.T) {
	ctx := context.Background()

	t.Run("default_when_missing", func(t *testing.T) {
		_, _, partition, _, _ := newMhs7905(t)
		assert.Equal(t, 120, partition.GetRetentionDays(ctx), "无配置应回默认 120 天")
	})

	cases := []struct {
		name  string
		value string
		want  int
	}{
		{"configured_45", "45", 45},
		{"below_min_clamped_to_default", "10", 120},
		{"non_numeric_falls_back", "abc", 120},
		{"zero_falls_back", "0", 120},
		{"negative_falls_back", "-5", 120},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, _, partition, _, _ := newMhs7905(t)
			require.NoError(t, db.Create(&models.Config{
				ConfigName:  "MAC历史保留期",
				ConfigKey:   "network.mac.history.retention_days",
				ConfigValue: tc.value,
				ConfigType:  models.ConfigTypeYes,
			}).Error)
			assert.Equal(t, tc.want, partition.GetRetentionDays(ctx))
		})
	}
}

// TestMhp7905_Partition_PG_Fake PG-only 段(R6 口径:假方言放行构造 + SQL 形态断言 + Exec 必失败)。
func TestMhp7905_Partition_PG_Fake(t *testing.T) {
	ctx := context.Background()
	fakeDB, captured := newMhs7905PGFake(t)
	partition := NewPartitionService(fakeDB)

	impl, ok := partition.(*partitionServiceImpl)
	require.True(t, ok)
	assert.True(t, impl.isPostgres(), "假方言 isPostgres 应为 true")

	t.Run("invalid_year", func(t *testing.T) {
		err := partition.CreateMonthlyPartition(ctx, 2019, 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的年份")
		err = partition.CreateMonthlyPartition(ctx, 2101, 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的年份")
	})

	t.Run("invalid_month", func(t *testing.T) {
		err := partition.CreateMonthlyPartition(ctx, 2026, 13)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的月份")
		err = partition.CreateMonthlyPartition(ctx, 2026, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无效的月份")
	})

	t.Run("ddl_string_shape_and_exec_guarded", func(t *testing.T) {
		*captured = nil
		err := partition.CreateMonthlyPartition(ctx, 2026, 3)
		t.Logf("DEBUG captured=%v err=%v", *captured, err)
		require.Error(t, err, "R6: 假方言 Exec 必失败(禁真建分区)")
		assert.Contains(t, err.Error(), "创建分区 sys_device_mac_history_2026_03 失败")
		require.Len(t, *captured, 1, "应恰好构造并尝试执行一条 DDL")
		ddl := (*captured)[0]
		assert.Contains(t, ddl, "CREATE TABLE IF NOT EXISTS sys_device_mac_history_2026_03")
		assert.Contains(t, ddl, "PARTITION OF sys_device_mac_history")
		assert.Contains(t, ddl, "FOR VALUES FROM ('2026-03-01') TO ('2026-04-01')")

		// 年末跨年边界(2026-12 → 2027-01)
		*captured = nil
		_ = partition.CreateMonthlyPartition(ctx, 2026, 12)
		require.Len(t, *captured, 1)
		assert.Contains(t, (*captured)[0], "FOR VALUES FROM ('2026-12-01') TO ('2027-01-01')")
	})

	t.Run("ensure_partitions_aggregates_errors", func(t *testing.T) {
		*captured = nil
		// monthsAhead=0 → 默认 2 → 循环 3 个月,全部失败(3 > 2)→ 整体报错
		err := partition.EnsurePartitionsExist(ctx, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "创建所有MAC历史分区失败")
		assert.Len(t, *captured, 3, "应尝试创建 3 个月度分区")
	})

	t.Run("drop_expired_partition_query_fails_wrapped", func(t *testing.T) {
		err := partition.DropExpiredPartitions(ctx)
		require.Error(t, err, "pg_inherits 系统目录在假方言下查询必失败")
		assert.Contains(t, err.Error(), "查询分区列表失败")
	})
}

// -------------------------------------------------------------------------
// mac_history_matview_service.go
// -------------------------------------------------------------------------

// TestMhm7905_MatView_SqliteSkip sqlite 分支(:56-59/:75-78)直接跳过。
func TestMhm7905_MatView_SqliteSkip(t *testing.T) {
	ctx := context.Background()
	_, _, _, matview, _ := newMhs7905(t)

	impl, ok := matview.(*macHistoryMatViewServiceImpl)
	require.True(t, ok)
	assert.False(t, impl.isPostgreSQL(), "sqlite 方言 isPostgreSQL 应为 false")

	require.NoError(t, matview.RefreshSingleMatView(ctx, "mv_mac_port_latest"))
	require.NoError(t, matview.RefreshAllMaterializedViews(ctx))
}

// TestMhm7905_MatView_PG_Fake PG-only 段(白名单 + REFRESH 字符串形态 + 部分失败容错 D-11)。
func TestMhm7905_MatView_PG_Fake(t *testing.T) {
	ctx := context.Background()
	fakeDB, captured := newMhs7905PGFake(t)
	matview := NewMACHistoryMatViewService(fakeDB)

	impl, ok := matview.(*macHistoryMatViewServiceImpl)
	require.True(t, ok)
	assert.True(t, impl.isPostgreSQL())

	t.Run("whitelist_rejects_unknown", func(t *testing.T) {
		err := matview.RefreshSingleMatView(ctx, "mv_not_in_whitelist")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "不在白名单中")
	})

	t.Run("refresh_single_sql_shape", func(t *testing.T) {
		*captured = nil
		err := matview.RefreshSingleMatView(ctx, "mv_mac_port_latest")
		require.Error(t, err, "R6: 假方言 Exec 必失败(禁真 REFRESH)")
		assert.Contains(t, err.Error(), "刷新物化视图 mv_mac_port_latest 失败")
		require.Len(t, *captured, 1)
		assert.Equal(t, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_mac_port_latest", (*captured)[0],
			"D-10 锁定: 必须带 CONCURRENTLY")
	})

	t.Run("refresh_all_partial_failure_returns_first_err", func(t *testing.T) {
		*captured = nil
		err := matview.RefreshAllMaterializedViews(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "物化视图刷新部分失败")
		require.Len(t, *captured, len(matViewRefreshOrder), "4 个 MV 均应被尝试(D-11: 单个失败不阻断)")
		// 顺序硬编码 MV-01 → MV-04
		assert.Contains(t, (*captured)[0], "mv_mac_port_latest")
		assert.Contains(t, (*captured)[1], "mv_mac_device_summary")
		assert.Contains(t, (*captured)[2], "mv_mac_long_occupancy_top")
		assert.Contains(t, (*captured)[3], "mv_mac_port_daily_count")
	})
}

// -------------------------------------------------------------------------
// mac_history_heatmap_service.go
// -------------------------------------------------------------------------

// TestMhh7905_Heatmap 热力图服务: sqlite 非 MV 回退分支 + perfTopN + 缓存装配。
func TestMhh7905_Heatmap(t *testing.T) {
	ctx := context.Background()
	db, _, _, _, heatmap := newMhs7905(t)

	t.Run("perfTopN_default_and_seed", func(t *testing.T) {
		assert.Equal(t, 100, newMhs7905Heatmap(t, db).perfTopN(ctx), "无配置应回默认 100")

		require.NoError(t, db.Create(&models.Config{
			ConfigName:  "MAC热力图TopN",
			ConfigKey:   MACPerfConfigHeatmapTopN,
			ConfigValue: "42",
			ConfigType:  models.ConfigTypeYes,
		}).Error)
		assert.Equal(t, 42, newMhs7905Heatmap(t, db).perfTopN(ctx), "应读取 DB 配置")

		for _, bad := range []string{"abc", "0", "-3"} {
			require.NoError(t, db.Model(&models.Config{}).
				Where("config_key = ?", MACPerfConfigHeatmapTopN).
				Update("config_value", bad).Error)
			assert.Equal(t, 100, newMhs7905Heatmap(t, db).perfTopN(ctx), "非法值 %q 应回默认", bad)
		}
	})

	t.Run("perfCacheTTL_fallback_and_config", func(t *testing.T) {
		impl := newMhs7905Heatmap(t, db)
		assert.Equal(t, 30*time.Minute, impl.perfCacheTTL(), "perfConfig 非 nil → 30 分钟兜底(QUIRK-79-05-A 同根)")
		impl.perfConfig = nil
		assert.Equal(t, 5*time.Minute, impl.perfCacheTTL(), "perfConfig nil → 5 分钟")
	})

	t.Run("query_sqlite_fallback_empty_shape", func(t *testing.T) {
		res, err := heatmap.QueryHeatmap(ctx, &HeatmapQuery{})
		require.NoError(t, err, "sqlite 应走非 MV 回退分支返回空结果")
		require.NotNil(t, res)
		assert.NotNil(t, res.Cells)
		assert.Empty(t, res.Cells)
		assert.Equal(t, 100, res.TopN, "缺省 TopN 应取 perfTopN 默认")
		assert.NotEmpty(t, res.Start)
		assert.NotEmpty(t, res.End)
		assert.NotEmpty(t, res.Snapshot)
		// 缺省时间窗: 7 天(req 时间被原地补齐)
		start, err := time.Parse(time.RFC3339, res.Start)
		require.NoError(t, err)
		end, err := time.Parse(time.RFC3339, res.End)
		require.NoError(t, err)
		assert.InDelta(t, 7*24*time.Hour.Hours(), end.Sub(start).Hours(), 1.0, "缺省窗口应为 7 天±1h")
	})

	t.Run("query_cached_parity", func(t *testing.T) {
		req := &HeatmapQuery{TopN: 7}
		first, err := heatmap.QueryHeatmap(ctx, req)
		require.NoError(t, err)
		second, err := heatmap.QueryHeatmap(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, first.TopN, second.TopN)
		assert.Equal(t, len(first.Cells), len(second.Cells))
	})

	t.Run("pg_fake_mv_query_fails_wrapped", func(t *testing.T) {
		fakeDB, captured := newMhs7905PGFake(t)
		hm := NewMACHistoryHeatmapService(fakeDB, nil, nil)

		impl, ok := hm.(*macHistoryHeatmapServiceImpl)
		require.True(t, ok)
		assert.True(t, impl.isPostgreSQL())

		res, err := hm.QueryHeatmap(ctx, &HeatmapQuery{
			StartTime: mhq7905Time(8, 0, 0).Format(time.RFC3339),
			EndTime:   mhq7905Time(12, 0, 0).Format(time.RFC3339),
		})
		require.Error(t, err, "R6: 假方言查询 mv_mac_port_daily_count 必失败")
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "查询物化视图 mv_mac_port_daily_count 失败")
		// captured[0] 是 perfTopN(TopN<=0 时)对 sys_config 的查询(假方言下同样失败 → 回默认 100)
		require.GreaterOrEqual(t, len(*captured), 2)
		mvSQL := (*captured)[len(*captured)-1]
		assert.Contains(t, mvSQL, "FROM mv_mac_port_daily_count")
		assert.Contains(t, mvSQL, "ORDER BY change_count DESC")
	})

	t.Run("pg_fake_query_skips_cache_branch", func(t *testing.T) {
		// dataCache == nil 时应直查(缓存装饰分支关闭)
		fakeDB, _ := newMhs7905PGFake(t)
		hm := NewMACHistoryHeatmapService(fakeDB, nil, nil)
		_, err := hm.QueryHeatmap(ctx, &HeatmapQuery{StartTime: mhq7905Time(8, 0, 0).Format(time.RFC3339), EndTime: mhq7905Time(12, 0, 0).Format(time.RFC3339)})
		require.Error(t, err)
	})
}

// newMhs7905Heatmap 在既有库上再装配一个 heatmap service(独立 MemoryCache)。
func newMhs7905Heatmap(t *testing.T, db *gorm.DB) *macHistoryHeatmapServiceImpl {
	t.Helper()
	mem := cache.NewMemoryCache(100, time.Minute)
	t.Cleanup(func() { mem.Close() })
	svc := NewMACHistoryHeatmapService(db, NewDataCacheService(mem), NewCacheConfigService(db))
	impl, ok := svc.(*macHistoryHeatmapServiceImpl)
	require.True(t, ok)
	return impl
}

// -------------------------------------------------------------------------
// mac_perf_config_seed.go
// -------------------------------------------------------------------------

// TestMps7905_SeedMACPerfConfigs SeedMACPerfConfigs(:25-46)幂等 upsert 3 键 + nil db 分支。
func TestMps7905_SeedMACPerfConfigs(t *testing.T) {
	ctx := context.Background()

	t.Run("nil_db_returns_nil", func(t *testing.T) {
		assert.NoError(t, SeedMACPerfConfigs(nil), "db==nil 应 Warnf + 返回 nil")
	})

	t.Run("seeds_three_keys_matching_defaults", func(t *testing.T) {
		db, _, _, _, _ := newMhs7905(t)
		require.NoError(t, SeedMACPerfConfigs(db))

		for _, want := range macPerfConfigDefaults {
			var got models.Config
			require.NoError(t, db.Where("config_key = ?", want.ConfigKey).First(&got).Error,
				"键 %s 应落库", want.ConfigKey)
			assert.Equal(t, want.ConfigValue, got.ConfigValue)
			assert.Equal(t, want.ConfigName, got.ConfigName)
			assert.Equal(t, want.ConfigType, got.ConfigType)
		}

		var count int64
		require.NoError(t, db.Model(&models.Config{}).Where("config_key LIKE ?", "network.mac.perf.%").Count(&count).Error)
		assert.Equal(t, int64(3), count, "应恰好 3 个 MAC 性能配置键")
	})

	t.Run("idempotent_no_overwrite", func(t *testing.T) {
		db, _, _, _, _ := newMhs7905(t)
		require.NoError(t, SeedMACPerfConfigs(db))

		// 人为改值 → 重跑不应覆盖(FirstOrCreate 语义)
		require.NoError(t, db.Model(&models.Config{}).
			Where("config_key = ?", MACPerfConfigHeatmapTopN).
			Update("config_value", "999").Error)

		require.NoError(t, SeedMACPerfConfigs(db))

		var got models.Config
		require.NoError(t, db.Where("config_key = ?", MACPerfConfigHeatmapTopN).First(&got).Error)
		assert.Equal(t, "999", got.ConfigValue, "已存在键不应被种子覆盖")

		var count int64
		require.NoError(t, db.Model(&models.Config{}).Where("config_key LIKE ?", "network.mac.perf.%").Count(&count).Error)
		assert.Equal(t, int64(3), count, "重复调用不新增行")
	})

	_ = ctx
}
