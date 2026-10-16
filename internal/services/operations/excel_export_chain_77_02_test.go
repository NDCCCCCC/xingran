package operations

// Phase 77 Plan 02: excel_service 导出链 sqlite 直测。
//
// 覆盖策略(D-08 切分 / D-06 全内存生成 / D-07 结构断言):
//   - legacy ExportData 分支 = 未注册进 GetExportConfig 的类型(user/asset/
//     reconciliationExceptionRule)。department/serverRoom 等在 GetExportConfig
//     有映射,走 excelExportServiceImpl 新导出链,不属于本 plan 的 legacy 路径
//     (D-03 有据:excel_export_config.go GetExportConfig 的 map 键清单)。
//   - queryData 的 sys_dept 字段名特判分支(name→dept_name/code→dept_code)无法经
//     ExportData 触达(department 已被新导出链截流),用白盒直调覆盖。
//   - appendWorkstationDeviceSheets 直接白盒调用:workstation 主表导出走
//     WorkstationQueryBuilder,含 ::uuid/::text PG-only 强转,sqlite 必报错
//     (P-77-1 现行为,由 TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL 文档化),
//     故不能经 PLAN 设想的 svc.ExportData("workstation") 入口触达追加链。
//   - 物理链路 sheet 经 physErr != nil 降级(P-77-10):GetPhysicalDevices* 的
//     DISTINCT ON/REGEXP_REPLACE CTE 在 sqlite 报错已被 77-01
//     TestWSD77_GetPhysicalDevicesByWorkstations_FrontSegment 实证。
//   - D-06:全部 xlsx 经 excelize.NewFile() 内存生成,零 testdata 二进制;
//   - D-07:断言仅 sheet 名/表头行/行数 + 抽查关键单元格(AD sheet 序列号列,
//     mergeBySerial 合并主键),禁全量逐单元格快照。

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// exp77HeaderIndex 按列 Field 定位导出表中该列的下标(断言锚点随列序漂移自动跟随)。
func exp77HeaderIndex(cfg ExcelConfig, field string) int {
	for i, c := range cfg.Columns {
		if c.Field == field {
			return i
		}
	}
	return -1
}

// exp77Headers 把 config.Columns 的 Header 序列拉平成期望表头行。
func exp77Headers(cfg ExcelConfig) []string {
	out := make([]string, 0, len(cfg.Columns))
	for _, c := range cfg.Columns {
		out = append(out, c.Header)
	}
	return out
}

// exp77CellByHeader 按中文表头文字取单元格值(require 定位失败即失败,行宽被
// excelize 尾部空白裁剪时报错并给出可读信息)。
func exp77CellByHeader(t *testing.T, cfg ExcelConfig, header string, row []string) string {
	t.Helper()
	for i, c := range cfg.Columns {
		if c.Header == header {
			require.Greaterf(t, len(row), i,
				"表头 %q 位于第 %d 列,但该数据行在 excelize 输出中被裁剪到 %d 宽", header, i+1, len(row))
			return row[i]
		}
	}
	require.Failf(t, "column missing", "表头 %q 不存在于 %s 配置", header, cfg.SheetName)
	return ""
}

// ===========================================================================
// Task 1: legacy ExportData 主路径(user / asset / reconciliationExceptionRule)
// ===========================================================================

// setupExp77LegacyDB 建 legacy 导出所需的最小 sqlite 表集(queryData 走
// SELECT * + WHERE deleted_at IS NULL,列集精简到配置实际引用列;
// 与 setupBatchTestDB 同风格。不建 status 列以避免在集成层重复锁定
// 「int64 值与 Options int 键不匹配 → 枚举文本化退化」行为——该现行为已在
// TestExp77_FormatCellValue 白盒层文档化)。
func setupExp77LegacyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			nickname TEXT,
			username TEXT,
			employee_no TEXT,
			email TEXT,
			phone TEXT,
			dept_id TEXT,
			deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_asset (
			id TEXT PRIMARY KEY,
			devicesn TEXT,
			device_model_name TEXT,
			device_type_name TEXT,
			machine_ip TEXT,
			user_id TEXT,
			deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_reconciliation_exception (
			id TEXT PRIMARY KEY,
			name TEXT,
			ip_range TEXT,
			conflict_types TEXT,
			exception_actions TEXT,
			severity_override TEXT,
			reason TEXT,
			deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			dept_code TEXT,
			status INTEGER DEFAULT 0,
			deleted_at DATETIME
		)`).Error)
	return db
}

func exp77Exec(t *testing.T, db *gorm.DB, sql string) {
	t.Helper()
	require.NoError(t, db.Exec(sql).Error)
}

// TestExp77_ExportData_LegacyTypes 三类 legacy 类型(user/asset/
// reconciliationExceptionRule)的 ExportData 结构断言:sheet 名存在、
// 表头行 == config.Columns[].Header 序列、数据行数 == 在册种子数(D-07)。
func TestExp77_ExportData_LegacyTypes(t *testing.T) {
	db := setupExp77LegacyDB(t)
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// ---- user:3 行种子其中 1 行软删 → 2 条数据行 ----
	exp77Exec(t, db, `INSERT INTO sys_user VALUES ('u1','张三','zhangsan','E001','z@x.com','13800000001',NULL,NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_user VALUES ('u2','李四','lisi',NULL,NULL,NULL,NULL,NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_user VALUES ('u3','王五','wangwu',NULL,NULL,NULL,NULL,'2026-07-01')`)

	userCfg, ok := GetExcelConfig("user")
	require.True(t, ok)
	fu, err := svc.ExportData(ctx, "user", nil)
	require.NoError(t, err)
	defer fu.Close()
	uRows, err := fu.GetRows(userCfg.SheetName)
	require.NoError(t, err)
	require.Len(t, uRows, 3, "表头 + 2 条在册用户(软删用户不计)")
	assert.Equal(t, exp77Headers(userCfg), uRows[0], "表头行必须等于 config.Columns[].Header 序列")
	assert.Equal(t, "zhangsan", exp77CellByHeader(t, userCfg, "用户名", uRows[1]))
	assert.Equal(t, "李四", exp77CellByHeader(t, userCfg, "昵称", uRows[2]))

	// ---- asset:3 行种子其中 1 行软删 → 2 条数据行 ----
	exp77Exec(t, db, `INSERT INTO ops_asset VALUES ('a1','SN-A1','ThinkPad T14','笔记本','10.2.0.1',NULL,NULL)`)
	exp77Exec(t, db, `INSERT INTO ops_asset VALUES ('a2','SN-A2',NULL,NULL,'10.2.0.2',NULL,NULL)`)
	exp77Exec(t, db, `INSERT INTO ops_asset VALUES ('a3','SN-DEAD',NULL,NULL,NULL,NULL,'2026-07-01')`)

	assetCfg, ok := GetExcelConfig("asset")
	require.True(t, ok)
	fa, err := svc.ExportData(ctx, "asset", map[string]any{})
	require.NoError(t, err)
	defer fa.Close()
	aRows, err := fa.GetRows(assetCfg.SheetName)
	require.NoError(t, err)
	require.Len(t, aRows, 3, "表头 + 2 台在册资产")
	assert.Equal(t, exp77Headers(assetCfg), aRows[0])
	assert.Equal(t, "SN-A1", exp77CellByHeader(t, assetCfg, "设备序列号", aRows[1]),
		"deviceSN 为 UpsertKey,是资产导出的锚点列")

	// ---- reconciliationExceptionRule:第三类 legacy 类型(name 可过滤) ----
	exp77Exec(t, db, `INSERT INTO sys_reconciliation_exception VALUES ('r1','外呼误报豁免','10.0.0.0/8','B,C','no_alert','','预警风暴豁免',NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_reconciliation_exception VALUES ('r2','盘点静默规则','192.168.0.0/16','D','skip_severity','','停用期豁免',NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_reconciliation_exception VALUES ('r3','已删规则','172.16.0.0/12','B','no_notice','','过期','2026-07-01')`)

	ruleCfg, ok := GetExcelConfig("reconciliationExceptionRule")
	require.True(t, ok)
	fr, err := svc.ExportData(ctx, "reconciliationExceptionRule", nil)
	require.NoError(t, err)
	defer fr.Close()
	rRows, err := fr.GetRows(ruleCfg.SheetName)
	require.NoError(t, err)
	require.Len(t, rRows, 3, "表头 + 2 条在册例外规则")
	assert.Equal(t, exp77Headers(ruleCfg), rRows[0])
	assert.Equal(t, "外呼误报豁免", exp77CellByHeader(t, ruleCfg, "规则名称", rRows[1]))

	// 未知类型兜底分支
	_, err = svc.ExportData(ctx, "no-such-type", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的实体类型")
}

// TestExp77_ExportData_QueryFilters queryData 过滤链:name LIKE / 空 param 跳过 /
// 非 dept 表未知 code 列报错 / sys_dept 字段名特判(name→dept_name,code→dept_code,
// 仅白盒可达,见文件头注)。
func TestExp77_ExportData_QueryFilters(t *testing.T) {
	db := setupExp77LegacyDB(t)
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	exp77Exec(t, db, `INSERT INTO sys_reconciliation_exception VALUES ('q1','外呼误报豁免','10.0.0.0/8','B,C','no_alert','','预警风暴豁免',NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_reconciliation_exception VALUES ('q2','盘点静默规则','192.168.0.0/16','D','skip_severity','','停用期豁免',NULL)`)

	ruleCfg, ok := GetExcelConfig("reconciliationExceptionRule")
	require.True(t, ok)
	nameIdx := exp77HeaderIndex(ruleCfg, "name")

	fHit, err := svc.ExportData(ctx, "reconciliationExceptionRule", map[string]any{"name": "外呼"})
	require.NoError(t, err)
	defer fHit.Close()
	hitRows, err := fHit.GetRows(ruleCfg.SheetName)
	require.NoError(t, err)
	require.Len(t, hitRows, 2, "name LIKE 只应命中 外呼误报豁免 1 行")
	assert.Equal(t, "外呼误报豁免", hitRows[1][nameIdx])

	fAll, err := svc.ExportData(ctx, "reconciliationExceptionRule", map[string]any{"name": "", "code": ""})
	require.NoError(t, err)
	defer fAll.Close()
	allRows, err := fAll.GetRows(ruleCfg.SheetName)
	require.NoError(t, err)
	require.Len(t, allRows, 3, "空串过滤值不应缩小结果集(表头 + 2 条规则)")

	_, err = svc.ExportData(ctx, "reconciliationExceptionRule", map[string]any{"code": "ANY"})
	require.Error(t, err, "code 默认映射为 code 列,recon 表无该列应报错(现行为)")
	assert.Contains(t, err.Error(), "查询数据失败")

	// ---- sys_dept 字段名特判分支(nameField→dept_name / codeField→dept_code,
	// 仅白盒可达,见文件头注)----
	exp77Exec(t, db, `INSERT INTO sys_dept VALUES ('d1','总部门市','D1',0,NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_dept VALUES ('d2','总部中心','D9',0,NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_dept VALUES ('d3','分部营业部','D1',1,NULL)`)
	exp77Exec(t, db, `INSERT INTO sys_dept VALUES ('d4','总部基地','D1',0,'2026-07-01')`)

	deptQueryConfig := ExcelConfig{TableName: "sys_dept"}
	data, err := svc.queryData(ctx, deptQueryConfig, map[string]any{"name": "总部", "code": "D1", "status": 0})
	require.NoError(t, err)
	require.Len(t, data, 1, "dept_name LIKE 总部 AND dept_code LIKE D1 AND status=0 应只命中 d1(d3 停用、d4 软删)")
	assert.Equal(t, "D1", data[0]["dept_code"])

	allDepts, err := svc.queryData(ctx, deptQueryConfig, nil)
	require.NoError(t, err)
	require.Len(t, allDepts, 3, "无过滤条件返回全部在册部门(软删 d4 不计)")

	unmatched, err := svc.queryData(ctx, deptQueryConfig, map[string]any{"name": "不存在关键词"})
	require.NoError(t, err)
	assert.Empty(t, unmatched, "LIKE 不命中应返回空集而非报错")
}

// ===========================================================================
// Task 1: 写行 / 格式化辅助函数
// ===========================================================================

// TestExp77_FormatCellValue 四分支:Options 反查命中(DBField 作键)/ createdAt+
// updatedAt 时间格式化 / nil 早退 / 默认 Sprintf 兜底。附加锁定「int64 扫描值
// 与 Options int 键类型不符 → 反查落空退化为数字文本」的现行为(D-03 无据判修,
// 生产 PG 扫描同样返回数值类型,不在本 plan 修改面内)。
func TestExp77_FormatCellValue(t *testing.T) {
	svc := &ExcelService{}

	optsCol := ExcelColumn{
		Field:   "nbfStatus",
		DBField: "nbf_status",
		Options: map[interface{}]string{int(0): "否", int(1): "是"},
	}
	// DBField 非空时以 DBField 取值
	assert.Equal(t, "否", svc.formatCellValue(optsCol, map[string]any{"nbf_status": int(0)}))
	assert.Equal(t, "是", svc.formatCellValue(optsCol, map[string]any{"nbf_status": int(1)}))
	// 未命中 Options 时回落 Sprintf,值格式化为数字文本(同层覆盖反查 miss)
	assert.Equal(t, "0", svc.formatCellValue(optsCol, map[string]any{"nbf_status": int64(0)}))

	plainCol := ExcelColumn{Field: "orderNum"}
	assert.Equal(t, "42", svc.formatCellValue(plainCol, map[string]any{"orderNum": 42}),
		"无 DBField 时以 Field 取值 + Sprintf 兜底")

	ts := time.Date(2026, 8, 27, 10, 30, 5, 0, time.UTC)
	timeCol := ExcelColumn{Field: "createdAt"}
	assert.Equal(t, "2026-08-27 10:30:05", svc.formatCellValue(timeCol, map[string]any{"createdAt": ts}))
	updCol := ExcelColumn{Field: "updatedAt"}
	assert.Equal(t, "2026-08-27 10:30:05", svc.formatCellValue(updCol, map[string]any{"updatedAt": ts}))
	// 时间型列但值不是 time.Time → 掉到 Sprintf 兜底而非 panic
	assert.Equal(t, "not-a-time", svc.formatCellValue(timeCol, map[string]any{"createdAt": "not-a-time"}))

	// nil 值(包括 key 缺失)最早早退,不受 Options 影响
	assert.Equal(t, "", svc.formatCellValue(optsCol, map[string]any{}))
}

// TestExp77_WriteInstructions 说明行写入:非空说明逐行合并写入首列,空串行跳过;
// 全部为空串时零合并;instructions 列表为空时整体早退零写入。
// 注:当前没有任何 ExcelConfig 配置 Instructions(GREP 实证),GenerateTemplate
// 入口不可达非空分支,故白盒直调。
func TestExp77_WriteInstructions(t *testing.T) {
	svc := &ExcelService{}
	sheet := "模板说明"
	cols := ExcelConfigs["department"].Columns // 13 列,保证 MergeCell 有合法区间

	f := excelize.NewFile()
	_, err := f.NewSheet(sheet)
	require.NoError(t, err)

	svc.writeInstructions(f, sheet, cols, []string{"第一行说明", "", "第三行说明"})

	rows, err := f.GetRows(sheet)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	assert.Equal(t, "第一行说明", rows[0][0],
		"QUIRK-77-2 回归: 首个说明必须留在 A1(修复前被第二次跨行合并吞掉)")

	merges, err := f.GetMergeCells(sheet)
	require.NoError(t, err)
	require.Len(t, merges, 2, "两行非空说明各自独占一行合并(空串行跳过)")
	starts := make([]string, 0, len(merges))
	for _, m := range merges {
		starts = append(starts, m.GetStartAxis())
		assert.True(t, strings.HasPrefix(m.GetEndAxis(), "M"),
			"合并区间 end 锚点列必须是第 13 列 M,不得跨行(实际 %s)", m.GetEndAxis())
	}
	assert.ElementsMatch(t, []string{"A1", "A3"}, starts,
		"空串说明行应跳过合并(两处,且遍历顺序无关)")

	sawThird := false
	for _, r := range rows {
		if len(r) > 0 && r[0] == "第三行说明" {
			sawThird = true
		}
	}
	assert.True(t, sawThird, "第三行说明应独立成行存在")

	// 全部说明为空串 → 一行都不写
	fEmpty := excelize.NewFile()
	_, err = fEmpty.NewSheet(sheet)
	require.NoError(t, err)
	svc.writeInstructions(fEmpty, sheet, cols, []string{"", ""})
	merges2, err := fEmpty.GetMergeCells(sheet)
	require.NoError(t, err)
	assert.Empty(t, merges2, "全空说明不应产生任何合并单元格")

	// nil instructions → 整体早退(表头说明区零写入)
	fNil := excelize.NewFile()
	_, err = fNil.NewSheet(sheet)
	require.NoError(t, err)
	svc.writeInstructions(fNil, sheet, cols, nil)
	nilRows, err := fNil.GetRows(sheet)
	require.NoError(t, err)
	assert.Empty(t, nilRows, "nil 说明不应写入任何内容")
}

// TestExp77_GetExampleValue 示例值补尾部:Options 首值 / 常见 Field 映射抽查 /
// Required 未匹配回落 "示例" / 非必填未匹配回落 ""(Reference 字段不被示例器消费)。
func TestExp77_GetExampleValue(t *testing.T) {
	svc := &ExcelService{}

	// 单选项 Options → 返回该选项文本(确定性)
	optCol := ExcelColumn{Field: "whatever", Options: map[interface{}]string{"internet": "互联网专线"}}
	assert.Equal(t, "互联网专线", svc.getExampleValue(optCol))
	// 多选项 Options → 遍历序不确定,断言落在集合内(flake discipline)
	multiOpt := ExcelColumn{Field: "level", Options: map[interface{}]string{1: "城市级汇总", 2: "具体楼宇"}}
	assert.Contains(t, []string{"城市级汇总", "具体楼宇"}, svc.getExampleValue(multiOpt))

	assert.Equal(t, "示例A", svc.getExampleValue(ExcelColumn{Field: "name"}))
	assert.Equal(t, "研发部", svc.getExampleValue(ExcelColumn{Field: "deptName"}))
	assert.Equal(t, "110000", svc.getExampleValue(ExcelColumn{Field: "cityCode"}))
	assert.Equal(t, "CN123456789", svc.getExampleValue(ExcelColumn{Field: "serialNumber"}))

	assert.Equal(t, "示例", svc.getExampleValue(ExcelColumn{Field: "unknownField", Required: true}),
		"Required 且未匹配任何映射 → 示例占位")
	assert.Equal(t, "", svc.getExampleValue(ExcelColumn{Field: "unknownField"}),
		"非必填未匹配 → 空串")
	// Reference 列:示例器只看 Field/Options,Reference 不参与分支
	assert.Equal(t, "（请填写所属楼宇名称）",
		svc.getExampleValue(ExcelColumn{Field: "buildingId", Reference: "ops_buildings.id"}))
}

// ===========================================================================
// Task 2: appendWorkstationDeviceSheets 三 sheet 追加链(P-77-10 sqlite 降级)
// ===========================================================================

var exp77LastLogon = time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC)

// TestExp77_AppendSheets_MissingDeviceService 未注入 deviceService 的守卫分支。
func TestExp77_AppendSheets_MissingDeviceService(t *testing.T) {
	db := setupWSD77DB(t)
	svc := NewExcelService(db, nil, nil, nil)
	exportSvc := NewExcelExportService(db).(*excelExportServiceImpl)

	f := excelize.NewFile()
	err := svc.appendWorkstationDeviceSheets(context.Background(), f, exportSvc, map[string]any{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deviceService 未注入")
	assert.Len(t, f.GetSheetList(), 1, "守卫分支不得追加任何设备 sheet")
}

// TestExp77_AppendSheets_NoMatchingWorkstation 过滤后无工位 → 静默跳过无追加。
func TestExp77_AppendSheets_NoMatchingWorkstation(t *testing.T) {
	db := setupWSD77DB(t) // 全空表
	svc := NewExcelService(db, nil, nil, nil).WithDeviceService(NewWorkstationDeviceService(db))
	exportSvc := NewExcelExportService(db).(*excelExportServiceImpl)

	f := excelize.NewFile()
	err := svc.appendWorkstationDeviceSheets(context.Background(), f, exportSvc, map[string]any{})
	require.NoError(t, err, "工位列表为空应静默返回而非报错")
	assert.Len(t, f.GetSheetList(), 1, "无工位时不追加任何设备 sheet")
}

// TestExp77_ExportData_WorkstationAppendSheets 追加链主路径:D-07 结构断言 —
// 三 sheet 存在、表头行逐字相等、AD sheet 行数 = 种子命中数、序列号列抽查
// (mergeBySerial 的合并主键)。物理链路 sheet 断言 0 数据行但表头在
// (P-77-10:PG-only CTE 在 sqlite 报错走 physErr 降级,已被 77-01 实证)。
func TestExp77_ExportData_WorkstationAppendSheets(t *testing.T) {
	db := setupWSD77DB(t)
	// 77-01 fixture 为查询链精简,未建 AD enrichment 所需的 last_logon 列 → 补列
	require.NoError(t, db.Exec(`ALTER TABLE sys_ad_computer ADD COLUMN last_logon DATETIME`).Error)
	ctx := context.Background()

	// 种子:工位 → 用户 → AD 用户 → AD 计算机(managed_by 命中)+ 资产(nowuser_name 命中)
	seedWSD77Workstation(t, db, wsd77WS1, "3f130", wsd77User1)
	seedWSD77User(t, db, wsd77User1, "zhangsan", "张三", "")
	seedWSD77ADUser(t, db, "60000000-0000-0000-0000-000000000001", "zhangsan", wsd77DN1)
	seedWSD77ADComputer(t, db, wsd77ADComp1, "AD-PC-MANAGED", "SN-EXP-B1",
		"AA:BB:CC:DD:EE:01", "10.1.1.1", "Windows 11", wsd77DN1, "")
	require.NoError(t, db.Exec(`UPDATE sys_ad_computer SET last_logon = ? WHERE id = ?`,
		exp77LastLogon, wsd77ADComp1).Error)
	seedWSD77Asset(t, db, wsd77Asset1, "SN-EXP-A1", "ThinkPad T14", "笔记本",
		"AA:BB:CC:DD:EE:11", "张三", "研发部", "10.2.2.1")

	exportSvc := NewExcelExportService(db).(*excelExportServiceImpl)
	svc := NewExcelService(db, nil, nil, nil).WithDeviceService(NewWorkstationDeviceService(db))

	f := excelize.NewFile()
	err := svc.appendWorkstationDeviceSheets(ctx, f, exportSvc, map[string]any{})
	require.NoError(t, err)

	sheets := f.GetSheetList()
	assert.Contains(t, sheets, "AD设备")
	assert.Contains(t, sheets, "资产设备")
	assert.Contains(t, sheets, "物理链路设备")

	// ---- AD 设备 sheet ----
	adRows, err := f.GetRows("AD设备")
	require.NoError(t, err)
	require.Len(t, adRows, 2, "表头 + 1 台 managed_by 命中的 AD 设备")
	assert.Equal(t, []string{"工位名称", "ComputerName", "OS", "MAC", "Serial", "LastLogon"}, adRows[0])
	assert.Equal(t, "3f130", adRows[1][0], "工位名称列(enrichment batchGetWorkstationNames)")
	assert.Equal(t, "Windows 11", adRows[1][2], "OS 列(batchGetADEnrichment)")
	assert.Equal(t, "SN-EXP-B1", adRows[1][4], "序列号列锚点(mergeBySerial 合并主键,D-07 抽查)")
	assert.Equal(t, "2026-08-01 09:30:00", adRows[1][5], "LastLogon 列按 2006-01-02 15:04:05 格式化")

	// ---- 资产设备 sheet ----
	assetRows, err := f.GetRows("资产设备")
	require.NoError(t, err)
	require.Len(t, assetRows, 2, "表头 + 1 台 nowuser_name 命中的资产")
	assert.Equal(t, []string{"工位名称", "DeviceName", "Model", "Type", "IP", "ResponsibleUser"}, assetRows[0])
	assert.Equal(t, "10.2.2.1", assetRows[1][4], "IP 列(batchGetAssetEnrichment 读 machine_ip)")

	// ---- 物理链路设备 sheet(physErr 降级:A1 假设成立)----
	physRows, err := f.GetRows("物理链路设备")
	require.NoError(t, err)
	require.Len(t, physRows, 1, "sqlite 下物理链路查询必报错 → 降级为 0 数据行但表头保留")
	assert.Equal(t, []string{"工位名称", "设备名称", "序列号", "型号", "类型", "MAC", "IP地址",
		"责任人", "Port", "InfoPoint", "LastSeen", "Confidence"}, physRows[0])
}

// TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL 文档化工位主表导出现行为:
// WorkstationQueryBuilder 含 ::uuid/::text 强转,sqlite 解析必败 → ExportData
// 直接报错、追加链无机会执行(因此本 plan 用白盒直调覆盖 appendWorkstationDeviceSheets)。
func TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL(t *testing.T) {
	db := setupWSD77DB(t)
	seedWSD77Workstation(t, db, wsd77WS1, "3f130", "")
	svc := NewExcelService(db, nil, nil, nil).WithDeviceService(NewWorkstationDeviceService(db))

	f, err := svc.ExportData(context.Background(), "workstation", map[string]any{})
	require.Error(t, err, "P-77-1 现行为::uuid/::text 强转在 sqlite 必报错")
	assert.Nil(t, f)
	assert.Contains(t, err.Error(), "查询数据失败")
}

// TestExp77_QueryWorkstationIDsForExport_ParamTypes 直查 sys_workstation 的
// 参数应用矩阵:string LIKE(FilterMapping name→workstation_name)/ int 等值 /
// []interface{} IN / bool 等值 / nil 或空串跳过 / 未声明字段回退 paramKey 报错。
func TestExp77_QueryWorkstationIDsForExport_ParamTypes(t *testing.T) {
	db := setupWSD77DB(t)
	ctx := context.Background()

	seedWSD77Workstation(t, db, wsd77WS1, "3f130", "")
	seedWSD77Workstation(t, db, wsd77WS2, "3f131", "")
	seedWSD77Workstation(t, db, wsd77WSNoUsr, "XYZ-9", "")
	wsd77Exec(t, db,
		`INSERT INTO sys_workstation (id, workstation_name, status, user_id, deleted_at) VALUES (?, ?, 1, NULL, NULL)`,
		wsd77WSMiss, "STATUS-ONE")

	svc := &ExcelService{db: db} // exportService 形参已不再被实现依赖(excel_service.go:2220 注释)

	// nil 参数 → 全量 4 工位
	ids, err := svc.queryWorkstationIDsForExport(ctx, nil, nil)
	require.NoError(t, err)
	assert.Len(t, ids, 4)

	// string → FilterMapping workstation_name LIKE
	ids, err = svc.queryWorkstationIDsForExport(ctx, nil, map[string]any{"name": "3f1"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{wsd77WS1, wsd77WS2}, ids)

	// int 等值(status)
	ids, err = svc.queryWorkstationIDsForExport(ctx, nil, map[string]any{"status": 1})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{wsd77WSMiss}, ids)

	// bool 等值(false → status=0)
	ids, err = svc.queryWorkstationIDsForExport(ctx, nil, map[string]any{"status": false})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{wsd77WS1, wsd77WS2, wsd77WSNoUsr}, ids)

	// []interface{} IN(workstation_name 双命中)
	ids, err = svc.queryWorkstationIDsForExport(ctx, nil, map[string]any{
		"name": []interface{}{"3f130", "3f131"},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{wsd77WS1, wsd77WS2}, ids)

	// nil / 空串参数一律跳过
	ids, err = svc.queryWorkstationIDsForExport(ctx, nil, map[string]any{"name": "", "status": nil})
	require.NoError(t, err)
	assert.Len(t, ids, 4)

	// 未声明字段回退 paramKey → 不存在列,SQL 报错包一层定位信息
	_, err = svc.queryWorkstationIDsForExport(ctx, nil, map[string]any{"ghostParam": "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询 sys_workstation.id 失败")
}
