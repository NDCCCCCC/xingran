package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// ==================== 值班池管理 ====================

// listDutyPools 查询值班池列表
func listDutyPools(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.DutyPoolListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		pools, total, err := dutySvc.GetDutyPoolList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     pools,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}
		response.Success(c, pageResp)
	}
}

// createDutyPool 创建值班池
func createDutyPool(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.DutyPoolCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())
		pool, err := dutySvc.CreateDutyPool(c.Request.Context(), &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, pool)
	}
}

// getDutyPool 获取值班池详情
func getDutyPool(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		poolID := c.Param("id")
		if poolID == "" {
			response.Error(c, response.ErrBadRequest, "值班池ID不能为空")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		pool, err := dutySvc.GetDutyPoolByID(c.Request.Context(), poolID)
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, pool)
	}
}

// updateDutyPool 更新值班池
func updateDutyPool(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		poolID := c.Param("id")
		if poolID == "" {
			response.Error(c, response.ErrBadRequest, "值班池ID不能为空")
			return
		}

		var req services.DutyPoolUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		req.ID = poolID
		userID, _ := c.Get("user_id")

		dutySvc := services.NewDutyService(core.DB.GetDB())
		if err := dutySvc.UpdateDutyPool(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteDutyPool 删除值班池
func deleteDutyPool(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		poolID := c.Param("id")
		if poolID == "" {
			response.Error(c, response.ErrBadRequest, "值班池ID不能为空")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		if err := dutySvc.DeleteDutyPool(c.Request.Context(), poolID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// ==================== 排班管理 ====================

// listDutySchedules 查询排班列表
func listDutySchedules(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.DutyScheduleListRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		schedules, total, err := dutySvc.GetDutyScheduleList(c.Request.Context(), &req)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		pageResp := response.PageResponse{
			List:     schedules,
			Total:    total,
			Current:  req.Current,
			PageSize: req.PageSize,
		}
		response.Success(c, pageResp)
	}
}

// generateDutySchedule 生成排班
func generateDutySchedule(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.GenerateScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())
		count, err := dutySvc.GenerateSchedule(c.Request.Context(), &req, userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"message": "生成排班成功",
			"count":   count,
		})
	}
}

// getTodayDuty 获取今日值班
func getTodayDuty(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		dutySvc := services.NewDutyService(core.DB.GetDB())
		members, err := dutySvc.GetTodayDuty(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrNotFound, err.Error())
			return
		}

		response.Success(c, gin.H{
			"date":    members[0].Username, // 简化处理，实际应该返回日期
			"members": members,
		})
	}
}

// getMonthlyDutySchedule 获取月度值班排班
func getMonthlyDutySchedule(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 支持从Query参数或JSON body获取参数
		yearStr := c.Query("year")
		monthStr := c.Query("month")

		// 如果Query参数为空，尝试从JSON body获取
		if yearStr == "" || monthStr == "" {
			var req struct {
				Year  int `json:"year"`
				Month int `json:"month"`
			}
			if err := c.ShouldBindJSON(&req); err == nil {
				if yearStr == "" && req.Year > 0 {
					yearStr = fmt.Sprintf("%d", req.Year)
				}
				if monthStr == "" && req.Month > 0 {
					monthStr = fmt.Sprintf("%d", req.Month)
				}
			}
		}

		if yearStr == "" || monthStr == "" {
			response.Error(c, response.ErrBadRequest, "年份和月份参数不能为空")
			return
		}

		var year, month int
		if _, err := fmt.Sscanf(yearStr, "%d", &year); err != nil {
			response.Error(c, response.ErrBadRequest, "年份格式错误")
			return
		}
		if _, err := fmt.Sscanf(monthStr, "%d", &month); err != nil {
			response.Error(c, response.ErrBadRequest, "月份格式错误")
			return
		}

		if month < 1 || month > 12 {
			response.Error(c, response.ErrBadRequest, "月份必须在1-12之间")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		schedule, err := dutySvc.GetMonthlyDutySchedule(c.Request.Context(), year, month)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, schedule)
	}
}

// swapDuty 调班
func swapDuty(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.SwapDutyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())
		if err := dutySvc.SwapDuty(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "调班成功"})
	}
}

// manualDuty 手动排班
func manualDuty(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.ManualDutyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())
		if err := dutySvc.ManualDuty(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "手动排班成功"})
	}
}

// deleteDutySchedule 删除排班记录
func deleteDutySchedule(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheduleID := c.Param("id")
		if scheduleID == "" {
			response.Error(c, response.ErrBadRequest, "排班记录ID不能为空")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		if err := dutySvc.DeleteDutySchedule(c.Request.Context(), scheduleID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// batchDeleteDutySchedules 批量删除排班记录
func batchDeleteDutySchedules(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			IDs []string `json:"ids" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		if err := dutySvc.BatchDeleteDutySchedules(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"message": "批量删除成功",
			"count":   len(req.IDs),
		})
	}
}

// ==================== 节假日管理 ====================

// listHolidays 查询节假日列表
func listHolidays(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Year int `json:"year"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		holidays, err := dutySvc.GetHolidayList(c.Request.Context(), req.Year)
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, holidays)
	}
}

// createHoliday 创建节假日
func createHoliday(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			HolidayDate string `json:"holidayDate" binding:"required"`
			HolidayName string `json:"holidayName" binding:"required,max=100"`
			IsOffday    bool   `json:"isOffday"`
			HolidayType string `json:"holidayType" binding:"required,oneof=legal workday custom"`
			Year        int    `json:"year" binding:"required"`
			Remark      string `json:"remark"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		date, err := time.Parse("2006-01-02", req.HolidayDate)
		if err != nil {
			response.Error(c, response.ErrBadRequest, "日期格式错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())

		holiday := &models.Holiday{
			BaseModel:   models.BaseModel{CreatedBy: userID.(string)},
			HolidayDate: date,
			HolidayName: req.HolidayName,
			IsOffday:    req.IsOffday,
			HolidayType: models.HolidayType(req.HolidayType),
			Year:        req.Year,
			Remark:      req.Remark,
		}

		if err := dutySvc.CreateHoliday(c.Request.Context(), holiday, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, holiday)
	}
}

// updateHoliday 更新节假日
func updateHoliday(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		holidayID := c.Param("id")
		if holidayID == "" {
			response.Error(c, response.ErrBadRequest, "节假日ID不能为空")
			return
		}

		var req struct {
			HolidayDate string `json:"holidayDate"`
			HolidayName string `json:"holidayName"`
			IsOffday    bool   `json:"isOffday"`
			HolidayType string `json:"holidayType"`
			Remark      string `json:"remark"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())

		var holiday models.Holiday
		if err := core.DB.GetDB().Where("id = ?", holidayID).First(&holiday).Error; err != nil {
			response.Error(c, response.ErrNotFound, "节假日不存在")
			return
		}

		if req.HolidayDate != "" {
			if date, err := time.Parse("2006-01-02", req.HolidayDate); err == nil {
				holiday.HolidayDate = date
			}
		}
		if req.HolidayName != "" {
			holiday.HolidayName = req.HolidayName
		}
		holiday.IsOffday = req.IsOffday
		if req.HolidayType != "" {
			holiday.HolidayType = models.HolidayType(req.HolidayType)
		}
		holiday.Remark = req.Remark

		if err := dutySvc.UpdateHoliday(c.Request.Context(), &holiday, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "更新成功"})
	}
}

// deleteHoliday 删除节假日
func deleteHoliday(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		holidayID := c.Param("id")
		if holidayID == "" {
			response.Error(c, response.ErrBadRequest, "节假日ID不能为空")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		if err := dutySvc.DeleteHoliday(c.Request.Context(), holidayID); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "删除成功"})
	}
}

// batchCreateHolidays 批量创建节假日
func batchCreateHolidays(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Holidays []struct {
				HolidayDate string `json:"holidayDate" binding:"required"`
				HolidayName string `json:"holidayName" binding:"required,max=100"`
				IsOffday    bool   `json:"isOffday"`
				HolidayType string `json:"holidayType" binding:"required,oneof=legal workday custom"`
				Year        int    `json:"year" binding:"required"`
				Remark      string `json:"remark"`
			} `json:"holidays" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())

		holidays := make([]models.Holiday, len(req.Holidays))
		for i, h := range req.Holidays {
			date, err := time.Parse("2006-01-02", h.HolidayDate)
			if err != nil {
				response.Error(c, response.ErrBadRequest, fmt.Sprintf("第%d条记录的日期格式错误", i+1))
				return
			}

			holidays[i] = models.Holiday{
				BaseModel:   models.BaseModel{CreatedBy: userID.(string)},
				HolidayDate: date,
				HolidayName: h.HolidayName,
				IsOffday:    h.IsOffday,
				HolidayType: models.HolidayType(h.HolidayType),
				Year:        h.Year,
				Remark:      h.Remark,
			}
		}

		if err := dutySvc.BatchCreateHolidays(c.Request.Context(), holidays, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"message": "批量创建成功",
			"count":   len(holidays),
		})
	}
}

// getHolidayYears 获取所有有节假日数据的年份列表
func getHolidayYears(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		dutySvc := services.NewDutyService(core.DB.GetDB())
		years, err := dutySvc.GetHolidayYears(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, years)
	}
}

// ==================== 值班配置管理 ====================

// getDutyConfig 获取值班配置
func getDutyConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		dutySvc := services.NewDutyService(core.DB.GetDB())
		config, err := dutySvc.GetDutyConfig(c.Request.Context())
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, config)
	}
}

// updateDutyConfig 更新值班配置
func updateDutyConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.DutyConfig
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		userID, _ := c.Get("user_id")
		dutySvc := services.NewDutyService(core.DB.GetDB())

		if err := dutySvc.UpdateDutyConfig(c.Request.Context(), &req, userID.(string)); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "配置更新成功"})
	}
}

// ==================== 我的值班 ====================

// getMyDutyStats 获取当前用户的值班统计
func getMyDutyStats(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取当前用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized, "未登录")
			return
		}

		dutySvc := services.NewDutyService(core.DB.GetDB())
		stats, err := dutySvc.GetMyDutyStats(c.Request.Context(), userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, stats)
	}
}
