// Package crypto 提供 nonce 存储的基准测试
package crypto

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

// 生成测试用的 nonce
func generateNonce(i int) string {
	return "nonce-" + strconv.Itoa(i)
}

// 基准测试：分段锁
func BenchmarkShardedNonceStorage_CheckAndStore(b *testing.B) {
	storage := NewShardedNonceStorage()
	now := time.Now().Unix()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			storage.CheckAndStore(generateNonce(i), now)
			i++
		}
	})
}

// 基准测试：sync.Map
func BenchmarkSyncMapNonceStorage_CheckAndStore(b *testing.B) {
	storage := NewSyncMapNonceStorage()
	now := time.Now().Unix()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			storage.CheckAndStore(generateNonce(i), now)
			i++
		}
	})
}

// 基准测试：全局锁（旧的实现）
func BenchmarkDefaultNonceStorage_CheckAndStore(b *testing.B) {
	storage := &defaultNonceStorage{
		nonces: make(map[string]int64),
	}
	now := time.Now().Unix()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			storage.CheckAndStore(generateNonce(i), now)
			i++
		}
	})
}

// 并发测试：模拟真实的高并发场景
func BenchmarkHighConcurrencyScenario(b *testing.B) {
	b.Run("ShardedMap", func(b *testing.B) {
		storage := NewShardedNonceStorage()
		now := time.Now().Unix()
		b.ResetTimer()

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < b.N/100; j++ {
					nonce := generateNonce(workerID*1000000 + j)
					storage.CheckAndStore(nonce, now)
				}
			}(i)
		}
		wg.Wait()
	})

	b.Run("GlobalLock", func(b *testing.B) {
		storage := &defaultNonceStorage{
			nonces: make(map[string]int64),
		}
		now := time.Now().Unix()
		b.ResetTimer()

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < b.N/100; j++ {
					nonce := generateNonce(workerID*1000000 + j)
					storage.CheckAndStore(nonce, now)
				}
			}(i)
		}
		wg.Wait()
	})
}

// 重复 nonce 检测测试
func BenchmarkDuplicateNonceDetection(b *testing.B) {
	storage := NewShardedNonceStorage()
	now := time.Now().Unix()
	testNonce := "test-duplicate-nonce"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 第一次会成功，后续都会被拒绝
			storage.CheckAndStore(testNonce, now)
		}
	})
}

// 清理过期 nonce 测试
func BenchmarkCleanupExpiredNonces(b *testing.B) {
	storage := NewShardedNonceStorage()
	now := time.Now().Unix()

	// 预先填充 10000 个 nonce
	for i := 0; i < 10000; i++ {
		storage.CheckAndStore(generateNonce(i), now-1000) // 过期的 nonce
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if sharded, ok := storage.(*shardedNonceStorage); ok {
			sharded.cleanupExpiredNonces()
		}
	}
}

// 内存分配测试
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("ShardedMap", func(b *testing.B) {
		b.ReportAllocs()
		storage := NewShardedNonceStorage()
		now := time.Now().Unix()

		for i := 0; i < b.N; i++ {
			storage.CheckAndStore(generateNonce(i), now)
		}
	})

	b.Run("SyncMap", func(b *testing.B) {
		b.ReportAllocs()
		storage := NewSyncMapNonceStorage()
		now := time.Now().Unix()

		for i := 0; i < b.N; i++ {
			storage.CheckAndStore(generateNonce(i), now)
		}
	})
}
