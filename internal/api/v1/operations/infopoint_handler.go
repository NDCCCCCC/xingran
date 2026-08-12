package operations

import (
	"context"
	"fmt"
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

type InfoPointHandler struct {
	service opsServices.InfoPointService
	core    *core.Core
}

func NewInfoPointHandler(service opsServices.InfoPointService) *InfoPointHandler {
	return &InfoPointHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *InfoPointHandler) WithCore(core *core.Core) *InfoPointHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// validateDevicePortConsistency 防漂移 (2026-07-01): 当 device_id 和 port_id 都非空时,
// 校验 port_status 表中该 port_id 归属 device_id 与 info_point 指定一致。
// 返回 (ok, errMsg)。ok=false 时 handler 应 4xx 返回。
//
// 设计目的: 阻止 REST API 产生新的 device_id/port_id 漂移(历史 1247 行由 Excel 导入累积)。
// 旧漂移数据不受影响 — query 端已通过砍 strict JOIN 兼容。
func (h *InfoPointHandler) validateDevicePortConsistency(ctx context.Context, deviceID, portID *string) (bool, string) {
	if deviceID == nil || *deviceID == "" || portID == nil || *portID == "" {
		return true, "" // 任意字段为空则跳过 (port_id 在 infoPoint 配置中非必填)
	}
	if h.core == nil || h.core.GetDB() == nil {
		return true, "" // 没有 DB (单元测试场景) 跳过
	}

	// sys_device_port_status.id 是 uuid PK; portID 是 varchar 存 UUID 字符串
	var portDeviceID string
	err := h.core.GetDB().WithContext(ctx).
		Table("sys_device_port_status").
		Select("device_id").
		Where("id = ?", *portID).
		Scan(&portDeviceID).Error
	if err != nil {
		return false, fmt.Sprintf("校验端口归属失败: %v", err)
	}
	if portDeviceID == "" {
		return false, fmt.Sprintf("端口不存在 (portId=%s)", *portID)
	}
	if portDeviceID != *deviceID {
		return false, fmt.Sprintf("端口归属设备不一致: info_point.device_id=%s, port.device_id=%s", *deviceID, portDeviceID)
	}
	return true, ""
}

// Statistics 信息点统计(读操作,不记操作日志)
func (h *InfoPointHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// SearchInfoPointOptions 信息点下拉数据源(name LIKE 模糊 + workstationId/type/status 筛选,LIMIT 50,读操作不写操作日志)
func (h *InfoPointHandler) SearchInfoPointOptions(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = map[string]interface{}{}
	}
	result, err := h.service.SearchInfoPointOptions(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// Create 创建信息点
// @Summary 创建信息点
// @Description 创建新的信息点
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body operations.OpsInfoPoint true "信息点"
// @Success 200 {object} response.Response{data=operations.OpsInfoPoint}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint [post]
func (h *InfoPointHandler) Create(c *gin.Context) {
	var infoPoint operations.OpsInfoPoint
	if !handleJSONBinding(c, &infoPoint) {
		return
	}

	// 防漂移 (2026-07-01): device_id↔port_id 一致性校验, 阻止 REST API 产生新漂移
	if ok, errMsg := h.validateDevicePortConsistency(c.Request.Context(), infoPoint.DeviceID, infoPoint.PortID); !ok {
		response.Error(c, http.StatusBadRequest, errMsg)
		return
	}

	if !handleServiceError(c, h.service.Create(c.Request.Context(), &infoPoint), "创建") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "信息点管理", operlog.OperTypeCreate)
	response.Success(c, infoPoint)
}

// List 查询信息点列表（类型安全版本）
// @Summary 查询信息点列表
// @Description 分页查询信息点列表，支持按条件筛选
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.InfoPointListRequest true "查询参数"
// @Success 200 {object} response.Response{data=object{list=[]operations.OpsInfoPoint,total=int}}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/list [post]
func (h *InfoPointHandler) List(c *gin.Context) {
	var req requests.InfoPointListRequest
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

// GetByID 获取信息点详情
// @Summary 获取信息点详情
// @Description 根据ID获取信息点详细信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "信息点ID"
// @Success 200 {object} response.Response{data=operations.OpsInfoPoint}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /ops/infoPoint/{id} [post]
func (h *InfoPointHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	infoPoint, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, apperrors.InfoPointNotFound())
		return
	}

	response.Success(c, infoPoint)
}

// Update 更新信息点
// @Summary 更新信息点
// @Description 更新信息点信息
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "信息点ID"
// @Param request body operations.OpsInfoPoint true "信息点"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/{id}/update [post]
func (h *InfoPointHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var infoPoint operations.OpsInfoPoint
	if !handleJSONBinding(c, &infoPoint) {
		return
	}

	infoPoint.ID = id
	// 防漂移 (2026-07-01): device_id↔port_id 一致性校验, 阻止 REST API 产生新漂移
	if ok, errMsg := h.validateDevicePortConsistency(c.Request.Context(), infoPoint.DeviceID, infoPoint.PortID); !ok {
		response.Error(c, http.StatusBadRequest, errMsg)
		return
	}

	if !handleServiceError(c, h.service.Update(c.Request.Context(), &infoPoint), "更新") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "信息点管理", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// Delete 删除信息点
// @Summary 删除信息点
// @Description 根据ID删除信息点
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param id path string true "信息点ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/{id}/delete [post]
func (h *InfoPointHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !handleServiceError(c, h.service.Delete(c.Request.Context(), id), "删除") {
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "信息点管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchOperation 批量操作（类型安全版本）
// @Summary 批量操作
// @Description 对信息点进行批量操作，如批量删除
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body requests.InfoPointBatchOperationRequest true "批量操作参数"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/batch [post]
func (h *InfoPointHandler) BatchOperation(c *gin.Context) {
	var req requests.InfoPointBatchOperationRequest
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "信息点管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}
