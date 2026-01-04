package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupDutyPoolsRouter 设置值班池路由
func SetupDutyPoolsRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/list", listDutyPools(core))
	r.POST("", createDutyPool(core))
	r.GET("/:id", getDutyPool(core))
	r.POST("/:id/update", updateDutyPool(core))
	r.POST("/:id/delete", deleteDutyPool(core))
}

// SetupDutySchedulesRouter 设置排班路由
func SetupDutySchedulesRouter(r *gin.RouterGroup, core *core.Core) {
	r.POST("/list", listDutySchedules(core))
	r.POST("/generate", generateDutySchedule(core))
	r.GET("/today", getTodayDuty(core))
	r.GET("/monthly", getMonthlyDutySchedule(core))
	r.POST("/swap", swapDuty(core))
	r.POST("/manual", manualDuty(core))
	r.POST("/:id/delete", deleteDutySchedule(core))
	r.POST("/batch-delete", batchDeleteDutySchedules(core))
}

// SetupDutyHolidaysRouter 设置节假日路由
func SetupDutyHolidaysRouter(r *gin.RouterGroup, core *core.Core) {
	r.GET("/list", listHolidays(core))
	r.GET("/years", getHolidayYears(core))
	r.POST("", createHoliday(core))
	r.POST("/:id/update", updateHoliday(core))
	r.POST("/:id/delete", deleteHoliday(core))
	r.POST("/batch", batchCreateHolidays(core))
}

// SetupDutyConfigRouter 设置值班配置路由
func SetupDutyConfigRouter(r *gin.RouterGroup, core *core.Core) {
	r.GET("", getDutyConfig(core))
	r.POST("/update", updateDutyConfig(core))
}

// SetupMyDutyRouter 设置我的值班路由
func SetupMyDutyRouter(r *gin.RouterGroup, core *core.Core) {
	r.GET("/stats", getMyDutyStats(core))
}
