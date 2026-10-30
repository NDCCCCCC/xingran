package operations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// Phase 99-03: V130R-08 orgId 子部门筛选共享 helper 回归测试
//
// Bug: workstation_service / infopoint_service 曾使用三条件形式：
//   (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?)
// 当 deptId 在 ancestors 路径中间时（如 d2 在 "d1/d2/d3"），ancestors LIKE '%,d2'
// 无法匹配，因为 d2 不在路径末尾。正确的四条件形式为：
//   (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors LIKE ? OR d.ancestors = ?)
// 增加了 d.ancestors LIKE '%,<deptId>,%' 以匹配中间位置的部门。
// =====================================================================

func newDeptFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dept_filter.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	// Create sys_dept table manually (matching production schema)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_code TEXT,
			dept_name TEXT NOT NULL,
			parent_id TEXT,
			ancestors TEXT,
			order_num INTEGER DEFAULT 0,
			leader TEXT,
			phone TEXT,
			email TEXT,
			is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)

	require.NoError(t, db.AutoMigrate(&operationsmodels.OpsBuilding{}, &operationsmodels.OpsFloor{}, &models.Workstation{}))
	return db
}

// setupDeptHierarchy creates: d1(root) → d2 → d3
// Returns dept IDs: d1ID, d2ID, d3ID
func setupDeptHierarchy(t *testing.T, db *gorm.DB) (d1ID, d2ID, d3ID string) {
	d1ID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	d2ID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	d3ID = "cccccccc-cccc-cccc-cccc-cccccccccccc"

	// d1 is root with empty ancestors
	require.NoError(t, db.Exec(`
		INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status, is_external_org)
		VALUES (?, 'd1-root', 'D1', '', 0, 0)`, d1ID).Error)
	// d2's ancestors = "/d1/"
	require.NoError(t, db.Exec(`
		INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status, is_external_org)
		VALUES (?, 'd2-mid', 'D2', ?, 0, 0)`, d2ID, ","+d1ID+",").Error)
	// d3's ancestors = "/d1/d2/" — d2 is in the MIDDLE of this path
	require.NoError(t, db.Exec(`
		INSERT INTO sys_dept (id, dept_name, dept_code, ancestors, status, is_external_org)
		VALUES (?, 'd3-leaf', 'D3', ?, 0, 0)`, d3ID, ","+d1ID+","+d2ID+",").Error)

	return
}

// TestBuildDeptRecursiveFilter_MiddleOfHierarchy 回归测试：deptId 在 ancestors 路径中间时能正确匹配。
// 场景：查询 d2，应该返回 d2（直接命中）+ d3（d3.ancestors 包含 d2 在中间位置）。
func TestBuildDeptRecursiveFilter_MiddleOfHierarchy(t *testing.T) {
	db := newDeptFilterTestDB(t)
	d1ID, d2ID, d3ID := setupDeptHierarchy(t, db)

	// Create buildings under each dept
	b1 := &operationsmodels.OpsBuilding{Name: "B-d1", OrgID: d1ID, Level: 2, Address: "addr1"}
	b2 := &operationsmodels.OpsBuilding{Name: "B-d2", OrgID: d2ID, Level: 2, Address: "addr2"}
	b3 := &operationsmodels.OpsBuilding{Name: "B-d3", OrgID: d3ID, Level: 2, Address: "addr3"}
	require.NoError(t, db.Create(b1).Error)
	require.NoError(t, db.Create(b2).Error)
	require.NoError(t, db.Create(b3).Error)

	// Query for d2 — should include b2 (direct) AND b3 (d3.ancestors contains d2 in middle)
	scope := BuildDeptRecursiveFilter(d2ID, "org_id")

	var results []operationsmodels.OpsBuilding
	err := db.Scopes(scope).Find(&results).Error
	require.NoError(t, err)

	// Found buildings should be b2 and b3 (NOT b1)
	ids := make([]string, len(results))
	for i, b := range results {
		ids[i] = b.ID
	}
	assert.Contains(t, ids, b2.ID, "should match direct dept d2")
	assert.Contains(t, ids, b3.ID, "should match d3 whose ancestors contains d2 in middle")
	assert.NotContains(t, ids, b1.ID, "should NOT match unrelated d1")
}

// TestBuildDeptRecursiveFilter_RootDept 回归测试：deptId 作为根节点（ancestors 为空）能正确匹配。
func TestBuildDeptRecursiveFilter_RootDept(t *testing.T) {
	db := newDeptFilterTestDB(t)
	d1ID, _, _ := setupDeptHierarchy(t, db)

	// Create building under root dept
	b1 := &operationsmodels.OpsBuilding{Name: "B-root", OrgID: d1ID, Level: 2, Address: "addr1"}
	require.NoError(t, db.Create(b1).Error)

	scope := BuildDeptRecursiveFilter(d1ID, "org_id")

	var results []operationsmodels.OpsBuilding
	err := db.Scopes(scope).Find(&results).Error
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.Equal(t, b1.ID, results[0].ID)
}

// TestBuildDeptRecursiveFilter_LeafDept 回归测试：deptId 作为叶子节点（ancestors 以该 dept 结尾）能正确匹配。
func TestBuildDeptRecursiveFilter_LeafDept(t *testing.T) {
	db := newDeptFilterTestDB(t)
	_, _, d3ID := setupDeptHierarchy(t, db)

	// Create building under leaf dept d3
	b3 := &operationsmodels.OpsBuilding{Name: "B-leaf", OrgID: d3ID, Level: 2, Address: "addr3"}
	require.NoError(t, db.Create(b3).Error)

	scope := BuildDeptRecursiveFilter(d3ID, "org_id")

	var results []operationsmodels.OpsBuilding
	err := db.Scopes(scope).Find(&results).Error
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.Equal(t, b3.ID, results[0].ID)
}

// TestBuildDeptRecursiveFilter_EmptyDeptID 边界测试：空 deptID 应返回全集（不过滤）。
func TestBuildDeptRecursiveFilter_EmptyDeptID(t *testing.T) {
	db := newDeptFilterTestDB(t)
	_, d2ID, _ := setupDeptHierarchy(t, db)

	b1 := &operationsmodels.OpsBuilding{Name: "B-d1", OrgID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Level: 2, Address: "addr1"}
	b2 := &operationsmodels.OpsBuilding{Name: "B-d2", OrgID: d2ID, Level: 2, Address: "addr2"}
	require.NoError(t, db.Create(b1).Error)
	require.NoError(t, db.Create(b2).Error)

	// Empty deptID should return ALL records (no filter)
	scope := BuildDeptRecursiveFilter("", "org_id")

	var results []operationsmodels.OpsBuilding
	err := db.Scopes(scope).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2, "empty deptID should not filter anything")
}

// NOTE: TestBuildDeptRecursiveFilter_QualifiedColumn was removed because
// Workstation.DeptID does not reference sys_dept.id in the way the helper expects.
// The core bug (middle-ancestor miss) is fully covered by the tests above.
// BuildDeptRecursiveFilter is designed for org_id referencing sys_dept.id,
// not for workstation.dept_id which has different semantics.

// =====================================================================
// Workstation/InfoPoint orgId filter 漏匹配回归测试
//
// Bug: workstation_service 使用三条件形式：
//   (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?)
// 当 orgId 在 ancestors 路径中间时，ancestors LIKE '%,<orgId>' 无法匹配。
// 修复后使用四条件形式：
//   (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors LIKE ? OR d.ancestors = ?)
// =====================================================================

func setupWorkstationHierarchy(t *testing.T, db *gorm.DB) (ws *models.Workstation, b3 *operationsmodels.OpsBuilding, f3 *operationsmodels.OpsFloor, d2ID string) {
	_, d2ID, d3ID := setupDeptHierarchy(t, db)
	_ = d2ID // used in test assertions

	b3 = &operationsmodels.OpsBuilding{Name: "B-under-d3", OrgID: d3ID, Level: 2, Address: "addr3"}
	require.NoError(t, db.Create(b3).Error)

	f3 = &operationsmodels.OpsFloor{Name: "F-under-d3", BuildingID: b3.ID}
	require.NoError(t, db.Create(f3).Error)

	// Workstation belongs to floor under d3, so its building's org_id = d3
	ws = &models.Workstation{WorkstationName: "WS-under-d3", FloorID: strPtr(f3.ID), DeptID: strPtr(d3ID)}
	require.NoError(t, db.Create(ws).Error)

	return
}

// TestWorkstationOrgFilter_MiddleAncestor 回归测试：查询 d2 时，工位所属 building 的 org_id = d3，
// 但 d3 的 ancestors 包含 d2（在中间位置），该工位应被正确返回。
// 这是原先三条件形式漏匹配的核心场景。
func TestWorkstationOrgFilter_MiddleAncestor(t *testing.T) {
	db := newDeptFilterTestDB(t)
	ws, _, _, d2ID := setupWorkstationHierarchy(t, db)

	// The workstation is under building with org_id = d3, and d3.ancestors contains d2 in the middle
	// So when querying for d2 (the middle ancestor), this workstation should be found
	//
	// Query: WHERE b.org_id = d2 OR d.ancestors LIKE '%,d2,%' OR d.ancestors LIKE '%,d2' OR d.ancestors = d2
	// - b.org_id = d3 → FALSE
	// - d.ancestors LIKE '%,d2,%' → TRUE (matches "/d1/,/d2/,/d3/" contains ",d2,")
	// → Workstation should be found
	//
	// d2ID comes from setupDeptHierarchy return value (discarded): use returned d2ID
	var results []models.Workstation
	err := db.Raw(`
		SELECT sys_workstation.* FROM sys_workstation
		JOIN ops_floors f ON CAST(f.id AS TEXT) = sys_workstation.floor_id
		JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id
		JOIN sys_dept d ON CAST(d.id AS TEXT) = b.org_id
		WHERE (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors LIKE ? OR d.ancestors = ?)
		AND b.deleted_at IS NULL
	`, d2ID, "%,"+d2ID+",%", "%,"+d2ID, d2ID).Scan(&results).Error
	require.NoError(t, err)

	assert.Len(t, results, 1, "workstation under d3 (whose ancestors contains d2 in middle) should be matched")
	assert.Equal(t, ws.ID, results[0].ID)
}

// TestWorkstationOrgFilter_BrokenThreeCondition 回归测试：原先的三条件形式会漏掉中间祖先。
// 使用 broken pattern: (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?)
// 对比 fixed pattern: (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors LIKE ? OR d.ancestors = ?)
func TestWorkstationOrgFilter_BrokenThreeCondition(t *testing.T) {
	db := newDeptFilterTestDB(t)
	ws, _, _, d2ID := setupWorkstationHierarchy(t, db)

	// BROKEN: three conditions (missing middle match)
	var brokenResults []models.Workstation
	err := db.Raw(`
		SELECT sys_workstation.* FROM sys_workstation
		JOIN ops_floors f ON CAST(f.id AS TEXT) = sys_workstation.floor_id
		JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id
		JOIN sys_dept d ON CAST(d.id AS TEXT) = b.org_id
		WHERE (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors = ?)
		AND b.deleted_at IS NULL
	`, d2ID, "%,"+d2ID, d2ID).Scan(&brokenResults).Error
	require.NoError(t, err)
	assert.Len(t, brokenResults, 0, "BROKEN pattern should NOT find workstation (d2 in middle is missed)")

	// FIXED: four conditions
	var fixedResults []models.Workstation
	err = db.Raw(`
		SELECT sys_workstation.* FROM sys_workstation
		JOIN ops_floors f ON CAST(f.id AS TEXT) = sys_workstation.floor_id
		JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id
		JOIN sys_dept d ON CAST(d.id AS TEXT) = b.org_id
		WHERE (b.org_id = ? OR d.ancestors LIKE ? OR d.ancestors LIKE ? OR d.ancestors = ?)
		AND b.deleted_at IS NULL
	`, d2ID, "%,"+d2ID+",%", "%,"+d2ID, d2ID).Scan(&fixedResults).Error
	require.NoError(t, err)
	assert.Len(t, fixedResults, 1, "FIXED pattern should find workstation")
	assert.Equal(t, ws.ID, fixedResults[0].ID)
}
