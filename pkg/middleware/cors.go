package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	// maxAge CORS预检请求缓存时间
	maxAge = 12 * time.Hour
)

// Cors 返回CORS中间件
// allowedOrigins: 允许的域名列表，如 ["http://localhost:3000", "https://example.com"]
//
//	空列表或包含 "*" 则允许所有来源（仅用于开发环境）
func Cors(allowedOrigins []string) gin.HandlerFunc {
	// 如果没有指定允许的域名或者包含通配符，则允许所有来源（仅开发环境）
	allowAll := len(allowedOrigins) == 0 || contains(allowedOrigins, "*")

	config := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Request-ID", "X-Request-Encrypted"},
		ExposeHeaders:    []string{"Content-Length", "Authorization"},
		AllowCredentials: !allowAll, // 如果允许所有来源，则不能携带凭证
		MaxAge:           maxAge,
	}

	// 如果允许所有来源，需要特殊处理
	if allowAll {
		config.AllowOrigins = nil
		config.AllowOriginFunc = func(origin string) bool {
			return true
		}
	}

	return cors.New(config)
}

// contains 检查字符串切片是否包含某个元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// CorsByPattern 根据模式匹配CORS（支持通配符域名）
// 例如: "*.example.com" 允许所有 example.com 的子域名
func CorsByPattern(allowedPatterns []string) gin.HandlerFunc {
	config := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Request-ID", "X-Request-Encrypted"},
		ExposeHeaders:    []string{"Content-Length", "Authorization"},
		AllowCredentials: true,
		MaxAge:           maxAge,
		AllowOriginFunc: func(origin string) bool {
			for _, pattern := range allowedPatterns {
				if matchDomainPattern(origin, pattern) {
					return true
				}
			}
			return false
		},
	}

	return cors.New(config)
}

// matchDomainPattern 匹配域名模式
// 支持: "*.example.com" (匹配所有子域名), "example.com" (精确匹配)
func matchDomainPattern(origin, pattern string) bool {
	// 移除协议部分
	origin = strings.TrimPrefix(origin, "http://")
	origin = strings.TrimPrefix(origin, "https://")
	origin = strings.TrimPrefix(origin, "www.")
	// 移除端口号
	if idx := strings.Index(origin, ":"); idx != -1 {
		origin = origin[:idx]
	}

	// 精确匹配
	if origin == pattern || "www."+origin == pattern {
		return true
	}

	// 通配符匹配
	if strings.HasPrefix(pattern, "*.") {
		domain := strings.TrimPrefix(pattern, "*.")
		return origin == domain || strings.HasSuffix(origin, "."+domain)
	}

	return false
}
