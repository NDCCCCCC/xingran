package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// DefaultThemeHandler 默认主题处理器
type DefaultThemeHandler struct {
	service systemServices.DefaultThemeService
	core    *core.Core
}

// NewDefaultThemeHandler 创建默认主题处理器实例
func NewDefaultThemeHandler(service systemServices.DefaultThemeService) *DefaultThemeHandler {
	return &DefaultThemeHandler{service: service}
}

// WithCore 注入 core 依赖（操作日志记录所需）。返回 receiver 自身以支持链式调用。
// 之所以单独提供此方法而非改写 NewDefaultThemeHandler 签名，是为了不破坏既有调用点。
func (h *DefaultThemeHandler) WithCore(core *core.Core) *DefaultThemeHandler {
	if h != nil {
		h.core = core
	}
	return h
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "默认主题", operlog.OperTypeUpdate)
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

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "默认主题", operlog.OperTypeSync,
		operlog.WithOperParam("user_id="+req.UserID))
	response.Success(c, gin.H{"message": "已同步用户配置到默认主题"})
}
