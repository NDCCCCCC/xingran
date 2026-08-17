package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// blacklistMockCache 可控行为的 cache.Cache 实现,用于黑名单服务测试。
// 内嵌接口提供未覆盖方法的默认(nil)实现;服务仅调用 Exists/Set/Delete。
type blacklistMockCache struct {
	cache.Cache
	existsCalls int
	exists      bool
	err         error
	setCalls    int
	delCalls    int
}

func (m *blacklistMockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.setCalls++
	return nil
}
func (m *blacklistMockCache) Delete(ctx context.Context, key string) error {
	m.delCalls++
	return nil
}
func (m *blacklistMockCache) Exists(ctx context.Context, key string) (bool, error) {
	m.existsCalls++
	return m.exists, m.err
}

// TestIsBlacklisted_NegativeCache "未拉黑"判定在 TTL 内不再打底层缓存。
// login-menu-timeout-20260817 Round 4: 黑名单检查在认证关键路径上直连远程
// Redis,每请求一次远程往返在网络退化时致命。
func TestIsBlacklisted_NegativeCache(t *testing.T) {
	mock := &blacklistMockCache{exists: false}
	svc := NewTokenBlacklistService(mock)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		blacklisted, err := svc.IsBlacklisted(ctx, "token-a")
		if err != nil {
			t.Fatalf("IsBlacklisted: %v", err)
		}
		if blacklisted {
			t.Fatalf("expected not blacklisted")
		}
	}
	if mock.existsCalls != 1 {
		t.Fatalf("expected 1 underlying Exists call, got %d", mock.existsCalls)
	}
}

// TestIsBlacklisted_PositiveNotCached "已拉黑"判定不进入负缓存,每次都真实查询。
func TestIsBlacklisted_PositiveNotCached(t *testing.T) {
	mock := &blacklistMockCache{exists: true}
	svc := NewTokenBlacklistService(mock)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		blacklisted, err := svc.IsBlacklisted(ctx, "token-b")
		if err != nil {
			t.Fatalf("IsBlacklisted: %v", err)
		}
		if !blacklisted {
			t.Fatalf("expected blacklisted")
		}
	}
	if mock.existsCalls != 3 {
		t.Fatalf("expected 3 underlying Exists calls, got %d", mock.existsCalls)
	}
}

// TestIsBlacklisted_ErrorBreaker 首次报错打开熔断窗口,窗口内不再打底层缓存,
// 且熔断命中直接返回 (false, nil) — 故障期间 fail-open,不再每请求阻塞
// poolTimeout(10s)。
func TestIsBlacklisted_ErrorBreaker(t *testing.T) {
	mock := &blacklistMockCache{err: errors.New("redis pool timeout")}
	svc := NewTokenBlacklistService(mock)
	ctx := context.Background()

	// 第一次:真实调用,报错
	if _, err := svc.IsBlacklisted(ctx, "token-c"); err == nil {
		t.Fatalf("expected error on first call")
	}
	// 后续:熔断窗口内直接放行,不再打缓存,不再报错
	for i := 0; i < 5; i++ {
		blacklisted, err := svc.IsBlacklisted(ctx, "token-c")
		if err != nil {
			t.Fatalf("breaker window should fail-open without error, got: %v", err)
		}
		if blacklisted {
			t.Fatalf("breaker window should return not-blacklisted")
		}
	}
	if mock.existsCalls != 1 {
		t.Fatalf("expected 1 underlying Exists call, got %d", mock.existsCalls)
	}
}

// TestAddToBlacklist_InvalidatesNegativeCache 拉黑操作立即使该 token 的负缓存失效。
func TestAddToBlacklist_InvalidatesNegativeCache(t *testing.T) {
	mock := &blacklistMockCache{exists: false}
	svc := NewTokenBlacklistService(mock)
	ctx := context.Background()

	// 建立负缓存
	if _, err := svc.IsBlacklisted(ctx, "token-d"); err != nil {
		t.Fatalf("IsBlacklisted: %v", err)
	}
	if mock.existsCalls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.existsCalls)
	}

	// 拉黑(未来过期时间)
	if err := svc.AddToBlacklist(ctx, "token-d", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("AddToBlacklist: %v", err)
	}

	// 再查:负缓存已失效,必须回源
	mock.exists = true
	blacklisted, err := svc.IsBlacklisted(ctx, "token-d")
	if err != nil {
		t.Fatalf("IsBlacklisted after AddToBlacklist: %v", err)
	}
	if !blacklisted {
		t.Fatalf("expected blacklisted after AddToBlacklist")
	}
	if mock.existsCalls != 2 {
		t.Fatalf("expected 2 calls (negative cache invalidated), got %d", mock.existsCalls)
	}
}
