package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type RoomDeviceHandler struct {
	service opsServices.RoomDeviceService
}

func NewRoomDeviceHandler(service opsServices.RoomDeviceService) *RoomDeviceHandler {
	return &RoomDeviceHandler{service: service}
}

// Create 创建机房设备
func (h *RoomDeviceHandler) Create(c *gin.Context) {
	var device operations.OpsRoomDevice
	if !handleJSONBinding(c, &device) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &device), "创建") {
		return
	}

	response.Success(c, device)
}

// List 查询机房设备列表（类型安全版本）
func (h *RoomDeviceHandler) List(c *gin.Context) {
	var req requests.RoomDeviceListRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetByID 获取机房设备详情
func (h *RoomDeviceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	device, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "设备不存在")
		return
	}

	response.Success(c, device)
}

// Update 更新机房设备
func (h *RoomDeviceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var device operations.OpsRoomDevice
	if !handleJSONBinding(c, &device) {
		return
	}

	device.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &device), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除机房设备
func (h *RoomDeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
func (h *RoomDeviceHandler) BatchOperation(c *gin.Context) {
	var req requests.RoomDeviceBatchOperationRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	switch req.Action {
	case "delete":
		if !handleServiceError(c, h.service.BatchDelete(c.Request.Context(), req.IDs), "批量删除") {
			return
		}
	default:
		response.Error(c, http.StatusBadRequest, "不支持的操作")
		return
	}

	response.Success(c, nil)
}
