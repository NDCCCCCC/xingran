package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type WallHandler struct {
	service opsServices.WallService
}

func NewWallHandler(service opsServices.WallService) *WallHandler {
	return &WallHandler{service: service}
}

func (h *WallHandler) Create(c *gin.Context) {
	var wall operationsmodels.Wall
	if !handleJSONBinding(c, &wall) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &wall), "创建") {
		return
	}

	response.Success(c, wall)
}

func (h *WallHandler) List(c *gin.Context) {
	var params map[string]interface{}
	c.ShouldBindJSON(&params)

	result, err := h.service.List(c.Request.Context(), params)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}

func (h *WallHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	wall, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "墙体不存在")
		return
	}

	response.Success(c, wall)
}

func (h *WallHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var wall operationsmodels.Wall
	if !handleJSONBinding(c, &wall) {
		return
	}

	wall.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &wall), "更新") {
		return
	}

	response.Success(c, nil)
}

func (h *WallHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

func (h *WallHandler) BatchOperation(c *gin.Context) {
	var req BatchOperationRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if req.Action != "delete" {
		response.Error(c, http.StatusBadRequest, "不支持的操作: "+req.Action)
		return
	}

	if !handleServiceError(c, h.service.BatchDelete(c.Request.Context(), req.IDs), "批量删除") {
		return
	}

	response.Success(c, nil)
}
