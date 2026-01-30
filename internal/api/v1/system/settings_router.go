package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupSettingsRouter 设置系统设置路由
func SetupSettingsRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建系统设置服务
	settingsService := systemServices.NewSettingsService(core.DB.GetDB())
	settingsHandler := NewSettingsHandler(settingsService)

	// 系统设置路由（所有已登录用户可访问）
	r.GET("/preferences", settingsHandler.GetUserPreferences)
	r.PUT("/preferences", settingsHandler.UpdateUserPreferences)
}
