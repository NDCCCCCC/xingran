package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// MenuWriteRouteGroups 菜单**写操作**路由分组（85-02 RBAC-04，范式对齐 UserWriteRouteGroups）。
//
// 对应关系：Add → system:menu:add；Edit → system:menu:edit（编辑 + 状态变更）；
// Remove → system:menu:remove（删除 + 批量删除）。
type MenuWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Remove *gin.RouterGroup
}

// SetupMenuRouter 设置菜单**读**路由
//
// 读操作保留组级 ANY-of 权限（system:menu:list / system:menu:view，
// 由 internal/api/router.go 施加）；写操作不在本函数注册——见 SetupMenuWriteRouter。
//
// ⚠️ 读组与写组必须是**兄弟组**（不是父子）：gin 的 Group() 会继承父组中间件链，
// 若写组挂在读组之下，读组的 ANY-of 会先于严格权限放行写操作。
func SetupMenuRouter(r *gin.RouterGroup, core *core.Core) {
	handler := newMenuHandler(core)

	// 读路由
	r.POST("/tree", handler.GetTree)                                     // 获取菜单树
	r.POST("/list", handler.List)                                        // 查询菜单列表
	r.POST("/tree-select", handler.GetTree)                              // 获取菜单树选择器
	r.POST("/role-menu-tree-select/:roleId", handler.RoleMenuTreeSelect) // 获取角色菜单树选择器
	r.POST("/:id", handler.GetByID)                                      // 获取菜单详情
}

// SetupMenuWriteRouter 设置菜单**写**路由（85-02 RBAC-04 逐动作拆分）。
//
// 每个写操作落到单一权限点的严格权限子组上（分组由 router.go 创建）。
func SetupMenuWriteRouter(groups MenuWriteRouteGroups, core *core.Core) {
	handler := newMenuHandler(core)

	// 新增
	groups.Add.POST("", handler.Create) // 创建菜单
	// 编辑 + 状态变更
	groups.Edit.POST("/:id/update", handler.Update)       // 更新菜单
	groups.Edit.POST("/:id/status", handler.UpdateStatus) // 更新菜单状态
	// 删除 + 批量删除
	groups.Remove.POST("/:id/delete", handler.Delete)        // 删除菜单
	groups.Remove.POST("/batch-delete", handler.BatchDelete) // 批量删除菜单
}

// newMenuHandler 构造菜单处理器（含缓存适配；读/写路由共用）。
func newMenuHandler(core *core.Core) *MenuHandler {
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
	return NewMenuHandler(menuService).WithCore(core)
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
