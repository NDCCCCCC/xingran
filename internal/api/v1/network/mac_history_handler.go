package network

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	mac_history_query_service "github.com/xingran-next/xingran-go-backend/internal/services"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// MACHistoryHandler MAC历史查询处理器
type MACHistoryHandler struct {
	historyQueryService mac_history_query_service.MACHistoryQueryService
}

// NewMACHistoryHandler 创建MAC历史查询处理器
func NewMACHistoryHandler(historySvc mac_history_query_service.MACHistoryQueryService) *MACHistoryHandler {
	return &MACHistoryHandler{historyQueryService: historySvc}
}

// QueryPortHistory 查询端口历史记录
// @Summary 查询端口MAC地址历史记录
// @Description 按设备ID和接口名查询MAC地址变化历史，支持时间范围过滤和分页
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body mac_history_query_service.PortHistoryQuery true "查询条件"
// @Success 200 {object} response.Response{data=mac_history_query_service.MACHistoryQueryResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/port [post]
func (h *MACHistoryHandler) QueryPortHistory(c *gin.Context) {
	var req mac_history_query_service.PortHistoryQuery
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	// 调用服务层查询
	result, err := h.historyQueryService.QueryPortHistory(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "查询端口MAC历史记录") {
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// QueryDeviceHistory 查询设备历史记录
// @Summary 查询设备MAC地址历史记录
// @Description 按设备ID查询所有端口的MAC地址变化历史，支持时间范围过滤和分页
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body mac_history_query_service.DeviceHistoryQuery true "查询条件"
// @Success 200 {object} response.Response{data=mac_history_query_service.MACHistoryQueryResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/device [post]
func (h *MACHistoryHandler) QueryDeviceHistory(c *gin.Context) {
	var req mac_history_query_service.DeviceHistoryQuery
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	// 调用服务层查询
	result, err := h.historyQueryService.QueryDeviceHistory(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "查询设备MAC历史记录") {
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// QueryConnectionStats 查询MAC连接时长统计
// @Summary 查询MAC连接时长统计
// @Description 输出明细(MAC×端口停留时长)+ Top-N(按MAC长期占用、端口热门连接)
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body mac_history_query_service.ConnectionStatsQuery true "统计条件"
// @Success 200 {object} response.Response{data=mac_history_query_service.ConnectionStatsResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/stats [post]
func (h *MACHistoryHandler) QueryConnectionStats(c *gin.Context) {
	var req mac_history_query_service.ConnectionStatsQuery
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	result, err := h.historyQueryService.QueryConnectionStats(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "查询连接时长统计") {
		return
	}

	response.Success(c, result)
}

// GetVendorRequest 厂商查询请求
type GetVendorRequest struct {
	MAC string `json:"mac" binding:"required"`
}

// GetVendorResponse 厂商查询响应
type GetVendorResponse struct {
	VendorName string `json:"vendorName"` // camelCase,与项目其他响应字段一致(Phase 13 W4 修复)
}

// GetVendor 查询MAC地址的厂商信息
// @Summary 查询MAC地址厂商信息
// @Description 根据MAC地址前6位（OUI）查询设备厂商
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body GetVendorRequest true "MAC地址"
// @Success 200 {object} response.Response{data=GetVendorResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/vendor [post]
func (h *MACHistoryHandler) GetVendor(c *gin.Context) {
	var req GetVendorRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	vendorName, err := h.historyQueryService.GetVendor(c.Request.Context(), req.MAC)
	if !responseHelpers.HandleServiceError(c, err, "查询MAC厂商信息") {
		return
	}

	response.Success(c, GetVendorResponse{
		VendorName: vendorName,
	})
}

// QueryHistory 通用MAC历史列表查询
// @Summary MAC历史列表（支持任意字段过滤 + 分页）
// @Description Phase 14 前端列表页使用，支持按 MAC/设备ID/接口名/VLAN/事件类型/状态/时间范围 过滤
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body mac_history_query_service.MACHistoryListQuery true "查询条件"
// @Success 200 {object} response.Response{data=mac_history_query_service.MACHistoryQueryResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/list [post]
func (h *MACHistoryHandler) QueryHistory(c *gin.Context) {
	var req mac_history_query_service.MACHistoryListQuery
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	result, err := h.historyQueryService.QueryHistory(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "查询MAC历史列表") {
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// ExportHistory 导出MAC历史为 xlsx
// @Summary 导出MAC历史为 Excel（xlsx）
// @Description Phase 14 前端导出按钮使用；强制 30 天时间上限，最多 100000 行
// @Tags 网络设备管理
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param current query int false "当前页" default(1)
// @Param pageSize query int false "每页大小" default(20)
// @Param mac query string false "MAC 地址"
// @Param deviceId query string false "设备ID (UUID)"
// @Param interfaceName query string false "接口名"
// @Param vlanId query int false "VLAN ID"
// @Param eventType query string false "事件类型"
// @Param status query int false "状态"
// @Param startTime query string false "开始时间 (RFC3339)"
// @Param endTime query string false "结束时间 (RFC3339)"
// @Param exportScope query string false "导出范围: current|all" default(current)
// @Success 200 {file} file "mac_history_<scope>_<ts>.xlsx"
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/list [get]
func (h *MACHistoryHandler) ExportHistory(c *gin.Context) {
	var req mac_history_query_service.MACHistoryListQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 默认 exportScope = current
	if req.ExportScope == "" {
		req.ExportScope = "current"
	}

	// 关键顺序：先把 xlsx 写入 bytes.Buffer，失败时返回 JSON envelope，不污染响应头
	buf := new(bytes.Buffer)
	if err := h.historyQueryService.ExportHistory(c.Request.Context(), &req, buf); err != nil {
		response.Error(c, http.StatusInternalServerError, "导出失败: "+err.Error())
		return
	}

	// 成功后再设置响应头
	filename := fmt.Sprintf("mac_history_%s_%s.xlsx", req.ExportScope, time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Status(http.StatusOK)
	if _, err := c.Writer.Write(buf.Bytes()); err != nil {
		applogger.Errorf("[MAC历史导出] 写入响应失败: %v", err)
	}
}
