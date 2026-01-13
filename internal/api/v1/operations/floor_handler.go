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
	if err := c.ShouldBindJSON(&floor); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &floor); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, floor)
}

// List 查询楼层列表
func (h *FloorHandler) List(c *gin.Context) {
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
	if err := c.ShouldBindJSON(&floor); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	floor.ID = id
	if err := h.service.Update(c.Request.Context(), &floor); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除楼层
func (h *FloorHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
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
