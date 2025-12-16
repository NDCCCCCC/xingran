package v1

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/db"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Captcha  string `json:"captcha,omitempty"`
	CaptchaID string `json:"captchaId,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User         *UserInfo   `json:"user"`
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	ExpiresIn    int64       `json:"expiresIn"`
	TokenType    string      `json:"tokenType"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	Nickname    *string   `json:"nickname"`
	Email       *string   `json:"email"`
	Gender      models.Gender `json:"gender"`
	Status      models.UserStatus `json:"status"`
	DeptName    *string   `json:"deptName"`
	Roles       []string  `json:"roles"`
	CreateTime  time.Time `json:"createTime"`
}

// SetupAuthRouter 设置认证路由
func SetupAuthRouter(r *gin.RouterGroup, core *core.Core) {
	// POST /api/v1/system/auth/login - 用户登录
	r.POST("/login", login(core))

	// POST /api/v1/system/auth/logout - 用户登出
	r.POST("/logout", logout(core))

	// POST /api/v1/system/auth/refresh - 刷新令牌
	r.POST("/refresh", refreshToken(core))

	// GET /api/v1/system/auth/captcha - 获取验证码
	r.GET("/captcha", getCaptcha(core))
}

// login 用户登录
// @Summary 用户登录
// @Description 用户登录接口
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录信息"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 400 {object} response.Response
// @Router /system/auth/login [post]
func login(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 查找用户
		var user models.User
		if err := core.DB.GetDB().Where("username = ?", req.Username).First(&user).Error; err != nil {
			response.Error(c, response.ErrUnauthorized, "用户名或密码错误")
			return
		}

		// 检查用户状态
		if user.Status != models.UserStatusEnabled {
			response.Error(c, response.ErrForbidden, "用户已被禁用")
			return
		}

	// 验证密码
	pwdManager := db.NewSimplePasswordManager()
	if ok, err := pwdManager.VerifyPassword(req.Password, user.Password); err != nil || !ok {
		response.Error(c, response.ErrUnauthorized, "用户名或密码错误")
		return
	}

		// 生成JWT令牌
		tokenPair, err := core.JWTManager.GenerateTokenPair(user.ID, user.Username, user.Roles)
		if err != nil {
			response.Error(c, response.ErrServerError, "生成令牌失败")
			return
		}

		// 构建用户信息
		userInfo := &UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			Nickname:    user.Nickname,
			Email:       user.Email,
			Gender:      user.Gender,
			Status:      user.Status,
			DeptName:    user.DeptName,
			Roles:       user.Roles,
			CreateTime:  user.CreatedAt,
		}

		// 构建响应
		loginResp := &LoginResponse{
			User:         userInfo,
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresIn:    7200, // 2小时
			TokenType:    "Bearer",
		}

		response.Success(c, loginResp)
	}
}

// logout 用户登出
// @Summary 用户登出
// @Description 用户登出接口
// @Tags 认证
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.Response
// @Router /system/auth/logout [post]
func logout(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现登出逻辑
		// 1. 将令牌加入黑名单
		// 2. 清理用户缓存

		response.Success(c, gin.H{
			"message": "登出成功",
		})
	}
}

// refreshToken 刷新令牌
// @Summary 刷新令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body map[string]string true "刷新令牌"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 400 {object} response.Response
// @Router /system/auth/refresh [post]
func refreshToken(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RefreshToken string `json:"refreshToken" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, response.ErrBadRequest, "请求参数错误")
			return
		}

		// 刷新令牌
		tokenPair, err := core.JWTManager.RefreshToken(req.RefreshToken)
		if err != nil {
			response.Error(c, err)
			return
		}

		response.Success(c, tokenPair)
	}
}

// getCaptcha 获取验证码
// @Summary 获取验证码
// @Description 获取图形验证码
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/auth/captcha [get]
func getCaptcha(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现验证码生成
		response.Success(c, gin.H{
			"captchaId": "temp-id",
			"captchaImg": "data:image/png;base64,temp-image",
		})
	}
}