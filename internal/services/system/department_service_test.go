package system

// =====================================================================
// department_service_test.go — covers department_service.go (499 lines)
// using glebarez sqlite + real DepartmentService
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

func setupDeptServiceDB(t *testing.T) *gorm.DB {
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

func seedDeptDirect(t *testing.T, db *gorm.DB, d *models.Department) string {
	t.Helper()
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	require.NoError(t, db.Create(d).Error)
	return d.ID
}

// TC1: Create - root dept
func TestDeptService_Create_Root(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)

	req := &requests.DepartmentCreateRequest{
		DeptName: "Root",
		DeptCode: "ROOT",
		Status:   models.DeptStatusNormal,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Department
	require.NoError(t, db.Where("dept_code = ?", "ROOT").First(&got).Error)
	assert.Equal(t, "Root", got.DeptName)
}

// TC2: Create - duplicate name fails
func TestDeptService_Create_DuplicateName(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D1", Status: models.DeptStatusNormal})

	req := &requests.DepartmentCreateRequest{
		DeptName: "D", DeptCode: "D2", Status: models.DeptStatusNormal,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC3: Create - duplicate code fails
func TestDeptService_Create_DuplicateCode(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	seedDeptDirect(t, db, &models.Department{DeptName: "D1", DeptCode: "DUP", Status: models.DeptStatusNormal})

	req := &requests.DepartmentCreateRequest{
		DeptName: "D2", DeptCode: "DUP", Status: models.DeptStatusNormal,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC4: Create - child sets ancestors
func TestDeptService_Create_ChildWithAncestors(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	parentID := seedDeptDirect(t, db, &models.Department{
		DeptName: "P", DeptCode: "P", Ancestors: "", Status: models.DeptStatusNormal,
	})
	req := &requests.DepartmentCreateRequest{
		DeptName: "C", DeptCode: "C", ParentID: &parentID, Status: models.DeptStatusNormal,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Department
	require.NoError(t, db.Where("dept_code = ?", "C").First(&got).Error)
	assert.Equal(t, parentID, got.Ancestors)
}

// TC5: Update - success
func TestDeptService_Update_Success(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	id := seedDeptDirect(t, db, &models.Department{DeptName: "Old", DeptCode: "OLD", Status: models.DeptStatusNormal})

	req := &requests.DepartmentUpdateRequest{
		ID:       id,
		DeptName: "New",
		DeptCode: "OLD",
		Status:   models.DeptStatusNormal,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Department
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.DeptName)
}

// TC6: Update - not found
func TestDeptService_Update_NotFound(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	req := &requests.DepartmentUpdateRequest{
		ID:       uuid.NewString(),
		DeptName: "X", DeptCode: "X", Status: models.DeptStatusNormal,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
}

// TC7: Delete - leaf
func TestDeptService_Delete_Leaf(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	id := seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	require.NoError(t, svc.Delete(context.Background(), id))
}

// TC8: Delete - has children fails
func TestDeptService_Delete_HasChildren(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	parentID := seedDeptDirect(t, db, &models.Department{DeptName: "P", DeptCode: "P", Status: models.DeptStatusNormal})
	seedDeptDirect(t, db, &models.Department{DeptName: "C", DeptCode: "C", ParentID: &parentID, Status: models.DeptStatusNormal})

	err := svc.Delete(context.Background(), parentID)
	assert.Error(t, err)
}

// TC9: Delete - has users fails
func TestDeptService_Delete_HasUsers(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	id := seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	require.NoError(t, db.Exec("INSERT INTO sys_user (id, username, dept_id, created_at, updated_at, version) VALUES (?, ?, ?, datetime('now'), datetime('now'), 0)", uuid.NewString(), "u1", id).Error)

	err := svc.Delete(context.Background(), id)
	assert.Error(t, err)
}

// TC10: Delete - not found
func TestDeptService_Delete_NotFound(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC11: GetByID - success
func TestDeptService_GetByID_Success(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	id := seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	dept, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "D", dept.DeptName)
}

// TC12: GetByID - not found
func TestDeptService_GetByID_NotFound(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC13: GetTree - 3-level tree
func TestDeptService_GetTree_Success(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	rootID := seedDeptDirect(t, db, &models.Department{
		DeptName: "Root", DeptCode: "ROOT", Ancestors: "", Status: models.DeptStatusNormal,
	})
	childID := seedDeptDirect(t, db, &models.Department{
		DeptName: "Child", DeptCode: "CHILD", ParentID: &rootID, Ancestors: rootID, Status: models.DeptStatusNormal,
	})
	seedDeptDirect(t, db, &models.Department{
		DeptName: "Grand", DeptCode: "GRAND", ParentID: &childID, Ancestors: rootID + "," + childID, Status: models.DeptStatusNormal,
	})

	tree, err := svc.GetTree(context.Background(), true)
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "Root", tree[0].DeptName)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "Child", tree[0].Children[0].DeptName)
	require.Len(t, tree[0].Children[0].Children, 1)
	assert.Equal(t, "Grand", tree[0].Children[0].Children[0].DeptName)
}

// TC14: GetTree - includeDisabled=false filters stopped
func TestDeptService_GetTree_ExcludesDisabled(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	seedDeptDirect(t, db, &models.Department{DeptName: "Active", DeptCode: "A", Ancestors: "", Status: models.DeptStatusNormal})
	seedDeptDirect(t, db, &models.Department{DeptName: "Stopped", DeptCode: "S", Ancestors: "", Status: models.DeptStatusStop})

	tree, err := svc.GetTree(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "Active", tree[0].DeptName)
}

// TC15: GetTreeWithFilter - filter by name
func TestDeptService_GetTreeWithFilter_ByName(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	seedDeptDirect(t, db, &models.Department{DeptName: "研发", DeptCode: "RD", Ancestors: "", Status: models.DeptStatusNormal})
	seedDeptDirect(t, db, &models.Department{DeptName: "销售", DeptCode: "SALES", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetTreeWithFilter(context.Background(), true, requests.DepartmentListParams{DeptName: "研发"})
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "研发", tree[0].DeptName)
}

// TC16: List - filter by status
func TestDeptService_List_FilterByStatus(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	seedDeptDirect(t, db, &models.Department{DeptName: "A", DeptCode: "A", Status: models.DeptStatusNormal})
	seedDeptDirect(t, db, &models.Department{DeptName: "B", DeptCode: "B", Status: models.DeptStatusStop})

	status := int(models.DeptStatusNormal)
	depts, err := svc.List(context.Background(), requests.DepartmentListParams{Status: &status})
	require.NoError(t, err)
	require.Len(t, depts, 1)
	assert.Equal(t, "A", depts[0].DeptName)
}

// TC17: BatchDelete - empty ids fails
func TestDeptService_BatchDelete_Empty(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	err := svc.BatchDelete(context.Background(), []string{})
	assert.Error(t, err)
}

// TC18: BatchDelete - has children fails
func TestDeptService_BatchDelete_HasChildren(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	parentID := seedDeptDirect(t, db, &models.Department{DeptName: "P", DeptCode: "P", Status: models.DeptStatusNormal})
	seedDeptDirect(t, db, &models.Department{DeptName: "C", DeptCode: "C", ParentID: &parentID, Status: models.DeptStatusNormal})

	err := svc.BatchDelete(context.Background(), []string{parentID})
	assert.Error(t, err)
}

// TC19: BatchDelete - has users fails
func TestDeptService_BatchDelete_HasUsers(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	id := seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	require.NoError(t, db.Exec("INSERT INTO sys_user (id, username, dept_id, created_at, updated_at, version) VALUES (?, ?, ?, datetime('now'), datetime('now'), 0)", uuid.NewString(), "u1", id).Error)

	err := svc.BatchDelete(context.Background(), []string{id})
	assert.Error(t, err)
}

// TC20: UpdateStatus - success
func TestDeptService_UpdateStatus_Success(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	id := seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	require.NoError(t, svc.UpdateStatus(context.Background(), id, 1))

	var got models.Department
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, models.DeptStatusStop, got.Status)
}

// TC21: UpdateStatus - not found
func TestDeptService_UpdateStatus_NotFound(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	err := svc.UpdateStatus(context.Background(), uuid.NewString(), 1)
	assert.Error(t, err)
}

// TC22: GetRoleDeptIDs - returns IDs
func TestDeptService_GetRoleDeptIDs(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	roleID := uuid.NewString()
	deptID := seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	require.NoError(t, db.Exec("INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)", roleID, deptID).Error)

	var ids []string
	require.NoError(t, svc.GetRoleDeptIDs(context.Background(), roleID, &ids))
	assert.Contains(t, ids, deptID)
}

// TC23: GetTreeWithCache - delegates
func TestDeptService_GetTreeWithCache(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetTreeWithCache(context.Background(), true)
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

// TC24: GetSelectDataWithCache - delegates
func TestDeptService_GetSelectDataWithCache(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "D", Ancestors: "", Status: models.DeptStatusNormal})

	tree, err := svc.GetSelectDataWithCache(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

// TC25: InvalidateDeptCache - no-op
func TestDeptService_InvalidateDeptCache(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	require.NoError(t, svc.InvalidateDeptCache(context.Background()))
}

// TC26: GetDB - returns db
func TestDeptService_GetDB(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	got := svc.GetDB()
	assert.Equal(t, db, got)
}

// TC27: List - empty
func TestDeptService_List_Empty(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	depts, err := svc.List(context.Background(), requests.DepartmentListParams{})
	require.NoError(t, err)
	assert.Empty(t, depts)
}

// TC28: GetTreeWithFilter - search returns ancestors when needed
func TestDeptService_GetTreeWithFilter_SearchExpandsAncestors(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	rootID := seedDeptDirect(t, db, &models.Department{DeptName: "公司", DeptCode: "CORP", Ancestors: "", Status: models.DeptStatusNormal})
	seedDeptDirect(t, db, &models.Department{DeptName: "技术研发", DeptCode: "RD", ParentID: &rootID, Ancestors: rootID, Status: models.DeptStatusNormal})

	tree, err := svc.GetTreeWithFilter(context.Background(), true, requests.DepartmentListParams{DeptName: "研发"})
	require.NoError(t, err)
	require.NotEmpty(t, tree)
	// recursive search: root should be present with the child (技术研发) nested
	// walk down to find the matched dept
	found := false
	var walk func(d *models.Department)
	walk = func(d *models.Department) {
		if d.DeptName == "技术研发" {
			found = true
		}
		for _, c := range d.Children {
			walk(c)
		}
	}
	for _, d := range tree {
		walk(d)
	}
	assert.True(t, found, "recursive search should include 技术研发")
}
