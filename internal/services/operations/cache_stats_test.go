package operations

import (
	"sync"
	"testing"
)

func TestCacheStats_RecordMemoryHit(t *testing.T) {
	stats := &CacheStats{}

	// 记录多次命中
	for i := 0; i < 5; i++ {
		stats.RecordMemoryHit()
	}

	data := stats.GetStats()
	if data.MemoryHits != 5 {
		t.Errorf("MemoryHits = %v, want 5", data.MemoryHits)
	}
}

func TestCacheStats_RecordRedisHit(t *testing.T) {
	stats := &CacheStats{}

	for i := 0; i < 3; i++ {
		stats.RecordRedisHit()
	}

	data := stats.GetStats()
	if data.RedisHits != 3 {
		t.Errorf("RedisHits = %v, want 3", data.RedisHits)
	}
}

func TestCacheStats_RecordAPICall(t *testing.T) {
	stats := &CacheStats{}

	for i := 0; i < 7; i++ {
		stats.RecordAPICall()
	}

	data := stats.GetStats()
	if data.APICalls != 7 {
		t.Errorf("APICalls = %v, want 7", data.APICalls)
	}
}

func TestCacheStats_RecordMiss(t *testing.T) {
	stats := &CacheStats{}

	for i := 0; i < 2; i++ {
		stats.RecordMiss()
	}

	data := stats.GetStats()
	if data.CacheMisses != 2 {
		t.Errorf("CacheMisses = %v, want 2", data.CacheMisses)
	}
}

func TestCacheStats_GetStats(t *testing.T) {
	stats := &CacheStats{}

	stats.RecordMemoryHit()
	stats.RecordMemoryHit()
	stats.RecordRedisHit()
	stats.RecordAPICall()
	stats.RecordMiss()

	data := stats.GetStats()

	if data.MemoryHits != 2 {
		t.Errorf("MemoryHits = %v, want 2", data.MemoryHits)
	}
	if data.RedisHits != 1 {
		t.Errorf("RedisHits = %v, want 1", data.RedisHits)
	}
	if data.APICalls != 1 {
		t.Errorf("APICalls = %v, want 1", data.APICalls)
	}
	if data.CacheMisses != 1 {
		t.Errorf("CacheMisses = %v, want 1", data.CacheMisses)
	}
	if data.TotalHits != 3 {
		t.Errorf("TotalHits = %v, want 3", data.TotalHits)
	}
	if data.TotalRequests != 4 {
		t.Errorf("TotalRequests = %v, want 4", data.TotalRequests)
	}
}

func TestCacheStatsData_GetHitRate(t *testing.T) {
	tests := []struct {
		name         string
		data         CacheStatsData
		expectedRate float64
	}{
		{
			name: "零请求",
			data: CacheStatsData{
				MemoryHits:    0,
				RedisHits:     0,
				APICalls:      0,
				TotalHits:     0,
				TotalRequests: 0,
			},
			expectedRate: 0,
		},
		{
			name: "100% 命中率",
			data: CacheStatsData{
				MemoryHits:    8,
				RedisHits:     2,
				APICalls:      0,
				TotalHits:     10,
				TotalRequests: 10,
			},
			expectedRate: 100.0,
		},
		{
			name: "50% 命中率",
			data: CacheStatsData{
				MemoryHits:    3,
				RedisHits:     2,
				APICalls:      5,
				TotalHits:     5,
				TotalRequests: 10,
			},
			expectedRate: 50.0,
		},
		{
			name: "33.33% 命中率",
			data: CacheStatsData{
				MemoryHits:    1,
				RedisHits:     2,
				APICalls:      6,
				TotalHits:     3,
				TotalRequests: 9,
			},
			expectedRate: 33.33333333333333,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := tt.data.GetHitRate()
			if rate != tt.expectedRate {
				t.Errorf("GetHitRate() = %v, want %v", rate, tt.expectedRate)
			}
		})
	}
}

func TestCacheStats_Reset(t *testing.T) {
	stats := &CacheStats{}

	// 添加一些数据
	stats.RecordMemoryHit()
	stats.RecordRedisHit()
	stats.RecordAPICall()
	stats.RecordMiss()

	// 验证数据已添加
	data := stats.GetStats()
	if data.MemoryHits == 0 || data.RedisHits == 0 {
		t.Error("数据未正确添加")
	}

	// 重置
	stats.Reset()

	// 验证重置后数据
	data = stats.GetStats()
	if data.MemoryHits != 0 {
		t.Errorf("MemoryHits after reset = %v, want 0", data.MemoryHits)
	}
	if data.RedisHits != 0 {
		t.Errorf("RedisHits after reset = %v, want 0", data.RedisHits)
	}
	if data.APICalls != 0 {
		t.Errorf("APICalls after reset = %v, want 0", data.APICalls)
	}
	if data.CacheMisses != 0 {
		t.Errorf("CacheMisses after reset = %v, want 0", data.CacheMisses)
	}
}

func TestCacheStats_Concurrent(t *testing.T) {
	stats := &CacheStats{}
	var wg sync.WaitGroup

	// 并发记录
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			stats.RecordMemoryHit()
		}()
		go func() {
			defer wg.Done()
			stats.RecordRedisHit()
		}()
		go func() {
			defer wg.Done()
			stats.RecordAPICall()
		}()
		go func() {
			defer wg.Done()
			stats.RecordMiss()
		}()
	}

	wg.Wait()

	data := stats.GetStats()
	if data.MemoryHits != 100 {
		t.Errorf("MemoryHits = %v, want 100", data.MemoryHits)
	}
	if data.RedisHits != 100 {
		t.Errorf("RedisHits = %v, want 100", data.RedisHits)
	}
	if data.APICalls != 100 {
		t.Errorf("APICalls = %v, want 100", data.APICalls)
	}
	if data.CacheMisses != 100 {
		t.Errorf("CacheMisses = %v, want 100", data.CacheMisses)
	}
}
