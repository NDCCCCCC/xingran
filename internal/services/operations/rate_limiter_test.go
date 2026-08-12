package operations

import (
	"sync"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	maxTokens := 50
	refillInterval := 500 * time.Millisecond

	limiter := NewRateLimiter(maxTokens, refillInterval)

	if limiter.maxTokens != maxTokens {
		t.Errorf("maxTokens = %v, want %v", limiter.maxTokens, maxTokens)
	}
	if limiter.refillRate != refillInterval {
		t.Errorf("refillRate = %v, want %v", limiter.refillRate, refillInterval)
	}
	if limiter.currentTokens != maxTokens {
		t.Errorf("currentTokens = %v, want %v", limiter.currentTokens, maxTokens)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	maxTokens := 3
	refillInterval := 100 * time.Millisecond
	limiter := NewRateLimiter(maxTokens, refillInterval)

	// 前三次请求应该被允许
	for i := 0; i < maxTokens; i++ {
		if !limiter.Allow() {
			t.Errorf("请求 %d 应该被允许", i+1)
		}
	}

	// 第四次请求应该被拒绝
	if limiter.Allow() {
		t.Error("第四次请求应该被拒绝")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	maxTokens := 2
	refillInterval := 50 * time.Millisecond
	limiter := NewRateLimiter(maxTokens, refillInterval)

	// 消耗所有令牌
	if !limiter.Allow() {
		t.Error("第一个请求应该被允许")
	}
	if !limiter.Allow() {
		t.Error("第二个请求应该被允许")
	}

	// 没有令牌了
	if limiter.Allow() {
		t.Error("应该没有令牌了")
	}

	// 等待令牌补充
	time.Sleep(150 * time.Millisecond)

	// 现在应该有令牌了
	if !limiter.Allow() {
		t.Error("补充后应该有令牌")
	}
}

func TestRateLimiter_GetAvailableTokens(t *testing.T) {
	maxTokens := 5
	limiter := NewRateLimiter(maxTokens, 100*time.Millisecond)

	if tokens := limiter.GetAvailableTokens(); tokens != maxTokens {
		t.Errorf("初始令牌数 = %v, want %v", tokens, maxTokens)
	}

	// 消耗一些令牌
	limiter.Allow()
	limiter.Allow()

	if tokens := limiter.GetAvailableTokens(); tokens != maxTokens-2 {
		t.Errorf("消耗后令牌数 = %v, want %v", tokens, maxTokens-2)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	maxTokens := 10
	limiter := NewRateLimiter(maxTokens, 100*time.Millisecond)

	// 消耗一些令牌
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	if tokens := limiter.GetAvailableTokens(); tokens != maxTokens-5 {
		t.Errorf("消耗后令牌数 = %v, want %v", tokens, maxTokens-5)
	}

	// 重置
	limiter.Reset()

	// 重置后应该恢复到最大值
	if tokens := limiter.GetAvailableTokens(); tokens != maxTokens {
		t.Errorf("重置后令牌数 = %v, want %v", tokens, maxTokens)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	maxTokens := 100
	refillInterval := 10 * time.Millisecond
	limiter := NewRateLimiter(maxTokens, refillInterval)

	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex

	// 并发请求
	for i := 0; i < maxTokens; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// 所有请求都应该被允许（因为令牌足够）
	if allowed != maxTokens {
		t.Errorf("允许的请求数 = %v, want %v", allowed, maxTokens)
	}

	// 下一个请求应该被拒绝
	if limiter.Allow() {
		t.Error("应该没有更多令牌了")
	}
}

func TestRateLimiter_TokenCapping(t *testing.T) {
	maxTokens := 5
	refillInterval := 50 * time.Millisecond
	limiter := NewRateLimiter(maxTokens, refillInterval)

	// 消耗一些令牌
	for i := 0; i < 3; i++ {
		limiter.Allow()
	}

	// 等待足够长的时间让令牌补充到最大值
	time.Sleep(500 * time.Millisecond)

	// 触发一次补充（会触发 refillTokens）
	limiter.Allow()

	// 令牌数应该被限制在最大值以下（因为消耗了一个）
	// 初始5个 - 消耗3个 = 2个
	// 等待500ms，补充 500/50 = 10个令牌
	// 但最多到5个，所以补充到5个
	// 再消耗1个，剩下4个
	expectedTokens := maxTokens - 1
	if tokens := limiter.GetAvailableTokens(); tokens != expectedTokens {
		t.Errorf("令牌数应该被限制在最大值: got %v, want %v", tokens, expectedTokens)
	}
}

func TestBaiduAPIRateLimiter(t *testing.T) {
	// 验证全局限流器已正确初始化
	if BaiduAPIRateLimiter == nil {
		t.Fatal("BaiduAPIRateLimiter 未初始化")
	}

	// 验证限流器配置
	if tokens := BaiduAPIRateLimiter.GetAvailableTokens(); tokens != BaiduAPIMaxTokens {
		t.Errorf("初始令牌数 = %v, want %v", tokens, BaiduAPIMaxTokens)
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a 小于 b", 3, 5, 3},
		{"a 大于 b", 7, 4, 4},
		{"a 等于 b", 6, 6, 6},
		{"负数", -5, -3, -5},
		{"零", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestRateLimiterConstants(t *testing.T) {
	// 验证常量定义
	if BaiduAPIMaxTokens != 100 {
		t.Errorf("BaiduAPIMaxTokens = %v, want 100", BaiduAPIMaxTokens)
	}

	expectedInterval := 600 * time.Millisecond
	if BaiduAPIRefillInterval != expectedInterval {
		t.Errorf("BaiduAPIRefillInterval = %v, want %v", BaiduAPIRefillInterval, expectedInterval)
	}
}
