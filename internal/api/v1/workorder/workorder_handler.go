package workorder

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/workorder"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// WorkOrderHandler 工单管理处理器
type WorkOrderHandler struct {
	service workorder.WorkOrderCacheService
	core    *core.Core
}

// NewWorkOrderHandler 创建工单管理处理器实例
func NewWorkOrderHandler(service workorder.WorkOrderCacheService) *WorkOrderHandler {
	return &WorkOrderHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *WorkOrderHandler) WithCore(core *core.Core) *WorkOrderHandler {
	if h != nil && core != nil {
		h.core = core
	}
	return h
}

// ==================== 工单基础操作 ====================

// List 查询工单列表
// @Summary 查询工单列表
// @Description 分页查询工单列表，支持多条件过滤
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body workorder.ListRequest false "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /workorder/orders/list [post]
func (h *WorkOrderHandler) List(c *gin.Context) {
	var req workorder.ListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = err // 允许空请求体，使用默认值
	}

	list, total, err := h.service.GetList(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "获取工单列表") {
		return
	}

	pageResp := response.PageResponse{
		List:     list,
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}
	response.Success(c, pageResp)
}

// GetStatusStatistics 工单状态统计(总数/待处理/处理中/已完成/已关闭)
// @Summary 工单状态统计
// @Description 返回工单总数及各状态计数,供列表页统计卡片使用;用 COUNT 聚合而非按当前页 list 计算
// @Tags 工单管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/orders/status-statistics [post]
func (h *WorkOrderHandler) GetStatusStatistics(c *gin.Context) {
	result, err := h.service.GetStatusStatistics(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取工单状态统计") {
		return
	}
	response.Success(c, result)
}

// GetMyPending 获取当前用户的待办工单
// @Summary 获取待办工单
// @Description 获取当前用户的待办工单（待处理和处理中）
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body workorder.GetMyPendingRequest false "查询参数"
// @Success 200 {object} response.Response
// @Router /workorder/orders/my-pending [post]
func (h *WorkOrderHandler) GetMyPending(c *gin.Context) {
	var req workorder.GetMyPendingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = err // 允许空请求体，使用默认值
		req.Limit = 5
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, apperrors.Unauthorized())
		return
	}

	list, total, err := h.service.GetMyPending(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "获取待办工单") {
		return
	}

	response.Success(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// GetByID 获取工单详情
// @Summary 获取工单详情
// @Description 根据工单ID获取详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id} [post]
func (h *WorkOrderHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	workOrder, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, workOrder)
}

// Create 创建工单
// @Summary 创建工单
// @Description 创建新的工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body workorder.CreateRequest true "工单信息"
// @Success 200 {object} response.Response
// @Router /workorder/orders [post]
func (h *WorkOrderHandler) Create(c *gin.Context) {
	var req workorder.CreateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	workOrder, err := h.service.Create(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "创建工单") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeCreate)
	response.Success(c, workOrder)
}

// Update 更新工单
// @Summary 更新工单
// @Description 更新工单信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Param request body workorder.UpdateRequest true "工单信息"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/update [post]
func (h *WorkOrderHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	var req workorder.UpdateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	req.ID = id
	userID, _ := c.Get("user_id")

	err := h.service.Update(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "更新工单") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除工单
// @Summary 删除工单
// @Description 删除指定工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/delete [post]
func (h *WorkOrderHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	err := h.service.Delete(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除工单") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除工单
// @Summary 批量删除工单
// @Description 批量删除多个工单
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body object{ids:[]string} true "工单ID列表"
// @Success 200 {object} response.Response
// @Router /workorder/orders/batch-delete [post]
func (h *WorkOrderHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.service.BatchDelete(c.Request.Context(), req.IDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除工单") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.IDs),
	})
}

// ==================== 工单分配与状态 ====================

// Assign 分配工单
// @Summary 分配工单
// @Description 将工单分配给指定处理人
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Param request body workorder.AssignRequest true "分配信息"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/assign [post]
func (h *WorkOrderHandler) Assign(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	var req workorder.AssignRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.Assignment().Assign(c.Request.Context(), id, &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "分配工单") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeGrant)
	response.Success(c, gin.H{"message": "分配成功"})
}

// AssignToTodayDuty 分配给当天值班人员
// @Summary 分配给值班人员
// @Description 将工单分配给当天值班人员
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/assign-duty [post]
func (h *WorkOrderHandler) AssignToTodayDuty(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.Assignment().AssignToTodayDuty(c.Request.Context(), id, userID.(string))
	// 当没有值班人员时，静默处理，不显示错误
	if err != nil && err.Error() == "今天没有值班人员" {
		response.Success(c, gin.H{"message": "今天没有值班人员，工单未分配"})
		return
	}
	if !responseHelpers.HandleServiceError(c, err, "分配给值班人员") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeGrant)
	response.Success(c, gin.H{"message": "已分配给当天值班人员"})
}

// UpdateStatus 更新工单状态
// @Summary 更新工单状态
// @Description 更新工单的处理状态
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Param request body workorder.UpdateStatusRequest true "状态信息"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/status [post]
func (h *WorkOrderHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	var req workorder.UpdateStatusRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.Assignment().UpdateStatus(c.Request.Context(), id, &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "更新状态") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeStatus)
	response.Success(c, gin.H{"message": "状态更新成功"})
}

// ==================== 工单评论 ====================

// GetComments 获取工单评论列表
// @Summary 获取工单评论
// @Description 获取指定工单的评论列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/comments [post]
func (h *WorkOrderHandler) GetComments(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	comments, err := h.service.Comment().GetList(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "获取评论") {
		return
	}

	response.Success(c, comments)
}

// AddComment 添加工单评论
// @Summary 添加工单评论
// @Description 为指定工单添加评论
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Param request body workorder.AddRequest true "评论内容"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/comments [post]
func (h *WorkOrderHandler) AddComment(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	var req workorder.AddRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.Comment().Add(c.Request.Context(), id, &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "添加评论") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单管理", operlog.OperTypeOther)
	response.Success(c, gin.H{"message": "评论添加成功"})
}

// ==================== 工单历史与统计 ====================

// GetHistory 获取工单操作历史
// @Summary 获取工单历史
// @Description 获取指定工单的操作历史记录
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "工单ID"
// @Success 200 {object} response.Response
// @Router /workorder/orders/{id}/history [post]
func (h *WorkOrderHandler) GetHistory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("工单ID"))
		return
	}

	workOrder, err := h.service.GetByID(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "获取工单历史") {
		return
	}

	response.Success(c, workOrder.History)
}

// GetStatistics 获取工单统计数据
// @Summary 获取工单统计
// @Description 获取工单的统计数据
// @Tags 工单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/statistics [post]
func (h *WorkOrderHandler) GetStatistics(c *gin.Context) {
	stats, err := h.service.GetStatistics(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取统计数据") {
		return
	}

	response.Success(c, stats)
}

// ==================== 工单分类 ====================

// ListCategories 查询工单分类列表
// @Summary 查询工单分类
// @Description 获取工单分类树形列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/categories/list [post]
func (h *WorkOrderHandler) ListCategories(c *gin.Context) {
	categories, err := h.service.Category().GetTree(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取分类列表") {
		return
	}

	response.Success(c, categories)
}

// GetEnabledCategories 获取启用的工单分类
// @Summary 获取启用的分类
// @Description 获取所有启用的工单分类（用于下拉选择）
// @Tags 工单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/categories/enabled [post]
func (h *WorkOrderHandler) GetEnabledCategories(c *gin.Context) {
	categories, err := h.service.Category().GetEnabled(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取启用的分类") {
		return
	}

	response.Success(c, categories)
}

// GetCategoryByID 获取工单分类详情
// @Summary 获取分类详情
// @Description 根据ID获取工单分类详细信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "分类ID"
// @Success 200 {object} response.Response
// @Router /workorder/categories/{id} [post]
func (h *WorkOrderHandler) GetCategoryByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("分类ID"))
		return
	}

	category, err := h.service.Category().GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, category)
}

// CreateCategory 创建工单分类
// @Summary 创建工单分类
// @Description 创建新的工单分类
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body models.WorkOrderCategory true "分类信息"
// @Success 200 {object} response.Response
// @Router /workorder/categories [post]
func (h *WorkOrderHandler) CreateCategory(c *gin.Context) {
	var req models.WorkOrderCategory
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.Category().Create(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "创建分类") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单分类", operlog.OperTypeCreate)
	response.Success(c, req)
}

// UpdateCategory 更新工单分类
// @Summary 更新工单分类
// @Description 更新工单分类信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "分类ID"
// @Param request body models.WorkOrderCategory true "分类信息"
// @Success 200 {object} response.Response
// @Router /workorder/categories/{id}/update [post]
func (h *WorkOrderHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("分类ID"))
		return
	}

	var req models.WorkOrderCategory
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}
	req.ID = id

	userID, _ := c.Get("user_id")
	err := h.service.Category().Update(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "更新分类") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单分类", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// DeleteCategory 删除工单分类
// @Summary 删除工单分类
// @Description 删除指定工单分类
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "分类ID"
// @Success 200 {object} response.Response
// @Router /workorder/categories/{id}/delete [post]
func (h *WorkOrderHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("分类ID"))
		return
	}

	err := h.service.Category().Delete(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除分类") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单分类", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// ==================== 周期性工单 ====================

// ListPeriodic 查询周期性工单模板列表
// @Summary 查询周期性工单模板
// @Description 获取周期性工单模板列表
// @Tags 工单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/periodic/list [post]
func (h *WorkOrderHandler) ListPeriodic(c *gin.Context) {
	var req workorder.PeriodicTemplateListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = err // 允许空请求体
	}

	list, total, err := h.service.Periodic().GetTemplateList(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "获取周期性工单模板") {
		return
	}

	response.Success(c, gin.H{
		"list":     list,
		"total":    total,
		"current":  req.Current,
		"pageSize": req.PageSize,
	})
}

// GetPeriodicStatistics 周期性工单模板统计(总数/启用/停用/累计生成数)
// @Summary 周期性工单模板统计
// @Description 返回模板总数/启停数及累计生成工单数,供列表页统计卡片使用;用 COUNT 聚合而非按当前页 list 计算
// @Tags 工单管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/periodic/templates/statistics [post]
func (h *WorkOrderHandler) GetPeriodicStatistics(c *gin.Context) {
	result, err := h.service.Periodic().GetStatistics(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取周期性工单模板统计") {
		return
	}
	response.Success(c, result)
}

// CreatePeriodic 创建周期性工单模板
// @Summary 创建周期性工单模板
// @Description 创建新的周期性工单模板
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body workorder.CreateTemplateRequest true "模板信息"
// @Success 200 {object} response.Response
// @Router /workorder/periodic [post]
func (h *WorkOrderHandler) CreatePeriodic(c *gin.Context) {
	var req workorder.CreateTemplateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	template, err := h.service.Periodic().CreateTemplate(c.Request.Context(), &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "创建周期性工单模板") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "周期性工单", operlog.OperTypeCreate)
	response.Success(c, template)
}

// UpdatePeriodic 更新周期性工单模板
// @Summary 更新周期性工单模板
// @Description 更新周期性工单模板信息
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Param request body workorder.UpdateTemplateRequest true "模板信息"
// @Success 200 {object} response.Response
// @Router /workorder/periodic/{id}/update [post]
func (h *WorkOrderHandler) UpdatePeriodic(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	var req workorder.UpdateTemplateRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	err := h.service.Periodic().UpdateTemplate(c.Request.Context(), id, &req, userID.(string))
	if !responseHelpers.HandleServiceError(c, err, "更新周期性工单模板") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "周期性工单", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// DeletePeriodic 删除周期性工单模板
// @Summary 删除周期性工单模板
// @Description 删除指定周期性工单模板
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param id path string true "模板ID"
// @Success 200 {object} response.Response
// @Router /workorder/periodic/{id}/delete [post]
func (h *WorkOrderHandler) DeletePeriodic(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("模板ID"))
		return
	}

	err := h.service.Periodic().DeleteTemplate(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除周期性工单模板") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "周期性工单", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// ==================== 工单配置 ====================

// GetConfig 获取工单配置
// @Summary 获取工单配置
// @Description 获取工单系统配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /workorder/config [post]
func (h *WorkOrderHandler) GetConfig(c *gin.Context) {
	config, err := h.service.Config().Get(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取配置") {
		return
	}

	response.Success(c, config)
}

// UpdateConfig 更新工单配置
// @Summary 更新工单配置
// @Description 更新工单系统配置
// @Tags 工单管理
// @Accept json
// @Produce json
// @Param request body models.WorkOrderConfig true "配置信息"
// @Success 200 {object} response.Response
// @Router /workorder/config/update [post]
func (h *WorkOrderHandler) UpdateConfig(c *gin.Context) {
	var req models.WorkOrderConfig
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.service.Config().Update(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "更新配置") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工单配置", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "配置更新成功"})
}
