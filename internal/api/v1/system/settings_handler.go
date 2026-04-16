package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	apperrors "github.com/ruoyi-next/ruoyi-go-backend/pkg/errors"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// SettingsHandler 系统设置处理器
type SettingsHandler struct {
	service systemServices.SettingsService
}

// NewSettingsHandler 创建系统设置处理器实例
func NewSettingsHandler(service systemServices.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

// GetUserPreferences 获取用户个人设置
// @Summary 获取用户个人设置
// @Description 获取当前登录用户的个人设置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=object{theme=string,themeStyle=string,layoutType=string,layoutDensity=string,sidebarWidth=int,sidebarCollapsedWidth=int,sidebarCollapsed=bool,pageSize=int,customPrimaryColor=string,customSidebarColor=string,language=string}}
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/settings/preferences [get]
func (h *SettingsHandler) GetUserPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.ErrUnauthorized)
		return
	}

	preferences, err := h.service.GetUserPreferences(c.Request.Context(), userID.(string))
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, preferences)
}

// UpdateUserPreferences 更新用户个人设置
// @Summary 更新用户个人设置
// @Description 更新当前登录用户的个人设置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body object{theme=string,themeStyle=string,layoutType=string,layoutDensity=string,sidebarWidth=int,sidebarCollapsedWidth=int,sidebarCollapsed=bool,pageSize=int,customPrimaryColor=string,customSidebarColor=string,language=string} true "用户设置"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/settings/preferences [put]
func (h *SettingsHandler) UpdateUserPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.ErrUnauthorized)
		return
	}

	var req systemServices.UserPreferences
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.UpdateUserPreferences(c.Request.Context(), userID.(string), &req); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "设置保存成功"})
}
