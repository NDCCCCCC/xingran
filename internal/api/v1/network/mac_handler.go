package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/lldp"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// MACAddressListRequest MAC地址列表请求
type MACAddressListRequest struct {
	base.BaseListRequest
	DeviceID      string `json:"deviceId,omitempty"`
	DeptID        string `json:"deptId,omitempty"` // 2026-06-30: 部门树联动 — 后端 JOIN sys_network_device.dept_id 过滤
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
// @Summary 查询MAC地址列表
// @Description 分页查询MAC地址列表，支持按设备ID、MAC地址、接口名称筛选
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body MACAddressListRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/mac/list [post]
func (h *MACHandler) List(c *gin.Context) {
	var req MACAddressListRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
	filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)
	macAddrs, total, err := svc.GetMACAddressList(c.Request.Context(), req.Current, req.PageSize,
		req.DeviceID, req.DeptID, req.MACAddress, req.InterfaceName,
		req.OrderByColumn, req.IsAsc)
	if !responseHelpers.HandleServiceError(c, err, "获取MAC地址列表") {
		return
	}

	response.Page(c, macAddrs, total, req.Current, req.PageSize)
}

// Collect 采集单个设备MAC地址
// @Summary 采集单个设备MAC地址
// @Description 连接指定设备并采集其MAC地址信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body MACCollectionRequest true "采集请求"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/mac/collect [post]
func (h *MACHandler) Collect(c *gin.Context) {
	var req MACCollectionRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
	filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)
	result, err := svc.CollectDevice(c.Request.Context(), req.DeviceID)
	if !responseHelpers.HandleServiceError(c, err, "采集设备MAC地址") {
		return
	}

	// Collect 从设备采集 MAC 地址 — OperTypeSync (设备→系统数据同步)
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "MAC地址采集", operlog.OperTypeSync)
	response.Success(c, result)
}

// CollectAll 采集所有设备MAC地址
// @Summary 采集所有设备MAC地址
// @Description 连接所有已配置的网络设备并采集MAC地址信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]interface{}}
// @Failure 500 {object} response.Response
// @Router /network/mac/collect-all [post]
func (h *MACHandler) CollectAll(c *gin.Context) {
	lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
	filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)
	results, err := svc.CollectAllDevices(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "采集所有设备MAC地址") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "MAC地址采集", operlog.OperTypeSync)
	response.Success(c, results)
}

// GetStats 获取MAC地址统计信息
// @Summary 获取MAC地址统计信息
// @Description 获取MAC地址的统计信息，包括总数、按设备分组等
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 500 {object} response.Response
// @Router /network/mac/stats [post]
func (h *MACHandler) GetStats(c *gin.Context) {
	lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
	filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)
	stats, err := svc.GetMACAddressStats(c.Request.Context())
	if !responseHelpers.HandleServiceError(c, err, "获取MAC地址统计") {
		return
	}

	response.Success(c, stats)
}

// Clean 清理旧的MAC地址记录
// @Summary 清理旧的MAC地址记录
// @Description 删除指定天数之前的MAC地址记录
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body CleanOldMACRecordsRequest true "清理请求"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /network/mac/clean [post]
func (h *MACHandler) Clean(c *gin.Context) {
	var req CleanOldMACRecordsRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
	filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)
	count, err := svc.CleanOldRecords(c.Request.Context(), req.Days)
	if !responseHelpers.HandleServiceError(c, err, "清理旧MAC地址记录") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "MAC地址采集", operlog.OperTypeClean)
	response.Success(c, gin.H{"deletedCount": count})
}

// BatchDelete 批量删除MAC地址记录
// @Summary 批量删除MAC地址记录
// @Description 根据ID列表批量删除MAC地址记录
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "记录ID列表"
// @Success 200 {object} response.Response
// @Router /network/mac/batch-delete [post]
func (h *MACHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	lldpSvc := lldp.NewLLDPService(h.core.DeviceExecutor)
	filterRuleSvc := topology.NewFilterRuleService(h.core.DB.GetDB())
	svc := services.NewMACCollectionService(h.core.DB.GetDB(), h.core.DeviceExecutor, lldpSvc, filterRuleSvc)
	count, err := svc.BatchDelete(c.Request.Context(), req.IDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除MAC地址记录") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "MAC地址采集", operlog.OperTypeBatch)
	response.Success(c, gin.H{"deletedCount": count})
}
