---
plan: 78-01
phase: 78-block-bp-unlock-by-foundation
executed: 2026-08-27
commits:
  - f67acc9 (test: task 1 GenerateCaptcha 真实链路双分支 — MemoryCache 降级 + miniredis 原子)
  - bc64607 (test: task 2 slider 全分支 + VerifySliderCaptcha 剩余 + abs 负数)
  - 07b3ae1 (test: task 3 captcha_background Upload 全分支 + GetRandomEnabled t.TempDir 隔离)
  - dec9cc4 (test: task 4 preGenerateForConfig + PreGeneratePool + GetFromCachePool 缓存池链)
  - 3d16eb5 (test: task 5 metrics_cache 边缘分支 + 三文件 per-file checkpoint 收口)
---

# 78-01 Summary — core captcha 三件套真实链路直测(QUIRK-01 解锁)

## 交付

3 个测试文件(1254 行,零生产 .go 改动),全部与源码同包共置(D-78-08):

- `internal/core/captcha_78_01_test.go`(591 行):**15 个 TestCap78_**。装配 helper
  `newCap78Mem` / `newCap78Redis`(sqlite glebarez 文件库 + sys_config + sys_captcha_background
  × MemoryCache / miniredis.RunT+RedisCache,t.Cleanup 全量 Close)。GenerateCaptcha 双限流分支
  首次独立直测 —— **since QUIRK-01 the IncrementBy 真实链路 is reachable**:MemoryCache 不实现
  RateLimitCache → :265 Increment+Expire 降级分支(TTL>0 可观察证据);miniredis+RedisCache →
  :257 IncrementWithExpire 原子分支(mr.TTL 断言 Lua EXPIRE 生效,R-1/R-2 纪律遵守)。
  另覆盖限流超限(双装配表驱动)/ fail-close(disabled miniredis) / disabled / getIPRateLimit 三档;
  slider 分支矩阵(auto/unknown/custom-nil-service/cache-pool-hit/db-hit-load-ok/load-file-fail/
  mixed 宽松口径)+ VerifySliderCaptcha 全剩余分支 + abs 负数 + 容差边界。
- `internal/core/captcha_background_78_01_test.go`(528 行):**16 个 TestBg78_**。
  helper `newBg78` 强制 `svc.config.StoragePath = t.TempDir()`(T-78-01-01 守护,仓库 uploads/
  零污染,已验证);真 PNG 全部 stdlib image/png 现场造。Upload 六分支含两处 os.Remove 清理断言
  (:97 尺寸失败 / :124 DB 失败)+ MkdirAll 失败;GetRandomEnabled 精确匹配/DROP-TABLE 后缓存命中/
  空结果;preGenerateForConfig 主力缺口(空早退/成功落池/BadFile 静默容错);
  PreGeneratePool 4×3 双循环;GetFromCachePool 三态(空/counter0/命中自减归零删key/坏JSON);
  IncrementUseCount(+1×2 + last_used_at + ghost-id 不报错)。
- `internal/core/metrics_cache_78_01_test.go`(135 行):**4 个 TestMx78_**。
  NewMetricsCacheService Cache nil/非nil 双形态、GetCurrentMetrics M-1 区间断言
  ([0,100] + burn 双基线防零增量)、GetServerInfo 必备键存在、Stop 幂等三连调
  (QUIRK-02 回归锁定,78-02 Core.Close 前置依赖)。

## Coverage checkpoint(D-78 达标口径落 SUMMARY)

per-file weighted(内部/core 包,cov profile 按 numStmt 加权):

| File | 基线(plan 2026-08-27 逐函数) | 实测 | 目标 | 结果 |
|---|---|---|---|---|
| captcha.go | GenerateCaptcha 56.5 / generateSliderWithBackground 35.4 / abs 66.7 … | **88.8%**(249 stmts 中 221 covered) | ≥85% | ✅ |
| captcha_background.go | Upload 57.9 / preGenerateForConfig 14.3 / GetRandomEnabled 76.5 … | **89.2%**(130 中 116) | ≥75% | ✅ |
| metrics_cache.go | NewMetricsCacheService 80 / GetCurrentMetrics 80 | **92.9%**(14 中 13) | 100% | ⚠ 见 Known gaps |
| 非 core.go 子集合计 | — | **89.1%**(393 中 350) | ≥65% | ✅ |
| internal/core 包总 | 43.7% | **54.2%**(+10.5pp) | ≥70%(78-02 收口) | ➡ wave 2 |

逐函数亮点:GenerateCaptcha 56.5→87.0;generateSliderWithBackground 35.4→91.7;
abs 66.7→100;VerifySliderCaptcha 70.3→81.1;Upload 57.9→94.7;
preGenerateForConfig 14.3→82.9;NewMetricsCacheService 80→100。

## Quirks 处置

无新 quirk(生产 .go 改动 0,D-78-10 escape hatch 未触发)。源码行为与据源一致:
- preGenerateForConfig 缓存键 `captcha:cache:pool:<shape>:<diff>:<slot>` 与 counter 自增语义与实现吻合(:295/:303);
- GetFromCachePool counter 归零走 Delete(:342)、异步补充 goroutine 带 30s timeout ctx(:346-350),测试按现行为锚定。

## Deviations / 决策裁决记录

1. **[D-78-01a] PG-only 分支不覆盖**:`GetRandomEnabled` 的 `allowed_shapes @>` /
   `jsonb_array_length` 分支(captcha_background.go:166-179)sqlite 下 Type=="sqlite" 直接绕过,
   接受不覆盖,勿为覆盖率改 SQL。
2. **[D-78-01b] mixed 模式宽松断言**:20 次调用「全部成功 + 至少一种形态命中」,
   不做 custom/auto 固定次数强断言(N=20 实跑 custom≈auto 各半,防 flake)。
3. **M-1 FailClose 主路径成立**:miniredis v2.38.0 的 `mr.Close()` 幂等(RunT cleanup 二次
   Close 返回 error 不 panic),未触发 fallback(`miniredis.Run()` + 手动 defer),
   与 78-VERIFICATION MED-M-1 的预警路径无关,主路径直接通过。
4. **TestBg78_Upload_MkdirFail Windows 可复现**:MkdirAll 对"路径是普通文件"在 Windows 返回
   ENOTDIR,该分支可稳定驱动;仍加 `testing.Short()` 守卫(D-78-10 平台敏感预防性 fallback)。
5. **[环境] go test -race 本地不可执行**:Windows 本机 cgo 工具链故障(cgo.exe exit status 2),
   改动前既有测试(TestCoreSplit_BackwardCompat 等)同样构建失败 —— 与本 plan 无关的预存环境限制。
   race 纪律由 t.Cleanup 全量防护(miniredis RunT 自动 cleanup / RedisCache+MemoryCache 显式
   Close Close())+ ci.yml Linux race job 兜底验证。

## Known gaps(待裁决/后续收口)

- `metrics_cache.go GetCurrentMetrics` error 分支(:36-38):需 manager.getRealtimeMetrics
  真实采样失败(CPU/Mem/disk 任一 gopsutil 错误),本地无法确定性触发;按 D-78-10 无据不改 +
  断言现 happy path。差 1 stmt(92.9%),目标口径为「全 100%」——若 verifier 判硬缺口,
  建议后续经 MetricsCacheManager 注入 seam 或 platform-specific stub 收口(需单独立项)。
- `LoadConfig` 80.4%、`VerifyCaptcha` 96.4% 未到 100:非本 plan 主目标(74-08 已有主体覆盖),
  缺口在 `Sscanf err` 同时发生等复合分支,投入产出比低,BLOCK-03 由 78-02 统一兜底。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestCap78_ ≥14 / TestBg78_ ≥14 / TestMx78_ ≥4 | ✅ 15 / 16 / 4(合计 35) |
| 含字面注释 "since QUIRK-01 the IncrementBy 真实链路 is reachable" | ✅(Task 1 注释块 ×2) |
| miniredis.RunT / mr.FastForward|R-1 TTL 禁 time.Sleep | ✅(time.Sleep 计数 0;TTL 用 mr.TTL 断言) |
| backgroundId 存在(cache-pool-hit/db-hit)与不存在(auto/unknown/no-svc/load-fail)双向用例 | ✅ |
| captcha.go ≥85% / captcha_background.go ≥75% / metrics_cache 100% | ✅ 88.8 / 89.2 / ⚠92.9(error 分支 1 stmt,Known gaps) |
| 非 core.go 子集 ≥65%,数字落 SUMMARY | ✅ 89.1% |
| `go build ./...` == 0;`go test ./...` == 0 | ✅ |
| 生产 .go 改动 = 0 | ✅(git diff --stat 仅 *_test.go 新增) |
| `git status --porcelain uploads/` 无残留 | ✅(uploads/captcha/backgrounds 空) |

## 手注(给 78-02)

三装配 helper 就位可直接复用:`newCap78Mem` / `newCap78Redis`(同包 sqlite+MemoryCache /
miniredis+RedisCache,自动 t.Cleanup)、`sysCaptchaBackgroundDDL`、`makePNGBytes` /
`makeBG78PNG`(stdlib 真 PNG 工厂)。Init 链(wave 2)记得 MultiLevelCache 一律走
`NewMultiLevelCacheSimple`(R-7 L2Writer 由你 plan 承担)+ `t.Cleanup(c.Cache.Close())`,
且 QUIRK-02 幂等 Stop 已在本 plan 回归锁定(TestMx78_Stop_Idempotent)。

## Self-Check: PASSED

- 文件存在:captcha_78_01_test.go / captcha_background_78_01_test.go /
  metrics_cache_78_01_test.go / 78-01-SUMMARY.md — 全 FOUND。
- 提交存在:f67acc9 / bc64607 / 07b3ae1 / dec9cc4 / 3d16eb5 — 全 FOUND(git log --all)。
- `go build ./...` exit 0;`go test ./...` exit 0。
