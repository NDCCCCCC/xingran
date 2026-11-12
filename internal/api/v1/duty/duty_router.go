package duty

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	dutyServices "github.com/xingran-next/xingran-go-backend/internal/services/duty"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupDutyPoolsRouter 设置值班池路由（读组：list/statistics/:id）。
//
// task-3 G1 整改（2026-11-11）：读/写兄弟分离。写操作（create/update/delete）
// 由 SetupDutyPoolsWriteRouter 在 router.go 的 strict 兄弟组接线，
// 读组 ANY-of 不再覆盖写端点（此前持 ops:duty:pool:list 即可调 delete）。
func SetupDutyPoolsRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	r.POST("/list", handler.ListPools)
	r.POST("/statistics", handler.StatisticsPools)
	r.POST("/:id", handler.GetPoolByID)
}

// SetupDutyPoolsWriteRouter 设置值班池写路由（严格写组接线：逐动作单一权限点）。
//
//	新增          POST /duty/pools           → ops:duty:pool:add
//	编辑          POST /duty/pools/:id/update → ops:duty:pool:edit
//	删除          POST /duty/pools/:id/delete → ops:duty:pool:delete
func SetupDutyPoolsWriteRouter(groups DutyPoolsWriteRouteGroups, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	groups.Add.POST("", handler.CreatePool)
	groups.Edit.POST("/:id/update", handler.UpdatePool)
	groups.Delete.POST("/:id/delete", handler.DeletePool)
}

// DutyPoolsWriteRouteGroups 值班池写操作路由分组（task-3 G1：逐动作拆分）。
type DutyPoolsWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupDutySchedulesRouter 设置排班路由（读组：list/today/monthly）。
//
// task-3 G1 整改（2026-11-11）：读/写兄弟分离。写操作（generate/swap/manual/
// delete/batch-delete）由 SetupDutySchedulesWriteRouter 在 strict 兄弟组接线。
func SetupDutySchedulesRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	r.POST("/list", handler.ListSchedules)
	r.POST("/today", handler.GetTodayDuty)
	r.POST("/monthly", handler.GetMonthlySchedule) // 改为POST以支持JSON body参数
}

// SetupDutySchedulesWriteRouter 设置排班写路由（严格写组接线：逐动作单一权限点）。
//
//	生成排班      POST /duty/schedules/generate      → ops:duty:schedule:edit
//	换班          POST /duty/schedules/swap          → ops:duty:schedule:edit
//	手工排班      POST /duty/schedules/manual        → ops:duty:schedule:edit
//	删除          POST /duty/schedules/:id/delete    → ops:duty:schedule:delete
//	批量删除      POST /duty/schedules/batch-delete  → ops:duty:schedule:delete
func SetupDutySchedulesWriteRouter(groups DutySchedulesWriteRouteGroups, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	groups.Edit.POST("/generate", handler.GenerateSchedule)
	groups.Edit.POST("/swap", handler.SwapDuty)
	groups.Edit.POST("/manual", handler.ManualDuty)
	groups.Delete.POST("/:id/delete", handler.DeleteSchedule)
	groups.Delete.POST("/batch-delete", handler.BatchDeleteSchedules)
}

// DutySchedulesWriteRouteGroups 排班写操作路由分组（task-3 G1：编辑/删除拆分）。
type DutySchedulesWriteRouteGroups struct {
	Edit   *gin.RouterGroup
	Delete *gin.RouterGroup
}

// SetupDutyHolidaysRouter 设置节假日路由（所有已登录用户可查看）。
//
// 口径记录（task-3）：holidays 整组为有意开放组（查看节假日对所有用户开放，
// 含创建/修改——由排班人员在同一入口维护），不在本次 granularity 拆分范围。
func SetupDutyHolidaysRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	r.POST("/list", handler.ListHolidays)
	r.POST("/years", handler.GetHolidayYears)
	r.POST("", handler.CreateHoliday)
	r.POST("/:id/update", handler.UpdateHoliday)
	r.POST("/:id/delete", handler.DeleteHoliday)
	r.POST("/batch", handler.BatchCreateHolidays)
}

// SetupDutyConfigRouter 设置值班配置路由（读端点：POST "" 读取配置）。
//
// task-3 G1 整改（2026-11-11）：读/写兄弟分离。本组沿用 system:config 单码，
// 写端点 /update 由 SetupDutyConfigWriteRouter 在 strict 兄弟组接线
// （单码组 ANY-of 与 strict 对持码者等价，拆分是为对齐全局「读 ANY-of / 写 strict」形态）。
func SetupDutyConfigRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	r.POST("", handler.GetConfig)
}

// SetupDutyConfigWriteRouter 设置值班配置写路由（严格写组：POST /update）。
func SetupDutyConfigWriteRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	r.POST("/update", handler.UpdateConfig)
}

// SetupMyDutyRouter 设置我的值班路由（所有已登录用户可访问，不在拆分范围）。
func SetupMyDutyRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service).WithCore(core)

	r.POST("/stats", handler.GetMyStats)
}

// createDutyService 创建值班服务（带缓存）
func createDutyService(core *core.Core) dutyServices.DutyCacheService {
	var cacheProvider systemServices.CacheProvider
	if core.DataCacheService != nil {
		cacheProvider = systemServices.NewCacheProvider(core.DataCacheService)
	} else {
		cacheProvider = &systemServices.NoOpCacheProvider{}
	}

	return dutyServices.NewDutyServiceWithCache(
		core.DB.GetDB(),
		cacheProvider,
		core.CacheConfigService,
	)
}
