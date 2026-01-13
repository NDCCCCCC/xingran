package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/middleware"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/permission"
)

// SetupNotificationConfigRouter 设置通知配置路由
// @title 通知配置管理
// @version 1.0
// @description 系统设置-通知配置管理API
func SetupNotificationConfigRouter(r *gin.RouterGroup, core *core.Core) {
	// 需要系统管理员权限
	r.Use(middleware.JWTAuth(core.JWTManager))
	r.Use(middleware.RequirePermissions([]string{
		string(permission.SystemConfig),
	}, core))

	// 邮箱配置路由
	emailConfigs := r.Group("/email-configs")
	{
		emailConfigs.POST("/list", listEmailConfigs(core))
		emailConfigs.GET("/:id", getEmailConfig(core))
		emailConfigs.POST("", createEmailConfig(core))
		emailConfigs.PUT("/:id", updateEmailConfig(core))
		emailConfigs.DELETE("/:id", deleteEmailConfig(core))
		emailConfigs.POST("/:id/test", testEmailConfig(core))
	}

	// API通知配置路由
	apiConfigs := r.Group("/api-notification-configs")
	{
		apiConfigs.POST("/list", listAPINotificationConfigs(core))
		apiConfigs.GET("/:id", getAPINotificationConfig(core))
		apiConfigs.POST("", createAPINotificationConfig(core))
		apiConfigs.PUT("/:id", updateAPINotificationConfig(core))
		apiConfigs.DELETE("/:id", deleteAPINotificationConfig(core))
	}
}
