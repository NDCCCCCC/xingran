//go:build embed
// +build embed

package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

//go:embed all:xingran-react-frontend/dist
var frontendFS embed.FS

var (
	detectedBaseOnce sync.Once
	detectedBase     string
)

// detectedFrontendBase 从嵌入 dist 的 index.html 自动探测前端构建 base。
//
// Vite 以 VITE_BASE 构建时,资源引用形如 src="/xingran/assets/xxx.js";
// 根路径构建时为 src="/assets/xxx.js"。取首个 "/...assets/" 引用的前缀:
//   - "/xingran/assets/" → 返回 "/xingran"(子路径构建)
//   - "/assets/"         → 返回 ""(根路径构建)
//
// 两处重定向行为(根路径 302、陈旧无前缀 SPA URL 补前缀)都以它为开关:
// 根路径构建的 dist 没有 basename,任何重定向到 /xingran 的行为都会把
// 浏览器送进不存在的地址 —— 必须全部停用。
func detectedFrontendBase() string {
	detectedBaseOnce.Do(func() {
		content, err := fs.ReadFile(frontendFS, "xingran-react-frontend/dist/index.html")
		if err != nil {
			return
		}
		html := string(content)
		for _, marker := range []string{"\"/" + strings.TrimPrefix(frontendSubPath, "/") + "/assets/", "\"/assets/"} {
			idx := strings.Index(html, marker)
			if idx < 0 {
				continue
			}
			rest := html[idx+1:]
			if strings.HasPrefix(rest, strings.TrimPrefix(frontendSubPath, "/")+"/assets/") {
				detectedBase = frontendSubPath
			} else {
				detectedBase = ""
			}
			return
		}
	})
	return detectedBase
}

// ServeRoot 处理根路径请求。
//
// 直连 :9000(没有 nginx 剥前缀)时,浏览器访问 "/" 与前端 Router 的
// basename "/xingran" 不匹配 → 白屏。这里 302 到 /xingran/,让浏览器落到
// basename 能匹配的地址上。
//
// nginx 剥前缀部署时,后端收到的 "/" 来自 /xingran/ 且请求带
// X-Forwarded-Prefix 头,此时直接返回 index.html,否则浏览器会在
// "/" 与 "/xingran/" 之间无限跳转(见 rootRedirectPath)。
func ServeRoot(c *gin.Context) {
	// 根路径构建的 dist(base="")没有子路径 basename,直接回 index.html;
	// 仅子路径构建(base="/xingran")才需要 302 引导浏览器落进 basename。
	if detectedFrontendBase() != "" {
		if target := rootRedirectPath(c); target != "" {
			c.Redirect(http.StatusFound, target)
			return
		}
	}
	serveFile(c, "index.html")
}

// ServeFrontend 提供前端静态文件服务（SPA fallback）
func ServeFrontend(c *gin.Context) {
	requestPath := stripSubPath(c.Request.URL.Path)

	// 根路径返回 index.html
	if requestPath == "/" {
		serveFile(c, "index.html")
		return
	}

	// 去掉前导 /
	embedPath := strings.TrimPrefix(requestPath, "/")

	// 检查是否是静态资源（有扩展名）
	ext := strings.ToLower(filepath.Ext(embedPath))
	if ext != "" {
		// 尝试读取静态资源
		content, err := fs.ReadFile(frontendFS, "xingran-react-frontend/dist/"+embedPath)
		if err == nil {
			serveContent(c, embedPath, content)
			return
		}
		// 静态资源不存在，返回 404
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}

	// 所有其他路径（SPA 路由）返回 index.html
	serveFile(c, "index.html")
}

// spaNoRedirectPrefixes 是不允许触发「补前缀重定向」的后端路径前缀。
// 这些路径有专属路由注册(或必须保持 404 语义),落到 NoRoute 的带前缀变体
// 已被 StripSubPathHandler 重写成无前缀形态 —— 无前缀又命中它们的请求是
// 真正的错误路径,重定向只会掩盖问题。
var spaNoRedirectPrefixes = []string{"/api", "/uploads", "/swagger", "/debug", "/assets"}

// ServeSPA 为 SPA 客户端路由提供 index.html fallback
// 用于处理 NoRoute 请求 — 仅对 GET 请求和非 API 路径生效
func ServeSPA(c *gin.Context) {
	requestPath := c.Request.URL.Path

	// 只处理 GET 请求
	if c.Request.Method != http.MethodGet {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// 陈旧无前缀 SPA URL 兜底:浏览器停在 basename 之外的地址(书签、
	// window.location.href = "/login" 这类根绝对路径跳转)时,直接回
	// index.html 会被前端 Router 以 basename 不匹配拒绝渲染(白屏,
	// 控制台报 "<Router basename=/xingran/> is not able to match")。
	// 这里 302 补上 frontendSubPath,让地址落回 basename 之下。
	//   - 根路径构建的 dist(base="")→ 无 basename 概念,整体停用
	//   - 带 X-Forwarded-Prefix → nginx 已剥前缀,请求本就处于子路径下,豁免
	//   - 带扩展名 → 陈旧 hash 资源,404 比循环跳转更好
	//   - 后端自有前缀(/api /uploads /swagger /debug /assets)→ 保持 404
	if detectedFrontendBase() != "" &&
		!strings.HasPrefix(requestPath, frontendSubPath+"/") && requestPath != frontendSubPath &&
		c.GetHeader("X-Forwarded-Prefix") == "" && filepath.Ext(requestPath) == "" {
		redirect := true
		for _, pre := range spaNoRedirectPrefixes {
			if strings.HasPrefix(requestPath, pre+"/") {
				redirect = false
				break
			}
		}
		if redirect {
			c.Redirect(http.StatusFound, frontendSubPath+requestPath)
			return
		}
	}

	requestPath = stripSubPath(requestPath)

	// 只处理 GET 请求
	if c.Request.Method != http.MethodGet {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// 跳过 API 路径和静态资源路径
	if strings.HasPrefix(requestPath, "/api/") ||
		strings.HasPrefix(requestPath, "/uploads/") ||
		strings.HasPrefix(requestPath, "/swagger/") ||
		strings.HasPrefix(requestPath, "/debug/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}

	// 对于有扩展名的文件请求，检查是否存在
	ext := strings.ToLower(filepath.Ext(requestPath))
	if ext != "" {
		embedPath := strings.TrimPrefix(requestPath, "/")
		content, err := fs.ReadFile(frontendFS, "xingran-react-frontend/dist/"+embedPath)
		if err == nil {
			serveContent(c, embedPath, content)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// SPA 路由：返回 index.html
	serveFile(c, "index.html")
}

func serveFile(c *gin.Context, filename string) {
	content, err := fs.ReadFile(frontendFS, "xingran-react-frontend/dist/"+filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
}

func serveContent(c *gin.Context, filename string, content []byte) {
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := "text/plain"

	switch ext {
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".html":
		contentType = "text/html; charset=utf-8"
	case ".svg":
		contentType = "image/svg+xml"
	default:
		contentType = http.DetectContentType(content)
	}

	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, content)
}
