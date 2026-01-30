package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
)

// SetupProfileRouter 设置个人设置路由
func SetupProfileRouter(r *gin.RouterGroup, core *core.Core) {
	// 创建个人设置服务
	profileService := systemServices.NewProfileService(core.DB.GetDB())
	profileHandler := NewProfileHandler(profileService)

	// 个人设置路由
	r.GET("/info", profileHandler.GetInfo)
	r.PUT("/info", profileHandler.UpdateInfo)
	r.POST("/change-password", profileHandler.ChangePassword)
	r.POST("/avatar", profileHandler.UploadAvatar)
}
