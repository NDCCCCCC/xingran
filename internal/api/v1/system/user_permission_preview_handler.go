package system

// user_permission_preview_handler.go —— 编辑用户页「权限预览」端点。
//
// 需求（用户 2026-11-17）：编辑用户时角色下方加"权限预览"，
// 除了菜单权限也要展示数据范围（含机构锚点命中明细）。
//
// 端点：POST /system/users/permission-preview
// 权限：读操作，挂在 usersRead 组（继承 system:user:list/view + DataScopePermission）。
// 请求体：{deptId, roleIds, postIds} —— 表单当前值，**尚未保存**。
// 响应：datascope.UserScopePreview（菜单权限 + 数据范围 + 锚点明细 + 可见部门）。
//
// 该端点**只读**，不记 operlog（与角色/岗位的 anchor-preview 一致）。

import (
	"github.com/gin-gonic/gin"

	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PreviewUserPermission 预览表单当前值对应的完整权限画像。
//
// @Summary 用户权限预览
// @Description 按表单当前 deptId/roleIds/postIds 模拟解析：菜单权限 + 数据范围（含机构锚点明细 + 可见部门）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body requests.UserPermissionPreviewRequest true "表单当前值"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /system/users/permission-preview [post]
func (h *UserHandler) PreviewUserPermission(c *gin.Context) {
	var req requests.UserPermissionPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
	if h.core == nil {
		response.Error(c, apperrors.BadRequest("无法预览权限（服务未初始化）"))
		return
	}

	preview, err := datascope.PreviewUserScope(
		c.Request.Context(), h.core.GetDB(), req.DeptID, req.RoleIDs, req.PostIDs,
	)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, preview)
}
