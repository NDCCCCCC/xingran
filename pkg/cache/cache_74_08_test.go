package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// 74-08: pkg/cache — MemoryCache 全 API 面(Get/Set/M*/Hash/JSON/TTL/
// LRU/cleanup/统计)+ errors.go 谓词 + CacheItem.IsExpired。
// Redis 实现依赖真实 Redis,不在单测范围(D-12 禁加 miniredis 依赖)。
// =====================================================================

func newMem(t *testing.T, maxSize int) *MemoryCache {
	t.Helper()
	m := NewMemoryCache(maxSize, 0) // interval<=0 → 不启清理协程(测试手工触发)
	t.Cleanup(func() { _ = m.Close() })
	return m
}

// ---------------- 基础 Get/Set ----------------

func TestMemoryCache_GetSet(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	// 空 key → ErrKeyEmpty
	_, err := m.Get(ctx, "")
	assert.ErrorIs(t, err, ErrKeyEmpty)
	assert.Error(t, m.Set(ctx, "", "v", 0))

	// nil value → ErrValueEmpty
	assert.ErrorIs(t, m.Set(ctx, "k", nil, 0), ErrValueEmpty)

	// string / []byte / int 的 Get 类型断言
	require.NoError(t, m.Set(ctx, "s", "str-value", 0))
	v, err := m.Get(ctx, "s")
	require.NoError(t, err)
	assert.Equal(t, "str-value", v)

	require.NoError(t, m.Set(ctx, "b", []byte("byte-value"), 0))
	v, err = m.Get(ctx, "b")
	require.NoError(t, err)
	assert.Equal(t, "byte-value", v)

	require.NoError(t, m.Set(ctx, "i", 42, 0))
	v, err = m.Get(ctx, "i")
	require.NoError(t, err)
	assert.Equal(t, "42", v)

	// 未命中 → ErrNotFound
	_, err = m.Get(ctx, "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	// 过期 → ErrExpired
	require.NoError(t, m.Set(ctx, "exp", "x", 20*time.Millisecond))
	time.Sleep(30 * time.Millisecond)
	_, err = m.Get(ctx, "exp")
	assert.ErrorIs(t, err, ErrExpired)
}

func TestMemoryCache_DeleteExists(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	assert.ErrorIs(t, m.Delete(ctx, ""), ErrKeyEmpty)
	_, err := m.Exists(ctx, "")
	assert.ErrorIs(t, err, ErrKeyEmpty)

	require.NoError(t, m.Set(ctx, "k", "v", 0))
	exists, err := m.Exists(ctx, "k")
	require.NoError(t, err)
	assert.True(t, exists)

	require.NoError(t, m.Delete(ctx, "k"))
	exists, err = m.Exists(ctx, "k")
	require.NoError(t, err)
	assert.False(t, exists)

	// 删除不存在的 key → nil(幂等)
	assert.NoError(t, m.Delete(ctx, "k"))

	// 过期项 → Exists=false
	require.NoError(t, m.Set(ctx, "exp", "x", 20*time.Millisecond))
	time.Sleep(30 * time.Millisecond)
	exists, err = m.Exists(ctx, "exp")
	require.NoError(t, err)
	assert.False(t, exists, "过期项视为不存在")
}

// ---------------- MGet / MSet / MDelete ----------------

func TestMemoryCache_MOps(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	// MSet 空参 → nil;奇数参 → 错误;非字符串键 → 错误
	assert.NoError(t, m.MSet(ctx))
	assert.ErrorContains(t, m.MSet(ctx, "k1"), "偶数")
	assert.ErrorContains(t, m.MSet(ctx, 123, "v"), "字符串")

	// 正常批量
	require.NoError(t, m.MSet(ctx, "a", "1", "b", "2"))

	// MGet:命中+未命中混合(未命中为空串)
	vals, err := m.MGet(ctx, "a", "nope", "b")
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "", "2"}, vals)

	// 空参 MGet → 空切片
	vals, err = m.MGet(ctx)
	require.NoError(t, err)
	assert.Empty(t, vals)

	// MDelete
	require.NoError(t, m.MDelete(ctx, "a", "b"))
	exists, _ := m.Exists(ctx, "a")
	assert.False(t, exists)
}

// ---------------- Increment / Decrement ----------------

func TestMemoryCache_IncrementDecrement(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	// QUIRK(D-12 不修复): IncrementBy 对不存在 key 直接 nil 解引用 panic
	// (memory.go:215 item.Expiration) — 必须预种 string 数字值。
	require.NoError(t, m.Set(ctx, "cnt", "10", 0))

	n, err := m.Increment(ctx, "cnt")
	require.NoError(t, err)
	assert.Equal(t, int64(11), n)

	n, err = m.IncrementBy(ctx, "cnt", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(16), n)

	n, err = m.Decrement(ctx, "cnt")
	require.NoError(t, err)
	assert.Equal(t, int64(15), n)

	n, err = m.DecrementBy(ctx, "cnt", 3)
	require.NoError(t, err)
	assert.Equal(t, int64(12), n)

	// int 值路径
	require.NoError(t, m.Set(ctx, "icnt", 7, 0))
	n, err = m.IncrementBy(ctx, "icnt", 1)
	require.NoError(t, err)
	assert.Equal(t, int64(8), n)

	// QUIRK(D-12 不修复): 非数值字符串解析失败被静默吞掉,按 0 计继续累加
	require.NoError(t, m.Set(ctx, "bad", "not-num", 0))
	n, err = m.IncrementBy(ctx, "bad", 1)
	require.NoError(t, err, "解析失败不上抛,按 0 累加")
	assert.Equal(t, int64(1), n)

	// 不可解析类型(map)→ ErrInvalidType
	require.NoError(t, m.Set(ctx, "mapv", map[string]int{"a": 1}, 0))
	_, err = m.IncrementBy(ctx, "mapv", 1)
	assert.ErrorIs(t, err, ErrInvalidType)
}

// ---------------- Expire / TTL ----------------

func TestMemoryCache_ExpireAndTTL(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	assert.ErrorIs(t, m.Expire(ctx, "", time.Minute), ErrKeyEmpty)
	assert.ErrorIs(t, m.Expire(ctx, "missing", time.Minute), ErrNotFound)
	_, err := m.TTL(ctx, "")
	assert.ErrorIs(t, err, ErrKeyEmpty)
	_, err = m.TTL(ctx, "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	// 永不过期 → -1
	require.NoError(t, m.Set(ctx, "forever", "v", 0))
	ttl, err := m.TTL(ctx, "forever")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl)

	// 设置 1 分钟 TTL → 正剩余
	require.NoError(t, m.Expire(ctx, "forever", time.Minute))
	ttl, err = m.TTL(ctx, "forever")
	require.NoError(t, err)
	assert.Greater(t, ttl, 30*time.Second)
	assert.LessOrEqual(t, ttl, time.Minute)

	// 过期后 TTL → ErrExpired
	require.NoError(t, m.Expire(ctx, "forever", 20*time.Millisecond))
	time.Sleep(30 * time.Millisecond)
	_, err = m.TTL(ctx, "forever")
	assert.ErrorIs(t, err, ErrExpired)
}

// ---------------- Keys / FlushDB ----------------

func TestMemoryCache_KeysAndFlush(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	require.NoError(t, m.Set(ctx, "user:1", "a", 0))
	require.NoError(t, m.Set(ctx, "user:2", "b", 0))
	require.NoError(t, m.Set(ctx, "role:1", "c", 0))

	// * → 全部
	keys, err := m.Keys(ctx, "*")
	require.NoError(t, err)
	assert.Len(t, keys, 3)

	// 前缀通配
	keys, err = m.Keys(ctx, "user:*")
	require.NoError(t, err)
	assert.Len(t, keys, 2)

	// 精确匹配
	keys, err = m.Keys(ctx, "role:1")
	require.NoError(t, err)
	assert.Equal(t, []string{"role:1"}, keys)

	// 无匹配
	keys, err = m.Keys(ctx, "nomatch:*")
	require.NoError(t, err)
	assert.Empty(t, keys)

	// FlushDB
	require.NoError(t, m.FlushDB(ctx))
	keys, _ = m.Keys(ctx, "*")
	assert.Empty(t, keys)
}

// ---------------- LRU 淘汰 ----------------

func TestMemoryCache_EvictLRU(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 3)

	for _, k := range []string{"k1", "k2", "k3"} {
		require.NoError(t, m.Set(ctx, k, "v", 0))
		time.Sleep(5 * time.Millisecond) // 保证 Created 可区分
	}

	// 第 4 个 key 触发 LRU:最老的 k1 被淘汰
	require.NoError(t, m.Set(ctx, "k4", "v", 0))
	_, err := m.Get(ctx, "k1")
	assert.ErrorIs(t, err, ErrNotFound, "最老项被 LRU 淘汰")
	v, err := m.Get(ctx, "k4")
	require.NoError(t, err)
	assert.Equal(t, "v", v)
}

// ---------------- 手动 cleanup ----------------

func TestMemoryCache_Cleanup(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	require.NoError(t, m.Set(ctx, "short", "x", 20*time.Millisecond))
	require.NoError(t, m.Set(ctx, "long", "y", time.Hour))
	time.Sleep(30 * time.Millisecond)

	m.cleanup() // 手工触发(协程在 interval=0 下未启动)

	keys, _ := m.Keys(ctx, "*")
	assert.Equal(t, []string{"long"}, keys, "过期项被清理")
}

// ---------------- JSON API ----------------

func TestMemoryCache_JSONOps(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	// SetJSON/GetJSON roundtrip
	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	require.NoError(t, m.SetJSON(ctx, "j1", payload{Name: "alice", N: 5}, 0))
	var out payload
	require.NoError(t, m.GetJSON(ctx, "j1", &out))
	assert.Equal(t, "alice", out.Name)
	assert.Equal(t, 5, out.N)

	// GetJSON 未命中 → ErrNotFound
	assert.ErrorIs(t, m.GetJSON(ctx, "nope", &out), ErrNotFound)

	// MSetJSON / MGetJSON
	require.NoError(t, m.MSetJSON(ctx, map[string]interface{}{
		"mj1": map[string]int{"a": 1},
		"mj2": "plain",
	}, time.Minute))
	got, err := m.MGetJSON(ctx, "mj1", "mj2", "missing")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, map[string]interface{}{"a": float64(1)}, got["mj1"])
	assert.Equal(t, "plain", got["mj2"])

	// MSetJSON 不可序列化值 → 错误
	assert.Error(t, m.MSetJSON(ctx, map[string]interface{}{"bad": make(chan int)}, time.Minute))
}

// ---------------- Int / Bool 访问器 ----------------

func TestMemoryCache_IntBoolAccessors(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	// SetInt/GetInt roundtrip
	require.NoError(t, m.SetInt(ctx, "num", 99, 0))
	n, err := m.GetInt(ctx, "num")
	require.NoError(t, err)
	assert.Equal(t, 99, n)

	// 非数字 → Atoi 错误
	require.NoError(t, m.Set(ctx, "notnum", "abc", 0))
	_, err = m.GetInt(ctx, "notnum")
	assert.Error(t, err)

	// SetBool/GetBool roundtrip("1"/"0")
	require.NoError(t, m.SetBool(ctx, "flag", true, 0))
	b, err := m.GetBool(ctx, "flag")
	require.NoError(t, err)
	assert.True(t, b)

	require.NoError(t, m.SetBool(ctx, "flag0", false, 0))
	b, err = m.GetBool(ctx, "flag0")
	require.NoError(t, err)
	assert.False(t, b)

	// 非布尔串 → ParseBool 错误
	require.NoError(t, m.Set(ctx, "notbool", "x", 0))
	_, err = m.GetBool(ctx, "notbool")
	assert.Error(t, err)
}

// ---------------- Hash API ----------------

func TestMemoryCache_HashOps(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	// HSet string / 非string 值
	require.NoError(t, m.HSet(ctx, "h1", "f1", "v1"))
	require.NoError(t, m.HSet(ctx, "h1", "f2", 42)) // 非 string → JSON 编码 "42"

	// HGet
	v, err := m.HGet(ctx, "h1", "f1")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	v, err = m.HGet(ctx, "h1", "f2")
	require.NoError(t, err)
	assert.Equal(t, "42", v)

	// HGet 未命中 → ErrNotFound
	_, err = m.HGet(ctx, "h1", "nope")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = m.HGet(ctx, "missing-hash", "f")
	assert.ErrorIs(t, err, ErrNotFound)

	// HGetAll
	all, err := m.HGetAll(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"f1": "v1", "f2": "42"}, all)

	// 不存在的 hash → 空 map(非错误)
	all, err = m.HGetAll(ctx, "missing")
	require.NoError(t, err)
	assert.Empty(t, all)

	// HKeys
	fields, err := m.HKeys(ctx, "h1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"f1", "f2"}, fields)

	// HDel 单字段 → 保留其余
	require.NoError(t, m.HDel(ctx, "h1", "f1"))
	v, err = m.HGet(ctx, "h1", "f1")
	assert.ErrorIs(t, err, ErrNotFound)
	v, err = m.HGet(ctx, "h1", "f2")
	require.NoError(t, err)
	assert.Equal(t, "42", v)

	// HDel 全部字段 → 整个 key 删除
	require.NoError(t, m.HDel(ctx, "h1", "f2"))
	exists, _ := m.Exists(ctx, "h1")
	assert.False(t, exists, "空 hash 连 key 一起删")

	// 非字符串值 key 的 HGet → not found 路径
	require.NoError(t, m.Set(ctx, "bin", []byte("x"), 0))
	_, err = m.HGet(ctx, "bin", "f")
	assert.Error(t, err)
}

// ---------------- GetStats ----------------

func TestMemoryCache_GetStats(t *testing.T) {
	ctx := context.Background()
	m := newMem(t, 100)

	require.NoError(t, m.Set(ctx, "a", "1", 0))
	_, _ = m.Get(ctx, "a") // hit
	_, _ = m.Get(ctx, "x") // miss

	stats, err := m.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, "memory", stats["cache_type"])
	assert.Equal(t, int64(1), stats["key_count"])
	assert.Equal(t, int64(1), stats["keyspace_hits"])
	assert.Equal(t, int64(1), stats["keyspace_misses"])
	assert.Greater(t, stats["hit_rate"], float64(0))

	// 空缓存:hit_rate=0.0 分支
	m2 := newMem(t, 10)
	stats2, err := m2.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0.0, stats2["hit_rate"])
}

// ---------------- 纯函数 ----------------

func TestMatchPattern(t *testing.T) {
	assert.True(t, matchPattern("any", "*"))
	assert.True(t, matchPattern("user:1", "user:*"))
	assert.False(t, matchPattern("role:1", "user:*"))
	assert.True(t, matchPattern("exact", "exact"))
	assert.False(t, matchPattern("exact1", "exact"))
}

func TestCacheItem_IsExpired(t *testing.T) {
	never := &CacheItem{Expiration: 0}
	assert.False(t, never.IsExpired(), "0 = 永不过期")

	future := &CacheItem{Expiration: time.Now().Add(time.Hour).UnixNano()}
	assert.False(t, future.IsExpired())

	past := &CacheItem{Expiration: time.Now().Add(-time.Hour).UnixNano()}
	assert.True(t, past.IsExpired())
}

func TestErrorPredicates(t *testing.T) {
	assert.True(t, IsNotFound(ErrNotFound))
	assert.False(t, IsNotFound(ErrExpired))
	assert.True(t, IsExpired(ErrExpired))
	assert.False(t, IsExpired(ErrKeyEmpty))
	assert.True(t, IsKeyEmpty(ErrKeyEmpty))
	assert.False(t, IsKeyEmpty(nil))
}
