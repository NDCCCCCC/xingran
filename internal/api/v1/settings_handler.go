package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// SetupSettingsRouter 设置系统设置路由
func SetupSettingsRouter(r *gin.RouterGroup, core *core.Core) {
	// 获取用户个人设置
	r.GET("/preferences", getUserPreferences(core))
	// 更新用户个人设置
	r.PUT("/preferences", updateUserPreferences(core))
}

// getUserPreferences 获取用户个人设置
func getUserPreferences(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		settingsService := services.NewSettingsService(core.DB.GetDB())
		preferences, err := settingsService.GetUserPreferences(c.Request.Context(), userID.(string))
		if err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, preferences)
	}
}

// updateUserPreferences 更新用户个人设置
func updateUserPreferences(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			response.Error(c, response.ErrUnauthorized)
			return
		}

		var req services.UserPreferences
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		settingsService := services.NewSettingsService(core.DB.GetDB())
		if err := settingsService.UpdateUserPreferences(c.Request.Context(), userID.(string), &req); err != nil {
			response.Error(c, response.ErrServerError, err.Error())
			return
		}

		response.Success(c, gin.H{"message": "设置保存成功"})
	}
}
