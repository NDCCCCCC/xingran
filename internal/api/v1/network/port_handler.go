package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/portcollection"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// PortStatusListRequest 端口状态列表请求
type PortStatusListRequest struct {
	Current             int    `json:"current" binding:"min=1"`
	PageSize            int    `json:"pageSize" binding:"min=1,max=1000"`
	DeviceID            string `json:"deviceId,omitempty"`
	InterfaceName       string `json:"interfaceName,omitempty"`
	AdminStatus         string `json:"adminStatus,omitempty"`
	OperStatus          string `json:"operStatus,omitempty"`
	Dot1xEnabled        *bool  `json:"dot1xEnabled,omitempty"`
	PortSecurityEnabled *bool  `json:"portSecurityEnabled,omitempty"`
}

// PortCollectionRequest 端口采集请求
type PortCollectionRequest struct {
	DeviceID string `json:"deviceId" binding:"required"`
}

// CleanOldPortRecordsRequest 清理旧记录请求
type CleanOldPortRecordsRequest struct {
	Days int `json:"days" binding:"required,min=1,max=365"`
}

// BatchDeletePortRecordsRequest 批量删除请求
type BatchDeletePortRecordsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// PortHandler 端口状态管理处理器
type PortHandler struct {
	core *core.Core
}

// NewPortHandler 创建端口状态管理处理器
func NewPortHandler(core *core.Core) *PortHandler {
	return &PortHandler{core: core}
}

// List 查询端口状态列表
func (h *PortHandler) List(c *gin.Context) {
	var req PortStatusListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	portStatuses, total, err := svc.Query.GetList(c.Request.Context(), &portcollection.ListRequest{
		Current:             req.Current,
		PageSize:            req.PageSize,
		DeviceID:            req.DeviceID,
		InterfaceName:       req.InterfaceName,
		AdminStatus:         req.AdminStatus,
		OperStatus:          req.OperStatus,
		Dot1xEnabled:        req.Dot1xEnabled,
		PortSecurityEnabled: req.PortSecurityEnabled,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Page(c, portStatuses, total, req.Current, req.PageSize)
}

// Collect 采集单个设备端口状态
func (h *PortHandler) Collect(c *gin.Context) {
	var req PortCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	result, err := svc.Collection.CollectDevice(c.Request.Context(), req.DeviceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// CollectAll 采集所有设备端口状态
func (h *PortHandler) CollectAll(c *gin.Context) {
	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	results, err := svc.Collection.CollectAllDevices(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, results)
}

// GetStats 获取端口状态统计信息
func (h *PortHandler) GetStats(c *gin.Context) {
	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	stats, err := svc.Query.GetStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, stats)
}

// BatchDelete 批量删除端口状态记录
func (h *PortHandler) BatchDelete(c *gin.Context) {
	var req BatchDeletePortRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.core.DB.GetDB().Delete(&models.DevicePortStatus{}, req.IDs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "批量删除失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"deletedCount": len(req.IDs)})
}

// Clean 清理旧的端口状态记录
func (h *PortHandler) Clean(c *gin.Context) {
	var req CleanOldPortRecordsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	count, err := svc.Query.CleanOldRecords(c.Request.Context(), req.Days)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"deletedCount": count})
}
