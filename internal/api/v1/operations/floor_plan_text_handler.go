package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type FloorPlanTextHandler struct {
	service opsServices.FloorPlanTextService
}

func NewFloorPlanTextHandler(service opsServices.FloorPlanTextService) *FloorPlanTextHandler {
	return &FloorPlanTextHandler{service: service}
}

// Create 创建平面图文本
func (h *FloorPlanTextHandler) Create(c *gin.Context) {
	var text operationsmodels.FloorPlanText
	if !handleJSONBinding(c, &text) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &text), "创建") {
		return
	}

	response.Success(c, text)
}

// List 查询平面图文本列表（类型安全版本）
func (h *FloorPlanTextHandler) List(c *gin.Context) {
	var req requests.FloorPlanTextListRequest
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

// GetByID 获取平面图文本详情
func (h *FloorPlanTextHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	text, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "文本不存在")
		return
	}

	response.Success(c, text)
}

// Update 更新平面图文本
func (h *FloorPlanTextHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var text operationsmodels.FloorPlanText
	if !handleJSONBinding(c, &text) {
		return
	}

	text.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &text), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除平面图文本
func (h *FloorPlanTextHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
func (h *FloorPlanTextHandler) BatchOperation(c *gin.Context) {
	var req requests.FloorPlanTextBatchOperationRequest
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
