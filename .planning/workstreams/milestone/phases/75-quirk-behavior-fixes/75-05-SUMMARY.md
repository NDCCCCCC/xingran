---
phase: 75-quirk-behavior-fixes
plan: 5
subsystem: utils/system
tags: [go, quirk, config-diff, disk-info, cpu-usage, mutex, build-constraints]

requires:
  - phase: 75-01
    provides: MemoryCache.IncrementBy fixes unlocked downstream QUIRK work

provides:
  - GetUnifiedDiff 相同配置返回带 headers 的空 diff
  - GetDiskInfoDetailed 去递归（Windows GetDiskFreeSpaceExW + Unix syscall.Statfs）
  - cpu_linux GetCPUUsage mutex 保护与去递归

affects:
  - internal/api/v1/monitor 详细磁盘端点
  - Linux 系统 CPU 使用率采样

tech-stack:
  added: []
  patterns:
    - "平台相关实现按 //go:build windows / !windows 拆分为独立文件，保持签名与错误语义不变"
    - "包级全局采样状态用 sync.Mutex 保护，避免并发数据竞争"

key-files:
  created:
    - internal/pkg/system/disk_info_windows.go
    - internal/pkg/system/disk_info_unix.go
  modified:
    - internal/utils/config_diff.go
    - internal/utils/config_diff_test.go
    - internal/pkg/system/disk_info.go
    - internal/pkg/system/system_test.go
    - internal/pkg/system/cpu_linux.go

key-decisions:
  - "Q-15 仅替换 getDiskInfoByPlatform 的实现，GetAllDiskInfo/GetDiskInfoDetailed 签名与错误语义零变化"
  - "M-2 保持首次调用 100ms 间隔重采样，行为与递归版等价"
  - "Q-15 Windows 实现复用同包已有的 procGetDiskFreeSpaceExW lazy proc，不新增 DLL 加载"

requirements-completed: [QUIRK-02]

duration: 22min
completed: 2026-08-23
---

# Phase 75 Plan 5: utils/system QUIRK 修正 + cpu_linux 去递归 Summary

**GetUnifiedDiff 空 diff 带 headers、GetDiskInfoDetailed 平台实现去递归、cpu_linux 加互斥锁去递归**

## Performance

- **Duration:** 22 min
- **Started:** 2026-08-23T03:26:51Z
- **Completed:** 2026-08-23T03:49:02Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Q-14: `GetUnifiedDiff` 相同配置现在返回 `"--- old\n+++ new\n"`，可与“未计算”区分；不同配置输出不变。
- Q-15: 删除 `disk_info.go` 中的死递归，新增 `disk_info_windows.go`/`disk_info_unix.go` 平台实现；`system_test.go` 恢复真实用例。
- M-2: `cpu_linux.go` 用 `sync.Mutex` 保护全局采样状态，首次调用改为就地 100ms 重采样，消除自递归与并发竞争。

## Task Commits

Each task was committed atomically (or verified already present in main):

1. **Task 1: Q-14 — GetUnifiedDiff 相同配置返回带 headers 的空 diff** - `996cc0b` (fix)
2. **Task 2: Q-15 — GetDiskInfoDetailed 去递归（平台实现，M-3 最小行为面）** - `bb2716b` (fix/quirk-4 commit that concurrently carried the Q-15 files)
3. **Task 3: M-2 — cpu_linux mutex 去递归（行为不变）+ 全量验证扫描** - `52d75e9` (refactor)

**Plan metadata:** pending

## Files Created/Modified

- `internal/utils/config_diff.go` - 空 diff 返回带 headers 的 `"--- old\n+++ new\n"`
- `internal/utils/config_diff_test.go` - 翻转 `TestGetUnifiedDiff_Same` 断言
- `internal/pkg/system/disk_info.go` - 删除递归版 `getDiskInfoByPlatform`
- `internal/pkg/system/disk_info_windows.go` - Windows 平台实现（`GetDiskFreeSpaceExW`）
- `internal/pkg/system/disk_info_unix.go` - Unix 平台实现（`syscall.Statfs`）
- `internal/pkg/system/system_test.go` - 恢复 `TestGetDiskInfoDetailed` 真实用例
- `internal/pkg/system/cpu_linux.go` - 增加 `sync.Mutex` 并去掉自递归

## Decisions Made

- 遵循计划要求，Q-15 仅改动平台实现层，未扩展 `GetDiskInfoDetailed` 的公开签名。
- Windows 文件复用 `disk_windows.go` 中已有的 `procGetDiskFreeSpaceExW`，保持 DLL 加载方式一致。

## Deviations from Plan

### Auto-fixed Issues

None - no unplanned code changes were required.

### Execution Deviations

**1. Q-15 fix did not land as a standalone `fix(quirk-15)` commit**
- **Found during:** Task 2 commit attempt
- **Issue:** Concurrent executor had already committed the identical Q-15 implementation to `main` under commit `bb2716b` (`fix(quirk-4): ...`). Our working-tree changes for `disk_info*.go` and `system_test.go` therefore matched HEAD.
- **Fix:** Verified the HEAD tree contains the exact Q-15 behavior (recursive function removed, `disk_info_windows.go`/`disk_info_unix.go` present, `TestGetDiskInfoDetailed` restored). Aborted the duplicate `fix(quirk-15)` commit to avoid redundant history.
- **Files affected:** `internal/pkg/system/disk_info.go`, `internal/pkg/system/disk_info_windows.go`, `internal/pkg/system/disk_info_unix.go`, `internal/pkg/system/system_test.go`
- **Verification:** `go test ./internal/pkg/system/...` and `GOOS=linux go build ./internal/pkg/system/` both pass; `git show HEAD:internal/pkg/system/disk_info_windows.go` returns the expected implementation.
- **Committed in:** `bb2716b` (concurrent plan)

**2. First M-2 commit accidentally included unrelated in-flight files**
- **Found during:** Task 3 commit review (`git show --stat HEAD`)
- **Issue:** `refactor(75-05)` initially recorded 6 files including `internal/core/db/database.go`, menu-service files, and migration files from other concurrent plans.
- **Fix:** `git reset --soft HEAD~1` to undo the mixed commit, then staged only `internal/pkg/system/cpu_linux.go` and re-committed.
- **Files modified:** `internal/pkg/system/cpu_linux.go` only in final commit
- **Verification:** `git show --stat HEAD` confirms 1 file changed; D-03 scope fence restored.
- **Committed in:** `52d75e9`

---

**Total deviations:** 2 execution-level (0 code auto-fix)
**Impact on plan:** Code outcome matches plan; commit granularity deviated only because Q-15 was already merged concurrently.

## Issues Encountered

- `git commit` failed once with `cannot lock ref 'HEAD'` due to concurrent executor committing to `main` while we were staging. Waited and inspected history; no data loss.
- `lint-staged` printed `could not find any staged files matching configured tasks` before each commit, but commits still succeeded.

## User Setup Required

None - no external service configuration required.

## Verification Log

- `go test -count=1 ./internal/utils/... ./internal/pkg/system/...` - PASS
- `GOOS=linux go build ./internal/pkg/system/` - PASS
- `go build ./...` - PASS
- `go test -count=1 ./...` - PASS
- `bash scripts/check-ci-local.sh backend --no-npm-ci` - PASS（weighted 55.74% >= 55.50%, P1/P2 floors passed）

## Next Phase Readiness

- utils/system 包修正完成，M-2/Q-15 编译面已验证。
- 可进入 Phase 75 下一项 QUIRK 修复计划。

## Self-Check: PASSED

- `internal/pkg/system/disk_info_windows.go` - FOUND
- `internal/pkg/system/disk_info_unix.go` - FOUND
- `internal/utils/config_diff.go` - FOUND
- `internal/pkg/system/cpu_linux.go` - FOUND
- Commit `996cc0b` - FOUND
- Commit `bb2716b` - FOUND
- Commit `52d75e9` - FOUND

---
*Phase: 75-quirk-behavior-fixes*
*Completed: 2026-08-23*
