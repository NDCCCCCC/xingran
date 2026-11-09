// Package operations: Phase 151 / SIDECHANNEL-04 —— COUNT/SUM 与明细同源恒等式。
//
// 复用 150 的差分攻击思路（两相邻 scope / N=5·M=2 与 N=9·M=2），SQLite 真跑
// + DryRun 静态断言，零新依赖；TEST_PG_DSN 仍为 DIALECT-04③ 前置依赖，本期
// 不建 services: postgres、不伪造 PG 双跑（t.Skip 可观测见 pg_contract_test.go）。
package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	homologyDeptA = "aaaaaaaa-0000-0000-0000-000000000001"
	homologyDeptB = "bbbbbbbb-0000-0000-0000-000000000002"
)

// homologyFixture 构造「全库 N 行 / 可见 M 行」最小夹具（ops_asset 单表，
// dept_id 直挂，避免维护 ops_* 全量 schema）。M 行 dept_id=deptA，其余 dept_id=deptB。
func homologyFixture(t *testing.T, visible, outOfScope int) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ops_asset (
		id TEXT PRIMARY KEY, dept_id TEXT, name TEXT, deleted_at DATETIME
	)`).Error)
	for i := 0; i < visible; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_asset(id,dept_id,name) VALUES(?,?,?)`,
			fmt.Sprintf("v-%03d", i), homologyDeptA, fmt.Sprintf("可见资产%d", i)).Error)
	}
	for i := 0; i < outOfScope; i++ {
		require.NoError(t, db.Exec(`INSERT INTO ops_asset(id,dept_id,name) VALUES(?,?,?)`,
			fmt.Sprintf("o-%03d", i), homologyDeptB, fmt.Sprintf("越界资产%d", i)).Error)
	}
	return db
}

func homologyCtx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

// homologyCountAndList 用 ScopedQuery 起链，同一条链上先后 Count 与 Find，
// 返回 (total, 明细行数)。SQL 形态断言另用 homologyDryRunSQL（DryRun 库）。
func homologyCountAndList(t *testing.T, db *gorm.DB, ctx context.Context) (int64, int) {
	t.Helper()
	q := ScopedQuery(ctx, db).Table("ops_asset")
	var total int64
	require.NoError(t, q.Count(&total).Error)
	var rows []map[string]any
	require.NoError(t, q.Find(&rows).Error)
	return total, len(rows)
}

// homologyDryRunSQL 用 DryRun 库捕获 ScopedQuery 起链后的 Count / Find SQL
// （真跑时 GORM 不保留 Statement.SQL，故 SQL 形态断言单独用 DryRun）。
func homologyDryRunSQL(t *testing.T, ctx context.Context) (string, string) {
	t.Helper()
	query.ResetRegistryForTest()
	require.NoError(t, SetupScopedDBRegistry())
	dry, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)
	q := ScopedQuery(ctx, dry).Table("ops_asset")
	var total int64
	countTx := q.Count(&total)
	findTx := q.Find(&[]map[string]any{})
	return countTx.Statement.SQL.String(), findTx.Statement.SQL.String()
}

// TestHomology_CountEqualsList_DeptTier 同源恒等式：同一 ScopeResult 下
// COUNT(*) == len(List) == M（8 个 /statistics + 分页 total 的结构性守护）。
func TestHomology_CountEqualsList_DeptTier(t *testing.T) {
	db := homologyFixture(t, 2, 7) // N=9, M=2
	ctx := homologyCtx(datascope.ScopeResult{DeptIDs: []string{homologyDeptA}})
	query.ResetRegistryForTest()
	require.NoError(t, SetupScopedDBRegistry())

	total, n := homologyCountAndList(t, db, ctx)
	assert.Equal(t, int64(2), total, "COUNT 必须只数可见集 M 行")
	assert.Equal(t, 2, n, "明细 List 必须只返回可见集 M 行")
	assert.Equal(t, int64(n), total, "COUNT 与 len(List) 同源恒等")
	countSQL, findSQL := homologyDryRunSQL(t, ctx)
	assert.Contains(t, strings.ToUpper(countSQL), "DEPT_ID", "Count SQL 必须带可见集谓词")
	assert.Contains(t, strings.ToUpper(findSQL), "DEPT_ID", "Find SQL 必须带可见集谓词")
}

// TestHomology_DiffInvariant_NChangesMFixed 差分不变式：改变 N 保持 M 不变 →
// 所有数值字段（total / count / 数组长度）逐字节相等（复用 150 两相邻 scope 思路）。
func TestHomology_DiffInvariant_NChangesMFixed(t *testing.T) {
	query.ResetRegistryForTest()
	require.NoError(t, SetupScopedDBRegistry())

	// 两相邻 scope（Dept / DeptChild，后者含无数据的子部门 → M 不变）
	scopes := []datascope.ScopeResult{
		{DeptIDs: []string{homologyDeptA}},
		{DeptIDs: []string{homologyDeptA, "cccccccc-0000-0000-0000-000000000003"}},
	}
	// 两组 N 不同但 M 相同
	fixtures := []struct{ visible, outOfScope int }{{2, 3}, {2, 7}}

	type snapshot struct {
		Total int64 `json:"total"`
		Count int    `json:"count"`
	}
	var byScope [2][]snapshot
	for si, s := range scopes {
		for _, f := range fixtures {
			db := homologyFixture(t, f.visible, f.outOfScope)
			total, n := homologyCountAndList(t, db, homologyCtx(s))
			byScope[si] = append(byScope[si], snapshot{Total: total, Count: n})
		}
	}
	for si := range scopes {
		a, _ := json.Marshal(byScope[si][0])
		b, _ := json.Marshal(byScope[si][1])
		assert.Equal(t, string(a), string(b),
			"scope#%d：N=5/M=2 与 N=9/M=2 的数值字段必须逐字节相等（不可枚举 N−M）", si)
		assert.Equal(t, int64(2), byScope[si][0].Total)
	}
}

// TestHomology_NoGlobalCountChain 静态 + 行为双断言：DryRun SQL 中不存在
// 「不带可见集谓词的 COUNT(*)」子查询（防 150 D-3 全库口径复活）。
func TestHomology_NoGlobalCountChain(t *testing.T) {
	query.ResetRegistryForTest()
	require.NoError(t, SetupScopedDBRegistry())
	ctx := homologyCtx(datascope.ScopeResult{DeptIDs: []string{homologyDeptA}})
	countSQL, findSQL := homologyDryRunSQL(t, ctx)
	// 恒等：Count 与 Find 的 WHERE 谓词一致（同源结构性保证）
	upperC := strings.ToUpper(countSQL)
	upperF := strings.ToUpper(findSQL)
	assert.Contains(t, upperC, "COUNT(*)", "Count 必须是聚合查询")
	assert.Contains(t, upperC, "DEPT_ID", "Count 必须带可见集谓词（非全库 COUNT）")
	assert.Contains(t, upperF, "DEPT_ID", "Find 必须带可见集谓词")
	// 源码静态：只检查 150 已删的「全库 countQ 死代码」token（与 150 的
	// TestScopeWriteImport_StaticContract 同口径；不误伤 8 个 /statistics
	// 里合法的 COUNT(*) AS total 同源聚合）。
	ws, err := os.ReadFile("workstation_service.go")
	require.NoError(t, err)
	assert.NotContains(t, string(ws), `countQ := s.db.WithContext(ctx).Table("sys_workstation")`,
		"150 D-3 全库 countQ 不得复活")
	assert.NotContains(t, string(ws), "countQ.Scan", "150 D-3 全库 countQ 扫描不得复活")
}

// TestHomology_AllTierZeroRegression All 档 / 未注入 ctx 零回归：原样返回，
// COUNT 与 List 均为全库 N 行，DryRun SQL 与改造前逐字一致（零额外 WHERE）。
func TestHomology_AllTierZeroRegression(t *testing.T) {
	query.ResetRegistryForTest()
	require.NoError(t, SetupScopedDBRegistry())

	// 真跑：数值零回归
	db := homologyFixture(t, 2, 7) // N=9
	// DryRun：SQL 零漂移（真跑时 GORM 不保留 Statement.SQL，必须用 DryRun）
	dry, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)

	for _, ctx := range []context.Context{
		context.Background(),
		homologyCtx(datascope.AllScope("u1")),
	} {
		total, n := homologyCountAndList(t, db, ctx)
		assert.Equal(t, int64(9), total, "All/未注入档按全库 N 行零回归")
		assert.Equal(t, 9, n)

		base := dry.WithContext(ctx).Table("ops_asset")
		var baseTotal int64
		baseCountSQL := base.Count(&baseTotal).Statement.SQL.String()
		baseFindSQL := base.Find(&[]map[string]any{}).Statement.SQL.String()

		q := ScopedQuery(ctx, dry).Table("ops_asset")
		var t2 int64
		countSQL := q.Count(&t2).Statement.SQL.String()
		findSQL := q.Find(&[]map[string]any{}).Statement.SQL.String()

		assert.Equal(t, baseCountSQL, countSQL, "All/未注入档 Count SQL 必须零漂移")
		assert.Equal(t, baseFindSQL, findSQL, "All/未注入档 Find SQL 必须零漂移")
		assert.NotContains(t, strings.ToUpper(findSQL), "DEPT_ID", "All/未注入档不得挂可见集谓词")
	}
}
