package monitor

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ServerHandler 服务器监控处理器
type ServerHandler struct {
	serverService monitorServices.ServerService
	core          *core.Core
}

// NewServerHandler 创建服务器监控处理器
func NewServerHandler(serverService monitorServices.ServerService) *ServerHandler {
	return &ServerHandler{
		serverService: serverService,
	}
}

// WithCore 注入 core 依赖（用于操作日志埋点），链式调用
func (h *ServerHandler) WithCore(core *core.Core) *ServerHandler {
	h.core = core
	return h
}

// GetServerInfo 获取服务器信息
// @Summary 获取服务器信息
// @Description 获取服务器基本信息和当前状态（优先从缓存获取）
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body object false "查询参数"
// @Success 200 {object} response.Response
// @Router /monitor/server-info/list [post]
func (h *ServerHandler) GetServerInfo(c *gin.Context) {
	var req struct {
		Current      int    `json:"current"`
		PageSize      int    `json:"pageSize"`
		OrderByColumn string `json:"orderByColumn"`
		IsAsc         bool   `json:"isAsc"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Current = 1
		req.PageSize = 10
	}

	if req.Current <= 0 {
		req.Current = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	servers, total, err := h.serverService.GetServerInfo(c.Request.Context(), monitorServices.ServerInfoParams{
		Current:      req.Current,
		PageSize:      req.PageSize,
		OrderByColumn: req.OrderByColumn,
		IsAsc:         req.IsAsc,
	})
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Page(c, servers, total, req.Current, req.PageSize)
}

// GetCurrentServerMetrics 获取当前服务器指标
// @Summary 获取当前服务器指标
// @Description 获取当前服务器的实时性能指标（优先从缓存获取）
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /monitor/server-metrics/current [post]
func (h *ServerHandler) GetCurrentServerMetrics(c *gin.Context) {
	metrics, err := h.serverService.GetCurrentServerMetrics(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取服务器指标") {
		return
	}

	responseData := map[string]interface{}{
		"cpuUsage":     metrics.CPUUsage,
		"memoryUsage":  metrics.MemoryUsage,
		"diskUsage":    metrics.DiskUsage,
		"networkRx":    metrics.NetworkRx,
		"networkTx":    metrics.NetworkTx,
		"processCount": metrics.ProcessNum,
		"totalMemory":  metrics.TotalMemory,
		"usedMemory":   metrics.UsedMemory,
		"timestamp":    metrics.Timestamp,
		"fromCache":    true,
	}

	response.Success(c, responseData)
}

// SaveSystemMetrics 保存系统指标
// @Summary 保存系统指标
// @Description 保存系统性能指标数据
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body object true "指标数据"
// @Success 200 {object} response.Response
// @Router /monitor/server-metrics/save [post]
func (h *ServerHandler) SaveSystemMetrics(c *gin.Context) {
	var req struct {
		ServerID     string  `json:"serverId"`
		CPUUsage     float64 `json:"cpuUsage"`
		MemoryUsage  float64 `json:"memoryUsage"`
		DiskUsage    float64 `json:"diskUsage"`
		NetworkRx    uint64  `json:"networkRx"`
		NetworkTx    uint64  `json:"networkTx"`
		ProcessCount int     `json:"processCount"`
		LoadAverage  float64 `json:"loadAverage"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("请求参数错误"))
		return
	}

	metrics := &models.SystemMetrics{
		ServerID:     req.ServerID,
		CPUUsage:     req.CPUUsage,
		MemoryUsage:  req.MemoryUsage,
		DiskUsage:    req.DiskUsage,
		NetworkRx:    req.NetworkRx,
		NetworkTx:    req.NetworkTx,
		ProcessCount: req.ProcessCount,
		LoadAverage:  req.LoadAverage,
		Timestamp:    time.Now(),
	}

	err := h.serverService.SaveSystemMetrics(c.Request.Context(), metrics)
	if !responseHelpers.HandleServiceError(c, err, "保存系统指标") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "服务监控", operlog.OperTypeSync)

	response.Success(c, gin.H{"message": "保存成功"})
}

// GetSystemMetricsHistory 获取系统指标历史
// @Summary 获取系统指标历史
// @Description 获取指定时间范围内的系统性能指标历史数据
// @Tags 系统监控
// @Accept json
// @Produce json
// @Param request body object false "查询参数"
// @Success 200 {object} response.Response
// @Router /monitor/server-metrics/history [post]
func (h *ServerHandler) GetSystemMetricsHistory(c *gin.Context) {
	var req struct {
		ServerID  *string `json:"serverId,omitempty"`
		StartTime *string `json:"startTime,omitempty"`
		EndTime   *string `json:"endTime,omitempty"`
		Current   int     `json:"current"`
		PageSize  int     `json:"pageSize"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Current = 1
		req.PageSize = 100
	}

	if req.Current <= 0 {
		req.Current = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 100
	}

	var serverID string
	if req.ServerID != nil {
		serverID = *req.ServerID
	}

	metrics, total, err := h.serverService.GetSystemMetricsHistory(c.Request.Context(), monitorServices.MetricsHistoryParams{
		ServerID:  serverID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Current:   req.Current,
		PageSize:  req.PageSize,
	})
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Page(c, metrics, total, req.Current, req.PageSize)
}
