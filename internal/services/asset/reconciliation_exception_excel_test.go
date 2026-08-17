package asset

// Phase 44 R3 / Plan 44-02 Task 4 — Excel 导入辅助函数测试
//
// 测试 ResolveReconScopeID (scope_name → scope_id 按 scope_type 解析)
// 与 ParseCSVToTextArray ("B,C,D" → ["B","C","D"] TEXT[] 转换)。
//
// 方案 B (WARN-7 锁定): 这两个 helper 由 ImportFromExcel service 方法调用。

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupScopeResolveTestDB 构造 sqlite + sys_dept + sys_user 表
func setupScopeResolveTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:recon_scope_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			dept_code TEXT,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

// TestResolveScopeID_Dept dept 类型解析 sys_dept.dept_name → id
func TestResolveScopeID_Dept(t *testing.T) {
	db := setupScopeResolveTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name) VALUES ('dept-001', '研发部')`).Error)

	scopeID, err := ResolveReconScopeID(context.Background(), db, "dept", "研发部")
	require.NoError(t, err)
	assert.Equal(t, "dept-001", scopeID, "scope_type=dept 应按 dept_name 解析")
}

// TestResolveScopeID_User user 类型解析 sys_user.username → id
func TestResolveScopeID_User(t *testing.T) {
	db := setupScopeResolveTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username) VALUES ('user-001', 'admin')`).Error)

	scopeID, err := ResolveReconScopeID(context.Background(), db, "user", "admin")
	require.NoError(t, err)
	assert.Equal(t, "user-001", scopeID, "scope_type=user 应按 username 解析")
}

// TestResolveScopeID_Global global 类型返回空字符串(scope_id 保持 NULL)
func TestResolveScopeID_Global(t *testing.T) {
	db := setupScopeResolveTestDB(t)
	scopeID, err := ResolveReconScopeID(context.Background(), db, "global", "")
	require.NoError(t, err)
	assert.Equal(t, "", scopeID, "scope_type=global 应返回空(scope_id 保持 NULL)")
}

// TestResolveScopeID_GlobalIgnoresName global 类型即使传了 name 也返回空
func TestResolveScopeID_GlobalIgnoresName(t *testing.T) {
	db := setupScopeResolveTestDB(t)
	scopeID, err := ResolveReconScopeID(context.Background(), db, "global", "任意值")
	require.NoError(t, err)
	assert.Equal(t, "", scopeID, "scope_type=global 忽略 name, 返回空")
}

// TestResolveScopeID_NotFound dept 名称不存在返回 error
func TestResolveScopeID_NotFound(t *testing.T) {
	db := setupScopeResolveTestDB(t)
	_, err := ResolveReconScopeID(context.Background(), db, "dept", "不存在的部门")
	assert.Error(t, err, "dept 名称不存在应返回 error")
}

// TestResolveScopeID_UserNotFound user 不存在返回 error
func TestResolveScopeID_UserNotFound(t *testing.T) {
	db := setupScopeResolveTestDB(t)
	_, err := ResolveReconScopeID(context.Background(), db, "user", "ghost")
	assert.Error(t, err, "user 不存在应返回 error")
}

// TestResolveScopeID_DeptEmptyName dept + 空 name 返回空(允许导入时不指定部门)
func TestResolveScopeID_DeptEmptyName(t *testing.T) {
	db := setupScopeResolveTestDB(t)
	scopeID, err := ResolveReconScopeID(context.Background(), db, "dept", "")
	require.NoError(t, err)
	assert.Equal(t, "", scopeID, "空 name 应返回空(允许 scope_id NULL)")
}

// TestParseCSVToTextArray TEXT[] 转换: "B,C,D" → ["B","C","D"]
func TestParseCSVToTextArray(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"standard csv", "B,C,D", []string{"B", "C", "D"}},
		{"single value", "no_alert", []string{"no_alert"}},
		{"spaces trimmed", " B , C , D ", []string{"B", "C", "D"}},
		{"empty returns nil", "", nil},
		{"multiple actions", "no_alert,no_notice,no_workorder", []string{"no_alert", "no_notice", "no_workorder"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCSVToTextArray(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
