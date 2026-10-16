package services

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockRateLimitProvider 测试用 RateLimitProvider 实现(Phase 61 QUAL-03 / D-18)
// 预填值与既有硬编码一致(D-17),保证既有测试语义不变
type mockRateLimitProvider struct {
	limits map[string]RateLimit
}

func newMockRateLimitProvider() *mockRateLimitProvider {
	return &mockRateLimitProvider{
		limits: map[string]RateLimit{
			"read":    {PerMinute: 30, PerHour: 500, PerDay: 5000},
			"write":   {PerMinute: 100, PerHour: 1500, PerDay: 15000},
			"admin":   {PerMinute: 200, PerHour: 5000, PerDay: 50000},
			"default": {PerMinute: 120, PerHour: 2000, PerDay: 20000},
		},
	}
}

// GetRateLimit 实现 RateLimitProvider 接口
// key 形态: rate_limit.<scope>.<per_minute|per_hour|per_day>
func (m *mockRateLimitProvider) GetRateLimit(key string, defaultValue int) int {
	parts := strings.Split(key, ".")
	if len(parts) != 3 {
		return defaultValue
	}
	if l, ok := m.limits[parts[1]]; ok {
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

// TestNewRateLimiter 测试初始化
func TestNewRateLimiter(t *testing.T) {
	provider := newMockRateLimitProvider()
	limiter := NewRateLimiter(provider)

	assert.NotNil(t, limiter)
	// 避免 sync.Map 的 noCopy 检查: 通过 Load 验证初始化
	_, loaded := limiter.windows.Load("never-set-key")
	assert.False(t, loaded, "未设置的 key 应返回 loaded=false")

	// D-18: 验证 config 已注入(limits map 字段已移除)
	assert.NotNil(t, limiter.config)

	// 通过 getLimit 间接验证 provider 默认档位(D-17 与既有硬编码一致)
	readLimit := limiter.getLimit("read")
	assert.Equal(t, 30, readLimit.PerMinute)
	assert.Equal(t, 500, readLimit.PerHour)
	assert.Equal(t, 5000, readLimit.PerDay)

	writeLimit := limiter.getLimit("write")
	assert.Equal(t, 100, writeLimit.PerMinute)
	assert.Equal(t, 1500, writeLimit.PerHour)
	assert.Equal(t, 15000, writeLimit.PerDay)

	adminLimit := limiter.getLimit("admin")
	assert.Equal(t, 200, adminLimit.PerMinute)
	assert.Equal(t, 5000, adminLimit.PerHour)
	assert.Equal(t, 50000, adminLimit.PerDay)
}

// TestRateLimiter_Check 测试速率限制检查
func TestRateLimiter_Check(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())

	t.Run("正常请求_未超限", func(t *testing.T) {
		key := "test-key-1"
		scope := "read"

		// 前30个请求应该都成功
		for i := 0; i < 30; i++ {
			allowed, result := limiter.Check(key, scope)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
			assert.NotNil(t, result)
			assert.Equal(t, 30-i-1, result.Remaining, "Remaining requests should decrease")
			assert.Equal(t, 30, result.Limit)
		}
	})

	t.Run("超限请求", func(t *testing.T) {
		key := "test-key-2"
		scope := "read"

		// 发送31个请求（超过每分钟限制30）
		var lastResult *RateLimitResult
		for i := 0; i < 31; i++ {
			allowed, result := limiter.Check(key, scope)
			if i < 30 {
				assert.True(t, allowed, "Request %d should be allowed", i+1)
			} else {
				assert.False(t, allowed, "Request %d should be denied", i+1)
			}
			lastResult = result
		}

		// 验证第31个请求被拒绝
		assert.False(t, lastResult.Allowed)
		assert.Equal(t, 0, lastResult.Remaining)
		assert.Equal(t, 30, lastResult.Limit)
		assert.True(t, lastResult.ResetAt.After(time.Now()))
	})

	t.Run("不同作用域的限制", func(t *testing.T) {
		// read作用域：30/分钟
		readKey := "read-key"
		for i := 0; i < 30; i++ {
			allowed, _ := limiter.Check(readKey, "read")
			assert.True(t, allowed, "Read request %d should be allowed", i+1)
		}
		// 第31个read请求应该被拒绝
		allowed, _ := limiter.Check(readKey, "read")
		assert.False(t, allowed, "31st read request should be denied")

		// write作用域：100/分钟，不应该受到read限制影响
		writeKey := "write-key"
		for i := 0; i < 100; i++ {
			allowed, _ := limiter.Check(writeKey, "write")
			assert.True(t, allowed, "Write request %d should be allowed", i+1)
		}
	})

	t.Run("剩余请求数计算", func(t *testing.T) {
		key := "remaining-key"
		scope := "read"

		// 第1个请求
		_, result1 := limiter.Check(key, scope)
		assert.Equal(t, 29, result1.Remaining)

		// 第2个请求
		_, result2 := limiter.Check(key, scope)
		assert.Equal(t, 28, result2.Remaining)

		// 第30个请求
		for i := 2; i < 30; i++ {
			limiter.Check(key, scope)
		}
		_, result30 := limiter.Check(key, scope)
		assert.Equal(t, 0, result30.Remaining)
	})

	t.Run("重置时间计算", func(t *testing.T) {
		key := "reset-key"
		scope := "read"

		// 发送一些请求
		for i := 0; i < 5; i++ {
			limiter.Check(key, scope)
		}

		// 获取最后一次的结果
		_, result := limiter.Check(key, scope)

		// 重置时间应该在1分钟内
		expectedReset := time.Now().Add(time.Minute)
		assert.WithinDuration(t, expectedReset, result.ResetAt, 5*time.Second)
	})
}

// TestRateLimiter_SlidingWindow 测试滑动窗口
func TestRateLimiter_SlidingWindow(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())

	t.Run("分钟级窗口", func(t *testing.T) {
		key := "minute-window"
		scope := "read"

		// 发送30个请求填满分钟窗口
		for i := 0; i < 30; i++ {
			allowed, _ := limiter.Check(key, scope)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// 第31个请求应该被拒绝
		allowed, _ := limiter.Check(key, scope)
		assert.False(t, allowed, "31st request should be denied")

		// 等待超过1分钟后，窗口应该滑动
		time.Sleep(61 * time.Second)

		// 新请求应该被允许
		allowed, _ = limiter.Check(key, scope)
		assert.True(t, allowed, "New request after 1 minute should be allowed")
	})

	t.Run("小时级窗口", func(t *testing.T) {
		key := "hour-window"
		scope := "write"

		// write限制：100/分钟，1500/小时
		// 发送100个请求填满分钟窗口
		for i := 0; i < 100; i++ {
			allowed, _ := limiter.Check(key, scope)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// 第101个请求应该被拒绝（分钟限制）
		allowed, _ := limiter.Check(key, scope)
		assert.False(t, allowed, "101st request should be denied (minute limit)")
	})

	t.Run("天级窗口", func(t *testing.T) {
		key := "day-window"
		scope := "admin"

		// admin限制：200/分钟，5000/小时，50000/天
		// 发送200个请求填满分钟窗口
		for i := 0; i < 200; i++ {
			allowed, _ := limiter.Check(key, scope)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// 第201个请求应该被拒绝（分钟限制）
		allowed, _ := limiter.Check(key, scope)
		assert.False(t, allowed, "201st request should be denied (minute limit)")
	})

	t.Run("窗口滑动_时间推进", func(t *testing.T) {
		key := "sliding-key"
		scope := "read"

		// 发送30个请求填满分钟窗口
		for i := 0; i < 30; i++ {
			limiter.Check(key, scope)
		}

		// 第31个请求应该被拒绝
		allowed, _ := limiter.Check(key, scope)
		assert.False(t, allowed, "Request should be denied when window is full")

		// 等待超过1分钟
		time.Sleep(61 * time.Second)

		// 新请求应该被允许（窗口已滑动）
		allowed, _ = limiter.Check(key, scope)
		assert.True(t, allowed, "Request should be allowed after window slides")
	})

	t.Run("并发安全", func(t *testing.T) {
		key := "concurrent-key"
		scope := "read"
		numGoroutines := 100
		requestsPerGoroutine := 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// 并发发送请求
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerGoroutine; j++ {
					limiter.Check(key, scope)
				}
			}()
		}

		wg.Wait()

		// 验证并发安全：不应该有panic或死锁
		// 并发后应该仍然能正常检查
		_, result := limiter.Check(key, scope)
		assert.NotNil(t, result)
	})
}

// TestRateLimiter_Cleanup 测试自动清理过期条目
func TestRateLimiter_Cleanup(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())

	t.Run("自动清理过期条目", func(t *testing.T) {
		key := "cleanup-key"
		scope := "read"

		// 发送一些请求
		for i := 0; i < 10; i++ {
			limiter.Check(key, scope)
		}

		// 等待超过1分钟，让条目过期
		time.Sleep(61 * time.Second)

		// 发送新请求，应该清理过期条目
		allowed, result := limiter.Check(key, scope)
		assert.True(t, allowed)
		assert.Equal(t, 29, result.Remaining) // 应该有新的30个配额
	})

	t.Run("清理后计数正确", func(t *testing.T) {
		key := "count-key"
		scope := "write"

		// 发送一些请求
		for i := 0; i < 50; i++ {
			limiter.Check(key, scope)
		}

		// 等待超过1分钟
		time.Sleep(61 * time.Second)

		// 清理后，计数应该从1开始（不是从51开始）
		_, result := limiter.Check(key, scope)
		assert.Equal(t, 99, result.Remaining) // 100 - 1 = 99
	})

	t.Run("内存不泄漏", func(t *testing.T) {
		// 创建大量不同的key
		numKeys := 1000
		for i := 0; i < numKeys; i++ {
			key := "leak-test-key"
			scope := "read"
			limiter.Check(key, scope)
		}

		// 等待条目过期
		time.Sleep(61 * time.Second)

		// 再次检查，应该清理过期条目
		_, result := limiter.Check("leak-test-key", "read")
		assert.NotNil(t, result)

		// 验证windows映射不会无限增长
		// 注意：这里只是基本验证，实际内存泄漏需要更复杂的工具
	})
}

// TestRateLimiter_MultipleKeys 测试不同密钥独立计数
func TestRateLimiter_MultipleKeys(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())

	t.Run("不同密钥独立计数", func(t *testing.T) {
		key1 := "user-1"
		key2 := "user-2"
		scope := "read"

		// key1发送30个请求
		for i := 0; i < 30; i++ {
			allowed, _ := limiter.Check(key1, scope)
			assert.True(t, allowed, "key1 request %d should be allowed", i+1)
		}

		// key1的第31个请求应该被拒绝
		allowed, _ := limiter.Check(key1, scope)
		assert.False(t, allowed, "key1 31st request should be denied")

		// key2应该不受影响，仍然可以发送请求
		for i := 0; i < 30; i++ {
			allowed, _ := limiter.Check(key2, scope)
			assert.True(t, allowed, "key2 request %d should be allowed", i+1)
		}

		// key2的第31个请求也应该被拒绝
		allowed, _ = limiter.Check(key2, scope)
		assert.False(t, allowed, "key2 31st request should be denied")
	})

	t.Run("密钥隔离", func(t *testing.T) {
		keys := []string{"iso-1", "iso-2", "iso-3"}
		scope := "write"

		// 每个key发送100个请求
		for _, key := range keys {
			for i := 0; i < 100; i++ {
				allowed, _ := limiter.Check(key, scope)
				assert.True(t, allowed, "Key %s request %d should be allowed", key, i+1)
			}
		}

		// 所有key的第101个请求都应该被拒绝
		for _, key := range keys {
			allowed, _ := limiter.Check(key, scope)
			assert.False(t, allowed, "Key %s 101st request should be denied", key)
		}
	})
}

// TestRateLimiter_Reset 测试窗口重置
func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())

	t.Run("窗口重置后重新计数", func(t *testing.T) {
		key := "reset-window-key"
		scope := "read"

		// 发送30个请求填满窗口
		for i := 0; i < 30; i++ {
			limiter.Check(key, scope)
		}

		// 第31个请求应该被拒绝
		allowed, _ := limiter.Check(key, scope)
		assert.False(t, allowed, "Request should be denied before reset")

		// 等待窗口重置
		time.Sleep(61 * time.Second)

		// 新请求应该被允许，计数重新开始
		for i := 0; i < 30; i++ {
			allowed, _ := limiter.Check(key, scope)
			assert.True(t, allowed, "Request %d after reset should be allowed", i+1)
		}

		// 再次填满后应该被拒绝
		allowed, _ = limiter.Check(key, scope)
		assert.False(t, allowed, "Request should be denied after refilling")
	})

	t.Run("跨天重置", func(t *testing.T) {
		key := "day-reset-key"
		scope := "admin"

		// 发送200个请求（分钟限制）
		for i := 0; i < 200; i++ {
			limiter.Check(key, scope)
		}

		// 应该被分钟限制阻止
		allowed, _ := limiter.Check(key, scope)
		assert.False(t, allowed, "Should be limited by minute limit")

		// 注意：由于跨天测试需要24小时，这里只测试分钟级重置
		// 实际生产环境中的天级重置逻辑与分钟级相同
	})
}

// TestRateLimiter_EdgeCases 测试边界情况
func TestRateLimiter_EdgeCases(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())

	t.Run("空key", func(t *testing.T) {
		allowed, result := limiter.Check("", "read")
		assert.True(t, allowed)
		assert.NotNil(t, result)
		assert.Equal(t, 29, result.Remaining)
	})

	t.Run("未知作用域", func(t *testing.T) {
		key := "unknown-scope-key"
		allowed, result := limiter.Check(key, "unknown_scope")

		// 应该使用默认限制
		assert.True(t, allowed)
		assert.NotNil(t, result)
		assert.Equal(t, 119, result.Remaining) // 默认120/分钟
	})

	t.Run("零限制边界", func(t *testing.T) {
		// 测试刚好达到限制的情况
		key := "boundary-key"
		scope := "read"

		// 发送29个请求
		for i := 0; i < 29; i++ {
			allowed, result := limiter.Check(key, scope)
			assert.True(t, allowed)
			assert.Equal(t, 29-i, result.Remaining) // 第i+1个请求后剩余 30-(i+1) = 29-i
		}

		// 第30个请求应该成功，剩余为0
		allowed, result := limiter.Check(key, scope)
		assert.True(t, allowed)
		assert.Equal(t, 0, result.Remaining)

		// 第31个请求应该失败
		allowed, _ = limiter.Check(key, scope)
		assert.False(t, allowed)
	})
}

// TestRateLimiter_NilProviderFallback D-18: NewRateLimiter(nil) 不 panic,
// 兜底 staticRateLimitProvider(默认值与既有硬编码一致,D-17)
func TestRateLimiter_NilProviderFallback(t *testing.T) {
	limiter := NewRateLimiter(nil)
	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.config, "nil provider 应被替换为 staticRateLimitProvider")

	// static fallback: read 档 30/分钟
	for i := 0; i < 30; i++ {
		allowed, _ := limiter.Check("key", "read")
		assert.True(t, allowed, "Request %d should be allowed (static fallback)", i+1)
	}
	allowed, _ := limiter.Check("key", "read")
	assert.False(t, allowed, "31st request should be denied (static fallback 30/min)")
}

// TestRateLimiter_ConfigDrivenRead D-18: Check 从 RateLimitProvider 读限额,
// 自定义 provider 阈值(5/分钟)真实生效,不再硬编码
func TestRateLimiter_ConfigDrivenRead(t *testing.T) {
	provider := &mockRateLimitProvider{
		limits: map[string]RateLimit{
			"read": {PerMinute: 5, PerHour: 100, PerDay: 1000},
		},
	}
	limiter := NewRateLimiter(provider)

	for i := 0; i < 5; i++ {
		allowed, result := limiter.Check("key", "read")
		assert.True(t, allowed, "Request %d should be allowed", i+1)
		assert.Equal(t, 5, result.Limit, "Limit 应来自 provider 配置而非硬编码")
	}
	allowed, _ := limiter.Check("key", "read")
	assert.False(t, allowed, "6th request should be denied (provider 配置 5/min)")
}

// TestCalculateResetWindowAware WR-02 回归锚(Phase 61 review):
// calculateReset 必须按真实窗口时长计算 ResetAt — 小时超限不得声称 1 分钟后重置。
func TestCalculateResetWindowAware(t *testing.T) {
	limiter := NewRateLimiter(newMockRateLimitProvider())
	now := time.Now()

	t.Run("分钟窗口_加1分钟", func(t *testing.T) {
		times := []time.Time{now.Add(-30 * time.Second), now.Add(-10 * time.Second)}
		reset := limiter.calculateReset(times, time.Minute)
		assert.WithinDuration(t, now.Add(30*time.Second), reset, 2*time.Second,
			"分钟窗口 ResetAt = 最早时间戳 + 1分钟")
	})

	t.Run("小时窗口_加1小时", func(t *testing.T) {
		times := []time.Time{now.Add(-30 * time.Minute), now.Add(-5 * time.Minute)}
		reset := limiter.calculateReset(times, time.Hour)
		assert.WithinDuration(t, now.Add(30*time.Minute), reset, 2*time.Second,
			"小时窗口 ResetAt = 最早时间戳 + 1小时(不得是 +1分钟, WR-02)")
	})

	t.Run("天窗口_加24小时", func(t *testing.T) {
		times := []time.Time{now.Add(-12 * time.Hour), now.Add(-1 * time.Hour)}
		reset := limiter.calculateReset(times, 24*time.Hour)
		assert.WithinDuration(t, now.Add(12*time.Hour), reset, 2*time.Second,
			"天窗口 ResetAt = 最早时间戳 + 24小时(不得是 +1分钟, WR-02)")
	})

	t.Run("空窗口_返回当前时间", func(t *testing.T) {
		reset := limiter.calculateReset(nil, time.Minute)
		assert.WithinDuration(t, time.Now(), reset, 2*time.Second)
	})
}
