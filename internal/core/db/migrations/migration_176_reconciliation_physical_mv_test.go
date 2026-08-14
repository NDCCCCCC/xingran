package migrations

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// stripGoComments176 解析指定 Go 源文件并返回非注释代码文本(声明+语句+字符串字面量)。
// 用于源码 grep 守卫,排除文档注释对标识符的误命中(migration_176 顶部的 R1/R2
// 历史注释会引用 schema 名称,需要剥离)。
//
// 复用 migration_202_port_write_audit_test.go:stripGoComments 的实现模式(同一项目内
// 多文件源码 grep 守卫的轻量 helper)。
func stripGoComments176(t *testing.T, path string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var sb strings.Builder
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Comment, *ast.CommentGroup:
			return false // skip comments
		case *ast.BasicLit:
			sb.WriteString(x.Value)
			sb.WriteByte(' ')
		case *ast.Ident:
			sb.WriteString(x.Name)
			sb.WriteByte(' ')
		}
		return true
	})
	return sb.String()
}

// freshSQLiteDB176 为 migration_176 / 175 源码 grep / sqlite skip 测试提供内存 SQLite。
// 沿用 menu_grant_helpers_test.go:freshSQLiteDB 与 migration_202_port_write_audit_test.go
// :freshSQLiteDBForMigrate202 的 DSN + BusyTimeout 模式(无需新建 models;PG-only 迁移走
// isPostgreSQL 守卫直接返回 nil)。
func freshSQLiteDB176(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file::memory:?cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

// ===== Task 1 — C1 加固 =====

// TestMigrate176_FastPathSchemaVersionCheck 源码断言:
//   - migration_176 含 "information_schema.columns"(MV schema 版本校验查询)
//   - 含 "backfillOpsAssetPhysical"(回填提取为私有函数)
//   - Type E 清理成功路径使用 applogger.Warnf(审计可见,每次记录)
//
// 排除注释区(顶部的 R5 升级说明引用了 schema 名),故用 stripGoComments176。
func TestMigrate176_FastPathSchemaVersionCheck(t *testing.T) {
	s := stripGoComments176(t, filepath.Join(".", "migration_176_reconciliation_physical_mv.go"))

	wants := []string{
		"information_schema.columns", // 2.4 MV schema 版本校验
		"backfillOpsAssetPhysical",  // 双路径回填函数名
		"applogger Warnf",           // Type E 审计可见性(stripGoComments 把 applogger.Warnf 拆成两个 Ident)
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Fatalf("migration_176 must contain %q (C1 fast-path schema validation + dual-path backfill + Warnf audit)", w)
		}
	}
}

// TestMigrate176_TypeECleanupGate 源码断言:
//   - Type E 清理 SQL 之前存在 ops_asset_physical 的 EXISTS 探测(前置门控)
//
// 排除注释区后,确认 SELECT 1 FROM ops_asset_physical LIMIT 1 出现在非注释代码段。
func TestMigrate176_TypeECleanupGate(t *testing.T) {
	s := stripGoComments176(t, filepath.Join(".", "migration_176_reconciliation_physical_mv.go"))

	gate := "SELECT 1 FROM ops_asset_physical LIMIT 1"
	if !strings.Contains(s, gate) {
		t.Fatalf("migration_176 must contain ops_asset_physical EXISTS gate %q before destructive UPDATE (C1 Tampering mitigation T-62-01)", gate)
	}
}

// TestMigrate176_SqliteDoubleInvocation 幂等性:
// 在 sqlite 内存库上连续两次调用 Migrate176ReconciliationPhysicalMV,均返回 nil
// (方言守卫路径直接 return nil,无副作用,第二次调用同样安全)。
//
// 复现项目惯例:menu_grant_helpers_test.go + migration_202_port_write_audit_test.go
// 已用同模式验证 PG-only 迁移在 sqlite 上跳过。
func TestMigrate176_SqliteDoubleInvocation(t *testing.T) {
	db := freshSQLiteDB176(t)

	if err := Migrate176ReconciliationPhysicalMV(db); err != nil {
		t.Fatalf("first Migrate176ReconciliationPhysicalMV on sqlite must return nil (isPostgreSQL guard), got: %v", err)
	}
	// 第二次调用同样 nil(方言守卫,无副作用)
	if err := Migrate176ReconciliationPhysicalMV(db); err != nil {
		t.Fatalf("second Migrate176ReconciliationPhysicalMV on sqlite must return nil (idempotent by short-circuit), got: %v", err)
	}
}

// TestMigrate176_NoObsoleteDocstring 源码守卫:
// 顶部 docstring 不再含 "不主动 UPDATE"(已替换为"门控 UPDATE"说明)。
// grep 注释原文,允许 strip 后的非注释代码段不含该字符串,但禁止源码任意位置保留旧文本。
func TestMigrate176_NoObsoleteDocstring(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "migration_176_reconciliation_physical_mv.go"))
	if err != nil {
		t.Fatalf("read migration_176: %v", err)
	}
	if strings.Contains(string(src), "不主动 UPDATE") {
		t.Fatalf("migration_176 docstring must NOT contain obsolete phrase \"不主动 UPDATE\" (C1 fix: gate-controlled UPDATE replaces unconditional silent mark-as-resolved)")
	}
}

// ===== Task 2 — CDX-M-IDX 支撑索引 =====

// TestMigrate175_NicknamePartialIndex 源码断言:
//   - migration_175 含 "idx_sys_user_nickname" 索引
//   - 含 "ON sys_user (nickname)" 列定义
//   - 含 "WHERE deleted_at IS NULL" 部分索引谓词
//
// 排除注释区(stripGoComments176)以避免顶部 docstring 引用字段名导致误命中。
func TestMigrate175_NicknamePartialIndex(t *testing.T) {
	s := stripGoComments176(t, filepath.Join(".", "migration_175_reconciliation_physical_link.go"))

	wants := []string{
		"idx_sys_user_nickname",
		"ON sys_user (nickname)",
		"WHERE deleted_at IS NULL",
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Fatalf("migration_175 must contain %q (CDX-M-IDX: nickname partial index for reconciliation_user_lookup scalar subquery)", w)
		}
	}
}

// TestMigrate176_ResolvedAssetTimeIndex 源码断言:
//   - migration_176 含 "idx_recon_resolved_asset_time" 索引
//   - 含 "sys_data_reconciliation (asset_id, resolved_at DESC)" 列定义
func TestMigrate176_ResolvedAssetTimeIndex(t *testing.T) {
	s := stripGoComments176(t, filepath.Join(".", "migration_176_reconciliation_physical_mv.go"))

	wants := []string{
		"idx_recon_resolved_asset_time",
		"sys_data_reconciliation (asset_id, resolved_at DESC)",
	}
	for _, w := range wants {
		if !strings.Contains(s, w) {
			t.Fatalf("migration_176 must contain %q (CDX-M-IDX: partial index for MV last_resolved LATERAL subquery)", w)
		}
	}
}

// TestMigrate176_AllDDLIdempotent 源码守卫:
// 176 新增的支撑索引 DDL 使用 IF NOT EXISTS,可在每次启动幂等执行。
func TestMigrate176_AllDDLIdempotent(t *testing.T) {
	s := stripGoComments176(t, filepath.Join(".", "migration_176_reconciliation_physical_mv.go"))

	if !strings.Contains(s, "CREATE INDEX IF NOT EXISTS idx_recon_resolved_asset_time") {
		t.Fatalf("migration_176 must use CREATE INDEX IF NOT EXISTS for idx_recon_resolved_asset_time (CDX-M-IDX)")
	}
}

// TestMigrate175_SqliteDoubleInvocation 幂等性:
// 在 sqlite 内存库上连续两次调用 Migrate175ReconciliationPhysicalLink,均返回 nil。
func TestMigrate175_SqliteDoubleInvocation(t *testing.T) {
	db := freshSQLiteDB176(t)

	if err := Migrate175ReconciliationPhysicalLink(db); err != nil {
		t.Fatalf("first Migrate175ReconciliationPhysicalLink on sqlite must return nil (isPostgreSQL guard), got: %v", err)
	}
	if err := Migrate175ReconciliationPhysicalLink(db); err != nil {
		t.Fatalf("second Migrate175ReconciliationPhysicalLink on sqlite must return nil (idempotent by short-circuit), got: %v", err)
	}
}
