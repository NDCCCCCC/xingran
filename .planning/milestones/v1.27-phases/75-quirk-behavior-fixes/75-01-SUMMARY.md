---
phase: 75-quirk-behavior-fixes
plan: 1
subsystem: cache / core
tags: [go, testing, quirk, memory-cache, captcha, redis-incr]

requires: []

provides:
  - "MemoryCache.IncrementBy 缺 key 按 0 起算并新建缓存项(对齐 Redis INCR)"
  - "MemoryCache.IncrementBy 非数字串返回 ErrNotInteger 而非静默 0 累加"
  - "pkg/cache/cache_74_08_test.go 锁定断言翻转 + 回归用例"
  - "internal/core/core_74_08_test.go 三处 captcha workaround 删除,GenerateCaptcha 真实链路直测"

affects:
  - "75-quirk-behavior-fixes"

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified:
    - "pkg/cache/memory.go"
    - "pkg/cache/errors.go"
    - "pkg/cache/cache_74_08_test.go"
    - "internal/core/core_74_08_test.go"

key-decisions:
  - "生产 Redis 路径零影响:MemoryCache 仅用于 dev/测试,Redis 走 RateLimitCache.IncrementWithExpire 不受影响"
  - "新建缓存项 TTL = 0(永不过期),对齐 Redis INCR 新建 key 无 TTL 的语义"
  - "ErrNotInteger 沿用中文错误值惯例,不额外加谓词函数"

patterns-established: []

requirements-completed:
  - QUIRK-01

duration: 45min
completed: 2026-08-23
---

# Phase 75 Plan 1: MemoryCache.IncrementBy QUIRK fix Summary

**MemoryCache.IncrementBy 对齐 Redis INCR 语义(缺 key 0 起算、非数字串返回 ErrNotInteger),core captcha 测试改为真实 GenerateCaptcha 链路直测**

## Performance

- **Duration:** 45 min
- **Started:** 2026-08-23T02:40:00Z
- **Completed:** 2026-08-23T03:23:09Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- 修复 `MemoryCache.IncrementBy` 缺 key 时 nil 解引用 panic,改为按 0 起算并新建缓存项
- 新增 `ErrNotInteger` 错误值,非数字串不再静默按 0 累加
- 翻转 `pkg/cache/cache_74_08_test.go` 两处锁定断言并补充回归用例(缺 key 返回 1、TTL -1、原值保持)
- 删除 `internal/core/core_74_08_test.go` 三处 captcha 手工种数据 workaround,改为直测 `GenerateCaptcha` 真实链路

## Task Commits

Each task was committed atomically:

1. **Task 1: Q-1 — IncrementBy 缺 key 0 起算 + 翻转断言 + 删 core captcha workaround** - `030fdf3` (fix)
2. **Task 2: Q-2 — IncrementBy 非法数字串返回 ErrNotInteger** - `cf9c88b` (fix)

**Plan metadata:** *(pending summary commit)*

## Files Created/Modified

- `pkg/cache/memory.go` - `IncrementBy` 缺 key / 过期按 0 起算、非数字串返回 `ErrNotInteger`
- `pkg/cache/errors.go` - 新增 `ErrNotInteger`
- `pkg/cache/cache_74_08_test.go` - 翻转 `IncrementDecrement` 断言,补充 `fresh` key 回归与 `ErrNotInteger` 回归
- `internal/core/core_74_08_test.go` - 删除三处 captcha workaround,真实生成验证码后取 code 验证

## Decisions Made

- 生产 Redis 路径零影响:本次改动仅针对 `MemoryCache`,生产 captcha 限流走 `RateLimitCache.IncrementWithExpire`
- 新建项 `Expiration: 0` 表示永不过期,对齐 Redis INCR 新建 key 无 TTL
- `ErrNotInteger` 沿用现有中文错误值惯例,未加谓词函数(RESEARCH Q-2 仅要求错误值进 `errors.go`)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 提交信息前缀调整以通过 commitlint**
- **Found during:** Task 1 / Task 2 commit
- **Issue:** 计划给出的 `fix(quirk-1): MemoryCache.IncrementBy ...` 与 `fix(quirk-2): IncrementBy ...` 因 subject 首字母大写触发仓库 commitlint `subject-case` 规则,提交被拒绝
- **Fix:** 在 subject 前加中文动词前缀,使首词为中文:"`fix(quirk-1): 修复 MemoryCache...`"、"`fix(quirk-2): 修复 IncrementBy...`"
- **Files modified:** 无代码文件变更,仅 commit message 与计划示例不同
- **Verification:** 两次提交均通过 pre-commit / commitlint 钩子并成功入库
- **Committed in:** `030fdf3`, `cf9c88b`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 仅为 commit message 格式调整,语义与计划一致;无代码/行为偏离

## Issues Encountered

- **`go test -race ./pkg/cache/... ./internal/core/...` 无法执行**:本机 Windows Git Bash 环境缺少 C 编译器(`gcc`/`clang`/`cl` 均未安装),`-race` 依赖 CGO 报错 `cgo: C compiler "gcc" not found`。代码层面已确保锁保护正确;race 验证需在已安装 gcc 的环境(如 CI Linux runner 或本地 MinGW-w64)补跑。
- **`bash scripts/check-ci-local.sh backend --no-npm-ci` 通过**,但 golangci-lint 输出一条与本次改动无关的 warning(`migration_helpers.go` 路径解析失败),lint 最终判定 `0 issues`。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- QUIRK-01 已完成,`GenerateCaptcha` 在 MemoryCache 下可直测,为 Phase 78 core Init 链真实覆盖打下基础
- 后续 plan(75-02..75-15)可继续按五步法执行
- 建议在下一次推送前,在含 gcc 的环境补跑 `go test -race ./pkg/cache/... ./internal/core/...` 以满足 RESEARCH 硬约束 6

## Self-Check

- [x] `pkg/cache/memory.go` modified
- [x] `pkg/cache/errors.go` modified
- [x] `pkg/cache/cache_74_08_test.go` modified
- [x] `internal/core/core_74_08_test.go` modified
- [x] Commit `030fdf3` exists (`fix(quirk-1)`)
- [x] Commit `cf9c88b` exists (`fix(quirk-2)`)
- [x] `go build ./...` PASS
- [x] `go test -count=1 ./...` PASS
- [x] `bash scripts/check-ci-local.sh backend --no-npm-ci` PASS
- [ ] `go test -race ./pkg/cache/... ./internal/core/...` BLOCKED by missing gcc (environment limitation)

## Self-Check: PASSED (with race-detector environment caveat)

---
*Phase: 75-quirk-behavior-fixes*
*Completed: 2026-08-23*
