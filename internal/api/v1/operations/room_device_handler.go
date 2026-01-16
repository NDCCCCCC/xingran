package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	if err := c.ShouldBindJSON(&device); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &device); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, device)
}

// List 查询机房设备列表
func (h *RoomDeviceHandler) List(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = make(map[string]interface{})
	}

	result, err := h.service.List(c.Request.Context(), params)
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
	if err := c.ShouldBindJSON(&device); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	device.ID = id
	if err := h.service.Update(c.Request.Context(), &device); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除机房设备
func (h *RoomDeviceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作
func (h *RoomDeviceHandler) BatchOperation(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	switch req.Action {
	case "delete":
		if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, http.StatusInternalServerError, "批量删除失败")
			return
		}
	default:
		response.Error(c, http.StatusBadRequest, "不支持的操作")
		return
	}

	response.Success(c, nil)
}
