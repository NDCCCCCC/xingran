package system

// =====================================================================
// Method Enumeration (Plan 72-10 Task 2)
//
// role_handler.go (279 lines) — method coverage:
//   - Create          POST /system/roles
//   - List            POST /system/roles/list
//   - Statistics      POST /system/roles/statistics
//   - GetByID         POST /system/roles/:id
//   - Update          POST /system/roles/:id/update
//   - Delete          POST /system/roles/:id/delete
//   - BatchDelete     POST /system/roles/batch-delete
//   - UpdateStatus    POST /system/roles/:id/status
//   - GetAllEnabled   POST /system/roles/all
//
// Per CLAUDE.md: RoleStatus 0=normal, 1=stopped (models.RoleStatusEnabled/Disabled).
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

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// setupRoleTestDB creates in-memory SQLite with sys_role + association tables.
func setupRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role (
			id TEXT PRIMARY KEY,
			role_name TEXT NOT NULL UNIQUE,
			role_key TEXT NOT NULL UNIQUE,
			role_sort INTEGER DEFAULT 0,
			data_scope INTEGER DEFAULT 1,
			menu_check_strictly INTEGER DEFAULT 1,
			dept_check_strictly INTEGER DEFAULT 1,
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
		CREATE TABLE sys_role_menu (
			role_id TEXT,
			menu_id TEXT,
			PRIMARY KEY (role_id, menu_id)
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role_dept (
			role_id TEXT,
			dept_id TEXT,
			PRIMARY KEY (role_id, dept_id)
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_menu (
			id TEXT PRIMARY KEY,
			menu_name TEXT,
			perms TEXT,
			order_num INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY,
			dept_name TEXT,
			order_num INTEGER DEFAULT 0,
			status INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

// newRoleTestHandler wires a real RoleService into the handler.
func newRoleTestHandler(t *testing.T, db *gorm.DB) (*RoleHandler, *gorm.DB) {
	t.Helper()
	svc := systemServices.NewRoleService(db)
	h := NewRoleHandler(svc)
	// Inject empty Core so operlog.Record short-circuits via nil checks.
	h.core = &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	}
	return h, db
}

// seedRoleRow inserts a sys_role row directly. Returns id.
func seedRoleRow(t *testing.T, db *gorm.DB, name, key string, status models.RoleStatus) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at)
		VALUES (?, ?, ?, 0, 1, ?, datetime('now'), datetime('now'))`,
		id, name, key, int(status)).Error)
	return id
}

// TC1: Create - success
func TestRoleHandler_Create_Success(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))

	body := requests.RoleCreateRequest{
		RoleName: "测试角色",
		RoleKey:  "test_role",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Create, "POST", "/system/roles", body, nil)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var count int64
	db.Raw("SELECT COUNT(*) FROM sys_role WHERE role_key = ?", "test_role").Scan(&count)
	assert.Equal(t, int64(1), count)
}

// TC2: Create - bad JSON returns 400
func TestRoleHandler_Create_BadJSON(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.Create, "POST", "/system/roles", "{not-json", nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC3: Create - duplicate role_name returns error
func TestRoleHandler_Create_DuplicateName(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "dup", "k1", models.RoleStatusEnabled)

	body := requests.RoleCreateRequest{
		RoleName: "dup",
		RoleKey:  "k2",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Create, "POST", "/system/roles", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC4: Create - duplicate role_key returns error
func TestRoleHandler_Create_DuplicateKey(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "n1", "k1", models.RoleStatusEnabled)

	body := requests.RoleCreateRequest{
		RoleName: "n2",
		RoleKey:  "k1",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Create, "POST", "/system/roles", body, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC5: List - empty
func TestRoleHandler_List_Empty(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.List, "POST", "/system/roles/list", map[string]interface{}{}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(0), resp.Data.Total)
}

// TC6: List - filter by roleName
func TestRoleHandler_List_FilterByName(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "管理员", "admin", models.RoleStatusEnabled)
	seedRoleRow(t, db, "普通用户", "user", models.RoleStatusEnabled)
	seedRoleRow(t, db, "审计员", "auditor", models.RoleStatusEnabled)

	w := doJSON(t, h.List, "POST", "/system/roles/list",
		map[string]interface{}{"roleName": "用户", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// "用户" matches "普通用户" only (1 row) - "管理员" and "审计员" don't contain "用户"
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC6b: List - filter by roleName matches multiple
func TestRoleHandler_List_FilterByName_Multiple(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "管理员", "admin", models.RoleStatusEnabled)
	seedRoleRow(t, db, "普通用户", "user", models.RoleStatusEnabled)
	seedRoleRow(t, db, "审计员", "auditor", models.RoleStatusEnabled)

	w := doJSON(t, h.List, "POST", "/system/roles/list",
		map[string]interface{}{"roleName": "员", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// "员" matches "管理员" and "审计员" (2 rows)
	assert.Equal(t, int64(2), resp.Data.Total)
}

// TC7: List - filter by status
func TestRoleHandler_List_FilterByStatus(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "active1", "k1", models.RoleStatusEnabled)
	seedRoleRow(t, db, "active2", "k2", models.RoleStatusEnabled)
	seedRoleRow(t, db, "inactive1", "k3", models.RoleStatusDisabled)

	w := doJSON(t, h.List, "POST", "/system/roles/list",
		map[string]interface{}{"status": "1", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC8: Statistics - returns counts
func TestRoleHandler_Statistics(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "a1", "k1", models.RoleStatusEnabled)
	seedRoleRow(t, db, "a2", "k2", models.RoleStatusEnabled)
	seedRoleRow(t, db, "i1", "k3", models.RoleStatusDisabled)

	w := doJSON(t, h.Statistics, "POST", "/system/roles/statistics", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total    int64 `json:"total"`
			Active   int64 `json:"active"`
			Inactive int64 `json:"inactive"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(3), resp.Data.Total)
	assert.Equal(t, int64(2), resp.Data.Active)
	assert.Equal(t, int64(1), resp.Data.Inactive)
}

// TC9: Statistics - empty DB
func TestRoleHandler_Statistics_Empty(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.Statistics, "POST", "/system/roles/statistics", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(0), resp.Data.Total)
}

// TC10: GetByID - success
func TestRoleHandler_GetByID_Success(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "管理员", "admin", models.RoleStatusEnabled)

	w := doJSON(t, h.GetByID, "POST", "/system/roles/"+id, nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data models.Role `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "管理员", resp.Data.RoleName)
	assert.Equal(t, "admin", resp.Data.RoleKey)
}

// TC11: GetByID - not found
func TestRoleHandler_GetByID_NotFound(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	missing := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/roles/"+missing, nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC12: GetByID - empty id returns error
func TestRoleHandler_GetByID_EmptyID(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.GetByID, "POST", "/system/roles/", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC13: Update - success
func TestRoleHandler_Update_Success(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "updateme", "upd", models.RoleStatusEnabled)

	body := requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "updateme2",
		RoleKey:  "upd",
		RoleSort: 2,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Update, "POST", "/system/roles/"+id+"/update", body, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var stored models.Role
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "updateme2", stored.RoleName)
}

// TC14: Update - role_name collides with another role
func TestRoleHandler_Update_DuplicateName(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "existing", "k1", models.RoleStatusEnabled)
	id2 := seedRoleRow(t, db, "other", "k2", models.RoleStatusEnabled)

	body := requests.RoleUpdateRequest{
		ID:       id2,
		RoleName: "existing",
		RoleKey:  "k2",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Update, "POST", "/system/roles/"+id2+"/update", body, map[string]string{"id": id2})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC15: Update - bad JSON returns 400
func TestRoleHandler_Update_BadJSON(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "upd", "k1", models.RoleStatusEnabled)
	w := doJSON(t, h.Update, "POST", "/system/roles/"+id+"/update", "{not-json", map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC16: Update - not found returns error
func TestRoleHandler_Update_NotFound(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	missing := uuid.NewString()
	body := requests.RoleUpdateRequest{
		ID:       missing,
		RoleName: "x",
		RoleKey:  "y",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Update, "POST", "/system/roles/"+missing+"/update", body, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC17: Update - role_key collision with another role
func TestRoleHandler_Update_DuplicateKey(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "name1", "common", models.RoleStatusEnabled)
	id2 := seedRoleRow(t, db, "name2", "unique", models.RoleStatusEnabled)

	body := requests.RoleUpdateRequest{
		ID:       id2,
		RoleName: "name2",
		RoleKey:  "common",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Update, "POST", "/system/roles/"+id2+"/update", body, map[string]string{"id": id2})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC18: Delete - success
func TestRoleHandler_Delete_Success(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "delme", "delme", models.RoleStatusEnabled)

	w := doJSON(t, h.Delete, "POST", "/system/roles/"+id+"/delete", nil, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_role WHERE id = ?", id).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt, "row should be soft-deleted")
}

// TC19: Delete - role assigned to users cannot be deleted
func TestRoleHandler_Delete_HasUsers(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "withusers", "withusers", models.RoleStatusEnabled)
	// Assign user to role
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`,
		uuid.NewString(), id).Error)

	w := doJSON(t, h.Delete, "POST", "/system/roles/"+id+"/delete", nil, map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC20: Delete - not found returns error
func TestRoleHandler_Delete_NotFound(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/roles/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC21: Delete - empty id returns error
func TestRoleHandler_Delete_EmptyID(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.Delete, "POST", "/system/roles//delete", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC22: BatchDelete - success
func TestRoleHandler_BatchDelete_Success(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id1 := seedRoleRow(t, db, "b1", "b1", models.RoleStatusEnabled)
	id2 := seedRoleRow(t, db, "b2", "b2", models.RoleStatusEnabled)

	w := doJSON(t, h.BatchDelete, "POST", "/system/roles/batch-delete",
		map[string]interface{}{"ids": []string{id1, id2}}, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var alive int64
	db.Raw("SELECT COUNT(*) FROM sys_role WHERE deleted_at IS NULL").Scan(&alive)
	assert.Equal(t, int64(0), alive)
}

// TC23: BatchDelete - empty ids returns 400
func TestRoleHandler_BatchDelete_Empty(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.BatchDelete, "POST", "/system/roles/batch-delete",
		map[string]interface{}{"ids": []string{}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC24: BatchDelete - has users returns error
func TestRoleHandler_BatchDelete_HasUsers(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id1 := seedRoleRow(t, db, "bu1", "bu1", models.RoleStatusEnabled)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`,
		uuid.NewString(), id1).Error)

	w := doJSON(t, h.BatchDelete, "POST", "/system/roles/batch-delete",
		map[string]interface{}{"ids": []string{id1}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC25: UpdateStatus - enable a disabled role
func TestRoleHandler_UpdateStatus_Enable(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "enableme", "em", models.RoleStatusDisabled)

	w := doJSON(t, h.UpdateStatus, "POST", "/system/roles/"+id+"/status",
		map[string]interface{}{"status": int(models.RoleStatusEnabled)},
		map[string]string{"id": id})
	require.Equal(t, http.StatusOK, w.Code)

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_role WHERE id = ?", id).Scan(&status).Error)
	assert.Equal(t, int(models.RoleStatusEnabled), status)
}

// TC26: UpdateStatus - disable an enabled role
func TestRoleHandler_UpdateStatus_Disable(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "disableme", "dm", models.RoleStatusEnabled)

	w := doJSON(t, h.UpdateStatus, "POST", "/system/roles/"+id+"/status",
		map[string]interface{}{"status": int(models.RoleStatusDisabled)},
		map[string]string{"id": id})
	require.Equal(t, http.StatusOK, w.Code)

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_role WHERE id = ?", id).Scan(&status).Error)
	assert.Equal(t, int(models.RoleStatusDisabled), status)
}

// TC27: UpdateStatus - out of range returns 400
func TestRoleHandler_UpdateStatus_OutOfRange(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "status", "st", models.RoleStatusEnabled)

	w := doJSON(t, h.UpdateStatus, "POST", "/system/roles/"+id+"/status",
		map[string]interface{}{"status": 99},
		map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code, "status=99 violates binding min=0,max=1")
}

// TC28: UpdateStatus - empty id returns error
func TestRoleHandler_UpdateStatus_EmptyID(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.UpdateStatus, "POST", "/system/roles//status",
		map[string]interface{}{"status": 0},
		map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC29: UpdateStatus - not found returns error
func TestRoleHandler_UpdateStatus_NotFound(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	missing := uuid.NewString()
	w := doJSON(t, h.UpdateStatus, "POST", "/system/roles/"+missing+"/status",
		map[string]interface{}{"status": 0},
		map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC30: UpdateStatus - bad JSON returns 400
func TestRoleHandler_UpdateStatus_BadJSON(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "st", "st", models.RoleStatusEnabled)
	w := doJSON(t, h.UpdateStatus, "POST", "/system/roles/"+id+"/status", "{not-json",
		map[string]string{"id": id})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC31: GetAllEnabled - returns only enabled
func TestRoleHandler_GetAllEnabled(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "active1", "a1", models.RoleStatusEnabled)
	seedRoleRow(t, db, "active2", "a2", models.RoleStatusEnabled)
	seedRoleRow(t, db, "inactive", "i1", models.RoleStatusDisabled)

	w := doJSON(t, h.GetAllEnabled, "POST", "/system/roles/all", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Len(t, resp.Data, 2, "only enabled roles should be returned")

	// Verify each simplified role has id, roleName, roleKey
	for _, r := range resp.Data {
		_, hasID := r["id"]
		_, hasName := r["roleName"]
		_, hasKey := r["roleKey"]
		assert.True(t, hasID && hasName && hasKey, "simplified role should have id, roleName, roleKey")
	}
}

// TC32: GetAllEnabled - empty
func TestRoleHandler_GetAllEnabled_Empty(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.GetAllEnabled, "POST", "/system/roles/all", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Empty(t, resp.Data)
}

// TC33: Create - with menuIds and deptIds
func TestRoleHandler_Create_WithMenuAndDept(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	menuID := uuid.NewString()
	deptID := uuid.NewString()
	// Pre-seed a menu/dept so FK constraints would be satisfied (not enforced in sqlite without FK pragma)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, status) VALUES (?, 'm', 0)`, menuID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, status) VALUES (?, 'd', 0)`, deptID).Error)

	body := requests.RoleCreateRequest{
		RoleName: "with-menus",
		RoleKey:  "wmenu",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
		MenuIds:  []string{menuID},
		DeptIds:  []string{deptID},
	}
	w := doJSON(t, h.Create, "POST", "/system/roles", body, nil)
	assert.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var roleID string
	require.NoError(t, db.Raw("SELECT id FROM sys_role WHERE role_key = ?", "wmenu").Scan(&roleID).Error)
	require.NotEmpty(t, roleID)

	var menuCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ?", roleID).Scan(&menuCount)
	assert.Equal(t, int64(1), menuCount)

	var deptCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_dept WHERE role_id = ?", roleID).Scan(&deptCount)
	assert.Equal(t, int64(1), deptCount)
}

// TC34: Update - replace menuIds drops old ones
func TestRoleHandler_Update_ReplaceMenus(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "r-upd", "r-upd", models.RoleStatusEnabled)
	oldMenu := uuid.NewString()
	newMenu := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`,
		id, oldMenu).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, status) VALUES (?, 'm', 0)`, newMenu).Error)

	body := requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "r-upd",
		RoleKey:  "r-upd",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
		MenuIds:  []string{newMenu},
	}
	w := doJSON(t, h.Update, "POST", "/system/roles/"+id+"/update", body, map[string]string{"id": id})
	require.Equal(t, http.StatusOK, w.Code)

	var oldMenuCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ? AND menu_id = ?", id, oldMenu).Scan(&oldMenuCount)
	assert.Equal(t, int64(0), oldMenuCount, "old menu assignment should be removed")

	var newMenuCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ? AND menu_id = ?", id, newMenu).Scan(&newMenuCount)
	assert.Equal(t, int64(1), newMenuCount, "new menu assignment should exist")
}

// TC35: List - service error bubbles up
func TestRoleHandler_List_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	w := doJSON(t, h.List, "POST", "/system/roles/list", map[string]interface{}{}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC36: Statistics - service error bubbles up
func TestRoleHandler_Statistics_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	w := doJSON(t, h.Statistics, "POST", "/system/roles/statistics", nil, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC37: GetByID - service error bubbles up
func TestRoleHandler_GetByID_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	missing := uuid.NewString()
	w := doJSON(t, h.GetByID, "POST", "/system/roles/"+missing, nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC38: Update - service error bubbles up
func TestRoleHandler_Update_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	missing := uuid.NewString()
	body := requests.RoleUpdateRequest{
		ID:       missing,
		RoleName: "x",
		RoleKey:  "y",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Update, "POST", "/system/roles/"+missing+"/update", body, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC39: UpdateStatus - service error bubbles up
func TestRoleHandler_UpdateStatus_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	missing := uuid.NewString()
	w := doJSON(t, h.UpdateStatus, "POST", "/system/roles/"+missing+"/status",
		map[string]interface{}{"status": 0},
		map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC40: WithCore - nil receiver returns nil
func TestRoleHandler_WithCore_NilReceiver(t *testing.T) {
	var h *RoleHandler
	result := h.WithCore(&core.Core{})
	assert.Nil(t, result)
}

// TC41: GetAllEnabled - service error bubbles up
func TestRoleHandler_GetAllEnabled_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	w := doJSON(t, h.GetAllEnabled, "POST", "/system/roles/all", nil, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC42: Delete - empty id returns error
func TestRoleHandler_Delete_EmptyID_v2(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.Delete, "POST", "/system/roles//delete", nil, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC43: BatchDelete - bad JSON returns 400
func TestRoleHandler_BatchDelete_BadJSON(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.BatchDelete, "POST", "/system/roles/batch-delete", "{not-json", nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC44: GetAllEnabled - includes simplified fields only
func TestRoleHandler_GetAllEnabled_SimplifiedFields(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "simplified", "simp", models.RoleStatusEnabled)

	w := doJSON(t, h.GetAllEnabled, "POST", "/system/roles/all", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	role := resp.Data[0]
	assert.Equal(t, "simplified", role["roleName"])
	assert.Equal(t, "simp", role["roleKey"])
	// Should not contain remark, status, etc.
	_, hasRemark := role["remark"]
	assert.False(t, hasRemark, "remark should be stripped from simplified list")
}

// TC45: Update - empty id returns error
func TestRoleHandler_Update_EmptyID(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	body := requests.RoleUpdateRequest{
		RoleName: "x",
		RoleKey:  "y",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	w := doJSON(t, h.Update, "POST", "/system/roles//update", body, map[string]string{"id": ""})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC46: Update - role with no existing menus + empty new MenuIds = no menu rows
func TestRoleHandler_Update_NoMenus(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	id := seedRoleRow(t, db, "no-menus", "nm", models.RoleStatusEnabled)

	body := requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "no-menus",
		RoleKey:  "nm",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
		MenuIds:  []string{},
	}
	w := doJSON(t, h.Update, "POST", "/system/roles/"+id+"/update", body, map[string]string{"id": id})
	assert.Equal(t, http.StatusOK, w.Code)

	var menuCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ?", id).Scan(&menuCount)
	assert.Equal(t, int64(0), menuCount)
}

// TC47: List - filter by roleKey
func TestRoleHandler_List_FilterByRoleKey(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	seedRoleRow(t, db, "admin1", "admin", models.RoleStatusEnabled)
	seedRoleRow(t, db, "user1", "user", models.RoleStatusEnabled)

	w := doJSON(t, h.List, "POST", "/system/roles/list",
		map[string]interface{}{"roleKey": "ad", "current": 1, "pageSize": 10}, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
}

// TC48: List - bad JSON falls back to default params
func TestRoleHandler_List_BadJSON(t *testing.T) {
	h, _ := newRoleTestHandler(t, setupRoleTestDB(t))
	w := doJSON(t, h.List, "POST", "/system/roles/list", "{not-json", nil)
	assert.Equal(t, http.StatusOK, w.Code, "list should tolerate malformed JSON")
}

// TC49: Delete - service error bubbles up
func TestRoleHandler_Delete_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	missing := uuid.NewString()
	w := doJSON(t, h.Delete, "POST", "/system/roles/"+missing+"/delete", nil, map[string]string{"id": missing})
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// TC50: BatchDelete - service error bubbles up
func TestRoleHandler_BatchDelete_ServiceError(t *testing.T) {
	h, db := newRoleTestHandler(t, setupRoleTestDB(t))
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)
	w := doJSON(t, h.BatchDelete, "POST", "/system/roles/batch-delete",
		map[string]interface{}{"ids": []string{uuid.NewString()}}, nil)
	assert.NotEqual(t, http.StatusOK, w.Code)
}
