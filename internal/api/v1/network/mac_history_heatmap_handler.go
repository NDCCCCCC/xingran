package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// MACHistoryHeatmapHandler 热力图 handler
type MACHistoryHeatmapHandler struct {
	heatmapService services.MACHistoryHeatmapService
}

// NewMACHistoryHeatmapHandler 构造函数
func NewMACHistoryHeatmapHandler(heatmapSvc services.MACHistoryHeatmapService) *MACHistoryHeatmapHandler {
	return &MACHistoryHeatmapHandler{heatmapService: heatmapSvc}
}

// QueryHeatmap POST /network/history/heatmap
func (h *MACHistoryHeatmapHandler) QueryHeatmap(c *gin.Context) {
	var req services.HeatmapQuery
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "请求参数错误: "+err.Error())
		return
	}
	result, err := h.heatmapService.QueryHeatmap(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, 500, "热力图查询失败: "+err.Error())
		return
	}
	response.Success(c, result)
}
