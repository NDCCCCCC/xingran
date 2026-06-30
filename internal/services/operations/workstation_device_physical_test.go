package operations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// normalizeIface 复刻 service.GetPhysicalDevices 中的 REGEXP_REPLACE 归一化逻辑。
// 规则: trim 前缀 gigabitethernet|gigabitether|ge|gi → 'ge';空格移除;大小写不敏感。
//
// 真实 CTE 用 PG 的 REGEXP_REPLACE 完成归一化(见 workstation_device_service.go)。
// 本测试在 SQLite 上手动预归一化以兼容 SQLite(无 REGEXP_REPLACE)。
func normalizeIface(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	re := regexp.MustCompile(`^(gigabitethernet|gigabitether|ge|gi)`)
	s = re.ReplaceAllString(s, "ge")
	return s
}

// strictJoin 复刻 service.GetPhysicalDevices 的 Phase 3 strict JOIN 逻辑:
//   - workstation_ports CTE: port.id::text = ip.port_id AND port.device_id::text = ip.device_id::text
//   - latest_mac JOIN: mac.norm_iface = wp.norm_iface AND mac.device_id::text = wp.device_id::text
//
// 返回命中的 MAC 数量。生产代码已改回 strict JOIN(migration_182 数据治理 + migration_183 FK 后)。
func strictJoin(db *gorm.DB, workstationID string) (macs []string, ports []string, infoNames []string) {
	// workstation_ports CTE
	var wpMaps []map[string]interface{}
	db.Raw(`
		SELECT port.interface_name AS port,
		       ip.name AS ipname,
		       port.device_id AS portdev
		  FROM ops_info_points ip
		  JOIN sys_device_port_status port
		    ON port.id = ip.port_id
		   AND port.device_id = ip.device_id
		 WHERE ip.workstation_id = ? AND ip.deleted_at IS NULL AND ip.status = 0
	`, workstationID).Scan(&wpMaps)

	if len(wpMaps) == 0 {
		return nil, nil, nil
	}

	wpPort, _ := wpMaps[0]["port"].(string)
	wpInfoName, _ := wpMaps[0]["ipname"].(string)
	wpPortDev, _ := wpMaps[0]["portdev"].(string)
	wpNorm := normalizeIface(wpPort)

	// latest_mac JOIN
	var macMaps []map[string]interface{}
	db.Raw(`SELECT mac_address AS mac, interface_name AS port, device_id AS dev
	          FROM sys_device_mac_address`).Scan(&macMaps)

	for _, m := range macMaps {
		mPort, _ := m["port"].(string)
		mMAC, _ := m["mac"].(string)
		mDev, _ := m["dev"].(string)
		if normalizeIface(mPort) != wpNorm {
			continue
		}
		if mDev != wpPortDev {
			continue
		}
		macs = append(macs, mMAC)
		ports = append(ports, wpPort)
		infoNames = append(infoNames, wpInfoName)
	}
	return
}

// TestGetPhysicalDevices_DeviceIdMatch (Phase 3 之后): 验证 strict JOIN 在 device_id 一致场景下
// 命中 1 条 MAC
func TestGetPhysicalDevices_DeviceIdMatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, status INTEGER DEFAULT 0, user_id TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_info_points (id TEXT PRIMARY KEY, name TEXT, workstation_id TEXT, device_id TEXT, port_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_port_status (id TEXT PRIMARY KEY, device_id TEXT, interface_name TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_mac_address (id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT)
	`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-2', 'A2', 1, 'u')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points VALUES ('ip-2', 'p2', 'ws-2', 'dev-A', 'port-2', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('port-2', 'dev-A', 'GE1/0/1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address VALUES ('m1', 'dev-A', 'aabb.ccdd.eeff', 'GigabitEthernet 1/0/1')`).Error)

	macs, ports, infoNames := strictJoin(db, "ws-2")
	require.Equal(t, 1, len(macs), "device_id 一致 + strict JOIN 应命中 1 条")
	require.Equal(t, "aabb.ccdd.eeff", macs[0])
	require.Equal(t, "GE1/0/1", ports[0])
	require.Equal(t, "p2", infoNames[0])
}

// TestGetPhysicalDevices_DeviceIdDrift (Phase 3 之后): 验证 strict JOIN 在 device_id 漂移场景下
// 返回 0 条 MAC(因为 port_status.device_id ≠ ip.device_id 时 workstation_ports CTE 即被 JOIN 条件过滤掉)
//
// 这是 Phase 3 改回 strict JOIN 后的预期行为:
//   - 数据未治理前:0 命中(用户报告的 5F003 现象)
//   - migration_182 数据治理后:port.device_id == ip.device_id,strict JOIN 可正常命中
//   - 治理后仍漂移:立即触发 migration_183 FK 约束 violation(orphan)或被 Phase 4 定时对账告警
func TestGetPhysicalDevices_DeviceIdDrift(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, status INTEGER DEFAULT 0, user_id TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_info_points (id TEXT PRIMARY KEY, name TEXT, workstation_id TEXT, device_id TEXT, port_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_port_status (id TEXT PRIMARY KEY, device_id TEXT, interface_name TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_mac_address (id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT)
	`).Error)

	// Seed: 工位 5F003 + 信息点 5D212
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-5F003', '5F003', 1, 'u')`).Error)
	// Seed: ip.device_id = dev-correct (用户手动绑定的"真"设备)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points VALUES ('ip-5D212', '5D212', 'ws-5F003', 'dev-correct', 'port-1', 0, NULL)`).Error)
	// Seed: port.device_id = dev-drift (历史漂移到另一台设备)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('port-1', 'dev-drift', 'GE5/44')`).Error)
	// Seed: MAC 数据落在 dev-correct(采集器按 info_point.device_id 写入)
	macs := []string{"9c7b.ef2f.2d5e", "9c7b.ef2f.31b8", "f88c.2187.6d7a"}
	for _, mac := range macs {
		require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address VALUES (?, 'dev-correct', ?, 'GigabitEthernet 5/44')`,
			"mac-"+mac, mac).Error)
	}

	macs2, _, _ := strictJoin(db, "ws-5F003")
	require.Equal(t, 0, len(macs2), "device_id 漂移 + strict JOIN 应返回 0 条(MAC 在 dev-correct,但 CTE 在 port.device_id 漂移时即被过滤)")
}

// TestGetPhysicalDevices_DataGoverned (Phase 3 之后): 模拟 migration_182 数据治理后
// 场景(原漂移现在被修复) + strict JOIN,验证仍能命中 3 条 MAC
//
// 模拟场景:
//   - 治理前:ip.device_id = dev-correct, port.device_id = dev-drift
//   - 治理后(migration_182):port.device_id 已 UPDATE 为 dev-correct
//   - 治理后 + strict JOIN: 命中 3 条 MAC
func TestGetPhysicalDevices_DataGoverned(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, status INTEGER DEFAULT 0, user_id TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_info_points (id TEXT PRIMARY KEY, name TEXT, workstation_id TEXT, device_id TEXT, port_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_port_status (id TEXT PRIMARY KEY, device_id TEXT, interface_name TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_mac_address (id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT)
	`).Error)

	// Seed: 模拟 migration_182 治理后状态
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-5F003', '5F003', 1, 'u')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points VALUES ('ip-5D212', '5D212', 'ws-5F003', 'dev-correct', 'port-1', 0, NULL)`).Error)
	// 治理后:port_status.device_id 已对齐到 ip.device_id
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('port-1', 'dev-correct', 'GE5/44')`).Error)
	macs := []string{"9c7b.ef2f.2d5e", "9c7b.ef2f.31b8", "f88c.2187.6d7a"}
	for _, mac := range macs {
		require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address VALUES (?, 'dev-correct', ?, 'GigabitEthernet 5/44')`,
			"mac-"+mac, mac).Error)
	}

	macs2, ports, infoNames := strictJoin(db, "ws-5F003")
	require.Equal(t, 3, len(macs2), "治理后 + strict JOIN 应命中 3 条 MAC(用户报告一致)")
	macsFound := map[string]bool{}
	for i, m := range macs2 {
		macsFound[m] = true
		require.Equal(t, "GE5/44", ports[i], "port_name 应保留原值 GE5/44")
		require.Equal(t, "5D212", infoNames[i])
	}
	require.True(t, macsFound["9c7b.ef2f.2d5e"])
	require.True(t, macsFound["9c7b.ef2f.31b8"])
	require.True(t, macsFound["f88c.2187.6d7a"])
}