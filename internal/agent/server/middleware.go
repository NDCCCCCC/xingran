package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 错误消息常量
const (
	errMissingAuthHeader = "missing authorization header"
	errInvalidAuthFormat = "invalid authorization header format"
)

// JWTAuth JWT 认证中间件
func JWTAuth(authenticator *JWTAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errMissingAuthHeader})
			return
		}

		// 检查 Bearer 前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidAuthFormat})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := authenticator.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": errInvalidToken})
			return
		}

		// 将 claims 存储到上下文
		c.Set("agent_id", claims.AgentID)
		c.Set("vm_id", claims.VMID)
		c.Next()
	}
}

// CORSMiddleware CORS 中间件（生产环境应配置具体域名）
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// OPTIONS 预检请求直接返回
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		c.Next()
	}
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware() gin.HandlerFunc {
	return gin.Logger()
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.Recovery()
}

// SecurityHeaders 添加安全响应头中间件
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy - 限制资源来源
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; object-src 'none'")

		// X-Frame-Options - 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// X-Content-Type-Options - 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// X-XSS-Protection - 启用 XSS 过滤
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict-Transport-Security - 强制 HTTPS (如果启用了 TLS)
		if c.Request.TLS != nil || c.Request.URL.Scheme == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer-Policy - 控制 Referer 信息泄露
		c.Header("Referrer-Policy", "no-referrer")

		// Permissions-Policy - 限制浏览器功能
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
