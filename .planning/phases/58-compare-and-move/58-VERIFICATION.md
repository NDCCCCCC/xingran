# Phase 58 验证报告 — SC1~SC6 逐条证据

**执行日期:** 2026-10-19
**执行 plan:** 58-07（Wave 6，Phase 58 收口）
**执行人:** Claude (执行 agent)

---

## 判定口径（本报告的规则）

> **每一条「VERIFIED」必须同时携带 (a) 可复现命令 与 (b) 该命令的**实际**输出。
> 只有"代码看起来对"而没有命令输出的条目，一律判 **NOT VERIFIED** 并写明原因。

**三方审计否决项**（本 phase 全程有效，命中任何一条则该 SC 判 NOT VERIFIED，不工作绕过）：

| 否决项 | 本次核对结论 |
|--------|-------------|
| 禁时间戳仲裁（不拿 `updated_at`/`created_at`/`ad_synced_at` 决定推/跳） | 未命中 —— 见 SC2 证据里 `TestPhase58_Drain_UsesLatestDBValues` 的断言对象是 `displayName`/`department`/`moveUserTargets`，全部是**值**而非时间戳 |
| 禁新旧值比较（闭包重读最新值 vs 目标，不做 old-vs-new diff 决定移动） | 未命中 —— `compareAndMoveOU` 的判据是 `NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN)`（当前值 vs 目标值），见 SC2 |
| 禁 sys_job 注册 | 未命中 —— `grep ... \| grep -i sys_job \| wc -l` == 0，见 SC1 |
| 禁 JSON 快照 / 禁删 AD 账号 | 未命中 —— 队列表 DDL 无快照列；disable 路径调 `DisableUser`（UAC 514），全仓 `DeleteUser` 无调用 |

---

## 总表

| SC | 判定 | 一句话证据指针 |
|----|------|--------------|
| SC1 | **VERIFIED** | `grep -rn "ad_push_queue" internal/scheduler/*.go internal/core/db/migrations/*.go \| grep -i "sys_job" \| wc -l` → `0`；`grep -c "s.cron.AddJob(drainSpec" internal/scheduler/ad_sync_tasks.go` → `1`；11 个 pickRows/退避/disable/上限用例全 PASS |
| SC2 | **VERIFIED** | 5 个 compare-and-move 单腿用例 + `TestPhase58_EndToEnd_RepeatDrainIdempotent` 全 PASS；第二轮 `moveUserCalls` 累计仍 == 1 |
| SC3 | **VERIFIED** | `grep -c "func getDefaultADConfigID" internal/scheduler/*.go` 每文件 `0`；`grep -rn "getDefaultADConfigIDForDept" internal/ \| wc -l` → `0`；`MatchADConfigByDN` 定义数 `1`；7 个路由用例 PASS |
| SC4 | **VERIFIED** | `grep -rn 'OU=%s,%s' --include=*.go internal/ \| wc -l` → `0`；`BuildOUDN` 定义数 `1`；`TestPhase58_BuildOUDN_EscapesSpecialChars` PASS |
| SC5 | **VERIFIED** | 5 个 code-32 相关用例全 PASS；`TestPhase58_EndToEnd_Code32ThenNoRetry` 断言第二轮 `updateUserAttrCalls` 停在 1 |
| SC6 | **VERIFIED** | 三条 grep：`user_handler.go` 直推符号 `0`、`user_import_handler.go` 直推符号 `0`、`userADSyncService` 字段保留 `1`（无任何互斥开关） |

---

## SC1 — 排水器 in-process cron / 两类动作行 / 失败退避至终态可查

> ROADMAP 原文：「排水器以 in-process cron 运行且未注册 sys_job；处理 dept 推送与 disable 账号两类动作行；失败行按退避重试，达上限转终态且可通过查询确认」

### 1a. 未注册 sys_job（静态证伪）

```
$ grep -rn "ad_push_queue" internal/scheduler/*.go internal/core/db/migrations/*.go | grep -i "sys_job" | wc -l
0
```

**实际输出:** `0` —— 期望 `0`。**符合。**

### 1b. in-process cron 接线（静态）

```
$ grep -c "s.cron.AddJob(drainSpec" internal/scheduler/ad_sync_tasks.go
1
```

**实际输出:** `1` —— 期望 `1`。**符合。**

配套：启动日志模板 `ad_sync_tasks.go:213` 含 `ad_push_queue 排水: %s`；账号池经 `s.getPool()` 取全局单例（`NewAccountPool(` 在该文件出现 `3` 次，与 58-05 锁定的 `:106` 熔断恢复局部池 + `:125` 全局单例 + `getPool()` 惰性兜底一致；**低于 3 即为误删**）。

### 1c. 拾取谓词 / 退避曲线 / 单轮上限 / 两类动作行 / 上限转终态（自动化）

```
$ go test ./internal/services/addomain/ -run "TestPhase58_PickRows|TestPhase58_Drain_Disable|TestPhase58_Drain_MaxAttempts|TestPhase58_Backoff" -count=1 -v
--- PASS: TestPhase58_BackoffDuration_Curve (0.00s)
--- PASS: TestPhase58_BackoffDuration_CappedAt30m (0.00s)
--- PASS: TestPhase58_PickRows_NullNextRetryIsPickable (0.03s)
--- PASS: TestPhase58_PickRows_FutureNextRetrySkipped (0.00s)
--- PASS: TestPhase58_PickRows_PastNextRetryPicked (0.00s)
--- PASS: TestPhase58_PickRows_Limit100 (0.01s)
--- PASS: TestPhase58_PickRows_OrderedByCreatedAt (0.00s)
--- PASS: TestPhase58_PickRows_TerminalRowsNotPicked (0.00s)
--- PASS: TestPhase58_Drain_DisableSuccessWritesPushedAt (0.00s)
--- PASS: TestPhase58_Drain_DisableSoftDeletedUserStillDrains (0.00s)
--- PASS: TestPhase58_Drain_MaxAttemptsGoesDead (0.00s)
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	2.062s
```

**11 PASS / 0 FAIL。** 逐条对应：

| SC1 分句 | 承载用例 | 断言 |
|---------|---------|------|
| 失败行按退避重试 | `BackoffDuration_Curve` / `_CappedAt30m` | 60s→2m→4m→8m→16m→30min 封顶 |
| 单轮有界 | `PickRows_Limit100` | 120 行输入只拾取 100 |
| 终态退出活跃谓词 | `PickRows_TerminalRowsNotPicked` | pushed/dead 行 `pickRows` 返回空 |
| disable 账号类动作行 | `Drain_DisableSuccessWritesPushedAt` | `disableUserCalls==1` + `pushed` + `pushed_at` 非空 |
| disable 行对应软删用户仍可推 | `Drain_DisableSoftDeletedUserStillDrains` | 软删后仍 `pushed`（P2 Unscoped 红线） |
| 达上限转终态可查 | `Drain_MaxAttemptsGoesDead` | `dead` + `attempts==5` + `last_error=='dead:max-attempts'` |
| dept 推送类动作行 | `TestPhase58_EndToEnd_UpdateUserPushedAfterDrain`（本 plan 新增） | 属性 Modify + OU 移动 + `pushed` |
| 「可通过查询确认」 | 全部终态用例 | 终态行留表（`markPushed`/`markDead` 只 `Updates` 不 `Delete`） |

**结论：VERIFIED。**

---

## SC2 — compare-and-move

> ROADMAP 原文：「OU 移动采用 compare-and-move：闭包内重读用户最新值 + NormalizeDN 归一化比较，目标一致时跳过 MoveUser；同一队列行重复排水幂等，不产生重复移动」

### 2a. 五个单腿用例

```
$ go test ./internal/services/addomain/ -run "TestPhase58_Drain_SkipsMove|TestPhase58_Drain_NormalizedComparison|TestPhase58_Drain_MoveUpdates|TestPhase58_Drain_RepeatDrain|TestPhase58_Drain_UsesLatest" -count=1 -v
--- PASS: TestPhase58_Drain_SkipsMoveWhenOUMatches (0.03s)
--- PASS: TestPhase58_Drain_NormalizedComparisonSkipsMove (0.00s)
--- PASS: TestPhase58_Drain_MoveUpdatesAdDnAndAdOuDn (0.00s)
--- PASS: TestPhase58_Drain_RepeatDrainIsIdempotent (0.00s)
--- PASS: TestPhase58_Drain_UsesLatestDBValues (0.00s)
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	2.289s
```

**5 PASS / 0 FAIL。** 分句对应：

| SC2 分句 | 承载用例 | 关键断言 |
|---------|---------|---------|
| 闭包内重读用户最新值 | `Drain_UsesLatestDBValues` | 入队后 DB 再改昵称与部门 → 推的 `displayName=='最新昵称'`、`department=='新部门'`、`moveUserTargets[0]=='OU=IT2,...'` |
| NormalizeDN 归一化比较 | `Drain_NormalizedComparisonSkipsMove` | 用户侧 `ou=it , dc=corp , dc=com`（小写+空格）vs 映射侧 `OU=IT,DC=corp,DC=com`（大写）→ `moveUserCalls==0` |
| 目标一致时跳过 MoveUser | `Drain_SkipsMoveWhenOUMatches` | `moveUserCalls==0` |
| 移动后回写双列（F-A） | `Drain_MoveUpdatesAdDnAndAdOuDn` | `ad_dn=='CN=zhangsan,OU=IT2,...'` 且 `ad_ou_dn=='OU=IT2,...'` |
| 同一行重复排水幂等 | `Drain_RepeatDrainIsIdempotent` | 第二轮 `moveUserCalls` 累计仍 == 1 |

**否决项交叉核对（关键）：** `compareAndMoveOU` 的判据源码是

```go
if user.AdOuDn != nil && NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN) {
    return false, nil
}
```

即 **当前值 vs 目标值**，既不是时间戳仲裁，也不是 old-vs-new diff。**符合两条否决项。**

### 2b. 端到端幂等（本 plan 新增）

```
$ go test ./internal/services/addomain/ -run "TestPhase58_EndToEnd_RepeatDrainIdempotent" -count=1 -v
--- PASS: TestPhase58_EndToEnd_RepeatDrainIdempotent (0.01s)
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	2.289s
```

**实际输出:** `--- PASS ... (0.01s)` —— 期望 PASS。**符合。**

本例与 2a 的 `Drain_RepeatDrainIsIdempotent` 的分工（防假幂等）：本例额外断言 `updateUserAttrCalls == 2`（属性每轮**确实**推了最新值）。若只断言 `moveUserCalls` 不增长，"第二轮什么都没干"也会让它变绿。

端到端第二轮走的是**生产入队路径**（`EnqueueUpdateBatch` 的 D-06 合并子句把 `pushed` 终态行打回 `pending` 并清零 `attempts`），而不是手工 `UPDATE` DB —— 这是真实生产形状「用户又被编辑一次」。

**结论：VERIFIED。**

---

## SC3 — 按 `ad_dn` 归属解析配置 + 双份 helper 合并

> ROADMAP 原文：「多 AD 配置环境下推送按用户 `ad_dn` 归属解析配置，不再依赖 `First()` 取首条；`getDefaultADConfigID` 双份实现合并为公共 helper」

### 3a. 双份 helper 已消灭

```
$ grep -c "func getDefaultADConfigID" internal/scheduler/*.go
internal/scheduler/ad_login_history_tasks.go:0
internal/scheduler/ad_sync_tasks.go:0
internal/scheduler/ad_sync_tasks_80_02_test.go:0
internal/scheduler/ad_sync_tasks_test.go:0
internal/scheduler/cron.go:0
internal/scheduler/cron_80_01_test.go:0
internal/scheduler/cron_test.go:0
internal/scheduler/dept_sync_tasks.go:0
internal/scheduler/dept_sync_tasks_80_02_test.go:0
internal/scheduler/mac_history_matview_tasks.go:0
internal/scheduler/mac_history_tasks.go:0
internal/scheduler/mac_tasks_80_02_test.go:0
internal/scheduler/reconciliation_crons.go:0
internal/scheduler/reconciliation_fix_suggestion_monitor.go:0
internal/scheduler/reconciliation_tasks.go:0
internal/scheduler/reconciliation_tasks_80_02_test.go:0
internal/scheduler/reconciliation_tasks_test.go:0
internal/scheduler/vdi_sync_tasks.go:0
internal/scheduler/vdi_tasks_80_02_test.go:0
internal/scheduler/workorder_tasks.go:0
internal/scheduler/workorder_tasks_80_02_test.go:0
```

**实际输出:** 21 个文件全部 `0` —— 期望每行 `0`。**符合**（`getDefaultADConfigID` 与 `getDefaultADConfigIDForDept` 两个定义均已不存在）。

```
$ grep -rn "getDefaultADConfigIDForDept" internal/ | wc -l
0
```

**实际输出:** `0` —— 期望 `0`。**符合。**（逐字拷贝的第二份实现连注释/调用点一并清除。）

```
$ grep -c "func MatchADConfigByDN(dn string, configs \[\]models.ADConfig)" internal/services/addomain/ad_config_resolver.go
1
```

**实际输出:** `1` —— 期望 `1`（公共 helper 单一定义，addomain 包）。**符合。**

### 3b. 路由纯函数（自动化）

```
$ go test ./internal/services/addomain/ -run "TestPhase58_MatchADConfigByDN" -count=1 -v
--- PASS: TestPhase58_MatchADConfigByDN_LongestSuffixWins (0.00s)
--- PASS: TestPhase58_MatchADConfigByDN_ParentChildSharedSuffix (0.00s)
--- PASS: TestPhase58_MatchADConfigByDN_CommaBoundaryRejectsXcorp (0.00s)
--- PASS: TestPhase58_MatchADConfigByDN_AmbiguousEqualsLength (0.00s)
--- PASS: TestPhase58_MatchADConfigByDN_CaseAndSpaceInsensitive (0.00s)
--- PASS: TestPhase58_MatchADConfigByDN_NoMatchIsError (0.00s)
--- PASS: TestPhase58_MatchADConfigByDN_ReadsBaseDnFromDB (0.03s)
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	1.992s
```

**7 PASS / 0 FAIL。** 关键反例：`ParentChildSharedSuffix` 锁住父子域共享后缀（`DC=na,DC=corp,DC=com` 字符串级 ends-with `DC=corp,DC=com`）必须按**最长后缀唯一**命中，`CommaBoundaryRejectsXcorp` 锁住逗号边界（`DC=xcorp,DC=com` 不得被 `DC=corp,DC=com` 误命中）。

### 3c. 「不再依赖 `First()` 取首条」的调度点核对

`groupByResolvedConfig` 一次性 `Find(&configs)` 取全量启用配置，再逐行 `MatchADConfigByDN` 纯函数解析；配置分组 map 的 key 是解析出的 `cfg.ID`。配套用例 `TestPhase58_Drain_UsesConfigScopedOUMapping`（Plan 04）用「cfg-b 的映射行 id 刻意排在前」的反例证明 OU 查询也按配置收敛。

**结论：VERIFIED。**

⚠ **本 SC 的残余人工项**：真实双域环境下的 longest-suffix 命中无法在 sqlite 基座证明 → 见 `58-HUMAN-UAT.md` UAT-58-4（单域环境标记 N/A）。

---

## SC4 — OU DN 特殊字符转义

> ROADMAP 原文：「含特殊字符的部门名经 OU DN 转义 helper 拼接后推送不产生非法 DN」

### 4a. 唯一拼接点已收口

```
$ grep -rn 'OU=%s,%s' --include=*.go internal/ | wc -l
0
```

**实际输出:** `0` —— 期望 `0`。**符合**（仓库内不再有任何裸 `OU=%s,%s` 字符串拼接）。

```
$ grep -c "func BuildOUDN(deptName, parentOUDN string) string" internal/services/addomain/dn_util.go
1
```

**实际输出:** `1` —— 期望 `1`。**符合。**

### 4b. 转义纯函数（自动化）

```
$ go test ./internal/services/addomain/ -run "TestPhase58_BuildOUDN" -count=1 -v
--- PASS: TestPhase58_BuildOUDN_EscapesSpecialChars (0.00s)
--- PASS: TestPhase58_BuildOUDN_PlainNameUnchanged (0.00s)
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	1.992s
```

**2 PASS / 0 FAIL。** `EscapesSpecialChars` 覆盖 `,` `=` `+` `<` `>` `#` `;` `\` `"` 等 LDAP DN 保留字符；`PlainNameUnchanged` 是反向对照（无特殊字符的部门名不得被无谓转义，否则 AD 侧会多出反斜杠）。

**结论：VERIFIED。**

⚠ **已知取舍（Q5，非缺陷）**：HYGIENE-02 只转义**新写入**的 OU DN。含特殊字符的**历史**部门若在本次部署前已建过 OU，其既有 `sys_dept_ou_mapping.ou_dn` 不会被回溯改写，需重跑部门同步。已列入 SUMMARY 的可见后果清单与 UAT 提醒。

---

## SC5 — LDAP code 32 → dead + 清 `ad_dn` + 不再重试

> ROADMAP 原文：「LDAP code 32（目标 DN 不存在）时队列行转 dead 终态并清空 `sys_user.ad_dn`，后续排水不再重试」

### 5a. 自动化

```
$ go test ./internal/services/addomain/ -run "TestPhase58_Drain_Code32|TestPhase58_Drain_MoveCode32|TestPhase58_Drain_DNExistsFalse|TestPhase58_EndToEnd_Code32|TestPhase58_PickRows_TerminalRowsNotPicked" -count=1 -v
--- PASS: TestPhase58_PickRows_TerminalRowsNotPicked (0.02s)
--- PASS: TestPhase58_Drain_Code32GoesDeadAndClearsAdDn (0.00s)
--- PASS: TestPhase58_Drain_DNExistsFalseIsDeadNotPushed (0.00s)
--- PASS: TestPhase58_Drain_MoveCode32GoesDead (0.00s)
--- PASS: TestPhase58_EndToEnd_Code32ThenNoRetry (0.01s)
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	2.059s
```

**5 PASS / 0 FAIL。** 分句对应：

| SC5 分句 | 承载用例 | 关键断言 |
|---------|---------|---------|
| 属性路径 code 32 → dead | `Drain_Code32GoesDeadAndClearsAdDn` | `dead` + `attempts==0` + `last_error=='unrecoverable:target-not-found'` |
| OU 移动路径 code 32 → dead | `Drain_MoveCode32GoesDead` | 同款终态（两条路径同源） |
| 清空 `sys_user.ad_dn` | 上述两条 + `DNExistsFalseIsDeadNotPushed` | `ad_dn`/`ad_ou_dn`/`ad_synced_at` 三列均 NULL |
| 后续排水不再重试 | `PickRows_TerminalRowsNotPicked` + `EndToEnd_Code32ThenNoRetry` | 终态行退出活跃谓词；即使被异常打回 `pending`，第二轮也**零 AD 写** |

### 5b. 「不再重试」的口径澄清（防止误读为假断言）

`TestPhase58_EndToEnd_Code32ThenNoRetry` 第二轮**会**被 `pickRows` 拾取（若断言"不再被拾取"就与 `groupByResolvedConfig` 的设计相反）。真实语义是：拾取后读到 `AdDn` 已空 → 归入 `deadRows` 走 `no-ad-dn` 分支 → 再转 `dead`，**全程零 AD 写**。

该用例的强断言是三重的：

1. `updateUserAttrCalls` 停在 `1`（code 32 之后不得再对 AD 发起 Modify）
2. `moveUserCalls == 0`（OU 移动根本没走到）
3. `dnExistsCalls` 与第一轮结束时的计数相同（不得再调 DNExists 预检）

**结论：VERIFIED。**

---

## SC6 — 两条遗留直推路径均已处置

> ROADMAP 原文：「两条遗留直推路径（`user_handler.go:194` 单用户编辑、`user_import_handler.go:114` 导入后批量）均已按移交待决项 1 处置：或删除、或与排水互斥。**不接受"两者并存且都可能触发"的终态**」

### 6a. 三条 grep

```
$ grep -c "buildADSyncMap\|adSyncInFlight\|SyncUserUpdateToAD" internal/api/v1/system/user_handler.go
0
$ grep -c "triggerADSyncAfterImport\|BatchSyncUsersToAD" internal/api/v1/system/user_import_handler.go
0
$ grep -c "userADSyncService \*addomain.UserADSyncService" internal/api/v1/system/user_handler.go
1
```

**实际输出:** `0` / `0` / `1` —— 期望 `0` / `0` / `1`。**全部符合。**

第三条的 `1` 是**保留项**而非漏删：`userADSyncService` 字段被 `SyncManagers` 复用（用户 → AD 全量同步管理面），与被删的「编辑后直推」是两个不同用途。

### 6b. 「不接受两者并存」的反向核对 —— 确实没有互斥开关

SC6 允许的两种终态是「都删」或「加互斥」。本次是**都删**，因此必须额外证明**没有**引入任何互斥开关（若既删了直推又留了个 `adSyncDirectPushEnabled` 之类的开关，说明存在半成品分支）。58-06 的 5 条 grep 静态断言在 `ab8a7231` 已全 0，本 plan 复跑未新增任何开关。

### 6c. 遗留注释的清理（本 plan 新增，见提交 `f8f88d4e`）

`internal/api/v1/system/user_handler_test.go:42` 仍把 `UserADSyncService.SyncUserUpdateToAD 被调用（异步）` 列为验证点，而该测试已是 `assert.True(t, true)` 占位、且被删路径确实不存在。已移除该失效注释行并替换为指向新链路的说明。**行为未变，无断言改动。**

**结论：VERIFIED。**

---

## F-13 归档核对（只核对，不改代码）

> Phase 57 F-13：「合并子句写回快照导致别名」的注释级问题。D-04 判定为「留档不修」，理由是 F-06 的 selective `Updates` 已从根上消解别名风险。

### 核对项 1 — diff 输入是**值拷贝**

plan 指向的 `oldUserSnapshot` 实际定义在 `internal/services/system/user_service.go`（不在 enqueuer 内，plan 的行号指针略有偏差）：

```
$ grep -n "oldUserSnapshot" internal/services/system/user_service.go
159:	// ⚠ 不变量（F-06 / F-13）：oldUserSnapshot 的四个 *string 字段（nickname/email/phone/
160:	// dept_id）**不得被原地解引用改写**（`*oldUserSnapshot.Nickname = ...`），否则新旧值
161:	// 共享同一字符串，HasADFieldChanged 恒返回 false → 入队静默失效且无任何测试转红。
162:	// 当前写路径不再改动 user 结构体（改为 selective Updates 字段映射），该不变量自然成立。
163:	oldUserSnapshot := user
209:		// 且 status 跃迁由 F-02 的入队逻辑显式 diff（oldUserSnapshot.Status vs req.Status），
257:		if err := enqueuer.EnqueueUpdateIfChanged(tx, &oldUserSnapshot, req); err != nil {
264:		if err := enqueuer.EnqueueDisableOnStatusTransition(tx, oldUserSnapshot.Status, req.Status, req.ID); err != nil {
```

**判定:** `oldUserSnapshot := user`（:163）是 `models.User` **结构体值拷贝**，取自 `UPDATE` 之前；`EnqueueUpdateIfChanged(tx, &oldUserSnapshot, req)`（:257）取其地址。闭包内的 `Updates` 走的是 `map[string]interface{}` 字段映射，**不回写 `user` 结构体的任何 `*string` 指针**，故旧值不可能被"自己改自己"。

⚠ 严格说这是**浅拷贝**：`oldUserSnapshot` 与 `user` 共享四个 `*string` 指针。要使别名风险不成立，前提是「闭包内不原地解引用改写这些指针」—— :214 的 selective `Updates` 恰好满足该前提（它写的是 `*req.Nickname` 的**值**，不是改写 `user.Nickname` 指向的字符串）。这正是 F-06 消解 F-13 的具体机制。

### 核对项 2 — 写路径是 F-06 的显式字段 `Updates`，不是 `Save` 全字段覆盖

```
$ grep -n "Save(\|Updates(" internal/services/system/user_service.go | head -20
169:		// 原实现是 tx.Save(&user) 全字段覆盖 —— 指针字段 nil 被 GORM 写成 NULL，
214:		if err := tx.Model(&models.User{}).Where("id = ?", req.ID).Updates(updates).Error; err != nil {
```

```
$ grep -rn "\.Save(" internal/services/addomain/ad_push_queue_service.go internal/services/addomain/ad_push_queue_drain_service.go | wc -l
0
$ grep -c "Updates(" internal/services/addomain/ad_push_queue_service.go
1
```

**判定:** 生产写路径唯一一处是 `:214` 的 `Updates(updates)`，其中 `updates` 是显式 `map[string]interface{}`（`nil` 字段根本不进 map，天然不写）；enqueuer 侧 `Save` 出现 `0` 次、`Updates` `1` 次（`EnqueueDisable` 的 F-01 压制子句）；drainer 侧 `Save` 出现 `0` 次。**符合。**

### 结论

**F-13 已核对归档，无需修改代码。** 两项核对均成立：别名风险已被 F-06 的 selective `Updates` 从根上消解 —— 写路径不回写结构体指针，故浅拷贝的共享指针不构成新旧值串扰。

**本 plan 未为 F-13 修改任何生产代码**（符合 D-04「留档不修」与 plan 的 frontmatter「零生产改动」口径）。

---

## 回归护栏

| 命令 | 实际输出 | 判定 |
|------|---------|------|
| `go build ./...` | 退出码 `0`，无输出 | PASS |
| `go vet ./internal/services/addomain/ ./internal/scheduler/ ./internal/services/system/ ./internal/api/v1/system/ ./internal/core/db/` | 无输出，退出码 `0` | PASS |
| `go test ./... -count=1` | 见 `58-07-SUMMARY.md`「全量回归」段 | PASS |
| `go test ./internal/services/system/ -run TestPhase57 -count=1` | `ok ... 4.579s` | PASS（零回归） |
| `go test ./internal/services/addomain/ -run TestPhase58 -count=1` | `ok ... 2.926s`（61 PASS） | PASS |
| `go test ./internal/services/system/ -run TestPhase57_ActiveRowPredicate_MatchesIndexDDL -count=1` | `--- PASS (0.02s)` | PASS（P7 证伪未触发） |

### P7 证伪（谓词常量 ↔ migration_215 索引 DDL）

```
$ go test ./internal/services/system/ -run TestPhase57_ActiveRowPredicate_MatchesIndexDDL -count=1 -v
--- PASS: TestPhase57_ActiveRowPredicate_MatchesIndexDDL (0.02s)
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/system	2.792s
```

**含义:** `models.ADPushQueueActivePredicate`（`"status IN ('pending', 'failed')"`）与 `migration_215` 的 `uq_ad_push_queue_active` 部分唯一索引 DDL 谓词仍**文本一致**。本 phase 大量新增 `dead` 终态写入，若某处误改谓词文本，此用例必红 —— 它是"终态行能否退出唯一索引"的物理基础。**PASS。**

---

## 本 phase 自动化覆盖总量

```
$ grep -rh "^func TestPhase58_" internal/ | wc -l
65
```

分布与实跑 PASS 数：

| 包 | 声明 | 实跑 PASS |
|----|------|----------|
| `internal/services/addomain` | 61 | 61 |
| `internal/services/system` | 3 | 3 |
| `internal/core/db/migrations` | 1 | 1 |
| **合计** | **65** | **65** |

---

## NOT VERIFIED / 需人工环境的事项

以下**均不由本报告的自动化证据覆盖**，已全部转留 `58-HUMAN-UAT.md`（不做默认"已验证"）：

| 编号 | 事项 | 为什么不可自动化 |
|------|------|----------------|
| UAT-58-1 | 真 AD 上的 compare-and-move（ModifyDN 真生效 + 二次排水不重复移动） | 需真域控，mock 无法证明 ModifyDN 的 wire 行为 |
| UAT-58-2 | 账号池熔断 30min 内的退避节奏（attempts 增长符合 60s→…→30min） | 需人为触发熔断 + 真实时钟跨度 |
| UAT-58-3 | `sys_config` 改 `drain_interval_seconds` 后重启生效 | 需重启后端进程 |
| UAT-58-4 | 多 AD 配置（父子域）真实环境 longest-suffix 命中 | 需第二个域；单域环境标 N/A |
| UAT-58-5 | code 32 真实触发（AD 端删对象 → 本地编辑 → 排水 → dead + 清空 ad_dn） | 需真域控 |

另有三项**非 SC 要求但用户可见的后果**，已作为独立 UAT 条目留档（UAT-58-6/7/8）：

- **UAT-58-6** 部门同步从未跑过（`sys_dept_ou_mapping` 为空）→ `update_user` 行批量转 `dead`（`unrecoverable:no-ou-mapping`）—— **这是决策 Q2 的刻意后果，不是回归**
- **UAT-58-7** 「编辑即续命」：持续编辑且持续失败的行永不终态 —— **已接受方向，非缺陷**
- **UAT-58-8** 含特殊字符的历史部门需重跑部门同步（Q5 只转义新写入）

---

*Phase: 58 · Plan: 07 · 生成于 2026-10-19*
