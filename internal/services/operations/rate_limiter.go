package operations

import (
	"sync"
	"time"
)

const (
	// BaiduAPIMaxTokens 百度API最大令牌数（每分钟请求数）
	BaiduAPIMaxTokens = 100
	// BaiduAPIRefillInterval 百度API令牌填充间隔（600ms = 100次/分钟）
	BaiduAPIRefillInterval = 600 * time.Millisecond
)

// RateLimiter 令牌桶限流器
// 用于控制API调用频率，保护第三方API配额
type RateLimiter struct {
	mu            sync.Mutex
	maxTokens     int           // 桶容量
	refillRate    time.Duration // 填充间隔
	currentTokens int
	lastRefill    time.Time
}

// NewRateLimiter 创建限流器
// maxTokens: 桶容量（最大请求数）
// refillInterval: 填充间隔（每多久填充一个令牌）
func NewRateLimiter(maxTokens int, refillInterval time.Duration) *RateLimiter {
	return &RateLimiter{
		maxTokens:     maxTokens,
		refillRate:    refillInterval,
		currentTokens: maxTokens,
		lastRefill:    time.Now(),
	}
}

// Allow 检查是否允许请求
// 返回 true 表示允许请求，false 表示超过限制
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillTokens()

	if rl.currentTokens > 0 {
		rl.currentTokens--
		return true
	}

	return false
}

// GetAvailableTokens 获取当前可用令牌数
func (rl *RateLimiter) GetAvailableTokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.currentTokens
}

// Reset 重置限流器
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.currentTokens = rl.maxTokens
	rl.lastRefill = time.Now()
}

// refillTokens 补充令牌
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.currentTokens = min(rl.currentTokens+tokensToAdd, rl.maxTokens)
		rl.lastRefill = now
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BaiduAPIRateLimiter 百度地图API限流器
// 每分钟最多100次请求
var BaiduAPIRateLimiter = NewRateLimiter(BaiduAPIMaxTokens, BaiduAPIRefillInterval)
