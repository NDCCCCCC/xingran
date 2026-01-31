package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (h *FloorPlanTextHandler) Create(c *gin.Context) {
	var text operationsmodels.FloorPlanText
	if err := c.ShouldBindJSON(&text); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &text); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, text)
}

func (h *FloorPlanTextHandler) List(c *gin.Context) {
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

func (h *FloorPlanTextHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	text, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, "文本不存在")
		return
	}

	response.Success(c, text)
}

func (h *FloorPlanTextHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var text operationsmodels.FloorPlanText
	if err := c.ShouldBindJSON(&text); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	// 解析 UUID
	parsedID, err := uuid.Parse(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID格式")
		return
	}
	text.ID = parsedID

	if err := h.service.Update(c.Request.Context(), &text); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *FloorPlanTextHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
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
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	switch req.Action {
	case "delete":
		if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
			response.Error(c, http.StatusInternalServerError, "批量删除失败: "+err.Error())
			return
		}
		response.Success(c, nil)
	default:
		response.Error(c, http.StatusBadRequest, "不支持的操作")
	}
}
