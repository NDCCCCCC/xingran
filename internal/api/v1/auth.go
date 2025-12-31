package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// EncryptedPassword 表示密码是否已加密（SM2）
	EncryptedPassword bool `json:"encryptedPassword,omitempty"`
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
	// GET /api/v1/system/auth/public-key - 获取 SM2 公钥（用于密码加密）
	r.GET("/public-key", getPublicKey(core))

	// POST /api/v1/system/auth/login - 用户登录
	r.POST("/login", login(core))

	// POST /api/v1/system/auth/logout - 用户登出
	r.POST("/logout", logout(core))

	// POST /api/v1/system/auth/refresh - 刷新令牌
	r.POST("/refresh", refreshToken(core))

	// 设置验证码路由
	SetupCaptchaRouter(r, core)
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

		// 检查验证码（如果启用）
		// 添加调试日志
		config := core.CaptchaService.GetConfig()
		fmt.Printf("[DEBUG] Captcha Enabled: %s (IsEnabled: %v)\n", config.Enabled, core.CaptchaService.IsEnabled())

		if core.CaptchaService.IsEnabled() {
			// 获取客户端IP
			clientIP := c.ClientIP()

			// 验证验证码
			if req.Captcha == "" || req.CaptchaID == "" {
				response.Error(c, response.ErrCaptchaError, "请输入验证码")
				return
			}

			if err := core.CaptchaService.VerifyCaptcha(c.Request.Context(), req.CaptchaID, req.Captcha, clientIP); err != nil {
				response.Error(c, response.ErrCaptchaError, err.Error())
				return
			}
		}

		// 检查账号是否被锁定
		if err := core.CaptchaService.CheckLoginLock(c.Request.Context(), req.Username); err != nil {
			response.Error(c, response.ErrForbidden, err.Error())
			return
		}

		// 查找用户
		var user models.User
		if err := core.DB.GetDB().Where("username = ?", req.Username).First(&user).Error; err != nil {
			if err.Error() == "record not found" {
				response.Error(c, response.ErrUserNotFound)
			} else {
				response.Error(c, response.ErrDatabaseError, "查询用户失败: "+err.Error())
			}
			return
		}

		// 检查用户状态
		if user.Status != models.UserStatusEnabled {
			response.Error(c, response.ErrForbidden, "用户已被禁用")
			return
		}

	// 解密密码（如果使用 SM2 加密）
	passwordToVerify := req.Password
	if req.EncryptedPassword {
		decrypted, err := core.JWTManager.DecryptPassword(req.Password)
		if err != nil {
			response.Error(c, response.ErrBadRequest, "密码解密失败: "+err.Error())
			return
		}
		passwordToVerify = decrypted
	}

	// 验证密码 - 使用SM3算法
	pwdManager := security.NewPasswordManager(nil)
	if ok, err := pwdManager.VerifyPassword(passwordToVerify, user.Password); err != nil || !ok {
		if err != nil {
			response.Error(c, response.ErrPasswordDecrypt, "密码验证失败: "+err.Error())
			return
		}

		// 记录登录失败
		if failErr := core.CaptchaService.RecordLoginFailure(c.Request.Context(), req.Username); failErr != nil {
			response.Error(c, response.ErrCredentialInvalid, failErr.Error())
		} else {
			response.Error(c, response.ErrCredentialInvalid)
		}
		return
	}

	// 登录成功，清除失败记录
	core.CaptchaService.ClearLoginFailure(c.Request.Context(), req.Username)

	// 查询用户角色
	var userRoles []models.UserRole
	var roleIds []string
	if err := core.DB.GetDB().Where("user_id = ?", user.ID).Find(&userRoles).Error; err == nil {
		for _, ur := range userRoles {
			roleIds = append(roleIds, ur.RoleID)
		}
	}

		// 生成JWT令牌
		tokenPair, err := core.JWTManager.GenerateTokenPair(user.ID, user.Username, roleIds)
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
			Roles:       roleIds, // 使用数据库中的角色ID
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

		// 验证刷新令牌并获取用户ID
		claims, err := core.JWTManager.ValidateToken(req.RefreshToken)
		if err != nil {
			response.Error(c, err)
			return
		}

		// 检查是否为刷新令牌
		if len(claims.Roles) != 1 || claims.Roles[0] != "refresh" {
			response.Error(c, response.ErrTokenInvalid)
			return
		}

		// 从数据库获取用户及其角色信息
		var user models.User
		if err := core.DB.GetDB().Where("id = ?", claims.UserID).First(&user).Error; err != nil {
			response.Error(c, response.ErrUserNotFound)
			return
		}

		// 查询用户角色
		var userRoles []models.UserRole
		var roleIds []string
		if err := core.DB.GetDB().Where("user_id = ?", user.ID).Find(&userRoles).Error; err == nil {
			for _, ur := range userRoles {
				roleIds = append(roleIds, ur.RoleID)
			}
		}

		// 生成新的令牌对
		tokenPair, err := core.JWTManager.GenerateTokenPair(user.ID, user.Username, roleIds)
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
			Roles:       roleIds, // 使用实际的角色ID
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

// getPublicKey 获取 SM2 公钥
// @Summary 获取 SM2 公钥
// @Description 获取 SM2 公钥用于前端加密密码
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]string}
// @Router /system/auth/public-key [get]
func getPublicKey(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		publicKey := core.JWTManager.GetPublicKey()
		if publicKey == "" {
			response.Error(c, response.ErrServerError, "SM2 未启用或公钥不可用")
			return
		}

		response.Success(c, gin.H{
			"publicKey": publicKey,
		})
	}
}
