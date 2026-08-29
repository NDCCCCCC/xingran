---
phase: 76-test-doubles
plan: "01"
subsystem: testing
tags: [miniredis, httpmock, test-doubles, redis, go-mod, coverage-infra]

# Dependency graph
requires: []
provides:
  - miniredis/v2 v2.38.0 + httpmock v1.4.2 pinned test-only 依赖（go.mod 落地）
  - newMiniredisCache(t) 真链路冒烟 helper 模式（RunT→NewRedisCache，R-3 握手哨兵）
  - httpmock const-URL 拦截模式（Activate(t) + RegisterResponder，tidy 保活锚点）
  - 三坑防护具名用例（TestRedisTTLFastForward / TestRedisGetStatsDegraded）
affects: [76-02, 76-03, 76-04, 76-05, phase-78-core-init-chain, phase-80-pkg-cache-coverage]

# Tech tracking
tech-stack:
  added: ["github.com/alicebob/miniredis/v2 v2.38.0 (test-only)", "github.com/jarcoal/httpmock v1.4.2 (test-only)", "github.com/yuin/gopher-lua v1.1.1 (indirect)"]
  patterns:
    - "newMiniredisCache: miniredis.RunT + net.SplitHostPort + NewRedisCache 真链路（构造器 PING 即 SETINFO 握手实证）"
    - "TTL 推进一律 mr.FastForward，禁真实睡眠"
    - "GetStats 在 miniredis 下断言降级面（缺席字段 + 空串）而非具体值"
    - "httpmock: Activate(t) 自动清理 + RegisterResponder 不带 query 匹配任意 query；const-URL 无注入点场景专用"
    - "同包测试污染共享全局（BaiduAPIRateLimiter）必须换私有令牌桶"

key-files:
  created:
    - pkg/cache/redis_miniredis_76_01_test.go
    - internal/services/operations/geocoding_httpmock_76_01_test.go
  modified:
    - go.mod
    - go.sum
    - pkg/cache/cache_74_08_test.go

key-decisions:
  - "go.mod diff 口径（verifier 按此验收，RESEARCH Open Question 1 裁决）：direct +2（miniredis/httpmock，均带 // test-only (v1.27 D-02)）；indirect +1（gopher-lua v1.1.1）；go-testdeep 未进 go.mod（Go 1.24 module graph pruning 剔除依赖的 test-only 依赖，仅 go.sum 落哈希）；既有生产 require 零新增/删除/版本变动"
  - "R-2 断言按 pinned v2.38.0 源码实况改写：INFO server/memory/keyspace 返回错误而非内容 → GetStats 降级为 key 存在但空串"
  - "tidy 揭示的 2 处预存分类陈旧（glebarez/go-sqlite 升 direct、x/net 降 indirect）保留规范化输出，版本零变更"

patterns-established:
  - "Pattern: miniredis 冒烟经 NewRedisCache 构造器真链路，helper 即 R-3 回归哨兵"
  - "Pattern: httpmock 仅用于 const-URL 无注入点；可注入场景一律 httptest/假 RoundTripper"

requirements-completed: [INFRA-01]

# Metrics
duration: 38min
completed: 2026-08-23
---

# Phase 76 Plan 01: miniredis + httpmock test-only 依赖落地 Summary

**miniredis/v2 v2.38.0 与 httpmock v1.4.2 以 pinned test-only 依赖进入 go.mod（direct +2 带 `// test-only (v1.27 D-02)` 注释），并以 pkg/cache 全命令面真链路冒烟（三坑防护具名用例）与 geocoding const-URL httpmock PoC 完成真实使用锚定（tidy 保活）**

## go.mod diff 形态预写（verifier 验收口径）

| 变更类型 | 内容 | 说明 |
|----------|------|------|
| direct +2 | `github.com/alicebob/miniredis/v2 v2.38.0 // test-only (v1.27 D-02)`、`github.com/jarcoal/httpmock v1.4.2 // test-only (v1.27 D-02)` | v1.27 D-02 解禁决策落地 |
| indirect +1 | `github.com/yuin/gopher-lua v1.1.1 // indirect` | miniredis EVAL 支撑 |
| （无）go-testdeep | 未出现在 go.mod | Go 1.24 module graph pruning：httpmock 的 test-only 依赖不进主模块 go.mod，仅 go.sum 落 2 行哈希 |
| 分类修正 ×2 | `github.com/glebarez/go-sqlite v1.21.2` indirect→direct；`golang.org/x/net v0.48.0` direct→indirect | 预存陈旧分类，由强制 `go mod tidy` 揭示（rpa/worker_service_test.go 一直直接 import go-sqlite；x/net 已无直接 import）。版本号零变更 |
| 生产 require | 零新增/删除/升版 | T-76-01-03 缓解兑现 |

## Performance

- **Duration:** 38 min（含 3 次完整 backend 收尾门，每次 ~8-10 分钟）
- **Started:** 2026-08-23T05:59:18Z
- **Completed:** 2026-08-23T06:37:43Z
- **Tasks:** 3/3（含 1 个 Rule 1 修复 commit）
- **Files modified:** 5（go.mod / go.sum / 2 新测试文件 / 1 注释联动）

## Accomplishments

- INFRA-01 go.mod 侧落地：两个 pinned test-only 依赖 + 注释 + tidy 保活实证
- pkg/cache Redis 命令面（Set/Get/Exists/Delete、Incr/IncrBy、MSet/MGet、Hash 家族、Keys SCAN、EVAL Lua、GetStats）经 NewRedisCache→miniredis 真链路全绿，零 Docker
- 三坑防护具名用例：TestRedisTTLFastForward（R-1）、TestRedisGetStatsDegraded（R-2）、newMiniredisCache 构造器 PING（R-3 哨兵）
- geocoding httpmock PoC：生产构造器 + DefaultTransport 拦截，兼作全仓使用纪律活样板
- 收尾门全绿：`bash scripts/check-ci-local.sh backend` EXIT=0（lint 0 issues + 全量测试 + coverage gate 56.02% ≥ 55.5% floor）

## Task Commits

1. **Task 1: go.mod 落地两个 test-only 依赖** - `e3e9b04` (chore)
2. **Task 2: pkg/cache miniredis 冒烟 + 三坑防护 + 过期注释联动** - `eb08571` (test)
3. **Task 3: geocoding httpmock PoC + tidy 保活** - `6903703` (test)
4. **Rule 1 修复: PoC 令牌桶隔离** - `288403d` (fix)

## Files Created/Modified

- `go.mod` - direct +2 test-only（注释）/ indirect +1 / tidy 分类修正
- `go.sum` - miniredis/httpmock/gopher-lua/go-testdeep 哈希行
- `pkg/cache/redis_miniredis_76_01_test.go` - miniredis 冒烟 + 三坑防护（218 行 ≥ 80 门槛）
- `pkg/cache/cache_74_08_test.go` - :15 过期 D-12 注释改指向新文件（doc-only）
- `internal/services/operations/geocoding_httpmock_76_01_test.go` - httpmock PoC（55 行 ≥ 40 门槛）

## Decisions Made

- **R-2 断言语义按 pinned 源码改写**（见 Deviation 1）：v2.38.0 的 `INFO <section>` 对 server/memory/keyspace 返回 `section (...) is not supported` 错误（cmd_info.go:37），而非计划假设的"返回 clients 内容"。GetStats 的 `dbInfo, _ :=` 吞错 → `stats["keyspace_info"]` 必为空串。断言"非空"必然红。
- **go-testdeep 缺席 go.mod**：Go 1.17+ module graph pruning 下，依赖模块自身 test-only 依赖不进主模块 go.mod；RESEARCH 预告的 indirect +2 实际为 +1，go.sum 仍落 go-testdeep v1.14.0 哈希。
- **geocodeOKBody 复用而非复制**：与 analog 同包（operations），重复声明无法编译；直接引用既有常量。
- **tidy 分类修正保留**：回退只会把漂移推迟到下一次任何人跑 tidy；版本零变更，构建图不变。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] R-2「keyspace_info 非空」断言与 pinned v2.38.0 行为矛盾**
- **Found during:** Task 2
- **Issue:** 计划断言 `stats["keyspace_info"]` 非空；源码实证（cmd_info.go + cmd_info_test.go）`INFO keyspace` 返回错误 → GetStats 降级为空串
- **Fix:** TestRedisGetStatsDegraded 改为断言真实降级面：key 存在且为空串、redis_version/used_memory 缺席、hit_rate=0、key_count 真实可信（更强的降级语义锁定）
- **Files modified:** pkg/cache/redis_miniredis_76_01_test.go
- **Verification:** `go test -count=1 ./pkg/cache/` 全绿
- **Committed in:** eb08571

**2. [Rule 1 - Bug] PoC 消耗共享 BaiduAPIRateLimiter 令牌致 TestBaiduAPIRateLimiter 红**
- **Found during:** plan 收尾门（第一次全量跑）
- **Issue:** 计划称"单次 Geocode 调用无需动共享限流器"，但 `TestBaiduAPIRateLimiter` 断言全局限流器满额 100 令牌；PoC 一次 Allow() 后剩 99 → FAIL（`rate_limiter_test.go:180`）
- **Fix:** 照 analog 白盒换私有令牌桶 `svc.rateLimiter = NewRateLimiter(100, time.Hour)`（计划明文许可的手法）；httpClient 保持不动，const-URL 拦截示范意义不变
- **Files modified:** internal/services/operations/geocoding_httpmock_76_01_test.go
- **Verification:** `go test -count=1 ./internal/services/operations/` 全包绿；完整收尾门重跑 EXIT=0
- **Committed in:** 288403d

**3. [Rule 3 - Blocking/格式] 强制 tidy 揭示 2 处预存 go.mod 分类陈旧**
- **Found during:** Task 3（`go mod tidy` 收尾步）
- **Issue:** `glebarez/go-sqlite` 应为 direct（rpa/worker_service_test.go:12 一直直接 import）；`golang.org/x/net` 应为 indirect（已无直接 import）——均与本 plan 无关的预存状态
- **Fix:** 保留 tidy 规范化输出（版本零变更）；SYNOPSIS 首段预写该口径
- **Files modified:** go.mod
- **Verification:** `go build ./...` + 收尾门全绿
- **Committed in:** 6903703

---

**Total deviations:** 3 auto-fixed（2 × Rule 1 bug，1 × Rule 3 blocking）
**Impact on plan:** 全部为正确性必需；无 scope 膨胀；生产代码零改动（diff 仅 go.mod/go.sum/测试文件/1 行注释）。

## TDD 说明

Task 2 标记 `tdd="true"`，但交付物即测试本身（能力由 Task 1 的 pinned 依赖提供，无生产代码可写 GREEN 步）——RED 阶段不适用。测试落地即全绿，符合冒烟验证目的。plan 非 `type: tdd`，无 plan 级 RED/GREEN/REFACTOR 门。

## Issues Encountered

- commitlint 拒绝首条 commit 的超长 body 行（>100 字符）——缩短措辞后通过，非代码问题。
- 首次收尾门经 `| tail` 管道吞掉脚本真实退出码（汇总显示"存在失败"但 exit 0）——改为重定向日志文件重跑定位，发现 Deviation 2。

## Verification Results（plan 收尾门全项）

- `[ "$(grep -c "// test-only (v1.27 D-02)" go.mod)" -eq 2 ]` → **PASS**（计数=2）
- `go test -count=1 -run 'TestRedis' ./pkg/cache/` → **PASS**（5 用例：BasicCommandSurface / KeysScan / TTLFastForward / GetStatsDegraded / IncrementWithExpire）
- `go test -count=1 -run 'TestGeocoding' ./internal/services/operations/` → **PASS**
- `go mod tidy` 后 httpmock 仍在 go.mod → **PASS**（tidy 保活实证）
- 全量 `go test`（零 Docker）→ **PASS**
- `bash scripts/check-ci-local.sh backend` → **PASS EXIT=0**（lint 0 issues；weighted coverage 56.02% ≥ 55.50%；P1 8 包 + P2 10 包 floor 全过）
- `go build ./...` → **PASS**

## User Setup Required

None - 无外部服务配置需求（miniredis/httpmock 全部进程内）。

## Next Phase Readiness

- 76-02（Driver 工厂）/ 76-03（LDAPClientIface）/ 76-04（re-exec stub）/ 76-05（AST 守护）可与本 plan 并行，无依赖冲突
- Phase 78（core Init 链 / pkg/cache Redis 路径）可直接复用 newMiniredisCache 模式
- Phase 80（pkg/cache ≥70%）的 Redis 侧测试基建已就绪
- 债务提示：`HKeys` 对字段名的前缀裁剪行为（本 plan 用短字段名规避，未修，超 scope）

## Self-Check: PASSED

- 文件存在：redis_miniredis_76_01_test.go / geocoding_httpmock_76_01_test.go / cache_74_08_test.go / go.mod / 76-01-SUMMARY.md 全部 FOUND
- 提交存在：e3e9b04 / eb08571 / 6903703 / 288403d / 1fc79fb 全部在 git log 中 FOUND
- 工作树干净（coverage.out 等生成物已被 gitignore 覆盖，gate 日志已清理）

---

*Phase: 76-test-doubles*
*Completed: 2026-08-23*
