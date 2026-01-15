package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	if err := c.ShouldBindJSON(&infoPoint); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &infoPoint); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, infoPoint)
}

// List 查询信息点列表
func (h *InfoPointHandler) List(c *gin.Context) {
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
	if err := c.ShouldBindJSON(&infoPoint); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	infoPoint.ID = id
	if err := h.service.Update(c.Request.Context(), &infoPoint); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除信息点
func (h *InfoPointHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作
func (h *InfoPointHandler) BatchOperation(c *gin.Context) {
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
