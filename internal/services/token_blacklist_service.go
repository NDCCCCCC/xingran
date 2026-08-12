package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
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
}

// NewTokenBlacklistService 创建令牌黑名单服务
func NewTokenBlacklistService(cache cache.Cache) TokenBlacklistService {
	return &tokenBlacklistService{
		cache: cache,
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
	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (s *tokenBlacklistService) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	key := s.getBlacklistKey(token)
	exists, err := s.cache.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("检查令牌黑名单失败: %w", err)
	}
	return exists, nil
}

// RemoveFromBlacklist 从黑名单中移除令牌（用于测试）
func (s *tokenBlacklistService) RemoveFromBlacklist(ctx context.Context, token string) error {
	key := s.getBlacklistKey(token)
	if err := s.cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("从令牌黑名单移除失败: %w", err)
	}
	return nil
}

// getBlacklistKey 获取黑名单键
func (s *tokenBlacklistService) getBlacklistKey(token string) string {
	return "token:blacklist:" + token
}
