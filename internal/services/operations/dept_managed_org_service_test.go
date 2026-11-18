//go:build !skip_db_tests
// +build !skip_db_tests

package operations

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupManagedOrgTestDB 条线部门→管理归属机构 映射 service 测试夹具。
// 与 location_alias_service_test.go 同构：内存 SQLite + AutoMigrate +
// 手写 partial unique（AutoMigrate 不支持 partial index 语法）。
func setupManagedOrgTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        "file::memory:?cache=shared",
	}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "connect sqlite test db")

	require.NoError(t,
		db.AutoMigrate(&models.SysDeptLocationAlias{}, &models.Department{}),
		"auto-migrate test tables",
	)

	require.NoError(t, db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_test_alias_dept_scope ON sys_dept_location_alias(dept_id, scope) WHERE deleted_at IS NULL`,
	).Error)

	require.NoError(t, db.Where("1=1").Delete(&models.SysDeptLocationAlias{}).Error)
	require.NoError(t, db.Exec("DELETE FROM sys_dept").Error)

	return db
}

// seedOrgDept 插入一条 sys_dept 记录（带 org_level），返回 id。
func seedOrgDept(t *testing.T, db *gorm.DB, name string, orgLevel models.OrgLevel) string {
	dept := &models.Department{
		DeptName: name,
		DeptCode: "code-" + name,
		Ancestors: "",
		OrgLevel:  orgLevel,
	}
	require.NoError(t, db.Create(dept).Error)
	return dept.ID
}

// TestValidateManagedOrg_SelfMappingRejected 自映射拦截。
func TestValidateManagedOrg_SelfMappingRejected(t *testing.T) {
	db := setupManagedOrgTestDB(t)
	svc := NewDeptManagedOrgService(db)
	ctx := context.Background()

	deptID := seedOrgDept(t, db, "潜江团客共享作业服务中心", 0)
	_, err := svc.Create(ctx, &DeptManagedOrgCreateRequest{DeptID: deptID, OrgID: deptID})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能相同")
}

// TestValidateManagedOrg_DeptMissing 条线部门不存在 → 拒绝。
func TestValidateManagedOrg_DeptMissing(t *testing.T) {
	db := setupManagedOrgTestDB(t)
	svc := NewDeptManagedOrgService(db)
	ctx := context.Background()

	orgID := seedOrgDept(t, db, "潜江中支", models.OrgLevelSubBang)
	_, err := svc.Create(ctx, &DeptManagedOrgCreateRequest{DeptID: "D-nope", OrgID: orgID})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "条线部门不存在")
}

// TestValidateManagedOrg_OrgMustBeOrgNode 归属目标必须是机构节点（org_level∈{1,2,3}）。
// 这是与工位 validateAlias（is_external_org=1）的分叉点。
func TestValidateManagedOrg_OrgMustBeOrgNode(t *testing.T) {
	db := setupManagedOrgTestDB(t)
	svc := NewDeptManagedOrgService(db)
	ctx := context.Background()

	deptID := seedOrgDept(t, db, "潜江团客共享作业服务中心", 0)
	plainDept := seedOrgDept(t, db, "某个普通科室", 0) // org_level=0：普通部门
	_, err := svc.Create(ctx, &DeptManagedOrgCreateRequest{DeptID: deptID, OrgID: plainDept})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "必须是机构节点")
}

// TestCreateManagedOrg_HappyPathAndDuplicate 合法创建 + scope 恒为 permission + 重复创建友好提示。
func TestCreateManagedOrg_HappyPathAndDuplicate(t *testing.T) {
	db := setupManagedOrgTestDB(t)
	svc := NewDeptManagedOrgService(db)
	ctx := context.Background()

	deptID := seedOrgDept(t, db, "潜江团客共享作业服务中心", 0)
	orgID := seedOrgDept(t, db, "潜江中支", models.OrgLevelSubBang)

	alias, err := svc.Create(ctx, &DeptManagedOrgCreateRequest{DeptID: deptID, OrgID: orgID, Remark: "条线归属"})
	require.NoError(t, err)
	assert.Equal(t, AliasScopePermission, alias.Scope, "写入 scope 恒为 permission")
	assert.Equal(t, orgID, alias.LocationID)

	// 重复创建（同 dept_id + scope）→ partial unique 冲突 → 友好中文
	_, err = svc.Create(ctx, &DeptManagedOrgCreateRequest{DeptID: deptID, OrgID: orgID})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已配置管理归属机构")
}

// TestListManagedOrg_ScopeIsolation List 只返回 permission 行（工位行不可见）。
func TestListManagedOrg_ScopeIsolation(t *testing.T) {
	db := setupManagedOrgTestDB(t)
	svc := NewDeptManagedOrgService(db)
	ctx := context.Background()

	deptID := seedOrgDept(t, db, "潜江团客共享作业服务中心", 0)
	orgID := seedOrgDept(t, db, "潜江中支", models.OrgLevelSubBang)
	wsDept := seedOrgDept(t, db, "某外部机构", 0)

	// 工位行（scope=workstation）+ 权限行（scope=permission）
	require.NoError(t, db.Create(&models.SysDeptLocationAlias{
		DeptID: wsDept, LocationID: orgID, Scope: "workstation",
	}).Error)
	_, err := svc.Create(ctx, &DeptManagedOrgCreateRequest{DeptID: deptID, OrgID: orgID})
	require.NoError(t, err)

	result, err := svc.List(ctx, 1, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, result.Total, "List 不得混入 workstation 行")
	items := result.List.([]DeptManagedOrgListItem)
	require.Len(t, items, 1)
	assert.Equal(t, deptID, items[0].DeptID)
	assert.Equal(t, "潜江团客共享作业服务中心", items[0].OriginDeptName, "JOIN sys_dept 部门名")
	assert.Equal(t, "潜江中支", items[0].LocationDeptName)
}

// TestDeleteManagedOrg_ScopeGuard Delete 只能删 permission 行（防误删工位映射）。
func TestDeleteManagedOrg_ScopeGuard(t *testing.T) {
	db := setupManagedOrgTestDB(t)
	svc := NewDeptManagedOrgService(db)
	ctx := context.Background()

	orgID := seedOrgDept(t, db, "潜江中支", models.OrgLevelSubBang)
	wsDept := seedOrgDept(t, db, "某外部机构", 0)
	ws := &models.SysDeptLocationAlias{DeptID: wsDept, LocationID: orgID, Scope: "workstation"}
	require.NoError(t, db.Create(ws).Error)

	// 工位行：拒绝删除
	err := svc.Delete(ctx, ws.ID)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "不是条线部门归属映射"), "错误需说明 scope 拦截原因")

	// permission 行：正常删除
	perm := &models.SysDeptLocationAlias{DeptID: wsDept, LocationID: orgID, Scope: AliasScopePermission}
	require.NoError(t, db.Create(perm).Error)
	require.NoError(t, svc.Delete(ctx, perm.ID))
	var count int64
	require.NoError(t, db.Model(&models.SysDeptLocationAlias{}).
		Where("id = ? AND deleted_at IS NULL", perm.ID).Count(&count).Error)
	assert.Zero(t, count, "软删除后不可再查到")
}
