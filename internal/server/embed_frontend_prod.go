//go:build embed
// +build embed

package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:xingran-react-frontend/dist
var frontendFS embed.FS

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

// ServeSPA 为 SPA 客户端路由提供 index.html fallback
// 用于处理 NoRoute 请求 — 仅对 GET 请求和非 API 路径生效
func ServeSPA(c *gin.Context) {
	requestPath := stripSubPath(c.Request.URL.Path)

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
