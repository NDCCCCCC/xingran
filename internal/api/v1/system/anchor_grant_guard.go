package system

// anchor_grant_guard.go —— 用户岗位分配的机构锚点准入保护（v1.54 安全补丁）。
//
// 背景（verifier 2026-11-17 报告的阻塞级漏洞，已确认闭环）：
//
//	user_service.Create/Update 直接写 sys_user_post，对 req.PostIds 零校验。
//	而 pkg/datascope/queryAnchorGrants 会消费任何 status=0 且 anchor_mode<>0 的岗位，
//	按任职人自己的部门链解析锚点并展开子树。
//
//	⇒ 持 system:user:edit 的中支管理员可以在自己的用户表单里把自己挂到
//	  省管理员创建的「分公司管理员」岗位(anchor_mode=5,param="1")，下一次请求
//	  可见集就从"本部门"跃升到"全省"。一次表单提交即提权。
//
//	这与既有设计无关：DataScopePermission 只限制**数据面**（能看见哪些行），
//	从不限制**岗位字典面**（能挂哪些岗位）。角色分配同理不校验，但角色授权的
//	语义风险由 system:role:edit 天然约束（能建角色的人本就能授角色）；
//	而岗位是"共享模板"，任何人都能把别人（或自己）挂上去，风险被放大。
//
// 本文件只做**准入校验的接线**，判定逻辑在 pkg/datascope/anchor_grant.go
// （纯策略，可单测，不依赖 gin）。

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	permission "github.com/xingran-next/xingran-go-backend/pkg/permission"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// grantAnchorPolicy 当前部署采用的机构锚点授予策略。
//
// 默认取 DefaultAnchorGrantPolicy()（必须 system:post:edit 才能授予携带锚点的
// 岗位）——这是**防提权的核心防线**：锚点岗位 = 数据范围授权，若任何持
// system:user:edit 的人都能把锚点岗位挂给自己，一次表单提交即可提权到全省。
//
// 若部署希望更严格，把 AllowAnchorOnlyForAdmin 置 true（仅超管可授予）。
//
// ⚠️ 强烈不建议把 RequirePostEditPerm 置 false：那会移除唯一的操作者权限校验，
// 只剩岗位存在/status 校验，等于对"谁能授予机构数据范围"不设防。仅当部署能
// 100% 保证所有 system:user:edit 持有者都是可信管理员（且岗位分配入口另有
// 审批流程兜底）时才考虑，并须在部署文档中显式记录该风险接受决定。
var grantAnchorPolicy = datascope.DefaultAnchorGrantPolicy()

// operatorGrantContext 判定当前操作者的授权上下文。
//
// 返回：
//   - operatorUserID 操作者 ID（空串表示取不到，按最保守策略处理）
//   - hasPostEdit 是否持有 system:post:edit
//   - isSuperAdmin 是否超级管理员（role_key=admin）
//
// 注意：不能复用 middleware.isSuperAdmin / checkUserPermission —— 它们是包私有，
// 且带 30s 进程内缓存（对"授权瞬间的自省"不合适，必须看实时权限）。
func (h *UserHandler) operatorGrantContext(c *gin.Context) (operatorUserID string, hasPostEdit bool, isSuperAdmin bool) {
	operatorUserID, _ = extractDataScopeParams(c)
	if h.core == nil || operatorUserID == "" {
		return operatorUserID, false, false
	}

	// 超管：role_key = 'admin' 且角色启用（与 middleware.isSuperAdmin 同口径）。
	var adminCount int64
	if err := h.core.GetDB().Raw(`
		SELECT COUNT(*) FROM sys_user_role ur
		INNER JOIN sys_role r ON CAST(r.id AS TEXT) = CAST(ur.role_id AS TEXT)
		WHERE CAST(ur.user_id AS TEXT) = ? AND r.role_key = ? AND r.status = ?
	`, operatorUserID, "admin", int(models.RoleStatusEnabled)).Scan(&adminCount).Error; err != nil {
		// 查询失败按"非超管、无权限"处理：宁可误拒，不可误授。
		return operatorUserID, false, false
	}
	isSuperAdmin = adminCount > 0

	// 启用角色下的菜单权限是否包含 system:post:edit。
	var hasPerm int64
	if err := h.core.GetDB().Raw(`
		SELECT COUNT(*) FROM sys_menu m
		INNER JOIN sys_role_menu rm ON CAST(rm.menu_id AS TEXT) = CAST(m.id AS TEXT)
		INNER JOIN sys_user_role ur ON CAST(rm.role_id AS TEXT) = CAST(ur.role_id AS TEXT)
		INNER JOIN sys_role r ON CAST(r.id AS TEXT) = CAST(ur.role_id AS TEXT)
		WHERE CAST(ur.user_id AS TEXT) = ?
		  AND m.perms = ?
		  AND m.status = ?
		  AND r.status = ?
	`, operatorUserID, string(permission.PostEdit), int(models.MenuStatusNormal), int(models.RoleStatusEnabled)).
		Scan(&hasPerm).Error; err != nil {
		return operatorUserID, false, isSuperAdmin
	}
	hasPostEdit = hasPerm > 0

	// 超管天然具备（admin 角色虽有全部菜单，但旁路语义上不应依赖菜单行存在）。
	if isSuperAdmin {
		hasPostEdit = true
	}
	return operatorUserID, hasPostEdit, isSuperAdmin
}

// guardPostGrants 校验一批待分配的岗位，被拒绝时直接写 400 响应并返回 false。
//
// 调用约定：handler 在**调用 service 之前**调用本函数；返回 false 时调用方必须
// 立即 return（响应已写出）。
//
// 允许的两种情况：
//   - postIDs 为空（只改用户基本信息，不涉及岗位）；
//   - 全部岗位合法且操作者有资格授予。
func (h *UserHandler) guardPostGrants(c *gin.Context, postIDs []string, targetUserID string) bool {
	if len(postIDs) == 0 {
		return true
	}
	if h.core == nil {
		// 未注入 core（部分单测直接构造 handler）：无法判定权限，
		// 但也不能放行 —— 返回明确错误而非静默通过。
		response.Error(c, apperrors.BadRequest("无法校验岗位分配权限（服务未初始化）"))
		return false
	}

	operatorUserID, hasPostEdit, isSuperAdmin := h.operatorGrantContext(c)

	decision, err := datascope.CheckPostGrants(
		c.Request.Context(), h.core.GetDB(), postIDs, hasPostEdit, isSuperAdmin, grantAnchorPolicy,
	)
	if err != nil {
		// SQL 失败按 500：既不放行，也不谎称"权限不足"。
		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, "校验岗位分配失败"))
		return false
	}

	if !decision.Allowed {
		// 授权被拒本身是安全事件：先留审计痕迹，再返回 400。
		datascope.LogAnchorGrant(c.Request.Context(), operatorUserID, targetUserID, decision.AnchorPosts)
		response.Error(c, apperrors.BadRequest(datascope.FormatRejections(decision.Rejected)))
		return false
	}

	// 授予成功且确实携带锚点 → 记一条可检索的 WARN（事后追责与异常检测用）。
	if len(decision.AnchorPosts) > 0 {
		datascope.LogAnchorGrant(c.Request.Context(), operatorUserID, targetUserID, decision.AnchorPosts)
	}
	return true
}
