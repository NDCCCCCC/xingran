package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupMenuRouter 设置菜单路由
func SetupMenuRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的菜单服务
	var menuService systemServices.MenuService
	if core.DataCacheService != nil {
		menuService = systemServices.NewMenuServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		menuService = systemServices.NewMenuService(core.DB.GetDB())
	}

	// 创建Handler
	handler := NewMenuHandler(menuService).WithCore(core)

	// 注册路由（注意：更具体的路由要放在前面，避免被通用路由匹配）
	r.POST("/tree", handler.GetTree)                                     // 获取菜单树
	r.POST("/list", handler.List)                                        // 查询菜单列表
	r.POST("/tree-select", handler.GetTree)                              // 获取菜单树选择器
	r.POST("/role-menu-tree-select/:roleId", handler.RoleMenuTreeSelect) // 获取角色菜单树选择器
	r.POST("/batch-delete", handler.BatchDelete)                         // 批量删除菜单
	r.POST("", handler.Create)                                           // 创建菜单
	r.POST("/:id", handler.GetByID)                                      // 获取菜单详情
	r.POST("/:id/update", handler.Update)                                // 更新菜单
	r.POST("/:id/delete", handler.Delete)                                // 删除菜单
	r.POST("/:id/status", handler.UpdateStatus)                          // 更新菜单状态
}

// SetupUserMenuRouter 设置用户菜单路由（所有已登录用户可访问）
// 路径：/system/my-menus/*
func SetupUserMenuRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的菜单服务
	var menuService systemServices.MenuService
	if core.DataCacheService != nil {
		menuService = systemServices.NewMenuServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		menuService = systemServices.NewMenuService(core.DB.GetDB())
	}

	// 创建Handler
	handler := NewMenuHandler(menuService).WithCore(core)

	// 用户菜单相关路由
	// 注意：由于路由组是 /my-menus，所以完整路径是 /system/my-menus/*
	r.POST("", handler.GetUserMenus)                   // 获取当前用户的菜单列表（不包含隐藏）
	r.POST("/all", handler.GetAllUserMenus)            // 获取用户所有菜单（包含隐藏）
	r.POST("/permissions", handler.GetUserPermissions) // 获取当前用户的权限列表
}
