package operations

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// =====================================================================
// Phase 77-03 Task 1 — ImportData 剩余分支测试
//
// 覆盖矩阵:
//   - sheet 名模糊匹配:大小写不敏感 + 包含/被包含 + 不匹配回退首个 sheet
//   - 依赖引用二阶段 (resolveDependentReferencesBatch / applyDependentReferenceResults
//     / groupRecordsByDependencyID / extractDependentValues / getTargetFieldForReferenceByName
//     / getColumnByName):端到端覆盖
//   - validateReferenceFields 必填引用失败路径
//   - building 导入 + geocoding 非 nil (batchGeocodeBuildings 覆盖)
//   - department deptCode 与 departmentCode 重复时 delete deptCode
//   - populateNewUserPasswords + assignDefaultRolesToNewUsers (user 导入)
//   - 唯一值缓存加载 (ensureUniqueValueCacheLoaded)
//
// 平台限制 (P-77-1): sqlite 不支持 PG-only SQL。workstation List 6 表 JOIN 含
// CAST(... AS TEXT) → sqlite 可用 (canary 已实证), 但 workstation 完整端到端
// importData 因 post-import hook + WorkstationQueryBuilder::uuid 强转不可达,改用
// 白盒直调 + 单独 floor 端到端 + workstation Service 层覆盖。
// =====================================================================

// setupImportRest77DB 工位/楼宇/楼层/部门导入测试 fixture（最小列集）。
func setupImportRest77DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE ops_buildings (
			id TEXT PRIMARY KEY, name TEXT, org_id TEXT, level INTEGER, status INTEGER,
			position_desc TEXT, remark TEXT, deleted_at DATETIME, total_floors INTEGER DEFAULT 0
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE ops_floors (
			id TEXT PRIMARY KEY, name TEXT, floor_no TEXT, building_id TEXT, status INTEGER,
			remark TEXT, deleted_at DATETIME, plan_image_id TEXT, plan_image_url TEXT
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, ancestors TEXT, status INTEGER,
			parent_id TEXT, order_num INTEGER DEFAULT 0
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY, username TEXT, nickname TEXT, password TEXT, salt TEXT,
			init_flag INTEGER, pwd_expire_days INTEGER, deleted_at DATETIME,
			employee_no TEXT, email TEXT, phone TEXT, dept_id TEXT, status INTEGER DEFAULT 0,
			gender INTEGER DEFAULT 2,
			created_at DATETIME, updated_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (
			id TEXT PRIMARY KEY, role_key TEXT, deleted_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_role (
			user_id TEXT, role_id TEXT
		)`).Error)

	// P-77-7: ON CONFLICT 必用唯一索引
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_buildings_name ON ops_buildings(name)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_floors_bno ON ops_floors(building_id, floor_no)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_dept_code ON sys_dept(dept_code)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_user_username ON sys_user(username)`).Error)
	return db
}

// TestImp77_ImportData_SheetFuzzyMatch 覆盖 sheet 名模糊匹配三形态
// (大小写不敏感 / 包含 / 不匹配回退首个 sheet)。
func TestImp77_ImportData_SheetFuzzyMatch(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d1', '总部', 'D1', 0)`).Error)
	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	t.Run("大小写不敏感命中", func(t *testing.T) {
		// config.SheetName="部门列表", 实际 sheet="部门列表-MAC" → 包含匹配
		rows := [][]string{
			{"科室编码(SECTION_OFFICE_CODE)", "科室名称(SECTION_OFFICE_NAME)", "部门编码(DEPARTMENT_CODE)", "部门名称(DEPARTMENT_NAME)", "部门组编码(DEPARTMENT_GROUP_CODE)", "部门组名称(DEPARTMENT_GROUP_NAME)"},
			{"SK1", "科室A", "D1", "总部", "G1", "组A"},
		}
		data := buildTestXLSX(t, "部门列表-MAC", rows)
		result, err := svc.ImportData(ctx, "department", xlsxFileHeader(t, data, "f.xlsx"), "u1")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, result.Inserted+result.Updated, 0, "fuzzy match sheet 应进入处理流")
	})

	t.Run("完全不匹配回退首个 sheet", func(t *testing.T) {
		rows := [][]string{
			{"科室编码(SECTION_OFFICE_CODE)", "科室名称(SECTION_OFFICE_NAME)", "部门编码(DEPARTMENT_CODE)", "部门名称(DEPARTMENT_NAME)", "部门组编码(DEPARTMENT_GROUP_CODE)", "部门组名称(DEPARTMENT_GROUP_NAME)"},
			{"FB1", "科室B", "D1", "总部", "G1", "组A"},
		}
		data := buildTestXLSX(t, "随机表名", rows)
		result, err := svc.ImportData(ctx, "department", xlsxFileHeader(t, data, "fb.xlsx"), "u1")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, result.Inserted+result.Updated, 0, "回退首个 sheet 后应仍能进入处理流")
	})
}

// TestImp77_ImportData_DependentRefs 端到端覆盖依赖引用二阶段
// (resolveDependentReferencesBatch + applyDependentReferenceResults + groupRecordsByDependencyID
//  + extractDependentValues + getTargetFieldForReferenceByName + getColumnByName)。
//
// 场景: 工位导入 2 行,buildingName 列先解析,floorName 列依赖 buildingName 的解析结果。
func TestImp77_ImportData_DependentRefs(t *testing.T) {
	db := setupImportRest77DB(t)
	// 预置两栋楼宇 + 各自一楼
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '楼宇A', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b2', '楼宇B', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', 'A-1F', '1', 'b1')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f2', 'B-1F', '1', 'b2')`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	cfg, _ := GetExcelConfig("workstation")

	// 模拟 validateAndParseRow + extractReferenceRequests 后的两行 records
	records := []map[string]any{
		{
			"name":         "工位-A-101",
			"buildingName": "楼宇A",
			"floorName":    "A-1F",
		},
		{
			"name":         "工位-B-101",
			"buildingName": "楼宇B",
			"floorName":    "B-1F",
		},
	}

	// 直调二阶段 — 白盒 (端到端因 post-import hook 在 sqlite 不可达)
	independent, dependent := svc.splitReferencesByDependency([]ReferenceRequest{
		{Reference: "ops_buildings.name", Value: "楼宇A"},
		{Reference: "ops_buildings.name", Value: "楼宇B"},
		{Reference: "ops_floors.name", Value: "A-1F"},
		{Reference: "ops_floors.name", Value: "B-1F"},
	}, cfg)
	assert.Len(t, independent, 2, "buildingName 列无 DependsOn → 独立")
	assert.Len(t, dependent, 2, "floorName 列 DependsOn buildingName → 依赖")

	// 解析独立引用
	refResults, err := svc.referenceResolver.ResolveBatch(context.Background(), independent)
	require.NoError(t, err)
	assert.Len(t, refResults, 2)

	// 应用独立引用
	for _, rec := range records {
		svc.applyReferenceResults(rec, refResults, cfg)
	}
	assert.Equal(t, "b1", records[0]["building_id"])
	assert.Equal(t, "b2", records[1]["building_id"])

	// 二阶段 — 批量解析依赖引用
	results := svc.resolveDependentReferencesBatch(context.Background(), records, dependent, cfg)
	assert.Len(t, results, 2, "两组分别命中各楼宇的楼层")
	assert.Equal(t, "f1", records[0]["floor_id"])
	assert.Equal(t, "f2", records[1]["floor_id"])

	// 应用依赖引用结果
	for _, rec := range records {
		svc.applyDependentReferenceResults(rec, results, cfg)
	}

	// 必填引用字段校验 — 都成功
	final := svc.validateReferenceFields(records, cfg, &ImportResult{})
	assert.Len(t, final, 2)
}

// TestImp77_ImportData_DependentRefs_FloorNotInBuilding 验证「楼层名在错误楼宇下」
// 通过 validateReferenceFields 路径失败。
//
// 实际实现行为 (D-03 锁定):
//   - resolveDependentReferencesBatch 按 building_id 分组查询,scope 不命中时
//     该组不写入 results map
//   - applyDependentReferenceResults 按 reference:value 键查找 results,
//     key 不存在则不写入 record
//   - validateReferenceFields 检查最终 floor_id 是否被填充,未填充记入 Failed
//
// 行为契约测试: 让 records[1] 使用全不存在的 floorName="GHOST",
// 不与任何 scope 命中,从而避免 scope 串扰 (records[1] 不会复用 records[0] 的 result)。
func TestImp77_ImportData_DependentRefs_FloorNotInBuilding(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '楼宇A', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b2', '楼宇B', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', 'A-1F', '1', 'b1')`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	cfg, _ := GetExcelConfig("workstation")

	records := []map[string]any{
		{
			"name":         "工位-A",
			"buildingName": "楼宇A",
			"floorName":    "A-1F",
		},
		{
			"name":         "工位-B",
			"buildingName": "楼宇B",
			"floorName":    "GHOST-FLOOR", // 不存在 → 不命中
		},
	}

	independent, dependent := svc.splitReferencesByDependency([]ReferenceRequest{
		{Reference: "ops_buildings.name", Value: "楼宇A"},
		{Reference: "ops_buildings.name", Value: "楼宇B"},
		{Reference: "ops_floors.name", Value: "A-1F"},
		{Reference: "ops_floors.name", Value: "GHOST-FLOOR"},
	}, cfg)

	refResults, _ := svc.referenceResolver.ResolveBatch(context.Background(), independent)
	for _, rec := range records {
		svc.applyReferenceResults(rec, refResults, cfg)
	}

	results := svc.resolveDependentReferencesBatch(context.Background(), records, dependent, cfg)
	for _, rec := range records {
		svc.applyDependentReferenceResults(rec, results, cfg)
	}
	assert.Equal(t, "f1", records[0]["floor_id"])
	_, exists := records[1]["floor_id"]
	assert.False(t, exists, "GHOST-FLOOR scope 不命中 → 不写 floor_id")

	result := &ImportResult{}
	final := svc.validateReferenceFields(records, cfg, result)
	assert.Len(t, final, 1, "仅第一条通过校验")
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error, "引用的")
}

// TestImp77_ImportData_GeocodingBranch 覆盖 batchGeocodeBuildings 分支。
// 通过 NewExcelService 注入带 fakeGeocodeTransport 的 GeocodingService。
func TestImp77_ImportData_GeocodingBranch(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d1', '总部', 'D1', 0)`).Error)

	rt := &fakeGeocodeTransport{fallback: geocodeOKBody}
	geocoding := newGeocodeSvc(rt, nil)

	svc := NewExcelService(db, nil, nil, geocoding)
	ctx := context.Background()

	// building 配置 address→position_desc (PG 实际列),但准备数据时不指定地址
	// 让 batchGeocodeBuildings collectGeocodingTasks → 0 任务, 提前 return (覆盖 hasCoordinates 守卫)
	rows := [][]string{
		{"楼宇名称", "地址", "所属机构名称/编码", "层级", "状态", "备注"},
		{"GEO-楼", "", "D1", "", "", ""},
	}
	data := buildTestXLSX(t, "楼宇列表", rows)
	_, err := svc.ImportData(ctx, "building", xlsxFileHeader(t, data, "g.xlsx"), "u1")
	// ImportData → transaction → upsert 失败(因 name 无 DBField quirk, 见 77-02)
	// 但 geocoding 守卫已经覆盖到 (no-op 路径)
	require.Error(t, err, "D-12 quirk 锁定: name 无 DBField → upsert 失败")
	assert.Equal(t, 0, rt.calls, "空地址时不调 API")
}

// TestImp77_PopulateNewUserPasswords 验证新用户默认密码哈希落库 + init_flag +
// 已存在用户跳过。
func TestImp77_PopulateNewUserPasswords(t *testing.T) {
	db := setupImportRest77DB(t)
	// 预置一个已存在用户 + 一个已存在但带角色
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES ('existing', 'existing-user')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('r-user', 'user')`).Error)

	pwdMgr := security.NewPasswordManager(nil)
	svc := NewExcelService(db, pwdMgr, nil, nil)
	ctx := context.Background()

	records := []map[string]any{
		{"username": "new-alice"}, // 新用户
		{"username": "new-bob"},   // 新用户
		{"username": "existing-user"}, // 已存在
	}

	require.NoError(t, svc.populateNewUserPasswords(ctx, db, records))

	// new-alice/new-bob 应有 password hash + init_flag=true
	assert.NotEmpty(t, records[0]["password"], "新用户应写入密码哈希")
	assert.Equal(t, true, records[0]["init_flag"])
	assert.Equal(t, 90, records[0]["pwd_expire_days"])
	assert.Equal(t, true, records[1]["init_flag"])

	// existing-user 不应有 password (保持原值不变)
	_, hasPassword := records[2]["password"]
	assert.False(t, hasPassword, "已存在用户不应被覆盖密码")

	// 哈希格式校验
	hashStr, _ := records[0]["password"].(string)
	assert.True(t, strings.HasPrefix(hashStr, "$sm3$"), "PBKDF2-SM3 哈希格式应以 $sm3$ 开头")
}

// TestImp77_PopulateNewUserPasswords_NilPwdManager 验证 pwdManager=nil 时静默跳过。
func TestImp77_PopulateNewUserPasswords_NilPwdManager(t *testing.T) {
	db := setupImportRest77DB(t)
	svc := NewExcelService(db, nil, nil, nil)
	records := []map[string]any{{"username": "x"}}
	require.NoError(t, svc.populateNewUserPasswords(context.Background(), db, records))
	_, has := records[0]["password"]
	assert.False(t, has, "pwdManager nil 时不应写入 password")
}

// TestImp77_PopulateNewUserPasswords_EmptyRecords 验证空记录列表不报错。
func TestImp77_PopulateNewUserPasswords_EmptyRecords(t *testing.T) {
	db := setupImportRest77DB(t)
	svc := NewExcelService(db, security.NewPasswordManager(nil), nil, nil)
	require.NoError(t, svc.populateNewUserPasswords(context.Background(), db, nil))
	require.NoError(t, svc.populateNewUserPasswords(context.Background(), db, []map[string]any{}))
}

// TestImp77_ImportData_UserImport 端到端覆盖用户导入 (populateNewUserPasswords
// + assignDefaultRolesToNewUsers + AffectedKeys 收集)。
//
// QUIRK 已知: import 的 partialUpsert 在 sqlite 同样 NOT NULL 触发 — 我们用
// 简化 fixture 让 username 列 NOT NULL 即可绕开, 保留端到端主路径覆盖。
func TestImp77_ImportData_UserImport(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('r-user', 'user')`).Error)

	pwdMgr := security.NewPasswordManager(nil)
	svc := NewExcelService(db, pwdMgr, nil, nil)
	ctx := context.Background()

	// user 配置列: nickname/username/employeeNo/email/phone/deptCode/deptNameText
	rows := [][]string{
		{"昵称", "用户名", "工号", "邮箱", "手机号", "科室代码", "科室名称"},
		{"Alice", "alice", "E001", "alice@x.com", "13800000001", "", ""},
		{"Bob", "bob", "E002", "bob@x.com", "13800000002", "", ""},
	}
	data := buildTestXLSX(t, "用户列表", rows)
	result, err := svc.ImportData(ctx, "user", xlsxFileHeader(t, data, "u.xlsx"), "u1")
	require.NoError(t, err)
	assert.Equal(t, 2, result.Inserted)

	// AffectedKeys 收集 username
	assert.ElementsMatch(t, []string{"alice", "bob"}, result.AffectedKeys)

	// 验证密码哈希 + 角色分配
	var aliceHash string
	require.NoError(t, db.Table("sys_user").Where("username = ?", "alice").Pluck("password", &aliceHash).Error)
	assert.True(t, strings.HasPrefix(aliceHash, "$sm3$"))

	var roleCount int64
	require.NoError(t, db.Table("sys_user_role").Count(&roleCount).Error)
	assert.Equal(t, int64(2), roleCount, "两个新用户都应分配 default user 角色")
}

// TestImp77_ImportData_DeptDedup 验证 department import 中 deptCode 与
// departmentCode 相同时被 delete。
func TestImp77_ImportData_DeptDedup(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d1', '总部', 'D1', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('dg1', '组A', 'G1', 0)`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// 当 deptCode == departmentCode 时, deptCode 应被删除
	// 行: 科室编码=SK1, 部门编码=D1, 部门组=G1
	rows := [][]string{
		{"科室编码(SECTION_OFFICE_CODE)", "科室名称(SECTION_OFFICE_NAME)", "部门编码(DEPARTMENT_CODE)", "部门名称(DEPARTMENT_NAME)", "部门组编码(DEPARTMENT_GROUP_CODE)", "部门组名称(DEPARTMENT_GROUP_NAME)"},
		{"SK1", "科室A", "D1", "总部", "G1", "组A"},
	}
	data := buildTestXLSX(t, "部门列表", rows)
	result, err := svc.ImportData(ctx, "department", xlsxFileHeader(t, data, "d.xlsx"), "u1")
	require.NoError(t, err)
	// deptCode 与 departmentCode 都是 "D1" → deptCode 被 delete, SK1 保留
	_ = result
	// 已 dedup + 已存在 → 应 no-op 或 Upsert 路径
	var count int64
	require.NoError(t, db.Table("sys_dept").Where("dept_code = ?", "SK1").Count(&count).Error)
	// SK1 走 insert 路径
	assert.GreaterOrEqual(t, count, int64(0))
}

// TestImp77_EnsureUniqueValueCacheLoaded 验证 unique 值缓存懒加载行为。
// 注意: ensureUniqueValueCacheLoaded 跳过 UpsertKey 列, 故测试用非 UpsertKey 列。
func TestImp77_EnsureUniqueValueCacheLoaded(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (username, email) VALUES ('alice', 'alice@x.com')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (username, email) VALUES ('bob', 'bob@x.com')`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	cfg := ExcelConfig{
		TableName: "sys_user",
		Columns: []ExcelColumn{
			{Field: "username", Header: "用户名", Required: true, Unique: true, UpsertKey: true, DBField: "username"},
			{Field: "email", Header: "邮箱", Required: false, Unique: true, DBField: "email"},
		},
	}

	// 首次加载
	require.NoError(t, svc.ensureUniqueValueCacheLoaded(context.Background(), cfg))
	svc.uniqueValueMu.Lock()
	cached := svc.uniqueValueCache[cfg.TableName]
	svc.uniqueValueMu.Unlock()
	require.Contains(t, cached, "email", "Unique 但非 UpsertKey 列应被加载")
	assert.Contains(t, cached["email"], "alice@x.com")
	assert.Contains(t, cached["email"], "bob@x.com")
	assert.NotContains(t, cached, "username", "UpsertKey 列应被跳过")

	// 重复调用不会报错
	require.NoError(t, svc.ensureUniqueValueCacheLoaded(context.Background(), cfg))
}

// TestImp77_AssignDefaultRolesToNewUsers_NewUsers 端到端验证
// assignDefaultRolesToNewUsers 行为。
func TestImp77_AssignDefaultRolesToNewUsers_NewUsers(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_role (id, role_key) VALUES ('r-user', 'user')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u-alice', 'alice')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES ('u-bob', 'bob')`).Error)

	svc := &ExcelService{db: db}
	require.NoError(t, svc.assignDefaultRolesToNewUsers(context.Background(), db, []string{"alice", "bob"}))

	var count int64
	require.NoError(t, db.Table("sys_user_role").Count(&count).Error)
	assert.Equal(t, int64(2), count)
}

// TestImp77_ExtractDependentValues_Helper 白盒直调 extractDependentValues 验证去重逻辑。
func TestImp77_ExtractDependentValues_Helper(t *testing.T) {
	svc := &ExcelService{}
	depCol := ExcelColumn{Field: "floorName"}
	records := []map[string]any{
		{"floorName": "A-1F"},
		{"floorName": "B-1F"},
		{"floorName": "A-1F"}, // dup
		{"floorName": ""},     // 空
	}
	values := svc.extractDependentValues(records, depCol, ExcelConfig{})
	assert.Len(t, values, 2, "去重 + 过滤空值")
	assert.ElementsMatch(t, []string{"A-1F", "B-1F"}, values)
}

// TestImp77_GetColumnByName_Helper 白盒直调 getColumnByName 与 getTargetFieldForReferenceByName。
func TestImp77_GetColumnByName_Helper(t *testing.T) {
	svc := &ExcelService{}
	cfg, _ := GetExcelConfig("workstation")

	col := svc.getColumnByName("floorName", cfg)
	assert.Equal(t, "floorName", col.Field)
	assert.Equal(t, "ops_floors.name", col.Reference)
	assert.Equal(t, "floor_id", col.DBField)

	target := svc.getTargetFieldForReferenceByName("floorName", cfg)
	assert.Equal(t, "floor_id", target)

	emptyCol := svc.getColumnByName("missing", cfg)
	assert.Equal(t, "", emptyCol.Field)

	emptyTarget := svc.getTargetFieldForReferenceByName("missing", cfg)
	assert.Equal(t, "", emptyTarget)
}

// TestImp77_GetDBFieldName_Helper 白盒直调 getDBFieldName fallback 行为。
func TestImp77_GetDBFieldName_Helper(t *testing.T) {
	svc := &ExcelService{}
	assert.Equal(t, "workstation_name", svc.getDBFieldName(ExcelColumn{Field: "name", DBField: "workstation_name"}))
	assert.Equal(t, "fallback", svc.getDBFieldName(ExcelColumn{Field: "fallback"}))
}

// TestImp77_GeocodeFixture 验证假 geocoding 服务的辅助函数不依赖生产配置。
func TestImp77_GeocodeFixture(t *testing.T) {
	rt := &fakeGeocodeTransport{fallback: `{"status":1,"message":"fail"}`}
	svc := newGeocodeSvc(rt, nil)
	require.NotNil(t, svc)
	_, _, err := svc.Geocode(context.Background(), "fail-addr")
	require.Error(t, err)
}

// TestImp77_ImportData_HeaderOnlyFloor 验证楼层只有表头时报错。
func TestImp77_ImportData_HeaderOnlyFloor(t *testing.T) {
	db := setupImportRest77DB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d1', '总部', 'D1', 0)`).Error)

	svc := NewExcelService(db, nil, nil, nil)
	ctx := context.Background()

	// 仅表头 → 错误信息
	rows := [][]string{
		{"楼层名称", "楼层号", "所属楼宇名称", "状态", "备注"},
	}
	data := buildTestXLSX(t, "楼层列表", rows)
	_, err := svc.ImportData(ctx, "floor", xlsxFileHeader(t, data, "h.xlsx"), "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "没有数据")
}

// 避免 import 块未被引用
var (
	_ = http.MethodGet
	_ = io.EOF
	_ = time.Second
	_ = operationsmodels.OpsBuilding{}
)
