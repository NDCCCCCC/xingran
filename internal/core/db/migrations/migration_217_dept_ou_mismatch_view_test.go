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

	// 3) body 形状守卫：body 是两方言共用的 SELECT 文本，断言它就等于断言 PG 侧的 SELECT 形状。
	// 这些断言在 PG 分支写错时立即失败，无需真实 PG 实例。
	body := buildDeptOUMismatchViewSelect()

	// 方言红线 1：每个 UNION ALL 分支的 ID/时间列必须 CAST（PG uuid 与 varchar 不能直接 UNION）
	if !strings.Contains(body, "CAST(") {
		t.Fatalf("视图 body 未使用 CAST(...) —— PG 侧 UNION 会报 types cannot be matched")
	}
	// 方言红线 2：布尔列取反写 NOT col（同时兼容 PG boolean 与 sqlite INTEGER）
	if !strings.Contains(body, "NOT mp.sync_enabled") {
		t.Fatalf("视图 body 未使用 NOT mp.sync_enabled —— PG boolean = 0 会报 operator does not exist")
	}
	// 方言红线 3：DN 后缀匹配必须用 SUBSTR + LENGTH（逗号边界），不得用 LIKE 通配符
	if !strings.Contains(body, "SUBSTR(") || !strings.Contains(body, "LENGTH(") {
		t.Fatalf("视图 body 未使用 SUBSTR/LENGTH 做 DN 后缀匹配")
	}
	if strings.Contains(body, "LIKE") {
		t.Fatalf("视图 body 出现 LIKE —— OU 名里的 _ / %% 会被当通配符污染匹配（Pitfall 5）")
	}
	// 方言红线 4：不得出现 sqlite 不支持的 PG 专有语法
	for _, banned := range []string{"ILIKE", "REGEXP_REPLACE", "MATERIALIZED VIEW"} {
		if strings.Contains(body, banned) {
			t.Fatalf("视图 body 出现方言专有/被禁语法 %q", banned)
		}
	}
	// D-11 十一列契约（第一分支决定列名）
	wantCols := []string{
		"dept_id", "dept_name", "dept_deleted_at", "ad_config_id", "config_name",
		"mapped_ou_dn", "user_id", "username", "user_ad_ou_dn", "mismatch_type", "mismatch_detail",
	}
	for _, c := range wantCols {
		if !strings.Contains(body, c) {
			t.Fatalf("视图 body 缺少 D-11 列 %q", c)
		}
	}
}

// execMigrate217 执行一批 SQL 语句，任一条失败即终止用例。
func execMigrate217(t *testing.T, db *gorm.DB, stmts []string) {
	t.Helper()
	for i, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("执行第 %d 条 SQL 失败: %v\nSQL: %s", i, err, stmt)
		}
	}
}

// migrate217ViewColumns 读 sqlite 的 PRAGMA table_info，返回视图列名（按序）。
func migrate217ViewColumns(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	rows, err := db.Raw("PRAGMA table_info(v_dept_ou_mismatch)").Rows()
	if err != nil {
		t.Fatalf("PRAGMA table_info 失败: %v", err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notNull   int
			dfltValue interface{}
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("扫描 PRAGMA 行失败: %v", err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("PRAGMA 迭代失败: %v", err)
	}
	return cols
}

// countMismatchType 统计某类口径的行数。
func countMismatchType(t *testing.T, db *gorm.DB, mismatchType string) int64 {
	t.Helper()
	var n int64
	if err := db.Raw("SELECT COUNT(*) FROM v_dept_ou_mismatch WHERE mismatch_type = ?", mismatchType).
		Scan(&n).Error; err != nil {
		t.Fatalf("统计 %s 失败: %v", mismatchType, err)
	}
	return n
}

// TestPhase59_Migration217_FourMismatchTypes 锁定四类口径在受控 fixture 下各产出恰一行，
// 且部门级分支（dept_soft_deleted / mapping_disabled）的 user_id 为 NULL；同时锁定 D-11 列顺序。
func TestPhase59_Migration217_FourMismatchTypes(t *testing.T) {
	db := freshSQLiteDBForMigrate217(t)

	execMigrate217(t, db, []string{
		// 唯一启用配置
		`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
		 VALUES ('cfg-1', '主域', 'DC=corp,DC=com', 0, NULL)`,
		// 三个部门：正常 / 已软删 / 映射停用但仍有用户
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-ok', 'IT部', NULL)`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-gone', '已删部门', '2026-01-01 00:00:00')`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-off', '停用映射部门', NULL)`,
		// 三条映射：正常启用 / 指向软删部门 / 停用
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-ok',   'cfg-1', 'OU=IT,DC=corp,DC=com',   1)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-gone', 'cfg-1', 'OU=Gone,DC=corp,DC=com', 1)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-off',  'cfg-1', 'OU=Off,DC=corp,DC=com',  0)`,
		// ① 用户 OU 与部门映射不等
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-mismatch', 'zhangsan', 'dept-ok', 'OU=Sales,DC=corp,DC=com', NULL)`,
		// ③ 停用映射部门下仍有活跃用户（该用户自身 OU 与映射相等，不产生 ①）
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-off', 'lisi', 'dept-off', 'OU=Off,DC=corp,DC=com', NULL)`,
		// ④ 用户 OU 不归属任何启用配置
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-unrouted', 'wangwu', 'dept-ok', 'OU=X,DC=other,DC=com', NULL)`,
	})

	if err := Migrate217DeptOUMismatchView(db); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	for _, mt := range []string{"user_ou_mismatch", "dept_soft_deleted", "mapping_disabled", "config_unrouted"} {
		if n := countMismatchType(t, db, mt); n != 1 {
			t.Fatalf("mismatch_type=%s 行数 = %d, 期望 1", mt, n)
		}
	}

	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM v_dept_ou_mismatch").Scan(&total).Error; err != nil {
		t.Fatalf("统计总行数失败: %v", err)
	}
	if total != 4 {
		t.Fatalf("视图总行数 = %d, 期望 4", total)
	}

	// ②③ 是部门级口径：user_id / username / user_ad_ou_dn 必须为 NULL（D-09）
	for _, mt := range []string{"dept_soft_deleted", "mapping_disabled"} {
		var nonNull int64
		if err := db.Raw(`SELECT COUNT(*) FROM v_dept_ou_mismatch
		                  WHERE mismatch_type = ?
		                    AND (user_id IS NOT NULL OR username IS NOT NULL OR user_ad_ou_dn IS NOT NULL)`,
			mt).Scan(&nonNull).Error; err != nil {
			t.Fatalf("检查 %s 的 user 列失败: %v", mt, err)
		}
		if nonNull != 0 {
			t.Fatalf("%s 有 %d 行的 user_id/username/user_ad_ou_dn 非 NULL, 期望全 NULL", mt, nonNull)
		}
	}

	// ④ 是用户级但无配置归属：ad_config_id / config_name / mapped_ou_dn 必须为 NULL
	var unroutedNonNull int64
	if err := db.Raw(`SELECT COUNT(*) FROM v_dept_ou_mismatch
	                  WHERE mismatch_type = 'config_unrouted'
	                    AND (ad_config_id IS NOT NULL OR config_name IS NOT NULL OR mapped_ou_dn IS NOT NULL)`).
		Scan(&unroutedNonNull).Error; err != nil {
		t.Fatalf("检查 config_unrouted 的配置列失败: %v", err)
	}
	if unroutedNonNull != 0 {
		t.Fatalf("config_unrouted 有 %d 行的配置列非 NULL, 期望全 NULL", unroutedNonNull)
	}

	// D-11 列顺序：视图列必须逐项等于约定 11 列
	wantCols := []string{
		"dept_id", "dept_name", "dept_deleted_at", "ad_config_id", "config_name",
		"mapped_ou_dn", "user_id", "username", "user_ad_ou_dn", "mismatch_type", "mismatch_detail",
	}
	gotCols := migrate217ViewColumns(t, db)
	if len(gotCols) != len(wantCols) {
		t.Fatalf("视图列数 = %d (%v), 期望 %d", len(gotCols), gotCols, len(wantCols))
	}
	for i := range wantCols {
		if gotCols[i] != wantCols[i] {
			t.Fatalf("视图第 %d 列 = %q, 期望 %q（完整列序 %v）", i+1, gotCols[i], wantCols[i], gotCols)
		}
	}
}

// TestPhase59_Migration217_NormalizedOUIgnoresCaseAndSpace 锁定第一类口径用**归一化**比较：
// 大小写与逗号/等号周边空格不同但归一化相等的 OU **不得**产生 user_ou_mismatch。
func TestPhase59_Migration217_NormalizedOUIgnoresCaseAndSpace(t *testing.T) {
	db := freshSQLiteDBForMigrate217(t)

	execMigrate217(t, db, []string{
		`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
		 VALUES ('cfg-1', '主域', 'DC=corp,DC=com', 0, NULL)`,
		`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-1', 'IT部', NULL)`,
		`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
		 VALUES ('dept-1', 'cfg-1', 'OU=IT,DC=corp,DC=com', 1)`,
		// 归一化后与上方映射逐字符相等：小写 + 逗号/等号两侧空格
		`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
		 VALUES ('u-1', 'zhangsan', 'dept-1', 'ou=it , dc=corp ,dc=com', NULL)`,
	})

	if err := Migrate217DeptOUMismatchView(db); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	if n := countMismatchType(t, db, "user_ou_mismatch"); n != 0 {
		t.Fatalf("归一化等价的 OU 产生了 %d 行 user_ou_mismatch, 期望 0（归一化比较失效）", n)
	}
	// 该用户在启用配置下可路由，因此也不得落到 config_unrouted
	if n := countMismatchType(t, db, "config_unrouted"); n != 0 {
		t.Fatalf("已路由用户被误判为 config_unrouted (%d 行)", n)
	}
}

// TestPhase59_Migration217_EmptyBaseDNIsNotRouted 锁定空 base_dn 的启用配置不参与匹配。
func TestPhase59_Migration217_EmptyBaseDNIsNotRouted(t *testing.T) {
	t.Run("only_empty_base", func(t *testing.T) {
		db := freshSQLiteDBForMigrate217(t)
		execMigrate217(t, db, []string{
			`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
			 VALUES ('cfg-empty', '空库', '', 0, NULL)`,
			`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-1', 'IT部', NULL)`,
			`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
			 VALUES ('u-1', 'zhangsan', 'dept-1', 'OU=Any,DC=corp,DC=com', NULL)`,
		})
		if err := Migrate217DeptOUMismatchView(db); err != nil {
			t.Fatalf("迁移失败: %v", err)
		}
		if n := countMismatchType(t, db, "config_unrouted"); n != 1 {
			t.Fatalf("config_unrouted 行数 = %d, 期望 1（空 base 不得命中）", n)
		}
	})

	t.Run("empty_base_does_not_steal_route", func(t *testing.T) {
		db := freshSQLiteDBForMigrate217(t)
		execMigrate217(t, db, []string{
			`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
			 VALUES ('cfg-empty', '空库', '   ', 0, NULL)`,
			`INSERT INTO sys_ad_config (id, config_name, base_dn, status, deleted_at)
			 VALUES ('cfg-real', '主域', 'DC=corp,DC=com', 0, NULL)`,
			`INSERT INTO sys_dept (id, dept_name, deleted_at) VALUES ('dept-1', 'IT部', NULL)`,
			`INSERT INTO sys_dept_ou_mapping (dept_id, ad_config_id, ou_dn, sync_enabled)
			 VALUES ('dept-1', 'cfg-real', 'OU=IT,DC=corp,DC=com', 1)`,
			`INSERT INTO sys_user (id, username, dept_id, ad_ou_dn, deleted_at)
			 VALUES ('u-1', 'zhangsan', 'dept-1', 'OU=Sales,DC=corp,DC=com', NULL)`,
		})
		if err := Migrate217DeptOUMismatchView(db); err != nil {
			t.Fatalf("迁移失败: %v", err)
		}
		if n := countMismatchType(t, db, "config_unrouted"); n != 0 {
			t.Fatalf("空 base 配置抢占了路由（config_unrouted = %d, 期望 0）", n)
		}
		var configID string
		if err := db.Raw(`SELECT ad_config_id FROM v_dept_ou_mismatch
		                  WHERE mismatch_type = 'user_ou_mismatch'`).Row().Scan(&configID); err != nil {
			t.Fatalf("读取 user_ou_mismatch 行失败: %v", err)
		}
		if configID != "cfg-real" {
			t.Fatalf("路由到的配置 = %q, 期望 cfg-real（空 base 不得参与匹配）", configID)
		}
	})
}
