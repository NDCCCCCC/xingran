package operations

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// T2：注册表覆盖 / 单一真实源 / 149 文件未触碰 / 双方言
// ---------------------------------------------------------------------------

func TestScopeRegistry_CoversAllEntities(t *testing.T) {
	// 装配一次（router.go 启动期调用；此处测试直接调，注册表进程内幂等由
	// Register 的「重复注册返回 error」守护，故先清空再装配）。
	resetScopedRegistryForTest(t)
	require.NoError(t, SetupScopedDBRegistry(), "注册表装配必须返回 nil")

	want := []string{
		"ops_buildings", "ops_floors", "sys_workstation", "ops_asset",
		"ops_info_points", "ops_server_rooms", "ops_dedicated_lines",
		"ops_room_devices", "ops_workstation_device", "ops_walls",
		"ops_doors", "ops_floor_plan_texts", "ops_room_photos",
		"sys_dept_location_alias",
	}
	sort.Strings(want)
	got := query.RegisteredTables()
	assert.Equal(t, want, got, "RegisteredTables() 必须与登记表逐项一致，缺一即红")
	for _, tbl := range want {
		e, ok := query.Lookup(tbl)
		assert.True(t, ok, "表 %s 必须已登记", tbl)
		assert.NotNil(t, e.Predicate, "表 %s 的 Predicate 不能为 nil", tbl)
		assert.NotNil(t, e.CacheKey, "表 %s 的 CacheKey 不能为 nil", tbl)
	}
}

func TestScopeRegistry_SingleSourceOfTruth(t *testing.T) {
	src := scope149ReadSource(t, "scope_registry.go")
	code := stripRegistryComments(src)

	// 禁止内联复制 SQL 片段 / 方言分支 / 档位判定。
	for _, forbidden := range []string{
		"COALESCE(", "NULLIF(", "1=0", "::uuid", "::text", "ILIKE",
		"isPostgres", "ON CONFLICT", "SET LOCAL", "internal/models/base/",
	} {
		assert.NotContains(t, code, forbidden, "scope_registry.go 禁止出现 %q（单一真实源 / 双方言 / 禁区）", forbidden)
	}
	// 必须引用既有单一真实源 token（零 SQL 复制、引用 149 成果）。
	for _, token := range []string{
		"effectiveOrgExpr", "entityExistsSub", "wrapScopeExists", "dept_id",
		"applyEntityReadScope", "applyBuildingScope", "applyAssetReadScope",
		"workstationOrgExistsSub", "scopeCacheKey",
	} {
		assert.Contains(t, src, token, "scope_registry.go 必须引用既有单一真实源 token %q", token)
	}
}

func TestScopeRegistry_149FilesUntouched(t *testing.T) {
	// 149 的 5 个 service + scope_entities.go 零改动：既有收口 token 仍在。
	checks := map[string][]string{
		"wall_service.go":            {"entityReadScope", "scopeEntityWall"},
		"door_service.go":            {"entityReadScope", "scopeEntityDoor"},
		"floor_plan_text_service.go": {"entityReadScope", "scopeEntityFloorPlanText"},
		"room_photo_service.go":      {"photoReadVisible", "entityReadScope"},
		"location_alias_service.go":  {"applyEntityReadScope", "scopeEntityLocationAlias"},
		"scope_entities.go":          {"entityExistsSub", "applyEntityReadScope", "entityFloorOrgExpr"},
	}
	for name, tokens := range checks {
		src := scope149ReadSource(t, name)
		for _, tok := range tokens {
			assert.Contains(t, src, tok, "%s 的 149 收口 token %q 不得被误改", name, tok)
		}
	}
}

func TestScopeRegistry_DualDialect(t *testing.T) {
	// 谓词 SQL 含 CAST(... AS TEXT)、不含 PG 专属 :: / ILIKE / ON CONFLICT
	// （沿用 149 TestScope149_Static_DualDialectAssertions 手法）。
	resetScopedRegistryForTest(t)
	require.NoError(t, SetupScopedDBRegistry())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{DeptIDs: []string{"d1"}})

	for _, tbl := range []string{"ops_asset", "ops_buildings", "ops_server_rooms", "sys_dept_location_alias"} {
		e, ok := query.Lookup(tbl)
		require.True(t, ok, "%s 必须已登记", tbl)
		tx := db.WithContext(ctx).Table(tbl)
		got := e.Predicate(tx, datascope.ScopeResult{DeptIDs: []string{"d1"}}).
			Find(&[]map[string]any{}).Statement.SQL.String()
		upper := strings.ToUpper(got)
		assert.Contains(t, upper, "CAST(", "%s 谓词必须双方言 CAST AS TEXT 形态: %s", tbl, got)
		assert.NotContains(t, got, "::", "%s 谓词禁 PG 专属 ::: %s", tbl, got)
		assert.NotContains(t, upper, "ILIKE", "%s 谓词禁 ILIKE: %s", tbl, got)
	}
}

func TestScopeRegistry_NoBareChain(t *testing.T) {
	// 与 T3 的 TestScopedLayer_ZeroBareChain 共用扫描口径；此处只断言
	// 8 核心 service 起链行已经改走 ScopedQuery / ScopedDB（T3 完成后绿）。
	bare := scanBareChainSources(t)
	for _, f := range scopeLayerCoreFiles {
		assert.Zero(t, bare[f], "%s 不得残留裸 s.db.WithContext(ctx) 起链", f)
	}
}

func TestScopeRegistry_RegistryAppliedAndIdempotent(t *testing.T) {
	// 幂等：已挂 131/149 既有谓词的链，注册表 Predicate 不再重复挂 → SQL 零漂移。
	resetScopedRegistryForTest(t)
	require.NoError(t, SetupScopedDBRegistry())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DryRun: true})
	require.NoError(t, err)
	ctx := datascope.WithContext(context.Background(), datascope.ScopeResult{DeptIDs: []string{"d1"}})

	e, ok := query.Lookup("ops_asset")
	require.True(t, ok)
	// 既有收口已挂
	existing := db.WithContext(ctx).Table("ops_asset")
	existing = applyAssetReadScope(existing, ctx)
	baseline := existing.Find(&[]map[string]any{}).Statement.SQL.String()

	// 注册表谓词应用在同一条已挂链上 → 幂等跳过
	got := e.Predicate(existing, datascope.ScopeResult{DeptIDs: []string{"d1"}}).
		Find(&[]map[string]any{}).Statement.SQL.String()
	assert.Equal(t, baseline, got, "已挂既有收口的链必须幂等（SQL 零漂移零回归）")

	// 漏挂 / 新增端点：未挂链自动补挂（机制化兜住），且补挂谓词与
	// 既有收口函数产出的 SQL 逐字一致 → 证明「零 SQL 复制、单一真实源」。
	fresh := db.WithContext(ctx).Table("ops_asset")
	auto := e.Predicate(fresh, datascope.ScopeResult{DeptIDs: []string{"d1"}}).
		Find(&[]map[string]any{}).Statement.SQL.String()
	assert.Equal(t, baseline, auto, "未挂链自动补挂的谓词必须与既有收口逐字一致（单一真实源零复制）")
	assert.Contains(t, strings.ToUpper(auto), "DEPT_ID", "补挂谓词应含 dept_id 直挂形态")
}

// ---------------------------------------------------------------------------
// T3：强制层静态断言（零裸起链 / All 兜底 / 基座读链写链分离）
// ---------------------------------------------------------------------------

// scopeLayerCoreFiles 是 151 必须改起链的 8 核心 + 旁路 + 导入/导出 + 缓存文件。
var scopeLayerCoreFiles = []string{
	"asset_service.go", "building_service.go", "floor_service.go",
	"workstation_service.go", "infopoint_service.go", "server_room_service.go",
	"dedicated_line_service.go", "room_device_service.go",
	"building_service_typesafe.go", "excel_service.go", "floor_cache_impl.go",
}

// zeroBareChainWhitelist 白名单：4 类保留文件 + 测试 / 非编译态，逐项登记保留原因。
//
// checker 风险点①（如实披露边界）：本白名单排除后，151 的「自动获得谓词」
// 实际只覆盖 8 核心 + GORMRepository 读链；这 4 类的可见集谓词仍由
// 149/148 既有收口承担，不得宣称「全 ops 域零裸起链」或「统一架构覆盖所有端点」。
var zeroBareChainWhitelist = map[string]string{
	"excel_export_service.go":      "归 148：resolveAssociations / CAST(id AS TEXT) 改写，151 禁改（PLAN §4.3 #11）",
	"workstation_device_service.go": "归 148：DIALECT-02 CTE 改写 / Go 层合并，151 禁改（PLAN §4.3 #15）",
	"location_alias_service.go":    "归 149：已用 dept_id ∈ 可见集直挂收口，151 只登记表不裸改（PLAN §4.3 #13）",
	"room_photo_service.go":        "归 149：已 10 方法逐方法登记 photoReadVisible 等，151 只登记表不裸改（PLAN §4.3 #14）",
	"pg_contract_test.go":          "测试文件：纳入静态断言扫描 + 追加同源断言，不计入生产起链（PLAN §4.3 #16）",
}

// scopedLayerReadFuncs 是「读端点」函数白名单：只有这些函数体内的起链必须
// 改走 ScopedQuery / ScopedDB（与 T3 起链改造脚本同口径）。
//
// 写方法（Create/Update/Delete/BatchDelete）、写侧前置校验（validate*）、
// 写侧辅助（populate*/attach*/postImport*/ensureUniqueValueCacheLoaded）、
// 150 的 BackfillDeptFromUser 写回填、EXISTS 子查询骨架、Raw SQL、事务，
// 一律保持裸起链一字不动 —— 挂读 scope 会污染 UPDATE/DELETE 的 WHERE
// 或改变写侧可见集校验语义（PLAN §4.2 写链禁改红线）。
//
// checker 风险点①（如实披露边界）：本断言覆盖的是「读端点起链」而非
// 「全文件 s.db.WithContext(ctx) 字面量 = 0」；写链保留裸起链是硬红线，
// 且 4 类白名单保留文件（148/149）由既有收口承担，不宣称「全 ops 域零裸起链」。
var scopedLayerReadFuncs = map[string]bool{
	"Statistics": true, "List": true, "GetTree": true, "GetFloorsByBuildingID": true,
	"GetFloorWorkstationsAll": true, "GetWorkstationDeptOptions": true, "CountsByFloor": true,
	"GetDeviceTypes": true, "GetDeviceCategories": true, "GetStatusValues": true,
	"attachFloorCount": true, "attachWorkstationStats": true,
	"SearchBuildingOptions": true, "SearchFloorOptions": true, "SearchWorkstationOptions": true,
	"SearchInfoPointOptions": true, "SearchServerRoomOptions": true,
	"SearchDedicatedLineOptions": true, "SearchRoomDeviceOptions": true,
	"queryWorkstationIDsForExport": true, "batchGetWorkstationNames": true,
	"batchGetADEnrichment": true, "batchGetAssetEnrichment": true, "queryData": true,
	"ExportData": true, "GetByDeviceSN": true, "GetByID": true,
}

// scanBareChainSources 扫描 operations/*.go 的「读端点内裸起链」出现数。
//
// 口径（与 PLAN §4.4 / §4.2「写链禁改」红线对齐）：
//   - 排除 *_test.go / *.backup / *.go.backup（非编译态）；
//   - 写链（Save/Create/Delete/Exec/Transaction/Updates/Unscoped/Raw）、
//     EXISTS 子查询骨架（Session NewDB）、写方法/写侧辅助函数体内的起链
//     保持裸起链一字不动 → 豁免；
//   - 仅 scopedLayerReadFuncs 内、且不含写标记的「读端点起链」必须改走
//     ScopedQuery / ScopedDB，残留即红。
func scanBareChainSources(t *testing.T) map[string]int {
	t.Helper()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	writeMarkers := []string{
		".Save(", ".Create(", ".Delete(", ".Exec(", ".Transaction(",
		".Unscoped(", ".Updates(", ".Raw(", "Session(&gorm.Session{NewDB: true})",
	}
	out := make(map[string]int)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".backup") ||
			strings.HasSuffix(name, ".go.backup") {
			continue
		}
		b, err := os.ReadFile(name)
		require.NoError(t, err)
		lines := strings.Split(string(b), "\n")
		curFunc := ""
		n := 0
		for i, line := range lines {
			if m := regexp.MustCompile(`^func \([^)]*\) ([A-Za-z0-9_]+)\(`).FindStringSubmatch(line); len(m) > 1 {
				curFunc = m[1]
			}
			if !strings.Contains(line, "s.db.WithContext(ctx)") {
				continue
			}
			if !scopedLayerReadFuncs[curFunc] {
				continue
			}
			window := strings.Join(lines[i:min(i+5, len(lines))], "\n")
			isWriteOrSub := false
			for _, m := range writeMarkers {
				if strings.Contains(window, m) {
					isWriteOrSub = true
					break
				}
			}
			if !isWriteOrSub {
				n++
			}
		}
		out[name] = n
	}
	return out
}

func TestScopedLayer_ZeroBareChain(t *testing.T) {
	bare := scanBareChainSources(t)
	for name, n := range bare {
		_, whitelisted := zeroBareChainWhitelist[name]
		if whitelisted {
			// 白名单逐项必须含「保留原因」注释，缺注释即红。
			assert.NotEmpty(t, zeroBareChainWhitelist[name], "%s 白名单必须登记保留原因", name)
			continue
		}
		assert.Zero(t, n, "%s 不得残留裸 s.db.WithContext(ctx) 起链（应改 ScopedQuery/ScopedDB）", name)
	}
	// 白名单自身：必须逐项含「归 148 / 归 149 / 测试」保留原因，防止随意扩大白名单。
	assert.Contains(t, zeroBareChainWhitelist["location_alias_service.go"], "归 149")
	assert.Contains(t, zeroBareChainWhitelist["room_photo_service.go"], "归 149")
	assert.Contains(t, zeroBareChainWhitelist["pg_contract_test.go"], "测试")
	assert.Contains(t, zeroBareChainWhitelist["excel_export_service.go"], "归 148")
	assert.Contains(t, zeroBareChainWhitelist["workstation_device_service.go"], "归 148")
}

func TestScopedLayer_AllFallbackIntact(t *testing.T) {
	filter := scope149ReadSource(t, "scope_filter.go")
	assert.Contains(t, filter, "return !ok || scope.All", "scope_filter.go:37-39 scopeIsAll 的 !ok || All 短路不得改")
	assert.Contains(t, filter, "FromContext", "scope_filter.go 必须继续经 datascope.FromContext 取值")

	ds, err := os.ReadFile("../../../pkg/datascope/datascope.go")
	require.NoError(t, err)
	assert.Contains(t, string(ds), "scope == models.DataScopeAll || scope == 0",
		"datascope.go:103-109 的 All/0 短路不得改")
}

func TestScopedLayer_BaseReadChainUsesScopedDB(t *testing.T) {
	svc, err := os.ReadFile("../../services/base/service.go")
	require.NoError(t, err)
	src := string(svc)
	// 读链：List / GetByID 必须走 query.ScopedDB
	assert.Contains(t, src, "query.ScopedDB(ctx, r.db).Model(new(T))", "GetByID/List 读链必须改走 ScopedDB")
	// 写链：Create/Update/Delete/BatchDelete 一字不动（禁挂读谓词）
	writeMethods := []string{"Create", "Update", "Delete", "BatchDelete"}
	for _, m := range writeMethods {
		// 逐个方法体切片检查不含 ScopedDB
		idx := strings.Index(src, "func (r *GORMRepository[T]) "+m+"(")
		require.GreaterOrEqual(t, idx, 0, "缺少方法 %s", m)
		end := strings.Index(src[idx:], "\nfunc (r *GORMRepository[T])")
		if end < 0 {
			end = len(src) - idx
		}
		body := src[idx : idx+end]
		assert.NotContains(t, body, "ScopedDB", "%s 写链禁止挂 ScopedDB（GORM Scopes 会污染 UPDATE/DELETE WHERE）", m)
		assert.Contains(t, body, "r.db.WithContext(ctx)", "%s 写链必须保持裸起链不变", m)
	}
}

func TestHomology_StaticChainContract(t *testing.T) {
	// F-11：service.go:113-124 的 Count(&total) 必须位于 scopes 循环之后
	// → scopes 在 Count 之前应用 = total/List 同源结构性保证。
	svc, err := os.ReadFile("../../services/base/service.go")
	require.NoError(t, err)
	src := string(svc)
	loop := strings.Index(src, "for _, s := range scopes")
	count := strings.Index(src, "query.Count(&total)")
	require.GreaterOrEqual(t, loop, 0, "List 必须含 scopes 循环")
	require.GreaterOrEqual(t, count, 0, "List 必须含 Count(&total)")
	assert.Less(t, loop, count, "scopes 必须在 Count 之前应用（同源结构性保证）")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// resetScopedRegistryForTest 清空注册表后供测试装配，避免跨用例「重复注册」干扰。
// 通过反射调用 pkg/query 的测试期重置钩子（同仓内以导出包装暴露）。
func resetScopedRegistryForTest(t *testing.T) {
	t.Helper()
	query.ResetRegistryForTest()
}

// stripRegistryComments 剔除 // 行注释，供静态断言只判定真实代码行。
func stripRegistryComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

// 编译期守护：确保本文件引用的路径 helper 存在（避免误删目录）。
var _ = filepath.Join
