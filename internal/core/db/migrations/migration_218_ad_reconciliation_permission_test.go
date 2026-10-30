package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 59 / GUARD-02 迁移 218 测试
//
// 覆盖：
//   - F 型按钮权限行恰好 1 条（重复执行幂等）
//   - 精准授权：仅已关联父菜单的角色获得新按钮权限
//   - 父菜单缺失降级：返回 nil、不 panic、不新增行
//
// 原生 DDL（不走 GORM AutoMigrate）：sqlite 测试路径不经
// sanitizeSQLiteModelDefaults，model 里的 type:uuid 等 PG-only 片段会直接报错
// （同 migration_217 测试口径）。sys_menu 列集对齐 models.Menu 全部持久化列。
// ============================================================================

var migrate218Statements = []string{
	`CREATE TABLE sys_menu (
	    id TEXT PRIMARY KEY,
	    menu_name TEXT,
	    parent_id TEXT,
	    order_num INTEGER DEFAULT 0,
	    path TEXT,
	    component TEXT,
	    menu_type TEXT DEFAULT 'M',
	    visible INTEGER DEFAULT 1,
	    status INTEGER DEFAULT 0,
	    perms TEXT,
	    icon TEXT,
	    remark TEXT DEFAULT '',
	    meta TEXT,
	    created_by TEXT DEFAULT '',
	    updated_by TEXT DEFAULT '',
	    version INTEGER DEFAULT 0,
	    created_at DATETIME,
	    updated_at DATETIME,
	    deleted_at DATETIME
	)`,
	`CREATE TABLE sys_role (
	    id TEXT PRIMARY KEY,
	    role_name TEXT,
	    status INTEGER DEFAULT 0
	)`,
	`CREATE TABLE sys_role_menu (
	    role_id TEXT,
	    menu_id TEXT
	)`,
}

// freshSQLiteDBForMigrate218 建内存 sqlite 库与三张最小表。
func freshSQLiteDBForMigrate218(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for i, stmt := range migrate218Statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("建表 %d 失败: %v", i, err)
		}
	}
	return db
}

// execMigrate218 执行一批 SQL，任一条失败即终止用例。
func execMigrate218(t *testing.T, db *gorm.DB, stmts []string) {
	t.Helper()
	for i, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("执行第 %d 条 SQL 失败: %v\nSQL: %s", i, err, stmt)
		}
	}
}

// countMigrate218Rows 统计任意 SQL 的单值 COUNT 结果。
func countMigrate218Rows(t *testing.T, db *gorm.DB, query string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("统计失败 (%s): %v", query, err)
	}
	return n
}

// seedMigrate218ParentAndRoles 插入父菜单 + 两个角色，仅角色 A 关联父菜单。
func seedMigrate218ParentAndRoles(t *testing.T, db *gorm.DB) {
	t.Helper()
	execMigrate218(t, db, []string{
		// 父菜单形态对齐真实库：menu_name='域控配置' + perms='ops:ad:config:list'
		// （migration_218 按 perms 定位父菜单，不按 menu_name —— 见迁移注释）
		`INSERT INTO sys_menu (id, menu_name, menu_type, visible, status, path, order_num, perms)
		 VALUES ('menu-adcfg', '域控配置', 'C', 1, 0, 'ad-config', 3, 'ops:ad:config:list')`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('role-a', 'AD管理员', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('role-b', '普通角色', 0)`,
		// 只有角色 A 关联了父菜单
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('role-a', 'menu-adcfg')`,
	})
}

// TestPhase59_Migration218_BtnRowAndPrecisionGrant 锁定：
// F 行恰好 1 条、仅父菜单已关联角色获得授权、重复执行幂等。
func TestPhase59_Migration218_BtnRowAndPrecisionGrant(t *testing.T) {
	db := freshSQLiteDBForMigrate218(t)
	seedMigrate218ParentAndRoles(t, db)

	// 执行两次（幂等）
	for round := 1; round <= 2; round++ {
		if err := Migrate218ADReconciliationPermission(db); err != nil {
			t.Fatalf("第 %d 次迁移失败: %v", round, err)
		}
	}

	// F 型按钮权限行恰好 1 条
	if n := countMigrate218Rows(t, db,
		`SELECT COUNT(*) FROM sys_menu WHERE perms = ? AND menu_type = 'F'`,
		"ops:ad:config:reconciliation:list"); n != 1 {
		t.Fatalf("perms 行数 = %d, 期望 1（重复执行必须幂等）", n)
	}

	// 新按钮挂在父菜单下
	var parentID string
	if err := db.Raw(`SELECT parent_id FROM sys_menu WHERE perms = ?`,
		"ops:ad:config:reconciliation:list").Row().Scan(&parentID); err != nil {
		t.Fatalf("读取按钮行失败: %v", err)
	}
	if parentID != "menu-adcfg" {
		t.Fatalf("按钮 parent_id = %q, 期望 menu-adcfg", parentID)
	}

	// 精准授权：角色 A 获得、角色 B 没有
	var newMenuID string
	if err := db.Raw(`SELECT id FROM sys_menu WHERE perms = ?`,
		"ops:ad:config:reconciliation:list").Row().Scan(&newMenuID); err != nil {
		t.Fatalf("读取新菜单 ID 失败: %v", err)
	}
	if n := countMigrate218Rows(t, db,
		`SELECT COUNT(*) FROM sys_role_menu WHERE role_id = 'role-a' AND menu_id = ?`, newMenuID); n != 1 {
		t.Fatalf("角色 A 应恰有 1 条新按钮授权, 实际 %d", n)
	}
	if n := countMigrate218Rows(t, db,
		`SELECT COUNT(*) FROM sys_role_menu WHERE role_id = 'role-b' AND menu_id = ?`, newMenuID); n != 0 {
		t.Fatalf("角色 B 未关联父菜单, 不得获得新按钮授权, 实际 %d", n)
	}
}

// TestPhase59_Migration218_MissingParentDegradesGracefully 锁定父菜单缺失降级：
// 返回 nil、不 panic、不新增 perms 行、不新增授权行。
func TestPhase59_Migration218_MissingParentDegradesGracefully(t *testing.T) {
	db := freshSQLiteDBForMigrate218(t)
	// 不插入父菜单 —— 只有角色与角色菜单关联
	execMigrate218(t, db, []string{
		`INSERT INTO sys_role (id, role_name, status) VALUES ('role-a', 'AD管理员', 0)`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('role-a', 'menu-somewhere')`,
	})

	if err := Migrate218ADReconciliationPermission(db); err != nil {
		t.Fatalf("父菜单缺失时迁移必须返回 nil（非阻断降级）, 实际: %v", err)
	}

	if n := countMigrate218Rows(t, db,
		`SELECT COUNT(*) FROM sys_menu WHERE perms = ?`,
		"ops:ad:config:reconciliation:list"); n != 0 {
		t.Fatalf("父菜单缺失时不得新增 perms 行, 实际 %d", n)
	}
}

// TestPhase59_Migration218_MissingParentAfterCreation 锁定「先建后删」的降级路径：
// 首次执行创建成功，父菜单随后被删，再次执行不 panic 且不新增任何行。
func TestPhase59_Migration218_MissingParentAfterCreation(t *testing.T) {
	db := freshSQLiteDBForMigrate218(t)
	seedMigrate218ParentAndRoles(t, db)

	if err := Migrate218ADReconciliationPermission(db); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}

	// 删除父菜单（硬删，模拟数据被人工清理）
	execMigrate218(t, db, []string{
		`DELETE FROM sys_menu WHERE id = 'menu-adcfg'`,
	})

	before := countMigrate218Rows(t, db, `SELECT COUNT(*) FROM sys_role_menu`)
	if err := Migrate218ADReconciliationPermission(db); err != nil {
		t.Fatalf("父菜单被删后重复执行必须返回 nil, 实际: %v", err)
	}

	if n := countMigrate218Rows(t, db,
		`SELECT COUNT(*) FROM sys_menu WHERE perms = ? AND menu_type = 'F'`,
		"ops:ad:config:reconciliation:list"); n != 1 {
		t.Fatalf("F 行数 = %d, 期望仍为 1（降级路径不得重复创建）", n)
	}
	if after := countMigrate218Rows(t, db, `SELECT COUNT(*) FROM sys_role_menu`); after != before {
		t.Fatalf("sys_role_menu 行数 %d → %d, 降级路径不得新增授权", before, after)
	}
}

// TestPhase59_Migration218_GrantsNewRolesLater 锁定幂等补授权：
// 迁移跑过一次后，新角色获得父菜单授权，重跑迁移会把新按钮也授给它。
func TestPhase59_Migration218_GrantsNewRolesLater(t *testing.T) {
	db := freshSQLiteDBForMigrate218(t)
	seedMigrate218ParentAndRoles(t, db)

	if err := Migrate218ADReconciliationPermission(db); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}

	// 新角色 C 此刻才被授权父菜单（模拟迁移之后才接入的角色）
	execMigrate218(t, db, []string{
		`INSERT INTO sys_role (id, role_name, status) VALUES ('role-c', '新接入角色', 0)`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('role-c', 'menu-adcfg')`,
	})

	if err := Migrate218ADReconciliationPermission(db); err != nil {
		t.Fatalf("二次迁移失败: %v", err)
	}

	var newMenuID string
	if err := db.Raw(`SELECT id FROM sys_menu WHERE perms = ?`,
		"ops:ad:config:reconciliation:list").Row().Scan(&newMenuID); err != nil {
		t.Fatalf("读取新菜单 ID 失败: %v", err)
	}
	if n := countMigrate218Rows(t, db,
		`SELECT COUNT(*) FROM sys_role_menu WHERE role_id = 'role-c' AND menu_id = ?`, newMenuID); n != 1 {
		t.Fatalf("后接入父菜单的角色 C 应在重跑后获得按钮授权, 实际 %d", n)
	}
}
