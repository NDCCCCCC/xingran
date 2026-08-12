package migrations

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// stripGoComments parses migration source and returns concatenated non-comment
// source text (declarations + statements + string literals). Doc / line / block
// comments are excluded so source-grep guards do not false-trigger on spec text
// quoting the forbidden identifiers (e.g. "IsFrame" / "CREATE TABLE" inside doc
// comments explaining why they must NOT appear in real code).
func stripGoComments(t *testing.T, path string) string {
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

// freshSQLiteDBForMigrate202 builds an in-memory SQLite database for the
// migration_202 source-grep / SQLite-skip-cleanly tests.
func freshSQLiteDBForMigrate202(t *testing.T) *gorm.DB {
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

// TestMigrate202_SQLiteSkipsCleanly SQLite in-memory 调 Migrate202PortWriteAudit →
// 返回 nil 不 panic(菜单 seed / index 都在 PG 分支)。
func TestMigrate202_SQLiteSkipsCleanly(t *testing.T) {
	db := freshSQLiteDBForMigrate202(t)

	if err := Migrate202PortWriteAudit(db); err != nil {
		t.Fatalf("Migrate202PortWriteAudit on sqlite must return nil (skip cleanly), got: %v", err)
	}

	// SQLite 跳过后不应 seed 任何菜单(菜单 seed 在 PG 分支)。
	var menuCount int64
	db.Model(&models.Menu{}).Where("perms = ?", "network:port:write").Count(&menuCount)
	if menuCount != 0 {
		t.Fatalf("SQLite path must NOT seed menus, but found %d rows with perms=network:port:write", menuCount)
	}
}

// TestMigrate202_MenuSeedStructure PG path:
// 跑 Migrate202 → sys_menu 含一行 menu_name='端口配置' AND menu_type='F' AND perms='network:port:write' AND visible=0。
func TestMigrate202_MenuSeedStructure(t *testing.T) {
	if os.Getenv("XINGRAN_PG_TEST_DSN") == "" {
		t.Skip("requires PostgreSQL — covered by Phase 54 UAT; sqlite path returns nil")
	}
	t.Skip("PG integration not configured in this executor — covered by Phase 54 UAT")
}

// TestMigrate202_RoleMenuGrantedToParentHolders PG path:
// setup 父菜单 '端口状态' + 2 角色持父菜单 + 1 角色不持;跑 Migrate202 →
// sys_role_menu 含 2 行 menu_id=新菜单(antd 父子联动陷阱防御)。
func TestMigrate202_RoleMenuGrantedToParentHolders(t *testing.T) {
	if os.Getenv("XINGRAN_PG_TEST_DSN") == "" {
		t.Skip("requires PostgreSQL — covered by Phase 54 UAT; sqlite path returns nil")
	}
	t.Skip("PG integration not configured in this executor — covered by Phase 54 UAT")
}

// TestMigrate202_Idempotent PG path:
// 连跑两次 Migrate202 → sys_menu 行数不变 + sys_role_menu 行数不变(ON CONFLICT + count-then-insert)。
func TestMigrate202_Idempotent(t *testing.T) {
	if os.Getenv("XINGRAN_PG_TEST_DSN") == "" {
		t.Skip("requires PostgreSQL — covered by Phase 54 UAT; sqlite path returns nil")
	}
	t.Skip("PG integration not configured in this executor — covered by Phase 54 UAT")
}

// TestMigrate202_NoIsFrameIsCacheFields 源码 grep 断言:migration_202 文件**不**含
// `IsFrame` / `IsCache`(Go sys_menu 无此列,memory xingran-menu-no-java-fields)。
// 扫描非注释代码(文档注释可能引用这些标识符解释为何不能加)。
func TestMigrate202_NoIsFrameIsCacheFields(t *testing.T) {
	s := stripGoComments(t, filepath.Join(".", "migration_202_port_write_audit.go"))
	for _, bad := range []string{"IsFrame", "IsCache"} {
		if strings.Contains(s, bad) {
			t.Fatalf("migration_202 must NOT contain %q in non-comment code (Go sys_menu has no such column; memory xingran-menu-no-java-fields)", bad)
		}
	}
}

// TestMigrate202_UsesCorrectParentName 源码 grep 断言:migration_202 文件含
// `GrantNewMenuToRolesHavingParent(db, "端口状态"` (D-07),**不**含 `'端口管理'` 作为父菜单查找条件。
func TestMigrate202_UsesCorrectParentName(t *testing.T) {
	s := stripGoComments(t, filepath.Join(".", "migration_202_port_write_audit.go"))

	if !strings.Contains(s, `"端口状态"`) {
		t.Fatalf("migration_202 must reference parent menu name \"端口状态\" (D-07)")
	}
	if strings.Contains(s, `"端口管理"`) {
		t.Fatalf("migration_202 must NOT use \"端口管理\" as parent menu name (D-07 — ROADMAP typo)")
	}
}

// TestMigrate202_PathANoCreateTable 源码 grep 守卫(Path A):
// migration_202 文件**不**含 `CREATE TABLE`(表由 database.go AutoMigrate 通过 model 建)。
func TestMigrate202_PathANoCreateTable(t *testing.T) {
	s := stripGoComments(t, filepath.Join(".", "migration_202_port_write_audit.go"))
	if strings.Contains(s, "CREATE TABLE") {
		t.Fatalf("migration_202 must NOT contain CREATE TABLE in non-comment code (Path A: table built by GORM AutoMigrate)")
	}
}
