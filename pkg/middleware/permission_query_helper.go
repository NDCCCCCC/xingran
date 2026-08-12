package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
)

// HasUserPermission 在 gin 上下文中"查询式"判断当前用户是否持有指定权限。
//
// 用途（cross-module-permission.md §2.3 + 45-R4 D-A1-03）：
//  - handler 内部需要根据用户权限做"静默降级"或"条件性增强响应字段"时调用
//  - 不应触发 401/403/Abort 流程（区别于 RequirePermissions 中间件）
//  - 调用方完全决定 false 时如何处理：静默 hide、跳过字段、记日志等
//
// 设计约定（与现有 permission.go 一致）：
//  - 同包（package middleware）复用内部 getUserIDAsString / isSuperAdmin / checkUserPermission
//  - 不导出这些 helper 本身（小写 + 内部可见性），避免扩大 API surface
//  - 用户未认证 / ID 格式异常 / 权限查询错误 → 一律返回 false，由调用方兜底
//
// 用法示例：
//
//	if middleware.HasUserPermission(c, core, "asset:reconciliation:list") {
//	    resp.ReconciliationVisible = true
//	} else {
//	    resp.ReconciliationVisible = false
//	    resp.ReconciliationHiddenReason = "无资产对账查看权限"
//	}
func HasUserPermission(c *gin.Context, core *core.Core, perm string) bool {
	if c == nil || core == nil || perm == "" {
		return false
	}
	userID, ok := getUserIDAsString(c)
	if !ok || userID == "" {
		return false
	}
	// 超级管理员直接通过（与 checkUserPermission 内部行为一致；前置以省 DB 查询）
	if isSuperAdmin(core, userID) {
		return true
	}
	return checkUserPermission(core, userID, perm)
}
