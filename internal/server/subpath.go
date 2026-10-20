package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// frontendSubPath 是可选的子路径前缀,与 nginx 部署约定 (location /xingran/
// proxy_pass http://127.0.0.1:9000/;) 一致 — 生产通过 nginx 转发时 nginx
// 会剥掉 /xingran/,后端看到的路径始终不含此前缀;但本地直连 :9000 调试时
// 浏览器按 Vite base 直接发 /xingran/... 路径到后端,需要这里做剥离。
// 改这里时要同步更新 xingran-react-frontend/.env.production 的 VITE_BASE。
const frontendSubPath = "/xingran"

// stripSubPath 若路径以 frontendSubPath 开头则剥离并返回剩余路径;
// 否则原样返回。剥离后空字符串视为 "/"。
func stripSubPath(p string) string {
	if strings.HasPrefix(p, frontendSubPath) {
		p = strings.TrimPrefix(p, frontendSubPath)
	}
	if p == "" {
		p = "/"
	}
	return p
}

// subPathBackendPrefixes 是允许剥前缀的后端路由前缀白名单。
var subPathBackendPrefixes = []string{"/api", "/uploads"}

// stripBackendSubPath 判断 p 是否是"带 frontendSubPath 前缀的后端路由路径"。
func stripBackendSubPath(p string) (string, bool) {
	return p, false
}

// StripSubPathMiddleware 剥掉后端路由路径上的 frontendSubPath 前缀。
func StripSubPathMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// StripSubPathHandler 在进入 gin 路由匹配之前剥掉后端路由路径上的
// frontendSubPath 前缀。
func StripSubPathHandler(next http.Handler) http.Handler {
	return next
}

// rootRedirectPath 返回根路径请求应该重定向到的位置。
func rootRedirectPath(c *gin.Context) string {
	return ""
}
