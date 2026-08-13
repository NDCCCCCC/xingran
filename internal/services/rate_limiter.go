package services

import (
	"strings"
	"sync"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
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
//
// Phase 61 QUAL-03 / D-18: 移除硬编码 limits map,改为从 RateLimitProvider
// (CacheConfigService) 运行时读取阈值;reload 后新请求即读到新阈值,
// 在途请求的滑动窗口时间戳不受影响(D-19)。
type RateLimiter struct {
	windows sync.Map          // 并发安全的窗口映射 (key -> *rateLimitWindow)
	config  RateLimitProvider // 限流阈值配置提供者(D-18)
}

// RateLimitResult 速率限制结果
type RateLimitResult struct {
	Allowed   bool      // 是否允许
	Remaining int       // 剩余请求数
	ResetAt   time.Time // 重置时间
	Limit     int       // 限制总数
}

// NewRateLimiter 创建速率限制器实例
//
// Phase 61 QUAL-03 / D-18: 接收 RateLimitProvider(生产传 core.CacheConfigService
// 字段),运行时从 sys_config 读取 rate_limit.* 阈值。config == nil 时兜底
// staticRateLimitProvider(与既有硬编码默认值一致,D-17),保证旧调用路径不 panic。
func NewRateLimiter(config RateLimitProvider) *RateLimiter {
	if config == nil {
		applogger.Warnf("[RATE_LIMITER] NewRateLimiter(nil), fallback to static defaults")
		config = newStaticRateLimitProvider()
	}
	return &RateLimiter{
		config: config,
	}
}

// getLimit 从 RateLimitProvider 读取指定 scope 的限流配置(私有方法)
// scope 不存在对应配置时,GetRateLimit 返回传入的 default 兜底值(120/2000/20000)
func (rl *RateLimiter) getLimit(scope string) RateLimit {
	if rl.config == nil {
		return RateLimit{PerMinute: 120, PerHour: 2000, PerDay: 20000}
	}
	return RateLimit{
		PerMinute: rl.config.GetRateLimit("rate_limit."+scope+".per_minute", 120),
		PerHour:   rl.config.GetRateLimit("rate_limit."+scope+".per_hour", 2000),
		PerDay:    rl.config.GetRateLimit("rate_limit."+scope+".per_day", 20000),
	}
}

// staticRateLimitProvider 静态兜底配置提供者(NewRateLimiter(nil) 时使用)
// 预填值与 Phase 61 之前既有硬编码一致(D-17)
type staticRateLimitProvider struct {
	limits map[string]RateLimit
}

func newStaticRateLimitProvider() *staticRateLimitProvider {
	return &staticRateLimitProvider{
		limits: map[string]RateLimit{
			APIKeyScopeRead:  {PerMinute: 30, PerHour: 500, PerDay: 5000},
			APIKeyScopeWrite: {PerMinute: 100, PerHour: 1500, PerDay: 15000},
			APIKeyScopeAdmin: {PerMinute: 200, PerHour: 5000, PerDay: 50000},
			"default":        {PerMinute: 120, PerHour: 2000, PerDay: 20000},
		},
	}
}

// GetRateLimit 实现 RateLimitProvider 接口
// key 形态: rate_limit.<scope>.<per_minute|per_hour|per_day>
func (p *staticRateLimitProvider) GetRateLimit(key string, defaultValue int) int {
	parts := strings.Split(key, ".")
	if len(parts) != 3 {
		return defaultValue
	}
	if l, ok := p.limits[parts[1]]; ok {
		switch parts[2] {
		case "per_minute":
			return l.PerMinute
		case "per_hour":
			return l.PerHour
		case "per_day":
			return l.PerDay
		}
	}
	return defaultValue
}

// Check 检查请求是否超过速率限制
// 参数: key - 唯一标识符（如API Key ID或用户ID）
//       scope - 作用域（read/write/admin）
// 返回: (bool, *RateLimitResult) - 是否允许，速率限制结果
func (rl *RateLimiter) Check(key string, scope string) (bool, *RateLimitResult) {
	// 获取作用域限制配置 (D-18: 运行时从 RateLimitProvider 读,
	// scope 无对应配置时由 GetRateLimit defaultValue 兜底 120/2000/20000)
	limit := rl.getLimit(scope)

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
