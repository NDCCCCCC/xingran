package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// Migration 247 测试 — 杂项 RBAC F 型按钮播种（task-4）
//
// 覆盖：声明块一致性 / 播种行数 / 幂等（复用 226 的 schema statements）。
// 防假绿约束：不硬编码码字面量，统一引用包内 miscRbacPermissionSeeds。
// ============================================================================

func freshSQLiteDBForMigrate247(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for _, stmt := range migrate226Statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec schema: %v", err)
		}
	}
	return db
}

func TestMigrate247DeclarationConsistency(t *testing.T) {
	if len(miscRbacFTypeSeedCodes) != len(miscRbacPermissionSeeds) {
		t.Fatalf("声明块 %d != 种子表 %d", len(miscRbacFTypeSeedCodes), len(miscRbacPermissionSeeds))
	}
	for i, code := range miscRbacFTypeSeedCodes {
		if code != miscRbacPermissionSeeds[i].perms {
			t.Fatalf("第 %d 项漂移: 声明(%s) != 种子(%s)", i, code, miscRbacPermissionSeeds[i].perms)
		}
	}
}

func TestMigrate247SeedsAndIsIdempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate247(t)
	// 父 C 型菜单：MAC地址（network:mac）+ 系统设置（system:config:list）
	parents := map[string]string{
		"network:mac":        "MAC地址",
		"system:config:list": "系统设置",
	}
	for perms, name := range parents {
		p := perms
		m := &models.Menu{MenuName: name, MenuType: models.MenuTypeMenu, Status: models.MenuStatusNormal, Perms: &p}
		if err := db.Create(m).Error; err != nil {
			t.Fatalf("seed parent %s: %v", perms, err)
		}
	}

	if err := Migrate247MiscRbacFTypeSeed(db); err != nil {
		t.Fatalf("run 247: %v", err)
	}
	var n int64
	if err := db.Model(&models.Menu{}).
		Where("menu_type = ?", models.MenuTypeButton).
		Where("perms IN ?", []string{"network:mac:query", "network:mac:collect", "network:mac:remove", "network:mac:rule:edit",
			"system:captchaBackground:list", "system:captchaBackground:view", "system:captchaBackground:add",
			"system:captchaBackground:edit", "system:captchaBackground:remove"}).
		Count(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if want := int64(len(miscRbacPermissionSeeds)); n != want {
		t.Fatalf("F menus = %d, want %d", n, want)
	}

	// 幂等
	if err := Migrate247MiscRbacFTypeSeed(db); err != nil {
		t.Fatalf("rerun 247: %v", err)
	}
	var n2 int64
	if err := db.Model(&models.Menu{}).
		Where("menu_type = ?", models.MenuTypeButton).
		Where("perms IN ?", []string{"network:mac:query", "network:mac:collect", "network:mac:remove", "network:mac:rule:edit",
			"system:captchaBackground:list", "system:captchaBackground:view", "system:captchaBackground:add",
			"system:captchaBackground:edit", "system:captchaBackground:remove"}).
		Count(&n2).Error; err != nil {
		t.Fatalf("count 2: %v", err)
	}
	if n2 != n {
		t.Fatalf("idempotency broken: first=%d second=%d", n, n2)
	}
}
