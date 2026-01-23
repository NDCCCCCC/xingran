package monitor

import (
	"context"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// ==================== 辅助函数 ====================

// getCachesFromCache 使用缓存接口获取缓存列表
func getCachesFromCache(cacheInterface interface {
	Get(ctx context.Context, key string) (string, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Exists(ctx context.Context, key string) (bool, error)
}, req CacheListRequest) ([]models.CacheInfo, error) {
	ctx := context.Background()
	var caches []models.CacheInfo

	// 使用Keys方法获取所有匹配的键
	pattern := "*"
	if req.Key != "" {
		pattern = "*" + req.Key + "*"
	}

	keys, err := cacheInterface.Keys(ctx, pattern)
	if err != nil {
		// 如果获取键失败，返回空列表而不是错误
		return []models.CacheInfo{}, nil
	}

	for _, key := range keys {
		// 获取缓存值
		value, err := cacheInterface.Get(ctx, key)
		if err != nil {
			continue // 跳过获取失败的键
		}

		// 获取TTL
		ttl, _ := cacheInterface.TTL(ctx, key)
		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		// 检查是否存在
		exists, _ := cacheInterface.Exists(ctx, key)

		cacheInfo := models.CacheInfo{
			Key:       key,
			Value:     value,
			Type:      "string", // 简化处理，都作为字符串处理
			Size:      int64(len(value)),
			TTL:       ttlSeconds,
			CreatedAt: time.Now(), // 缓存接口不提供创建时间，使用当前时间
			UpdatedAt: time.Now(),
		}

		// 如果指定了类型过滤，只返回匹配的缓存（简化处理）
		if req.Type != "" && !strings.Contains(strings.ToLower(cacheInfo.Type), strings.ToLower(req.Type)) {
			continue
		}

		// 如果键不存在，跳过
		if !exists {
			continue
		}

		caches = append(caches, cacheInfo)
	}

	return caches, nil
}

// getCachesFromCacheWithLevel 使用缓存接口获取指定层级的缓存列表
func getCachesFromCacheWithLevel(cacheInterface interface{}, req CacheListRequest) ([]models.CacheInfo, error) {
	ctx := context.Background()

	pattern := "*"
	if req.Key != "" {
		pattern = "*" + req.Key + "*"
	}

	// 类型断言为 MultiLevelCache
	type MultiLevelCacheInterface interface {
		KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
		GetKeyLocations(ctx context.Context, pattern string) (map[string]string, error)
		Get(ctx context.Context, key string) (string, error)
		TTL(ctx context.Context, key string) (time.Duration, error)
		Exists(ctx context.Context, key string) (bool, error)
	}

	mlCache, ok := cacheInterface.(MultiLevelCacheInterface)
	if !ok {
		// 如果不是 MultiLevelCache，返回空结果
		return []models.CacheInfo{}, nil
	}

	var keys []string
	var locations map[string]string
	var err error

	// 根据 level 获取 keys
	switch req.Level {
	case "l1", "L1":
		keys, err = mlCache.KeysByLevel(ctx, pattern, "l1")
		locations = make(map[string]string)
		for _, k := range keys {
			locations[k] = "l1"
		}
	case "l2", "L2":
		keys, err = mlCache.KeysByLevel(ctx, pattern, "l2")
		locations = make(map[string]string)
		for _, k := range keys {
			locations[k] = "l2"
		}
	default: // "all" 或其他
		keys, err = mlCache.KeysByLevel(ctx, pattern, "all")
		locations, _ = mlCache.GetKeyLocations(ctx, pattern)
	}

	if err != nil {
		return []models.CacheInfo{}, nil
	}

	var caches []models.CacheInfo
	for _, key := range keys {
		value, err := mlCache.Get(ctx, key)
		if err != nil {
			continue
		}

		ttl, _ := mlCache.TTL(ctx, key)
		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		exists, _ := mlCache.Exists(ctx, key)
		if !exists {
			continue
		}

		location := "unknown"
		if loc, ok := locations[key]; ok {
			location = loc
		}

		cacheInfo := models.CacheInfo{
			Key:       key,
			Value:     value,
			Type:      "string",
			Size:      int64(len(value)),
			TTL:       ttlSeconds,
			Location:  location,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		caches = append(caches, cacheInfo)
	}

	return caches, nil
}

// formatCacheStats 格式化缓存统计信息
func formatCacheStats(stats map[string]interface{}, cacheName string) map[string]interface{} {
	hitCount, _ := stats["keyspace_hits"].(int64)
	missCount, _ := stats["keyspace_misses"].(int64)
	hitRate, _ := stats["hit_rate"].(float64)
	usedMemory, _ := stats["used_memory"].(int64)
	keyCount, _ := stats["key_count"].(int64)

	var totalMemory int64
	if totalMem, ok := stats["total_system_memory"].(int64); ok && totalMem > 0 {
		totalMemory = totalMem
	} else if maxMem, ok := stats["maxmemory"].(int64); ok && maxMem > 0 {
		totalMemory = maxMem
	} else {
		totalMemory = usedMemory * 2
	}

	return map[string]interface{}{
		"cacheName":   cacheName,
		"cacheType":   "memory",
		"hitCount":    hitCount,
		"missCount":   missCount,
		"hitRate":     hitRate,
		"totalMemory": totalMemory,
		"usedMemory":  usedMemory,
		"keyCount":    keyCount,
		"collectTime": time.Now(),
	}
}

// formatCacheStatsForMonitor 格式化缓存统计信息（用于监控接口）
func formatCacheStatsForMonitor(stats map[string]interface{}) map[string]interface{} {
	hitCount, _ := stats["keyspace_hits"].(int64)
	missCount, _ := stats["keyspace_misses"].(int64)
	hitRate, _ := stats["hit_rate"].(float64)
	usedMemory, _ := stats["used_memory"].(int64)
	keyCount, _ := stats["key_count"].(int64)

	var totalMemory int64
	if totalMem, ok := stats["total_system_memory"].(int64); ok && totalMem > 0 {
		totalMemory = totalMem
	} else if maxMem, ok := stats["maxmemory"].(int64); ok && maxMem > 0 {
		totalMemory = maxMem
	} else {
		totalMemory = usedMemory * 2
	}

	return map[string]interface{}{
		"hitCount":    hitCount,
		"missCount":   missCount,
		"hitRate":     hitRate,
		"totalMemory": totalMemory,
		"usedMemory":  usedMemory,
		"keyCount":    keyCount,
	}
}
