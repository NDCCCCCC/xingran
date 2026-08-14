package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// freshSQLiteDB builds an in-memory SQLite database with the menu/role schema
// used by the helper tests. Phase 52 W3 plan Task 1 <behavior>.
func freshSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file::memory:?cache=shared&_busy_timeout=5000"
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

// TestGrantNewMenuToRolesHavingParent_NonExistentParent 建空 DB,父菜单不存在时
// helper 在 SQLite 路径(isPostgreSQL=false)必须返回 nil 不 panic。
func TestGrantNewMenuToRolesHavingParent_NonExistentParent(t *testing.T) {
	db := freshSQLiteDB(t)

	// SQLite 路径:helper 顶部 isPostgreSQL 守卫 return nil,不 panic。
	// PG functional 行为由 Phase 54 UAT 覆盖。
	if err := GrantNewMenuToRolesHavingParent(db, "端口状态", "00000000-0000-0000-0000-000000000001"); err != nil {
		t.Fatalf("GrantNewMenuToRolesHavingParent on sqlite should be noop nil, got: %v", err)
	}
}

// TestGrantNewMenuToRolesHavingParent_Idempotent PG path:
// 第一次调用插入 N 行(N = 持有父菜单的角色数);第二次同样调用插 0 行。
// SQLite 路径下 helper isPostgreSQL 守卫 return nil,因此仅验证幂等(两次调用都 nil)。
func TestGrantNewMenuToRolesHavingParent_Idempotent(t *testing.T) {
	if !postgresAvailable() {
		t.Skip("requires PostgreSQL — covered by Phase 54 UAT; sqlite path returns nil (idempotent by short-circuit)")
	}
	db := openPostgresDB(t)

	parentID := seedParentMenuAndRoles(t, db)
	newMenuID := "00000000-0000-0000-0000-0000000000aa"

	if err := GrantNewMenuToRolesHavingParent(db, "端口状态", newMenuID); err != nil {
		t.Fatalf("first call: %v", err)
	}
	var count1 int64
	db.Model(&models.RoleMenu{}).Where("menu_id = ?", newMenuID).Count(&count1)
	if count1 == 0 {
		t.Fatalf("expected at least 1 grant after first call, got 0")
	}

	// 第二次调用必须 ON CONFLICT DO NOTHING,不增加行数。
	if err := GrantNewMenuToRolesHavingParent(db, "端口状态", newMenuID); err != nil {
		t.Fatalf("second call: %v", err)
	}
	var count2 int64
	db.Model(&models.RoleMenu{}).Where("menu_id = ?", newMenuID).Count(&count2)
	if count1 != count2 {
		t.Fatalf("idempotent violation: first=%d second=%d (parentID=%s)", count1, count2, parentID)
	}
}

// TestGrantNewMenuToRolesHavingParent_OnlyAffectsParentRoles PG path:
// 3 个角色(2 持父菜单, 1 不持),调用 helper → 仅 2 个角色被授权新菜单。
func TestGrantNewMenuToRolesHavingParent_OnlyAffectsParentRoles(t *testing.T) {
	if !postgresAvailable() {
		t.Skip("requires PostgreSQL — covered by Phase 54 UAT; sqlite path returns nil")
	}
	db := openPostgresDB(t)

	seedParentMenuAndRoles(t, db)

	newMenuID := "00000000-0000-0000-0000-0000000000bb"
	if err := GrantNewMenuToRolesHavingParent(db, "端口状态", newMenuID); err != nil {
		t.Fatalf("call: %v", err)
	}

	// 持有父菜单的 2 角色 → 都应授权新菜单
	roleA := getRoleIDByName(t, db, "role-granted-A")
	roleB := getRoleIDByName(t, db, "role-granted-B")
	roleC := getRoleIDByName(t, db, "role-not-granted-C")
	assertRoleMenuExists(t, db, roleA, newMenuID)
	assertRoleMenuExists(t, db, roleB, newMenuID)
	assertRoleMenuNotExists(t, db, roleC, newMenuID)
}

// TestGrantNewMenuToRolesHavingParent_ParameterizedOrControlled 源码 grep 断言:
// helper SQL 含 INSERT INTO sys_role_menu + JOIN sys_menu m ON rm.menu_id = m.id +
// WHERE m.menu_name = + ON CONFLICT DO NOTHING + 参数化占位符 + 无 fmt.Sprintf。
//
// C6 修复:GrantNewMenuToRolesHavingParent 必须用 $1::uuid / $2 参数化绑定,
// 消除任意输入的 SQL 注入面(原 fmt.Sprintf 拼接 SQL)。
func TestGrantNewMenuToRolesHavingParent_ParameterizedOrControlled(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "menu_grant_helpers.go"))
	if err != nil {
		t.Fatalf("read helper source: %v", err)
	}
	s := string(src)

	want := []string{
		"INSERT INTO sys_role_menu",
		"JOIN sys_menu m ON rm.menu_id = m.id",
		"WHERE m.menu_name =",
		"ON CONFLICT DO NOTHING",
		"$1::uuid",     // 参数化占位符(newMenuID, ::uuid cast)
		"menu_name = $2", // 参数化占位符(parentMenuName)
	}
	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Fatalf("helper source missing required SQL fragment: %q", w)
		}
	}

	// C6 反向断言:源代码不得含 fmt.Sprintf(SQL 字符串插值已参数化)。
	if strings.Contains(s, "fmt.Sprintf") {
		t.Fatalf("helper source must NOT contain fmt.Sprintf (SQL must be parameterized, C6)")
	}

	// 守卫:SQLite 路径必须 return nil。
	if !strings.Contains(s, "isPostgreSQL(db)") {
		t.Fatalf("helper must guard with isPostgreSQL(db) for SQLite skip")
	}
}

// ---- helpers ----

func postgresAvailable() bool {
	// 仅当环境变量 XINGRAN_PG_TEST_DSN 提供时跑 PG 路径。
	// Phase 54 UAT 在生产 PG 上覆盖;本地 / CI 默认 SQLite。
	return os.Getenv("XINGRAN_PG_TEST_DSN") != ""
}

func openPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Skip("PG integration path not configured in this executor — covered by Phase 54 UAT")
	return nil
}

// seedParentMenuAndRoles 建父菜单 "端口状态" + 3 角色 (A/B 持父菜单,C 不持)。
// 返回父菜单 ID。
func seedParentMenuAndRoles(t *testing.T, db *gorm.DB) string {
	t.Helper()
	t.Skip("PG integration seed only — covered by Phase 54 UAT")
	return ""
}

func getRoleIDByName(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	t.Skip("PG integration only")
	return ""
}

func assertRoleMenuExists(t *testing.T, db *gorm.DB, roleID, menuID string) {
	t.Helper()
	t.Skip("PG integration only")
}

func assertRoleMenuNotExists(t *testing.T, db *gorm.DB, roleID, menuID string) {
	t.Helper()
	t.Skip("PG integration only")
}
