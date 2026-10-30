package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// SetupNotificationConfigRouter 设置通知配置路由
func SetupNotificationConfigRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建服务实例
	emailConfigService := systemServices.NewEmailConfigService(core.DB.GetDB())
	apiNotificationService := systemServices.NewAPINotificationConfigService(core.DB.GetDB())
	emailSenderService := services.NewEmailSenderService(core.DB.GetDB())

	// 创建Handler
	handler := NewNotificationConfigHandler(emailConfigService, apiNotificationService, emailSenderService).WithCore(core)

	// 邮箱配置路由
	emailConfigs := r.Group("/email-configs")
	{
		emailConfigs.POST("/list", handler.ListEmailConfigs)
		emailConfigs.GET("/:id", handler.GetEmailConfig)
		emailConfigs.POST("", handler.CreateEmailConfig)
		emailConfigs.PUT("/:id", handler.UpdateEmailConfig)
		emailConfigs.DELETE("/:id", handler.DeleteEmailConfig)
		emailConfigs.POST("/:id/test", handler.TestEmailConfig)
	}

	// API通知配置路由
	apiConfigs := r.Group("/api-notification-configs")
	{
		apiConfigs.POST("/list", handler.ListAPINotificationConfigs)
		apiConfigs.GET("/:id", handler.GetAPINotificationConfig)
		apiConfigs.POST("", handler.CreateAPINotificationConfig)
		apiConfigs.PUT("/:id", handler.UpdateAPINotificationConfig)
		apiConfigs.DELETE("/:id", handler.DeleteAPINotificationConfig)
	}
}
