package services

// Phase 72 #3 (DBG-03): MAC 写路径格式防线回归测试。
//
// 不变量: 任何写入 sys_device_mac_address / sys_device_mac_history 的 MAC,
// 无论上游喂进什么原始格式(点分/连字符/小写冒号/无分隔/垃圾),
// 入库一律为 canonical 大写冒号格式 `AA:BB:CC:DD:EE:FF`,接口名经归一化非空;
// 不合格记录被显式丢弃并计数(Warn 可观测),绝不原样入库。
//
// 写入方枚举(生产代码全量 grep Create/Save/Upsert/Exec/SkipHooks):
//   W1 = mac_collection_service.go Step8 Create(&macRecords)  → SanitizeMACRecords
//   W2 = mac_history_service.go Create(&historyRecords)        → SanitizeMACHistoryRecords
// 枚举数(2) = normalize 覆盖数(2)。

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestSanitizeMACRecords_NormalizesMixedFormats 锁定 W1 防线不变量:
// 混合格输入全部归一为大写冒号(与 DB 存量格式一致,2026-10-27 实测)。
func TestSanitizeMACRecords_NormalizesMixedFormats(t *testing.T) {
	inputs := []struct {
		raw string
		want string
	}{
		{"b022.7a2e.4a4f", "B0:22:7A:2E:4A:4F"},              // Cisco 点分
		{"00-e0-fc-09-bc-f9", "00:E0:FC:09:BC:F9"},            // 连字符小写
		{"00:E0:FC:09:BC:F9", "00:E0:FC:09:BC:F9"},            // 已 canonical(幂等)
		{"00e0fc09bcf9", "00:E0:FC:09:BC:F9"},                 // 无分隔符
		{"d89e.f327.2d19", "D8:9E:F3:27:2D:19"},               // 点分小写
	}
	for _, tt := range inputs {
		records := []*models.DeviceMACAddress{{
			DeviceID:      "dev-1",
			MACAddress:    tt.raw,
			InterfaceName: "GigabitEthernet 2/25",
			CollectedAt:   time.Now(),
		}}
		clean, rejected := SanitizeMACRecords("test-dev", records)
		require.Len(t, clean, 1, "raw=%s 应通过防线", tt.raw)
		assert.Zero(t, rejected)
		assert.Equal(t, tt.want, clean[0].MACAddress, "raw=%s 必须归一为大写冒号", tt.raw)
		assert.True(t, isCanonicalMAC(clean[0].MACAddress))
		assert.Equal(t, "GE2/25", clean[0].InterfaceName, "接口名必须归一(全称+空格→短名)")
	}
}

// TestSanitizeMACRecords_RejectsGarbage 锁定防线拒绝语义:
// 垃圾 MAC / 空 MAC / 空接口名被丢弃且计数(Warn 可观测)。
func TestSanitizeMACRecords_RejectsGarbage(t *testing.T) {
	records := []*models.DeviceMACAddress{
		{DeviceID: "d", MACAddress: "Flags:", InterfaceName: "GE0/1", CollectedAt: time.Now()},    // 垃圾表头
		{DeviceID: "d", MACAddress: "Total", InterfaceName: "GE0/1", CollectedAt: time.Now()},     // 汇总行
		{DeviceID: "d", MACAddress: "", InterfaceName: "GE0/1", CollectedAt: time.Now()},          // 空 MAC
		{DeviceID: "d", MACAddress: "00:E0:FC:09:BC:F9", InterfaceName: "", CollectedAt: time.Now()}, // 空接口名(not null 列)
		{DeviceID: "d", MACAddress: "00:E0:FC:09:BC:F9", InterfaceName: "GE0/1", CollectedAt: time.Now()}, // 合法
	}
	clean, rejected := SanitizeMACRecords("test-dev", records)
	assert.Equal(t, 4, rejected, "4 条不合格应被拦截")
	require.Len(t, clean, 1)
	assert.Equal(t, "00:E0:FC:09:BC:F9", clean[0].MACAddress)
}

// TestSanitizeMACHistoryRecords_LocksInvariant 锁定 W2 (sys_device_mac_history) 防线。
func TestSanitizeMACHistoryRecords_LocksInvariant(t *testing.T) {
	now := time.Now()
	records := []*models.DeviceMACHistory{
		{DeviceID: "d", MACAddress: "b022.7a2e.4a4f", InterfaceName: "GigabitEthernet 2/25",
			EventType: models.EventAppeared, FirstSeen: now, LastSeen: now, CollectedAt: now},
		{DeviceID: "d", MACAddress: "9c7b.ef2f.0222", InterfaceName: "FastEthernet 1/0",
			EventType: models.EventAppeared, FirstSeen: now, LastSeen: now, CollectedAt: now},
		{DeviceID: "d", MACAddress: "Invalid", InterfaceName: "GE0/1",
			EventType: models.EventAppeared, FirstSeen: now, LastSeen: now, CollectedAt: now},
	}
	clean, rejected := SanitizeMACHistoryRecords("test-dev", records)
	assert.Equal(t, 1, rejected)
	require.Len(t, clean, 2)
	assert.Equal(t, "B0:22:7A:2E:4A:4F", clean[0].MACAddress)
	assert.Equal(t, "9C:7B:EF:2F:02:22", clean[1].MACAddress)
	assert.Equal(t, "FE1/0", clean[1].InterfaceName, "FastEthernet 1/0(含空格)必须归一")
	for _, r := range clean {
		assert.True(t, isCanonicalMAC(r.MACAddress))
	}
}

// TestParseMACAddressTable_UnknownVendorFailsLoudly 锁定 default 分支语义:
// 未知厂商 parseMACAddressTable 显式返回 error(可观测失败),不再静默产出空 entries。
func TestParseMACAddressTable_UnknownVendorFailsLoudly(t *testing.T) {
	svc := &MACCollectionService{}
	// vendor "juniper" 无解析分支 — 修复前这里静默返回空切片
	entries, err := svc.parseMACAddressTable("100 00:11:22:33:44:55 Dynamic Gi0/1", "juniper", MACCommandTypeMacTable)
	require.Error(t, err, "未知厂商必须显式报错(静默丢弃=R2 复发根因之一)")
	assert.Nil(t, entries)
	assert.Contains(t, err.Error(), "juniper")

	// parseMACLine 行级 default 兜底同样显式拒绝
	_, ok := svc.parseMACLine("100 00:11:22:33:44:55 Dynamic Gi0/1", "juniper", MACCommandTypeMacTable)
	assert.False(t, ok)
}

// TestWritePathInvariant_SQLiteEndToEnd 端到端锁定不变量:
// 混格式记录经 sanitize 后真实 Create 入库(SQLite),读回后 100% canonical 大写冒号。
// 同时覆盖 models.BeforeCreate hook 兜底层(直接 Create 未 sanitize 的记录)。
func TestWritePathInvariant_SQLiteEndToEnd(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.DeviceMACAddress{}, &models.DeviceMACHistory{}))

	now := time.Now()
	raw := []*models.DeviceMACAddress{
		{DeviceID: "d1", MACAddress: "b022.7a2e.4a4f", InterfaceName: "GigabitEthernet 2/25", VLANID: intPtr(308), CollectedAt: now},
		{DeviceID: "d1", MACAddress: "00-e0-fc-09-bc-f9", InterfaceName: "GE0/1", CollectedAt: now},
		{DeviceID: "d1", MACAddress: "Flags:", InterfaceName: "GE0/2", CollectedAt: now}, // 垃圾
	}
	clean, rejected := SanitizeMACRecords("test-dev", raw)
	require.Equal(t, 1, rejected)
	require.Len(t, clean, 2)
	require.NoError(t, db.Create(&clean).Error)

	var stored []models.DeviceMACAddress
	require.NoError(t, db.Find(&stored).Error)
	require.Len(t, stored, 2)
	for _, r := range stored {
		assert.True(t, isCanonicalMAC(r.MACAddress),
			"入库 MAC %q 必须 canonical 大写冒号格式", r.MACAddress)
		assert.False(t, strings.Contains(r.InterfaceName, " "), "接口名不得含空格")
	}

	// hook 兜底层: 未经 sanitize 的混格式记录直接 Create,BeforeCreate 必须归一
	history := &models.DeviceMACHistory{
		DeviceID: "d1", MACAddress: "b022.7a2e.4a4f", InterfaceName: "GigabitEthernet 2/25",
		EventType: models.EventAppeared, FirstSeen: now, LastSeen: now, CollectedAt: now,
	}
	require.NoError(t, db.Create(history).Error)
	assert.Equal(t, "B0:22:7A:2E:4A:4F", history.MACAddress, "BeforeCreate hook 必须兜底归一")

	var histStored models.DeviceMACHistory
	require.NoError(t, db.First(&histStored, "id = ?", history.ID).Error)
	assert.True(t, isCanonicalMAC(histStored.MACAddress))
}

func intPtr(i int) *int { return &i }
