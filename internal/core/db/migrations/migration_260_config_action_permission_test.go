package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Migration 260 测试 — system:config 四个动作细码 F 型播种（孤儿码转正）
//
// 覆盖：
//   1) 声明块与种子表 1:1 一致（防漂移）
//   2) 播种 4 个 F 型按钮行，父菜单 = 参数配置（perms=system:config）
//   3) 精准授权：只授给「已持有父菜单」的角色，不碰其它角色
//   4) 幂等：重复执行不新增行、不重复授权
//   5) 父菜单缺失时非阻断（不建行、不报错）
//   6) 端到端：播种后 menuService 口径的 perms 查询能取到这 4 个码
//
// 防假绿：码值统一引用包内 configActionPermissionSeeds，不硬编码字面量。
// ============================================================================

func freshSQLiteDBForMigrate260(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 复用 migration_226 测试的 schema statements（sys_menu / sys_role / sys_role_menu），
	// 再用 AutoMigrate 补齐本用例额外需要的列（Role 的时间戳/role_key、User/UserRole 关联表）。
	// 注：这是 migration 单测（验证播种与授权 SQL 语义），不是 handler 契约验收，
	// 故不适用「必须走 db.NewDatabase + AutoMigrate 全量建表」那条约束。
	for _, stmt := range migrate226Statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec schema: %v", err)
		}
	}
	if err := db.AutoMigrate(&models.Menu{}, &models.Role{}, &models.RoleMenu{},
		&models.User{}, &models.UserRole{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

// seedConfigParent260 建父 C 型菜单「参数配置」（perms=system:config）。
func seedConfigParent260(t *testing.T, db *gorm.DB) *models.Menu {
	t.Helper()
	perms := "system:config"
	m := &models.Menu{
		MenuName: "参数配置",
		MenuType: models.MenuTypeMenu,
		Status:   models.MenuStatusNormal,
		Perms:    &perms,
	}
	if err := db.Create(m).Error; err != nil {
		t.Fatalf("seed 父菜单: %v", err)
	}
	// GORM 对 Visible 零值会按 default 省略，显式回写保持与生产一致
	if err := db.Model(m).Update("visible", models.VisibleShow).Error; err != nil {
		t.Fatalf("fix parent visible: %v", err)
	}
	return m
}

func TestMigrate260DeclarationConsistency(t *testing.T) {
	if len(configActionPermissionFTypeSeedCodes) != len(configActionPermissionSeeds) {
		t.Fatalf("声明块 %d != 种子表 %d",
			len(configActionPermissionFTypeSeedCodes), len(configActionPermissionSeeds))
	}
	for i, code := range configActionPermissionFTypeSeedCodes {
		if code != configActionPermissionSeeds[i].perms {
			t.Fatalf("第 %d 项漂移: 声明(%s) != 种子(%s)", i, code, configActionPermissionSeeds[i].perms)
		}
	}
}

func TestMigrate260SeedsFourButtonsUnderConfigParent(t *testing.T) {
	db := freshSQLiteDBForMigrate260(t)
	parent := seedConfigParent260(t, db)

	if err := Migrate260ConfigActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("run 260: %v", err)
	}

	codes := make([]string, 0, len(configActionPermissionSeeds))
	for _, s := range configActionPermissionSeeds {
		codes = append(codes, s.perms)
	}

	var rows []models.Menu
	if err := db.Where("menu_type = ? AND perms IN ?", models.MenuTypeButton, codes).
		Find(&rows).Error; err != nil {
		t.Fatalf("query buttons: %v", err)
	}
	if len(rows) != len(codes) {
		t.Fatalf("应播种 %d 个 F 行，实际 %d", len(codes), len(rows))
	}
	for _, row := range rows {
		if row.ParentID == nil || *row.ParentID != parent.ID {
			t.Fatalf("按钮 %s 的父菜单应为参数配置(%s)，实际 %v",
				*row.Perms, parent.ID, row.ParentID)
		}
		if row.Status != models.MenuStatusNormal {
			t.Fatalf("按钮 %s 状态应为正常，实际 %v", *row.Perms, row.Status)
		}
	}
}

func TestMigrate260GrantsOnlyToParentHolders(t *testing.T) {
	db := freshSQLiteDBForMigrate260(t)
	parent := seedConfigParent260(t, db)

	// holder：已持有父菜单（模拟超级管理员/参数配置管理员）
	holder := &models.Role{RoleName: "配置管理员", RoleKey: "cfg_admin", Status: models.RoleStatusEnabled}
	// other：完全不相关角色
	other := &models.Role{RoleName: "运维专员", RoleKey: "ops_user", Status: models.RoleStatusEnabled}
	for _, r := range []*models.Role{holder, other} {
		if err := db.Create(r).Error; err != nil {
			t.Fatalf("seed role: %v", err)
		}
	}
	if err := db.Create(&models.RoleMenu{RoleID: holder.ID, MenuID: parent.ID}).Error; err != nil {
		t.Fatalf("grant parent to holder: %v", err)
	}

	if err := Migrate260ConfigActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("run 260: %v", err)
	}

	var buttons []models.Menu
	if err := db.Where("menu_type = ? AND parent_id = ?", models.MenuTypeButton, parent.ID).
		Find(&buttons).Error; err != nil {
		t.Fatalf("query buttons: %v", err)
	}
	if len(buttons) == 0 {
		t.Fatal("未播种任何按钮行")
	}

	// holder 应被授予全部 4 个按钮
	var holderGrants int64
	if err := db.Model(&models.RoleMenu{}).Where("role_id = ?", holder.ID).Count(&holderGrants).Error; err != nil {
		t.Fatalf("count holder grants: %v", err)
	}
	// 1（父菜单）+ 4（四个按钮）
	if holderGrants != int64(1+len(buttons)) {
		t.Fatalf("holder 应有 %d 条授权（父 1 + 按钮 %d），实际 %d",
			1+len(buttons), len(buttons), holderGrants)
	}

	// other 不应被授予任何按钮
	var otherGrants int64
	if err := db.Model(&models.RoleMenu{}).Where("role_id = ?", other.ID).Count(&otherGrants).Error; err != nil {
		t.Fatalf("count other grants: %v", err)
	}
	if otherGrants != 0 {
		t.Fatalf("无关角色不应被授予按钮权限，实际 %d 条", otherGrants)
	}
}

func TestMigrate260IsIdempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate260(t)
	parent := seedConfigParent260(t, db)

	holder := &models.Role{RoleName: "配置管理员", RoleKey: "cfg_admin", Status: models.RoleStatusEnabled}
	if err := db.Create(holder).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&models.RoleMenu{RoleID: holder.ID, MenuID: parent.ID}).Error; err != nil {
		t.Fatalf("grant parent: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := Migrate260ConfigActionPermissionFTypeSeed(db); err != nil {
			t.Fatalf("第 %d 次执行失败: %v", i+1, err)
		}
	}

	// 按钮行数不得增长
	var buttonCount int64
	if err := db.Model(&models.Menu{}).Where("menu_type = ?", models.MenuTypeButton).Count(&buttonCount).Error; err != nil {
		t.Fatalf("count buttons: %v", err)
	}
	if buttonCount != int64(len(configActionPermissionSeeds)) {
		t.Fatalf("重复执行后按钮数应恒为 %d，实际 %d", len(configActionPermissionSeeds), buttonCount)
	}

	// 授权行数不得增长
	var grantCount int64
	if err := db.Model(&models.RoleMenu{}).Where("role_id = ?", holder.ID).Count(&grantCount).Error; err != nil {
		t.Fatalf("count grants: %v", err)
	}
	if grantCount != int64(1+len(configActionPermissionSeeds)) {
		t.Fatalf("重复执行后授权数应恒为 %d，实际 %d", 1+len(configActionPermissionSeeds), grantCount)
	}
}

func TestMigrate260SkipsWhenParentMissing(t *testing.T) {
	db := freshSQLiteDBForMigrate260(t)
	// 故意不建父菜单

	if err := Migrate260ConfigActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("父菜单缺失时不得报错（非阻断约定），实际: %v", err)
	}

	var buttonCount int64
	if err := db.Model(&models.Menu{}).Where("menu_type = ?", models.MenuTypeButton).Count(&buttonCount).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if buttonCount != 0 {
		t.Fatalf("父菜单缺失时不应建按钮行，实际 %d", buttonCount)
	}
}

// TestMigrate260PermissionsVisibleToMenuService 端到端：播种后按 menuService
// GetUserPermissions 的口径（sys_menu.perms, status=正常）能取到四个码。
// 这是「超管提示无 system:config:edit 权限」的直接回归护栏。
func TestMigrate260PermissionsVisibleToMenuService(t *testing.T) {
	db := freshSQLiteDBForMigrate260(t)
	parent := seedConfigParent260(t, db)

	holder := &models.Role{RoleName: "超级管理员", RoleKey: "admin", Status: models.RoleStatusEnabled}
	user := &models.User{ /* 仅需 ID 关联 */ }
	if err := db.Create(holder).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.UserRole{UserID: user.ID, RoleID: holder.ID}).Error; err != nil {
		t.Fatalf("bind user role: %v", err)
	}
	if err := db.Create(&models.RoleMenu{RoleID: holder.ID, MenuID: parent.ID}).Error; err != nil {
		t.Fatalf("grant parent: %v", err)
	}

	if err := Migrate260ConfigActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("run 260: %v", err)
	}

	// 复刻 menuService.GetUserPermissions 的取码 SQL 形状
	var roleIDs []string
	if err := db.Model(&models.UserRole{}).Where("user_id = ?", user.ID).Pluck("role_id", &roleIDs).Error; err != nil {
		t.Fatalf("pluck role ids: %v", err)
	}
	var menuIDs []string
	if err := db.Model(&models.RoleMenu{}).Distinct("menu_id").Where("role_id IN ?", roleIDs).
		Pluck("menu_id", &menuIDs).Error; err != nil {
		t.Fatalf("pluck menu ids: %v", err)
	}
	var perms []string
	if err := db.Model(&models.Menu{}).
		Where("id IN ? AND perms IS NOT NULL AND perms != '' AND status = ?", menuIDs, models.MenuStatusNormal).
		Pluck("perms", &perms).Error; err != nil {
		t.Fatalf("pluck perms: %v", err)
	}

	found := make(map[string]bool, len(perms))
	for _, p := range perms {
		found[p] = true
	}
	for _, seed := range configActionPermissionSeeds {
		if !found[seed.perms] {
			t.Fatalf("播种后用户权限集仍缺 %s（该码永远不会被授予）—— 实测集合: %v", seed.perms, perms)
		}
	}
}
