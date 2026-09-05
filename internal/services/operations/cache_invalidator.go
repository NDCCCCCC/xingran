package operations

import (
	"context"

	"github.com/xingran-next/xingran-go-backend/internal/services/base"
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

	// 失效底层统一委托 base 泛型函数（Phase 92-03 D-04）：
	// nil 防护与删除循环已内聚于 base，本方法只保留 entityType 分发语义
	base.InvalidatePattern(ctx, c.cache, patterns, entityType)
	return nil
}

// InvalidateByPatterns 直接根据模式列表清理缓存
func (c *CacheInvalidator) InvalidateByPatterns(
	ctx context.Context,
	patterns []string,
	module string,
) error {
	// 同 InvalidateByEntityType：底层委托 base（Phase 92-03 D-04）
	base.InvalidatePattern(ctx, c.cache, patterns, module)
	return nil
}
