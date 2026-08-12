package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// Recovery 返回恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer handlePanic(c)
		c.Next()
	}
}

// handlePanic 处理panic
func handlePanic(c *gin.Context) {
	if err := recover(); err != nil {
		logPanic(err)
		response.Error(c, response.ErrServerError)
		c.Abort()
	}
}

// logPanic 记录panic信息和堆栈
func logPanic(err interface{}) {
	logger.Errorf("Panic recovered: %v\n%s", err, debug.Stack())
	fmt.Printf("Panic recovered: %v\n%s", err, debug.Stack())
}
