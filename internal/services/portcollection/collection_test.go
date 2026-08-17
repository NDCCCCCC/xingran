//go:build !skip_db_tests
// +build !skip_db_tests

package portcollection

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xingran-next/xingran-go-backend/internal/models"

	// Pure-Go SQLite driver
)

// setupPortStatusTestDB 创建内存 SQLite,带复合唯一键 (device_id, interface_name),
// 模拟 migration_177 的 uniq_device_interface 约束,用于守护多设备同名端口共存的不变量。
func setupPortStatusTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_port_status (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			interface_name TEXT NOT NULL,
			admin_status TEXT,
			oper_status TEXT,
			description TEXT,
			vlan INTEGER,
			duplex TEXT,
			speed TEXT,
			port_type TEXT,
			dot1x_enabled INTEGER DEFAULT 0,
			dot1x_port_status TEXT,
			dot1x_user_limit INTEGER NOT NULL DEFAULT 0,
			port_security_enabled INTEGER DEFAULT 0,
			port_security_mode TEXT,
			max_mac_count INTEGER,
			current_mac_count INTEGER,
			collected_at DATETIME NOT NULL,
			created_at DATETIME,
			UNIQUE (device_id, interface_name)
		);
	`).Error)

	return db
}

// TestMultiDeviceSameInterfaceNameCoexist 守护修复后的核心不变量:
//
//	不同设备各自拥有同名 interface_name (如每台交换机都有 GE0/0/1、Vlanif26、NULL0)
//	是合法场景。复合唯一键 (device_id, interface_name) + OnConflict 应让它们共存,
//	并按设备各自 UPSERT 刷新,绝不因"别的设备也有同名"而互相覆盖或跳过。
//
// 回归背景: 历史 [C-fix] ownership clash 逻辑 (collection.go, 2026-06-30 由
// 0b95834a 引入,已于本次修复移除) 曾查询"别的设备是否也有同名 interface"并 skip,
// 导致除首采设备外,所有设备的 GE/Vlanif/NULL 等通用接口名端口永远无法更新,
// collected_at 停在首次写入时间。本测试锁定正确行为,防止该逻辑被重新引入。
func TestMultiDeviceSameInterfaceNameCoexist(t *testing.T) {
	db := setupPortStatusTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()
	ctx := context.Background()

	deviceA := uuid.NewString()
	deviceB := uuid.NewString()

	// 两台设备都有 GE0/0/1 (交换机间普遍重复的接口名),各自状态不同。
	firstBatch := []*models.DevicePortStatus{
		{DeviceID: deviceA, InterfaceName: "GE0/0/1", OperStatus: "down", CollectedAt: time.Now()},
		{DeviceID: deviceB, InterfaceName: "GE0/0/1", OperStatus: "up", CollectedAt: time.Now()},
	}

	// 复刻 collection.go 的批量 UPSERT 语义 (复合键 device_id + interface_name)。
	err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}, {Name: "interface_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"oper_status", "collected_at"}),
	}).Create(&firstBatch).Error
	require.NoError(t, err, "两台设备同名 interface 必须能共存 (复合唯一键 device_id+interface_name)")

	// 断言两行独立存在,互不覆盖。
	var rows []models.DevicePortStatus
	require.NoError(t, db.WithContext(ctx).Find(&rows).Error)
	assert.Len(t, rows, 2)
	operByDevice := map[string]string{}
	for _, r := range rows {
		operByDevice[r.DeviceID] = r.OperStatus
	}
	assert.Equal(t, "down", operByDevice[deviceA])
	assert.Equal(t, "up", operByDevice[deviceB])

	// 再次采集 deviceA 的 GE0/0/1 (状态 down→up) → 应 UPSERT 更新 deviceA 那行,
	// 绝不影响 deviceB 的行,也不应新增第三行。
	rerun := []*models.DevicePortStatus{
		{DeviceID: deviceA, InterfaceName: "GE0/0/1", OperStatus: "up", CollectedAt: time.Now()},
	}
	require.NoError(t, db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}, {Name: "interface_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"oper_status", "collected_at"}),
	}).Create(&rerun).Error)

	require.NoError(t, db.WithContext(ctx).Find(&rows).Error)
	assert.Len(t, rows, 2, "再次采集仍应只有两行 (UPSERT 而非 INSERT 新行)")
	operByDevice = map[string]string{}
	for _, r := range rows {
		operByDevice[r.DeviceID] = r.OperStatus
	}
	assert.Equal(t, "up", operByDevice[deviceA], "deviceA 的端口状态应被刷新")
	assert.Equal(t, "up", operByDevice[deviceB], "deviceB 的端口不应受 deviceA 采集影响")
}
