package core

// =====================================================================
// Phase 78-02: internal/core 装配层(core.go)Init/Close 全链测试。
//
// Task 1 = ROADMAP L211 明示的 **Init 深度探针实验**(Gap G1 / DQ3):
//   先证 core.Core 完整 Init 图能否用 glebarez sqlite + MemoryCache 拼出,
//   把「实际可达的子系统清单 / 失败点 / 耗时 / goroutine 收尾行为」落 SUMMARY;
//   后续 Task 2-5 在探针结论上叠加深度断言(D-78-02b: 探针通过条件是
//   「给出结论且不 hang」,而非「Init 必须返回 nil」)。
//
// 关键纪律:
//   - D-78-02a  sqlite 用 t.TempDir 文件库而非 :memory:(Init 链跨多 GORM 会话)
//   - D-78-02b  禁为过测改生产装配代码;fail-fast/warn 分支按现行为锚定
//   - D-78-03   Close hang 处置阶梯 (c): 测试侧硬超时守卫界定问题范围,
//               零新依赖(不引入 goleak),NumGoroutine 只做宽松回落断言
//   - R-7       所有启 goroutine 的阶段用 t.Cleanup 兜底收尾
//   - T-78-02-01 每处 Init/Close 调用均带硬超时守卫 channel
//   - 本文件禁 t.Parallel()(scheduler/addomain 全局状态 + 包级共享)
// =====================================================================

import (
	"encoding/base64"
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// init78SM4Key 合法 base64 的 16 字节测试密钥(T-78-02-04: 禁真实生产密钥)。
// 注意刻意避开仓库默认值 dGVzdC1zZWNyZXQxNiEhIQ==("test-secret16!!!"),
// 默认值仅告警放行的既有设计由 Task 5 的 TestInit78_New_ErrorPaths 单独覆盖。
var init78SM4Key = base64.StdEncoding.EncodeToString([]byte("init78-test-key!"))

const (
	init78InitTimeout    = 60 * time.Second // Init 整体硬超时(T-78-02-01)
	init78CloseTimeout   = 30 * time.Second // Close 整体硬超时(与生产 coreShutdownTimeout 同量级)
	init78PollInterval   = 50 * time.Millisecond
	init78WaitUpperBound = 5 * time.Second // 预热/goroutine 回落的轮询上限(R-1/R-7 禁裸 sleep)
)

// newInit78Config 全字段自造 *config.Config(sqlite + memory/redis 双分支)。
//
// - Server.Mode="debug": 避免 SKIP_AUTOMIGRATE 的 release fatal 守卫误触(CDX-H2);
//   个别用例需要 release 形态时显式改 cfg.Server.Mode 后再调 initDBAndData。
// - Database.Type="sqlite" + t.TempDir 文件库(D-78-02a)。
// - Cache.Type=cacheType("memory"/"redis");redis 时 Addr 来自调用方(miniredis.RunT)。
// - JWT/SM4Key 均为合法测试字面量(NewJWTManager/crypto.NewSM4Cipher 能构造)。
// - Server.SkipSetup=true: 跳过 InitData / 默认角色菜单 seed(耗时且非装配断言目标);
//   需要验证 seed 链的用例显式置 false。
// - RPA 默认 Disabled(initRPAServices 早退);RPA happy path 由 Task 4 直调覆盖。
func newInit78Config(t *testing.T, cacheType string, redisAddr string) *config.Config {
	t.Helper()
	cfg := &config.Config{}

	cfg.Server.Name = "core-init-78-test"
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0            // 不绑定端口,避免占用/T-78-02-03(禁打外部系统)
	cfg.Server.Mode = "debug"      // 见上
	cfg.Server.SkipSetup = true    // 跳过 seed,保组装测试确定性

	cfg.Database.Type = "sqlite"
	cfg.Database.Path = filepath.Join(t.TempDir(), "core_init.db") // D-78-02a 文件库

	cfg.Cache.Type = cacheType
	cfg.Cache.MaxSize = 1000
	cfg.Cache.CleanupTime = 60
	if cacheType == "redis" {
		host, portStr, err := net.SplitHostPort(redisAddr)
		require.NoError(t, err)
		port, err := strconv.Atoi(portStr)
		require.NoError(t, err)
		cfg.Cache.Host = host
		cfg.Cache.Port = port
	}

	cfg.JWT.SecretKey = "init78-test-secret-key-not-for-production-32bytes"
	cfg.JWT.AccessKeyExpire = 7200
	cfg.JWT.RefreshKeyExpire = 604800
	cfg.JWT.Issuer = "xingran-init78-test"

	cfg.Security.SM4Key = init78SM4Key

	return cfg
}

// init78CloseResult 记录一次受守卫的 Close 的观察结果。
type init78CloseResult struct {
	duration time.Duration
	hung     bool // 硬超时未返回
	panicked bool // panic(recover 捕获)
	panicVal any
}

// closeCore78Guarded 在带硬超时守卫的 goroutine 中执行 c.Close()(T-78-02-01),
// 并以 recover 兜住 panic(D-78-03 (c) 幂等探测期把 panic 变成可观察信号而非进程崩)。
// 函数必然在 init78CloseTimeout+ε 内返回;hang 时调用方决定 fail-fast 与否。
func closeCore78Guarded(c *Core) init78CloseResult {
	done := make(chan init78CloseResult, 1)
	go func() {
		res := init78CloseResult{}
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				res.panicked = true
				res.panicVal = r
			}
			res.duration = time.Since(start)
			done <- res
		}()
		c.Close()
	}()
	select {
	case res := <-done:
		return res
	case <-time.After(init78CloseTimeout):
		return init78CloseResult{hung: true}
	}
}

// openInit78Core 装配一个已成功 New 的 Core 并注册 t.Cleanup 兜底关闭(R-7)。
// 兜底 Close 也走守卫形态:即使用例自身已 Close 过,这里的三次幂等调用同样被验证。
func openInit78Core(t *testing.T, cfg *config.Config) *Core {
	t.Helper()
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		res := closeCore78Guarded(c)
		if res.hung {
			t.Logf("[CLEANUP] Close 硬超时 %v 未返回(hang)", init78CloseTimeout)
		}
	})
	return c
}

// initCore78Guarded 在带硬超时守卫的 goroutine 中执行 c.Init(),返回 (err, hung)。
func initCore78Guarded(c *Core) (err error, hung bool) {
	done := make(chan error, 1)
	go func() { done <- c.Init() }()
	select {
	case e := <-done:
		return e, false
	case <-time.After(init78InitTimeout):
		return nil, true
	}
}

// =====================================================================
// Task 1: Init 深度探针实验
// =====================================================================

// TestInit78_Probe_SqliteMemoryCache ROADMAP L211 研究缺口(Gap G1/DQ3)的可执行探针:
// sqlite + memory cache 下跑完整 Init → 子系统清单盘点 → Close 收尾 → 二次幂等探测。
//
// 通过条件是「给出结论(A/B/C)且不 hang」,不是「Init 必须返回 nil」(D-78-02b)。
func TestInit78_Probe_SqliteMemoryCache(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")

	c := openInit78Core(t, cfg)

	// New 阶段产物断言(:120-151 明确契约)
	assert.NotNil(t, c.CoreInfra, "New 必须装配 CoreInfra")
	assert.NotNil(t, c.CoreServices, "New 必须装配 CoreServices")
	assert.NotNil(t, c.JWTManager, "New 必须能构造 JWTManager")
	assert.NotNil(t, c.PwdManager, "New 必须构造 PasswordManager")
	assert.NotNil(t, c.SM4Cipher, "合法 SM4Key 下 SM4Cipher 非 nil")

	// --- Init(硬超时守卫) ---
	start := time.Now()
	initErr, hung := initCore78Guarded(c)
	initDur := time.Since(start)
	if hung {
		t.Fatalf("PROBE 结论 C(hang): Init 在 %v 内未返回 — SUMMARY 必须按 D-78-03 阶梯处置", init78InitTimeout)
	}
	t.Logf("[PROBE] Init 返回耗时=%v err=%v", initDur, initErr)

	// --- 子系统清单探测(探针主产出,原样抄进 SUMMARY ## Probe Findings)---
	subsystems := []struct {
		name string
		set  bool
	}{
		{"DB(infra)", c.DB != nil},
		{"Cache(infra)", c.Cache != nil},
		{"JWTManager", c.JWTManager != nil},
		{"PwdManager", c.PwdManager != nil},
		{"SM4Cipher", c.SM4Cipher != nil},
		{"reaperCancel(private)", c.reaperCancel != nil},
		{"MetricsCacheService", c.MetricsCacheService != nil},
		{"Scheduler", c.Scheduler != nil},
		{"DeviceExecutor", c.DeviceExecutor != nil},
		{"DeviceDiscoveryService", c.DeviceDiscoveryService != nil},
		{"DeviceInfoCollectionService", c.DeviceInfoCollectionService != nil},
		{"DeviceMonitorService", c.DeviceMonitorService != nil},
		{"PartitionService", c.PartitionService != nil},
		{"CaptchaService", c.CaptchaService != nil},
		{"CaptchaBackgroundService", c.CaptchaBackgroundService != nil},
		{"AuthFactory", c.AuthFactory != nil},
		{"OperLogService", c.OperLogService != nil},
		{"TokenBlacklistService", c.TokenBlacklistService != nil},
		{"DataCacheService", c.DataCacheService != nil},
		{"CacheConfigService", c.CacheConfigService != nil},
		{"CacheManager", c.CacheManager != nil},
		{"RPAScalingService", c.RPAScalingService != nil},
		{"APIEndpointService", c.APIEndpointService != nil},
		{"NoticeHub", c.NoticeHub != nil},
	}
	setCount := 0
	for _, s := range subsystems {
		status := "nil "
		if s.set {
			status = "SET "
			setCount++
		}
		t.Logf("[PROBE] %-26s %s", s.name, status)
	}
	t.Logf("[PROBE] 子系统装配计数: %d/%d", setCount, len(subsystems))
	t.Logf("[PROBE] Cache 动态类型: %T", c.Cache)

	// Cache 动态类型与链路互证:memory 分支应为 *cache.MemoryCache(nil-panic 前 assert 即可)
	if initErr == nil && c.Cache != nil {
		assert.IsType(t, &cache.MemoryCache{}, c.Cache,
			"Cache.Type=memory 应走 initCache else 分支的纯内存缓存")
	}

	// --- 阶段直调核证(若 Init 成功,各私有阶段语义已被顶层串起)---
	if initErr == nil {
		// 挑 3 个关键系统表验证 AutoMigrate 真建表(任取存在的: sys_user/sys_role/sys_menu)
		if c.GetDB() != nil {
			for _, tbl := range []string{"sys_user", "sys_role", "sys_menu"} {
				var cnt int64
				err := c.GetDB().Table(tbl).Count(&cnt).Error
				assert.NoError(t, err, "关键系统表 %s 应存在(AutoMigrate)", tbl)
			}
		}
	} else {
		t.Logf("[PROBE] 结论 B: Init 返回 error=%v — 上表仅为失败点之前可达的阶段", initErr)
	}

	// --- Close(硬超时守卫)---
	closeRes := closeCore78Guarded(c)
	if closeRes.hung {
		t.Fatalf("PROBE 结论 C(hang): Close 在 %v 内未返回 — D-78-03 阶梯触发点,SUMMARY 必录", init78CloseTimeout)
	}
	t.Logf("[PROBE] Close 返回耗时=%v panicked=%v", closeRes.duration, closeRes.panicked)

	// --- 二次 Close 幂等探测(Q-7 已修 MetricsCacheService.Stop;整链幂等未验证)---
	//
	// [QUIRK-78-02-P1] 实测(2026-08-27 探针):二次 Close panic "close of closed channel"
	//   根因 pkg/cache/memory.go:312 MemoryCache.Close() 裸 close(stopChan) 无 sync.Once。
	//   按 D-78-03 (c):不在本 plan 擅改生产(pkg/cache 不在 internal/core 变更面内,
	//   且 Close 的 doc 注释从未声明幂等契约,D-78-10 无据不改);
	//   本文件及后续 Task 一律遵循「首次 Close 即终态」语义,SUMMARY 待裁决给 Phase 79/80。
	close2 := closeCore78Guarded(c)
	t.Logf("[PROBE] 二次 Close: hung=%v panicked=%v panicVal=%v dur=%v (QUIRK-78-02-P1 观察值)",
		close2.hung, close2.panicked, close2.panicVal, close2.duration)
	assert.False(t, close2.hung, "二次 Close 即使 panic 也必须立即返回,不允许 hang")

	// --- 探针结论分流(SUMMARY ## Probe Findings 抄录)---
	switch {
	case initErr != nil:
		t.Logf("[PROBE] ===== 结论 B: Init 返回 error(失败点见上方日志),后续 Task 改分步直调路径 =====")
	case closeRes.hung || closeRes.panicked || close2.hung:
		t.Logf("[PROBE] ===== 结论 C: 收尾阶段 hang,按 D-78-03 阶梯先 (c) 测试侧界定 =====")
	case close2.panicked:
		t.Logf("[PROBE] ===== 结论 A'(A-with-quirk): Init 成功返回 nil,首次 Close 不 hang 不 panic;" +
			"二次 Close panic(QUIRK-78-02-P1)→ 后续 Task 按「首次 Close 即终态」叠加断言 =====")
	default:
		t.Logf("[PROBE] ===== 结论 A: Init 成功返回 nil + Close 收尾不 hang + 二次 Close 幂等 =====")
	}

	// ADSyncScheduler 是 initSchedulerAndTasks 里 StartADSyncScheduler 启动的全局组件,
	// 显式停一次防跨用例残留(R-7;Scheduler.Stop 已在 Close 内)。
	scheduler.StopADSyncScheduler()
}
