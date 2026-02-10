package duty

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	dutyServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/duty"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
)

// SetupDutyPoolsRouter 设置值班池路由
func SetupDutyPoolsRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service)

	r.POST("/list", handler.ListPools)
	r.POST("", handler.CreatePool)
	r.POST("/:id", handler.GetPoolByID)
	r.POST("/:id/update", handler.UpdatePool)
	r.POST("/:id/delete", handler.DeletePool)
}

// SetupDutySchedulesRouter 设置排班路由
func SetupDutySchedulesRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service)

	r.POST("/list", handler.ListSchedules)
	r.POST("/generate", handler.GenerateSchedule)
	r.POST("/today", handler.GetTodayDuty)
	r.POST("/monthly", handler.GetMonthlySchedule) // 改为POST以支持JSON body参数
	r.POST("/swap", handler.SwapDuty)
	r.POST("/manual", handler.ManualDuty)
	r.POST("/:id/delete", handler.DeleteSchedule)
	r.POST("/batch-delete", handler.BatchDeleteSchedules)
}

// SetupDutyHolidaysRouter 设置节假日路由
func SetupDutyHolidaysRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service)

	r.POST("/list", handler.ListHolidays)
	r.POST("/years", handler.GetHolidayYears)
	r.POST("", handler.CreateHoliday)
	r.POST("/:id/update", handler.UpdateHoliday)
	r.POST("/:id/delete", handler.DeleteHoliday)
	r.POST("/batch", handler.BatchCreateHolidays)
}

// SetupDutyConfigRouter 设置值班配置路由
func SetupDutyConfigRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service)

	r.POST("", handler.GetConfig)
	r.POST("/update", handler.UpdateConfig)
}

// SetupMyDutyRouter 设置我的值班路由
func SetupMyDutyRouter(r *gin.RouterGroup, core *core.Core) {
	service := createDutyService(core)
	handler := NewDutyHandler(service)

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
