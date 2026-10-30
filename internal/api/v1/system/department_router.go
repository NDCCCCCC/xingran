package system

import (
	"github.com/gin-gonic/gin"
	opsAPI "github.com/xingran-next/xingran-go-backend/internal/api/v1/operations"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
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

// DeptWriteRouteGroups 部门**写操作**路由分组（85-02 RBAC-04，范式对齐 UserWriteRouteGroups）。
//
// 对应关系：Add → system:dept:add；Edit → system:dept:edit（编辑 + 状态变更）；
// Remove → system:dept:remove（删除 + 批量删除）；Import → system:dept:import；
// Export → system:dept:export（Excel 导入导出）。
type DeptWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Remove *gin.RouterGroup
	Import *gin.RouterGroup
	Export *gin.RouterGroup
}

// SetupDepartmentRouter 设置部门**读**路由
//
// 读操作保留组级 ANY-of 权限（system:dept:list / system:dept:view，
// 由 internal/api/router.go 施加，含 OpsSelectorReadPerms queryExtra 放行）；
// 写操作不在本函数注册——见 SetupDepartmentWriteRouter。
//
// ⚠️ 读组与写组必须是**兄弟组**（不是父子）：gin 的 Group() 会继承父组中间件链，
// 若写组挂在读组之下，读组的 ANY-of（含 queryExtra）会先于严格权限放行写操作。
func SetupDepartmentRouter(r *gin.RouterGroup, core *core.Core) {
	handler := newDepartmentHandler(core)

	// 读路由
	r.POST("/tree", handler.GetTree)                                     // 获取部门树
	r.POST("/list", handler.List)                                        // 查询部门列表
	r.POST("/tree-select", handler.GetTree)                              // 获取部门树选择器
	r.POST("/:id", handler.GetByID)                                      // 获取部门详情
	r.POST("/role-dept-tree-select/:roleId", handler.RoleDeptTreeSelect) // 获取角色部门树选择器
	r.GET("/:id/users", handler.GetUsers)                                // 获取部门用户列表

	// Excel 模板下载（读操作，保留在读组）
	opsAPI.SetupExcelReadRouter(r, "department", core)
}

// SetupDepartmentWriteRouter 设置部门**写**路由（85-02 RBAC-04 逐动作拆分）。
//
// 每个写操作落到单一权限点的严格权限子组上（分组由 router.go 创建）。
func SetupDepartmentWriteRouter(groups DeptWriteRouteGroups, core *core.Core) {
	handler := newDepartmentHandler(core)

	// 新增
	groups.Add.POST("", handler.Create) // 创建部门
	// 编辑 + 状态变更
	groups.Edit.POST("/:id/update", handler.Update)       // 更新部门
	groups.Edit.POST("/:id/status", handler.UpdateStatus) // 更新部门状态
	// 删除 + 批量删除
	groups.Remove.POST("/:id/delete", handler.Delete) // 删除部门
	groups.Remove.POST("/batch", handler.BatchDelete) // 批量删除部门
	// Excel 导入/导出（各自单一权限点严格子组）
	opsAPI.SetupExcelImportRouter(groups.Import, "department", core)
	opsAPI.SetupExcelExportRouter(groups.Export, "department", core)
}

// newDepartmentHandler 构造部门处理器（含缓存适配与 AD 级联注入；读/写路由共用）。
func newDepartmentHandler(core *core.Core) *DepartmentHandler {
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
	return NewDepartmentHandler(departmentService).WithCore(core)
}
