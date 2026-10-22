# Phase 58 目标反向验证（GOAL-BACKWARD VERIFICATION）

**执行日期:** 2026-10-19
**验证者:** Claude（独立验证 agent，非执行 agent、非 58-REVIEW 的 reviewer）
**方法:** goal-backward —— 对每条 SC 先问「若此条为假，用户/运维会观察到什么」，再去找能抓到这个观察的代码与测试；执行者的 `58-VERIFICATION.md` 只作为**待核对的声明集**，不作为事实来源。
**被验证的代码:** HEAD = `65f9e151`（工作区除 `.planning/` 外无本 pass 引入的改动；两个临时探针文件已删除，`git status --porcelain internal/services/addomain/` 为空）

---

## 0. 本次独立执行的命令与实际输出

| 命令 | 实际输出 |
|------|---------|
| `go build ./...` | 无输出，EXIT=0 |
| `go test ./internal/services/addomain/ -run TestPhase58 -count=1` | `ok … 2.375s` |
| `go test ./internal/services/system/ -run TestPhase57 -count=1` | `ok … 24.264s`（Phase 57 哨兵零回归） |
| `go test ./internal/core/db/migrations/ -run TestPhase58 -count=1` | `ok … 2.852s` |
| `go vet ./internal/services/addomain/ ./internal/scheduler/` | 无输出，EXIT=0 |
| 探针 A（本次新增，跑完即删）：池侧失败下退避档位 | 见 §4 H-03 判定 |
| 探针 B（本次新增，跑完即删）：长时域控宕机后 attempts 膨胀 | 见 §5 MISSED-02 |

结论：执行者记录的构建/测试结论**属实**，不是回填的。本报告的分歧不在「测试是否绿」，而在「绿的测试到底证明了什么」以及「测试没覆盖的行为」。

---

## 1. SC 逐条判定

### SC1 — 排水器 in-process cron / 两类动作行 / 失败退避至终态可查

**VERIFIED（含一条必须随附的告警）**

| 分句 | 若为假会观察到什么 | 我找到的证据 | 方式 |
|------|------------------|-------------|------|
| 以 in-process cron 运行 | 启动日志无 `ad_push_queue 排水`；重启后队列不消化 | `ad_sync_tasks.go:208-217` `NewADPushDrainer` 在 `AddJob` 之前构造一次、闭包捕获同一实例（H-01 修复后的真实形态），`cron.NewChain(Recover, SkipIfStillRunning).Then(FuncJob(...))`；`core.go:552` `StartADSyncScheduler(c.GetDB())` 在启动路径上 | 读 |
| 未注册 sys_job | `sys_job` 表出现排水任务，UI 可暂停/改 cron | `grep -rn "ad_push_queue" internal/core/db/migrations/*.go \| grep -i sys_job` → **0**；`internal/scheduler/ad_sync_tasks.go` 内 2 处 `sys_job` 全是注释（`:174`）；migration_216 只 seed `sys_config` 一行 | 执行 |
| 处理 disable 行 | 删用户后 AD 账号未停用 | `drain_service.go:374-381` → `DisableUser`（UAC 514 Replace，幂等） | 读 + `TestPhase58_Drain_DisableSuccessWritesPushedAt` PASS |
| 处理「dept 推送」行 | 改部门后 AD 的 OU / department 不变 | `drainRow` 的 `update_user` 分支（`:383-416`）：属性 Modify（`department` 属性走 `syncUserAttributes`）+ `compareAndMoveOU` 搬 OU | 读 + `TestPhase58_EndToEnd_UpdateUserPushedAfterDrain` PASS |
| 失败退避重试 | 行永远 pending 或直接死 | `markFailed:310-331`：`attempts+1` + `next_retry_at = now + backoffDuration(n)` | 读 + 5 个退避/上限用例 PASS |
| 达上限转终态 | 行无限重推 | `markFailed:318` `newAttempts >= maxDrainAttempts(5)` → `markDead("dead:max-attempts")` | 读 + `TestPhase58_Drain_MaxAttemptsGoesDead` PASS |
| 可通过查询确认 | 终态行消失，运维看不到为什么 | `markPushed/markDead` 只 `Updates` 不 `Delete`；模型无 `DeletedAt`；`last_error` 有可 grep 前缀约定（`unrecoverable:* / dead:* / retry:* / pool:*`） | 读 |

**告警（不改变判定，但必须记录）：** 「attempts 上限 = 5 次业务失败」这一**意图**在生产中被基础设施故障消耗掉了。探针 B 实测：一次约 2 小时的域控不可达（12 轮全部拨号失败，H-04 修复后走 `poolExhausted` 分支）会把 `attempts` 累到 **12**；AD 恢复后的**第一次**业务失败立刻把行打成 `dead`（attempts=13）。即：**任何一次长时 AD 宕机之后，整批队列行都处在「离终态只差一次错误」的状态**，且运维从 `attempts`/`last_error` 上看不出其中 12 次没有一次是业务失败。详见 §5 MISSED-02。

**口径澄清（执行者的读法我接受，但要写明）：** ROADMAP SC1 原文「处理 dept 推送与 disable 账号两类动作行」。队列表当前只有 `entity_type='user'`（`models/ad_push_queue.go:18` 仅 `ADPushEntityUser` 一个常量），**不存在 dept 实体行**。执行者把「dept 推送」解读为「承载部门变更的 `update_user` 行」。我按同一解读判定 VERIFIED，但 `pickRows:171-179` **不过滤 `entity_type`**（review L-01），若将来加 dept 实体行会被当 user 处理并批量误判 dead —— 这是一个已知的、尚未引爆的扩展位缺口。

**另一条口径**：终态「可查询」当前**只有 DB 查询**（无 API / 无前端页面，`grep -rl ADPushQueue internal/api/` 仅命中测试文件）。这符合 QUEUE-03 的「终态留表可查」与 Phase 59 的读端点计划，但运维在 UI 上看不到队列。

---

### SC2 — compare-and-move

**VERIFIED**

| 分句 | 若为假会观察到什么 | 证据 | 方式 |
|------|------------------|------|------|
| 闭包内重读最新值 | AD 上是入队那一刻的旧昵称/旧部门 | `drainRow:357` `loadUserUnscoped` 在执行点重读；`buildDrainUpdateReq(user)` 只用重读值构造载荷；全函数无「入队快照」变量 | 读 |
| NormalizeDN 归一化比较 | 已搬移的用户被二次 `ModifyDN` → code 32 → 绑定被清 | `compareAndMoveOU:470` `NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN)` | 读 |
| 目标一致时跳过 MoveUser | 日志里每 30 秒一条 ModifyDN | 同上行 `return false, nil` | 读 |
| 重复排水幂等 | 第二次排水仍发 MoveUser | — | 执行 |

**测试是否真的撑得住（这是我重点复核的一条，因为「幂等」类断言最容易是空断言）：**
- `TestPhase58_Drain_NormalizedComparisonSkipsMove:892` 铺用户侧 `ou=it , dc=corp , dc=com` vs 映射侧 `OU=IT,DC=corp,DC=com`，断言 `moveUserCalls==0`。**把 `NormalizeDN` 换成裸比较，本用例必红** —— 归一化这一步是有牙齿的。
- 反面（真会移动的腿）由 `TestPhase58_Drain_MoveUpdatesAdDnAndAdOuDn:913` 兜住：断言 `moveUserCalls==1` 且 `ad_dn` 被改写成 `CN=zhangsan,OU=IT2,...`。**把 `compareAndMoveOU` 整个删掉，三条 skip/幂等用例会绿，但这一条会红** —— 所以整个套件不会被「生产逻辑删了还绿」骗过。
- `TestPhase58_EndToEnd_RepeatDrainIdempotent:1573` 额外断言 `updateUserAttrCalls==2`，防止「第二轮什么都没干也算幂等」的假绿；并新增 `countQueueRowsForEntity == 1` 把「同一行」变成可证伪前提。

⚠ **文档与代码脱节（review 已提、修复未做，我独立确认仍然存在）：** `58-VERIFICATION.md:195` 至今写着「端到端第二轮走的是**生产入队路径**（`EnqueueUpdateBatch` 的 D-06 合并子句把 `pushed` 终态行打回 `pending` 并清零 `attempts`）……这是真实生产形状『用户又被编辑一次』」。当前测试（`:1595`）是**裸 SQL `UPDATE` 打回 pending**，不走入队路径；而且它描述的那套合并语义**根本不可能发生** —— `pushed` 行已退出 `uq_ad_push_queue_active` 部分唯一索引，合并子句永远不会命中它（这正是当初 flaky 的根因）。这是本次唯一一处**执行者的证据叙述在事实上是错的**的地方（不是测试错，是报告错）。

---

### SC3 — 按 `ad_dn` 归属解析配置 + 双份 helper 合并

**VERIFIED**

- 路由：排水器不再用 `First()`。`groupByResolvedConfig:184-231` 一次 `Find` 全量启用配置，逐行 `MatchADConfigByDN` 纯函数解析，组 key = 解析出的 `cfg.ID`。
- 纯函数正确性：`MatchADConfigByDN` 带逗号边界 `strings.HasSuffix(norm, ","+base)`、取最长、并列即歧义报错、显式跳过空 `base_dn`（否则空串后缀恒命中）。反例有测试：`ParentChildSharedSuffix`（父子域共享后缀）、`CommaBoundaryRejectsXcorp`（`DC=xcorp,DC=com` 不得被 `DC=corp,DC=com` 命中）、`AmbiguousEqualsLength`。我逐行推演了「等长但不同的两个 base 字符串能否同时是同一 norm 的逗号边界后缀」—— 不能（长度 L 的后缀由末尾 L 字符唯一确定），所以歧义分支只会由「两条启用配置指向同一域」触发，报错转 dead 符合 D-02 契约。
- 卫生项：`func getDefaultADConfigID` / `getDefaultADConfigIDForDept` 两个定义在 `internal/scheduler/*.go` 21 个文件中均为 **0**；全仓 `getDefaultADConfigIDForDept` 引用 **0**。三个原调用点收敛为 addomain 包两个显式入口：`ad_sync_tasks.go:383` → `FirstEnabledADConfigID`（只判 status，刻意保持旧行为）、`dept_sync_tasks.go` → 同款、`ou_mapping_handler.go:119` → `FirstSyncEnabledADConfigID`（双条件，无 ORDER BY，刻意不改语义）。**没有偷偷把两个语义不同的入口合并成一个**，这是对的。
- 反例测试 `TestPhase58_FirstEnabledADConfigID_SkipsSyncDisabled` 锁定「启用但 sync 关闭」时两者取到不同行。

⚠ 残余人工项：真实双域环境下的 longest-suffix 命中 sqlite 证不了（UAT-58-4，单域环境标 N/A）。

---

### SC4 — OU DN 特殊字符转义

**VERIFIED（范围：新写入的 OU DN）**

- 全仓无裸 `OU=%s,%s` 拼接（`grep -rn 'OU=%s' --include=*.go internal/ | grep -v _test` 只剩 4 条日志文案）。
- `BuildOUDN`（`dn_util.go:63`）是唯一定义，唯一生产调用点是 `dept_sync_service.go:119` —— 而 `syncDeptTree` 正是 `sys_dept_ou_mapping.ou_dn` 的**唯一写者**（`UpsertMapping`）。也就是说：部门同步产出的 OU DN 一定过转义 helper。
- 转义走 `ldap.EscapeDN`（RFC4514）而非手写转义表，并补了 `EscapeDN` 不转义 `=` 的残留边界。`TestPhase58_BuildOUDN_EscapesSpecialChars` 覆盖 `, = + < > # ; \ "`；`PlainNameUnchanged` 是反向对照（防止无谓转义导致 AD 侧多出反斜杠）—— 这条反向对照比正向那条更重要，写对了。
- **已知且已记录的取舍**：存量含特殊字符的部门，其历史 `ou_dn` 不回溯改写，需重跑部门同步（UAT-58-8）。这意味着 HYGIENE-02 的生效是**时间性的**，不是部署即生效。

⚠ **本次新增观察（review 未提）：** HYGIENE-02 只护住了 **OU 侧**，**RDN 侧没有同等处理**，而 Phase 58 把 RDN 侧变成了每 30 秒跑一次的自动路径。`extractRDNFromDN`（`ldap_client.go:430-436`）用 `dn[:strings.Index(dn, ",")]` 切分，不识别 RFC4514 转义逗号：`CN=Zhang\,San,OU=IT,...` 会切出 `CN=Zhang\`，随后被 `MoveUser` 当作 `newRelativeDN` 提交给 `ModifyDN(deleteOldRDN=true)` —— **AD 会把用户重命名为 `Zhang\` 并删掉原条目**。review 记为 M-06 并评估可触发性低（RuoYi 用户名不允许逗号），我同意严重度判断；但 Phase 58 刚刚为 OU 侧引入了转义、却让 RDN 侧留着不对称缺口，这个不一致是本 phase **新造成**的，不是继承的。

---

### SC5 — LDAP code 32 → dead + 清 `sys_user.ad_dn` + 不再重试

**VERIFIED**

- 三条终止路径各自落库且**不烧 attempts**（传 `row.Attempts` 原值）：`handlePushError:539` `ErrADTargetNotFound` 哨兵、`:544` `ldap.IsErrorWithCode(…, LDAPResultNoSuchObject)`（`%w` 包装可穿透）+ `clearUserADRefs` 清 `ad_dn/ad_ou_dn/ad_synced_at` 三列、`drainRow:394-398` 的 `DNExists=false` 后重读判空（因为 `syncUserAttributes` 清完 `ad_dn` 是 **return nil** 不报错，不重读就会把「AD 端不存在」误报成 pushed）。
- OU 路径同源：`MoveUser` 的 code 32 被 `%w` 包住后由同一个 `handlePushError` 分支接住（`TestPhase58_Drain_MoveCode32GoesDead` 真跑通了，不是假绿）。
- 终态不覆盖不变量：`compareAndMoveOU` 写完 dead 返回 `(true, nil)`，`drainRow:413` 见到 `terminal` 立即 return，绝不落到 `markPushed`。**没有覆盖漏洞**。
- 「不再重试」口径正确：第二轮**会**被拾取（终态行被打回 pending 的场景），但读到 `AdDn` 已空 → 归入 `no-ad-dn` deadRows → 全程零 AD 写。`TestPhase58_EndToEnd_Code32ThenNoRetry` 的三重断言（`updateUserAttrCalls` 停在 1、`moveUserCalls==0`、`dnExistsCalls` 不前进）能抓住「第二轮真去 AD 试写」。

⚠ **H-04 与 QUEUE-05 的相互约束已被反向断言锁住**（这一点做得好）：`TestPhase58_Failover_AllDialFailureWrapsPoolSentinel:679-686` 明确断言「拨通但业务全失败」时 **不得** 携带任何池侧哨兵 —— 否则 code 32 会被误判为池侧故障、绕过 `handlePushError` 的不可恢复分支，直接破坏 QUEUE-05。这是本次我在代码里找到的最有价值的一条防御性断言。

⚠ 一处小的不对称：`handlePushError:550` 只判 `ErrAllAccountsUnavailable`，**不判** `ErrAllAccountsDialFailed`。当前不可达（`executeGroup` 的闭包恒返回 nil，`drainRow` 的行级错误不上抛），但两个哨兵在 `executeGroup:283` 是同款处理的、在 `handlePushError` 只处理一个 —— 将来若改动让 failover 级错误冒泡到这里，dial-failed 会烧 attempts。

---

### SC6 — 两条遗留直推路径均已处置

**VERIFIED**

- `user_handler.go`：`buildADSyncMap` / `adSyncInFlight` / `SyncUserUpdateToAD` **0** 命中。残留的 `userADSyncService` 字段（1 处）被 `SyncManagers` 复用，与「编辑后直推」是两个用途，保留正确。
- `user_import_handler.go`：`triggerADSyncAfterImport` / `BatchSyncUsersToAD` **0** 命中；替代物 `enqueueImportUsersToAD:128` 在**导入事务外**自开短事务，先 `Pluck` 出 `ad_dn` 非空的用户 ID 再 `EnqueueUpdateBatch`，入队失败仅记 ErrorLog 不影响导入结果。
- 反向核对（SC6 明令「不接受两者并存且都可能触发」）：全仓 `SyncUserUpdateToAD` / `BatchSyncUsersToAD` **无任何非测试调用方**（两处死代码 + 注释自述）。没有引入任何互斥开关。**终态是「都删」，与 SC6 允许的两种终态之一吻合。**
- 入队侧 `ad_dn` 非空预过滤落在唯一调用方而非 `EnqueueUpdateBatch` 内部 —— 功能等价，但契约与实现位置不一致（review M-05），下一个调用方容易踩空。

---

## 2. 审计否决项 —— 本次独立复核

我按代码逐行核对，**不采信 58-VERIFICATION.md 的自证，也不采信 58-REVIEW.md 的裁定**。全部**未命中**：

| 否决项 | 裁定 | 本人独立行证据 |
|--------|------|----------------|
| **禁时间戳仲裁** | ✅ 未命中 | `ad_push_queue_drain_service.go` 全文对时间戳的引用只有 5 处：`:176` `Order("created_at ASC")`（**批内排序**，决定先后，不决定某行推不推）、`:302/:329/:348` `"updated_at": now`（写审计列）、`:566` `ad_synced_at: nil`（**清空**）。`compareAndMoveOU:450-512` 的决策输入只有 `user.DeptID` / `user.AdOuDn` / `user.AdDn` 与 mapper 的 `targetOUDN`；`markFailed:318` 的判死依据是 `attempts` 计数。**无任何时间戳参与推/跳决策。** |
| **禁新旧值比较** | ✅ 未命中 | `drainRow:385` `buildDrainUpdateReq(user)` 只消费重读值，全函数无旧值变量；`compareAndMoveOU:470` 判据是 `NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN)`（**当前值 vs 目标值**）。`user_ad_sync_service.go:80` 的 `newDeptIDStr != currentDeptID` 是唯一的新旧值比较，**但它在一个已无生产调用方的死方法里**（`SyncUserUpdateToAD`），不构成推送路径命中（review L-07 建议加「禁重新接线」显式标记，未做——建议采纳，属低成本防回归）。 |
| **禁 `sys_job` 注册** | ✅ 未命中 | `internal/scheduler/ad_sync_tasks.go` 内 `sys_job` 仅 2 处，全是注释；`internal/core/db/migrations/*.go` 中 `ad_push_queue` 与 `sys_job` 交叉命中 **0**；migration_216 只写 `sys_config` 一行；排水器只经 `s.cron.AddJob`（robfig in-process）。 |
| **禁 JSON 快照**（次级） | ✅ 未命中 | `models/ad_push_queue.go:66-81` 队列表 11 个字段全是标量（id / entity_type / entity_id / action_type / status / attempts / next_retry_at / last_error / created_at / updated_at / pushed_at / failed_at），**无 payload / snapshot / data 列**；文件里的 `json:` tag 是 HTTP 响应序列化，与存储快照无关。 |
| **禁删 AD 账号**（次级） | ✅ 未命中 | `internal/services/addomain/` 全包**无 `DeleteUser`**；唯一的 `conn.Del` 在 `ldap_client.go:402` 的 **`DeleteGroup`**，与用户无关。disable 路径走 `DisableUser` → `setUserAccountControl(uacDisabledAccount)`（UAC 514 Replace，幂等可重放）。 |

**审计否决项裁定：五条全部未命中。执行者与 review 的结论在本项上一致，且我用独立 grep 复现。**

---

## 3. 人工不可机验的部分（不洗白）

以下**没有任何自动化证据**，我不把它们计入 VERIFIED：

| 项 | 为何机验不了 | 对应 SC 的削弱 |
|----|-------------|---------------|
| UAT-58-1 真 AD 上 ModifyDN 真生效 + 二次排水不重复移动 | mock 证不了 LDAP wire 行为 | **SC2 的核心语义（compare-and-move 真的把对象搬对了）目前只有 mock 证据** |
| UAT-58-2 熔断 30min 内的退避节奏 | 需人为触发熔断 + 真实时钟跨度 | SC1 的退避契约 |
| UAT-58-3 `drain_interval_seconds` 改后重启生效 | 需重启进程 | SC1/D-05 |
| UAT-58-4 多 AD 配置（父子域）最长后缀命中 | 需第二个域（单域环境 N/A） | **SC3 的路由在多域下未验证** |
| UAT-58-5 code 32 真实触发 | 需真域控删对象 | SC5 |

另有 4 项「用户必须知晓」的后果：UAT-58-6（部门同步从未跑过 → `update_user` 批量转 dead）、UAT-58-7（编辑即续命）、UAT-58-8（含特殊字符的历史部门需重跑部门同步）、UAT-58-9（H-05 残余）。

**其中 UAT-58-6 是最需要提前告知的**：如果目标环境从未跑过部门同步（`sys_dept_ou_mapping` 为空），SC6 删掉导入直推之后，导入更新的每一个存量 AD 用户都会在排水时被判 `unrecoverable:no-ou-mapping` 转 **dead 终态**。这是决策 Q2 的刻意后果、不是 bug，但它意味着 **SC6 的价值兑现（导入推送生效）以「目标环境已跑过部门同步」为前提**，而这个前提目前**没有任何机器检查**。建议在部署前置清单里加一条 `SELECT count(*) FROM sys_dept_ou_mapping`。

---

## 4. H-03 的现状：修复前定级已过期，结论要分两半说

review 的 H-03 成立于修复前，但 **H-04 的修复改变了它的结论**，而执行者的报告（`58-VERIFICATION.md:33`，提交 `550f35e3` 早于 `18d6421e`）没有回头更新代码注释，也没有在报告里说明这一点。

**探针 A 实测（跑完即删）**，池侧失败（全部拨号失败，`poolExhausted=true`）路径下 `markFailed` 的退避档位：

```
PROBE startAttempts=0 -> status=failed attempts=1 backoff=1m0s
PROBE startAttempts=3 -> status=failed attempts=4 backoff=8m0s
PROBE startAttempts=4 -> status=failed attempts=5 backoff=16m0s
PROBE startAttempts=5 -> status=failed attempts=6 backoff=30m0s
PROBE startAttempts=6 -> status=failed attempts=7 backoff=30m0s
```

**判定，分两半：**

| 半边 | 状态 | 依据 |
|------|------|------|
| **池侧路径（熔断 / 全部拨号失败）** | **H-03 已 moot —— 16min/30min 两档在生产中可达且正是 D-05 想要的** | `markFailed` 的判死条件带 `!poolExhausted` 短路，`poolExhausted=true` 时 `attempts` 无上限累加，`backoffDuration(attempts)` 随之爬到 16min 与 30min 封顶。**D-05 的立论「退避窗桥接账号池 30min 熔断周期」在池侧路径上现在成立了**（而且是 H-04 修复带来的副产品，不是设计意图）。 |
| **业务失败路径** | **H-03 仍然为真** | 探针确认：attempts=3 → 4 时仍是 `failed`，第 5 次才 dead，实际重试窗 = 60s+2m+4m+8m = **15min**，非 D-05 声明的 ~61min。 |
| **代码注释** | **现在是错的** | `ad_push_queue_drain_service.go:47-54` 写「markFailed 在 newAttempts >= maxDrainAttempts(=5) 时直接转 dead，故本函数**只会被以 1/2/3/4 调用** —— 16min 与 30min 封顶两档在生产中永不执行」。`18d6421e` 之后这句话对池侧路径不再成立。**一个写给未来维护者的错误不变量，会诱导下一个人去「修」一个已经正确的行为。** |

**是否动摇任何 SC：不动摇。** D-05 的退避曲线写在 CONTEXT 里而非 ROADMAP SC 里，SC1 只要求「失败行按退避重试，达上限转终态且可查询」——这条在两条路径上都成立。H-03 是**实现与决策表述不一致**的问题，不是 SC 未达成。但代码注释必须改（成本：一段注释）。

---

## 5. H-05 残余风险：目标「达成」还是「仅可见」

**判定：SC2/SC4/SC5 不受影响；SC1 与「队列被可靠排水」的 phase goal 达成到「可见」为止，「自愈」未达成。**

逐层拆：

1. **未修复前的危害是不可逆的资产失联**（被 `pushed` 掩盖 → 下次编辑 code 32 → 清空三列 → 永久丢绑定）。这一点 `3f55b7f7` 修好了：`compareAndMoveOU:507` `return false, fmt.Errorf(...)` 上抛 → `handlePushError` → `markFailed` 退避，`last_error` 以 `retry:move_ou:` 开头暴露真因；`TestPhase58_Drain_MoveWriteBackFailureIsNotPushed:1137` 是**真注入**（用 GORM `Before("gorm:update")` + `AddError` 让 SQL **根本不发出**，而不是用 After 回调做成「假失败」），并反向断言本地 `ad_dn` 确实没被改写。这个用例是实的。
2. **未自愈的部分是真实的**，且我认为「MoveUser 重放不安全」的独立判定成立：重试时行重读到**陈旧** `ad_dn` → `syncUserAttributes` 的 `DNExists(陈旧 DN)` 返回 false → 清三列 → return nil → `drainRow` 重读判空 → dead。**回写失败后 `dead` 是当前的、也是合理的终态**——系统说的是「我无法证明 AD 绑定还在」，这比谎称 `pushed` 诚实。
3. **代价**：一次瞬时 DB 写失败 = 一个用户与 AD 的绑定需要**人工重建**（UAT-58-9 已如实登记为「必须知晓」）。

所以我的措辞是：**目标达成到「可见」为止，不达成到「自愈」为止。** 这不是 SC 的失败（H-05 不在任何一条 SC 的文字里），但它意味着 phase goal 里「可靠」一词有一个已知例外。而这个例外恰好与 UAT-58-1 叠加：**如果真域控上 ModifyDN 成功、回写失败，UAT 会以「用户绑定丢失」的形式暴露出来**。

---

## 6. 本次新发现（review 与执行者均未提）

### MISSED-01 — H-03 的代码不变量在 H-04 修复后已过期，是一条**主动误导**的注释
`ad_push_queue_drain_service.go:47-54`（详见 §4）。探针 A 证明池侧路径下 `backoffDuration` 会被以 5/6/7… 调用。「只会被以 1/2/3/4 调用」这句话现在是假的，且它的语气（「永不执行」）足以让下一个维护者去删档位或改 `maxDrainAttempts`。
**建议**：改为「业务失败路径只到 8min（实际窗口 15min）；池侧路径 `poolExhausted` 短路判死，attempts 可越过 5，故 16min/30min 两档在池故障期可达」。

### MISSED-02 — 池侧退避会「预支」attempts，导致长时宕机后**第一次业务失败即终态**（新引入的缺陷）
**探针 B 实测（跑完即删）：**
```
PROBE after-outage status=failed attempts=12                      ← 12 轮全部拨号失败（~2h）
PROBE first-business-failure-after-outage status=dead attempts=13  ← AD 恢复后第一次业务失败
```
**机制**：`markFailed` 的 `poolExhausted` 短路只跳过判死，**`attempts: newAttempts` 仍然照写**（`:322-330`）。于是基础设施故障与业务失败共用同一个计数器。
**失败场景**：域控维护窗口 2 小时 → 整个批次 `attempts` 升到 12+ → 维护结束后队列里**每一行都只剩一次错误的容错额度** → 用户改昵称时 AD 返回一次瞬时错误（LDAP 忙、并发上限）→ 行立刻 `dead` 终态。dead 行退出活跃谓词与唯一索引，虽然下一次编辑会插入新行而复活，但**运维看到的是「AD 已经恢复了，队列却在大批变 dead」**，且 `last_error` 里没有任何线索表明这行的历史失败全是池侧故障。
**为什么重要**：H-03/H-04 修的正是「基础设施故障不该杀死业务推送」这一类问题；H-04 修好了「不死」，但没修「不该计数」，等于把同一个问题的另一半放回去了。
**最小修法**：`poolExhausted` 分支不递增 `attempts`（只排退避），或加一列 `pool_attempts` 分离两个计数器。前者一行改动。
**注**：现有 5 条 Phase 58 退避/上限用例**全都测不出这个问题**——它们要么用纯函数断言（`BackoffDuration_Curve`，H-03 已指出是盲区），要么只跑一轮。**没有任何一条测试覆盖「池侧失败与业务失败混合」序列。**

### MISSED-03 — `58-VERIFICATION.md:195` 的端到端幂等叙述与代码不符（review 第 7 节第 3 条已指出，执行者未修，我独立确认仍在 HEAD）
声称第二轮走生产入队路径 / D-06 合并子句把 `pushed` 行打回 `pending`；实际是裸 SQL `UPDATE`。且所述合并语义**物理上不可能**（`pushed` 行不在 `uq_ad_push_queue_active` 内）。这是本次唯一一处执行者的证据叙述失实。

### MISSED-04 — HYGIENE-02 引入的新不对称：OU 侧有 RFC4514 转义，RDN 侧没有，且 RDN 侧现在跑在每 30 秒的自动路径上
review 的 M-06 记录了 `extractRDNFromDN` 截断问题，但把它归为「既有代码、本 phase 未改动」的 MEDIUM。**我补一条本 phase 特有的归因**：Phase 58 刚为 OU DN 引入转义 helper，却没给 RDN 侧同等处理，并把 `MoveUser` 变成了自动周期路径 —— 不一致是本 phase 新造成的。可触发性我同意很低（RuoYi 用户名不允许逗号），但修法便宜（按未转义逗号切分，或直接 `ldap.ParseDN`）。**建议 Phase 62 卫生收尾时一并做。**

### 已知但未修（review 已列，非本次新发现，仅确认仍然成立）
- **M-02 清空字段无法传播但行标记 pushed**：昵称改成空串 → `pushed`，AD 上仍是旧昵称，且 Phase 59 的守卫会把「无 pending/failed 行」读成「系统一致」。**这一条会直接削弱 phase goal「可靠排水」的语义**，且被 UAT 清单漏掉了（不在 9 项之列）。
- **M-03 `markPushed/markFailed/markDead` 的返回错误被丢弃且零日志**（`:154/:156/:381/:416`）。终态写入失败 = 行静默留在 pending，排障只能靠间接现象。SC1 的「可通过查询确认」在这条上是有洞的。
- **M-05 dead 行无界增长 + 入队侧 `ad_dn` 预过滤位置与 D-01 契约不符**。
- **L-02 `executeGroup:251` 的 `First` 无 `status` 条件**：配置被软删时整组永久打死。
- **L-07 建议给 `user_ad_sync_service.go:80` 的新旧值比较加「禁重新接线」标记，未做。**

---

## 7. 总判定

| SC | 判定 | 确认方式 | 一句话 |
|----|------|---------|--------|
| SC1 | **VERIFIED**（带告警） | 执行 + 阅读 | in-process cron 单实例接线、无 sys_job、两类动作行、退避与终态全部落地；**但「5 次业务失败」的容错额度会被基础设施故障预支**（MISSED-02），且「dept 推送」是对 `update_user` 行的解读而非 dept 实体行 |
| SC2 | **VERIFIED** | 执行 + 阅读 + 测试质量复核 | 归一化比较与「真会移动」的腿互相咬合，不会出现「删掉生产逻辑测试仍绿」；**真域控 ModifyDN 有效性待 UAT-58-1** |
| SC3 | **VERIFIED** | 执行 + 阅读 | 路由走最长后缀唯一匹配、逗号边界与歧义都有反例；双份 helper 归一且未偷合并两个语义不同的入口；**多域真实命中待 UAT-58-4** |
| SC4 | **VERIFIED**（限新写入） | 执行 + 阅读 | 唯一拼接点唯一写者已收口到 `BuildOUDN`，反向对照测试到位；**存量不回溯 + RDN 侧不对称**（MISSED-04） |
| SC5 | **VERIFIED** | 执行 + 阅读 | 三条终止路径各自落库不烧 attempts、终态不覆盖不变量成立、「不再重试」口径正确；**H-04 的反向断言正确护住了 QUEUE-05** |
| SC6 | **VERIFIED** | 执行 + 阅读 | 两条直推路径确实删除且无任何生产调用方，替代的批量入队已接线；**价值兑现以「目标环境已跑过部门同步」为未验证前提**（UAT-58-6） |

### 总体判定：**ACHIEVED WITH CONDITIONS**

Phase 58 的目标——**队列被可靠排水：按 `ad_dn` 路由正确 AD 配置、compare-and-move 幂等移动 OU、失败退避重试至终态可查、code 32 转终态不死循环**——在代码层面**是达成的**。五条审计否决项全部未命中。7 个 plan 的产物与 SC 一一对应，没有发现「任务完成但目标未达」的形态。

**条件（不是打折，是「达成到什么程度」的准确表述）：**

1. **9 项人工 UAT 未完成，其中 5 项是唯一能证伪 SC 的证据**。最关键的两项：UAT-58-1（真域控上 compare-and-move 是否真的把对象搬对且二次不重搬 —— mock 只能证「我们调了 MoveUser 且参数正确」，证不了 AD 是否接受）、UAT-58-4（多域下 longest-suffix 是否真的命中正确域）。**在 UAT-58-1 与 UAT-58-5 回填之前，SC2 与 SC5 应视为「代码路径正确、行为未证」**。
2. **SC6 的价值兑现有未验证前置条件**：目标环境必须已跑过部门同步，否则导入更新会以 `unrecoverable:no-ou-mapping` 批量转 dead。建议加入部署前置检查 `SELECT count(*) FROM sys_dept_ou_mapping WHERE sync_enabled`。
3. **H-05 的残余风险仍在**：OU 移动成功但本地回写失败 → 该用户 AD 绑定需人工重建。目标达成到「可见」为止，不达成到「自愈」为止。
4. **H-03 结论已因 H-04 而分裂**：池侧路径 moot（16min/30min 可达，D-05 桥接熔断的立论反而成立了）、业务失败路径仍为真（实际 15min vs 声明 ~61min）、代码不变量注释已过期（MISSED-01，必须改）。
5. **新增一条应在本 phase 收尾修掉的缺陷**：MISSED-02（池侧退避预支 attempts → 长时宕机后一次业务失败即终态）。它不违反任何一条 SC 的字面，但直接削弱「可靠」的语义，且与 H-03/H-04 修的是同一类问题。
6. **两处 observability 缺口削弱 SC1 的「可通过查询确认」**：终态写入错误被静默丢弃（M-03）；队列状态无 API/UI，运维只能查库；清空字段被静默吞掉但行报 pushed（M-02，且未进 UAT 清单）。

**我不认为本 phase 应该被判 NOT VERIFIED**：它的六个 SC 都能在代码里找到真实承载物，能找到会在 SC 为假时变红的测试，且审计否决项干净。上述条件是「达成到什么程度」的界定，不是「没做到」。

---

*本 pass 为只读验证：未修改任何生产代码。两个临时探针测试文件（`zz_goalverify_test.go`、`zz_goalverify2_test.go`）在取得输出后立即删除，`git status --porcelain internal/services/addomain/` 确认为空。*
*本文件不覆盖 `58-VERIFICATION.md`（执行者声明集），两者可并存对照。*

---

## 8. 补修处置（2026-10-19，编排者执行，闭环 §6）

用户裁定：修 MISSED-01 / 02 / 03；MISSED-04 转独立 phase。

| 编号 | 处置 | 结果 |
|------|------|------|
| **MISSED-02** | `markFailed` 的 `poolExhausted` 分支**不再自增 attempts**；attempts 自此只计业务失败 | ✅ FIXED + RED→GREEN 验证 |
| **MISSED-01** | 重写 `backoffDuration` 的不变量注释，区分业务/池侧两条路径 | ✅ FIXED |
| **MISSED-03** | 更正 `58-VERIFICATION.md` 关于端到端幂等第二轮形状的叙述 | ✅ FIXED |
| **MISSED-04** | `extractRDNFromDN` 遇 RFC4514 转义逗号截断 → `ModifyDN(deleteOldRDN=true)` 会**重命名 AD 对象** | ⏭ 转独立 phase，优先处理（数据破坏级，且 Phase 58 已把它推上 30 秒自动路径） |

### ⚠ §4 关于 H-03 的结论已因 MISSED-02 而**再次反转**（请以此节为准）

§4 判定「池侧路径 moot、16min/30min 两档重新可达」，该观测是在 **H-04 单独生效、MISSED-02 未修的半修状态**下取得的：当时池路径虽不再判死，却仍在自增 attempts，故能爬到 5+ 并用上高两档。

MISSED-02 把那次自增去掉后，**两档重新不可达**，`backoffDuration` 再次只被以 1/2/3/4 调用。代码注释已据此重写，并显式记录：

> **H-04 与 MISSED-02 是一对修复。** 单独修 H-04 会得到「不判死但持续计数」的半修状态 —— 正是 §6 MISSED-02 所描述的缺陷。任何后续改动必须同时维持两者。

由此产生的一个**净效果**优于 D-05 原设计：池侧故障期间 attempts 不变，因此**任何时长的 AD 断连都不会把行推向终态阈值**（D-05 原本只求 ~61min 容错窗）。代价是池侧故障期间退避固定 60s 不递增 —— 可接受，因为账号池全熔断时 `ListAvailable` 直接返回空，一次 LDAP 拨号都不会发生。

**仍然为真的部分：** 业务失败路径的重试窗仍是 15min 而非 D-05 声明的 ~61min。H-03 的**核心问题**（D-05 自身两条声明「5 次判 dead」与「~61min 窗口」互相矛盾）未被解决，维持 DEFERRED，等待真实熔断频率数据。

### 补修后回归

```
$ go build ./...                                        → 无输出, EXIT=0
$ go test ./internal/services/addomain/ -run TestPhase58 -count=1
    ok  … 2.270s
$ go test ./internal/services/addomain/ -count=10 -run "TestPhase58_Drain_|TestPhase58_EndToEnd"
    ok  … 2.824s          ← 重复 10 次无 flake
$ go test ./internal/services/system/ -run TestPhase57 -count=1
    ok  … 2.559s          ← Phase 57 sentinel 仍绿
$ go vet ./internal/services/addomain/ ./internal/services/system/ ./internal/scheduler/
    → 无输出, EXIT=0
```

**新用例做了 RED→GREEN 验证**（临时把 `if !poolExhausted` 改回 `if true` 后两条均 FAIL，恢复后均 PASS），排除假绿：

- `TestPhase58_Drain_PoolExhaustedIsRetryable` —— 断言从 `attempts == 1` 改为 `== 0`。旧值编码的正是缺陷本身，故此为**语义修正**而非弱化。
- `TestPhase58_Drain_PoolFailureDoesNotArmTerminalState` —— 12 轮池侧故障后 attempts 必须仍为 0，随后首次业务失败才 +1 且不判死；`last_error` 须反映业务失败而非 `pool:`。

### SC1 告警的最终状态

§1 SC1 与 §7 表格中「5 次业务失败额度会被基础设施故障预支」这条**告警已消解**。SC1 判定维持 **VERIFIED**。§7 条件 1（部署前置：目标环境须先跑过部门同步）、条件 3（H-05 残余风险）、条件 6（M-02/M-03 observability 缺口）**仍然成立**。
