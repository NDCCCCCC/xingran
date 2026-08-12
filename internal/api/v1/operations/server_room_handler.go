package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

type ServerRoomHandler struct {
	service opsServices.ServerRoomService
	core    *core.Core
}

func NewServerRoomHandler(service opsServices.ServerRoomService) *ServerRoomHandler {
	return &ServerRoomHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *ServerRoomHandler) WithCore(core *core.Core) *ServerRoomHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 机房统计(读操作,不记操作日志)
func (h *ServerRoomHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// SearchServerRoomOptions 机房下拉数据源(name LIKE 模糊 + buildingId/floorId/status/orgId 筛选,LIMIT 50,读操作不写操作日志)
func (h *ServerRoomHandler) SearchServerRoomOptions(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = map[string]interface{}{}
	}
	result, err := h.service.SearchServerRoomOptions(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// Create 创建机房
// @Summary 创建机房
// @Description 创建新的机房信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body operations.OpsServerRoom true "机房信息"
// @Success 200 {object} response.Response{data=operations.OpsServerRoom}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom [post]
func (h *ServerRoomHandler) Create(c *gin.Context) {
	var room operations.OpsServerRoom
	if !handleJSONBinding(c, &room) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &room), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "机房管理", operlog.OperTypeCreate)
	response.Success(c, room)
}

// List 查询机房列表（类型安全版本）
// @Summary 查询机房列表
// @Description 分页查询机房列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.ServerRoomListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]operations.OpsServerRoom,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom/list [post]
func (h *ServerRoomHandler) List(c *gin.Context) {
	var req requests.ServerRoomListRequest
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

// GetByID 获取机房详情
// @Summary 获取机房详情
// @Description 根据ID获取机房详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "机房ID"
// @Success 200 {object} response.Response{data=operations.OpsServerRoom}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/serverRoom/{id} [post]
func (h *ServerRoomHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	room, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.ServerRoomNotFound())
		return
	}

	response.Success(c, room)
}

// Update 更新机房
// @Summary 更新机房
// @Description 更新机房信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "机房ID"
// @Param request body operations.OpsServerRoom true "机房信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom/{id}/update [post]
func (h *ServerRoomHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var room operations.OpsServerRoom
	if !handleJSONBinding(c, &room) {
		return
	}

	room.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &room), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "机房管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除机房
// @Summary 删除机房
// @Description 根据ID删除机房
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "机房ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom/{id}/delete [post]
func (h *ServerRoomHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "机房管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
// @Summary 批量操作
// @Description 对机房进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.ServerRoomBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom/batch [post]
func (h *ServerRoomHandler) BatchOperation(c *gin.Context) {
	var req requests.ServerRoomBatchOperationRequest
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "机房管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}
