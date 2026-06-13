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
	shards         [shardCount]*shard
	replayWindowSec int
}

// NewShardedNonceStorage 创建分段锁的 nonce 存储（推荐用于高并发场景）
//
// P1 fix: 启动后台清理 goroutine 定期调用 cleanupExpiredNonces,
// 之前 cleanup 函数虽已定义但从未被调用,导致内存版 nonce 表无限增长 —
// 长跑进程几天内会因为累积的过期 nonce 占用大量内存。
// 清理周期 = replayWindowSec (默认 60s, P1-S2 收紧),与 anti-replay 窗口对齐。
// 过期阈值 = 2 * replayWindowSec,允许一个完整窗口的 clock skew + 处理延迟。
func NewShardedNonceStorage() NonceStorage {
	return NewShardedNonceStorageWithConfig(DefaultReplayWindowSec)
}

// NewShardedNonceStorageWithConfig 使用自定义窗口创建分段锁 nonce 存储
//
// P1 fix (P1-S2): 接受可配置窗口,与 RequestEncryptor.replayWindowSec 保持同步,
// 避免清理周期与验证窗口脱节。
// 窗口必须 > 0,否则 fallback 到 DefaultReplayWindowSec。
func NewShardedNonceStorageWithConfig(replayWindowSec int) NonceStorage {
	if replayWindowSec <= 0 {
		replayWindowSec = DefaultReplayWindowSec
	}
	s := &shardedNonceStorage{
		replayWindowSec: replayWindowSec,
	}
	for i := 0; i < shardCount; i++ {
		s.shards[i] = &shard{
			nonces: make(map[string]int64),
		}
	}

	// 启动后台清理 goroutine。本进程单例使用,进程退出时随之结束;
	// 加 panic recover 防御任何 map 迭代时的隐性错误。
	go func() {
		defer func() {
			_ = recover()
		}()
		ticker := time.NewTicker(time.Duration(replayWindowSec) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanupExpiredNonces()
		}
	}()

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
//
// P1 fix (P1-S2): 使用 s.replayWindowSec * 2 作为过期阈值,
// 与 anti-replay 窗口保持同步 — 留 1 个完整窗口的 buffer
// (clock skew + 网络/处理延迟),保证已被验证的 nonce 不会
// 在验证窗口内被误清理。
func (s *shardedNonceStorage) cleanupExpiredNonces() {
	now := time.Now().Unix()
	threshold := int64(s.replayWindowSec) * 2

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
				if now-ts > threshold {
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
