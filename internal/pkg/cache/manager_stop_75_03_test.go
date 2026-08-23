package cache

import (
	"sync"
	"testing"
)

func TestMetricsCacheManager_Stop_Idempotent(t *testing.T) {
	m := NewMetricsCacheManager(nil)

	// 多次停止不应 panic
	m.Stop()
	m.Stop()
	m.Stop()
}

func TestMetricsCacheManager_Stop_Concurrent(t *testing.T) {
	m := NewMetricsCacheManager(nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Stop()
		}()
	}
	wg.Wait()
}
