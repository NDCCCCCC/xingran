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

type DedicatedLineHandler struct {
	service opsServices.DedicatedLineService
	core    *core.Core
}

func NewDedicatedLineHandler(service opsServices.DedicatedLineService) *DedicatedLineHandler {
	return &DedicatedLineHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *DedicatedLineHandler) WithCore(core *core.Core) *DedicatedLineHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 专线统计(读操作,不记操作日志)
func (h *DedicatedLineHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// SearchDedicatedLineOptions 专线下拉数据源(name LIKE 模糊 + lineType/ISP/sourceRoomId/destRoomId/status 筛选,LIMIT 50,读操作不写操作日志)
func (h *DedicatedLineHandler) SearchDedicatedLineOptions(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = map[string]interface{}{}
	}
	result, err := h.service.SearchDedicatedLineOptions(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// Create 创建专线
// @Summary 创建专线
// @Description 创建新的专线信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body operations.OpsDedicatedLine true "专线信息"
// @Success 200 {object} response.Response{data=operations.OpsDedicatedLine}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine [post]
func (h *DedicatedLineHandler) Create(c *gin.Context) {
	var line operations.OpsDedicatedLine
	if !handleJSONBinding(c, &line) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &line), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "专线管理", operlog.OperTypeCreate)
	response.Success(c, line)
}

// List 查询专线列表（类型安全版本）
// @Summary 查询专线列表
// @Description 分页查询专线列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.DedicatedLineListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]operations.OpsDedicatedLine,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine/list [post]
func (h *DedicatedLineHandler) List(c *gin.Context) {
	var req requests.DedicatedLineListRequest
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

// GetByID 获取专线详情
// @Summary 获取专线详情
// @Description 根据ID获取专线详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "专线ID"
// @Success 200 {object} response.Response{data=operations.OpsDedicatedLine}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/dedicatedLine/{id} [post]
func (h *DedicatedLineHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	line, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.DedicatedLineNotFound())
		return
	}

	response.Success(c, line)
}

// Update 更新专线
// @Summary 更新专线
// @Description 更新专线信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "专线ID"
// @Param request body operations.OpsDedicatedLine true "专线信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine/{id}/update [post]
func (h *DedicatedLineHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var line operations.OpsDedicatedLine
	if !handleJSONBinding(c, &line) {
		return
	}

	line.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &line), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "专线管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除专线
// @Summary 删除专线
// @Description 根据ID删除专线
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "专线ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine/{id}/delete [post]
func (h *DedicatedLineHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "专线管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
// @Summary 批量操作
// @Description 对专线进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.DedicatedLineBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine/batch [post]
func (h *DedicatedLineHandler) BatchOperation(c *gin.Context) {
	var req requests.DedicatedLineBatchOperationRequest
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "专线管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}
