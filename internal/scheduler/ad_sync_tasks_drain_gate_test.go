//go:build !skip_db_tests
// +build !skip_db_tests

// quick 261021-iqq（2026-10-21）：方案A热重载的守护测试。
//   1. drainGate.shouldDrain 门控语义（零值首排 / 未到期跳过 / 到期放行 / 间隔变化重置）；
//   2. taskRegistry 防删守护 —— ad_data_sync / ad_account_pool_recover_breakers 必须已注册，
//      jobService 的 InvokeTarget 白名单校验（F-08）依赖它；误删注册代码时本文件必红。
package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDrainGate_FirstTickDrainsImmediately 零值 lastDrain：启动后首个 tick 立即排水
//（与旧实现"启动即挂 cron、首个刻度排水"行为一致）。
func TestDrainGate_FirstTickDrainsImmediately(t *testing.T) {
	g := &drainGate{}
	now := time.Now()
	assert.True(t, g.shouldDrain(now, 30), "首个 tick 必须立即放行")
}

// TestDrainGate_ThrottlesWithinInterval 距上次排水不足 interval → 跳过；到点 → 放行。
func TestDrainGate_ThrottlesWithinInterval(t *testing.T) {
	g := &drainGate{}
	t0 := time.Now()
	require.True(t, g.shouldDrain(t0, 300))

	// +30s（一个 tick 后）< 300s 间隔 → 9 个 tick 全部跳过
	for i := 1; i <= 9; i++ {
		assert.False(t, g.shouldDrain(t0.Add(time.Duration(i)*30*time.Second), 300),
			"interval=300 时 %ds 处的 tick 必须被门控跳过", i*30)
	}
	// +300s 整 → 放行
	assert.True(t, g.shouldDrain(t0.Add(300*time.Second), 300))
}

// TestDrainGate_IntervalChangeResetsAndLogs 间隔变化：立即重置基线（新配置下一 tick 生效）
// 且不再按旧间隔节流。
func TestDrainGate_IntervalChangeResetsAndLogs(t *testing.T) {
	g := &drainGate{}
	t0 := time.Now()
	require.True(t, g.shouldDrain(t0, 30))
	// +10s < 30s 会被旧间隔跳过；改成 5s 后必须放行（重置基线语义）
	assert.True(t, g.shouldDrain(t0.Add(10*time.Second), 5),
		"间隔变化应重置门控并立即放行")
	// 新间隔 5s 内的下一个 tick 跳过
	assert.False(t, g.shouldDrain(t0.Add(10*time.Second).Add(3*time.Second), 5))
}

// TestDrainGate_NonPositiveIntervalFallsBack interval<=0 兜底默认 30s，不得 panic / 死锁。
func TestDrainGate_NonPositiveIntervalFallsBack(t *testing.T) {
	g := &drainGate{}
	t0 := time.Now()
	assert.True(t, g.shouldDrain(t0, 0))
	assert.False(t, g.shouldDrain(t0.Add(10*time.Second), -5), "兜底 30s 内跳过")
	assert.True(t, g.shouldDrain(t0.Add(30*time.Second), 0))
}

// TestADSyncTaskRegistry_GuardTasksPresent taskRegistry 防删守护（quick 261021-iqq PLAN task2）：
// 这两个 taskType 是 jobService InvokeTarget 白名单（F-08）的合法值，也是运维在
// 定时任务 UI 里配置全量同步 / 熔断恢复的前提。StartADSyncScheduler 之外的任何重构
// 若误删 RegisterADSyncTasks 的注册代码，此处必红。
func TestADSyncTaskRegistry_GuardTasksPresent(t *testing.T) {
	s := NewScheduler(nil)
	RegisterADSyncTasks(s)
	RegisterADAccountPoolTasks(s)

	assert.True(t, s.IsTaskRegistered("ad_data_sync"),
		"ad_data_sync 必须已注册（InvokeTarget 白名单依赖）")
	assert.True(t, s.IsTaskRegistered("ad_account_pool_recover_breakers"),
		"ad_account_pool_recover_breakers 必须已注册（InvokeTarget 白名单依赖）")
}
