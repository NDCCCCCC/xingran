//go:build !skip_db_tests
// +build !skip_db_tests

package addomain

// HYGIENE-05（62-03）：syncDeptTree 同名兄弟部门 OU DN 冲突 → 显式计失败，不静默覆盖。
// 复用 test_fixtures_test.go 的 createDeptOUMappingTable/createDeptTable（62-01 迁入）。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestSyncDeptTree_OUConflictCountedAsFailed：
// 预置 OU=人资部,DC=corp 归属 dept-A → 同步同名兄弟形态的 dept-B（BuildOUDN 产出同 DN，
// mock CreateOU 无错）→ UpsertMapping 命中异部门冲突 → result.FailedDepts==1、
// Errors 记 dept-B、SuccessDepts==0，且 dept-A 映射行未被删（SC3：映射保留给先占方）。
func TestSyncDeptTree_OUConflictCountedAsFailed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(createDeptOUMappingTable).Error)
	require.NoError(t, db.Exec(createDeptTable).Error)

	// 预置：dept-A 已占 OU=人资部,DC=corp；sys_dept 两行（供冲突错误反查部门名）
	require.NoError(t, db.Exec("INSERT INTO sys_dept (id, dept_name) VALUES ('dept-A', '财务部'), ('dept-B', '人资部')").Error)
	pre := &models.DeptOUMapping{ID: "row-A", DeptID: "dept-A", ADConfigID: "cfg-1", OUDN: "OU=人资部,DC=corp", OUName: "人资部", SyncStatus: "synced"}
	require.NoError(t, db.Create(pre).Error)

	// 裸构造沿 dept_sync_service_test.go 先例（仅需 db + mapper）
	svc := &DeptToADSyncService{db: db, mapper: &DeptOUmapper{db: db}}
	config := models.ADConfig{BaseDN: "DC=corp"}
	config.ID = "cfg-1"
	deptB := &models.Department{DeptName: "人资部"}
	deptB.ID = "dept-B" // 与 dept-A 无亲属关系，但 BuildOUDN("人资部","DC=corp") 与预置行同 DN → 争用
	result := &DeptSyncResult{}

	err = svc.syncDeptTree(context.Background(), &mockLDAPClient{}, &config, deptB, "DC=corp", result)
	require.NoError(t, err, "冲突不得中断整棵树")

	assert.Equal(t, 1, result.FailedDepts, "冲突必须计入 FailedDepts")
	assert.Equal(t, 0, result.SuccessDepts, "冲突部门不得计入 SuccessDepts")
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "dept-B", result.Errors[0].DeptID)
	assert.Equal(t, "人资部", result.Errors[0].DeptName)

	var countA int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-A").Count(&countA)
	assert.Equal(t, int64(1), countA, "dept-A 映射不得被静默删除")
	var countB int64
	db.Model(&models.DeptOUMapping{}).Where("dept_id = ?", "dept-B").Count(&countB)
	assert.Equal(t, int64(0), countB)
}
