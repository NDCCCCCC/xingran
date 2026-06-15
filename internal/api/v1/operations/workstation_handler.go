package operations

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

type WorkstationHandler struct {
	service opsServices.WorkstationService
	core    *core.Core
}

func NewWorkstationHandler(service opsServices.WorkstationService) *WorkstationHandler {
	return &WorkstationHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *WorkstationHandler) WithCore(core *core.Core) *WorkstationHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Create 创建工位
// @Summary 创建工位
// @Description 创建新的工位信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body models.Workstation true "工位信息"
// @Success 200 {object} response.Response{data=models.Workstation}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation [post]
func (h *WorkstationHandler) Create(c *gin.Context) {
	var workstation models.Workstation
	if !handleJSONBinding(c, &workstation) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &workstation), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeCreate)
	response.Success(c, workstation)
}

// List 查询工位列表
// @Summary 查询工位列表
// @Description 分页查询工位列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int,name=string,floorId=string} true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]models.Workstation,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation/list [post]
func (h *WorkstationHandler) List(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = make(map[string]interface{})
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
		return
	}

	response.Success(c, result)
}

// GetByID 获取工位详情
// @Summary 获取工位详情
// @Description 根据ID获取工位详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response{data=models.Workstation}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/workstation/{id} [post]
func (h *WorkstationHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	workstation, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.WorkstationNotFound())
		return
	}

	response.Success(c, workstation)
}

// Update 更新工位
// @Summary 更新工位
// @Description 更新工位信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Param request body models.Workstation true "工位信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation/{id}/update [post]
func (h *WorkstationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var workstation models.Workstation
	if !handleJSONBinding(c, &workstation) {
		return
	}

	workstation.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &workstation), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除工位
// @Summary 删除工位
// @Description 根据ID删除工位
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "工位ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation/{id}/delete [post]
func (h *WorkstationHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作
// @Summary 批量操作
// @Description 对工位进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string,action=string} true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation/batch [post]
func (h *WorkstationHandler) BatchOperation(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
	}

	if !handleJSONBinding(c, &req) {
		return
	}

	switch req.Action {
	case "delete":
		if !handleServiceError(c, h.service.BatchDelete(c.Request.Context(), req.IDs), "批量删除") {
			return
		}
	default:
		response.Error(c, apperrors.InvalidOperation("不支持的操作"))
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}

// BatchUpdatePositions 批量更新工位位置
// @Summary 批量更新工位位置
// @Description 批量更新工位的位置坐标信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{items=[]object{id=string,x=float64,y=float64}} true "位置更新参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation/positions [post]
func (h *WorkstationHandler) BatchUpdatePositions(c *gin.Context) {
	var req struct {
		Items []opsServices.PositionUpdateItem `json:"items" binding:"required"`
	}

	if !handleJSONBinding(c, &req) {
		return
	}

	if !handleServiceError(c, h.service.BatchUpdatePositions(c.Request.Context(), req.Items), "批量更新位置") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "工位管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}
