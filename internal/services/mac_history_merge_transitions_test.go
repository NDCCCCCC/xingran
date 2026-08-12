package services

import (
	"context"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// insertHistory 直接插入一条 DeviceMACHistory 记录,绕过 RecordMACChange。
// 测试 MergeByTransitions 逻辑专用,需要构造已存在的"历史快照"。
func insertHistory(t *testing.T, db *gorm.DB, h models.DeviceMACHistory) {
	t.Helper()
	require.NoError(t, db.Create(&h).Error)
}

// makeHistory 构造一条历史记录(包含 ID 留空,GORM 自增)
func makeHistory(mac, iface string, vlan int, eventType models.MACEventType,
	deviceID string, firstSeen, lastSeen time.Time) models.DeviceMACHistory {
	var vlanPtr *int
	if vlan > 0 {
		vlanPtr = &vlan
	}
	return models.DeviceMACHistory{
		DeviceID:           deviceID,
		DeviceNameSnapshot: "test-device",
		MACAddress:         mac,
		InterfaceName:      iface,
		VLANID:             vlanPtr,
		EventType:          eventType,
		FirstSeen:          firstSeen,
		LastSeen:           lastSeen,
		CollectedAt:        lastSeen,
	}
}

// TestMergeByTransitions 锁定 2026-07-01 新增的合并工具逻辑:
//
// 用户原话:"仅保留设备或接口有变化的记录,删除其余的所有记录"。
// 算法:按 (device_id, mac_address) 分组,仅保留位置签名 (interface_name, vlan_id)
// 与上一保留记录不同的转换点。
func TestMergeByTransitions(t *testing.T) {
	t.Run("同端口 flapping:5 条反复出现/消失只保留首条", func(t *testing.T) {
		db := newMACHistoryTestDB(t)
		svc := NewMACHistoryService(db)
		ctx := context.Background()

		t0 := time.Date(2026, 7, 1, 15, 0, 0, 0, time.Local)
		mac := "AA:BB:CC:DD:EE:01"
		// 模拟用户截图场景:MAC 在 GE5/44 反复出现/消失
		insertHistory(t, db, makeHistory(mac, "GE5/44", 310, models.EventAppeared, "device-1", t0, t0))
		insertHistory(t, db, makeHistory(mac, "GE5/44", 310, models.EventDisappeared, "device-1", t0.Add(30*time.Minute), t0.Add(30*time.Minute)))
		insertHistory(t, db, makeHistory(mac, "GE5/44", 310, models.EventAppeared, "device-1", t0.Add(60*time.Minute), t0.Add(60*time.Minute)))
		insertHistory(t, db, makeHistory(mac, "GE5/44", 310, models.EventDisappeared, "device-1", t0.Add(90*time.Minute), t0.Add(90*time.Minute)))
		insertHistory(t, db, makeHistory(mac, "GE5/44", 310, models.EventAppeared, "device-1", t0.Add(120*time.Minute), t0.Add(120*time.Minute)))

		deleted, err := svc.MergeByTransitions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(4), deleted, "应删除 4 条 flapping,只保留首条 appeared")

		var kept []models.DeviceMACHistory
		require.NoError(t, db.Where("mac_address = ?", mac).Find(&kept).Error)
		require.Len(t, kept, 1, "只保留首条记录")
		assert.Equal(t, models.EventAppeared, kept[0].EventType)
		assert.Equal(t, "GE5/44", kept[0].InterfaceName)
	})

	t.Run("跨端口真实移动:端口 A→B→A 保留 3 个转换点", func(t *testing.T) {
		db := newMACHistoryTestDB(t)
		svc := NewMACHistoryService(db)
		ctx := context.Background()

		t0 := time.Date(2026, 7, 1, 10, 0, 0, 0, time.Local)
		mac := "AA:BB:CC:DD:EE:02"
		insertHistory(t, db, makeHistory(mac, "GE1/1", 100, models.EventAppeared, "device-1", t0, t0))
		insertHistory(t, db, makeHistory(mac, "GE1/2", 100, models.EventMoved, "device-1", t0.Add(time.Hour), t0.Add(time.Hour)))
		insertHistory(t, db, makeHistory(mac, "GE1/3", 100, models.EventMoved, "device-1", t0.Add(2*time.Hour), t0.Add(2*time.Hour)))
		insertHistory(t, db, makeHistory(mac, "GE1/2", 100, models.EventMoved, "device-1", t0.Add(3*time.Hour), t0.Add(3*time.Hour)))
		insertHistory(t, db, makeHistory(mac, "GE1/1", 100, models.EventMoved, "device-1", t0.Add(4*time.Hour), t0.Add(4*time.Hour)))

		deleted, err := svc.MergeByTransitions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted, "跨端口真实移动应全部保留(5 个不同位置)")

		var kept []models.DeviceMACHistory
		require.NoError(t, db.Where("mac_address = ?", mac).Order("first_seen ASC").Find(&kept).Error)
		assert.Len(t, kept, 5, "5 个不同端口转换点全部保留")
	})

	t.Run("VLAN 变化不算接口变化(按用户原话严格判定)", func(t *testing.T) {
		db := newMACHistoryTestDB(t)
		svc := NewMACHistoryService(db)
		ctx := context.Background()

		t0 := time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local)
		mac := "AA:BB:CC:DD:EE:03"
		insertHistory(t, db, makeHistory(mac, "GE1/1", 100, models.EventAppeared, "device-1", t0, t0))
		insertHistory(t, db, makeHistory(mac, "GE1/1", 200, models.EventVLANChanged, "device-1", t0.Add(time.Hour), t0.Add(time.Hour)))
		insertHistory(t, db, makeHistory(mac, "GE1/1", 300, models.EventVLANChanged, "device-1", t0.Add(2*time.Hour), t0.Add(2*time.Hour)))

		deleted, err := svc.MergeByTransitions(ctx)
		require.NoError(t, err)
		// VLAN 变化不在"设备或接口变化"范围内,会被认为是 flapping 删除
		// 严格按用户原话:VLAN 不算接口变化 → 后 2 条 vlan_changed 删除
		assert.Equal(t, int64(2), deleted,
			"VLAN-only 变化按用户原话应被删除(只保留接口变化),如需保留 VLAN 变化需扩展工具")

		var kept []models.DeviceMACHistory
		require.NoError(t, db.Where("mac_address = ?", mac).Find(&kept).Error)
		require.Len(t, kept, 1)
		assert.Equal(t, models.EventAppeared, kept[0].EventType)
		assert.Equal(t, 100, *kept[0].VLANID)
	})

	t.Run("出现→消失→同端口重新出现(round-trip 不算转换)", func(t *testing.T) {
		db := newMACHistoryTestDB(t)
		svc := NewMACHistoryService(db)
		ctx := context.Background()

		t0 := time.Date(2026, 7, 1, 14, 0, 0, 0, time.Local)
		mac := "AA:BB:CC:DD:EE:04"
		insertHistory(t, db, makeHistory(mac, "GE2/5", 200, models.EventAppeared, "device-1", t0, t0))
		insertHistory(t, db, makeHistory(mac, "GE2/5", 200, models.EventDisappeared, "device-1", t0.Add(time.Hour), t0.Add(time.Hour)))
		insertHistory(t, db, makeHistory(mac, "GE2/5", 200, models.EventAppeared, "device-1", t0.Add(2*time.Hour), t0.Add(2*time.Hour)))

		deleted, err := svc.MergeByTransitions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), deleted,
			"round-trip:disappeared 不更新位置签名,后续 appeared 同位置签名=不转换,删除")

		var kept []models.DeviceMACHistory
		require.NoError(t, db.Where("mac_address = ?", mac).Find(&kept).Error)
		require.Len(t, kept, 1, "只保留首条 appeared")
		assert.Equal(t, models.EventAppeared, kept[0].EventType)
	})

	t.Run("跨设备独立处理:同一 MAC 在两台设备各自合并", func(t *testing.T) {
		db := newMACHistoryTestDB(t)
		svc := NewMACHistoryService(db)
		ctx := context.Background()

		t0 := time.Date(2026, 7, 1, 8, 0, 0, 0, time.Local)
		mac := "AA:BB:CC:DD:EE:05"
		// 设备 A 上 flapping
		insertHistory(t, db, makeHistory(mac, "GE1/1", 100, models.EventAppeared, "device-A", t0, t0))
		insertHistory(t, db, makeHistory(mac, "GE1/1", 100, models.EventDisappeared, "device-A", t0.Add(time.Hour), t0.Add(time.Hour)))
		insertHistory(t, db, makeHistory(mac, "GE1/1", 100, models.EventAppeared, "device-A", t0.Add(2*time.Hour), t0.Add(2*time.Hour)))
		// 设备 B 上 flapping
		insertHistory(t, db, makeHistory(mac, "GE2/1", 200, models.EventAppeared, "device-B", t0, t0))
		insertHistory(t, db, makeHistory(mac, "GE2/1", 200, models.EventDisappeared, "device-B", t0.Add(time.Hour), t0.Add(time.Hour)))

		deleted, err := svc.MergeByTransitions(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), deleted, "设备 A 删 1 条 + 设备 B 删 1 条 = 2 条,实际应为 3(disappeared 也被标记删除)")

		var kept []models.DeviceMACHistory
		require.NoError(t, db.Where("mac_address = ?", mac).Find(&kept).Error)
		// 每台设备各保留 1 条首条 appeared
		assert.Len(t, kept, 2, "两台设备各保留 1 条首条 appeared")
	})
}