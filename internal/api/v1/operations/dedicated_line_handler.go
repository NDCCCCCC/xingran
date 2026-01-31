package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type DedicatedLineHandler struct {
	service opsServices.DedicatedLineService
}

func NewDedicatedLineHandler(service opsServices.DedicatedLineService) *DedicatedLineHandler {
	return &DedicatedLineHandler{service: service}
}

// Create 创建专线
func (h *DedicatedLineHandler) Create(c *gin.Context) {
	var line operations.OpsDedicatedLine
	if !handleJSONBinding(c, &line) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &line), "创建") {
		return
	}

	response.Success(c, line)
}

// List 查询专线列表（类型安全版本）
func (h *DedicatedLineHandler) List(c *gin.Context) {
	var req requests.DedicatedLineListRequest
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

// GetByID 获取专线详情
func (h *DedicatedLineHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	line, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "专线不存在")
		return
	}

	response.Success(c, line)
}

// Update 更新专线
func (h *DedicatedLineHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var line operations.OpsDedicatedLine
	if !handleJSONBinding(c, &line) {
		return
	}

	line.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &line), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除专线
func (h *DedicatedLineHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
func (h *DedicatedLineHandler) BatchOperation(c *gin.Context) {
	var req requests.DedicatedLineBatchOperationRequest
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
