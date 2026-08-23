package cache

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// Phase 76-01: RedisCache miniredis 冒烟（INFRA-01，v1.27 D-02 解禁）。
// 经 NewRedisCache 真链路连进程内 miniredis（真 TCP、零 Docker），实证
// pkg/cache 命令面。三坑防护纪律：
//   R-1 TTL: miniredis 的 TTL 不随真实时间流逝，过期推进一律用
//      mr.FastForward，严禁真实睡眠等待过期（必然假绿/假红）。
//   R-2 INFO 断言降级: v2.38.0 对 server/memory/keyspace section 返回
//      "section not supported" 错误，GetStats 对段错误静默跳过
//      （err==nil 守卫 / dbInfo 忽略错误），故只断言可达成字段。
//   R-3 CLIENT SETINFO 握手兼容: NewRedisCache 构造器内的 PING 已走完
//      go-redis v9.7.0 完整握手（HELLO → SETINFO×2），错误被客户端显式
//      丢弃，组合开箱即用——newMiniredisCache 构造成功即 R-3 实证，
//      无需（也不应）做任何客户端身份相关特殊处理。
// =====================================================================

// newMiniredisCache 启动 miniredis 并经 NewRedisCache 构造真客户端。
// RunT 失败即 t.Fatal，测试结束自动关闭 server 与客户端。
// 本 helper 是 R-3 的回归哨兵：未来升级 go-redis 后握手若失效，
// 所有依赖它的用例会在构造处立即失败。
func newMiniredisCache(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)

	host, portStr, err := net.SplitHostPort(mr.Addr())
	if err != nil {
		t.Fatalf("拆解 miniredis 地址失败: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("解析 miniredis 端口失败: %v", err)
	}

	// 构造器内的 rdb.Ping 走完完整连接握手（R-3 冒烟点）。
	r, err := NewRedisCache(&CacheConfig{Host: host, Port: port}, "xingran")
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, mr
}

// ---------------- 命令面冒烟 ----------------

// TestRedisBasicCommandSurface 覆盖 Set/Get/Exists/Delete、
// Increment/IncrementBy、MSet/MGet、HSet/HGet/HGetAll/HDel/HKeys。
func TestRedisBasicCommandSurface(t *testing.T) {
	ctx := context.Background()
	r, _ := newMiniredisCache(t)

	// Set / Get
	require.NoError(t, r.Set(ctx, "k1", "v1", 0))
	v, err := r.Get(ctx, "k1")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	// Exists / Delete（未命中 → ErrNotFound）
	exists, err := r.Exists(ctx, "k1")
	require.NoError(t, err)
	assert.True(t, exists)
	require.NoError(t, r.Delete(ctx, "k1"))
	exists, err = r.Exists(ctx, "k1")
	require.NoError(t, err)
	assert.False(t, exists)
	_, err = r.Get(ctx, "k1")
	assert.ErrorIs(t, err, ErrNotFound)

	// Increment / IncrementBy
	n, err := r.Increment(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	n, err = r.IncrementBy(ctx, "counter", 41)
	require.NoError(t, err)
	assert.Equal(t, int64(42), n)

	// MSet / MGet（缺失键 → 空串占位）
	require.NoError(t, r.MSet(ctx, "m1", "a", "m2", "b"))
	vals, err := r.MGet(ctx, "m1", "m2", "missing")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", ""}, vals)

	// HSet / HGet / HGetAll / HKeys / HDel
	// （字段名保持短于前缀长度，规避 HKeys 对字段名的历史前缀裁剪行为）
	require.NoError(t, r.HSet(ctx, "h1", "f1", "hv1"))
	require.NoError(t, r.HSet(ctx, "h1", "f2", "hv2"))
	fv, err := r.HGet(ctx, "h1", "f1")
	require.NoError(t, err)
	assert.Equal(t, "hv1", fv)
	all, err := r.HGetAll(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"f1": "hv1", "f2": "hv2"}, all)
	fields, err := r.HKeys(ctx, "h1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"f1", "f2"}, fields)
	require.NoError(t, r.HDel(ctx, "h1", "f1"))
	_, err = r.HGet(ctx, "h1", "f1")
	assert.Error(t, err, "HDel 后字段应返回 redis.Nil（HGet 保持原语义）")
}

// TestRedisKeysScan SCAN 路径（Keys 内部 scanKeys 游标遍历）。
// miniredis 游标一次归零，不做 SCAN 分批大小断言。
func TestRedisKeysScan(t *testing.T) {
	ctx := context.Background()
	r, mr := newMiniredisCache(t)

	for _, k := range []string{"scan:a", "scan:b", "scan:c", "other:x"} {
		require.NoError(t, r.Set(ctx, k, "v", 0))
	}

	keys, err := r.Keys(ctx, "scan:*")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"scan:a", "scan:b", "scan:c"}, keys)

	// buildKey 自动加 xingran: 前缀：经 cache 方法断言无感；
	// 直查 mr 时必须带前缀（CLAUDE.md 缓存前缀约定）。
	assert.True(t, mr.Exists("xingran:other:x"))
	assert.False(t, mr.Exists("other:x"))
}

// ---------------- 三坑防护具名用例 ----------------

// TestRedisTTLFastForward R-1：TTL 推进必须用 mr.FastForward。
// FastForward 后所有 <=0 的 TTL 键被移除，Get 应返回 ErrNotFound。
func TestRedisTTLFastForward(t *testing.T) {
	ctx := context.Background()
	r, mr := newMiniredisCache(t)

	require.NoError(t, r.Set(ctx, "ttl-key", "v", 0))
	require.NoError(t, r.Expire(ctx, "ttl-key", 10*time.Second))

	ttl, err := r.TTL(ctx, "ttl-key")
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0), "设置 Expire 后 TTL 应大于 0")

	mr.FastForward(11 * time.Second)

	_, err = r.Get(ctx, "ttl-key")
	assert.ErrorIs(t, err, ErrNotFound, "FastForward 越过 TTL 后键应被移除")
}

// TestRedisGetStatsDegraded R-2：INFO 断言降级。
// v2.38.0 下 INFO server/memory/keyspace 返回错误 → GetStats 静默跳过；
// INFO stats 与 DBSize 可用。只断言可达成字段，不断言
// redis_version/used_memory/hit_rate 的具体值（miniredis 下无意义）。
func TestRedisGetStatsDegraded(t *testing.T) {
	ctx := context.Background()
	r, _ := newMiniredisCache(t)

	require.NoError(t, r.Set(ctx, "stat:1", "v", 0))
	require.NoError(t, r.Set(ctx, "stat:2", "v", 0))

	stats, err := r.GetStats(ctx)
	require.NoError(t, err, "GetStats 对段错误静默降级，永不返回 error")

	// DBSize 正常：key_count 真实可信
	keyCount, ok := stats["key_count"].(int64)
	require.True(t, ok, "key_count 应存在且为 int64")
	assert.GreaterOrEqual(t, keyCount, int64(2))

	// 降级面实证：server/memory 段错误被跳过 → 版本/内存字段缺席
	_, hasVersion := stats["redis_version"]
	assert.False(t, hasVersion, "miniredis 下 redis_version 应缺席")
	_, hasMem := stats["used_memory"]
	assert.False(t, hasMem, "miniredis 下 used_memory 应缺席")

	// keyspace 段错误被忽略 → keyspace_info 键存在但为空串
	kv, hasKS := stats["keyspace_info"]
	require.True(t, hasKS, "keyspace_info 键无条件赋值，必然存在")
	assert.Equal(t, "", kv, "v2.38.0 INFO keyspace 报错 → 降级为空串")

	// stats 段可用但 keyspace_hits 缺席 → hit_rate 固定 0.0
	_, hasHits := stats["keyspace_hits"]
	assert.False(t, hasHits)
	rate, hasRate := stats["hit_rate"]
	require.True(t, hasRate)
	assert.Equal(t, float64(0), rate)
}

// ---------------- EVAL / Lua 路径 ----------------

// TestRedisIncrementWithExpire EVAL 路径（gopher-lua 驱动）：
// 首次 INCR 返回 1 并设置 EXPIRE（TTL > 0）；
// 已存在的 key 仅递增、不重置过期时间（Lua 脚本语义）。
func TestRedisIncrementWithExpire(t *testing.T) {
	ctx := context.Background()
	r, mr := newMiniredisCache(t)

	n, err := r.IncrementWithExpire(ctx, "lua-counter", 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	ttl, err := r.TTL(ctx, "lua-counter")
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0), "首次 INCR 应设置过期时间")

	// 推进 10s 后再次递增：current != 1 → 不重置 TTL，剩余应约 20s
	mr.FastForward(10 * time.Second)
	n, err = r.IncrementWithExpire(ctx, "lua-counter", 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	ttl, err = r.TTL(ctx, "lua-counter")
	require.NoError(t, err)
	assert.InDelta(t, 20*time.Second, ttl, float64(2*time.Second),
		"已存在 key 递增不应重置过期时间")
}
