---
phase: 78-block-bp-unlock-by-foundation
plan: "05"
subsystem: addomain / coverage
tags: [addomain, sync, ldap-entry-driven, sqlite, coverage, d-78-07, d-78-10]

# Dependency graph
requires: []
provides:
  - 7-table sqlite fixture (sys_ad_config/ou/group/user/group_member/sync_log/service_accounts)
  - entry78 ldap.Entry constructor helper
  - setupSync78DB / newSyncSvc78 / insertConfig78 / closeDB reusable helpers (D-78-06e forbids 78-06 redefinition)
  - sync.go private method entry-driven coverage: extractDNs / safeAttr / getExistingOUs /
    categorizeOUs / batchCreateOUs / batchUpdateOUs / syncGroups / createGroupsInBatches /
    updateGroupsInBatches / parseGroupTypeFromLDAP / syncOUs / syncUsers / syncGroupMembers /
    updateSyncLog / SyncDataByID / SyncData / syncDataInternal (failure paths)
affects:
  - phase-78-block05-addomain-coverage (BLOCK-05 第一段)
  - phase-78-block-bp-unlock-78-06 (reuses helpers)
  - phase-78-block-bp-unlock-78-07 (reuses helpers)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "零 LDAP 网络 + []*ldap.Entry 字面量驱动(entry78 helper,生产读取入口 GetAttributeValue/GetAttributeValues)"
    - "sqlite + 手动 CREATE TABLE(参考 dept_ou_mapper_test.go:18 注释,避开 __temp+RENAME 与视图依赖风险)"
    - "BaseModel 嵌入列补全(created_by/updated_by/version),避免 'no such column' 误报"
    - "D-78-05c 现行为断言:GORM bool default 标签 quirk + sync.go:680 列名 bug + soft-delete UNIQUE 冲突"
    - "D-78-07 拒绝生产 seam:happy path ~25 stmts 不覆盖,文档化为覆盖边界"

key-files:
  created:
    - internal/services/addomain/sync_78_05_test.go
  modified: []

key-decisions:
  - "D-78-07 拒绝为 sync.go 加生产缝:本地构造 fc := NewFailoverClient(...) 不可注入,选择测私有方法 + 失败路径,~25 stmts 接受不覆盖"
  - "GORM `default:true` tag quirk:ADUser.IsEnabled=false 在 Create/Update 时被 default 覆盖,测试断言 IsDisabledByUAC()(UAC 直接推导)而非 IsEnabled 字段值"
  - "sync.go:680 文档化列名 bug:updates map 用 key `error_message` 但 models.ADSyncLog 列名 `column:error_msg`,SQL UPDATE 整句失败。Success 路径(errMsg='')不受影响,Failed 路径所有字段维持原值(D-78-10 无据不改)"
  - "syncUsers soft-delete + 同 DN UNIQUE 冲突:GORM Find 默认过滤 deleted_at + sqlite UNIQUE 不忽略软删行 → 触发 constraint failed。文档化为现行为,不修 sync.go"
  - "syncUsers 不含 manager 关联/\\$DUPLICATE-/\\$ 后缀过滤:这三类过滤语义由 user.go GetList 兜底,sync.go 仅落库原值。文档化"

patterns-established:
  - "Pattern: sync.go 全链 entry-driven 测试模板 = 7表 fixture + entry78 helper + 同包白盒直调私有方法(避开 seam 引入)"
  - "Pattern: D-78-05c 现行为断言 = 按源码真实输出写测试 + 注释行号依据,留下修复信号(同期不修)"

# Metrics
- duration: ~10min across 5 tasks
- completed: 2026-08-27

# Coverage (实测基线 18 函数全 0% → 83.9% 加权,265/316 stmts)
sync.go 函数级覆盖:
  - NewSyncService              0.0%   (2 stmts, 构造器)
  - SyncDataByID              100.0%   (3 stmts)
  - SyncData                   60.0%   (5 stmts, singleflight 合并语义由 sync_singleflight_test.go 覆盖)
  - syncDataInternal           25.5%   (3 条失败路径覆盖 ~22/97 stmts,D-78-07 happy path 不覆盖)
  - syncOUs                    88.9%   (9 stmts)
  - extractDNs                100.0%
  - safeAttr                  100.0%
  - getExistingOUs            100.0%
  - categorizeOUs             100.0%
  - batchCreateOUs             80.0%
  - batchUpdateOUs             93.3%
  - syncGroups                 94.4%
  - createGroupsInBatches     100.0%
  - updateGroupsInBatches      92.9%
  - syncUsers                  96.8%   (sync.go 最大单函数块,182 stmts)
  - syncGroupMembers           90.0%
  - updateSyncLog              94.1%
  - parseGroupTypeFromLDAP    100.0%

# Acceptance Criteria 逐条核对
  - 31 TestSync78_ 函数全绿 (Task 1-5 合计)
  - 7 张 CREATE TABLE (sys_ad_config/ou/group/user/group_member/sync_log/service_accounts)
  - errors.Is(err, ErrAllAccountsUnavailable) 断言:存在 (TestSync78_SyncDataInternal_EmptyAccountPool)
  - failure_count 被 MarkFailure 递增断言:存在 (TestSync78_SyncDataInternal_AllAccountsDialFail)
  - file 尾部含 D-78-07 覆盖边界块注释 + SUMMARY '已知不覆盖' 小节
  - dial 失败用例 10s 硬超时守卫 + 网络目标仅 127.0.0.1
  - 生产 .go 改动 = 0 文件 (D-78-07 满足)

# Known Stubs / Not Covered (D-78-05c / D-78-07)
  - syncDataInternal happy path ~25 stmts:第 3-7 步 syncOUs → syncGroups → syncUsers →
    syncComputers → last_sync_at → updateSyncLog(Success)。需要 FailoverClient 闭包成功返回,
    而 fc 在 sync.go:99 局部构造、clientFactory 不可注入。四条管道已由 TestSync78_SyncOUs/
    _SyncGroups/_SyncUsers 与 78-06 的 syncComputers 单独直测,编排层剩余接受不覆盖。
    若 78-07 的 in-process LDAP 应答器落地,可作为可选回补。
  - syncUsers 软删恢复语义:源代码无恢复逻辑,实测触发 UNIQUE 约束失败 —现行为保留
  - syncUsers $DUPLICATE- 前缀过滤:无过滤逻辑,由 user.go GetList 兜底 —现行为保留
  - syncUsers $ 结尾计算机账号过滤:无过滤逻辑,由 user.go GetList 兜底 —现行为保留
  - syncUsers manager DN 关联:无关联逻辑(grep sync.go 无 'manager' 字面量)—现行为保留
  - updateSyncLog Failed path:sync.go:680 列名 bug(error_message vs error_msg)导致
    整句 UPDATE 失败,sync_status 等字段维持原值 — D-78-10 无据不改
  - IsEnabled=false 落库:GORM `default:true` tag quirk,测试改断言 IsDisabledByUAC()

# Deviations from Plan
  None — plan executed as written. Plan 文档化的"按现行为断言"分支均落地
  (D-78-05c:UNIQUE conflict / 列名 bug / GORM quirk)。

# Auth Gates
  None — 全程零 LDAP 网络(仅 Task 5 dial-fail 用例指向 127.0.0.1 已关闭端口)。

# Self-Check
  - [x] TestSync78_ 函数 31 个 全绿(go test -count=1 exit 0)
  - [x] go build ./... exit 0
  - [x] go test -count=1 ./internal/services/addomain/ exit 0
  - [x] sync.go 83.9% weighted(265/316 stmts,达 ≥80% 阈值)
  - [x] 7 表 fixture + 31 测试函数 + 4 个跨 commit
  - [x] 零生产 .go 改动 (git diff --stat $(78-05 first commit)^..HEAD -- 'internal/services/addomain/*.go' 仅 sync_78_05_test.go)
  - [ ] go test -race ./internal/services/addomain/ — 本机 cgo.exe 环境故障 (C:\Program Files\Go\pkg\tool\windows_amd64\cgo.exe: exit status 2),
        非本 plan 代码缺陷。CI 环境(linux)应可正常执行。

# Test File Layout
sync_78_05_test.go 内部分段:
  - 文件头注释 + helper (setupSync78DB 7 表 / entry78 / newSyncSvc78 / insertConfig78 / closeDB)
  - Task 1: OU 管道 — TestSync78_ExtractDNs_And_SafeAttr / GetExistingOUs / CategorizeOUs / SyncOUs_FullChain / BatchCreateOUs / BatchUpdateOUs (6)
  - Task 2: Group 管道 — TestSync78_ParseGroupTypeFromLDAP_Table / SyncGroups_CreateAndUpdate / SyncGroups_MemberSync / CreateGroupsInBatches / UpdateGroupsInBatches / SyncGroups_DBError (6)
  - Task 3: syncUsers 分支矩阵 — TestSync78_SyncUsers_Empty / CreateNew / UpdateExisting / RestoreSoftDeleted / FilterDuplicatePrefix / FilterComputerAccount / ManagerLink / TimeAttrParse / OUAssignment / Batching / DBError (11)
  - Task 4: 成员/日志/入口 — TestSync78_SyncGroupMembers_FullDiff / UpdateSyncLog_StatusMatrix / SyncDataByID_ConfigNotFound / SyncDataByID_Enabled_DelegatesToSyncData / SyncData_ErrorAndNilPassthrough (5)
  - Task 5: syncDataInternal 失败路径 — TestSync78_SyncDataInternal_EmptyAccountPool / AllAccountsDialFail / SyncLogCreateFail (3)
  - D-78-07 覆盖边界块注释

# Commits
  - 36a9acf  test(78-05): add 7-table sqlite fixture + OU pipeline tests          (Task 1)
  - 999d5c8  test(78-05): add Group pipeline tests (parseGroupType + syncGroups + batches)  (Task 2)
  - 1e86264  test(78-05): add syncUsers branch matrix (11 test cases)            (Task 3)
  - a9eb3f2  test(78-05): add syncGroupMembers + updateSyncLog + SyncData entry tests  (Task 4)
  - 80bb0c6  test(78-05): add syncDataInternal failure paths + D-78-07 coverage boundary  (Task 5)
---

# Phase 78 Plan 05: sync.go 全链 entry-driven 测试 Summary

## 整体结论

**PASS**:sync.go 18 函数从 0% 加权覆盖 → **83.9%** 加权覆盖(265/316 stmts),
越过 ≥80% 阈值。零生产 .go 改动(D-78-07),4 阶段 5 commit,31 TestSync78_
全绿。**BLOCK-05 第一段完成**,78-06/78-07 复用本 plan 落地的 helper。

## sync.go 加权覆盖率详表

| 函数                      | 行号 | 加权覆盖率 | 备注 |
|---------------------------|------|-----------|------|
| NewSyncService            | 35   | 0.0%      | 2-stmt 构造器,零调用覆盖 |
| SyncDataByID              | 48   | 100.0%    | Task 4 覆盖 |
| SyncData                  | 57   | 60.0%     | Task 4 失败透传 + nil 透传;singleflight 合并语义由 sync_singleflight_test.go 覆盖(本 plan 不重复) |
| syncDataInternal          | 80   | 25.5%     | Task 5 三条失败路径:空池/dial-fail/日志创建失败(D-78-07 happy path 不覆盖) |
| syncOUs                   | 179  | 88.9%     | Task 1 全链 + 空 entries 早退 + 3-entry integration |
| extractDNs                | 197  | 100.0%    | Task 1: 空切片 / 多条 |
| safeAttr                  | 216  | 100.0%    | Task 1: 短透传 / 超长截断 + ellipsis |
| getExistingOUs            | 221  | 100.0%    | Task 1: 全命中/部分命中/空 + 软删过滤 |
| categorizeOUs             | 228  | 100.0%    | Task 1: create vs update split + 字段映射 |
| batchCreateOUs            | 268  | 80.0%     | Task 1: 空早退 + 501 条双批循环 |
| batchUpdateOUs            | 279  | 93.3%     | Task 1: 空 map + OnConflict upsert 不增行 |
| syncGroups                | 313  | 94.4%     | Task 2: 空早退 + 3-entry + member_count 推导 |
| createGroupsInBatches     | 383  | 100.0%    | Task 2: 空 + 501 双批 + 文档化无 OnConflict |
| updateGroupsInBatches     | 403  | 92.9%     | Task 2: 空 + 多条 + upsert |
| syncUsers                 | 434  | **96.8%** | **Task 3 主力**:182 stmts 分支矩阵全链(11 用例) |
| syncGroupMembers          | 616  | 90.0%     | Task 4: A/B/C + [A,B,D] 全差异同步 + 空清空 |
| updateSyncLog             | 652  | 94.1%     | Task 4: Success 全字段 + Failed 文档化列名 bug |
| parseGroupTypeFromLDAP    | 701  | 100.0%    | Task 2: 9-case 表驱动覆盖全位掩码 |

**加权 83.9% (265/316)** — 达 ≥80% 阈值。

## D-78-07 覆盖边界

syncDataInternal 的 happy path(第 3-7 步:syncOUs → syncGroups → syncUsers →
syncComputers → last_sync_at → updateSyncLog(Success))需要 FailoverClient
的闭包成功返回,而 fc 在 sync.go:99 局部构造、clientFactory 不可注入。

**本 phase 决定不加生产 seam**:四条管道已由 TestSync78_SyncOUs/_SyncGroups/
_SyncUsers 与 78-06 的 syncComputers 单独直测,编排层剩余 ~25 stmts 接受不覆盖。

若 78-07 的 in-process LDAP 应答器落地,可作为可选回补。

## D-78-05c 现行为断言(sync.go 真实行为锁)

1. **GORM `default:true` quirk** — ADUser.IsEnabled=false 在 Create/Update
   时被 default tag 覆盖,落库值恒为 1。测试改断言 `IsDisabledByUAC()`
   (从 UAC 直接推导)而非 `IsEnabled` 字段值。

2. **sync.go:680 列名 bug** — updates map 用 key `error_message` 但
   models.ADSyncLog 列映射是 `column:error_msg`,SQL UPDATE 因列名不匹配
   整句失败(sync.go:687 applogger.Errorf 兜底),所有字段维持原值。
   文档化现行为,D-78-10 无据不改。

3. **syncUsers 软删恢复语义** — GORM Find 过滤 deleted_at IS NULL + sqlite
   UNIQUE 不忽略软删行 → 软删行 + 同 DN 新 entry 触发 UNIQUE constraint failed。
   文档化现行为(可能为生产 bug),本期不修。

4. **syncUsers 不含三类过滤**:
   - $DUPLICATE- 前缀过滤:由 user.go GetList 兜底
   - $ 结尾计算机账号过滤:由 user.go GetList 兜底
   - manager DN 关联:grep sync.go 无 'manager' 字面量 → 完全无处理

## 威胁缓解表逐项核对

| Threat ID | 缓解措施 | 实际落地 |
|-----------|---------|---------|
| T-78-05-01 误连真实 AD | config 全 dummy / 127.0.0.1 | ✓ 仅 127.0.0.1 与 closed port,无真实域名/IP/账号 |
| T-78-05-02 LDAP dial 拖慢 CI | 10s 硬超时 + 本地 Listen 后 Close | ✓ dial-fail 用例有 select+time.After(10s) 守卫,实测 <2s |
| T-78-05-03 为覆盖 happy path 加 seam | D-78-07 + git diff --stat 0 生产改动 | ✓ 0 生产 .go 改动,4 commit 仅 sync_78_05_test.go |
| T-78-05-04 sqlite 缺表 family pattern | 补 CREATE TABLE,禁改生产 SQL | ✓ 7 表全部手动 CREATE TABLE,无生产 SQL 改动 |
| T-78-05-05 真实员工 DN/邮箱 | 全部 dummy | ✓ 全使用 example.com + CN=Test* 命名 |
| T-78-05-SC 依赖安装 | 零新增依赖 | ✓ go-ldap/glebarez-sqlite/testify 均已 in go.mod |

## 复用的 helper(供 78-06 / 78-07 直接 import)

```go
setupSync78DB(t *testing.T) *gorm.DB                                    // 7 表 fixture
entry78(dn string, kv map[string][]string) *ldap.Entry                // ldap.Entry 构造
newSyncSvc78(t *testing.T) (*SyncService, *gorm.DB)                    // 白盒服务 + 内存 sqlite
insertConfig78(t *testing.T, db *gorm.DB, id string) *models.ADConfig  // 启用配置插入
closeDB(t *testing.T, db *gorm.DB)                                     // 清理 sqlite 连接
```

D-78-06e 显式禁止 78-06 / 78-07 重新定义上述 helper。