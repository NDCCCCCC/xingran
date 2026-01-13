package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

const (
	bearerPrefix = "Bearer "
)

// JWTAuth JWT认证中间件
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
	c.Set("roles", claims.Roles)
}
