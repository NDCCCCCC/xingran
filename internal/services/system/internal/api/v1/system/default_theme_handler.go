package system

import (
	"github.com/gin-gonic/gin"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// DefaultThemeHandler 默认主题处理器
type DefaultThemeHandler struct {
	service systemServices.DefaultThemeService
}

// NewDefaultThemeHandler 创建默认主题处理器实例
func NewDefaultThemeHandler(service systemServices.DefaultThemeService) *DefaultThemeHandler {
	return &DefaultThemeHandler{service: service}
}

// GetDefaultThemeConfig 获取默认主题配置
// @Summary 获取默认主题配置
// @Description 获取系统默认主题配置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=object{mode=string,style=string,customColors=object}}
// @Failure 500 {object} response.Response
// @Router /system/config/theme/default [get]
func (h *DefaultThemeHandler) GetDefaultThemeConfig(c *gin.Context) {
	config, err := h.service.GetDefaultThemeConfig(c.Request.Context())
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, config)
}

// SetDefaultThemeConfig 设置默认主题配置
// @Summary 设置默认主题配置
// @Description 设置系统默认主题配置（需要管理员权限）
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body object{mode=string,style=string,customColors=object} true "主题配置"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/config/theme/default [post]
func (h *DefaultThemeHandler) SetDefaultThemeConfig(c *gin.Context) {
	var req systemServices.ThemeConfiguration
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.SetDefaultThemeConfig(c.Request.Context(), &req); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "默认主题配置已更新"})
}

// SyncUserThemeToDefault 从用户配置同步到默认主题
// @Summary 从用户配置同步到默认主题
// @Description 将指定用户的主题配置同步为系统默认主题（需要管理员权限）
// @Tags 系统设置
// @Accept json
// @Produce json
// @Param request body object{user_id=string} true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/config/theme/sync [post]
func (h *DefaultThemeHandler) SyncUserThemeToDefault(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.SyncUserThemeToDefault(c.Request.Context(), req.UserID); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "已同步用户配置到默认主题"})
}
