package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
)

// SetupReconciliationRouter 设置资产对账(异常列表)读端点路由
//
// R1 端点 (D-18: R1 不上"标记已解决" UI):
//   POST /exception/list   -> handler.ListExceptions
//   POST /exception/:id    -> handler.GetExceptionByID
//
// 本函数是 reconciliation 读端点入口,42-04 会再 SetupReconciliationStatisticsRouter;
// 42-06 整合到 internal/api/router.go 主路由(避免 Excel 导入路由冲突陷阱)。
func SetupReconciliationRouter(r *gin.RouterGroup, core *core.Core) {
	svc := asset.NewReconciliationService(core.DB.GetDB())
	handler := NewReconciliationHandler(svc).WithCore(core)

	r.POST("/exception/list", handler.ListExceptions)
	r.POST("/exception/:id", handler.GetExceptionByID)
}
