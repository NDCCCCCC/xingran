package services

import (
	"context"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/require"
)

// TestPurgeMeaninglessRecords_DryRun 验证 dry-run 模式仅统计不修改 DB
func TestPurgeMeaninglessRecords_DryRun(t *testing.T) {
	db := newMACHistoryTestDB(t)
	svc := NewMACHistoryService(db)

	// 构造测试数据
	t0 := time.Date(2026, 6, 30, 10, 0, 0, 0, time.Local)
	records := []models.DeviceMACHistory{
		// dev-A mac-X:首次 appeared (KEEP)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/1", EventType: models.EventAppeared, FirstSeen: t0, LastSeen: t0, CollectedAt: t0},
		// dev-A mac-X:冗余 appeared (DELETE)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/1", EventType: models.EventAppeared, FirstSeen: t0.Add(1 * time.Hour), LastSeen: t0.Add(1 * time.Hour), CollectedAt: t0.Add(1 * time.Hour)},
		// dev-A mac-X:disappeared (DELETE)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/1", EventType: models.EventDisappeared, FirstSeen: t0.Add(2 * time.Hour), LastSeen: t0.Add(2 * time.Hour), CollectedAt: t0.Add(2 * time.Hour)},
		// dev-A mac-X:moved (KEEP)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/2", EventType: models.EventMoved, FirstSeen: t0.Add(3 * time.Hour), LastSeen: t0.Add(3 * time.Hour), CollectedAt: t0.Add(3 * time.Hour)},
		// dev-B mac-Y:首次 appeared (KEEP)
		{DeviceID: "dev-B", MACAddress: "AA:BB:CC:DD:EE:02", InterfaceName: "GE2/0/5", EventType: models.EventAppeared, FirstSeen: t0.Add(30 * time.Minute), LastSeen: t0.Add(30 * time.Minute), CollectedAt: t0.Add(30 * time.Minute)},
		// dev-B mac-Y:disappeared (DELETE)
		{DeviceID: "dev-B", MACAddress: "AA:BB:CC:DD:EE:02", InterfaceName: "GE2/0/5", EventType: models.EventDisappeared, FirstSeen: t0.Add(4 * time.Hour), LastSeen: t0.Add(4 * time.Hour), CollectedAt: t0.Add(4 * time.Hour)},
		// dev-B mac-Y:vlan_changed (KEEP)
		{DeviceID: "dev-B", MACAddress: "AA:BB:CC:DD:EE:02", InterfaceName: "GE2/0/5", EventType: models.EventVLANChanged, FirstSeen: t0.Add(5 * time.Hour), LastSeen: t0.Add(5 * time.Hour), CollectedAt: t0.Add(5 * time.Hour)},
	}
	require.NoError(t, db.Create(&records).Error)

	var initialCount int64
	require.NoError(t, db.Model(&models.DeviceMACHistory{}).Count(&initialCount).Error)
	require.Equal(t, int64(7), initialCount, "测试数据应为 7 条")

	// Dry-run:应预测 3 条删除(disappeared 2 + 冗余 appeared 1),备份表名为空
	deleted, backupTable, err := svc.PurgeMeaninglessRecords(context.Background(), true)
	require.NoError(t, err)
	require.Equal(t, int64(3), deleted, "dry-run 应预测删除 3 条")
	require.Equal(t, "", backupTable, "dry-run 不创建备份表")

	// DB 行数应不变
	var afterCount int64
	require.NoError(t, db.Model(&models.DeviceMACHistory{}).Count(&afterCount).Error)
	require.Equal(t, int64(7), afterCount, "dry-run 不应修改 DB")

	// 验证 7 条记录全部还在
	var kept []models.DeviceMACHistory
	require.NoError(t, db.Find(&kept).Error)
	require.Equal(t, 7, len(kept))
}

// TestPurgeMeaninglessRecords_RealRun 验证真跑模式删除正确 + 保留正确
func TestPurgeMeaninglessRecords_RealRun(t *testing.T) {
	db := newMACHistoryTestDB(t)
	svc := NewMACHistoryService(db)

	t0 := time.Date(2026, 6, 30, 10, 0, 0, 0, time.Local)
	records := []models.DeviceMACHistory{
		// dev-A mac-X:首次 appeared (KEEP)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/1", EventType: models.EventAppeared, FirstSeen: t0, LastSeen: t0, CollectedAt: t0},
		// dev-A mac-X:冗余 appeared (DELETE)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/1", EventType: models.EventAppeared, FirstSeen: t0.Add(1 * time.Hour), LastSeen: t0.Add(1 * time.Hour), CollectedAt: t0.Add(1 * time.Hour)},
		// dev-A mac-X:disappeared (DELETE)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/1", EventType: models.EventDisappeared, FirstSeen: t0.Add(2 * time.Hour), LastSeen: t0.Add(2 * time.Hour), CollectedAt: t0.Add(2 * time.Hour)},
		// dev-A mac-X:moved (KEEP)
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/2", EventType: models.EventMoved, FirstSeen: t0.Add(3 * time.Hour), LastSeen: t0.Add(3 * time.Hour), CollectedAt: t0.Add(3 * time.Hour)},
		// dev-B mac-Y:首次 appeared (KEEP)
		{DeviceID: "dev-B", MACAddress: "AA:BB:CC:DD:EE:02", InterfaceName: "GE2/0/5", EventType: models.EventAppeared, FirstSeen: t0.Add(30 * time.Minute), LastSeen: t0.Add(30 * time.Minute), CollectedAt: t0.Add(30 * time.Minute)},
		// dev-B mac-Y:disappeared (DELETE)
		{DeviceID: "dev-B", MACAddress: "AA:BB:CC:DD:EE:02", InterfaceName: "GE2/0/5", EventType: models.EventDisappeared, FirstSeen: t0.Add(4 * time.Hour), LastSeen: t0.Add(4 * time.Hour), CollectedAt: t0.Add(4 * time.Hour)},
		// dev-B mac-Y:vlan_changed (KEEP)
		{DeviceID: "dev-B", MACAddress: "AA:BB:CC:DD:EE:02", InterfaceName: "GE2/0/5", EventType: models.EventVLANChanged, FirstSeen: t0.Add(5 * time.Hour), LastSeen: t0.Add(5 * time.Hour), CollectedAt: t0.Add(5 * time.Hour)},
	}
	require.NoError(t, db.Create(&records).Error)

	// 真跑
	deleted, backupTable, err := svc.PurgeMeaninglessRecords(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, int64(3), deleted, "应删除 3 条(1 redundant appeared + 2 disappeared)")
	require.NotEmpty(t, backupTable, "应创建备份表")

	// 验证剩余 4 条 = moved(1) + vlan_changed(1) + first_appeared(2)
	var kept []models.DeviceMACHistory
	require.NoError(t, db.Find(&kept).Error)
	require.Equal(t, 4, len(kept), "应保留 4 条")

	// 验证保留的记录类型正确
	eventTypeCounts := map[models.MACEventType]int{}
	for _, r := range kept {
		eventTypeCounts[r.EventType]++
	}
	require.Equal(t, 1, eventTypeCounts[models.EventMoved], "1 条 moved")
	require.Equal(t, 1, eventTypeCounts[models.EventVLANChanged], "1 条 vlan_changed")
	require.Equal(t, 2, eventTypeCounts[models.EventAppeared], "2 条 first-appeared (dev-A + dev-B 各 1)")
	require.Equal(t, 0, eventTypeCounts[models.EventDisappeared], "0 条 disappeared")

	// 验证 first-appeared 是按 first_seen ASC 取的最早一条
	var devAFirst []models.DeviceMACHistory
	require.NoError(t, db.Where("device_id = ? AND mac_address = ? AND event_type = ?",
		"dev-A", "AA:BB:CC:DD:EE:01", models.EventAppeared).Find(&devAFirst).Error)
	require.Equal(t, 1, len(devAFirst), "dev-A mac-X 应保留 1 条 appeared")
	require.True(t, devAFirst[0].FirstSeen.Equal(t0), "应保留最早的 appeared(t0)")
}

// TestPurgeMeaninglessRecords_EmptyDB 验证空表场景不会报错
func TestPurgeMeaninglessRecords_EmptyDB(t *testing.T) {
	db := newMACHistoryTestDB(t)
	svc := NewMACHistoryService(db)

	deleted, backupTable, err := svc.PurgeMeaninglessRecords(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, int64(0), deleted, "空表应预测删除 0 条")
	require.Equal(t, "", backupTable, "空表不创建备份表")
}

// TestPurgeMeaninglessRecords_OnlyMovedAndVlanChanged 验证极端场景:全部是真实事件
func TestPurgeMeaninglessRecords_OnlyMovedAndVlanChanged(t *testing.T) {
	db := newMACHistoryTestDB(t)
	svc := NewMACHistoryService(db)

	t0 := time.Date(2026, 6, 30, 10, 0, 0, 0, time.Local)
	records := []models.DeviceMACHistory{
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:01", InterfaceName: "GE1/0/1", EventType: models.EventMoved, FirstSeen: t0, LastSeen: t0, CollectedAt: t0},
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:02", InterfaceName: "GE1/0/2", EventType: models.EventMoved, FirstSeen: t0, LastSeen: t0, CollectedAt: t0},
		{DeviceID: "dev-A", MACAddress: "AA:BB:CC:DD:EE:03", InterfaceName: "GE1/0/3", EventType: models.EventVLANChanged, FirstSeen: t0, LastSeen: t0, CollectedAt: t0},
	}
	require.NoError(t, db.Create(&records).Error)

	deleted, backupTable, err := svc.PurgeMeaninglessRecords(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, int64(0), deleted, "全是真实事件应预测删除 0 条")
	require.Equal(t, "", backupTable, "无删除时不创建备份表")

	var kept []models.DeviceMACHistory
	require.NoError(t, db.Find(&kept).Error)
	require.Equal(t, 3, len(kept), "全部应保留")
}