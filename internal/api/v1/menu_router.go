package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupMenuRouter 设置菜单路由
func SetupMenuRouter(r *gin.RouterGroup, core *core.Core) {
	// 查询菜单列表
	r.POST("/list", listMenus(core))
	// 获取用户菜单树
	r.POST("/user-menus", getUserMenus(core))
	// 获取用户权限列表
	r.POST("/user-permissions", getUserPermissions(core))
	// 创建菜单
	r.POST("", createMenu(core))
	// 获取菜单详情
	r.POST("/:id", getMenu(core))
	// 更新菜单
	r.POST("/:id/update", updateMenu(core))
	// 删除菜单
	r.POST("/:id/delete", deleteMenu(core))
	// 获取菜单树选择器
	r.POST("/tree-select", getMenuTreeSelect(core))
	// 获取角色菜单树选择器
	r.POST("/role-menu-tree-select/:roleId", getRoleMenuTreeSelect(core))
}