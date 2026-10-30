package network

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	networkServices "github.com/xingran-next/xingran-go-backend/internal/services/network"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	responseHelpers "github.com/xingran-next/xingran-go-backend/pkg/response"
)

// DeviceHandler 网络设备处理器
type DeviceHandler struct {
	deviceService networkServices.CacheService
	core          *core.Core
}

// NewDeviceHandler 创建网络设备处理器实例
func NewDeviceHandler(deviceService networkServices.CacheService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *DeviceHandler) WithCore(core *core.Core) *DeviceHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 网络设备统计(总数/在线/离线/未知)
// @Summary 网络设备统计
// @Description 用 COUNT 聚合返回设备统计,供统计卡片使用
// @Tags 网络设备管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /network/devices/statistics [post]
func (h *DeviceHandler) Statistics(c *gin.Context) {
	stats, err := h.deviceService.GetDeviceStatistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}

// List 查询网络设备列表
// @Summary 查询网络设备列表
// @Description 分页查询网络设备列表
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body services.ListDeviceRequest true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /network/devices/list [post]
func (h *DeviceHandler) List(c *gin.Context) {
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		rawReq = make(map[string]interface{})
	}

	req := services.ListDeviceRequest{
		BaseListRequest: base.BaseListRequest{
			Current:  getIntField(rawReq, "current", 1),
			PageSize: getIntField(rawReq, "pageSize", 10),
			OrderByColumn: func() string {
				if v, ok := rawReq["orderByColumn"].(string); ok {
					return v
				}
				return ""
			}(),
			IsAsc: func() *bool {
				if v, ok := rawReq["isAsc"].(bool); ok {
					return &v
				}
				return nil
			}(),
		},
	}

	if val, ok := rawReq["deviceName"].(string); ok && val != "" {
		req.DeviceName = &val
	}
	if val, ok := rawReq["deviceType"].(string); ok && val != "" {
		dt := models.DeviceType(val)
		req.DeviceType = &dt
	}
	if val, ok := rawReq["vendor"].(string); ok && val != "" {
		vendor := models.DeviceVendor(val)
		req.Vendor = &vendor
	}
	if val, ok := rawReq["ip"].(string); ok && val != "" {
		req.IP = &val
	}
	if val, ok := rawReq["status"]; ok && val != nil {
		switch v := val.(type) {
		case float64:
			s := models.DeviceStatus(int(v))
			req.Status = &s
		case int:
			s := models.DeviceStatus(v)
			req.Status = &s
		}
	}
	if val, ok := rawReq["deptId"].(string); ok && val != "" {
		req.DeptID = &val
	}

	devices, total, err := h.deviceService.List(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	pageResp := response.PageResponse{
		List:     devices,
		Total:    total,
		Current:  req.Current,
		PageSize: req.PageSize,
	}
	response.Success(c, pageResp)
}

// GetByID 获取网络设备详情
// @Summary 获取网络设备详情
// @Description 根据设备ID获取详细信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} response.Response
// @Router /network/devices/{id} [post]
func (h *DeviceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("设备ID"))
		return
	}

	device, err := h.deviceService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InternalServerError(err))
		return
	}

	response.Success(c, device)
}

// Create 创建网络设备
// @Summary 创建网络设备
// @Description 添加新的网络设备
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body services.CreateDeviceRequest true "设备信息"
// @Success 200 {object} response.Response
// @Router /network/devices [post]
func (h *DeviceHandler) Create(c *gin.Context) {
	var req services.CreateDeviceRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	userID, _ := c.Get("user_id")
	req.CreatedBy = userID.(string)

	device, err := h.deviceService.Create(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "创建设备") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备", operlog.OperTypeCreate)
	response.Success(c, device)
}

// Update 更新网络设备
// @Summary 更新网络设备
// @Description 更新网络设备信息
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param request body services.UpdateDeviceRequest true "设备信息"
// @Success 200 {object} response.Response
// @Router /network/devices/{id}/update [post]
func (h *DeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("设备ID"))
		return
	}

	var req services.UpdateDeviceRequest
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	req.ID = id

	userID, _ := c.Get("user_id")
	req.UpdatedBy = userID.(string)

	err := h.deviceService.Update(c.Request.Context(), &req)
	if !responseHelpers.HandleServiceError(c, err, "更新设备") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除网络设备
// @Summary 删除网络设备
// @Description 删除指定网络设备
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} response.Response
// @Router /network/devices/{id}/delete [post]
func (h *DeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("设备ID"))
		return
	}

	err := h.deviceService.Delete(c.Request.Context(), id)
	if !responseHelpers.HandleServiceError(c, err, "删除设备") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备", operlog.OperTypeDelete)
	response.Success(c, gin.H{"message": "删除成功"})
}

// BatchDelete 批量删除网络设备
// @Summary 批量删除网络设备
// @Description 批量删除多个网络设备
// @Tags 网络设备管理
// @Accept json
// @Produce json
// @Param request body object{ids:[]string} true "设备ID列表"
// @Success 200 {object} response.Response
// @Router /network/devices/batch-delete [post]
func (h *DeviceHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if !responseHelpers.HandleJSONBinding(c, &req) {
		return
	}

	err := h.deviceService.BatchDelete(c.Request.Context(), req.IDs)
	if !responseHelpers.HandleServiceError(c, err, "批量删除设备") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备", operlog.OperTypeBatch)
	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.IDs),
	})
}
	// QuickCreate 快速创建设备（自动探测并创建）
	// @Summary 快速创建网络设备
	// @Description 通过 SNMP 探测自动获取设备信息并创建设备
	// @Tags 网络设备管理
	// @Accept json
	// @Produce json
	// @Param request body services.QuickCreateRequest true "快速创建请求"
	// @Success 200 {object} response.Response
	// @Router /network/devices/quick-create [post]
	func (h *DeviceHandler) QuickCreate(c *gin.Context) {
		var req services.QuickCreateRequest
		if !responseHelpers.HandleJSONBinding(c, &req) {
			return
		}

		userID, _ := c.Get("user_id")
		req.CreatedBy = userID.(string)

		device, err := h.deviceService.QuickCreateDevice(c.Request.Context(), &req)
		if !responseHelpers.HandleServiceError(c, err, "快速创建设备") {
			return
		}

		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络设备", operlog.OperTypeCreate)
		response.Success(c, gin.H{"id": device.ID})
	}

// ==================== 辅助函数 ====================

// getIntField 从map中获取整数字段，提供默认值
func getIntField(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return defaultValue
}

// getStringField 从map中获取字符串字段，缺省返回空串
func getStringField(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// getBoolFieldPtr 从map中获取布尔字段,返回 *bool(nil 表示未提供)。
// isAsc=true / isAsc=false 都要保留语义,因此用指针传递。
func getBoolFieldPtr(m map[string]interface{}, key string) *bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return &b
		}
	}
	return nil
}

// getOrderByColumn 从map中获取 orderByColumn 字符串字段,无值返回 ""。
// 用于服务端排序白名单透传,详见 internal/services/base/list_request.go。
func getOrderByColumn(m map[string]interface{}) string {
	if val, ok := m["orderByColumn"].(string); ok {
		return val
	}
	return ""
}

// getIsAscPtr 从map中获取 isAsc 布尔字段,返回 *bool 指针(nil 表示未传)。
// Service 层用 *bool 区分"未传(nil)" vs "传了 true/false" 三态。
func getIsAscPtr(m map[string]interface{}) *bool {
	if val, ok := m["isAsc"].(bool); ok {
		return &val
	}
	return nil
}
