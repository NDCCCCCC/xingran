package middleware

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/sirupsen/logrus"
)

const (
	maxBodyLogSize = 1000 // 最大请求体日志大小
)

// Logger 返回日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		bodyBytes := readRequestBody(c)

		c.Next()

		logRequest(c, startTime, bodyBytes)
	}
}

// readRequestBody 读取请求体
func readRequestBody(c *gin.Context) []byte {
	if c.Request.Body == nil || c.Request.Method == "GET" {
		return nil
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil
	}

	// 恢复请求体以便后续处理器读取
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return bodyBytes
}

// logRequest 记录请求日志
func logRequest(c *gin.Context, startTime time.Time, bodyBytes []byte) {
	latency := time.Since(startTime)
	statusCode := c.Writer.Status()

	logger := buildLogEntry(c, statusCode, latency, bodyBytes)

	logByStatus(logger, statusCode)
}

// buildLogEntry 构建日志条目
func buildLogEntry(c *gin.Context, statusCode int, latency time.Duration, bodyBytes []byte) *logrus.Entry {
	fields := logrus.Fields{
		"status_code": statusCode,
		"latency":     latency.Milliseconds(),
		"client_ip":   c.ClientIP(),
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"user_agent":  c.Request.UserAgent(),
	}

	// 添加请求ID
	if requestID, exists := c.Get("request_id"); exists {
		fields["request_id"] = requestID
	}

	// 添加错误信息
	if len(c.Errors) > 0 {
		fields["error"] = c.Errors.String()
	}

	// 添加请求体
	if len(bodyBytes) > 0 {
		fields["request_body"] = formatBody(bodyBytes)
	}

	return logger.WithFields(fields)
}

// formatBody 格式化请求体
func formatBody(bodyBytes []byte) string {
	bodyStr := string(bodyBytes)
	if len(bodyStr) > maxBodyLogSize {
		return bodyStr[:maxBodyLogSize] + "..."
	}
	return bodyStr
}

// logByStatus 根据状态码记录日志
func logByStatus(logger *logrus.Entry, statusCode int) {
	switch {
	case statusCode >= 500:
		logger.Error("Internal server error")
	case statusCode >= 400:
		logger.Warn("Client error")
	default:
		logger.Info("Request processed")
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
