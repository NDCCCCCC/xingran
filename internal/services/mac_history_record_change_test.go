package services

import (
	"context"
	"testing"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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

// TestRecordMACChange_SkipsRedundantEvents 锁定 2026-06-30 quick 的写入去重契约:
//
// 之前逻辑按 (mac, interface, vlan) 三元组比对,collector 漏采 1 个周期
// 会产生 disappeared + 重新出现产生 appeared 配对,导致 busy MAC 7 天写 366 条
// 冗余记录。修复后用 normalized MAC 集合判定 + history 表同端口 re-check 跳过。
//
// 关联: [[mac-address-normalize-returns-colon-format]]
func TestRecordMACChange_SkipsRedundantEvents(t *testing.T) {
	t0 := time.Date(2026, 6, 30, 10, 0, 0, 0, time.Local)

	t.Run("同端口漏采 1 个周期不写任何事件(需先有 appeared 历史)", func(t *testing.T) {
		// 用独立 DB,先跑一次"首次出现"建立 history 记录,再跑"漏采"
		db := newMACHistoryTestDB(t)
		svc := NewMACHistoryService(db)
		device := makeTestDevice("device-1")
		ctx := context.Background()

		// Step 1: 首次出现 → 写 1 条 appeared(建立 history)
		require.NoError(t, svc.RecordMACChange(ctx, device,
			svc.BuildMACStateMap([]models.DeviceMACAddress{}),
			svc.BuildMACStateMap([]models.DeviceMACAddress{
				makeMAC("9C:7B:EF:2F:31:B8", "GigabitEthernet 5/44", 310, t0),
			}),
		))
		var after1 []models.DeviceMACHistory
		require.NoError(t, db.Find(&after1).Error)
		require.Len(t, after1, 1, "首次出现应写 1 条 appeared")

		// Step 2: 漏采 1 个周期(同端口)→ L2 history re-check 命中,跳过
		t1 := t0.Add(30 * time.Minute)
		require.NoError(t, svc.RecordMACChange(ctx, device,
			svc.BuildMACStateMap([]models.DeviceMACAddress{
				makeMAC("9C:7B:EF:2F:31:B8", "GigabitEthernet 5/44", 310, t1),
			}),
			svc.BuildMACStateMap([]models.DeviceMACAddress{}),
		))
		var after2 []models.DeviceMACHistory
		require.NoError(t, db.Find(&after2).Error)
		assert.Len(t, after2, 1, "同端口漏采应不写新事件(总数仍 1)")
	})

	t.Run("同端口空 -> 出现不应触发 appeared", func(t *testing.T) {
		count, _ := runRecordChangeSetup(t,
			[]models.DeviceMACAddress{},
			[]models.DeviceMACAddress{
				makeMAC("9C:7B:EF:2F:31:B8", "GigabitEthernet 5/44", 310, t0),
			},
		)
		// 首次出现:oldMACs 空 + newMACs 有 → L1 通过(no oldMACNormSet match)
		// 但 L2 检查:history 表无记录 → 不跳过 → 写 1 条 appeared
		assert.Equal(t, 1, count, "首次出现应写 1 条 appeared")
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
		assert.Equal(t, "GigabitEthernet 1/2", records[0].InterfaceName, "moved 记录用新端口(语义=现在在哪)")
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

// TestRecordMACChange_StableMACOver7Days 端到端:模拟 7 天同端口稳定 MAC + 10% 漏采,
// 验证修复后总事件数应远小于原 366,实际应为 1(首次出现)+ 0(后续稳定/漏采)。
func TestRecordMACChange_StableMACOver7Days(t *testing.T) {
	db := newMACHistoryTestDB(t)
	svc := NewMACHistoryService(db)
	device := makeTestDevice("device-1")
	ctx := context.Background()

	// 模拟 7 天、每 30 分钟一次采集,~336 周期
	// 制造少量漏采(每 10 周期漏 1 次,模拟 SNMP 抖动),验证修复后无影响
	startTime := time.Date(2026, 6, 23, 0, 0, 0, 0, time.Local)
	const totalCycles = 336

	prevState := svc.BuildMACStateMap([]models.DeviceMACAddress{})
	for i := 0; i < totalCycles; i++ {
		now := startTime.Add(time.Duration(i) * 30 * time.Minute)
		var cycleMACs []models.DeviceMACAddress
		// 每 10 周期漏 1 次(模拟 SNMP 抖动),其他周期 MAC 在同端口
		if i%10 != 0 {
			cycleMACs = []models.DeviceMACAddress{
				makeMAC("9C:7B:EF:2F:31:B8", "GigabitEthernet 5/44", 310, now),
			}
		}
		newState := svc.BuildMACStateMap(cycleMACs)
		require.NoError(t, svc.RecordMACChange(ctx, device, prevState, newState))
		prevState = newState
	}

	var count int64
	db.Model(&models.DeviceMACHistory{}).Count(&count)
	// 第 1 个非漏采周期写 1 条 appeared(这是真正的新 MAC)
	// 后续稳定/漏采周期全部被去重契约跳过(L1 norm set + L2 history 同端口 re-check)
	assert.Equal(t, int64(1), count,
		"同端口稳定 MAC(10%% 漏采模拟)7 天后应只写 1 条首次 appeared,原逻辑会产生 ~360 冗余条")
}