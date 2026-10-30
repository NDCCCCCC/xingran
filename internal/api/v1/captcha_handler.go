// Package v1 验证码相关API处理器
package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// SliderVerifyRequest 滑动验证码验证请求
type SliderVerifyRequest struct {
	CaptchaID string `json:"captchaId" binding:"required"`
	XPos      int    `json:"xPos" binding:"required"`
	Token     string `json:"token" binding:"required"`
}

// SetupCaptchaRouter 设置验证码路由
func SetupCaptchaRouter(r *gin.RouterGroup, core *core.Core) {
	// POST /api/v1/system/auth/captcha - 获取验证码
	r.POST("/captcha", getCaptcha(core))

	// POST /api/v1/system/auth/captcha/verify/slider - 验证滑动验证码
	r.POST("/captcha/verify/slider", verifySliderCaptcha(core))

	// POST /api/v1/system/auth/captcha/config - 获取验证码配置
	r.POST("/captcha/config", getCaptchaConfig(core))

	// POST /api/v1/system/auth/captcha/config/reload - 重新加载验证码配置
	r.POST("/captcha/config/reload", reloadCaptchaConfig(core))
}

// getCaptcha 获取验证码
// @Summary 获取验证码
// @Description 获取图形验证码（数字字母或滑动拼图）
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/auth/captcha [post]
func getCaptcha(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端IP
		clientIP := c.ClientIP()

		// 生成验证码
		captchaResp, err := core.CaptchaService.GenerateCaptcha(c.Request.Context(), clientIP)
		if err != nil {
			applogger.Warnf("获取验证码失败: %v, clientIP: %s", err, clientIP)
			response.Error(c, response.ErrBadRequest, err.Error())
			return
		}

		// 如果验证码未启用，返回空数据
		if captchaResp == nil {
			applogger.Infof("验证码未启用, clientIP: %s", clientIP)
			response.Success(c, gin.H{})
			return
		}

		response.Success(c, captchaResp)
	}
}

// verifySliderCaptcha 验证滑动验证码
// @Summary 验证滑动验证码
// @Description 验证滑动拼图验证码的滑动位置
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body SliderVerifyRequest true "滑动验证码数据"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /system/auth/captcha/verify/slider [post]
func verifySliderCaptcha(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SliderVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 验证滑动验证码
		err := core.CaptchaService.VerifySliderCaptcha(c.Request.Context(), req.CaptchaID, req.XPos, req.Token)
		if err != nil {
			response.Error(c, response.ErrCaptchaError, err.Error())
			return
		}

		response.Success(c, gin.H{
			"success": true,
			"token":   req.Token,
			"message": "验证成功",
		})
	}
}

// getCaptchaConfig 获取验证码配置
// @Summary 获取验证码配置
// @Description 获取当前验证码的配置信息
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /system/auth/captcha/config [post]
func getCaptchaConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := core.CaptchaService.GetConfig()

		response.Success(c, gin.H{
			"enabled":     string(config.Enabled),
			"type":        config.Type,
			"expireTime":  config.ExpireTime,
			"maxAttempts": config.MaxAttempts,
		})
	}
}

// reloadCaptchaConfig 重新加载验证码配置
// @Summary 重新加载验证码配置
// @Description 从数据库重新加载验证码配置，无需重启服务
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /system/auth/captcha/config/reload [post]
func reloadCaptchaConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从数据库重新加载配置
		if err := core.CaptchaService.LoadConfig(c.Request.Context()); err != nil {
			response.Error(c, response.ErrServerError, "重新加载配置失败: "+err.Error())
			return
		}

		// 获取新配置
		config := core.CaptchaService.GetConfig()

		response.Success(c, gin.H{
			"message": "配置重新加载成功",
			"config": gin.H{
				"enabled":     string(config.Enabled),
				"type":        config.Type,
				"expireTime":  config.ExpireTime,
				"maxAttempts": config.MaxAttempts,
			},
		})
	}
}
