package services

import (
	"sync"
	"time"
)

// API Key 作用域名称（rate_limiter 与 system/apikey_service 各持一份以避免跨包耦合）
const (
	APIKeyScopeRead  = "read"
	APIKeyScopeWrite = "write"
	APIKeyScopeAdmin = "admin"
)

// RateLimit 速率限制配置
type RateLimit struct {
	PerMinute int // 每分钟限制
	PerHour   int // 每小时限制
	PerDay    int // 每天限制
}

// rateLimitWindow 速率限制窗口（滑动窗口）
type rateLimitWindow struct {
	minute []time.Time // 分钟级窗口
	hour   []time.Time // 小时级窗口
	day    []time.Time // 天级窗口
	mu     sync.Mutex  // 互斥锁
}

// RateLimiter 速率限制器
type RateLimiter struct {
	windows sync.Map          // 并发安全的窗口映射 (key -> *rateLimitWindow)
	limits  map[string]RateLimit // 作用域限制配置
}

// RateLimitResult 速率限制结果
type RateLimitResult struct {
	Allowed   bool      // 是否允许
	Remaining int       // 剩余请求数
	ResetAt   time.Time // 重置时间
	Limit     int       // 限制总数
}

// NewRateLimiter 创建速率限制器实例
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: map[string]RateLimit{
			APIKeyScopeRead:  {PerMinute: 30, PerHour: 500, PerDay: 5000},
			APIKeyScopeWrite: {PerMinute: 100, PerHour: 1500, PerDay: 15000},
			APIKeyScopeAdmin: {PerMinute: 200, PerHour: 5000, PerDay: 50000},
		},
	}
}

// Check 检查请求是否超过速率限制
// 参数: key - 唯一标识符（如API Key ID或用户ID）
//       scope - 作用域（read/write/admin）
// 返回: (bool, *RateLimitResult) - 是否允许，速率限制结果
func (rl *RateLimiter) Check(key string, scope string) (bool, *RateLimitResult) {
	// 获取作用域限制配置
	limit, ok := rl.limits[scope]
	if !ok {
		// 如果作用域不存在，使用默认限制
		limit = RateLimit{PerMinute: 120, PerHour: 2000, PerDay: 20000}
	}

	// 获取或创建窗口
	window := rl.getOrCreateWindow(key)

	// 加锁处理
	window.mu.Lock()
	defer window.mu.Unlock()

	now := time.Now()

	// 清理过期条目
	window.minute = rl.cleanOlderThan(window.minute, now.Add(-time.Minute))
	window.hour = rl.cleanOlderThan(window.hour, now.Add(-time.Hour))
	window.day = rl.cleanOlderThan(window.day, now.Add(-24*time.Hour))

	// 检查限制
	if len(window.minute) >= limit.PerMinute {
		// 超过每分钟限制
		result := &RateLimitResult{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   rl.calculateReset(window.minute),
			Limit:     limit.PerMinute,
		}
		return false, result
	}

	if len(window.hour) >= limit.PerHour {
		// 超过每小时限制
		result := &RateLimitResult{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   rl.calculateReset(window.hour),
			Limit:     limit.PerHour,
		}
		return false, result
	}

	if len(window.day) >= limit.PerDay {
		// 超过每天限制
		result := &RateLimitResult{
			Allowed:   false,
			Remaining: 0,
			ResetAt:   rl.calculateReset(window.day),
			Limit:     limit.PerDay,
		}
		return false, result
	}

	// 未超限，添加当前请求时间
	window.minute = append(window.minute, now)
	window.hour = append(window.hour, now)
	window.day = append(window.day, now)

	// 计算剩余请求数（取最小值）
	remaining := rl.calculateRemaining(limit, len(window.minute), len(window.hour), len(window.day))

	result := &RateLimitResult{
		Allowed:   true,
		Remaining: remaining,
		ResetAt:   now.Add(time.Minute),
		Limit:     limit.PerMinute,
	}

	return true, result
}

// getOrCreateWindow 获取或创建窗口（私有方法）
func (rl *RateLimiter) getOrCreateWindow(key string) *rateLimitWindow {
	// 尝试从 sync.Map 加载
	if window, ok := rl.windows.Load(key); ok {
		return window.(*rateLimitWindow)
	}

	// 创建新窗口
	newWindow := &rateLimitWindow{
		minute: make([]time.Time, 0),
		hour:   make([]time.Time, 0),
		day:    make([]time.Time, 0),
	}

	// 存储到 sync.Map（如果已存在则使用现有值）
	actual, _ := rl.windows.LoadOrStore(key, newWindow)
	return actual.(*rateLimitWindow)
}

// cleanOlderThan 清理早于指定时间的条目（私有方法）
func (rl *RateLimiter) cleanOlderThan(times []time.Time, cutoff time.Time) []time.Time {
	// 使用二分查找找到第一个不早于 cutoff 的时间点
	left, right := 0, len(times)
	for left < right {
		mid := (left + right) / 2
		if times[mid].Before(cutoff) {
			left = mid + 1
		} else {
			right = mid
		}
	}

	// 返回清理后的切片
	return times[left:]
}

// calculateRemaining 计算剩余请求数（私有方法）
// 返回三个窗口中的最小剩余值
func (rl *RateLimiter) calculateRemaining(limit RateLimit, minuteCount, hourCount, dayCount int) int {
	remainingMinute := limit.PerMinute - minuteCount
	remainingHour := limit.PerHour - hourCount
	remainingDay := limit.PerDay - dayCount

	// 返回最小值
	if remainingMinute <= remainingHour && remainingMinute <= remainingDay {
		return remainingMinute
	}
	if remainingHour <= remainingMinute && remainingHour <= remainingDay {
		return remainingHour
	}
	return remainingDay
}

// calculateReset 计算重置时间（私有方法）
// 返回最早过期时间
func (rl *RateLimiter) calculateReset(times []time.Time) time.Time {
	if len(times) == 0 {
		return time.Now()
	}

	// 返回最早的时间点 + 1分钟（因为分钟窗口的清理时间是1分钟）
	return times[0].Add(time.Minute)
}
