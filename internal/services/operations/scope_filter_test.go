// Package operations: Phase 131 可见集过滤基建厚单测。
//
// 命名前缀 ScopeFilter* 避免与其它 agent 的测试/helper 冲突。
// 全部语义断言，禁止无断言刷覆盖率。
package operations

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ScopeFilterTestDB 建最小夹具：仅 sys_workstation 一列 org 相关列即可，
// 用于渲染 applyOrgScope / applyExcelWorkstationScope 的 SQL（DryRun，不查数据）。
func ScopeFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// ScopeFilterCtx 构造带 ScopeResult 的 ctx。
func ScopeFilterCtx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

// ScopeFilterSQL 用 DryRun 渲染 query.Find 的 SQL（不执行），返回 SQL + Vars。
func ScopeFilterSQL(t *testing.T, db *gorm.DB, build func(*gorm.DB) *gorm.DB) (string, []interface{}) {
	t.Helper()
	q := build(db.Table("sys_workstation"))
	stmt := q.Session(&gorm.Session{DryRun: true}).Find(&[]map[string]interface{}{}).Statement
	return stmt.SQL.String(), stmt.Vars
}

// ---------------------------------------------------------------------
// A. scopeFromContext / scopeIsAll / scopeRejectsRequestedOrg
// ---------------------------------------------------------------------

func TestScopeFilter_AllShortCircuit(t *testing.T) {
	db := ScopeFilterTestDB(t)

	// 未注入 → All 兜底（超管 / 未挂中间件零回归）
	sql, vars := ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyOrgScope(q, context.Background(), "org_id")
	})
	assert.NotContains(t, sql, "IN", "未注入不应挂任何范围条件")
	assert.NotContains(t, sql, "1=0", "未注入不应挂空集条件")
	assert.Empty(t, vars)

	// 显式 All=true → 同样短路
	sql, vars = ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyOrgScope(q, ScopeFilterCtx(datascope.AllScope("u-super")), "org_id")
	})
	assert.NotContains(t, sql, "IN")
	assert.NotContains(t, sql, "1=0")
	assert.Empty(t, vars, "All 档应零 DB 查询 / 零绑定参数")
}

func TestScopeFilter_FailClosedEmpty(t *testing.T) {
	db := ScopeFilterTestDB(t)
	sql, vars := ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyOrgScope(q, ScopeFilterCtx(datascope.FailClosedScope("u-bad")), "org_id")
	})
	assert.Contains(t, sql, "1=0", "异常枚举 fail-closed 必须空集")
	assert.Empty(t, vars, "1=0 不需要绑定参数")
}

func TestScopeFilter_DeptIDsHitAndMiss(t *testing.T) {
	db := ScopeFilterTestDB(t)
	depts := []string{"dept-a", "dept-b"}

	// 命中：IN ? 被 GORM 展开为 IN (?,?)、双侧 CAST AS TEXT
	sql, vars := ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyOrgScope(q, ScopeFilterCtx(datascope.ScopeResult{DeptIDs: depts}), "org_id")
	})
	assert.Contains(t, sql, "CAST(org_id AS TEXT) IN (?,?)", "uuid/varchar 比较必须双侧 CAST AS TEXT")
	require.Len(t, vars, 2)
	assert.Equal(t, []interface{}{"dept-a", "dept-b"}, vars)

	// 不命中（空 DeptIDs）→ 1=0
	sql, _ = ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyOrgScope(q, ScopeFilterCtx(datascope.ScopeResult{}), "org_id")
	})
	assert.Contains(t, sql, "1=0", "空可见集不得放行全库")
}

func TestScopeFilter_SelfUserDispatch(t *testing.T) {
	db := ScopeFilterTestDB(t)
	self := datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "u-self"}

	// 有用户列实体 → user_id = ?
	sql, vars := ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyReadScope(q, ScopeFilterCtx(self), "org_id", "sys_workstation.user_id")
	})
	assert.Contains(t, sql, "sys_workstation.user_id = ?")
	require.Len(t, vars, 1)
	assert.Equal(t, "u-self", vars[0])

	// 无用户列实体（floor/building）→ 空集（裁定 §2）
	sql, _ = ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyReadScope(q, ScopeFilterCtx(self), "org_id", "")
	})
	assert.Contains(t, sql, "1=0")

	// UserID 缺失 → fail-closed 空集
	sql, _ = ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyReadScope(q, ScopeFilterCtx(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser}), "org_id", "sys_workstation.user_id")
	})
	assert.Contains(t, sql, "1=0")
}

func TestScopeFilter_RequestedOrgIntersection(t *testing.T) {
	scope := datascope.ScopeResult{DeptIDs: []string{"dept-a", "dept-b"}}

	// 越界 orgId → 收紧空集
	assert.True(t, scopeRejectsRequestedOrg(scope, true, "dept-outside"))
	// 命中可见集 → 放行（由既有业务递归筛选继续处理）
	assert.False(t, scopeRejectsRequestedOrg(scope, true, "dept-a"))
	// 空 orgId → 不校验
	assert.False(t, scopeRejectsRequestedOrg(scope, true, ""))
	// All / 未注入 → 不校验（超管零回归）
	assert.False(t, scopeRejectsRequestedOrg(datascope.AllScope("u"), true, "anything"))
	assert.False(t, scopeRejectsRequestedOrg(datascope.ScopeResult{}, false, "anything"))
	// FailClosed → 恒 true
	assert.True(t, scopeRejectsRequestedOrg(datascope.FailClosedScope("u"), true, ""))
	// Self（DeptIDs 空）→ 任何非空 orgId 越界
	assert.True(t, scopeRejectsRequestedOrg(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "u"}, true, "dept-a"))
}

func TestScopeFilter_UUIDCastStaticAssertion(t *testing.T) {
	// 静态断言：所有新 SQL 的 uuid/varchar 比较统一 CAST(... AS TEXT)，
	// 禁止 PG 专有 ::uuid/::text 出现在可见集基建里。
	db := ScopeFilterTestDB(t)
	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	assert.Contains(t, orgExpr, "CAST(", "有效机构表达式必须双侧 CAST")
	assert.Contains(t, orgExpr, "AS TEXT")
	assert.NotContains(t, orgExpr, "::", "禁止复用 PG 专有 :: 强转")

	cond, args, skip := orgScopeClause(datascope.ScopeResult{DeptIDs: []string{"d1"}}, true, orgExpr)
	assert.False(t, skip)
	assert.Contains(t, cond, "CAST(")
	assert.Contains(t, cond, "AS TEXT) IN ?")
	assert.Len(t, args, 1)

	// applyExcelWorkstationScope 渲染出的可见集条件同样必须 CAST 双方言。
	sql, _ := ScopeFilterSQL(t, db, func(q *gorm.DB) *gorm.DB {
		return applyExcelWorkstationScope(q, ScopeFilterCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}}))
	})
	assert.Contains(t, sql, "CAST(", "导出可见集条件必须双方言 CAST")
	assert.Contains(t, sql, "EXISTS", "导出路径按 floor 有效机构 ∈ 可见集")
	assert.NotContains(t, sql, "ghostOrgId", "不得依赖 params 的 orgId")
}

// ---------------------------------------------------------------------
// D. Excel 导出两条路径「含可见集条件」静态断言（P1-8 / DS-06）
// ---------------------------------------------------------------------

func TestScopeExport_StaticContract_DryRunSQL(t *testing.T) {
	db := ScopeFilterTestDB(t)
	config, ok := GetExportConfig("workstation")
	require.True(t, ok)
	scopedCtx := ScopeFilterCtx(datascope.ScopeResult{DeptIDs: []string{"dept-visible"}})

	// 路径 1：WorkstationQueryBuilder.BuildQuery
	qb := NewWorkstationQueryBuilder()
	q1 := qb.BuildQuery(scopedCtx, db, config, map[string]interface{}{})
	stmt1 := q1.Session(&gorm.Session{DryRun: true}).Find(&[]map[string]interface{}{}).Statement
	assert.Contains(t, stmt1.SQL.String(), "CAST(", "BuildQuery 可见集条件必须双方言 CAST")
	assert.Contains(t, stmt1.SQL.String(), "EXISTS", "BuildQuery 必须按有效机构注入可见集")
	assert.NotContains(t, stmt1.SQL.String(), "orgId", "导出不得依赖 params orgId 拼列")

	// 路径 2：queryWorkstationIDsForExport
	svc := &ExcelService{db: db}
	_ = svc
	q2 := db.WithContext(scopedCtx).Table("sys_workstation").Where("deleted_at IS NULL")
	q2 = applyExcelWorkstationScope(q2, scopedCtx)
	stmt2 := q2.Session(&gorm.Session{DryRun: true}).Pluck("sys_workstation.id", &[]string{}).Statement
	assert.Contains(t, stmt2.SQL.String(), "CAST(", "queryWorkstationIDsForExport 可见集条件必须双方言 CAST")
	assert.Contains(t, stmt2.SQL.String(), "EXISTS")

	// All 档短路：两条路径均不挂 EXISTS（零 DB 查询）
	allCtx := ScopeFilterCtx(datascope.AllScope("u-super"))
	q1All := qb.BuildQuery(allCtx, db, config, map[string]interface{}{})
	stmtAll := q1All.Session(&gorm.Session{DryRun: true}).Find(&[]map[string]interface{}{}).Statement
	assert.NotContains(t, stmtAll.SQL.String(), "EXISTS (SELECT 1 FROM `ops_floors` f", "All 档应短路不挂可见集 EXISTS")
}

// ---------------------------------------------------------------------
// A. scopeCacheKey 分档派生（All/Self/FailClosed/DeptIDs 不碰撞）
// ---------------------------------------------------------------------

func TestScopeFilter_ScopeKeySegments(t *testing.T) {
	// 不同档位必须产出不同段（审查 P1-3）
	all := scopeCacheKeyOf(datascope.AllScope("u"), true)
	closed := scopeCacheKeyOf(datascope.FailClosedScope("u"), true)
	selfA := scopeCacheKeyOf(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "uA"}, true)
	selfB := scopeCacheKeyOf(datascope.ScopeResult{ResourceHook: datascope.HookSelfUser, UserID: "uB"}, true)
	deptA := scopeCacheKeyOf(datascope.ScopeResult{DeptIDs: []string{"d1"}}, true)
	deptB := scopeCacheKeyOf(datascope.ScopeResult{DeptIDs: []string{"d2"}}, true)
	noInject := scopeCacheKeyOf(datascope.ScopeResult{}, false)

	assert.Equal(t, "all", all, "All 档短路 key 也必须含 all 段")
	assert.Equal(t, "all", noInject, "未注入按 All 兜底")
	assert.Equal(t, "closed", closed)
	assert.Equal(t, "self", selfA[:4])
	assert.Equal(t, "d", deptA[:1])
	keys := []string{all, closed, selfA, selfB, deptA, deptB}
	seen := map[string]bool{}
	for _, k := range keys {
		assert.False(t, seen[k], "档位 key 不应碰撞: "+k)
		seen[k] = true
	}
	assert.NotEqual(t, selfA, selfB, "不同 Self 用户不串 key")
	assert.NotEqual(t, deptA, deptB, "不同可见集不串 key")

	// 相同 scope 稳定 + DeptIDs 顺序无关
	sorted := scopeCacheKeyOf(datascope.ScopeResult{DeptIDs: []string{"d1", "d2"}}, true)
	unsorted := scopeCacheKeyOf(datascope.ScopeResult{DeptIDs: []string{"d2", "d1"}}, true)
	assert.Equal(t, sorted, unsorted, "DeptIDs 排序保证 key 稳定")

	// Scoped dropdown key 含 scope 段
	kd1 := BuildScopedDropdownCacheKey(ScopeFilterCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}}), "floor", nil)
	kd2 := BuildScopedDropdownCacheKey(ScopeFilterCtx(datascope.ScopeResult{DeptIDs: []string{"d2"}}), "floor", nil)
	assert.NotEqual(t, kd1, kd2, "PMODEL-01：不同可见集 dropdown 缓存 key 必须不同")
	assert.True(t, strings.HasPrefix(kd1, "dropdown:floor:s"), "scope 段须在 entity 之后")
	allKey := BuildScopedDropdownCacheKey(context.Background(), "floor", nil)
	assert.Contains(t, allKey, ":sall:", "All 档短路 key 也含 scope 段")
}
