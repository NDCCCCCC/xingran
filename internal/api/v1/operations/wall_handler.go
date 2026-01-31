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
	if err := c.ShouldBindJSON(&wall); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &wall); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, wall)
}

func (h *WallHandler) List(c *gin.Context) {
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
	if err := c.ShouldBindJSON(&wall); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	wall.ID = id
	if err := h.service.Update(c.Request.Context(), &wall); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *WallHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *WallHandler) BatchOperation(c *gin.Context) {
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
