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

// physicalJoin 复刻 service.GetPhysicalDevices 的 Phase 3 v3 逻辑:
//   - workstation_ports CTE: port.id::text = ip.port_id (无 device_id 严格匹配),
//                           effective_device_id = ip.device_id
//   - latest_mac JOIN: mac.norm_iface = wp.norm_iface AND mac.device_id = wp.effective_device_id
//   - ip.device_id 必须在 sys_network_device 存在 (EXISTS 防御)
//
// 返回命中的 MAC 数量。Phase 3 v3 关键修复: MAC JOIN 锚点用 ip.device_id 而非 port.device_id,
// 因为 MAC 是 collector 按 info_point 写入的(mac_collection_service.go:279), 物理真实位置 = ip.device_id。
// port.device_id 是历史脏数据, 不影响 query。
func physicalJoin(db *gorm.DB, workstationID string) (macs []string, ports []string, infoNames []string) {
	// workstation_ports CTE
	var wpMaps []map[string]interface{}
	db.Raw(`
		SELECT port.interface_name AS port,
		       ip.name AS ipname,
		       ip.device_id AS effdev,
		       port.device_id AS portdev
		  FROM ops_info_points ip
		  JOIN sys_device_port_status port
		    ON port.id = ip.port_id
		 WHERE ip.workstation_id = ? AND ip.deleted_at IS NULL AND ip.status = 0
		   AND EXISTS (SELECT 1 FROM sys_network_device WHERE id = ip.device_id)
	`, workstationID).Scan(&wpMaps)

	if len(wpMaps) == 0 {
		return nil, nil, nil
	}

	wpPort, _ := wpMaps[0]["port"].(string)
	wpInfoName, _ := wpMaps[0]["ipname"].(string)
	wpEffDev, _ := wpMaps[0]["effdev"].(string)
	wpNorm := normalizeIface(wpPort)

	// latest_mac JOIN: MAC 锚定 ip.device_id (collector 写入依据)
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
		// MAC 锚定 ip.device_id (effective_device_id), 不要求 port.device_id 一致
		if mDev != wpEffDev {
			continue
		}
		macs = append(macs, mMAC)
		ports = append(ports, wpPort)
		infoNames = append(infoNames, wpInfoName)
	}
	return
}

// TestGetPhysicalDevices_DeviceIdMatch (Phase 3 v2): 验证 device_id 一致场景下
// 非 strict JOIN 仍命中 1 条 MAC (回归守护)
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
		CREATE TABLE sys_network_device (id TEXT PRIMARY KEY, device_name TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_mac_address (id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT)
	`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-2', 'A2', 1, 'u')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device VALUES ('dev-A', 'devA')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points VALUES ('ip-2', 'p2', 'ws-2', 'dev-A', 'port-2', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('port-2', 'dev-A', 'GE1/0/1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address VALUES ('m1', 'dev-A', 'aabb.ccdd.eeff', 'GigabitEthernet 1/0/1')`).Error)

	macs, ports, infoNames := physicalJoin(db, "ws-2")
	require.Equal(t, 1, len(macs), "device_id 一致 + 非 strict JOIN 应命中 1 条")
	require.Equal(t, "aabb.ccdd.eeff", macs[0])
	require.Equal(t, "GE1/0/1", ports[0])
	require.Equal(t, "p2", infoNames[0])
}

// TestGetPhysicalDevices_DeviceIdDriftStillResolves (Phase 3 v2): 验证 device_id 漂移场景下
// 非 strict JOIN 仍能命中 MAC。这是 5F003 用户问题的核心修复:
//   - 数据未治理前: strict JOIN 返回 0 (用户报告现象)
//   - Phase 3 v2 (砍 strict JOIN): MAC 锚定 port.device_id, 仍能命中 3 条
func TestGetPhysicalDevices_DeviceIdDriftStillResolves(t *testing.T) {
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
		CREATE TABLE sys_network_device (id TEXT PRIMARY KEY, device_name TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_mac_address (id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT)
	`).Error)

	// 工位 5F003 + 信息点 5D212
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-5F003', '5F003', 1, 'u')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device VALUES ('dev-correct', '5D212')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device VALUES ('dev-drift', '4F设备')`).Error)
	// ip.device_id = dev-correct (用户填的), 但 port.device_id = dev-drift (历史脏数据)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points VALUES ('ip-5D212', '5D212', 'ws-5F003', 'dev-correct', 'port-1', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('port-1', 'dev-drift', 'GE5/44')`).Error)
	// Phase 3 v3: MAC 锚定 ip.device_id (dev-correct), 模拟 collector 按 info_point 写入的真实路径
	macs := []string{"9c7b.ef2f.2d5e", "9c7b.ef2f.31b8", "f88c.2187.6d7a"}
	for _, mac := range macs {
		require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address VALUES (?, 'dev-correct', ?, 'GigabitEthernet 5/44')`,
			"mac-"+mac, mac).Error)
	}

	macs2, _, _ := physicalJoin(db, "ws-5F003")
	require.Equal(t, 3, len(macs2), "device_id 漂移 + Phase 3 v3 锚点 ip.device_id 应命中 3 条 MAC")
}

// TestGetPhysicalDevices_DataGoverned (Phase 3 v2): 模拟数据治理后场景
// (port.device_id == ip.device_id) + 非 strict JOIN,验证仍能命中 3 条 MAC
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
		CREATE TABLE sys_network_device (id TEXT PRIMARY KEY, device_name TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_mac_address (id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT)
	`).Error)

	// 治理后: ip.device_id = port.device_id = dev-correct
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-5F003', '5F003', 1, 'u')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device VALUES ('dev-correct', '5D212')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points VALUES ('ip-5D212', '5D212', 'ws-5F003', 'dev-correct', 'port-1', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('port-1', 'dev-correct', 'GE5/44')`).Error)
	macs := []string{"9c7b.ef2f.2d5e", "9c7b.ef2f.31b8", "f88c.2187.6d7a"}
	for _, mac := range macs {
		require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address VALUES (?, 'dev-correct', ?, 'GigabitEthernet 5/44')`,
			"mac-"+mac, mac).Error)
	}

	macs2, ports, infoNames := physicalJoin(db, "ws-5F003")
	require.Equal(t, 3, len(macs2), "治理后 + 非 strict JOIN 应命中 3 条 MAC")
	macsFound := map[string]bool{}
	for i, m := range macs2 {
		macsFound[m] = true
		require.Equal(t, "GE5/44", ports[i])
		require.Equal(t, "5D212", infoNames[i])
	}
	require.True(t, macsFound["9c7b.ef2f.2d5e"])
	require.True(t, macsFound["9c7b.ef2f.31b8"])
	require.True(t, macsFound["f88c.2187.6d7a"])
}

// TestGetPhysicalDevices_InvalidDeviceId (Phase 3 v2 防御):
// 当 ip.device_id 指向不存在的 sys_network_device (orphan), EXISTS 过滤生效, 返回 0 条
func TestGetPhysicalDevices_InvalidDeviceId(t *testing.T) {
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
		CREATE TABLE sys_network_device (id TEXT PRIMARY KEY, device_name TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_device_mac_address (id TEXT PRIMARY KEY, device_id TEXT, mac_address TEXT, interface_name TEXT)
	`).Error)

	// 工位存在, 但 ip.device_id = 'orphan-dev' (不在 sys_network_device)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-3', 'A3', 1, 'u')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_network_device VALUES ('dev-A', 'devA')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_info_points VALUES ('ip-3', 'p3', 'ws-3', 'orphan-dev', 'port-3', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_port_status VALUES ('port-3', 'dev-A', 'GE1/1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_device_mac_address VALUES ('m1', 'dev-A', 'aabb.ccdd.eeff', 'GE1/1')`).Error)

	macs, _, _ := physicalJoin(db, "ws-3")
	require.Equal(t, 0, len(macs), "ip.device_id 是 orphan 时 EXISTS 过滤应返回 0 条")
}