package lldp

import (
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// LLDPCacheEntry LLDP缓存条目
type LLDPCacheEntry struct {
	Neighbors   map[string]*models.LLDPNeighborInfo
	CachedAt    time.Time
}

// LLDPCache LLDP缓存服务
type LLDPCache struct {
	cache map[string]*LLDPCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewLLDPCache 创建LLDP缓存
func NewLLDPCache(ttl time.Duration) *LLDPCache {
	return &LLDPCache{
		cache: make(map[string]*LLDPCacheEntry),
		ttl:   ttl,
	}
}

// Get 获取缓存的邻居信息
func (c *LLDPCache) Get(deviceID string) (map[string]*models.LLDPNeighborInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.cache[deviceID]
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Since(entry.CachedAt) > c.ttl {
		return nil, false
	}

	return entry.Neighbors, true
}

// Set 设置缓存
func (c *LLDPCache) Set(deviceID string, neighbors map[string]*models.LLDPNeighborInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[deviceID] = &LLDPCacheEntry{
		Neighbors: neighbors,
		CachedAt:  time.Now(),
	}
}

// Delete 删除缓存
func (c *LLDPCache) Delete(deviceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, deviceID)
}
