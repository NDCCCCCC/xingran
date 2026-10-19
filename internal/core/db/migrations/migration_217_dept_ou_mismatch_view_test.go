package migrations

import (
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ============================================================================
// Phase 59 / GUARD-02 迁移 217 测试
//
// 覆盖：
//   - sqlite 建视图 + 幂等（连跑两次不报错）
//   - 视图可查询、空 fixture 下 0 行
//   - user_ou_mismatch 分支恰好产出 1 行（受控 fixture）
//   - PG 方言 DDL 文本守卫（无 PG 实例时 PG 分支唯一的可失败检查）
// ============================================================================

// migrate217Statements 视图引用的四张表的最小列集（原生 DDL，不走 GORM AutoMigrate）。
//
// 为什么手写 DDL 而不是 AutoMigrate(&models.…)：sqlite 测试路径不经
// sanitizeSQLiteModelDefaults，model 里的 gen_random_uuid() / type:uuid 等 PG-only 片段
// 会直接报语法错误；手写 DDL 同时把「视图实际依赖哪几列」变成显式契约。
//
// ⚠ 列类型对齐迁移运行期的真实形态：sqlite 全 TEXT（无强类型），
// 布尔列用 INTEGER（sync_enabled / status），CAST(x AS TEXT) 在两侧都成立。
var migrate217Statements = []string{
	`CREATE TABLE sys_user (
	    id TEXT PRIMARY KEY,
	    username TEXT,
	    dept_id TEXT,
	    ad_ou_dn TEXT,
	    deleted_at TEXT
	)`,
	`CREATE TABLE sys_dept (
	    id TEXT PRIMARY KEY,
	    dept_name TEXT,
	    deleted_at TEXT
	)`,
	`CREATE TABLE sys_dept_ou_mapping (
	    dept_id TEXT,
	    ad_config_id TEXT,
	    ou_dn TEXT,
	    sync_enabled INTEGER DEFAULT 1
	)`,
	`CREATE TABLE sys_ad_config (
	    id TEXT PRIMARY KEY,
	    config_name TEXT,
	    base_dn TEXT,
	    status INTEGER DEFAULT 0,
	    deleted_at TEXT
	)`,
}

// freshSQLiteDBForMigrate217 构建内存 SQLite 库并建好视图依赖的四张表。
// 形状照 migration_216_adpush_drain_sysconfig_test.go 的 freshSQLiteDBForMigrate216。
func freshSQLiteDBForMigrate217(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for i, stmt := range migrate217Statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("建表 %d 失败: %v", i, err)
		}
	}
	return db
}

// TestPhase59_Migration217_SQLiteCreatesViewAndIsIdempotent 锁定：
// 迁移可在 sqlite 建出视图、连跑两次不报错、视图可查询。
func TestPhase59_Migration217_SQLiteCreatesViewAndIsIdempotent(t *testing.T) {
	db := freshSQLiteDBForMigrate217(t)

	if err := Migrate217DeptOUMismatchView(db); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}
	// 幂等：sqlite 分支先 DROP VIEW IF EXISTS 再 CREATE VIEW，第二次必须同样成功。
	if err := Migrate217DeptOUMismatchView(db); err != nil {
		t.Fatalf("二次迁移失败（幂等被破坏）: %v", err)
	}

	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM v_dept_ou_mismatch").Scan(&total).Error; err != nil {
		t.Fatalf("查询 v_dept_ou_mismatch 失败: %v", err)
	}
	if total != 0 {
		t.Fatalf("空 fixture 下视图行数 = %d, 期望 0", total)
	}
}

// TestPhase59_Migration217_UserOUMismatchRow 锁定第一类口径：
// 用户 ad_ou_dn 与其所属部门在该路由配置下的映射 ou_dn 不等 → 恰 1 行 user_ou_mismatch。
func TestPhase59_Migration217_UserOUMismatchRow(t *testing.T) {
	db := freshSQLiteDBForMigrate217(t)

	seed := []string{
		`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
		 VALUES ('cfg-1', '主域', 'DC=corp,DC=com', 0, NULL)`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at)
		 VALUES ('dept-1', 'IT部', NULL)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-1', 'cfg-1', 'OU=IT,DC=corp,DC=com', 1)`,
		// 用户的 OU 归一化后 ≠ 映射 OU → 触发 user_ou_mismatch
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('user-1', 'zhangsan', 'dept-1', 'OU=Sales,DC=corp,DC=com', NULL)`,
	}
	for i, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed %d 失败: %v", i, err)
		}
	}

	if err := Migrate217DeptOUMismatchView(db); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	var rows int64
	if err := db.Raw(`SELECT COUNT(*) FROM v_dept_ou_mismatch WHERE mismatch_type = 'user_ou_mismatch'`).
		Scan(&rows).Error; err != nil {
		t.Fatalf("统计 user_ou_mismatch 失败: %v", err)
	}
	if rows != 1 {
		t.Fatalf("user_ou_mismatch 行数 = %d, 期望 1", rows)
	}

	var (
		deptID     string
		deptName   string
		configID   string
		mappedOUDN string
		userID     string
		username   string
		userOUDN   string
		detail     string
	)
	if err := db.Raw(`SELECT dept_id, dept_name, ad_config_id, mapped_ou_dn,
	                         user_id, username, user_ad_ou_dn, mismatch_detail
	                  FROM v_dept_ou_mismatch`).
		Row().Scan(&deptID, &deptName, &configID, &mappedOUDN, &userID, &username, &userOUDN, &detail); err != nil {
		t.Fatalf("读取 user_ou_mismatch 行失败: %v", err)
	}
	if deptID != "dept-1" || deptName != "IT部" || configID != "cfg-1" {
		t.Fatalf("部门/配置列不符: deptID=%q deptName=%q configID=%q", deptID, deptName, configID)
	}
	if mappedOUDN != "OU=IT,DC=corp,DC=com" {
		t.Fatalf("mapped_ou_dn = %q", mappedOUDN)
	}
	if userID != "user-1" || username != "zhangsan" || userOUDN != "OU=Sales,DC=corp,DC=com" {
		t.Fatalf("用户列不符: userID=%q username=%q userOUDN=%q", userID, username, userOUDN)
	}
	if !strings.Contains(detail, "用户 OU「OU=Sales,DC=corp,DC=com」") {
		t.Fatalf("mismatch_detail 文案不符: %q", detail)
	}
}

// TestPhase59_Migration217_PGDialectDDL 是 PG 分支在无 PG 实例时**唯一**的可失败检查。
//
// 背景：sqlite 测试环境跑不到 Migrate217DeptOUMismatchView 的 PG 分支（isPostgreSQL 为假），
// 因此把该分支的产物抽成纯函数 pgDeptOUMismatchViewDDL / sqliteDeptOUMismatchViewDDLs，
// 直接断言 DDL 文本形状。body 是两方言共用文本，故断言 body 等于断言 PG 侧 SELECT 形状。
func TestPhase59_Migration217_PGDialectDDL(t *testing.T) {
	// 1) PG：必须是 CREATE OR REPLACE VIEW —— 重复启动幂等的前提
	got := pgDeptOUMismatchViewDDL("BODY")
	want := "CREATE OR REPLACE VIEW v_dept_ou_mismatch AS BODY"
	if got != want {
		t.Fatalf("PG DDL = %q, 期望 %q", got, want)
	}

	// 2) sqlite：恰两条，DROP 在前 CREATE 在后（sqlite 无 CREATE OR REPLACE VIEW）
	ddls := sqliteDeptOUMismatchViewDDLs("BODY")
	if len(ddls) != 2 {
		t.Fatalf("sqlite DDL 条数 = %d, 期望 2", len(ddls))
	}
	if !strings.Contains(ddls[0], "DROP VIEW IF EXISTS v_dept_ou_mismatch") {
		t.Fatalf("sqlite DDL[0] 应含 DROP VIEW IF EXISTS v_dept_ou_mismatch, 实际 %q", ddls[0])
	}
	if !strings.Contains(ddls[1], "CREATE VIEW v_dept_ou_mismatch AS BODY") {
		t.Fatalf("sqlite DDL[1] 应含 CREATE VIEW v_dept_ou_mismatch AS BODY, 实际 %q", ddls[1])
	}
}
