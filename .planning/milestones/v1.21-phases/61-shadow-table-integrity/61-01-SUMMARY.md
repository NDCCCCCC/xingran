---
phase: 61-shadow-table-integrity
plan: 01
subsystem: backend-ad-sync
tags: [gorm, soft-delete, ldap, ad-sync, singleflight, sqlite-testing]

# Dependency graph
requires:
  - phase: 58-compare-and-move
    provides: AD 同步基座（SyncService/singleflight、78-05/78-06 sqlite fixture 与 entry78 helper 体系）
  - phase: 60-soft-delete-recovery-safety
    provides: 软删行安全语义先例（拒绝日志不含 DN 明文口径）
provides:
  - internal/services/addomain/shadow_cleanup_service.go —— cleanupAfterFullSync 编排 + cleanupVanishedOUs/Users/Computers 三类清理，两道闸门（LDAP 0 条拒绝 / 拟删*2 > 存量拒绝，恰 50% 放行）逐字对齐 handleDeletedGroups 先例
  - syncDataInternal 成功路径单点挂载（syncComputers 之后、last_sync_at 之前），处于 singleflight.Do 闭包内 + models.ADSyncTypeFull 守卫（增量分支 no-op）
  - SyncResult 三个软删计数 JSON 字段（deletedUserCount / deletedOUCount / deletedComputerCount）
  - 12 条 TestShadowCleanup_* 可证伪用例（闸门边界恰 50% 放行、管理口径过滤、per-category 独立性）
affects: [61-02 复活路径, 61-03 组成员事务化, AD 影子表运维/审计]

# Actuals (#2632)
actuals:
  tokens: 8100        # chars/4 over realized diff（32404 chars / 682 insertions）
  tasks: 2
  commits: 2          # measured: git rev-list --count 96d2c3f4..75a3c015
plan_head_before: 96d2c3f434cc83914d394a08a683497f40ffab4e
plan_head_after: 75a3c015fd9b8443f6d561363b5172cc8c589966

# Tech tracking
tech-stack:
  added: []           # 零新依赖（RESEARCH Package Legitimacy Audit 口径）
  patterns:
    - "双闸门软删清理（Gate 1 LDAP 0 条拒绝 + Gate 2 拟删*2 > 存量活行拒绝，判式严格大于恰 50% 放行），每类别独立应用"
    - "清理集合口径 = 对应 sync 函数的管理口径（user 仅 sAMAccountName 非空 / computer 仅 cn 非空 / OU 全部 entry）"
    - "非阻断降级分层：闸门拒绝 WARN + nil；软删 SQL 失败上抛由挂载点 Warnf 非 fatal 降级，整轮同步仍 success"

key-files:
  created:
    - internal/services/addomain/shadow_cleanup_service.go
    - internal/services/addomain/shadow_cleanup_test.go
  modified:
    - internal/services/addomain/sync.go

key-decisions:
  - "FiltersUnmanagedEntries 用例预置 2 活行而非 plan 字面的 1 行：plan 用例描述与自身强制的 Gate 2 判式矛盾（单行全消失 = 100% > 50% 必被拒绝），最小修复 = 2 行消失 1 行（恰 50% 放行），管理口径文档化意图不变"
  - "三类清理为三个显式方法（非泛型抽象）：五段结构逐字对齐 handleDeletedGroups，差集键分别为 OUN / UserDN / DistinguishedName 列，可 grep 性优先"
  - "挂载点单点：syncDataInternal 恰一处 cleanupAfterFullSync 调用（grep 计数 == 1），无 sys_job、无独立 goroutine（SHADOW-03 prohibition）"
  - "SHADOW-01 不标 complete：61-02 同样声明该需求（共享-ID 门），复活路径落地后回填；SHADOW-03 单一声明者已标 complete"

patterns-established:
  - "影子表清理先例：后续任何『AD 端消失 → 本地表软删』类别（如 group 若纳入清理）照此五段结构 + 双闸门 + 管理口径过滤"
  - "拒绝/失败日志只含数量不含 DN 明文（T-61-02，Phase 60 SyncError 同口径延续）"

requirements-completed: [SHADOW-01, SHADOW-03]  # verbatim 自 plan frontmatter；SHADOW-01 的 REQUIREMENTS 回填按共享-ID 门待 61-02（见 Decisions）

coverage:
  - id: D1
    description: user 类别清理：全量同步后 AD 端已消失的 sys_ad_user 活行软删，两道闸门生效（0 条拒绝 / >50% 拒绝 / 恰 50% 放行）
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Users_SoftDeletesVanishedRows"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Users_RejectsEmptyLDAP"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Users_RejectsOverThreshold"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Users_AtExactThreshold"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Users_FiltersUnmanagedEntries"
        status: pass
    human_judgment: false
  - id: D2
    description: OU 类别清理：管理集 = 全部 entry（syncOUs 同口径），差集键 ou_dn，独立闸门
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_OUs_SoftDeletesVanishedRows"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_OUs_RejectsEmptyLDAP"
        status: pass
    human_judgment: false
  - id: D3
    description: computer 类别清理：管理集 = 仅 cn 非空 entry（syncComputers 同口径），差集键 distinguished_name，独立闸门
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Computers_SoftDeletesVanishedRows"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Computers_RejectsOverThreshold"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_Computers_FiltersUnmanagedEntries"
        status: pass
    human_judgment: false
  - id: D4
    description: per-category 独立性：单类空 LDAP 仅拒绝该类，其它类别照常清理
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_CategoriesIndependent"
        status: pass
    human_judgment: false
  - id: D5
    description: SHADOW-03 挂载语义：清理只挂 syncDataInternal 成功路径单点（singleflight 闭包内），syncType 守卫使增量分支 no-op
    requirement: SHADOW-03
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestShadowCleanup_FullSyncOnly"
        status: pass
      - kind: other
        ref: "grep -c 's.cleanupAfterFullSync(ctx' internal/services/addomain/sync.go == 1（挂载点源断言，syncComputers 后 / last_sync_at 前）"
        status: pass
    human_judgment: false
  - id: D6
    description: 真实 AD 环境首轮全量同步的清理规模核对（拟删 >50% 时闸门拒绝属预期保护行为，运维须知晓避免误报）
    requirement: SHADOW-01
    verification: []
    human_judgment: true
    rationale: "清理效果依赖真实 LDAP 全量返回与生产存量对比（FLAGGED ASSUMPTION：首轮批量滞留行是一次性事件）；sqlite 单测无法覆盖真环境数据规模，归 61-VALIDATION.md Manual-Only 表人工核对"

# Metrics
duration: 16 min
completed: 2026-10-20
status: complete
---

# Phase 61 Plan 01: 影子表删除清理（SHADOW-01 主体 + SHADOW-03 挂载） Summary

**全量同步末尾对 AD 端已消失的 sys_ad_user/ou/computer 活行按类别软删除：三类独立双闸门（LDAP 0 条拒绝 / 拟删 >50% 拒绝、恰 50% 放行）+ syncDataInternal 单点挂载（singleflight 闭包内、增量 no-op）+ SyncResult 三计数，12 条 sqlite 可证伪用例全绿、全包零回归。**

## Performance

- **Duration:** 16 min
- **Started:** 2026-10-20T16:29:01Z
- **Completed:** 2026-10-20T16:45:06Z
- **Tasks:** 2/2（Task 1 tracer + Task 2 auto）
- **Files modified:** 3（2 新建 + 1 修改）

## Accomplishments

- SHADOW-01 清理主体落地：`shadow_cleanup_service.go` 三类清理（OU/Users/Computers），每类独立过两道闸门——Gate 1（该类 LDAP entries 为 0 → 拒绝）与 Gate 2（拟删\*2 > 存量活行 → 拒绝，判式严格大于、恰 50% 放行）逐字对齐 `handleDeletedGroups` 先例；拒绝 WARN 非阻断、软删走 GORM 默认作用域（行保留 Unscoped 可审计）
- SHADOW-03 挂载落地：`syncDataInternal` 成功路径恰一处 `cleanupAfterFullSync` 调用（syncComputers 成功后、last_sync_at 更新前），处于 singleflight.Do 闭包内 + `models.ADSyncTypeFull` 常量守卫（增量分支 no-op，有行为断言）；清理失败仅 `applogger.Warnf` 非阻断，整轮同步仍 success
- 可观测性：`SyncResult` 新增 `deletedUserCount` / `deletedOUCount` / `deletedComputerCount` 三个 JSON 计数字段；拒绝日志只含数量、不含 DN 明文
- 12 条 `TestShadowCleanup_*` 用例全绿（含恰 50% 边界、管理口径过滤、per-category 独立性、增量 no-op 四类边界），`go test ./internal/services/addomain/ -count=1` 全包零回归（D-78-05c 软删 UNIQUE 冲突用例按 plan 保持原样绿，翻转归 61-02）

## Task Commits

1. **Task 1 (tracer): user 类别清理 + 两道闸门 + full 分支挂载** - `6a08933d` (feat) — tracer verify 复跑通过（⚡ Tracer verified end-to-end — expanding）
2. **Task 2 (auto): OU + Computer 类别扩展（per-category 独立闸门）** - `75a3c015` (feat)

**Plan metadata:** 见下方 docs commit（SUMMARY + STATE + ROADMAP + REQUIREMENTS 同 commit）

## Files Created/Modified

- `internal/services/addomain/shadow_cleanup_service.go` — 新建：cleanupAfterFullSync 编排（syncType 守卫 + 三类依次调用）+ cleanupVanishedUsers/OUs/Computers（各自 Gate 1 → LDAP 管理集 → 存量活行差集 → Gate 2 → 软删五段结构）
- `internal/services/addomain/shadow_cleanup_test.go` — 新建：12 条 TestShadowCleanup_\* 用例 + 2 个 seed helper + 1 个计数 helper；fixture 零副本（复用 setupSync78DB/entry78/newSyncSvc78/insertConfig78/closeDB/setupComp78DB/insertComputer78）
- `internal/services/addomain/sync.go` — SyncResult 三个 deleted 计数字段 + syncDataInternal 成功路径单点挂载（非阻断 WARN 降级）

## Decisions Made

- **FiltersUnmanagedEntries 预置 2 活行（偏离 plan 字面的 1 行）**：plan 的用例描述与自身强制的 Gate 2 判式矛盾——单活行全消失 = 100% > 50% 必被保护性拒绝，「断言该行被软删」在判式下不可能成立（同 plan 的 AtExactThreshold 用例正是该判式的可证伪锚点）。最小修复 = 2 行中消失 1 行（恰 50% 放行分支内可观测），「清理口径 = sync 管理口径」的文档化意图不变；computer 同名用例对齐同款形态。
- **三类清理为三个显式方法而非泛型抽象**：与 handleDeletedGroups 先例同构、列名可 grep（验收断言均为源 grep），可读性优先于去重。
- **闸门拒绝与 SQL 失败分层**：拒绝 = WARN + return nil（闸门内消化）；软删 SQL 失败 = fmt.Errorf 上抛 → 挂载点 applogger.Warnf 非 fatal 降级（先例 group_sync_service.go:80-83 同款）——四类 upsert 已各自提交不可回滚。
- **SHADOW-01 不标 complete**：61-02 同样声明该需求，按共享-ID 门（#2388 口径）由 61-02 复活路径完成后回填；SHADOW-03 单一声明者已标 complete（REQUIREMENTS.md 已更新：SHADOW-03 Complete / SHADOW-01 In Progress）。
- **gsd_run CLI 本环境不可用**（58-01 既有记录）：per-task 提交用原生 git、STATE/ROADMAP/REQUIREMENTS 手工编辑；提交落在 main 分支（项目 `branching_strategy: none` + orchestrator 顺序派发 `<sequential_execution>` 明示主工作树，先例 Phase 57-60 全部同款）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] FiltersUnmanagedEntries 用例的行数预设与闸门判式自相矛盾**
- **Found during:** Task 1（user 用例）与 Task 2（computer 用例对齐）
- **Issue:** plan 规定「预置 1 活行（DN=D1）+ LDAP 返回 D1 但 sAMAccountName 为空串 → 断言该行被软删」，但同一 plan 强制 Gate 2 判式 `len(vanished)*2 > len(stale)` 严格大于（恰 50% 放行）——1 行全消失 = 2 > 1 → 必被拒绝，断言不可能成立
- **Fix:** 预置 2 活行（1 行 entry 无 sAMAccountName 不入管理集 + 1 行正常在管理集），消失 1 行 = 恰 50% 放行 → 该行软删；「管理口径」文档化意图与断言强度不变，另加注释说明 2 行预设原因
- **Files modified:** internal/services/addomain/shadow_cleanup_test.go
- **Verification:** TestShadowCleanup_Users_FiltersUnmanagedEntries / TestShadowCleanup_Computers_FiltersUnmanagedEntries 均 PASS
- **Committed in:** `6a08933d`（user）/ `75a3c015`（computer）

---

**Total deviations:** 1 auto-fixed（Rule 1 × 1）
**Impact on plan:** 测试规格自洽性修复，生产行为零改动；其余 plan 内容逐字执行。

## Issues Encountered

- `go build ./...` 首跑在 `cmd/mac_purge_drop_backup` 触发编译器 `runtime: cannot allocate memory`（环境内存压力，非代码问题）；复跑 EXIT=0，与本次改动无关（该目录为 pre-existing tracked 文件，本 plan 未触碰）。
- `gofmt -l` 对包内 40+ 文件报格式差异（含未触碰的既有文件）——为 CRLF 检出 + 既有空行差异（`gofmt -d sync.go` 唯一 hunk 在 :97 既有区域），本 plan 两个新文件 gofmt 干净，未引入新偏差。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **61-02（Wave 2）就绪**：复活路径是本 plan 清理的安全配套——清理软删行后，同 DN/cn 对象在 AD 复现时 upsert 会撞不含 deleted_at 的唯一约束（Pitfall 1）。本 plan 未改任何 upsert 行为，`TestSync78_SyncUsers_RestoreSoftDeleted` 等 D-78-05c 文档化断言保持原样绿，翻转动作归 61-02。
- **SHADOW-01 回填**：REQUIREMENTS.md 中 SHADOW-01 = In Progress（清理主体已交付），61-02 完成后标 complete；SHADOW-03 已 Complete。
- **运维告知（FLAGGED ASSUMPTION 传承）**：首轮清理对存量滞留行是一次性批量事件；若 AD 端真的大批量消失（拟删 >50%），两道闸门会拒绝执行、数据继续滞留——这是预期保护行为，部署清单须告知运维避免误报（真环境核对归 61-VALIDATION Manual-Only / 本 SUMMARY D6）。
- **61-03（Wave 3）**：组成员重放事务化，与本 plan 无共享代码面冲突（syncGroupMembers 在 sync.go:615-642，本 plan 仅动 SyncResult 结构体与 syncDataInternal 成功路径段）。

## Self-Check: PASSED

- 文件存在性：`internal/services/addomain/shadow_cleanup_service.go`、`internal/services/addomain/shadow_cleanup_test.go` 均 FOUND
- 提交存在性：`6a08933d`、`75a3c015` 均 merge-base --is-ancestor HEAD 通过
- 验证复跑：`go build ./...` EXIT=0；`go test ./internal/services/addomain/ -run "TestShadowCleanup" -count=1` 12/12 PASS；`go test ./internal/services/addomain/ -count=1` 全包 ok
- 提交计数实测：`git rev-list --count 96d2c3f4..HEAD` == 2（与 frontmatter `commits: 2` 一致，无未提交代码残留）

---
*Phase: 61-shadow-table-integrity*
*Completed: 2026-10-20*
