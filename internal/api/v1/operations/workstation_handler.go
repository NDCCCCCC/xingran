package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/operations"
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
	var workstation operations.OpsWorkstation
	if err := c.ShouldBindJSON(&workstation); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &workstation); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
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
	var workstation operations.OpsWorkstation
	if err := c.ShouldBindJSON(&workstation); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	workstation.ID = id
	if err := h.service.Update(c.Request.Context(), &workstation); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// Delete 删除工位
func (h *WorkstationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
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
