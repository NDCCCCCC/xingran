package middleware

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// Gzip 返回Gzip压缩中间件
func Gzip() gin.HandlerFunc {
	return gzip.Gzip(gzip.DefaultCompression)
}
