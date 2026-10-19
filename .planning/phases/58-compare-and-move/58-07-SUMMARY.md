# Phase 58 Plan 07 Summary — 收口：端到端用例 + SC1~6 证据 + 人工 UAT + REQUIREMENTS 回填

**日期:** 2026-10-19
**Plan:** 58-07（Wave 6 / 7，Phase 58 最后一个 plan）
**前置:** 58-01 ~ 58-06 全部完成
**性质:** 收口 plan —— 正常路径**零生产代码改动**

---

## 一句话

Phase 58 收口：新增 3 个走生产符号的端到端用例（入队 → 排水 → compare-and-move → 终态），
把 SC1~6 六条 Success Criteria 逐条落到「命令 + 实际输出」的证据上，
产出 8 项人工 UAT 清单，并把 QUEUE-04 / QUEUE-05 回填为 Complete。

---

## 交付物

| 文件 | 性质 | 内容 |
|------|------|------|
| `internal/services/addomain/ad_push_queue_drain_test.go` | **追加**测试 | 3 个 `TestPhase58_EndToEnd_*` + 2 个 helper（`ptrStr` / `countQueueRowsForEntity`）。**未改动任何既有用例** |
| `internal/api/v1/system/user_handler_test.go` | 仅注释 | 移除已失效的直推断言点注释（见「偏差 D-1」） |
| `.planning/phases/58-compare-and-move/58-VERIFICATION.md` | 新建 | SC1~6 逐条证据 + F-13 归档核对 + 回归护栏 + 两项不稳定项披露 |
| `.planning/phases/58-compare-and-move/58-HUMAN-UAT.md` | 新建 | 8 项人工 UAT（含 2 项强制告知条目） |
| `.planning/REQUIREMENTS.md` | 5 行 | QUEUE-04 / QUEUE-05 转 Complete + 证据指针 + Coverage 尾注 |

---

## SC 证据表（六条 + F-13）

> 完整命令与完整输出见 [`58-VERIFICATION.md`](58-VERIFICATION.md)。此处为摘要。

| SC | 命令 | 期望 | 实际 | 结论 |
|----|------|------|------|------|
| **SC1** | `grep -rn "ad_push_queue" internal/scheduler/*.go internal/core/db/migrations/*.go \| grep -i "sys_job" \| wc -l` | `0` | **`0`** | **VERIFIED** |
| SC1 | `grep -c "s.cron.AddJob(drainSpec" internal/scheduler/ad_sync_tasks.go` | `1` | `1` | VERIFIED |
| SC1 | `go test … -run "TestPhase58_PickRows\|TestPhase58_Drain_Disable\|TestPhase58_Drain_MaxAttempts\|TestPhase58_Backoff" -v` | 全 PASS | **11 PASS / 0 FAIL** | VERIFIED |
| **SC2** | `go test … -run "TestPhase58_Drain_SkipsMove\|…NormalizedComparison\|…MoveUpdates\|…RepeatDrain\|…UsesLatest\|TestPhase58_EndToEnd_RepeatDrainIdempotent" -v` | 全 PASS | **6 PASS / 0 FAIL** | **VERIFIED** |
| **SC3** | `grep -c "func getDefaultADConfigID" internal/scheduler/*.go` | 每文件 `0` | **21 文件全 `0`** | **VERIFIED** |
| SC3 | `grep -rn "getDefaultADConfigIDForDept" internal/ \| wc -l` | `0` | `0` | VERIFIED |
| SC3 | `go test … -run "TestPhase58_MatchADConfigByDN" -v` | 全 PASS | **7 PASS / 0 FAIL** | VERIFIED |
| **SC4** | `grep -rn 'OU=%s,%s' --include=*.go internal/ \| wc -l` | `0` | **`0`** | **VERIFIED** |
| SC4 | `go test … -run "TestPhase58_BuildOUDN" -v` | 全 PASS | **2 PASS / 0 FAIL** | VERIFIED |
| **SC5** | `go test … -run "TestPhase58_Drain_Code32\|…MoveCode32\|…DNExistsFalse\|TestPhase58_EndToEnd_Code32\|…TerminalRowsNotPicked" -v` | 全 PASS | **5 PASS / 0 FAIL** | **VERIFIED** |
| **SC6** | `grep -c "buildADSyncMap\|adSyncInFlight\|SyncUserUpdateToAD" internal/api/v1/system/user_handler.go` | `0` | **`0`** | **VERIFIED** |
| SC6 | `grep -c "triggerADSyncAfterImport\|BatchSyncUsersToAD" internal/api/v1/system/user_import_handler.go` | `0` | **`0`** | VERIFIED |
| SC6 | `grep -c "userADSyncService \*addomain.UserADSyncService" internal/api/v1/system/user_handler.go` | `1`（保留项） | `1` | VERIFIED |
| **F-13** | `grep -n "oldUserSnapshot" internal/services/system/user_service.go` + `grep -n "Save(\|Updates(" …` | 值拷贝 + selective Updates | `:163 oldUserSnapshot := user`、`:214 Updates(updates)`、`Save(` 0 次 | **已核对归档，无需修改** |

**六条 SC 全部 VERIFIED。** 无一条出现「已验证但无输出」。

### 四条审计否决项的交叉核对

| 否决项 | 核对方式 | 结果 |
|--------|---------|------|
| 禁时间戳仲裁 | 检查 SC2 证据的断言对象是 `displayName`/`department`/`moveUserTargets`（全是**值**） | 未命中 |
| 禁新旧值比较 | 读 `compareAndMoveOU` 源码：判据是 `NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN)`（**当前值 vs 目标值**） | 未命中 |
| 禁 sys_job 注册 | SC1 grep → `0` | 未命中 |
| 禁 JSON 快照 / 禁删 AD 账号 | 队列表 DDL 无快照列；disable 走 `DisableUser`（UAC 514） | 未命中 |

---

## 新增端到端用例（TASK 1）

三个用例全部走**生产符号**（`NewADPushEnqueuer().EnqueueUpdateBatch` / `ADPushDrainer.DrainOnce`），
只有 LDAP 客户端（`mockLDAPClient`）与账号池（`fakePool78`）是 fake。

| 用例 | 覆盖 | 核心断言 |
|------|------|---------|
| `TestPhase58_EndToEnd_UpdateUserPushedAfterDrain` | SC1 + SC2 | `updateUserAttrCalls==1` 且推的是闭包内重读的最新值、`moveUserCalls==1` 且目标为 `OU=IT2,...`、`ad_dn`/`ad_ou_dn` 双写、行 `pushed` + `pushed_at` 非空 |
| `TestPhase58_EndToEnd_RepeatDrainIdempotent` | SC2 | 第二轮 `moveUserCalls` **累计仍 == 1**（compare-and-move 幂等点跨腿成立）、`updateUserAttrCalls==2`（属性每轮真推最新值，防「假幂等」）、`ad_dn` 不变 |
| `TestPhase58_EndToEnd_Code32ThenNoRetry` | SC5 | 第一轮 code 32 → `dead` + `attempts==0` + `ad_dn`/`ad_ou_dn`/`ad_synced_at` 三列清空；第二轮 `updateUserAttrCalls` **停在 1**、`moveUserCalls==0`、`dnExistsCalls` 不增长、终态仍 `dead` |

**与既有单腿用例的分工**（不重复）：Plan 03/04/06 的用例各自锁定一条腿（拾取谓词、退避曲线、
目标一致跳过 MoveUser、单动作终态…），用 `seedDrainQueueRow` **直接铺行**、不经入队器；
本节三例是唯一从**入队器**出发、走完整条生产路径的用例。

**「不再重试」的口径澄清（防误读为假断言）：** 第三例第二轮**会**被 `pickRows` 拾取 ——
若断言「不再被拾取」就与 `groupByResolvedConfig` 的设计相反。真实语义是：拾取后读到 `AdDn`
已空 → 归入 `deadRows` 走 `no-ad-dn` 分支 → 再转 `dead`，**全程零 AD 写**。故断言用
「被拾取但零 AD 写 + 终态仍 dead」，并额外锁死 `dnExistsCalls` 不增长。

---

## 全量回归与 vet（实际输出）

```
$ go build ./...
(无输出，退出码 0)

$ go vet ./internal/services/addomain/ ./internal/scheduler/ ./internal/services/system/ ./internal/api/v1/system/ ./internal/core/db/
(无输出，退出码 0)

$ go test ./... -count=1
(无 FAIL 行，退出码 0，78 个包 ok)

$ go test ./internal/services/system/ -run TestPhase57 -count=1
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/system	3.737s
(41 PASS / 0 FAIL —— Phase 57 零回归)

$ go test ./internal/services/addomain/ -run TestPhase58 -count=1
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	2.417s
(61 PASS)

$ go test ./internal/services/system/ -run TestPhase57_ActiveRowPredicate_MatchesIndexDDL -count=1 -v
--- PASS: TestPhase57_ActiveRowPredicate_MatchesIndexDDL (0.03s)
```

### 自动化覆盖总量

| 包 | `TestPhase58_*` 声明 | 实跑 PASS |
|----|------------------|----------|
| `internal/services/addomain` | 61 | 61 |
| `internal/services/system` | 3 | 3 |
| `internal/core/db/migrations` | 1 | 1 |
| **合计** | **65** | **65** |

### ⚠ 回归期间发现并处置的三个不稳定项（均**非** Phase 58 生产代码问题）

处置原则：先复现，再用 `git worktree` 在**改动前的基线 commit `5aa9fded`** 上对照确认归因。
**全程未使用 `git stash`**（遵守本 phase 的 stash 禁令）。

| # | 用例 | 症状 | 归因 | 处置 |
|---|------|------|------|------|
| 1 | `TestPhase58_EndToEnd_RepeatDrainIdempotent` | 首轮全量回归偶发 `expected "pending", actual "pushed"`；单独跑通过 | **本 plan 新增用例自身的缺陷**：`pushed` 行已退出部分唯一索引，再入队是**新增并行行**（D-06 正确设计）而非合并，导致「同一行」前提失效 + `First` 无 `ORDER BY` ⇒ latent flake | 已修（`30af6251`）：改用行重置 + 新增 `countQueueRowsForEntity` 把前提变可证伪。**断言未弱化**，`-count=10` 确定性通过 |
| 2 | `TestInit78_Close_PartialInit` | 某轮全量回归耗时 31.28s 失败（守卫恰为 30s） | **既有负载敏感 flaky**（`internal/core`，Phase 78）：单独跑 PASS(8.18s)；基线 commit 全量 78 包全 ok；本 tree 复跑全量 78 包全 ok | **如实披露，不隐藏**。非本 phase 引入 |
| 3 | `TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage` | 仅 `-count=5` 失败，`-count=1` 全绿 | **既有共享缓存 DSN 问题**（Plan 05 交付，本 plan 未触碰）：`file:<t.Name()>?mode=memory&cache=shared` 在同进程跨轮次存活。基线 `-count=5` 同样失败 | **不修**（超出 58 范围、与 SC 无关），记录待后续清理 |

---

## F-13 归档核对结论

**F-13 已核对归档，无需修改代码。** 两项均成立：

1. **diff 输入是值拷贝** —— `user_service.go:163 oldUserSnapshot := user`（结构体值拷贝，取自
   `UPDATE` 之前），`:257 EnqueueUpdateIfChanged(tx, &oldUserSnapshot, req)` 取其地址。
   闭包内的写走 `map[string]interface{}` 字段映射，**不回写 `user` 的任何 `*string` 指针**。
2. **写路径是 F-06 的 selective `Updates`** —— `user_service.go:214`
   `tx.Model(&models.User{}).Where(...).Updates(updates)`，其中 `updates` 是显式 map
   （`nil` 字段根本不进 map）；enqueuer 与 drainer 侧 `Save` 出现 **0** 次。

> ⚠ 严格说 `oldUserSnapshot := user` 是**浅拷贝**（共享四个 `*string` 指针）。使别名风险
> 不成立的前提是「闭包内不原地解引用改写这些指针」—— `:214` 的 selective `Updates` 恰好
> 满足（写的是 `*req.Nickname` 的**值**）。这正是 F-06 消解 F-13 的具体机制。

**本 plan 未为 F-13 修改任何生产代码**（符合 D-04「留档不修」与 frontmatter「零生产改动」口径）。

---

## 人工 UAT 清单（8 项）

`58-HUMAN-UAT.md` 共 **8 项**（plan 要求 ≥5）：

| 编号 | 标题 | 归属 | 状态 |
|------|------|------|------|
| UAT-58-1 | 真 AD 上的 compare-and-move（含二次排水幂等） | QUEUE-04 / SC2 | [pending] |
| UAT-58-2 | 账号池熔断期间的退避节奏（attempts 60s→…→30min） | QUEUE-03 / SC1 | [pending] |
| UAT-58-3 | `sys_config` 排水间隔旋钮重启生效 | D-05 | [pending] |
| UAT-58-4 | 多 AD 配置（父子域）最长后缀命中 | D-02 / SC3 | [pending / **单域环境标 N/A**] |
| UAT-58-5 | code 32 真实触发 | QUEUE-05 / SC5 | [pending] |
| **UAT-58-6** | ⚠ **部门同步从未跑过时 `update_user` 批量转 `dead`** | **Q2 刻意后果** | **[pending — 必须知晓]** |
| **UAT-58-7** | ⚠ **「编辑即续命」：持续失败永不终态** | **F-11 已接受方向** | **[accepted by design]** |
| UAT-58-8 | ⚠ 含特殊字符的历史部门需重跑部门同步 | Q5 / HYGIENE-02 取舍 | [pending] |

每项含「前置 / 步骤 / 观察点 / 期望 / 判据 SQL / 失败时该回报什么」六要素。

### 两项强制告知条目的要点

**UAT-58-6（最可能被误报为「功能坏了」）** —— 部门同步从未跑过（`sys_dept_ou_mapping` 为空）时，
用户一编辑部门，队列行会**批量**转 `dead`（`last_error='unrecoverable:no-ou-mapping'`、
`attempts=0`、`pushed_at=NULL`、LDAP 写 0 次）。
**这是 Q2 决策的刻意后果，不是回归。** 清单给出「判别三问」（`last_error` 前缀是否恰好是该串 /
`sys_dept_ou_mapping` 是否为 0 / 域控侧是否未被移动）与处置步骤（先跑部门同步 → 重新编辑或
批量打回 `pending`）。根因：查不到 OU 映射时若退避重试只会无限堆积 `pending`；
若误报 `pushed`，Phase 59 的信号守卫会判成「无待推送 → 已一致」，永久掩盖偏差。

**UAT-58-7** —— 「编辑即续命」：每次编辑经 D-06 合并子句把 `attempts` 清零，故持续编辑 +
持续失败的行**永不进 `dead`**。这是 F-11 的**已接受方向**（用户还在管这件事，反复重试是期望
行为）。清单说明排水器为何不能简单「凭裸 `attempts` 判死」
（`TestPhase58_Drain_NoBareAttemptsDeadAfterMerge` 锁死：`attempts=5` 的行本轮成功必须转
`pushed`），并给出观测 SQL 与手工终结手段。

---

## REQUIREMENTS 回填

diff **恰好 5 行**（2 个 checkbox + 2 行 Traceability + Coverage 尾注）：

```diff
-- - [ ] **QUEUE-04**: 排水执行 OU 移动采用 **compare-and-move**…
-- - [ ] **QUEUE-05**: LDAP code 32（目标 DN 不存在）时队列行转 dead 终态…
+- [x] **QUEUE-04**: …
+- [x] **QUEUE-05**: …
- | QUEUE-04 | Phase 58 | In Progress (…58-07 回填端到端证据后转 Complete) |
- | QUEUE-05 | Phase 58 | In Progress (…58-07 回填端到端证据后转 Complete) |
+ | QUEUE-04 | Phase 58 | Done (2026-10-19, Phase 58 verified — …) |
+ | QUEUE-05 | Phase 58 | Done (2026-10-19, Phase 58 verified — …) |
+
+ Phase 58 完成 QUEUE-03/04/05 + HYGIENE-01/02（5 项…）；v1.21 剩余 GUARD/SHADOW/SAFE/HYGIENE-03~05 待 Phase 59-62。
```

**Phase 57 的 Done 项未被误伤**（机械核对）：`grep -c "QUEUE-01\|QUEUE-02\|QUEUE-06\|HYGIENE-06"`
改动前 `8` → 改动后 `8`，完全一致。

QUEUE-03 / HYGIENE-01 / HYGIENE-02 **本 plan 未动**（此前已是 `[x]`）。

---

## 偏差记录

| 编号 | 类型 | 内容 | 依据 |
|------|------|------|------|
| **D-1** | 偏差（自动修，符合 deviation policy） | 删除 `user_handler_test.go:42` 的失效注释行（`UserADSyncService.SyncUserUpdateToAD 被调用（异步）`）。该测试已是 `assert.True(t, true)` 占位，且被 D-01/D-03 删除的直推路径确实不存在 —— 注释描述了一个**不存在的断言点**，易误导维护者以为该路径仍有行为契约。 | 用户指令明确要求；行为完全未变（仅注释）；补充了指向新链路（队列入队 + 排水器）的说明 |
| **D-2** | 偏差（自动修，修复自身缺陷） | `TestPhase58_EndToEnd_RepeatDrainIdempotent` 首版用「再入队」制造第二轮 → latent flake。改为 plan 原意的「行重置」。 | 首轮全量回归暴露。**根因在用例写法而非生产代码**（再入队新增并行行是 D-06 的正确设计）。断言未弱化 |
| **D-3** | 偏差（自动修，`uint8` 无关） | SC 判定口径细化：第三例第二轮断言写「**被拾取但零 AD 写 + 终态仍 dead**」而非「不再被拾取」。 | plan 的 revision 段已给出该澄清。写「不再被拾取」会与 `groupByResolvedConfig` 设计相反，执行者可能为迎合断言而改坏分组逻辑 |
| **D-4** | 偏差（扩展，超出 plan 的 5 项） | UAT 从 plan 的 5 项扩到 **8 项**，新增 UAT-58-6 / 7 / 8。 | 用户指令的两条强制条目（Q2 后果、F-11 已接受方向）+ Q5 已知取舍 |
| **D-5** | 偏差（行号指针修正） | F-13 核对时发现 plan 指向的 `oldUserSnapshot` 不在 `ad_push_queue_service.go`（enqueuer 内），实际在 `internal/services/system/user_service.go:163`。 | 如实核对代码事实后在 VERIFICATION 中标注；结论不变 |
| **D-6** | 偏差（护栏修正） | `go vet` 的第五个包按用户指令用 `./internal/core/db/`（plan 写的是 `./internal/core/db/migrations/`）。两个都跑，均无输出。 | 用户回归护栏显式列出 `internal/core/db/` |

**未做的事（刻意）：** 未修改任何生产代码；未弱化/删除任何既有断言；
未加 `t.Skip`；未使用 `git stash`；未动 Phase 57 的 Complete 行。

---

## 本 phase 的可见后果与已知取舍（供 Phase 59 消费）

1. **Q2 后果** —— 部门同步未跑过时，`update_user` 行会批量转 `dead`
   （`unrecoverable:no-ou-mapping`），需先跑部门同步。**见 UAT-58-6。**
2. **F-11 后果** —— 「编辑即续命」：持续编辑且持续失败的行永不终态。**见 UAT-58-7。**
3. **D-01 后果** —— 导入路径的入队在 ExcelService 事务**之外**（outbox-lite「同生共死」
   不变量的一处已接受例外，crash 窗口可能丢推送）。
4. **D-01 后果** —— AD 侧更新延迟到排水时才可见（默认 ≤1min）。
5. **Q5 后果** —— HYGIENE-02 只转义**新写入**的 OU DN，含特殊字符的历史部门需重跑部门同步。
   **见 UAT-58-8。**
6. **Q5 后果** —— `SyncUserUpdateToAD` / `BatchSyncUsersToAD` 已无生产调用方，
   待 Phase 62 清理（死代码）。

### 给 Phase 59 GUARD-01 信号守卫的输入

排水器产出的终态是 `pushed` 与 `dead` **两个不同的终态**，且 `last_error` 前缀可区分：

| 前缀 | 含义 | 守卫读数建议 |
|------|------|------------|
| `retry:*` | 可恢复失败（属性 `update_user` / OU 移动 `move_ou`） | **待推送**信号 —— 与 pending 同等对待 |
| `pool:*` | 账号池熔断（可恢复） | **待推送**信号，且应视为「域控侧不可用」而非「数据不一致」 |
| `unrecoverable:user-not-found` | 孤儿行（用户已不存在，F-07） | 终态，**不得**计入「无待推送 → 已一致」 |
| `unrecoverable:no-ad-dn` | 该用户无 AD 引用 | 终态，**不得**计入一致 |
| `unrecoverable:no-ad-config` | 按 `ad_dn` 解析不出配置（无匹配/歧义，D-02） | 终态，**值得告警**（多域配置可能已漂移） |
| `unrecoverable:target-not-found` | code 32 / `DNExists=false`（QUEUE-05） | 终态，**值得告警**（AD 端对象消失） |
| `unrecoverable:no-ou-mapping` | 目标 OU 映射缺失（Q2） | 终态，**指向运营动作**（跑部门同步），不是数据不一致 |
| `dead:max-attempts` | 连续失败达 5 次 | 终态，**值得告警** |

⚠ **守卫设计的关键约束：** `dead` 行**退出** `uq_ad_push_queue_active` 部分唯一索引
（谓词 `status IN ('pending','failed')`），因此 `dead` 行仍在表里可查，但**不占用**活跃索引。
守卫若用「查活跃谓词判有无待推送」，会把这 6 类 `dead` 行当作「无待推送 → 已一致」——
而它们的语义恰恰是「系统认为推过了，AD 侧其实没有/推不动」。**守卫必须单独把 `dead` 行
作为异常信号读出，不能靠「活跃行数为 0」推断一致性。**

---

## 提交

| Commit | 内容 |
|--------|------|
| `1d090937` | `test(58-07)`: 端到端收尾用例 —— 入队→排水→compare-and-move→终态 |
| `f8f88d4e` | `docs(58-07)`: 清理 `user_handler_test.go` 的失效直推断言点注释 |
| `23cc1308` | `docs(58-07)`: SC1~6 逐条证据 + 8 项人工 UAT 清单 + REQUIREMENTS 回填 |
| `30af6251` | `fix(58-07)`: 端到端幂等用例改用行重置而非再入队（原写法是潜在随机 flake） |

---

## Self-Check: PASSED

| 验收命令 | 实际输出 | 结论 |
|---------|---------|------|
| `go build ./...` | 无输出，退出码 `0` | ✅ |
| `go vet ./internal/services/addomain/ ./internal/scheduler/ ./internal/services/system/ ./internal/api/v1/system/ ./internal/core/db/` | 无输出，退出码 `0` | ✅ |
| `go test ./... -count=1` | 无 FAIL 行，退出码 `0`，78 包 ok（复跑一次同结果） | ✅ |
| `go test ./internal/services/system/ -run TestPhase57 -count=1` | `ok … 3.737s`（41 PASS / 0 FAIL） | ✅ |
| `go test ./internal/services/addomain/ -run TestPhase58 -count=1` | `ok … 2.417s`（61 PASS） | ✅ |
| `go test ./internal/services/addomain/ -run "TestPhase58_EndToEnd" -count=1 -v` | 3 PASS / 0 FAIL | ✅ |
| `go test ./internal/services/addomain/ -run "TestPhase58_EndToEnd" -count=10` | `ok … 2.158s`（确定性） | ✅ |
| `grep -c "^func TestPhase58_EndToEnd" internal/services/addomain/ad_push_queue_drain_test.go` | `3` | ✅ |
| `go test ./internal/services/system/ -run TestPhase57_ActiveRowPredicate_MatchesIndexDDL -count=1 -v` | `--- PASS (0.03s)` | ✅ |
| `grep -c "UAT-58-" 58-HUMAN-UAT.md` | `24`（8 个条目 × 编号出现于标题/Summary/正文） | ✅ ≥5 |
| `grep -c "^- \[x\] \*\*QUEUE-04\|…QUEUE-05" .planning/REQUIREMENTS.md` | QUEUE-04/05 均为 `- [x]` | ✅ |
| `grep -c "QUEUE-01\|QUEUE-02\|QUEUE-06\|HYGIENE-06" .planning/REQUIREMENTS.md` | 改动前 `8` → 改动后 `8` | ✅ Phase 57 未误伤 |

**六条 SC 判定：SC1 / SC2 / SC3 / SC4 / SC5 / SC6 全部 VERIFIED**，每条均附命令与实际输出。
**F-13：已核对归档，无需修改代码。**
**人工 UAT：8 项（要求 ≥5），含 2 项强制告知条目。**

---

*Phase: 58 · Plan: 07 · Wave 6/7 · 完成于 2026-10-19*
