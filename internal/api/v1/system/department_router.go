package system

import (
	"github.com/gin-gonic/gin"
	opsAPI "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
)

// adcascadeSetter 部门服务的 AD 级联注入点（缓存包装内嵌基础服务，方法集透传）。
type adcascadeSetter interface {
	SetADCascade(cascade *addomain.DeptADCascadeService)
}

// wireDeptADCascade 给部门服务注入 AD OU 级联（域控审查 P0-2）。
// pool 复用 AuthFactory 的全局共享单例（Pitfall 4：禁临时 NewAccountPool）。
func wireDeptADCascade(core *core.Core, svc systemServices.DepartmentService) {
	if core.DB == nil || core.AuthFactory == nil {
		return
	}
	setter, ok := svc.(adcascadeSetter)
	if !ok {
		return
	}
	setter.SetADCascade(addomain.NewDeptADCascadeService(core.DB.GetDB(), core.AuthFactory.GetAccountPool()))
}

// SetupDepartmentRouter 设置部门路由
func SetupDepartmentRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建缓存提供者适配器
	cacheProvider := systemServices.NewCacheProvider(core.DataCacheService)

	// 创建带缓存的部门服务
	var departmentService systemServices.DepartmentService
	if core.DataCacheService != nil {
		departmentService = systemServices.NewDepartmentServiceWithCache(
			core.DB.GetDB(),
			cacheProvider,
			core.CacheConfigService,
		)
	} else {
		departmentService = systemServices.NewDepartmentService(core.DB.GetDB())
	}
	wireDeptADCascade(core, departmentService)

	// 创建Handler
	handler := NewDepartmentHandler(departmentService).WithCore(core)

	// 注册路由
	r.POST("/tree", handler.GetTree)                                     // 获取部门树
	r.POST("/list", handler.List)                                        // 查询部门列表
	r.POST("/batch", handler.BatchDelete)                                // 批量删除部门
	r.POST("/tree-select", handler.GetTree)                              // 获取部门树选择器
	r.POST("", handler.Create)                                           // 创建部门
	r.POST("/:id", handler.GetByID)                                      // 获取部门详情
	r.POST("/:id/update", handler.Update)                                // 更新部门
	r.POST("/:id/delete", handler.Delete)                                // 删除部门
	r.POST("/:id/status", handler.UpdateStatus)                          // 更新部门状态
	r.POST("/role-dept-tree-select/:roleId", handler.RoleDeptTreeSelect) // 获取角色部门树选择器
	r.GET("/:id/users", handler.GetUsers)                                // 获取部门用户列表

	// Excel导入导出路由
	opsAPI.SetupExcelRouter(r, "department", core)
}
