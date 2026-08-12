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

type DoorHandler struct {
	service opsServices.DoorService
	core    *core.Core
}

func NewDoorHandler(service opsServices.DoorService) *DoorHandler {
	return &DoorHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *DoorHandler) WithCore(core *core.Core) *DoorHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Create 创建门
// @Summary 创建门
// @Description 创建新的门信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{floorId=string,wallId=string,position=string,angle=int,type=string,direction=string,width=int,length=int,color=string,name=string,remark=string} true "门信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/doors [post]
func (h *DoorHandler) Create(c *gin.Context) {
	var door operationsmodels.Door
	if !handleJSONBinding(c, &door) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &door), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "门窗管理", operlog.OperTypeCreate)
	response.Success(c, door)
}

// List 查询门列表
// @Summary 查询门列表
// @Description 分页查询门列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.DoorListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]object,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/doors/list [post]
func (h *DoorHandler) List(c *gin.Context) {
	var req requests.DoorListRequest
	if !handleJSONBinding(c, &req) {
		return
	}

	result, err := h.service.List(c.Request.Context(), req)
	if !handleServiceError(c, err, "查询") {
		return
	}

	response.Success(c, result)
}

// GetByID 获取门详情
// @Summary 获取门详情
// @Description 根据ID获取门详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "门ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/doors/{id} [post]
func (h *DoorHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	door, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.DoorNotFound())
		return
	}

	response.Success(c, door)
}

// Update 更新门
// @Summary 更新门
// @Description 更新门信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "门ID"
// @Param request body object{floorId=string,wallId=string,position=string,angle=int,type=string,direction=string,width=int,length=int,color=string,name=string,remark=string} true "门信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/doors/{id}/update [post]
func (h *DoorHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var door operationsmodels.Door
	if !handleJSONBinding(c, &door) {
		return
	}

	door.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &door), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "门窗管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除门
// @Summary 删除门
// @Description 根据ID删除门
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "门ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/doors/{id}/delete [post]
func (h *DoorHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "门窗管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作
// @Summary 批量操作
// @Description 对门进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.DoorBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/doors/batch [post]
func (h *DoorHandler) BatchOperation(c *gin.Context) {
	var req requests.DoorBatchOperationRequest
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "门窗管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}
