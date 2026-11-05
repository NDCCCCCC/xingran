package operations

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xuri/excelize/v2"
)

// createWorkstationExcelFile 帮助函数：用传入的表头和数据行构造一个多部件文件
func createWorkstationExcelFile(t *testing.T, headers []string, rows [][]string) *multipart.FileHeader {
	t.Helper()
	f := excelize.NewFile()
	sheetName := "工位列表"
	index, err := f.NewSheet(sheetName)
	require.NoError(t, err)
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	// 写入表头（第 1 行）
	for colIdx, h := range headers {
		cell, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		require.NoError(t, err)
		require.NoError(t, f.SetCellValue(sheetName, cell, h))
	}

	// 写入数据（从第 2 行开始）
	for rowIdx, row := range rows {
		for colIdx, val := range row {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			require.NoError(t, err)
			require.NoError(t, f.SetCellValue(sheetName, cell, val))
		}
	}

	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))

	// 包装为 multipart.FileHeader
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "workstations.xlsx")
	require.NoError(t, err)
	_, err = part.Write(buf.Bytes())
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(10<<20))

	return req.MultipartForm.File["file"][0]
}

// TestWorkstationImport_HeaderAliases 验证常用业务别名表头能正确匹配对应字段，避免错位
func TestWorkstationImport_HeaderAliases(t *testing.T) {
	cfg, ok := GetExcelConfig("workstation")
	require.True(t, ok)

	// 用户自定义别名且列序打乱：工位 | 所属部门 | 人员 | 所属楼宇 | 所属楼层
	userHeaders := []string{"工位", "所属部门", "人员", "所属楼宇", "所属楼层"}
	effective, matchedRequired := resolveColumnsByHeader(userHeaders, cfg)

	// 必填列是 name 和 floorName
	assert.Equal(t, 2, matchedRequired, "必填列 name 与 floorName 均应通过别名成功匹配")

	assert.Equal(t, "name", effective[0].Field, "第 0 列应映射到 name")
	assert.Equal(t, "deptName", effective[1].Field, "第 1 列 '所属部门' 应映射到 deptName")
	assert.Equal(t, "userName", effective[2].Field, "第 2 列 '人员' 应映射到 userName")
	assert.Equal(t, "buildingName", effective[3].Field, "第 3 列 '所属楼宇' 应映射到 buildingName")
	assert.Equal(t, "floorName", effective[4].Field, "第 4 列 '所属楼层' 应映射到 floorName")
}

// TestOtherModules_HeaderAliases 验证楼宇、楼层、机房设备、信息点模块常用别名表头能正确匹配，避免回退位置匹配
func TestOtherModules_HeaderAliases(t *testing.T) {
	// 1. 验证 floor 模块：用户常用的 "楼层" 和 "所属楼宇" 能匹配到 name 和 buildingName
	floorCfg, ok := GetExcelConfig("floor")
	require.True(t, ok)
	floorHeaders := []string{"楼层", "楼层编码", "所属楼宇"}
	effectiveFloor, matchedFloor := resolveColumnsByHeader(floorHeaders, floorCfg)
	assert.Equal(t, 3, matchedFloor, "楼层必填字段均应通过别名成功匹配")
	assert.Equal(t, "name", effectiveFloor[0].Field)
	assert.Equal(t, "floorNo", effectiveFloor[1].Field)
	assert.Equal(t, "buildingName", effectiveFloor[2].Field)

	// 2. 验证 building 模块：用户常用的 "楼宇"、"所属机构"、"详细地址" 能匹配
	bldCfg, ok := GetExcelConfig("building")
	require.True(t, ok)
	bldHeaders := []string{"楼宇", "所属机构", "详细地址"}
	effectiveBld, matchedBld := resolveColumnsByHeader(bldHeaders, bldCfg)
	assert.Equal(t, 2, matchedBld, "楼宇的 name 和 orgName 必填列均应通过别名匹配")
	assert.Equal(t, "name", effectiveBld[0].Field)
	assert.Equal(t, "orgName", effectiveBld[1].Field)
	assert.Equal(t, "address", effectiveBld[2].Field)

	// 3. 验证 roomDevice 模块："设备"、"所属机房" 能匹配
	rdCfg, ok := GetExcelConfig("roomDevice")
	require.True(t, ok)
	rdHeaders := []string{"设备", "设备编码", "所属机房"}
	effectiveRd, matchedRd := resolveColumnsByHeader(rdHeaders, rdCfg)
	assert.Equal(t, 3, matchedRd, "机房设备 3 个必填列均应匹配")
	assert.Equal(t, "name", effectiveRd[0].Field)
	assert.Equal(t, "deviceCode", effectiveRd[1].Field)
	assert.Equal(t, "roomName", effectiveRd[2].Field)

	// 4. 验证 infoPoint 模块："信息点"、"所属工位" 能匹配
	ipCfg, ok := GetExcelConfig("infoPoint")
	require.True(t, ok)
	ipHeaders := []string{"信息点", "所属工位"}
	effectiveIp, matchedIp := resolveColumnsByHeader(ipHeaders, ipCfg)
	assert.Equal(t, 2, matchedIp, "信息点必填列 name 和 workstationName 均应匹配")
	assert.Equal(t, "name", effectiveIp[0].Field)
	assert.Equal(t, "workstationName", effectiveIp[1].Field)
}

// TestWorkstationImport_PreciseErrorOnBuildingMissing 验证楼宇不存在时，报错精准指向楼宇，而不是误报楼层不存在
func TestWorkstationImport_PreciseErrorOnBuildingMissing(t *testing.T) {
	db := setupImportRest77DB(t)
	// 预置楼宇和楼层
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '真实大楼', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '1楼', '1', 'b1')`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// 上传表格：所属楼宇填写了不存在的“假大楼”
	headers := []string{"工位名称", "所属楼宇", "所属楼层", "所属部门", "所属人员"}
	rows := [][]string{
		{"W-001", "不存在的大楼", "1楼", "技术部", "张三"},
	}
	file := createWorkstationExcelFile(t, headers, rows)

	res, err := svc.ImportData(ctx, "workstation", file, "test-user-id")
	require.NoError(t, err)
	require.Equal(t, 1, res.Failed)
	require.Len(t, res.Errors, 1)

	// 错误字段应精准指向"所属楼宇"，且错误信息说明是楼宇不存在导致无法定位楼层
	errItem := res.Errors[0]
	assert.Equal(t, "所属楼宇", errItem.Field, "错误字段应为所属楼宇，而不是所属楼层")
	assert.Contains(t, errItem.Error, "引用的 所属楼宇 不存在")
	assert.Contains(t, errItem.Error, "不存在的大楼")
}

// TestWorkstationImport_DeptAndUserImportSuccess 验证工位导入时部门和人员能成功解析并写入数据库
func TestWorkstationImport_DeptAndUserImportSuccess(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.Workstation{}))
	// 补建工位唯一索引（对齐 PostgreSQL 生产约束）
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_workstations_floor_name ON sys_workstation(floor_id, workstation_name)`).Error)

	// 1. 预置基础数据
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '研发大楼', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '3楼', '3', 'b1')`).Error)
	// 部门：支持按 dept_name 或 dept_code 匹配
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d1', '信息技术部', 'TECH_DEPT', 0)`).Error)
	// 用户：支持按 username 或 nickname 匹配
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, status) VALUES ('u1', 'zhangsan', '张三', 0)`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// 2. 构造导入数据：
	// 行 1：填中文部门名称“信息技术部”，填中文姓名“张三”
	// 行 2：填部门代码“TECH_DEPT”，填登录账号“zhangsan”
	headers := []string{"工位名称", "所属楼宇", "所属楼层", "所属部门", "所属人员"}
	rows := [][]string{
		{"W-101", "研发大楼", "3楼", "信息技术部", "张三"},
		{"W-102", "研发大楼", "3楼", "TECH_DEPT", "zhangsan"},
	}
	file := createWorkstationExcelFile(t, headers, rows)

	res, err := svc.ImportData(ctx, "workstation", file, "admin")
	require.NoError(t, err)
	assert.Empty(t, res.Errors, "不应有校验错误: %+v", res.Errors)
	assert.Equal(t, 2, res.Inserted)

	// 3. 验证数据库中的记录
	var ws1, ws2 sysmodels.Workstation
	require.NoError(t, db.Where("workstation_name = ?", "W-101").First(&ws1).Error)
	assert.Equal(t, "b1", *ws1.BuildingID)
	assert.Equal(t, "f1", *ws1.FloorID)
	require.NotNil(t, ws1.DeptID, "W-101 的 dept_id 应该成功导入")
	assert.Equal(t, "d1", *ws1.DeptID)
	require.NotNil(t, ws1.UserID, "W-101 的 user_id 应该成功导入")
	assert.Equal(t, "u1", *ws1.UserID)
	assert.Equal(t, sysmodels.WorkstationStatus(1), ws1.Status, "分配了人员的工位状态应自动联动为占用(1)")

	require.NoError(t, db.Where("workstation_name = ?", "W-102").First(&ws2).Error)
	assert.Equal(t, "d1", *ws2.DeptID, "通过部门代码应成功导入 dept_id")
	assert.Equal(t, "u1", *ws2.UserID, "通过用户名应成功导入 user_id")
	assert.Equal(t, sysmodels.WorkstationStatus(1), ws2.Status)
}

// TestWorkstationImport_DeptCodePrecedenceAndMaintainState 验证部门代码优先于部门名称，且工位维护状态被严格保留
func TestWorkstationImport_DeptCodePrecedenceAndMaintainState(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.Workstation{}))
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_ws_floor_name2 ON sys_workstation(floor_id, workstation_name)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '大楼A', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '1F', '1', 'b1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d_code', '部门名称A', 'CODE_PRIORITY', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d_name', '部门名称B', 'OTHER_CODE', 0)`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// 1. 同时包含部门代码与所属部门列，验证以部门代码解析出的 d_code 为准
	// 2. 状态明确填写为“维护”，验证即使没填用户，也不会被错误打平为空闲
	headers := []string{"工位名称", "所属楼宇", "所属楼层", "部门代码", "所属部门", "状态"}
	rows := [][]string{
		{"WS-PRE", "大楼A", "1F", "CODE_PRIORITY", "部门名称B", "维护"},
	}
	file := createWorkstationExcelFile(t, headers, rows)

	res, err := svc.ImportData(ctx, "workstation", file, "admin")
	require.NoError(t, err)
	assert.Empty(t, res.Errors)
	assert.Equal(t, 1, res.Inserted)

	var ws sysmodels.Workstation
	require.NoError(t, db.Where("workstation_name = ?", "WS-PRE").First(&ws).Error)
	assert.Equal(t, "d_code", *ws.DeptID, "同时存在部门代码与部门名称时，必须以部门代码(CODE_PRIORITY -> d_code)为准")
	assert.Equal(t, sysmodels.WorkstationStatus(2), ws.Status, "维护状态应被完整保留，不能被强制改为 0 或 1")
}

// TestWorkstationImport_DeptCodeNameMatrix 覆盖「部门代码 / 所属部门」各种填写组合，
// 验证：两列同时填写时以部门代码为准；单填任一列都能导入；某一列解析失败时另一列可兜底。
func TestWorkstationImport_DeptCodeNameMatrix(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.Workstation{}))
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_ws_floor_name3 ON sys_workstation(floor_id, workstation_name)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '大楼A', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '1F', '1', 'b1')`).Error)
	// 两个不同的部门：名称与代码互不相同，便于区分最终命中哪一个
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d_alpha', 'Alpha部', 'CODE_A', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d_beta', 'Beta部', 'CODE_B', 0)`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	headers := []string{"工位名称", "所属楼宇", "所属楼层", "部门代码", "所属部门"}
	rows := [][]string{
		{"WS-1", "大楼A", "1F", "CODE_A", ""},          // 只填代码
		{"WS-2", "大楼A", "1F", "", "Beta部"},           // 只填名称
		{"WS-3", "大楼A", "1F", "CODE_A", "Beta部"},     // 同时填且指向不同部门 → 代码优先
		{"WS-4", "大楼A", "1F", "CODE_A", "Alpha部"},    // 同时填且指向同一部门 → 该部门
		{"WS-5", "大楼A", "1F", "NOT_EXIST", "Beta部"},  // 代码无效 + 名称有效 → 回退名称
		{"WS-6", "大楼A", "1F", "CODE_B", "NOT_EXIST"}, // 代码有效 + 名称无效 → 以代码为准
	}
	file := createWorkstationExcelFile(t, headers, rows)

	res, err := svc.ImportData(ctx, "workstation", file, "admin")
	require.NoError(t, err)
	assert.Empty(t, res.Errors, "所有组合都应导入成功: %+v", res.Errors)
	assert.Equal(t, 6, res.Inserted)

	expect := map[string]string{
		"WS-1": "d_alpha", // 只填代码 → 代码对应部门
		"WS-2": "d_beta",  // 只填名称 → 名称对应部门
		"WS-3": "d_alpha", // 同时填 → 部门代码优先
		"WS-4": "d_alpha", // 同时填且一致 → 该部门
		"WS-5": "d_beta",  // 代码无效 → 回退到名称
		"WS-6": "d_beta",  // 名称无效 → 以代码为准
	}
	for name, wantDept := range expect {
		var ws sysmodels.Workstation
		require.NoError(t, db.Where("workstation_name = ?", name).First(&ws).Error, name)
		require.NotNil(t, ws.DeptID, "%s 的 dept_id 应该已导入", name)
		assert.Equal(t, wantDept, *ws.DeptID, "%s 的部门归属错误", name)
	}
}

// TestWorkstationImport_DeptColumnIsolation 验证「部门代码」列与「所属部门」列的命中结果互不污染：
// 构造"某部门的名称恰好等于另一部门的代码"的交叉数据，确保两列各自命中正确的部门。
func TestWorkstationImport_DeptColumnIsolation(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.Workstation{}))
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_ws_floor_name4 ON sys_workstation(floor_id, workstation_name)`).Error)

	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '大楼A', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '1F', '1', 'b1')`).Error)
	// 交叉数据：d_x 的名称 == d_y 的代码
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d_x', 'CROSS', 'CODE_X', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d_y', 'Y部', 'CROSS', 0)`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	headers := []string{"工位名称", "所属楼宇", "所属楼层", "部门代码", "所属部门"}
	rows := [][]string{
		{"WS-CODE", "大楼A", "1F", "CROSS", ""}, // 「部门代码」列填 CROSS → 应命中代码为 CROSS 的 d_y
		{"WS-NAME", "大楼A", "1F", "", "CROSS"}, // 「所属部门」列填 CROSS → 应命中名称为 CROSS 的 d_x
	}
	file := createWorkstationExcelFile(t, headers, rows)

	res, err := svc.ImportData(ctx, "workstation", file, "admin")
	require.NoError(t, err)
	assert.Empty(t, res.Errors, "不应有校验错误: %+v", res.Errors)
	assert.Equal(t, 2, res.Inserted)

	var wsCode, wsName sysmodels.Workstation
	require.NoError(t, db.Where("workstation_name = ?", "WS-CODE").First(&wsCode).Error)
	require.NoError(t, db.Where("workstation_name = ?", "WS-NAME").First(&wsName).Error)

	require.NotNil(t, wsCode.DeptID)
	require.NotNil(t, wsName.DeptID)
	assert.Equal(t, "d_y", *wsCode.DeptID, "「部门代码」列应按 dept_code 命中 d_y")
	assert.Equal(t, "d_x", *wsName.DeptID, "「所属部门」列应按 dept_name 命中 d_x")
}
