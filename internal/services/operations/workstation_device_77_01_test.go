package operations

// Phase 77 Plan 01: workstation_device_service 全量 sqlite 直测。
//
// 覆盖策略(RESEARCH §77-01 实证:该 service 唯一依赖 *gorm.DB,零 LDAP/AD 调用,
// "AD 设备"全部读 sys_ad_computer/sys_ad_user 预落库行):
//   - 7 表手动 CREATE TABLE fixture(physical_test.go:86-103 先例),列集精简到被测函数实际引用列
//   - 种子行照 analog 裸 INSERT INTO 风格
//   - GetPhysicalDevices* 仅覆盖 sqlite 可达前段:越过全部早退后 PG-only SQL
//     (DISTINCT ON/REGEXP_REPLACE/::text) 在 sqlite 必报错属预期 —
//     「报错即证明越过早退」手法照抄 physical_test.go:249-284(P-77-1:勿为覆盖率改 SQL)
//   - 状态/来源断言一律引用 internal/models 具名常量,禁裸 0/1(status_constants_test.go 锁值)
//
// 本文件 Task 1 = 查询链(GetADDevices 家族 + GetPhysicalDevices 前段 + 纯函数边角);
// Task 2 = 写链(SyncFrom*/Add/Update/Delete/SetPrimary*/mergeBySerial 三态)。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// wsd77 常量:测试种子行固定 ID(工位/用户用合法 UUID,其余表无 UUID 校验要求)
const (
	wsd77WS1     = "10000000-0000-0000-0000-000000000001"
	wsd77WS2     = "10000000-0000-0000-0000-000000000002"
	wsd77WS3     = "10000000-0000-0000-0000-000000000003"
	wsd77WSNoUsr = "10000000-0000-0000-0000-000000000004"
	wsd77WSMiss  = "10000000-0000-0000-0000-000000000404" // 不存在的工位 UUID
	wsd77User1   = "20000000-0000-0000-0000-000000000001"
	wsd77User2   = "20000000-0000-0000-0000-000000000002"
	wsd77User3   = "20000000-0000-0000-0000-000000000003"
	wsd77Dept1   = "40000000-0000-0000-0000-000000000001"
	wsd77Dept2   = "40000000-0000-0000-0000-000000000002"
	wsd77ADComp1 = "30000000-0000-0000-0000-000000000001"
	wsd77ADComp2 = "30000000-0000-0000-0000-000000000002"
	wsd77ADComp3 = "30000000-0000-0000-0000-000000000003"
	wsd77Asset1  = "50000000-0000-0000-0000-000000000001"
	wsd77Asset2  = "50000000-0000-0000-0000-000000000002"
	wsd77Asset3  = "50000000-0000-0000-0000-000000000003"
	wsd77Asset4  = "50000000-0000-0000-0000-000000000004"
	wsd77DN1     = "CN=zhangsan,OU=users,DC=corp"
	wsd77DN2     = "CN=lisi,OU=users,DC=corp"
)

// setupWSD77DB 建 7 表 sqlite :memory: fixture。
//
// 列集 = 被测函数实际引用列(RESEARCH §77-01 7 表清单):
//   sys_workstation / sys_user / sys_ad_user / sys_ad_computer
//   (managed_by, original_description, serial_number) / ops_asset
//   (devicesn, nowuser_name, deptname, mac1, machine_ip) / sys_dept / ops_workstation_device
//
// ops_workstation_device 列集与 models.WorkstationDevice 落库字段一一对应
// (Confidence/HistoryLastSeen 是 gorm:"-" 不落库)。
func setupWSD77DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY,
			workstation_name TEXT,
			status INTEGER DEFAULT 0,
			user_id TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			nickname TEXT,
			dept_id TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			user_dn TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_computer (
			id TEXT PRIMARY KEY,
			computer_name TEXT,
			serial_number TEXT,
			mac_address TEXT,
			ip_address TEXT,
			operating_system TEXT,
			managed_by TEXT,
			original_description TEXT,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			device_model_name TEXT,
			device_type_name TEXT,
			mac1 TEXT,
			mac2 TEXT,
			nowuser_name TEXT,
			deptname TEXT,
			machine_ip TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_workstation_device (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0,
			workstation_id TEXT,
			asset_id TEXT,
			ad_computer_id TEXT,
			device_source TEXT,
			device_serial TEXT,
			device_name TEXT,
			device_model TEXT,
			device_type TEXT,
			mac_address TEXT,
			ip_address TEXT,
			responsible_user TEXT,
			responsible_user_id TEXT,
			status INTEGER DEFAULT 0,
			is_primary BOOLEAN DEFAULT 0,
			priority INTEGER DEFAULT 0,
			description TEXT
		)
	`).Error)
	return db
}

func wsd77Exec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	require.NoError(t, db.Exec(sql, args...).Error)
}

// seedWSD77Workstation 插入工位; userID 为空写 NULL。
func seedWSD77Workstation(t *testing.T, db *gorm.DB, id, name, userID string) {
	t.Helper()
	if userID == "" {
		wsd77Exec(t, db,
			`INSERT INTO sys_workstation (id, workstation_name, status, user_id, deleted_at) VALUES (?, ?, 0, NULL, NULL)`,
			id, name)
		return
	}
	wsd77Exec(t, db,
		`INSERT INTO sys_workstation (id, workstation_name, status, user_id, deleted_at) VALUES (?, ?, 0, ?, NULL)`,
		id, name, userID)
}

// seedWSD77User 插入系统用户; nickname/deptID 为空写 NULL。
func seedWSD77User(t *testing.T, db *gorm.DB, id, username, nickname, deptID string) {
	t.Helper()
	wsd77Exec(t, db,
		`INSERT INTO sys_user (id, username, nickname, dept_id, deleted_at) VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULL)`,
		id, username, nickname, deptID)
}

func seedWSD77ADUser(t *testing.T, db *gorm.DB, id, username, userDN string) {
	t.Helper()
	wsd77Exec(t, db,
		`INSERT INTO sys_ad_user (id, username, user_dn, deleted_at) VALUES (?, ?, ?, NULL)`,
		id, username, userDN)
}

func seedWSD77ADComputer(t *testing.T, db *gorm.DB, id, name, sn, mac, ip, os, managedBy, desc string) {
	t.Helper()
	wsd77Exec(t, db,
		`INSERT INTO sys_ad_computer (id, computer_name, serial_number, mac_address, ip_address, operating_system, managed_by, original_description, updated_at, deleted_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL)`,
		id, name, sn, mac, ip, os, managedBy, desc)
}

func seedWSD77Asset(t *testing.T, db *gorm.DB, id, sn, model, typ, mac1, nowuser, deptname, ip string) {
	t.Helper()
	wsd77Exec(t, db,
		`INSERT INTO ops_asset (id, devicesn, device_model_name, device_type_name, mac1, mac2, nowuser_name, deptname, machine_ip, deleted_at)
		 VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULL, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULL)`,
		id, sn, model, typ, mac1, nowuser, deptname, ip)
}

// seedWSD77DeviceRow 直插 ops_workstation_device 行(写链测试的旧数据预置)。
func seedWSD77DeviceRow(t *testing.T, db *gorm.DB, id, wsID, source, sn string, isPrimary bool) {
	t.Helper()
	wsd77Exec(t, db,
		`INSERT INTO ops_workstation_device (id, workstation_id, device_source, device_serial, is_primary, status, deleted_at)
		 VALUES (?, ?, ?, ?, ?, ?, NULL)`,
		id, wsID, source, sn, isPrimary, models.WorkstationDeviceStatusNormal)
}

// wsd77LiveDevices 直查落库的未软删 ops_workstation_device 行(按 device_serial 索引)。
func wsd77LiveDevices(t *testing.T, db *gorm.DB, wsID string) map[string]*models.WorkstationDevice {
	t.Helper()
	var rows []models.WorkstationDevice
	require.NoError(t, db.
		Where("workstation_id = ? AND deleted_at IS NULL", wsID).
		Find(&rows).Error)
	bySN := make(map[string]*models.WorkstationDevice, len(rows))
	for i := range rows {
		r := &rows[i]
		if r.DeviceSerial != nil {
			bySN[*r.DeviceSerial] = r
		} else {
			bySN["<nil-sn:"+r.ID[:8]+">"] = r
		}
	}
	return bySN
}

// ===========================================================================
// Task 1: 查询链测试
// ===========================================================================

// TestWSD77_GetADDevices 工位→sys_user→sys_ad_user→sys_ad_computer 四表链端到端。
func TestWSD77_GetADDevices(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", wsd77User1)
	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", "")
	seedWSD77ADUser(t, db, "60000000-0000-0000-0000-000000000001", "zhangsan", wsd77DN1)
	// 命中策略1: managed_by = UserDN
	seedWSD77ADComputer(t, db, wsd77ADComp1, "AD-PC-MANAGED", "SN-AD-1",
		"AA:BB:CC:DD:EE:01", "10.1.1.1", "Windows 11", wsd77DN1, "")
	// 命中策略2: original_description LIKE '%|zhangsan|%'
	seedWSD77ADComputer(t, db, wsd77ADComp2, "AD-PC-DESC", "SN-AD-2",
		"AA:BB:CC:DD:EE:02", "10.1.1.2", "Windows 10", "CN=other,DC=corp", "工位机|zhangsan|主用")

	devices, err := svc.GetADDevices(ctx, wsd77WS1)
	require.NoError(t, err)
	require.Len(t, devices, 2, "managed_by 命中 + original_description 命中应各返回一台")

	bySN := make(map[string]*models.WorkstationDevice, len(devices))
	for _, d := range devices {
		require.Equal(t, wsd77WS1, d.WorkstationID)
		require.Equal(t, models.DeviceSourceAD, d.DeviceSource)
		require.Equal(t, models.WorkstationDeviceStatusNormal, d.Status)
		require.False(t, d.IsPrimary)
		require.NotNil(t, d.DeviceSerial)
		bySN[*d.DeviceSerial] = d
	}
	require.Contains(t, bySN, "SN-AD-1")
	require.Contains(t, bySN, "SN-AD-2")

	m1 := bySN["SN-AD-1"]
	require.NotNil(t, m1.ADComputerID)
	assert.Equal(t, wsd77ADComp1, *m1.ADComputerID)
	require.NotNil(t, m1.DeviceName)
	assert.Equal(t, "AD-PC-MANAGED", *m1.DeviceName)
	require.NotNil(t, m1.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:01", *m1.MACAddress)
	// quirk-77-1 回归: GetADDevices 转换此前漏掉 IPAddress,导致接口注释 :58 的
	// 「ipAddress 优先取 AD」合并规则在 mergeBySerial 链路永不可达(批量实现 :1574
	// 一直有 IPAddress: &ip)。修复后单工位查询与批量行为一致。
	require.NotNil(t, m1.IPAddress)
	assert.Equal(t, "10.1.1.1", *m1.IPAddress)

	// 实时查询不落库
	var cnt int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM ops_workstation_device`).Scan(&cnt).Error)
	assert.Zero(t, cnt, "GetADDevices 是实时查询,不应写 ops_workstation_device")

	// 分支: 工位未绑定用户 → 空切片不报错
	seedWSD77Workstation(t, db, wsd77WSNoUsr, "3f999", "")
	empty, err := svc.GetADDevices(ctx, wsd77WSNoUsr)
	require.NoError(t, err)
	require.Empty(t, empty)

	// 分支: 工位不存在
	_, err = svc.GetADDevices(ctx, wsd77WSMiss)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位不存在")

	// 分支: 空工位 ID
	_, err = svc.GetADDevices(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
}

// TestWSD77_GetAssetDevices 工位→sys_user→ops_asset 链,断言资产字段透传。
func TestWSD77_GetAssetDevices(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", wsd77User1)
	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", "")
	seedWSD77Asset(t, db, wsd77Asset1, "SN-ASSET-1", "ThinkPad T14", "笔记本",
		"AA:BB:CC:DD:EE:11", "张三", "研发部", "10.2.2.1")

	devices, err := svc.GetAssetDevices(ctx, wsd77WS1)
	require.NoError(t, err)
	require.Len(t, devices, 1)

	d := devices[0]
	require.Equal(t, wsd77WS1, d.WorkstationID)
	require.Equal(t, models.DeviceSourceAsset, d.DeviceSource)
	require.Equal(t, models.WorkstationDeviceStatusNormal, d.Status)
	require.False(t, d.IsPrimary)
	require.NotNil(t, d.AssetID)
	assert.Equal(t, wsd77Asset1, *d.AssetID)
	require.NotNil(t, d.DeviceSerial)
	assert.Equal(t, "SN-ASSET-1", *d.DeviceSerial) // devicesn 透传
	require.NotNil(t, d.DeviceModel)
	assert.Equal(t, "ThinkPad T14", *d.DeviceModel)
	require.NotNil(t, d.DeviceType)
	assert.Equal(t, "笔记本", *d.DeviceType)
	require.NotNil(t, d.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:11", *d.MACAddress)
	require.NotNil(t, d.ResponsibleUser)
	assert.Equal(t, "张三", *d.ResponsibleUser) // nowuser_name 透传

	// 分支: 工位未绑定用户 → 空切片不报错
	seedWSD77Workstation(t, db, wsd77WSNoUsr, "3f999", "")
	empty, err := svc.GetAssetDevices(ctx, wsd77WSNoUsr)
	require.NoError(t, err)
	require.Empty(t, empty)

	// 分支: 工位绑定的用户在 sys_user 不存在 → 查询用户信息失败
	seedWSD77Workstation(t, db, wsd77WS2, "3f131", "20000000-0000-0000-0000-00000000999")
	_, err = svc.GetAssetDevices(ctx, wsd77WS2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户信息失败")

	// 分支: 工位不存在 / 空 ID
	_, err = svc.GetAssetDevices(ctx, wsd77WSMiss)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位不存在")
	_, err = svc.GetAssetDevices(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
}

// TestWSD77_GetADDevicesByUser 双命中策略 + 用户不存在/无 AD 记录两处 warn-and-empty 分支。
func TestWSD77_GetADDevicesByUser(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", "")
	seedWSD77ADUser(t, db, "60000000-0000-0000-0000-000000000001", "zhangsan", wsd77DN1)
	seedWSD77ADComputer(t, db, wsd77ADComp1, "AD-PC-MANAGED", "SN-AD-1",
		"AA:BB:CC:DD:EE:01", "10.1.1.1", "Windows 11", wsd77DN1, "")
	seedWSD77ADComputer(t, db, wsd77ADComp2, "AD-PC-DESC", "SN-AD-2",
		"AA:BB:CC:DD:EE:02", "10.1.1.2", "Windows 10", "CN=other,DC=corp", "工位机|zhangsan|主用")

	matches, err := svc.GetADDevicesByUser(ctx, wsd77User1)
	require.NoError(t, err)
	require.Len(t, matches, 2, "managed_by ∪ last_logged_user 双策略应返回 2 台")

	bySN := make(map[string]*ADDeviceMatch, len(matches))
	for _, m := range matches {
		bySN[m.DeviceSerial] = m
	}
	m1 := bySN["SN-AD-1"]
	require.NotNil(t, m1, "managed_by = UserDN 应命中 SN-AD-1")
	assert.Equal(t, wsd77ADComp1, m1.ADComputerID)
	assert.Equal(t, "AD-PC-MANAGED", m1.DeviceName)
	assert.Equal(t, "AA:BB:CC:DD:EE:01", m1.MACAddress)
	assert.Equal(t, "10.1.1.1", m1.IPAddress)
	assert.Equal(t, "Windows 11", m1.OperatingSystem)
	m2 := bySN["SN-AD-2"]
	require.NotNil(t, m2, "original_description LIKE '%|zhangsan|%' 应命中 SN-AD-2")
	assert.Equal(t, wsd77ADComp2, m2.ADComputerID)

	// 分支: 用户在 sys_user 不存在 → 空切片不报错(:608 warn-and-empty)
	empty, err := svc.GetADDevicesByUser(ctx, "20000000-0000-0000-0000-00000000888")
	require.NoError(t, err)
	require.Empty(t, empty)

	// 分支: 用户存在但无 sys_ad_user 记录 → 空切片不报错(:622 warn-and-empty)
	seedWSD77User(t, db, wsd77User2, "wangwu", "王五", "")
	empty2, err := svc.GetADDevicesByUser(ctx, wsd77User2)
	require.NoError(t, err)
	require.Empty(t, empty2)

	// 分支: 空 userID
	_, err = svc.GetADDevicesByUser(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
}

// TestWSD77_GetAssetDevicesByUser 同名+部门精确匹配两策略:
// 策略1 姓名匹配;策略2 同名多台时按 (姓名, 部门) 精确过滤,过滤无果保留策略1 全量。
func TestWSD77_GetAssetDevicesByUser(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	wsd77Exec(t, db, `INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES (?, ?, NULL)`, wsd77Dept1, "研发部")
	wsd77Exec(t, db, `INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES (?, ?, NULL)`, wsd77Dept2, "综合部")

	// 场景A: 两个同名 asset + 用户带 dept → 精确匹配过滤生效
	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", wsd77Dept1)
	seedWSD77Asset(t, db, wsd77Asset1, "SN-A1", "ThinkPad T14", "笔记本", "AA:11", "张三", "研发部", "10.0.0.1")
	seedWSD77Asset(t, db, wsd77Asset2, "SN-A2", "OptiPlex 7090", "台式机", "AA:22", "张三", "市场部", "10.0.0.2")

	matches, err := svc.GetAssetDevicesByUser(ctx, wsd77User1, "zhangsan", "张三")
	require.NoError(t, err)
	require.Len(t, matches, 1, "同名 2 台 + 部门精确匹配应过滤为研发部 1 台")
	assert.Equal(t, "SN-A1", matches[0].DeviceSN)
	assert.Equal(t, wsd77Asset1, matches[0].AssetID)
	require.NotNil(t, matches[0].DeviceModel)
	assert.Equal(t, "ThinkPad T14", *matches[0].DeviceModel)
	require.NotNil(t, matches[0].MACAddress)
	assert.Equal(t, "AA:11", *matches[0].MACAddress)
	require.NotNil(t, matches[0].ResponsibleUser)
	assert.Equal(t, "张三", *matches[0].ResponsibleUser)

	// 场景B: dept 不匹配(用户在综合部, asset 都在市场部)→ 保留策略 1 全量
	seedWSD77User(t, db, wsd77User2, "lisi", "李四", wsd77Dept2)
	seedWSD77Asset(t, db, wsd77Asset3, "SN-A3", "MacBook Pro", "笔记本", "AA:33", "李四", "市场部", "10.0.0.3")
	seedWSD77Asset(t, db, wsd77Asset4, "SN-A4", "Mac mini", "迷你主机", "AA:44", "李四", "市场部", "10.0.0.4")

	matches2, err := svc.GetAssetDevicesByUser(ctx, wsd77User2, "lisi", "李四")
	require.NoError(t, err)
	require.Len(t, matches2, 2, "部门精确匹配无结果时应保留策略1 的全部 2 台")

	// 分支: nickname 为空 → warn-and-empty 空切片不报错
	empty, err := svc.GetAssetDevicesByUser(ctx, wsd77User1, "zhangsan", "")
	require.NoError(t, err)
	require.Empty(t, empty)

	// 分支: 用户不存在 → 查询用户部门信息失败
	_, err = svc.GetAssetDevicesByUser(ctx, "20000000-0000-0000-0000-00000000777", "x", "y")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户部门信息失败")
}

// TestWSD77_GetADDevicesByWorkstations 批量链: managed_by 命中 + original_description
// 匹配分支 + owner 不可解析 continue 分支 + 空入参 + 表缺失错误分支。
func TestWSD77_GetADDevicesByWorkstations(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", wsd77User1)
	seedWSD77Workstation(t, db, wsd77WS2, "3f131", wsd77User2)
	seedWSD77Workstation(t, db, wsd77WS3, "3f132", "") // 无用户
	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", "")
	seedWSD77User(t, db, wsd77User2, "lisi", "李四", "")
	seedWSD77ADUser(t, db, "60000000-0000-0000-0000-000000000001", "zhangsan", wsd77DN1)
	seedWSD77ADUser(t, db, "60000000-0000-0000-0000-000000000002", "lisi", wsd77DN2)
	// managed_by 命中 → 归属 W1
	seedWSD77ADComputer(t, db, wsd77ADComp1, "AD-PC-Z", "SN-B1",
		"AA:BB:CC:DD:EE:01", "10.1.1.1", "Windows 11", wsd77DN1, "")
	// original_description 匹配分支: managed_by 指向无关 DN,靠 |lisi| 命中 → 归属 W2
	seedWSD77ADComputer(t, db, wsd77ADComp2, "AD-PC-L", "SN-B2",
		"AA:BB:CC:DD:EE:02", "10.1.1.2", "Windows 10", "CN=unrelated,DC=corp", "工位机|lisi|主用")
	// owner 不可解析(managed_by 不在 DN 表,desc 无任何 username 标记)→ 不归属任何工位
	seedWSD77ADComputer(t, db, wsd77ADComp3, "AD-PC-X", "SN-B3",
		"AA:BB:CC:DD:EE:03", "10.1.1.3", "Windows 10", "CN=ghost,DC=corp", "no markers here")

	result, err := svc.GetADDevicesByWorkstations(ctx, []string{wsd77WS1, wsd77WS2, wsd77WS3})
	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Len(t, result[wsd77WS1], 1, "W1 应命中 managed_by 的 SN-B1")
	require.Len(t, result[wsd77WS2], 1, "W2 应命中 original_description 的 SN-B2")
	require.Empty(t, result[wsd77WS3], "无用户工位应为空切片")

	d1 := result[wsd77WS1][0]
	require.Equal(t, models.DeviceSourceAD, d1.DeviceSource)
	require.Equal(t, models.WorkstationDeviceStatusNormal, d1.Status)
	require.NotNil(t, d1.DeviceSerial)
	assert.Equal(t, "SN-B1", *d1.DeviceSerial)
	require.NotNil(t, d1.ADComputerID)
	assert.Equal(t, wsd77ADComp1, *d1.ADComputerID)
	require.NotNil(t, d1.IPAddress)
	assert.Equal(t, "10.1.1.1", *d1.IPAddress)

	d2 := result[wsd77WS2][0]
	require.NotNil(t, d2.DeviceSerial)
	assert.Equal(t, "SN-B2", *d2.DeviceSerial, "desc 匹配分支应把 |lisi| 电脑归属 W2")

	// 分支: 空入参 → 空 map 不报错
	emptyResult, err := svc.GetADDevicesByWorkstations(ctx, []string{})
	require.NoError(t, err)
	require.Empty(t, emptyResult)

	// 错误分支: sys_ad_user 表缺失 → 批量查询 sys_ad_user 失败
	wsd77Exec(t, db, `DROP TABLE sys_ad_user`)
	_, err = svc.GetADDevicesByWorkstations(ctx, []string{wsd77WS1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "批量查询 sys_ad_user 失败")
}

// TestWSD77_GetAssetDevicesByWorkstations 批量资产链 + 空昵称早退 + 表缺失错误分支。
func TestWSD77_GetAssetDevicesByWorkstations(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", wsd77User1)
	seedWSD77Workstation(t, db, wsd77WS2, "3f131", "")             // 无用户 → 空切片
	seedWSD77Workstation(t, db, wsd77WS3, "3f132", wsd77User3)     // 用户昵称为空 → 早退
	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", "")
	seedWSD77User(t, db, wsd77User3, "wangwu", "", "")             // nickname NULL
	seedWSD77Asset(t, db, wsd77Asset1, "SN-C1", "ThinkPad T14", "笔记本", "AA:11", "张三", "研发部", "10.0.0.1")
	seedWSD77Asset(t, db, wsd77Asset2, "SN-C2", "OptiPlex 7090", "台式机", "AA:22", "张三", "研发部", "10.0.0.2")

	result, err := svc.GetAssetDevicesByWorkstations(ctx, []string{wsd77WS1, wsd77WS2, wsd77WS3})
	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Len(t, result[wsd77WS1], 2, "张三的 2 台资产应归属 W1")
	require.Empty(t, result[wsd77WS2])
	require.Empty(t, result[wsd77WS3], "用户昵称为空时走早退,结果为空切片")

	d1 := result[wsd77WS1][0]
	require.Equal(t, models.DeviceSourceAsset, d1.DeviceSource)
	require.Equal(t, models.WorkstationDeviceStatusNormal, d1.Status)
	require.NotNil(t, d1.ResponsibleUser)
	assert.Equal(t, "张三", *d1.ResponsibleUser)
	bySN := map[string]*models.WorkstationDevice{}
	for _, d := range result[wsd77WS1] {
		require.NotNil(t, d.DeviceSerial)
		require.NotNil(t, d.AssetID)
		bySN[*d.DeviceSerial] = d
	}
	require.Contains(t, bySN, "SN-C1")
	require.Contains(t, bySN, "SN-C2")
	require.NotNil(t, bySN["SN-C1"].DeviceModel)
	assert.Equal(t, "ThinkPad T14", *bySN["SN-C1"].DeviceModel)

	// 分支: 全部工位的用户昵称为空 → nicknames 空 → 整体早退(空 map 值均为空切片)
	only3, err := svc.GetAssetDevicesByWorkstations(ctx, []string{wsd77WS3})
	require.NoError(t, err)
	require.Len(t, only3, 1)
	require.Empty(t, only3[wsd77WS3])

	// 分支: 空入参 → 空 map 不报错
	emptyResult, err := svc.GetAssetDevicesByWorkstations(ctx, []string{})
	require.NoError(t, err)
	require.Empty(t, emptyResult)

	// 错误分支: ops_asset 表缺失 → 批量查询 ops_asset 失败
	wsd77Exec(t, db, `DROP TABLE ops_asset`)
	_, err = svc.GetAssetDevicesByWorkstations(ctx, []string{wsd77WS1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "批量查询 ops_asset 失败")
}

// TestWSD77_GetPhysicalDevices_FrontSegment 仅 sqlite 可达前段:
// ParamMissing / ParamInvalid / 工位不存在三分支正常断言;越过后 PG-only SQL
// (DISTINCT ON/REGEXP_REPLACE/::text) 在 sqlite 必报错 — 报错即证明已越过全部早退
// (physical_test.go:249-284 手法;P-77-1:勿为覆盖率改 SQL,D-03 无据)。
func TestWSD77_GetPhysicalDevices_FrontSegment(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	// 分支: 空 ID
	_, err := svc.GetPhysicalDevices(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")

	// 分支: 非 UUID
	_, err = svc.GetPhysicalDevices(ctx, "not-a-uuid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数无效")

	// 分支: 合法 UUID 但工位不存在
	_, err = svc.GetPhysicalDevices(ctx, wsd77WSMiss)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位不存在")

	// 越过全部早退: 工位存在(甚至不绑定用户)后进入物理链路 CTE,
	// PG-only SQL 在 sqlite 报错属预期 — 不含对 DISTINCT ON 结果行的断言。
	seedWSD77Workstation(t, db, wsd77WS1, "3f130", "")
	devices, err := svc.GetPhysicalDevices(ctx, wsd77WS1)
	require.Error(t, err,
		"service 必须越过工位存在性校验进入物理链路 CTE(PG-only SQL 在 sqlite 报错属预期)")
	require.Empty(t, devices)
	assert.Contains(t, err.Error(), "查询物理链路设备失败")
}

// TestWSD77_GetPhysicalDevicesByWorkstations_FrontSegment 批量物理链路仅前段:
// 空入参 + 越过批量 sys_workstation 查询后 PG-only CTE 报错。
func TestWSD77_GetPhysicalDevicesByWorkstations_FrontSegment(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	// 分支: 空入参 → 空 map 不报错
	result, err := svc.GetPhysicalDevicesByWorkstations(ctx, []string{})
	require.NoError(t, err)
	require.Empty(t, result)

	// 越过前段: 批量 sys_workstation 查询成功后进入 PG-only CTE,sqlite 必报错。
	seedWSD77Workstation(t, db, wsd77WS1, "3f130", wsd77User1)
	result, err = svc.GetPhysicalDevicesByWorkstations(ctx, []string{wsd77WS1})
	require.Error(t, err,
		"越过批量 sys_workstation 查询后进入 PG-only CTE,sqlite 报错属预期(报错即证明已越过前段)")
	assert.Contains(t, err.Error(), "批量查询物理链路设备失败")
	require.Empty(t, result[wsd77WS1], "错误路径返回的预置 map 值应为空切片")
}

// TestWSD77_macIDFragment 纯函数边角: nil → "nil",非 nil → 原值。
func TestWSD77_macIDFragment(t *testing.T) {
	assert.Equal(t, "nil", macIDFragment(nil))
	mac := "b022.7a2e.4a4f"
	assert.Equal(t, "b022.7a2e.4a4f", macIDFragment(&mac))
	empty := ""
	assert.Equal(t, "", macIDFragment(&empty))
}

// TestWSD77_stringFromMap 纯函数边角: nil map / 缺 key / nil 值 / string / 非 string。
func TestWSD77_stringFromMap(t *testing.T) {
	assert.Equal(t, "", stringFromMap(nil, "id"))
	assert.Equal(t, "", stringFromMap(map[string]interface{}{}, "missing"))
	assert.Equal(t, "", stringFromMap(map[string]interface{}{"id": nil}, "id"))
	assert.Equal(t, "ws-1", stringFromMap(map[string]interface{}{"id": "ws-1"}, "id"))
	// 非 string 值走 fmt.Sprintf("%v") 兜底
	assert.Equal(t, "42", stringFromMap(map[string]interface{}{"n": 42}, "n"))
}

// ===========================================================================
// Task 2: 写链 + mergeBySerial 三态
// ===========================================================================

// seedWSD77ADChain 预置完整 AD 查询链(工位→sys_user→sys_ad_user→sys_ad_computer)。
// managedSN 走 managed_by 命中,descSN 走 original_description 命中(空串跳过)。
func seedWSD77ADChain(t *testing.T, db *gorm.DB, wsID, userID, managedSN, descSN string) {
	t.Helper()
	seedWSD77Workstation(t, db, wsID, "3f130", userID)
	seedWSD77User(t, db, userID, "zhangsan", "张三", "")
	seedWSD77ADUser(t, db, "60000000-0000-0000-0000-000000000001", "zhangsan", wsd77DN1)
	if managedSN != "" {
		seedWSD77ADComputer(t, db, wsd77ADComp1, "AD-PC-MANAGED", managedSN,
			"AA:BB:CC:DD:EE:01", "10.1.1.1", "Windows 11", wsd77DN1, "")
	}
	if descSN != "" {
		seedWSD77ADComputer(t, db, wsd77ADComp2, "AD-PC-DESC", descSN,
			"AA:BB:CC:DD:EE:02", "10.1.1.2", "Windows 10", "CN=other,DC=corp", "工位机|zhangsan|主用")
	}
}

// TestWSD77_SyncFromAD 读表→删旧→插新语义(RESEARCH 实证零 LDAP 调用):
// 旧 ad 行软删、manual 行保留、新行按 models.DeviceSourceAD 插入。
func TestWSD77_SyncFromAD(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77ADChain(t, db, wsd77WS1, wsd77User1, "SN-AD-1", "SN-AD-2")
	// 旧 ad 主设备行(应被软删)与 manual 行(应保留)
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000001", wsd77WS1,
		string(models.DeviceSourceAD), "SN-OLD", true)
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000002", wsd77WS1,
		string(models.DeviceSourceManual), "SN-KEEP", false)

	require.NoError(t, svc.SyncFromAD(ctx, wsd77WS1))

	live := wsd77LiveDevices(t, db, wsd77WS1)
	require.Len(t, live, 3, "SN-AD-1 + SN-AD-2 新行 + SN-KEEP manual 保留")
	require.Contains(t, live, "SN-AD-1")
	require.Contains(t, live, "SN-AD-2")
	require.Contains(t, live, "SN-KEEP")

	var oldLive int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM ops_workstation_device WHERE device_serial = 'SN-OLD' AND deleted_at IS NULL`,
	).Scan(&oldLive).Error)
	assert.Zero(t, oldLive, "旧 ad 来源行应被软删")

	d1 := live["SN-AD-1"]
	require.Equal(t, models.DeviceSourceAD, d1.DeviceSource)
	require.Equal(t, models.WorkstationDeviceStatusNormal, d1.Status)
	assert.False(t, d1.IsPrimary, "同步插入的新行不应是主设备")
	require.NotNil(t, d1.ADComputerID)
	assert.Equal(t, wsd77ADComp1, *d1.ADComputerID)
	require.NotNil(t, d1.DeviceName)
	assert.Equal(t, "AD-PC-MANAGED", *d1.DeviceName)
	require.NotNil(t, d1.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:01", *d1.MACAddress)
	assert.Equal(t, models.DeviceSourceManual, live["SN-KEEP"].DeviceSource,
		"manual 来源行不受 ad 同步影响")

	// 分支: 工位不存在 / 空 ID / 工位未绑定用户
	err := svc.SyncFromAD(ctx, wsd77WSMiss)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位不存在")
	err = svc.SyncFromAD(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	seedWSD77Workstation(t, db, wsd77WSNoUsr, "3f999", "")
	err = svc.SyncFromAD(ctx, wsd77WSNoUsr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位未绑定用户ID")
}

// TestWSD77_SyncFromAsset 资产同步: 旧 asset 行软删、新行按 models.DeviceSourceAsset
// 插入并携带 model/type/mac1/nowuser 透传;用户链缺失分支。
func TestWSD77_SyncFromAsset(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", wsd77User1)
	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", "") // 无部门 → 策略2 不触发
	seedWSD77Asset(t, db, wsd77Asset1, "SN-AS-1", "ThinkPad T14", "笔记本",
		"AA:BB:CC:DD:EE:11", "张三", "研发部", "10.2.2.1")
	seedWSD77Asset(t, db, wsd77Asset2, "SN-AS-2", "OptiPlex 7090", "台式机",
		"AA:BB:CC:DD:EE:12", "张三", "研发部", "10.2.2.2")
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000001", wsd77WS1,
		string(models.DeviceSourceAsset), "SN-OLD", true)
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000002", wsd77WS1,
		string(models.DeviceSourceManual), "SN-KEEP", false)

	require.NoError(t, svc.SyncFromAsset(ctx, wsd77WS1))

	live := wsd77LiveDevices(t, db, wsd77WS1)
	require.Len(t, live, 3, "SN-AS-1 + SN-AS-2 新行 + SN-KEEP manual 保留")

	var oldLive int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM ops_workstation_device WHERE device_serial = 'SN-OLD' AND deleted_at IS NULL`,
	).Scan(&oldLive).Error)
	assert.Zero(t, oldLive, "旧 asset 来源行应被软删")

	d1 := live["SN-AS-1"]
	require.Equal(t, models.DeviceSourceAsset, d1.DeviceSource)
	require.Equal(t, models.WorkstationDeviceStatusNormal, d1.Status)
	assert.False(t, d1.IsPrimary)
	require.NotNil(t, d1.AssetID)
	assert.Equal(t, wsd77Asset1, *d1.AssetID)
	require.NotNil(t, d1.DeviceModel)
	assert.Equal(t, "ThinkPad T14", *d1.DeviceModel)
	require.NotNil(t, d1.DeviceType)
	assert.Equal(t, "笔记本", *d1.DeviceType)
	require.NotNil(t, d1.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:11", *d1.MACAddress)
	require.NotNil(t, d1.ResponsibleUser)
	assert.Equal(t, "张三", *d1.ResponsibleUser)

	// 分支: 工位绑定用户但 sys_user 行缺失 → 查询用户信息失败
	seedWSD77Workstation(t, db, wsd77WS2, "3f131", "20000000-0000-0000-0000-00000000999")
	err := svc.SyncFromAsset(ctx, wsd77WS2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户信息失败")

	// 分支: 工位不存在 / 空 ID / 未绑定用户
	err = svc.SyncFromAsset(ctx, wsd77WSMiss)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位不存在")
	err = svc.SyncFromAsset(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	seedWSD77Workstation(t, db, wsd77WSNoUsr, "3f999", "")
	err = svc.SyncFromAsset(ctx, wsd77WSNoUsr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位未绑定用户ID")
}

// TestWSD77_AddDeviceManual 资产命中 / 未命中 / 参数与错误分支。
func TestWSD77_AddDeviceManual(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", "")
	seedWSD77Asset(t, db, wsd77Asset1, "SN-X", "ThinkPad T14", "笔记本",
		"AA:11", "张三", "研发部", "10.0.0.1")

	// 命中: AssetID/model/type 取资产,其余取 req
	name, mac, ip, resp, desc := "REQ-NAME", "REQ-MAC", "REQ-IP", "REQ-USER", "REQ-DESC"
	dev, err := svc.AddDeviceManual(ctx, &AddDeviceRequest{
		WorkstationID:   wsd77WS1,
		DeviceSerial:    "SN-X",
		DeviceName:      &name,
		MACAddress:      &mac,
		IPAddress:       &ip,
		ResponsibleUser: &resp,
		Description:     &desc,
	})
	require.NoError(t, err)
	require.NotNil(t, dev)
	require.NotEmpty(t, dev.ID, "BeforeCreate 应生成 UUID")
	require.Equal(t, models.DeviceSourceManual, dev.DeviceSource)
	require.Equal(t, models.WorkstationDeviceStatusNormal, dev.Status)
	assert.False(t, dev.IsPrimary)
	require.NotNil(t, dev.AssetID)
	assert.Equal(t, wsd77Asset1, *dev.AssetID)
	require.NotNil(t, dev.DeviceModel)
	assert.Equal(t, "ThinkPad T14", *dev.DeviceModel, "资产命中时 model 取资产")
	require.NotNil(t, dev.DeviceType)
	assert.Equal(t, "笔记本", *dev.DeviceType)
	require.NotNil(t, dev.DeviceName)
	assert.Equal(t, "REQ-NAME", *dev.DeviceName)

	var persisted models.WorkstationDevice
	require.NoError(t, db.Where("id = ? AND deleted_at IS NULL", dev.ID).First(&persisted).Error)
	require.NotNil(t, persisted.DeviceSerial)
	assert.Equal(t, "SN-X", *persisted.DeviceSerial, "AddDeviceManual 应落库")

	// 未命中: AssetID/model/type 为空,req 字段原样
	dev2, err := svc.AddDeviceManual(ctx, &AddDeviceRequest{
		WorkstationID: wsd77WS1,
		DeviceSerial:  "SN-NONE",
		DeviceName:    &name,
	})
	require.NoError(t, err)
	assert.Nil(t, dev2.AssetID, "资产未命中 AssetID 应为 nil")
	assert.Nil(t, dev2.DeviceModel)
	assert.Nil(t, dev2.DeviceType)
	require.NotNil(t, dev2.DeviceName)
	assert.Equal(t, "REQ-NAME", *dev2.DeviceName)

	// 参数分支
	_, err = svc.AddDeviceManual(ctx, &AddDeviceRequest{DeviceSerial: "SN"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	_, err = svc.AddDeviceManual(ctx, &AddDeviceRequest{WorkstationID: "not-a-uuid", DeviceSerial: "SN"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数无效")
	_, err = svc.AddDeviceManual(ctx, &AddDeviceRequest{WorkstationID: wsd77WS1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")

	// 错误分支: ops_asset 表缺失 → 查询资产失败
	wsd77Exec(t, db, `DROP TABLE ops_asset`)
	_, err = svc.AddDeviceManual(ctx, &AddDeviceRequest{WorkstationID: wsd77WS1, DeviceSerial: "SN-E"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询资产失败")
}

// TestWSD77_UpdateDevice 11 个可选字段的 updates map:
// 请求只带 2 个字段时其余字段不被覆盖;全字段更新覆盖全部 if 分支。
func TestWSD77_UpdateDevice(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()
	devID := "70000000-0000-0000-0000-000000000011"

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", "")
	wsd77Exec(t, db,
		`INSERT INTO ops_workstation_device
		   (id, workstation_id, device_source, device_serial, device_name, device_model, device_type,
		    mac_address, ip_address, responsible_user, status, is_primary, priority, description, deleted_at)
		 VALUES (?, ?, 'manual', 'SN-U1', 'OLD-NAME', 'OLD-MODEL', 'OLD-TYPE',
		    'OLD-MAC', '10.0.0.9', 'OLD-USER', ?, 1, 5, 'OLD-DESC', NULL)`,
		devID, wsd77WS1, models.WorkstationDeviceStatusStopped)

	// 部分更新: 仅 DeviceName + MACAddress
	newName, newMAC := "NEW-NAME", "NEW-MAC"
	require.NoError(t, svc.UpdateDevice(ctx, devID, &UpdateDeviceRequest{
		DeviceName: &newName,
		MACAddress: &newMAC,
	}))

	var d models.WorkstationDevice
	require.NoError(t, db.Where("id = ? AND deleted_at IS NULL", devID).First(&d).Error)
	require.NotNil(t, d.DeviceName)
	assert.Equal(t, "NEW-NAME", *d.DeviceName)
	require.NotNil(t, d.MACAddress)
	assert.Equal(t, "NEW-MAC", *d.MACAddress)
	// 未提交字段不得被覆盖
	require.NotNil(t, d.DeviceModel)
	assert.Equal(t, "OLD-MODEL", *d.DeviceModel)
	require.NotNil(t, d.DeviceType)
	assert.Equal(t, "OLD-TYPE", *d.DeviceType)
	require.NotNil(t, d.IPAddress)
	assert.Equal(t, "10.0.0.9", *d.IPAddress)
	require.NotNil(t, d.ResponsibleUser)
	assert.Equal(t, "OLD-USER", *d.ResponsibleUser)
	assert.Equal(t, models.WorkstationDeviceStatusStopped, d.Status)
	assert.True(t, d.IsPrimary)
	assert.Equal(t, 5, d.Priority)
	require.NotNil(t, d.Description)
	assert.Equal(t, "OLD-DESC", *d.Description)

	// 全字段更新(覆盖 11 个可选字段 if 分支)
	serial := "SN-U1-NEW"
	model := "NEW-MODEL"
	typ := "NEW-TYPE"
	ip := "10.9.9.9"
	user := "NEW-USER"
	stopped := models.WorkstationDeviceStatusStopped
	prio := 9
	primary := false
	desc := "NEW-DESC"
	require.NoError(t, svc.UpdateDevice(ctx, devID, &UpdateDeviceRequest{
		DeviceSerial:    &serial,
		DeviceName:      &newName,
		DeviceModel:     &model,
		DeviceType:      &typ,
		MACAddress:      &newMAC,
		IPAddress:       &ip,
		ResponsibleUser: &user,
		Status:          &stopped,
		Priority:        &prio,
		IsPrimary:       &primary,
		Description:     &desc,
	}))
	require.NoError(t, db.Where("id = ? AND deleted_at IS NULL", devID).First(&d).Error)
	require.NotNil(t, d.DeviceSerial)
	assert.Equal(t, "SN-U1-NEW", *d.DeviceSerial)
	require.NotNil(t, d.DeviceModel)
	assert.Equal(t, "NEW-MODEL", *d.DeviceModel)
	require.NotNil(t, d.IPAddress)
	assert.Equal(t, "10.9.9.9", *d.IPAddress)
	assert.Equal(t, models.WorkstationDeviceStatusStopped, d.Status)
	assert.False(t, d.IsPrimary)
	assert.Equal(t, 9, d.Priority)

	// 分支: 设备不存在 / 空 ID
	err := svc.UpdateDevice(ctx, "70000000-0000-0000-0000-00000000404", &UpdateDeviceRequest{DeviceName: &newName})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备不存在")
	err = svc.UpdateDevice(ctx, "", &UpdateDeviceRequest{DeviceName: &newName})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
}

// TestWSD77_DeleteDevice 软删 + 不存在 id 不报错 + 空 id 参数分支。
func TestWSD77_DeleteDevice(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()
	devID := "70000000-0000-0000-0000-000000000021"

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", "")
	seedWSD77DeviceRow(t, db, devID, wsd77WS1, string(models.DeviceSourceManual), "SN-D1", false)

	require.NoError(t, svc.DeleteDevice(ctx, devID))

	var liveCnt, softCnt int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM ops_workstation_device WHERE id = ? AND deleted_at IS NULL`, devID,
	).Scan(&liveCnt).Error)
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM ops_workstation_device WHERE id = ? AND deleted_at IS NOT NULL`, devID,
	).Scan(&softCnt).Error)
	assert.Zero(t, liveCnt, "删除后 live 行应为 0")
	assert.Equal(t, int64(1), softCnt, "删除应为软删(deleted_at 置值)")

	// 分支: 不存在的 id → RowsAffected 0 不构成错误(现行为断言)
	require.NoError(t, svc.DeleteDevice(ctx, "70000000-0000-0000-0000-00000000404"))

	// 分支: 空 ID
	err := svc.DeleteDevice(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
}

// TestWSD77_SetPrimaryDevice 事务两步: 旧主 is_primary=false、新行 is_primary=true。
func TestWSD77_SetPrimaryDevice(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()
	oldID := "70000000-0000-0000-0000-000000000031"
	newID := "70000000-0000-0000-0000-000000000032"

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", "")
	seedWSD77DeviceRow(t, db, oldID, wsd77WS1, string(models.DeviceSourceManual), "SN-P-OLD", true)
	seedWSD77DeviceRow(t, db, newID, wsd77WS1, string(models.DeviceSourceManual), "SN-P-NEW", false)

	require.NoError(t, svc.SetPrimaryDevice(ctx, newID))

	var oldRow, newRow models.WorkstationDevice
	require.NoError(t, db.Where("id = ?", oldID).First(&oldRow).Error)
	require.NoError(t, db.Where("id = ?", newID).First(&newRow).Error)
	assert.False(t, oldRow.IsPrimary, "旧主设备应被取消")
	assert.True(t, newRow.IsPrimary, "新设备应成为主设备")

	// 分支: 设备不存在 / 空 ID
	err := svc.SetPrimaryDevice(ctx, "70000000-0000-0000-0000-00000000404")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "设备不存在")
	err = svc.SetPrimaryDevice(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
}

// TestWSD77_SetPrimaryAndSave 转手动保存事务: ad/asset 旧行被清、旧主 false、
// 新 manual 行 is_primary=true 且字段按 :53-61 合并优先级填充。
func TestWSD77_SetPrimaryAndSave(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	// 双命中链: SN-BOTH 同时在 sys_ad_computer 与 ops_asset
	seedWSD77ADChain(t, db, wsd77WS1, wsd77User1, "SN-BOTH", "")
	seedWSD77Asset(t, db, wsd77Asset1, "SN-BOTH", "ASSET-MODEL", "ASSET-TYPE",
		"AA:BB:CC:DD:EE:99", "张三", "研发部", "10.9.9.9")
	// 预置旧行: ad + asset 来源行与既有 manual 主设备
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000041", wsd77WS1,
		string(models.DeviceSourceAD), "SN-BOTH", false)
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000042", wsd77WS1,
		string(models.DeviceSourceAsset), "SN-BOTH", false)
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000043", wsd77WS1,
		string(models.DeviceSourceManual), "SN-OLDMAN", true)

	reqName, reqModel, reqType := "REQ-NAME", "REQ-MODEL", "REQ-TYPE"
	reqMAC, reqIP, reqUser := "REQ-MAC", "REQ-IP", "REQ-USER"
	require.NoError(t, svc.SetPrimaryAndSave(ctx, "ad-0", &SetPrimaryAndSaveRequest{
		WorkstationID:   wsd77WS1,
		DeviceSerial:    "SN-BOTH",
		DeviceName:      reqName,
		DeviceModel:     &reqModel,
		DeviceType:      &reqType,
		MACAddress:      &reqMAC,
		IPAddress:       &reqIP,
		ResponsibleUser: &reqUser,
	}))

	live := wsd77LiveDevices(t, db, wsd77WS1)
	require.Len(t, live, 2, "ad/asset 行被清后应剩 旧manual + 新manual 两行")

	var adLive int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM ops_workstation_device WHERE device_source IN ('ad','asset') AND deleted_at IS NULL`,
	).Scan(&adLive).Error)
	assert.Zero(t, adLive, "ad/asset 来源行应被软删")

	oldMan := live["SN-OLDMAN"]
	require.NotNil(t, oldMan)
	assert.False(t, oldMan.IsPrimary, "旧主设备应被取消")

	merged := live["SN-BOTH"]
	require.NotNil(t, merged, "应写入 SN-BOTH 的 manual 新行")
	require.Equal(t, models.DeviceSourceManual, merged.DeviceSource)
	assert.True(t, merged.IsPrimary, "新 manual 行应为主设备")
	require.Equal(t, models.WorkstationDeviceStatusNormal, merged.Status)
	// 合并优先级(:53-61,D-03 有据锚点):
	require.NotNil(t, merged.DeviceName)
	assert.Equal(t, "AD-PC-MANAGED", *merged.DeviceName, "deviceName: AD > req")
	require.NotNil(t, merged.DeviceModel)
	assert.Equal(t, "ASSET-MODEL", *merged.DeviceModel, "deviceModel: Asset > req")
	require.NotNil(t, merged.DeviceType)
	assert.Equal(t, "ASSET-TYPE", *merged.DeviceType, "deviceType: Asset > req")
	require.NotNil(t, merged.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:01", *merged.MACAddress, "mac: AD > Asset > req")
	require.NotNil(t, merged.IPAddress)
	assert.Equal(t, "10.1.1.1", *merged.IPAddress, "ip: AD > req(quirk-77-1 修复后可达)")
	require.NotNil(t, merged.ResponsibleUser)
	assert.Equal(t, "张三", *merged.ResponsibleUser, "responsibleUser: Asset > req")
	require.NotNil(t, merged.AssetID)
	assert.Equal(t, wsd77Asset1, *merged.AssetID, "assetID: 命中填")
	require.NotNil(t, merged.ADComputerID)
	assert.Equal(t, wsd77ADComp1, *merged.ADComputerID, "adComputerID: 命中填")

	// 参数分支: nil req / 空 ws / 空 serial / 工位不存在
	err := svc.SetPrimaryAndSave(ctx, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	err = svc.SetPrimaryAndSave(ctx, "", &SetPrimaryAndSaveRequest{WorkstationID: "", DeviceSerial: "SN"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	err = svc.SetPrimaryAndSave(ctx, "", &SetPrimaryAndSaveRequest{WorkstationID: wsd77WS1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	err = svc.SetPrimaryAndSave(ctx, "", &SetPrimaryAndSaveRequest{WorkstationID: wsd77WSMiss, DeviceSerial: "SN"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位不存在")
}

// TestWSD77_SetPrimaryAndSaveBySerial req nil 归一化分支:
// (workstationID, serial) 入参 + nil req → 全部字段依赖 AD/Asset 实时数据。
func TestWSD77_SetPrimaryAndSaveBySerial(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	seedWSD77ADChain(t, db, wsd77WS1, wsd77User1, "SN-BY-SERIAL", "")
	seedWSD77Asset(t, db, wsd77Asset1, "SN-BY-SERIAL", "ASSET-MODEL", "ASSET-TYPE",
		"AA:BB:CC:DD:EE:99", "张三", "研发部", "10.9.9.9")
	seedWSD77DeviceRow(t, db, "70000000-0000-0000-0000-000000000051", wsd77WS1,
		string(models.DeviceSourceAD), "SN-STALE", false)

	require.NoError(t, svc.SetPrimaryAndSaveBySerial(ctx, wsd77WS1, "SN-BY-SERIAL", nil))

	live := wsd77LiveDevices(t, db, wsd77WS1)
	merged := live["SN-BY-SERIAL"]
	require.NotNil(t, merged, "req nil 时应以 serial 参数为键写入 manual 行")
	require.Equal(t, models.DeviceSourceManual, merged.DeviceSource)
	assert.True(t, merged.IsPrimary)
	require.NotNil(t, merged.DeviceName)
	assert.Equal(t, "AD-PC-MANAGED", *merged.DeviceName, "req nil → name 全依赖 AD")
	require.NotNil(t, merged.DeviceModel)
	assert.Equal(t, "ASSET-MODEL", *merged.DeviceModel)
	require.NotNil(t, merged.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:01", *merged.MACAddress)
	require.NotNil(t, merged.IPAddress)
	assert.Equal(t, "10.1.1.1", *merged.IPAddress)
	require.NotNil(t, merged.ResponsibleUser)
	assert.Equal(t, "张三", *merged.ResponsibleUser)

	// 旧 ad 来源行被清
	var adLive int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM ops_workstation_device WHERE device_source = 'ad' AND deleted_at IS NULL`,
	).Scan(&adLive).Error)
	assert.Zero(t, adLive)

	// 非 nil req 且 DeviceSerial 为空 → 归一化为 serial 参数
	reqName := "REQ-NAME"
	require.NoError(t, svc.SetPrimaryAndSaveBySerial(ctx, wsd77WS1, "SN-BY-SERIAL", &SetPrimaryAndSaveRequest{
		DeviceName: reqName,
	}))
	live2 := wsd77LiveDevices(t, db, wsd77WS1)
	merged2 := live2["SN-BY-SERIAL"]
	require.NotNil(t, merged2)
	// 已有 AD 命中 → name 仍取 AD(AD > req),但归一化后的行仍是 SN-BY-SERIAL
	require.NotNil(t, merged2.DeviceName)
	assert.Equal(t, "AD-PC-MANAGED", *merged2.DeviceName)

	// 参数分支: 空 ws / 空 serial / 工位不存在
	err := svc.SetPrimaryAndSaveBySerial(ctx, "", "SN", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	err = svc.SetPrimaryAndSaveBySerial(ctx, wsd77WS1, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "参数缺失")
	err = svc.SetPrimaryAndSaveBySerial(ctx, wsd77WSMiss, "SN", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工位不存在")
}

// TestWSD77_mergeBySerial_ThreeStates 合并优先级三态(D-03 有据,
// 据源 = workstation_device_service.go:53-61 接口注释):
//   双命中(SN 同时在 sys_ad_computer 与 ops_asset)/ 仅 Asset / 都没有(req fallback)
// 断言落库 ops_workstation_device 行字段严格按注释优先级。
func TestWSD77_mergeBySerial_ThreeStates(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewWorkstationDeviceService(db)
	ctx := context.Background()

	// AD 侧仅 SN-BOTH;Asset 侧有 SN-BOTH + SN-ASSET-ONLY;SN-REQ-ONLY 两侧皆无
	seedWSD77ADChain(t, db, wsd77WS1, wsd77User1, "SN-BOTH", "")
	seedWSD77Asset(t, db, wsd77Asset1, "SN-BOTH", "ASSET-MODEL", "ASSET-TYPE",
		"AA:BB:CC:DD:EE:99", "张三", "", "10.9.9.9")
	seedWSD77Asset(t, db, wsd77Asset2, "SN-ASSET-ONLY", "ASSET2-MODEL", "ASSET2-TYPE",
		"AA:BB:CC:DD:EE:88", "张三", "", "10.9.9.8")

	req := func(name, model, typ, mac, ip, user string) *SetPrimaryAndSaveRequest {
		return &SetPrimaryAndSaveRequest{
			WorkstationID:   wsd77WS1,
			DeviceSerial:    "", // 由 SetPrimaryAndSaveBySerial 归一化填充
			DeviceName:      name,
			DeviceModel:     &model,
			DeviceType:      &typ,
			MACAddress:      &mac,
			IPAddress:       &ip,
			ResponsibleUser: &user,
		}
	}

	// 态1: 双命中 — AD>req / Asset>req / mac AD>Asset / ip AD>req / 双 ID 填充
	require.NoError(t, svc.SetPrimaryAndSaveBySerial(ctx, wsd77WS1, "SN-BOTH",
		req("REQ-NAME-1", "REQ-MODEL-1", "REQ-TYPE-1", "REQ-MAC-1", "REQ-IP-1", "REQ-USER-1")))
	// 态2: 仅 Asset — model/type/mac 取资产,name/ip 回退 req
	require.NoError(t, svc.SetPrimaryAndSaveBySerial(ctx, wsd77WS1, "SN-ASSET-ONLY",
		req("REQ-NAME-2", "REQ-MODEL-2", "REQ-TYPE-2", "REQ-MAC-2", "REQ-IP-2", "REQ-USER-2")))
	// 态3: 都没有 — 全部回退 req,双 ID 为 nil
	require.NoError(t, svc.SetPrimaryAndSaveBySerial(ctx, wsd77WS1, "SN-REQ-ONLY",
		req("REQ-NAME-3", "REQ-MODEL-3", "REQ-TYPE-3", "REQ-MAC-3", "REQ-IP-3", "REQ-USER-3")))

	live := wsd77LiveDevices(t, db, wsd77WS1)
	require.Len(t, live, 3)

	// 态1: 双命中
	both := live["SN-BOTH"]
	require.NotNil(t, both)
	require.Equal(t, models.DeviceSourceManual, both.DeviceSource)
	require.NotNil(t, both.DeviceName)
	assert.Equal(t, "AD-PC-MANAGED", *both.DeviceName, "态1 deviceName: AD > req")
	require.NotNil(t, both.DeviceModel)
	assert.Equal(t, "ASSET-MODEL", *both.DeviceModel, "态1 deviceModel: Asset > req")
	require.NotNil(t, both.DeviceType)
	assert.Equal(t, "ASSET-TYPE", *both.DeviceType, "态1 deviceType: Asset > req")
	require.NotNil(t, both.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:01", *both.MACAddress,
		"态1 mac: AD > Asset(资产 mac1=...99 不得胜出) > req")
	require.NotNil(t, both.IPAddress)
	assert.Equal(t, "10.1.1.1", *both.IPAddress,
		"态1 ip: AD > req(资产 machine_ip=10.9.9.9 不是 ip 来源)")
	require.NotNil(t, both.ResponsibleUser)
	assert.Equal(t, "张三", *both.ResponsibleUser, "态1 responsibleUser: Asset > req")
	require.NotNil(t, both.AssetID)
	assert.Equal(t, wsd77Asset1, *both.AssetID)
	require.NotNil(t, both.ADComputerID)
	assert.Equal(t, wsd77ADComp1, *both.ADComputerID)

	// 态2: 仅 Asset
	only := live["SN-ASSET-ONLY"]
	require.NotNil(t, only)
	require.NotNil(t, only.DeviceName)
	assert.Equal(t, "REQ-NAME-2", *only.DeviceName, "态2 deviceName: 无 AD → req")
	require.NotNil(t, only.DeviceModel)
	assert.Equal(t, "ASSET2-MODEL", *only.DeviceModel, "态2 deviceModel: Asset > req")
	require.NotNil(t, only.DeviceType)
	assert.Equal(t, "ASSET2-TYPE", *only.DeviceType)
	require.NotNil(t, only.MACAddress)
	assert.Equal(t, "AA:BB:CC:DD:EE:88", *only.MACAddress, "态2 mac: 无 AD → Asset > req")
	require.NotNil(t, only.IPAddress)
	assert.Equal(t, "REQ-IP-2", *only.IPAddress, "态2 ip: 无 AD → req")
	require.NotNil(t, only.ResponsibleUser)
	assert.Equal(t, "张三", *only.ResponsibleUser)
	require.NotNil(t, only.AssetID)
	assert.Equal(t, wsd77Asset2, *only.AssetID)
	assert.Nil(t, only.ADComputerID, "态2 无 AD 命中 → adComputerID 为 nil")

	// 态3: req fallback
	none := live["SN-REQ-ONLY"]
	require.NotNil(t, none)
	require.NotNil(t, none.DeviceName)
	assert.Equal(t, "REQ-NAME-3", *none.DeviceName)
	require.NotNil(t, none.DeviceModel)
	assert.Equal(t, "REQ-MODEL-3", *none.DeviceModel)
	require.NotNil(t, none.DeviceType)
	assert.Equal(t, "REQ-TYPE-3", *none.DeviceType)
	require.NotNil(t, none.MACAddress)
	assert.Equal(t, "REQ-MAC-3", *none.MACAddress)
	require.NotNil(t, none.IPAddress)
	assert.Equal(t, "REQ-IP-3", *none.IPAddress)
	require.NotNil(t, none.ResponsibleUser)
	assert.Equal(t, "REQ-USER-3", *none.ResponsibleUser)
	assert.Nil(t, none.AssetID, "态3 双 ID 均为 nil")
	assert.Nil(t, none.ADComputerID, "态3 双 ID 均为 nil")
	assert.True(t, none.IsPrimary, "最后一次调用对应的行是主设备")
}
