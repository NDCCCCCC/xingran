package retry

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Retryable 判断错误是否可重试
type Retryable func(error) bool

// Config 重试配置
type Config struct {
	MaxRetries    int           // 最大重试次数
	InitialDelay  time.Duration // 初始延迟
	MaxDelay      time.Duration // 最大延迟
	Multiplier    float64       // 延迟倍数
	Jitter        bool          // 是否添加随机抖动
	Retryable     Retryable     // 判断函数
}

// DefaultConfig 默认重试配置
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		Retryable:    IsNetworkError,
	}
}

// IsNetworkError 判断是否为网络错误（可重试）
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// 网络相关错误
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"no such host",
		"temporary failure",
		"network is unreachable",
		"EOF",
		"broken pipe",
	}

	for _, pattern := range networkErrors {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// IsHTTPRetryable 判断 HTTP 状态码是否可重试
func IsHTTPRetryable(statusCode int) bool {
	// 429 Too Many Requests
	// 5xx 服务器错误
	return statusCode == 429 || (statusCode >= 500 && statusCode < 600)
}

// DoWithRetry 执行带重试的操作
func DoWithRetry(ctx context.Context, config *Config, fn func() error) error {
	if config == nil {
		config = DefaultConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待后重试
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry canceled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		// 执行操作
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if config.Retryable != nil && !config.Retryable(err) {
			return err
		}

		// 计算下次延迟
		delay = time.Duration(float64(delay) * config.Multiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// 添加抖动（避免惊群效应）
		if config.Jitter {
			jitter := time.Duration(rand.Int63n(int64(delay) / 2))
			delay = delay - jitter
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
}

// contains 字符串包含检查（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s[:len(substr)] == substr ||
		 containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
