package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type FloorHandler struct {
	service opsServices.FloorService
}

func NewFloorHandler(service opsServices.FloorService) *FloorHandler {
	return &FloorHandler{service: service}
}

// Create 创建楼层
func (h *FloorHandler) Create(c *gin.Context) {
	var floor operations.OpsFloor
	if !handleJSONBinding(c, &floor) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &floor), "创建") {
		return
	}

	response.Success(c, floor)
}

// List 查询楼层列表
func (h *FloorHandler) List(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		// 如果JSON解析失败，使用空参数
		params = make(map[string]interface{})
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, result)
}

// GetTree 获取楼宇-楼层树
func (h *FloorHandler) GetTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "查询失败: "+err.Error())
		return
	}

	response.Success(c, tree)
}

// GetByID 获取楼层详情
func (h *FloorHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	floor, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "楼层不存在")
		return
	}

	response.Success(c, floor)
}

// Update 更新楼层
func (h *FloorHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var floor operations.OpsFloor
	if !handleJSONBinding(c, &floor) {
		return
	}

	floor.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &floor), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除楼层
func (h *FloorHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作
func (h *FloorHandler) BatchOperation(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
	}

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
