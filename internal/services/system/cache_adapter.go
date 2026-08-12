package system

import (
	"context"
	"encoding/json"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// CacheAdapter 将 pkg/cache.Cache 适配为 system.CacheProvider 接口
// 解决两个接口不兼容的问题
type CacheAdapter struct {
	cache cache.Cache
}

// NewCacheAdapter 创建缓存适配器
func NewCacheAdapter(c cache.Cache) CacheProvider {
	return &CacheAdapter{cache: c}
}

// adaptCacheStats 将 pkg/cache 统计信息适配为 system.CacheStats
func adaptCacheStats(stats map[string]interface{}) *CacheStats {
	if stats == nil {
		return &CacheStats{}
	}

	result := &CacheStats{}

	// 从 map 中提取统计信息，使用类型断言
	if v, ok := stats["hits"]; ok {
		switch val := v.(type) {
		case int64:
			result.Hits = val
		case int:
			result.Hits = int64(val)
		}
	}

	if v, ok := stats["misses"]; ok {
		switch val := v.(type) {
		case int64:
			result.Misses = val
		case int:
			result.Misses = int64(val)
		}
	}

	if v, ok := stats["count"]; ok {
		switch val := v.(type) {
		case int64:
			result.Count = val
		case int:
			result.Count = int64(val)
		}
	}

	if v, ok := stats["key_count"]; ok {
		switch val := v.(type) {
		case int:
			result.KeyCount = val
		case int64:
			result.KeyCount = int(val)
		}
	}

	// 计算命中率
	if result.Hits > 0 || result.Misses > 0 {
		result.HitRate = float64(result.Hits) / float64(result.Hits+result.Misses)
	}

	return result
}

// GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
func (a *CacheAdapter) GetOrSet(
	ctx context.Context,
	key string,
	dest interface{},
	expiration time.Duration,
	query func() (interface{}, error),
) error {
	// 尝试从缓存获取
	cached, err := a.cache.Get(ctx, key)
	if err == nil && cached != "" {
		// 缓存命中，反序列化并设置到目标变量
		if err := json.Unmarshal([]byte(cached), dest); err != nil {
			logger.Warnf("反序列化缓存失败: key=%s, error=%v", key, err)
			// 反序列化失败，继续执行查询
		} else {
			return nil
		}
	}

	// 缓存未命中或反序列化失败，执行查询
	result, err := query()
	if err != nil {
		return err
	}

	// 序列化结果
	data, err := json.Marshal(result)
	if err != nil {
		logger.Warnf("序列化结果失败: key=%s, error=%v", key, err)
		// 序列化失败，仍然设置结果到目标变量
		setValue(dest, result)
		return nil
	}

	// 保存到缓存
	if cacheErr := a.cache.Set(ctx, key, string(data), expiration); cacheErr != nil {
		logger.Warnf("保存缓存失败: key=%s, error=%v", key, cacheErr)
	}

	// 设置结果到目标变量
	setValue(dest, result)

	return nil
}

// setValue 使用反射设置目标变量的值（定义在 cache_provider.go 中）

// Delete 删除缓存
func (a *CacheAdapter) Delete(ctx context.Context, key string) error {
	return a.cache.Delete(ctx, key)
}

// DeleteByPattern 根据模式删除缓存
// 使用Keys方法获取匹配的键，然后批量删除
func (a *CacheAdapter) DeleteByPattern(ctx context.Context, pattern string) error {
	// 获取匹配模式的所有键
	keys, err := a.cache.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	// 批量删除
	if len(keys) > 0 {
		return a.cache.MDelete(ctx, keys...)
	}

	return nil
}

// MGet 批量获取缓存
func (a *CacheAdapter) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	values, err := a.cache.MGet(ctx, keys...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(keys))
	for i, key := range keys {
		if i < len(values) && values[i] != "" {
			result[key] = values[i]
		}
	}

	return result, nil
}

// MDelete 批量删除缓存
func (a *CacheAdapter) MDelete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return a.cache.MDelete(ctx, keys...)
}

// Exists 检查缓存是否存在
func (a *CacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.cache.Exists(ctx, key)
}

// SetTTL 设置缓存过期时间
func (a *CacheAdapter) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return a.cache.Expire(ctx, key, expiration)
}

// GetTTL 获取缓存过期时间
func (a *CacheAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return a.cache.TTL(ctx, key)
}

// GetStats 获取缓存统计信息
func (a *CacheAdapter) GetStats(ctx context.Context) (*CacheStats, error) {
	// 基础统计
	statsMap := map[string]interface{}{
		"hits":      0,
		"misses":    0,
		"count":     0,
		"key_count": 0,
	}

	// 尝试获取底层缓存的完整统计（如果支持）
	type cacheWithStats interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}
	if cacheWithStats, ok := a.cache.(cacheWithStats); ok {
		if stats, err := cacheWithStats.GetStats(ctx); err == nil {
			statsMap = stats
		}
	}

	// 尝试获取键数量
	if keys, err := a.cache.Keys(ctx, "*"); err == nil {
		statsMap["key_count"] = len(keys)
		statsMap["count"] = int64(len(keys))
	}

	// 适配基础统计
	result := adaptCacheStats(statsMap)

	// 添加扩展统计（L2Writer、RetryWorker等）
	result.ExtendedStats = extractExtendedStats(statsMap)

	return result, nil
}

// extractExtendedStats 提取扩展统计信息（L2Writer、RetryWorker等）
func extractExtendedStats(statsMap map[string]interface{}) map[string]interface{} {
	extended := make(map[string]interface{})

	// 提取 L2Writer 统计
	if l2WriterStats, ok := statsMap["l2_writer"].(map[string]interface{}); ok {
		extended["l2_writer"] = l2WriterStats
	}

	// 提取 RetryWorker 统计
	if retryStats, ok := statsMap["retry_worker"].(map[string]interface{}); ok {
		extended["retry_worker"] = retryStats
	}

	// 提取 L1/L2 统计
	if l1Stats, ok := statsMap["l1"].(map[string]interface{}); ok {
		extended["l1"] = l1Stats
	}
	if l2Stats, ok := statsMap["l2"].(map[string]interface{}); ok {
		extended["l2"] = l2Stats
	}

	return extended
}
