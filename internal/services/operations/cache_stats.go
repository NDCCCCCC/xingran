package operations

import (
	"sync/atomic"
)

// CacheStats 缓存统计
type CacheStats struct {
	memoryHits  uint64
	redisHits   uint64
	apiCalls    uint64
	cacheMisses uint64
}

// RecordMemoryHit 记录内存缓存命中
func (s *CacheStats) RecordMemoryHit() {
	atomic.AddUint64(&s.memoryHits, 1)
}

// RecordRedisHit 记录 Redis 缓存命中
func (s *CacheStats) RecordRedisHit() {
	atomic.AddUint64(&s.redisHits, 1)
}

// RecordAPICall 记录 API 调用
func (s *CacheStats) RecordAPICall() {
	atomic.AddUint64(&s.apiCalls, 1)
}

// RecordMiss 记录缓存未命中
func (s *CacheStats) RecordMiss() {
	atomic.AddUint64(&s.cacheMisses, 1)
}

// GetStats 获取统计数据
func (s *CacheStats) GetStats() CacheStatsData {
	memoryHits := atomic.LoadUint64(&s.memoryHits)
	redisHits := atomic.LoadUint64(&s.redisHits)
	apiCalls := atomic.LoadUint64(&s.apiCalls)
	cacheMisses := atomic.LoadUint64(&s.cacheMisses)

	return CacheStatsData{
		MemoryHits:    memoryHits,
		RedisHits:     redisHits,
		APICalls:      apiCalls,
		CacheMisses:   cacheMisses,
		TotalHits:     memoryHits + redisHits,
		TotalRequests: memoryHits + redisHits + apiCalls,
	}
}

// Reset 重置统计
func (s *CacheStats) Reset() {
	s.resetAllFields(
		&s.memoryHits,
		&s.redisHits,
		&s.apiCalls,
		&s.cacheMisses,
	)
}

// resetAllFields 重置所有字段
func (s *CacheStats) resetAllFields(fields ...*uint64) {
	for _, field := range fields {
		atomic.StoreUint64(field, 0)
	}
}

// CacheStatsData 缓存统计数据
type CacheStatsData struct {
	MemoryHits    uint64 `json:"memoryHits"`
	RedisHits     uint64 `json:"redisHits"`
	APICalls      uint64 `json:"apiCalls"`
	CacheMisses   uint64 `json:"cacheMisses"`
	TotalHits     uint64 `json:"totalHits"`
	TotalRequests uint64 `json:"totalRequests"`
}

// GetHitRate 获取命中率
func (d *CacheStatsData) GetHitRate() float64 {
	if d.TotalRequests == 0 {
		return 0
	}
	return float64(d.TotalHits) / float64(d.TotalRequests) * 100
}
