package operations

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type RoomDeviceHandler struct {
	service opsServices.RoomDeviceService
}

func NewRoomDeviceHandler(service opsServices.RoomDeviceService) *RoomDeviceHandler {
	return &RoomDeviceHandler{service: service}
}

// Create 创建机房设备
// @Summary 创建机房设备
// @Description 创建新的机房设备信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body operations.OpsRoomDevice true "机房设备信息"
// @Success 200 {object} response.Response{data=operations.OpsRoomDevice}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice [post]
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
// @Summary 查询机房设备列表
// @Description 分页查询机房设备列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.RoomDeviceListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]operations.OpsRoomDevice,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice/list [post]
func (h *RoomDeviceHandler) List(c *gin.Context) {
	var req requests.RoomDeviceListRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
		return
	}

	response.Success(c, result)
}

// GetByID 获取机房设备详情
// @Summary 获取机房设备详情
// @Description 根据ID获取机房设备详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "机房设备ID"
// @Success 200 {object} response.Response{data=operations.OpsRoomDevice}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/roomDevice/{id} [post]
func (h *RoomDeviceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	device, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.RoomDeviceNotFound())
		return
	}

	response.Success(c, device)
}

// Update 更新机房设备
// @Summary 更新机房设备
// @Description 更新机房设备信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "机房设备ID"
// @Param request body operations.OpsRoomDevice true "机房设备信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice/{id}/update [post]
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
// @Summary 删除机房设备
// @Description 根据ID删除机房设备
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "机房设备ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice/{id}/delete [post]
func (h *RoomDeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作
// @Summary 批量操作
// @Description 对机房设备进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.RoomDeviceBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice/batch [post]
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
		response.Error(c, apperrors.InvalidOperation("不支持的操作"))
		return
	}

	response.Success(c, nil)
}
