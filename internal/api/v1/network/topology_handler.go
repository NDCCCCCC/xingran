package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/topology"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// TopologyHandler 拓扑管理处理器
type TopologyHandler struct {
	filterRuleService topology.FilterRuleService
	db                *gorm.DB
	core              *core.Core
}

// NewTopologyHandler 创建拓扑管理处理器实例
func NewTopologyHandler(filterRuleSvc topology.FilterRuleService, db *gorm.DB) *TopologyHandler {
	return &TopologyHandler{
		filterRuleService: filterRuleSvc,
		db:                db,
	}
}

// WithCore 注入 core 依赖（操作日志记录所需）。Phase 34 操作日志全模块覆盖。
func (h *TopologyHandler) WithCore(core *core.Core) *TopologyHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// CreateRule 创建MAC过滤规则
// @Summary 创建MAC过滤规则
// @Description 创建新的MAC地址过滤规则
// @Tags 拓扑管理
// @Accept json
// @Produce json
// @Param request body topology.CreateFilterRuleRequest true "规则信息"
// @Success 200 {object} response.Response{data=models.MACFilterRule}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/network/topology/rules [post]
func (h *TopologyHandler) CreateRule(c *gin.Context) {
	var req topology.CreateFilterRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	req.CreatedBy = getCreatedByFromContext(c)

	rule, err := h.filterRuleService.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络拓扑", operlog.OperTypeCreate)
	response.Success(c, rule)
}

// UpdateRule 更新MAC过滤规则
// @Summary 更新MAC过滤规则
// @Description 更新指定的MAC地址过滤规则
// @Tags 拓扑管理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Param request body topology.UpdateFilterRuleRequest true "规则信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/network/topology/rules/{id}/update [post]
func (h *TopologyHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	var req topology.UpdateFilterRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请求参数错误")
		return
	}

	req.ID = id
	req.UpdatedBy = getCreatedByFromContext(c)

	if err := h.filterRuleService.Update(c.Request.Context(), id, &req); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络拓扑", operlog.OperTypeUpdate)
	response.Success(c, nil)
}

// DeleteRule 删除MAC过滤规则
// @Summary 删除MAC过滤规则
// @Description 删除指定的MAC地址过滤规则
// @Tags 拓扑管理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/network/topology/rules/{id}/delete [post]
func (h *TopologyHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	if err := h.filterRuleService.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "网络拓扑", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// GetRule 获取MAC过滤规则详情
// @Summary 获取MAC过滤规则详情
// @Description 根据规则ID获取MAC过滤规则详细信息
// @Tags 拓扑管理
// @Accept json
// @Produce json
// @Param id path string true "规则ID"
// @Success 200 {object} response.Response{data=models.MACFilterRule}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/network/topology/rules/{id} [post]
func (h *TopologyHandler) GetRule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, http.StatusBadRequest, "规则ID不能为空")
		return
	}

	rule, err := h.filterRuleService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, rule)
}

// ListRules 查询MAC过滤规则列表
// @Summary 查询MAC过滤规则列表
// @Description 分页查询MAC过滤规则列表，支持设备类型和厂商过滤
// @Tags 拓扑管理
// @Accept json
// @Produce json
// @Param request body topology.ListFilterRulesParams true "查询条件"
// @Success 200 {object} response.Response{data=topology.PageResult}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/network/topology/rules/list [post]
func (h *TopologyHandler) ListRules(c *gin.Context) {
	var req topology.ListFilterRulesParams
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果是空请求体或解析失败，使用默认值
		req = topology.ListFilterRulesParams{
			Current:  1,
			PageSize: 10,
		}
	}

	// 设置默认分页值
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	result, err := h.filterRuleService.List(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}

// GetEffectiveRule 获取设备的有效MAC过滤规则
// @Summary 获取设备的有效MAC过滤规则
// @Description 根据设备类型和厂商获取生效的MAC过滤规则（调试用）
// @Tags 拓扑管理
// @Accept json
// @Produce json
// @Param deviceId query string true "设备ID"
// @Success 200 {object} response.Response{data=models.MACFilterRule}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/network/topology/rules/effective [get]
func (h *TopologyHandler) GetEffectiveRule(c *gin.Context) {
	deviceID := c.Query("deviceId")
	if deviceID == "" {
		response.Error(c, http.StatusBadRequest, "设备ID不能为空")
		return
	}

	// 从数据库获取设备信息
	var device models.NetworkDevice
	if err := h.db.Where("id = ?", deviceID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "设备不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, "查询设备失败")
		return
	}

	rule, err := h.filterRuleService.GetEffectiveRule(c.Request.Context(), &device)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, rule)
}

// getCreatedByFromContext 从上下文中获取当前用户名
func getCreatedByFromContext(c *gin.Context) string {
	// 从认证上下文中获取用户名（由认证中间件设置）
	if username, exists := c.Get("username"); exists {
		if usernameStr, ok := username.(string); ok {
			return usernameStr
		}
	}
	return "system"
}
