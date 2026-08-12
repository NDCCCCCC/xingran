package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

const (
	bearerPrefix = "Bearer "
)

// JWTAuth JWT认证中间件（不含黑名单检查）
func JWTAuth(jwtManager *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Error(c, response.ErrUnauthorized, "缺少认证令牌")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		setUserContext(c, claims)
		c.Next()
	}
}

// JWTAuthWithBlacklist JWT认证中间件（包含黑名单检查）
func JWTAuthWithBlacklist(jwtManager *security.JWTManager, blacklistSvc services.TokenBlacklistService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.Error(c, response.ErrUnauthorized, "缺少认证令牌")
			c.Abort()
			return
		}

		// 检查令牌是否在黑名单中
		if blacklisted, err := blacklistSvc.IsBlacklisted(c.Request.Context(), token); err != nil {
			response.Error(c, response.ErrServerError, "检查令牌状态失败")
			c.Abort()
			return
		} else if blacklisted {
			response.Error(c, response.ErrUnauthorized, "令牌已失效，请重新登录")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		// 将令牌和 claims 存储到上下文中，供登出使用
		c.Set("token", token)
		c.Set("claims", claims)
		setUserContext(c, claims)
		c.Next()
	}
}

// extractToken 从请求中提取令牌
func extractToken(c *gin.Context) string {
	// 首先尝试从 Authorization 头获取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		return extractBearerToken(authHeader)
	}

	// WebSocket 连接可能通过 query 参数传递 token
	return c.Query("token")
}

// extractBearerToken 从 Bearer 格式提取令牌
func extractBearerToken(authHeader string) string {
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return ""
	}
	return authHeader[len(bearerPrefix):]
}

// setUserContext 设置用户信息到上下文
func setUserContext(c *gin.Context, claims *security.CustomClaims) {
	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("nickname", claims.Nickname)
	c.Set("roles", claims.Roles)
}
