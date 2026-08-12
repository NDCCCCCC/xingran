package operations

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

type FloorHandler struct {
	service opsServices.FloorService
	core    *core.Core
}

func NewFloorHandler(service opsServices.FloorService) *FloorHandler {
	return &FloorHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *FloorHandler) WithCore(core *core.Core) *FloorHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Statistics 楼层统计(读操作,不记操作日志)
func (h *FloorHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// SearchFloorOptions 楼层下拉数据源(name LIKE 模糊 + buildingId/orgId/status 筛选,LIMIT 50,读操作不写操作日志)
// 替代 frontend Select/Cascader children 用分页 list 当全集源导致的截断问题。
func (h *FloorHandler) SearchFloorOptions(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = map[string]interface{}{}
	}
	result, err := h.service.SearchFloorOptions(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// Create 创建楼层
// @Summary 创建楼层
// @Description 创建新的楼层信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body operations.OpsFloor true "楼层信息"
// @Success 200 {object} response.Response{data=operations.OpsFloor}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor [post]
func (h *FloorHandler) Create(c *gin.Context) {
	var floor operations.OpsFloor
	if !handleJSONBinding(c, &floor) {
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &floor), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层管理", operlog.OperTypeCreate)
	response.Success(c, floor)
}

// List 查询楼层列表
// @Summary 查询楼层列表
// @Description 分页查询楼层列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{current=int,pageSize=int,name=string,buildingId=string} true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]operations.OpsFloor,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor/list [post]
func (h *FloorHandler) List(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		// 如果JSON解析失败，使用空参数
		params = make(map[string]interface{})
	}

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
		return
	}

	response.Success(c, result)
}

// GetTree 获取楼宇-楼层树
// @Summary 获取楼宇-楼层树
// @Description 获取楼宇和楼层的树形结构数据
// @Tags 运维管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]object}
// @Failure 500 {object} response.Response
// @Router /ops/floor/tree [post]
func (h *FloorHandler) GetTree(c *gin.Context) {
	tree, err := h.service.GetTree(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
		return
	}

	response.Success(c, tree)
}

// GetByID 获取楼层详情
// @Summary 获取楼层详情
// @Description 根据ID获取楼层详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "楼层ID"
// @Success 200 {object} response.Response{data=operations.OpsFloor}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/floor/{id} [post]
func (h *FloorHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	floor, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.FloorNotFound())
		return
	}

	response.Success(c, floor)
}

// Update 更新楼层
// @Summary 更新楼层
// @Description 更新楼层信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "楼层ID"
// @Param request body operations.OpsFloor true "楼层信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor/{id}/update [post]
func (h *FloorHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var floor operations.OpsFloor
	if !handleJSONBinding(c, &floor) {
		return
	}

	floor.ID = id
	if !handleServiceError(c, h.service.Update(c.Request.Context(), &floor), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除楼层
// @Summary 删除楼层
// @Description 根据ID删除楼层
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "楼层ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor/{id}/delete [post]
func (h *FloorHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作
// @Summary 批量操作
// @Description 对楼层进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string,action=string} true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor/batch [post]
func (h *FloorHandler) BatchOperation(c *gin.Context) {
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼层管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}
