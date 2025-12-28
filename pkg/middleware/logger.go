package middleware

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

const requestBodyKey = "request_body"

// Logger 返回日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 在请求处理之前读取请求体
		startTime := time.Now()

		// 读取请求体并存储到上下文中
		var bodyBytes []byte
		if c.Request.Body != nil && c.Request.Method != "GET" {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			// 恢复请求体以便后续处理器读取
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 处理请求
		c.Next()

		// 请求处理完成后记录日志
		latency := time.Since(startTime)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		userAgent := c.Request.UserAgent()

		// 构建日志字段
		logger := applogger.WithFields(map[string]interface{}{
			"status_code": statusCode,
			"latency":     latency.Milliseconds(),
			"client_ip":   clientIP,
			"method":      method,
			"path":        path,
			"user_agent":  userAgent,
		})

		// 添加请求ID（如果有）
		if requestID, exists := c.Get("request_id"); exists {
			logger = logger.WithField("request_id", requestID)
		}

		// 添加错误信息（如果有）
		if len(c.Errors) > 0 {
			logger = logger.WithField("error", c.Errors.String())
		}

		// 记录请求体（排除敏感信息）
		if len(bodyBytes) > 0 {
			bodyStr := string(bodyBytes)
			// 排除密码等敏感字段
			if len(bodyStr) > 1000 {
				bodyStr = bodyStr[:1000] + "..."
			}
			logger = logger.WithField("request_body", bodyStr)
		}

		// 根据状态码选择日志级别
		if statusCode >= 500 {
			logger.Error("Internal server error")
		} else if statusCode >= 400 {
			logger.Warn("Client error")
		} else {
			logger.Info("Request processed")
		}
	}
}

// RequestID 生成请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}