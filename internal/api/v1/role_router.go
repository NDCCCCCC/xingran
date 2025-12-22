package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupRoleRouter 设置角色路由
func SetupRoleRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询角色列表
	r.POST("/list", listRoles(core))
	// 获取所有启用角色（用于下拉选择）
	r.POST("/all", getAllRoles(core))
	// 创建角色
	r.POST("", createRole(core))
	// 获取角色详情
	r.POST("/:id", getRole(core))
	// 更新角色
	r.POST("/:id/update", updateRole(core))
	// 删除角色
	r.POST("/:id/delete", deleteRole(core))
	// 批量删除角色
	r.POST("/batch-delete", batchDeleteRoles(core))
	// 更改角色状态
	r.POST("/:id/status", updateRoleStatus(core))
	// 更新角色菜单权限
	r.POST("/:id/menus", updateRoleMenus(core))
}