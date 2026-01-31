package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type ServerRoomHandler struct {
	service opsServices.ServerRoomService
}

func NewServerRoomHandler(service opsServices.ServerRoomService) *ServerRoomHandler {
	return &ServerRoomHandler{service: service}
}

// Create 创建机房
func (h *ServerRoomHandler) Create(c *gin.Context) {
	var room operations.OpsServerRoom
	if !handleJSONBinding(c, &room) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &room), "创建") {
		return
	}

	response.Success(c, room)
}

// List 查询机房列表（类型安全版本）
func (h *ServerRoomHandler) List(c *gin.Context) {
	var req requests.ServerRoomListRequest
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

// GetByID 获取机房详情
func (h *ServerRoomHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	room, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "机房不存在")
		return
	}

	response.Success(c, room)
}

// Update 更新机房
func (h *ServerRoomHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var room operations.OpsServerRoom
	if !handleJSONBinding(c, &room) {
		return
	}

	room.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &room), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除机房
func (h *ServerRoomHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
func (h *ServerRoomHandler) BatchOperation(c *gin.Context) {
	var req requests.ServerRoomBatchOperationRequest
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
