package core

// =====================================================================
// Phase 78-02: core.Core Init 全链装配 + Close 收尾测试 (BLOCK-03 收口段)。
//
// 关键纪律 (PLAN 78-02 must_haves / decision_audit):
//   - D-78-02a: sqlite 用 t.TempDir 文件库而非 :memory:(Init 链跨多 GORM 会话,
//     :memory: 多连接各自建库造成表凭空消失的假失败)。
//   - D-78-02b: 探针通过条件是「给出结论且不 hang」而非「Init 必须返回 nil」;
//     禁为过测改生产装配代码。
//   - D-78-02c: startSubprocessReaper 只覆盖当前平台实现,禁 GOOS 条件 t.Skip。
//   - D-78-01: 默认形态走 MemoryCache / NewMultiLevelCacheSimple(无 L2Writer),
//     生产 WithWriter 形态单独用例并显式断言 Close 后 worker 停止(R-7)。
//   - T-78-02-01/02: 所有 Init/Close 调用均包硬超时守卫 channel + t.Cleanup;
//     所有 goroutine(cron/采集/L2Writer/reaper/refreshView)显式 Stop/Close。
//   - T-78-02-03: 仅 sqlite(miniredis 只用于 Cache.Type=redis 分支),禁生产 PG/Redis 地址。
//   - 命名前缀 TestInit78_(D-78-08);本文件零生产 .go 改动。
// =====================================================================

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
)

const (
	// init78GuardTimeout Init 探针硬超时守卫。AutoMigrate ~200 张 model 表在 sqlite
	// 文件库上需数秒,60s 余量足够区分「慢」与「真 hang」(T-78-02-01);
	// 测试二进制层面另有 go test -timeout 300s 兜底。
	init78GuardTimeout = 60 * time.Second

	// close78GuardTimeout Close 硬超时守卫。Core.Close 自带 coreShutdownTimeout=30s
	// 总 deadline,守卫给到 30s 即可判定「Close 自身已无法自恢复」的场景。
	close78GuardTimeout = 30 * time.Second

	// init78SM4Key 合法 base64 的 16 字节测试 key(base64("78Init02TestKey16"))。
	// 刻意避开仓库默认值 dGVzdC1zZWNyZXQxNiEhIQ==(T-78-02-04:不引入任何真实生产 key),
	// 保证 crypto.NewSM4Cipher 能构造成功(NormalNew happy path)。
	init78SM4Key = "SW5pdDc4VGVzdEtleTE2IQ==" // base64("Init78TestKey16!") = 16 字节

	// init78JWTSecret 满足 NewJWTManager 非空 + ≥16 字节校验的测试密钥,
	// 且避开已知弱默认值 xingran-next-secret-key(security/jwt.go F-04)。
	init78JWTSecret = "init78-02-jwt-secret-key-test-only"
)

// newInit78Config 构造一份 Core.Init 可跑的全字段配置(D-78-02b 装配 helper)。
//
//   - Server.Mode="debug":避免 SKIP_AUTOMIGRATE release fatal 守卫误触(core.go CDX-H2);
//   - Server.Port=0:不占真实端口(T-78-02-03);
//   - SkipSetup=false:真跑 InitData / 默认角色菜单(warn-continue 双分支可观察);
//   - Database.Type="sqlite" + t.TempDir 文件库(D-78-02a);
//   - Cache.Type 由调用方指定:"memory" | "redis"(redis 时必须传 miniredis 地址);
//   - Cache.WarmUpEnabled 缺省 false:预热链异步行为由 Task 3 TestInit78_CacheWarmUp 单独打开;
//   - RPA.Enabled 缺省 false:RPA 服务段由 Task 4 TestInit78_ReaperAndRPA 单独打开。
func newInit78Config(t *testing.T, cacheType string, redisAddr string) *config.Config {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Name:      "core-init-78-02-test",
			Host:      "127.0.0.1",
			Port:      0,
			Mode:      "debug",
			SkipSetup: false,
		},
		Database: config.DatabaseConfig{
			Type:         "sqlite",
			Path:         filepath.Join(t.TempDir(), "core_init.db"),
			MaxOpenConns: 20,
			MaxIdleConns: 5,
		},
		Cache: config.CacheConfig{
			Type:        cacheType,
			MaxSize:     1000,
			CleanupTime: 300,
		},
		JWT: config.JWTConfig{
			SecretKey:        init78JWTSecret,
			AccessKeyExpire:  7200,
			RefreshKeyExpire: 604800,
			Issuer:           "xingran-init78-test",
		},
		Security: config.SecurityConfig{
			SM4Key: init78SM4Key,
		},
	}
	if cacheType == "redis" {
		require.NotEmpty(t, redisAddr, "Cache.Type=redis 时必须传 miniredis.RunT(t).Addr()")
		host, portStr, err := net.SplitHostPort(redisAddr)
		require.NoError(t, err)
		port, err := strconv.Atoi(portStr)
		require.NoError(t, err)
		cfg.Cache.Host = host
		cfg.Cache.Port = port
	}
	return cfg
}

// init78CallResult 带 panic 隔离与超时判定的守卫调用结果。
//
//   - completed=false → 被测调用未在守卫内返回(hang 结论由此给出;goroutine 泄漏
//     属预期伴生现象,由 go test -timeout 300s 二进制级兜底回收)。
//   - panicErr 非 nil → 被测调用 panic,已在 goroutine 内 recover 捕获转成 error,
//     防 panic 逃逸击穿整个测试二进制(PLAN Task1 步骤 5「不允许 panic 逃逸」)。
type init78CallResult struct {
	completed bool
	panicErr  error
}

// runInit78Guarded 在带硬超时守卫 + recover 隔离的独立 goroutine 中执行 fn。
func runInit78Guarded(guard time.Duration, what string, fn func()) init78CallResult {
	done := make(chan init78CallResult, 1)
	go func() {
		res := init78CallResult{completed: true}
		defer func() {
			if r := recover(); r != nil {
				res.panicErr = fmt.Errorf("%s PANIC: %v", what, r)
			}
			done <- res
		}()
		fn()
	}()
	select {
	case res := <-done:
		return res
	case <-time.After(guard):
		return init78CallResult{completed: false}
	}
}

// cleanupUploadsDir78 清理 initCaptchaServices 以相对路径创建的存储目录
// ("./uploads/captcha/backgrounds",相对 internal/core 包运行目录)。
// 只做空目录移除(os.Remove 对非空目录报错即静默跳过),绝不误删真实上传数据;
// 避免 git 工作树被测试残留污染(T-78-02-03 同类纪律)。
func cleanupUploadsDir78() {
	for _, dir := range []string{
		filepath.Join("uploads", "captcha", "backgrounds"),
		filepath.Join("uploads", "captcha"),
		"uploads",
	} {
		_ = os.Remove(dir)
	}
}

// TestInit78_Probe_SqliteMemoryCache Task 1 深度探针实验(PLAN Task1 主产出)。
//
// 目标:回答研究缺口 Gap G1/DQ3 —— "core.Core 完整 Init 图能否用 sqlite + MemoryCache
// 拼出、能否在守卫内收尾"。本用例产出三份实证数据(全部抄录 SUMMARY ## Probe Findings):
//  1. Init 终态(A 返回 nil / B 返回 error / C hang)+ 耗时;
//  2. 逐子系统非空清单(Init 各阶段产物字段的可达性证据);
//  3. Close 收尾行为(一次调用耗时 / 二次调用是否幂等 / 是否 panic)。
//
// 通过条件遵循 D-78-02b:「给出结论且不 hang」,因此本用例对 Init 返回 error 只记录不失败;
// 只有 Init 或首次 Close 在硬超时内未返回(结论 C)才判 Fatal。
func TestInit78_Probe_SqliteMemoryCache(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	c, err := New(cfg)
	require.NoError(t, err)
	// New() 的直产字段(core.go:120-151)必须全部就位
	require.NotNil(t, c.CoreInfra, "New 必须装配 CoreInfra(向后兼容 embedding)")
	require.NotNil(t, c.CoreServices, "New 必须装配 CoreServices")
	require.NotNil(t, c.JWTManager)
	require.NotNil(t, c.PwdManager)
	require.NotNil(t, c.SM4Cipher, "合法 SM4Key 下 initSM4Cipher 必须产出 cipher")

	// 兜底收尾:此后任何断言失败 / Init 中途失败都要尝试 Close(LIFO:最后注册先执行)
	closeStillPending := true
	t.Cleanup(func() {
		if !closeStillPending {
			return
		}
		res := runInit78Guarded(close78GuardTimeout, "Close#cleanup", func() { c.Close() })
		if !res.completed {
			t.Logf("PROBE 收尾结论: cleanup Close 在 %v 内未返回(hang)", close78GuardTimeout)
		} else if res.panicErr != nil {
			t.Logf("PROBE 收尾结论: cleanup Close panic — %v", res.panicErr)
		}
	})
	t.Cleanup(cleanupUploadsDir78)

	// ---- ① Init 终态 + 耗时 -------------------------------------------------
	initStart := time.Now()
	var initErr error
	initRes := runInit78Guarded(init78GuardTimeout, "Init", func() { initErr = c.Init() })
	initElapsed := time.Since(initStart)

	switch {
	case !initRes.completed:
		// 结论 C(hang):PLAN Task1 明示 Fatal 并把结论写进日志/SUMMARY
		t.Fatalf("PROBE 结论 C: Init 在 %v 内未返回(疑似 hang)——Gap G1/DQ3 记录为 hang 分支", init78GuardTimeout)
	case initRes.panicErr != nil:
		// 结论 B(panic 被隔离,未逃逸)
		t.Logf("PROBE 结论 B(panic): Init panic — %v (耗时 %v)", initRes.panicErr, initElapsed)
	default:
		t.Logf("PROBE: Init 返回 err=%v (耗时 %v)", initErr, initElapsed)
	}

	// ---- ② 逐子系统非空清单(探针主产出,原样抄录 SUMMARY)------------------
	// 字段名严格取自 core_infra.go / core_services.go;分两档:
	//   alwaysSet:对应阶段无早期 return 旁路,Init 成功时理应必非 nil;
	//   condSet  :条件产物(API 元数据文件缺失 → nil / RPA 未启用 → nil /
	//             NoticeHub 由 router 层注入,Init 链不建)。
	alwaysSet := []string{
		"DB", "Cache", "CacheConfigService", "DataCacheService", "CacheManager",
		"MetricsCacheService", "Scheduler", "DeviceExecutor", "DeviceDiscoveryService",
		"DeviceInfoCollectionService", "DeviceMonitorService", "PartitionService",
		"CaptchaService", "CaptchaBackgroundService", "OperLogService",
		"TokenBlacklistService", "AuthFactory",
	}
	condSet := map[string]bool{
		"APIEndpointService": c.APIEndpointService != nil, // initAPIEndpointService 读 ./configs/api_metadata.yaml
		"RPAScalingService":  c.RPAScalingService != nil,  // RPA.Enabled=false → 不建
		"NoticeHub":          c.NoticeHub != nil,          // router 层注入(internal/api/router.go:91)
	}

	inventory := make(map[string]bool, len(alwaysSet)+len(condSet))
	for _, name := range alwaysSet {
		inventory[name] = probeField78(c, name)
	}
	for name, v := range condSet {
		inventory[name] = v
	}
	names := make([]string, 0, len(inventory))
	for name := range inventory {
		names = append(names, name)
	}
	sort.Strings(names)
	t.Logf("=== PROBE 子系统清单 (Init err=%v) ===", initErr)
	for _, name := range names {
		t.Logf("  %-28s present=%v", name, inventory[name])
	}
	t.Logf("=== Init 耗时=%v ===", initElapsed)

	if initErr == nil {
		missing := make([]string, 0, len(alwaysSet))
		for _, name := range alwaysSet {
			if !inventory[name] {
				missing = append(missing, name)
			}
		}
		assert.Empty(t, missing,
			"Init 返回 nil 时下列无旁路的子系统必须全部就位(conclusion A 完整性),缺失=%v", missing)
	}

	// ---- ③ Close 行为:一次调用 + 二次幂等探测 ------------------------------
	closeStart := time.Now()
	closeRes := runInit78Guarded(close78GuardTimeout, "Close#1", func() { c.Close() })
	closeElapsed := time.Since(closeStart)
	closeStillPending = false

	// 首次 Close 是收尾底线:hang(结论 C)/ panic 都要在此浮出水面
	require.True(t, closeRes.completed,
		"PROBE 结论 C: 首次 Close 在 %v 内未返回(hang)——coreShutdownTimeout=30s 自身未兜住", close78GuardTimeout)
	require.NoError(t, closeRes.panicErr, "单次 Close 不允许 panic(装配完整链上 Close 的每步都有 nil 守卫)")
	t.Logf("PROBE: Close#1 完成 (耗时 %v)", closeElapsed)

	// 幂等探测(PLAN Task1 步骤 5):二次 Close 必须 recover 隔离。
	// 若发现非幂等 → 按 D-78-03(c) 只记 quirk 进 SUMMARY 待裁决,不在本任务改生产。
	r2 := runInit78Guarded(close78GuardTimeout, "Close#2", func() { c.Close() })
	if !r2.completed || r2.panicErr != nil {
		t.Logf("QUIRK-CANDIDATE [D-78-03c]: 二次 Close 非幂等 (completed=%v, panic=%v) — "+
			"SUMMARY 记为待裁决 quirk,本 plan 不改生产 Close;后续 Task4 幂等断言按此降级",
			r2.completed, r2.panicErr)
	} else {
		t.Log("PROBE: Close#2 幂等成立(completed=true, 无 panic)")
	}
}

// probeField78 按 initXxx 阶段产物逐字段判空(反射不可达的私有字段除外)。
// 用具名分支而非 reflect,保证字段改名时编译期立刻暴露。
func probeField78(c *Core, field string) bool {
	switch field {
	case "DB":
		return c.DB != nil
	case "Cache":
		return c.Cache != nil
	case "CacheConfigService":
		return c.CacheConfigService != nil
	case "DataCacheService":
		return c.DataCacheService != nil
	case "CacheManager":
		return c.CacheManager != nil
	case "MetricsCacheService":
		return c.MetricsCacheService != nil
	case "Scheduler":
		return c.Scheduler != nil
	case "DeviceExecutor":
		return c.DeviceExecutor != nil
	case "DeviceDiscoveryService":
		return c.DeviceDiscoveryService != nil
	case "DeviceInfoCollectionService":
		return c.DeviceInfoCollectionService != nil
	case "DeviceMonitorService":
		return c.DeviceMonitorService != nil
	case "PartitionService":
		return c.PartitionService != nil
	case "CaptchaService":
		return c.CaptchaService != nil
	case "CaptchaBackgroundService":
		return c.CaptchaBackgroundService != nil
	case "OperLogService":
		return c.OperLogService != nil
	case "TokenBlacklistService":
		return c.TokenBlacklistService != nil
	case "AuthFactory":
		return c.AuthFactory != nil
	default:
		return false
	}
}
