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

// ServeFrontend 提供前端静态文件服务（SPA fallback）
func ServeFrontend(c *gin.Context) {
	requestPath := c.Request.URL.Path

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
	requestPath := c.Request.URL.Path

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
