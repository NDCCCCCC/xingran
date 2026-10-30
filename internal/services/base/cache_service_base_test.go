// =====================================================================
// Phase 92-01 Task 3: base 缓存抽象双装配测试(MemoryCache + miniredis)。
//
// 覆盖目标(CACHE-UNIFY-05 / D-09): GetOrSetJSON[T] 命中/穿透/错误透传、
// TTL 过期(mr.FastForward)、Invalidate/InvalidatePattern 失效、NoOp 透传
// (含 Pitfall 5 指针类型单向改善断言)、GetStats 读取、GetExpiration
// nil-config(typed-nil) → default 分支(Pitfall 1 端到端)。
//
// 关键纪律(照抄 data_cache_service_79_01_test.go 头部):
//   - 双装配 helper newBase92Redis / newBase92Memory,名字带 plan 后缀
//     (R5 防同目录重名)。
//   - t.Cleanup 单次 Close。
//   - 禁并行子测试(装配含后台清理 goroutine 与 miniredis 实例)。
//   - TTL 推进一律 miniredis mr.FastForward(R-1 纪律,禁裸 sleep)。
//   - package base_test(外部测试包): 测试需 import internal/services 与
//     internal/services/system 做双装配,in-package 会成环
//     (system→services root→base)。
//
// =====================================================================
package base_test

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// base92Sample GetOrSetJSON 往返断言用样例结构体。
type base92Sample struct {
	ID   int
	Name string
}

// newBase92Redis 装配 miniredis + RedisCache + DataCacheService + 适配 provider
// (真实 go-redis 握手,照抄 data_cache_service_79_01_test.go:58-70)。
func newBase92Redis(t *testing.T) (base.CacheProvider, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	host, portStr, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	rc, err := cache.NewRedisCache(&cache.CacheConfig{Host: host, Port: port}, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	return system.NewCacheProvider(services.NewDataCacheService(rc)), mr
}

// newBase92Memory 装配 MemoryCache + DataCacheService + 适配 provider
// (纯进程内,无外部依赖)。
func newBase92Memory(t *testing.T) base.CacheProvider {
	t.Helper()
	mc := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mc.Close() })
	return system.NewCacheProvider(services.NewDataCacheService(mc))
}

// base92Assemblies 双装配表驱动公共切片。
func base92Assemblies() []struct {
	name  string
	setup func(t *testing.T) base.CacheProvider
} {
	return []struct {
		name  string
		setup func(t *testing.T) base.CacheProvider
	}{
		{"MemoryCache", func(t *testing.T) base.CacheProvider { return newBase92Memory(t) }},
		{"RedisCache", func(t *testing.T) base.CacheProvider {
			p, _ := newBase92Redis(t)
			return p
		}},
	}
}

// TestBase92_GetOrSetJSON_Hit 预置缓存键后调用: query 闭包调用计数 == 0,
// 返回缓存值(命中反序列化路径)。
func TestBase92_GetOrSetJSON_Hit(t *testing.T) {
	for _, asm := range base92Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			p := asm.setup(t)

			// 预置: 首次调用写入缓存
			count := 0
			_, err := base.GetOrSetJSON(ctx, p, "base92:hit", time.Minute, func() (*base92Sample, error) {
				count++
				return &base92Sample{ID: 1, Name: "预置"}, nil
			})
			require.NoError(t, err)
			require.Equal(t, 1, count)

			// 命中: query 不再执行,返回缓存值
			got, err := base.GetOrSetJSON(ctx, p, "base92:hit", time.Minute, func() (*base92Sample, error) {
				count++
				return &base92Sample{ID: 2, Name: "不应出现"}, nil
			})
			require.NoError(t, err)
			assert.Equal(t, 1, count, "命中时 query 不得执行")
			require.NotNil(t, got)
			assert.Equal(t, base92Sample{ID: 1, Name: "预置"}, *got)
		})
	}
}

// TestBase92_GetOrSetJSON_MissThenCache 空缓存穿透: query 调用计数 == 1,
// 结果写入缓存(第二次调用计数不再增长)。
func TestBase92_GetOrSetJSON_MissThenCache(t *testing.T) {
	for _, asm := range base92Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			p := asm.setup(t)

			count := 0
			query := func() (*base92Sample, error) {
				count++
				return &base92Sample{ID: 42, Name: "穿透"}, nil
			}

			got, err := base.GetOrSetJSON(ctx, p, "base92:miss", time.Minute, query)
			require.NoError(t, err)
			assert.Equal(t, 1, count)
			require.NotNil(t, got)
			assert.Equal(t, base92Sample{ID: 42, Name: "穿透"}, *got)

			// 值已写入缓存: 第二次调用不再穿透
			again, err := base.GetOrSetJSON(ctx, p, "base92:miss", time.Minute, query)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "第二次调用应命中缓存,query 计数不再增长")
			require.NotNil(t, again)
			assert.Equal(t, base92Sample{ID: 42, Name: "穿透"}, *again)
		})
	}
}

// TestBase92_GetOrSetJSON_ErrorPassthrough query 返回 error: 错误透传
// (经底层包装,errors.Is 可溯源),且不写缓存。
func TestBase92_GetOrSetJSON_ErrorPassthrough(t *testing.T) {
	for _, asm := range base92Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			p := asm.setup(t)

			sentinel := errors.New("base92 查询失败")
			got, err := base.GetOrSetJSON(ctx, p, "base92:err", time.Minute,
				func() (*base92Sample, error) { return nil, sentinel })
			assert.Nil(t, got)
			require.Error(t, err)
			assert.True(t, errors.Is(err, sentinel), "query 错误应可经错误链溯源: %v", err)

			// 不写缓存
			exists, err := p.Exists(ctx, "base92:err")
			require.NoError(t, err)
			assert.False(t, exists, "query 出错时不得写缓存")
		})
	}
}

// TestBase92_TTLExpire_FastForward 写入后 mr.FastForward(ttl + 1s) 推进 TTL,
// 再调用触发穿透(query 计数 == 2)。miniredis 专用(FastForward 依赖 mock 时钟)。
func TestBase92_TTLExpire_FastForward(t *testing.T) {
	ctx := context.Background()
	p, mr := newBase92Redis(t)

	count := 0
	query := func() (*base92Sample, error) {
		count++
		return &base92Sample{ID: count, Name: "TTL"}, nil
	}

	ttl := 30 * time.Second
	_, err := base.GetOrSetJSON(ctx, p, "base92:ttl", ttl, query)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// 未过期: 命中
	_, err = base.GetOrSetJSON(ctx, p, "base92:ttl", ttl, query)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// FastForward 推进过 TTL: 过期 → 穿透
	mr.FastForward(ttl + time.Second)
	got, err := base.GetOrSetJSON(ctx, p, "base92:ttl", ttl, query)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "TTL 过期后应重新穿透")
	require.NotNil(t, got)
	assert.Equal(t, 2, got.ID)
}

// TestBase92_Invalidate_Keys 预置 N 个键,Invalidate 后全部 miss。
func TestBase92_Invalidate_Keys(t *testing.T) {
	for _, asm := range base92Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			p := asm.setup(t)

			keys := []string{"base92:inv:1", "base92:inv:2", "base92:inv:3"}
			for _, k := range keys {
				_, err := base.GetOrSetJSON(ctx, p, k, time.Minute,
					func() (*base92Sample, error) { return &base92Sample{ID: 1}, nil })
				require.NoError(t, err)
			}

			base.Invalidate(ctx, p, keys, "TEST92")

			for _, k := range keys {
				exists, err := p.Exists(ctx, k)
				require.NoError(t, err)
				assert.False(t, exists, "Invalidate 后键 %s 应失效", k)
			}
		})
	}
}

// TestBase92_InvalidatePattern 预置匹配 pattern 的键 + 不匹配键:
// 调用后匹配键失效、不匹配键保留。
func TestBase92_InvalidatePattern(t *testing.T) {
	for _, asm := range base92Assemblies() {
		t.Run(asm.name, func(t *testing.T) {
			ctx := context.Background()
			p := asm.setup(t)

			matching := []string{"base92:pat:a:1", "base92:pat:a:2"}
			keep := "base92:other:keep"
			for _, k := range append(append([]string{}, matching...), keep) {
				_, err := base.GetOrSetJSON(ctx, p, k, time.Minute,
					func() (*base92Sample, error) { return &base92Sample{ID: 1}, nil })
				require.NoError(t, err)
			}

			base.InvalidatePattern(ctx, p, []string{"base92:pat:a:*"}, "TEST92")

			for _, k := range matching {
				exists, err := p.Exists(ctx, k)
				require.NoError(t, err)
				assert.False(t, exists, "匹配键 %s 应失效", k)
			}
			exists, err := p.Exists(ctx, keep)
			require.NoError(t, err)
			assert.True(t, exists, "不匹配键应保留")
		})
	}
}

// TestBase92_Invalidate_NilProvider nil provider: 不 panic,静默返回
// (D-04 nil 防护内聚于底层)。
func TestBase92_Invalidate_NilProvider(t *testing.T) {
	ctx := context.Background()
	assert.NotPanics(t, func() {
		base.Invalidate(ctx, nil, []string{"base92:nil:1"}, "TEST92")
	})
	assert.NotPanics(t, func() {
		base.InvalidatePattern(ctx, nil, []string{"base92:nil:*"}, "TEST92")
	})
}

// TestBase92_NoOp_Passthrough NoOpCacheProvider 透传: 每次调用都执行 query
// 且不写缓存; T 为指针类型时结果正确回填(Pitfall 5: &result 双指针
// assignable 成立,不再静默丢零值——单向改善断言)。
func TestBase92_NoOp_Passthrough(t *testing.T) {
	ctx := context.Background()
	noop := &base.NoOpCacheProvider{}

	count := 0
	query := func() (*base92Sample, error) {
		count++
		return &base92Sample{ID: 7, Name: "noop"}, nil
	}

	got, err := base.GetOrSetJSON(ctx, noop, "base92:noop", time.Minute, query)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 第二次调用: NoOp 无缓存 → query 必调
	_, err = base.GetOrSetJSON(ctx, noop, "base92:noop", time.Minute, query)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "NoOp 每次调用都应执行 query")

	// Pitfall 5: 指针类型 T 结果正确回填(非零值)
	require.NotNil(t, got, "指针类型结果应正确回填,不得静默丢成 nil")
	assert.Equal(t, base92Sample{ID: 7, Name: "noop"}, *got)
}

// TestBase92_GetStats 经 miniredis 装配调用 p.GetStats(ctx) 返回非 nil CacheStats。
func TestBase92_GetStats(t *testing.T) {
	ctx := context.Background()
	p, _ := newBase92Redis(t)

	require.NoError(t, p.Delete(ctx, "base92:stats:pre")) // GetStats 前清理潜在残留
	_, err := base.GetOrSetJSON(ctx, p, "base92:stats", time.Minute,
		func() (*base92Sample, error) { return &base92Sample{ID: 1}, nil })
	require.NoError(t, err)

	stats, err := p.GetStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats, "GetStats 应返回非 nil CacheStats")
	assert.Positive(t, stats.KeyCount)
}

// TestBase92_GetExpiration_NilConfig GetExpiration nil-config 分支:
// 零值 CacheServiceBase 返回 defaultVal; typed-nil
// (*services.CacheConfigService)(nil) 同样返回 defaultVal 且不 panic
// (Pitfall 1 nil-receiver 防护端到端验证)。
func TestBase92_GetExpiration_NilConfig(t *testing.T) {
	// 零值(untyped nil 接口字段)
	var zero base.CacheServiceBase
	assert.Equal(t, 5*time.Minute, zero.GetExpiration("cache.config.x", 5*time.Minute))

	// typed-nil: 具体指针 nil 装入接口 → if Config != nil 判定为非 nil,
	// 进入 GetDurationWithDefault → nil-receiver 防护返回 default(不 panic)
	typedNil := base.CacheServiceBase{Config: (*services.CacheConfigService)(nil)}
	assert.NotPanics(t, func() {
		got := typedNil.GetExpiration("cache.config.x", 7*time.Minute)
		assert.Equal(t, 7*time.Minute, got, "typed-nil config 应回退 defaultVal")
	})
}
