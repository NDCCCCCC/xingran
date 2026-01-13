// Package crypto 提供高并发场景下的 Nonce 存储实现
package crypto

import (
	"hash/fnv"
	"sync"
	"time"
)

// shardCount 分片数量，建议使用质数或 2 的幂
const shardCount = 256

// shard 单个分片，包含自己的 map 和锁
type shard struct {
	nonces map[string]int64
	mu     sync.RWMutex
}

// shardedNonceStorage 分段锁的 nonce 存储
// 通过 hash 将 nonce 分散到不同的分片中，减少锁竞争
type shardedNonceStorage struct {
	shards [shardCount]*shard
}

// NewShardedNonceStorage 创建分段锁的 nonce 存储（推荐用于高并发场景）
func NewShardedNonceStorage() NonceStorage {
	s := &shardedNonceStorage{}
	for i := 0; i < shardCount; i++ {
		s.shards[i] = &shard{
			nonces: make(map[string]int64),
		}
	}
	return s
}

// getShard 根据 nonce 计算 hash 并返回对应的分片
func (s *shardedNonceStorage) getShard(nonce string) *shard {
	h := fnv.New32a()
	h.Write([]byte(nonce))
	return s.shards[h.Sum32()%shardCount]
}

// CheckAndStore 检查并存储 nonce
func (s *shardedNonceStorage) CheckAndStore(nonce string, timestamp int64) bool {
	shard := s.getShard(nonce)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, exists := shard.nonces[nonce]; exists {
		return false
	}
	shard.nonces[nonce] = timestamp
	return true
}

// cleanupExpiredNonces 清理过期的 nonce
func (s *shardedNonceStorage) cleanupExpiredNonces() {
	now := time.Now().Unix()

	// 并发清理所有分片
	var wg sync.WaitGroup
	for i := 0; i < shardCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			shard := s.shards[idx]
			shard.mu.Lock()
			defer shard.mu.Unlock()

			for nonce, ts := range shard.nonces {
				if now-ts > maxTimeDiff*2 {
					delete(shard.nonces, nonce)
				}
			}
		}(i)
	}
	wg.Wait()
}

// syncMapNonceStorage 使用 sync.Map 的 nonce 存储
// 适合读多写少的场景
type syncMapNonceStorage struct {
	nonces sync.Map
}

// NewSyncMapNonceStorage 创建基于 sync.Map 的 nonce 存储
func NewSyncMapNonceStorage() NonceStorage {
	return &syncMapNonceStorage{}
}

// CheckAndStore 检查并存储 nonce
func (s *syncMapNonceStorage) CheckAndStore(nonce string, timestamp int64) bool {
	if _, exists := s.nonces.Load(nonce); exists {
		return false
	}
	s.nonces.Store(nonce, timestamp)
	return true
}

// cleanupExpiredNonces 清理过期的 nonce
func (s *syncMapNonceStorage) cleanupExpiredNonces() {
	now := time.Now().Unix()
	s.nonces.Range(func(key, value interface{}) bool {
		if ts, ok := value.(int64); ok {
			if now-ts > maxTimeDiff*2 {
				s.nonces.Delete(key)
			}
		}
		return true
	})
}

// GetNonceCount 获取当前存储的 nonce 数量（用于监控）
func (s *shardedNonceStorage) GetNonceCount() int {
	count := 0
	for i := 0; i < shardCount; i++ {
		s.shards[i].mu.RLock()
		count += len(s.shards[i].nonces)
		s.shards[i].mu.RUnlock()
	}
	return count
}

// GetNonceCount 获取当前存储的 nonce 数量（用于监控）
func (s *syncMapNonceStorage) GetNonceCount() int {
	count := 0
	s.nonces.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
