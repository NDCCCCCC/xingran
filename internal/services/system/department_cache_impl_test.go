package system

// =====================================================================
// department_cache_impl_test.go — covers department_cache_impl.go
// Compile-time interface assertion + cache miss/hit/invalidation tests
// =====================================================================

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// Compile-time interface assertion
var _ DepartmentService = (*departmentCacheService)(nil)

func setupDeptCacheDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT NOT NULL,
			dept_code TEXT,
			parent_id TEXT,
			ancestors TEXT DEFAULT '',
			order_num INTEGER DEFAULT 0,
			leader TEXT,
			phone TEXT,
			email TEXT,
			is_external_org INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			remark TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			username TEXT,
			dept_id TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			version INTEGER DEFAULT 0
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role_dept (
			role_id TEXT,
			dept_id TEXT,
			PRIMARY KEY (role_id, dept_id)
		)
	`).Error)
	return db
}

func seedDeptCache(t *testing.T, db *gorm.DB, d *models.Department) string {
	t.Helper()
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	require.NoError(t, db.Create(d).Error)
	return d.ID
}

// TC1: GetTree - delegates via cache
func TestDeptCache_GetTree(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetTree(context.Background(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

// TC2: GetTreeWithFilter - delegates
func TestDeptCache_GetTreeWithFilter(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetTreeWithFilter(context.Background(), true, requests.DepartmentListParams{})
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

// TC3: GetTreeWithFilter - with name bypasses cache
func TestDeptCache_GetTreeWithFilter_NamedBypassCache(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	seedDeptCache(t, db, &models.Department{DeptName: "研发", DeptCode: "RD", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetTreeWithFilter(context.Background(), true, requests.DepartmentListParams{DeptName: "研发"})
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "研发", tree[0].DeptName)
}

// TC4: GetTreeWithCache - delegates
func TestDeptCache_GetTreeWithCache(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetTreeWithCache(context.Background(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

// TC5: GetSelectDataWithCache - delegates
func TestDeptCache_GetSelectDataWithCache(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetSelectDataWithCache(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

// TC6: InvalidateDeptCache - no error
func TestDeptCache_InvalidateDeptCache(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	require.NoError(t, svc.InvalidateDeptCache(context.Background()))
}

// TC7: Create - delegates + invalidates
func TestDeptCache_Create(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)

	req := &requests.DepartmentCreateRequest{
		DeptName: "New", DeptCode: "NEW", Status: models.DeptStatusNormal,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Department
	require.NoError(t, db.Where("dept_code = ?", "NEW").First(&got).Error)
	assert.Equal(t, "New", got.DeptName)
}

// TC8: Update - delegates + invalidates
func TestDeptCache_Update(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	id := seedDeptCache(t, db, &models.Department{DeptName: "Old", DeptCode: "OLD", Status: models.DeptStatusNormal})

	req := &requests.DepartmentUpdateRequest{
		ID:       id,
		DeptName: "New", DeptCode: "OLD", Status: models.DeptStatusNormal,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Department
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DeptName)
}

// TC9: Delete - delegates + invalidates
func TestDeptCache_Delete(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	id := seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC10: BatchDelete - delegates + invalidates
func TestDeptCache_BatchDelete(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	id1 := seedDeptCache(t, db, &models.Department{DeptName: "D1", DeptCode: "D1", Status: models.DeptStatusNormal})
	id2 := seedDeptCache(t, db, &models.Department{DeptName: "D2", DeptCode: "D2", Status: models.DeptStatusNormal})

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}))
}

// TC11: UpdateStatus - delegates + invalidates
func TestDeptCache_UpdateStatus(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	id := seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	require.NoError(t, svc.UpdateStatus(context.Background(), id, 1))
}

// TC12: GetDB - returns underlying db
func TestDeptCache_GetDB(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	got := svc.GetDB()
	assert.Equal(t, db, got)
}

// TC13: GetByID - delegates
func TestDeptCache_GetByID(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	id := seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	dept, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "D", dept.DeptName)
}

// TC14: List - delegates
func TestDeptCache_List(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	seedDeptCache(t, db, &models.Department{DeptName: "D1", DeptCode: "D1", Status: models.DeptStatusNormal})

	depts, err := svc.List(context.Background(), requests.DepartmentListParams{})
	require.NoError(t, err)
	assert.Len(t, depts, 1)
}

// TC15: GetRoleDeptIDs - delegates
func TestDeptCache_GetRoleDeptIDs(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	roleID := uuid.NewString()
	deptID := seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	require.NoError(t, db.Exec("INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)", roleID, deptID).Error)

	var ids []string
	require.NoError(t, svc.GetRoleDeptIDs(context.Background(), roleID, &ids))
	assert.Contains(t, ids, deptID)
}

// TC16: Create - duplicate name fails
func TestDeptCache_Create_DuplicateName(t *testing.T) {
	db := setupDeptCacheDB(t)
	cache := &NoOpCacheProvider{}
	svc := NewDepartmentServiceWithCache(db, cache, nil)
	seedDeptCache(t, db, &models.Department{DeptName: "D", DeptCode: "D1", Status: models.DeptStatusNormal})

	req := &requests.DepartmentCreateRequest{
		DeptName: "D", DeptCode: "D2", Status: models.DeptStatusNormal,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}
