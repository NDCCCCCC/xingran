package operations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// =========================================================================
// Phase 35: 工位导出增强 — 单元测试
// =========================================================================
//
// 覆盖范围:
//   1. parsePhysicalPortInfo: 正则解析物理链路 Description (纯函数)
//   2. stringValueOrEmpty:    *string → string 安全转换
//   3. batchGetWorkstationNames / batchGetADEnrichment / batchGetAssetEnrichment:
//      批量 enrichment 查询 (内存 SQLite)
//   4. writeDeviceSheet:      设备 sheet 写入器 (Excel 结构验证)

// ----- 1. parsePhysicalPortInfo -----

func TestParsePhysicalPortInfo_FullFormat(t *testing.T) {
	desc := "端口 GE2/6 (信息点 WH-04F-130)"
	port, infoPoint := parsePhysicalPortInfo(&desc)
	assert.Equal(t, "GE2/6", port)
	assert.Equal(t, "WH-04F-130", infoPoint)
}

func TestParsePhysicalPortInfo_NoInfoPoint(t *testing.T) {
	desc := "端口 GE2/6"
	port, infoPoint := parsePhysicalPortInfo(&desc)
	assert.Equal(t, "GE2/6", port)
	assert.Equal(t, "", infoPoint)
}

func TestParsePhysicalPortInfo_WithHistoryHint(t *testing.T) {
	// 实际生产中 Description 包含历史关联提示 (B-3f130 修复后的格式)
	desc := "端口 GE2/6 (信息点 WH-04F-130)\n历史关联 (最后上线时间: 2026-07-21 12:00:00)"
	port, infoPoint := parsePhysicalPortInfo(&desc)
	assert.Equal(t, "GE2/6", port)
	assert.Equal(t, "WH-04F-130", infoPoint)
}

func TestParsePhysicalPortInfo_HintWithoutInfoPoint(t *testing.T) {
	// 历史关联但端口无信息点 (工位迁移后)
	desc := "端口 GE2/6\n历史关联 (最后上线时间: 2026-07-21 12:00:00)"
	port, infoPoint := parsePhysicalPortInfo(&desc)
	assert.Equal(t, "GE2/6", port)
	assert.Equal(t, "", infoPoint)
}

func TestParsePhysicalPortInfo_NilDesc(t *testing.T) {
	port, infoPoint := parsePhysicalPortInfo(nil)
	assert.Equal(t, "", port)
	assert.Equal(t, "", infoPoint)
}

func TestParsePhysicalPortInfo_NoMatch(t *testing.T) {
	desc := "no port pattern here"
	port, infoPoint := parsePhysicalPortInfo(&desc)
	assert.Equal(t, "", port)
	assert.Equal(t, "", infoPoint)
}

func TestParsePhysicalPortInfo_DifferentPortFormats(t *testing.T) {
	// 工位 3f130 真实端口: GE2/6
	// 验证不同接口命名格式都能正确解析
	cases := []struct {
		desc       string
		wantPort   string
		wantInfoPt string
	}{
		{"端口 GigabitEthernet0/0/1 (信息点 IDC-A01)", "GigabitEthernet0/0/1", "IDC-A01"},
		{"端口 TenGigabitE1/24 (信息点 核心-A)", "TenGigabitE1/24", "核心-A"},
		{"端口 Eth-Trunk1 (信息点 汇聚)", "Eth-Trunk1", "汇聚"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			port, infoPoint := parsePhysicalPortInfo(&tc.desc)
			assert.Equal(t, tc.wantPort, port)
			assert.Equal(t, tc.wantInfoPt, infoPoint)
		})
	}
}

// ----- 2. stringValueOrEmpty -----

func TestStringValueOrEmpty(t *testing.T) {
	empty := ""
	value := "hello"

	assert.Equal(t, "", stringValueOrEmpty(nil), "nil 指针应返回空字符串")
	assert.Equal(t, "", stringValueOrEmpty(&empty), "空字符串应原样返回")
	assert.Equal(t, "hello", stringValueOrEmpty(&value), "非空字符串应原样返回")
}

// ----- 3. 批量 enrichment 函数 (内存 SQLite) -----

func setupEnrichmentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 工位 (含 deleted_at)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY,
			workstation_name TEXT,
			deleted_at DATETIME
		)
	`).Error)

	// AD 计算机 (无 deleted_at)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_computer (
			id TEXT PRIMARY KEY,
			computer_name TEXT,
			operating_system TEXT,
			last_logon DATETIME
		)
	`).Error)

	// 资产 (含 deleted_at)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			machine_ip TEXT,
			deleted_at DATETIME
		)
	`).Error)

	return db
}

func TestBatchGetWorkstationNames_Basic(t *testing.T) {
	db := setupEnrichmentTestDB(t)

	// 工位 3f130 + 3f131, 3f132 已被软删除
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-1', '3f130', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-2', '3f131', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-3', '3f132', '2026-07-21 00:00:00')`).Error)

	svc := &ExcelService{db: db}
	result := svc.batchGetWorkstationNames(context.Background(), []string{"ws-1", "ws-2", "ws-3"})

	assert.Equal(t, "3f130", result["ws-1"], "正常工位应返回名称")
	assert.Equal(t, "3f131", result["ws-2"], "正常工位应返回名称")
	_, exists := result["ws-3"]
	assert.False(t, exists, "软删除工位不应在结果中")
}

func TestBatchGetWorkstationNames_EmptyInput(t *testing.T) {
	db := setupEnrichmentTestDB(t)
	svc := &ExcelService{db: db}

	result := svc.batchGetWorkstationNames(context.Background(), []string{})
	assert.Empty(t, result, "空输入应返回空 map")
}

func TestBatchGetADEnrichment_Basic(t *testing.T) {
	db := setupEnrichmentTestDB(t)

	now := time.Now()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_computer VALUES ('ad-1', 'PC-001', 'Windows 11', ?)
	`, now).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_computer VALUES ('ad-2', 'PC-002', 'Ubuntu 22.04', NULL)
	`).Error)

	svc := &ExcelService{db: db}
	devices := []*models.WorkstationDevice{
		{ADComputerID: stringPtr("ad-1")},
		{ADComputerID: stringPtr("ad-2")},
		{ADComputerID: nil},
	}

	result := svc.batchGetADEnrichment(context.Background(), devices)

	require.Contains(t, result, "ad-1")
	assert.Equal(t, "Windows 11", result["ad-1"].os)
	assert.NotEmpty(t, result["ad-1"].lastLogon, "有 LastLogon 应格式化")

	require.Contains(t, result, "ad-2")
	assert.Equal(t, "Ubuntu 22.04", result["ad-2"].os)
	assert.Empty(t, result["ad-2"].lastLogon, "NULL LastLogon 应为空字符串")
}

func TestBatchGetAssetEnrichment_Basic(t *testing.T) {
	db := setupEnrichmentTestDB(t)

	require.NoError(t, db.Exec(`INSERT INTO ops_asset VALUES ('a-1', 'SN-001', '10.1.1.100', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset VALUES ('a-2', 'SN-002', '192.168.1.50', '2026-07-21 00:00:00')`).Error)

	svc := &ExcelService{db: db}
	devices := []*models.WorkstationDevice{
		{AssetID: stringPtr("a-1")},
		{AssetID: stringPtr("a-2")},
		{AssetID: nil},
	}

	result := svc.batchGetAssetEnrichment(context.Background(), devices)

	assert.Equal(t, "10.1.1.100", result["a-1"], "正常资产应返回 IP")
	_, exists := result["a-2"]
	assert.False(t, exists, "软删除资产不应在结果中")
}

// ----- 4. writeDeviceSheet (Excel 结构验证) -----

func TestWriteDeviceSheet_EmptyDevices(t *testing.T) {
	f := excelize.NewFile()
	svc := &ExcelService{}

	headers := []string{"工位名称", "MAC", "Port"}
	err := svc.writeDeviceSheet(f, "测试Sheet", headers, nil, func(d *models.WorkstationDevice) []string {
		t.Fatal("空设备列表不应调用 rowMapper")
		return nil
	})
	require.NoError(t, err)

	// 验证 sheet 存在且包含表头
	rows, err := f.GetRows("测试Sheet")
	require.NoError(t, err)
	assert.Len(t, rows, 1, "空设备列表应只有 1 行表头")
	assert.Equal(t, []string{"工位名称", "MAC", "Port"}, rows[0])
}

func TestWriteDeviceSheet_WithData(t *testing.T) {
	f := excelize.NewFile()
	svc := &ExcelService{}

	headers := []string{"工位名称", "ComputerName", "MAC"}
	mac := "b022.7a2e.4a4f"
	cn := "PC-001"
	devices := []*models.WorkstationDevice{
		{WorkstationID: "ws-1", DeviceName: &cn, MACAddress: &mac},
		{WorkstationID: "ws-1", DeviceName: &cn, MACAddress: &mac},
	}

	err := svc.writeDeviceSheet(f, "AD", headers, devices, func(d *models.WorkstationDevice) []string {
		return []string{
			"3f130",
			stringValueOrEmpty(d.DeviceName),
			stringValueOrEmpty(d.MACAddress),
		}
	})
	require.NoError(t, err)

	rows, err := f.GetRows("AD")
	require.NoError(t, err)
	require.Len(t, rows, 3, "1 行表头 + 2 行数据")
	assert.Equal(t, "工位名称", rows[0][0])
	assert.Equal(t, "3f130", rows[1][0])
	assert.Equal(t, "b022.7a2e.4a4f", rows[2][2])
}

func TestWriteDeviceSheet_PhysicalDevicesWithRegex(t *testing.T) {
	// 集成测试: writeDeviceSheet + parsePhysicalPortInfo 协同
	// Phase 35 增强版: 12 列 (工位名称, 设备名称, 序列号, 型号, 类型, MAC, IP, 责任人, Port, InfoPoint, LastSeen, Confidence)
	f := excelize.NewFile()
	svc := &ExcelService{}

	headers := []string{"工位名称", "设备名称", "序列号", "型号", "类型", "MAC", "IP地址", "责任人", "Port", "InfoPoint", "LastSeen", "Confidence"}

	lastSeen := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	confidence := 1.0
	mac := "b022.7a2e.4a4f"
	ip := "10.62.51.83"
	sn := "5CD017GY9D"
	model := "HP 430 G6"
	devName := "CXHUB-17GY9D"
	owner := "王冰"
	desc := "端口 GE2/6 (信息点 WH-04F-130)"

	devices := []*models.WorkstationDevice{
		{
			WorkstationID:   "ws-1",
			DeviceName:      &devName,
			DeviceSerial:    &sn,
			DeviceModel:     &model,
			MACAddress:      &mac,
			IPAddress:       &ip,
			ResponsibleUser: &owner,
			Description:     &desc,
			HistoryLastSeen: &lastSeen,
			Confidence:      &confidence,
		},
	}

	err := svc.writeDeviceSheet(f, "物理链路", headers, devices, func(d *models.WorkstationDevice) []string {
		port, infoPoint := parsePhysicalPortInfo(d.Description)
		ls := ""
		if d.HistoryLastSeen != nil && !d.HistoryLastSeen.IsZero() {
			ls = d.HistoryLastSeen.Format("2006-01-02 15:04:05")
		}
		c := ""
		if d.Confidence != nil {
			c = formatFloat(*d.Confidence)
		}
		return []string{
			"3f130",
			stringValueOrEmpty(d.DeviceName),
			stringValueOrEmpty(d.DeviceSerial),
			stringValueOrEmpty(d.DeviceModel),
			stringValueOrEmpty(d.DeviceType),
			stringValueOrEmpty(d.MACAddress),
			stringValueOrEmpty(d.IPAddress),
			stringValueOrEmpty(d.ResponsibleUser),
			port,
			infoPoint,
			ls,
			c,
		}
	})
	require.NoError(t, err)

	rows, err := f.GetRows("物理链路")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Len(t, rows[1], 12, "应输出 12 列")
	assert.Equal(t, "3f130", rows[1][0])
	assert.Equal(t, "CXHUB-17GY9D", rows[1][1])
	assert.Equal(t, "5CD017GY9D", rows[1][2])
	assert.Equal(t, "HP 430 G6", rows[1][3])
	assert.Equal(t, "b022.7a2e.4a4f", rows[1][5])
	assert.Equal(t, "10.62.51.83", rows[1][6])
	assert.Equal(t, "王冰", rows[1][7])
	assert.Equal(t, "GE2/6", rows[1][8])
	assert.Equal(t, "WH-04F-130", rows[1][9])
	assert.Equal(t, "2026-07-21 12:00:00", rows[1][10])
	assert.Equal(t, "1.00", rows[1][11])
}

// ----- 5. queryWorkstationIDsForExport (集成测试, 不复用 WorkstationQueryBuilder 复杂 JOIN) -----

// TestQueryWorkstationIDsForExport_NoJoinIssue 验证修复: 不复用 WorkstationQueryBuilder
// 后, 即便 workstation 表中有 floor_id/building_id 为空字符串的工位, 也能正常返回 ID。
//
// 修复前: queryWorkstationIDsForExport 复用 WorkstationQueryBuilder, 该 builder 包含
//   LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid
// 在 PG 上若 floor_id 为空字符串, ''::uuid 会报错 → 整个 query 失败 →
// workstationIDs 返回空 → appendWorkstationDeviceSheets 提前返回 →
// 用户看到只有工位列表 sheet, 没有设备 sheet。
//
// 修复后: 直接查询 sys_workstation 表, 不带 JOIN, 应用 FilterMapping 过滤。
func TestQueryWorkstationIDsForExport_NoJoinIssue(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// sys_workstation 表 (含可能被 NULL 或空字符串的 floor_id/building_id)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY,
			workstation_name TEXT,
			floor_id TEXT,
			building_id TEXT,
			dept_id TEXT,
			user_id TEXT,
			status INTEGER,
			deleted_at DATETIME
		)
	`).Error)

	// 工位 A: floor_id = '' (空字符串, 会让旧版 queryBuilder 失败)
	require.NoError(t, db.Exec(`
		INSERT INTO sys_workstation VALUES ('ws-A', '3f130', '', '', '', '', 0, NULL)
	`).Error)
	// 工位 B: 正常数据
	require.NoError(t, db.Exec(`
		INSERT INTO sys_workstation VALUES ('ws-B', '3f131', 'f1', 'b1', 'd1', 'u1', 0, NULL)
	`).Error)
	// 工位 C: 软删除
	require.NoError(t, db.Exec(`
		INSERT INTO sys_workstation VALUES ('ws-C', '3f132', 'f2', 'b2', 'd2', 'u2', 0, '2026-07-21')
	`).Error)

	svc := &ExcelService{db: db}

	// 不带 filter, 期望返回 2 个非软删除工位
	ids, err := svc.queryWorkstationIDsForExport(context.Background(), nil, map[string]any{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"ws-A", "ws-B"}, ids,
		"修复后应跳过 JOIN, 返回所有未软删除工位的 ID (含 floor_id='' 的工位 A)")
}

// TestQueryWorkstationIDsForExport_FilterMapping 验证 FilterMapping (name → workstation_name)
func TestQueryWorkstationIDsForExport_FilterMapping(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY,
			workstation_name TEXT,
			floor_id TEXT,
			building_id TEXT,
			dept_id TEXT,
			user_id TEXT,
			status INTEGER,
			deleted_at DATETIME
		)
	`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-1', '3f130', '', '', '', '', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-2', '3f131', '', '', '', '', 0, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-3', 'XYZ-001', '', '', '', '', 0, NULL)`).Error)

	svc := &ExcelService{db: db}

	// 用 name=3f1 模糊匹配, 应返回 ws-1, ws-2
	ids, err := svc.queryWorkstationIDsForExport(context.Background(), nil, map[string]any{
		"name": "3f1",
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"ws-1", "ws-2"}, ids)
}

// ----- 工具函数 -----

func stringPtr(s string) *string {
	return &s
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// =========================================================================
// Phase 35: 批量查询方法单元测试
// =========================================================================

// setupBatchTestDB 构造批量查询测试所需的最小 SQLite schema
func setupBatchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_workstation (
			id TEXT PRIMARY KEY,
			workstation_name TEXT,
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
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_user (
			username TEXT PRIMARY KEY,
			user_dn TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_ad_computer (
			id TEXT PRIMARY KEY,
			serial_number TEXT,
			computer_name TEXT,
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
			ip_address TEXT,
			machine_ip TEXT,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

func TestGetADDevicesByWorkstations_EmptyInput(t *testing.T) {
	db := setupBatchTestDB(t)
	svc := NewWorkstationDeviceService(db)

	result, err := svc.GetADDevicesByWorkstations(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, result, "空输入应返回空 map")
}

func TestGetADDevicesByWorkstations_NoUserBound(t *testing.T) {
	db := setupBatchTestDB(t)

	// 工位未绑 user_id (B-3f130 真实场景: 工位无 user)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-3f130', '3f130', NULL, NULL)`).Error)

	svc := NewWorkstationDeviceService(db)
	result, err := svc.GetADDevicesByWorkstations(context.Background(), []string{"ws-3f130"})
	require.NoError(t, err)
	assert.Empty(t, result["ws-3f130"], "未绑 user 的工位应返回空 AD 设备列表")
}

func TestGetADDevicesByWorkstations_UserNotInAD(t *testing.T) {
	db := setupBatchTestDB(t)

	// 工位绑了用户, 但用户在 AD 中不存在
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-1', '3f130', 'u-1', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u-1', 'alice', 'Alice', NULL, NULL)`).Error)

	svc := NewWorkstationDeviceService(db)
	result, err := svc.GetADDevicesByWorkstations(context.Background(), []string{"ws-1"})
	require.NoError(t, err)
	assert.Empty(t, result["ws-1"], "用户无 AD 记录应返回空")
}

func TestGetADDevicesByWorkstations_BasicMatch(t *testing.T) {
	db := setupBatchTestDB(t)

	// 工位 → 用户 → AD 用户 → AD 计算机 (managed_by 匹配)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-1', '3f130', 'u-1', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u-1', 'alice', 'Alice', NULL, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_ad_user VALUES ('alice', 'CN=Alice,DC=test', NULL)`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO sys_ad_computer VALUES ('ad-1234567890', 'SN-1', 'PC-Alice', 'AA:BB:CC:DD:EE:FF', '10.1.1.1', 'Win11', 'CN=Alice,DC=test', NULL, '2026-07-21', NULL)
	`).Error)

	svc := NewWorkstationDeviceService(db)
	result, err := svc.GetADDevicesByWorkstations(context.Background(), []string{"ws-1"})
	require.NoError(t, err)
	require.Len(t, result["ws-1"], 1, "managed_by 匹配应返回 1 台 AD 设备")
	assert.Equal(t, "ad-1234567890", *result["ws-1"][0].ADComputerID)
	assert.Equal(t, "PC-Alice", *result["ws-1"][0].DeviceName)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", *result["ws-1"][0].MACAddress)
}

func TestGetAssetDevicesByWorkstations_EmptyInput(t *testing.T) {
	db := setupBatchTestDB(t)
	svc := NewWorkstationDeviceService(db)

	result, err := svc.GetAssetDevicesByWorkstations(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetAssetDevicesByWorkstations_NoNickname(t *testing.T) {
	db := setupBatchTestDB(t)

	// 用户 nickname 为空 (无法匹配资产)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-1', '3f130', 'u-1', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u-1', 'alice', '', NULL, NULL)`).Error)

	svc := NewWorkstationDeviceService(db)
	result, err := svc.GetAssetDevicesByWorkstations(context.Background(), []string{"ws-1"})
	require.NoError(t, err)
	assert.Empty(t, result["ws-1"], "无 nickname 的用户应返回空资产列表")
}

func TestGetAssetDevicesByWorkstations_BasicMatch(t *testing.T) {
	db := setupBatchTestDB(t)

	// 工位 → 用户 (nickname=张三) → 资产 (nowuser_name=张三)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-1', '3f130', 'u-1', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u-1', 'zhangsan', '张三', NULL, NULL)`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO ops_asset VALUES ('a-1234567890', 'SN-001', 'PC-Model', 'PC-Type', 'AA:BB:CC:DD:EE:FF', NULL, '张三', '研发部', '10.1.1.1', '10.1.1.1', NULL)
	`).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO ops_asset VALUES ('a-2345678901', 'SN-002', 'PC-Model2', 'PC-Type2', 'AA:BB:CC:DD:EE:GG', NULL, '李四', '研发部', '10.1.1.2', '10.1.1.2', NULL)
	`).Error)

	svc := NewWorkstationDeviceService(db)
	result, err := svc.GetAssetDevicesByWorkstations(context.Background(), []string{"ws-1"})
	require.NoError(t, err)
	require.Len(t, result["ws-1"], 1, "应只返回 张三 的 1 台资产")
	assert.Equal(t, "a-1234567890", *result["ws-1"][0].AssetID)
	assert.Equal(t, "张三", *result["ws-1"][0].ResponsibleUser)
	assert.Equal(t, models.DeviceSourceAsset, result["ws-1"][0].DeviceSource)
}

func TestGetAssetDevicesByWorkstations_MultipleWorkstations(t *testing.T) {
	db := setupBatchTestDB(t)

	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-1', '3f130', 'u-1', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-2', '3f131', 'u-2', NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_workstation VALUES ('ws-3', '3f132', 'u-3', NULL)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u-1', 'u1', '张三', NULL, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u-2', 'u2', '李四', NULL, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u-3', 'u3', '王五', NULL, NULL)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO ops_asset VALUES ('a-1111111111', 'SN-1', NULL, NULL, NULL, NULL, '张三', NULL, NULL, NULL, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset VALUES ('a-2222222222', 'SN-2', NULL, NULL, NULL, NULL, '李四', NULL, NULL, NULL, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset VALUES ('a-3333333333', 'SN-3', NULL, NULL, NULL, NULL, '王五', NULL, NULL, NULL, NULL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_asset VALUES ('a-4444444444', 'SN-4', NULL, NULL, NULL, NULL, '张三', NULL, NULL, NULL, NULL)`).Error)

	svc := NewWorkstationDeviceService(db)

	result, err := svc.GetAssetDevicesByWorkstations(context.Background(), []string{"ws-1", "ws-2", "ws-3"})
	require.NoError(t, err)
	assert.Len(t, result["ws-1"], 2, "张三名下 2 台资产")
	assert.Len(t, result["ws-2"], 1, "李四名下 1 台资产")
	assert.Len(t, result["ws-3"], 1, "王五名下 1 台资产")
}

// GetPhysicalDevicesByWorkstations 的核心 SQL 在 SQLite 上无法运行 (PG 语法),
// 端到端测试需在 PG 环境执行。这里只验证空输入 / 表缺失边界。
func TestGetPhysicalDevicesByWorkstations_EmptyInput(t *testing.T) {
	db := setupBatchTestDB(t)
	svc := NewWorkstationDeviceService(db)

	result, err := svc.GetPhysicalDevicesByWorkstations(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, result, "空输入应返回空 map")
}

func TestGetPhysicalDevicesByWorkstations_MissingTableFails(t *testing.T) {
	// 故意不建 sys_workstation 表, 让 workstation 查询报错
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	svc := NewWorkstationDeviceService(db)
	_, err = svc.GetPhysicalDevicesByWorkstations(context.Background(), []string{"ws-1"})
	assert.Error(t, err, "缺表应报错 (批量查 workstation 失败)")
}