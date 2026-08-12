package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// DiscoveryHandler 设备发现处理器
type DiscoveryHandler struct {
	discoveryService *services.DeviceDiscoveryService
	core             *core.Core
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *DiscoveryHandler) WithCore(core *core.Core) *DiscoveryHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 设备发现统计(总数/待执行/执行中/已完成/失败/累计发现设备数)
// @Summary 设备发现统计
// @Description 用 COUNT 聚合返回发现任务统计,供统计卡片使用
// @Tags 网络设备管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /network/discoveries/statistics [post]
func (h *DiscoveryHandler) Statistics(c *gin.Context) {
	result, err := h.discoveryService.GetStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Probe 探测单台设备
// @Summary 探测单台设备
// @Description 使用指定凭证探测单台设备的连通性和基本信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body services.DeviceProbeRequest true "探测参数"
// @Success 200 {object} response.Response{data=services.DeviceProbeResult}
// @Router /network/devices/discover [post]
func (h *DiscoveryHandler) Probe(c *gin.Context) {
	var req services.DeviceProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responseHelpers.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}

	result, err := h.discoveryService.ProbeSingleDevice(c.Request.Context(), &req)
	if err != nil {
		responseHelpers.Error(c, 500, "探测失败: "+err.Error())
		return
	}

	// Probe 探测单台设备连通性 — 高价值审计（"谁探测了哪台设备"）
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备发现", operlog.OperTypeOther)
	response.Success(c, result)
}

// NewDiscoveryHandler 创建设备发现处理器实例
func NewDiscoveryHandler(discoveryService *services.DeviceDiscoveryService) *DiscoveryHandler {
	return &DiscoveryHandler{discoveryService: discoveryService}
}

// List 查询发现任务列表
// @Summary 查询发现任务列表
// @Description 分页查询设备发现任务列表
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int} true "分页参数"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/discoveries/list [post]
func (h *DiscoveryHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	current := getIntField(rawReq, "current", 1)
	pageSize := getIntField(rawReq, "pageSize", 10)

	tasks, total, err := h.discoveryService.GetDiscoveryList(c.Request.Context(), current, pageSize, getOrderByColumn(rawReq), getIsAscPtr(rawReq))
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	pageResp := response.PageResponse{
		List:     tasks,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}

	response.Success(c, pageResp)
}

// Create 创建发现任务
// @Summary 创建发现任务
// @Description 创建新的设备发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param request body services.DiscoveryRequest true "发现任务信息"
// @Success 200 {object} response.Response
// @Router /network/discoveries/create [post]
func (h *DiscoveryHandler) Create(c *gin.Context) {
	var req services.DiscoveryRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	// 设置默认SNMP端口
	if req.SNMPPort == 0 {
		req.SNMPPort = 161
	}

	userID, _ := c.Get("user_id")
	req.CreatedBy = userID.(string)

	discoveryID, err := h.discoveryService.CreateDiscoveryTask(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "创建发现任务") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备发现", operlog.OperTypeCreate)
	response.Success(c, gin.H{"discoveryId": discoveryID})
}

// GetByID 获取发现任务详情
// @Summary 获取发现任务详情
// @Description 根据ID获取发现任务详情
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id [post]
func (h *DiscoveryHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	task, err := h.discoveryService.GetDiscoveryByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.NotFound(err.Error()))
		return
	}

	response.Success(c, task)
}

// GetResults 获取发现结果
// @Summary 获取发现结果
// @Description 获取发现任务的设备发现结果
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/results [post]
func (h *DiscoveryHandler) GetResults(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	devices, err := h.discoveryService.GetDiscoveryResults(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, gin.H{"devices": devices})
}

// Execute 执行发现任务
// @Summary 执行发现任务
// @Description 执行设备发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/execute [post]
func (h *DiscoveryHandler) Execute(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	result, err := h.discoveryService.ExecuteDiscovery(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "执行发现任务") {
		return
	}

	// Execute 扫描网段发现设备 — OperTypeSync (外部网络同步)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备发现", operlog.OperTypeSync)
	response.Success(c, result)
}

// Cancel 取消发现任务
// @Summary 取消发现任务
// @Description 取消正在执行的发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/cancel [post]
func (h *DiscoveryHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	err := h.discoveryService.CancelDiscovery(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "取消发现任务") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备发现", operlog.OperTypeOther)
	response.Success(c, gin.H{"message": "取消成功"})
}

// Delete 删除发现任务
// @Summary 删除发现任务
// @Description 删除指定的发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/delete [post]
func (h *DiscoveryHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	err := h.discoveryService.DeleteDiscovery(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除发现任务") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备发现", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除发现任务
// @Summary 批量删除发现任务
// @Description 批量删除多个发现任务
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param request body object{discoveryIds=[]string} true "任务ID列表"
// @Success 200 {object} response.Response
// @Router /network/discoveries/batch-delete [post]
func (h *DiscoveryHandler) BatchDelete(c *gin.Context) {
	var req struct {
		DiscoveryIDs []string `json:"discoveryIds" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.discoveryService.BatchDeleteDiscoveries(c.Request.Context(), req.DiscoveryIDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除发现任务") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备发现", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.DiscoveryIDs),
	})
}

// ImportDevices 导入发现的设备
// @Summary 导入发现的设备
// @Description 将发现的设备导入系统
// @Tags 设备发现
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} response.Response
// @Router /network/discoveries/:id/import [post]
func (h *DiscoveryHandler) ImportDevices(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("任务ID"))
		return
	}

	userID, _ := c.Get("user_id")
	count, err := h.discoveryService.ImportDiscoveredDevices(c.Request.Context(), id, nil, userID.(string))
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	// Import 把发现的设备导入系统 — OperTypeImport
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备发现", operlog.OperTypeImport)
	response.Success(c, gin.H{"count": count})
}
