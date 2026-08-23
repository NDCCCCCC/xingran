package system

// =====================================================================
// menu_service_test.go — internal/services/system
// Covers menu_service.go (603 lines) using glebarez sqlite + real MenuService
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

// setupMenuServiceDB creates an in-memory SQLite with the menu schema.
func setupMenuServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`
		CREATE TABLE sys_menu (
			id TEXT PRIMARY KEY,
			menu_name TEXT NOT NULL,
			parent_id TEXT,
			order_num INTEGER DEFAULT 0,
			path TEXT,
			component TEXT,
			menu_type TEXT DEFAULT 'M',
			visible INTEGER DEFAULT 1,
			status INTEGER DEFAULT 0,
			perms TEXT,
			icon TEXT,
			remark TEXT,
			meta TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER DEFAULT 0
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_user_role (
			user_id TEXT,
			role_id TEXT,
			PRIMARY KEY (user_id, role_id)
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_role_menu (
			role_id TEXT,
			menu_id TEXT,
			PRIMARY KEY (role_id, menu_id)
		)
	`).Error)
	return db
}

func seedMenuDirect(t *testing.T, db *gorm.DB, m *models.Menu) string {
	t.Helper()
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	// Use raw SQL to bypass GORM zero-value skipping (Visible=0 would be skipped)
	// and to ensure `visible` is stored exactly as specified.
	var parentID, path, component, perms, icon, remark, meta *string
	if m.ParentID != nil {
		parentID = m.ParentID
	}
	if m.Path != nil {
		path = m.Path
	}
	if m.Component != nil {
		component = m.Component
	}
	if m.Perms != nil {
		perms = m.Perms
	}
	if m.Icon != nil {
		icon = m.Icon
	}
	if m.Remark != "" {
		r := m.Remark
		remark = &r
	}
	_ = meta
	require.NoError(t, db.Exec(`INSERT INTO sys_menu
		(id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'), 0)`,
		m.ID, m.MenuName, parentID, m.OrderNum, path, component, string(m.MenuType), int(m.Visible), int(m.Status), perms, icon, remark).Error)
	return m.ID
}

// TC1: Create - success
func TestMenuService_Create_Success(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)

	req := &requests.MenuCreateRequest{
		MenuName: "测试菜单",
		MenuType: models.MenuTypeMenu,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	err := svc.Create(context.Background(), req)
	require.NoError(t, err)

	var got models.Menu
	require.NoError(t, db.Where("menu_name = ?", "测试菜单").First(&got).Error)
	assert.Equal(t, "测试菜单", got.MenuName)
}

// TC2: Create - duplicate name in same parent fails
func TestMenuService_Create_DuplicateName(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	seedMenuDirect(t, db, &models.Menu{MenuName: "Dup", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	req := &requests.MenuCreateRequest{
		MenuName: "Dup",
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

// TC3: Create - parent empty string normalized to nil
func TestMenuService_Create_EmptyParent_Normalized(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	empty := ""
	req := &requests.MenuCreateRequest{
		MenuName: "Root",
		ParentID: &empty,
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Menu
	require.NoError(t, db.Where("menu_name = ?", "Root").First(&got).Error)
	assert.Nil(t, got.ParentID)
}

// TC3b: Create - parent "0" normalized to nil
func TestMenuService_Create_ZeroParent_Normalized(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	zero := "0"
	req := &requests.MenuCreateRequest{
		MenuName: "RootFromZero",
		ParentID: &zero,
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	require.NoError(t, svc.Create(context.Background(), req))

	var got models.Menu
	require.NoError(t, db.Where("menu_name = ?", "RootFromZero").First(&got).Error)
	assert.Nil(t, got.ParentID)
}

// TC3c: Update - parent "0" normalized to nil
func TestMenuService_Update_ZeroParent_Normalized(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	id := seedMenuDirect(t, db, &models.Menu{MenuName: "Old", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	zero := "0"
	req := &requests.MenuUpdateRequest{
		ID:       id,
		MenuName: "Updated",
		ParentID: &zero,
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Menu
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Nil(t, got.ParentID)
}

// TC3d: normalizeParentID unit test covering nil/""/"0"/valid UUID
func TestNormalizeParentID(t *testing.T) {
	empty := ""
	zero := "0"
	uuid := "a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"

	cases := []struct {
		name     string
		input    *string
		expected *string
	}{
		{"nil", nil, nil},
		{"empty", &empty, nil},
		{"zero", &zero, nil},
		{"uuid", &uuid, &uuid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeParentID(tc.input)
			if tc.expected == nil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, *tc.expected, *got)
			}
		})
	}
}

// TC4: Update - success
func TestMenuService_Update_Success(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	id := seedMenuDirect(t, db, &models.Menu{MenuName: "Old", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	req := &requests.MenuUpdateRequest{
		ID:       id,
		MenuName: "New",
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleHidden,
		Status:   models.MenuStatusNormal,
		OrderNum: 5,
	}
	require.NoError(t, svc.Update(context.Background(), req))

	var got models.Menu
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, "New", got.MenuName)
	assert.Equal(t, 5, got.OrderNum)
	assert.Equal(t, models.VisibleHidden, got.Visible)
}

// TC5: Update - not found
func TestMenuService_Update_NotFound(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	req := &requests.MenuUpdateRequest{
		ID:       uuid.NewString(),
		MenuName: "X",
		MenuType: models.MenuTypeDir,
		Visible:  models.VisibleShow,
		Status:   models.MenuStatusNormal,
	}
	err := svc.Update(context.Background(), req)
	assert.Error(t, err)
}

// TC6: Delete - leaf
func TestMenuService_Delete_Leaf(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	id := seedMenuDirect(t, db, &models.Menu{MenuName: "D", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, svc.Delete(context.Background(), id, false))

	// soft delete
	var got models.Menu
	err := db.First(&got, "id = ?", id).Error
	if err == nil {
		assert.NotZero(t, got.DeletedAt.Time)
	}
}

// TC7: Delete - has children, no cascade fails
func TestMenuService_Delete_HasChildrenNoCascade(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	parentID := seedMenuDirect(t, db, &models.Menu{MenuName: "P", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	seedMenuDirect(t, db, &models.Menu{MenuName: "C", ParentID: &parentID, MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	err := svc.Delete(context.Background(), parentID, false)
	assert.Error(t, err)
}

// TC8: Delete - cascade=true succeeds
func TestMenuService_Delete_Cascade(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	parentID := seedMenuDirect(t, db, &models.Menu{MenuName: "P", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	seedMenuDirect(t, db, &models.Menu{MenuName: "C", ParentID: &parentID, MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, svc.Delete(context.Background(), parentID, true))
}

// TC9: Delete - not found
func TestMenuService_Delete_NotFound(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	err := svc.Delete(context.Background(), uuid.NewString(), false)
	assert.Error(t, err)
}

// TC10: GetByID - success
func TestMenuService_GetByID_Success(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	id := seedMenuDirect(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	menu, err := svc.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "M", menu.MenuName)
}

// TC11: GetByID - not found
func TestMenuService_GetByID_NotFound(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	_, err := svc.GetByID(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC12: GetTree - 3-level chain
func TestMenuService_GetTree_Success(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	rootID := seedMenuDirect(t, db, &models.Menu{MenuName: "Root", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal, OrderNum: 1})
	childID := seedMenuDirect(t, db, &models.Menu{MenuName: "Child", ParentID: &rootID, MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal, OrderNum: 1})
	seedMenuDirect(t, db, &models.Menu{MenuName: "Grand", ParentID: &childID, MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal, OrderNum: 1})

	tree, err := svc.GetTree(context.Background())
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "Root", tree[0].MenuName)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "Child", tree[0].Children[0].MenuName)
	require.Len(t, tree[0].Children[0].Children, 1)
	assert.Equal(t, "Grand", tree[0].Children[0].Children[0].MenuName)
}

// TC13: List - filter by name
func TestMenuService_List_FilterByName(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	seedMenuDirect(t, db, &models.Menu{MenuName: "权限", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	seedMenuDirect(t, db, &models.Menu{MenuName: "角色", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	menus, err := svc.List(context.Background(), requests.MenuListParams{MenuName: "权限"})
	require.NoError(t, err)
	require.Len(t, menus, 1)
	assert.Equal(t, "权限", menus[0].MenuName)
}

// TC14: List - filter by status
func TestMenuService_List_FilterByStatus(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	seedMenuDirect(t, db, &models.Menu{MenuName: "Active", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	seedMenuDirect(t, db, &models.Menu{MenuName: "Stopped", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusStop})

	menus, err := svc.List(context.Background(), requests.MenuListParams{Status: "0"})
	require.NoError(t, err)
	require.Len(t, menus, 1)
	assert.Equal(t, "Active", menus[0].MenuName)
}

// TC15: BatchDelete - success
func TestMenuService_BatchDelete_Success(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	id1 := seedMenuDirect(t, db, &models.Menu{MenuName: "M1", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	id2 := seedMenuDirect(t, db, &models.Menu{MenuName: "M2", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, svc.BatchDelete(context.Background(), []string{id1, id2}, false))
}

// TC16: BatchDelete - empty fails
func TestMenuService_BatchDelete_Empty(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	err := svc.BatchDelete(context.Background(), []string{}, false)
	assert.Error(t, err)
}

// TC17: BatchDelete - has children, no cascade fails
func TestMenuService_BatchDelete_HasChildrenNoCascade(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	parentID := seedMenuDirect(t, db, &models.Menu{MenuName: "P", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	seedMenuDirect(t, db, &models.Menu{MenuName: "C", ParentID: &parentID, MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	err := svc.BatchDelete(context.Background(), []string{parentID}, false)
	assert.Error(t, err)
}

// TC18: UpdateStatus - success
func TestMenuService_UpdateStatus_Success(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	id := seedMenuDirect(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, svc.UpdateStatus(context.Background(), id, 1))

	var got models.Menu
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, models.MenuStatusStop, got.Status)
}

// TC19: UpdateStatus - not found
func TestMenuService_UpdateStatus_NotFound(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	err := svc.UpdateStatus(context.Background(), uuid.NewString(), 1)
	assert.Error(t, err)
}

// TC20: GetRoleMenuIDs - returns role's menu IDs
func TestMenuService_GetRoleMenuIDs(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	roleID := uuid.NewString()
	menuID := seedMenuDirect(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	require.NoError(t, db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error)

	var ids []string
	require.NoError(t, svc.GetRoleMenuIDs(context.Background(), roleID, &ids))
	assert.Contains(t, ids, menuID)
}

// TC21: GetUserMenus - returns visible+enabled menus
func TestMenuService_GetUserMenus_HidesDisabledAndHidden(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	userID := uuid.NewString()
	roleID := uuid.NewString()

	visible := seedMenuDirect(t, db, &models.Menu{MenuName: "Visible", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	hidden := seedMenuDirect(t, db, &models.Menu{MenuName: "Hidden", MenuType: models.MenuTypeMenu, Visible: models.VisibleHidden, Status: models.MenuStatusNormal})
	stopped := seedMenuDirect(t, db, &models.Menu{MenuName: "Stopped", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusStop})

	require.NoError(t, db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)", userID, roleID).Error)
	for _, m := range []string{visible, hidden, stopped} {
		require.NoError(t, db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, m).Error)
	}

	menus, err := svc.GetUserMenus(context.Background(), userID)
	require.NoError(t, err)
	t.Logf("GetUserMenus returned %d menus", len(menus))
	for _, m := range menus {
		t.Logf("  - %s visible=%d status=%d", m.MenuName, int(m.Visible), int(m.Status))
	}
	names := make(map[string]bool)
	for _, m := range menus {
		names[m.MenuName] = true
	}
	assert.True(t, names["Visible"], "visible+enabled should be included")
	assert.False(t, names["Hidden"], "hidden should be excluded")
	assert.False(t, names["Stopped"], "stopped should be excluded")
}

// TC22: GetAllUserMenus - includes hidden
func TestMenuService_GetAllUserMenus_IncludesHidden(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	userID := uuid.NewString()
	roleID := uuid.NewString()

	visible := seedMenuDirect(t, db, &models.Menu{MenuName: "Visible", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	hidden := seedMenuDirect(t, db, &models.Menu{MenuName: "Hidden", MenuType: models.MenuTypeMenu, Visible: models.VisibleHidden, Status: models.MenuStatusNormal})

	require.NoError(t, db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)", userID, roleID).Error)
	for _, m := range []string{visible, hidden} {
		require.NoError(t, db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, m).Error)
	}

	menus, err := svc.GetAllUserMenus(context.Background(), userID)
	require.NoError(t, err)
	names := make(map[string]bool)
	for _, m := range menus {
		names[m.MenuName] = true
	}
	assert.True(t, names["Visible"])
	assert.True(t, names["Hidden"], "GetAllUserMenus should include hidden menus")
}

// TC23: GetUserPermissions - returns perms
func TestMenuService_GetUserPermissions(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	userID := uuid.NewString()
	roleID := uuid.NewString()

	perms := "user:create"
	seedMenuDirect(t, db, &models.Menu{MenuName: "Button", MenuType: models.MenuTypeButton, Visible: models.VisibleShow, Status: models.MenuStatusNormal, Perms: &perms})

	// bind user role & role menu
	menuID := seedMenuDirect(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	require.NoError(t, db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)", userID, roleID).Error)
	// can't re-query by name; seed directly with the menu we have by querying
	var got models.Menu
	require.NoError(t, db.Where("menu_name = ?", "Button").First(&got).Error)
	require.NoError(t, db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, got.ID).Error)
	_ = menuID

	permsList, err := svc.GetUserPermissions(context.Background(), userID)
	require.NoError(t, err)
	assert.Contains(t, permsList, "user:create")
}

// TC24: GetUserMenus - empty user (no roles)
func TestMenuService_GetUserMenus_NoRoles(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	menus, err := svc.GetUserMenus(context.Background(), uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, menus)
}

// TC25: GetUserPermissions - empty user
func TestMenuService_GetUserPermissions_NoRoles(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	perms, err := svc.GetUserPermissions(context.Background(), uuid.NewString())
	require.NoError(t, err)
	assert.Empty(t, perms)
}

// TC26: GetTreeWithCache - delegates to GetTree
func TestMenuService_GetTreeWithCache(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	seedMenuDirect(t, db, &models.Menu{MenuName: "M", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	tree, err := svc.GetTreeWithCache(context.Background(), false)
	require.NoError(t, err)
	assert.NotEmpty(t, tree)
}

// TC27: GetRouterDataWithCache - returns only enabled
func TestMenuService_GetRouterDataWithCache(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	seedMenuDirect(t, db, &models.Menu{MenuName: "Active", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	seedMenuDirect(t, db, &models.Menu{MenuName: "Stopped", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusStop})

	menus, err := svc.GetRouterDataWithCache(context.Background())
	require.NoError(t, err)
	names := make(map[string]bool)
	for _, m := range menus {
		names[m.MenuName] = true
	}
	assert.True(t, names["Active"])
	assert.False(t, names["Stopped"], "stopped should not be in router data")
}

// TC28: InvalidateMenuCache - no-op returns nil
func TestMenuService_InvalidateMenuCache(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	require.NoError(t, svc.InvalidateMenuCache(context.Background()))
}

// TC29: InvalidateUserMenuCache - no-op returns nil
func TestMenuService_InvalidateUserMenuCache(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	require.NoError(t, svc.InvalidateUserMenuCache(context.Background()))
}

// TC30: List - empty params returns all
func TestMenuService_List_Empty(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	seedMenuDirect(t, db, &models.Menu{MenuName: "M1", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	seedMenuDirect(t, db, &models.Menu{MenuName: "M2", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	menus, err := svc.List(context.Background(), requests.MenuListParams{})
	require.NoError(t, err)
	assert.Len(t, menus, 2)
}
