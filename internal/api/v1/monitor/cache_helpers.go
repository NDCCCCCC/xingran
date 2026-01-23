package monitor

import (
	"context"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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
	pattern := "*"
	if req.Key != "" {
		pattern = "*" + req.Key + "*"
	}

	keys, err := cacheInterface.Keys(ctx, pattern)
	if err != nil {
		return []models.CacheInfo{}, nil
	}

	var caches []models.CacheInfo
	for _, key := range keys {
		value, err := cacheInterface.Get(ctx, key)
		if err != nil {
			continue
		}

		ttl, _ := cacheInterface.TTL(ctx, key)
		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		exists, _ := cacheInterface.Exists(ctx, key)

		cacheInfo := models.CacheInfo{
			Key:       key,
			Value:     value,
			Type:      "string",
			Size:      int64(len(value)),
			TTL:       ttlSeconds,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if req.Type != "" && !strings.Contains(strings.ToLower(cacheInfo.Type), strings.ToLower(req.Type)) {
			continue
		}

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

	type MultiLevelCacheInterface interface {
		KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
		Get(ctx context.Context, key string) (string, error)
		TTL(ctx context.Context, key string) (time.Duration, error)
	}

	type directRedisInterface interface {
		DirectRedisKeys(ctx context.Context, pattern string) ([]string, error)
		DirectRedisGet(ctx context.Context, key string) (string, error)
		DirectRedisTTL(ctx context.Context, key string) (time.Duration, error)
	}

	mlCache, ok := cacheInterface.(MultiLevelCacheInterface)
	if !ok {
		return []models.CacheInfo{}, nil
	}

	var keys []string
	var err error
	var useDirectRedis bool

	if req.Level == "l2" || req.Level == "L2" || req.Level == "all" {
		if directRedis, ok := cacheInterface.(directRedisInterface); ok {
			allRedisKeys, redisErr := directRedis.DirectRedisKeys(ctx, pattern)
			if redisErr == nil && len(allRedisKeys) > 0 {
				keys = allRedisKeys
				useDirectRedis = true
				logger.Debugf("[Cache] 使用直接Redis访问，找到 %d 个键", len(keys))
			}
		}
	}

	if !useDirectRedis {
		switch strings.ToLower(req.Level) {
		case "l1":
			keys, err = mlCache.KeysByLevel(ctx, pattern, "l1")
		case "l2":
			keys, err = mlCache.KeysByLevel(ctx, pattern, "l2")
		default:
			keys, err = mlCache.KeysByLevel(ctx, pattern, "all")
		}

		if err != nil {
			return []models.CacheInfo{}, nil
		}
	}

	var caches []models.CacheInfo
	for _, key := range keys {
		if isSystemKey(key) {
			continue
		}

		var value string
		var ttl time.Duration
		var getErr error

		if useDirectRedis {
			directRedis := cacheInterface.(directRedisInterface)
			value, getErr = directRedis.DirectRedisGet(ctx, key)
			if getErr == nil {
				ttl, _ = directRedis.DirectRedisTTL(ctx, key)
			}
		} else {
			value, getErr = mlCache.Get(ctx, key)
			if getErr == nil {
				ttl, _ = mlCache.TTL(ctx, key)
			}
		}

		if getErr != nil || value == "" {
			continue
		}

		ttlSeconds := int64(-1)
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		location := "l2"
		if req.Level == "l1" {
			location = "l1"
		} else if !useDirectRedis && req.Level == "all" {
			l1Keys, _ := mlCache.KeysByLevel(ctx, key, "l1")
			l2Keys, _ := mlCache.KeysByLevel(ctx, key, "l2")
			l1Has := len(l1Keys) > 0
			l2Has := len(l2Keys) > 0
			if l1Has && l2Has {
				location = "both"
			} else if l1Has {
				location = "l1"
			} else if l2Has {
				location = "l2"
			}
		}

		caches = append(caches, models.CacheInfo{
			Key:       key,
			Value:     value,
			Type:      "string",
			Size:      int64(len(value)),
			TTL:       ttlSeconds,
			Location:  location,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	return caches, nil
}

// isSystemKey 判断是否为系统键
func isSystemKey(key string) bool {
	systemKeyPrefixes := []string{
		"__:", "redis:", "monitor:", "perf:", "stats:",
		"cluster:", "node:", "replication:", "sentinel:",
	}

	lowerKey := strings.ToLower(key)
	for _, prefix := range systemKeyPrefixes {
		if strings.HasPrefix(lowerKey, prefix) {
			return true
		}
	}

	// 跳过纯数字键（通常是 Redis 内部键）
	if len(key) > 0 && key[0] >= '0' && key[0] <= '9' {
		return true
	}

	return false
}

// formatCacheStats 格式化缓存统计信息
// includeExtraInfo: 是否包含额外信息（cacheName, cacheType, collectTime）
func formatCacheStats(stats map[string]interface{}, cacheName string, includeExtraInfo bool) map[string]interface{} {
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

	result := map[string]interface{}{
		"hitCount":    hitCount,
		"missCount":   missCount,
		"hitRate":     hitRate,
		"totalMemory": totalMemory,
		"usedMemory":  usedMemory,
		"keyCount":    keyCount,
	}

	if includeExtraInfo {
		result["cacheName"] = cacheName
		result["cacheType"] = "memory"
		result["collectTime"] = time.Now()
	}

	return result
}

// formatCacheStatsForMonitor 格式化缓存统计信息（用于监控接口）
// 保留以向后兼容
func formatCacheStatsForMonitor(stats map[string]interface{}) map[string]interface{} {
	return formatCacheStats(stats, "", false)
}
