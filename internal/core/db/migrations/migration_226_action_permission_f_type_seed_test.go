package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 87 / RBAC-05 迁移 226 测试
//
// 覆盖：
//   - 87-NN-01: F 型按钮权限行恰好 1 条（menu_type='F', status=0, visible=0）
//   - 87-NN-02: 迁移幂等性（连跑 2 次行数不变）
//   - 87-NN-03: 精准授权（持 live 父角色获授权，持 status=1 停用父角色与未持父角色不获授权）
//   - 87-NN-04: 父菜单缺失优雅降级（返回 nil，不 panic，零新增行）
//
// 原生 DDL（不走 GORM AutoMigrate）：避免模型中 PG 专有定义在 sqlite 报错。
// ============================================================================

var migrate226Statements = []string{
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

func freshSQLiteDBForMigrate226(t *testing.T) *gorm.DB {
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

func execMigrate226(t *testing.T, db *gorm.DB, stmts []string) {
	t.Helper()
	for i, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("执行第 %d 条 SQL 失败: %v\nSQL: %s", i, err, stmt)
		}
	}
}

func countMigrate226Rows(t *testing.T, db *gorm.DB, query string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("统计失败 (%s): %v", query, err)
	}
	return n
}

// seedTracerParentAndRoles 建立试点码的父 C 菜单（包含 live 与停用副本）和测试角色
func seedTracerParentAndRoles(t *testing.T, db *gorm.DB) {
	t.Helper()
	execMigrate226(t, db, []string{
		// 1) live 角色管理父菜单 (status=0)
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible)
		 VALUES ('m-role-live', '角色管理', 'system:role', 'C', 0, 1)`,
		// 2) 停用旧树角色管理副本 (status=1) —— 越权防护断言用
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible)
		 VALUES ('m-role-stopped', '角色管理', 'system:role', 'C', 1, 1)`,
		// 3) 角色 r-holder 持有 live 父
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-holder', '持有者角色', 0)`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-holder', 'm-role-live')`,
		// 4) 角色 r-stopped-holder 仅持有停用父
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-stopped-holder', '停用父持有者', 0)`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stopped-holder', 'm-role-stopped')`,
		// 5) 角色 r-other 未关联任何父菜单
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-other', '未授权角色', 0)`,
	})
}

// TestMigrate226_SeedsAllCodes 验证试点码正确播种且字段口径符合约定。
func TestMigrate226_SeedsAllCodes(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedTracerParentAndRoles(t, db)

	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("Migrate226 失败: %v", err)
	}

	// 断言试点码在 sys_menu 有且仅有 1 行
	totalButtons := countMigrate226Rows(t, db,
		"SELECT count(*) FROM sys_menu WHERE menu_type = 'F' AND perms = 'system:role:add'")
	if totalButtons != 1 {
		t.Fatalf("期望试点码按钮行数=1, 实际=%d", totalButtons)
	}

	// 断言 status=0, visible=0, parent_id 指向 live 父菜单
	var menu struct {
		ParentID string
		Status   int
		Visible  int
	}
	if err := db.Raw("SELECT parent_id, status, visible FROM sys_menu WHERE perms = 'system:role:add'").Scan(&menu).Error; err != nil {
		t.Fatalf("查询试点码字段失败: %v", err)
	}
	if menu.Status != 0 {
		t.Errorf("期望 status=0, 实际=%d", menu.Status)
	}
	if menu.Visible != 0 {
		t.Errorf("期望 visible=0, 实际=%d", menu.Visible)
	}
	if menu.ParentID != "m-role-live" {
		t.Errorf("期望 parent_id='m-role-live', 实际=%s", menu.ParentID)
	}
}

// TestMigrate226_Idempotent 验证连续执行两次迁移具有严格幂等性。
func TestMigrate226_Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedTracerParentAndRoles(t, db)

	// 第 1 次执行
	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("第 1 次执行失败: %v", err)
	}
	menuCount1 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")
	roleMenuCount1 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_role_menu")

	// 第 2 次执行
	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("第 2 次执行失败: %v", err)
	}
	menuCount2 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")
	roleMenuCount2 := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_role_menu")

	if menuCount1 != menuCount2 {
		t.Errorf("sys_menu 破坏幂等: 第1次=%d, 第2次=%d", menuCount1, menuCount2)
	}
	if roleMenuCount1 != roleMenuCount2 {
		t.Errorf("sys_role_menu 破坏幂等: 第1次=%d, 第2次=%d", roleMenuCount1, roleMenuCount2)
	}
}

// TestMigrate226_GrantsOnlyParentHolders 验证精准授权与停用副本防越权。
func TestMigrate226_GrantsOnlyParentHolders(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedTracerParentAndRoles(t, db)

	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("Migrate226 失败: %v", err)
	}

	var buttonID string
	if err := db.Raw("SELECT id FROM sys_menu WHERE perms = 'system:role:add'").Scan(&buttonID).Error; err != nil {
		t.Fatalf("获取按钮 ID 失败: %v", err)
	}

	// 1) 持有 live 父的角色获得授权
	holderGranted := countMigrate226Rows(t, db,
		"SELECT count(*) FROM sys_role_menu WHERE role_id = 'r-holder' AND menu_id = ?", buttonID)
	if holderGranted != 1 {
		t.Errorf("期望 r-holder 获得授权, 实际计数=%d", holderGranted)
	}

	// 2) 越权防线：持有 status=1 停用副本角色的不获授权
	stoppedHolderGranted := countMigrate226Rows(t, db,
		"SELECT count(*) FROM sys_role_menu WHERE role_id = 'r-stopped-holder' AND menu_id = ?", buttonID)
	if stoppedHolderGranted != 0 {
		t.Errorf("越权漏洞：r-stopped-holder 获得了授权, 计数=%d", stoppedHolderGranted)
	}

	// 3) 未持有任何父的角色不获授权
	otherGranted := countMigrate226Rows(t, db,
		"SELECT count(*) FROM sys_role_menu WHERE role_id = 'r-other' AND menu_id = ?", buttonID)
	if otherGranted != 0 {
		t.Errorf("越权漏洞：r-other 获得了授权, 计数=%d", otherGranted)
	}
}

// TestMigrate226_MissingParentDegradesGracefully 验证父菜单不存在时优雅降级。
func TestMigrate226_MissingParentDegradesGracefully(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	// 不插入任何父菜单与角色

	menuCountBefore := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")
	roleMenuCountBefore := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_role_menu")

	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("期望父缺失时返回 nil, 实际报错: %v", err)
	}

	menuCountAfter := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")
	roleMenuCountAfter := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_role_menu")

	if menuCountBefore != menuCountAfter {
		t.Errorf("父缺失时 sys_menu 产生新增行: 前=%d, 后=%d", menuCountBefore, menuCountAfter)
	}
	if roleMenuCountBefore != roleMenuCountAfter {
		t.Errorf("父缺失时 sys_role_menu 产生新增行: 前=%d, 后=%d", roleMenuCountBefore, roleMenuCountAfter)
	}
}
