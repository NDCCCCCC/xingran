package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// MACAddressListRequest MAC地址列表请求
type MACAddressListRequest struct {
	Current       int    `json:"current" binding:"min=1"`
	PageSize      int    `json:"pageSize" binding:"min=1,max=100"`
	DeviceID      string `json:"deviceId,omitempty"`
	MACAddress    string `json:"macAddress,omitempty"`
	InterfaceName string `json:"interfaceName,omitempty"`
}

// MACCollectionRequest MAC采集请求
type MACCollectionRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

// CleanOldMACRecordsRequest 清理旧记录请求
type CleanOldMACRecordsRequest struct {
	Days int `json:"days" binding:"required,min=1,max=365"`
}

// MACHandler MAC地址管理处理器
type MACHandler struct {
	core *core.Core
}

// NewMACHandler 创建MAC地址管理处理器
func NewMACHandler(core *core.Core) *MACHandler {
	return &MACHandler{core: core}
}

// List 查询MAC地址列表
func (h *MACHandler) List(c *gin.Context) {
	var req MACAddressListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	macAddrs, total, err := svc.GetMACAddressList(c.Request.Context(), req.Current, req.PageSize,
		req.DeviceID, req.MACAddress, req.InterfaceName)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, macAddrs, total, req.Current, req.PageSize)
}

// Collect 采集单个设备MAC地址
func (h *MACHandler) Collect(c *gin.Context) {
	var req MACCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	result, err := svc.CollectDevice(c.Request.Context(), req.DeviceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// CollectAll 采集所有设备MAC地址
func (h *MACHandler) CollectAll(c *gin.Context) {
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	results, err := svc.CollectAllDevices(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, results)
}

// GetStats 获取MAC地址统计信息
func (h *MACHandler) GetStats(c *gin.Context) {
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	stats, err := svc.GetMACAddressStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, stats)
}

// Clean 清理旧的MAC地址记录
func (h *MACHandler) Clean(c *gin.Context) {
	var req CleanOldMACRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	count, err := svc.CleanOldRecords(c.Request.Context(), req.Days)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"deletedCount": count})
}
