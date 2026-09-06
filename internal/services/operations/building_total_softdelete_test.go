package operations

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
)

// =====================================================================
// Phase 99-01: V130R-06 soft-delete Total 口径回归测试
//
// validateNameUnique 和 SearchBuildingOptions 曾使用 .Table("ops_buildings") 起链，
// 导致 Count() 绕过了 GORM 的 deleted_at IS NULL scope，使 Total 虚高。
// 本测试锁定修复后 List Total 正确排除软删除记录。
// =====================================================================

func newBuildingSoftDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "building_softdel.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, db.AutoMigrate(&operationsmodels.OpsBuilding{}))
	// 注入一个有效的 sys_dept，以满足 validateOrg 检查
	require.NoError(t, db.Exec(
		`CREATE TABLE IF NOT EXISTS sys_dept (id TEXT PRIMARY KEY, dept_name TEXT, dept_code TEXT, ancestors TEXT, status INTEGER, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status) VALUES (?, '测试部门', 'D1', '', 0)`,
		"11111111-1111-1111-1111-111111111111").Error)
	// ops_floors required by List's workstation_count subquery
	require.NoError(t, db.Exec(
		`CREATE TABLE IF NOT EXISTS ops_floors (id TEXT PRIMARY KEY, name TEXT, floor_no TEXT, building_id TEXT, deleted_at DATETIME)`).Error)
	// sys_workstation required by List's workstation_count subquery
	require.NoError(t, db.Exec(
		`CREATE TABLE IF NOT EXISTS sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, floor_id TEXT, deleted_at DATETIME)`).Error)
	return db
}

// TestBuildingList_TotalExcludesSoftDeleted 锁定 List 的 Total 计数排除软删除记录。
// 对比 validateNameUnique（直接 Count）和 SearchBuildingOptions（List 结果）两条路径。
func TestBuildingList_TotalExcludesSoftDeleted(t *testing.T) {
	db := newBuildingSoftDeleteTestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	const testOrgID = "11111111-1111-1111-1111-111111111111"

	// 创建 3 栋楼宇
	b1 := &operationsmodels.OpsBuilding{Name: "B-Test1", OrgID: testOrgID, Level: 2, Address: "addr1"}
	b2 := &operationsmodels.OpsBuilding{Name: "B-Test2", OrgID: testOrgID, Level: 2, Address: "addr2"}
	b3 := &operationsmodels.OpsBuilding{Name: "B-Test3", OrgID: testOrgID, Level: 2, Address: "addr3"}
	require.NoError(t, svc.Create(ctx, b1))
	require.NoError(t, svc.Create(ctx, b2))
	require.NoError(t, svc.Create(ctx, b3))

	// 软删除 b2
	require.NoError(t, svc.Delete(ctx, b2.ID))

	// List 应返回 2，不含软删除的 b2
	result, err := svc.List(ctx, map[string]interface{}{"current": 1, "pageSize": 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total, "Total 应排除软删除记录 b2")

	// SearchBuildingOptions 也应排除软删除记录（底层同样走 .Model()）
	opts, err := svc.SearchBuildingOptions(ctx, map[string]interface{}{"orgId": testOrgID})
	require.NoError(t, err)
	assert.Len(t, opts, 2, "SearchBuildingOptions 应排除软删除记录 b2")
}

// TestValidateNameUnique_ExcludesSoftDeleted 锁定 validateNameUnique 的 Count 排除软删除记录。
// 用唯一性校验间接验证：同名软删记录不阻塞 Create。
func TestValidateNameUnique_ExcludesSoftDeleted(t *testing.T) {
	db := newBuildingSoftDeleteTestDB(t)
	svc := NewBuildingService(db)
	ctx := context.Background()

	const testOrgID = "11111111-1111-1111-1111-111111111111"

	// 创建 b1
	b1 := &operationsmodels.OpsBuilding{Name: "B-Unique", OrgID: testOrgID, Level: 2, Address: "addr"}
	require.NoError(t, svc.Create(ctx, b1))

	// 软删除 b1
	require.NoError(t, svc.Delete(ctx, b1.ID))

	// 同名同机构重建应成功（validateNameUnique 的 Count 不应命中软删的 b1）
	b2 := &operationsmodels.OpsBuilding{Name: "B-Unique", OrgID: testOrgID, Level: 2, Address: "addr2"}
	err := svc.Create(ctx, b2)
	assert.NoError(t, err, "软删除的同名记录不应阻塞 Create（validateNameUnique Count 应排除软删）")
}
