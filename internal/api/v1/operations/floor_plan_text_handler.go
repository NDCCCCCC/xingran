package operations

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

type FloorPlanTextHandler struct {
	service opsServices.FloorPlanTextService
	core    *core.Core
}

func NewFloorPlanTextHandler(service opsServices.FloorPlanTextService) *FloorPlanTextHandler {
	return &FloorPlanTextHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *FloorPlanTextHandler) WithCore(core *core.Core) *FloorPlanTextHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Create 创建平面图文本
// @Summary 创建平面图文本
// @Description 创建新的平面图文本信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{floorId=string,text=string,position=object,fontSize=int,fontFamily=string,color=string,backgroundColor=string,bold=bool,italic=bool} true "平面图文本信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor-plan-texts [post]
func (h *FloorPlanTextHandler) Create(c *gin.Context) {
	var text operationsmodels.FloorPlanText
	if !handleJSONBinding(c, &text) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &text), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层文本", operlog.OperTypeCreate)
	response.Success(c, text)
}

// List 查询平面图文本列表（类型安全版本）
// @Summary 查询平面图文本列表
// @Description 分页查询平面图文本列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.FloorPlanTextListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]object,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor-plan-texts/list [post]
func (h *FloorPlanTextHandler) List(c *gin.Context) {
	var req requests.FloorPlanTextListRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	result, err := h.service.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
		return
	}

	response.Success(c, result)
}

// GetByID 获取平面图文本详情
// @Summary 获取平面图文本详情
// @Description 根据ID获取平面图文本详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "平面图文本ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/floor-plan-texts/{id} [post]
func (h *FloorPlanTextHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	text, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.FloorPlanTextNotFound())
		return
	}

	response.Success(c, text)
}

// Update 更新平面图文本
// @Summary 更新平面图文本
// @Description 更新平面图文本信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "平面图文本ID"
// @Param request body object{floorId=string,text=string,position=object,fontSize=int,fontFamily=string,color=string,backgroundColor=string,bold=bool,italic=bool} true "平面图文本信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor-plan-texts/{id}/update [post]
func (h *FloorPlanTextHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var text operationsmodels.FloorPlanText
	if !handleJSONBinding(c, &text) {
		return
	}

	text.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &text), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层文本", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除平面图文本
// @Summary 删除平面图文本
// @Description 根据ID删除平面图文本
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "平面图文本ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor-plan-texts/{id}/delete [post]
func (h *FloorPlanTextHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层文本", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
// @Summary 批量操作
// @Description 对平面图文本进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.FloorPlanTextBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor-plan-texts/batch [post]
func (h *FloorPlanTextHandler) BatchOperation(c *gin.Context) {
	var req requests.FloorPlanTextBatchOperationRequest
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层文本", operlog.OperTypeBatch)
	response.Success(c, nil)
}
