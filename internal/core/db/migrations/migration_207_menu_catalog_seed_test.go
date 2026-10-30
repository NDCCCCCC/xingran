package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate207 构建内存 SQLite 库 (含菜单域最小表集)。
func freshSQLiteDBForMigrate207(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Menu{},
		&models.Role{},
		&models.RoleMenu{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// aliveMenuCount 统计存活菜单数。
func aliveMenuCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&models.Menu{}).Count(&count).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	return count
}

// TestMigrate207FreshDBSeedsFullCatalog 全新库: 种子 239 条规范目录, 顶级根齐全,
// 无悬空 parent_id, 重复执行幂等。
func TestMigrate207FreshDBSeedsFullCatalog(t *testing.T) {
	db := freshSQLiteDBForMigrate207(t)

	if err := Migrate207SeedCanonicalMenuCatalog(db); err != nil {
		t.Fatalf("Migrate207 first run: %v", err)
	}

	if got := aliveMenuCount(t, db); got != 239 {
		t.Fatalf("alive menu count = %d, want 239", got)
	}

	// 规范顶级根: 系统管理 + 运维管理
	for _, id := range []string{canonicalSystemRootID, canonicalOpsRootID} {
		var c int64
		if err := db.Model(&models.Menu{}).Where("id = ?", id).Count(&c).Error; err != nil {
			t.Fatalf("query root %s: %v", id, err)
		}
		if c != 1 {
			t.Fatalf("canonical root %s missing", id)
		}
	}

	// 顶级目录族齐全 (生产事实: 9 M + 1 C 仪表盘)
	var topCount int64
	if err := db.Model(&models.Menu{}).Where("parent_id IS NULL").Count(&topCount).Error; err != nil {
		t.Fatalf("count top-level: %v", err)
	}
	if topCount != 10 {
		t.Fatalf("top-level menu count = %d, want 10", topCount)
	}

	// 无悬空 parent_id (所有非空 parent_id 必须指向存活菜单)
	var dangling int64
	if err := db.Model(&models.Menu{}).
		Where("parent_id IS NOT NULL AND parent_id NOT IN (SELECT id FROM sys_menu WHERE deleted_at IS NULL)").
		Count(&dangling).Error; err != nil {
		t.Fatalf("dangling check: %v", err)
	}
	if dangling != 0 {
		t.Fatalf("dangling parent_id count = %d, want 0", dangling)
	}

	// 幂等: 第二次运行不增不减
	if err := Migrate207SeedCanonicalMenuCatalog(db); err != nil {
		t.Fatalf("Migrate207 second run: %v", err)
	}
	if got := aliveMenuCount(t, db); got != 239 {
		t.Fatalf("after re-run alive menu count = %d, want 239", got)
	}
}

// TestMigrate207RespectsUserDeletion 用户在 UI 删除的菜单不被种子复活
// (快速路径: 规范根存在即整体跳过)。
func TestMigrate207RespectsUserDeletion(t *testing.T) {
	db := freshSQLiteDBForMigrate207(t)

	if err := Migrate207SeedCanonicalMenuCatalog(db); err != nil {
		t.Fatalf("Migrate207: %v", err)
	}

	// 模拟用户删除一个菜单 (软删)
	var victim models.Menu
	if err := db.Where("menu_name = ?", "角色管理").First(&victim).Error; err != nil {
		t.Fatalf("find victim: %v", err)
	}
	if err := db.Delete(&victim).Error; err != nil {
		t.Fatalf("soft-delete victim: %v", err)
	}

	if err := Migrate207SeedCanonicalMenuCatalog(db); err != nil {
		t.Fatalf("Migrate207 re-run: %v", err)
	}

	var c int64
	if err := db.Model(&models.Menu{}).Where("id = ?", victim.ID).Count(&c).Error; err != nil {
		t.Fatalf("count victim: %v", err)
	}
	if c != 0 {
		t.Fatalf("user-deleted menu %s was resurrected by seed", victim.ID)
	}
	if got := aliveMenuCount(t, db); got != 238 {
		t.Fatalf("alive menu count = %d, want 238", got)
	}
}

// TestMigrate207ReconcilesLegacyOpsTree 已有 Go 种子遗留「运维管理」子树的库:
// 迁移软删遗留子树 + 清理其授权, 规范运维管理根就位, 无双份顶级目录。
func TestMigrate207ReconcilesLegacyOpsTree(t *testing.T) {
	db := freshSQLiteDBForMigrate207(t)

	// 模拟 init_data.go createOperationsManagementMenus 的遗留产物 (随机 UUID)
	legacyRoot := models.Menu{
		MenuName: "运维管理",
		MenuType: models.MenuTypeDir,
		Status:   models.MenuStatusNormal,
		Visible:  models.VisibleShow,
	}
	legacyRoot.ID = uuid.NewString()
	if err := db.Create(&legacyRoot).Error; err != nil {
		t.Fatalf("create legacy root: %v", err)
	}
	legacyPage := models.Menu{
		MenuName: "楼宇管理",
		ParentID: &legacyRoot.ID,
		MenuType: models.MenuTypeMenu,
		Status:   models.MenuStatusNormal,
		Visible:  models.VisibleShow,
	}
	legacyPage.ID = uuid.NewString()
	if err := db.Create(&legacyPage).Error; err != nil {
		t.Fatalf("create legacy page: %v", err)
	}
	legacyButton := models.Menu{
		MenuName: "楼宇查询",
		ParentID: &legacyPage.ID,
		MenuType: models.MenuTypeButton,
		Status:   models.MenuStatusNormal,
		Visible:  models.VisibleShow,
	}
	legacyButton.ID = uuid.NewString()
	if err := db.Create(&legacyButton).Error; err != nil {
		t.Fatalf("create legacy button: %v", err)
	}

	// admin 角色持有遗留子树授权
	adminRole := models.Role{RoleName: "超级管理员", RoleKey: "admin", Status: models.RoleStatusEnabled}
	adminRole.ID = uuid.NewString()
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	for _, mid := range []string{legacyRoot.ID, legacyPage.ID, legacyButton.ID} {
		if err := db.Create(&models.RoleMenu{RoleID: adminRole.ID, MenuID: mid}).Error; err != nil {
			t.Fatalf("grant legacy menu: %v", err)
		}
	}

	if err := Migrate207SeedCanonicalMenuCatalog(db); err != nil {
		t.Fatalf("Migrate207: %v", err)
	}

	// 遗留子树整支软删
	for _, mid := range []string{legacyRoot.ID, legacyPage.ID, legacyButton.ID} {
		var c int64
		if err := db.Unscoped().Model(&models.Menu{}).
			Where("id = ? AND deleted_at IS NOT NULL", mid).Count(&c).Error; err != nil {
			t.Fatalf("check soft-deleted %s: %v", mid, err)
		}
		if c != 1 {
			t.Fatalf("legacy menu %s not soft-deleted", mid)
		}
	}

	// 遗留授权已清理
	var grantCount int64
	if err := db.Model(&models.RoleMenu{}).
		Where("menu_id IN ?", []string{legacyRoot.ID, legacyPage.ID, legacyButton.ID}).
		Count(&grantCount).Error; err != nil {
		t.Fatalf("count legacy grants: %v", err)
	}
	if grantCount != 0 {
		t.Fatalf("legacy role_menu grants = %d, want 0", grantCount)
	}

	// 存活「运维管理」顶级目录恰好一个 (规范根)
	var opsRoots int64
	if err := db.Model(&models.Menu{}).
		Where("menu_name = ? AND menu_type = ? AND parent_id IS NULL", "运维管理", "M").
		Count(&opsRoots).Error; err != nil {
		t.Fatalf("count ops roots: %v", err)
	}
	if opsRoots != 1 {
		t.Fatalf("alive 运维管理 roots = %d, want 1", opsRoots)
	}

	// 规范目录全量就位: 239 存活 (遗留 3 条已软删不计)
	if got := aliveMenuCount(t, db); got != 239 {
		t.Fatalf("alive menu count = %d, want 239", got)
	}

	// 幂等重跑
	if err := Migrate207SeedCanonicalMenuCatalog(db); err != nil {
		t.Fatalf("Migrate207 re-run: %v", err)
	}
	if got := aliveMenuCount(t, db); got != 239 {
		t.Fatalf("after re-run alive menu count = %d, want 239", got)
	}
}

// TestMigrate207GrantPathIntegration 端到端断言: 种子后 assignAllMenusToAdmin
// 等价物 (差集授权) 能授予 admin 全部 239 条 —— 验证"超管始终拥有全部权限"闭环。
// (真正的 assignAllMenusToAdmin 在 pkg/permission, 此处用同语义 SQL 避免包循环依赖。)
func TestMigrate207GrantPathIntegration(t *testing.T) {
	db := freshSQLiteDBForMigrate207(t)

	adminRole := models.Role{RoleName: "超级管理员", RoleKey: "admin", Status: models.RoleStatusEnabled}
	adminRole.ID = uuid.NewString()
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}

	if err := Migrate207SeedCanonicalMenuCatalog(db); err != nil {
		t.Fatalf("Migrate207: %v", err)
	}

	// assignAllMenusToAdmin 同语义: 差集 INSERT
	var allIDs []string
	if err := db.Model(&models.Menu{}).Pluck("id", &allIDs).Error; err != nil {
		t.Fatalf("pluck menu ids: %v", err)
	}
	for _, mid := range allIDs {
		if err := db.Create(&models.RoleMenu{RoleID: adminRole.ID, MenuID: mid}).Error; err != nil {
			t.Fatalf("grant %s: %v", mid, err)
		}
	}

	// admin 菜单数 == 存活菜单数 == 239 (含按钮; 读取链路的 M/C 过滤在前端树构建层)
	var granted int64
	if err := db.Model(&models.RoleMenu{}).Where("role_id = ?", adminRole.ID).Count(&granted).Error; err != nil {
		t.Fatalf("count grants: %v", err)
	}
	if granted != 239 {
		t.Fatalf("admin grants = %d, want 239", granted)
	}

	// 读取链路 (menuService.GetUserMenus 语义): 授权 ∩ 存活 ∩ 正常 ∩ 可见
	var visible int64
	if err := db.Model(&models.Menu{}).
		Where("id IN (SELECT menu_id FROM sys_role_menu WHERE role_id = ?) AND status = ? AND visible = ?",
			adminRole.ID, int(models.MenuStatusNormal), int(models.VisibleShow)).
		Count(&visible).Error; err != nil {
		t.Fatalf("count visible granted: %v", err)
	}
	if visible == 0 {
		t.Fatalf("admin visible granted menus = 0, menu endpoint would return empty tree")
	}
}
