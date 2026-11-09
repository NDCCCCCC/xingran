// Package operations: Phase 131 四域读端点 + Excel 导出的「可见集」过滤基建。
//
// 设计契约（.planning/notes/261107-v152-self-scope-decision.md LOCKED）：
//   - 只消费 pkg/datascope 的 ScopeResult，本包不重新裁定档位、不查 sys_dept；
//   - Phase 151 / ARCH-01：本层由 pkg/query.ScopedDB 在起链时**强制调用**，
//     wrapScopeExists 不再是「唯一可选入口」——资源归属注册表登记后，所有
//     经 ScopedQuery / ScopedDB 起链的查询自动获得可见集谓词，靠机制而非约定；
//   - All / 0 / unset 特判短路：原样返回 query，零 DB 查询、零额外 SQL（超管零回归）；
//   - 未挂注入中间件（FromContext 返回 false）同样按 All 兜底，保证存量调用方
//     （如直接以 context.Background() 调用的 SQLite 单测 / 内部任务）行为不变；
//   - FailClosed（异常枚举）→ Where("1=0") 空集；
//   - Custom / Dept / DeptChild → CAST(有效机构 AS TEXT) IN 可见集 deptIDs；
//   - Self（HookSelfUser）：部门集为空，按裁定「无用户列实体空集、有用户列实体
//     按资源归属链上的用户列 = UserID 过滤」，由调用方经 userCol 分派；
//   - 「有效机构」表达式必须由调用方传 effectiveOrgExpr(...) 的结果（单一真实源
//     dept_filter.go），本文件禁止内联复制 COALESCE/NULLIF 表达式；
//   - 所有 uuid/varchar 比较统一 CAST(... AS TEXT)，PG/SQLite 双方言安全。
package operations

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
)

// scopeFromContext 是 datascope.FromContext 的薄封装。
// 返回 (ScopeResult, ok)；ok=false 表示请求未挂 DataScopePermission 注入中间件，
// 调用方一律按「All 档」兜底（0/unset → All 超管零回归硬约束）。
func scopeFromContext(ctx context.Context) (datascope.ScopeResult, bool) {
	return datascope.FromContext(ctx)
}

// scopeIsAll 判定是否应完全短路（不挂任何范围条件、零 DB 查询）。
// 未注入（ok=false）与 All=true 都短路；超管 / 0 值 / 未挂中间件路径零回归。
func scopeIsAll(scope datascope.ScopeResult, ok bool) bool {
	return !ok || scope.All
}

// scopeCacheKey 由 ScopeResult 稳定派生缓存 key 的「scope 维度」段。
//
// PMODEL-01 防跨范围泄漏：不同可见集必须产出不同 key；All 档短路路径
// 同样含 scope 段（LOCKED §1 关键不变量 3）。规则：
//
//	未注入 / All          → "all"
//	FailClosed            → "closed"
//	Self(HookSelfUser)    → "self" + sha1(UserID) 前 8 hex（不同用户不串）
//	DeptIDs 非空          → "d" + sha1(排序后 deptIDs 逗号拼接) 前 8 hex
//
// DeptIDs 在 datascope.Resolve 侧已 normalizeIDs（去重+去空+排序），此处再
// 排序一次保证任何构造来源下 key 都稳定。
func scopeCacheKey(ctx context.Context) string {
	scope, ok := scopeFromContext(ctx)
	return scopeCacheKeyOf(scope, ok)
}

// scopeCacheKeyOf 是 scopeCacheKey 的纯函数版本，便于单测直接构造 ScopeResult。
func scopeCacheKeyOf(scope datascope.ScopeResult, ok bool) string {
	if !ok || scope.All {
		return "all"
	}
	if scope.FailClosed {
		return "closed"
	}
	if scope.ResourceHook == datascope.HookSelfUser {
		h := sha1.Sum([]byte("self|" + scope.UserID))
		return "self" + hex.EncodeToString(h[:8])
	}
	ids := append([]string(nil), scope.DeptIDs...)
	sort.Strings(ids)
	h := sha1.Sum([]byte("dept|" + strings.Join(ids, ",")))
	return "d" + hex.EncodeToString(h[:8])
}

// orgScopeClause 构建「有效机构 ∈ 可见集」的 SQL 条件片段。
//
// 供 EXISTS 子查询形态使用（workstation / floor / infopoint 的有效机构跨
// ops_floors / ops_buildings 两张表，列引用只在该子查询作用域内合法）：
//
//	sub := workstationOrgExistsSub(...)
//	if cond, args, skip := orgScopeClause(scope, ok, workstationOrgColumn); !skip {
//	    sub = sub.Where(cond, args...)
//	}
//	query = query.Where("EXISTS (?)", sub)
//
// 返回值：
//   - skip=true：All / 未注入，调用方应完全跳过（不挂 EXISTS，零 DB 查询）；
//   - FailClosed / Self 兜底 → "1=0"，args 为空；
//   - DeptIDs 非空 → "CAST(<orgExpr> AS TEXT) IN ?"，args=[deptIDs]。
//
// orgExpr 必须传 effectiveOrgExpr(...) 的结果（含双侧 CAST），外层再包一层
// CAST(... AS TEXT) 保证与 []string 参数在 PG/SQLite 双方言下类型一致。
func orgScopeClause(scope datascope.ScopeResult, ok bool, orgExpr string) (string, []any, bool) {
	if scopeIsAll(scope, ok) {
		return "", nil, true
	}
	if scope.FailClosed || scope.ResourceHook == datascope.HookSelfUser || len(scope.DeptIDs) == 0 {
		return "1=0", nil, false
	}
	return "CAST(" + orgExpr + " AS TEXT) IN ?", []any{scope.DeptIDs}, false
}

// applyOrgScope 对查询直接挂「有效机构 ∈ 可见集」条件（本表/当前查询作用域内
// 可直接引用 orgExpr 的场景，如 building.org_id、Excel 导出的 sys_workstation 子查询）。
//
// 规格签名三参不变：
//   - All / 未注入 → 原样返回（短路，零 DB 查询）；
//   - FailClosed / Self / DeptIDs 空集 → Where("1=0")；
//   - DeptIDs 非空 → Where("CAST(<orgExpr> AS TEXT) IN ?", deptIDs)。
//
// Self 档在「无用户列实体」（floor/building）上按裁定返回空集；有用户列的实体
// （workstation/infopoint）请改用 applyReadScope 或 selfUserClause 分派。
func applyOrgScope(query *gorm.DB, ctx context.Context, orgExpr string) *gorm.DB {
	scope, ok := scopeFromContext(ctx)
	cond, args, skip := orgScopeClause(scope, ok, orgExpr)
	if skip {
		return query
	}
	return query.Where(cond, args...)
}

// applyReadScope 是四域读路径的统一分派入口（本表列可直接引用 orgExpr 的形态）：
//   - userCol == ""（floor / building，无用户归属列）：Self 档按裁定空集；
//   - userCol != ""（workstation / infopoint）：Self 档追加
//     "userCol = ?" + ScopeResult.UserID（UserID 为空则退回 1=0 fail-closed）。
//
// All / 未注入短路原样返回；FailClosed → 1=0；DeptIDs 非空 → IN 条件。
func applyReadScope(query *gorm.DB, ctx context.Context, orgExpr, userCol string) *gorm.DB {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return query
	}
	if scope.ResourceHook == datascope.HookSelfUser && !scope.FailClosed {
		if userCol == "" || scope.UserID == "" {
			return query.Where("1=0")
		}
		return query.Where(userCol+" = ?", scope.UserID)
	}
	cond, args, skip := orgScopeClause(scope, ok, orgExpr)
	if skip {
		return query
	}
	return query.Where(cond, args...)
}

// selfUserClause 返回 Self 档「用户列 = UserID」条件片段（供 EXISTS 子查询 / 需要
// 组合 OR 的场景），非 Self 档或 UserID 缺失时 applied=false。
//
// 与 applyReadScope 的 userCol 分支同源，避免 workstation / infopoint 两处
// 各写一份 user_id 条件造成语义漂移。
func selfUserClause(scope datascope.ScopeResult, ok bool, userCol string) (string, []any, bool) {
	if !ok || scope.All || scope.FailClosed || scope.ResourceHook != datascope.HookSelfUser {
		return "", nil, false
	}
	if userCol == "" || scope.UserID == "" {
		return "1=0", nil, true
	}
	return userCol + " = ?", []any{scope.UserID}, true
}

// scopeRejectsRequestedOrg 实现「越界 orgId 求交」：
// 请求携带的筛选 orgId 若不在可见集内 → 返回 true，调用方应直接收紧为空集
// （读返回空、不报错，符合「前端 orgId 不可越权读取」）。
//
// 语义边界（fail-closed 对齐 PMODEL-01）：
//   - All / 未注入：不校验（orgId 本身是业务筛选，由既有部门子树逻辑处理）；
//   - FailClosed：恒 true（本来就是空集）；
//   - Self：DeptIDs 为空集，任何非空 orgId 都视为越界（Self 由 user_id 限定）；
//   - DeptIDs 非空：requestedOrgID 为空放行；非空则必须精确命中可见集。
func scopeRejectsRequestedOrg(scope datascope.ScopeResult, ok bool, requestedOrgID string) bool {
	if scopeIsAll(scope, ok) {
		return false
	}
	if scope.FailClosed {
		return true
	}
	if requestedOrgID == "" {
		return false
	}
	for _, id := range scope.DeptIDs {
		if id == requestedOrgID {
			return false
		}
	}
	return true
}

// applyScopeToSub 把可见集条件加进「有效机构 EXISTS 子查询」骨架。
//
// 四域中 workstation / floor / infopoint 的有效机构跨 ops_floors(f) 与
// ops_buildings(b) 两张表，列引用只在子查询作用域内合法；调用方先构造好
// 骨架（含表别名与实体关联条件），本函数只负责追加范围条件：
//
//	sub := workstationOrgExistsSub(db, "CAST(f.id AS TEXT) = sys_workstation.floor_id")
//	query = wrapScopeExists(query, ctx, sub, workstationOrgColumn, "")
//
// 参数：
//   - orgExpr：effectiveOrgExpr("f.org_id","b.org_id") 的结果（单一真实源）；
//   - subUserCol：子查询内可引用的用户列（workstation/infopoint 传
//     "sys_workstation.user_id" / "w.user_id"，floor/building 传 "" = 无用户列）。
//
// 返回 (sub, skipAll)：skipAll=true 表示 All / 未注入，调用方应完全跳过
// 该 EXISTS（零 DB 查询、零额外 SQL）；否则返回已挂条件的 sub 供 EXISTS 使用。
func applyScopeToSub(sub *gorm.DB, ctx context.Context, orgExpr, subUserCol string) (*gorm.DB, bool) {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return sub, true
	}
	// Self 档优先按用户列分派（部门集为空，不能落到 IN 空集分支）。
	if scope.ResourceHook == datascope.HookSelfUser && !scope.FailClosed {
		cond, args, applied := selfUserClause(scope, ok, subUserCol)
		if !applied {
			return sub, true
		}
		return sub.Where(cond, args...), false
	}
	cond, args, skip := orgScopeClause(scope, ok, orgExpr)
	if skip {
		return sub, true
	}
	return sub.Where(cond, args...), false
}

// wrapScopeExists 组合 applyScopeToSub + "EXISTS (?)" 挂接，是四域读路径
// （List/Statistics/SearchOptions/GetByID/workstations-all/counts-by-floor）
// 的统一收口入口。
//
// All / 未注入 → 原样返回 query（零额外 SQL，超管零回归）；
// 其余档位 → query.Where("EXISTS (?)", 已挂可见集条件的 sub)。
func wrapScopeExists(query *gorm.DB, ctx context.Context, sub *gorm.DB, orgExpr, subUserCol string) *gorm.DB {
	scopedSub, skip := applyScopeToSub(sub, ctx, orgExpr, subUserCol)
	if skip {
		return query
	}
	return query.Where("EXISTS (?)", scopedSub)
}

// scopeRejectsRequestedOrgFromContext 是 scopeRejectsRequestedOrg 的 ctx 便捷版，
// 供各读函数在应用业务 orgId 筛选前统一求交（越界返回空，不报错）。
func scopeRejectsRequestedOrgFromContext(ctx context.Context, requestedOrgID string) bool {
	scope, ok := scopeFromContext(ctx)
	return scopeRejectsRequestedOrg(scope, ok, requestedOrgID)
}

// applyBuildingScope 楼宇读路径专用：可见楼宇 = 「楼宇自身 org_id ∈ 可见集」
// OR 「其下任一楼层有效机构 ∈ 可见集」（一栋楼跨机构办公，见 quick-261106-wso）。
//
// 与既有 applyDeptFilter 双分支语义对齐，但范围集合来自 ScopeResult 而非
// 请求 orgId 的子树展开（service 层强制，禁止信任 params）。
// 有效机构表达式走 effectiveOrgExpr 单一真实源；All / 未注入短路原样返回。
func applyBuildingScope(query *gorm.DB, ctx context.Context) *gorm.DB {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return query
	}
	// Self 档（无用户列）/ FailClosed / 空可见集 → 整楼宇集空。
	if scope.FailClosed || scope.ResourceHook == datascope.HookSelfUser || len(scope.DeptIDs) == 0 {
		return query.Where("1=0")
	}
	orgExpr := effectiveOrgExpr("f.org_id", "ops_buildings.org_id")
	// 两侧条件各带一个 IN ? 占位符，共用同一可见集切片。
	cond := "CAST(" + orgExpr + " AS TEXT) IN ?"
	buildingCond := "CAST(ops_buildings.org_id AS TEXT) IN ?"
	return query.Where("("+buildingCond+` OR EXISTS (
		SELECT 1 FROM ops_floors f
		WHERE CAST(f.building_id AS TEXT) = CAST(ops_buildings.id AS TEXT)
		  AND `+cond+`
		  AND f.deleted_at IS NULL
	))`, scope.DeptIDs, scope.DeptIDs)
}

// applyExcelWorkstationScope 是 Excel 导出两条独立查询路径的可见集收口入口
// （DS-06：导出不豁免、不走 List、不能依赖 params 里的 orgId）。
//
// 与读端点 wrapScopeExists 的区别：导出路径不复用 service 的 EXISTS 骨架，
// 而是按导出查询自身的表形态注入 EXISTS 子查询，保持两条路径各自独立：
//
//	EXISTS (SELECT 1 FROM ops_floors f
//	        JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id
//	        WHERE CAST(f.id AS TEXT) = sys_workstation.floor_id
//	          AND <有效机构 ∈ 可见集 或 Self 用户列>
//	          AND f.deleted_at IS NULL AND b.deleted_at IS NULL)
//
// orgExpr 由调用方传 effectiveOrgExpr("f.org_id","b.org_id") 的结果。
// 新增条件统一 CAST(... AS TEXT)（PG/SQLite 双方言），与 BuildQuery 既有的
// PG 专有 ::uuid JOIN 互不干扰（P1-8：::uuid 属既有技术债，本 phase 不改）。
//
// All / 未注入 → 原样返回（短路，零 DB 查询）；FailClosed / 空集 → 1=0；
// Self → 在子查询内按 sys_workstation.user_id = UserID（工位有用户列）。
func applyExcelWorkstationScope(query *gorm.DB, ctx context.Context) *gorm.DB {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return query
	}
	// Self 档：部门集为空，按导出实体自身的用户列过滤（外层 query 可直接引用
	// sys_workstation.user_id，不进 EXISTS 子查询）。
	if scope.ResourceHook == datascope.HookSelfUser && !scope.FailClosed {
		cond, args, applied := selfUserClause(scope, ok, "sys_workstation.user_id")
		if applied {
			return query.Where(cond, args...)
		}
		return query.Where("1=0")
	}
	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	sub := query.Session(&gorm.Session{NewDB: true}).
		Table("ops_floors f").
		Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
		Where("CAST(f.id AS TEXT) = sys_workstation.floor_id").
		Where("f.deleted_at IS NULL AND b.deleted_at IS NULL").
		Select("1")
	return wrapScopeExists(query, ctx, sub, orgExpr, "")
}

// ---------------------------------------------------------------------------
// Phase 151 / ARCH-01：强制查询层入口 + 缓存 key 薄封装
// ---------------------------------------------------------------------------

// ScopedQuery 是 ops 包内**统一查询起链点**（不可绕过的查询层）。
//
// 直接委托 pkg/query.ScopedDB：未注入 ctx / All 档原样返回（零回归短路），
// 其余档位由 GORM Scopes 在 SQL 构建时按表名查「资源归属注册表」强制挂
// 可见集谓词。8 核心 service / 类型安全旁路 / Excel 起链一律改走本函数；
// wrapScopeExists 保持同名同签名（131/149 存量调用零改），但其角色已从
// 「唯一可选入口」降为「注册表谓词复用的既有实现」。
func ScopedQuery(ctx context.Context, db *gorm.DB) *gorm.DB {
	return query.ScopedDB(ctx, db)
}

// ScopedCacheKey 是 scopeCacheKey(ctx)（本文件 :53-74）的导出薄封装。
//
// 唯一算法实现仍是 scopeCacheKey，pkg/query 经注册表 CacheKey 回调引用，
// **禁止复制算法**；供 floor:tree 等固定 key 挂 scope 维度（PMODEL-01）。
func ScopedCacheKey(ctx context.Context) string {
	return scopeCacheKey(ctx)
}
