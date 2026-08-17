package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

const (
	// blacklistNegativeTTL "未拉黑"判定的进程内负缓存 TTL。
	// 黑名单检查在认证关键路径上,底层是远程 Redis(Upstash TLS, RTT 高、
	// poolTimeout=10s) — 每请求一次远程往返在网络退化时会把登录链路打成 500
	// (login-menu-timeout-20260817 Round 4)。负缓存使同一 token 30s 内只查一次。
	// 权衡: 拉黑事件后,本进程已负缓存的 token 最长 30s 内仍被放行 — 远小于
	// JWT 自身 7200s 过期,与 pkg/middleware/permission.go 的 30s 权限缓存同级。
	blacklistNegativeTTL = 30 * time.Second

	// blacklistErrorBreakerWindow 缓存故障熔断窗口。
	// Exists 报错(如 Redis 网络退化阻塞至 poolTimeout)后,窗口内直接返回
	// "未拉黑"不再打缓存,避免每个请求都阻塞 10s — 故障期间全链路 fail-open。
	blacklistErrorBreakerWindow = 30 * time.Second

	// blacklistNegativeMaxEntries 负缓存条目上限,超过时先清理过期项。
	blacklistNegativeMaxEntries = 1024
)

// TokenBlacklistService 令牌黑名单服务接口
type TokenBlacklistService interface {
	// AddToBlacklist 将令牌加入黑名单
	AddToBlacklist(ctx context.Context, token string, expiry time.Time) error
	// IsBlacklisted 检查令牌是否在黑名单中
	IsBlacklisted(ctx context.Context, token string) (bool, error)
	// RemoveFromBlacklist 从黑名单中移除令牌（用于测试）
	RemoveFromBlacklist(ctx context.Context, token string) error
}

// tokenBlacklistService 令牌黑名单服务实现
type tokenBlacklistService struct {
	cache cache.Cache

	// 进程内负缓存与错误熔断(login-menu-timeout-20260817 Round 4):
	// negUntil[token] = "未拉黑"判定的过期时间;errUntil = 熔断窗口结束时间。
	mu       sync.RWMutex
	negUntil map[string]time.Time
	errUntil time.Time
}

// NewTokenBlacklistService 创建令牌黑名单服务
func NewTokenBlacklistService(cache cache.Cache) TokenBlacklistService {
	return &tokenBlacklistService{
		cache:    cache,
		negUntil: make(map[string]time.Time),
	}
}

// AddToBlacklist 将令牌加入黑名单
func (s *tokenBlacklistService) AddToBlacklist(ctx context.Context, token string, expiry time.Time) error {
	key := s.getBlacklistKey(token)
	// 计算TTL：令牌过期时间 - 当前时间
	ttl := time.Until(expiry)
	if ttl <= 0 {
		// 令牌已过期，无需加入黑名单
		return nil
	}

	// 加入黑名单，TTL与令牌过期时间一致
	if err := s.cache.Set(ctx, key, "1", ttl); err != nil {
		return fmt.Errorf("加入令牌黑名单失败: %w", err)
	}

	// 使该 token 的负缓存立即失效(本进程拉黑立即可见)
	s.mu.Lock()
	delete(s.negUntil, token)
	s.mu.Unlock()
	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (s *tokenBlacklistService) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	now := time.Now()

	s.mu.RLock()
	if until, ok := s.negUntil[token]; ok && now.Before(until) {
		s.mu.RUnlock()
		return false, nil
	}
	inBreakerWindow := now.Before(s.errUntil)
	s.mu.RUnlock()

	if inBreakerWindow {
		// 缓存故障窗口内直接放行,避免每请求阻塞 poolTimeout(10s)
		return false, nil
	}

	key := s.getBlacklistKey(token)
	exists, err := s.cache.Exists(ctx, key)
	if err != nil {
		// 打开熔断窗口:后续请求在窗口内不再打缓存
		s.mu.Lock()
		s.errUntil = now.Add(blacklistErrorBreakerWindow)
		s.mu.Unlock()
		return false, fmt.Errorf("检查令牌黑名单失败: %w", err)
	}
	if !exists {
		s.rememberNegative(token, now)
	}
	return exists, nil
}

// rememberNegative 记录"未拉黑"判定;超过容量上限时先清理过期项
func (s *tokenBlacklistService) rememberNegative(token string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.negUntil) >= blacklistNegativeMaxEntries {
		for k, until := range s.negUntil {
			if now.After(until) {
				delete(s.negUntil, k)
			}
		}
	}
	s.negUntil[token] = now.Add(blacklistNegativeTTL)
}

// RemoveFromBlacklist 从黑名单中移除令牌（用于测试）
func (s *tokenBlacklistService) RemoveFromBlacklist(ctx context.Context, token string) error {
	key := s.getBlacklistKey(token)
	if err := s.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("从令牌黑名单移除失败: %w", err)
	}
	s.mu.Lock()
	delete(s.negUntil, token)
	s.mu.Unlock()
	return nil
}

// getBlacklistKey 获取黑名单键
func (s *tokenBlacklistService) getBlacklistKey(token string) string {
	return fmt.Sprintf(constants.TokenBlacklistKeyFormat, token)
}
