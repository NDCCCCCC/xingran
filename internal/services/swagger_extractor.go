package services

import (
	"github.com/gin-gonic/gin"
	"strings"
)

// RouteInfo 路由信息
type RouteInfo struct {
	Path   string
	Method string
}

// SwaggerExtractor 从Gin引擎提取路由信息
type SwaggerExtractor struct {
	engine *gin.Engine
}

// NewSwaggerExtractor 创建Swagger提取器
func NewSwaggerExtractor(engine *gin.Engine) *SwaggerExtractor {
	return &SwaggerExtractor{engine: engine}
}

// ExtractRoutes 提取所有注册的路由
func (s *SwaggerExtractor) ExtractRoutes() []RouteInfo {
	routes := s.engine.Routes()

	var result []RouteInfo
	for _, route := range routes {
		// 过滤掉Swagger相关路由和中间件路由
		if s.shouldExcludeRoute(route.Path) {
			continue
		}

		result = append(result, RouteInfo{
			Path:   route.Path,
			Method: route.Method,
		})
	}

	return result
}

// shouldExcludeRoute 判断是否应该排除该路由
func (s *SwaggerExtractor) shouldExcludeRoute(path string) bool {
	// 排除路径列表
	excludePrefixes := []string{
		"/swagger",
		"/metrics",
		"/favicon.ico",
		"/assets",
	}

	for _, prefix := range excludePrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// RouteExists 检查路由是否存在
func (s *SwaggerExtractor) RouteExists(path, method string) bool {
	routes := s.ExtractRoutes()
	for _, route := range routes {
		if route.Path == path && route.Method == method {
			return true
		}
	}
	return false
}
