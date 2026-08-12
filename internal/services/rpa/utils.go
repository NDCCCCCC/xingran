package rpa

import (
	"fmt"
	"strings"
	"time"
)

// FormatTimestamp 格式化时间戳为标准格式
const timestampFormat = "2006-01-02 15:04:05"

// FormatTimestamp 格式化时间为标准字符串格式
func FormatTimestamp(t time.Time) string {
	return t.Format(timestampFormat)
}

// FormatLog 格式化日志条目，添加时间戳前缀
func FormatLog(message string) string {
	return fmt.Sprintf("[%s] %s", FormatTimestamp(time.Now()), message)
}

// AppendLog 追加日志到现有日志，添加时间戳
func AppendLog(existingLogs, newMessage string) string {
	timestamp := FormatTimestamp(time.Now())
	logEntry := fmt.Sprintf("\n[%s] %s", timestamp, newMessage)
	return existingLogs + logEntry
}

// SanitizeLogMessage 清理日志消息中的敏感信息
func SanitizeLogMessage(message string) string {
	// 移除潜在的敏感信息（如密码、token等）
	sensitivePatterns := []string{
		"password=[^\\s]*",
		"token=[^\\s]*",
		"apiKey=[^\\s]*",
		"secret=[^\\s]*",
	}

	result := message
	for _, pattern := range sensitivePatterns {
		result = strings.ReplaceAll(result, pattern, "***")
	}
	return result
}

// CalculateProgress 计算进度百分比
func CalculateProgress(current, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(current) / float64(total) * 100
}

// FormatProgress 格式化进度显示
func FormatProgress(current, total int, message string) string {
	if total <= 0 {
		return fmt.Sprintf("%s (步骤 %d)", message, current)
	}
	percentage := CalculateProgress(current, total)
	return fmt.Sprintf("%s (%d/%d - %.1f%%)", message, current, total, percentage)
}
