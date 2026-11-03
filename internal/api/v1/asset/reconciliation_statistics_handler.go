package asset

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// StatisticsHandler 资产对账统计 handler
//
// R1 范围(Phase 42 R1 plan 04 / D-06 + F2 路由注册):
//   - 6 个 POST 端点,全部读操作,不调 operlog.Record
//   - 复用 ModuleReconciliation 常量(读端点仍统一在一个 module 下便于统计)
//
// days / limit 入参校验:
//   - days 范围 [1, 365](T-42-14 mitigates HealthTrend DoS)
//   - limit 范围 [1, 10000](T-42-12 mitigates TopUnresolved DoS)
//
// Pattern:对标 internal/api/v1/operations/asset_handler.go:198-207 Statistics handler
type StatisticsHandler struct {
	service asset.ReconciliationStatistics
	core    *core.Core
}

// NewStatisticsHandler 构造 StatisticsHandler
func NewStatisticsHandler(svc asset.ReconciliationStatistics) *StatisticsHandler {
	return &StatisticsHandler{service: svc}
}

// WithCore 注入 core 依赖(operlog 记录所需)。Phase 34 操作日志全模块覆盖。
// 虽然 R1 全部读端点不调 operlog,但保留 WithCore 模式以备 R2+ 写端点扩展。
func (h *StatisticsHandler) WithCore(core *core.Core) *StatisticsHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Summary 5 KPI 卡片数据源(D-06)
//
// POST /statistics/summary
// Request body:{"days": 7}  // 可选,默认 7
// @Summary 对账统计总览
// @Description 查询资产对账统计总览（默认近 7 天）
// @Tags 资产管理
// @Accept json
// @Produce json
// @Param request body object{days=int} false "统计天数"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /asset/reconciliation/statistics/summary [post]
func (h *StatisticsHandler) Summary(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// body 为空或格式错误时,用默认 days=7 兜底
		req.Days = 0
	}

	result, err := h.service.Summary(c.Request.Context(), asset.StatsFilter{Days: req.Days})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// ByConflictType 按冲突类型聚合(A-F 6 值)
//
// POST /statistics/by-conflict-type
// @Summary 按冲突类型统计
// @Description 按冲突类型分组统计资产对账异常
// @Tags 资产管理
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /asset/reconciliation/statistics/by-conflict-type [post]
func (h *StatisticsHandler) ByConflictType(c *gin.Context) {
	result, err := h.service.ByConflictType(c.Request.Context(), asset.StatsFilter{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// BySeverity 按严重级聚合(low/medium/high/critical 4 值)
//
// POST /statistics/by-severity
// @Summary 按严重级别统计
// @Description 按严重级别分组统计资产对账异常
// @Tags 资产管理
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /asset/reconciliation/statistics/by-severity [post]
func (h *StatisticsHandler) BySeverity(c *gin.Context) {
	result, err := h.service.BySeverity(c.Request.Context(), asset.StatsFilter{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// HealthTrend 按天聚合的健康度趋势
//
// POST /statistics/health-trend
// Request body:{"days": 7}  // 可选,默认 7,MaxDays=365
//
// 注意:此端点使用 PG `FILTER (WHERE ...)` 语法,SQLite 不支持,dev 必须用 PG。
// @Summary 资产健康趋势
// @Description 查询资产健康度趋势（默认近 7 天）
// @Tags 资产管理
// @Accept json
// @Produce json
// @Param request body object{days=int} false "趋势天数"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /asset/reconciliation/statistics/health-trend [post]
func (h *StatisticsHandler) HealthTrend(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Days = 0
	}

	result, err := h.service.HealthTrend(c.Request.Context(), asset.StatsFilter{Days: req.Days})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// TopUnresolved 长期未解决异常 Top N
//
// POST /statistics/top-unresolved
// Request body:{"limit": 10}  // 可选,默认 10,MaxPageSize=10000 兜底
// @Summary 未解决异常 TOP
// @Description 查询长期未解决对账异常排行
// @Tags 资产管理
// @Accept json
// @Produce json
// @Param request body object{limit=int} false "返回条数"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /asset/reconciliation/statistics/top-unresolved [post]
func (h *StatisticsHandler) TopUnresolved(c *gin.Context) {
	var req struct {
		Limit int `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Limit = 0
	}

	result, err := h.service.TopUnresolved(c.Request.Context(), req.Limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}

// ExceptionRuleStats 按例外规则聚合的命中数(R3 接入后有数据,R1 返回空)
//
// POST /statistics/exception-rule-stats
// @Summary 例外规则命中统计
// @Description 统计各例外规则命中情况
// @Tags 资产管理
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /asset/reconciliation/statistics/exception-rule-stats [post]
func (h *StatisticsHandler) ExceptionRuleStats(c *gin.Context) {
	result, err := h.service.ExceptionRuleStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, result)
}