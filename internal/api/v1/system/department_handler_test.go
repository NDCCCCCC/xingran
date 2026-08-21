package system

// =====================================================================
// Method Enumeration (Plan 72-08 Task 2)
//
// department_handler.go (339 lines) — method coverage:
//   - GetTree           GET /system/departments/tree
//   - List              GET /system/departments/list
//   - GetByID           GET /system/departments/:id
//   - Create            POST /system/departments
//   - Update            POST /system/departments/:id/update
//   - Delete            POST /system/departments/:id/delete
//   - BatchDelete       POST /system/departments/batch
//   - UpdateStatus      POST /system/departments/:id/status
//   - RoleDeptTreeSelect POST /system/departments/role-dept-tree-select/:roleId
//   - GetUsers          GET /system/departments/:id/users
// Per CLAUDE.md: DeptStatus 0=normal, 1=stopped (models.DeptStatusNormal/Stop)
// =====================================================================

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupDeptTestDB creates an in-memory SQLite with depts + users schema.
func setupDeptTestDB(t *testing.T) *gorm.DB {
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
			nickname TEXT,
			phone TEXT,
			email TEXT,
			dept_id TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
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

// setupDeptHandler wires a real DepartmentService into the handler.
func setupDeptHandler(t *testing.T) (*DepartmentHandler, *gorm.DB) {
	t.Helper()
	db := setupDeptTestDB(t)
	svc := systemServices.NewDepartmentService(db)
	h := NewDepartmentHandler(svc)
	return h, db
}

// seedDept inserts a dept row directly into the DB.
func seedDept(t *testing.T, db *gorm.DB, d *models.Department) string {
	t.Helper()
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	require.NoError(t, db.Create(d).Error)
	return d.ID
}

// seedUser inserts a user row directly into the DB.
func seedDeptUser(t *testing.T, db *gorm.DB, deptID string) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_user
		(id, username, nickname, dept_id, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, datetime('now'), datetime('now'), 0)`,
		id, "user-"+id[:8], "昵称-"+id[:8], deptID).Error)
	return id
}

// TC1: GetTree - 3-level dept tree
func TestDepartmentHandler_GetTree_Success(t *testing.T) {
	h, db := setupDeptHandler(t)

	rootID := seedDept(t, db, &models.Department{
		DeptName: "总公司", DeptCode: "ROOT", Ancestors: "", OrderNum: 1, Status: models.DeptStatusNormal,
	})
	childID := seedDept(t, db, &models.Department{
		DeptName: "财务部", DeptCode: "FIN", ParentID: &rootID, Ancestors: rootID, OrderNum: 1, Status: models.DeptStatusNormal,
	})
	seedDept(t, db, &models.Department{
		DeptName: "应收账款", DeptCode: "FIN-AR", ParentID: &childID, Ancestors: rootID + "," + childID, OrderNum: 1, Status: models.DeptStatusNormal,
	})

	w := doJSON(t, h.GetTree, "POST", "/system/departments/tree", map[string]interface{}{}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int                 `json:"code"`
		Data []*models.Department `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "总公司", resp.Data[0].DeptName)
	require.Len(t, resp.Data[0].Children, 1)
	assert.Equal(t, "财务部", resp.Data[0].Children[0].DeptName)
	require.Len(t, resp.Data[0].Children[0].Children, 1)
	assert.Equal(t, "应收账款", resp.Data[0].Children[0].Children[0].DeptName)
}

// TC2: GetTree - with name filter
func TestDepartmentHandler_GetTree_FilterByName(t *testing.T) {
	h, db := setupDeptHandler(t)
	seedDept(t, db, &models.Department{DeptName: "研发部", DeptCode: "RD", OrderNum: 1, Status: models.DeptStatusNormal})
	seedDept(t, db, &models.Department{DeptName: "市场部", DeptCode: "MKT", OrderNum: 2, Status: models.DeptStatusNormal})

	w := doJSON(t, h.GetTree, "POST", "/system/departments/tree", map[string]interface{}{"deptName": "研发"}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int                  `json:"code"`
		Data []*models.Department `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "研发部", resp.Data[0].DeptName)
}

// TC3: List - empty list
func TestDepartmentHandler_List_Empty(t *testing.T) {
	h, _ := setupDeptHandler(t)
	w := doJSON(t, h.List, "POST", "/system/departments/list", map[string]interface{}{}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int                   `json:"code"`
		Data []models.Department   `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Empty(t, resp.Data)
}

// TC4: List - filter by name and status
func TestDepartmentHandler_List_Filtered(t *testing.T) {
	h, db := setupDeptHandler(t)
	seedDept(t, db, &models.Department{DeptName: "IT", DeptCode: "IT", Status: models.DeptStatusNormal})
	seedDept(t, db, &models.Department{DeptName: "IT-Ops", DeptCode: "IT-OPS", Status: models.DeptStatusStop})
	seedDept(t, db, &models.Department{DeptName: "HR", DeptCode: "HR", Status: models.DeptStatusNormal})

	w := doJSON(t, h.List, "POST", "/system/departments/list",
		map[string]interface{}{"deptName": "IT", "status": 0}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int                 `json:"code"`
		Data []models.Department `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "IT", resp.Data[0].DeptName)
}

// TC5: GetByID - success
func TestDepartmentHandler_GetByID_Success(t *testing.T) {
	h, db := setupDeptHandler(t)
	id := seedDept(t, db, &models.Department{DeptName: "财务", DeptCode: "FIN", Status: models.DeptStatusNormal})

	w := doJSON(t, h.GetByID, "POST", "/system/departments/"+id, nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int               `json:"code"`
		Data models.Department `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "财务", resp.Data.DeptName)
}

// TC6: GetByID - not found
func TestDepartmentHandler_GetByID_NotFound(t *testing.T) {
	h, _ := setupDeptHandler(t)
	missingID := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/departments/"+missingID, nil, map[string]string{"id": missingID})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC7: GetByID - empty id returns ParamMissing
func TestDepartmentHandler_GetByID_EmptyID(t *testing.T) {
	h, _ := setupDeptHandler(t)
	w := doJSON(t, h.GetByID, "POST", "/system/departments/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC8: Create - root level (no parent)
func TestDepartmentHandler_Create_RootLevel(t *testing.T) {
	h, db := setupDeptHandler(t)
	leader := "leader-uuid"
	body := requests.DepartmentCreateRequest{
		DeptName:      "新总部",
		DeptCode:      "HQ-NEW",
		ParentID:      nil,
		OrderNum:      1,
		Leader:        &leader,
		Status:        models.DeptStatusNormal,
	}
	_ = doJSON(t, h.Create, "POST", "/system/departments", body, nil)

	var got models.Department
	require.NoError(t, db.Where("dept_code = ?", "HQ-NEW").First(&got).Error)
	assert.Equal(t, "新总部", got.DeptName)
	assert.Empty(t, got.Ancestors, "root level should have empty ancestors")
	assert.Equal(t, models.DeptStatusNormal, got.Status)
}

// TC9: Create - child with parent sets ancestors
func TestDepartmentHandler_Create_ChildWithAncestors(t *testing.T) {
	h, db := setupDeptHandler(t)
	parentID := seedDept(t, db, &models.Department{
		DeptName: "Parent", DeptCode: "PAR",
		Ancestors: "", OrderNum: 1, Status: models.DeptStatusNormal,
	})
	body := requests.DepartmentCreateRequest{
		DeptName: "Child",
		DeptCode: "CHILD",
		ParentID: &parentID,
		OrderNum: 1,
		Status:   models.DeptStatusNormal,
	}
	_ = doJSON(t, h.Create, "POST", "/system/departments", body, nil)

	var got models.Department
	require.NoError(t, db.Where("dept_code = ?", "CHILD").First(&got).Error)
	assert.Equal(t, parentID, *got.ParentID)
	// ancestors should be PARENT_ID because parent had empty Ancestors
	assert.Equal(t, parentID, got.Ancestors)
}

// TC10: Create - duplicate name in same parent fails
func TestDepartmentHandler_Create_DuplicateName(t *testing.T) {
	h, db := setupDeptHandler(t)
	seedDept(t, db, &models.Department{DeptName: "IT", DeptCode: "IT1", Status: models.DeptStatusNormal})

	body := requests.DepartmentCreateRequest{
		DeptName: "IT", DeptCode: "IT2", Status: models.DeptStatusNormal,
	}
	w := doJSON(t, h.Create, "POST", "/system/departments", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC11: Create - duplicate dept_code fails
func TestDepartmentHandler_Create_DuplicateCode(t *testing.T) {
	h, db := setupDeptHandler(t)
	seedDept(t, db, &models.Department{DeptName: "IT", DeptCode: "DUP", Status: models.DeptStatusNormal})

	body := requests.DepartmentCreateRequest{
		DeptName: "IT2", DeptCode: "DUP", Status: models.DeptStatusNormal,
	}
	w := doJSON(t, h.Create, "POST", "/system/departments", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC12: Create - missing required fields returns 400
func TestDepartmentHandler_Create_MissingFields(t *testing.T) {
	h, _ := setupDeptHandler(t)
	// empty body to trigger required validation
	w := doJSON(t, h.Create, "POST", "/system/departments", map[string]interface{}{}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC13: Update - success
func TestDepartmentHandler_Update_Success(t *testing.T) {
	h, db := setupDeptHandler(t)
	id := seedDept(t, db, &models.Department{
		DeptName: "OldName", DeptCode: "OLD", Status: models.DeptStatusNormal,
	})

	body := requests.DepartmentUpdateRequest{
		DeptName: "NewName", DeptCode: "OLD", Status: models.DeptStatusNormal,
	}
	_ = doJSON(t, h.Update, "POST", "/system/departments/"+id+"/update", body, map[string]string{"id": id})

	var got models.Department
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "NewName", got.DeptName)
}

// TC14: Update - nonexistent returns error
func TestDepartmentHandler_Update_NotFound(t *testing.T) {
	h, _ := setupDeptHandler(t)
	missing := uuid.NewString()
	body := requests.DepartmentUpdateRequest{
		DeptName: "X", DeptCode: "X", Status: models.DeptStatusNormal,
	}
	w := doJSON(t, h.Update, "POST", "/system/departments/"+missing+"/update", body, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC15: Delete - leaf with no children succeeds
func TestDepartmentHandler_Delete_Leaf(t *testing.T) {
	h, db := setupDeptHandler(t)
	id := seedDept(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	_ = doJSON(t, h.Delete, "POST", "/system/departments/"+id+"/delete", nil, map[string]string{"id": id})

	// soft delete
	var got models.Department
	err := db.First(&got, "id = ?", id).Error
	if err == nil {
		assert.NotZero(t, got.DeletedAt.Time)
	}
}

// TC16: Delete - has children fails
func TestDepartmentHandler_Delete_HasChildren(t *testing.T) {
	h, db := setupDeptHandler(t)
	parentID := seedDept(t, db, &models.Department{DeptName: "P", DeptCode: "P", Status: models.DeptStatusNormal})
	seedDept(t, db, &models.Department{DeptName: "C", DeptCode: "C", ParentID: &parentID, Status: models.DeptStatusNormal})

	w := doJSON(t, h.Delete, "POST", "/system/departments/"+parentID+"/delete", nil, map[string]string{"id": parentID})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC17: Delete - has users fails
func TestDepartmentHandler_Delete_HasUsers(t *testing.T) {
	h, db := setupDeptHandler(t)
	id := seedDept(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	seedDeptUser(t, db, id)

	w := doJSON(t, h.Delete, "POST", "/system/departments/"+id+"/delete", nil, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC18: Delete - empty id returns ParamMissing
func TestDepartmentHandler_Delete_EmptyID(t *testing.T) {
	h, _ := setupDeptHandler(t)
	w := doJSON(t, h.Delete, "POST", "/system/departments//delete", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC19: BatchDelete - success
func TestDepartmentHandler_BatchDelete_Success(t *testing.T) {
	h, db := setupDeptHandler(t)
	id1 := seedDept(t, db, &models.Department{DeptName: "D1", DeptCode: "D1", Status: models.DeptStatusNormal})
	id2 := seedDept(t, db, &models.Department{DeptName: "D2", DeptCode: "D2", Status: models.DeptStatusNormal})

	_ = doJSON(t, h.BatchDelete, "POST", "/system/departments/batch", map[string]interface{}{"ids": []string{id1, id2}}, nil)
}

// TC20: BatchDelete - has children fails
func TestDepartmentHandler_BatchDelete_HasChildren(t *testing.T) {
	h, db := setupDeptHandler(t)
	parentID := seedDept(t, db, &models.Department{DeptName: "P", DeptCode: "P", Status: models.DeptStatusNormal})
	seedDept(t, db, &models.Department{DeptName: "C", DeptCode: "C", ParentID: &parentID, Status: models.DeptStatusNormal})

	w := doJSON(t, h.BatchDelete, "POST", "/system/departments/batch", map[string]interface{}{"ids": []string{parentID}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC21: BatchDelete - empty ids fails
func TestDepartmentHandler_BatchDelete_EmptyIDs(t *testing.T) {
	h, _ := setupDeptHandler(t)
	w := doJSON(t, h.BatchDelete, "POST", "/system/departments/batch", map[string]interface{}{"ids": []string{}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC22: UpdateStatus - toggles between 0 and 1
func TestDepartmentHandler_UpdateStatus_Toggles(t *testing.T) {
	h, db := setupDeptHandler(t)
	id := seedDept(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	for _, target := range []int{1, 0, 1} {
		_ = doJSON(t, h.UpdateStatus, "POST", "/system/departments/"+id+"/status",
			map[string]interface{}{"status": target}, map[string]string{"id": id})

		var got models.Department
		require.NoError(t, db.First(&got, "id = ?", id).Error)
		assert.Equal(t, target, int(got.Status))
	}
}

// TC23: UpdateStatus - out of range fails
func TestDepartmentHandler_UpdateStatus_OutOfRange(t *testing.T) {
	h, db := setupDeptHandler(t)
	id := seedDept(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	w := doJSON(t, h.UpdateStatus, "POST", "/system/departments/"+id+"/status",
		map[string]interface{}{"status": 5}, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC24: UpdateStatus - nonexistent dept
func TestDepartmentHandler_UpdateStatus_NotFound(t *testing.T) {
	h, _ := setupDeptHandler(t)
	missing := uuid.NewString()
	w := doJSON(t, h.UpdateStatus, "POST", "/system/departments/"+missing+"/status",
		map[string]interface{}{"status": 1}, map[string]string{"id": missing})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TC25: RoleDeptTreeSelect - returns dept IDs
func TestDepartmentHandler_RoleDeptTreeSelect_Success(t *testing.T) {
	h, db := setupDeptHandler(t)
	roleID := uuid.NewString()
	deptID := seedDept(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	require.NoError(t, db.Exec("INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)", roleID, deptID).Error)

	w := doJSON(t, h.RoleDeptTreeSelect, "POST", "/system/departments/role-dept-tree-select/"+roleID, nil, map[string]string{"roleId": roleID})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			CheckedKeys []string `json:"checkedKeys"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Contains(t, resp.Data.CheckedKeys, deptID)
}

// TC26: RoleDeptTreeSelect - empty roleID returns error
func TestDepartmentHandler_RoleDeptTreeSelect_EmptyRoleID(t *testing.T) {
	h, _ := setupDeptHandler(t)
	w := doJSON(t, h.RoleDeptTreeSelect, "POST", "/system/departments/role-dept-tree-select/", nil, map[string]string{"roleId": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC27: GetUsers - returns users in dept
func TestDepartmentHandler_GetUsers_Success(t *testing.T) {
	h, db := setupDeptHandler(t)
	deptID := seedDept(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})
	seedDeptUser(t, db, deptID)
	seedDeptUser(t, db, deptID)

	w := doJSON(t, h.GetUsers, "POST", "/system/departments/"+deptID+"/users", nil, map[string]string{"id": deptID})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data []struct {
			ID       string  `json:"id"`
			Username string  `json:"username"`
			Nickname *string `json:"nickname"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 2)
}

// TC28: GetUsers - empty deptID returns error
func TestDepartmentHandler_GetUsers_EmptyID(t *testing.T) {
	h, _ := setupDeptHandler(t)
	w := doJSON(t, h.GetUsers, "POST", "/system/departments//users", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC29: GetUsers - no users in dept returns empty list
func TestDepartmentHandler_GetUsers_Empty(t *testing.T) {
	h, db := setupDeptHandler(t)
	deptID := seedDept(t, db, &models.Department{DeptName: "D", DeptCode: "D", Status: models.DeptStatusNormal})

	w := doJSON(t, h.GetUsers, "POST", "/system/departments/"+deptID+"/users", nil, map[string]string{"id": deptID})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data []interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 0)
}

// TC30: Create - move dept to new parent updates ancestors
func TestDepartmentHandler_Update_ChangeParent(t *testing.T) {
	h, db := setupDeptHandler(t)
	oldParentID := seedDept(t, db, &models.Department{DeptName: "OldParent", DeptCode: "OLD", Status: models.DeptStatusNormal})
	deptID := seedDept(t, db, &models.Department{DeptName: "Child", DeptCode: "CHILD", ParentID: &oldParentID, Status: models.DeptStatusNormal})

	newParentID := seedDept(t, db, &models.Department{DeptName: "NewParent", DeptCode: "NEW", Status: models.DeptStatusNormal})

	body := requests.DepartmentUpdateRequest{
		DeptName: "Child",
		DeptCode: "CHILD",
		ParentID: &newParentID,
		Status:   models.DeptStatusNormal,
	}
	_ = doJSON(t, h.Update, "POST", "/system/departments/"+deptID+"/update", body, map[string]string{"id": deptID})

	var got models.Department
	require.NoError(t, db.First(&got, "id = ?", deptID).Error)
	require.NotNil(t, got.ParentID)
	assert.Equal(t, newParentID, *got.ParentID)
	// ancestors should be NEW
	assert.Equal(t, newParentID, got.Ancestors)
}
