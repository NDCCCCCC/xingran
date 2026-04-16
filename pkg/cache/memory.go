package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryCache 内存缓存实现
type MemoryCache struct {
	items    map[string]*CacheItem
	mutex    sync.RWMutex
	maxSize  int
	stopChan chan struct{}
	hits     int64 // 命中次数
	misses   int64 // 未命中次数
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache(maxSize int, cleanupTime time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items:    make(map[string]*CacheItem),
		maxSize:  maxSize,
		stopChan: make(chan struct{}),
	}

	// 启动清理协程
	go cache.startCleanup(cleanupTime)

	return cache
}

// Get 获取缓存值
func (m *MemoryCache) Get(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", ErrKeyEmpty
	}

	m.mutex.RLock()
	item, exists := m.items[key]
	m.mutex.RUnlock()

	if !exists {
		atomic.AddInt64(&m.misses, 1)
		return "", ErrNotFound
	}

	if item.IsExpired() {
		atomic.AddInt64(&m.misses, 1)
		return "", ErrExpired
	}

	atomic.AddInt64(&m.hits, 1)

	// 类型断言
	switch v := item.Value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// Set 设置缓存值
func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if key == "" {
		return ErrKeyEmpty
	}
	if value == nil {
		return ErrValueEmpty
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查缓存大小
	if len(m.items) >= m.maxSize {
		m.evictLRU()
	}

	var expirationTime int64
	if expiration > 0 {
		expirationTime = time.Now().Add(expiration).UnixNano()
	}

	m.items[key] = &CacheItem{
		Value:      value,
		Expiration: expirationTime,
		Created:    time.Now(),
	}

	return nil
}

// Delete 删除缓存
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrKeyEmpty
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.items, key)
	return nil
}

// Exists 检查键是否存在
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, ErrKeyEmpty
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, exists := m.items[key]
	if !exists {
		return false, nil
	}

	return !item.IsExpired(), nil
}

// MGet 批量获取
func (m *MemoryCache) MGet(ctx context.Context, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	values := make([]string, len(keys))
	for i, key := range keys {
		value, err := m.Get(ctx, key)
		if err != nil {
			values[i] = ""
		} else {
			values[i] = value
		}
	}

	return values, nil
}

// MSet 批量设置
func (m *MemoryCache) MSet(ctx context.Context, pairs ...interface{}) error {
	if len(pairs)%2 != 0 {
		return fmt.Errorf("参数数量必须为偶数")
	}

	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return fmt.Errorf("键必须是字符串类型")
		}
		value := pairs[i+1]

		if err := m.Set(ctx, key, value, 0); err != nil {
			return err
		}
	}

	return nil
}

// MDelete 批量删除
func (m *MemoryCache) MDelete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		if err := m.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Increment 递增
func (m *MemoryCache) Increment(ctx context.Context, key string) (int64, error) {
	return m.IncrementBy(ctx, key, 1)
}

// IncrementBy 按值递增
func (m *MemoryCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	item, exists := m.items[key]
	var currentValue int64

	if exists && !item.IsExpired() {
		// 尝试转换当前值为整数
		switch v := item.Value.(type) {
		case int:
			currentValue = int64(v)
		case int64:
			currentValue = v
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				currentValue = parsed
			}
		default:
			return 0, ErrInvalidType
		}
	}

	currentValue += value

	// 更新值
	m.items[key] = &CacheItem{
		Value:      currentValue,
		Expiration: item.Expiration,
		Created:    time.Now(),
	}

	return currentValue, nil
}

// Decrement 递减
func (m *MemoryCache) Decrement(ctx context.Context, key string) (int64, error) {
	return m.IncrementBy(ctx, key, -1)
}

// DecrementBy 按值递减
func (m *MemoryCache) DecrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return m.IncrementBy(ctx, key, -value)
}

// Expire 设置过期时间
func (m *MemoryCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if key == "" {
		return ErrKeyEmpty
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	item, exists := m.items[key]
	if !exists {
		return ErrNotFound
	}

	var expirationTime int64
	if expiration > 0 {
		expirationTime = time.Now().Add(expiration).UnixNano()
	}

	item.Expiration = expirationTime
	return nil
}

// TTL 获取剩余时间
func (m *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if key == "" {
		return 0, ErrKeyEmpty
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, exists := m.items[key]
	if !exists {
		return 0, ErrNotFound
	}

	if item.Expiration == 0 {
		return -1, nil // 永不过期
	}

	remaining := time.Duration(item.Expiration - time.Now().UnixNano())
	if remaining <= 0 {
		return 0, ErrExpired
	}

	return remaining, nil
}

// Keys 模式匹配（简单实现）
func (m *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var keys []string
	for key := range m.items {
		// 简单的通配符匹配
		if matchPattern(key, pattern) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// FlushDB 清空数据库
func (m *MemoryCache) FlushDB(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.items = make(map[string]*CacheItem)
	return nil
}

// Close 关闭缓存
func (m *MemoryCache) Close() error {
	close(m.stopChan)
	return nil
}

// MGetJSON 批量获取JSON对象
func (m *MemoryCache) MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]interface{})
	for _, key := range keys {
		if item, exists := m.items[key]; exists && !item.IsExpired() {
			strValue, ok := item.Value.(string)
			if !ok {
				result[key] = item.Value
				continue
			}
			var value interface{}
			if err := json.Unmarshal([]byte(strValue), &value); err == nil {
				result[key] = value
			} else {
				result[key] = strValue
			}
		}
	}

	return result, nil
}

// MSetJSON 批量设置JSON对象
func (m *MemoryCache) MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range data {
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return err
		}

		var expirationTime int64
		if expiration > 0 {
			expirationTime = time.Now().Add(expiration).UnixNano()
		}

		// 检查容量限制
		if len(m.items) >= m.maxSize {
			m.evictLRU()
		}

		m.items[key] = &CacheItem{
			Value:      string(jsonValue),
			Expiration: expirationTime,
			Created:    time.Now(),
		}
	}

	return nil
}

// HGet 获取Hash字段（内存缓存简化实现）
func (m *MemoryCache) HGet(ctx context.Context, key, field string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if item, exists := m.items[key]; exists && !item.IsExpired() {
		// 安全地获取值
		strValue, ok := item.Value.(string)
		if !ok {
			return "", fmt.Errorf("key or field not found")
		}

		// 简化实现：将Hash存储为JSON对象
		var hashMap map[string]string
		if err := json.Unmarshal([]byte(strValue), &hashMap); err == nil {
			if val, exists := hashMap[field]; exists {
				return val, nil
			}
		}
	}

	return "", ErrNotFound
}

// HSet 设置Hash字段（内存缓存简化实现）
func (m *MemoryCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var hashMap map[string]string
	var expirationTime int64

	// 获取现有Hash
	if item, exists := m.items[key]; exists && !item.IsExpired() {
		if strValue, ok := item.Value.(string); ok {
			if err := json.Unmarshal([]byte(strValue), &hashMap); err != nil {
				hashMap = make(map[string]string)
			}
			expirationTime = item.Expiration
		} else {
			hashMap = make(map[string]string)
		}
	} else {
		hashMap = make(map[string]string)
	}

	// 设置字段值
	if strValue, ok := value.(string); ok {
		hashMap[field] = strValue
	} else {
		jsonValue, _ := json.Marshal(value)
		hashMap[field] = string(jsonValue)
	}

	// 存储回缓存
	jsonValue, err := json.Marshal(hashMap)
	if err != nil {
		return err
	}

	m.items[key] = &CacheItem{
		Value:      string(jsonValue),
		Expiration: expirationTime,
		Created:    time.Now(),
	}

	return nil
}

// HGetAll 获取Hash所有字段
func (m *MemoryCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if item, exists := m.items[key]; exists && !item.IsExpired() {
		strValue, ok := item.Value.(string)
		if !ok {
			return make(map[string]string), nil
		}
		var hashMap map[string]string
		if err := json.Unmarshal([]byte(strValue), &hashMap); err == nil {
			return hashMap, nil
		}
	}

	return make(map[string]string), nil
}

// HDel 删除Hash字段
func (m *MemoryCache) HDel(ctx context.Context, key string, fields ...string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if item, exists := m.items[key]; exists && !item.IsExpired() {
		strValue, ok := item.Value.(string)
		if !ok {
			return nil
		}
		var hashMap map[string]string
		if err := json.Unmarshal([]byte(strValue), &hashMap); err == nil {
			for _, field := range fields {
				delete(hashMap, field)
			}

			// 如果Hash为空，删除整个key
			if len(hashMap) == 0 {
				delete(m.items, key)
			} else {
				jsonValue, _ := json.Marshal(hashMap)
				m.items[key] = &CacheItem{
					Value:      string(jsonValue),
					Expiration: item.Expiration,
					Created:    item.Created,
				}
			}
		}
	}

	return nil
}

// HKeys 获取Hash所有字段名
func (m *MemoryCache) HKeys(ctx context.Context, key string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if item, exists := m.items[key]; exists && !item.IsExpired() {
		var hashMap map[string]string
		if err := json.Unmarshal([]byte(item.Value.(string)), &hashMap); err == nil {
			keys := make([]string, 0, len(hashMap))
			for field := range hashMap {
				keys = append(keys, field)
			}
			return keys, nil
		}
	}

	return []string{}, nil
}

// startCleanup 启动清理协程
func (m *MemoryCache) startCleanup(interval time.Duration) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopChan:
			return
		}
	}
}

// cleanup 清理过期项
func (m *MemoryCache) cleanup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, item := range m.items {
		if item.IsExpired() {
			delete(m.items, key)
		}
	}
}

// evictLRU LRU淘汰策略
func (m *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range m.items {
		if oldestKey == "" || item.Created.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.Created
		}
	}

	if oldestKey != "" {
		delete(m.items, oldestKey)
	}
}

// matchPattern 简单的模式匹配
func matchPattern(key, pattern string) bool {
	// 这里实现一个简单的通配符匹配
	// 只支持 * 通配符
	if pattern == "*" {
		return true
	}

	// 简单的前缀匹配
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return key == pattern
}

// GetJSON 获取JSON对象
func (m *MemoryCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := m.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// SetJSON 设置JSON对象
func (m *MemoryCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.Set(ctx, key, data, expiration)
}

// GetInt 获取整数
func (m *MemoryCache) GetInt(ctx context.Context, key string) (int, error) {
	data, err := m.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(data)
}

// SetInt 设置整数
func (m *MemoryCache) SetInt(ctx context.Context, key string, value int, expiration time.Duration) error {
	return m.Set(ctx, key, strconv.Itoa(value), expiration)
}

// GetBool 获取布尔值
func (m *MemoryCache) GetBool(ctx context.Context, key string) (bool, error) {
	data, err := m.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(data)
}

// SetBool 设置布尔值
func (m *MemoryCache) SetBool(ctx context.Context, key string, value bool, expiration time.Duration) error {
	var strValue string
	if value {
		strValue = "1"
	} else {
		strValue = "0"
	}
	return m.Set(ctx, key, strValue, expiration)
}

// GetStats 获取内存缓存统计信息
func (m *MemoryCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]interface{})

	// 统计键的数量
	stats["cache_type"] = "memory"
	stats["key_count"] = int64(len(m.items))

	// 估算内存使用
	totalSize := 0
	for key, item := range m.items {
		totalSize += len(key)
		if str, ok := item.Value.(string); ok {
			totalSize += len(str)
		}
	}
	stats["used_memory"] = int64(totalSize)
	stats["used_memory_rss"] = int64(totalSize)
	stats["maxmemory"] = int64(m.maxSize * 1024) // 估算最大内存
	stats["total_system_memory"] = int64(m.maxSize * 1024)

	// 使用真实的命中统计
	hits := atomic.LoadInt64(&m.hits)
	misses := atomic.LoadInt64(&m.misses)
	stats["keyspace_hits"] = hits
	stats["keyspace_misses"] = misses

	// 计算命中率
	if hits+misses > 0 {
		hitRate := float64(hits) / float64(hits+misses) * 100
		stats["hit_rate"] = hitRate
	} else {
		stats["hit_rate"] = 0.0
	}

	// Redis版本（内存缓存没有版本）
	stats["redis_version"] = "memory"

	return stats, nil
}
