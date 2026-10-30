package permission

// =====================================================================
// Phase 80-05 Task 2: service.go 81 stmts 全 0% 清欠(8 小包中最大余量 +50)。
//
// 覆盖面:InitDefaultRolesAndMenus / createDefaultAdminRole / assignAllMenusToAdmin
// (幂等差集 + 软删过滤)/ Get·UpdateRoleMenus / Get·UpdateRoleDepts /
// GetUserMenus(递归 CTE,includeHidden 两态)/ buildMenuTree(嵌套)/
// GetUserPermissions(status 过滤)。
//
// 纪律:
//   - glebarez sqlite(t.TempDir 文件库);visible/status 一律引用 models 常量
//     (VisibleShow/VisibleHidden/MenuStatusNormal/MenuStatusStop/RoleStatusEnabled)。
//   - 零 sleep、零 t.Parallel(共享 sqlite fixture)。
// =====================================================================

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// newPms8005 装配 service + sqlite 库(sys_role/sys_menu/sys_role_menu/
// sys_role_dept/sys_user_role 五表)。
func newPms8005(t *testing.T) (*service, *gorm.DB) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "pms8005.db")), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, gormDB.AutoMigrate(
		&models.Role{},
		&models.Menu{},
		&models.RoleMenu{},
		&models.RoleDept{},
		&models.UserRole{},
	))

	s, ok := NewService().(*service)
	require.True(t, ok, "NewService 应返回 *service 具体类型")
	return s, gormDB
}

// seedMenu8005 建一条菜单行(必填项:MenuName;可选项按参数)。
// QUIRK-80-05-C:Menu.Visible 带 gorm:"default:1" 标签,Create 时零值
// (VisibleHidden=0)被省略 → 落库为 DB default 1(与 QUIRK-80-03-D 同型)。
// 故 VisibleHidden 场景 Create 后必须显式 Update(Update 不省略零值)。
func seedMenu8005(t *testing.T, db *gorm.DB, m *models.Menu) string {
	t.Helper()
	// 意图必须在 Create 前捕获:GORM 会在 Create 后把 default 标签字段的
	// 零值回填为 DB default(Create(VisibleHidden) → m.Visible 变回 1)。
	wantHidden := m.Visible == models.VisibleHidden
	require.NoError(t, db.Create(m).Error)
	if wantHidden {
		require.NoError(t, db.Model(&models.Menu{}).Where("id = ?", m.ID).
			Update("visible", models.VisibleHidden).Error)
	}
	return m.ID
}

// seedRole8005 建一条角色行,返回角色 ID。
func seedRole8005(t *testing.T, db *gorm.DB, roleName, roleKey string) string {
	t.Helper()
	role := models.Role{
		RoleName: roleName,
		RoleKey:  roleKey,
		DataScope: models.DataScopeAll,
		Status:   models.RoleStatusEnabled,
	}
	require.NoError(t, db.Create(&role).Error)
	return role.ID
}

// TestPms8005_InitDefaultRolesAndMenus:空库 → Init → admin 角色落库;
// 幂等:二次 Init 不重复不报错。
func TestPms8005_InitDefaultRolesAndMenus(t *testing.T) {
	s, db := newPms8005(t)

	require.NoError(t, s.InitDefaultRolesAndMenus(db))

	var roles []models.Role
	require.NoError(t, db.Where("role_key = ?", "admin").Find(&roles).Error)
	require.Len(t, roles, 1, "应恰好创建一个 admin 角色")
	assert.Equal(t, "超级管理员", roles[0].RoleName)
	assert.Equal(t, models.RoleStatusEnabled, roles[0].Status)

	// 幂等:第二次 Init 不报错、不产生重复行。
	require.NoError(t, s.InitDefaultRolesAndMenus(db))
	var count int64
	require.NoError(t, db.Model(&models.Role{}).Where("role_key = ?", "admin").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestPms8005_CreateDefaultAdminRole:空库直调 → admin 行;已存在 → 跳过分支。
func TestPms8005_CreateDefaultAdminRole(t *testing.T) {
	s, db := newPms8005(t)

	require.NoError(t, s.createDefaultAdminRole(db))
	var role models.Role
	require.NoError(t, db.Where("role_key = ?", "admin").First(&role).Error)
	assert.Equal(t, "超级管理员", role.RoleName)
	assert.True(t, role.MenuCheckStrictly)
	assert.True(t, role.DeptCheckStrictly)

	// 已存在 → count != 0 → 跳过 Create 分支(不报错、不重复)。
	require.NoError(t, s.createDefaultAdminRole(db))
	var count int64
	require.NoError(t, db.Model(&models.Role{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestPms8005_AssignAllMenusToAdmin:种子 3 菜单 → 关联数 == 菜单数;
// 幂等快速路径(差集为空秒过);软删菜单被 Pluck 过滤不进差集。
func TestPms8005_AssignAllMenusToAdmin(t *testing.T) {
	s, db := newPms8005(t)
	roleID := seedRole8005(t, db, "超级管理员", "admin")
	_ = roleID

	m1 := seedMenu8005(t, db, &models.Menu{MenuName: "系统管理", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	m2 := seedMenu8005(t, db, &models.Menu{MenuName: "用户管理", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal, ParentID: &m1})
	m3 := seedMenu8005(t, db, &models.Menu{MenuName: "角色管理", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal, ParentID: &m1})

	require.NoError(t, s.assignAllMenusToAdmin(db))

	var admin models.Role
	require.NoError(t, db.Where("role_key = ?", "admin").First(&admin).Error)
	ids, err := s.GetRoleMenus(db, admin.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{m1, m2, m3}, ids, "admin 应关联全部现存菜单")

	// 幂等:第二次调用走差集为空快速路径,关联不重复。
	require.NoError(t, s.assignAllMenusToAdmin(db))
	var count int64
	require.NoError(t, db.Model(&models.RoleMenu{}).Where("role_id = ?", admin.ID).Count(&count).Error)
	assert.Equal(t, int64(3), count)

	// 软删菜单:差集 Pluck 自动过滤 deleted_at IS NULL。
	// (字符串主键须用条件式 Delete;直传 string 会被 GORM 当裸 SQL 内联,
	//  生成无引号的 WHERE <uuid> → sqlite "unrecognized token"。)
	require.NoError(t, db.Delete(&models.Menu{}, "id = ?", m3).Error)
	require.NoError(t, s.assignAllMenusToAdmin(db))
	ids, err = s.GetRoleMenus(db, admin.ID)
	require.NoError(t, err)
	assert.Len(t, ids, 3, "assign 不清理陈旧关联(设计说明),仍为 3 条")

	// admin 角色不存在 → First 报错分支。
	s2, db2 := newPms8005(t)
	require.Error(t, s2.assignAllMenusToAdmin(db2), "无 admin 角色时应报 ErrRecordNotFound")
}

// TestPms8005_RoleMenus_RoundTrip:Update(3)→Get 一致;Update 空集 → 清空;
// Get 不存在角色 → 空集。
func TestPms8005_RoleMenus_RoundTrip(t *testing.T) {
	s, db := newPms8005(t)
	roleID := seedRole8005(t, db, "访客", "guest")

	m1 := seedMenu8005(t, db, &models.Menu{MenuName: "菜单A", MenuType: models.MenuTypeMenu})
	m2 := seedMenu8005(t, db, &models.Menu{MenuName: "菜单B", MenuType: models.MenuTypeMenu})
	m3 := seedMenu8005(t, db, &models.Menu{MenuName: "菜单C", MenuType: models.MenuTypeMenu})

	require.NoError(t, s.UpdateRoleMenus(db, roleID, []string{m1, m2, m3}))
	got, err := s.GetRoleMenus(db, roleID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{m1, m2, m3}, got)

	// 覆盖式更新为子集。
	require.NoError(t, s.UpdateRoleMenus(db, roleID, []string{m2}))
	got, err = s.GetRoleMenus(db, roleID)
	require.NoError(t, err)
	assert.Equal(t, []string{m2}, got)

	// 空集 → 事务清空关联。
	require.NoError(t, s.UpdateRoleMenus(db, roleID, nil))
	got, err = s.GetRoleMenus(db, roleID)
	require.NoError(t, err)
	assert.Empty(t, got)

	// 不存在角色 → 空集(无错误)。
	got, err = s.GetRoleMenus(db, "no-such-role")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestPms8005_RoleDepts_RoundTrip:GetRoleDepts/UpdateRoleDepts 往返 + 空集清空。
func TestPms8005_RoleDepts_RoundTrip(t *testing.T) {
	s, db := newPms8005(t)
	roleID := seedRole8005(t, db, "区域管理员", "region-admin")

	require.NoError(t, s.UpdateRoleDepts(db, roleID, []string{"dept-a", "dept-b", "dept-c"}))
	got, err := s.GetRoleDepts(db, roleID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dept-a", "dept-b", "dept-c"}, got)

	require.NoError(t, s.UpdateRoleDepts(db, roleID, []string{"dept-a"}))
	got, err = s.GetRoleDepts(db, roleID)
	require.NoError(t, err)
	assert.Equal(t, []string{"dept-a"}, got)

	require.NoError(t, s.UpdateRoleDepts(db, roleID, nil))
	got, err = s.GetRoleDepts(db, roleID)
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = s.GetRoleDepts(db, "no-such-role")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestPms8005_GetUserMenus:种子用户-角色-菜单链 → includeHidden true/false
// 两态(models.VisibleShow/VisibleHidden 常量)+ buildMenuTree 父子嵌套。
func TestPms8005_GetUserMenus(t *testing.T) {
	s, db := newPms8005(t)

	roleID := seedRole8005(t, db, "普通角色", "user-role")
	userID := "user-8005"
	require.NoError(t, db.Create(&models.UserRole{UserID: userID, RoleID: roleID}).Error)

	// 树形:root(M)→ child(C, 显示)→ hiddenChild(C, 隐藏)。
	root := seedMenu8005(t, db, &models.Menu{MenuName: "根目录", MenuType: models.MenuTypeDir, Visible: models.VisibleShow, Status: models.MenuStatusNormal})
	child := seedMenu8005(t, db, &models.Menu{MenuName: "子菜单", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal, ParentID: &root})
	hidden := seedMenu8005(t, db, &models.Menu{MenuName: "隐藏菜单", MenuType: models.MenuTypeMenu, Visible: models.VisibleHidden, Status: models.MenuStatusNormal, ParentID: &root})

	require.NoError(t, s.UpdateRoleMenus(db, roleID, []string{root, child, hidden}))

	// includeHidden=true:含隐藏菜单,树形嵌套断言。
	menus, err := s.GetUserMenus(db, userID, true)
	require.NoError(t, err)
	require.Len(t, menus, 1, "根集合应只有 parentID 为空的根目录")
	assert.Equal(t, root, menus[0].ID)
	require.Len(t, menus[0].Children, 2, "根目录下应有 2 个子菜单(含隐藏)")
	childIDs := []string{menus[0].Children[0].ID, menus[0].Children[1].ID}
	assert.ElementsMatch(t, []string{child, hidden}, childIDs)

	// includeHidden=false:VisibleShow 过滤,只剩 root + 显示子菜单。
	menus, err = s.GetUserMenus(db, userID, false)
	require.NoError(t, err)
	require.Len(t, menus, 1)
	assert.Equal(t, root, menus[0].ID)
	require.Len(t, menus[0].Children, 1, "隐藏菜单应被 visible 过滤")
	assert.Equal(t, child, menus[0].Children[0].ID)

	// 无权限用户 → 空树。
	menus, err = s.GetUserMenus(db, "nobody", true)
	require.NoError(t, err)
	assert.Empty(t, menus)
}

// TestPms8005_GetUserPermissions:perms 聚合只看 status(停用菜单的权限无效),
// 不看 visible(隐藏菜单权限仍有效)。
func TestPms8005_GetUserPermissions(t *testing.T) {
	s, db := newPms8005(t)

	roleID := seedRole8005(t, db, "权限角色", "perm-role")
	userID := "perm-user-8005"
	require.NoError(t, db.Create(&models.UserRole{UserID: userID, RoleID: roleID}).Error)

	perms := func(s string) *string { return &s }
	m1 := seedMenu8005(t, db, &models.Menu{MenuName: "正常+隐藏", MenuType: models.MenuTypeMenu, Visible: models.VisibleHidden, Status: models.MenuStatusNormal, Perms: perms("sys:user:list")})
	m2 := seedMenu8005(t, db, &models.Menu{MenuName: "正常+显示", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal, Perms: perms("sys:user:add")})
	m3 := seedMenu8005(t, db, &models.Menu{MenuName: "停用", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusStop, Perms: perms("sys:user:delete")})
	m4 := seedMenu8005(t, db, &models.Menu{MenuName: "无perms", MenuType: models.MenuTypeMenu, Visible: models.VisibleShow, Status: models.MenuStatusNormal})

	require.NoError(t, s.UpdateRoleMenus(db, roleID, []string{m1, m2, m3, m4}))

	got, err := s.GetUserPermissions(db, userID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"sys:user:list", "sys:user:add"}, got,
		"停用菜单权限无效;隐藏菜单权限仍有效;空 perms 排除")

	got, err = s.GetUserPermissions(db, "nobody")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestPms8005_BuildMenuTree:纯函数面 —— 空输入 / 平铺集合按 parentID 分层 /
// 多级嵌套。
func TestPms8005_BuildMenuTree(t *testing.T) {
	s := NewService().(*service)

	assert.Empty(t, s.buildMenuTree(nil, ""))

	// 平铺三级:A(root) → B(父 A) → C(父 B)。
	pA := "id-a"
	pB := "id-b"
	menus := []models.Menu{
		{BaseModel: models.BaseModel{ID: "id-b"}, MenuName: "B", ParentID: &pA},
		{BaseModel: models.BaseModel{ID: "id-c"}, MenuName: "C", ParentID: &pB},
		{BaseModel: models.BaseModel{ID: "id-a"}, MenuName: "A"},
		{BaseModel: models.BaseModel{ID: "id-x"}, MenuName: "X", ParentID: &pB}, // X 与 C 同层
	}

	tree := s.buildMenuTree(menus, "")
	require.Len(t, tree, 1, "根集合应只有 A")
	require.Equal(t, "id-a", tree[0].ID)
	require.Len(t, tree[0].Children, 1, "A 下应只有 B")
	require.Len(t, tree[0].Children[0].Children, 2, "B 下应有 C 与 X")
}
