package scheduler

// Phase 134 / FIX-10：全量 AD 同步失败退避 / 节流（重试风暴治理）。
//
// 根因（2026-11-08 开放项盘点 #A6）：
//   - 全量 cron 每 5min tick 检查一次；sys_ad_config.last_sync_at **只在同步成功时**更新
//     （internal/services/addomain/sync.go:177-184）。
//   - 同步失败 → last_sync_at 停在旧值 → elapsed 持续增长 → 每个 5min tick 的
//     checkAndSyncADConfigs 都判定 shouldSync=true → 失败后每 5min 无限重试 = 重试风暴。
//
// 方案（配置化退避门控，零新依赖 / 零迁移 / 零表结构变更）：
//   - 内存状态机记录每个 config 的「连续失败次数」与「下次允许同步时间」；
//   - 连续失败 → 退避按 multiplier 指数递增（base → base*mult → ...，封顶 max）；
//   - 成功 → 清零（退出退避，恢复正常 3h 全量周期）；
//   - 旋钮走 sys_config（热重载，与 addomain.LoadDrainIntervalSeconds 同款 raw SQL、
//     双方言兼容）；db 为 nil / 表缺失 / 值非法一律回落默认值，绝不抛错阻断同步；
//   - **默认值不改变现有正常行为**：成功路径零影响；只有连续失败才叠加退避，
//     base 默认等于 tick 间隔（300s），max 默认等于 3h 全量间隔（10800s）。
//
// 为什么放独立文件：ad_sync_tasks.go 是本 Phase 的接线面；退避状态机是纯逻辑，
// 独立文件便于精确单测，且与 Phase 138 的 workorder_tasks.go 文件面完全隔离。

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ==================== 可配置旋钮（sys_config 热重载） ====================

const (
	// 旋钮 key 前缀。每 5min tick 读取一次，改后无需重启。
	adSyncBackoffKeyEnabled    = "ad_sync.retry_backoff.enabled"
	adSyncBackoffKeyBase       = "ad_sync.retry_backoff.base_seconds"
	adSyncBackoffKeyMax        = "ad_sync.retry_backoff.max_seconds"
	adSyncBackoffKeyMultiplier = "ad_sync.retry_backoff.multiplier"

	// 默认值。
	//   - base = 300s = 5min tick 间隔：第 1 次失败后至少跳过 1 个 tick；
	//   - max  = 10800s = 3h 全量间隔：退避不会比正常同步周期更长；
	//   - multiplier = 2：连续失败退避指数递增。
	adSyncBackoffDefaultBase       = 300
	adSyncBackoffDefaultMax        = 10800
	adSyncBackoffDefaultMultiplier = 2.0
)

// adSyncBackoffConfig 全量同步失败退避配置。
type adSyncBackoffConfig struct {
	Enabled    bool
	Base       int     // 首次失败退避秒数
	Max        int     // 退避封顶秒数
	Multiplier float64 // 连续失败递增倍率
}

// defaultADSyncBackoffConfig 返回默认配置（Enabled=true，但成功路径零影响）。
func defaultADSyncBackoffConfig() adSyncBackoffConfig {
	return adSyncBackoffConfig{
		Enabled:    true,
		Base:       adSyncBackoffDefaultBase,
		Max:        adSyncBackoffDefaultMax,
		Multiplier: adSyncBackoffDefaultMultiplier,
	}
}

// loadADSyncBackoffConfig 从 sys_config 读取退避旋钮（每 tick 一次，热重载）。
//
// 读不到 / 表缺失 / 值非法 → **逐项**回落默认值（不是整体失败），保证任何环境下
// 同步都不被配置读取阻断。db 为 nil（仅测试构造的 scheduler）直接返回默认。
// raw SQL 与 addomain.LoadDrainIntervalSeconds 同款：SQLite + PG 双方言兼容。
func loadADSyncBackoffConfig(db *gorm.DB) adSyncBackoffConfig {
	cfg := defaultADSyncBackoffConfig()
	if db == nil {
		return cfg
	}
	// 表缺失（sqlite 测试基座 / 未初始化环境）→ 直接回落默认，避免每个 5min tick
	// 都打一条 "no such table: sys_config" 的 GORM ERROR 噪音（零回归要求）。
	if !db.Migrator().HasTable("sys_config") {
		return cfg
	}
	read := func(key string) (string, bool) {
		var v string
		if err := db.Raw(
			"SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL",
			key,
		).Scan(&v).Error; err != nil || v == "" {
			return "", false
		}
		return v, true
	}

	if v, ok := read(adSyncBackoffKeyEnabled); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "0", "false", "no", "off":
			cfg.Enabled = false
		case "1", "true", "yes", "on":
			cfg.Enabled = true
		default:
			applogger.Warnf("[ad_sync:backoff] %s 值非法(%q)，回落默认 true", adSyncBackoffKeyEnabled, v)
		}
	}
	if v, ok := read(adSyncBackoffKeyBase); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			cfg.Base = n
		} else {
			applogger.Warnf("[ad_sync:backoff] %s 值非法(%q)，回落默认 %d", adSyncBackoffKeyBase, v, adSyncBackoffDefaultBase)
		}
	}
	if v, ok := read(adSyncBackoffKeyMax); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			cfg.Max = n
		} else {
			applogger.Warnf("[ad_sync:backoff] %s 值非法(%q)，回落默认 %d", adSyncBackoffKeyMax, v, adSyncBackoffDefaultMax)
		}
	}
	if v, ok := read(adSyncBackoffKeyMultiplier); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 && !math.IsInf(f, 0) && !math.IsNaN(f) {
			cfg.Multiplier = f
		} else {
			applogger.Warnf("[ad_sync:backoff] %s 值非法(%q)，回落默认 %.1f", adSyncBackoffKeyMultiplier, v, adSyncBackoffDefaultMultiplier)
		}
	}

	// 不变量：max 不得小于 base（否则封顶比首退还短，语义倒置）。
	if cfg.Max < cfg.Base {
		applogger.Warnf("[ad_sync:backoff] max(%d) < base(%d)，强制对齐 max=base", cfg.Max, cfg.Base)
		cfg.Max = cfg.Base
	}
	return cfg
}

// ==================== 退避时长纯函数 ====================

// adSyncBackoffDelay 计算第 n 次连续失败后的退避时长（纯函数，可单测）。
//
//   - !Enabled / n<=0 → 0（不施加退避，保持旧行为）；
//   - n=1 → base；n=2 → base*mult；n=3 → base*mult² ...；
//   - 封顶 max；multiplier<1 时至少不低于 base（避免退避比首次还短）；
//   - 用 float64 秒计算再转 Duration，避免 math.Pow 溢出 / time.Duration 溢出。
func adSyncBackoffDelay(consecutiveFailures int, cfg adSyncBackoffConfig) time.Duration {
	if !cfg.Enabled || consecutiveFailures <= 0 {
		return 0
	}
	base := time.Duration(cfg.Base) * time.Second
	max := time.Duration(cfg.Max) * time.Second
	if consecutiveFailures == 1 {
		return base
	}
	factor := math.Pow(cfg.Multiplier, float64(consecutiveFailures-1))
	secs := float64(cfg.Base) * factor
	if math.IsInf(secs, 0) || math.IsNaN(secs) || secs >= float64(cfg.Max) {
		return max
	}
	d := time.Duration(secs * float64(time.Second))
	if d < base {
		return base
	}
	return d
}

// ==================== 门控状态机 ====================

// adSyncBackoffState 单个 AD 配置的退避状态。
type adSyncBackoffState struct {
	consecutiveFailures int
	nextAllowedAt       time.Time
}

// adSyncBackoffGate 失败退避门控（并发安全：allowSync 在 cron 单 goroutine，
// record 在异步同步 goroutine 内，二者可能并发 → 用 mutex 保护）。
//
// 状态存内存、不落库：进程重启后退避重置为 0（可接受：重启本身就是运维动作，
// 且 last_sync_at 仍是旧值，下一 tick 会立即重试一次再进入退避）。不引入迁移/表。
type adSyncBackoffGate struct {
	mu     sync.Mutex
	states map[string]*adSyncBackoffState
}

func newADSyncBackoffGate() *adSyncBackoffGate {
	return &adSyncBackoffGate{states: make(map[string]*adSyncBackoffState)}
}

// allowSync 判断是否允许本 tick 真正触发同步。
//
//   - 无退避状态（从未失败 / 已成功清零）→ true；
//   - now >= nextAllowedAt → true（退避窗口已过）；
//   - 否则 false（退避窗口内，跳过本 tick，避免风暴）。
//
// cfg.Enabled=false 时恒 true（显式关闭退避 = 回归旧行为，便于应急）。
func (g *adSyncBackoffGate) allowSync(configID string, now time.Time, cfg adSyncBackoffConfig) bool {
	if !cfg.Enabled {
		return true
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	st, ok := g.states[configID]
	if !ok {
		return true
	}
	return !now.Before(st.nextAllowedAt)
}

// record 记录一次同步结果，推进状态机。
//   - success=true → 删除该 config 的退避状态（连续失败清零，恢复正常周期）；
//   - success=false → 连续失败次数 +1，计算并写入 nextAllowedAt = now + delay。
func (g *adSyncBackoffGate) record(configID string, success bool, now time.Time, cfg adSyncBackoffConfig) {
	if !cfg.Enabled {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if success {
		delete(g.states, configID)
		return
	}
	st, ok := g.states[configID]
	if !ok {
		st = &adSyncBackoffState{}
		g.states[configID] = st
	}
	st.consecutiveFailures++
	delay := adSyncBackoffDelay(st.consecutiveFailures, cfg)
	st.nextAllowedAt = now.Add(delay)
	applogger.Warnf("[ad_sync:backoff] AD配置 %s 同步失败(连续第 %d 次)，退避 %.0f 秒，下次允许同步时间 %s",
		configID, st.consecutiveFailures, delay.Seconds(), st.nextAllowedAt.Format("2006-01-02 15:04:05"))
}

// consecutiveFailures 读取某 config 当前连续失败次数（排障 / 单测用）。
func (g *adSyncBackoffGate) consecutiveFailures(configID string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if st, ok := g.states[configID]; ok {
		return st.consecutiveFailures
	}
	return 0
}
