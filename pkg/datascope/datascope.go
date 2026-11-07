// Package datascope 是 v1.52「运维域数据权限统一收口」的 ScopeProvider 单一真实源。
//
// 职责边界（LOCKED 裁定 .planning/notes/261107-v152-self-scope-decision.md §1）：
//   - 只统一「登录人可见部门集」的展开（All/Custom/Dept/DeptChild 四档 + 异常枚举），
//     不感知任何 ops 业务表名，Self/All 的资源语义通过 ResourceHook 交给各域自行注入。
//   - 0 / unset → All（超管零回归硬要求），严禁把 0 送进 fail-closed 分支。
//   - 异常枚举（非 0 且 ∉ 1..5）→ FailClosed=true + 空集。
//
// 各域 service 的读/写函数签名已含 ctx context.Context，通过 FromContext 取
// ScopeResult 后注入可见集过滤即可，无需改 handler 签名或再动 router.go。
package datascope

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// ResourceHook 资源级过滤钩子，供各域按自身语义注入过滤（ScopeProvider 不感知表名）。
type ResourceHook int

const (
	// HookNone 无资源级钩子：调用方按 DeptIDs 做「有效机构 ∈ 集合」过滤。
	HookNone ResourceHook = iota
	// HookSelfUser Self 档：按资源归属链上的用户列（= 登录人 UUID）过滤；
	// 实体无用户归属列时调用方应返回空集（fail-closed，见裁定 §2）。
	HookSelfUser
)

// ScopeResult 是 Resolve 的返回值，也是经 request context 向各域透传的数据范围快照。
//
// 语义（裁定 §1 / §2）：
//   - All=true：All/0/unset 特判短路，调用方不加机构过滤；零 DB 查询。
//   - DeptIDs：Custom/Dept/DeptChild 展开后的可见部门 ID 集；Self/异常枚举为空。
//   - ResourceHook：HookNone | HookSelfUser（仅 Self 档置 HookSelfUser）。
//   - FailClosed：异常枚举（非 0 且 ∉ 1..5）置 true，调用方读返回空 / 写 403。
//   - UserID：登录人 UUID 原样透传（Self 档按用户列过滤、131/132 缓存 key 维度需要）；
//     All/0 短路同样填值（超管未注入时允许为空串，非超管 All(1) 必有值）。
type ScopeResult struct {
	All          bool
	DeptIDs      []string
	ResourceHook ResourceHook
	FailClosed   bool
	UserID       string
}

// ---------------------------------------------------------------------------
// context 透传（Lead 2026-11-07 补充约定）：DataScopePermission 把 ScopeResult
// 注入 c.Request 的 context，131/132 只需 ctx 取值。私有 typed key 防跨包键冲突。
// ---------------------------------------------------------------------------

type scopeContextKey struct{}

// WithContext 把 ScopeResult 挂到 ctx 上，返回派生 context。
func WithContext(ctx context.Context, r ScopeResult) context.Context {
	return context.WithValue(ctx, scopeContextKey{}, r)
}

// FromContext 从 ctx 取 ScopeResult；未注入时返回 (零值, false)。
func FromContext(ctx context.Context) (ScopeResult, bool) {
	if ctx == nil {
		return ScopeResult{}, false
	}
	r, ok := ctx.Value(scopeContextKey{}).(ScopeResult)
	return r, ok
}

// AllScope 构造超管 / 0 值 / unset 的特判短路结果（零 DB 查询）。
func AllScope(userID string) ScopeResult {
	return ScopeResult{All: true, UserID: userID}
}

// FailClosedScope 构造异常枚举的 fail-closed 结果。
func FailClosedScope(userID string) ScopeResult {
	return ScopeResult{FailClosed: true, UserID: userID}
}

// ---------------------------------------------------------------------------
// 子树展开（唯一实现）
// ---------------------------------------------------------------------------

// queryDeptChildIDs 单条 SQL 展开本部门及子树 ID 集。

// Resolve 解析登录人的可见部门集（ScopeProvider 单一真实源）。
//
// 分档（scope=models.DataScope）：
//   - DataScopeAll(1) / 0 / unset：All=true 且零 DB 查询（特判短路，调用方不加过滤）。
//   - DataScopeCustom(2)：sys_role_dept ⋈ sys_user_role（仅启用角色）部门集。
//   - DataScopeDept(3)：{用户 dept_id}（查 sys_user.dept_id）。
//   - DataScopeDeptChild(4)：本部门 + 子树（复用 deptChildCondition 四/五条件语义）。
//   - DataScopeSelf(5)：DeptIDs 空集 + ResourceHook=HookSelfUser。
//   - 异常枚举（非 0 且 ∉ 1..5）：FailClosed=true + 空集 + HookNone。
//
// 所有路径都把 userID 原样写入 ScopeResult.UserID。空 userID 时除 All/0 外返回空集
// （调用方按 1=0 应用），不额外查库。
func Resolve(ctx context.Context, db *gorm.DB, userID string, scope models.DataScope) (ScopeResult, error) {
	base := ScopeResult{UserID: userID}

	// All / 0 / unset：特判短路，零 DB 查询（超管零回归）。
	// 注意：短路是「不查部门集」，不是「跳过 scope 段」——UserID 仍透传，
	// 调用方缓存 key 必须继续含 scope 维度（PMODEL-01 教训）。
	if scope == models.DataScopeAll || scope == 0 {
		base.All = true
		return base, nil
	}

	switch scope {
	case models.DataScopeCustom:
		if userID == "" {
			base.FailClosed = true
			return base, nil
		}
		ids, err := queryCustomDeptIDs(ctx, db, userID)
		if err != nil {
			return base, err
		}
		base.DeptIDs = normalizeIDs(ids)
		if len(base.DeptIDs) == 0 {
			// 无可见部门 → fail-closed 空集，不放行全库。
			base.FailClosed = true
		}
		return base, nil

	case models.DataScopeDept:
		if userID == "" {
			base.FailClosed = true
			return base, nil
		}
		deptID, err := queryUserDeptID(ctx, db, userID)
		if err != nil {
			return base, err
		}
		if deptID != "" {
			base.DeptIDs = []string{deptID}
		} else {
			base.FailClosed = true
		}
		return base, nil

	case models.DataScopeDeptChild:
		if userID == "" {
			base.FailClosed = true
			return base, nil
		}
		deptID, err := queryUserDeptID(ctx, db, userID)
		if err != nil {
			return base, err
		}
		if deptID == "" {
			base.FailClosed = true
			return base, nil
		}
		ids, err := queryDeptChildIDs(ctx, db, deptID)
		if err != nil {
			return base, err
		}
		base.DeptIDs = normalizeIDs(ids)
		if len(base.DeptIDs) == 0 {
			base.FailClosed = true
		}
		return base, nil

	case models.DataScopeSelf:
		// Self 不纳入部门集展开：空集 + 资源级钩子（裁定 §1/§2）。
		base.ResourceHook = HookSelfUser
		return base, nil

	default:
		// 异常枚举（非 0 且 ∉ 1..5）：fail-closed 空集。
		base.FailClosed = true
		return base, nil
	}
}

// normalizeIDs 去重 + 去空 + 排序，保证 ScopeResult 稳定（缓存 key 派生依赖稳定性）。
func normalizeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// queryCustomDeptIDs 查 sys_role_dept ⋈ sys_user_role（仅启用角色）的部门集。
// 双侧 CAST AS TEXT 防 uuid/varchar 漂移；status=0 过滤停用角色（对齐
// getUserMaxDataScope 只计 r.status=0 的语义）。
func queryCustomDeptIDs(ctx context.Context, db *gorm.DB, userID string) ([]string, error) {
	var ids []string
	err := db.WithContext(ctx).Raw(`
		SELECT DISTINCT CAST(rd.dept_id AS TEXT)
		FROM sys_role_dept rd
		INNER JOIN sys_user_role ur ON CAST(ur.role_id AS TEXT) = CAST(rd.role_id AS TEXT)
		INNER JOIN sys_role r ON CAST(r.id AS TEXT) = CAST(ur.role_id AS TEXT)
		WHERE CAST(ur.user_id AS TEXT) = ? AND r.status = ?
	`, userID, models.RoleStatusEnabled).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("查询自定义数据权限部门集失败: %w", err)
	}
	return ids, nil
}

// queryUserDeptID 查用户登录人所属部门 id。
// COALESCE 兜底 NULL：用户无部门时 Pluck 到 string 会因 NULL 扫描失败（SQLite/PG 同）。
func queryUserDeptID(ctx context.Context, db *gorm.DB, userID string) (string, error) {
	var deptID string
	err := db.WithContext(ctx).Raw(
		"SELECT COALESCE(CAST(dept_id AS TEXT), '') FROM sys_user WHERE CAST(id AS TEXT) = ?", userID,
	).Scan(&deptID).Error
	if err != nil {
		return "", fmt.Errorf("查询用户所属部门失败: %w", err)
	}
	return deptID, nil
}

// queryDeptChildIDs 单条 SQL 展开本部门及子树 ID 集（唯一实现，见 deptChildCondition）。
func queryDeptChildIDs(ctx context.Context, db *gorm.DB, deptID string) ([]string, error) {
	var ids []string
	// 与 BuildDeptRecursiveFilter 四条件语义对齐 + 链首 LIKE（生产 ancestors 无首尾逗号）。
	err := db.WithContext(ctx).Raw(`
		SELECT CAST(id AS TEXT) FROM sys_dept
		WHERE id = ?
		   OR ancestors = ?
		   OR ancestors LIKE ?
		   OR ancestors LIKE ?
		   OR ancestors LIKE ?
	`, deptID, deptID, deptID+",%", "%,"+deptID+",%", "%,"+deptID).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("展开本部门及子部门失败: %w", err)
	}
	return ids, nil
}
