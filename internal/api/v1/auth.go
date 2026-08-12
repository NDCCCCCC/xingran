package v1

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/middleware"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// getNicknameOrUsername 安全地获取用户昵称，如果为空则返回用户名
func getNicknameOrUsername(user *models.User) string {
	if user.Nickname != nil && *user.Nickname != "" {
		return *user.Nickname
	}
	return user.Username
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// AuthMode 认证模式：local=本地, ad=AD域控, hybrid=混合（可选，为空时使用系统默认配置）
	AuthMode string `json:"authMode,omitempty"`
	// EncryptedPassword 表示密码是否已加密（SM2）
	EncryptedPassword bool   `json:"encryptedPassword,omitempty"`
	Captcha           string `json:"captcha,omitempty"`
	CaptchaID         string `json:"captchaId,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User         *UserInfo `json:"user"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresIn    int64     `json:"expiresIn"`
	TokenType    string    `json:"tokenType"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID         string            `json:"id"`
	Username   string            `json:"username"`
	Nickname   *string           `json:"nickname"`
	Email      *string           `json:"email"`
	Gender     models.Gender     `json:"gender"`
	Status     models.UserStatus `json:"status"`
	DeptName   *string           `json:"deptName"`
	Roles      []string          `json:"roles"`
	CreateTime time.Time         `json:"createTime"`
}

// SetupAuthRouter 设置认证路由
func SetupAuthRouter(r *gin.RouterGroup, core *core.Core) {
	// GET /api/v1/system/auth/public-key - 获取 SM2 公钥（用于密码加密）
	r.GET("/public-key", getPublicKey(core))

	// GET /api/v1/system/auth/test-sm2 - 测试 SM2 加密解密（调试用）
	r.GET("/test-sm2", testSM2(core))

	// GET /api/v1/system/auth/config - 获取认证配置（公开接口，用于登录页面）
	r.GET("/config", getAuthConfig(core))

	// GET /api/v1/system/auth/encryption-config - 获取加密配置（公开接口，无需认证）
	r.GET("/encryption-config", getEncryptionConfig(core))

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

		ctx := c.Request.Context()

		// 1. 检查验证码（如果启用）
		if core.CaptchaService.IsEnabled() {
			clientIP := c.ClientIP()
			if req.Captcha == "" || req.CaptchaID == "" {
				response.Error(c, response.ErrCaptchaError, "请输入验证码")
				return
			}
			if err := core.CaptchaService.VerifyCaptcha(ctx, req.CaptchaID, req.Captcha, clientIP); err != nil {
				response.Error(c, response.ErrCaptchaError, err.Error())
				return
			}
		}

		// 2. 检查账号是否被锁定
		if err := core.CaptchaService.CheckLoginLock(ctx, req.Username); err != nil {
			response.Error(c, response.ErrForbidden, err.Error())
			return
		}

		// 3. 解密密码（如果使用 SM2 加密）
		passwordToVerify := req.Password
		if req.EncryptedPassword {
			decrypted, err := core.JWTManager.DecryptPassword(req.Password)
			if err != nil {
				response.Error(c, response.ErrBadRequest, "密码解密失败: "+err.Error())
				return
			}
			passwordToVerify = decrypted
		}

		// 4. 确定认证模式
		authMode := req.AuthMode
		if authMode == "" {
			// 从配置读取默认认证模式
			factory := core.AuthFactory
			if factory != nil {
				defaultMode, err := factory.GetDefaultAuthMode(ctx)
				if err != nil {
					applogger.Warnf("获取默认认证模式失败: %v，降级到本地认证", err)
				} else {
					authMode = defaultMode
				}
			}
			if authMode == "" {
				authMode = "local"
			}
		}

		// 5. 获取认证器并执行认证
		factory := core.AuthFactory
		if factory == nil {
			// 工厂未初始化，回退到直接本地认证（向后兼容）
			applogger.Warnf("认证策略工厂未初始化，使用直接本地认证")
			loginLocalDirect(c, core, req.Username, passwordToVerify)
			return
		}

		authenticator, err := factory.GetAuthenticator(authMode)
		if err != nil {
			applogger.Errorf("获取认证器失败(mode=%s): %v", authMode, err)
			response.Error(c, response.ErrServerError, "认证服务异常")
			return
		}

		authReq := &security.AuthRequest{
			Username: req.Username,
			Password: passwordToVerify,
			IP:       c.ClientIP(),
		}

		authResult, err := authenticator.Authenticate(ctx, authReq)
		if err != nil {
			applogger.Warnf("用户 %s 认证失败(mode=%s): %v", req.Username, authMode, err)

			// 记录登录失败
			if failErr := core.CaptchaService.RecordLoginFailure(ctx, req.Username); failErr != nil {
				applogger.Warnf("记录登录失败次数失败: %v", failErr)
			}

			// 返回认证错误
			if err == security.ErrUserNotFound || err == security.ErrInvalidCredentials {
				response.Error(c, response.ErrCredentialInvalid)
				recordLoginLog(c, core, req.Username, nil, 1, "用户名或密码错误")
			} else if err == security.ErrUserDisabled {
				response.Error(c, response.ErrForbidden, "用户已被禁用")
				recordLoginLog(c, core, req.Username, nil, 1, "用户已被禁用")
			} else if err == security.ErrADConfigNotFound || err == security.ErrADConnectionFailed {
				// AD配置/连接问题
				response.Error(c, response.ErrServerError, "认证服务暂时不可用")
				recordLoginLog(c, core, req.Username, nil, 1, "认证服务不可用")
			} else {
				response.Error(c, response.ErrCredentialInvalid)
				recordLoginLog(c, core, req.Username, nil, 1, "认证失败")
			}
			return
		}

		// 6. 认证成功 - 获取完整用户信息
		// AD 用户 bind 已通过(ErrInvalidCredentials 在 ad_authenticator.go:86 提前返回),
		// 但 authResult.User==nil 表示后续 admin 搜索/sync 失败。
		// 原因由 AuthResult.SyncErrorReason 标识(见 ad_authenticator.go 各分支及
		// security.AuthResult 定义)。如实反馈给用户,不降级到本地用户:
		//   管理员账号被锁/密码错是 AD 基础设施问题,不应让用户用陈旧本地数据登录;
		//   "认证成功但用户信息缺失" 是误导性旧文案,已替换为按原因如实提示。
		if authResult.User == nil {
			applogger.Errorf("AD 认证成功但用户信息同步失败: username=%s, syncErrorReason=%s",
				req.Username, authResult.SyncErrorReason)
			response.Error(c, response.ErrServerError, syncErrorReasonMessage(authResult.SyncErrorReason))
			return
		}

		var user models.User
		if err := core.DB.GetDB().Where("id = ?", authResult.User.ID).First(&user).Error; err != nil {
			applogger.Errorf("查询认证用户完整信息失败: %v", err)
			response.Error(c, response.ErrServerError, "获取用户信息失败")
			return
		}

		// 7. 登录成功后处理
		core.CaptchaService.ClearLoginFailure(ctx, req.Username)
		recordLoginLog(c, core, req.Username, user.Nickname, 0, "登录成功")

		// 8. 查询用户角色
		var userRoles []models.UserRole
		var roleIds []string
		if err := core.DB.GetDB().Where("user_id = ?", user.ID).Find(&userRoles).Error; err == nil {
			for _, ur := range userRoles {
				roleIds = append(roleIds, ur.RoleID)
			}
		}

		// 9. 生成JWT令牌
		tokenPair, err := core.JWTManager.GenerateTokenPair(user.ID, user.Username, getNicknameOrUsername(&user), roleIds)
		if err != nil {
			response.Error(c, response.ErrServerError, "生成令牌失败")
			return
		}

		// 10. 构建响应
		userInfo := &UserInfo{
			ID:         user.ID,
			Username:   user.Username,
			Nickname:   user.Nickname,
			Email:      user.Email,
			Gender:     user.Gender,
			Status:     user.Status,
			DeptName:   user.DeptName,
			Roles:      roleIds,
			CreateTime: user.CreatedAt,
		}

		loginResp := &LoginResponse{
			User:         userInfo,
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
			ExpiresIn:    7200,
			TokenType:    "Bearer",
		}

		response.Success(c, loginResp)
		applogger.Infof("用户 %s 登录成功，认证模式: %s，认证源: %s", user.Username, authMode, authResult.AuthSource)
	}
}

// loginLocalDirect 直接本地认证（工厂未初始化时的回退路径）
func loginLocalDirect(c *gin.Context, core *core.Core, username, password string) {
	ctx := c.Request.Context()

	var user models.User
	if err := core.DB.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		response.Error(c, response.ErrCredentialInvalid)
		return
	}

	if user.Status != models.UserStatusEnabled {
		response.Error(c, response.ErrForbidden, "用户已被禁用")
		return
	}

	pwdManager := security.NewPasswordManager(nil)
	if ok, err := pwdManager.VerifyPassword(password, user.Password); err != nil || !ok {
		core.CaptchaService.RecordLoginFailure(ctx, username)
		response.Error(c, response.ErrCredentialInvalid)
		recordLoginLog(c, core, username, user.Nickname, 1, "用户或密码错误")
		return
	}

	core.CaptchaService.ClearLoginFailure(ctx, username)
	recordLoginLog(c, core, username, user.Nickname, 0, "登录成功")

	var userRoles []models.UserRole
	var roleIds []string
	if err := core.DB.GetDB().Where("user_id = ?", user.ID).Find(&userRoles).Error; err == nil {
		for _, ur := range userRoles {
			roleIds = append(roleIds, ur.RoleID)
		}
	}

	tokenPair, err := core.JWTManager.GenerateTokenPair(user.ID, user.Username, getNicknameOrUsername(&user), roleIds)
	if err != nil {
		response.Error(c, response.ErrServerError, "生成令牌失败")
		return
	}

	userInfo := &UserInfo{
		ID: user.ID, Username: user.Username, Nickname: user.Nickname,
		Email: user.Email, Gender: user.Gender, Status: user.Status,
		DeptName: user.DeptName, Roles: roleIds, CreateTime: user.CreatedAt,
	}

	response.Success(c, &LoginResponse{
		User: userInfo, AccessToken: tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken, ExpiresIn: 7200, TokenType: "Bearer",
	})
}

// logout 用户登出
// @Summary 用户登出
// @Description 用户登出接口，将令牌加入黑名单
// @Tags 认证
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.Response
// @Router /system/auth/logout [post]
func logout(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取令牌
		token, exists := c.Get("token")
		if !exists {
			response.Success(c, gin.H{
				"message": "登出成功",
			})
			return
		}

		tokenStr, ok := token.(string)
		if !ok || tokenStr == "" {
			response.Success(c, gin.H{
				"message": "登出成功",
			})
			return
		}

		// 从上下文获取 claims（由 JWT 中间件设置）
		claimsInterface, exists := c.Get("claims")
		if !exists {
			response.Success(c, gin.H{
				"message": "登出成功",
			})
			return
		}

		// 类型断言获取 claims
		claims, ok := claimsInterface.(*security.CustomClaims)
		if !ok {
			response.Success(c, gin.H{
				"message": "登出成功",
			})
			return
		}

		// 将令牌加入黑名单
		expiry := claims.ExpiresAt.Time
		if err := core.TokenBlacklistService.AddToBlacklist(c.Request.Context(), tokenStr, expiry); err != nil {
			// 黑名单添加失败不影响登出，但记录警告日志
			applogger.Warnf("Failed to add token to blacklist: %v", err)
		}

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
		tokenPair, err := core.JWTManager.GenerateTokenPair(user.ID, user.Username, getNicknameOrUsername(&user), roleIds)
		if err != nil {
			response.Error(c, response.ErrServerError, "生成令牌失败")
			return
		}

		// 构建用户信息
		userInfo := &UserInfo{
			ID:         user.ID,
			Username:   user.Username,
			Nickname:   user.Nickname,
			Email:      user.Email,
			Gender:     user.Gender,
			Status:     user.Status,
			DeptName:   user.DeptName,
			Roles:      roleIds, // 使用实际的角色ID
			CreateTime: user.CreatedAt,
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

// getAuthConfig 获取认证配置（公开接口，无需认证）
// 返回AD认证是否启用及默认认证模式，供登录页面决定是否显示认证模式选择器
func getAuthConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		adEnabled := false
		defaultMode := "local"

		// 从sys_config读取AD认证是否启用
		var config models.Config
		if err := core.DB.GetDB().Where("config_key = ?", "sys.auth.ad.enabled").First(&config).Error; err == nil {
			adEnabled = config.ConfigValue == "true"
		}

		// 从sys_config读取默认认证模式
		if err := core.DB.GetDB().Where("config_key = ?", "sys.auth.default.mode").First(&config).Error; err == nil {
			if config.ConfigValue == "local" || config.ConfigValue == "ad" || config.ConfigValue == "hybrid" {
				defaultMode = config.ConfigValue
			}
		}

		response.Success(c, gin.H{
			"adEnabled":   adEnabled,
			"defaultMode": defaultMode,
		})
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

// getEncryptionConfig 获取加密配置（公共端点，无需认证）
// @Summary 获取加密配置
// @Description 获取当前请求体加密的开关状态（公共端点，无需认证）
// @Tags 认证
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /system/auth/encryption-config [get]
func getEncryptionConfig(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 直接读取中间件缓存（30秒TTL）
		enabled := middleware.GetEncryptionConfigFromCache()

		response.Success(c, gin.H{
			"enabled": enabled,
			"source":  "cache",
		})
	}
}

// testSM2 测试 SM2 加密解密（调试用）
func testSM2(core *core.Core) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 SM2 密钥对
		sm2PrivateKey, sm2PublicKey := core.JWTManager.GetSM2KeyPair()
		if sm2PrivateKey == nil || sm2PublicKey == nil {
			response.Error(c, response.ErrServerError, "SM2 未启用")
			return
		}

		// 测试数据：32字符十六进制字符串（模拟 SM4 密钥）
		testData := "0123456789abcdef0123456789abcdef"

		// 使用后端公钥加密
		encrypted, err := crypto.EncryptWithSM2(testData, sm2PublicKey)
		if err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("加密失败: %v", err))
			return
		}

		// 使用后端私钥解密
		decrypted, err := crypto.DecryptWithSM2(encrypted, sm2PrivateKey)
		if err != nil {
			response.Error(c, response.ErrServerError, fmt.Sprintf("解密失败: %v", err))
			return
		}

		// 验证结果
		success := decrypted == testData

		response.Success(c, gin.H{
			"testData":  testData,
			"encrypted": encrypted,
			"decrypted": decrypted,
			"success":   success,
			"message":   map[bool]string{true: "SM2 加密解密测试通过", false: "SM2 加密解密测试失败"}[success],
		})
	}
}

// recordLoginLog 记录登录日志
// nickname 可选:登录失败(用户不存在)时无 nickname;成功或本地用户密码错时从 models.User 取
func recordLoginLog(c *gin.Context, core *core.Core, username string, nickname *string, status int, msg string) {
	clientIP := utils.GetClientIP(c)
	userAgent := c.Request.UserAgent()

	browser, os := parseUserAgent(userAgent)

	loginLog := models.LoginLog{
		Username:      username,
		Nickname:      nickname,
		IPAddr:        clientIP,
		LoginLocation: nil,
		Browser:       &browser,
		OS:            &os,
		Status:        status,
		Msg:           &msg,
		LoginTime:     time.Now(),
	}

	go func() {
		if err := core.DB.GetDB().Create(&loginLog).Error; err != nil {
			applogger.Errorf("记录登录日志失败 (user: %s, ip: %s, status: %d): %v", username, clientIP, status, err)
		}
	}()
}

// parseUserAgent 解析 User-Agent 获取浏览器和操作系统信息
func parseUserAgent(userAgent string) (browser, os string) {
	if userAgent == "" {
		return "Unknown", "Unknown"
	}

	switch {
	case utils.Contains(userAgent, "Edg"):
		browser = "Edge"
	case utils.Contains(userAgent, "Chrome"):
		browser = "Chrome"
	case utils.Contains(userAgent, "Firefox"):
		browser = "Firefox"
	case utils.Contains(userAgent, "Safari") && !utils.Contains(userAgent, "Chrome"):
		browser = "Safari"
	case utils.Contains(userAgent, "Opera") || utils.Contains(userAgent, "OPR"):
		browser = "Opera"
	default:
		browser = "Other"
	}

	switch {
	case utils.Contains(userAgent, "Windows"):
		os = "Windows"
	case utils.Contains(userAgent, "Mac OS X") || utils.Contains(userAgent, "Macintosh"):
		os = "Mac OS X"
	case utils.Contains(userAgent, "Linux"):
		os = "Linux"
	case utils.Contains(userAgent, "Android"):
		os = "Android"
	case utils.Contains(userAgent, "iOS") || utils.Contains(userAgent, "iPhone") || utils.Contains(userAgent, "iPad"):
		os = "iOS"
	default:
		os = "Other"
	}

	return browser, os
}

// syncErrorReasonMessage 将 AD 用户同步失败原因(AuthResult.SyncErrorReason)
// 映射为对用户如实的中文提示。不降级、不模糊,直接告知是 AD 管理员账号问题
// 还是用户搜索/同步问题,便于用户判断找谁处理。
// 原因码定义在 ad_authenticator.go 各 User==nil 返回分支。
func syncErrorReasonMessage(reason string) string {
	switch reason {
	case "admin_dial", "admin_bind":
		// 最常见:sys_ad_config 配置的管理员账号被 AD 锁(密码错多次触发
		// Account Lockout Policy)或密码过期/错。明确告知"管理员账号"问题。
		return "AD 管理员账号配置异常（账号可能被锁定或密码错误），请联系系统管理员"
	case "user_search":
		return "AD 用户信息搜索失败，请联系系统管理员"
	case "user_sync":
		return "AD 用户同步到本地失败，请联系系统管理员"
	case "no_syncer":
		return "AD 用户同步服务未配置，请联系系统管理员"
	default:
		return "AD 用户信息同步失败，请联系系统管理员"
	}
}
