package cache

// =====================================================================
// Phase 80-05 Task 1: redis.go typed helpers + MultiLevelCache 收口。
// (D-80-05 重锚:pkg/cache 缺口 +161 → +49,基线 64.7% → ≥70%。)
//
// 76-01 (redis_miniredis_76_01_test.go) 已覆盖基础命令面
// (Set/Get/Exists/Delete/Increment·By/MSet/MGet/H*/Keys/Expire/TTL/GetStats/Lua),
// 本文件只补缺口,不重写既有用例:
//   - typed helpers: GetJSON/SetJSON/GetInt/SetInt/GetBool/SetBool/
//     MGetJSON/MSetJSON/MDelete/Decrement·By/FlushDB/getClient/getPrefix
//   - MultiLevelCache: 4+1 个构造器 + 全方法(redis.go :595-1057 与
//     RedisCache 同文件,一并计入 redis.go 覆盖)
//
// 纪律:
//   - R-1: miniredis 的 TTL 不随真实时间流逝,过期推进一律 mr.FastForward,
//     禁裸 time.Sleep。
//   - 79-R7: 起 worker 的 MultiLevelCache 构造器(NewMultiLevelCache /
//     WithWriter / WithRetry / WithRetryAndWriter)必须 Close + t.Cleanup。
//   - asyncSetL2 为 detached goroutine,其 L2 落盘断言一律 assert.Eventually
//     (local-vs-ci: 异步断言禁 sleep 轮询)。
// =====================================================================

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

// rds8005Payload JSON round-trip 载荷(struct 序列化字段序确定)。
type rds8005Payload struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

// newRds8005 启动 miniredis 并经 NewRedisCache 构造真客户端(空前缀,
// 直查 mr 时键名无需拼前缀)。Close 单次:用例内显式 Close 后 Cleanup 跳过。
func newRds8005(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	rc, err := NewRedisCache(&CacheConfig{Host: host, Port: port}, "")
	require.NoError(t, err, "构造器内 rdb.Ping 应走完 go-redis 完整握手")
	closed := false
	t.Cleanup(func() {
		if !closed {
			closed = true
			_ = rc.Close()
		}
	})
	return rc, mr
}

// newMlc8005 构造 Simple 多级缓存(无后台 writer,79-R7 首选)。
// l2IsRedis=false 时 L2 也用 MemoryCache(不实现 redisCacheInterface,
// 覆盖 DirectRedis* 的类型断言失败分支)。
func newMlc8005(t *testing.T, l2IsRedis bool) (*MultiLevelCache, *RedisCache, *MemoryCache, *miniredis.Miniredis) {
	t.Helper()
	var mr *miniredis.Miniredis
	var rc *RedisCache
	if l2IsRedis {
		rc, mr = newRds8005(t)
	}
	mem := NewMemoryCache(1000, time.Minute)
	t.Cleanup(func() { _ = mem.Close() })

	var l2 Cache
	if rc != nil {
		l2 = rc
	} else {
		l2 = mem
	}
	mlc := NewMultiLevelCacheSimple(mem, l2)
	t.Cleanup(func() { _ = mlc.Close() })
	return mlc, rc, mem, mr
}

// ---------------- TestRds8005_: typed helpers ----------------

// TestRds8005_JSON_RoundTrip_SetJSON 落库为原始 JSON 文本;坏 JSON 走
// GetJSON 的 json.Unmarshal 错误分支(区别于 ErrNotFound)。
func TestRds8005_JSON_RoundTrip(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)

	want := rds8005Payload{Name: "钟离", N: 7}
	require.NoError(t, rc.SetJSON(ctx, "json:ok", want, 0))

	// L2 落盘断言:直查 miniredis 原始键(空前缀,键名原样)。
	raw, err := mr.Get("json:ok")
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"钟离","n":7}`, raw)

	var got rds8005Payload
	require.NoError(t, rc.GetJSON(ctx, "json:ok", &got))
	assert.Equal(t, want, got)

	// 缺键 → ErrNotFound(Get 的 redis.Nil 映射)。
	var missing rds8005Payload
	assert.ErrorIs(t, rc.GetJSON(ctx, "json:absent", &missing), ErrNotFound)

	// 坏 JSON → json.Unmarshal 错误分支。
	require.NoError(t, rc.Set(ctx, "json:bad", "{not-json", 0))
	assert.Error(t, rc.GetJSON(ctx, "json:bad", &missing))
}

// TestRds8005_Int_RoundTrip 覆盖 SetInt/GetInt 正路与非数字值错误分支。
func TestRds8005_Int_RoundTrip(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)

	require.NoError(t, rc.SetInt(ctx, "int:ok", 42, 0))
	raw, err := mr.Get("int:ok")
	require.NoError(t, err)
	assert.Equal(t, "42", raw, "SetInt 应以十进制字符串落库")

	n, err := rc.GetInt(ctx, "int:ok")
	require.NoError(t, err)
	assert.Equal(t, 42, n)

	_, err = rc.GetInt(ctx, "int:absent")
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, rc.Set(ctx, "int:nan", "abc", 0))
	_, err = rc.GetInt(ctx, "int:nan")
	assert.Error(t, err, "strconv.Atoi 对非数字值应报错")
}

// TestRds8005_Bool_RoundTrip:SetBool(true)→"1"、(false)→"0";
// ParseBool 拒绝的值走错误分支。
func TestRds8005_Bool_RoundTrip(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)

	require.NoError(t, rc.SetBool(ctx, "bool:t", true, 0))
	v, err := mr.Get("bool:t")
	require.NoError(t, err)
	assert.Equal(t, "1", v)

	got, err := rc.GetBool(ctx, "bool:t")
	require.NoError(t, err)
	assert.True(t, got)

	require.NoError(t, rc.SetBool(ctx, "bool:f", false, 0))
	v, err = mr.Get("bool:f")
	require.NoError(t, err)
	assert.Equal(t, "0", v)
	got, err = rc.GetBool(ctx, "bool:f")
	require.NoError(t, err)
	assert.False(t, got)

	_, err = rc.GetBool(ctx, "bool:absent")
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, rc.Set(ctx, "bool:bad", "yes", 0))
	_, err = rc.GetBool(ctx, "bool:bad")
	assert.Error(t, err, "strconv.ParseBool 不接受 'yes'")
}

// TestRds8005_MGetJSON_PartialAndRaw:部分缺失键语义(缺失键不出现在
// 结果 map);非 JSON 字符串值原样返回(MGetJSON 容错分支)。
func TestRds8005_MGetJSON_PartialAndRaw(t *testing.T) {
	ctx := context.Background()
	rc, _ := newRds8005(t)

	require.NoError(t, rc.SetJSON(ctx, "mg:obj", rds8005Payload{Name: "a", N: 1}, 0))
	require.NoError(t, rc.Set(ctx, "mg:raw", "plain-text", 0))

	res, err := rc.MGetJSON(ctx, "mg:obj", "mg:raw", "mg:absent")
	require.NoError(t, err)
	require.Len(t, res, 2, "缺失键不应出现在结果中")

	obj, ok := res["mg:obj"].(map[string]interface{})
	require.True(t, ok, "合法 JSON 应解为对象")
	assert.Equal(t, "a", obj["name"])
	assert.Equal(t, float64(1), obj["n"])

	assert.Equal(t, "plain-text", res["mg:raw"], "非 JSON 字符串应原样返回")

	// 空键集 → 空 map。
	empty, err := rc.MGetJSON(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// TestRds8005_MSetJSON_TTLAndEmpty:MSetJSON 批量落库 + expiration 触发
// Expire;TTL 推进一律 mr.FastForward(R-1)。
func TestRds8005_MSetJSON_TTLAndEmpty(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)

	// QUIRK-80-05-B:RedisCache.MSetJSON 名不符实 —— 值不经 json.Marshal,
	// 直传 client.MSet;struct/map 值会报 "can't marshal (implement
	// encoding.BinaryMarshaler)"。仅 go-redis 原生可编码值(string/int/...)
	// 可用。本测试按标量值锁行为,只锁不修(零生产改动纪律)。
	data := map[string]interface{}{
		"ms:a": "va",
		"ms:b": "vb",
	}
	require.NoError(t, rc.MSetJSON(ctx, data, 30*time.Second))
	require.True(t, mr.Exists("ms:a"))
	require.True(t, mr.Exists("ms:b"))
	rawA, err := mr.Get("ms:a")
	require.NoError(t, err)
	assert.Equal(t, "va", rawA, "标量值按原样落库(无 JSON 包装)")
	assert.Greater(t, mr.TTL("ms:a"), time.Duration(0), "expiration>0 应设置 Expire")

	// struct 值确实不可用(QUIRK-80-05-B 行为锁定)。
	assert.Error(t, rc.MSetJSON(ctx, map[string]interface{}{"ms:s": rds8005Payload{}}, time.Second))

	// 空数据 → no-op(不报错、不落任何键)。
	require.NoError(t, rc.MSetJSON(ctx, map[string]interface{}{}, time.Second))
	assert.False(t, mr.Exists("ms:none"))

	// FastForward 越过 TTL 后键应被移除。
	mr.FastForward(31 * time.Second)
	assert.False(t, mr.Exists("ms:a"), "FastForward 后过期键应被移除")
	assert.False(t, mr.Exists("ms:b"))
}

// TestRds8005_MDelete_Batch:批量删除 + 空切片/不存在键分支。
func TestRds8005_MDelete_Batch(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)

	require.NoError(t, rc.Set(ctx, "md:a", "1", 0))
	require.NoError(t, rc.Set(ctx, "md:b", "2", 0))
	require.NoError(t, rc.MDelete(ctx, "md:a", "md:b"))
	assert.False(t, mr.Exists("md:a"))
	assert.False(t, mr.Exists("md:b"))

	require.NoError(t, rc.MDelete(ctx), "空切片应直接返回 nil")
	require.NoError(t, rc.MDelete(ctx, "md:absent"))
}

// TestRds8005_Counter_Family:Increment/IncrementBy/Decrement/DecrementBy。
// Redis 缺键自增语义 = 从 0 起算(首步结果 1);对照 QUIRK-01 修复前
// MemoryCache RateLimitCache 缺键语义差异 —— Redis 路径为 INCR 原子语义。
func TestRds8005_Counter_Family(t *testing.T) {
	ctx := context.Background()
	rc, _ := newRds8005(t)

	// Increment 缺键 → 1。
	n, err := rc.Increment(ctx, "cnt:a")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = rc.IncrementBy(ctx, "cnt:a", 41)
	require.NoError(t, err)
	assert.Equal(t, int64(42), n)

	n, err = rc.IncrementBy(ctx, "cnt:b", -5)
	require.NoError(t, err)
	assert.Equal(t, int64(-5), n, "IncrementBy 缺键按 0 + delta 计算")

	// Decrement 缺键 → -1。
	n, err = rc.Decrement(ctx, "cnt:c")
	require.NoError(t, err)
	assert.Equal(t, int64(-1), n)

	n, err = rc.DecrementBy(ctx, "cnt:c", 9)
	require.NoError(t, err)
	assert.Equal(t, int64(-10), n)
}

// TestRds8005_Exists_FlushDB:Exists 有/无两态 + FlushDB 清空。
func TestRds8005_Exists_FlushDB(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)

	require.NoError(t, rc.Set(ctx, "ef:1", "v", 0))
	ok, err := rc.Exists(ctx, "ef:1")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = rc.Exists(ctx, "ef:absent")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, rc.FlushDB(ctx))
	assert.False(t, mr.Exists("ef:1"), "FlushDB 后键应被清空")
	ok, err = rc.Exists(ctx, "ef:1")
	require.NoError(t, err)
	assert.False(t, ok)
}

// TestRds8005_TTL_FastForward_Miss:SetJSON 带 TTL → FastForward 越过 →
// GetJSON miss(R-1:零真实睡眠)。
func TestRds8005_TTL_FastForward_Miss(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)

	require.NoError(t, rc.SetJSON(ctx, "ttl:j", rds8005Payload{Name: "x", N: 1}, time.Second))
	var got rds8005Payload
	require.NoError(t, rc.GetJSON(ctx, "ttl:j", &got))
	assert.Equal(t, 1, got.N)

	mr.FastForward(2 * time.Second)
	assert.ErrorIs(t, rc.GetJSON(ctx, "ttl:j", &got), ErrNotFound)
}

// TestRds8005_ClientAndPrefix_Accessors:同包直调 getClient/getPrefix;
// 附带一个带前缀客户端验证 buildKey 拼接路径。
func TestRds8005_ClientAndPrefix_Accessors(t *testing.T) {
	rc, _ := newRds8005(t)
	assert.NotNil(t, rc.getClient(), "getClient 应返回底层 go-redis client")
	assert.Empty(t, rc.getPrefix(), "空前缀客户端 getPrefix 应为空串")

	// 带前缀客户端:buildKey 走 prefix + ":" + key 分支。
	mr2 := miniredis.RunT(t)
	host, portStr, err := net.SplitHostPort(mr2.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	prefixed, err := NewRedisCache(&CacheConfig{Host: host, Port: port}, "p8005")
	require.NoError(t, err)
	t.Cleanup(func() { _ = prefixed.Close() })

	assert.Equal(t, "p8005", prefixed.getPrefix())
	require.NoError(t, prefixed.Set(context.Background(), "k", "v", 0))
	_, err = mr2.Get("p8005:k")
	require.NoError(t, err, "带前缀客户端落库键应为 p8005:k")
}

// TestRds8005_MSet_MGet_ErrorBranches:MSet 奇数参数/非字符串键错误分支,
// MGet/MDelete 空参短路。
func TestRds8005_MSet_MGet_ErrorBranches(t *testing.T) {
	ctx := context.Background()
	rc, _ := newRds8005(t)

	assert.Error(t, rc.MSet(ctx, "odd"), "奇数个参数应报错")
	assert.Error(t, rc.MSet(ctx, 123, "v"), "非字符串键应报错")

	vals, err := rc.MGet(ctx)
	require.NoError(t, err)
	assert.Empty(t, vals)

	require.NoError(t, rc.MDelete(ctx))
}

// ---------------- TestMlc8005_: MultiLevelCache ----------------

// TestMlc8005_Simple_Constructor_RoundTrips:NewMultiLevelCacheSimple(无
// 后台 writer)全方法往返。
func TestMlc8005_Simple_Constructor_RoundTrips(t *testing.T) {
	ctx := context.Background()
	mlc, rc, _, mr := newMlc8005(t, true)

	require.NoError(t, mlc.Set(ctx, "s:k", "v1", 0))
	v, err := mlc.Get(ctx, "s:k")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	// Simple 模式 L2 走 asyncSetL2 detached goroutine → Eventually 断言落盘。
	assert.Eventually(t, func() bool { return mr.Exists("s:k") }, 2*time.Second, 10*time.Millisecond,
		"asyncSetL2 应把值写入 L2(miniredis)")

	ok, err := mlc.Exists(ctx, "s:k")
	require.NoError(t, err)
	assert.True(t, ok)

	// MGet/MSet/MDelete 委托 L2。
	require.NoError(t, rc.Set(ctx, "s:m1", "a", 0))
	vals, err := mlc.MGet(ctx, "s:m1", "s:absent")
	require.NoError(t, err)
	assert.Equal(t, []string{"a", ""}, vals)

	require.NoError(t, mlc.MSet(ctx, "s:m2", "b", "s:m3", "c"))
	vals, err = mlc.MGet(ctx, "s:m2", "s:m3")
	require.NoError(t, err)
	assert.Equal(t, []string{"b", "c"}, vals)

	require.NoError(t, mlc.MDelete(ctx, "s:m2", "s:m3"))
	vals, err = mlc.MGet(ctx, "s:m2", "s:m3")
	require.NoError(t, err)
	assert.Equal(t, []string{"", ""}, vals)

	// Increment/Decrement 家族委托 L2。
	n, err := mlc.Increment(ctx, "s:cnt")
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	n, err = mlc.IncrementBy(ctx, "s:cnt", 9)
	require.NoError(t, err)
	assert.Equal(t, int64(10), n)
	n, err = mlc.Decrement(ctx, "s:cnt")
	require.NoError(t, err)
	assert.Equal(t, int64(9), n)
	n, err = mlc.DecrementBy(ctx, "s:cnt", 4)
	require.NoError(t, err)
	assert.Equal(t, int64(5), n)

	// Expire/TTL 委托 L2。
	require.NoError(t, mlc.Expire(ctx, "s:k", 30*time.Second))
	ttl, err := mlc.TTL(ctx, "s:k")
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))

	// Delete 双级删除。
	require.NoError(t, mlc.Delete(ctx, "s:k"))
	ok, err = mlc.Exists(ctx, "s:k")
	require.NoError(t, err)
	assert.False(t, ok)
	_, err = mlc.Get(ctx, "s:k")
	assert.Error(t, err, "双级删除后 Get 应 miss")
}

// TestMlc8005_Simple_L2FallbackAndBackfill:L1 miss → L2 hit → 回填 L1;
// L1/L2 双 miss → ErrNotFound。
func TestMlc8005_Simple_L2FallbackAndBackfill(t *testing.T) {
	ctx := context.Background()
	mlc, rc, mem, _ := newMlc8005(t, true)

	// 只写 L2(RedisCache 直写)→ mlc.Get 走 L2 回填分支。
	require.NoError(t, rc.Set(ctx, "fb:only-l2", "v2", 0))
	v, err := mlc.Get(ctx, "fb:only-l2")
	require.NoError(t, err)
	assert.Equal(t, "v2", v)

	// 回填断言:L1(MemoryCache)此时应持有该值。
	v, err = mem.Get(ctx, "fb:only-l2")
	require.NoError(t, err)
	assert.Equal(t, "v2", v, "L2 命中后应回填 L1(context.WithoutCancel 路径)")

	// 双 miss → ErrNotFound。
	_, err = mlc.Get(ctx, "fb:absent")
	assert.ErrorIs(t, err, ErrNotFound)

	// L1 命中短路(Exists 的 L1 前置分支)。
	ok, err := mlc.Exists(ctx, "fb:only-l2")
	require.NoError(t, err)
	assert.True(t, ok)

	// 只写 L1 → Exists 走 L1 hit 分支;再删 L1 → Exists 回源 L2。
	require.NoError(t, mem.Set(ctx, "fb:only-l1", "l1v", 0))
	ok, err = mlc.Exists(ctx, "fb:only-l1")
	require.NoError(t, err)
	assert.True(t, ok)
	require.NoError(t, mem.Delete(ctx, "fb:only-l1"))
	ok, err = mlc.Exists(ctx, "fb:only-l1")
	require.NoError(t, err)
	assert.False(t, ok, "L1 删除后 Exists 应回源 L2 判定(L2 无此键)")
}

// TestMlc8005_JSON_Helpers_TwoLevel:SetJSON/GetJSON/MGetJSON/MSetJSON
// 的 L1/L2 两级行为(L2 落盘以 miniredis 原始键证明)。
func TestMlc8005_JSON_Helpers_TwoLevel(t *testing.T) {
	ctx := context.Background()
	mlc, rc, mem, mr := newMlc8005(t, true)

	want := rds8005Payload{Name: "两level", N: 3}
	require.NoError(t, mlc.SetJSON(ctx, "j:one", want, 0))

	// L1 同步写入:MemoryCache 应立即持有 JSON 文本。
	l1raw, err := mem.Get(ctx, "j:one")
	require.NoError(t, err)
	assert.Contains(t, l1raw, "两level")

	// L2 经 asyncSetL2(Simple 模式)→ Eventually 落盘。
	assert.Eventually(t, func() bool { return mr.Exists("j:one") }, 2*time.Second, 10*time.Millisecond)

	var got rds8005Payload
	require.NoError(t, mlc.GetJSON(ctx, "j:one", &got))
	assert.Equal(t, want, got, "GetJSON 应命中 L1")

	// 只在 L2 的 JSON 键 → GetJSON 走 L2 分支。
	require.NoError(t, rc.SetJSON(ctx, "j:two", rds8005Payload{Name: "l2", N: 4}, 0))
	var got2 rds8005Payload
	require.NoError(t, mlc.GetJSON(ctx, "j:two", &got2))
	assert.Equal(t, 4, got2.N)

	// L1 坏 JSON → 解析失败落回 L2 → L2 无此键 → ErrNotFound。
	require.NoError(t, mem.Set(ctx, "j:bad", "not-json", 0))
	var got3 rds8005Payload
	assert.Error(t, mlc.GetJSON(ctx, "j:bad", &got3))

	// MSetJSON:L1 存 JSON 文本 + L2 同步循环写原始值(MSetJSON 的 L2 写是
	// 同步 for 循环)。L2 值受 QUIRK-80-05-B 限制用标量。
	require.NoError(t, mlc.MSetJSON(ctx, map[string]interface{}{
		"j:m1": "m1v",
		"j:m2": "m2v",
	}, 30*time.Second))
	require.True(t, mr.Exists("j:m1"), "MSetJSON 的 L2 写为同步循环,应立即可见")
	require.True(t, mr.Exists("j:m2"))

	// MGetJSON:L1 命中 + L2 fallback 回填 + 缺失键缺席,三态同测。
	// j:m1 在 L1 存的是 JSON 文本 `"m1v"`(带引号)→ 解析回 "m1v";
	// j:two 只在 L2 → fallback 回填。
	res, err := mlc.MGetJSON(ctx, "j:m1", "j:two", "j:absent")
	require.NoError(t, err)
	require.Contains(t, res, "j:m1")
	assert.Equal(t, "m1v", res["j:m1"])
	require.Contains(t, res, "j:two")
	assert.NotContains(t, res, "j:absent")

	// L2 fallback 后 L1 应有回填(j:two 原本只在 L2)。
	_, err = mem.Get(ctx, "j:two")
	require.NoError(t, err, "MGetJSON 的 L2 fallback 应回填 L1")

	// 空键集 → 空 map;空 data → no-op。
	empty, err := mlc.MGetJSON(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)
	require.NoError(t, mlc.MSetJSON(ctx, map[string]interface{}{}, time.Second))
}

// TestMlc8005_WithWriter_Close:起 L2Writer 的构造器必须 Close 收口
// (79-R7);Close 后 IsL2WriterEnabled 翻转。
func TestMlc8005_WithWriter_Close(t *testing.T) {
	ctx := context.Background()
	rc, mr := newRds8005(t)
	mem := NewMemoryCache(1000, time.Minute)
	t.Cleanup(func() { _ = mem.Close() })

	cfg := &L2WriterConfig{
		WorkerCount:          1,
		QueueSize:            16,
		EnqueueTimeout:       time.Second,
		WriteTimeout:         2 * time.Second,
		FallbackWriteTimeout: time.Second,
	}
	mlc := NewMultiLevelCacheWithWriter(mem, rc, cfg)
	t.Cleanup(func() { _ = mlc.Close() })

	assert.True(t, mlc.IsL2WriterEnabled(), "WithWriter 构造器应启用 L2Writer")
	assert.NotNil(t, mlc.GetL2WriterStats())
	assert.GreaterOrEqual(t, mlc.GetL2WriterQueueSize(), 0)

	// writer 模式:Set 同步写 L1 + 异步入队写 L2。
	require.NoError(t, mlc.Set(ctx, "w:k", "wv", 0))
	v, err := mlc.Get(ctx, "w:k")
	require.NoError(t, err)
	assert.Equal(t, "wv", v)
	assert.Eventually(t, func() bool { return mr.Exists("w:k") }, 2*time.Second, 10*time.Millisecond,
		"L2Writer 应异步把值写入 L2")

	require.NoError(t, mlc.Close())
	assert.False(t, mlc.IsL2WriterEnabled(), "Close 后 worker 已停,IsL2WriterEnabled 应翻转")
}

// TestMlc8005_ConstructorFamily_Close:其余三个起 worker 的构造器
// (NewMultiLevelCache / WithRetry / WithRetryAndWriter)各自装配 + Close
// 收口(79-R7)。各构造器独享 mem/rc,避免 Close 互相踩。
func TestMlc8005_ConstructorFamily_Close(t *testing.T) {
	ctx := context.Background()

	// NewMultiLevelCache:默认 L2Writer。
	rc1, mr1 := newRds8005(t)
	mem1 := NewMemoryCache(100, time.Minute)
	mlc1 := NewMultiLevelCache(mem1, rc1)
	t.Cleanup(func() { _ = mlc1.Close() })
	assert.True(t, mlc1.IsL2WriterEnabled())
	require.NoError(t, mlc1.Set(ctx, "c1:k", "v", 0))
	assert.Eventually(t, func() bool { return mr1.Exists("c1:k") }, 2*time.Second, 10*time.Millisecond)

	// NewMultiLevelCacheWithRetry:默认 L2Writer + retry worker。
	rc2, _ := newRds8005(t)
	mem2 := NewMemoryCache(100, time.Minute)
	mlc2 := NewMultiLevelCacheWithRetry(mem2, rc2, DefaultRetryConfig(), 1)
	t.Cleanup(func() { _ = mlc2.Close() })
	assert.True(t, mlc2.IsRetryEnabled())
	assert.True(t, mlc2.IsL2WriterEnabled())

	// NewMultiLevelCacheWithRetryAndWriter:自定义两者。
	// QUIRK-80-05-A:MultiLevelCache.Close 非 idempotent —— retryWorker 非 nil 时
	// AsyncRetryWorker.Stop() 直接 close(closeChan) 无 CAS 守卫,二次 Close panic
	// (l2Writer.Stop 有 CompareAndSwap 守卫,retry 无)。测试侧用 guarded cleanup
	// 收口,只锁不修(零生产改动纪律)。
	rc3, _ := newRds8005(t)
	mem3 := NewMemoryCache(100, time.Minute)
	mlc3 := NewMultiLevelCacheWithRetryAndWriter(mem3, rc3, DefaultRetryConfig(), 1, DefaultL2WriterConfig())
	closed3 := false
	t.Cleanup(func() {
		if !closed3 {
			closed3 = true
			_ = mlc3.Close()
		}
	})
	assert.True(t, mlc3.IsRetryEnabled())
	require.NoError(t, mlc3.Close())
	closed3 = true
	assert.False(t, mlc3.IsL2WriterEnabled(), "显式 Close 后应停机")
}

// TestMlc8005_RetryToggle_Stats:运行时 EnableRetry + GetRetryStats 两态;
// Simple 模式下 L2Writer 统计为 nil/0。
func TestMlc8005_RetryToggle_Stats(t *testing.T) {
	ctx := context.Background()
	mlc, _, _, _ := newMlc8005(t, true)

	// Simple 模式:无 L2Writer。
	assert.Nil(t, mlc.GetL2WriterStats(), "未启用 L2Writer 时统计应为 nil")
	assert.Equal(t, 0, mlc.GetL2WriterQueueSize())
	assert.False(t, mlc.IsL2WriterEnabled())

	// 重试未启用态。
	assert.False(t, mlc.IsRetryEnabled())
	stats := mlc.GetRetryStats()
	assert.Equal(t, false, stats["retry_enabled"])

	// 运行时启用 → enabled;重复启用走早退分支(跳过)。
	mlc.EnableRetry(DefaultRetryConfig(), 1)
	assert.True(t, mlc.IsRetryEnabled())
	// 第二次 EnableRetry 走 "已启用，跳过" 早退分支(不 panic、不改状态)。
	mlc.EnableRetry(DefaultRetryConfig(), 1)
	stats = mlc.GetRetryStats()
	assert.Equal(t, true, stats["retry_enabled"])

	// L1/L2 GetStats 聚合(RedisCache 与 MemoryCache 均实现 GetStats)。
	agg, err := mlc.GetStats(ctx)
	require.NoError(t, err)
	assert.Contains(t, agg, "l2")
}

// TestMlc8005_DirectRedis_Accessors:DirectRedis* 走 redisCacheInterface
// 断言成功(Redis L2)与失败(MemoryCache L2)两分支。
func TestMlc8005_DirectRedis_Accessors(t *testing.T) {
	ctx := context.Background()
	mlc, rc, _, _ := newMlc8005(t, true)

	require.NoError(t, rc.Set(ctx, "dr:1", "dv", 0))
	keys, err := mlc.DirectRedisKeys(ctx, "dr:*")
	require.NoError(t, err)
	assert.Equal(t, []string{"dr:1"}, keys)

	v, err := mlc.DirectRedisGet(ctx, "dr:1")
	require.NoError(t, err)
	assert.Equal(t, "dv", v)

	v, err = mlc.DirectRedisGet(ctx, "dr:absent")
	require.NoError(t, err, "redis.Nil 应降级为空串 + nil")
	assert.Empty(t, v)

	ttl, err := mlc.DirectRedisTTL(ctx, "dr:1")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl, "无过期键 TTL = -1")

	require.NoError(t, mlc.DirectRedisXAdd(ctx, "stream:8005", map[string]interface{}{"k": "v"}))
	assert.NotNil(t, mlc.GetRedisClient())

	// L2 非 Redis(MemoryCache 不实现 getClient/getPrefix)→ 降级分支。
	memOnly, _, _, _ := newMlc8005(t, false)
	keys, err = memOnly.DirectRedisKeys(ctx, "*")
	require.NoError(t, err)
	assert.Empty(t, keys)
	v, err = memOnly.DirectRedisGet(ctx, "dr:1")
	require.NoError(t, err)
	assert.Empty(t, v)
	ttl, err = memOnly.DirectRedisTTL(ctx, "dr:1")
	require.NoError(t, err)
	assert.Zero(t, ttl)
	assert.Error(t, memOnly.DirectRedisXAdd(ctx, "s", map[string]interface{}{"k": "v"}),
		"L2 不支持 Redis 操作时应报错")
	assert.Nil(t, memOnly.GetRedisClient())
}

// TestMlc8005_KeysMerge_Levels_Flush:Keys 双级合并去重 + KeysByLevel
// 三分支 + FlushDB 双级清空。
func TestMlc8005_KeysMerge_Levels_Flush(t *testing.T) {
	ctx := context.Background()
	mlc, rc, mem, _ := newMlc8005(t, true)

	require.NoError(t, mem.Set(ctx, "km:l1", "v", 0))
	require.NoError(t, rc.Set(ctx, "km:l2", "v", 0))
	require.NoError(t, rc.Set(ctx, "km:both", "v", 0))
	require.NoError(t, mem.Set(ctx, "km:both", "v", 0))

	all, err := mlc.Keys(ctx, "km:*")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"km:l1", "km:l2", "km:both"}, all,
		"双级合并应去重后返回并集")

	l1Keys, err := mlc.KeysByLevel(ctx, "km:*", "l1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"km:l1", "km:both"}, l1Keys)

	l2Keys, err := mlc.KeysByLevel(ctx, "km:*", "l2")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"km:l2", "km:both"}, l2Keys)

	merged, err := mlc.KeysByLevel(ctx, "km:*", "all")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"km:l1", "km:l2", "km:both"}, merged, "默认分支应双级合并")

	require.NoError(t, mlc.FlushDB(ctx))
	all, err = mlc.Keys(ctx, "km:*")
	require.NoError(t, err)
	assert.Empty(t, all, "FlushDB 后双级应全空")
}

// TestMlc8005_Stats_Hash_IntDelegate:GetStats 聚合 + Hash/SetInt/GetInt
// 委托 L2(Redis)全往返。
func TestMlc8005_Stats_Hash_IntDelegate(t *testing.T) {
	ctx := context.Background()
	mlc, _, _, _ := newMlc8005(t, true)

	agg, err := mlc.GetStats(ctx)
	require.NoError(t, err)
	assert.Contains(t, agg, "l2", "Redis L2 实现 GetStats,聚合应含 l2 段")

	require.NoError(t, mlc.HSet(ctx, "h:8005", "field-a", "fa"))
	fv, err := mlc.HGet(ctx, "h:8005", "field-a")
	require.NoError(t, err)
	assert.Equal(t, "fa", fv)
	fields, err := mlc.HKeys(ctx, "h:8005")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"field-a"}, fields)
	all, err := mlc.HGetAll(ctx, "h:8005")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"field-a": "fa"}, all)
	require.NoError(t, mlc.HDel(ctx, "h:8005", "field-a"))
	all, err = mlc.HGetAll(ctx, "h:8005")
	require.NoError(t, err)
	assert.Empty(t, all)

	require.NoError(t, mlc.SetInt(ctx, "i:8005", 7, 0))
	n, err := mlc.GetInt(ctx, "i:8005")
	require.NoError(t, err)
	assert.Equal(t, 7, n)
}
