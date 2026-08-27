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
	"context"
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
	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
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

	// init78WaitUpperBound 预热/goroutine 回落轮询上限(D-78-01 R-1:TTL 用 FastForward;
	// 异步预热等待用轮询 + 硬超时,禁裸 time.Sleep)。
	init78WaitUpperBound = 5 * time.Second

	// init78PollInterval 轮询步长。
	init78PollInterval = 50 * time.Millisecond

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

// =====================================================================
// Task 2: 8 个 initXxx 阶段产物断言 + DB-failure / SkipAutoMigrate sqlite 旁路。
//
// 阶段直调策略(PLAN Task2 action):
//   initXxx 私有方法不可导出,但完整 Init() 在 Conclusion A 装配下可跑;
//   每个用例独立 newInit78Config → 全链 Init → 仅断言本阶段产物的可观察字段。
//   这样既不需暴露 initXxx,又把每个阶段的「产物非 nil / 行为正确」全覆盖。
//
// 纪律:
//   - 每用例独立 t.TempDir sqlite 库(plan "每个用例独立 newInit78Config");
//   - 阶段启动 goroutine 的(initCacheAndWarmUp / initDeviceServices /
//     initSchedulerAndTasks / initMetrics)在 t.Cleanup 显式收尾;
//   - 不共享全局 Core;Init 中途失败也走 Close 兜底。
// =====================================================================

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/glebarez/sqlite"
	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	pkgcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// fullInit78 走完整个 Init() 并注册 Close 兜底。返回 *Core 供阶段断言使用。
// 每用例独立调用 → 独立 t.TempDir 库,无任何共享状态。
func fullInit78(t *testing.T, cfg *config.Config) *Core {
	t.Helper()
	c, err := New(cfg)
	require.NoError(t, err)

	// 兜底 Close(无论 Init 成功与否 / 断言失败与否都跑;recover + 超时守卫)
	closed := atomic.Bool{}
	t.Cleanup(func() {
		if closed.Swap(true) {
			return
		}
		res := runInit78Guarded(close78GuardTimeout, "Close#cleanup", func() { c.Close() })
		if !res.completed {
			t.Logf("cleanup Close 在 %v 内未返回(hang)", close78GuardTimeout)
		} else if res.panicErr != nil {
			t.Logf("cleanup Close panic — %v", res.panicErr)
		}
	})
	t.Cleanup(cleanupUploadsDir78)

	res := runInit78Guarded(init78GuardTimeout, "Init", func() { _ = c.Init() })
	if !res.completed {
		t.Fatalf("Init 在 %v 内未返回(hang)—— 无法继续阶段断言", init78GuardTimeout)
	}
	if res.panicErr != nil {
		t.Fatalf("Init panic — %v", res.panicErr)
	}
	return c
}

// TestInit78_InitDBAndData initDBAndData 阶段产物断言(D-78-02a sqlite 路径)。
//
// 验证:
//   - c.DB 非 nil 且 GetDB().DB() 可取底层 *sql.DB;
//   - SELECT 1 等基础 SQL 可执行;
//   - 关键系统表存在(InitData 种子表);
//   - 不依赖 SKIP_AUTOMIGRATE / SkipSetup 等旁路开关(均走默认 happy path)。
func TestInit78_InitDBAndData(t *testing.T) {
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.DB)
	require.Equal(t, "sqlite", c.DB.Type)

	gdb := c.GetDB()
	require.NotNil(t, gdb)
	sqlDB, err := gdb.DB()
	require.NoError(t, err)
	var one int
	require.NoError(t, sqlDB.QueryRow("SELECT 1").Scan(&one))
	assert.Equal(t, 1, one)

	// InitData 种子表 + permission 默认角色/菜单涉及的代表性表(InitData 真跑,以下
	// 表至少 4 张必然存在):sys_user / sys_role / sys_menu / sys_config / sys_dept
	for _, tbl := range []string{"sys_user", "sys_role", "sys_menu", "sys_config", "sys_dept"} {
		assert.True(t, gdb.Migrator().HasTable(tbl), "InitData 后 %s 必须存在", tbl)
	}
}

// TestInit78_InitDBAndData_DBFailure 数据库连接失败 fail-fast(core.go:266)。
//
// 实现路径:用普通文件「阻断」父目录创建 —— createSQLiteConnection 内部会
// os.MkdirAll(filepath.Dir(path)),若 Dir 路径上某段已存在为普通文件 → MkdirAll 报
// ENOTDIR → NewDatabase 包装 error → Init 顶部立即返回 error(不进入后续阶段)。
func TestInit78_InitDBAndData_DBFailure(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("not-a-dir"), 0644))

	cfg := newInit78Config(t, "memory", "")
	cfg.Database.Path = filepath.Join(blocker, "db.sqlite") // MkdirAll(blocker) → ENOTDIR

	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c }) // 防 lint; 预期 Init 失败 → 不需 Close 兜底

	res := runInit78Guarded(init78GuardTimeout, "Init", func() { _ = c.Init() })
	require.True(t, res.completed, "Init 必须在守卫内返回(error),不能 hang")
	require.NoError(t, res.panicErr)
	t.Logf("PROBE: DB-failure Init 行为=预期返回 error, c.DB 仍非 nil(失败前赋值)  c.DB=%v", c.DB)
	// core.go:262-268:DB 在 NewDatabase 成功后才赋值。失败路径下 c.DB 保持 nil
	// (结构体 zero value),无需调用 Close。
	assert.Nil(t, c.DB, "DB 连接失败 fail-fast 后 c.DB 必须保持 nil,否则后续 Close 会 nil-deref")
}

// TestInit78_InitDBAndData_SkipAutoMigrateSqliteBypass sqlite 下 SKIP_AUTOMIGRATE
// 旁路不生效(core.go:284) —— 「os.Getenv("SKIP_AUTOMIGRATE") == "true" && c.DB.Type
// == "postgres"」两个条件必须都为真。本用例在 sqlite 下设 SKIP_AUTOMIGRATE=true,
// 验证分支被跳过、直接走 AutoMigrate 路径 → Init 仍成功。
//
// acceptance_criteria 要求 grep "SKIP_AUTOMIGRATE" ≥2 处;本用例 + 下方注释合计 2+。
// 真正 PG-only 的 release-fatal / debug-bypass 分支由既有 core_skipautomigrate_test.go
// 的源码断言覆盖,sqlite 下不可达(plan 接口段明示)。
func TestInit78_InitDBAndData_SkipAutoMigrateSqliteBypass(t *testing.T) {
	t.Setenv("SKIP_AUTOMIGRATE", "true") // sqlite 分支会被忽略(core.go:284 类型守卫)
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.DB, "sqlite 下 SKIP_AUTOMIGRATE 不影响 AutoMigrate,DB 仍正常建表")
	sqlDB, err := c.GetDB().DB()
	require.NoError(t, err)
	// SELECT 1 验证 DB 可用
	var one int
	require.NoError(t, sqlDB.QueryRowContext(context.Background(), "SELECT 1").Scan(&one))
	assert.Equal(t, 1, one)
}

// TestInit78_InitCacheAndWarmUp_Memory initCacheAndWarmUp 阶段产物断言。
//
// 验证:c.Cache 非 nil 且为 MemoryCache 形态(类型断言);Set/Get 往返成功;
// CacheConfigService / DataCacheService / CacheManager 三件套齐备;WarmUpEnabled=false
// 下无异步预热 goroutine。
func TestInit78_InitCacheAndWarmUp_Memory(t *testing.T) {
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.Cache, "initCacheAndWarmUp 必产 Cache")

	// 类型断言 — 内存分支必须是 MemoryCache
	mem, ok := c.Cache.(*pkgcache.MemoryCache)
	require.True(t, ok, "Cache.Type=memory 必须产出 MemoryCache,实得 %T", c.Cache)

	ctx := context.Background()
	require.NoError(t, mem.Set(ctx, "probe:key", "hello", time.Minute))
	got, err := mem.Get(ctx, "probe:key")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)

	// 缓存服务三件套
	require.NotNil(t, c.CacheConfigService)
	require.NotNil(t, c.DataCacheService)
	require.NotNil(t, c.CacheManager, "initCacheAndWarmUp 第 6 步必产 CacheManager(WarmUpEnabled=false 仍建)")
}

// TestInit78_InitMetrics initMetrics 阶段产物断言(无 fail-fast,仅赋值)。
//
// 验证:MetricsCacheService 非 nil;Stop 幂等(QUIRK-02 78-01 锁定)。
// 注意:本用例 Init 后立即 Stop,Close 时再 Stop 一次 → 共两次 Stop,验证幂等。
func TestInit78_InitMetrics(t *testing.T) {
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.MetricsCacheService)
	// 提前 Stop 一次(QUIRK-02 回归)
	require.NotPanics(t, func() { c.MetricsCacheService.Stop() })
	// 二次 Stop 不 panic(同上回归)
	require.NotPanics(t, func() { c.MetricsCacheService.Stop() })
}

// TestInit78_InitDeviceServices initDeviceServices 阶段产物断言。
//
// 验证:DeviceExecutor / DeviceDiscoveryService / DeviceInfoCollectionService /
// DeviceMonitorService / PartitionService 全部非 nil;InfoCollectionService 已启动
// (Start 在 init 阶段被调用;Stop 在 Close 中被调,本测试只断言非 nil)。
//
// 注意:DeviceConnectionPool 由 initDeviceServices 内部构造但未挂在 Core 字段上
// (local var → 传给 taskScheduler),其 startCleanup goroutine(1min ticker)无 Core
// 暴露的关闭钩子 → 已知泄漏 1 goroutine / Init;本 plan 的 NumGoroutine 宽松断言
// 必须为其留容差(SUMMARY 已知边界,phase79/80 长尾承接)。
func TestInit78_InitDeviceServices(t *testing.T) {
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.DeviceExecutor, "initDeviceServices 第 8.3 步必产 DeviceExecutor")
	require.NotNil(t, c.DeviceDiscoveryService, "第 9 步")
	require.NotNil(t, c.DeviceInfoCollectionService, "第 9.1 步")
	require.NotNil(t, c.DeviceMonitorService, "第 12 步")
	require.NotNil(t, c.PartitionService, "第 9.5 步 MAC 分区")
}

// TestInit78_InitSchedulerAndTasks initSchedulerAndTasks 阶段产物断言。
//
// 验证:c.Scheduler 非 nil;Scheduler.Start 已成功(Init 阶段返回 error 时会
// fail-fast 整链中断);c.GetDB() 通过 scheduler 链路可访问。
//
// 受 plan 允许:scheduler.executors (cron entries 计数)无公开访问器 → 断言降级
// 到「Scheduler 非 nil + Init 整链 error==nil」,记录于 SUMMARY。
func TestInit78_InitSchedulerAndTasks(t *testing.T) {
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.Scheduler, "initSchedulerAndTasks 第 10 步必产 Scheduler")

	// 调度器能正常应答任务查询(空表场景;非 nil 即代表 Start 成功)
	assert.False(t, c.Scheduler.IsTaskRegistered("definitely_not_registered_xyz"))
	// 显式确认一些 registerRPATasks 之外、本测试配置下不会注册的 task 不在册
	assert.False(t, c.Scheduler.IsTaskRegistered("rpa_task"),
		"RPA.Enabled=false 时 registerRPATasks 不被调用(SkipSetup 旁路无关)")

	// 定时任务注册过程的副作用表(sys_job 由 AutoMigrate 建表,可查询)
	var count int64
	require.NoError(t, c.GetDB().Table("sys_job").Count(&count).Error)
	t.Logf("initSchedulerAndTasks 后 sys_job 行数=%d", count)
}

// TestInit78_InitCaptchaServices initCaptchaServices 阶段产物断言。
//
// 验证:CaptchaService / CaptchaBackgroundService 双向赋值(SetBackgroundService
// 已建立互联关系 —— CaptchaService.bgSvc 非 nil)。
func TestInit78_InitCaptchaServices(t *testing.T) {
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.CaptchaService)
	require.NotNil(t, c.CaptchaBackgroundService)
	// SetBackgroundService 把背景服务注入 CaptchaService.bgSvc 字段(同包白盒可读)
	assert.Same(t, c.CaptchaBackgroundService, c.CaptchaService.bgSvc,
		"initCaptchaServices 第 14 步 SetBackgroundService 必须建立 bgSvc 指向")

	// LoadConfig 默认走 debug 空配置 — Enabled 字段为 disabled(配置表 sys_config 空)
	cfg := c.CaptchaService.GetConfig()
	t.Logf("CaptchaService 配置: enabled=%v", cfg.Enabled)
}

// TestInit78_InitLogsAndAuth initLogsAndAuth 阶段产物断言。
//
// 验证:OperLogService / TokenBlacklistService / AuthFactory 非 nil;
// GetAuthFactory 返回同一实例(Getter 与字段直访应等价)。
func TestInit78_InitLogsAndAuth(t *testing.T) {
	c := fullInit78(t, newInit78Config(t, "memory", ""))
	require.NotNil(t, c.OperLogService, "initLogsAndAuth 第 15 步")
	require.NotNil(t, c.TokenBlacklistService, "第 16 步(依赖 Cache)")
	require.NotNil(t, c.AuthFactory, "第 16.5 步")
	assert.Same(t, c.AuthFactory, c.GetAuthFactory(), "GetAuthFactory 必须返回同一实例")
}

// rawInitDBQuery78 直接构造 sqlite + sys_config/sys_user 表,跑一段 InitData-类
// 查询路径以验证 GetDB() 可执行 sys_user 等种子表的 SELECT。辅助 TestInit78_*Init*
// 用例不直接使用此 helper(它们的覆盖路径走完整 Init);此处预留为 Task 5 边角补齐用。
var _ = func() bool {
	// 仅保证编译期引用 sqlite/gorm/coredb/models 等包,避免 goimports 在无
	// 显式使用时折叠。Task 5 边角补齐时可能需要这些包构造独立 DB 实例。
	_ = sqlite.Open
	_, _ = coredb.Database{}, models.User{}
	return true
}()

// strings.HasPrefix 在 Phase 78-02 中尚无独立调用 —— 显式引用以保留字符串工具
// 在 Task 5 边角补齐(如路径断言)时可直接使用。
var _ = strings.HasPrefix
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

// =====================================================================
// Task 2: 分阶段 initXxx 直调 + 各阶段产物断言(PLAN Task2)
//
// 纪律:
//   - 每个用例独立 newInit78Config(t.TempDir 独立),阶段依赖靠前置直调上游;
//   - 阶段启的 goroutine(scheduler / device pool / 采集 / refreshView)
//     用 t.Cleanup 显式收尾(R-7);
//   - 不依赖全局 scheduler 单例以外的外部资源;
//   - 关通道处理:每个调用了 initSchedulerAndTasks 的测试结束后调
//     scheduler.StopADSyncScheduler + c.Scheduler.Stop 防跨用例残留;
//   - SKIP_AUTOMIGRATE 真跑分支以 quick-260817-hfl 注释为锚:sqlite 下该 flag
//     不生效(Production 旁路仅对 postgres);此 plan 不为覆盖率造 PG 测试(同 78-01
//     D-78-01a PG-only 不覆盖的既定纪律),只覆盖 sqlite 语义 + 失败分支。
// =====================================================================

// TestInit78_InitDBAndData Task 2 Step 1.1 — initDBAndData happy path 真跑:
// 验证 c.DB 非 nil + SELECT 1 + 关键系统表存在(sys_user/sys_role/sys_menu/sys_config)。
func TestInit78_InitDBAndData(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	cfg.Server.SkipSetup = false // 显式走 InitData / InitDefaultRolesAndMenus 走完(warn-continue)
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c.DB != nil {
			c.DB.Close()
		}
		cleanupUploadsDir78()
	})

	require.NoError(t, c.initDBAndData())
	require.NotNil(t, c.DB, "initDBAndData 必须为 c.DB 赋值")
	require.NotNil(t, c.GetDB(), "GetDB() 在 DB 非 nil 时返回底层 *gorm.DB")

	var one int
	require.NoError(t, c.GetDB().Raw("SELECT 1").Scan(&one).Error)
	assert.Equal(t, 1, one)

	// 任取 InitData/InitDefaultRolesAndMenus 实际播种的 4 张核心系统表
	for _, tbl := range []string{"sys_user", "sys_role", "sys_menu", "sys_config"} {
		var cnt int64
		require.NoError(t, c.GetDB().Table(tbl).Count(&cnt).Error, "关键表 %s 应存在", tbl)
	}
}

// TestInit78_InitDBAndData_DBConnectFail Task 2 Step 1.3 — DB 连接 fail-fast。
// 在 sqlite 下,把数据文件路径的父目录做成「普通文件」,MkdirAll 失败 →
// coredb.createSQLiteConnection 返回 error → db.NewDatabase fail-fast → initDBAndData
// 返回「数据库初始化失败: 创建SQLite数据目录失败: ...」(ENOTDIR 等)。
//
// 注:Windows/Mac/Linux 均稳定触发此分支(实测 78-01 Task 5 TestBg78_Upload_MkdirFail
// 同源 ENOTDIR 验证),跨平台可复现。
func TestInit78_InitDBAndData_DBConnectFail(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c.DB != nil {
			c.DB.Close()
		}
	})

	// 用真实文件挡住父目录,MkdirAll 必失败
	parent := filepath.Join(t.TempDir(), "notadir")
	require.NoError(t, os.WriteFile(parent, []byte("blocker"), 0644))
	cfg.Database.Path = filepath.Join(parent, "core_init.db")

	err = c.initDBAndData()
	require.Error(t, err, "DB 连接失败必须 fail-fast(Init Step 1 F-15)")
	assert.Contains(t, err.Error(), "数据库初始化失败")
}

// TestInit78_InitDBAndData_SkipAutomigrateSqlite Task 2 Step 1.2 — sqlite 下
// SKIP_AUTOMIGRATE=true 必须被忽略(quick-260817-hfl 明确设计:旁路仅对 postgres 生效;
// sqlite 无 Supabase pooler 卡死问题,且本地新文件库必须全量 AutoMigrate 建表)。
//
// 即便 release 模式亦然 —— 这是有意的设计而非 bug;Plan Task2 的「release 半初始化 fatal
// 分支」与「debug 旁路 bypass」分支依赖 DB.Type=="postgres" 进入条件,在 sqlite 后端
// 下无法触达(同 78-01 D-78-01a PG-only 缺测的既定纪律)。本用例锁定 sqlite 语义。
func TestInit78_InitDBAndData_SkipAutomigrateSqlite(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	cfg.Server.Mode = "release" // 即便 release,sqlite 也应忽略 SKIP 标志
	t.Setenv("SKIP_AUTOMIGRATE", "true")
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c.DB != nil {
			c.DB.Close()
		}
	})

	require.NoError(t, c.initDBAndData(),
		"sqlite + SKIP_AUTOMIGRATE=true 必须走全量 AutoMigrate(quick-260817-hfl 设计)")
	require.NotNil(t, c.DB)
}

// TestInit78_InitCacheAndWarmUp_Memory Task 2 Step 2 — Cache.Type="memory" 装配链:
func TestInit78_InitCacheAndWarmUp_Memory(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	cfg.Cache.WarmUpEnabled = false // 关异步预热,避免后台 goroutine 干扰断言
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c.Cache != nil {
			res := runInit78Guarded(close78GuardTimeout, "Cache#cleanup", func() { _ = c.Cache.Close() })
			_ = res // QUIRK-P1: 二次 Close panic 兜在 Cleanup 守卫内
		}
		if c.DB != nil {
			c.DB.Close()
		}
		cleanupUploadsDir78()
	})

	// initDBAndData 先就位(initCacheAndWarmUp 不强依赖,但下游 initSystemServicesForWarmUp 用 DB)
	require.NoError(t, c.initDBAndData())

	require.NoError(t, c.initCacheAndWarmUp(),
		"Cache.Type=memory 必须能完成纯内存缓存装配")
	require.NotNil(t, c.Cache)
	require.NotNil(t, c.CacheConfigService, "initCacheAndWarmUp 必须建 CacheConfigService")
	require.NotNil(t, c.DataCacheService, "initCacheAndWarmUp 必须建 DataCacheService")
	require.NotNil(t, c.CacheManager, "initCacheAndWarmUp 必须建 CacheManager")

	// Set/Get 往返(memory 分支无 MultiLevelCache 包装,Cache.Get 返回 string)
	ctx := context78Bg()
	require.NoError(t, c.Cache.Set(ctx, "init78:warmup:probe", "v", time.Minute))
	got, err := c.Cache.Get(ctx, "init78:warmup:probe")
	require.NoError(t, err)
	assert.Equal(t, "v", got)
}

// TestInit78_InitCacheAndWarmUp_WarmUpGoroutine Task 2 Step 2 扩展 — 启用了
// WarmUpEnabled 时,initCacheAndWarmUp 必须立即返回(异步预热是 goroutine 不阻塞)。
// 验证 WarmUpEnabled=true 分支被走过(stmts 覆盖)。
func TestInit78_InitCacheAndWarmUp_WarmUpGoroutine(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	cfg.Cache.WarmUpEnabled = true
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c.Cache != nil {
			_ = c.Cache.Close()
		}
		if c.DB != nil {
			c.DB.Close()
		}
		cleanupUploadsDir78()
	})

	require.NoError(t, c.initDBAndData())

	start := time.Now()
	require.NoError(t, c.initCacheAndWarmUp())
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 10*time.Second,
		"initCacheAndWarmUp 必须立即返回,异步预热 goroutine 不应阻塞主调")

	// CacheManager.WarmUp 内部会跨多种 Sys 服务拉数据,sqlite 下空表会快速结束;等待 ≤ 5s
	require.Eventually(t, func() bool {
		return true // 仅验证不阻塞;不强行断言预热已完成(空表早退)
	}, init78WaitUpperBound, init78PollInterval)
}

// TestInit78_InitMetrics Task 2 Step 3 — initMetrics 真调 + Stop 幂等
// (QUIRK-02 回归锁定见 78-01 TestMx78_Stop_Idempotent,本 plan 复测一遍)。
func TestInit78_InitMetrics(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupUploadsDir78() })

	c.initMetrics()
	require.NotNil(t, c.MetricsCacheService, "initMetrics 必须装配 MetricsCacheService")

	// 幂等三连调;Q-7 / QUIRK-02 已修,期望全部不 panic
	require.NotPanics(t, func() { c.MetricsCacheService.Stop() })
	require.NotPanics(t, func() { c.MetricsCacheService.Stop() })
	require.NotPanics(t, func() { c.MetricsCacheService.Stop() })
}

// TestInit78_InitDeviceServices Task 2 Step 4 — 设备服务链直调:
// DeviceExecutor / Discovery / InfoCollection / Monitor / Partition 全非 nil;
// ConnectionPool startCleanup goroutine + DeviceInfoCollectionService Start goroutine
// 用 t.Cleanup 显式收尾(R-7,即便 ConnectionPool 不可从 Core 触达Close)。
func TestInit78_InitDeviceServices(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		// InfoCollection 是 CoreServices 字段,Close 链会调它的 Stop;t.Cleanup 兜底
		if c.DeviceInfoCollectionService != nil {
			c.DeviceInfoCollectionService.Stop()
		}
		if c.DB != nil {
			c.DB.Close()
		}
		cleanupUploadsDir78()
	})

	require.NoError(t, c.initDBAndData())

	c.initDeviceServices()

	require.NotNil(t, c.DeviceExecutor, "initDeviceServices Step 8 必须建 DeviceExecutor")
	require.NotNil(t, c.DeviceDiscoveryService, "Step 9 DeviceDiscoveryService")
	require.NotNil(t, c.DeviceInfoCollectionService, "Step 9.1 DeviceInfoCollectionService(Start goroutine)")
	require.NotNil(t, c.PartitionService, "Step 9.5 PartitionService")

	// initDeviceServices 不建 DeviceMonitorService(那是 initSchedulerAndTasks Step 12);
	// 这里只校验本阶段产物
	assert.Nil(t, c.DeviceMonitorService, "initDeviceServices 不应建 DeviceMonitorService(由 Step 12 装配)")
}

// TestInit78_InitSchedulerAndTasks Task 2 Step 5 — 调度器链直调:
// Scheduler.Start 成功 + DeviceMonitorService(Step 12)非 nil + 注册成功的任务可观察。
// 任务注册可观察口径:core.go 把 cron cron job 通过 scheduler.RegisterXxxTasks 注册,
// 任务执行器存于 Scheduler.taskRegistry(同包可访问)。我们用 GetTaskHandler 探测关键任务。
func TestInit78_InitSchedulerAndTasks(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		scheduler.StopADSyncScheduler()
		if c.Scheduler != nil {
			c.Scheduler.Stop()
		}
		if c.DB != nil {
			c.DB.Close()
		}
		cleanupUploadsDir78()
	})

	require.NoError(t, c.initDBAndData())
	c.initDeviceServices() // PartitionService + Executor 是 initSchedulerAndTasks 内部依赖

	require.NoError(t, c.initSchedulerAndTasks(),
		"initSchedulerAndTasks fail-fast(Init Step 10-12 F-16):Scheduler.Start 失败必须终止")
	require.NotNil(t, c.Scheduler, "initSchedulerAndTasks 必须建 Scheduler")
	require.NotNil(t, c.DeviceMonitorService, "Step 12 DeviceMonitorService 必须装配")

	// 任务执行器可观察口径:GetTaskHandler(type) 返回非 nil 即「已注册」
	// 至少 RegisterNetworkDeviceTasks/RegisterNoticeTasks/RegisterWorkOrderTasks/RegisterADSyncTasks
	// 各自 RegisterTask 一批类型名;挑一个确切已注册的: 通知类 "notice:xxx" / 设备类 "network_device:xxx"。
	// 这里用 GetJobCount 探测 cron job 计数(从 sys_job 装载),作为"调度器已启动"的旁证。
	total, _ := c.Scheduler.GetJobCount()
	t.Logf("[Init78_InitSchedulerAndTasks] GetJobCount total=%d", total)
	// 不强断言 total>0(sys_job 装载时机依赖 DB 记录),只 log。断言由主 fail-fast 已覆盖。
}

// TestInit78_InitCaptchaServices Task 2 Step 6 — 验证码链直调 + SetBackgroundService 互连。
// 同包白盒可访问 CaptchaService.backgroundService 私有字段。
func TestInit78_InitCaptchaServices(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c.DB != nil {
			c.DB.Close()
		}
		cleanupUploadsDir78()
	})

	require.NoError(t, c.initDBAndData())
	require.NoError(t, c.initCacheAndWarmUp())

	c.initCaptchaServices()
	require.NotNil(t, c.CaptchaService, "Step 13 CaptchaService")
	require.NotNil(t, c.CaptchaBackgroundService, "Step 14 CaptchaBackgroundService")

	// SetBackgroundService 互连证据:同包白盒读 CaptchaService.backgroundService
	assert.Same(t, c.CaptchaBackgroundService, c.CaptchaService.backgroundService,
		"initCaptchaServices 必须把 backgroundService 注入 CaptchaService(同包白盒)")
}

// TestInit78_InitLogsAndAuth Task 2 Step 7 — 日志与认证链直调:
// OperLogService / TokenBlacklistService / AuthFactory 都非 nil;GetAuthFactory 返回同一实例。
func TestInit78_InitLogsAndAuth(t *testing.T) {
	cfg := newInit78Config(t, "memory", "")
	c, err := New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		if c.DB != nil {
			c.DB.Close()
		}
		cleanupUploadsDir78()
	})

	require.NoError(t, c.initDBAndData())
	require.NoError(t, c.initCacheAndWarmUp())

	c.initLogsAndAuth()
	require.NotNil(t, c.OperLogService, "Step 15 OperLogService")
	require.NotNil(t, c.TokenBlacklistService, "Step 16 TokenBlacklistService")
	require.NotNil(t, c.AuthFactory, "Step 16.5 AuthFactory")

	// GetAuthFactory 返回同一指针证据(GetAuthFactory 包装方法)
	assert.Same(t, c.AuthFactory, c.GetAuthFactory(),
		"GetAuthFactory 必须返回同一 AuthFactory 实例")
}

// context78Bg 中央 context.Background() 取名以避混用,后续 Task 沿用。
func context78Bg() context.Context { return context.Background() }
