package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupAPIKeyRouter 设置API密钥路由
func SetupAPIKeyRouter(r *gin.RouterGroup, coreCore *core.Core) {
	// 创建API密钥服务实例
	apiKeyService := systemServices.NewAPIKeyService(coreCore.GetDB())

	// 创建API密钥处理器实例
	apikeyHandler := NewAPIKeyHandler(apiKeyService).WithCore(coreCore)

	// 注册路由
	r.POST("", apikeyHandler.Create)
	r.POST("/list", apikeyHandler.List)
	r.POST("/:id", apikeyHandler.GetByID)
	r.POST("/:id/update", apikeyHandler.Update)
	r.POST("/:id/delete", apikeyHandler.Delete)
	r.POST("/:id/toggle", apikeyHandler.ToggleStatus)
	r.POST("/:id/logs", apikeyHandler.ListUsageLogs)
	r.GET("/:id/summary", apikeyHandler.GetUsageSummary)
}
