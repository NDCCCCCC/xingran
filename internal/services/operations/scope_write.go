// Package operations: Phase 132 四域「写端点」可见集校验基建。
//
// 设计契约（.planning/notes/261107-v152-batch-import-decision.md LOCKED §2 /
// .planning/notes/261107-v152-self-scope-decision.md LOCKED §1/§2）：
//   - 单条 Create/Update/Delete 越界 → ErrScopeDenied（handler 透传 HTTP 403）；
//   - 批量 Delete/BatchOperation/positions/backfill → 按可见集裁剪目标集，
//     越界行剔除 + 计数入 operlog，不因含越界行 403 整批；
//   - All / 0 / unset（含未挂注入中间件）→ 短路放行，超管零回归、零 DB 查询；
//   - FailClosed（异常枚举）→ 越界；
//   - Custom/Dept/DeptChild → 「有效机构 ∈ DeptIDs」放行；
//   - Self（HookSelfUser）→ workstation 有 user_id 列可放行本人工位，其余实体
//     无用户列 → 越界（fail-closed）；
//   - 空 orgID：必须按「继承楼宇/楼层有效机构」解析后再判定；无法解析 → 越界。
//
// 与 Phase 131 scope_filter.go 读侧基建的关系：复用 scopeFromContext /
// scopeIsAll / orgScopeClause / selfUserClause / effectiveOrgExpr 单一真实源，
// 本文件只补「写」语义（拒绝 vs 裁剪），不重写读侧过滤。
//
// 403 陷阱（agent-8 审查 P0-2）：pkg/errors/codes.go:205 把 1020-1029 段
// （含 CodePermissionDenied=1025）的 DefaultHTTPStatus 映射为 401，
// codes_80_04_test.go:47 已把 401 锁成「预期行为」。故 ErrScopeDenied 必须用
// NewWithHTTPStatus 显式指定 http.StatusForbidden；response.HandleServiceError
// 只对 IsAppError 的错误透传其 HTTPStatus，裸 fmt.Errorf 会退化为 500。
package operations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// ErrScopeDenied 是四域写端点「越界」的统一哨兵错误。
//
// 显式 HTTP 403（见文件头 403 陷阱说明）；返回同一 *AppError 指针，
// errors.Is(err, ErrScopeDenied) 可稳定判定；文案四域统一。
var ErrScopeDenied = apperrors.NewWithHTTPStatus(
	apperrors.CodePermissionDenied, http.StatusForbidden, "无权操作该数据",
)

// IsScopeDenied 判定错误是否为「写越界」（errors.Is 兼容 wrap 后的错误）。
func IsScopeDenied(err error) bool {
	return errors.Is(err, ErrScopeDenied)
}

// scopeWriteResult 抽象一次「可见集判定」的结果，供 check / filter 共用。
type scopeWriteResult struct {
	all             bool // All / 未注入 → 完全放行、零 DB 查询
	deny            bool // FailClosed / Self 无用户列 / 空集 → 拒绝或空集
	deptIDs         []string
	selfUserID      string // Self 档且有用户列时非空
	userColRequired bool   // Self 档需要按用户列过滤
}

// classifyWriteScope 把 ScopeResult 归一成写侧判定所需的最小三态。
// entityHasUserCol 表示目标实体是否有「用户归属列」（workstation=true，
// floor/building/infopoint=false），Self 档据此分派。
func classifyWriteScope(ctx context.Context, entityHasUserCol bool) scopeWriteResult {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return scopeWriteResult{all: true}
	}
	if scope.FailClosed {
		return scopeWriteResult{deny: true}
	}
	if scope.ResourceHook == datascope.HookSelfUser {
		// Self：部门集为空，按用户列过滤；无用户列实体 / 缺 UserID → fail-closed。
		if !entityHasUserCol || scope.UserID == "" {
			return scopeWriteResult{deny: true}
		}
		return scopeWriteResult{
			selfUserID:      scope.UserID,
			userColRequired: true,
		}
	}
	if len(scope.DeptIDs) == 0 {
		return scopeWriteResult{deny: true}
	}
	return scopeWriteResult{deptIDs: scope.DeptIDs}
}

// orgInVisibleSet 判定「有效机构」是否落在可见部门集内（O(1) 集合查表）。
// 空 orgID 视为无法解析 → false（fail-closed）。
func orgInVisibleSet(orgID string, deptIDs []string) bool {
	if orgID == "" {
		return false
	}
	for _, id := range deptIDs {
		if id == orgID {
			return true
		}
	}
	return false
}

// checkWriteScope 校验「提交的 orgID」是否在可见集内，越界返回 ErrScopeDenied。
//
// 仅适用于 orgID 已经是「有效机构」的场景（building 直挂 org_id）。
// orgID 为空时：调用方须先用 checkWriteScopeEffective 传入按归属链解析出的
// 有效机构；直接传空串会按「无法解析」fail-closed 拒绝，绝不猜测或回退 All。
//
// entityHasUserCol 为可选变参：workstation 传 true 表示 Self 档放行本人工位；
// 其余实体省略（false），Self 档一律拒绝。保持任务书 checkWriteScope(ctx, orgID)
// 的两参签名可用。
func checkWriteScope(ctx context.Context, orgID string, entityHasUserCol ...bool) error {
	hasUserCol := len(entityHasUserCol) > 0 && entityHasUserCol[0]
	r := classifyWriteScope(ctx, hasUserCol)
	if r.all {
		return nil
	}
	if r.deny {
		return ErrScopeDenied
	}
	if r.userColRequired {
		// Self 档有用户列：单条写是否放行由「资源归属链上的用户列 = 登录人」
		// 决定，不能仅凭 orgID 判定。调用方应使用 checkWriteScopeSelfUser 或
		// filterVisibleIDs（带 userCol）在资源级校验；此处保守拒绝，
		// 避免「提交任意 orgID 即放行」造成越权。
		return ErrScopeDenied
	}
	if !orgInVisibleSet(orgID, r.deptIDs) {
		return ErrScopeDenied
	}
	return nil
}

// checkWriteScopeEffective 校验「按归属链解析后的有效机构」是否在可见集内。
//
// 用于空 orgID 的场景：调用方（如 floor Create/Update）先按
// ops_floors.org_id 优先、NULL/空串继承 ops_buildings.org_id 解析出
// effectiveOrgID，再交给本函数判定；解析不出（空串）→ ErrScopeDenied。
func checkWriteScopeEffective(ctx context.Context, effectiveOrgID string, entityHasUserCol ...bool) error {
	return checkWriteScope(ctx, effectiveOrgID, entityHasUserCol...)
}

// checkWriteScopeSelfUser 校验 Self 档「资源归属用户列」是否等于登录人。
// 非 Self 档（All/DeptIDs/FailClosed）不应调用本函数；All 放行，其余拒绝。
func checkWriteScopeSelfUser(ctx context.Context, resourceUserID string) error {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return nil
	}
	if scope.ResourceHook == datascope.HookSelfUser && !scope.FailClosed &&
		scope.UserID != "" && resourceUserID != "" && resourceUserID == scope.UserID {
		return nil
	}
	return ErrScopeDenied
}

// filterVisibleIDs 批量裁剪：一次性带 `WHERE id IN ? AND <orgExpr> IN ?`
// 取回可见集内的 ID 子集，在 service 层、任何写 SQL 之前调用。
//
// 硬约束（裁定 §2）：禁止「先取全量 ID 再在内存过滤后写」——后者一旦过滤
// 实现有 bug 仍会写出可见集外的行；本函数让 DB 自己按可见集条件筛，
// 返回的 ID 天然全部可见。
//
// 参数：
//   - table：目标表名（编译期常量白名单见 filterVisibleIDsAllowedTables）；
//   - ids：请求的全部目标 ID（空 → 直接返回空，零查询）；
//   - orgExpr：effectiveOrgExpr(...) 的结果（单一真实源，禁止内联复制）；
//     若查询作用域内需要 JOIN（f/b 别名），调用方应在 orgExpr 里带上
//     已 JOIN 的别名，并保证 filterVisibleIDs 生成的 SQL 能引用到它们——
//     四域统一用 scopedVisibleSub 预先构造好带 JOIN 的 EXISTS 骨架更稳妥，
//     故本函数对「本表可直接引用 orgExpr」的形态（building.org_id）使用，
//     跨表形态请用 filterVisibleIDsByExists。
//   - userCol：Self 档按用户列过滤的列名（本表列，如 "user_id"）；无用户列
//     实体传 ""，Self 档直接返回空集。
//
// 返回 (可见 ID 列表, error)。All / 未注入短路：零 DB 查询、原样返回 ids
// （超管零回归）。FailClosed / 空集 → 返回空切片。
func filterVisibleIDs(ctx context.Context, db *gorm.DB, table string, ids []string, orgExpr string, userCol ...string) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	if !filterVisibleIDsAllowedTables[table] {
		return nil, apperrors.New(apperrors.CodeParamError, "非法的批量裁剪表名")
	}
	selfCol := ""
	if len(userCol) > 0 {
		selfCol = userCol[0]
	}
	r := classifyWriteScope(ctx, selfCol != "")
	if r.all {
		return append([]string(nil), ids...), nil
	}
	if r.deny {
		return []string{}, nil
	}

	q := db.WithContext(ctx).Table(table).
		Select("DISTINCT CAST(id AS TEXT)").
		Where("id IN ?", ids)
	if r.userColRequired {
		q = q.Where(selfCol+" = ?", r.selfUserID)
	} else {
		// orgScopeClause 复用读侧单一真实源：All/未注入 skip、FailClosed 1=0、
		// DeptIDs 非空 → CAST(orgExpr AS TEXT) IN ?。
		cond, args, skip := orgScopeClauseFromResult(r, orgExpr)
		if skip {
			return append([]string(nil), ids...), nil
		}
		q = q.Where(cond, args...)
	}
	var visible []string
	if err := q.Pluck("CAST(id AS TEXT)", &visible).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "批量裁剪可见集失败")
	}
	return visible, nil
}

// filterVisibleIDsByExists 跨表「有效机构」形态的批量裁剪：一次性
// `WHERE id IN ? AND EXISTS (<已挂可见集条件的子查询>)`。
//
// 供 workstation / floor / infopoint 使用：这些实体的有效机构横跨
// ops_floors(f) 与 ops_buildings(b)，列引用只在 EXISTS 子查询作用域内合法。
// sub 由调用方按 wrapScopeExists 同一骨架构造（表别名 + 实体关联条件 +
// f/b 软删过滤），本函数只负责追加可见集条件并组合 id IN。
//
// subUserCol：子查询内可引用的用户列（workstation 传 "sys_workstation.user_id"
// 或 "w.user_id"；无用户列实体传 ""）。
func filterVisibleIDsByExists(ctx context.Context, db *gorm.DB, table string, ids []string, sub *gorm.DB, orgExpr, subUserCol string) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	if !filterVisibleIDsAllowedTables[table] {
		return nil, apperrors.New(apperrors.CodeParamError, "非法的批量裁剪表名")
	}
	// 先用 classifyWriteScope 做无歧义的三态判定：
	//   - All/未注入 → 零查询原样放行；
	//   - FailClosed / Self 无用户列 / 空集 → 空集（不依赖 applyScopeToSub 的
	//     skip 语义，避免把 Self 无用户列误当 All 放行，写侧必须 fail-closed）。
	r := classifyWriteScope(ctx, subUserCol != "")
	if r.all {
		return append([]string(nil), ids...), nil
	}
	if r.deny {
		return []string{}, nil
	}
	// 复用读侧 wrapScopeExists 的 Self / FailClosed / 空集分派生成 scopedSub，
	// 保证「写侧裁剪」与「读侧过滤」共用同一段 CAST/IN/selfUser 条件。
	scopedSub, skip := applyScopeToSub(sub, ctx, orgExpr, subUserCol)
	if skip {
		return []string{}, nil
	}
	q := db.WithContext(ctx).Table(table).
		Select("DISTINCT CAST(id AS TEXT)").
		Where("id IN ?", ids).
		Where("EXISTS (?)", scopedSub)
	var visible []string
	if err := q.Pluck("CAST(id AS TEXT)", &visible).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "批量裁剪可见集失败")
	}
	return visible, nil
}

// orgScopeClauseFromResult 把 classifyWriteScope 的结果转回 orgScopeClause
// 所需的 ScopeResult，保证「写侧」与「读侧」共用同一段 CAST/IN 条件生成，
// 不内联复制 COALESCE/NULLIF 表达式（裁定 §2 硬约束）。
func orgScopeClauseFromResult(r scopeWriteResult, orgExpr string) (string, []any, bool) {
	scope := datascope.ScopeResult{
		All:          r.all,
		DeptIDs:      r.deptIDs,
		FailClosed:   r.deny,
		ResourceHook: datascope.HookNone,
	}
	return orgScopeClause(scope, true, orgExpr)
}

// filterVisibleIDsAllowedTables 防 SQL 注入：table 必须来自编译期白名单。
var filterVisibleIDsAllowedTables = map[string]bool{
	"ops_buildings":   true,
	"ops_floors":      true,
	"sys_workstation": true,
	"ops_info_points": true,
}

// visibleIDsForEntityWrite 四域批量写（BatchDelete / BatchOperation /
// BatchUpdatePositions / backfill）统一的「service 层可见集裁剪」入口。
//
// 硬约束（裁定 §2）：在任何写 SQL 之前、service 层一次性带
// `WHERE id IN ? AND <有效机构 ∈ 可见集>` 取回可见 ID 子集；
// 禁止「先取全量再内存过滤」（过滤 bug 会写出越界行）。
//
// 返回 (可见 ID, 越界行数, error)。越界行数 = len(ids) - len(可见 ID)，
// 供 operlog 记「请求行数 / 可见集内行数 / 越界行数」（裁定 §2 硬约束 3）。
// All / 未注入 → 零查询原样返回、越界行数 0（超管零回归）。
func visibleIDsForEntityWrite(ctx context.Context, db *gorm.DB, entity string, ids []string) ([]string, int, error) {
	if len(ids) == 0 {
		return []string{}, 0, nil
	}
	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	var visible []string
	var err error
	switch entity {
	case scopeWriteEntityBuilding:
		// 楼宇：可见 = 自身 org_id ∈ 可见集 OR 其下任一楼层有效机构 ∈ 可见集
		// （与 applyBuildingScope 读侧同口径，一栋楼跨机构办公）。
		visible, err = filterVisibleBuildingIDs(ctx, db, ids)
	case scopeWriteEntityFloor:
		// 与读侧同款骨架：f/b 两个别名都在 EXISTS 子查询作用域内，
		// 子查询按 f.id 关联外层 ops_floors.id，避免相关引用歧义。
		sub := db.Session(&gorm.Session{NewDB: true}).
			Table("ops_floors f").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(f.id AS TEXT) = CAST(ops_floors.id AS TEXT)").
			Where("f.deleted_at IS NULL AND b.deleted_at IS NULL").
			Select("1")
		visible, err = filterVisibleIDsByExists(ctx, db, "ops_floors", ids, sub, orgExpr, "")
	case scopeWriteEntityWorkstation:
		sub := workstationOrgExistsSub(db, "CAST(f.id AS TEXT) = sys_workstation.floor_id")
		visible, err = filterVisibleIDsByExists(ctx, db, "sys_workstation", ids, sub, orgExpr, "sys_workstation.user_id")
	case scopeWriteEntityInfoPoint:
		sub := db.Session(&gorm.Session{NewDB: true}).
			Table("sys_workstation w").
			Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(w.id AS TEXT) = ops_info_points.workstation_id").
			Where("w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL").
			Select("1")
		visible, err = filterVisibleIDsByExists(ctx, db, "ops_info_points", ids, sub, orgExpr, "")
	default:
		return nil, 0, apperrors.New(apperrors.CodeParamError, "非法的写裁剪实体")
	}
	if err != nil {
		return nil, 0, err
	}
	return visible, len(ids) - len(visible), nil
}

// filterVisibleBuildingIDs 楼宇可见集裁剪：沿用 applyBuildingScope 的双分支
// 语义（自身 org_id ∈ 可见集 OR 其下楼层有效机构 ∈ 可见集），但以 ScopeResult
// 为唯一输入并在 id IN 之前一次性完成，All 短路零查询。
func filterVisibleBuildingIDs(ctx context.Context, db *gorm.DB, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return []string{}, nil
	}
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return append([]string(nil), ids...), nil
	}
	if scope.FailClosed || scope.ResourceHook == datascope.HookSelfUser || len(scope.DeptIDs) == 0 {
		return []string{}, nil
	}
	orgExpr := effectiveOrgExpr("f.org_id", "ops_buildings.org_id")
	cond := "CAST(" + orgExpr + " AS TEXT) IN ?"
	buildingCond := "CAST(ops_buildings.org_id AS TEXT) IN ?"
	q := db.WithContext(ctx).Table("ops_buildings").
		Select("DISTINCT CAST(id AS TEXT)").
		Where("id IN ?", ids).
		Where("("+buildingCond+` OR EXISTS (
			SELECT 1 FROM ops_floors f
			WHERE CAST(f.building_id AS TEXT) = CAST(ops_buildings.id AS TEXT)
			  AND `+cond+`
			  AND f.deleted_at IS NULL
		))`, scope.DeptIDs, scope.DeptIDs)
	var visible []string
	if err := q.Pluck("CAST(id AS TEXT)", &visible).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "批量裁剪楼宇可见集失败")
	}
	return visible, nil
}

// scopeWriteEntity 四域写端点实体名（编译期常量，防 SQL 注入）。
const (
	scopeWriteEntityBuilding    = "building"
	scopeWriteEntityFloor       = "floor"
	scopeWriteEntityWorkstation = "workstation"
	scopeWriteEntityInfoPoint   = "infopoint"
)

// 导出实体名常量供 handler 层 operlog 计数调用（值与包内常量一致）。
const (
	ScopeWriteEntityBuilding    = scopeWriteEntityBuilding
	ScopeWriteEntityFloor       = scopeWriteEntityFloor
	ScopeWriteEntityWorkstation = scopeWriteEntityWorkstation
	ScopeWriteEntityInfoPoint   = scopeWriteEntityInfoPoint
)

// ScopeWriteTier 返回当前 ctx 的范围档位字符串（operlog / 前端文案共用）：
// "All" / "Custom/Dept/DeptChild" / "Self" / "FailClosed" / "All"(未注入兜底)。
func ScopeWriteTier(ctx context.Context) string {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return "All"
	}
	if scope.FailClosed {
		return "FailClosed"
	}
	if scope.ResourceHook == datascope.HookSelfUser {
		return "Self"
	}
	if len(scope.DeptIDs) > 0 {
		return "Custom/Dept/DeptChild"
	}
	return "FailClosed"
}

// scopeWriteWorkstationVisibleSQL 返回「工位有效机构 ∈ 可见集」的 SQL 条件
// 片段与参数，供 backfill 候选查询 / COUNT / CASE UPDATE 三层复用（裁定 §2
// 要求三层都注入可见集，且预览与执行由同一 ScopeResult 快照决定）。
//
// 返回 (cond, args, skipAll)：skipAll=true 表示 All / 未注入 → 不挂条件
// （零额外 SQL、超管零回归）；FailClosed / Self 无用户列 / 空集 → cond="1=0"。
// Self 档 workstation 有 user_id 列 → 按 w.user_id = UserID 过滤。
//
// wAlias 为 sys_workstation 在查询中的别名（如 "w" 或 "sys_workstation"）。
func scopeWriteWorkstationVisibleSQL(ctx context.Context, wAlias string) (string, []any, bool) {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return "", nil, true
	}
	if scope.FailClosed {
		return "1=0", nil, false
	}
	if scope.ResourceHook == datascope.HookSelfUser {
		if scope.UserID == "" {
			return "1=0", nil, false
		}
		return "CAST(" + wAlias + ".user_id AS TEXT) = ?", []any{scope.UserID}, false
	}
	if len(scope.DeptIDs) == 0 {
		return "1=0", nil, false
	}
	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	// 与读侧 wrapScopeExists 同款 EXISTS 骨架：工位 → 楼层 f → 楼宇 b，
	// 有效机构按 CAST(... AS TEXT) IN 可见集（双侧 CAST，PG/SQLite 双方言）。
	cond := "EXISTS (SELECT 1 FROM ops_floors f " +
		"JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id " +
		"WHERE CAST(f.id AS TEXT) = " + wAlias + ".floor_id " +
		"AND f.deleted_at IS NULL AND b.deleted_at IS NULL " +
		"AND CAST(" + orgExpr + " AS TEXT) IN ?)"
	return cond, []any{scope.DeptIDs}, false
}

// scopeWriteBackfillSnapshot 生成 backfill 的「范围快照」回显字段：
// 档位 / 可见部门名（前 3）/ 部门总数 / FailClosed。预览与执行两次请求
// 各自从同一 ctx 的 ScopeResult 派生，前端确认框据此展示「本次范围」，
// 禁止确认阶段重新解析用户档位（裁定 §3 防范围漂移）。
func scopeWriteBackfillSnapshot(ctx context.Context, db *gorm.DB) (tier string, deptNames []string, deptTotal int, failClosed bool) {
	scope, ok := scopeFromContext(ctx)
	tier = ScopeWriteTier(ctx)
	failClosed = !scopeIsAll(scope, ok) && scope.FailClosed
	if scope.ResourceHook == datascope.HookSelfUser && !failClosed {
		return tier, nil, 0, false
	}
	if len(scope.DeptIDs) == 0 {
		return tier, nil, 0, failClosed
	}
	deptTotal = len(scope.DeptIDs)
	// 只查前 3 个部门名用于前端文案（不逐条回显越界集合，P1-5 侧信道收敛）。
	limit := 3
	if len(scope.DeptIDs) < limit {
		limit = len(scope.DeptIDs)
	}
	var names []string
	if err := db.WithContext(ctx).Table("sys_dept").
		Select("dept_name").
		Where("CAST(id AS TEXT) IN ?", scope.DeptIDs[:limit]).
		Pluck("dept_name", &names).Error; err != nil {
		// 部门名查不到不阻断回填，前端退化为「N 个部门」文案。
		return tier, nil, deptTotal, failClosed
	}
	return tier, names, deptTotal, failClosed
}

// ScopeWriteBatchStats 供 handler 层 operlog 记录「范围档位 / 请求行数 /
// 可见集内行数 / 越界行数」（裁定 §2 硬约束 3/4）。
//
// 复用与 service 裁剪完全相同的 visibleIDsForEntityWrite，保证 operlog 计数
// 与实际写入集合一致；All 档短路零查询、越界行数 0（审计链同样完整）。
func ScopeWriteBatchStats(ctx context.Context, db *gorm.DB, entity string, ids []string) (tier string, requested, visibleCount, outOfScope int, err error) {
	tier = ScopeWriteTier(ctx)
	requested = len(ids)
	visible, outOfScope, err := visibleIDsForEntityWrite(ctx, db, entity, ids)
	if err != nil {
		return tier, requested, 0, 0, err
	}
	return tier, requested, len(visible), outOfScope, nil
}

// scopeWriteImportVerdict 是 Excel 导入逐行可见集判定的结果。
type scopeWriteImportVerdict struct {
	skipAll bool            // All / 未注入 → 零校验、零查询（超管零回归）
	denyAll bool            // FailClosed → 全部越界
	set     map[string]bool // 可见部门集（O(1) 判定，裁定 §6）
	selfUID string          // Self 档登录人（工位按 user_id 判定）
	selfCol bool            // 目标实体是否有用户列（仅 workstation）
}

// newScopeWriteImportVerdict 从 ctx 的 ScopeResult 构造导入判定器（一次解析，
// 供整批所有行复用，避免逐行查库 / 逐行解析档位）。
func newScopeWriteImportVerdict(ctx context.Context, entityType string) scopeWriteImportVerdict {
	scope, ok := scopeFromContext(ctx)
	if scopeIsAll(scope, ok) {
		return scopeWriteImportVerdict{skipAll: true}
	}
	v := scopeWriteImportVerdict{
		denyAll: scope.FailClosed,
		selfCol: entityType == "workstation",
		selfUID: scope.UserID,
		set:     make(map[string]bool, len(scope.DeptIDs)),
	}
	for _, id := range scope.DeptIDs {
		v.set[id] = true
	}
	if scope.ResourceHook == datascope.HookSelfUser {
		// Self 档部门集为空：无用户列实体全部越界，有用户列实体按 user_id。
		v.denyAll = !v.selfCol || v.selfUID == ""
	}
	return v
}

// scopeWriteImportRowOK 判定单行解析后的有效机构 / dept_id 是否在可见集内。
//
// 参数（均为引用解析链已经落定的值，见 excel_config 的 DBField）：
//   - effOrgID：楼层覆盖优先的有效机构（building 的 org_id / floor 继承 /
//     workstation/infopoint 经归属链解析），uuid/varchar 比较已由调用方
//     统一 CAST AS TEXT 处理；
//   - deptID：工位导入的 dept_id（可空）；
//   - userID：工位导入的 user_id（Self 档按本人判定，可空）。
//
// 语义（裁定 §4）：越界返回 false，该行不写入、不阻断整批，计入 ImportResult
// 错误明细（含行号、不含敏感值）。空 effOrgID 且无法由归属链解析 → false
// （fail-closed）。All 档短路返回 true（超管零回归）。
func scopeWriteImportRowOK(v scopeWriteImportVerdict, effOrgID, deptID, userID string) bool {
	if v.skipAll {
		return true
	}
	if v.denyAll {
		return false
	}
	if v.selfCol && v.selfUID != "" {
		// Self 档 workstation：只放行本人工位（同 Self 裁定 §2）。
		return userID != "" && userID == v.selfUID
	}
	// Custom/Dept/DeptChild：有效机构必须命中可见集；工位 dept_id 非空时
	// 也必须命中（防把工位挂到越界部门）。集合 map O(1) 判定（裁定 §6）。
	if effOrgID == "" || !v.set[effOrgID] {
		return false
	}
	if deptID != "" && !v.set[deptID] {
		return false
	}
	return true
}

// scopeWriteStr 从导入行 map 里取字符串值（兼容 nil / 非 string）。
func scopeWriteStr(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return strings.TrimSpace(s)
}

// scopeWriteImportRowNum 取回解析阶段打的原始 Excel 行号；缺省用
// config.StartRow + i 兜底（与 excel_service.go:1304 既有行号口径一致）。
func scopeWriteImportRowNum(m map[string]any, i, startRow int) int {
	if v, ok := m["__excel_row__"]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return startRow + i
}

// scopeWriteResolveEffOrg 批量解析导入行的「有效机构」（一次 SQL，不逐行查库）。
//
// 复用 effectiveOrgExpr 单一真实源（楼层覆盖优先、NULL/空串继承楼宇），
// 双侧 CAST AS TEXT 防 uuid/varchar 漂移；按实体归属链：
//   - building：行内 org_id 直接作为有效机构；
//   - floor：行内 org_id 非空则用之，否则按 building_id 查 ops_buildings.org_id；
//   - workstation：按 floor_id → ops_floors/ops_buildings 有效机构；
//   - infopoint：按 workstation_id → workstation → floor → building 有效机构。
//
// 返回与 records 等长的有效机构切片；断链 / 查不到 → ""（fail-closed）。
func scopeWriteResolveEffOrg(ctx context.Context, db *gorm.DB, entityType string, records []map[string]any) ([]string, error) {
	out := make([]string, len(records))
	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	// 收集需要查库的 FK 值（去重），一次 IN 查询。
	byFK := make(map[string][]int)
	switch entityType {
	case "building":
		for i, rec := range records {
			out[i] = scopeWriteStr(rec, "org_id")
		}
		return out, nil
	case "floor":
		for i, rec := range records {
			if org := scopeWriteStr(rec, "org_id"); org != "" {
				out[i] = org
				continue
			}
			fk := scopeWriteStr(rec, "building_id")
			if fk != "" {
				byFK[fk] = append(byFK[fk], i)
			}
		}
	case "workstation":
		for i, rec := range records {
			fk := scopeWriteStr(rec, "floor_id")
			if fk != "" {
				byFK[fk] = append(byFK[fk], i)
			}
		}
	case "infoPoint":
		for i, rec := range records {
			fk := scopeWriteStr(rec, "workstation_id")
			if fk != "" {
				byFK[fk] = append(byFK[fk], i)
			}
		}
	default:
		// 非四域实体（department/user/asset 等）不在 Phase 132 收口范围，
		// 返回空串由调用方 skipAll 判定；不擅自拒绝，避免扩大作用域。
		return out, nil
	}
	if len(byFK) == 0 {
		return out, nil
	}
	fks := make([]string, 0, len(byFK))
	for fk := range byFK {
		fks = append(fks, fk)
	}

	type fkOrg struct {
		FK      string `gorm:"column:fk"`
		EffOrg  string `gorm:"column:eff_org"`
	}
	var rows []fkOrg
	var q *gorm.DB
	switch entityType {
	case "floor":
		q = db.WithContext(ctx).Table("ops_buildings b").
			Where("CAST(b.id AS TEXT) IN ? AND b.deleted_at IS NULL", fks).
			Select("CAST(b.id AS TEXT) AS fk, CAST(b.org_id AS TEXT) AS eff_org")
	case "workstation":
		q = db.WithContext(ctx).Table("ops_floors f").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(f.id AS TEXT) IN ? AND f.deleted_at IS NULL AND b.deleted_at IS NULL", fks).
			Select("CAST(f.id AS TEXT) AS fk, CAST(" + orgExpr + " AS TEXT) AS eff_org")
	case "infoPoint":
		q = db.WithContext(ctx).Table("sys_workstation w").
			Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(w.id AS TEXT) IN ? AND w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", fks).
			Select("CAST(w.id AS TEXT) AS fk, CAST(" + orgExpr + " AS TEXT) AS eff_org")
	}
	if q == nil {
		return out, nil
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.CodeDatabaseError, "解析导入行有效机构失败")
	}
	resolved := make(map[string]string, len(rows))
	for _, r := range rows {
		resolved[r.FK] = r.EffOrg
	}
	for fk, idxs := range byFK {
		for _, i := range idxs {
			out[i] = resolved[fk]
		}
	}
	return out, nil
}

// filterScopeWriteImportRows 是 Excel 导入逐行可见集校验的唯一落点
// （裁定 §4）：在 Upsert 之前（事务内 prepare records 阶段）对每行解析出的
// org_id（楼层覆盖优先的有效机构）/ dept_id 过可见集；越界行不写入、
// 不阻断整批，计入 ImportResult 越界错误明细（含行号，不含敏感值 Value）。
//
// 复用 effectiveOrgExpr 单一真实源（scopeWriteResolveEffOrg 内）、集合 map
// 做 O(1) 判定避免逐行查库（裁定 §6）。All 档短路：零校验、零查询、零回归。
// 返回 (可见行切片, 越界行数)。
func filterScopeWriteImportRows(ctx context.Context, db *gorm.DB, entityType string, config ExcelConfig, records []map[string]any, result *ImportResult) ([]map[string]any, int, error) {
	v := newScopeWriteImportVerdict(ctx, entityType)
	if v.skipAll {
		// All 档也要清掉内部行号标记，避免误入 Upsert 字段集。
		for _, rec := range records {
			delete(rec, "__excel_row__")
		}
		return records, 0, nil
	}
	effOrgs, err := scopeWriteResolveEffOrg(ctx, db, entityType, records)
	if err != nil {
		return nil, 0, err
	}
	kept := make([]map[string]any, 0, len(records))
	outOfScope := 0
	for i, rec := range records {
		rowNum := scopeWriteImportRowNum(rec, i, config.StartRow)
		effOrg := effOrgs[i]
		deptID := scopeWriteStr(rec, "dept_id")
		userID := scopeWriteStr(rec, "user_id")
		if !scopeWriteImportRowOK(v, effOrg, deptID, userID) {
			// 越界：不写入、不阻断整批，只记聚合明细（含行号，不含敏感值）。
			outOfScope++
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowNum,
				Field: "org_id/dept_id",
				Error: "无权操作该数据：该行归属机构不在本次可见范围内",
			})
			delete(rec, "__excel_row__")
			continue
		}
		delete(rec, "__excel_row__")
		kept = append(kept, rec)
	}
	return kept, outOfScope, nil
}

// ScopeWriteOperParam 生成批量写 operlog 的 OperParam JSON 字符串，供四域
// handler 直接传给 operlog.WithOperParam（无需额外 fmt 导入）。
// 含 action / scopeTier / requested / visible / outOfScope 五个字段，全部
// 为聚合计数，不逐条回显越界 ID（裁定 §2）。统计查询失败时降级为
// 「档位 + 请求行数 + 全越界」的保守值，绝不静默丢日志。
func ScopeWriteOperParam(ctx context.Context, db *gorm.DB, entity, action string, ids []string) string {
	tier, requested, visibleCount, outOfScope, err := ScopeWriteBatchStats(ctx, db, entity, ids)
	if err != nil {
		tier = ScopeWriteTier(ctx)
		visibleCount, outOfScope = 0, requested
	}
	return fmt.Sprintf(
		`{"action":%q,"scopeTier":%q,"requested":%d,"visible":%d,"outOfScope":%d}`,
		action, tier, requested, visibleCount, outOfScope)
}

// scopeWriteHasUserCol 判定实体是否有「用户归属列」：按任务书 A 口径与
// Self 裁定 §2，四域中仅 workstation 有 sys_workstation.user_id 列；
// floor/building/infopoint 一律视为无用户列 → Self 档 fail-closed。
func scopeWriteHasUserCol(entity string) bool {
	return entity == scopeWriteEntityWorkstation
}

// scopeWriteOrgPtr 把楼层可空 org_id（*string）归一成字符串；
// nil / 空串 / 纯空白 → ""，交回 checkSubmittedWriteScope 按归属链解析
// （空串本身不会被当作有效机构放行，fail-closed）。
func scopeWriteOrgPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// scopeWritePtr 通用 *string → string（工位 floorId/userId 等可空外键）。
func scopeWritePtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// checkResourceWriteScope 校验「改前已存在资源」是否在可见集内
// （Delete / Update 双向校验的「改前」侧），越界返回 ErrScopeDenied。
//
// 按实体归属链解析有效机构（effectiveOrgExpr 单一真实源、双侧 CAST、软删
// 过滤、断链 → 空串 → fail-closed）：
//   - building：自身 org_id；
//   - floor：ops_floors.org_id 优先、空则继承 ops_buildings.org_id；
//   - workstation：经 floor_id → ops_floors/ops_buildings 有效机构；
//   - infopoint：经 workstation_id → workstation → floor → building 有效机构。
//
// Self 档 workstation：改按 sys_workstation.user_id = ScopeResult.UserID 判定
// （本人工位放行）；其余实体 Self 档一律拒绝。
func checkResourceWriteScope(ctx context.Context, db *gorm.DB, entity, id string) error {
	if id == "" {
		return ErrScopeDenied
	}
	r := classifyWriteScope(ctx, scopeWriteHasUserCol(entity))
	if r.all {
		return nil
	}
	if r.deny {
		return ErrScopeDenied
	}

	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	var row struct {
		EffOrg       string
		ResourceUser string
	}
	var q *gorm.DB
	switch entity {
	case scopeWriteEntityBuilding:
		q = db.WithContext(ctx).Table("ops_buildings b").
			Where("CAST(b.id AS TEXT) = ? AND b.deleted_at IS NULL", id).
			Select("CAST(b.org_id AS TEXT) AS eff_org, '' AS resource_user")
	case scopeWriteEntityFloor:
		q = db.WithContext(ctx).Table("ops_floors f").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(f.id AS TEXT) = ? AND f.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Select("CAST(" + orgExpr + " AS TEXT) AS eff_org, '' AS resource_user")
	case scopeWriteEntityWorkstation:
		q = db.WithContext(ctx).Table("sys_workstation w").
			Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(w.id AS TEXT) = ? AND w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Select("CAST(" + orgExpr + " AS TEXT) AS eff_org, CAST(w.user_id AS TEXT) AS resource_user")
	case scopeWriteEntityInfoPoint:
		q = db.WithContext(ctx).Table("ops_info_points ip").
			Joins("JOIN sys_workstation w ON CAST(w.id AS TEXT) = ip.workstation_id").
			Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
			Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
			Where("CAST(ip.id AS TEXT) = ? AND ip.deleted_at IS NULL AND w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", id).
			Select("CAST(" + orgExpr + " AS TEXT) AS eff_org, '' AS resource_user")
	default:
		return ErrScopeDenied
	}
	if err := q.Scan(&row).Error; err != nil {
		return apperrors.Wrap(err, apperrors.CodeDatabaseError, "校验资源归属失败")
	}
	// 资源不存在 / 归属链断链（无行）→ fail-closed。
	if row.EffOrg == "" && row.ResourceUser == "" {
		return ErrScopeDenied
	}
	if r.userColRequired {
		if row.ResourceUser == "" || row.ResourceUser != r.selfUserID {
			return ErrScopeDenied
		}
		return nil
	}
	if !orgInVisibleSet(row.EffOrg, r.deptIDs) {
		return ErrScopeDenied
	}
	return nil
}

// checkSubmittedWriteScope 校验「提交的资源归属」是否在可见集内
// （Create / Update 双向校验的「提交」侧），越界返回 ErrScopeDenied。
//
// 参数：
//   - submittedOrgID：直挂 org_id（building 的 OrgID / floor 的可空 OrgID）；
//     非空时它本身就是（或覆盖为）有效机构，无需再查库；
//   - fkID：归属链外键（floor→buildingID / workstation→floorID /
//     infopoint→workstationID），submittedOrgID 为空时按它回溯楼宇有效机构。
//
// 空 orgID 且无法由归属链解析 → 空串 → fail-closed（不回退 All、不猜测）。
// submittedUserID 为可选变参：Self 档 workstation 的「本人工位」判定依据，
// 提交 user_id == ScopeResult.UserID 时放行（Create / Update 提交侧）。
func checkSubmittedWriteScope(ctx context.Context, db *gorm.DB, entity, submittedOrgID, fkID string, submittedUserID ...string) error {
	r := classifyWriteScope(ctx, scopeWriteHasUserCol(entity))
	if r.all {
		return nil
	}
	if r.deny {
		return ErrScopeDenied
	}
	if r.userColRequired {
		// Self 档 workstation：按「提交的本人工位」放行（资源级 user_id 判定）；
		// 提交 user_id 为空或指向他人 → 拒。改前侧的归属由
		// checkResourceWriteScope 独立校验，二者共同构成双向防搬移。
		if scopeWriteHasUserCol(entity) && len(submittedUserID) > 0 &&
			submittedUserID[0] != "" && submittedUserID[0] == r.selfUserID {
			return nil
		}
		return ErrScopeDenied
	}

	orgExpr := effectiveOrgExpr("f.org_id", "b.org_id")
	effOrg := submittedOrgID
	if effOrg == "" {
		var resolved string
		var q *gorm.DB
		switch entity {
		case scopeWriteEntityBuilding:
			// building 无外键可回溯；空 orgID 保持空串 → 下方 fail-closed。
		case scopeWriteEntityFloor:
			// floor 的 fkID 是 buildingID：楼层 org_id 为空时直接继承
			// 目标楼宇 org_id（此时楼层尚未入库，不能按 f.id 查）。
			q = db.WithContext(ctx).Table("ops_buildings b").
				Where("CAST(b.id AS TEXT) = ? AND b.deleted_at IS NULL", fkID).
				Select("CAST(b.org_id AS TEXT)")
		case scopeWriteEntityWorkstation:
			q = db.WithContext(ctx).Table("ops_floors f").
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
				Where("CAST(f.id AS TEXT) = ? AND f.deleted_at IS NULL AND b.deleted_at IS NULL", fkID).
				Select("CAST(" + orgExpr + " AS TEXT)")
		case scopeWriteEntityInfoPoint:
			q = db.WithContext(ctx).Table("sys_workstation w").
				Joins("JOIN ops_floors f ON CAST(f.id AS TEXT) = w.floor_id").
				Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
				Where("CAST(w.id AS TEXT) = ? AND w.deleted_at IS NULL AND f.deleted_at IS NULL AND b.deleted_at IS NULL", fkID).
				Select("CAST(" + orgExpr + " AS TEXT)")
		default:
			return ErrScopeDenied
		}
		if q != nil {
			if err := q.Scan(&resolved).Error; err != nil {
				return apperrors.Wrap(err, apperrors.CodeDatabaseError, "解析提交归属失败")
			}
			effOrg = resolved
		}
	}
	if !orgInVisibleSet(effOrg, r.deptIDs) {
		return ErrScopeDenied
	}
	return nil
}
