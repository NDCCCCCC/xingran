package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestMigrate254_AddsColumnsAndBackfills migration 254 必须：
//  1. 给已存在的 sys_port_write_audit 补四列（AutoMigrate 只建缺失表、不补列）；
//  2. 回填历史行的设备名/IP/接口名/描述；
//  3. 幂等（重复执行不报错、不覆盖已有快照）。
func TestMigrate254_AddsColumnsAndBackfills(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&models.NetworkDevice{}, &models.DevicePortStatus{}, &models.PortWriteAudit{},
	))
	require.NoError(t, db.Create(&models.NetworkDevice{
		BaseModel:  models.BaseModel{ID: "dev-254"},
		DeviceName: "核心交换机-254",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.254.0.1",
	}).Error)
	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-254", DeviceID: "dev-254", InterfaceName: "GE0/0/1", Description: "上行",
	}).Error)

	// 历史行（快照列为空）——模拟 migration 254 之前的审计数据
	require.NoError(t, db.Create(&models.PortWriteAudit{
		DeviceID: "dev-254", PortID: "port-254", Action: "shutdown", Status: "succeeded",
	}).Error)
	// 孤儿行（主数据不存在）——回填后仍为空，不得报错
	require.NoError(t, db.Create(&models.PortWriteAudit{
		DeviceID: "dev-gone", PortID: "port-gone", Action: "shutdown", Status: "failed",
	}).Error)

	require.NoError(t, Migrate254PortWriteAuditNameSnapshot(db))

	type row struct {
		DeviceID                string
		DeviceNameSnapshot      string
		DeviceIPSnapshot        string
		InterfaceNameSnapshot   string
		PortDescriptionSnapshot string
	}
	var rows []row
	require.NoError(t, db.Model(&models.PortWriteAudit{}).
		Select("device_id, device_name_snapshot, device_ip_snapshot, interface_name_snapshot, port_description_snapshot").
		Scan(&rows).Error)
	require.Len(t, rows, 2)

	byDevice := map[string]row{}
	for _, r := range rows {
		byDevice[r.DeviceID] = r
	}
	assert.Equal(t, "核心交换机-254", byDevice["dev-254"].DeviceNameSnapshot)
	assert.Equal(t, "10.254.0.1", byDevice["dev-254"].DeviceIPSnapshot)
	assert.Equal(t, "GE0/0/1", byDevice["dev-254"].InterfaceNameSnapshot)
	assert.Equal(t, "上行", byDevice["dev-254"].PortDescriptionSnapshot)
	assert.Empty(t, byDevice["dev-gone"].DeviceNameSnapshot, "孤儿行不回填、不报错")

	// 幂等：重复执行不报错，且不覆盖已固化快照
	require.NoError(t, db.Model(&models.PortWriteAudit{}).
		Where("device_id = ?", "dev-254").
		Update("device_name_snapshot", "人工修正名称").Error)
	require.NoError(t, Migrate254PortWriteAuditNameSnapshot(db))
	var after models.PortWriteAudit
	require.NoError(t, db.First(&after, "device_id = ?", "dev-254").Error)
	assert.Equal(t, "人工修正名称", after.DeviceNameSnapshot, "回填只动空值行")
}
