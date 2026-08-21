package system

// =====================================================================
// role_service_test.go — covers role_service.go (515 lines)
// Extends existing role_service_apperrors_test.go (PRESERVED)
//
// Per Plan 72-10 Task 3
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

// setupRoleServiceTestDB creates in-memory SQLite for role service tests.
func setupRoleServiceTestDB(t *testing.T) *gorm.DB {
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

// seedRoleSvc inserts a sys_role row directly. Returns id.
func seedRoleSvc(t *testing.T, db *gorm.DB, name, key string, status models.RoleStatus) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO sys_role (id, role_name, role_key, role_sort, data_scope, status, created_at, updated_at, version)
		VALUES (?, ?, ?, 0, 1, ?, datetime('now'), datetime('now'), 0)`,
		id, name, key, int(status)).Error)
	return id
}

// TC1: Create - success
func TestRoleService_Create_Success(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	req := &requests.RoleCreateRequest{
		RoleName: "管理员",
		RoleKey:  "admin",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var stored models.Role
	require.NoError(t, db.Where("role_key = ?", "admin").First(&stored).Error)
	assert.Equal(t, "管理员", stored.RoleName)
}

// TC2: Create - with menuIds
func TestRoleService_Create_WithMenuIds(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	menuID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, status) VALUES (?, 'm', 0)`, menuID).Error)

	req := &requests.RoleCreateRequest{
		RoleName: "with-menus",
		RoleKey:  "wm",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
		MenuIds:  []string{menuID},
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var roleID string
	require.NoError(t, db.Raw("SELECT id FROM sys_role WHERE role_key = ?", "wm").Scan(&roleID).Error)
	var count int64
	db.Raw("SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ?", roleID).Scan(&count)
	assert.Equal(t, int64(1), count)
}

// TC3: Create - with deptIds
func TestRoleService_Create_WithDeptIds(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	deptID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, status) VALUES (?, 'd', 0)`, deptID).Error)

	req := &requests.RoleCreateRequest{
		RoleName: "with-depts",
		RoleKey:  "wd",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
		DeptIds:  []string{deptID},
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var roleID string
	require.NoError(t, db.Raw("SELECT id FROM sys_role WHERE role_key = ?", "wd").Scan(&roleID).Error)
	var count int64
	db.Raw("SELECT COUNT(*) FROM sys_role_dept WHERE role_id = ?", roleID).Scan(&count)
	assert.Equal(t, int64(1), count)
}

// TC4: Update - success
func TestRoleService_Update_Success(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "upd", "upd", models.RoleStatusEnabled)

	req := &requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "upd2",
		RoleKey:  "upd",
		RoleSort: 2,
		Status:   models.RoleStatusEnabled,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var stored models.Role
	require.NoError(t, db.Where("id = ?", id).First(&stored).Error)
	assert.Equal(t, "upd2", stored.RoleName)
	assert.Equal(t, 2, stored.RoleSort)
}

// TC5: Update - replaces menuIds
func TestRoleService_Update_ReplaceMenuIds(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "r", "r", models.RoleStatusEnabled)
	oldMenu := uuid.NewString()
	newMenu := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, id, oldMenu).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, status) VALUES (?, 'm', 0)`, newMenu).Error)

	req := &requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "r",
		RoleKey:  "r",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
		MenuIds:  []string{newMenu},
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var oldCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ? AND menu_id = ?", id, oldMenu).Scan(&oldCount)
	assert.Equal(t, int64(0), oldCount)
}

// TC6: Delete - success
func TestRoleService_Delete_Success(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "del", "del", models.RoleStatusEnabled)

	require.NoError(t, svc.Delete(context.Background(), id))

	var deletedAt *string
	require.NoError(t, db.Raw("SELECT deleted_at FROM sys_role WHERE id = ?", id).Scan(&deletedAt).Error)
	assert.NotNil(t, deletedAt)
}

// TC7: Delete - cascades sys_role_menu
func TestRoleService_Delete_CascadesRoles(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "c", "c", models.RoleStatusEnabled)
	menuID := uuid.NewString()
	deptID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, id, menuID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)`, id, deptID).Error)

	require.NoError(t, svc.Delete(context.Background(), id))

	var menuCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ?", id).Scan(&menuCount)
	assert.Equal(t, int64(0), menuCount)
}

// TC8: Delete - has users returns error
func TestRoleService_Delete_HasUsers(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "hu", "hu", models.RoleStatusEnabled)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, uuid.NewString(), id).Error)

	err := svc.Delete(context.Background(), id)
	assert.Error(t, err)
}

// TC9: GetByID - success
func TestRoleService_GetByID_Success(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "g", "g", models.RoleStatusEnabled)

	role, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "g", role.RoleKey)
}

// TC10: GetByID - not found
func TestRoleService_GetByID_NotFound(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	role, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
	assert.Nil(t, role)
}

// TC11: List - empty
func TestRoleService_List_Empty(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	result, err := svc.List(context.Background(), requests.DefaultRoleListParams())
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
}

// TC12: List - filter by name
func TestRoleService_List_FilterByName(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "管理员", "admin", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "普通用户", "user", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "审计员", "auditor", models.RoleStatusEnabled)

	params := requests.DefaultRoleListParams()
	params.RoleName = "员"
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}

// TC13: List - filter by key
func TestRoleService_List_FilterByKey(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "admin1", "admin", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "user1", "user", models.RoleStatusEnabled)

	params := requests.DefaultRoleListParams()
	params.RoleKey = "ad"
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC14: List - filter by status
func TestRoleService_List_FilterByStatus(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "a1", "k1", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "a2", "k2", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "i1", "k3", models.RoleStatusDisabled)

	params := requests.DefaultRoleListParams()
	params.Status = "1"
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
}

// TC15: List - custom orderBy
func TestRoleService_List_CustomOrderBy(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "n1", "sys.n1", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "n2", "sys.n2", models.RoleStatusEnabled)

	params := requests.DefaultRoleListParams()
	params.OrderByColumn = "role_key"
	asc := true
	params.IsAsc = &asc
	result, err := svc.List(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
}

// TC16: Statistics - returns counts
func TestRoleService_Statistics(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "a1", "k1", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "a2", "k2", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "i1", "k3", models.RoleStatusDisabled)

	result, err := svc.Statistics(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	assert.Equal(t, int64(2), result.Active)
	assert.Equal(t, int64(1), result.Inactive)
}

// TC17: GetAllEnabled - returns only enabled
func TestRoleService_GetAllEnabled(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "a1", "k1", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "a2", "k2", models.RoleStatusEnabled)
	seedRoleSvc(t, db, "i1", "k3", models.RoleStatusDisabled)

	roles, err := svc.GetAllEnabled(context.Background())
	require.NoError(t, err)
	assert.Len(t, roles, 2)
	for _, r := range roles {
		assert.Equal(t, models.RoleStatusEnabled, r.Status)
	}
}

// TC18: BatchDelete - success
func TestRoleService_BatchDelete_Success(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id1 := seedRoleSvc(t, db, "b1", "b1", models.RoleStatusEnabled)
	id2 := seedRoleSvc(t, db, "b2", "b2", models.RoleStatusEnabled)

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}))

	var alive int64
	db.Raw("SELECT COUNT(*) FROM sys_role WHERE deleted_at IS NULL").Scan(&alive)
	assert.Equal(t, int64(0), alive)
}

// TC19: BatchDelete - empty ids fails
func TestRoleService_BatchDelete_Empty(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	err := svc.BatchDelete(context.Background(), []string{})
	assert.Error(t, err)
}

// TC20: BatchDelete - has users fails
func TestRoleService_BatchDelete_HasUsers(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id1 := seedRoleSvc(t, db, "bu", "bu", models.RoleStatusEnabled)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`, uuid.NewString(), id1).Error)

	err := svc.BatchDelete(context.Background(), []string{id1})
	assert.Error(t, err)
}

// TC21: UpdateStatus - disable
func TestRoleService_UpdateStatus_Disable(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "d", "d", models.RoleStatusEnabled)

	require.NoError(t, svc.UpdateStatus(context.Background(), id, int(models.RoleStatusDisabled)))

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_role WHERE id = ?", id).Scan(&status).Error)
	assert.Equal(t, int(models.RoleStatusDisabled), status)
}

// TC22: UpdateStatus - enable
func TestRoleService_UpdateStatus_Enable(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "e", "e", models.RoleStatusDisabled)

	require.NoError(t, svc.UpdateStatus(context.Background(), id, int(models.RoleStatusEnabled)))

	var status int
	require.NoError(t, db.Raw("SELECT status FROM sys_role WHERE id = ?", id).Scan(&status).Error)
	assert.Equal(t, int(models.RoleStatusEnabled), status)
}

// TC23: UpdateStatus - not found
func TestRoleService_UpdateStatus_NotFound(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	err := svc.UpdateStatus(context.Background(), uuid.NewString(), 0)
	assert.Error(t, err)
}

// TC24: GetAllEnabledWithCache - returns enabled
func TestRoleService_GetAllEnabledWithCache(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "a1", "k1", models.RoleStatusEnabled)

	roles, err := svc.GetAllEnabledWithCache(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, roles)
}

// TC25: GetMenusWithCache - empty when no menus assigned
func TestRoleService_GetMenusWithCache_NoMenus(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "r", "r", models.RoleStatusEnabled)

	menus, err := svc.GetMenusWithCache(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, menus)
}

// TC26: GetMenusWithCache - returns menus for role
func TestRoleService_GetMenusWithCache_WithMenus(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "r", "r", models.RoleStatusEnabled)
	menuID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_menu (id, menu_name, perms, status) VALUES (?, 'm', 'p', 0)`, menuID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`, id, menuID).Error)

	menus, err := svc.GetMenusWithCache(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, menus, 1)
}

// TC27: GetDeptsWithCache - empty when no depts assigned
func TestRoleService_GetDeptsWithCache_NoDepts(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "r", "r", models.RoleStatusEnabled)

	depts, err := svc.GetDeptsWithCache(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, depts)
}

// TC28: GetDeptsWithCache - returns depts for role
func TestRoleService_GetDeptsWithCache_WithDepts(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "r", "r", models.RoleStatusEnabled)
	deptID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, status) VALUES (?, 'd', 0)`, deptID).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)`, id, deptID).Error)

	depts, err := svc.GetDeptsWithCache(context.Background(), id)
	require.NoError(t, err)
	assert.Len(t, depts, 1)
}

// TC29: InvalidateRoleCache - no-op
func TestRoleService_InvalidateRoleCache(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, svc.InvalidateRoleCache(context.Background(), uuid.NewString()))
}

// TC30: checkRoleNameExists - true when exists
func TestRoleService_CheckRoleNameExists_True(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "exists", "exists", models.RoleStatusEnabled)

	// Use internal method via struct assertion
	rs := svc.(*roleService)
	exists, err := rs.checkRoleNameExists(context.Background(), "exists", "")
	require.NoError(t, err)
	assert.True(t, exists)
}

// TC31: checkRoleNameExists - false when not exists
func TestRoleService_CheckRoleNameExists_False(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)

	rs := svc.(*roleService)
	exists, err := rs.checkRoleNameExists(context.Background(), "missing", "")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TC32: checkRoleKeyExists - true
func TestRoleService_CheckRoleKeyExists_True(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	seedRoleSvc(t, db, "n", "exists", models.RoleStatusEnabled)

	rs := svc.(*roleService)
	exists, err := rs.checkRoleKeyExists(context.Background(), "exists", "")
	require.NoError(t, err)
	assert.True(t, exists)
}

// TC33: List - service error bubbles up
func TestRoleService_List_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	_, err := svc.List(context.Background(), requests.DefaultRoleListParams())
	assert.Error(t, err)
}

// TC34: Statistics - service error
func TestRoleService_Statistics_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	_, err := svc.Statistics(context.Background())
	assert.Error(t, err)
}

// TC35: GetByID - service error
func TestRoleService_GetByID_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC36: Update - service error
func TestRoleService_Update_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	err := svc.Update(context.Background(), &requests.RoleUpdateRequest{
		ID:       uuid.NewString(),
		RoleName: "x",
		RoleKey:  "y",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	})
	assert.Error(t, err)
}

// TC37: Create - service error
func TestRoleService_Create_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	err := svc.Create(context.Background(), &requests.RoleCreateRequest{
		RoleName: "x",
		RoleKey:  "y",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	})
	assert.Error(t, err)
}

// TC38: Delete - service error
func TestRoleService_Delete_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	err := svc.Delete(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC39: BatchDelete - service error
func TestRoleService_BatchDelete_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	err := svc.BatchDelete(context.Background(), []string{uuid.NewString()})
	assert.Error(t, err)
}

// TC40: UpdateStatus - service error
func TestRoleService_UpdateStatus_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	err := svc.UpdateStatus(context.Background(), uuid.NewString(), 0)
	assert.Error(t, err)
}

// TC41: GetAllEnabled - service error
func TestRoleService_GetAllEnabled_ServiceError(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_role").Error)

	_, err := svc.GetAllEnabled(context.Background())
	assert.Error(t, err)
}

// TC42: Update - excludes self when checking name uniqueness
func TestRoleService_Update_ExcludesSelf(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "self", "self", models.RoleStatusEnabled)

	// Update with same name - should succeed (exclude self)
	req := &requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "self",
		RoleKey:  "self",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
	}
	require.NoError(t, svc.Update(context.Background(), req))
}

// TC43: Update - replaces deptIds
func TestRoleService_Update_ReplaceDeptIds(t *testing.T) {
	db := setupRoleServiceTestDB(t)
	svc := NewRoleService(db)
	id := seedRoleSvc(t, db, "r", "r", models.RoleStatusEnabled)
	oldDept := uuid.NewString()
	newDept := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES (?, ?)`, id, oldDept).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, dept_name, status) VALUES (?, 'd', 0)`, newDept).Error)

	req := &requests.RoleUpdateRequest{
		ID:       id,
		RoleName: "r",
		RoleKey:  "r",
		RoleSort: 1,
		Status:   models.RoleStatusEnabled,
		DeptIds:  []string{newDept},
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var oldCount int64
	db.Raw("SELECT COUNT(*) FROM sys_role_dept WHERE role_id = ? AND dept_id = ?", id, oldDept).Scan(&oldCount)
	assert.Equal(t, int64(0), oldCount)
}
