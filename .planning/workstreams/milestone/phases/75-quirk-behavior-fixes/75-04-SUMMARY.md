---
phase: 75-quirk-behavior-fixes
plan: 4
subsystem: agent
tags: [go, agent, retry, tls, logger, quirk]

requires:
  - phase: 75-01
    provides: MemoryCache.IncrementBy 修复,允许本计划测试翻转

provides:
  - Q-10 retry.containsIgnoreCase 精确匹配(IsNetworkError 不再对任意长串恒 true)
  - Q-12 InitLogger 非法 level 返回 error(同时降级 info 保证 logger 可用)
  - Q-13 NewTLSConfigFromConfig 全空参数返回 error + cmd/agent/main.go TLS 禁用 guard

affects:
  - Phase 77 agent config 测试(将直接断言 InitLogger/NewTLSConfigFromConfig 新行为)
  - cmd/agent/main.go 启动链

tech-stack:
  added: []
  patterns:
    - "非法配置返回 error + caller guard 同 commit,避免启动 Fatal"

key-files:
  created: []
  modified:
    - internal/agent/pkg/retry/retry.go
    - internal/agent/pkg/retry/retry_74_12_test.go
    - internal/agent/server/logger.go
    - internal/agent/server/jwt_auth.go
    - internal/agent/server/agent_smoke_test.go
    - cmd/agent/main.go

key-decisions:
  - "Q-10 业务影响面为零:全仓 grep 确认 IsNetworkError/retry. 在非 retry 包测试/生产代码中零调用方"
  - "Q-13 校验与 cmd/agent/main.go guard 必须同 commit,否则 TLS 禁用配置在启动链 Fatal"

patterns-established: []

requirements-completed: [QUIRK-02]

duration: 35min
completed: 2026-08-23
---

# Phase 75 Plan 4: Agent 三项 QUIRK 行为修正

**修复 retry.containsIgnoreCase 精确匹配、InitLogger 非法 level 报错、NewTLSConfigFromConfig 空参校验及启动链 TLS 禁用 guard,agent/retry 包全绿。**

## Performance

- **Duration:** 35 min
- **Started:** 2026-08-23T03:15:00Z
- **Completed:** 2026-08-23T03:50:00Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Q-10: `retry.containsIgnoreCase` 改为 `strings.Contains(strings.ToLower(s), strings.ToLower(substr))`,业务长串不再被误判为网络错误;三处锁定断言翻转。
- Q-12: `InitLogger` 解析非法 level 时降级为 info 并完整初始化 logger,同时返回 error;调用方 `cmd/agent/main.go` 已判 err 可安全继续。
- Q-13: `NewTLSConfigFromConfig` 全空参数返回 error;`cmd/agent/main.go` 仅当提供 TLS 文件时才调用,否则传 nil 走 `NewJWTAuthenticator` 默认 TLS1.3 安全分支。
- 全量验证: `go build ./...` + `go test -count=1 ./...` + `bash scripts/check-ci-local.sh backend --no-npm-ci` 全绿。

## Task Commits

每个任务按原计划应为原子 commit:

1. **Task 1: Q-10 retry.containsIgnoreCase 精确匹配** - `46c1485` (`fix(quirk-10)`)
2. **Task 2: Q-12 InitLogger 非法 level 返回 error** - 变更被合并入 `2b6b7d1` (见 Deviations)
3. **Task 3: Q-13 NewTLSConfigFromConfig 空参校验 + main.go guard** - `2b6b7d1` (`fix(quirk-13)`)

**Plan metadata:** 待本 SUMMARY 提交后补充。

## Files Created/Modified

- `internal/agent/pkg/retry/retry.go` - 引入 `strings`,`containsIgnoreCase` 改为精确忽略大小写包含。
- `internal/agent/pkg/retry/retry_74_12_test.go` - 翻转 `TestIsNetworkError`/`TestDoWithRetry_NonRetryableFailsFast`/`TestContainsHelpers` 三处断言。
- `internal/agent/server/logger.go` - `InitLogger` 非法 level 返回 error,logger 仍完成初始化。
- `internal/agent/server/jwt_auth.go` - `NewTLSConfigFromConfig` 全空参数校验。
- `internal/agent/server/agent_smoke_test.go` - 追加 `InitLogger` 回归测试,翻转 `NewTLSConfigFromConfig` 全空断言。
- `cmd/agent/main.go` - TLS 禁用配置走 nil 默认分支,避免启动 Fatal。

## Decisions Made

- **Q-10 零生产调用方复核:** 执行 `grep -rn "IsNetworkError\|retry\." --include="*.go" . | grep -v "_test.go" | grep -v "internal/agent/pkg/retry"` 结果为空,确认业务影响面为零,风险全在测试翻转。
- **Q-13 同 commit 原则:** 空参校验与 `cmd/agent/main.go` guard 必须同 commit,否则 TLS 禁用配置启动即 Fatal。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 并发执行导致 Q-12/Q-13 commit 边界混合且夹带其他计划文件**
- **Found during:** Task 3 commit 阶段
- **Issue:** Task 2 的 `fix(quirk-12)` 提交后,其他并发 executor 重置/推进了 `main`,导致 Task 3 重试提交时 `fix(quirk-13)` 同时包含了 Q-12 的 `logger.go` 变更、`agent_smoke_test.go` 的 Q-12 回归测试,以及两个其他计划文件(`internal/device/device_74_08_test.go`、`internal/device/model_extractor.go`)。最终 `main` 上不存在独立的 `fix(quirk-12)` commit。
- **Fix:** 代码语义与测试已正确落地;未重写历史(避免干扰并发 executor 已基于当前 HEAD 的工作)。将 commit 边界异常记录在本 SUMMARY 中。
- **Files modified in fix(quirk-13):** `cmd/agent/main.go`, `internal/agent/server/jwt_auth.go`, `internal/agent/server/logger.go`, `internal/agent/server/agent_smoke_test.go`, `internal/device/device_74_08_test.go`, `internal/device/model_extractor.go`
- **Verification:** `go build ./...` + `go test -count=1 ./...` + `bash scripts/check-ci-local.sh backend --no-npm-ci` 全绿。
- **Committed in:** `2b6b7d1` (mixed boundary)

**2. [Rule 3 - Blocking] 最终 docs(75-04) commit 夹带了其他 workstream 的未跟踪文件**
- **Found during:** 最终 metadata commit 阶段
- **Issue:** `git add` 仅显式指定了 `75-04-SUMMARY.md` 与 `STATE.md`,但提交后发现 `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-CONTEXT.md` 与 `82-DISCUSSION-LOG.md` 被一并提交( Commit `40965fb`)。推测是并发 executor 此前已将这些文件 staged 到 index 但未成功 commit,本提交继承了已 staged 的 index。
- **Fix:** 文件内容本身为合法 planning 产物,未删除(避免丢失他人工作);在 SUMMARY 中记录 attribution 异常。
- **Files modified in docs(75-04):** `.planning/workstreams/milestone/STATE.md`, `.planning/workstreams/milestone/phases/75-quirk-behavior-fixes/75-04-SUMMARY.md`, `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-CONTEXT.md`, `.planning/workstreams/frontend-coverage/phases/82-coverage-caliber-and-governance/82-DISCUSSION-LOG.md`
- **Committed in:** `40965fb` (mixed attribution)

---

**Total deviations:** 2 auto-fixed (2 blocking, both caused by concurrent execution interference)
**Impact on plan:** 代码行为正确且 gate 全绿;commit 边界/归因未达预期,需 verifier 注意 `fix(quirk-13)` 与 `docs(75-04)` 两个 commit 的范围超出本计划文件。

## Issues Encountered

- `bash scripts/check-ci-local.sh backend --no-npm-ci` 首次失败:遗留 `go.exe` 进程锁定 `coverage.out`,导致脚本 `rm -f coverage.out` 报 "Device or resource busy"。终止残留 `go.exe` 后重试通过。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Q-10/Q-12/Q-13 行为已修正,Phase 77 agent config 测试可直接断言新行为。
- 无阻塞问题;gate 无倒退。

## Self-Check: PASSED

- [x] `internal/agent/pkg/retry/retry.go` 包含 `strings.Contains(strings.ToLower(s), strings.ToLower(substr))`
- [x] `internal/agent/server/logger.go` 包含 `return fmt.Errorf("无效的日志级别 %q: %w", logLevel, parseErr)`
- [x] `internal/agent/server/jwt_auth.go` 包含 TLS 全空校验
- [x] `cmd/agent/main.go` 包含 `TLSCertFile != "" || TLSKeyFile != "" || CAFile != ""` guard
- [x] `go test -count=1 ./internal/agent/pkg/retry/... ./internal/agent/server/...` pass
- [x] `go build ./...` pass
- [x] `go test -count=1 ./...` pass
- [x] `bash scripts/check-ci-local.sh backend --no-npm-ci` pass
- [x] git log 包含 `fix(quirk-10)` (`46c1485`) 与 `fix(quirk-13)` (`2b6b7d1`)

---
*Phase: 75-quirk-behavior-fixes*
*Completed: 2026-08-23*
