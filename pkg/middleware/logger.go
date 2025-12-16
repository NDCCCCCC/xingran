package middleware

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger 返回日志中间件
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 记录请求和响应信息
		logger := logrus.WithFields(logrus.Fields{
			"status_code": param.StatusCode,
			"latency":     param.Latency,
			"client_ip":   param.ClientIP,
			"method":      param.Method,
			"path":        param.Path,
			"user_agent":  param.Request.UserAgent(),
			"error":       param.ErrorMessage,
		})

		// 如果有请求体，记录请求体（排除敏感信息）
		if param.Request.Body != nil && param.Request.Method != "GET" {
			bodyBytes, _ := io.ReadAll(param.Request.Body)
			param.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// 排除密码等敏感字段
			bodyStr := string(bodyBytes)
			if len(bodyStr) > 1000 {
				bodyStr = bodyStr[:1000] + "..."
			}
			logger = logger.WithField("request_body", bodyStr)
		}

		// 根据状态码选择日志级别
		if param.StatusCode >= 500 {
			logger.Error("Internal server error")
		} else if param.StatusCode >= 400 {
			logger.Warn("Client error")
		} else {
			logger.Info("Request processed")
		}

		return ""
	})
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