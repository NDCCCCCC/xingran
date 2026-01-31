package operations

import (
	"sync/atomic"
)

type CacheStats struct {
	memoryHits  uint64
	redisHits   uint64
	apiCalls    uint64
	cacheMisses uint64
}

func (s *CacheStats) RecordMemoryHit() {
	atomic.AddUint64(&s.memoryHits, 1)
}

func (s *CacheStats) RecordRedisHit() {
	atomic.AddUint64(&s.redisHits, 1)
}

func (s *CacheStats) RecordAPICall() {
	atomic.AddUint64(&s.apiCalls, 1)
}

func (s *CacheStats) RecordMiss() {
	atomic.AddUint64(&s.cacheMisses, 1)
}

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

func (s *CacheStats) Reset() {
	atomic.StoreUint64(&s.memoryHits, 0)
	atomic.StoreUint64(&s.redisHits, 0)
	atomic.StoreUint64(&s.apiCalls, 0)
	atomic.StoreUint64(&s.cacheMisses, 0)
}

type CacheStatsData struct {
	MemoryHits    uint64 `json:"memoryHits"`
	RedisHits     uint64 `json:"redisHits"`
	APICalls      uint64 `json:"apiCalls"`
	CacheMisses   uint64 `json:"cacheMisses"`
	TotalHits     uint64 `json:"totalHits"`
	TotalRequests uint64 `json:"totalRequests"`
}

func (d *CacheStatsData) GetHitRate() float64 {
	if d.TotalRequests == 0 {
		return 0
	}
	return float64(d.TotalHits) / float64(d.TotalRequests) * 100
}
