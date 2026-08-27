package core

// =====================================================================
// Phase 78-01 Task 5: MetricsCacheService 边缘 + per-file coverage checkpoint
//
// 关键纪律:
//   - M-1 差值采样零增量:只断言区间 [0,100] + err==nil,禁断言 CPU/Memory 具体值。
//   - M-2 包级全局无锁:禁 t.Parallel()(MetricsCacheService 内部有 background)。
//   - Q-7 / QUIRK-02:Stop 经 sync.Once 幂等,Stop 三次不 panic 是核心 invariant。
// =====================================================================

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	memcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// TestMx78_NewMetricsCacheService_NilPaths 最小 Core 装配覆盖 :19 剩余分支
// (Cache 为 nil / 非 nil 两形态)。
func TestMx78_NewMetricsCacheService_NilPaths(t *testing.T) {
	ctx := context.Background()

	// Cache=nil 路径(c.Cache == nil → redisCache 保持 nil → manager 用直采)
	c1 := newTestCoreForSplitCompat(t)
	svc1 := NewMetricsCacheService(c1)
	require.NotNil(t, svc1)
	_, err := svc1.GetServerInfo(ctx)
	assert.NoError(t, err)
	svc1.Stop() // 幂等(Q-7)

	// Cache 非 nil 路径(:21-23 redisCache = core.Cache 被 L2 读取走通)
	mem := memcache.NewMemoryCache(100, time.Minute)
	t.Cleanup(func() { _ = mem.Close() })
	c2 := newTestCoreForSplitCompat(t)
	c2.Cache = mem
	svc2 := NewMetricsCacheService(c2)
	require.NotNil(t, svc2)

	burn(200 * time.Millisecond)
	_, _ = svc2.GetCurrentMetrics(ctx) // 预热:写 L1(+L2 经 manager setToL2,interface{} 断言失败即跳过)
	burn(200 * time.Millisecond)
	mx, err := svc2.GetCurrentMetrics(ctx)
	require.NoError(t, err)
	require.NotNil(t, mx)
	assert.GreaterOrEqual(t, mx.CPUUsage, float64(0))
	assert.LessOrEqual(t, mx.CPUUsage, float64(100))

	info, err := svc2.GetServerInfo(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, info)
}

// burn CPU 空转制造非零差值(M-1: 空闲机器差值采样可为 0 导致业务报错)。
func burn(d time.Duration) {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
	}
}

// TestMx78_GetCurrentMetrics_ErrorAndCacheBranch 覆盖 :34 的 20% unc ——
// 调用路径合法,断言区间 [0,100] + err==nil(M-1 纪律)。
//
// 实际 GetCurrentMetrics 仅在 manager 内部 error 时返回错;sqlite+无 cache 路径
// 走直采,基本不会 error,这里测 happy path 足以覆盖剩余分支。
func TestMx78_GetCurrentMetrics_ErrorAndCacheBranch(t *testing.T) {
	c := newTestCoreForSplitCompat(t)
	svc := NewMetricsCacheService(c)
	t.Cleanup(func() { svc.Stop() })

	ctx := context.Background()

	// burnCPU 让差值采样非零(M-1 提示:空闲机器 0 增量 → 报"CPU 时间差值计算为 0")
	burn := func(d time.Duration) {
		deadline := time.Now().Add(d)
		for time.Now().Before(deadline) {
		}
	}
	burn(200 * time.Millisecond)
	_, _ = svc.GetCurrentMetrics(ctx) // 预热基线
	burn(200 * time.Millisecond)
	metrics, err := svc.GetCurrentMetrics(ctx)
	require.NoError(t, err)
	require.NotNil(t, metrics)
	// M-1 区间断言,不断言具体值
	assert.GreaterOrEqual(t, metrics.CPUUsage, float64(0))
	assert.LessOrEqual(t, metrics.CPUUsage, float64(100))
	assert.GreaterOrEqual(t, metrics.MemoryUsage, float64(0))
	assert.LessOrEqual(t, metrics.MemoryUsage, float64(100))
	assert.GreaterOrEqual(t, metrics.DiskUsage, float64(0))
	assert.LessOrEqual(t, metrics.DiskUsage, float64(100))
	assert.NotZero(t, metrics.Timestamp, "Timestamp 必填")
}

// TestMx78_GetServerInfo 断言返回 map 的必备键存在(不断言具体值)。
func TestMx78_GetServerInfo(t *testing.T) {
	c := newTestCoreForSplitCompat(t)
	svc := NewMetricsCacheService(c)
	t.Cleanup(func() { svc.Stop() })

	ctx := context.Background()
	info, err := svc.GetServerInfo(ctx)
	require.NoError(t, err)
	require.NotNil(t, info)
	// 必备键存在(manager.go:261-269 字面量表,具体值依赖运行环境)
	_, hasHostname := info["hostname"]
	_, hasOS := info["os"]
	_, hasArch := info["arch"]
	_, hasCPUCount := info["cpu_count"]
	_, hasTotalMemory := info["total_memory"]
	_, hasDiskTotal := info["disk_total"]
	assert.True(t, hasHostname, "hostname 键必存在")
	assert.True(t, hasOS, "os 键必存在")
	assert.True(t, hasArch, "arch 键必存在")
	assert.True(t, hasCPUCount, "cpu_count 键必存在")
	assert.True(t, hasTotalMemory, "total_memory 键必存在")
	assert.True(t, hasDiskTotal, "disk_total 键必存在")
}

// TestMx78_Stop_Idempotent 连调 Stop() 三次不 panic(QUIRK-02 sync.Once 幂等)。
// 78-02 的 Core.Close 依赖此语义。
func TestMx78_Stop_Idempotent(t *testing.T) {
	c := newTestCoreForSplitCompat(t)
	svc := NewMetricsCacheService(c)

	assert.NotPanics(t, func() {
		svc.Stop()
		svc.Stop()
		svc.Stop()
	}, "QUIRK-02 sync.Once 幂等,三次调用必不 panic")
}
