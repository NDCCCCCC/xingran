package query

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
)

// scopedTestModel 用 Tabler 断言验证 resolveTable 的 Model 分支
// （GORM Scopes 回调在 Parse 之前执行，Statement.Table/Schema 为空）。
type scopedTestModel struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func (scopedTestModel) TableName() string { return "scoped_test_table" }

func newScopedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("建 SQLite DryRun 库失败: %v", err)
	}
	return db
}

func deptCtx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

// ---------------------------------------------------------------------------
// T1：All / 未注入短路零回归（DryRun SQL 逐字比对 + 零额外 WHERE/EXISTS）
// ---------------------------------------------------------------------------

func TestScopedDB_NoInjection_AllFallback(t *testing.T) {
	db := newScopedTestDB(t)
	ctx := context.Background()

	baseline := db.WithContext(ctx).Model(&scopedTestModel{}).Find(&[]scopedTestModel{}).Statement.SQL.String()
	got := ScopedDB(ctx, db).Model(&scopedTestModel{}).Find(&[]scopedTestModel{}).Statement.SQL.String()

	if got != baseline {
		t.Fatalf("未注入 ctx 必须原样返回（零回归短路）：\n baseline=%s\n got     =%s", baseline, got)
	}
	if strings.Contains(strings.ToUpper(got), "EXISTS") || strings.Contains(strings.ToUpper(got), "WHERE") {
		t.Fatalf("未注入 ctx 不得产生任何额外 WHERE / EXISTS：%s", got)
	}
}

func TestScopedDB_AllTier_ZeroRegression(t *testing.T) {
	db := newScopedTestDB(t)
	ctx := deptCtx(datascope.AllScope("u1"))

	baseline := db.WithContext(ctx).Model(&scopedTestModel{}).Find(&[]scopedTestModel{}).Statement.SQL.String()
	got := ScopedDB(ctx, db).Model(&scopedTestModel{}).Find(&[]scopedTestModel{}).Statement.SQL.String()

	if got != baseline {
		t.Fatalf("All 档必须原样返回（零回归短路）：\n baseline=%s\n got     =%s", baseline, got)
	}
}

// ---------------------------------------------------------------------------
// T1：注册表谓词生效 + 双方言形态 + Count/Find 不重复叠加 + 未注册表原样返回
// ---------------------------------------------------------------------------

func TestScopedDB_RegistryApplied(t *testing.T) {
	resetRegistryForTest()
	t.Cleanup(resetRegistryForTest)
	if err := Register("scoped_test_table", Entry{
		Predicate: func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB {
			return tx.Where("CAST(org_id AS TEXT) IN ?", r.DeptIDs)
		},
		CacheKey: func(context.Context) string { return "d-test" },
	}); err != nil {
		t.Fatalf("Register 失败: %v", err)
	}

	db := newScopedTestDB(t)
	ctx := deptCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}})
	sql := ScopedDB(ctx, db).Model(&scopedTestModel{}).Find(&[]scopedTestModel{}).Statement.SQL.String()

	if !strings.Contains(sql, "CAST(org_id AS TEXT) IN") {
		t.Fatalf("注册表谓词未生效: %s", sql)
	}
	if strings.Contains(sql, "::") || strings.Contains(strings.ToUpper(sql), "ILIKE") {
		t.Fatalf("谓词必须双方言（禁 :: / ILIKE）: %s", sql)
	}
}

func TestScopedDB_NoDuplicatePredicate_CountAndFind(t *testing.T) {
	resetRegistryForTest()
	t.Cleanup(resetRegistryForTest)
	var calls int
	var mu sync.Mutex
	if err := Register("scoped_test_table", Entry{
		Predicate: func(tx *gorm.DB, r datascope.ScopeResult) *gorm.DB {
			mu.Lock()
			calls++
			mu.Unlock()
			return tx.Where("visible_flag = 1")
		},
	}); err != nil {
		t.Fatalf("Register 失败: %v", err)
	}

	db := newScopedTestDB(t)
	ctx := deptCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}})
	q := ScopedDB(ctx, db).Model(&scopedTestModel{})

	var total int64
	countSQL := q.Count(&total).Statement.SQL.String()
	findSQL := q.Find(&[]scopedTestModel{}).Statement.SQL.String()

	if n := strings.Count(countSQL, "visible_flag = 1"); n != 1 {
		t.Fatalf("Count 谓词应恰好 1 次，实际 %d 次: %s", n, countSQL)
	}
	if n := strings.Count(findSQL, "visible_flag = 1"); n != 1 {
		t.Fatalf("Find 谓词应恰好 1 次，实际 %d 次: %s", n, findSQL)
	}
	// GORM executeScopes 会在 Count 时消费并清空 Statement.scopes，谓词
	// 固化进 Clauses；Find 复用同一链不再二次应用 → 恰好 1 次调用，
	// 且两条 SQL 各含且仅含 1 次谓词（同源不重复叠加的结构性保证）。
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("共享链 Count+Find 应只应用一次 scopes（谓词固化进 Clauses 不重复叠加），实际 %d 次", calls)
	}
	// Count 与 Find 同源：谓词一致，total 与明细行数不会分叉。
	if !strings.Contains(countSQL, "count(*)") || !strings.Contains(findSQL, "scoped_test_table") {
		t.Fatalf("Count/Find 起链异常: count=%s find=%s", countSQL, findSQL)
	}
}

func TestScopedDB_UnknownTable_NoPredicate(t *testing.T) {
	resetRegistryForTest()
	t.Cleanup(resetRegistryForTest)
	db := newScopedTestDB(t)
	ctx := deptCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}})

	baseline := db.WithContext(ctx).Table("unregistered_table").Find(&[]map[string]any{}).Statement.SQL.String()
	got := ScopedDB(ctx, db).Table("unregistered_table").Find(&[]map[string]any{}).Statement.SQL.String()
	if got != baseline {
		t.Fatalf("未注册表应原样返回（不静默放行全库，由静态断言兜住）：\n baseline=%s\n got=%s", baseline, got)
	}
}

// ---------------------------------------------------------------------------
// T1：静态断言（分层红线 + All 兜底短路 + 单一真实源）
// ---------------------------------------------------------------------------

func TestScopedDB_StaticLayerAndFallbackContract(t *testing.T) {
	src, err := os.ReadFile("scoped_db.go")
	if err != nil {
		t.Fatalf("读 scoped_db.go 失败: %v", err)
	}
	text := string(src)
	if !strings.Contains(text, "datascope.FromContext") {
		t.Fatal("ScopedDB 必须通过 datascope.FromContext 取 ScopeResult（单一真实源）")
	}
	if !strings.Contains(text, "!ok || r.All") {
		t.Fatal("ScopedDB 必须保留 `!ok || r.All` 零回归短路")
	}
	// 只扫描「代码行」：剔除 // 行注释与包裹注释后判定，避免误伤契约说明文字。
	code := stripGoComments(text)
	if strings.Contains(code, "COALESCE(") || strings.Contains(code, "NULLIF(") || strings.Contains(code, "1=0") {
		t.Fatal("ScopedDB 禁止内联复制 SQL 片段（COALESCE/NULLIF/1=0）")
	}
	if strings.Contains(code, "internal/services/") || strings.Contains(code, "internal/models/") {
		t.Fatal("pkg/query 禁止 import internal/**（回调注册制分层红线）")
	}

	regSrc, err := os.ReadFile("scoped_registry.go")
	if err != nil {
		t.Fatalf("读 scoped_registry.go 失败: %v", err)
	}
	reg := stripGoComments(string(regSrc))
	if strings.Contains(reg, "internal/services/") || strings.Contains(reg, "internal/models/") {
		t.Fatal("scoped_registry.go 禁止 import internal/**")
	}
	if !strings.Contains(string(regSrc), "datascope.ScopeResult") {
		t.Fatal("注册表必须消费 pkg/datascope 的 ScopeResult（单一真实源）")
	}
}

// stripGoComments 剔除全行 // 注释与 /* */ 块注释，供静态断言只检查真实代码行。
func stripGoComments(src string) string {
	var b strings.Builder
	inBlock := false
	for _, line := range strings.Split(src, "\n") {
		if inBlock {
			if i := strings.Index(line, "*/"); i >= 0 {
				inBlock = false
				b.WriteString(line[i+2:])
				b.WriteString("\n")
			}
			continue
		}
		if i := strings.Index(line, "/*"); i >= 0 {
			b.WriteString(line[:i])
			if j := strings.Index(line[i+2:], "*/"); j >= 0 {
				b.WriteString(line[i+2+j+2:])
			} else {
				inBlock = true
			}
			b.WriteString("\n")
			continue
		}
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func TestScopedDB_ScopeCacheKeyFallback(t *testing.T) {
	resetRegistryForTest()
	t.Cleanup(resetRegistryForTest)
	if got := ScopeCacheKey(context.Background()); got != "all" {
		t.Fatalf("未注入 ctx 的缓存 key 应为 all，实际 %q", got)
	}
	if got := ScopeCacheKey(deptCtx(datascope.AllScope("u1"))); got != "all" {
		t.Fatalf("All 档的缓存 key 应为 all，实际 %q", got)
	}
	if err := Register("scoped_test_table", Entry{
		Predicate: func(tx *gorm.DB, _ datascope.ScopeResult) *gorm.DB { return tx },
		CacheKey:  func(context.Context) string { return "d-abc" },
	}); err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	if got := ScopeCacheKey(deptCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}})); got != "d-abc" {
		t.Fatalf("非 All 档应委托注册表 CacheKey 回调，实际 %q", got)
	}
}
