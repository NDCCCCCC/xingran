package system

import (
	"context"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// filterSlice 通用切片过滤
func filterSlice[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// contains 字符串包含检查（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// paginate 内存分页工具函数
func paginate[T any](items []T, current, pageSize int) ([]T, int64) {
	total := int64(len(items))
	if total == 0 {
		return []T{}, 0
	}

	start := (current - 1) * pageSize
	if start >= len(items) {
		return []T{}, total
	}

	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	return items[start:end], total
}

// ==================== 缓存服务通用辅助函数 ====================

// CacheServiceBase 缓存服务基础结构
type CacheServiceBase struct {
	Config *services.CacheConfigService
}

// GetExpiration 获取缓存过期时间（通用方法）
func (b *CacheServiceBase) GetExpiration(configKey string, defaultVal time.Duration) time.Duration {
	if b.Config != nil {
		return b.Config.GetDurationWithDefault(configKey, defaultVal)
	}
	return defaultVal
}

// InvalidateCacheByPattern 根据模式列表失效缓存（通用方法）
func InvalidateCacheByPattern(ctx context.Context, cache CacheProvider, patterns []string, module string) {
	for _, pattern := range patterns {
		if err := cache.DeleteByPattern(ctx, pattern); err != nil {
			logger.Warnf("[%s] 清除缓存失败: %v", module, err)
		}
	}
}

// InvalidateCacheByKey 根据键列表失效缓存（通用方法）
func InvalidateCacheByKey(ctx context.Context, cache CacheProvider, keys []string, module string) {
	for _, key := range keys {
		if err := cache.Delete(ctx, key); err != nil {
			logger.Warnf("[%s] 清除缓存失败: %v", module, err)
		}
	}
}
