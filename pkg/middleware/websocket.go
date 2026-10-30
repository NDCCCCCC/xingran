package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
)

// WebSocketAuth WebSocket认证中间件
func WebSocketAuth(jwtManager interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从query获取token
		token := c.Query("token")
		if token == "" {
			// 尝试从Authorization header获取
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token = parts[1]
				}
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少token"})
			c.Abort()
			return
		}

		// 验证token
		core, ok := jwtManager.(*core.Core)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器配置错误"})
			c.Abort()
			return
		}

		claims, err := core.JWTManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效token"})
			c.Abort()
			return
		}

		// 设置用户信息到context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
