package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ==================== 缓存键命名规范迁移说明 ====================
//
// **状态 (P2-A2)**: 本文件 (data_cache_service.go) 已不再定义任何 CacheKey*
// 常量 — 缓存键定义已统一迁移到 internal/services/system/cache_keys.go,
// 该文件是项目内缓存键的 **单一真实来源 (single source of truth)**。
//
// **新规范**:
//   - 格式: cache:{module}:{type}:{params}
//   - 构造方式: system.BuildXxxCacheKey(keyType, params...) 或直接使用 system.CacheKey* 常量
//   - 优势: 统一前缀, 集中管理, 避免多套键定义之间的命名冲突
//
// **历史**: 此文件先前定义了 21 个 CacheKey* 常量与 8 个 Get*Key() 辅助函数。
// 这些常量/函数与 system/cache_keys.go 中的同名常量存在命名冲突,
// 现已全部移除以消除歧义。
//
// **迁移路径**:
//   - system/cache_keys.go 提供 BuildXxxCacheKey() 风格构造器
//   - 例如: services.GetDeptTreeKey()  → system.BuildDeptCacheKey("tree")
//   - 例如: services.CacheKeyUserByID   → system.BuildUserCacheKey("id") 或 system.CacheKeyUserByID (命名片段)
//
// ===========================================================

// DataCacheService 数据缓存服务
type DataCacheService struct {
	cache       cache.Cache
	cacheConfig *CacheConfigService
}

// NewDataCacheService 创建数据缓存服务
func NewDataCacheService(cache cache.Cache) *DataCacheService {
	return &DataCacheService{cache: cache}
}

// SetCacheConfig 设置缓存配置服务
func (s *DataCacheService) SetCacheConfig(cacheConfig *CacheConfigService) {
	s.cacheConfig = cacheConfig
}

// GetExpiration 获取缓存过期时间
func (s *DataCacheService) GetExpiration(configKey string, defaultExpiration time.Duration) time.Duration {
	if s.cacheConfig != nil {
		return s.cacheConfig.GetDurationWithDefault(configKey, defaultExpiration)
	}
	return defaultExpiration
}

// Get 获取缓存数据（自动反序列化）
func (s *DataCacheService) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("获取缓存失败: %w", err)
	}
	if data == "" {
		return apperrors.CacheKeyNotFound()
	}
	return json.Unmarshal([]byte(data), dest)
}

// Set 设置缓存数据（自动序列化）
func (s *DataCacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	return s.cache.Set(ctx, key, string(data), expiration)
}

// GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
func (s *DataCacheService) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, query func() (interface{}, error)) error {
	err := s.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	data, err := query()
	if err != nil {
		return fmt.Errorf("查询数据失败: %w", err)
	}

	// 序列化到目标变量
	dataBytes, _ := json.Marshal(data)
	_ = json.Unmarshal(dataBytes, dest)

	// P0 #9: 不再起裸 goroutine 异步写缓存。底层 cache（MultiLevelCache）的 Set
	// 已自带 L1 同步写入 + L2 经 L2Writer worker pool 异步写入（带重试/队列/降级）。
	// 早期此处的裸 go func() 因无并发上限，在高 QPS 下堆积 goroutine、耗尽 Redis
	// 连接（审查报告 #9）。改为同步写入后，开销仅为 L1 写 + L2 入队（微秒级），
	// 既消除 goroutine 堆积，又复用底层重试/降级能力。用 context.Background()
	// 避免请求提前取消导致缓存未写入。
	if err := s.Set(context.Background(), key, data, expiration); err != nil {
		applogger.Warnf("[CACHE] GetOrSet 缓存写入失败: key=%s, err=%v", key, err)
	}

	return nil
}

// Delete 删除缓存
func (s *DataCacheService) Delete(ctx context.Context, key string) error {
	return s.cache.Delete(ctx, key)
}

// DeleteByPattern 根据模式删除缓存
func (s *DataCacheService) DeleteByPattern(ctx context.Context, pattern string) error {
	keys, err := s.cache.Keys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("查询缓存键失败: %w", err)
	}
	if len(keys) > 0 {
		return s.cache.MDelete(ctx, keys...)
	}
	return nil
}

// MGet 批量获取缓存
func (s *DataCacheService) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	values, err := s.cache.MGet(ctx, keys...)
	if err != nil {
		return nil, fmt.Errorf("批量获取缓存失败: %w", err)
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
func (s *DataCacheService) MDelete(ctx context.Context, keys ...string) error {
	return s.cache.MDelete(ctx, keys...)
}

// Exists 检查缓存是否存在
func (s *DataCacheService) Exists(ctx context.Context, key string) (bool, error) {
	return s.cache.Exists(ctx, key)
}

// SetTTL 设置缓存过期时间
func (s *DataCacheService) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return s.cache.Expire(ctx, key, expiration)
}

// GetTTL 获取缓存过期时间
func (s *DataCacheService) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return s.cache.TTL(ctx, key)
}

// GetStats 获取缓存统计信息
func (s *DataCacheService) GetStats(ctx context.Context) (*CacheStats, error) {
	// 返回基本统计信息
	stats := &CacheStats{}
	if keys, err := s.cache.Keys(ctx, "*"); err == nil {
		stats.KeyCount = len(keys)
		stats.Count = int64(len(keys))
	}
	return stats, nil
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits          int64                  // 命中次数
	Misses        int64                  // 未命中次数
	Count         int64                  // 缓存项数量
	MemorySize    int64                  // 内存占用（字节）
	HitRate       float64                // 命中率
	KeyCount      int                    // 键数量
	ExtendedStats map[string]interface{} // 扩展统计
}

// ==================== 缓存键构建器 (CacheKeyBuilder) ====================
//
// 注意: CacheKeyBuilder 是 data_cache_service 提供的 **通用键构造工具**,
// 不与 system.CacheKey* 冲突,继续保留。它接受任意 prefix 和 params 拼接键。

// CacheKeyBuilder 缓存键构建器
type CacheKeyBuilder struct {
	prefix string
}

// NewCacheKeyBuilder 创建缓存键构建器
func NewCacheKeyBuilder(prefix string) *CacheKeyBuilder {
	return &CacheKeyBuilder{prefix: prefix}
}

// Build 构建缓存键
// 遵循 Go 最佳实践：使用 strings.Builder 避免循环中的多次内存分配
func (b *CacheKeyBuilder) Build(params ...interface{}) string {
	if len(params) == 0 {
		return b.prefix
	}

	// 预分配容量：前缀长度 + (参数数量 * (1个冒号 + 平均16字符))
	var sb strings.Builder
	sb.Grow(len(b.prefix) + len(params)*17)
	sb.WriteString(b.prefix)

	for _, p := range params {
		sb.WriteByte(':')
		// 使用类型断言进行更高效的类型转换
		// 注意：需要按具体类型断言，而不是使用逗号分隔的类型列表
		switch v := p.(type) {
		case string:
			sb.WriteString(v)
		case int:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int8:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int16:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int32:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case int64:
			sb.WriteString(strconv.FormatInt(int64(v), 10))
		case uint:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint8:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint16:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint32:
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		case uint64:
			sb.WriteString(strconv.FormatUint(v, 10))
		case float32:
			sb.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 64))
		case float64:
			sb.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			if v {
				sb.WriteString("true")
			} else {
				sb.WriteString("false")
			}
		default:
			// 对于其他类型，回退到 fmt.Sprintf
			sb.WriteString(fmt.Sprintf("%v", v))
		}
	}

	return sb.String()
}
