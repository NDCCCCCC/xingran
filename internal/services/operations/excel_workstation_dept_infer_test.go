package operations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// 工位导入「部门填写不规范 → 按所属人员反推部门」回归测试。
//
// 复用 setupImportRest77DB(excel_import_rest_77_03_test.go:42) 的最小表结构
// 与 createWorkstationExcelFile(excel_workstation_import_fix_test.go:18) 的
// 内存 xlsx 构造，走与 HTTP 上传完全一致的 ImportData 入口。
//
// 用户决策（quick 261106-wsd）：仅兜底 —— 只有 deptCode/deptName 都没解析出
// dept_id 时才反推；显式填对的部门优先，不被覆盖。

// setupDeptInferDB 在既有 fixture 上补工位表、楼宇/楼层/部门/用户基础数据。
func setupDeptInferDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupImportRest77DB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.Workstation{}))
	require.NoError(t, db.Exec(
		`CREATE UNIQUE INDEX ux_test_ws_deptinfer ON sys_workstation(floor_id, workstation_name)`).Error)

	require.NoError(t, db.Exec(
		`INSERT INTO ops_buildings (id, name, org_id, level) VALUES ('b1', '研发大楼', 'D1', 2)`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO ops_floors (id, name, floor_no, building_id) VALUES ('f1', '3楼', '3', 'b1')`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d1', '信息技术部', 'TECH_DEPT', 0)`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, status) VALUES ('d2', '财务部', 'FIN_DEPT', 0)`).Error)
	return db
}

// deptInferHeaders 同时包含「所属部门」与「所属人员」，两者都走表头别名匹配。
var deptInferHeaders = []string{"工位名称", "所属楼宇", "所属楼层", "所属部门", "所属人员"}

// importDeptInferRows 构造并导入一张工位表，返回导入结果。
func importDeptInferRows(t *testing.T, db *gorm.DB, rows [][]string) *ImportResult {
	t.Helper()
	file := createWorkstationExcelFile(t, deptInferHeaders, rows)
	res, err := NewExcelService(db, nil, nil, nil).ImportData(context.Background(), "workstation", file, "test-user")
	require.NoError(t, err)
	return res
}

// readDeptInferWorkstation 按工位名称读取落库后的 dept_id / user_id。
func readDeptInferWorkstation(t *testing.T, db *gorm.DB, name string) (deptID, userID string) {
	t.Helper()
	var ws sysmodels.Workstation
	require.NoError(t, db.Where("workstation_name = ?", name).First(&ws).Error, name)
	if ws.DeptID != nil {
		deptID = *ws.DeptID
	}
	if ws.UserID != nil {
		userID = *ws.UserID
	}
	return deptID, userID
}

// TestWorkstationImport_DeptInferFromUser_Fallback
// 部门名称填错，但人员有效 → 用该用户的部门补全 dept_id。
func TestWorkstationImport_DeptInferFromUser_Fallback(t *testing.T) {
	db := setupDeptInferDB(t)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u1', 'zhangsan', '张三', 'd2', 0)`).Error)

	res := importDeptInferRows(t, db, [][]string{
		{"W-INFER-1", "研发大楼", "3楼", "不存在的部门名", "张三"},
	})
	assert.Empty(t, res.Errors, "不应有校验错误: %+v", res.Errors)
	assert.Equal(t, 1, res.Inserted)
	assert.Equal(t, 1, res.InferredDeptCount, "应记录 1 条由用户反推的部门")

	deptID, userID := readDeptInferWorkstation(t, db, "W-INFER-1")
	assert.Equal(t, "u1", userID, "人员应正常关联")
	assert.Equal(t, "d2", deptID, "dept_id 应由用户所属部门(财务部)反推补全")
}

// TestWorkstationImport_DeptInferFromUser_UserWithoutDept
// 用户 dept_id 为 NULL → 反推不到，不回填、不报错。
func TestWorkstationImport_DeptInferFromUser_UserWithoutDept(t *testing.T) {
	db := setupDeptInferDB(t)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u1', 'lisi', '李四', NULL, 0)`).Error)

	res := importDeptInferRows(t, db, [][]string{
		{"W-INFER-2", "研发大楼", "3楼", "不存在的部门名", "李四"},
	})
	assert.Empty(t, res.Errors, "部门反推失败不应产生导入错误: %+v", res.Errors)
	assert.Equal(t, 1, res.Inserted)
	assert.Equal(t, 0, res.InferredDeptCount)

	deptID, userID := readDeptInferWorkstation(t, db, "W-INFER-2")
	assert.Equal(t, "u1", userID, "人员仍应正常关联")
	assert.Empty(t, deptID, "用户无部门时不应臆造 dept_id")
}

// TestWorkstationImport_DeptInferFromUser_DirtyUserDept
// 用户 dept_id 是脏值（部门名称字符串 / 指向不存在的部门）→ INNER JOIN 过滤，不回填。
func TestWorkstationImport_DeptInferFromUser_DirtyUserDept(t *testing.T) {
	db := setupDeptInferDB(t)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u1', 'wangwu', '王五', '信息技术部', 0)`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u2', 'zhaoliu', '赵六', 'ghost-dept-id', 0)`).Error)

	res := importDeptInferRows(t, db, [][]string{
		{"W-INFER-3A", "研发大楼", "3楼", "不存在的部门名", "王五"},
		{"W-INFER-3B", "研发大楼", "3楼", "不存在的部门名", "赵六"},
	})
	assert.Empty(t, res.Errors, "脏 dept_id 不应导致导入失败: %+v", res.Errors)
	assert.Equal(t, 2, res.Inserted)
	assert.Equal(t, 0, res.InferredDeptCount)

	for _, name := range []string{"W-INFER-3A", "W-INFER-3B"} {
		deptID, userID := readDeptInferWorkstation(t, db, name)
		assert.NotEmpty(t, userID, "%s 人员应正常关联", name)
		assert.Empty(t, deptID, "%s 不应把脏 dept_id 写入工位", name)
	}
}

// TestWorkstationImport_DeptInferFromUser_ExplicitDeptWins
// 用户确认的「仅兜底」策略：显式填对的部门优先，即使与用户所属部门不同也不覆盖。
func TestWorkstationImport_DeptInferFromUser_ExplicitDeptWins(t *testing.T) {
	db := setupDeptInferDB(t)
	// 用户属于 d2(财务部)，Excel 显式填 d1(信息技术部)。
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u1', 'zhangsan', '张三', 'd2', 0)`).Error)

	res := importDeptInferRows(t, db, [][]string{
		{"W-INFER-4", "研发大楼", "3楼", "信息技术部", "张三"},
	})
	assert.Empty(t, res.Errors, "不应有校验错误: %+v", res.Errors)
	assert.Equal(t, 1, res.Inserted)
	assert.Equal(t, 0, res.InferredDeptCount, "显式部门已解析成功，不应触发反推")

	deptID, _ := readDeptInferWorkstation(t, db, "W-INFER-4")
	assert.Equal(t, "d1", deptID, "显式填写的部门必须优先于用户所属部门")
}

// TestWorkstationImport_DeptInferFromUser_NoUser
// 部门填错且未填人员 → 没有 user_id 可反推，保持空部门、不报错。
func TestWorkstationImport_DeptInferFromUser_NoUser(t *testing.T) {
	db := setupDeptInferDB(t)

	res := importDeptInferRows(t, db, [][]string{
		{"W-INFER-5", "研发大楼", "3楼", "不存在的部门名", ""},
	})
	assert.Empty(t, res.Errors, "部门列可选，解析失败不应报错: %+v", res.Errors)
	assert.Equal(t, 1, res.Inserted)
	assert.Equal(t, 0, res.InferredDeptCount)

	deptID, userID := readDeptInferWorkstation(t, db, "W-INFER-5")
	assert.Empty(t, userID)
	assert.Empty(t, deptID)
}

// TestWorkstationImport_DeptInferFromUser_Mixed
// 一批里混合「可反推 / 用户无部门 / 用户脏部门 / 显式部门 / 无用户」，
// 验证逐行判定与 InferredDeptCount 计数准确。
func TestWorkstationImport_DeptInferFromUser_Mixed(t *testing.T) {
	db := setupDeptInferDB(t)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u1', 'zhangsan', '张三', 'd2', 0)`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u2', 'lisi', '李四', NULL, 0)`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname, dept_id, status) VALUES ('u3', 'wangwu', '王五', 'ghost-dept', 0)`).Error)

	res := importDeptInferRows(t, db, [][]string{
		{"W-MIX-1", "研发大楼", "3楼", "不存在的部门名", "张三"},  // 反推 → d2
		{"W-MIX-2", "研发大楼", "3楼", "不存在的部门名", "李四"},  // 用户无部门 → 空
		{"W-MIX-3", "研发大楼", "3楼", "不存在的部门名", "王五"},  // 用户脏部门 → 空
		{"W-MIX-4", "研发大楼", "3楼", "信息技术部", "张三"},    // 显式部门 → d1
		{"W-MIX-5", "研发大楼", "3楼", "不存在的部门名", ""},   // 无用户 → 空
	})
	assert.Empty(t, res.Errors, "不应有校验错误: %+v", res.Errors)
	assert.Equal(t, 5, res.Inserted)
	assert.Equal(t, 1, res.InferredDeptCount, "只有 W-MIX-1 应由用户反推")

	expectDept := map[string]string{
		"W-MIX-1": "d2",
		"W-MIX-2": "",
		"W-MIX-3": "",
		"W-MIX-4": "d1",
		"W-MIX-5": "",
	}
	for name, want := range expectDept {
		deptID, _ := readDeptInferWorkstation(t, db, name)
		assert.Equal(t, want, deptID, "%s 的 dept_id 不符合预期", name)
	}
}
