---
phase: 75-quirk-behavior-fixes
plan: 6
subsystem: database
tags: [go, gorm, menu, migration, quirk-11]

requires:
  - phase: 75-01
    provides: MemoryCache QUIRK fix 完成,为本 plan 的菜单树行为修正提供执行上下文

provides:
  - normalizeParentID 语义统一(menu_service.go 与 requests 一致)
  - migration_210: 存量 sys_menu.parent_id='0' 归一为 NULL
  - database.go 双方言分支注册 Migrate210
  - 三层回归测试(迁移幂等+孤儿检查、normalizeParentID 单元、Create/Update "0" 落库 NULL)

affects:
  - menu-tree rendering
  - sys_menu data consistency
  - Phase 78 core Close 链(无孤儿节点假设)

tech-stack:
  added: []
  patterns:
    - "幂等迁移: WHERE 精确匹配旧值 + RowsAffected 判定 changed"
    - "双方言注册: sqlite/postgres 分支并列挂载"
    - "census-before-migration: 修复前实测存量行数"

key-files:
  created:
    - internal/core/db/migrations/migration_210_normalize_menu_parent_id.go
    - internal/core/db/migrations/migration_210_normalize_menu_parent_id_test.go
  modified:
    - internal/services/system/menu_service.go
    - internal/services/system/menu_service_test.go
    - internal/core/db/database.go

key-decisions:
  - "menu_service.normalizeParentID 与 requests.normalizeParentID 统一为 nil/\"\"/\"0\" 全塌缩为 nil,避免 Update 路径落库字面 '0'"
  - "迁移 210 不强制失效菜单缓存: 当前 menuService 缓存方法为空操作,且迁移在启动早期执行;按 209 同款仅记录 changed 日志,不扩 core.go 范围"
  - "census 结论: 种子面无 parent_id='0' 来源,本地 SQLite 0 行;迁移以幂等兜底为主"

requirements-completed: [QUIRK-03]

duration: 45min
completed: 2026-08-23
---

# Phase 75 Plan 6: 修复 Q-11 normalizeParentID 双实现分歧并归一 parent_id='0'

**统一 menu_service 与 requests 的 normalizeParentID 语义,新增迁移 210 将存量 sys_menu.parent_id='0' 归一为 NULL,消除菜单树孤儿节点**

## Performance

- **Duration:** 45 min
- **Started:** 2026-08-23T03:07:00Z (estimated)
- **Completed:** 2026-08-23T03:52:27Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- 统一 `internal/services/system/menu_service.go` 的 `normalizeParentID` 为 requests 语义,`nil`/""/"0" 全塌缩为 `nil`
- 新建 `migration_210_normalize_menu_parent_id.go`,幂等将 `parent_id='0'` 更新为 `NULL`
- 在 `internal/core/db/database.go` 的 sqlite 与 postgres 分支各注册 `Migrate210`
- 新增三层回归测试:迁移幂等 + 迁移后孤儿计数为 0、`normalizeParentID` 单元四态覆盖、Update "0" 落库 `NULL`
- 完成 census-before-migration:种子面无 `parent_id='0'` 来源,本地 SQLite 0 行

## Task Commits

1. **Task 2: Q-11 — normalizeParentID 统一 + 迁移 210 + 注册 + 三层测试(单一原子 commit)** - `8172f21` (fix)

**Plan metadata:** 待最终 `docs(75-06): complete ...` commit

## Files Created/Modified

- `internal/services/system/menu_service.go` - `normalizeParentID` 统一为 requests 语义
- `internal/services/system/menu_service_test.go` - 新增 `TestMenuService_Create_ZeroParent_Normalized`、`TestMenuService_Update_ZeroParent_Normalized`、`TestNormalizeParentID`
- `internal/core/db/migrations/migration_210_normalize_menu_parent_id.go` - 新建迁移 210
- `internal/core/db/migrations/migration_210_normalize_menu_parent_id_test.go` - 迁移幂等、无脏行、孤儿检查回归
- `internal/core/db/database.go` - sqlite/postgres 双分支注册 `Migrate210NormalizeMenuParentID`

## Decisions Made

- **统一语义优先:** `menu_service.go` 主动对齐 `requests/menu_requests.go`,避免双实现分歧导致 Update 落库 "0"。
- **不扩 core.go 缓存失效:** 当前 `menuService` 缓存实现为空操作,迁移内仅记录 `changed` 日志,不引入新的 `Database` 字段。
- **迁移幂等兜底:** 即使 census 为 0,迁移仍以 `WHERE parent_id='0'` 幂等注册,保护未来可能的数据漂移。

## Deviations from Plan

### Issues Encountered / Scoped Verification

**1. `scripts/check-ci-local.sh` 因并发占用 coverage.out 无法原样跑完**
- **Found during:** Task 3 (full verification)
- **Issue:** 并行 executor 中其他 plan 持有 `coverage.out`,`rm -f coverage.out` 报 `Device or resource busy`,脚本在 backend Test 段退出
- **Fix:** 按 prompt "scope verification" 指示,用替代覆盖文件 `coverage_75_06.out` 跑完同等的 `go test -coverprofile` + `bash .github/scripts/check-coverage.sh coverage_75_06.out .coverage-threshold`;lint 段已在本机通过
- **Files modified:** 无(临时文件 `coverage_75_06.out` 已清理)
- **Verification:**
  - `go build ./...` PASS
  - `go test -count=1 ./...` PASS
  - `golangci-lint run --timeout=5m ./...` PASS (0 issues)
  - `go test -timeout 15m -count=1 -coverprofile=coverage_75_06.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` PASS
  - `bash .github/scripts/check-coverage.sh coverage_75_06.out .coverage-threshold` PASS (weighted 55.75% >= 55.50%, P1/P2 floors PASS)
- **Committed in:** N/A (verification artifact,未提交)

---

**Total deviations:** 1 scoped-verification workaround (由并发文件锁导致,非代码改动)
**Impact on plan:** 代码改动与测试全部通过;仅标准脚本因并发资源占用无法原样执行,已通过等效命令验证。

## Issues Encountered

- `git commit --only` 在 Windows/Git Bash 下对未跟踪新文件匹配异常,改用 `git add` 后正常 `git commit` 完成单一原子 commit。
- 并发 executor 占用 `coverage.out`,导致 `scripts/check-ci-local.sh` 的 `rm` 步骤失败;已用替代覆盖文件完成同等验证。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Q-11 修复完成,menu_service 与 requests 语义一致
- 迁移 210 已双方言注册,启动期自动执行
- 菜单树查询无孤儿节点假设已用测试锁定
- 无阻塞项,可继续 Phase 75 后续 QUIRK 修复

## Self-Check: PASSED

- [x] `internal/core/db/migrations/migration_210_normalize_menu_parent_id.go` exists
- [x] `internal/core/db/migrations/migration_210_normalize_menu_parent_id_test.go` exists
- [x] Commit `8172f21` exists in git log
- [x] `go build ./...` PASS
- [x] `go test -count=1 ./...` PASS
- [x] Coverage gate (alternative file) PASS

---
*Phase: 75-quirk-behavior-fixes*
*Completed: 2026-08-23*
