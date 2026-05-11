package network

import (
	"net/http"

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

// GetStats 获取MAC历史统计信息
// @Summary 获取MAC历史统计信息
// @Description 获取MAC历史记录的统计信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /network/history/stats [post]
func (h *MACHistoryHandler) GetStats(c *gin.Context) {
	// TODO: 实现统计功能（Phase 13）
	response.Error(c, http.StatusNotImplemented, "统计功能尚未实现")
}
