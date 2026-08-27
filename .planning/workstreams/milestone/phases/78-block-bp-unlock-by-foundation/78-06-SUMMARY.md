---
phase: 78-block-bp-unlock-by-foundation
plan: "06"
subsystem: addomain / coverage
tags: [addomain, computer, ou-group-mapping, group-config, account-pool, sqlite, coverage, d-78-06, d-78-10]

# Dependency graph
requires:
  - 78-05 (setupSync78DB 7 表 fixture / entry78 / insertConfig78 / closeDB)
  - account_pool_test.go (setupTestPool / insertAccount)
provides:
  - computer.go 全链测试模板（Task 1 List + Task 2 syncComputers entry-driven）
  - ou_group_mapping / group_config / config 三个纯 sqlite CRUD service 全覆盖
  - account_pool RecoverExpiredBreakers + StartHotReload + PickAvailable 剩余分支
  - 现行为锁 5 项（D-78-06f/g/h + D-78-06c/d）
affects:
  - phase-78-block05-addomain-coverage (BLOCK-05 第二段)
  - phase-78-block-bp-unlock-78-07 (wave 3 收口 addomain ≥70%)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "复用 78-05 helper（setupSync78DB / entry78 / insertConfig78 / closeDB）— D-78-06e 禁止重定义"
    - "复用 account_pool_test.go helper（setupTestPool / insertAccount）— 同包同函数禁重定义"
    - "raw SQL INSERT 规避 GORM default:true / default:0 quirk（78-05 现行为锁同族）"
    - "现行为锁（lock current behavior）+ 注释行号依据，留 D-78-10 fix 信号"
    - "D-78-06a 仅覆盖 redisPubSub=nil 早退；不引入 miniredis 测 Redis pub/sub"

key-files:
  created:
    - internal/services/addomain/computer_78_06_test.go (16 TestComp78_, 835 lines)
    - internal/services/addomain/ou_group_mapping_78_06_test.go (10 TestOGM78_, 519 lines)
    - internal/services/addomain/group_config_78_06_test.go (5 TestGC78_, 263 lines)
    - internal/services/addomain/config_78_06_test.go (8 TestCfg78_, 416 lines)
    - internal/services/addomain/account_pool_78_06_test.go (9 TestPool78_, 305 lines)
  modified: []

key-decisions:
  - "D-78-06a StartHotReload 仅覆盖 redisPubSub=nil 早退；不引入 miniredis"
  - "D-78-06b TestConnection bind 成功路径归 78-07（in-process LDAP 应答器）；本 plan 两条失败分支即覆盖 config.go ≥70%"
  - "D-78-06c isUniqueConstraintError 用真实 sqlite UNIQUE 约束驱动（不伪造字符串）— 实测可识别（D-78-06c 不命中未触发）"
  - "D-78-06d PickAvailable 多账号用例禁断言具体挑中项（防 random flake）"
  - "D-78-06e 复用 78-05 helper（setupSync78DB / entry78 / insertConfig78）；禁同包重定义"
  - "D-78-06f 现行为锁：ListMappings + GroupName 过滤触发 JOIN sys_ad_group 后 ORDER BY created_at 跨表歧义；测试改断言错误信息；本期不修"
  - "D-78-06g 现行为锁：GetGroupSyncConfig 空字符串覆盖被读侧忽略回退 default；UpdateGroupSyncConfig 不调用 ValidateConfig，非法值落库"
  - "D-78-06h 现行为锁：ConfigService.Update 不修改 admin_username / admin_password；Status 仅在 req.Status 非 nil 时更新"

patterns-established:
  - "Pattern: AD 域 LDAP 集成测试 = sqlite + []*ldap.Entry 字面量 + 同包白盒直调私有方法（78-05 模板的扩展）"
  - "Pattern: D-78-06*f 现行为锁 = 按源码真实输出写测试 + 注释行号依据 + 留给后续 plan 修复信号"

# Metrics
- duration: ~25min across 5 tasks (5 commits)
- completed: 2026-08-27

# Coverage (实测基线 vs 78-06 完成)

| 文件                       | 基线 (78-05 末) | 78-06 完成 | 加权 ≥70% 目标 |
|---------------------------|----------------|-----------|--------------|
| computer.go               | ~6/24 函数 0%  | 96.06% (244/254 stmts) | ✓ 达 |
| ou_group_mapping_service.go | 11 函数全 0% | 88.61% (70/79) | ✓ 达 |
| group_config_service.go   | 7 函数全 0%   | 86.67% (52/60) | ✓ 达 |
| config.go                 | 7 函数全 0%   | 83.02% (44/53) | ✓ 达 |
| account_pool.go           | 85% (基线 ~85% + 2 个 0% 函数) | 82.04% (169/206) | ✗ 未达 90% (见 deviations) |

包级 addomain: 34.6% (基线) → 51.7% (本 plan 末) — 距 78-07 收口的 ≥70% 还差 ~18%。

# Coverage 加权详表

## computer.go: 96.06% (244/254 stmts)
| 函数                      | 行号 | 覆盖率 | 备注 |
|--------------------------|------|-------|------|
| NewComputerService       | 29   | 100.0% | Task 1 |
| List                     | 56   | 90.9%  | Task 1 |
| normalizePagination      | 78   | 100.0% | Task 1 |
| buildComputerQuery       | 91   | 100.0% | Task 1 |
| countComputers           | 108  | 100.0% | Task 1 |
| fetchComputers           | 117  | 88.9%  | Task 1 |
| convertToDetails         | 135  | 100.0% | Task 1 |
| GetByDN                  | 147  | 85.7%  | Task 1 |
| parseComputerDescriptionFor | 167 | 100.0% | 既有 (computer_pure_test.go) |
| parseComputerDescription | 181  | 100.0% | 既有 |
| extractCapacityValue     | 223  | 100.0% | 既有 |
| parseDateTime            | 232  | 100.0% | 既有 |
| determineComputerStatus  | 254  | 100.0% | 既有 |
| buildComputerFromEntry   | 262  | 100.0% | Task 2 |
| updateComputerFields     | 307  | 93.1%  | Task 2 |
| batchCreate              | 346  | 100.0% | Task 2 (250 条 3 批) |
| syncComputers            | 372  | 92.6%  | Task 2 编排 |
| queryExistingComputers   | 478  | 100.0% | Task 2 |
| queryAllComputerNames    | 515  | 100.0% | Task 2 |
| buildComputerMaps        | 526  | 100.0% | Task 2 (后写覆盖) |
| processComputerEntry     | 547  | 100.0% | Task 2 (新建/DN命中/改名 3 分支) |
| batchUpdate              | 580  | 95.2%  | Task 2 (250 条 2 批 ON CONFLICT) |

## ou_group_mapping_service.go: 88.61% (70/79)
| 函数                | 行号 | 覆盖率 | 备注 |
|--------------------|------|-------|------|
| NewOUGroupMappingService | 19 | 100.0% | Task 3 |
| ListMappings       | 61   | 95.2%  | Task 3（D-78-06f: GroupName 过滤歧义已覆盖到 JOIN 路径） |
| CreateMapping      | 114  | 84.6%  | Task 3 |
| GetMapping         | 154  | 83.3%  | Task 3 |
| UpdateMapping      | 168  | 93.3%  | Task 3 |
| DeleteMapping      | 201  | 83.3%  | Task 3 |
| GetMappingsByOU    | 215  | 75.0%  | Task 3 |
| CreateSyncLog      | 231  | 66.7%  | Task 3 |
| UpdateSyncStatus   | 239  | 75.0%  | Task 3 |
| isUniqueConstraintError | 252 | 100.0% | Task 3（D-78-06c: 真 sqlite UNIQUE 错误可识别） |
| containsIgnoreCase | 263  | 100.0% | Task 3 |

## group_config_service.go: 86.67% (52/60)
| 函数                | 行号 | 覆盖率 | 备注 |
|--------------------|------|-------|------|
| NewGroupConfigService | 18 | 100.0% | Task 4 |
| getConfigByKey     | 25   | 100.0% | Task 4 |
| setConfigByKey     | 35   | 87.5%  | Task 4 |
| GetGroupSyncConfig | 57   | 100.0% | Task 4 |
| IsGroupSyncEnabled | 104  | 75.0%  | Task 4 |
| UpdateGroupSyncConfig | 113 | 53.8%  | Task 4（D-78-06g: 不调用 ValidateConfig；非法值落库） |
| ValidateConfig     | 148  | 100.0% | Task 4 (5 条非法输入子用例) |

## config.go: 83.02% (44/53)
| 函数                | 行号 | 覆盖率 | 备注 |
|--------------------|------|-------|------|
| NewConfigService   | 22   | 100.0% | Task 4 |
| GetList            | 42   | 85.7%  | Task 4 (防注入回落) |
| GetByID            | 69   | 85.7%  | Task 4 |
| Create             | 96   | 75.0%  | Task 4 |
| Update             | 140  | 90.0%  | Task 4 (version 自增) |
| Delete             | 177  | 83.3%  | Task 4 (软删) |
| TestConnection     | 189  | 72.7%  | Task 4（D-78-06b: 两条失败分支；bind 成功归 78-07） |

## account_pool.go: 82.04% (169/206) — 未达 ≥90%
| 函数                | 行号 | 覆盖率 | 备注 |
|--------------------|------|-------|------|
| NewAccountPool     | 127  | 100.0% | 既有 |
| ListAvailable      | 139  | 100.0% | 既有 + Task 5 (MultiConfig 缓存失效观察) |
| PickAvailable      | 186  | 100.0% | Task 5 (4 子用例：空/单/多/DBError) |
| ListAll            | 202  | 78.6%  | 既有 |
| CountByStatus      | 234  | 85.7%  | 既有 |
| PickFirstAvailable | 264  | 85.7%  | 既有 |
| Create             | 281  | 80.0%  | 既有 |
| Update             | 292  | 75.0%  | 既有 |
| Delete             | 301  | 75.0%  | 既有 |
| MarkSuccess        | 317  | 92.3%  | 既有 |
| sanitizeFailureReason | 349 | 100.0% | 既有 |
| MarkFailure        | 371  | 88.0%  | 既有 |
| ManualUnlock       | 420  | 92.3%  | 既有 |
| SetEnabled         | 463  | 86.7%  | 既有 |
| RecoverExpiredBreakers | 495 | 75.0%  | Task 5 (5 用例：Basic/None/MultiConfig/DBError/NoPubSub) |
| InvalidateCache    | 542  | 100.0% | Task 5 (warm → raw INSERT → 命中 → Invalidate → 落 DB) |
| StartHotReload     | 553  | 11.8%  | Task 5（D-78-06a: 仅 redisPubSub=nil 早退覆盖） |

# 达标核对 (per plan §success_criteria)

| 验收项 | 计划值 | 实测值 | 通过 |
|--------|-------|-------|------|
| 5 个测试文件存在 | 5 | 5 | ✓ |
| TestComp78_ ≥16 | 16 | 16 | ✓ |
| TestOGM78_ ≥9 | 10 | 10 | ✓ |
| TestGC78_ ≥5 | 5 | 5 | ✓ |
| TestCfg78_ ≥8 | 8 | 8 | ✓ |
| TestPool78_ ≥8 | 9 | 9 | ✓ |
| computer.go 加权 ≥70% | 70% | 96.06% | ✓ |
| ou_group_mapping_service.go 加权 ≥70% | 70% | 88.61% | ✓ |
| group_config_service.go 加权 ≥70% | 70% | 86.67% | ✓ |
| config.go 加权 ≥70% | 70% | 83.02% | ✓ |
| **account_pool.go 加权 ≥90%** | 90% | **82.04%** | **✗** |
| 零生产 .go 改动 | 0 | 0 | ✓ |
| `go build ./...` exit 0 | ✓ | ✓ | ✓ |
| `go test -count=1 ./internal/services/addomain/` exit 0 | ✓ | ✓ | ✓ |
| `go test -race -count=1 ./internal/services/addomain/` | ✓ | (本机 cgo.exe 故障，CI 环境可跑) | — |
| 零 LDAP 网络 / 零 Redis 依赖 | ✓ | ✓ | ✓ |

# Deviations from Plan

### 未达 90% 阈值 (account_pool.go 加权 82.04%)

**Finding during Task 5:**
account_pool.go 加权未达 ≥90% 计划阈值。差距主要在 StartHotReload (11.8% × 17 stmts ≈ 15 未覆盖 stmts) 的 Redis pub/sub 订阅路径。

**Root cause:** 计划 §success_criteria 设定 account_pool.go ≥90%，但 D-78-06a 显式禁止 miniredis 测 Redis pub/sub 订阅路径（78-04 已用尽 miniredis 预算；pubsub 订阅是跨进程语义单测价值低；78-RESEARCH §3 标 LOW/"已覆盖主体"）。

**Fix applied:** 不修生产代码，按 D-78-10 + D-78-06a 双约束诚实标注。StartHotReload 11.8% 是当前可达上限。

**Files modified:** 无（仅记录偏差）。

**Commit:** 1b5ff4d

**Recommendation for 78-07 lead:**
- 选项 A：放宽本 plan success_criteria 到 ≥80%（接受 D-78-06a 边界）
- 选项 B：78-07 引入 miniredis 测 StartHotReload Redis 订阅路径，可推到 ≥90%
- 选项 C：将 StartHotReload 整体重构成 nil-safe 默认 + 可选订阅路径，并在订阅失败时 panic，便于测试

### Auto-fixed / Current-behavior locks

**1. [D-78-06f] ListMappings + GroupName JOIN 跨表歧义**
- **Found during:** Task 3
- **Issue:** `ListMappings` 收到 `GroupName` 过滤时加 `JOIN sys_ad_group ON ...`，最终 `ORDER BY created_at DESC` 跨表歧义（"ambiguous column name: created_at"），sqlite/PG 同样会报。
- **Fix:** 测试改断言错误信息（含"ambiguous column name: created_at"），实证覆盖到 JOIN 路径。本期不修生产 SQL。
- **Files modified:** 无
- **Commit:** c254678
- **Recommendation:** 后续 plan 应将 `ORDER BY created_at DESC` 改为 `ORDER BY sys_ou_group_mapping.created_at DESC`。

**2. [D-78-06g] UpdateGroupSyncConfig 不调用 ValidateConfig + 空字符串覆盖被读侧忽略**
- **Found during:** Task 4
- **Issue:** `UpdateGroupSyncConfig` 直接写 DB 不调 `ValidateConfig`；`GetGroupSyncConfig` 把空字符串视为"无覆盖"回退 default。
- **Fix:** 测试断言现行为 + 注释行号依据。
- **Files modified:** 无
- **Commit:** e4c39f8

**3. [D-78-06h] ConfigService.Update 不改 admin 凭据字段**
- **Found during:** Task 4
- **Issue:** `Config.go:147-159` updates map 未含 `admin_username` / `admin_password`；不修改 Status 除非 `req.Status` 非 nil。
- **Fix:** 测试断言"密码字段保留原值"语义需用 AdminPassword 旧值假说——此处 service 不动该字段，跳过断言。
- **Files modified:** 无
- **Commit:** e4c39f8

**4. [D-78-06c] isUniqueConstraintError 真实 sqlite UNIQUE 错误可识别**
- **Found during:** Task 3
- **Issue:** 计划要求用真实 sqlite driver error 触发（不伪造字符串），实证是否仅适配 PG。
- **Fix:** 实测 glebarez sqlite 的 UNIQUE 错误文案含"UNIQUE constraint failed"，case-insensitive 匹配 "unique constraint" 字符串 → **isUniqueConstraintError 可识别**，无需按 D-78-10 记待裁决。
- **Files modified:** 无
- **Commit:** c254678

**5. [D-78-06d] PickAvailable 多账号用例不断言具体挑中项**
- **Found during:** Task 5
- **Issue:** 防 random pick flake（local-vs-ci-test-divergence 教训）。
- **Fix:** 多账号用例断言"返回值 ∈ 可用账号集合"而非具体 ID。
- **Files modified:** 无
- **Commit:** 1b5ff4d

# Threat Flags

无新增威胁面。所有用例：
- 零 LDAP 网络（仅 TestCfg78_TestConnection_AllBindFail 指向 127.0.0.1 本地已关闭端口）
- 零 Redis（StartHotReload / RecoverExpiredBreakers 仅 redisPubSub=nil 形态）
- 零生产 .go 改动（git diff --stat 仅 test 文件）
- 全部 dummy CN / IP / 域名 / 密码（"encrypted_pwd" / "127.0.0.1" / "DC=example,DC=com" / "svc-N"）

# Known Stubs / Not Covered (per D-78-06 系列裁决)

| 项 | 原因 | 决策 |
|---|------|------|
| StartHotReload pubsub 订阅路径 (~15 stmts) | D-78-06a: 禁 miniredis；订阅是跨进程语义 | 文档化边界 |
| TestConnection bind 成功路径 (~5 stmts) | D-78-06b: 需 in-process LDAP 应答器 | 归 78-07 |
| ListMappings GroupName 过滤 happy path | D-78-06f: ORDER BY 跨表歧义 | 待后续 plan 修复 |
| UpdateGroupSyncConfig 调用 ValidateConfig | D-78-06g: 当前实现不校验 | 待后续 plan 加固 |
| ConfigService.Update 改 admin 凭据 | D-78-06h: 当前实现不动 | 待后续 plan 加固 |

# Auth Gates
None.

# Self-Check
- [x] 5 个测试文件存在（computer / ou_group_mapping / group_config / config / account_pool）
- [x] TestComp78_ 16 个全绿
- [x] TestOGM78_ 10 个全绿
- [x] TestGC78_ 5 个全绿（含 5 子用例）
- [x] TestCfg78_ 8 个全绿
- [x] TestPool78_ 9 个全绿
- [x] `go build ./...` exit 0
- [x] `go test -count=1 ./internal/services/addomain/` exit 0 (51.7% 包级)
- [x] 4/5 目标文件加权 ≥70% (computer 96% / ou_group_mapping 89% / group_config 87% / config 83%)
- [ ] account_pool.go 加权 ≥90% — **未达 82.04%**，需 78-07 决定是否引入 miniredis（D-78-06a 冲突）
- [x] 零生产 .go 改动（git diff --stat 仅 test 文件）
- [x] D-78-06e：复用 78-05 helper 而非重定义

# Test File Layout

| 文件 | 行数 | 测试函数数 | 覆盖函数（行号） |
|------|------|----------|----------------|
| computer_78_06_test.go | 835 | 16 | computer.go:29-623 (List 链 + syncComputers 全链) |
| ou_group_mapping_78_06_test.go | 519 | 10 | ou_group_mapping_service.go:19-267 (11 函数全覆盖) |
| group_config_78_06_test.go | 263 | 5 (+ 5 子) | group_config_service.go:18-164 (7 函数全覆盖) |
| config_78_06_test.go | 416 | 8 | config.go:22-208 (7 函数全覆盖, TestConnection 两条失败分支) |
| account_pool_78_06_test.go | 305 | 9 | account_pool.go:495-582 (RecoverExpiredBreakers + StartHotReload + PickAvailable + InvalidateCache) |

# Commits
- fb2620f  test(78-06): add computer.go List 查询链 + GetByDN 测试 (Task 1)
- 153e9c3  test(78-06): add syncComputers entry-driven 全链测试 (Task 2)
- c254678  test(78-06): add ou_group_mapping_service.go 11 函数测试 (Task 3)
- e4c39f8  test(78-06): add group_config_service.go + config.go 测试 (Task 4)
- 1b5ff4d  test(78-06): add account_pool RecoverExpiredBreakers + StartHotReload + PickAvailable 剩余分支 (Task 5)

# Phase 78 Plan 06: addomain 第二段 — computer + 三个 CRUD + account_pool 剩余

## 整体结论

**PARTIAL PASS**: 5 个测试文件落地，48 个新增测试函数全绿；4/5 目标文件加权 ≥70% (computer 96% / ou_group_mapping 89% / group_config 87% / config 83%)。**account_pool.go 加权 82.04% 未达计划 ≥90%** —— 差距源于 D-78-06a 禁止 miniredis 测 StartHotReload pubsub 路径的硬约束。该冲突需 78-07 决策。

**BLOCK-05 第二段完成**（除 account_pool 90% 阈值冲突）。零生产 .go 改动（D-78-07 满足），5 阶段 5 commit。48 Test*78_ 函数全绿。

## D-78-06 现行为锁 5 项（已记入 deviations）

1. **D-78-06c** isUniqueConstraintError 实测可识别 sqlite UNIQUE 文案 ✓ 不触发待裁决
2. **D-78-06f** ListMappings + GroupName 触发 ORDER BY 跨表歧义（待后续 plan 修复）
3. **D-78-06g** UpdateGroupSyncConfig 不调用 ValidateConfig（待后续 plan 加固）
4. **D-78-06h** ConfigService.Update 不改 admin 凭据字段（待后续 plan 加固）
5. **D-78-06a** StartHotReload 仅 redisPubSub=nil 早退覆盖（account_pool 加权未达 90% 的根因）

## 78-07 待办

- 决策 account_pool 90% 阈值冲突（miniredis 引入 vs 接受 82% 边界）
- wave 3 收口 addomain ≥70%（当前 51.7%，差 ~18%）
- ListMappings GroupName ORDER BY 跨表歧义修复（可选）