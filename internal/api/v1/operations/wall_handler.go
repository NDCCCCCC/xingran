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

type WallHandler struct {
	service opsServices.WallService
	core    *core.Core
}

func NewWallHandler(service opsServices.WallService) *WallHandler {
	return &WallHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *WallHandler) WithCore(core *core.Core) *WallHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Create 创建墙体
// @Summary 创建墙体
// @Description 创建新的墙体信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{floorId=string,type=string,points=string,thickness=int,height=float64,color=string,name=string,remark=string} true "墙体信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/walls [post]
func (h *WallHandler) Create(c *gin.Context) {
	var wall operationsmodels.Wall
	if !handleJSONBinding(c, &wall) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &wall), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "墙体管理", operlog.OperTypeCreate)
	response.Success(c, wall)
}

// List 查询墙体列表
// @Summary 查询墙体列表
// @Description 分页查询墙体列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.WallListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]object,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/walls/list [post]
func (h *WallHandler) List(c *gin.Context) {
	var req requests.WallListRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	result, err := h.service.List(c.Request.Context(), req)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}

// GetByID 获取墙体详情
// @Summary 获取墙体详情
// @Description 根据ID获取墙体详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "墙体ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/walls/{id} [post]
func (h *WallHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	wall, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.WallNotFound())
		return
	}

	response.Success(c, wall)
}

// Update 更新墙体
// @Summary 更新墙体
// @Description 更新墙体信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "墙体ID"
// @Param request body object{floorId=string,type=string,points=string,thickness=int,height=float64,color=string,name=string,remark=string} true "墙体信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/walls/{id}/update [post]
func (h *WallHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var wall operationsmodels.Wall
	if !handleJSONBinding(c, &wall) {
		return
	}

	wall.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &wall), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "墙体管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除墙体
// @Summary 删除墙体
// @Description 根据ID删除墙体
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "墙体ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/walls/{id}/delete [post]
func (h *WallHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "墙体管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作
// @Summary 批量操作
// @Description 对墙体进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.WallBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/walls/batch [post]
func (h *WallHandler) BatchOperation(c *gin.Context) {
	var req requests.WallBatchOperationRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	if req.Action != "delete" {
		response.Error(c, apperrors.InvalidOperationWithMsg("不支持的操作: "+req.Action))
		return
	}

	if !handleServiceError(c, h.service.BatchDelete(c.Request.Context(), req.IDs), "批量删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "墙体管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}
