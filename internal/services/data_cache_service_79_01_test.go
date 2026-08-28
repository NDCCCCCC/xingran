package services

// =====================================================================
// Phase 79-01 Task 1: DataCacheService 全方法双装配测试(MemoryCache + miniredis)
//
// 覆盖目标: data_cache_service.go 19.5% → ≥70%(基线 77 stmts / 62 unc,79-RESEARCH §2)。
//
// 关键纪律:
//   - 双装配 helper newDcs7901 / newDcs7901Redis,名字带 plan 后缀(R5 防同包重名),
//     供 79-02..79-06 同包复用。
//   - t.Cleanup 单次 Close。QUIRK-P1 状态更新:MemoryCache.Close() 已于 2026-08-27
//     经 quick commit 4282983 幂等化(stopOnce sync.Once 守卫),二次 Close 不再 panic;
//     本文件仍守单次 Close 纪律(plan 约定不变)。
//   - 禁 t.Parallel()(装配含后台清理 goroutine 与 miniredis 实例)。
//   - TTL 推进一律 miniredis mr.FastForward(R-1 纪律,禁裸 time.Sleep)。
//   - P0 #9 锁定:GetOrSet 为同步写缓存(禁裸 goroutine),以注释 + 断言防回退。
//
// QUIRK-79-01-A(就地记录,零生产改动):data_cache_service.go:68-70 的
// `data == "" → apperrors.CacheKeyNotFound()` 分支对既有生产装配不可达 —
// MemoryCache.Get(miss) 返回 (\"\", ErrNotFound)、RedisCache.Get(redis.Nil) 同样翻译为
// ErrNotFound(pkg/cache/redis.go:78-80),两个实现都不会返回 (\"\", nil)。
// plan interfaces 段『cache miss(data=="")→ CacheKeyNotFound』与实装不符;按 quirk 纪律
// 断言现行为 + 以接口合规 test double 驱动该分支,SUMMARY 记录待裁决。
// =====================================================================

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
)

// dcs7901Sample GetOrSet / Set / Get 往返断言用样例结构体。
type dcs7901Sample struct {
	ID   int
	Name string
	Tags []string
}

// newDcs7901 装配 DataCacheService + MemoryCache(纯进程内,无外部依赖)。
func newDcs7901(t *testing.T) (*DataCacheService, *cache.MemoryCache) {
	t.Helper()
	mc := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mc.Close() })
	return NewDataCacheService(mc), mc
}

// newDcs7901Redis 装配 DataCacheService + miniredis + RedisCache(真实 go-redis 握手)。
func newDcs7901Redis(t *testing.T) (*DataCacheService, *miniredis.Miniredis, *cache.RedisCache) {
	t.Helper()
	mr := miniredis.RunT(t)
	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	rc, err := cache.NewRedisCache(&cache.CacheConfig{Host: host, Port: port}, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	return NewDataCacheService(rc), mr, rc
}

// dcs7901Assemblies 双装配表驱动公共切片。
func dcs7901Assemblies() []struct {
	name  string
	setup func(t *testing.T) *DataCacheService
} {
	return []struct {
		name  string
		setup func(t *testing.T) *DataCacheService
	}{
		{"MemoryCache", func(t *testing.T) *DataCacheService {
			svc, _ := newDcs7901(t)
			return svc
		}},
		{"RedisCache", func(t *testing.T) *DataCacheService {
			svc, _, _ := newDcs7901Redis(t)
			return svc
		}},
	}
}

// TestDcs7901_SetGetRoundTrip 两装配表驱动:结构体值 → Set(JSON 序列化) → Get(反序列化) 往返。
func TestDcs7901_SetGetRoundTrip(t *testing.T) {
	want := dcs7901Sample{ID: 42, Name: "往返样例", Tags: []string{"a", "b"}}
	for _, asm := range dcs7901Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			svc := asm.setup(t)
			require.NoError(t, svc.Set(ctx, "dcs7901:rt", want, 10*time.Minute))

			var got dcs7901Sample
			require.NoError(t, svc.Get(ctx, "dcs7901:rt", &got))
			assert.Equal(t, want, got)
		})
	}
}

// dcs7901EmptyGetCache QUIRK-79-01-A 探针:按 cache.Cache 接口契约返回 ("", nil) 的
// 合规实现,用于驱动 DataCacheService.Get 的 data=="" → apperrors.CacheKeyNotFound()
// 分支(:68-70)。既有 MemoryCache/RedisCache 对 miss 一律返回 cache.ErrNotFound,
// 该分支对生产装配实际不可达(见文件头 QUIRK-79-01-A 记录,零生产改动)。
type dcs7901EmptyGetCache struct {
	cache.Cache
}

func (m *dcs7901EmptyGetCache) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

// TestDcs7901_Get_MissAndBadJSON 未命中键 → 现行为锁定:cache.ErrNotFound 包装错误
// (QUIRK-79-01-A:生产缓存 miss 返回 ErrNotFound 而非 ("", nil),故
// apperrors.CacheKeyNotFound() 分支对 Memory/Redis 装配不可达,仅接口合规 double 可驱动);
// 预置非 JSON 字符串 → Unmarshal 错误分支。
func TestDcs7901_Get_MissAndBadJSON(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDcs7901(t)

	var dest dcs7901Sample
	err := svc.Get(ctx, "dcs7901:no-such-key", &dest)
	require.Error(t, err, "缓存未命中必须报错")
	assert.True(t, errors.Is(err, cache.ErrNotFound),
		"未命中应包装 cache.ErrNotFound(现行为锁定), got %v", err)
	assert.NotContains(t, err.Error(), "缓存键不存在",
		"QUIRK-79-01-A:生产装配 miss 不会走到 apperrors.CacheKeyNotFound() 分支")

	// QUIRK-79-01-A 探针:接口合规 double 返回 ("", nil) → 命中 :69 CacheKeyNotFound 分支
	svcEmpty := NewDataCacheService(&dcs7901EmptyGetCache{})
	err = svcEmpty.Get(ctx, "dcs7901:empty-val", &dest)
	require.Error(t, err)
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr, "(\"\", nil) 应走 CacheKeyNotFound 分支, got %v", err)
	assert.Equal(t, apperrors.CodeCacheKeyNotFound, appErr.Code)

	// 预置非 JSON 字符串值 → json.Unmarshal 错误分支
	require.NoError(t, svc.cache.Set(ctx, "dcs7901:bad-json", "not-json", time.Minute))
	err = svc.Get(ctx, "dcs7901:bad-json", &dest)
	require.Error(t, err)
	var synErr *json.SyntaxError
	require.ErrorAs(t, err, &synErr, "坏 JSON 应走 Unmarshal 错误分支, got %v", err)
}

// TestDcs7901_Set_MarshalFail value 传不可 JSON 序列化的 chan → "序列化失败"。
func TestDcs7901_Set_MarshalFail(t *testing.T) {
	for _, asm := range dcs7901Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			svc := asm.setup(t)
			err := svc.Set(ctx, "dcs7901:chan", make(chan int), time.Minute)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "序列化失败")
		})
	}
}

// TestDcs7901_GetOrSet_HitMissQueryError GetOrSet 三态:命中(query 不被调)/
// 未命中(query 被调 + dest 填充 + 二次命中)/ query 失败(错误包装上抛)。
func TestDcs7901_GetOrSet_HitMissQueryError(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDcs7901(t)

	// (1) 命中:预置键,query 内置 fail-fast sentinel,断言未被调用
	pre := dcs7901Sample{ID: 1, Name: "预置命中"}
	require.NoError(t, svc.Set(ctx, "dcs7901:gos-hit", pre, time.Minute))
	var hitDest dcs7901Sample
	require.NoError(t, svc.GetOrSet(ctx, "dcs7901:gos-hit", &hitDest, time.Minute,
		func() (any, error) {
			t.Fatal("命中路径不应执行 query")
			return nil, errors.New("sentinel: query must not run")
		}))
	assert.Equal(t, pre, hitDest, "命中路径 dest 应来自缓存而非 query")

	// (2) 未命中:query 被调,dest 被填充;二次调用 query 不再被调
	calls := 0
	var missDest dcs7901Sample
	require.NoError(t, svc.GetOrSet(ctx, "dcs7901:gos-miss", &missDest, time.Minute,
		func() (any, error) {
			calls++
			return dcs7901Sample{ID: 2, Name: "query 结果"}, nil
		}))
	assert.Equal(t, 1, calls, "未命中路径 query 应被调用一次")
	assert.Equal(t, 2, missDest.ID)

	var again dcs7901Sample
	require.NoError(t, svc.GetOrSet(ctx, "dcs7901:gos-miss", &again, time.Minute,
		func() (any, error) {
			calls++
			return nil, nil
		}))
	assert.Equal(t, 1, calls, "第二次调用应命中缓存,query 不再被调")
	assert.Equal(t, 2, again.ID, "二次 GetOrSet dest 应来自缓存")

	// (3) query 返回 error → "查询数据失败: %w" 包装,errors.Is 可解包
	qErr := errors.New("db down")
	err := svc.GetOrSet(ctx, "dcs7901:gos-err", &missDest, time.Minute,
		func() (any, error) { return nil, qErr })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询数据失败")
	assert.True(t, errors.Is(err, qErr), "错误应以 %w 包装,支持 errors.Is 解包")
}

// TestDcs7901_GetOrSet_SyncWriteLocks_P0_9 未命中路径返回后缓存立即可见。
//
// P0 #9 锁定:GetOrSet 为同步写缓存(context.Background(),禁裸 goroutine)。
// 若有人改回异步写,未命中路径返回后 Exists 必为 false → 本用例红。
// 两装配各验一遍:MemoryCache(纯同步)与 RedisCache(go-redis 同步命令)。
func TestDcs7901_GetOrSet_SyncWriteLocks_P0_9(t *testing.T) {
	for _, asm := range dcs7901Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			svc := asm.setup(t)

			var dest dcs7901Sample
			require.NoError(t, svc.GetOrSet(ctx, "dcs7901:p09", &dest, 10*time.Minute,
				func() (any, error) {
					return dcs7901Sample{ID: 7, Name: "sync-write"}, nil
				}))

			// P0 #9 锁定断言 1:GetOrSet 返回后键必须已存在(同步写,无裸 goroutine 竞态窗口)
			exists, err := svc.Exists(ctx, "dcs7901:p09")
			require.NoError(t, err)
			assert.True(t, exists, "P0 #9 锁定:GetOrSet 返回后缓存必须已同步写入(禁裸 goroutine 异步写)")

			// P0 #9 锁定断言 2:写入值可立即被 Get 读回
			var round dcs7901Sample
			require.NoError(t, svc.Get(ctx, "dcs7901:p09", &round))
			assert.Equal(t, 7, round.ID)
			assert.Equal(t, "sync-write", round.Name)
		})
	}
}

// TestDcs7901_DeleteByPattern_HitAndEmpty 预置 3 键(2 命中模式)→ 删除后仅剩 1 键;
// 模式无匹配 → 返回 nil 且不 panic(:123 len==0 分支)。
func TestDcs7901_DeleteByPattern_HitAndEmpty(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDcs7901(t)

	require.NoError(t, svc.Set(ctx, "dcs7901:pat:a", "va", time.Minute))
	require.NoError(t, svc.Set(ctx, "dcs7901:pat:b", "vb", time.Minute))
	require.NoError(t, svc.Set(ctx, "dcs7901:other:c", "vc", time.Minute))

	require.NoError(t, svc.DeleteByPattern(ctx, "dcs7901:pat:*"))

	left, err := svc.cache.Keys(ctx, "*")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dcs7901:other:c"}, left,
		"按模式删除后应仅剩未命中模式的 1 键")

	// 空集分支:无匹配模式 → len(keys)==0 直接返回 nil,不 panic、不走 MDelete
	require.NoError(t, svc.DeleteByPattern(ctx, "dcs7901:nothing:*"))
	left, err = svc.cache.Keys(ctx, "*")
	require.NoError(t, err)
	assert.Len(t, left, 1, "空集删除不应影响现存键")
}

// TestDcs7901_MGet_MDelete_Partial 3 键删 1 → MGet 结果 map 只含现存 2 键
// (缺失键不出现,非空串占位);MDelete 后 Exists 全 false。
func TestDcs7901_MGet_MDelete_Partial(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDcs7901(t)

	for i, v := range []string{"m0", "m1", "m2"} {
		require.NoError(t, svc.Set(ctx, "dcs7901:mg:"+strconv.Itoa(i), v, time.Minute))
	}

	// 走 svc.Delete 透传分支删 1 键
	require.NoError(t, svc.Delete(ctx, "dcs7901:mg:1"))

	got, err := svc.MGet(ctx, "dcs7901:mg:0", "dcs7901:mg:1", "dcs7901:mg:2")
	require.NoError(t, err)
	assert.Len(t, got, 2, "MGet 结果应只含现存 2 键")
	_, hasMissing := got["dcs7901:mg:1"]
	assert.False(t, hasMissing, "缺失键不得出现在结果 map(非空串占位)")
	assert.Equal(t, `"m0"`, got["dcs7901:mg:0"], "MGet 返回 Set 写入的 JSON 字符串")

	require.NoError(t, svc.MDelete(ctx, "dcs7901:mg:0", "dcs7901:mg:2"))
	for _, key := range []string{"dcs7901:mg:0", "dcs7901:mg:2"} {
		exists, err := svc.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists, "MDelete 后 %s 应不存在", key)
	}
}

// TestDcs7901_TTL_Ops SetTTL 后 GetTTL>0;miniredis 变体用 mr.FastForward 推进过期
// 后断言 miss(R-1 纪律:禁裸 time.Sleep 推 TTL)。MemoryCache 另验无过期键 TTL=-1。
func TestDcs7901_TTL_Ops(t *testing.T) {
	ctx := context.Background()

	// MemoryCache:SetTTL(Expire) 改写 TTL 可观察(从 30 分钟档抬升到 1 小时档)
	svcMem, _ := newDcs7901(t)
	require.NoError(t, svcMem.Set(ctx, "dcs7901:ttl-mem", "v", 30*time.Minute))
	require.NoError(t, svcMem.SetTTL(ctx, "dcs7901:ttl-mem", time.Hour))
	ttlMem, err := svcMem.GetTTL(ctx, "dcs7901:ttl-mem")
	require.NoError(t, err)
	assert.Greater(t, ttlMem, 30*time.Minute, "SetTTL 应把 TTL 抬过原 30 分钟档")

	// 无过期键 → TTL 返回 -1(永不过期语义)
	require.NoError(t, svcMem.Set(ctx, "dcs7901:ttl-forever", "v", 0))
	ttlForever, err := svcMem.GetTTL(ctx, "dcs7901:ttl-forever")
	require.NoError(t, err)
	assert.Negative(t, ttlForever, "永不过期键 TTL 应为 -1")

	// miniredis:FastForward 推进 2 小时 → 键过期 → Get 返回 CacheKeyNotFound 语义
	svcRedis, mr, _ := newDcs7901Redis(t)
	require.NoError(t, svcRedis.Set(ctx, "dcs7901:ttl-redis", "v", time.Hour))
	ttlRedis, err := svcRedis.GetTTL(ctx, "dcs7901:ttl-redis")
	require.NoError(t, err)
	assert.Greater(t, ttlRedis, time.Duration(0))

	mr.FastForward(2 * time.Hour)

	exists, err := svcRedis.Exists(ctx, "dcs7901:ttl-redis")
	require.NoError(t, err)
	assert.False(t, exists, "FastForward 2h 后 1h TTL 键应已过期")

	var dest dcs7901Sample
	err = svcRedis.Get(ctx, "dcs7901:ttl-redis", &dest)
	require.Error(t, err, "过期键 Get 必须报错")
	assert.ErrorIs(t, err, cache.ErrNotFound,
		"过期键走 cache.ErrNotFound(QUIRK-79-01-A 现行为锁定)")
}

// TestDcs7901_GetStats 预置 2 键 → KeyCount==2 && Count==2(:165-173 实现口径)。
func TestDcs7901_GetStats(t *testing.T) {
	for _, asm := range dcs7901Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			svc := asm.setup(t)
			require.NoError(t, svc.Set(ctx, "dcs7901:stats:a", "1", time.Minute))
			require.NoError(t, svc.Set(ctx, "dcs7901:stats:b", "2", time.Minute))

			stats, err := svc.GetStats(ctx)
			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, 2, stats.KeyCount, "Keys(*) 计数应与预置键数一致")
			assert.Equal(t, int64(2), stats.Count, "Count 与 KeyCount 同源")
		})
	}
}

// TestDcs7901_GetExpiration_NilConfig 不注入 cacheConfig → GetExpiration 直接返回
// default(:56 nil 分支;注入分支由 Task 2 的 TestCcs7901_GetExpiration_Wired 收口)。
func TestDcs7901_GetExpiration_NilConfig(t *testing.T) {
	def := 42 * time.Minute
	for _, asm := range dcs7901Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			svc := asm.setup(t)
			assert.Nil(t, svc.cacheConfig, "新装配不应注入 cacheConfig")
			assert.Equal(t, def, svc.GetExpiration(CacheConfigDeptTree, def),
				"cacheConfig 为 nil 时应直接返回 default")
		})
	}
}

// TestDcs7901_KeyBuilder_TypeMatrix CacheKeyBuilder 的整型/无符号/浮点/布尔/default
// 类型分支矩阵(既有 TestCacheKeyBuilder_* 未覆盖的 case;只增不改)。
func TestDcs7901_KeyBuilder_TypeMatrix(t *testing.T) {
	type dcs7901Point struct{ X int }

	tests := []struct {
		name string
		in   any
		want string
	}{
		{"int8", int8(8), "kb:8"},
		{"int16", int16(1600), "kb:1600"},
		{"int32 负数", int32(-32), "kb:-32"},
		{"int64", int64(64), "kb:64"},
		{"uint", uint(1), "kb:1"},
		{"uint8", uint8(255), "kb:255"},
		{"uint16", uint16(65535), "kb:65535"},
		{"uint32", uint32(4294967295), "kb:4294967295"},
		{"uint64", uint64(18446744073709551615), "kb:18446744073709551615"},
		{"float32", float32(1.5), "kb:1.5"},
		{"bool false", false, "kb:false"},
		{"default 分支(结构体)", dcs7901Point{X: 5}, "kb:{5}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCacheKeyBuilder("kb").Build(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}
