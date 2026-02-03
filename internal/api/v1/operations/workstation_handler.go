package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type WorkstationHandler struct {
	service opsServices.WorkstationService
}

func NewWorkstationHandler(service opsServices.WorkstationService) *WorkstationHandler {
	return &WorkstationHandler{service: service}
}

// Create 创建工位
func (h *WorkstationHandler) Create(c *gin.Context) {
	var workstation models.Workstation
	if !handleJSONBinding(c, &workstation) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &workstation), "创建") {
		return
	}

	response.Success(c, workstation)
}

// List 查询工位列表
func (h *WorkstationHandler) List(c *gin.Context) {
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

// GetByID 获取工位详情
func (h *WorkstationHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	workstation, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "工位不存在")
		return
	}

	response.Success(c, workstation)
}

// Update 更新工位
func (h *WorkstationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var workstation models.Workstation
	if !handleJSONBinding(c, &workstation) {
		return
	}

	workstation.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &workstation), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除工位
func (h *WorkstationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作
func (h *WorkstationHandler) BatchOperation(c *gin.Context) {
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

// BatchUpdatePositions 批量更新工位位置
func (h *WorkstationHandler) BatchUpdatePositions(c *gin.Context) {
	var req struct {
		Items []opsServices.PositionUpdateItem `json:"items" binding:"required"`
	}

	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.service.BatchUpdatePositions(c.Request.Context(), req.Items), "批量更新位置") {
		return
	}

	response.Success(c, nil)
}
