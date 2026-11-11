package migrations

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate241 构建内存 SQLite 库（仅 sys_menu 单表最小表集，
// 与 migration_209_update_settings_menu_component_test.go 同款建库方式）。
func freshSQLiteDBForMigrate241(t *testing.T, name string) *gorm.DB {
	t.Helper()
	safe := strings.NewReplacer("/", "_", " ", "_", "=", "_").Replace(name)
	dsn := "file:" + safe + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Menu{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// seedMenu241 写入一条菜单（固定 ID 便于断言），返回其指针。
func seedMenu241(t *testing.T, db *gorm.DB, id, name, path, component, perms string,
	menuType models.MenuType, parentID *string, visible models.VisibleType) *models.Menu {
	t.Helper()
	menu := &models.Menu{
		MenuName: name,
		ParentID: parentID,
		MenuType: menuType,
		Visible:  visible,
		Status:   models.MenuStatusNormal,
	}
	menu.ID = id
	if path != "" {
		p := path
		menu.Path = &p
	}
	if component != "" {
		c := component
		menu.Component = &c
	}
	if perms != "" {
		p := perms
		menu.Perms = &p
	}
	if err := db.Create(menu).Error; err != nil {
		t.Fatalf("seed menu %s: %v", name, err)
	}
	return menu
}

// loadMenu241 按 ID 读取菜单，不存在直接失败。
func loadMenu241(t *testing.T, db *gorm.DB, id string) models.Menu {
	t.Helper()
	var menu models.Menu
	if err := db.Where("id = ?", id).First(&menu).Error; err != nil {
		t.Fatalf("load menu %s: %v", id, err)
	}
	return menu
}

func assertVisible241(t *testing.T, db *gorm.DB, id string, want models.VisibleType) {
	t.Helper()
	if got := loadMenu241(t, db, id).Visible; got != want {
		t.Fatalf("menu %s visible = %d, want %d", id, got, want)
	}
}

func assertParent241(t *testing.T, db *gorm.DB, id, want string) {
	t.Helper()
	menu := loadMenu241(t, db, id)
	if menu.ParentID == nil {
		t.Fatalf("menu %s parent_id is nil, want %s", id, want)
	}
	if *menu.ParentID != want {
		t.Fatalf("menu %s parent_id = %s, want %s", id, *menu.ParentID, want)
	}
}

// 测试用固定菜单 ID（manager 两个 C 菜单在 migration_221 里是运行期生成的 UUID，
// 存量库具体值不固定，因此此处也仅用于本测试内定位）。
const (
	test241ManagerRulesMenuID = "b0000000-0000-4000-8000-000000002411"
	test241ManagerOrgsMenuID  = "b0000000-0000-4000-8000-000000002412"
	test241BackupListBtnID    = "b0000000-0000-4000-8000-000000002421"
	test241BackupExportBtnID  = "b0000000-0000-4000-8000-000000002422"
	test241BackupRestoreBtnID = "b0000000-0000-4000-8000-000000002423"
	test241ManagerRuleEditID  = "b0000000-0000-4000-8000-000000002431"
	test241ManagerRuleDelID   = "b0000000-0000-4000-8000-000000002432"
	test241ManagerOrgEditID   = "b0000000-0000-4000-8000-000000002433"
	test241ManagerOrgDelID    = "b0000000-0000-4000-8000-000000002434"
)

// seedConsolidateFixture241 构造「合并前」的完整存量库快照：
// 系统设置 + 2 个经理同步 C 菜单 + 备份 C 菜单 + manager 4 个 F（edit/delete ×2）+ backup 3 个 F。
func seedConsolidateFixture241(t *testing.T, db *gorm.DB) {
	t.Helper()

	// 系统管理根（仅用于让 parent 结构真实）
	rootID := canonicalSystemRootID
	seedMenu241(t, db, rootID, "系统管理", "", "", "", models.MenuTypeDir, nil, models.VisibleShow)

	// 合并目标：系统设置 C 菜单（种子固定 id）
	seedMenu241(t, db, settingsMenuID, "系统设置", "settings-page", settingsMenuNewComponent,
		"system:config:list", models.MenuTypeMenu, &rootID, models.VisibleShow)

	// 待隐藏的 3 个旧 C 菜单
	seedMenu241(t, db, test241ManagerRulesMenuID, "经理同步规则", "manager-rules",
		"system/manager-rules/index", managerRuleListPerms, models.MenuTypeMenu, &rootID, models.VisibleShow)
	seedMenu241(t, db, test241ManagerOrgsMenuID, "经理同步机构", "manager-orgs",
		"system/manager-orgs/index", managerOrgListPerms, models.MenuTypeMenu, &rootID, models.VisibleShow)
	seedMenu241(t, db, settingsConsolidateBackupMenuID, "数据备份与恢复", "backup",
		"system/backup/index", "system:backup", models.MenuTypeMenu, &rootID, models.VisibleShow)

	// manager 4 个 F 按钮：改挂前分别挂在各自 C 菜单下
	rulesID, orgsID := test241ManagerRulesMenuID, test241ManagerOrgsMenuID
	seedMenu241(t, db, test241ManagerRuleEditID, "规则新增/编辑", "", "", managerRuleEditPerms,
		models.MenuTypeButton, &rulesID, models.VisibleHidden)
	seedMenu241(t, db, test241ManagerRuleDelID, "规则删除", "", "", managerRuleDeletePerms,
		models.MenuTypeButton, &rulesID, models.VisibleHidden)
	seedMenu241(t, db, test241ManagerOrgEditID, "机构新增/编辑", "", "", managerOrgEditPerms,
		models.MenuTypeButton, &orgsID, models.VisibleHidden)
	seedMenu241(t, db, test241ManagerOrgDelID, "机构删除", "", "", managerOrgDeletePerms,
		models.MenuTypeButton, &orgsID, models.VisibleHidden)

	// backup 3 个 F 按钮：改挂前挂在备份 C 菜单下
	backupID := settingsConsolidateBackupMenuID
	seedMenu241(t, db, test241BackupListBtnID, "备份列表查询", "", "", "system:backup:list",
		models.MenuTypeButton, &backupID, models.VisibleHidden)
	seedMenu241(t, db, test241BackupExportBtnID, "备份导出下载", "", "", "system:backup:export",
		models.MenuTypeButton, &backupID, models.VisibleHidden)
	seedMenu241(t, db, test241BackupRestoreBtnID, "备份恢复", "", "", "system:backup:restore",
		models.MenuTypeButton, &backupID, models.VisibleHidden)
}

// TestMigrate241ConsolidatesMenusAndIsIdempotent 主路径：三个旧 C 菜单隐藏、
// 7 个 F 按钮改挂到系统设置、权限码不变、重复执行幂等。
func TestMigrate241ConsolidatesMenusAndIsIdempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())
	seedConsolidateFixture241(t, db)

	if err := Migrate241SettingsConsolidateMenus(db); err != nil {
		t.Fatalf("Migrate241 first run: %v", err)
	}

	// 1) 三个旧 C 菜单已隐藏
	for _, id := range []string{
		test241ManagerRulesMenuID,
		test241ManagerOrgsMenuID,
		settingsConsolidateBackupMenuID,
	} {
		assertVisible241(t, db, id, models.VisibleHidden)
	}
	// 系统设置本身必须保持可见（唯一入口不能被误伤）
	assertVisible241(t, db, settingsMenuID, models.VisibleShow)

	// 2) 7 个 F 按钮全部改挂到系统设置
	for _, id := range []string{
		test241ManagerRuleEditID, test241ManagerRuleDelID,
		test241ManagerOrgEditID, test241ManagerOrgDelID,
		test241BackupListBtnID, test241BackupExportBtnID, test241BackupRestoreBtnID,
	} {
		assertParent241(t, db, id, settingsMenuID)
	}

	// 3) 权限码原值不变（只改 parent，不改 perms）
	for id, wantPerms := range map[string]string{
		test241ManagerRuleEditID:  managerRuleEditPerms,
		test241ManagerOrgDelID:    managerOrgDeletePerms,
		test241BackupRestoreBtnID: "system:backup:restore",
	} {
		menu := loadMenu241(t, db, id)
		if menu.Perms == nil || *menu.Perms != wantPerms {
			t.Fatalf("menu %s perms = %v, want %s", id, menu.Perms, wantPerms)
		}
	}

	// 4) 重复执行：无错误、状态不变（幂等）
	if err := Migrate241SettingsConsolidateMenus(db); err != nil {
		t.Fatalf("Migrate241 second run: %v", err)
	}
	for _, id := range []string{
		test241ManagerRulesMenuID,
		test241ManagerOrgsMenuID,
		settingsConsolidateBackupMenuID,
	} {
		assertVisible241(t, db, id, models.VisibleHidden)
	}
	for _, id := range []string{
		test241ManagerRuleEditID, test241ManagerOrgEditID,
		test241BackupListBtnID, test241BackupRestoreBtnID,
	} {
		assertParent241(t, db, id, settingsMenuID)
	}
}

// TestMigrate241MissingSettingsMenuSkips 系统设置菜单缺失（瘦库/自定义菜单）：
// 返回 nil 不报错，且不做只隐藏不改挂的半完成变更（旧入口保持可用）。
func TestMigrate241MissingSettingsMenuSkips(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())

	// 只有两个旧 C 菜单 + 备份 + F 按钮，没有「系统设置」
	rootID := canonicalSystemRootID
	seedMenu241(t, db, test241ManagerRulesMenuID, "经理同步规则", "manager-rules",
		"system/manager-rules/index", managerRuleListPerms, models.MenuTypeMenu, &rootID, models.VisibleShow)
	seedMenu241(t, db, settingsConsolidateBackupMenuID, "数据备份与恢复", "backup",
		"system/backup/index", "system:backup", models.MenuTypeMenu, &rootID, models.VisibleShow)
	rulesID := test241ManagerRulesMenuID
	seedMenu241(t, db, test241ManagerRuleEditID, "规则新增/编辑", "", "", managerRuleEditPerms,
		models.MenuTypeButton, &rulesID, models.VisibleHidden)

	if err := Migrate241SettingsConsolidateMenus(db); err != nil {
		t.Fatalf("Migrate241 on missing settings menu: %v", err)
	}

	// 降级跳过：旧入口未被隐藏，按钮父节点未被改动
	assertVisible241(t, db, test241ManagerRulesMenuID, models.VisibleShow)
	assertVisible241(t, db, settingsConsolidateBackupMenuID, models.VisibleShow)
	assertParent241(t, db, test241ManagerRuleEditID, test241ManagerRulesMenuID)

	// 空库同样返回 nil
	emptyDB := freshSQLiteDBForMigrate241(t, t.Name()+"_empty")
	if err := Migrate241SettingsConsolidateMenus(emptyDB); err != nil {
		t.Fatalf("Migrate241 on empty db: %v", err)
	}
}

// TestMigrate241FallbackLocators 名称漂移的存量库：系统设置按 component 兜底定位，
// 经理同步两个菜单按 path 兜底隐藏。
func TestMigrate241FallbackLocators(t *testing.T) {
	db := freshSQLiteDBForMigrate241(t, t.Name())

	rootID := canonicalSystemRootID
	// 系统设置被改名为「系统偏好」，仅 component 仍是规范值
	seedMenu241(t, db, settingsMenuID, "系统偏好", "settings-page", settingsMenuNewComponent,
		"system:config:list", models.MenuTypeMenu, &rootID, models.VisibleShow)

	// 两个经理同步菜单被改名，仅 path 保留
	seedMenu241(t, db, test241ManagerRulesMenuID, "同步规则（旧）", "manager-rules",
		"system/manager-rules/index", managerRuleListPerms, models.MenuTypeMenu, &rootID, models.VisibleShow)
	seedMenu241(t, db, test241ManagerOrgsMenuID, "同步机构（旧）", "manager-orgs",
		"system/manager-orgs/index", managerOrgListPerms, models.MenuTypeMenu, &rootID, models.VisibleShow)
	// 备份菜单改名且 id 非固定值 → 走 menu_name 兜底
	seedMenu241(t, db, "b0000000-0000-4000-8000-000000002499", "数据备份与恢复", "backup",
		"system/backup/index", "system:backup", models.MenuTypeMenu, &rootID, models.VisibleShow)

	rulesID := test241ManagerRulesMenuID
	seedMenu241(t, db, test241ManagerRuleEditID, "规则新增/编辑", "", "", managerRuleEditPerms,
		models.MenuTypeButton, &rulesID, models.VisibleHidden)

	if err := Migrate241SettingsConsolidateMenus(db); err != nil {
		t.Fatalf("Migrate241 fallback run: %v", err)
	}

	assertVisible241(t, db, test241ManagerRulesMenuID, models.VisibleHidden)
	assertVisible241(t, db, test241ManagerOrgsMenuID, models.VisibleHidden)
	assertVisible241(t, db, "b0000000-0000-4000-8000-000000002499", models.VisibleHidden)
	assertParent241(t, db, test241ManagerRuleEditID, settingsMenuID)
}
