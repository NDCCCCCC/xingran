package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 覆盖 NewMetricsCacheManager 默认行为(无 redis) + L1 get/set + L2 miss +
// GetCacheStats / InvalidateCache / ClearL1Cache / Stop。
//
// 避免触发 startBackgroundTasks 的真实 system.GetSystemMetrics 路径(慢),
// 所有断言围绕缓存逻辑本身。

func TestNewMetricsCacheManager_NoRedis(t *testing.T) {
	m := NewMetricsCacheManager(nil)
	require.NotNil(t, m)
	require.NotNil(t, m.stopChan)
	assert.NotEmpty(t, m.hostname)
	// 启动后 L1 / L2 均为空
	stats := m.GetCacheStats()
	assert.Equal(t, 0, stats["l1_cache_size"])
	assert.Equal(t, false, stats["redis_enabled"])
	m.Stop()
}

func TestL1Cache_SetGetExpire(t *testing.T) {
	m := NewMetricsCacheManager(nil)
	defer m.Stop()

	key := m.getCacheKey("metrics:current")
	m.setToL1(key, "value-x", 100*time.Millisecond)

	// 立即读 → 命中
	v, ok := m.getFromL1(key)
	require.True(t, ok)
	assert.Equal(t, "value-x", v)

	// 过期 → miss + 删除
	time.Sleep(150 * time.Millisecond)
	v, ok = m.getFromL1(key)
	assert.False(t, ok)
	assert.Nil(t, v)

	// 再次读 → miss
	_, ok = m.getFromL1(key)
	assert.False(t, ok)
}

func TestL2Cache_NoRedis(t *testing.T) {
	m := NewMetricsCacheManager(nil)
	defer m.Stop()

	_, err := m.getFromL2(context.Background(), "any")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未初始化")

	err = m.setToL2(context.Background(), "any", "v", time.Minute)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未初始化")
}

func TestInvalidateCache_NoRedis(t *testing.T) {
	m := NewMetricsCacheManager(nil)
	defer m.Stop()

	key := m.getCacheKey("metrics:current")
	m.setToL1(key, "v", time.Minute)
	m.InvalidateCache(key)
	_, ok := m.getFromL1(key)
	assert.False(t, ok)
}

func TestClearL1Cache(t *testing.T) {
	m := NewMetricsCacheManager(nil)
	defer m.Stop()

	for i := 0; i < 3; i++ {
		m.setToL1(m.getCacheKey("k"+string(rune('0'+i))), i, time.Minute)
	}
	m.ClearL1Cache()
	stats := m.GetCacheStats()
	assert.Equal(t, 0, stats["l1_cache_size"])
}

func TestCacheItemStruct(t *testing.T) {
	item := CacheItem{
		Value:     "v",
		ExpiresAt: time.Now(),
	}
	assert.Equal(t, "v", item.Value)
}

func TestMetricsDataStruct(t *testing.T) {
	md := MetricsData{
		CPUUsage:    1.5,
		MemoryUsage: 2.5,
		DiskUsage:   3.5,
		NetworkRx:   10,
		NetworkTx:   20,
		ProcessNum:  100,
		TotalMemory: 1024,
		UsedMemory:  512,
	}
	assert.Equal(t, 1.5, md.CPUUsage)
	assert.Equal(t, uint64(1024), md.TotalMemory)
}