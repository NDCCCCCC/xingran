package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	operationsmodels "github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

const (
	errInvalidParams  = "参数错误"
	errCreateFailed   = "创建失败"
	errUpdateFailed   = "更新失败"
	errDeleteFailed   = "删除失败"
	errQueryFailed    = "查询失败"
	errNotFound       = "文本不存在"
	errInvalidAction  = "不支持的操作"
	errBatchDelete    = "批量删除失败"
)

type FloorPlanTextHandler struct {
	service opsServices.FloorPlanTextService
}

func NewFloorPlanTextHandler(service opsServices.FloorPlanTextService) *FloorPlanTextHandler {
	return &FloorPlanTextHandler{service: service}
}

func (h *FloorPlanTextHandler) Create(c *gin.Context) {
	var text operationsmodels.FloorPlanText
	if err := c.ShouldBindJSON(&text); err != nil {
		response.Error(c, http.StatusBadRequest, errInvalidParams+": "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &text); err != nil {
		response.Error(c, http.StatusInternalServerError, errCreateFailed+": "+err.Error())
		return
	}

	response.Success(c, text)
}

func (h *FloorPlanTextHandler) List(c *gin.Context) {
	params := make(map[string]interface{})
	_ = c.ShouldBindJSON(&params)

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errQueryFailed+": "+err.Error())
		return
	}

	response.Success(c, result)
}

func (h *FloorPlanTextHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	text, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, errNotFound)
		return
	}

	response.Success(c, text)
}

func (h *FloorPlanTextHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var text operationsmodels.FloorPlanText
	if err := c.ShouldBindJSON(&text); err != nil {
		response.Error(c, http.StatusBadRequest, errInvalidParams+": "+err.Error())
		return
	}

	text.ID = id
	if err := h.service.Update(c.Request.Context(), &text); err != nil {
		response.Error(c, http.StatusInternalServerError, errUpdateFailed+": "+err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *FloorPlanTextHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, errDeleteFailed+": "+err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *FloorPlanTextHandler) BatchOperation(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errInvalidParams)
		return
	}

	if req.Action == "delete" {
		if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, http.StatusInternalServerError, errBatchDelete+": "+err.Error())
			return
		}
		response.Success(c, nil)
		return
	}

	response.Error(c, http.StatusBadRequest, errInvalidAction)
}
