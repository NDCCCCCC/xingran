package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 87 批次二（OQ-6）补播迁移 227 测试
// ============================================================================

func freshSQLiteDBForMigrate227(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for i, stmt := range migrate226Statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("建表 %d 失败: %v", i, err)
		}
	}
	return db
}

func seedFloorParentAndRoles(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		// 楼层管理父菜单（live 与 stopped）
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-floor-live', '楼层管理', 'ops:floor:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-floor-stop', '楼层管理', 'ops:floor:list', 'C', 1, 1)`,

		// 测试角色
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-floor-live', '持楼层角色', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-floor-stop', '持停用角色', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-none', '无权角色', 0)`,

		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-floor-live', 'p-ops-floor-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-floor-stop', 'p-ops-floor-stop')`,
	}
	execMigrate226(t, db, stmts)
}

func TestMigrate227_SeedsAllHoldCodes(t *testing.T) {
	db := freshSQLiteDBForMigrate227(t)
	seedFloorParentAndRoles(t, db)

	if err := Migrate227ActionPermissionFTypeSeedHold(db); err != nil {
		t.Fatalf("Migrate227 失败: %v", err)
	}

	totalButtons := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu WHERE menu_type = 'F'")
	if totalButtons != int64(len(actionPermissionSeedsHold)) {
		t.Fatalf("期望创建 %d 条 F 按钮, 实际创建=%d", len(actionPermissionSeedsHold), totalButtons)
	}

	for _, seed := range actionPermissionSeedsHold {
		var row struct {
			ID       string
			ParentID string
			Status   int
			Visible  int
		}
		err := db.Raw("SELECT id, parent_id, status, visible FROM sys_menu WHERE perms = ? AND menu_type = 'F'",
			seed.perms).Scan(&row).Error
		if err != nil {
			t.Fatalf("查询码 %s 失败: %v", seed.perms, err)
		}
		if row.ID == "" {
			t.Errorf("码 %s 未被创建", seed.perms)
		}
		if row.ParentID != "p-ops-floor-live" {
			t.Errorf("码 %s parent_id 错误: 实际=%s", seed.perms, row.ParentID)
		}
		if row.Visible != 0 {
			t.Errorf("码 %s visible 必须为 0, 实际=%d", seed.perms, row.Visible)
		}
	}
}

func TestMigrate227_Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate227(t)
	seedFloorParentAndRoles(t, db)

	if err := Migrate227ActionPermissionFTypeSeedHold(db); err != nil {
		t.Fatalf("第 1 次执行失败: %v", err)
	}
	menuCount1 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")
	roleMenuCount1 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_role_menu")

	if err := Migrate227ActionPermissionFTypeSeedHold(db); err != nil {
		t.Fatalf("第 2 次执行失败: %v", err)
	}
	menuCount2 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")
	roleMenuCount2 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_role_menu")

	if menuCount1 != menuCount2 || roleMenuCount1 != roleMenuCount2 {
		t.Errorf("破坏幂等性: 菜单(%d->%d), 授权(%d->%d)", menuCount1, menuCount2, roleMenuCount1, roleMenuCount2)
	}
}

func TestMigrate227_GrantsOnlyParentHolders(t *testing.T) {
	db := freshSQLiteDBForMigrate227(t)
	seedFloorParentAndRoles(t, db)

	if err := Migrate227ActionPermissionFTypeSeedHold(db); err != nil {
		t.Fatalf("Migrate227 失败: %v", err)
	}

	// 持 live 楼层角色的应获得 7 条
	liveGranted := countMigrate226Rows(t, db,
		`SELECT count(*) FROM sys_role_menu rm
		 JOIN sys_menu m ON rm.menu_id = m.id
		 WHERE rm.role_id = 'r-floor-live' AND m.menu_type = 'F'`)
	if liveGranted != 7 {
		t.Errorf("期望 r-floor-live 获 7 条授权, 实际=%d", liveGranted)
	}

	// 持 stop 的不能获得
	stopGranted := countMigrate226Rows(t, db,
		`SELECT count(*) FROM sys_role_menu rm
		 JOIN sys_menu m ON rm.menu_id = m.id
		 WHERE rm.role_id = 'r-floor-stop' AND m.menu_type = 'F'`)
	if stopGranted != 0 {
		t.Errorf("越权：r-floor-stop 获得了 %d 条授权", stopGranted)
	}

	// 无权角色不能获得
	noneGranted := countMigrate226Rows(t, db,
		`SELECT count(*) FROM sys_role_menu rm
		 JOIN sys_menu m ON rm.menu_id = m.id
		 WHERE rm.role_id = 'r-none' AND m.menu_type = 'F'`)
	if noneGranted != 0 {
		t.Errorf("越权：r-none 获得了 %d 条授权", noneGranted)
	}
}
