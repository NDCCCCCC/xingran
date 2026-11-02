---
phase: 58
slug: compare-and-move
status: draft
researched: 2026-10-17
baseline_commit_verified: "go build ./... exit 0; go test ./internal/services/system/ -run TestPhase57 ok (2.788s)"
---

# Phase 58 排水器与 compare-and-move 推送 — 研究

**Researched**: 2026-10-17
**Domain**: Go / Gin / GORM / robfig-cron v3 / go-ldap v3（AD outbox-lite 排水执行面 + OU DN 转义 + 多 AD 配置路由）
**Confidence**: HIGH（现有代码全部逐行核对；go-ldap / robfig-cron 的库行为直接读模块源码确认）

---

## Summary

Phase 58 要落地的是 Phase 57 建好的 `ad_push_queue` 的**执行面**：一个 in-process 定时排水器，按用户 `ad_dn` 归属解析 AD 配置、在 FailoverClient 闭包内重读 `sys_user` 最新值做 compare-and-move、失败按 60s→2m→4m→8m→16min（封顶 30min）退避重试、attempts 达 5 或遇到不可恢复错误（LDAP code 32 / 配置无匹配）转 `dead` 终态留表可查。同时收掉 D-03 合并的双份 `getDefaultADConfigID`（HYGIENE-01）、给部门名到 OU DN 的拼接加 RFC4514 转义 helper（HYGIENE-02）、删掉两条遗留直推路径（D-01/SC6），并按 D-04 三档复核 F-07~F-16。

**本研究最重要的一句话**：排水器应落在 `internal/services/addomain` 包，**而不是** `internal/scheduler`。理由有三——① `addomain` 包内可直接复用 `UserADSyncService.syncUserAttributes` / `moveUserToNewOU` 两个未导出方法（它们已封装 DNExists 预检、code 32 清空 ad_dn、UAC 514 幂等写入），落在 scheduler 包就只能复制粘贴或把它们导出；② `internal/scheduler` 单向依赖 `addomain`（`ad_sync_tasks.go:12` 反向不存在），把逻辑放 addomain 后 cron 侧只做薄接线，依赖方向干净；③ D-03/D-02 的 helper（配置解析、DN 归一化、OU 转义）也都需要被 `ou_mapping_handler.go`、`dept_sync_tasks.go`、`dept_sync_service.go` 复用，同包才能零环共享。

**次重要的一句话**：ROADMAP 与 STATE 里反复出现的 `NormalizeDN` **在本仓并不存在**（全仓 grep 无此函数），它是本 phase 要新建的单一真实源，不是"复用既有实现"。规划时不要写成"复用 NormalizeDN"。

**Primary recommendation**：新建 `internal/services/addomain/ad_push_queue_drain_service.go`（排水 service + 退避/终态状态机 + D-02 配置解析 + compare-and-move 执行），在 `internal/scheduler/ad_sync_tasks.go` 的 `Start()` 里加第三个 `AddJob`（30s，`cron.NewChain(cron.SkipIfStillRunning(...))`），新增 migration_216 seed `ad_push_queue.drain_interval_seconds`，Wave 0 补齐 sqlite fixture 的 `sys_ad_config.base_dn` 列。

---

## 需求 → 研究支撑映射

| Req | 一句话语义 | 本研究提供的落地依据 |
|-----|-----------|----------------------|
| QUEUE-03 | in-process cron 排水器，不注册 sys_job；退避重试至终态可查 | §4.1 接线点、§4.2 状态机与退避曲线、§4.3 拾取谓词 |
| QUEUE-04 | OU 移动 compare-and-move（闭包内重读 + DN 归一化比较），重复排水幂等 | §4.4 执行面（**含 F-A 关键缺陷**）、§4.5 NormalizeDN 新建 |
| QUEUE-05 | LDAP code 32 → 队列行 dead + 清空 `sys_user.ad_dn`，不重试 | §4.6（`ldap.IsErrorWithCode` 用 `errors.As`，`%w` 包装后仍可判定） |
| HYGIENE-01 | 按 `ad_dn` 归属解析 AD 配置，消灭 `First()`；双份 helper 合并 | §4.7 D-02 最长后缀唯一匹配契约 + §4.8 D-03 合并（**含 F-D 行为差异**） |
| HYGIENE-02 | OU DN 拼接走转义 helper | §4.9（`ldap.EscapeDN` 已存在于 go-ldap v3.4.12） |
| SC6 / D-01 | 两条遗留直推路径均删除，导入路径换批量入队 | §4.10 直推删除影响全景（含 **F-I 编译级依赖**） |
| D-04 | F-07~F-16 三档复核 | §5 逐条落点表 |

---

## 1. 现状事实核对（逐行读码，非推断）

### 1.1 队列基建（排水对象）

| 事实 | 位置 | 对规划的含义 |
|------|------|------------|
| 队列表无 `DeletedAt`、无 `uniqueIndex` tag（防 GORM DROP 冲突） | `internal/models/ad_push_queue.go:66-81` + 注释 `:60-65` | 排水器**不可**依赖软删语义清历史；终态行必须留表 |
| `ADPushQueueActivePredicate = "status IN ('pending','failed')"` 是单一事实源，DDL 与 ON CONFLICT `TargetWhere` 共用 | `models/ad_push_queue.go:53`；`ad_push_queue_service.go:145-171` | **红线**：排水器若引入新状态（如 `processing`），绝不能进这个谓词，否则占用唯一索引 → 同实体新入队直接撞约束 |
| 合并子句每次编辑清零 `attempts` / `last_error` / `next_retry_at` / `failed_at` | `ad_push_queue_service.go:161-169` | 即 F-11：排水器禁止裸 `attempts` 判 dead |
| `BeforeCreate` 生成 UUID + 兜底 `Status=pending`；`BeforeUpdate` 刷 `UpdatedAt`（本地时区，非 UTC） | `models/ad_push_queue.go:85-106` | F-12 一行修的落点；`pushed_at`/`failed_at` 全仓**无写入方**（57-REVIEW 输入清单第 2 条已确认），排水器将是首个写入方 |
| `pushed_at` / `failed_at` 是 `*time.Time`；`next_retry_at` 是 `*time.Time` | `models/ad_push_queue.go:74-80` | `next_retry_at IS NULL` = 立即可拾取（D-05 已锁），索引 `idx_ad_push_queue_status_retry (status, next_retry_at)` 物理支持该谓词 |
| 表 + 三条索引由 migration_215 建，PG/sqlite 双分支 `ensureADPushQueue()` 复用同一函数，**索引失败阻断启动** | `internal/core/db/database.go:1058/1091/1273/1304-1312`；`migrations/migration_215_create_ad_push_queue.go` | 排水器不需要新 DDL（migration_216 只做 sys_config seed） |

### 1.2 AD 执行面（排水器要调的东西）

| 事实 | 位置 | 对规划的含义 |
|------|------|------------|
| `LDAPClientIface` 已含 `MoveUser` / `DisableUser` / `DNExists` / `UpdateUserAttribute` | `internal/services/addomain/ldap_iface.go:18-52` | 排水器执行面无需扩展接口；测试可复用 `ldap_client_mock_test.go` 的 mock |
| `MoveUser` = `ModifyDN(dn, rdn, deleteOldRDN=true, newParent)`，RDN 由 `extractRDNFromDN`（首个逗号前）切出 | `ldap_client.go:350-361, 430-436` | 幂等性：重复对同一 DN 调第二次会 code 32（源 DN 已不存在）→ 必须先比较再移 |
| `DisableUser` = `userAccountControl` Replace 成 `uacDisabledAccount` | `ldap_client.go:337-339, 341-346` | UAC Replace 幂等可重放，天然安全重试 |
| `DNExists` 用 base-object 搜索，code 32 语义转成 `(false, nil)` 不上抛 | `ldap_client.go:509-535` | 排水器的 not-found 预检可直接复用，不需要新方法 |
| `ldap.IsErrorWithCode` 内部用 `errors.As`，**能穿透 `%w` 包装** | go-ldap v3.4.12 `error.go:240-262` | 排水器包了三层 `%w` 仍可判 code 32；不必自己解包 |
| `ErrADTargetNotFound` 哨兵已定义于 `account_pool.go:44` | 同上 | QUEUE-05 终态判定可 `errors.Is` 复用 |
| `FailoverClient.ExecuteWithFailover` 顺序遍历账号；**闭包返回 error 会 MarkFailure 并换账号重跑整个闭包** | `failover_client.go:44-93`（`:87` MarkFailure，`:75-91` 重试循环） | **关键陷阱**（§6-P1）：排水闭包必须**吞掉行级错误、返回 nil**，否则一次 Modify 失败会被放大成"换账号重跑整批" + 熔断累加（记忆：`ad-modify-fail-double-counts-breaker` 前科） |
| `ErrAllAccountsUnavailable` 在闭包调用**之前**返回（池空） | `failover_client.go:52-54` | 账号池熔断耗尽 = 队列级可重试失败，正好由 D-05 的 61min 重试窗桥接 30min 熔断周期 |
| `AccountPool` 常量：`FailureThreshold=3` / `CircuitBreakerDuration=30min` / `DefaultMaxHops=10` | `account_pool.go:29-32` | D-05 退避曲线桥接依据成立 |
| `syncUserAttributes` 已封装：属性映射（displayName/mail/telephoneNumber/department）+ DNExists 预检 + code 32 清 `ad_dn`/`ad_ou_dn`/`ad_synced_at` + 返回 `ErrADTargetNotFound` | `user_ad_sync_service.go:141-230` | 排水器**直接复用**，不复制粘贴 |
| `moveUserToNewOU`：查 mapping → 读 user → `MoveUser` → 只更新 `ad_ou_dn` | `user_ad_sync_service.go:103-136` | **F-A 缺陷**：见 §3 |
| `BatchSyncUsersToAD` 明确注释"不移动 OU"（无导入前旧 dept 快照） | `user_ad_sync_service.go:671-686, 749-756` | D-01 删直推的依据成立：它是 compare-and-move 的严格子集 |

### 1.3 cron 接线面

| 事实 | 位置 | 对规划的含义 |
|------|------|------------|
| `ADSyncScheduler` 用 `cron.New(cron.WithSeconds())`，已注册两个 5min `AddFunc` | `ad_sync_tasks.go:62, 150-172` | 排水器作为第三个 `AddFunc`/`AddJob` 挂在同一 cron 实例 |
| `StartADSyncScheduler(db)` 由 `internal/core/core.go:552` 调用，包在 `if` 块内 | `core.go:548-553` | 唯一启动挂载点，排水器不需要新入口 |
| `globalADSyncScheduler.pool` 是全局账号池单例，`getPool()` 惰性兜底 | `ad_sync_tasks.go:126, 399-414` | 排水器的 FailoverClient 必须复用 `getPool()`，**禁止**自己 `NewAccountPool`（Pitfall 4：独立缓存 → 熔断后仍选到坏账号） |
| `dept_sync_tasks.go` 通过 `getGlobalADAccountPool()` 取全局池，文件内明令"不得直接创建账号池" | `dept_sync_tasks.go:16-27` | 同款约束要写进排水器注释 |

### 1.4 OU DN 与映射

| 事实 | 位置 | 对规划的含义 |
|------|------|------------|
| **OU DN 拼接点全仓只有一个**：`fmt.Sprintf("OU=%s,%s", dept.DeptName, parentOUDN)` | `dept_sync_service.go:115` | HYGIENE-02 的转义 helper 只需改这一处 + 新增 helper |
| `DeptOUmapper.FindOUDNByDeptID` 只按 `dept_id + sync_enabled=true` 过滤，**无 ad_config_id**，且无 ORDER BY | `dept_ou_mapper.go:50-62` | **F-B**：多配置下 `First()` 顺序不确定 → 排水器需要 config 作用域的新方法 |
| `sys_dept_ou_mapping` 唯一键 `(dept_id, ad_config_id)`；**无 DeletedAt（无软删）** | `models/dept_ou_mapping.go:14-15, 19` | 一个 dept 在 N 个配置下有 N 行合法记录 → F-B 的根因 |
| `ldap.EscapeDN` 存在于 go-ldap v3.4.12，按 RFC4514 转义 `"+,;<>\\` + 前导 `#` + 首尾空格，NUL→`\00`；**不转义 `=`** | `ldap.go:351-392` | HYGIENE-02 有现成库函数，**禁手写**；但要注意 `=` 边界（§6-P4） |

### 1.5 其他关键事实

- **`NormalizeDN` 不存在**（`grep -rn "NormalizeDN" --include=*.go` 全仓 0 命中）。addomain 包内现有 DN 相关函数只有 `ParseOUDN`（`utils.go:295`，纯 split + TrimSpace，**不做大小写/空格归一**）与 `extractRDNFromDN`（`ldap_client.go:430`）。→ QUEUE-04 的归一化比较 helper 需**新建**，并作为 DN 语义的单一真实源（与 OU 转义 helper 同放一处）。
- **迁移编号从 216 顺延**（215 已被 `ad_push_queue` 占用；STATE.md Todos 已记录）。
- `migration_203_connection_pool_sysconfig.go` 是 sys_config 单旋钮的完整先例：seed（幂等 count>0 跳过）+ `ConfigTypeNo` / `ConfigIsSystemNo` + "改后重启生效"备注 + `core.go:44-79` `loadConnectionPoolConfig` 启动读取（`readInt` 带 fallback、垃圾值兜底）。→ D-05 单旋钮照抄。
- migrations 包**有** `_test.go` 先例（202/207/208/209/210/213），`freshSQLiteDBForMigrate210` 模式可直接套（57-VALIDATION 里"migrations 包无测试文件"的说法已过时）。

---

## 2. 关键技术栈事实核对（库行为，读模块源码确认）

### 2.1 robfig/cron v3.0.1（`chain.go` / `parser.go` / `cron.go`）

| 事实 | 影响 |
|------|------|
| `SkipIfStillRunning(logger Logger) JobWrapper` 包的是 **`Job`**，其互斥是**闭包私有 channel** | `AddFunc` 不暴露包装入口 → 必须用 `c.AddJob(spec, cron.NewChain(cron.SkipIfStillRunning(lg)).Then(cron.FuncJob(fn)))` |
| 该互斥**无法被手动触发路径复用** | D-05 要求"手动触发复用同一互斥" → 必须在 drain service 内自持 `sync.Mutex`（或 `atomic.Bool`）+ `TryLock()`，让 cron 与手动两条路径共用；cron 的 chain wrapper 可作为第二道保险，但**不能**当作互斥事实源 |
| `AddJob(spec string, cmd Job)` 只接受 **spec 字符串**，不接受 `cron.Schedule` 对象 | `ConstantDelaySchedule` / `cron.Every(d)` **不可用** |
| 6 段 spec 的步进语义：`*/N` → `start=0, end=59, step=N`；`0/N` → `start=0, end=0`（只触发一次/分钟，**不是**"每 N 秒"） | **F-E（关键）**：`drain_interval_seconds` 若配成 60 的因子（30/60/120/300）可映射为 `*/30 * * * * *` / `0 */1 * * * *` / `0 */2 * * * *` / `0 */5 * * * *`；**非因子值（如 45、90）在 6 段 spec 下无法表达**，必须回退默认并 WARN，不能静默按错频率跑 |
| `cron.New(cron.WithSeconds())` 未配 logger → 默认 `PrintfLogger(stdout)` | 排水轮次日志建议加 `cron.WithLogger(...)`，把默认 stdout 打印换成 `applogger`（需要 `PrintfLogger(log.New(...))` 适配，因为 `applogger` 不实现 `Printf`） |

### 2.2 go-ldap/ldap v3.4.12

| 事实 | 影响 |
|------|------|
| `ldap.EscapeDN(dn string) string`（`ldap.go:351-392`）按 RFC4514 转义 | HYGIENE-02 **禁手写转义表**，直接调库函数 |
| `ldap.ParseDN(str)`（`dn.go:246`）与 `DN.EqualFold` / `AncestorOfFold` / `String()` 均导出 | NormalizeDN helper 可基于 `ParseDN` 构造"规范化字符串"，或基于 `strings.ToLower + 去空格"的轻量实现。**取舍**：`ParseDN` 更贴近 RFC4514（含转义解码），但对历史未转义 DN 会解码失败 → helper 必须"解析失败则退回字符串归一"，不能 panic |
| `ldap.IsErrorAnyOf` 用 `errors.As` | `%w` 包装后的 code 32 可判定（§1.2） |

---

## 3. 研究发现的既有缺陷（本 phase 必须处理，planning 时容易漏）

### F-A（阻断级）`MoveUser` 成功后不更新 `sys_user.ad_dn`

`moveUserToNewOU`（`user_ad_sync_service.go:126-134`）与 `BatchMoveUsersToNewOU`（`:319-324`）在 `MoveUser` 成功后**只回写 `ad_ou_dn`，不回写 `ad_dn`**。

ModifyDN 之后用户在 AD 侧的 DN 已经改变，而 `sys_user.ad_dn` 仍指向旧 DN。在 D-02（排水时按 `ad_dn` 解析配置）之下，后果是链式的：

1. 下一轮排水按**陈旧 `ad_dn`** 解析配置 → 若该 DN 的后缀已不属于任何启用配置（或跨域移动后落到另一个 base_dn）→ 该行被误判"配置无匹配"→ **误转 dead**；
2. Phase 59 GUARD-01 / SAFE-01 也以 `ad_dn` 为判据，会一并失真；
3. 属性 Modify 会打到旧 DN → code 32 → 又一次误判 dead。

**结论**：compare-and-move 落地时，MoveUser 成功后必须同时回写 `ad_dn = RDN + "," + newOUDN` 与 `ad_ou_dn = newOUDN`（RDN 用 `extractRDNFromDN` 取）。这是 QUEUE-04 "重复排水幂等"能够成立的前提——否则第二轮比较用的是上一轮已经失效的 DN。

### F-B（阻断级，多 AD 场景）`FindOUDNByDeptID` 未按 ad_config_id 收敛

见 §1.4。排水器解析目标 OU 时必须用 config 作用域的查询，否则同一 dept 在多配置环境下目标 OU 可能取错（把 A 域用户移进 B 域的 OU）。建议新增 `FindOUDNByDeptIDForConfig(ctx, deptID, adConfigID)`，保留旧方法不动（其它调用方：`user_ad_sync_service.go:105`、`:244`、以及 `user_ou_service` 系列）以免波及其它路径。

### F-C（规划事实）`NormalizeDN` 是新建函数，不是复用

见 §1.5。规划文档里不要写"复用 NormalizeDN"。

### F-D（决策项）双份 `getDefaultADConfigID` 的启用条件与 `IsADSyncEnabled` 不一致

| 函数 | 过滤条件 | 位置 |
|------|---------|------|
| `getDefaultADConfigID` | `status = enabled`（**不看 sync_enabled**）+ `ORDER BY created_at ASC` | `ad_sync_tasks.go:306-330` |
| `getDefaultADConfigIDForDept` | 同上，逐字一致 | `dept_sync_tasks.go:47-72` |
| `IsADSyncEnabled` | `sync_enabled = true AND status = enabled` | `ad_push_queue_service.go:43-51` |
| `ou_mapping_handler.go` 第三处 | `sync_enabled = true AND status = enabled` `.First()` | `ou_mapping_handler.go:117` |

合并成一个公共 helper（放 addomain）时，"自动挑选"分支到底判不判 `sync_enabled` 是个**行为变更点**：合并后若统一加上 `sync_enabled`，`ad_sync_tasks.go` 的手动同步任务在"配置启用但同步关闭"时从"能同步"变成"报未找到配置"。**建议**：helper 提供两个显式入口——`FirstEnabledADConfigID`（只判 status，保留两处 scheduler 现行行为）与 `FirstSyncEnabledADConfigID`（判两项，供 ou_mapping 与排水器用），不要合并成一个含混的语义。

### F-E（实现约束）cron spec 频率表达力

见 §2.1。`drain_interval_seconds` 必须经过"能映射成合法 6 段 spec"的校验，不能因子值一律接受。

### F-F（测试基座）sqlite fixture 的 `sys_ad_config` **缺 `base_dn` 列**

`internal/services/system/user_service_test.go:118-126` 的 `CREATE TABLE IF NOT EXISTS sys_ad_config (id, sync_enabled, status, created_at, updated_at, deleted_at)` —— 没有 `base_dn`、`config_name`。D-02 的后缀匹配单测要读 `base_dn`，Wave 0 必须补列（否则只能靠直接 SQL 绕开 GORM，测试价值打折）。注意三套 fixture（`setupUserServiceTestDB` / `setupUserTestDB` / `setupUserCacheTestDB`）是否都要补，按 drain 测试落在哪个包决定。

### F-G（编译级依赖）删 `buildADSyncMap` 会打断既有测试

`internal/api/v1/system/user_handler_test.go:568-582` 的 `TestUserHandler_BuildADSyncMap`（TC22）直接调用 `h.buildADSyncMap(req)`。D-01/D-03 删直推时若连 `buildADSyncMap` 一起删，该用例编译失败 → `go test ./...` 红。处置：删该用例或改写为"该函数已下线"的记录性用例。同文件 `:29-46` 的 `TestUpdateUserWithADSync` 只是注释+`assert.True(true)` 空壳，不受影响。

---

## 4. 落地设计要点（供 planner 决策，均在 CONTEXT D-01~D-05 边界内）

### 4.1 接线点（SC1：in-process cron，不注册 sys_job）

- **逻辑落点**：`internal/services/addomain/ad_push_queue_drain_service.go`（新建）
- **cron 薄接线**：`internal/scheduler/ad_sync_tasks.go` 的 `(*ADSyncScheduler).Start()` 内加第三个 entry
  ```go
  // Phase 58 D-05: 排水器（禁注册 sys_job —— 里程碑锁定）
  interval := addomain.LoadDrainIntervalSeconds(s.db)      // 读 sys_config，启动时一次
  spec, ok := addomain.SecondsToCronSpec(interval)          // 映射失败则回落 30s + WARN
  lg := cron.WithLogger(...)                                // 可选，见 §2.1
  _, err = s.cron.AddJob(spec,
      cron.NewChain(cron.Recover(lg), cron.SkipIfStillRunning(lg)).Then(cron.FuncJob(func() {
          d := addomain.NewADPushDrainer(s.db, s.getPool())
          d.DrainOnce(context.Background())                 // 内部自持互斥
      })))
  ```
- **互斥事实源**：`Drainer` 结构体内 `mu sync.Mutex`，`TryLock()` 失败即本轮跳过（cron 与手动触发共用，`DrainOnce` 内统一处理）。
- **不注册 sys_job**：不写 `sys_job` seed、不加 `RegisterTask`、不加 `RegisterADPushDrainTasks`。（里程碑锁定项，验证清单要能 grep 证伪）

### 4.2 状态机与退避（D-05 已锁，研究只补实现细节）

| 动作 | 状态跃迁 | attempts | next_retry_at | pushed_at / failed_at |
|------|---------|----------|---------------|----------------------|
| 拾取 | `pending`/`failed` → `failed`（失败后）| +1 | `now + backoff(attempts)` | `failed_at = now` |
| 成功 | → `pushed` | 不变（建议归零或保留，排障看得到） | `NULL` | `pushed_at = now` |
| 不可恢复 | → `dead` | **不 +1**（短路不烧） | `NULL` | `failed_at = now` |
| 达到上限 | → `dead` | =5 | `NULL` | `failed_at = now` |

- 退避曲线 `60s → 2m → 4m → 8m → 16m → 30m(封顶)`：建议实现为**纯函数** `backoffDuration(attempts int) time.Duration`（可直接单测、无副作用），仿 `HasADFieldChanged` / `DedupeEntityIDs` 的纯函数先例。
- 时间戳一律 `.UTC()`（F-12 一行修顺带覆盖）。
- `last_error` 建议结构化前缀便于 grep：`unrecoverable:target-not-found`、`unrecoverable:no-ad-config`、`retry:<err>`、`dead:max-attempts`。

### 4.3 拾取谓词（D-05 已锁）

```sql
status IN ('pending','failed')
AND (next_retry_at IS NULL OR next_retry_at <= ?)
ORDER BY created_at ASC        -- F-01 压制后仍需考虑 update_user/disable 相对次序
LIMIT 100                      -- 单轮批量上限
```
命中 `idx_ad_push_queue_status_retry`。`next_retry_at IS NULL` 的显式分支不可省（反向语义会造成永不可拾取的僵尸行）。

**行级认领（planner 决策）**：
- **选项 A（推荐，与 CONTEXT 现状一致）**：不认领。项目是单机部署（REQUIREMENTS 明确"消息队列（MQ）替代队列表 / 单机部署，无收益"），cron `SkipIfStillRunning` + `Drainer.mu.TryLock()` 已覆盖单进程轮次重叠。零 schema 语义变更、零 DDL 风险。
- **选项 B（若未来多实例）**：引入 `status='processing'` + 条件 UPDATE 认领（`UPDATE ... WHERE id=? AND status IN ('pending','failed')`，用 RowsAffected 判定抢到）。**红线**：`processing` 绝不能进 `ADPushQueueActivePredicate`（否则占用唯一索引阻塞入队），且必须配"陈旧 processing 回收"规则（如 `updated_at < now()-10min` 复位为 pending），否则 crash 留永久僵尸。

### 4.4 compare-and-move 执行面（QUEUE-04 / SC2）

**单轮执行形状（推荐）**：先把本轮 ≤100 行按"解析出的 ad_config_id"分组，每个配置一次 `FailoverClient.ExecuteWithFailover`，闭包内串行处理该组所有行。这样 100 行 = 1~2 次 bind，而不是 100 次 bind（`ExecuteWithFailover` 每次调用都 `client.Connect()`）。

```
DrainOnce:
  rows := pick(limit=100)                        // 按 D-05 谓词
  cfgCache := map[string]*models.ADConfig{}        // 配置/账号池级解析结果复用
  groups := groupByResolvedConfig(rows)           // 解析失败 → 直接 dead（不 burn attempts）
  for cfgID, group := range groups:
      cfg := loadConfig(cfgID)                    // 缺失/未启用 → 该组全 dead
      fc := NewFailoverClient(pool, cfg)
      fc.ExecuteWithFailover(ctx, func(c LDAPClientIface) error {
          for row := range group:
              drainRow(ctx, c, row)               // 收集行级错误，**绝不上抛**
          return nil                              // ← 关键：返回 nil 避免换账号重跑整组
      })
```

**行处理（drainRow）**：
1. `Unscoped().Where("id=?", row.EntityID).First(&user)`（**必须 Unscoped**，57-REVIEW 输入清单第 1 条；`disable` 行对应软删用户，DN 仍在行内）
2. `user.AdDn` 为空 → 不可恢复，转 dead（`no-ad-dn`）；**不可**当成成功（否则 Phase 59 守卫误判为"无待推送 → 一致"）
3. 属性推送：复用 `UserADSyncService.syncUserAttributes(ctx, ldapClient, &user, updateReq)`，`updateReq` 从**重读到的最新 user**构造（`deptId/nickname/email/phone`）——这就是"闭包内重读最新值"，不依赖任何入队时刻的旧值
4. OU 移动（仅 `update_user`）：
   - `targetOUDN := mapper.FindOUDNByDeptIDForConfig(ctx, *user.DeptID, cfgID)`（F-B）
   - 无映射 → planner 决策点（§7-Q2，推荐 dead + 明确 `last_error`，不留 pending 无限重试）
   - `NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN)` → **跳过 MoveUser**（幂等点，SC2）
   - 否则 `MoveUser(*user.AdDn, targetOUDN)` → 成功后回写 `ad_dn` + `ad_ou_dn`（F-A）
5. `disable` 动作：`DisableUser(*user.AdDn)`（UAC 514 Replace，幂等），**不**做 OU 移动
6. code 32（属性路径由 `syncUserAttributes` 返回 `ErrADTargetNotFound` 已清空 ad_dn；**OU 路径需排水器自己判**）→ `dead` + 确认清空 `sys_user.ad_dn`

**OU 移动失败是否继续推属性**：现有 `SyncUserUpdateToAD` 的语义是"OU 移动失败仅 ERROR 日志、继续推属性"。compare-and-move 建议保持同一宽容语义（属性推成功即算本轮推进），但把 OU 失败的错误写进 `last_error` 并按可重试处理——否则"OU 永远移不动"的行会被属性成功掩盖成 `pushed`，SC2 的可观测性就没了。这是 planner 的一个显式决策点（§7-Q1）。

### 4.5 NormalizeDN（新建，F-C）

建议语义（轻量、无解析依赖、能处理历史未转义 DN）：
1. `strings.TrimSpace`
2. 按 `,` split，逐段 `TrimSpace`
3. 段内按 `=` 首次出现切分：attr 名 `ToLower`；**值保留原样但去掉首尾空格**
4. 用 `,` 重新拼接

对比"基于 `ldap.ParseDN`"的方案：更贴近 RFC4514 且能正确处理转义序列，但对存量未转义 DN（`dept_sync_service.go:115` 写入的）可能解析失败 → 只能退回字符串路径。建议 **两段式**：`NormalizeDN` 走字符串轻量归一（永不失败、纯函数、可单测），并明确注释"仅用于比较，不用于回写 AD"。

### 4.6 QUEUE-05（code 32 → dead）

- 属性路径已由 `syncUserAttributes` 兑现（`:195-225` 清 `ad_dn`/`ad_ou_dn`/`ad_synced_at` + 返回 `ErrADTargetNotFound`）→ 排水器只需 `errors.Is(err, addomain.ErrADTargetNotFound)` → `dead`。
- OU 路径（`MoveUser` / `DNExists`）：`ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject)` 判定（`%w` 穿透 OK）→ 清 `ad_dn`/`ad_ou_dn` → `dead`。
- **不烧 attempts**（D-05 明文），`last_error` 写可 grep 前缀。
- 后续排水不再重试：行已 `dead`，退出活跃谓词，拾取谓词不命中。

### 4.7 D-02 配置解析契约（HYGIENE-01，SC3）

**纯函数**（可单测、无 DB）：`MatchADConfigByDN(dn string, configs []models.ADConfig) (*models.ADConfig, error)`

```
1. 目标 DN 归一：NormalizeDN(dn)（大小写不敏感）
2. 候选 = 所有 status=enabled 的配置，base_dn 归一后为 dn 的**逗号边界后缀**
   判定必须带逗号边界：strings.HasSuffix(normDN, ","+normBaseDN)
   （不能用裸 HasSuffix —— "…CN=a,DC=xcorp,DC=com" 会被 "DC=corp,DC=com" 误命中）
3. 取 base_dn 最长者（最长后缀唯一匹配）
4. 最长长度并列 → 歧义 → 报错（队列行 dead）
5. 无候选 → 报错（队列行 dead）
```

**必须锁定的反例（CONTEXT 点名的验收项）**：
- 父子域共享后缀：configs = [`DC=corp,DC=com`, `DC=na,DC=corp,DC=com`]，dn = `CN=zhangsan,OU=IT,DC=na,DC=corp,DC=com` → 必须命中**子域**（`DC=na,DC=corp,DC=com`），而不是"第一条"。
- 同 base_dn 双配置（配置重复）→ 歧义 → dead。
- 大小写/空格差异（`cn=Zhang San, ou=IT, dc=corp, dc=com` vs `CN=ZhangSan,OU=IT,DC=corp,DC=com`）→ 命中。
- `DC=xcorp,DC=com` 不得被 `DC=corp,DC=com` 命中（逗号边界反例）。
- `ad_dn` 为空 / 无任何启用配置 → 各自的行为见 §7-Q3。

### 4.8 D-03 helper 合并（HYGIENE-01）

放 `internal/services/addomain`（`api/v1/system` 与 `internal/scheduler` **都已** import addomain，`ou_mapping_handler.go:8` 已 import → 零包环）。
签名按 §4.7 F-D 拆两个入口。三个调用点收敛后：
- `ad_sync_tasks.go:306` → `addomain.FirstEnabledADConfigID(...)`
- `dept_sync_tasks.go:47` → 同上（删除本地副本）
- `ou_mapping_handler.go:117` → `addomain.FirstSyncEnabledADConfigID(...)`（或保留并加注释说明为何第三处仍需 `First()` —— 它是"当前操作的 AD 配置"，无 dn 可依据）

### 4.9 HYGIENE-02 OU DN 转义

- helper：`BuildOUDN(deptName, parentOUDN string) string` = `fmt.Sprintf("OU=%s,%s", ldap.EscapeDN(deptName), parentOUDN)`，放 addomain。
- 调用点：`dept_sync_service.go:115`（**全仓唯一拼接点**）。
- **存量数据口径**（planner 必须决策，§7-Q4）：`sys_dept_ou_mapping.ou_dn` 存量是未转义值。转义只对新写入生效；若比较侧（compare-and-move）与 AD 侧用的是未转义 DN，而新写入用转义 DN，会造成"同一部门两套 DN"。建议：新写入转义 + 比较侧对**两侧都**跑一遍"去转义后归一"（`ldap.ParseDN` 成功则取 `String()`，失败则原样），或明确写进文档"历史数据需重跑部门同步"。

### 4.10 D-01 批量入队变体 + 直推删除影响全景（SC6）

**新变体**（仿 `EnqueueDisable` 的 `ad_push_queue_service.go:263-332`）：
```
EnqueueUpdateBatch(tx, userIDs []string) error
  - DedupeEntityIDs（复用现有纯函数）
  - 调用方负责 ad_dn 非空预过滤（Pluck），或函数内自过滤
  - IsADSyncEnabled(tx) 前置门控（无启用配置 → WARN + 放行）
  - 无 diff 判定（导入场景"无旧值快照"，全量入队是正确语义；这点与 EnqueueUpdateIfChanged 不同，需在注释里写明差异）
  - CreateInBatches(rows, pushQueueBatchSize=500) + adPushOnConflictClause
```
- **挂接点**：`user_import_handler.go:78-81`（`ImportData` 返回后、operlog 前）。`AffectedKeys` 是 **username** 列表 → 需要 `Pluck("id")` 转 userID（这一步已经在被删的 `triggerADSyncAfterImport` 里做过，逻辑可平移）。
- **事务边界**：入队在 ExcelService 事务**外**（`excel_service.go:427` 的 `Transaction` 已提交后才返回 handler）→ 需要自己开一个短事务；D-01 已明确接受 crash 窗口打折，**必须在注释与 SUMMARY 写明这是与 outbox-lite 同生共死不变量的一处例外**。

**直推删除影响全景**：

| 位置 | 内容 | 处置 |
|------|------|------|
| `user_handler.go:28-32` | `adSyncInFlight sync.Map` 字段 + 注释 | 随直推一起删 |
| `user_handler.go:192-238` | goroutine 直推（含 3 次重试 + code 32 短路 break） | 删 |
| `user_handler.go:244-260` | `buildADSyncMap` | 删（注意 F-G 测试依赖） |
| `user_import_handler.go:104-148` | `triggerADSyncAfterImport` goroutine | 换成同步批量入队 |
| `user_handler_test.go:568-582` | `TestUserHandler_BuildADSyncMap` | 删或改写（F-G） |
| `user_handler_test.go:29-46` | `TestUpdateUserWithADSync`（空壳） | 可留（无编译依赖） |
| `user_import_handler_test.go:21/114/217-222` | `userADSyncService=nil` 语义断言 | 保留（该字段仍被 `SyncManagersToAD` 用，`:230`） |
| `user_router.go:35-38` | `NewUserADSyncService(core.DB, pool, nil, mapper)` 注入 | **保留**（`SyncManagersToAD` 仍需） |
| `UserADSyncService.SyncUserUpdateToAD` | 删直推后无调用方 | planner 决策（§7-Q5）：留（未来复用/对比面）或删 |
| `UserADSyncService.BatchSyncUsersToAD` | 删导入直推后无调用方 | 同上 |

---

## 5. F-07~F-16 复核落点（D-04 三档）

| 编号 | 内容 | 落点文件 | 建议动作 | 回归面 |
|------|------|---------|---------|--------|
| F-07 排水器侧 | 孤儿/已删 disable 行 → dead | drain service（与 code 32 同一分支） | 必做（边际成本≈0） | — |
| F-07 入队侧 | `BatchDelete` 未 Pluck 存在性 → 孤儿行 | `internal/services/system/user_service.go:640`（`BatchDelete` 事务内、软删之后入队之前） | 事务内 `tx.Model(&models.User{}).Where("id IN ?", ids).Pluck("id", &existing)`，只对 `existing` 入队；同步调整 `Delete` 是否对称 | **42 个 Phase 57 既有用例**（`go test ./internal/services/system/ -run TestPhase57`） |
| F-10 | Delete/BatchDelete 回滚测试缺失 | `internal/services/system/ad_push_queue_test.go` | 新增 2 例，复制 `TestPhase57_UserService_Update_Disable_RollsBackWhenQueueInsertFails` 的 DROP 表手法；断言 `deleted_at IS NULL` 且 `sys_user_role` 行数不变 | 同上 |
| F-11 | 合并清零 attempts → 毒丸永不 dead | drain service | **禁令进验收**；具体方向见 §7-Q1/§7-Q6 | — |
| F-12 | UTC 不一致（4 处） | `ad_push_queue_service.go:182,273` + `models/ad_push_queue.go:92,104` | 一行改 `.UTC()` | 全量 |
| F-13 | 快照别名注释 | 不修（F-06 已消解），核对归档 | 仅确认注释已改写 | — |
| F-14 | model `status` 缺 `default:'pending'` | `models/ad_push_queue.go:71` | 加 tag 与 DDL 对齐 | `TestPhase57_ADPushQueue_ModelRoundTrip` |
| F-15 | fixture 无注释 | `user_cache_impl_test.go:154-165`、`user_handler_test.go:157-172` | 补"此处故意不 seed 配置行"注释 | — |
| F-16 | `BareInsertWithoutID` 不断言 | `ad_push_queue_test.go:97` | 加 `t.Skip`（PG 契约断言留 v1.22） | — |

---

## 6. 实施陷阱清单（研究期识别，planner 应写进 plan）

| # | 陷阱 | 后果 | 规避 |
|---|------|------|------|
| P1 | FailoverClient 闭包上抛行级 error | 换账号重跑**整组** → 重复 LDAP 写 + `MarkFailure("operation:...")` 累加 → 3 次即熔断 30min（记忆：`ad-modify-fail-double-counts-breaker`） | 闭包内收集行级错误、恒返回 `nil` |
| P2 | 忘了 `Unscoped()` 读 `sys_user` | 软删用户的 disable 行读不到 → 永久失败 → 消耗 attempts | 全部走 `Unscoped()`；单测锁死 |
| P3 | MoveUser 后不回写 `ad_dn` | F-A：下一轮按陈旧 DN 解析配置 → 误转 dead | 见 §3 F-A |
| P4 | 手写 DN 转义表 | 漏字符 / 双重转义 | 用 `ldap.EscapeDN`；注意它**不转义 `=`**（前导 `=` 的部门名是残留边界，必要时自行补 `\`） |
| P5 | 引入 `processing` 状态时把它加进 `ADPushQueueActivePredicate` | 占用部分唯一索引 → 同实体新入队撞 42P10 → 用户编辑整体 500 回滚 | 绝不入谓词；改谓词需同步改 migration DDL + 重跑测试 |
| P6 | 裸 `HasSuffix(dn, baseDN)` | `DC=xcorp` 被 `DC=corp` 误命中 → 推错域 | 必须带逗号边界 |
| P7 | 直接改 `models.ADPushQueueActivePredicate` 文本 | DDL 与 TargetWhere 双双失配 | 该常量是 Phase 57 红线，Phase 58 不动 |
| P8 | 在 drain 闭包里 `time.Now()` 多次取值 | `pushed_at`/`next_retry_at` 与行内容不对应 | 每轮 `now := time.Now().UTC()` 一次，透传 |
| P9 | `drain_interval_seconds` 直接当秒数写进 cron spec | 非 60 因子值得到错误频率 | `SecondsToCronSpec` + 非法回退 + WARN（F-E） |
| P10 | 删直推时漏删 `buildADSyncMap` 的测试 | `go test ./...` 编译红（F-G） | Wave 0 就把该用例一起处置 |
| P11 | 把 drain 放 scheduler 包 | 复用不了未导出方法 → 复制粘贴 `syncUserAttributes` 的 code 32 分支 → 两处漂移 | 放 addomain |
| P12 | 每次 drain 每行一次 `ExecuteWithFailover` | 100 行 = 100 次 bind（~10s+），且账号池 MarkSuccess 被刷 100 次 | 按配置分组，每组一次闭包 |
| P13 | migration_216 只在 PG 分支注册 | sqlite dev/测试环境无 `sys_config` 行 → 回落默认值（可接受但不一致） | PG + sqlite 双分支各注册一次（Phase 57 先例） |
| P14 | 排水器忘了 operlog 约定 | — | 排水器是后台任务非 handler 写操作，`operlog.Record` 不适用；但**若**新增手动触发端点，按 CLAUDE.md 约定在 `response.Success` 前 `operlog.Record(..., OperTypeOther)` |

---

## 7. 待 planner 决策的开放问题（CONTEXT 已明确留给 discretion 的部分）

| # | 问题 | 备选 | 倾向 |
|---|------|------|------|
| Q1 | **F-11 dead 判定的具体方向** | (a) 引入独立 merge-proof 标记列/字段；(b) "failed 且 updated_at 超窗"(c) 显式接受"编辑即续命"并写明后果 | **(c) + 显式文档**：加 (a) 需要新列（Phase 58 迁移面扩大，且与"最小动作行"精神冲突）；建议 (c) 并在 `last_error`/代码注释写清"持续失败行每次编辑都会重置 attempts，故其终态依赖最后一次编辑后的连续失败"，验收项只锁"禁止裸 attempts 判 dead" |
| Q2 | **目标 OU 映射缺失时的处置** | (a) dead；(b) 留 failed 退避重试；(c) 推属性但不移动并转 pushed | 推荐 **(a) dead + 明确 last_error**：映射缺失需运营动作（跑部门同步），重试无意义；但需在 SC 验收里承认"部门同步未跑过时会批量产生 dead 行"这一可见后果 |
| Q3 | **"无匹配"的两种语义必须分开** | (i) 全局零启用 AD 配置；(ii) 单用户 ad_dn 匹配不到 | 推荐 **(i) defer 不动行 + WARN**（配置可能临时关闭，burn attempts 会误伤）；**(ii) dead**（D-02 明文） |
| Q4 | **HYGIENE-02 存量映射口径** | (a) 只转义新写入 + 文档声明需重跑部门同步；(b) 加一步数据迁移重写存量 ou_dn | 推荐 **(a)**：数据迁移要处理 AD 侧是否同步建 OU 的双写问题，风险远大于收益；写进文档 + 验收 |
| Q5 | **删直推后 `SyncUserUpdateToAD` / `BatchSyncUsersToAD` 是否保留** | 保留 / 删除 | 推荐 **保留 + 加注释标注"Phase 58 后无生产调用方，保留作显式管理面/未来复用"**；若删，会牵连 `manager_sync_test.go` 等既有用例 |
| Q6 | **行级认领（§4.3）** | A 不认领 / B `processing` | 推荐 **A**（单机部署 + 已有互斥）；B 留文档 |
| Q7 | 手动触发排水是否需要 HTTP 端点 | 加端点 / 仅代码内手动调用 | SC 未要求；`ad_domain_router.go:117` 已有 `users/batch-sync` 手动同步端点可作宿主先例。若加，按 operlog 约定记录 |

---

## 8. Validation Architecture（Nyquist VALIDATION.md 依赖本节）

### 8.1 Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + `stretchr/testify` (`assert`/`require`) + `github.com/glebarez/sqlite`（in-memory） |
| **Config file** | 无 —— sqlite 测试 `gorm.Open(sqlite.Open(":memory:"))`（先例 `internal/services/system/user_service_test.go:49`；migrations 包用 `file:<TestName>?mode=memory&cache=shared`，先例 `migration_210_normalize_menu_parent_id_test.go:11`） |
| **新增依赖** | 无（robfig/cron、go-ldap、testify 均已在 go.mod） |
| **测试落地文件（建议）** | ① `internal/services/addomain/ad_push_queue_drain_test.go`（新建，排水核心）<br>② `internal/services/addomain/ad_dn_config_match_test.go` 或并入 ①（D-02 纯函数 + NormalizeDN + OU 转义纯函数）<br>③ `internal/services/system/ad_push_queue_test.go`（追加 F-07/F-10）<br>④ `internal/api/v1/system/user_handler_test.go`（F-G 用例处置）<br>⑤ `internal/core/db/migrations/migration_216_*.go` + 可选 `_test.go` |
| **Quick run command** | `go test ./internal/services/addomain/ -run TestPhase58 -count=1` |
| **Second env sentinel** | `go test ./internal/services/system/ -run TestPhase57 -count=1`（Phase 57 的 42 个既有用例必须保持绿 —— D-04 触碰 BatchDelete 的回归面） |
| **Full suite command** | `go test ./...` |
| **Baseline (research 时实测)** | `go build ./...` exit 0；`go test ./internal/services/system/ -run TestPhase57` → ok 2.788s |

### 8.2 Sampling Rate

- **Per task commit**：`go build ./...` + `go test ./internal/services/addomain/ -run TestPhase58 -count=1`
- **Per wave merge**：`go test ./...` + `go vet ./internal/services/addomain/ ./internal/scheduler/ ./internal/services/system/`
- **触碰 Phase 57 代码的 wave（D-01 删除直推 / F-07 BatchDelete Pluck）**：额外跑 sentinel `go test ./internal/services/system/ -run TestPhase57 -count=1`
- **Phase gate**：full suite 全绿 + 静态断言（grep）通过后才进 `/gsd:verify-work`

### 8.3 Req → Test Map（建议测试名，planner 可改名但不得删除断言）

| Req | 行为 | Test Type | 命令 | 建议测试名 |
|-----|------|-----------|------|-----------|
| D-02 / HYGIENE-01 | 父子域共享后缀 → 命中**最长**后缀（子域） | unit（纯函数） | `go test ./internal/services/addomain/ -run TestPhase58_MatchADConfigByDN -v` | `TestPhase58_MatchADConfigByDN_LongestSuffixWins`<br>`TestPhase58_MatchADConfigByDN_ParentChildSharedSuffix`<br>`TestPhase58_MatchADConfigByDN_AmbiguousEqualsLength`<br>`TestPhase58_MatchADConfigByDN_CommaBoundaryRejectsXcorp`<br>`TestPhase58_MatchADConfigByDN_CaseAndSpaceInsensitive`<br>`TestPhase58_MatchADConfigByDN_NoMatchIsError` |
| HYGIENE-02 | 特殊字符部门名 → 合法 DN | unit（纯函数） | 同上 `-run TestPhase58_BuildOUDN` | `TestPhase58_BuildOUDN_EscapesSpecialChars`<br>`TestPhase58_BuildOUDN_PlainNameUnchanged` |
| QUEUE-04 | NormalizeDN 归一化 | unit | 同上 `-run TestPhase58_NormalizeDN` | `TestPhase58_NormalizeDN_CaseSpaceComma`<br>`TestPhase58_NormalizeDN_Idempotent` |
| QUEUE-03 | 拾取谓词（NULL = 可拾取 + 上限 100 + created_at 序） | unit（sqlite） | 同上 `-run TestPhase58_DrainPick` | `TestPhase58_PickRows_NullNextRetryIsPickable`<br>`TestPhase58_PickRows_FutureNextRetrySkipped`<br>`TestPhase58_PickRows_Limit100`<br>`TestPhase58_PickRows_OrderedByCreatedAt`<br>`TestPhase58_PickRows_TerminalRowsNotPicked` |
| QUEUE-03 | 退避曲线纯函数 | unit | 同上 `-run TestPhase58_Backoff` | `TestPhase58_BackoffDuration_Curve`<br>`TestPhase58_BackoffDuration_CappedAt30m` |
| QUEUE-03 | 成功 → pushed（写 pushed_at、清 next_retry_at） | unit + mock LDAP | 同上 `-run TestPhase58_DrainSuccess` | `TestPhase58_Drain_UpdateUser_SuccessWritesPushedAt` |
| QUEUE-03 | 失败 → failed + attempts+1 + next_retry_at 按曲线 | unit + mock LDAP | 同上 | `TestPhase58_Drain_FailureSchedulesBackoff` |
| QUEUE-03 | attempts 达 5 → dead 且可查（留表） | unit + mock LDAP | 同上 | `TestPhase58_Drain_MaxAttemptsGoesDead` |
| QUEUE-03 | `ErrAllAccountsUnavailable`（池熔断）→ 可重试不直接 dead | unit（mock pool） | 同上 | `TestPhase58_Drain_PoolExhaustedIsRetryable` |
| QUEUE-03 | **F-11 禁令**：编辑（合并）后 attempts 归零仍不得误判 dead | unit | `go test ./internal/services/system/ -run TestPhase58 -v` | `TestPhase58_Drain_NoBareAttemptsDeadAfterMerge` |
| QUEUE-04 | compare-and-move：目标 OU 相同 → 不调 MoveUser | unit + mock LDAP | `go test ./internal/services/addomain/ -run TestPhase58_CompareAndMove` | `TestPhase58_Drain_SkipsMoveWhenOUMatches`<br>`TestPhase58_Drain_RepeatDrainIsIdempotent` |
| QUEUE-04 | 目标不同 → 调 MoveUser 且**回写 ad_dn + ad_ou_dn**（F-A） | unit + mock LDAP | 同上 | `TestPhase58_Drain_MoveUpdatesAdDnAndAdOuDn` |
| QUEUE-04 | 闭包内重读最新值（入队后 DB 又改过 → 推最新值） | unit + mock LDAP | 同上 | `TestPhase58_Drain_UsesLatestDBValues` |
| QUEUE-04 | 映射按 config 收敛（F-B） | unit | 同上 | `TestPhase58_Drain_UsesConfigScopedOUMapping` |
| QUEUE-05 | 属性路径 code 32 → dead + ad_dn 清空 + 不烧 attempts | unit + mock LDAP | 同上 `-run TestPhase58_Terminal` | `TestPhase58_Drain_Code32GoesDeadAndClearsAdDn` |
| QUEUE-05 | OU 路径 code 32（MoveUser / DNExists）→ 同上 | unit + mock LDAP | 同上 | `TestPhase58_Drain_MoveCode32GoesDead` |
| QUEUE-05 | 后续排水不再重试该行 | unit | 同上 | `TestPhase58_Drain_DeadRowNotRetried` |
| QUEUE-05 | F-07：孤儿 disable 行（用户已不存在）→ dead | unit | 同上 | `TestPhase58_Drain_OrphanDisableRowGoesDead` |
| D-01 | 批量 `EnqueueUpdateBatch`：预过滤 + 门控 + 去重 + 合并 | unit | `go test ./internal/services/system/ -run TestPhase58 -v` | `TestPhase58_EnqueueUpdateBatch_DedupesAndFilters`<br>`TestPhase58_EnqueueUpdateBatch_NoADConfigSkips` |
| HYGIENE-01 | D-03 三处调用点收敛（**源码断言**） | source-assert | `grep -c "func getDefaultADConfigID" internal/scheduler/*.go` == 0 | 静态 grep，无运行时测试 |
| D-05 | `SecondsToCronSpec`：30/60/120/300 正确，45 非法回落 | unit | `go test ./internal/scheduler/ -run TestPhase58 -v` | `TestPhase58_SecondsToCronSpec_Valid`<br>`TestPhase58_SecondsToCronSpec_InvalidFallsBack` |
| D-05 | migration_216 seed 幂等 | unit | `go test ./internal/core/db/migrations/ -run TestPhase58 -v` | `TestPhase58_Migration216_SeedIdempotent` |
| D-05 / SC1 | 排水器**未**注册 sys_job | source-assert | `grep -rn "ad_push_queue" internal/core/db/migrations/*.go \| grep -i "sys_job"` 无命中 + `sys_job` 表无 `ad_push_queue` seed | 静态 grep |
| F-10 | Delete/BatchDelete 回滚 | unit | `go test ./internal/services/system/ -run TestPhase58 -v` | `TestPhase58_UserService_Delete_RollsBackWhenQueueInsertFails`<br>`TestPhase58_UserService_BatchDelete_RollsBackWhenQueueInsertFails` |
| F-07 | BatchDelete 只对存在行入队（入队数 == 软删数） | unit | 同上 | `TestPhase58_UserService_BatchDelete_EnqueuesOnlyExisting` |
| F-12 | 时间戳 UTC | unit | `go test ./internal/services/addomain/ -run TestPhase58 -v` | `TestPhase58_Drain_WritesUTCTimestamps` |

### 8.4 Wave 0 Requirements（执行前必须先落地，否则后续任务全红）

- [ ] `internal/services/addomain/ad_push_queue_drain_test.go` —— 新建（RED）。需自带 sqlite fixture：`ad_push_queue`（直调 `migrations.Migrate215CreateADPushQueue`）、`sys_user`（含 `ad_dn`/`ad_ou_dn`/`ad_synced_at`/`deleted_at` 全列）、`sys_ad_config`（**必须含 `base_dn` + `config_name`**，见 F-F）、`sys_dept`、`sys_dept_ou_mapping`。**注意 `internal/services/addomain` 包目前没有这种 fixture**，需新建（可复制 `user_ou_service_test.go` 的建表风格）。
- [ ] mock LDAP：`ldap_client_mock_test.go` 已有 `mockLDAPClient`，需确认其已实现 `MoveUser` / `DisableUser` / `DNExists`（接口已要求，实现必然有，但需确认**可注入错误**以驱动 code 32 / 失败路径）。若 mock 无法按用例注入不同错误，需要一个可配置的 fake。
- [ ] mock AccountPool：`account_pool_*_test.go` 已有先例（`ListAvailable` / `MarkFailure` / `MarkSuccess`），需确认可返回空列表以驱动 `ErrAllAccountsUnavailable`。
- [ ] `internal/services/system/user_service_test.go` —— 补 `sys_ad_config.base_dn` 列（F-F）。
- [ ] `internal/api/v1/system/user_handler_test.go` —— 处置 `TestUserHandler_BuildADSyncMap`（F-G），否则 D-01 删除直推时编译红。
- [ ] Framework install：**无**（依赖已具备）。

### 8.5 无法自动化验证的项（必须进人工 UAT 清单）

| 项 | 原因 |
|----|------|
| 真实 AD 上的 compare-and-move（MoveDN 真生效 + 二次排水确实不重复移动） | 需要真域控 |
| 账号池熔断 30min 内排水器的退避行为（attempts 增长节奏） | 需人为触发熔断 |
| `sys_config` 改 `drain_interval_seconds` 后重启生效 | 需重启后端 |
| 多 AD 配置（父子域）在真实环境下 longest-suffix 命中 | 需第二个域 |
| code 32 真实触发（AD 端删对象 → 本地编辑 → 排水 → dead + ad_dn 清空） | 需真域控 |

---

## 9. 环境与工具可用性

| 依赖 | 用途 | 可用 | 版本 | 备注 |
|------|------|------|------|------|
| Go toolchain | 编译/测试 | ✓ | go1.24.5（go.mod `toolchain go1.24.5`） | `sync.Mutex.TryLock` 可用（Go ≥1.18） |
| robfig/cron/v3 | 排水 cron | ✓ | v3.0.1 | `Chain` / `SkipIfStillRunning` / `Recover` 齐备 |
| go-ldap/ldap/v3 | AD 操作 + DN 转义 | ✓ | v3.4.12 | `EscapeDN` / `IsErrorWithCode` / `ParseDN` 齐备 |
| glebarez/go-sqlite | 单测 | ✓ | 见 go.mod | 5 套 sqlite fixture（含 2 套 handler 层） |
| testify | 断言 | ✓ | 见 go.mod | — |
| PostgreSQL | 生产实机验证 | ⚠ 仅人工 | — | 本 phase 无 PG 专属 DDL（仅 sys_config seed，双方言共用），PG 专属风险仅剩 `ON CONFLICT` 二次命中（F-16 留 v1.22） |

**缺失且无回退**：无。**需要人工 UAT 的**：见 §8.5。

---

## 10. 参考资料

**规划文档**
- `.planning/phases/58-compare-and-move/58-CONTEXT.md` — D-01~D-05 锁定决策（本研究全程在其边界内）
- `.planning/REQUIREMENTS.md:11-13, 30-31, 54-76` — QUEUE-03/04/05、HYGIENE-01/02 原文
- `.planning/ROADMAP.md:78-105` — Phase 58 SC1-6 + wave 顺序约束
- `.planning/phases/57-ad-outbox-lite/57-REVIEW.md:147-244` — F-07~F-16 全文 + "给 Phase 58 的输入清单"
- `.planning/phases/57-ad-outbox-lite/57-REVIEW-FIX.md:199-204` — "给 Phase 58 的输入补充"
- `.planning/STATE.md` — 迁移编号顺延、三方审计否决项

**代码（本研究逐行核对）**
- `internal/services/addomain/ad_push_queue_service.go` — 入队 service 全部既有行为
- `internal/models/ad_push_queue.go` — 队列表 + 单一事实源谓词
- `internal/core/db/migrations/migration_215_create_ad_push_queue.go` / `migration_203_connection_pool_sysconfig.go`
- `internal/core/db/database.go:1040-1100, 1260-1312` — 迁移双分支注册 + `ensureADPushQueue`
- `internal/services/addomain/ldap_client.go:337-361, 430-436, 509-535`
- `internal/services/addomain/user_ad_sync_service.go:44-230, 640-803`
- `internal/services/addomain/failover_client.go` / `account_pool.go:25-45`
- `internal/services/addomain/dept_ou_mapper.go:48-62` / `dept_sync_service.go:113-160`
- `internal/models/dept_ou_mapping.go:12-41` / `internal/models/ad_domain.go:22-45`
- `internal/scheduler/ad_sync_tasks.go:62-180, 300-340, 399-414` / `dept_sync_tasks.go:16-72`
- `internal/api/v1/system/user_handler.go:24-45, 159-260` / `user_import_handler.go:66-152`
- `internal/services/operations/excel_service.go:427-500` — 导入事务边界
- `internal/services/system/user_service.go:618-650`（BatchDelete）
- `internal/core/core.go:44-79, 548-553`

**库源码（模块缓存直读）**
- `go-ldap/ldap/v3@v3.4.12/ldap.go:316-392`（`EscapeFilter` / `EscapeDN`）
- `go-ldap/ldap/v3@v3.4.12/error.go:240-262`（`IsErrorAnyOf` 用 `errors.As`）
- `go-ldap/ldap/v3@v3.4.12/dn.go`（`ParseDN` / `DN.EqualFold` / `DN.AncestorOfFold`）
- `robfig/cron/v3@v3.0.1/chain.go`（`Chain` / `Recover` / `DelayIfStillRunning` / `SkipIfStillRunning`）
- `robfig/cron/v3@v3.0.1/parser.go:240-280`（`getField` / `getRange` 步进语义）

---

## 11. 未解问题（RESOLVED —— 下列 4 问已在 Phase 58 规划期全部拍板，无悬空项）

1. **生产是否真的存在多 AD 配置？** 代码层面 `sys_ad_config` 是列表模型、登录路径也只取单个默认配置（`ad_authenticator.go:66` `getADConfig`）。若生产只有 1 个配置，D-02 的复杂度主要是防御性的；这不影响按契约实现，但会影响 §7-Q3 的 dead 策略激进程度。
   - **处置（RESOLVED）**：**N/A —— 按防御性契约实现，dead 策略不因单域而放宽**。D-02 的「最长后缀唯一匹配 / 无匹配或多匹配 → 转 dead 不推」是 CONTEXT 锁定的验收级契约，多域与单域走同一条代码路径，不开特例分支（特例只会把跨域误路由的风险藏起来）。
   - **落点**：契约实现 58-02 Task 2（7 个用例含父子域反例）；单域生产下的验证按 **N/A** 处理 —— 58-07 Task 3 的 `UAT-58-4` 前置写明「生产若只有单域则标 N/A 并在 SUMMARY 注明」，**不得**默认成「已验证」。
2. **`sys_dept_ou_mapping.ou_dn` 存量数据是否含已转义值？** 决定 §7-Q4 的选择。需实机 `SELECT ou_dn FROM sys_dept_ou_mapping WHERE ou_dn LIKE '%\\\\%'` 探测。
   - **处置（RESOLVED）**：**Q4 —— 只转义新写入，不做存量数据迁移**。`BuildOUDN` 只作用于部门同步的**新写入点**（全仓唯一拼接点），历史 `ou_dn` 原样保留；含特殊字符的部门需重跑部门同步才会拿到转义后的 DN。这是刻意取舍：在 `NormalizeDN` 明确「仅用于比较、不得回写」的前提下反推存量迁移，风险大于收益。
   - **落点**：58-02 Task 1 第 3 步（接线处注释写明存量口径 + 重跑要求）；可见后果列入 58-07 输出清单第 5 条（「Q4 后果：HYGIENE-02 只转义新写入」）；`TestPhase58_BuildOUDN_PlainNameUnchanged` 锁住「无特殊字符时与旧行为逐字一致」，保证普通部门的存量口径零影响。
3. **迁移 216 的失败策略**：Phase 57 对队列表建表选了"失败阻断启动"（F-04），而对普通 seed 迁移一律"非阻断"。`sys_config` seed 应沿用哪一档？（倾向：非阻断，照 203 先例 —— 缺旋钮只是回落默认 30s，不会阻断业务。）
   - **处置（RESOLVED）**：**非阻断档，照 `migration_203_connection_pool_sysconfig.go` 先例**。建表阻断是因为 `uq_ad_push_queue_active` 缺失会让用户编辑全线 500；seed 缺失只让 `LoadDrainIntervalSeconds` 回落默认 30s，与业务可用性不同量级。
   - **落点**：58-05 Task 1（`Migrate216` 内 `Create` 失败仅 `log.Printf` + `return nil`；`database.go` 的 PG 与 sqlite 两个分支调用点错误均只 `applogger.Errorf`；`TestPhase58_Migration216_SeedIdempotent` 同时锁「幂等不覆盖管理员改动」）。
4. **管理员可感知性**：QUEUE-03 要求"终态可查询"，但本 phase 没有 UI/端点要求。是否需要一个只读查询端点，还是仅靠 SQL 排障？（CONTEXT 未列，建议不加端点，走 SQL。）
   - **处置（RESOLVED）**：**不开端点（Q7）—— 走 SQL 排障**。「可查询」的兑现形态是：`pushed` / `dead` 两类终态**留表不删**（QUEUE-03），且 `last_error` 有可 grep 的结构化前缀（`unrecoverable:*` / `dead:*` / `retry:*` / `pool:*`），管理员按前缀分类即可定位成因。新增队列查询端点会同时带来权限模型、脱敏（`last_error` 可能含 DN）与 operlog 三项成本，超出本 phase 边界。
   - **落点**：前缀约定由 58-03 Task 1 的 `markFailed` / `markDead` 注释锁定；58-07 Task 3 的五项 UAT 各自给出判据 SQL（如 `SELECT status, attempts, last_error, pushed_at FROM ad_push_queue WHERE ...`），把 SQL 排障路径固化进文档。若将来开端点，须按 CLAUDE.md 在 `response.Success` 前 `operlog.Record(..., OperTypeOther)` 并做权限校验与 DN 脱敏。
