package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupUserMenuRouter 设置用户菜单和权限路由（所有已登录用户可访问）
func SetupUserMenuRouter(r *gin.RouterGroup, core *core.Core) {
	menus := r.Group("/menus")
	{
		// 获取用户菜单树（不包含隐藏菜单，用于导航栏）
		menus.POST("/user-menus", getUserMenus(core))
		// 获取用户所有菜单（包含隐藏菜单，用于标签页标题）
		menus.POST("/user-all", getAllUserMenus(core))
		// 获取用户权限列表
		menus.POST("/user-permissions", getUserPermissions(core))
	}
}

// SetupMenuRouter 设置菜单路由（需要菜单管理权限）
func SetupMenuRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询菜单列表
	r.POST("/list", listMenus(core))
	// 创建菜单
	r.POST("", createMenu(core))
	// 获取菜单详情
	r.POST("/:id", getMenu(core))
	// 更新菜单
	r.POST("/:id/update", updateMenu(core))
	// 删除菜单
	r.POST("/:id/delete", deleteMenu(core))
	// 批量删除菜单
	r.POST("/batch-delete", batchDeleteMenus(core))
	// 获取菜单树选择器
	r.POST("/tree-select", getMenuTreeSelect(core))
	// 获取角色菜单树选择器
	r.POST("/role-menu-tree-select/:roleId", getRoleMenuTreeSelect(core))
}
