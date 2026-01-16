package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	if err := c.ShouldBindJSON(&line); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &line); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, line)
}

// List 查询专线列表
func (h *DedicatedLineHandler) List(c *gin.Context) {
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
	if err := c.ShouldBindJSON(&line); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	line.ID = id
	if err := h.service.Update(c.Request.Context(), &line); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除专线
func (h *DedicatedLineHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// BatchOperation 批量操作
func (h *DedicatedLineHandler) BatchOperation(c *gin.Context) {
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
