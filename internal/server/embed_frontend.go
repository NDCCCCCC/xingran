//go:build !embed
// +build !embed

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ServeFrontend 开发模式：返回错误提示
// 在开发模式下，前端应该由 Vite 开发服务器（端口 4000）提供
func ServeFrontend(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":   "Frontend not available in development mode",
		"message": "Please run the frontend dev server: cd xingran-react-frontend && npm run dev",
	})
}

// ServeSPA 开发模式：返回错误提示
// 在开发模式下，SPA fallback 由 Vite 开发服务器处理
func ServeSPA(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error":   "Not found (development mode)",
		"message": "SPA routes are handled by Vite dev server in development mode",
	})
}
