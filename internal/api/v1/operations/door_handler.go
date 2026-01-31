package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	if err := c.ShouldBindJSON(&door); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Create(c.Request.Context(), &door); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建失败: "+err.Error())
		return
	}

	response.Success(c, door)
}

func (h *DoorHandler) List(c *gin.Context) {
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
	if err := c.ShouldBindJSON(&door); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	door.ID = id
	if err := h.service.Update(c.Request.Context(), &door); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *DoorHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *DoorHandler) BatchOperation(c *gin.Context) {
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
