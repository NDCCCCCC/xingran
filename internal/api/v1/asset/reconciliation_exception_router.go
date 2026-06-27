package asset

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/asset"
)

// SetupReconciliationExceptionRouter 设置资产对账例外规则路由(R1 skeleton)
//
// R1 端点:
//   POST /exception-rule/list   -> handler.ListRules
//   POST /exception-rule/:id    -> handler.GetRuleByID
//
// R3 会扩展 Create / Update / Delete / Enable / Disable handler。
// 42-06 整合到 internal/api/router.go 主路由。
func SetupReconciliationExceptionRouter(r *gin.RouterGroup, core *core.Core) {
	svc := asset.NewReconciliationExceptionService(core.DB.GetDB())
	handler := NewReconciliationExceptionHandler(svc).WithCore(core)

	r.POST("/exception-rule/list", handler.ListRules)
	r.POST("/exception-rule/:id", handler.GetRuleByID)
}
