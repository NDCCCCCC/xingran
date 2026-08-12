package services

import (
	"context"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newMACHistoryTestDB 构造一个 sqlite 内存库 + 自动迁移 DeviceMACHistory 表。
// 单元测试专用,无需真实 PG。
func newMACHistoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.DeviceMACHistory{}))
	return db
}

// makeTestDevice 构造一个 NetworkDevice(ID 通过 BaseModel 嵌入字段设置)。
func makeTestDevice(id string) *models.NetworkDevice {
	return &models.NetworkDevice{
		BaseModel:   models.BaseModel{ID: id},
		DeviceName:  "test-device",
		IPAddress:   "10.0.0.1",
		DeviceType:  models.DeviceTypeSwitch,
		Vendor:      models.VendorHuawei,
	}
}

// makeMAC 构造一条 DeviceMACAddress 用于 BuildMACStateMap 输入。
func makeMAC(mac string, iface string, vlan int, collectedAt time.Time) models.DeviceMACAddress {
	var vlanPtr *int
	if vlan > 0 {
		vlanPtr = &vlan
	}
	return models.DeviceMACAddress{
		DeviceID:      "device-1",
		MACAddress:    mac,
		InterfaceName: iface,
		VLANID:        vlanPtr,
		MACType:       models.MACTypeDynamic,
		CollectedAt:   collectedAt,
	}
}

// runRecordChangeSetup 在独立 DB 上跑 RecordMACChange 并返回写入条数 + 所有 records
// (每个子测试用自己的 DB,避免 records 累积污染断言)。
func runRecordChangeSetup(
	t *testing.T,
	oldMACs, newMACs []models.DeviceMACAddress,
) (int, []models.DeviceMACHistory) {
	t.Helper()
	db := newMACHistoryTestDB(t)
	svc := NewMACHistoryService(db)
	device := makeTestDevice("device-1")
	oldState := svc.BuildMACStateMap(oldMACs)
	newState := svc.BuildMACStateMap(newMACs)
	require.NoError(t, svc.RecordMACChange(context.Background(), device, oldState, newState))
	var records []models.DeviceMACHistory
	require.NoError(t, db.Find(&records).Error)
	return len(records), records
}

// TestRecordMACChange_RealEventsRecorded 锁定 2026-07-01 修正后的契约:
//
// 之前 (commit 467df3fc) 加了 L2 同端口 history re-check,会把"同一 MAC 同端口反复出现/
// 消失/迁移"的真实事件吞掉 — 用户报告的 bug 现象:同一 MAC 在 GE5/44 多次出现/迁移/
// 消失全部缺失。
//
// 修正后:RecordMACChange 不再做 re-check,所有真实状态变化都写入。flapping 由
// MergeFlappingRecords (2h 窗口) 兜底合并;长尾由 PurgeMeaninglessRecords 兜底
// 清理(只保留每 MAC 首条 appeared)。
//
// 关联: [[mac-address-normalize-returns-colon-format]]
func TestRecordMACChange_RealEventsRecorded(t *testing.T) {
	t0 := time.Date(2026, 7, 1, 15, 0, 0, 0, time.Local)

	t.Run("首次出现应写 1 条 appeared", func(t *testing.T) {
		count, _ := runRecordChangeSetup(t,
			[]models.DeviceMACAddress{},
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 5/44", 310, t0),
			},
		)
		assert.Equal(t, 1, count, "首次出现应写 1 条 appeared")
	})

	t.Run("同端口反复出现/消失/迁移(用户 bug 场景)全部写入", func(t *testing.T) {
		// 模拟用户截图场景:MAC 在 GE5/44 多次出现/消失/迁移
		db := newMACHistoryTestDB(t)
		svc := NewMACHistoryService(db)
		device := makeTestDevice("device-1")
		ctx := context.Background()

		// Cycle 1 (15:00): MAC 首次出现在 GE5/44
		t1 := t0
		require.NoError(t, svc.RecordMACChange(ctx, device,
			svc.BuildMACStateMap([]models.DeviceMACAddress{}),
			svc.BuildMACStateMap([]models.DeviceMACAddress{
				makeMAC("F8:8C:21:87:6D:7A", "GigabitEthernet 5/44", 310, t1),
			}),
		))

		// Cycle 2 (15:30): MAC 离开 GE5/44(disappeared)
		t2 := t1.Add(30 * time.Minute)
		require.NoError(t, svc.RecordMACChange(ctx, device,
			svc.BuildMACStateMap([]models.DeviceMACAddress{
				makeMAC("F8:8C:21:87:6D:7A", "GigabitEthernet 5/44", 310, t2),
			}),
			svc.BuildMACStateMap([]models.DeviceMACAddress{}),
		))

		// Cycle 3 (16:00): MAC 回到 GE5/44(appeared)
		t3 := t2.Add(30 * time.Minute)
		require.NoError(t, svc.RecordMACChange(ctx, device,
			svc.BuildMACStateMap([]models.DeviceMACAddress{}),
			svc.BuildMACStateMap([]models.DeviceMACAddress{
				makeMAC("F8:8C:21:87:6D:7A", "GigabitEthernet 5/44", 310, t3),
			}),
		))

		// Cycle 4 (16:30): MAC 离开(disappeared)
		t4 := t3.Add(30 * time.Minute)
		require.NoError(t, svc.RecordMACChange(ctx, device,
			svc.BuildMACStateMap([]models.DeviceMACAddress{
				makeMAC("F8:8C:21:87:6D:7A", "GigabitEthernet 5/44", 310, t4),
			}),
			svc.BuildMACStateMap([]models.DeviceMACAddress{}),
		))

		// 验证:4 个真实事件全部写入(不吞掉任何一次出现/消失)
		var records []models.DeviceMACHistory
		require.NoError(t, db.Find(&records).Error)
		assert.Len(t, records, 4, "同端口反复出现/消失应全部记录 4 条事件")

		// 顺序:appeared → disappeared → appeared → disappeared
		assert.Equal(t, models.EventAppeared, records[0].EventType)
		assert.Equal(t, models.EventDisappeared, records[1].EventType)
		assert.Equal(t, models.EventAppeared, records[2].EventType)
		assert.Equal(t, models.EventDisappeared, records[3].EventType)
	})

	t.Run("同端口同 VLAN 持续存在应写 1 条 appeared(moved 路径不触发)", func(t *testing.T) {
		_, records := runRecordChangeSetup(t,
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/1", 100, t0),
			},
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/1", 100, t0),
			},
		)
		// 同 MAC + 同接口 + 同 VLAN = 无任何变化,不写事件
		assert.Len(t, records, 0, "完全相同状态不应产生事件")
	})

	t.Run("完全新增 MAC 应只写 1 条 appeared", func(t *testing.T) {
		_, records := runRecordChangeSetup(t,
			[]models.DeviceMACAddress{},
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/1", 100, t0),
			},
		)
		require.Len(t, records, 1)
		assert.Equal(t, models.EventAppeared, records[0].EventType)
		assert.Equal(t, "AA:BB:CC:DD:EE:FF", records[0].MACAddress)
	})

	t.Run("MAC 从端口 A 移到端口 B 应只写 1 条 moved(不写 appeared/disappeared)", func(t *testing.T) {
		t1 := t0.Add(time.Hour)
		beforeRun := time.Now()
		_, records := runRecordChangeSetup(t,
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/1", 100, t0),
			},
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/2", 100, t1),
			},
		)
		require.Len(t, records, 1, "真实移动只写 1 条 moved")
		assert.Equal(t, models.EventMoved, records[0].EventType)
		// 2026-07-01: BuildMACStateMap + DeviceMACHistory.BeforeCreate hook 双重归一化,
		// moved 记录的 InterfaceName 落短名(GE1/2)而非全称(GigabitEthernet 1/2)
		assert.Equal(t, "GE1/2", records[0].InterfaceName, "moved 记录用新端口(归一化短名,语义=现在在哪)")
		assert.Equal(t, t0, records[0].FirstSeen, "moved 起点 FirstSeen = 旧 CollectedAt(何时离开旧端口)")
		// LastSeen = collectionTime(本次 RecordMACChange 调用时的 time.Now()),不固定
		assert.True(t, !records[0].LastSeen.Before(beforeRun),
			"moved 终点 LastSeen 应 >= 调用前时间(实测 %v)", records[0].LastSeen)
	})

	t.Run("MAC 完全消失应只写 1 条 disappeared", func(t *testing.T) {
		_, records := runRecordChangeSetup(t,
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/1", 100, t0),
			},
			[]models.DeviceMACAddress{},
		)
		require.Len(t, records, 1)
		assert.Equal(t, models.EventDisappeared, records[0].EventType)
		assert.Equal(t, t0, records[0].FirstSeen)
	})

	t.Run("同端口换 VLAN 应只写 1 条 vlan_changed", func(t *testing.T) {
		t1 := t0.Add(time.Hour)
		_, records := runRecordChangeSetup(t,
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/1", 100, t0),
			},
			[]models.DeviceMACAddress{
				makeMAC("AA:BB:CC:DD:EE:FF", "GigabitEthernet 1/1", 200, t1),
			},
		)
		require.Len(t, records, 1, "同端口 VLAN 变化只写 1 条 vlan_changed")
		assert.Equal(t, models.EventVLANChanged, records[0].EventType)
		assert.Equal(t, 200, *records[0].VLANID, "新 VLAN")
	})
}

// TestRecordMACChange_StableMAC7Days_AllEventsWritten 端到端:模拟 7 天同端口稳定 MAC +
// 10% 漏采,验证修复后事件全部写入(不再过度优化)。
//
// 修复前的过度优化会让稳定 MAC 7 天只写 1 条;修复后所有真实状态变化都写,
// 由 MergeFlappingRecords 后续合并(2h 窗口内 flapping 合并为单条)。
//
// 已知 trade-off:MergeFlappingRecords 当前用 `appeared.FirstSeen = collectionTime` vs
// `disappeared.FirstSeen = oldMAC.CollectedAt` 时间尺度不一致,2h 窗口可能错过合并;
// 长尾清理由 PurgeMeaninglessRecords 兜底(每月 cron)。本测试只断言事件写入符合预期。
func TestRecordMACChange_StableMAC7Days_AllEventsWritten(t *testing.T) {
	db := newMACHistoryTestDB(t)
	svc := NewMACHistoryService(db)
	device := makeTestDevice("device-1")
	ctx := context.Background()

	// 模拟 7 天、每 30 分钟一次采集,~336 周期
	// 制造少量漏采(每 10 周期漏 1 次,模拟 SNMP 抖动)
	startTime := time.Date(2026, 6, 24, 0, 0, 0, 0, time.Local)
	const totalCycles = 336
	const missedCycles = totalCycles / 10 // 33 漏采周期

	prevState := svc.BuildMACStateMap([]models.DeviceMACAddress{})
	for i := 0; i < totalCycles; i++ {
		now := startTime.Add(time.Duration(i) * 30 * time.Minute)
		var cycleMACs []models.DeviceMACAddress
		if i%10 != 0 { // 每 10 周期漏 1 次
			cycleMACs = []models.DeviceMACAddress{
				makeMAC("9C:7B:EF:2F:31:B8", "GigabitEthernet 5/44", 310, now),
			}
		}
		newState := svc.BuildMACStateMap(cycleMACs)
		require.NoError(t, svc.RecordMACChange(ctx, device, prevState, newState))
		prevState = newState
	}

	// 预期事件数:
	//   - Cycle 0 (首次出现) → 1 appeared
	//   - 33 漏采周期(空 → 有或 有 → 空)各产生 1 disappeared 或 1 appeared
	//   - 周期之间无变化 → 0 事件
	//   - 共 1 + 33×2 = 67 条
	const expectedCount = int64(1 + missedCycles*2)
	var count int64
	db.Model(&models.DeviceMACHistory{}).Count(&count)
	assert.Equal(t, expectedCount, count,
		"7 天稳定 MAC + 10%% 漏采应产生 ~%d 条真实事件(实测 %d),不再被 L2 吞掉",
		expectedCount, count)
}