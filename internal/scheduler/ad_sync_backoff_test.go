package scheduler

// Phase 134 / FIX-10：全量同步失败退避 / 节流单测。
//
// 核心验收（2026-11-08 开放项盘点 #A6 / REQUIREMENTS.md FIX-10）：
//   - 「连续失败不成风暴」：AD 持续不可用时，每 5min tick 不再每次都触发同步。
//     用状态机模拟 N 个连续 tick，断言被门控放行的次数远小于 tick 数，
//     且退避时长按倍率递增、封顶 max。
//   - 配置化：sys_config 旋钮（enabled/base/max/multiplier）逐项生效；
//     db=nil / 表缺失 / 非法值 → 回落默认，不 panic、不报错。
//   - 默认值不改变现有正常行为：从未失败 / 成功后清零 → 每个 tick 恒放行。
//   - 成功清零：一次成功即退出退避，恢复正常 3h 全量周期。

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ==================== 退避时长纯函数 ====================

func TestPhase134_BackoffDelay_Curve(t *testing.T) {
	cfg := defaultADSyncBackoffConfig() // base=300, max=10800, mult=2
	assert.Equal(t, time.Duration(0), adSyncBackoffDelay(0, cfg))
	assert.Equal(t, 300*time.Second, adSyncBackoffDelay(1, cfg))
	assert.Equal(t, 600*time.Second, adSyncBackoffDelay(2, cfg))
	assert.Equal(t, 1200*time.Second, adSyncBackoffDelay(3, cfg))
	assert.Equal(t, 2400*time.Second, adSyncBackoffDelay(4, cfg))
	// 封顶：300*2^n 超过 10800 后恒为 max
	assert.Equal(t, 10800*time.Second, adSyncBackoffDelay(20, cfg))
	assert.Equal(t, 10800*time.Second, adSyncBackoffDelay(100, cfg))
}

func TestPhase134_BackoffDelay_CustomConfig(t *testing.T) {
	cfg := adSyncBackoffConfig{Enabled: true, Base: 60, Max: 600, Multiplier: 3}
	assert.Equal(t, 60*time.Second, adSyncBackoffDelay(1, cfg))
	assert.Equal(t, 180*time.Second, adSyncBackoffDelay(2, cfg))
	assert.Equal(t, 540*time.Second, adSyncBackoffDelay(3, cfg))
	assert.Equal(t, 600*time.Second, adSyncBackoffDelay(4, cfg), "封顶 max")
}

func TestPhase134_BackoffDelay_DisabledAndEdge(t *testing.T) {
	// 显式关闭 → 恒 0（应急回归旧行为）
	off := defaultADSyncBackoffConfig()
	off.Enabled = false
	assert.Equal(t, time.Duration(0), adSyncBackoffDelay(5, off))

	// multiplier < 1：退避不得短于 base（避免越退越短）
	shrink := adSyncBackoffConfig{Enabled: true, Base: 300, Max: 10800, Multiplier: 0.5}
	assert.Equal(t, 300*time.Second, adSyncBackoffDelay(2, shrink))
	assert.Equal(t, 300*time.Second, adSyncBackoffDelay(10, shrink))

	// 极大失败次数不溢出 / 不 NaN
	assert.Equal(t, 10800*time.Second, adSyncBackoffDelay(1<<20, defaultADSyncBackoffConfig()))
}

// ==================== 门控状态机：连续失败不成风暴 ====================

// TestPhase134_BackoffGate_NoRetryStorm 核心 RED 断言：
// 模拟 3h 全量周期内持续 AD 故障、每 5min 一个 tick（共 36 个 tick）。
// 旧行为（无退避）= 36 次同步；新行为 = 门控放行次数被压到个位数，且退避逐次拉长。
func TestPhase134_BackoffGate_NoRetryStorm(t *testing.T) {
	cfg := defaultADSyncBackoffConfig()
	g := newADSyncBackoffGate()
	t0 := time.Now()
	const configID = "cfg-storm"
	const tick = 5 * time.Minute

	triggered := 0
	for i := 0; i < 36; i++ { // 3h / 5min = 36 ticks
		now := t0.Add(time.Duration(i) * tick)
		if !g.allowSync(configID, now, cfg) {
			continue // 被退避门控抑制，本 tick 不触发同步 = 不产生重试风暴
		}
		triggered++
		// 模拟同步持续失败
		g.record(configID, false, now, cfg)
	}

	assert.Less(t, triggered, 10,
		"连续失败 3h / 36 个 tick，实际触发同步次数必须远小于 tick 数（旧行为=36，即风暴）")
	assert.GreaterOrEqual(t, triggered, 1, "首次失败前必须至少放行一次（正常首次同步不受影响）")
	assert.Equal(t, triggered, g.consecutiveFailures(configID), "连续失败次数应等于实际触发次数")
	t.Logf("连续失败 3h：36 个 tick 中仅 %d 次触发同步（旧行为 36 次）", triggered)
}

// TestPhase134_BackoffGate_BackoffWindowGrows 退避窗口确实随连续失败递增（非固定间隔）。
func TestPhase134_BackoffGate_BackoffWindowGrows(t *testing.T) {
	cfg := defaultADSyncBackoffConfig()
	g := newADSyncBackoffGate()
	t0 := time.Now()
	const id = "cfg-grow"

	g.record(id, false, t0, cfg) // 第 1 次失败 → +300s
	assert.False(t, g.allowSync(id, t0.Add(299*time.Second), cfg), "第 1 次退避窗口内不得放行")
	assert.True(t, g.allowSync(id, t0.Add(300*time.Second), cfg), "第 1 次退避到期放行")

	g.record(id, false, t0.Add(300*time.Second), cfg) // 第 2 次失败 → +600s
	assert.False(t, g.allowSync(id, t0.Add(300*time.Second).Add(599*time.Second), cfg), "第 2 次退避应比第 1 次长")
	assert.True(t, g.allowSync(id, t0.Add(300*time.Second).Add(600*time.Second), cfg))
}

// TestPhase134_BackoffGate_SuccessClears 一次成功即清零退避 → 恢复正常周期。
func TestPhase134_BackoffGate_SuccessClears(t *testing.T) {
	cfg := defaultADSyncBackoffConfig()
	g := newADSyncBackoffGate()
	t0 := time.Now()
	const id = "cfg-recover"

	for i := 0; i < 5; i++ {
		g.record(id, false, t0.Add(time.Duration(i)*time.Hour), cfg)
	}
	assert.Equal(t, 5, g.consecutiveFailures(id))

	g.record(id, true, t0.Add(10*time.Hour), cfg)
	assert.Equal(t, 0, g.consecutiveFailures(id), "成功后必须清零连续失败")
	// 清零后每个 tick 恒放行（恢复正常 3h 全量周期，不改变既有行为）
	for i := 0; i < 10; i++ {
		assert.True(t, g.allowSync(id, t0.Add(10*time.Hour+time.Duration(i)*5*time.Minute), cfg))
	}
}

// TestPhase134_BackoffGate_DisabledAlwaysAllows 显式关闭退避（应急）→ 恒放行。
func TestPhase134_BackoffGate_DisabledAlwaysAllows(t *testing.T) {
	cfg := defaultADSyncBackoffConfig()
	cfg.Enabled = false
	g := newADSyncBackoffGate()
	t0 := time.Now()
	for i := 0; i < 20; i++ {
		g.record("cfg-off", false, t0.Add(time.Duration(i)*time.Minute), cfg)
		assert.True(t, g.allowSync("cfg-off", t0.Add(time.Duration(i)*time.Minute), cfg))
	}
	assert.Equal(t, 0, g.consecutiveFailures("cfg-off"), "关闭时不应记录退避状态")
}

// ==================== 配置读取（sys_config 热重载 + 兜底） ====================

func newBackoffConfigDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Config{}))
	return db
}

func seedBackoffConfig(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Create(&models.Config{ConfigKey: key, ConfigValue: value, ConfigName: "FIX-10 退避旋钮"}).Error)
}

func TestPhase134_LoadBackoffConfig_DefaultsAndFallback(t *testing.T) {
	// db=nil → 默认
	assert.Equal(t, defaultADSyncBackoffConfig(), loadADSyncBackoffConfig(nil))

	// 表缺失 → 默认（不读表、不报错、不 panic）
	dbNoTable, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	assert.Equal(t, defaultADSyncBackoffConfig(), loadADSyncBackoffConfig(dbNoTable))

	// 空表 → 默认
	db := newBackoffConfigDB(t)
	assert.Equal(t, defaultADSyncBackoffConfig(), loadADSyncBackoffConfig(db))

	// 非法值逐项回落默认（不影响其它合法项）
	seedBackoffConfig(t, db, adSyncBackoffKeyBase, "not-a-number")
	seedBackoffConfig(t, db, adSyncBackoffKeyMultiplier, "0")
	seedBackoffConfig(t, db, adSyncBackoffKeyMax, "-1")
	seedBackoffConfig(t, db, adSyncBackoffKeyEnabled, "maybe")
	got := loadADSyncBackoffConfig(db)
	assert.True(t, got.Enabled)
	assert.Equal(t, adSyncBackoffDefaultBase, got.Base)
	assert.Equal(t, adSyncBackoffDefaultMax, got.Max)
	assert.Equal(t, adSyncBackoffDefaultMultiplier, got.Multiplier)
}

func TestPhase134_LoadBackoffConfig_CustomValues(t *testing.T) {
	db := newBackoffConfigDB(t)
	seedBackoffConfig(t, db, adSyncBackoffKeyEnabled, "true")
	seedBackoffConfig(t, db, adSyncBackoffKeyBase, "120")
	seedBackoffConfig(t, db, adSyncBackoffKeyMax, "1800")
	seedBackoffConfig(t, db, adSyncBackoffKeyMultiplier, "1.5")

	got := loadADSyncBackoffConfig(db)
	assert.True(t, got.Enabled)
	assert.Equal(t, 120, got.Base)
	assert.Equal(t, 1800, got.Max)
	assert.Equal(t, 1.5, got.Multiplier)
	assert.Equal(t, 120*time.Second, adSyncBackoffDelay(1, got))
	assert.Equal(t, 180*time.Second, adSyncBackoffDelay(2, got))

	// enabled=false 关闭退避
	db2 := newBackoffConfigDB(t)
	seedBackoffConfig(t, db2, adSyncBackoffKeyEnabled, "false")
	assert.False(t, loadADSyncBackoffConfig(db2).Enabled)
}

func TestPhase134_LoadBackoffConfig_MaxBelowBaseIsClamped(t *testing.T) {
	db := newBackoffConfigDB(t)
	seedBackoffConfig(t, db, adSyncBackoffKeyBase, "600")
	seedBackoffConfig(t, db, adSyncBackoffKeyMax, "60") // 小于 base，语义倒置
	got := loadADSyncBackoffConfig(db)
	assert.Equal(t, 600, got.Base)
	assert.Equal(t, 600, got.Max, "max 不得小于 base")
}
