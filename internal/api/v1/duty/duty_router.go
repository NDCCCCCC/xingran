package duty

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
)

// SetupDutyPoolsRouter 设置值班池路由
func SetupDutyPoolsRouter(r *gin.RouterGroup, core *core.Core) {
	service := services.NewDutyService(core.DB.GetDB())
	handler := NewDutyHandler(service)

	r.POST("/list", handler.ListPools)
	r.POST("", handler.CreatePool)
	r.POST("/:id", handler.GetPoolByID)
	r.POST("/:id/update", handler.UpdatePool)
	r.POST("/:id/delete", handler.DeletePool)
}

// SetupDutySchedulesRouter 设置排班路由
func SetupDutySchedulesRouter(r *gin.RouterGroup, core *core.Core) {
	service := services.NewDutyService(core.DB.GetDB())
	handler := NewDutyHandler(service)

	r.POST("/list", handler.ListSchedules)
	r.POST("/generate", handler.GenerateSchedule)
	r.POST("/today", handler.GetTodayDuty)
	r.GET("/monthly", handler.GetMonthlySchedule) // 使用GET支持Query参数
	r.POST("/swap", handler.SwapDuty)
	r.POST("/manual", handler.ManualDuty)
	r.POST("/:id/delete", handler.DeleteSchedule)
	r.POST("/batch-delete", handler.BatchDeleteSchedules)
}

// SetupDutyHolidaysRouter 设置节假日路由
func SetupDutyHolidaysRouter(r *gin.RouterGroup, core *core.Core) {
	service := services.NewDutyService(core.DB.GetDB())
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
	service := services.NewDutyService(core.DB.GetDB())
	handler := NewDutyHandler(service)

	r.POST("", handler.GetConfig)
	r.POST("/update", handler.UpdateConfig)
}

// SetupMyDutyRouter 设置我的值班路由
func SetupMyDutyRouter(r *gin.RouterGroup, core *core.Core) {
	service := services.NewDutyService(core.DB.GetDB())
	handler := NewDutyHandler(service)

	r.POST("/stats", handler.GetMyStats)
}
