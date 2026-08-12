package operations

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// CacheInvalidator 缓存清理器
// 用于在Excel导入后清理相关实体的缓存
type CacheInvalidator struct {
	cache system.CacheProvider
}

// NewCacheInvalidator 创建缓存清理器
func NewCacheInvalidator(cache system.CacheProvider) *CacheInvalidator {
	return &CacheInvalidator{cache: cache}
}

// InvalidateByEntityType 根据实体类型清理缓存
// patterns从ExcelConfig的CachePatterns字段获取
func (c *CacheInvalidator) InvalidateByEntityType(
	ctx context.Context,
	entityType string,
	patterns []string,
) error {
	if len(patterns) == 0 {
		logger.Debugf("[%s] 没有配置缓存清理模式", entityType)
		return nil
	}

	// 如果没有配置缓存，直接返回
	if c.cache == nil {
		logger.Debugf("[%s] 未配置缓存提供者，跳过缓存清理", entityType)
		return nil
	}

	for _, pattern := range patterns {
		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			logger.Warnf("[%s] 清除缓存失败: pattern=%s, error=%v", entityType, pattern, err)
		} else {
			logger.Debugf("[%s] 清除缓存成功: pattern=%s", entityType, pattern)
		}
	}

	return nil
}

// InvalidateByPatterns 直接根据模式列表清理缓存
func (c *CacheInvalidator) InvalidateByPatterns(
	ctx context.Context,
	patterns []string,
	module string,
) error {
	// 如果没有配置缓存，直接返回
	if c.cache == nil {
		logger.Debugf("[%s] 未配置缓存提供者，跳过缓存清理", module)
		return nil
	}

	for _, pattern := range patterns {
		if err := c.cache.DeleteByPattern(ctx, pattern); err != nil {
			logger.Warnf("[%s] 清除缓存失败: pattern=%s, error=%v", module, pattern, err)
		}
	}
	return nil
}
