package operations

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

type WorkstationHandler struct {
	service          opsServices.WorkstationService
	reconciliationSvc asset.ReconciliationService
	core             *core.Core
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

// WithReconciliationService 注入跨模块 ReconciliationService 依赖 (Phase 45 R4 / D-A1-01)
//
// 跨模块 service 注入模式 (cross-module-permission.md §2.3 + ADAccountHandler 类比):
//   - WorkstationHandler 持有 asset.ReconciliationService 句柄
//   - GetByID 内根据 hasReconciliationPerm 决定是否调 GetByWorkstation + 注入 ReconciliationVisible
//   - 无权限时 → 静默降级(ReconciliationVisible=false + ReconciliationHiddenReason)
//   - 不调 c.Abort(),不返回 403,与 D-A1-03 锁定一致
func (h *WorkstationHandler) WithReconciliationService(svc asset.ReconciliationService) *WorkstationHandler {
	if h != nil {
		h.reconciliationSvc = svc
	}
	return h
}

// Statistics 工位统计(读操作,不记操作日志;支持 orgId 部门筛选)
func (h *WorkstationHandler) Statistics(c *gin.Context) {
	var params map[string]interface{}
	_ = c.ShouldBindJSON(&params)
	result, err := h.service.Statistics(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// GetWorkstationDeptOptions 工位编辑"所属部门"下拉数据源(orgId 子孙 + alias union)
// @Summary 工位部门下拉选项
// @Description union 查询: orgId 子孙节点 + alias 映射节点 (含 isAlias 标记)
// @Tags 运维管理
// @Accept json
// @Produce json
// @Param request body object{orgId=string} true "orgId"
// @Success 200 {object} response.Response{data=[]opsServices.DeptOption}
// @Failure 500 {object} response.Response
// @Router /ops/workstation/dept-options [post]
func (h *WorkstationHandler) GetWorkstationDeptOptions(c *gin.Context) {
	var req struct {
		OrgID string `json:"orgId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.OrgID = ""
	}
	result, err := h.service.GetWorkstationDeptOptions(c.Request.Context(), req.OrgID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// SearchWorkstationOptions 工位下拉数据源(name LIKE 模糊 + floorId/floorCode/status/type/orgId 筛选,LIMIT 50,读操作不写操作日志)
// 修复 info-points/index.tsx「所属工位」下拉用 pageSize:1000 + filterOption 客户端截断的 bug。
func (h *WorkstationHandler) SearchWorkstationOptions(c *gin.Context) {
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		params = map[string]interface{}{}
	}
	result, err := h.service.SearchWorkstationOptions(c.Request.Context(), params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
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
	ctx := c.Request.Context()
	id := c.Param("id")
	workstation, err := h.service.GetByID(ctx, id)
	if err != nil {
		response.Error(c, apperrors.WorkstationNotFound())
		return
	}

	// R4 (Phase 45) — 跨模块注入对账健康度 (D-A1-01/03 + cross-module-permission.md §2.3)
	//
	// 行为:
	//  - hasReconciliationPerm=true:调 reconciliationSvc.GetByWorkstation,填 Reconciliation/Visible
	//  - hasReconciliationPerm=false:静默降级(ReconciliationVisible=false + HiddenReason 标记)
	//  - reconciliationSvc 注入失败或 GetByWorkstation 报错:可见=true 但 Reconciliation=nil
	//    (前端 HealthCard 渲染错误态,不抛 500 阻塞工位主信息返回)
	//
	// 类型转换:Workstation.Reconciliation 是 map[string]interface{} 弱类型(避免 models→asset
	// 循环依赖),从 *asset.ByWorkstationResponse 经 json.Marshal/Unmarshal 转换。
	visible := h.hasReconciliationPerm(c)
	workstation.ReconciliationVisible = visible
	if visible {
		if h.reconciliationSvc != nil {
			if recon, rErr := h.reconciliationSvc.GetByWorkstation(ctx, id, "7d"); rErr == nil {
				if recon != nil {
					if data, mErr := json.Marshal(recon); mErr == nil {
						var m map[string]interface{}
						if uErr := json.Unmarshal(data, &m); uErr == nil {
							workstation.Reconciliation = m
						}
					}
				}
			} else {
				applogger.Warnf("[workstation:reconciliation] GetByWorkstation 失败 workstationID=%s: %v", id, rErr)
			}
		}
	} else {
		workstation.ReconciliationHiddenReason = "无资产对账查看权限"
	}

	response.Success(c, workstation)
}

// hasReconciliationPerm 内部权限检查 (per cross-module-permission.md §2.3 + D-A1-03)
//
// 复用 pkg/middleware.HasUserPermission 复用底层 checkUserPermission / isSuperAdmin 链路。
// 静默降级:不调 c.Abort(),调用方决定如何处理 false。
func (h *WorkstationHandler) hasReconciliationPerm(c *gin.Context) bool {
	return middleware.HasUserPermission(c, h.core, "asset:reconciliation:list")
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
