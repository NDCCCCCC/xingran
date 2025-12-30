package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
)

// JWTAuth JWT认证中间件
func JWTAuth(jwtManager *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// 首先尝试从 Authorization 头获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// 检查Bearer前缀
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				response.Error(c, response.ErrUnauthorized, "认证令牌格式错误")
				c.Abort()
				return
			}
			token = authHeader[len(bearerPrefix):]
		} else {
			// WebSocket 连接可能通过 query 参数传递 token
			token = c.Query("token")
		}

		if token == "" {
			response.Error(c, response.ErrUnauthorized, "缺少认证令牌")
			c.Abort()
			return
		}

		// 验证令牌
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}
