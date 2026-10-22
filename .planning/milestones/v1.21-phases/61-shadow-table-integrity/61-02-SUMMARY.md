---
phase: 61-shadow-table-integrity
plan: 02
subsystem: backend-ad-sync
tags: [gorm, soft-delete, upsert, on-conflict, ad-sync, sqlite-testing, tdd]

# Dependency graph
requires:
  - phase: 61-shadow-table-integrity (61-01)
    provides: 影子表清理 writer（cleanupAfterFullSync + 三类双闸门软删）——复活是其安全前提，软删行由此产生
  - phase: 78 (78-05/78-06)
    provides: sqlite fixture 与 entry78 helper 体系（setupSync78DB/setupComp78DB/newSyncSvc78/insertConfig78）
provides:
  - 四类 upsert 路径（OU/Group/User/Computer）的软删行复活：existing 查询 Unscoped 化（软删行进入 update 分支，裸 Create 不再收到复活行）+ OnConflict AssignmentColumns 补 deleted_at（SET deleted_at=NULL 即复活机制本体）
  - 六处 existing 加载点命中行 DeletedAt 清零（plan 未言明的必要一环：Unscoped Find 会把旧时间戳读进 struct，不清零则冲突更新把旧时间戳写回、行依旧不可见）
  - sys_ad_group 存量潜伏缺陷（handleDeletedGroups 软删 → 同 DN 组复现撞裸 Create）一并消除：全量 syncGroups 与增量 syncGroupEntries 两条路径均复活；SyncSingleGroup 仅补 AssignmentColumns
  - 5 条 TestSync_Resurrect*（OU/Group 同 DN、Computer 同 DN/同名换 DN、Group 增量路径）先 RED（UNIQUE 2067）后 GREEN；D-78-05c 两个既有用例翻转
affects: [61-03 组成员事务化, 61-VALIDATION 生产约束核对, AD 影子表运维]

# Actuals (#2632)
actuals:
  tokens: 7100        # chars/4 over realized diff（28552 chars / 276 insertions + 35 deletions）
  tasks: 2
  commits: 4          # measured: git rev-list --count 7c6ba489..3bcd1c84
plan_head_before: 7c6ba489b13c5e9646d81b2c0084a40a5e728ec0
plan_head_after: 3bcd1c840e3cd8f4bcc87078a9a884fb6ebe2e15

# Tech tracking
tech-stack:
  added: []           # 零新依赖
  patterns:
    - "upsert 复活三件套：existing 查询 Unscoped → 命中行 DeletedAt 清零（struct 零值 → INSERT deleted_at=NULL）→ OnConflict AssignmentColumns 含 deleted_at（SET deleted_at=EXCLUDED=NULL）"
    - "Unscoped 化必须同步检查查询链里的显式 deleted_at IS NULL 谓词（queryAllComputerNames 同款）——Unscoped 只解除 GORM 默认过滤，显式谓词会继续过滤软删行使复活失效"

key-files:
  created: []
  modified:
    - internal/services/addomain/sync.go
    - internal/services/addomain/computer.go
    - internal/services/addomain/group_sync_service.go
    - internal/services/addomain/shadow_cleanup_test.go
    - internal/services/addomain/sync_78_05_test.go
    - internal/services/addomain/computer_78_06_test.go

key-decisions:
  - "existing 加载点命中行 DeletedAt 清零（plan 机制注释的隐含前提显式化）：Unscoped().Find 把软删行旧时间戳读进 struct，若不清零 ON CONFLICT SET deleted_at=EXCLUDED 会写回旧时间戳、复活失效；活行 SET NULL 与库中现值一致无副作用"
  - "queryAllComputerNames 显式 deleted_at IS NULL 谓词按 plan 删除，既有 TestComp78_QueryExistingComputers_And_AllNames 断言同步改写（软删行纳入 name-map，4 行）——plan 行为变化的计划内披露，非断言弱化"
  - "D-78-05c『软删行 + 同 DN → UNIQUE 冲突』文档化正式推翻：两用例翻转 + 文件头注释记翻转依据（SHADOW-01 清理上线后软删行复现必须自愈）"
  - "SHADOW-01 按 shared-ID 门在本 plan 标 complete：两个声明者（61-01/61-02）均已完成"

patterns-established:
  - "复活四件套断言：返回 nil / 默认作用域可见 / 字段已更新 / Unscoped 计数不变——后续任何纳入软删行复活的 upsert 路径照此锁定"
  - "Unscoped 化审计点：查询链中显式软删谓词与 Unscoped 同行出现即假改造（本 plan 在 queryAllComputerNames 防住一处）"

requirements-completed: [SHADOW-01]  # verbatim 自 plan frontmatter

coverage:
  - id: D1
    description: OU 复活：软删 OU 同 DN 在 AD 复现 → syncOUs 返回 nil、行恢复可见、OUName 已更新、Unscoped 计数不变
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestSync_ResurrectOU_SameDN"
        status: pass
    human_judgment: false
  - id: D2
    description: Group 全量路径复活：软删组同 DN 复现 → syncGroups 返回 nil、行恢复可见（裸 Create 不再收到复活行，Q3 存量缺陷消除）
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestSync_ResurrectGroup_SameDN"
        status: pass
    human_judgment: false
  - id: D3
    description: User 复活 + D-78-05c 翻转：软删用户同 DN entry → syncUsers 返回 nil、默认作用域可见、DisplayName/Email/UAC 推导已更新、Unscoped 计数 1；getExistingOUs 现命中软删行
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/sync_78_05_test.go#TestSync78_SyncUsers_RestoreSoftDeleted"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/sync_78_05_test.go#TestSync78_GetExistingOUs"
        status: pass
    human_judgment: false
  - id: D4
    description: Computer 复活两形态：软删行同 DN 复现自愈；软删行 DN-A/cn=X + LDAP DN-B/cn=X 经 name-map 命中 update 分支复活且 distinguished_name 更新为新 DN
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestSync_ResurrectComputer_SameDN"
        status: pass
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestSync_ResurrectComputer_SameNameNewDN"
        status: pass
    human_judgment: false
  - id: D5
    description: Group 增量路径复活（syncGroupEntries）+ SyncSingleGroup 冲突更新复活就位（AssignmentColumns 补 deleted_at）
    requirement: SHADOW-01
    verification:
      - kind: unit
        ref: "internal/services/addomain/shadow_cleanup_test.go#TestSync_ResurrectGroup_IncrementalPath"
        status: pass
      - kind: other
        ref: "grep -c '\"deleted_at\"' internal/services/addomain/group_sync_service.go == 2（SyncSingleGroup + syncGroupEntries 两处 AssignmentColumns 源断言）"
        status: pass
    human_judgment: false
  - id: D6
    description: 禁地守卫：零 DDL（internal/core/db/ 零改动、无 partial unique index 路线）、零 .sql 变更（无 PG TRIGGER 路线）、注释零 Phase 62 边界引用；读路径保持默认作用域
    requirement: SHADOW-01
    verification:
      - kind: other
        ref: "git status internal/core/db/ 无本 plan 改动；grep -c 'user_ou_service|GetUserDeptByADOU|BatchMoveUsersToNewOU' 六个触碰文件全 0"
        status: pass
    human_judgment: false
  - id: D7
    description: 生产 PG 中 sys_ad_* 表实际唯一约束现状核对（FLAGGED ASSUMPTION A1：约束缺失时故障形态为重复行而非约束冲突，复活路径设计不变）
    requirement: SHADOW-01
    verification: []
    human_judgment: true
    rationale: "本机无 PG 实例，sqlite 单测无法覆盖生产约束形态；启动冒烟后人工核对（\\d sys_ad_user 等）列入 61-VALIDATION Manual-Only（61-01 D6 同款传承）"

# Metrics
duration: 33 min
completed: 2026-10-20
status: complete
---

# Phase 61 Plan 02: 影子表复活路径（SHADOW-01 配套 + Q3 组缺陷一并修） Summary

**四类 upsert（OU/Group/User/Computer）软删行复活：existing 查询 Unscoped 化 + OnConflict 补 deleted_at + 命中行 DeletedAt 清零，5 条 TestSync_Resurrect 先 RED（UNIQUE 2067）后 GREEN，D-78-05c 两用例翻转，全包零回归。**

## Performance

- **Duration:** 33 min
- **Started:** 2026-10-20T17:00:58Z
- **Completed:** 2026-10-20T17:33:52Z
- **Tasks:** 2/2（均 tdd="true"，各产 test → feat 两个原子提交）
- **Files modified:** 6（3 生产 + 3 测试）

## Accomplishments

- SHADOW-01 复活路径落地：`sync.go` 三处 existing 查询（getExistingOUs / syncGroups / syncUsers 批查）Unscoped 化，软删行进入 update 分支，裸 Create（batchCreateOUs / createGroupsInBatches / usersToCreate）不再收到复活行；三处 OnConflict AssignmentColumns 补 `deleted_at`——复活行 INSERT 值 deleted_at=NULL → 冲突更新 SET deleted_at=NULL → 行自愈，四类路径不再整轮同步失败或行静默不可见（Pitfall 1 关闭）
- 复活机制补上 plan 未言明的必要一环：六处 existing 加载点命中行 `DeletedAt` 清零——`Unscoped().Find` 会把软删行旧时间戳读进 struct，不清零则 `SET deleted_at = EXCLUDED.deleted_at` 写回旧时间戳、行依旧不可见（活行 SET NULL 与库中现值一致，无副作用）
- `sys_ad_group` 存量潜伏缺陷（Q3）一并消除：`group_sync_service.go` syncGroupEntries existing 查询 Unscoped 化 + 两处 OnConflict（SyncSingleGroup / syncGroupEntries）补 deleted_at——handleDeletedGroups 今天就在软删组，同 DN 组复现不再撞 UNIQUE(ad_config_id, group_dn)
- `computer.go` 复活化：queryExistingComputers Unscoped 化；queryAllComputerNames Unscoped 化并**移除**显式 `AND deleted_at IS NULL` 谓词（Unscoped 只解除 GORM 默认过滤，保留谓词复活失效）；batchUpdate OnConflict 补 deleted_at——OU 搬迁场景（软删行 DN-A/cn=X + LDAP 返回 DN-B/cn=X）经 name-map 命中复活且 DN 更新
- D-78-05c 两用例翻转：TestSync78_GetExistingOUs 改断言软删行被命中（2 条含 OU=B）；TestSync78_SyncUsers_RestoreSoftDeleted 改 require.NoError + 默认作用域可见 + DisplayName/Email/UAC 推导已更新 + Unscoped 计数仍 1；文件头注释区记翻转依据
- 全包零回归：`go test ./internal/services/addomain/ -count=1` ok（含 61-01 的 12 条 TestShadowCleanup、既有 F-01/F-02 防冲突、threshold 四用例、SyncSingleGroup 主链）

## Task Commits

每个 task 按 TDD 契约产 test → feat 两个原子提交：

1. **Task 1: sync.go 三类 upsert 复活化 + D-78-05c 翻转** - `c441c802` (test, RED) + `af162852` (feat, GREEN)
2. **Task 2: computer.go + group_sync_service.go 复活化 + 全包回归** - `bdc8c452` (test, RED) + `3bcd1c84` (feat, GREEN)

**Plan metadata:** 见下方 docs commit（SUMMARY + STATE + ROADMAP + REQUIREMENTS 同 commit）

### RED → GREEN 证据

- Task 1 RED：`go test ./internal/services/addomain/ -run "TestSync_Resurrect" -count=1 -v` EXIT=1，TestSync_ResurrectOU_SameDN / TestSync_ResurrectGroup_SameDN 均以目标断言失败——`批量创建OU/用户组失败: constraint failed: UNIQUE constraint failed: sys_ad_ou.ad_config_id, sys_ad_ou.ou_dn (2067)`（batchCreateOUs / createGroupsInBatches 裸 Create 撞软删行），非编译/fixture 故障，VALID RED
- Task 2 RED：`go test -run "TestSync_ResurrectComputer|TestSync_ResurrectGroup_IncrementalPath"` EXIT=1，三条均 UNIQUE (2067)——computer `批量创建失败: ... sys_ad_computer.computer_name (2067)`（batchCreate :361）、group `批量创建用户组失败: ... sys_ad_group.group_dn (2067)`（裸 Create :303），VALID RED
- GREEN：Task 1 后 34 个 TestSync78/TestSync_Resurrect 全 PASS；Task 2 后全包 ok。gsd_run CLI 本环境不可用（58-01/61-01 既有记录），RED 证据以原生命令输出留档，未走 `check tdd-red-evidence` 分类器

## Files Created/Modified

- `internal/services/addomain/sync.go` — getExistingOUs / syncGroups existing / syncUsers 批查 Unscoped 化 + 命中行 DeletedAt 清零；batchUpdateOUs / updateGroupsInBatches / syncUsers 批量更新三处 OnConflict AssignmentColumns 补 deleted_at（含机制注释）
- `internal/services/addomain/computer.go` — queryExistingComputers / queryAllComputerNames Unscoped 化（后者移除显式软删谓词）+ 清零；batchUpdate OnConflict 补 deleted_at
- `internal/services/addomain/group_sync_service.go` — syncGroupEntries existing Unscoped 化 + 清零；SyncSingleGroup / syncGroupEntries 两处 OnConflict 补 deleted_at
- `internal/services/addomain/shadow_cleanup_test.go` — 新增 5 条 TestSync_Resurrect*（OU/Group 同 DN、Computer 同 DN/同名换 DN、Group 增量）+ time import
- `internal/services/addomain/sync_78_05_test.go` — TestSync78_GetExistingOUs / TestSync78_SyncUsers_RestoreSoftDeleted 按复活语义翻转 + 文件头 Phase 61 注记
- `internal/services/addomain/computer_78_06_test.go` — TestComp78_QueryExistingComputers_And_AllNames 断言改写（软删行纳入 name-map，4 行）

## Decisions Made

- **existing 加载点命中行 DeletedAt 清零（plan 未言明的必要环节）**：plan 机制注释假设「复活行的 struct DeletedAt 为零值 → INSERT 值 NULL」，但 Unscoped Find 把旧时间戳读进 struct——不清零则冲突更新写回旧时间戳、复活失效。实现 = 六处 Find 后一行循环清零；SyncSingleGroup 无 existing 预查、新建 struct 天然零值，仅补 AssignmentColumns 即可。行为由 5 条 TestSync_Resurrect 锁定。
- **queryAllComputerNames 显式谓词删除属 plan 明文指令**（"漏删谓词是最可能的假改造"）；既有用例断言随之改写并在 Deviations 披露。
- **SHADOW-01 标 complete**：REQUIREMENTS.md 回填（61-01 清理主体 + 61-02 复活路径两段拼图齐备）；真环境约束核对（FLAGGED ASSUMPTION A1）归 61-VALIDATION Manual-Only。
- **提交落在 main 分支**（项目 `branching_strategy: none` + orchestrator `<sequential_execution>` 明示主工作树，先例 Phase 57-61 同款）；gsd_run CLI 不可用，per-task 提交用原生 git、STATE/ROADMAP/REQUIREMENTS 手工编辑。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] plan 复活机制在 update 分支缺少 DeletedAt 清零，按字面实现复活会失效**
- **Found during:** Task 1（实现前源码推演）
- **Issue:** plan 机制为「existing Unscoped → 命中行进 update 分支 + OnConflict AssignmentColumns 补 deleted_at → INSERT 值 deleted_at=NULL（struct 零值）→ SET deleted_at=NULL 复活」。但 `Unscoped().Find` 会把软删行的旧时间戳读进 struct（`gorm.DeletedAt.Valid=true`），INSERT 值携带旧时间戳 → `SET deleted_at = EXCLUDED.deleted_at` 写回旧值 → 行依旧不可见，四件套断言（默认作用域可见）不可能通过
- **Fix:** 六处 existing 加载点（getExistingOUs / syncGroups / syncUsers 批查 / queryExistingComputers / queryAllComputerNames / syncGroupEntries）在 Find 后将命中行 `DeletedAt` 清零为 struct 零值，使 plan 注释的前提（struct 零值 → INSERT NULL）实际成立；活行 SET NULL 与库中现值一致无副作用
- **Files modified:** internal/services/addomain/sync.go, computer.go, group_sync_service.go
- **Verification:** 5 条 TestSync_Resurrect* 全部经 RED（UNIQUE 2067）→ GREEN（默认作用域可见 + Unscoped 计数不变），plan 行为验收原样成立
- **Committed in:** `af162852` / `3bcd1c84`

**2. [Rule 1 - Bug] 既有 TestComp78_QueryExistingComputers_And_AllNames 断言与复活语义冲突（计划内行为变化）**
- **Found during:** Task 2（GREEN 全包回归）
- **Issue:** 该用例断言 queryAllComputerNames 过滤软删行（3 行、PC-SOFT 不在结果中）——plan action 2 明文删除该显式谓词后必然红
- **Fix:** 断言按复活语义改写（4 行、PC-SOFT 纳入 name-map），注释说明 Unscoped 化语义与「同名行至多一条、name-map 无遮蔽风险」依据；queryExistingComputers 部分断言不动
- **Files modified:** internal/services/addomain/computer_78_06_test.go
- **Verification:** 全包 ok；plan action 6「有用例因 Unscoped 行为变化需更新断言，逐个披露」的授权范围内
- **Committed in:** `3bcd1c84`

---

**Total deviations:** 2 auto-fixed（Rule 1 × 2，其一为 plan 授权范围内的既有断言改写）
**Impact on plan:** 偏差 1 是复活机制的正确性补全（不改验收面，行为验收按 plan 原样达成）；偏差 2 是 plan 明文行为的连带断言更新。无范围扩张。

## Issues Encountered

- `go test ./... -count=1` 全套件在 `internal/device` 报 `TestSNMPClient_WaitForReady_Timeout` FAIL（0.50s）——该包与 addomain 零依赖，单跑两轮均 PASS（0.20s），为全套件并发压力下的既有 timing flake，按 58-07 既有不稳定项口径如实披露、不在本 plan 修复（scope boundary）。
- 工作树存在大量与本 plan 无关的既有未提交状态（`.planning/phases/11-mac/` 等历史目录删除、`internal/services/system/asset_columns_schema.json` 修改、一条既有 stash）——均未触碰、未纳入本 plan 提交。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **61-03（Wave 3）就绪**：组成员重放事务化（SHADOW-02），与本 plan 无共享代码面冲突（syncGroupMembers 在 sync.go:631-657 未触碰，本 plan 只动 existing 查询与 OnConflict 列表）。注意 61-03 的事务化会进一步保证复活行成员重放的原子性。
- **SHADOW-01 已标 complete**（REQUIREMENTS.md 回填）；SHADOW-02 待 61-03。
- **部署耦合约束传承**：61-01 + 61-02 + 61-03 三个 plan 必须整体部署（清理已激活而复活未就绪的窗口在本 phase 内已关闭，但仍需与 61-03 同批）。
- **UAT 人工项（D7 传承 61-01 D6）**：生产 PG 启动冒烟后人工核对 `sys_ad_*` 表实际唯一约束（FLAGGED ASSUMPTION A1：若生产无唯一约束，复活问题本会以「重复行」而非「约束冲突」形态存在，复活路径设计不变）。
- **运维语义变化提示**：软删行复活意味着「清理后同 DN 复现的行」会自动恢复可见且字段更新——这是设计目标（误删自愈），审计时可经 Unscoped 查询追溯（T-61-05 accept 处置）。

## Self-Check: PASSED

- 文件存在性：3 生产文件 + 3 测试文件 + SUMMARY.md 均 FOUND
- 提交存在性：`c441c802` / `af162852` / `bdc8c452` / `3bcd1c84` / `b8b70a52`（docs）均 merge-base --is-ancestor HEAD 通过
- 提交计数实测：`git rev-list --count 7c6ba489..HEAD` == 5（4 任务提交 + 1 docs 提交，frontmatter `commits: 4` 记任务提交口径）
- 验证复跑：`go build ./...` EXIT=0；`go test ./internal/services/addomain/ -count=1` 全包 ok（19 条定向 PASS 含 5 条 TestSync_Resurrect + 2 条翻转 + 12 条 TestShadowCleanup）

---
*Phase: 61-shadow-table-integrity*
*Completed: 2026-10-20*
