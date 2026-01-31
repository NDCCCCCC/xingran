package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type InfoPointHandler struct {
	service opsServices.InfoPointService
}

func NewInfoPointHandler(service opsServices.InfoPointService) *InfoPointHandler {
	return &InfoPointHandler{service: service}
}

// Create 创建信息点
func (h *InfoPointHandler) Create(c *gin.Context) {
	var infoPoint operations.OpsInfoPoint
	if !handleJSONBinding(c, &infoPoint) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &infoPoint), "创建") {
		return
	}

	response.Success(c, infoPoint)
}

// List 查询信息点列表（类型安全版本）
func (h *InfoPointHandler) List(c *gin.Context) {
	var req requests.InfoPointListRequest
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

// GetByID 获取信息点详情
func (h *InfoPointHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	infoPoint, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "信息点不存在")
		return
	}

	response.Success(c, infoPoint)
}

// Update 更新信息点
func (h *InfoPointHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var infoPoint operations.OpsInfoPoint
	if !handleJSONBinding(c, &infoPoint) {
		return
	}

	infoPoint.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &infoPoint), "更新") {
		return
	}

	response.Success(c, nil)
}

// Delete 删除信息点
func (h *InfoPointHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
func (h *InfoPointHandler) BatchOperation(c *gin.Context) {
	var req requests.InfoPointBatchOperationRequest
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
