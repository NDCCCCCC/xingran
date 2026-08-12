package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupUserRouter 设置用户路由
func SetupUserRouter(r *gin.RouterGroup, core *core.Core) {
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
	handler := NewUserHandler(userService, userADSyncService).WithCore(core)

	// 注册路由
	r.POST("", handler.Create)
	r.POST("/list", handler.List)
	r.POST("/statistics", handler.Statistics)
	r.POST("/import", handler.ImportUser)
	r.GET("/import/template", handler.DownloadImportTemplate)
	r.POST("/batch-delete", handler.BatchDelete)
	r.POST("/sync-managers", handler.SyncManagers)
	r.POST("/:id", handler.GetByID)
	r.POST("/:id/update", handler.Update)
	r.POST("/:id/delete", handler.Delete)
	r.POST("/:id/status", handler.UpdateStatus)
	r.POST("/:id/reset-password", handler.ResetPassword)
}
