package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// UserWriteRouteGroups 用户**写操作**路由分组。
//
// v1.24 Phase 78（#16）：每个分组对应一个「单一权限点 + 严格权限」中间件，
// 由 internal/api/router.go 创建（那里是权限映射的单一落点，便于静态核对）。
// 分组与权限点的对应关系：
//
//	Add      → system:user:add      （新增）
//	Import   → system:user:import   （导入 + 导入模板下载）
//	Edit     → system:user:edit     （编辑 + 状态变更 + 同步经理）
//	Remove   → system:user:remove   （删除 + 批量删除）
//	ResetPwd → system:user:resetPwd （重置密码）
//
// 严格权限（RequireStrictPermissions）只做 sys_menu.perms 精确匹配，
// 不做 F 型继承与模块级降级——这是「只读角色不得写」的闭环前提。
type UserWriteRouteGroups struct {
	Add      *gin.RouterGroup
	Import   *gin.RouterGroup
	Edit     *gin.RouterGroup
	Remove   *gin.RouterGroup
	ResetPwd *gin.RouterGroup
}

// SetupUserRouter 设置用户**读**路由
//
// 读操作保留组级 ANY-of 权限（system:user:list / system:user:view，
// 由 internal/api/router.go 施加）+ DataScopePermission；
// 写操作不在本函数注册 —— 见 SetupUserWriteRouter。
//
// ⚠️ 读组与写组必须是**兄弟组**（不是父子）：gin 的 Group() 会继承父组中间件链，
// 若写组挂在读组之下，读组的 ANY-of 会先于严格权限放行写操作。
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
	handler := newUserHandler(core)

	// 读路由
	r.POST("/list", handler.List)
	r.POST("/statistics", handler.Statistics)
	r.POST("/:id", handler.GetByID)
}

// SetupUserWriteRouter 设置用户**写**路由
//
// 每个写操作落到单一权限点的严格权限子组上（分组由 router.go 创建）。
// 用户 Excel 导入在此处理（带 AD 域控同步触发），不复用
// operations.SetupExcelRouter —— 后者的 importData 为通用导入，不触发 AD 同步。
func SetupUserWriteRouter(groups UserWriteRouteGroups, core *core.Core) {
	handler := newUserHandler(core)

	// 新增
	groups.Add.POST("", handler.Create)
	// 导入 + 导入模板下载
	groups.Import.POST("/import", handler.ImportUser)
	groups.Import.GET("/import/template", handler.DownloadImportTemplate)
	// 编辑 + 状态变更 + 同步经理
	groups.Edit.POST("/:id/update", handler.Update)
	groups.Edit.POST("/:id/status", handler.UpdateStatus)
	groups.Edit.POST("/sync-managers", handler.SyncManagers)
	// 删除 + 批量删除
	groups.Remove.POST("/:id/delete", handler.Delete)
	groups.Remove.POST("/batch-delete", handler.BatchDelete)
	// 重置密码
	groups.ResetPwd.POST("/:id/reset-password", handler.ResetPassword)
}

// newUserHandler 构造用户处理器（含缓存适配与 AD 同步服务注入）。
func newUserHandler(core *core.Core) *UserHandler {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建用户服务（带缓存或无缓存）
	var userService systemServices.UserService
	if core.DataCacheService != nil {
		userService = systemServices.NewUserServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
			systemServices.NewPasswordManagerAdapter(core.PwdManager),
		)
	} else {
		userService = systemServices.NewUserService(
			core.DB.GetDB(),
			systemServices.NewPasswordManagerAdapter(core.PwdManager),
		)
	}

	// 创建AD同步服务（依赖DeptOUmapper）
	// Phase 38 Wave 1 (W-04): 接入 core 共享的 AccountPool 实例（Pitfall 4：缓存共享，
	// 避免重复 New 导致熔断后账号仍被选中）。pool 来自 core.AuthFactory（core.initAuthFactory 创建）。
	mapper := addomain.NewDeptOUmapper(core.DB.GetDB())
	userADSyncService := addomain.NewUserADSyncService(core.DB.GetDB(), core.GetAuthFactory().GetAccountPool(), nil, mapper)

	// 创建用户处理器，注入AD同步服务
	return NewUserHandler(userService, userADSyncService).WithCore(core)
}
