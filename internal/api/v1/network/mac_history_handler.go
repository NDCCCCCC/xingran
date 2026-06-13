package network

import (
	mac_history_query_service "github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	responseHelpers "github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
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

// QueryTrajectory 查询MAC地址移动轨迹
// @Summary 查询MAC地址移动轨迹
// @Description 按MAC地址查询跨设备/端口的移动路径，返回按时间排序的轨迹节点
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body mac_history_query_service.MACTrajectoryQuery true "查询条件"
// @Success 200 {object} response.Response{data=mac_history_query_service.MACTrajectoryResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/history/trajectory [post]
func (h *MACHistoryHandler) QueryTrajectory(c *gin.Context) {
	var req mac_history_query_service.MACTrajectoryQuery
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	result, err := h.historyQueryService.QueryMACTrajectory(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "查询MAC轨迹") {
		return
	}

	response.Success(c, result)
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
	VendorName string `json:"vendor_name"`
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
