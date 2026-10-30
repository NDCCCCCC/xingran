package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// RoleWriteRouteGroups 角色**写操作**路由分组（85-02 RBAC-04，范式对齐 UserWriteRouteGroups）。
//
// 每个分组对应一个「单一权限点 + 严格权限」中间件，由 internal/api/router.go
// 创建（那里是权限映射的单一落点，便于静态核对）。对应关系：
//
//	Add    → system:role:add    （新增）
//	Edit   → system:role:edit   （编辑 + 状态变更）
//	Remove → system:role:remove （删除 + 批量删除）
//
// 严格权限（RequireStrictPermissions）只做 sys_menu.perms 精确匹配，
// 不做 F 型继承与模块级降级——这是「只读角色不得写」的闭环前提。
type RoleWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Remove *gin.RouterGroup
}

// SetupRoleRouter 设置角色**读**路由
//
// 读操作保留组级 ANY-of 权限（system:role:list / system:role:view，
// 由 internal/api/router.go 施加）；写操作不在本函数注册——见 SetupRoleWriteRouter。
//
// ⚠️ 读组与写组必须是**兄弟组**（不是父子）：gin 的 Group() 会继承父组中间件链，
// 若写组挂在读组之下，读组的 ANY-of 会先于严格权限放行写操作。
func SetupRoleRouter(r *gin.RouterGroup, core *core.Core) {
	handler := newRoleHandler(core)

	// 读路由
	r.POST("/list", handler.List)
	r.POST("/statistics", handler.Statistics)
	r.POST("/all", handler.GetAllEnabled)
	r.POST("/:id", handler.GetByID)
}

// SetupRoleWriteRouter 设置角色**写**路由（85-02 RBAC-04 逐动作拆分）。
//
// 每个写操作落到单一权限点的严格权限子组上（分组由 router.go 创建）。
func SetupRoleWriteRouter(groups RoleWriteRouteGroups, core *core.Core) {
	handler := newRoleHandler(core)

	// 新增
	groups.Add.POST("", handler.Create)
	// 编辑 + 状态变更
	groups.Edit.POST("/:id/update", handler.Update)
	groups.Edit.POST("/:id/status", handler.UpdateStatus)
	// 删除 + 批量删除
	groups.Remove.POST("/:id/delete", handler.Delete)
	groups.Remove.POST("/batch", handler.BatchDelete)
}

// newRoleHandler 构造角色处理器（含缓存适配；读/写路由共用）。
func newRoleHandler(core *core.Core) *RoleHandler {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的角色服务
	var roleService systemServices.RoleService
	if core.DataCacheService != nil {
		roleService = systemServices.NewRoleServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		roleService = systemServices.NewRoleService(core.DB.GetDB())
	}

	// 创建Handler
	return NewRoleHandler(roleService).WithCore(core)
}
