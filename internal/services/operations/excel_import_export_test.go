package operations

import (
	"bytes"
	"context"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// =====================================================================
// Phase 74-07: ExcelService 端到端（GenerateTemplate / ImportData / ExportData）
// + 楼宇地理编码 hook + 工位主设备导入后处理。
// 内存 xlsx 经 multipart.FileHeader 走与 handler 完全一致的入口。
// =====================================================================

// buildTestXLSX 生成单 sheet 内存 Excel（rows[0] 为表头）。
func buildTestXLSX(t *testing.T, sheetName string, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	_, err := f.NewSheet(sheetName)
	require.NoError(t, err)
	require.NoError(t, f.DeleteSheet("Sheet1"))
	for r, row := range rows {
		for c, v := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			require.NoError(t, err)
			require.NoError(t, f.SetCellValue(sheetName, cell, v))
		}
	}
	data, err := f.WriteToBuffer()
	require.NoError(t, err)
	return data.Bytes()
}

// xlsxFileHeader 把字节流包装成 *multipart.FileHeader（与上传入口同构）。
func xlsxFileHeader(t *testing.T, data []byte, name string) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", name)
	require.NoError(t, err)
	_, err = fw.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	form, err := multipart.NewReader(&buf, w.Boundary()).ReadForm(32 << 20)
	require.NoError(t, err)
	require.Len(t, form.File["file"], 1)
	return form.File["file"][0]
}

func seedImportDept(t *testing.T, db *gorm.DB, id, name, code string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status, order_num)
		 VALUES (?, ?, ?, '', 0, 0)`, id, name, code).Error)
}

// createImportIndexes 补建 ON CONFLICT 目标列的唯一索引。
// BatchUpsert 走 ON CONFLICT (UpsertKey/UniqueKeys)，生产 PG 由迁移建唯一约束，
// sqlite AutoMigrate 不会自动生成 → 不建索引会报
// "ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint"。
func createImportIndexes(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_buildings_name ON ops_buildings(name)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_floors_bno ON ops_floors(building_id, floor_no)`).Error)
	// QUIRK: building 配置 address 列 DBField=position_desc 是 PG 生产列名，
	// OpsBuilding 模型（sqlite AutoMigrate 来源）列名是 address → 补列对齐 PG schema。
	require.NoError(t, db.Exec(`ALTER TABLE ops_buildings ADD COLUMN position_desc TEXT`).Error)
}

func TestExcelService_GenerateTemplates(t *testing.T) {
	svc := NewExcelService(nil, nil, nil, nil)

	for _, entityType := range []string{"building", "floor", "workstation", "department", "serverRoom", "dedicatedLine", "infoPoint"} {
		t.Run(entityType, func(t *testing.T) {
			f, err := svc.GenerateTemplate(entityType)
			require.NoError(t, err)
			defer f.Close()

			config, _ := GetExcelConfig(entityType)
			rows, err := f.GetRows(config.SheetName)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(rows), 2, "模板应含表头+示例行")

			// 某行应包含全部表头（说明行存在时表头整体下移）
			joined := strings.Join(rows[0], "|") + strings.Join(rows[1], "|")
			assert.Contains(t, joined, config.Columns[0].Header)
		})
	}

	// 未知类型
	_, err := svc.GenerateTemplate("nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的实体类型")
}

func TestExcelService_ImportBuilding_Quirk(t *testing.T) {
	db := newPhotoTestDB(t)
	createImportIndexes(t, db)
	seedImportDept(t, db, "d1", "总部", "D1")
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	rows := [][]string{
		{"楼宇名称", "地址", "所属机构名称/编码", "层级", "状态", "备注"},
		{"楼宇甲", "某路1号", "D1", "具体楼宇", "正常", "备注x"},
		{"楼宇乙", "", "D1", "城市级汇总", "", ""},
	}
	// QUIRK（疑似生产级 bug，D-12 记录不修）：building 配置的 name 列
	// （UpsertKey）无 DBField，prepareRecordsForUpsert 对无 DBField 列直接
	// continue → INSERT 列集不含 name → NOT NULL 约束失败（PG 同样 23502）。
	// 事务回滚 → 库中无残留行。
	_, err := svc.ImportData(ctx, "building", xlsxFileHeader(t, buildTestXLSX(t, "楼宇列表", rows), "b.xlsx"), "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "批量保存数据失败")

	var count int64
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).Count(&count).Error)
	assert.Zero(t, count, "事务应整体回滚")
}

func TestExcelService_ImportBuilding_Errors(t *testing.T) {
	db := newPhotoTestDB(t)
	createImportIndexes(t, db)
	seedImportDept(t, db, "d1", "总部", "D1")
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// 行级错误：缺必填 / 超长 / 未知机构。
	// QUIRK: 非法 Options 值（如状态填"异常状态值"）不报错 —— parseFieldValue
	// 反查失败时返回原字符串透传（D-12 记录不修），故此处不用非法枚举行。
	longRemark := strings.Repeat("长", 600)
	rows := [][]string{
		{"楼宇名称", "地址", "所属机构名称/编码", "层级", "状态", "备注"},
		{"", "x", "D1", "", "", ""},        // 缺楼宇名称
		{"超长备注楼", "", "D1", "", "", longRemark}, // MaxLength 500 超限
		{"幽灵机构楼", "", "NOPE", "", "", ""},  // 引用解析失败
	}
	result, err := svc.ImportData(ctx, "building", xlsxFileHeader(t, buildTestXLSX(t, "楼宇列表", rows), "bad.xlsx"), "u1")
	require.NoError(t, err)
	assert.Equal(t, 3, result.Failed)
	require.Len(t, result.Errors, 3)
	assert.Contains(t, result.Errors[1].Error, "长度超过最大限制")
	assert.Contains(t, result.Errors[2].Error, "不存在")
	var count int64
	require.NoError(t, db.Model(&operationsmodels.OpsBuilding{}).Count(&count).Error)
	assert.Zero(t, count, "全部行非法时不应入库")

	// 未知实体类型
	_, err = svc.ImportData(ctx, "nope", xlsxFileHeader(t, buildTestXLSX(t, "S", [][]string{{"a"}, {"1"}}), "x.xlsx"), "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的实体类型")

	// 非 Excel 内容
	_, err = svc.ImportData(ctx, "building", xlsxFileHeader(t, []byte("not-an-excel"), "x.xlsx"), "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析Excel文件失败")

	// 只有表头 → 无数据
	headOnly := buildTestXLSX(t, "楼宇列表", rows[:1])
	_, err = svc.ImportData(ctx, "building", xlsxFileHeader(t, headOnly, "h.xlsx"), "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "没有数据")

	// sheet 名不匹配 → 回退首个 sheet 后解析仍执行（随后因 name quirk 落入同一 upsert 错误）
	fallback := buildTestXLSX(t, "随便什么名字", [][]string{
		{"楼宇名称", "地址", "所属机构名称/编码", "层级", "状态", "备注"},
		{"回退楼", "", "D1", "", "", ""},
	})
	_, err = svc.ImportData(ctx, "building", xlsxFileHeader(t, fallback, "f.xlsx"), "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "批量保存数据失败")
}

func TestExcelService_ImportFloor(t *testing.T) {
	db := newPhotoTestDB(t)
	createImportIndexes(t, db)
	seedImportDept(t, db, "d1", "总部", "D1")
	buildingID, _ := seedBuildingFloor(t, db, "fx-b") // fx-b / fx-b-F1
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	rows := [][]string{
		{"楼层名称", "楼层号", "所属楼宇名称", "状态", "备注"},
		{"二楼", "2", "fx-b", "正常", "r"},
		{"三楼", "3", "fx-b", "停用", ""},
	}
	result, err := svc.ImportData(ctx, "floor", xlsxFileHeader(t, buildTestXLSX(t, "楼层列表", rows), "f.xlsx"), "u1")
	require.NoError(t, err)
	assert.Equal(t, 2, result.Inserted)

	var floors []operationsmodels.OpsFloor
	require.NoError(t, db.Where("building_id = ? AND name = ?", buildingID, "二楼").Find(&floors).Error)
	require.Len(t, floors, 1)
	assert.Equal(t, "2", floors[0].FloorNo)

	var stopped int64
	require.NoError(t, db.Model(&operationsmodels.OpsFloor{}).
		Where("building_id = ? AND name = ? AND status = ?", buildingID, "三楼", 1).Count(&stopped).Error)
	assert.Equal(t, int64(1), stopped, "停用选项应映射为 status=1")

	// 二次导入：UniqueKeys(building_id, floor_no) 命中唯一索引 → 走更新路径
	rows[1][0] = "二楼改"
	result, err = svc.ImportData(ctx, "floor", xlsxFileHeader(t, buildTestXLSX(t, "楼层列表", rows), "f2.xlsx"), "u1")
	require.NoError(t, err)
	assert.Equal(t, 2, result.Updated)
	var renamed operationsmodels.OpsFloor
	require.NoError(t, db.Where("building_id = ? AND floor_no = ?", buildingID, "2").First(&renamed).Error)
	assert.Equal(t, "二楼改", renamed.Name)
}

func TestExcelService_ImportDepartment_ThreeLevel(t *testing.T) {
	db := newPhotoTestDB(t)
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	rows := [][]string{
		{"科室编码(SECTION_OFFICE_CODE)", "科室名称(SECTION_OFFICE_NAME)", "部门编码(DEPARTMENT_CODE)",
			"部门名称(DEPARTMENT_NAME)", "部门组编码(DEPARTMENT_GROUP_CODE)", "部门组名称(DEPARTMENT_GROUP_NAME)",
			"负责人", "联系电话", "邮箱", "显示顺序", "是否外部机构", "状态", "备注"},
		{"S1", "科室一", "DP1", "部门一", "DG1", "部门组一", "张三", "010", "a@b.c", "1", "否", "正常", "rk"},
		{"S2", "科室二", "DP1", "部门一", "DG1", "部门组一", "", "", "", "2", "是", "停用", ""},
		{"X1", "自同级", "X1", "部门一", "DG1", "部门组一", "", "", "", "", "", "", ""}, // 部门编码==科室编码 → 删除 deptCode 走部门级 upsert
	}
	result, err := svc.ImportData(ctx, "department", xlsxFileHeader(t, buildTestXLSX(t, "部门列表", rows), "d.xlsx"), "u1")
	require.NoError(t, err)
	// QUIRK: X1 行（部门编码==科室编码）在导入前删除 deptCode（UpsertKey）→
	// partialUpsert 对无冲突键的记录静默跳过 → 只入库 S1/S2 两条科室。
	// X1 本体由 processThreeLevelDepartments 以"部门级"创建。
	assert.Equal(t, 2, result.Inserted)

	// 三级结构：1 组 + 1 部门 + 3 科室 = 5 行
	var deptCount int64
	require.NoError(t, db.Model(&sysmodels.Department{}).Where("deleted_at IS NULL").Count(&deptCount).Error)
	assert.Equal(t, int64(5), deptCount)

	var group, dept sysmodels.Department
	require.NoError(t, db.Where("dept_code = ?", "DG1").First(&group).Error)
	require.NoError(t, db.Where("dept_code = ?", "DP1").First(&dept).Error)
	assert.Empty(t, group.ParentID, "部门组为顶级")
	require.NotNil(t, dept.ParentID)
	assert.Equal(t, group.ID, *dept.ParentID)

	var section sysmodels.Department
	require.NoError(t, db.Where("dept_code = ?", "S1").First(&section).Error)
	require.NotNil(t, section.ParentID)
	assert.Equal(t, dept.ID, *section.ParentID)
	assert.Contains(t, section.Ancestors, group.ID, "ancestors 链应包含组 ID")
	// QUIRK: leader/phone/email/status 列均无 DBField → prepareRecordsForUpsert
	// 丢弃 → 科室的负责人等列不落库（仅 order_num/remark/is_external_org 持久化）。
	assert.Nil(t, section.Leader)
	assert.Equal(t, 1, section.OrderNum)

	// X1：部门编码==科室编码分支 → 科室行被跳过，X1 以"部门级"挂在组下
	var x1 sysmodels.Department
	require.NoError(t, db.Where("dept_code = ?", "X1").First(&x1).Error)
	require.NotNil(t, x1.ParentID)
	assert.Equal(t, group.ID, *x1.ParentID)

	// 重复导入 → PartialUpdate 更新路径
	rows[1][1] = "科室一改"
	result, err = svc.ImportData(ctx, "department", xlsxFileHeader(t, buildTestXLSX(t, "部门列表", rows), "d2.xlsx"), "u1")
	require.NoError(t, err)
	assert.Equal(t, 2, result.Updated)
	var renamed sysmodels.Department
	require.NoError(t, db.Where("dept_code = ?", "S1").First(&renamed).Error)
	assert.Equal(t, "科室一改", renamed.DeptName)
}

func TestExcelService_ExportBuildingAndFloor(t *testing.T) {
	db := newPhotoTestDB(t)
	seedImportDept(t, db, "d1", "总部", "D1")
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	buildingID, floorID := seedBuildingFloor(t, db, "ex-b")
	_ = floorID
	require.NoError(t, db.Create(&operationsmodels.OpsBuilding{Name: "ex-b2", OrgID: "d1", Level: 2}).Error)

	// 楼宇导出：name 过滤命中 1 条 + 状态列文本化
	f, err := svc.ExportData(ctx, "building", map[string]any{"name": "ex-b2"})
	require.NoError(t, err)
	defer f.Close()
	rows, err := f.GetRows("楼宇列表")
	require.NoError(t, err)
	require.Len(t, rows, 2, "表头 + 1 数据行")
	assert.Contains(t, rows[0][0], "楼宇名称")
	assert.Equal(t, "ex-b2", rows[1][0])

	// 全量导出
	f2, err := svc.ExportData(ctx, "building", map[string]any{"status": 0})
	require.NoError(t, err)
	defer f2.Close()
	rows, _ = f2.GetRows("楼宇列表")
	assert.Len(t, rows, 3)

	// 楼层导出：buildingName JOIN 填充
	f3, err := svc.ExportData(ctx, "floor", nil)
	require.NoError(t, err)
	defer f3.Close()
	rows, _ = f3.GetRows("楼层列表")
	require.Len(t, rows, 2)
	assert.Equal(t, buildingID, rows[1][1], "所属楼宇ID列")
	assert.Equal(t, "ex-b", rows[1][2], "所属楼宇名 JOIN 填充")

	// 未知类型
	_, err = svc.ExportData(ctx, "nope", nil)
	require.Error(t, err)
}

func TestExcelService_GeocodingHook(t *testing.T) {
	rt := &fakeGeocodeTransport{
		responses: map[string]string{
			"地址甲": geocodeOKBody,
			"地址丙": `{"status":1,"message":"bad"}`,
		},
		fallback: geocodeOKBody,
	}
	geo := newGeocodeSvc(rt, nil)
	svc := NewExcelService(nil, nil, nil, geo)
	ctx := context.Background()

	records := []map[string]any{
		{"address": "地址甲"},
		{"address": ""},
		{"address": "地址乙"}, // 走 fallback
		{"address": "地址丁", "longitude": 1.0, "latitude": 2.0}, // 已有坐标跳过
		{"address": "地址丙"}, // 解析失败
		{"name": "no-address"},
	}
	svc.batchGeocodeBuildings(ctx, records)

	assert.Equal(t, 116.404, records[0]["longitude"])
	assert.Equal(t, 39.915, records[0]["latitude"])
	assert.Equal(t, 116.404, records[2]["longitude"])
	assert.Equal(t, 2.0, records[3]["latitude"], "已有坐标的记录不应被覆盖")
	assert.Nil(t, records[4]["longitude"], "解析失败的记录不应写入坐标")
	assert.Nil(t, records[5]["longitude"])

	// 纯函数路径（新集合：0/2 已有坐标，3 已有坐标，仅地址丙 + 新地址计入）
	tasks := svc.collectGeocodingTasks(append(records, map[string]any{"address": "地址戊"}))
	require.Len(t, tasks, 2)
	assert.True(t, svc.hasCoordinates(map[string]any{"longitude": 1, "latitude": 2}))
	assert.False(t, svc.hasCoordinates(map[string]any{"longitude": 1}))
}

// stubDeviceSvc 只实现 postImportWorkstationPrimaryDevice 需要的方法。
type stubDeviceSvc struct {
	WorkstationDeviceService
	calls []string
	err   error
}

func (s *stubDeviceSvc) SetPrimaryAndSaveBySerial(_ context.Context, workstationID, serial string, _ *SetPrimaryAndSaveRequest) error {
	s.calls = append(s.calls, workstationID+":"+serial)
	return s.err
}

func TestExcelService_PostImportPrimaryDeviceHook(t *testing.T) {
	db := newPhotoTestDB(t)
	buildingID, floorID := seedBuildingFloor(t, db, "hk-b")
	_ = buildingID
	ws := &sysmodels.Workstation{WorkstationName: "W1"}
	ws.FloorID = &floorID
	require.NoError(t, db.Create(ws).Error)

	ctx := context.Background()
	records := []map[string]any{
		{"name": "W1"},                                        // 无序列号 → 跳过
		{"deviceSerial": "SN1", "floor_id": floorID, "name": "W1"}, // 成功
		{"deviceSerial": "SN2", "floor_id": "", "name": ""},       // 缺定位信息
		{"deviceSerial": "SN3", "floor_id": floorID, "name": "ghost"}, // 工位未找到
		{"deviceSerial": "SN4", "floor_id": floorID, "name": "W1"},    // 设备同步失败
	}

	result := &ImportResult{Errors: make([]ImportError, 0)}
	svc := NewExcelService(db, nil, nil, nil).WithDeviceService(&stubDeviceSvc{err: assert.AnError})
	svc.postImportWorkstationPrimaryDevice(ctx, records, result)

	// SN1/SN4 设备服务恒错；SN2 缺定位；SN3 工位未找到 → 4 条错误按行序排列
	assert.Equal(t, 4, result.Failed)
	require.Len(t, result.Errors, 4)
	assert.Contains(t, result.Errors[0].Error, "主设备同步失败")
	assert.Contains(t, result.Errors[1].Error, "缺少工位定位信息")
	assert.Contains(t, result.Errors[2].Error, "工位未找到")
	assert.Contains(t, result.Errors[3].Error, "主设备同步失败")

	// 无设备服务注入 → 静默跳过
	plain := NewExcelService(db, nil, nil, nil)
	plain.postImportWorkstationPrimaryDevice(ctx, records, &ImportResult{})
}
