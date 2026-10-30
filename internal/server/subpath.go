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

// stripSubPath 剥离 frontendSubPath 前缀后返回剩余路径;
// 不匹配时原样返回。剥离后空字符串视为 "/"。
func stripSubPath(p string) string {
	p = strings.TrimPrefix(p, frontendSubPath)
	if p == "" {
		p = "/"
	}
	return p
}

// subPathBackendPrefixes 是允许剥前缀的后端路由前缀白名单。
// 只有后端真实注册了路由的前缀才重写:
//   - /api     → cmd/main.go 的 engine.Group("/api/v1")
//   - /uploads → cmd/main.go 的 engine.Static("/uploads", "./uploads")
//
// /assets 刻意不在白名单:静态资源由 ServeFrontend/ServeSPA 内部的
// stripSubPath 处理(embed 变体剥前缀后按扩展名读嵌入 FS),重复重写
// 没有收益、只会多一处行为面。
var subPathBackendPrefixes = []string{"/api", "/uploads"}

// stripBackendSubPath 判断 p 是否是"带 frontendSubPath 前缀的后端路由路径"。
// 命中白名单时返回剥掉前缀的路径与 true;否则返回原路径与 false。
//
// 判定规则(防误剥):
//   - 必须带尾斜杠:/xingran/api/... 命中,/xingranfoo/... 不命中
//   - 剥掉前缀后必须正好是白名单前缀,或以其 + "/" 开头:
//     /api 命中,/apixyz 不命中
func stripBackendSubPath(p string) (string, bool) {
	if !strings.HasPrefix(p, frontendSubPath+"/") {
		return p, false
	}
	rest := strings.TrimPrefix(p, frontendSubPath)
	for _, pre := range subPathBackendPrefixes {
		if rest == pre || strings.HasPrefix(rest, pre+"/") {
			return rest, true
		}
	}
	return p, false
}

// StripSubPathMiddleware 剥掉后端路由路径上的 frontendSubPath 前缀。
//
// ⚠ gin v1.10 在执行中间件之前就完成了路由匹配(gin.Engine.handleHTTPRequest
// 先用 c.Request.URL.Path 查路由树,再执行 handlers 链),所以中间件里的
// 改写只影响 handler 看到的路径,不会让 /xingran/api/... 命中
// engine.Group("/api/v1")。让带前缀的 API 请求真正落到 API 组的是
// StripSubPathHandler —— 它在进入 gin 之前就改写了路径。
// 本中间件保证任何直接使用 gin engine 的场景下,handler 看到的也是
// 剥掉前缀的路径(与 StripSubPathHandler 共用同一套判定规则)。
func StripSubPathMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rest, ok := stripBackendSubPath(c.Request.URL.Path); ok {
			c.Request.URL.Path = rest
			c.Request.URL.RawPath = ""
		}
		c.Next()
	}
}

// StripSubPathHandler 在进入 gin 路由匹配之前剥掉后端路由路径上的
// frontendSubPath 前缀。
//
// 直连 :9000 时浏览器按 Vite base 发 /xingran/api/v1/... 到后端;若不在
// 这里剥前缀,gin 的路由树里只有 /api/v1/... ,请求会落到 NoRoute 并被
// ServeSPA 的 API 排除列表挡成 404。/xingran/uploads/... 同理 ——
// engine.Static 用请求路径做前缀裁剪,带 /xingran 会取不到文件。
//
// nginx 剥前缀部署时后端收到的路径本来就不含前缀,本包装器是幂等的。
func StripSubPathHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rest, ok := stripBackendSubPath(r.URL.Path); ok {
			r.URL.Path = rest
			r.URL.RawPath = ""
		}
		next.ServeHTTP(w, r)
	})
}

// rootRedirectPath 返回根路径请求应该重定向到的位置;空串表示不重定向。
//
// X-Forwarded-Prefix 是"请求已经处在子路径下、nginx 已剥过前缀"的信号:
// nginx 部署时浏览器访问 /xingran/ → nginx 剥前缀 → 后端收到 "/",此时若
// 还 302 到 /xingran/,浏览器会在 "/" 与 "/xingran/" 之间无限跳转。
// 直连 :9000 时没有这个头 → 才重定向到 /xingran/,让浏览器落到与前端
// Router basename 匹配的地址上。
func rootRedirectPath(c *gin.Context) string {
	if c.GetHeader("X-Forwarded-Prefix") != "" {
		return ""
	}
	return frontendSubPath + "/"
}
