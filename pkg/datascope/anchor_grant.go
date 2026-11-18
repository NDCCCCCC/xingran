// Package datascope：机构锚点授权的**准入保护**（v1.54 安全补丁）。
//
// 背景（verifier 于 2026-11-17 报告的阻塞级发现，已确认为权限提升漏洞）：
//
//	user_service.Create/Update 直接写 sys_user_post，对 req.PostIds **零校验**：
//	不校验岗位存在、不校验 status、更关键的是**不校验该岗位是否携带机构锚点**。
//	而 queryAnchorGrants 会消费任何 status=0 且 anchor_mode<>0 的岗位，按任职人
//	自己的部门链解析锚点并展开子树。
//
//	⇒ 攻击路径（已在代码上闭环）：
//	  1. 省管理员按正常业务创建岗位「分公司管理员」(anchor_mode=5, param="1")；
//	  2. 任一持 system:user:edit 的中支/部门管理员在自己的用户表单里把**自己**
//	     挂到该岗位 —— DataScopePermission 只限制数据面，不管岗位字典面；
//	  3. 下次请求解析到「某分公司」→ 展开全子树 → 可见集 = 全省。
//
//	即「一个只该看本部门的人，一次表单提交即可提权到全省」。
//
// 本文件提供准入校验：**授予携带锚点的岗位 = 授予数据范围，必须与角色授权同等严格**。
//
// 设计取舍（不做「静默跳过」而做「显式拒绝」）：
//   - 静默跳过会让管理员以为"已经挂上了"，实际没生效 —— 这与本功能反复强调的
//     "拒绝静默失败"原则冲突；
//   - 显式拒绝 + 中文原因，让越权尝试在界面上可见，同时天然留下 operlog 痕迹。
package datascope

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// PostGrantCandidate 一次岗位分配请求中的单个岗位及其锚点信息。
type PostGrantCandidate struct {
	PostID      string
	PostName    string
	Exist       bool // 岗位是否存在且未软删
	Status      models.PostStatus
	AnchorMode  models.AnchorMode
	AnchorParam string
}

// AnchorCarries 岗位是否携带机构锚点（会产出数据范围）。
func (c PostGrantCandidate) AnchorCarries() bool {
	return c.Exist && c.AnchorMode != models.AnchorModeNone && models.IsValidAnchorMode(c.AnchorMode)
}

// AnchorDescription 人类可读的锚点描述（错误提示与审计用）。
func (c PostGrantCandidate) AnchorDescription() string {
	switch c.AnchorMode {
	case models.AnchorModeDept:
		return "指定部门（含全部下级部门）"
	case models.AnchorModeExternalOrg:
		return "最近的外部机构（含全部下级）"
	case models.AnchorModeLevel:
		return fmt.Sprintf("机构层级 %s（含全部下级）", c.AnchorParam)
	default:
		return "机构锚点"
	}
}

// GrantDecision 准入判定结果。
type GrantDecision struct {
	// Allowed 是否可以落库。
	Allowed bool
	// Rejected 被拒绝的岗位（含原因），供 handler 拼 400 文案。
	Rejected []PostGrantRejection
	// AnchorPosts 本次授予中**携带锚点**的岗位（审计/operlog 用）。
	AnchorPosts []PostGrantCandidate
}

// PostGrantRejection 单个岗位被拒绝的原因。
type PostGrantRejection struct {
	PostID   string
	PostName string
	Reason   string
}

// AnchorGrantPolicy 判定"谁能把谁挂到携带锚点的岗位上"的策略开关。
//
// 为什么需要开关：不同部署的组织规模差异很大。小规模部署里"岗位分配"本身
// 就是超管/人事专员的活，强制要求 system:post:edit 反而制造额外负担；
// 大规模部署（本功能的目标场景：省-中支-支公司三层）则必须收紧。
//
// 默认值由调用方（中间件/Service）决定；本文件只定义语义，不读配置。
type AnchorGrantPolicy struct {
	// RequirePostEditPerm 授予携带锚点的岗位时，操作者必须具备 system:post:edit。
	// 推荐生产开启：该权限意味着"能定义机构授权模板"，与"能把它授予某人"应同权。
	RequirePostEditPerm bool
	// AllowAnchorOnlyForAdmin 只有超管（role_key=admin）可授予携带锚点的岗位。
	// 最严格的档位，供高合规要求部署使用（与 RequirePostEditPerm 取或）。
	AllowAnchorOnlyForAdmin bool
}

// DefaultAnchorGrantPolicy 生产推荐策略：需要 system:post:edit。
//
// 理由：能创建/修改「中支管理员」这种模板的人，才应该能把它们发给别人。
// 单靠 system:user:edit 就能授予省级范围，等于把数据范围授权降级成"改个用户"。
func DefaultAnchorGrantPolicy() AnchorGrantPolicy {
	return AnchorGrantPolicy{RequirePostEditPerm: true}
}

// CheckPostGrants 校验一批 PostIds 是否可被当前操作者授予。
//
// 参数：
//   - postIDs 请求体里的岗位 ID 列表（允许含重复/空值，内部归一化）
//   - operatorHasPostEdit 操作者是否持有 system:post:edit（由 handler 判定后传入）
//   - operatorIsSuperAdmin 操作者是否超管
//
// 返回 nil error 表示 SQL 正常（不代表 Allowed=true，要看 Decision.Allowed）。
//
// 规则（逐条）：
//  1. 岗位不存在或已软删 → 拒绝（"岗位不存在"）。
//  2. 岗位 status=停用 → 拒绝（停用岗位本就不产生任何权限，挂上会让人误以为生效）。
//  3. 岗位携带锚点（anchor_mode∈{2,3,5}）且策略要求 post:edit → 操作者无此权限则拒绝。
//  4. 岗位携带锚点且策略要求超管 → 操作者非超管则拒绝。
//  5. 其余 → 允许；携带锚点的岗位记入 AnchorPosts（供 operlog 审计"谁授予了机构范围"）。
func CheckPostGrants(
	ctx context.Context,
	db *gorm.DB,
	postIDs []string,
	operatorHasPostEdit bool,
	operatorIsSuperAdmin bool,
	policy AnchorGrantPolicy,
) (*GrantDecision, error) {

	decision := &GrantDecision{Allowed: true}

	ids := normalizeIDs(postIDs)
	if len(ids) == 0 {
		return decision, nil
	}

	// 一次查询取回全部岗位（含已软删的，以便区分"不存在"与"已删除"由 Exist 表达）。
	// GORM 默认带 deleted_at 过滤，故这里用 Unscoped 会查到软删行；但我们要的是
	// "未软删才有效"，所以**不加** Unscoped，让查不到的按"不存在或已删除"处理。
	var rows []struct {
		ID          string
		PostName    string
		Status      int
		AnchorMode  int
		AnchorParam string
	}
	if err := db.WithContext(ctx).Raw(`
		SELECT CAST(p.id AS TEXT) AS id,
		       COALESCE(p.post_name, '') AS post_name,
		       COALESCE(p.status, 0) AS status,
		       COALESCE(p.anchor_mode, 0) AS anchor_mode,
		       COALESCE(p.anchor_param, '') AS anchor_param
		  FROM sys_post p
		 WHERE CAST(p.id AS TEXT) IN ? AND p.deleted_at IS NULL
	`, ids).Scan(&rows).Error; err != nil {
		// 缺表（精简夹具/极端老库）→ 无法校验，按"不放行携带锚点的岗位"处理：
		// 查询失败时不阻断普通岗位分配（那会让既有用户编辑功能整体不可用），
		// 但也绝不能放过锚点岗位 —— 查不到锚点信息时无锚点信息即"未知"，
		// 按 fail-closed 拒绝所有岗位分配更安全，故这里返回 error 交由调用方 500。
		return nil, fmt.Errorf("校验岗位分配失败: %w", err)
	}

	found := make(map[string]struct {
		PostName    string
		Status      int
		AnchorMode  int
		AnchorParam string
	}, len(rows))
	for _, r := range rows {
		found[r.ID] = struct {
			PostName    string
			Status      int
			AnchorMode  int
			AnchorParam string
		}{r.PostName, r.Status, r.AnchorMode, r.AnchorParam}
	}

	for _, pid := range ids {
		r, ok := found[pid]
		cand := PostGrantCandidate{PostID: pid}
		if !ok {
			decision.Allowed = false
			decision.Rejected = append(decision.Rejected, PostGrantRejection{
				PostID: pid,
				Reason: "岗位不存在或已删除",
			})
			continue
		}
		cand.PostName = r.PostName
		cand.Exist = true
		cand.Status = models.PostStatus(r.Status)
		cand.AnchorMode = models.AnchorMode(r.AnchorMode)
		cand.AnchorParam = r.AnchorParam

		// 规则 2：停用岗位
		if cand.Status != models.PostStatusEnabled {
			decision.Allowed = false
			decision.Rejected = append(decision.Rejected, PostGrantRejection{
				PostID:   pid,
				PostName: r.PostName,
				Reason:   "岗位已停用，停用岗位不产生任何权限",
			})
			continue
		}

		// 规则 3/4：携带锚点的岗位 = 授予数据范围
		if cand.AnchorCarries() {
			denied := ""
			if policy.AllowAnchorOnlyForAdmin && !operatorIsSuperAdmin {
				denied = "该岗位携带机构授权范围，仅超级管理员可授予"
			} else if policy.RequirePostEditPerm && !operatorHasPostEdit {
				denied = "该岗位携带机构授权范围，授予需要「岗位管理」编辑权限（system:post:edit）"
			}
			if denied != "" {
				decision.Allowed = false
				decision.Rejected = append(decision.Rejected, PostGrantRejection{
					PostID:   pid,
					PostName: r.PostName,
					Reason:   fmt.Sprintf("%s；岗位「%s」锚点为 %s", denied, r.PostName, cand.AnchorDescription()),
				})
				continue
			}
			decision.AnchorPosts = append(decision.AnchorPosts, cand)
		}
	}

	return decision, nil
}

// FormatRejections 把拒绝项拼成一句可直接返回给前端的 400 文案。
//
// 只拼接前 N 项，避免岗位很多时错误信息过长（其余以"等 N 项"概括）。
func FormatRejections(rejections []PostGrantRejection) string {
	if len(rejections) == 0 {
		return ""
	}
	const maxShown = 3
	parts := make([]string, 0, maxShown+1)
	for i, r := range rejections {
		if i >= maxShown {
			parts = append(parts, fmt.Sprintf("等 %d 项岗位无法授予", len(rejections)))
			break
		}
		if r.PostName != "" {
			parts = append(parts, fmt.Sprintf("「%s」：%s", r.PostName, r.Reason))
		} else {
			parts = append(parts, r.Reason)
		}
	}
	return "岗位分配被拒绝：" + strings.Join(parts, "；")
}

// LogAnchorGrant 记录一次机构锚点岗位的授予（审计用）。
//
// 为什么单独记日志而不是只靠 operlog：operlog 记录的是"用户管理"这个模块的
// 写操作，粒度为整个请求；而"这次请求里具体授予了哪些机构范围"是与权限提升
// 直接相关的安全事实，需要在日志里独立可查（便于事后追责与异常检测）。
func LogAnchorGrant(ctx context.Context, operatorUserID, targetUserID string, posts []PostGrantCandidate) {
	if len(posts) == 0 {
		return
	}
	names := make([]string, 0, len(posts))
	for _, p := range posts {
		names = append(names, fmt.Sprintf("%s(anchor_mode=%d,param=%q)", p.PostName, p.AnchorMode, p.AnchorParam))
	}
	sort.Strings(names)
	applogger.Warnf(
		"[机构锚点授权] operator=%s target=%s posts=%s —— 该操作会扩大目标用户的数据可见范围",
		operatorUserID, targetUserID, strings.Join(names, ", "),
	)
}
