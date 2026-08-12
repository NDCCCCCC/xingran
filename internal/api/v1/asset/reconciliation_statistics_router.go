package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/asset"
)

// SetupReconciliationStatisticsRouter 设置资产对账统计端点路由
//
// R1 6 个 POST 端点(D-06 + F2 路由注册):
//
//	POST /statistics/summary              -> handler.Summary              (5 KPI 卡片)
//	POST /statistics/by-conflict-type     -> handler.ByConflictType       (A-F 饼图)
//	POST /statistics/by-severity          -> handler.BySeverity           (4 级柱状图)
//	POST /statistics/health-trend         -> handler.HealthTrend          (7d/30d 折线图,PG only)
//	POST /statistics/top-unresolved       -> handler.TopUnresolved        (TopN 列表)
//	POST /statistics/exception-rule-stats -> handler.ExceptionRuleStats   (R3 接入后)
//
// 42-06 会在 internal/api/router.go 主路由中调用本函数(避免 Excel 导入路由冲突陷阱)。
// 路由前缀约定:挂载到 /asset/reconciliation/* 下,与 SetupReconciliationRouter 平级。
func SetupReconciliationStatisticsRouter(r *gin.RouterGroup, core *core.Core) {
	svc := asset.NewReconciliationStatistics(core.DB.GetDB())
	handler := NewStatisticsHandler(svc).WithCore(core)

	r.POST("/statistics/summary", handler.Summary)
	r.POST("/statistics/by-conflict-type", handler.ByConflictType)
	r.POST("/statistics/by-severity", handler.BySeverity)
	r.POST("/statistics/health-trend", handler.HealthTrend)
	r.POST("/statistics/top-unresolved", handler.TopUnresolved)
	r.POST("/statistics/exception-rule-stats", handler.ExceptionRuleStats)
}