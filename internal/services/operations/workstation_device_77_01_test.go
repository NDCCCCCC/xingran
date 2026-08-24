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
