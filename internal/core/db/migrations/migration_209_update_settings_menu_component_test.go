package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// freshSQLiteDBForMigrate209 构建内存 SQLite 库 (仅 sys_menu 单表最小表集)。
func freshSQLiteDBForMigrate209(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Menu{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// getComponent 读取指定菜单的 component 值 (不存在时 ok=false)。
func getComponent(t *testing.T, db *gorm.DB, id string) (string, bool) {
	t.Helper()
	var menu models.Menu
	if err := db.Where("id = ?", id).First(&menu).Error; err != nil {
		return "", false
	}
	if menu.Component == nil {
		return "", true
	}
	return *menu.Component, true
}

// TestMigrate209UpdatesOldComponentIdempotently 旧值就位: 迁移改写为新路径且
// changed=true; 重复执行值不变且 changed=false (幂等)。
func TestMigrate209UpdatesOldComponentIdempotently(t *testing.T) {
	db := freshSQLiteDBForMigrate209(t)

	oldComponent := settingsMenuOldComponent
	perms := "system:config:list"
	path := "settings-page"
	seed := models.Menu{
		MenuName:  "系统设置",
		Path:      &path,
		Component: &oldComponent,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
		Perms:     &perms,
	}
	seed.ID = settingsMenuID
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}

	// 第一次运行: 旧值 → 新值, changed=true
	changed, err := Migrate209UpdateSettingsMenuComponent(db)
	if err != nil {
		t.Fatalf("Migrate209 first run: %v", err)
	}
	if !changed {
		t.Fatalf("first run changed = false, want true (old component present)")
	}
	got, ok := getComponent(t, db, settingsMenuID)
	if !ok {
		t.Fatalf("menu %s missing after migration", settingsMenuID)
	}
	if got != settingsMenuNewComponent {
		t.Fatalf("component = %q, want %q", got, settingsMenuNewComponent)
	}

	// 迁移只改 component: path/perms 不被触碰 (安全边界 T-70-01/T-70-0601)
	var after models.Menu
	if err := db.Where("id = ?", settingsMenuID).First(&after).Error; err != nil {
		t.Fatalf("reload menu: %v", err)
	}
	if after.Path == nil || *after.Path != "settings-page" {
		t.Fatalf("path was modified: %v", after.Path)
	}
	if after.Perms == nil || *after.Perms != "system:config:list" {
		t.Fatalf("perms was modified: %v", after.Perms)
	}

	// 第二次运行: 旧值已不存在 → RowsAffected=0, 值不变, changed=false
	changed, err = Migrate209UpdateSettingsMenuComponent(db)
	if err != nil {
		t.Fatalf("Migrate209 second run: %v", err)
	}
	if changed {
		t.Fatalf("second run changed = true, want false (idempotency broken)")
	}
	got, ok = getComponent(t, db, settingsMenuID)
	if !ok || got != settingsMenuNewComponent {
		t.Fatalf("after re-run component = %q (ok=%v), want %q", got, ok, settingsMenuNewComponent)
	}
}

// TestMigrate209GuardedByDoubleCondition 双条件守护: 同 id 但 component 为其他值、
// 以及旧值挂在不同 id 上 —— 两者都必须不被误改, changed=false。
func TestMigrate209GuardedByDoubleCondition(t *testing.T) {
	db := freshSQLiteDBForMigrate209(t)

	// 场景 1: 目标 id 的 component 是无关值 (已手工改过的库) —— 不被覆盖
	otherComponent := "custom/other/path"
	target := models.Menu{
		MenuName:  "系统设置",
		Component: &otherComponent,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
	}
	target.ID = settingsMenuID
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("seed target menu: %v", err)
	}

	// 场景 2: 旧 component 值挂在另一个菜单 id 上 —— 不被误改
	strayOldComponent := settingsMenuOldComponent
	stray := models.Menu{
		MenuName:  "旧值占位菜单",
		Component: &strayOldComponent,
		MenuType:  models.MenuTypeMenu,
		Visible:   models.VisibleShow,
		Status:    models.MenuStatusNormal,
	}
	stray.ID = "11111111-2222-3333-4444-555555555555"
	if err := db.Create(&stray).Error; err != nil {
		t.Fatalf("seed stray menu: %v", err)
	}

	changed, err := Migrate209UpdateSettingsMenuComponent(db)
	if err != nil {
		t.Fatalf("Migrate209: %v", err)
	}
	if changed {
		t.Fatalf("changed = true, want false (no row matched id+oldValue double condition)")
	}

	if got, ok := getComponent(t, db, settingsMenuID); !ok || got != "custom/other/path" {
		t.Fatalf("target id component = %q (ok=%v), want custom/other/path untouched", got, ok)
	}
	if got, ok := getComponent(t, db, stray.ID); !ok || got != settingsMenuOldComponent {
		t.Fatalf("stray id component = %q (ok=%v), want old value untouched", got, ok)
	}

	// 空库 (无任何菜单): 同样 changed=false 不报错
	emptyDB := freshSQLiteDBForMigrate209(t)
	changed, err = Migrate209UpdateSettingsMenuComponent(emptyDB)
	if err != nil {
		t.Fatalf("Migrate209 on empty db: %v", err)
	}
	if changed {
		t.Fatalf("empty db changed = true, want false")
	}
}
