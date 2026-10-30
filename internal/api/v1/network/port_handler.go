package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// PortStatusListRequest 端口状态列表请求
type PortStatusListRequest struct {
	base.BaseListRequest
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
// @Summary 查询端口状态列表
// @Description 分页查询端口状态列表，支持按设备ID、接口名称、状态筛选
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body PortStatusListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/port/list [post]
func (h *PortHandler) List(c *gin.Context) {
	var req PortStatusListRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	portStatuses, total, err := svc.Query.GetList(c.Request.Context(), &portcollection.ListRequest{
		BaseListRequest:     req.BaseListRequest,
		DeviceID:            req.DeviceID,
		InterfaceName:       req.InterfaceName,
		AdminStatus:         req.AdminStatus,
		OperStatus:          req.OperStatus,
		Dot1xEnabled:        req.Dot1xEnabled,
		PortSecurityEnabled: req.PortSecurityEnabled,
	})
	if !responseHelpers.HandleServiceError(c, err, "获取端口状态列表") {
		return
	}

	response.Page(c, portStatuses, total, req.Current, req.PageSize)
}

// Collect 采集单个设备端口状态
// @Summary 采集单个设备端口状态
// @Description 连接指定设备并采集其端口状态信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body PortCollectionRequest true "采集请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/port/collect [post]
func (h *PortHandler) Collect(c *gin.Context) {
	var req PortCollectionRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	result, err := svc.Collection.CollectDevice(c.Request.Context(), req.DeviceID)
	if !responseHelpers.HandleServiceError(c, err, "采集设备端口状态") {
		return
	}

	// Collect 采集端口状态 — OperTypeSync (设备→系统数据同步)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口管理", operlog.OperTypeSync)
	response.Success(c, result)
}

// CollectAll 采集所有设备端口状态
// @Summary 采集所有设备端口状态
// @Description 连接所有已配置的网络设备并采集端口状态信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]interface{}}
// @Failure 500 {object} response.Response
// @Router /network/port/collect-all [post]
func (h *PortHandler) CollectAll(c *gin.Context) {
	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	results, err := svc.Collection.CollectAllDevices(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "采集所有设备端口状态") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口管理", operlog.OperTypeSync)
	response.Success(c, results)
}

// GetStats 获取端口状态统计信息
// @Summary 获取端口状态统计信息
// @Description 获取端口状态的统计信息，包括总数、按状态分组等
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /network/port/stats [post]
func (h *PortHandler) GetStats(c *gin.Context) {
	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	stats, err := svc.Query.GetStats(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取端口状态统计") {
		return
	}

	response.Success(c, stats)
}

// BatchDelete 批量删除端口状态记录
// @Summary 批量删除端口状态记录
// @Description 批量删除指定的端口状态记录
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body BatchDeletePortRecordsRequest true "删除请求"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/port/batch-delete [post]
func (h *PortHandler) BatchDelete(c *gin.Context) {
	var req BatchDeletePortRecordsRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	if err := h.core.DB.GetDB().Delete(&models.DevicePortStatus{}, req.IDs).Error; err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeServerError, "批量删除失败"))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口管理", operlog.OperTypeBatch)
	response.Success(c, gin.H{"deletedCount": len(req.IDs)})
}

// Clean 清理旧的端口状态记录
// @Summary 清理旧的端口状态记录
// @Description 删除指定天数之前的端口状态记录
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body CleanOldPortRecordsRequest true "清理请求"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/port/clean [post]
func (h *PortHandler) Clean(c *gin.Context) {
	var req CleanOldPortRecordsRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	svc := portcollection.NewPortCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor)
	count, err := svc.Query.CleanOldRecords(c.Request.Context(), req.Days)
	if !responseHelpers.HandleServiceError(c, err, "清理旧端口记录") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口管理", operlog.OperTypeClean)
	response.Success(c, gin.H{"deletedCount": count})
}
