package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// APIKeyWriteRouteGroups API密钥**写操作**路由分组（85-03 RBAC-01）。
// toggle 为状态变更=编辑语义。
type APIKeyWriteRouteGroups struct {
	Add    *gin.RouterGroup
	Edit   *gin.RouterGroup
	Remove *gin.RouterGroup
}

// SetupAPIKeyRouter 设置API密钥**读**路由（写操作见 SetupAPIKeyWriteRouter）。
func SetupAPIKeyRouter(r *gin.RouterGroup, coreCore *core.Core) {
	apikeyHandler := newAPIKeyHandler(coreCore)

	// 注册读路由
	r.POST("/list", apikeyHandler.List)
	r.POST("/:id", apikeyHandler.GetByID)
	r.POST("/:id/logs", apikeyHandler.ListUsageLogs)
	r.GET("/:id/summary", apikeyHandler.GetUsageSummary)
}

// SetupAPIKeyWriteRouter 设置API密钥**写**路由（85-03 RBAC-01 逐动作拆分）。
func SetupAPIKeyWriteRouter(groups APIKeyWriteRouteGroups, coreCore *core.Core) {
	apikeyHandler := newAPIKeyHandler(coreCore)

	groups.Add.POST("", apikeyHandler.Create)
	groups.Edit.POST("/:id/update", apikeyHandler.Update)
	groups.Edit.POST("/:id/toggle", apikeyHandler.ToggleStatus)
	groups.Remove.POST("/:id/delete", apikeyHandler.Delete)
}

// newAPIKeyHandler 构造API密钥处理器（读/写路由共用）。
func newAPIKeyHandler(coreCore *core.Core) *APIKeyHandler {
	// 创建API密钥服务实例
	apiKeyService := systemServices.NewAPIKeyService(coreCore.GetDB())

	// 创建API密钥处理器实例
	return NewAPIKeyHandler(apiKeyService).WithCore(coreCore)
}
