package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/operations/requests"
	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type DoorHandler struct {
	service opsServices.DoorService
}

func NewDoorHandler(service opsServices.DoorService) *DoorHandler {
	return &DoorHandler{service: service}
}

func (h *DoorHandler) Create(c *gin.Context) {
	var door operationsmodels.Door
	if !handleJSONBinding(c, &door) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &door), "创建") {
		return
	}

	response.Success(c, door)
}

func (h *DoorHandler) List(c *gin.Context) {
	var req requests.DoorListRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	result, err := h.service.List(c.Request.Context(), req)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}

func (h *DoorHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	door, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "门不存在")
		return
	}

	response.Success(c, door)
}

func (h *DoorHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var door operationsmodels.Door
	if !handleJSONBinding(c, &door) {
		return
	}

	door.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &door), "更新") {
		return
	}

	response.Success(c, nil)
}

func (h *DoorHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	response.Success(c, nil)
}

func (h *DoorHandler) BatchOperation(c *gin.Context) {
	var req requests.DoorBatchOperationRequest
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
