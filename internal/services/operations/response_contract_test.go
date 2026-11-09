// Package operations: Phase 151 / T5 响应契约层静态断言（防 150 成果复活）。
//
// 全部静态断言恒跑、零 DSN 依赖；复用 150 的差分攻击测试思路，不删改
// 150 的任何测试 / 响应逻辑。TEST_PG_DSN unset → PG contract t.Skip 可观测。
package operations

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
)

func responseContractRead(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	require.NoError(t, err, "读取 %s", name)
	return string(b)
}

// TestResponseContract_NoDerivedScale：任何端点响应体不含「全库规模 − 可见规模」
// 派生数值（6 个被删字段名 0 命中 / ScopeOutOfScope json:"-" / 越界分支不复活 /
// 响应体键不含全库规模语义）。
func TestResponseContract_NoDerivedScale(t *testing.T) {
	ws := responseContractRead(t, "workstation_service.go")
	excel := responseContractRead(t, "excel_service.go")
	scopeWrite := responseContractRead(t, "scope_write.go")

	// 6 个 150 已删规模字段：响应结构体里 0 命中（只查字段声明形态，不误伤注释）
	for _, field := range []string{
		"ScopeTotal", "ScopeVisible", "ScopeOutOfScope", "ScopeDeptTotal",
		"SkippedNoUser", "SkippedBadUserDept",
	} {
		assert.NotContains(t, ws, field+" int", "BackfillDeptResult 不得复活规模字段 %q", field)
		assert.NotContains(t, ws, field+" int64", "BackfillDeptResult 不得复活规模字段 %q", field)
	}
	// ImportResult.ScopeOutOfScope 保持 json:"-"（仅供 operlog，永不进 HTTP）
	assert.Contains(t, excel, "ScopeOutOfScope int", "字段名/类型不变（operlog 仍读）")
	assert.Contains(t, excel, `json:"-"`, "ScopeOutOfScope 必须为 json:\"-\"")
	assert.NotContains(t, excel, `json:"scopeOutOfScope`, "不得回显 scopeOutOfScope 响应键")
	// 越界分支不复活：不 append Errors、不 Failed++
	assert.NotContains(t, scopeWrite, "result.Errors = append", "越界行静默跳过不得复活（D-6）")
	// 响应体键不含「全库规模 − 可见规模」语义（只查 json tag / gin.H 键形态，
	// 不误伤 150 的说明性注释文字）。
	for _, forbidden := range []string{
		`json:"scopeTotal`, `json:"scopeVisible`, `json:"scopeOutOfScope`,
		`json:"scopeDeptTotal`, `json:"skippedNoUser`, `json:"skippedBadUserDept`,
		`"outOfScope":`, "总数-可见", "总数 - 可见",
	} {
		assert.NotContains(t, ws, forbidden, "响应体构造不得含派生规模键 %q", forbidden)
	}
	// 合法聚合（8 个 /statistics 同源 COUNT）不受影响：只锁死 150 的双口径死代码
	assert.NotContains(t, ws, `countQ := s.db.WithContext(ctx).Table("sys_workstation")`,
		"150 D-3 全库 countQ 不得复活")
}

// TestResponseContract_NoRevival：把 150 的 5 个差分攻击测试 + 静态契约测试
// 以测试名清单静态登记，缺一即红（防执行期误删）。
func TestResponseContract_NoRevival(t *testing.T) {
	src := responseContractRead(t, "sidechannel_diff_attack_test.go")
	for _, name := range []string{
		"TestSideChannel_Backfill_ResponseHasNoOutOfScopeFields",
		"TestSideChannel_Backfill_DiffAttack_OnlyVariesWithM",
		"TestSideChannel_Import_DiffAttack_SilentSkip",
		"TestSideChannel_RepeatedRequests_CannotEnumerateN",
		"TestSideChannel_Backfill_AllTier_ZeroRegression",
	} {
		assert.Contains(t, src, "func "+name+"(", "150 差分攻击测试 %s 不得被误删", name)
	}
	assert.Contains(t, src, "OnlyVariesWithM", "差分不变式 token 存在")
	assert.Contains(t, src, "CannotEnumerateN", "不可枚举 N token 存在")
	pg := responseContractRead(t, "pg_contract_test.go")
	assert.Contains(t, pg, "func TestScopeWriteImport_StaticContract(", "150 静态契约测试不得被误删")
}

// TestResponseContract_OperlogChain：审计链不降级 —— ScopeWriteOperParam
// 三字段 / Excel 导入 outOfScope operlog / backfill OperParam 格式串仍在；
// 25 个 OperType 常量锁由 internal/utils/operlog/regression_test.go 守护。
func TestResponseContract_OperlogChain(t *testing.T) {
	sw := responseContractRead(t, "scope_write.go")
	// ScopeWriteOperParam 的 operlog 格式串：三字段聚合计数全保留（审计链不降级）
	for _, field := range []string{`"requested":%d`, `"visible":%d`, `"outOfScope":%d`} {
		assert.Contains(t, sw, field, "ScopeWriteOperParam 审计链字段 %q 不得删", field)
	}
	eh := responseContractRead(t, "../../api/v1/operations/excel_handler.go")
	assert.Contains(t, eh, `"outOfScope":%d`, "Excel 导入 operlog 必须保留 outOfScope 审计参数")
	wh := responseContractRead(t, "../../api/v1/operations/workstation_handler.go")
	assert.Contains(t, wh, `"fixMismatch":%t`, "backfill OperParam 格式串不得改（150 D-4）")
}

// TestResponseContract_DiffAttack_NChangesMFixed：差分攻击复跑 —— Dept /
// DeptChild 两相邻 scope 各请求一次，断言响应体数值字段仅随 M 变化、不随
// N−M 变化（N=5/M=2 vs N=9/M=2 逐字节相等）。
func TestResponseContract_DiffAttack_NChangesMFixed(t *testing.T) {
	type snapshot struct {
		Total int64 `json:"total"`
		Count int    `json:"count"`
	}
	// 两相邻档位（Dept / DeptChild）的同源覆盖见 TestHomology_DiffInvariant_NChangesMFixed；
	// 此处锁「响应体数值字段仅随 M 变化、不随 N−M 变化」的防复活不变式。
	query.ResetRegistryForTest()
	require.NoError(t, SetupScopedDBRegistry())

	var results [][]snapshot
	for _, f := range []struct{ visible, outOfScope int }{{2, 3}, {2, 7}} {
		db := homologyFixture(t, f.visible, f.outOfScope)
		ctx := homologyCtx(datascope.ScopeResult{DeptIDs: []string{homologyDeptA}})
		total, n := homologyCountAndList(t, db, ctx)
		results = append(results, []snapshot{{Total: total, Count: n}})
	}
	a, _ := json.Marshal(results[0])
	b, _ := json.Marshal(results[1])
	assert.Equal(t, string(a), string(b), "N=5/M=2 与 N=9/M=2 响应体数值字段逐字节相等")
	assert.Equal(t, int64(2), results[0][0].Total)
}

// TestRouter151_ScopedDBMountContract：装配调用恰好 1 处、位于 ops 组定义之后、
// roomPhotos 补挂行仍存在且未改（与 149 的 Router 契约测试同思路）。
func TestRouter151_ScopedDBMountContract(t *testing.T) {
	src := responseContractRead(t, "../../api/router.go")
	assert.Equal(t, 1, strings.Count(src, "MustSetupScopedDBRegistry()"),
		"ScopedDB 注册表装配调用必须恰好 1 处")
	opsIdx := strings.Index(src, `ops := r.Group("/ops")`)
	mountIdx := strings.Index(src, "MustSetupScopedDBRegistry()")
	require.GreaterOrEqual(t, opsIdx, 0, "ops 组定义必须存在")
	require.GreaterOrEqual(t, mountIdx, 0, "装配调用必须存在")
	assert.Less(t, opsIdx, mountIdx, "装配调用必须位于 ops 组定义之后")
	// 149 的 :1176 补挂行（内容不因行号 +2 而改变）
	assert.Contains(t, src, "roomPhotos.Use(middleware.DataScopePermission(core))",
		"149 的 rooms/photos DataScopePermission 补挂行不得改动")
	// 不新增中间件重复挂载 / 不增删路由：DataScopePermission 挂载数与 149 一致
	assert.NotContains(t, src, "ScopedDBPermission", "不得新增 ScopedDB 专用中间件")
}
