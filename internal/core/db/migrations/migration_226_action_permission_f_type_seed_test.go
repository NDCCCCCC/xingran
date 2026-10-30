package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 87 / RBAC-05 迁移 226 测试（全量 65 条动作权限）
//
// 覆盖：
//   - 87-NN-01: 65 条动作码在 sys_menu 各有且仅有 1 行（menu_type='F', status=0, visible=0, parent_id 指向 live 父）
//   - 87-NN-02: 迁移幂等性（连跑 2 次行数不变）
//   - 87-NN-03: 精准授权（持 live 父角色获授权，持 status=1 停用父角色与未持父角色不获授权）
//   - 87-NN-04: 父菜单缺失优雅降级（返回 nil，不 panic，零新增行）
//   - 87-NN-05: 多父集合支持（network:command 的 2 个父 C 菜单角色都能获授权）
//   - AGENTS.md P1: visible 显式归零（所有 F 按钮 visible 必为 0）
//   - 87-NN-16: 既有行零改动（SC3 自动化等价值：既有菜单与角色关联字段逐字未改）
//
// OQ-11 防假绿约束：测试文件不得硬编码 65 码字面量列表，统一引用包内 actionPermissionSeeds。
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

// seedFullParentsAndRoles 为全量 65 码准备对应所有的 live 父菜单、停用旧树副本以及测试角色
func seedFullParentsAndRoles(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		// 1. system 4 组父菜单 (live)
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-sys-role', '角色管理', 'system:role', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-sys-menu', '菜单管理', 'system:menu', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-sys-dept', '部门管理', 'system:dept', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-sys-post', '岗位管理', 'system:post', 'C', 0, 1)`,

		// 2. ops 8 组父菜单 (各造 live status=0 与 stopped status=1 两套平行树)
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-building-live', '楼宇管理', 'ops:building:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-building-stop', '楼宇管理', 'ops:building:list', 'C', 1, 1)`,

		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-floor-live', '楼层管理', 'ops:floor:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-floor-stop', '楼层管理', 'ops:floor:list', 'C', 1, 1)`,

		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-workstation-live', '工位管理', 'ops:workstation:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-workstation-stop', '工位管理', 'ops:workstation:list', 'C', 1, 1)`,

		// ops:asset 特例父菜单 (perms 为 NULL)
		`INSERT INTO sys_menu (id, menu_name, perms, component, menu_type, status, visible) VALUES ('p-ops-asset-live', '资产列表', NULL, 'operations/assets/index', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, component, menu_type, status, visible) VALUES ('p-ops-asset-stop', '资产列表', NULL, 'operations/assets/index', 'C', 1, 1)`,

		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-infopoint-live', '信息点管理', 'ops:infopoint:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-infopoint-stop', '信息点管理', 'ops:infopoint:list', 'C', 1, 1)`,

		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-serverroom-live', '机房管理', 'ops:serverroom:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-serverroom-stop', '机房管理', 'ops:serverroom:list', 'C', 1, 1)`,

		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-dedicatedline-live', '专线管理', 'ops:dedicatedline:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-dedicatedline-stop', '专线管理', 'ops:dedicatedline:list', 'C', 1, 1)`,

		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-roomdevice-live', '机房设备管理', 'ops:roomdevice:list', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-ops-roomdevice-stop', '机房设备管理', 'ops:roomdevice:list', 'C', 1, 1)`,

		// 3. network 6 组父菜单
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-net-device', '设备管理', 'network:device', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-net-credential', '授权凭据', 'network:credential', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-net-template', '配置模板', 'network:template', 'C', 0, 1)`,

		// network:command 包含两个 status=0 的父菜单：命令分发 + 配置执行 (多父集合)
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-net-cmd-dist', '命令分发', 'network:command', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-net-cmd-exec', '配置执行', 'network:command', 'C', 0, 1)`,

		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-net-backup', '配置备份', 'network:backup', 'C', 0, 1)`,
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible) VALUES ('p-net-discovery', '设备发现', 'network:discovery', 'C', 0, 1)`,

		// 4. 测试角色准备
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-all-parents', '全持 live 父角色', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-stop-parents', '全持 stopped 父角色', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-cmd-dist-only', '仅持命令分发角色', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-cmd-exec-only', '仅持配置执行角色', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('r-none', '无任何权限角色', 0)`,

		// r-all-parents 关联全部 live 父
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-sys-role')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-sys-menu')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-sys-dept')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-sys-post')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-building-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-floor-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-workstation-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-asset-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-infopoint-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-serverroom-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-dedicatedline-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-ops-roomdevice-live')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-net-device')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-net-credential')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-net-template')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-net-cmd-dist')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-net-cmd-exec')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-net-backup')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'p-net-discovery')`,

		// r-stop-parents 仅关联全部停用父
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-building-stop')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-floor-stop')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-workstation-stop')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-asset-stop')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-infopoint-stop')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-serverroom-stop')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-dedicatedline-stop')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-stop-parents', 'p-ops-roomdevice-stop')`,

		// 多父细分验证角色
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-cmd-dist-only', 'p-net-cmd-dist')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-cmd-exec-only', 'p-net-cmd-exec')`,
	}
	execMigrate226(t, db, stmts)
}

// TestMigrate226_SeedsAllCodes 验证全量 65 码在 sys_menu 各有且仅有 1 行，且 parent_id 全部指向 live 父
func TestMigrate226_SeedsAllCodes(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedFullParentsAndRoles(t, db)

	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("Migrate226 执行失败: %v", err)
	}

	// 1) 验证总 F 行数正好为 65
	totalButtons := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu WHERE menu_type = 'F'")
	if totalButtons != int64(len(actionPermissionSeeds)) {
		t.Fatalf("期望创建 %d 条 F 按钮, 实际创建=%d", len(actionPermissionSeeds), totalButtons)
	}

	// 2) 逐条验证包内 actionPermissionSeeds 中的每个码
	for _, seed := range actionPermissionSeeds {
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
		if row.Status != 0 {
			t.Errorf("码 %s 期望 status=0, 实际=%d", seed.perms, row.Status)
		}
		if row.Visible != 0 {
			t.Errorf("码 %s 期望 visible=0, 实际=%d", seed.perms, row.Visible)
		}

		// 断言 parent_id 指向 live 父菜单（不能包含 -stop）
		var parentStatus int
		err = db.Raw("SELECT status FROM sys_menu WHERE id = ?", row.ParentID).Scan(&parentStatus).Error
		if err != nil {
			t.Fatalf("查询父菜单 %s 状态失败: %v", row.ParentID, err)
		}
		if parentStatus != 0 {
			t.Errorf("码 %s 挂接到了非 live 父菜单(status=%d, parent_id=%s)", seed.perms, parentStatus, row.ParentID)
		}
	}
}

// TestMigrate226_Idempotent 验证连续执行两次迁移具有严格幂等性
func TestMigrate226_Idempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedFullParentsAndRoles(t, db)

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

// TestMigrate226_GrantsOnlyParentHolders 验证精准授权与停用副本防越权
func TestMigrate226_GrantsOnlyParentHolders(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedFullParentsAndRoles(t, db)

	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("Migrate226 失败: %v", err)
	}

	// 1) 持有全部 live 父的角色应该获得全部 65 条 F 行的授权
	grantedToHolder := countMigrate226Rows(t, db,
		`SELECT count(*) FROM sys_role_menu rm
		 JOIN sys_menu m ON rm.menu_id = m.id
		 WHERE rm.role_id = 'r-all-parents' AND m.menu_type = 'F'`)
	if grantedToHolder != 65 {
		t.Errorf("期望 r-all-parents 获得全部 65 条 F 行授权, 实际=%d", grantedToHolder)
	}

	// 2) 仅持有停用副本的角色，绝不能获得任何新增 F 行授权（停用旧树防越权）
	grantedToStopHolder := countMigrate226Rows(t, db,
		`SELECT count(*) FROM sys_role_menu rm
		 JOIN sys_menu m ON rm.menu_id = m.id
		 WHERE rm.role_id = 'r-stop-parents' AND m.menu_type = 'F'`)
	if grantedToStopHolder != 0 {
		t.Errorf("越权漏洞：r-stop-parents 获得了 F 行授权, 实际计数=%d", grantedToStopHolder)
	}

	// 3) 无任何权限的角色，绝不能获得任何新增 F 行授权
	grantedToNone := countMigrate226Rows(t, db,
		`SELECT count(*) FROM sys_role_menu rm
		 JOIN sys_menu m ON rm.menu_id = m.id
		 WHERE rm.role_id = 'r-none' AND m.menu_type = 'F'`)
	if grantedToNone != 0 {
		t.Errorf("越权漏洞：r-none 获得了 F 行授权, 实际计数=%d", grantedToNone)
	}
}

// TestMigrate226_MissingParentDegradesGracefully 验证父菜单缺失时的优雅降级
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

// TestMigrate226_MultiParent 验证 network:command 的 2 个父 C 菜单角色都能获授权
func TestMigrate226_MultiParent(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedFullParentsAndRoles(t, db)

	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("Migrate226 执行失败: %v", err)
	}

	// 查询 network:command 的 2 个按钮菜单 ID
	var cmdMenuIDs []string
	if err := db.Raw("SELECT id FROM sys_menu WHERE parent_id IN ('p-net-cmd-dist', 'p-net-cmd-exec') AND menu_type = 'F'").Scan(&cmdMenuIDs).Error; err != nil {
		t.Fatalf("查询命令按钮 ID 失败: %v", err)
	}
	if len(cmdMenuIDs) != 2 {
		t.Fatalf("期望找到 2 个 network:command 按钮, 实际=%d", len(cmdMenuIDs))
	}

	// 1) 仅持有命令分发(p-net-cmd-dist)的角色获得这 2 个按钮的授权
	distGranted := countMigrate226Rows(t, db,
		"SELECT count(*) FROM sys_role_menu WHERE role_id = 'r-cmd-dist-only' AND menu_id IN ?", cmdMenuIDs)
	if distGranted != 2 {
		t.Errorf("期望 r-cmd-dist-only 获得 2 个命令按钮授权, 实际=%d", distGranted)
	}

	// 2) 仅持有配置执行(p-net-cmd-exec)的角色也获得这 2 个按钮的授权
	execGranted := countMigrate226Rows(t, db,
		"SELECT count(*) FROM sys_role_menu WHERE role_id = 'r-cmd-exec-only' AND menu_id IN ?", cmdMenuIDs)
	if execGranted != 2 {
		t.Errorf("期望 r-cmd-exec-only 获得 2 个命令按钮授权, 实际=%d", execGranted)
	}
}

// TestMigrate226_VisibleIsZero 验证所有 65 条 F 按钮在落库后 visible 必须严格为 0
func TestMigrate226_VisibleIsZero(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedFullParentsAndRoles(t, db)

	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("Migrate226 执行失败: %v", err)
	}

	nonZeroVisible := countMigrate226Rows(t, db,
		"SELECT count(*) FROM sys_menu WHERE menu_type = 'F' AND visible != 0")
	if nonZeroVisible != 0 {
		t.Errorf("违反 AGENTS.md P1 规范：有 %d 条 F 按钮菜单 visible != 0", nonZeroVisible)
	}
}

// TestMigrate226_ExistingRowsUntouched 验证既有行零改动（SC3 自动化等价值）
func TestMigrate226_ExistingRowsUntouched(t *testing.T) {
	db := freshSQLiteDBForMigrate226(t)
	seedFullParentsAndRoles(t, db)

	// 预置既有非本批 F 型行、既有 C 型行、既有授权行
	execMigrate226(t, db, []string{
		`INSERT INTO sys_menu (id, menu_name, perms, menu_type, status, visible, parent_id)
		 VALUES ('m-exist-f', '既有用户导出', 'system:user:export', 'F', 0, 0, 'p-sys-role')`,
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('r-all-parents', 'm-exist-f')`,
	})

	// 获取既有行特征
	var beforeMenu struct {
		MenuName string
		Perms    string
		MenuType string
		Status   int
		Visible  int
		ParentID string
	}
	if err := db.Raw("SELECT menu_name, perms, menu_type, status, visible, parent_id FROM sys_menu WHERE id = 'm-exist-f'").Scan(&beforeMenu).Error; err != nil {
		t.Fatalf("查询既有行失败: %v", err)
	}

	initialMenuCount := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")

	// 执行迁移
	if err := Migrate226ActionPermissionFTypeSeed(db); err != nil {
		t.Fatalf("Migrate226 执行失败: %v", err)
	}

	// 1) 验证既有行字段逐字未变
	var afterMenu struct {
		MenuName string
		Perms    string
		MenuType string
		Status   int
		Visible  int
		ParentID string
	}
	if err := db.Raw("SELECT menu_name, perms, menu_type, status, visible, parent_id FROM sys_menu WHERE id = 'm-exist-f'").Scan(&afterMenu).Error; err != nil {
		t.Fatalf("查询迁移后既有行失败: %v", err)
	}

	if beforeMenu != afterMenu {
		t.Errorf("SC3 破坏：既有菜单行被篡改! 前=%+v, 后=%+v", beforeMenu, afterMenu)
	}

	// 2) 验证总数增量恰好等于 65
	afterMenuCount := countMigrate226Rows(t, db, "SELECT count(*) FROM sys_menu")
	if afterMenuCount != initialMenuCount+65 {
		t.Errorf("sys_menu 总数增量不等于 65: 前=%d, 后=%d", initialMenuCount, afterMenuCount)
	}
}
