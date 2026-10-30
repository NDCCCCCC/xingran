package duty

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	dutyServices "github.com/xingran-next/xingran-go-backend/internal/services/duty"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// DutyHandler 值班管理处理器
type DutyHandler struct {
	service dutyServices.DutyCacheService
	core    *core.Core
}

// NewDutyHandler 创建值班管理处理器实例
func NewDutyHandler(service dutyServices.DutyCacheService) *DutyHandler {
	return &DutyHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *DutyHandler) WithCore(core *core.Core) *DutyHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// ==================== 值班池管理 ====================

// ListPools 查询值班池列表
// @Summary 查询值班池列表
// @Description 分页查询值班池列表
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body services.DutyPoolListRequest false "查询参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /duty/pools/list [post]
func (h *DutyHandler) ListPools(c *gin.Context) {
	var req services.DutyPoolListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = err // 允许空请求体
	}

	pools, total, err := h.service.GetDutyPoolList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
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

// StatisticsPools 值班池统计(总数/启用/停用/成员总数)
// @Summary 值班池统计
// @Description 返回值班池总数、启停状态计数与成员总数,供统计卡片使用;用 COUNT 聚合而非按当前页 list 计算
// @Tags 值班管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /duty/pools/statistics [post]
func (h *DutyHandler) StatisticsPools(c *gin.Context) {
	result, err := h.service.GetDutyPoolStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// CreatePool 创建值班池
// @Summary 创建值班池
// @Description 创建新的值班池
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body services.DutyPoolCreateRequest true "值班池信息"
// @Success 200 {object} response.Response
// @Router /duty/pools [post]
func (h *DutyHandler) CreatePool(c *gin.Context) {
	var req services.DutyPoolCreateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	pool, err := h.service.CreateDutyPool(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "创建值班池") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班池", operlog.OperTypeCreate)
	response.Success(c, pool)
}

// GetPoolByID 获取值班池详情
// @Summary 获取值班池详情
// @Description 根据ID获取值班池详细信息
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path string true "值班池ID"
// @Success 200 {object} response.Response
// @Router /duty/pools/{id} [post]
func (h *DutyHandler) GetPoolByID(c *gin.Context) {
	poolID := c.Param("id")
	if poolID == "" {
		response.Error(c, apperrors.ParamMissing("值班池ID"))
		return
	}

	pool, err := h.service.GetDutyPoolByID(c.Request.Context(), poolID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, pool)
}

// UpdatePool 更新值班池
// @Summary 更新值班池
// @Description 更新值班池信息
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path string true "值班池ID"
// @Param request body services.DutyPoolUpdateRequest true "值班池信息"
// @Success 200 {object} response.Response
// @Router /duty/pools/{id}/update [post]
func (h *DutyHandler) UpdatePool(c *gin.Context) {
	poolID := c.Param("id")
	if poolID == "" {
		response.Error(c, apperrors.ParamMissing("值班池ID"))
		return
	}

	var req services.DutyPoolUpdateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	req.ID = poolID
	userID, _ := c.Get("user_id")

	err := h.service.UpdateDutyPool(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "更新值班池") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班池", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// DeletePool 删除值班池
// @Summary 删除值班池
// @Description 删除指定值班池
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path string true "值班池ID"
// @Success 200 {object} response.Response
// @Router /duty/pools/{id}/delete [post]
func (h *DutyHandler) DeletePool(c *gin.Context) {
	poolID := c.Param("id")
	if poolID == "" {
		response.Error(c, apperrors.ParamMissing("值班池ID"))
		return
	}

	err := h.service.DeleteDutyPool(c.Request.Context(), poolID)
	if !responseHelpers.HandleServiceError(c, err, "删除值班池") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班池", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// ==================== 排班管理 ====================

// ListSchedules 查询排班列表
// @Summary 查询排班列表
// @Description 分页查询排班列表
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body services.DutyScheduleListRequest false "查询参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /duty/schedules/list [post]
func (h *DutyHandler) ListSchedules(c *gin.Context) {
	var req services.DutyScheduleListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = err // 允许空请求体
	}

	schedules, total, err := h.service.GetDutyScheduleList(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
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

// GenerateSchedule 生成排班
// @Summary 生成排班
// @Description 根据规则自动生成排班
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body services.GenerateScheduleRequest true "生成参数"
// @Success 200 {object} response.Response
// @Router /duty/schedules/generate [post]
func (h *DutyHandler) GenerateSchedule(c *gin.Context) {
	var req services.GenerateScheduleRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	count, err := h.service.GenerateSchedule(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "生成排班") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班排班", operlog.OperTypeOther)
	response.Success(c, gin.H{
		"message": "生成排班成功",
		"count":   count,
	})
}

// GetTodayDuty 获取今日值班
// @Summary 获取今日值班
// @Description 获取今天的值班人员
// @Tags 值班管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /duty/schedules/today [post]
func (h *DutyHandler) GetTodayDuty(c *gin.Context) {
	members, err := h.service.GetTodayDuty(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"members": members,
	})
}

// GetMonthlySchedule 获取月度值班排班
// @Summary 获取月度排班
// @Description 获取指定月份的值班排班表
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param year query int false "年份"
// @Param month query int false "月份"
// @Success 200 {object} response.Response
// @Router /duty/schedules/monthly [get]
func (h *DutyHandler) GetMonthlySchedule(c *gin.Context) {
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
		response.Error(c, apperrors.ParamMissing("年份和月份参数"))
		return
	}

	var year, month int
	if _, err := fmt.Sscanf(yearStr, "%d", &year); err != nil {
		response.Error(c, apperrors.ParamInvalid("年份"))
		return
	}
	if _, err := fmt.Sscanf(monthStr, "%d", &month); err != nil {
		response.Error(c, apperrors.ParamInvalid("月份"))
		return
	}

	if month < 1 || month > 12 {
		response.Error(c, apperrors.ParamInvalid("月份"))
		return
	}

	schedule, err := h.service.GetMonthlyDutySchedule(c.Request.Context(), year, month)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, schedule)
}

// SwapDuty 调班
// @Summary 调班
// @Description 交换两个值班人员的班次
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body services.SwapDutyRequest true "调班信息"
// @Success 200 {object} response.Response
// @Router /duty/schedules/swap [post]
func (h *DutyHandler) SwapDuty(c *gin.Context) {
	var req services.SwapDutyRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.SwapDuty(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "调班") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班排班", operlog.OperTypeOther)
	response.Success(c, gin.H{"message": "调班成功"})
}

// ManualDuty 手动排班
// @Summary 手动排班
// @Description 手动为指定日期添加值班人员
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body services.ManualDutyRequest true "排班信息"
// @Success 200 {object} response.Response
// @Router /duty/schedules/manual [post]
func (h *DutyHandler) ManualDuty(c *gin.Context) {
	var req services.ManualDutyRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.ManualDuty(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "手动排班") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班排班", operlog.OperTypeOther)
	response.Success(c, gin.H{"message": "手动排班成功"})
}

// DeleteSchedule 删除排班记录
// @Summary 删除排班记录
// @Description 删除指定排班记录
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path string true "排班记录ID"
// @Success 200 {object} response.Response
// @Router /duty/schedules/{id}/delete [post]
func (h *DutyHandler) DeleteSchedule(c *gin.Context) {
	scheduleID := c.Param("id")
	if scheduleID == "" {
		response.Error(c, apperrors.ParamMissing("排班记录ID"))
		return
	}

	err := h.service.DeleteDutySchedule(c.Request.Context(), scheduleID)
	if !responseHelpers.HandleServiceError(c, err, "删除排班记录") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班排班", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDeleteSchedules 批量删除排班记录
// @Summary 批量删除排班记录
// @Description 批量删除多个排班记录
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body object{ids:[]string} true "排班记录ID列表"
// @Success 200 {object} response.Response
// @Router /duty/schedules/batch-delete [post]
func (h *DutyHandler) BatchDeleteSchedules(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.service.BatchDeleteDutySchedules(c.Request.Context(), req.IDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除排班记录") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班排班", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.IDs),
	})
}

// ==================== 节假日管理 ====================

// ListHolidays 查询节假日列表
// @Summary 查询节假日列表
// @Description 获取指定年份的节假日列表
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body object{year:int} true "年份"
// @Success 200 {object} response.Response
// @Router /duty/holidays/list [post]
func (h *DutyHandler) ListHolidays(c *gin.Context) {
	var req struct {
		Year int `json:"year"`
	}

	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	holidays, err := h.service.GetHolidayList(c.Request.Context(), req.Year)
	if !responseHelpers.HandleServiceError(c, err, "获取节假日列表") {
		return
	}

	response.Success(c, holidays)
}

// CreateHoliday 创建节假日
// @Summary 创建节假日
// @Description 添加新的节假日
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body object true "节假日信息"
// @Success 200 {object} response.Response
// @Router /duty/holidays [post]
func (h *DutyHandler) CreateHoliday(c *gin.Context) {
	var req struct {
		HolidayDate string `json:"holidayDate" binding:"required"`
		HolidayName string `json:"holidayName" binding:"required,max=100"`
		IsOffday    bool   `json:"isOffday"`
		HolidayType string `json:"holidayType" binding:"required,oneof=legal workday custom"`
		Year        int    `json:"year" binding:"required"`
		Remark      string `json:"remark"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	date, err := time.Parse("2006-01-02", req.HolidayDate)
	if err != nil {
		response.Error(c, apperrors.ParamInvalid("日期"))
		return
	}

	userID, _ := c.Get("user_id")

	holiday := &models.Holiday{
		BaseModel:   models.BaseModel{CreatedBy: userID.(string)},
		HolidayDate: date,
		HolidayName: req.HolidayName,
		IsOffday:    req.IsOffday,
		HolidayType: models.HolidayType(req.HolidayType),
		Year:        req.Year,
		Remark:      req.Remark,
	}

	err = h.service.CreateHoliday(c.Request.Context(), holiday, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "创建节假日") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班节假日", operlog.OperTypeCreate)
	response.Success(c, holiday)
}

// UpdateHoliday 更新节假日
// @Summary 更新节假日
// @Description 更新节假日信息
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path string true "节假日ID"
// @Param request body object true "节假日信息"
// @Success 200 {object} response.Response
// @Router /duty/holidays/{id}/update [post]
func (h *DutyHandler) UpdateHoliday(c *gin.Context) {
	holidayID := c.Param("id")
	if holidayID == "" {
		response.Error(c, apperrors.ParamMissing("节假日ID"))
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
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	// 这里需要从数据库获取原有的holiday记录
	// 为了简化，我们暂时跳过这个功能
	response.Error(c, apperrors.NotImplemented())
}

// DeleteHoliday 删除节假日
// @Summary 删除节假日
// @Description 删除指定节假日
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param id path string true "节假日ID"
// @Success 200 {object} response.Response
// @Router /duty/holidays/{id}/delete [post]
func (h *DutyHandler) DeleteHoliday(c *gin.Context) {
	holidayID := c.Param("id")
	if holidayID == "" {
		response.Error(c, apperrors.ParamMissing("节假日ID"))
		return
	}

	err := h.service.DeleteHoliday(c.Request.Context(), holidayID)
	if !responseHelpers.HandleServiceError(c, err, "删除节假日") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班节假日", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchCreateHolidays 批量创建节假日
// @Summary 批量创建节假日
// @Description 批量添加多个节假日
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body object{holidays:[]object} true "节假日列表"
// @Success 200 {object} response.Response
// @Router /duty/holidays/batch [post]
func (h *DutyHandler) BatchCreateHolidays(c *gin.Context) {
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
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")

	holidays := make([]models.Holiday, len(req.Holidays))
	for i, h := range req.Holidays {
		date, err := time.Parse("2006-01-02", h.HolidayDate)
		if err != nil {
			response.Error(c, apperrors.ParamInvalid(fmt.Sprintf("第%d条记录的日期", i+1)))
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

	err := h.service.BatchCreateHolidays(c.Request.Context(), holidays, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "批量创建节假日") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班节假日", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量创建成功",
		"count":   len(holidays),
	})
}

// GetHolidayYears 获取所有有节假日数据的年份列表
// @Summary 获取节假日年份列表
// @Description 获取所有包含节假日数据的年份
// @Tags 值班管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /duty/holidays/years [post]
func (h *DutyHandler) GetHolidayYears(c *gin.Context) {
	years, err := h.service.GetHolidayYears(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, years)
}

// ==================== 值班配置管理 ====================

// GetConfig 获取值班配置
// @Summary 获取值班配置
// @Description 获取系统值班配置
// @Tags 值班管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /duty/config [post]
func (h *DutyHandler) GetConfig(c *gin.Context) {
	config, err := h.service.GetDutyConfig(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, config)
}

// UpdateConfig 更新值班配置
// @Summary 更新值班配置
// @Description 更新系统值班配置
// @Tags 值班管理
// @Accept json
// @Produce json
// @Param request body models.DutyConfig true "配置信息"
// @Success 200 {object} response.Response
// @Router /duty/config/update [post]
func (h *DutyHandler) UpdateConfig(c *gin.Context) {
	var req models.DutyConfig
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")

	err := h.service.UpdateDutyConfig(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "更新配置") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "值班配置", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "配置更新成功"})
}

// ==================== 我的值班 ====================

// GetMyStats 获取当前用户的值班统计
// @Summary 获取我的值班统计
// @Description 获取当前登录用户的值班统计数据
// @Tags 值班管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /duty/my-duty/stats [post]
func (h *DutyHandler) GetMyStats(c *gin.Context) {
	// 从上下文获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	stats, err := h.service.GetMyDutyStats(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, stats)
}
