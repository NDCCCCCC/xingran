package network

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	networkServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/network"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	responseHelpers "github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
)

// DeviceHandler 网络设备处理器
type DeviceHandler struct {
	deviceService networkServices.CacheService
}

// NewDeviceHandler 创建网络设备处理器实例
func NewDeviceHandler(deviceService networkServices.CacheService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
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
		Current:  getIntField(rawReq, "current", 1),
		PageSize: getIntField(rawReq, "pageSize", 10),
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

	response.Success(c, gin.H{
		"message": "批量删除成功",
		"count":   len(req.IDs),
	})
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
