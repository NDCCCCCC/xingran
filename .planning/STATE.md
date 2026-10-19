---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)
status: in_progress
last_updated: "2026-10-19T15:20:00.000Z"
last_activity: 2026-10-19 -- Phase 58 复审与补修闭环：code-review 独立复审得 0 BLOCKER / 5 HIGH（H-01 mutex 生产失效、H-02 孪生用例、H-03 退避两档不可达、H-04 拨号全失败整批判死、H-05 回写失败被写成 pushed）→ 用户裁定修 4 留 1 → 全部修复并复跑回归；gsd-verifier 目标反向验证判 ACHIEVED WITH CONDITIONS 并新发现 MISSED-01~04 → 再裁定修前三项、MISSED-04 转独立 phase。三次审计否决项全程未命中。⚠ 下一步优先级：MISSED-04（extractRDNFromDN 转义逗号截断 → ModifyDN 会重命名 AD 对象，数据破坏级）应先于 Phase 59 处理。
progress:
  total_phases: 8
  completed_phases: 2
  total_plans: 10
  completed_plans: 10
  percent: 100
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 Phase 58 COMPLETE（含复审与补修闭环；下一步：MISSED-04 独立 phase，优先于 Phase 59 GUARD 守卫）
**Last activity**: 2026-10-19 — Phase 58 复审 + 补修闭环：code-review 判 0 BLOCKER / 5 HIGH，H-01/02/04/05 已修、H-03 DEFERRED；gsd-verifier 判 ACHIEVED WITH CONDITIONS 并新发现 MISSED-01~04，其中 MISSED-02（池侧故障自增 attempts）/01/03 已修且新用例做过 RED→GREEN，MISSED-04（`extractRDNFromDN` 转义逗号截断 → `ModifyDN` 重命名 AD 对象）转独立 phase。部署前置：目标环境须先跑过部门同步，否则 `update_user` 会以 `unrecoverable:no-ou-mapping` 批量转 dead

## Current Position

Phase: 58 (排水器与 compare-and-move 推送) — **COMPLETE**（7/7 plans）
Plan: 全部完成。下一步 = `/gsd:plan-phase 59`（GUARD-01 信号守卫 + GUARD-02 对账 VIEW）
Status: Phase 58 全量交付并收口。全量回归 `go test ./... -count=1` 退出码 0（78 包 ok）；五包 `go vet` 无输出；Phase 57 sentinel 41 PASS 零回归；65 个 `TestPhase58_*` 全 PASS
Last activity: 2026-10-19 -- 58-07 四个提交（`1d090937` / `f8f88d4e` / `23cc1308` / `30af6251` / `f89dbb57`）
Progress: 2/6 phases [██████████] 100%（10/10 plans）

## Accumulated Context

### 58-07 执行期决策（2026-10-19，Wave 6 收口）

- **零生产代码改动**：正常路径下 58-07 只**消费** 58-01~58-06 的产物写收尾用例与核对。
  唯一例外是计划内的失效注释清理（`user_handler_test.go:42`）。**六条 SC 全部 VERIFIED**，
  每条均附「命令 + 实际输出」，无一条出现「已验证但无输出」。
- **回归期间抓到的 latent flake（根因在用例写法，非生产代码）**：
  `TestPhase58_EndToEnd_RepeatDrainIdempotent` 首版用「再入队」制造第二轮 ——
  `pushed` 行已退出 `uq_ad_push_queue_active` 部分唯一索引（谓词 `status IN ('pending','failed')`），
  第二次 INSERT **不冲突而是新增并行行**（Phase 57 D-06 的**正确**设计：终态行必须退出索引，
  否则同实体下一次编辑撞 42P10）。这让「同一行被重复排水」的前提失效，且 `First` 无 `ORDER BY`
  在两行间不确定 ⇒ 随机通过/随机失败。**修复**：改用 plan 原意的行重置 + 新增
  `countQueueRowsForEntity` 把「同一行」变可证伪断言。**断言未弱化**，`-count=10` 确定性通过。
- **F-13 归档核对结论（留档不修，未改代码）**：plan 指向的 `oldUserSnapshot` 不在 enqueuer 内，
  实际在 `internal/services/system/user_service.go:163`（`oldUserSnapshot := user`，值拷贝），
  写路径是 `:214` 的 F-06 selective `Updates(updates)`（显式 map，nil 不进 map），
  enqueuer/drainer 侧 `Save` 出现 **0** 次。⚠ 严格说值拷贝是**浅拷贝**（共享 `*string` 指针），
  使别名风险不成立的前提是「闭包内不原地解引用改写」—— selective `Updates` 恰好满足。
- **两个既有不稳定项如实披露，均非本 phase 引入**（用 `git worktree` 在基线 commit `5aa9fded`
  对照归因，**全程未用 `git stash`**）：
  ① `TestInit78_Close_PartialInit` 偶发超 30s 守卫（`internal/core`，Phase 78）；
  ② `TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage` 仅在 `-count=5` 失败
  （Plan 05 的 `cache=shared` DSN 跨轮次存活）。两者 `-count=1` 全绿，**本 plan 不修**。
- **给 Phase 59 GUARD-01 的硬约束**：6 类 `dead` 行都**退出**活跃谓词（部分唯一索引），
  但仍在表里可查。守卫若靠「活跃行数为 0」推断一致性，会把「系统认为推过了、AD 侧其实没有」
  当成一致。**必须单独把 `dead` 行作为异常信号读出**（前缀表见 58-07-SUMMARY.md）。
- **给部署/运维的两条强制告知**：① 部门同步从未跑过（`sys_dept_ou_mapping` 为空）时，
  `update_user` 行会**批量**转 `dead`（`unrecoverable:no-ou-mapping`）—— **Q2 刻意后果，
  非回归**，判别三问见 UAT-58-6；② 「编辑即续命」（每次编辑清零 `attempts`）使持续失败行
  永不终态 —— **F-11 已接受方向**，见 UAT-58-7。

## Accumulated Context

### Decisions

- 58-06 执行期决策（2026-10-19，Wave 5 SC6 + D-01 批量入队 + F-07/F-10）：
  - **SC6 是硬删除，不加互斥开关**：ROADMAP 明文拒绝「两者并存 + 互斥」路线，故 `user_handler.go` 的单用户编辑直推与 `user_import_handler.go` 的导入直推**均整块删除**，无 feature flag / 无 config toggle。SC6 的残留证据由 5 条 grep 静态证伪承担（期望值全 0）。
  - **被删注释里不写字面名**：SC6 断言是 grep == 0，注释自指会让断言假阳（58-05 同款矛盾）。注释改写为「按用户 ID 的 in-flight 去重」「四字段 map 构造方法」「已删除的导入后直推方法」，并在注释里写明**为什么**不写字面名，避免后来者"好心"补回去。
  - **`userADSyncService` 字段保留**（Q5）：grep 确认 `SyncManagers`（`user_import_handler.go:226/231`）仍用；`NewUserHandler` 可变参构造与 `user_router.go:38` 注入不动。连带 `user_handler.go` 的 `addomain` import 与 `user_import_handler.go` 的 `context` / `models` import 均**必须保留**，只删 `errors` / `time` / `sync`（逐个 `go build` 确认，不凭猜删）。
  - **`EnqueueUpdateBatch` 不做 diff，且这一差异必须可证伪**：plan 描述的 `NoDiffFiltering` 用例单独写会**不可证伪**（"无条件对所有 id 入队"的错误实现同样变绿）。改为先跑 diff 变体 `EnqueueUpdateIfChanged` 断言**不入队**、再跑批量变体断言**入队 2 行**。同理 `MergeOnSecondCall` 追加"把两行打回 failed 再入队 → status 回 pending + attempts 归零"作为「合并 ≠ 幂等跳过」的判别式。两处**提高**断言强度。
  - **`EnqueueUpdateBatch` 不做 F-01 压制**：压制只发生在 disable 打 update_user 上，update 之间无压制关系。且它**只产 `update_user` 行**，disable 仍只由 `Delete`/`BatchDelete`/`UpdateStatus` 三条 service 事务路径产生（T-58-06-04）。
  - **F-07 Pluck 位置铁律**：`tx.Model(&models.User{})` 走 GORM 默认软删作用域（自动追加 `deleted_at IS NULL`），Pluck **必须排在软删语句之前**，否则本轮刚软删的行被过滤成空 → `EnqueueDisable` 变 no-op → Phase 57 `TestPhase57_UserService_BatchDelete_EnqueuesDisable` 必红。`Unscoped()` 备选**显式禁用**：会把历史已软删、本轮未删的 id 算进 `existing`，破坏「入队数 == 软删数」的可断言性。不变量：`len(入队 disable 行) == len(existing) == 实际软删数`。
  - **`Delete`（单条）不动**：事务外 `First` 已保证 id 存在，CONTEXT D-04 只锁 BatchDelete。
  - **D-01 的事务外例外照实写注释、不重构**：入队在 `ExcelService.ImportData` 的 `Transaction` 提交**之后**（`excel_service.go:427`），是 outbox-lite「同生共死」不变量的**一处已接受例外**（进程崩溃窗口丢推送）。CONTEXT 记录为用户批准的取舍，本 plan **未**重构 ExcelService 事务去关闭该窗口。
  - **Phase 57 sentinel 实测是 41 个顶层用例，不是 plan/CONTEXT 写的 42**：把 `internal/` 回退到 pre-plan 的 `d41e16cb` 重跑同样 41 → 非回归，是文档计数口径差。58-07 勿按 42 核对。
  - **本 plan 不新标 QUEUE-04 / QUEUE-05 为 complete**：沿用 58-03/58-04/58-05 分档口径（实现已由 58-04 落地，端到端证据归 58-07 回填）；REQUIREMENTS traceability 表改为 In Progress 并注明 58-04 已交付 + 58-07 回填。

- 58-05 执行期决策（2026-10-19，Wave 4 cron 接线 + sys_config 单旋钮 + 禁 sys_job）：
  - **D-05 单旋钮仅进 sys_config 一项**：`ad_push_queue.drain_interval_seconds`（默认 30s，建议 30，域控压力大时可放宽到 300），与 `network.connection_pool.*` 同档（web 可改、改后重启生效）。
  - **迁移 216 在 PG / sqlite / BootstrapMissingTables 三路径均注册一次**（Phase 57 F-03 同款盲区：旁路曾漏建 ad_push_queue 表致运行期 ON CONFLICT 42P10 全线 500）。三处错误档位统一为 `applogger.Errorf` 非阻断；缺旋钮只让 `LoadDrainIntervalSeconds` 回落默认 30s。
  - **Seed 失败为非阻断档**（与 `ensureADPushQueue` 的阻断档区分）：缺队列表会让入队 ON CONFLICT 42P10 致业务事务回滚，缺旋钮只让频率回落默认 30s。
  - **Pool 计数基线 3 不动**（`internal/scheduler/ad_sync_tasks.go:106` 熔断恢复任务局部池 + `:125` 全局单例 + `getPool()` 惰性兜底）—— 删任一处会破坏既有路径；把断言写成 `== 1` 或 `== 2` 同样属于破坏。
  - **`s.db == nil` 守卫（仅测试构造）**：`TestADSyncScheduler_StartStop` 用 `NewADSyncScheduler(nil, 5)`，原 `Start()` 只调 `AddFunc`（无 DB 依赖）；新接线下 `LoadDrainIntervalSeconds(nil)` 会 nil 解引用 panic。在 `Start()` 内 `LoadDrainIntervalSeconds` 之前加 `if s.db != nil { ... }`（默认 `drainSpec = "*/30 * * * * *"`）。生产路径（`StartADSyncScheduler(db)` 永远非 nil）语义不变。
  - **本 plan 标记 QUEUE-03 为 complete**（cron 接线落地、禁 sys_job 静态证伪、spec 校验闭环）；QUEUE-04/QUEUE-05 沿用分档口径由 58-07 回填。
  - **规整 SC1 注释措辞**：注释本身不能与 `ad_push_queue` 同行出现 `sys_job` 字面（否则 plan 自己的 grep 证伪会因注释自指假阳）；`addomain.NewAccountPool` 字面同理不进注释（保持 grep 计数 == 3）。两处矛盾均以"plan 自己写的可证伪断言"为准裁决（与 58-02/58-03/58-04 同款处置口径）。

- 58-04 执行期决策（2026-10-19，Wave 3 compare-and-move + F-A/F-B）：
  - **F-A 同时回写 `ad_dn` 与 `ad_ou_dn`（三处载体）**：`MoveUser`（ModifyDN, deleteOldRDN=true）后 AD 侧 DN 已变；只回写 `ad_ou_dn` 会让 D-02 的下一轮按陈旧 DN 解析归属配置 → 跨域移动后误判「配置无匹配」→ 误转 dead，且属性 Modify 会打到旧 DN → 二次 code 32。落点：排水器 `compareAndMoveOU` 第 5 步 + `user_ad_sync_service.go:135`（moveUserToNewOU）+ `:328`（moveSingleUserToOU），三处均为 `Updates(map){ad_dn: extractRDNFromDN(旧 ad_dn)+","+新 OU, ad_ou_dn: 新 OU}`（`Unscoped()`，因软删用户也要能回写）。
  - **终态不被 `markPushed` 覆盖**：`compareAndMoveOU` 返回 `(terminal bool, err error)`；`no-ou-mapping` 分支内已 `markDead` 并返回 `(true, nil)`，调用方 `if terminal { return }` 在 `markPushed` **之前**短路（`ad_push_queue_drain_service.go:384-387`）。行为证据：`TestPhase58_Drain_NoOUMappingGoesDead` 断言 `status == "dead"`。
  - **有意相对 `SyncUserUpdateToAD` 收紧语义**：旧路径 OU 移动失败仅 `applogger.Errorf` 并继续推属性；排水器侧改为上抛 → 行转 `failed` + 退避（`last_error` 前缀 `retry:move_ou:`）。理由：否则「OU 永远移不动」的行会被属性成功掩盖成 `pushed`，SC2 的可观测性消失。差异写入 `compareAndMoveOU` 注释与 SUMMARY。
  - **Plan 03 的 `TestPhase58_Drain_UpdateUserSuccessWritesPushedAt` fixture 补 OU 映射铺底**：叠加 compare-and-move 后，有 `dept_id` 而无映射会按 Q2 转 dead，原 fixture 语义失效。补一行与该用户当前 `ad_ou_dn` 归一化相等的映射（目标一致 → 跳过 MoveUser → pushed），断言强度不变（不是为变绿弱化断言）。
  - **`TestPhase58_FindOUDNByDeptIDForConfig_*` 放在 `dept_ou_mapper_test.go`（不带 `skip_db_tests` tag）**：排水基座 `setupDrainTestDB` 与 `drainNow()` 都定义在带 `!skip_db_tests` 的文件里，mapper 用例引用它们会让 `-tags skip_db_tests` 构建红。改复用 `user_ou_service_test.go` 的 `createDeptOUMappingTable`（同包无 tag）。
  - **旧方法 `FindOUDNByDeptID` 保留不动**：其调用方（`moveUserToNewOU` / `BatchMoveUsersToNewOU`）本轮只修 F-A 回写、不改触发条件（不引入 `NormalizeDN` 比较），避免波及其它路径；排水器内 `FindOUDNByDeptID(ctx` 命中数为 0。
  - **本 plan 不标记 QUEUE-04 / QUEUE-05 为 complete**：沿用 58-01/58-03 分档口径（cron 接线在 58-05、端到端证据在 58-07），最终回填归 58-07。

- 58-03 执行期决策（2026-10-19，Wave 2 排水器骨架 + 状态机）：
  - **`deadRows` 元素携带 reason（`drainDeadRow`）**：plan 的 `DrainOnce` 伪码给全部 deadRows 写死 `reason='unrecoverable:no-ad-config'`，与同 plan 的孤儿用例断言 `last_error` 含 `unrecoverable:user-not-found` 冲突；改为逐行携带 reason（`user-not-found` / `no-ad-dn` / `no-ad-config` 分别落库），循环体形态不变（内嵌提升 `row.ID`/`row.Attempts`）。
  - **`drainRow` 及下游函数随 Task 1 落地**：`DrainOnce → executeGroup → drainRow` 是硬编译依赖，Task 1 的 `go build ./...` 验收在缺 drainRow 时不可能通过。Task 2 的落地物相应为 12 个用例 + `newDrainTestDrainer` + F-12/F-14。
  - **F-11 口径确认**：`attempts` = "自上次编辑以来的连续失败计数"；dead 只在 `markFailed` 单点按 `attempts+1 >= 5` 判定（源码仅 1 处）；**"编辑即续命"（持续编辑的失败行永不终态）是有意设计**，写进 `markFailed` 注释。`markDead` 的 `attempts` 为显式入参：8 处传 `row.Attempts`（不烧计数）+ 1 处传 `newAttempts`（达上限）。
  - **W-1 注释口径**：`models/ad_push_queue.go` 第 61 行 Phase 57 既有注释含 `uniqueIndex` 字面量，与 plan 的 `grep -c uniqueIndex == 0` 断言冲突（改动前即不成立）；注释改为"唯一索引 tag"（语义未变）。
  - **F-12 的 `grep "time.Now()" == 0` 断言不可满足**：plan 规定的修复即 `time.Now()` → `time.Now().UTC()`，后者是前者超串，grep 必然命中。按 plan 的规定执行修复，改用语义等价口径 `grep "time.Now()" | grep -v "\.UTC()" == 0`。
  - **本 plan 不标记 QUEUE-03 / QUEUE-05 为 complete**：二者均为跨 plan 需求（QUEUE-03 的 cron 接线在 58-05；QUEUE-05 的 OU 路径 code 32 在 58-04），沿用 58-01 分档口径，最终回填归 58-07。

- 58-02 执行期决策（2026-10-19，Wave 1 DN 单一真实源 + D-02 契约 + HYGIENE-01）：
  - **`BuildOUDN` 用字符串拼接而非 `fmt.Sprintf("OU=%s,%s", ...)`**：plan 的 action 建议格式化串，但同 task 的验收断言要求 `grep -rn 'OU=%s,%s' --include=*.go internal/` 零命中 —— 冲突时以可证伪断言为准（拼接与格式化逐字等价），`dn_util.go` 的 `fmt` import 随之移除。
  - **歧义（等长并列）的真实触发形态**：数学上两个等长但不同的字符串不可能同时是同一 DN 的逗号边界后缀，故等长并列只能是"两条启用配置归一后 base_dn 相同"（同一域重复配置）。`TestPhase58_MatchADConfigByDN_AmbiguousEqualsLength` 据此构造（`DC=corp,DC=com` / `dc=CORP,dc=COM`），断言强度未降低。
  - **`MatchADConfigByDN` 的逗号边界推论**：DN 恰好等于 base_dn 时**不算命中**（需要 `,<base>` 前缀）——刻意如此，`sys_user.ad_dn` 恒为完整用户 DN（含 `CN=`），不会退化成域根。
  - **`FirstSyncEnabledADConfigID` 不带 ORDER BY**：原 `ou_mapping_handler.go` 就是无序 `First`，收敛只消除重复实现、不改语义；加 ORDER BY 会静默改变取到的配置。
  - **注释内不出现被删函数名字面量**：沿用 58-01 F-G 的 tombstone 口径，满足 `grep -rn "getDefaultADConfigIDForDept" internal/` 零命中的验收断言。
  - **本 plan 标记 HYGIENE-01 / HYGIENE-02 为 complete**：本 plan 即为二者的全部落地范围（前缀匹配 helper 与全仓唯一拼接点）。

- 58-01 执行期决策（2026-10-19，Wave 0 测试基座）：
  - **基座文件带 `!skip_db_tests` build tag**：新文件 `internal/services/addomain/ad_push_queue_drain_test.go` 引用同包 `fakePool78`（定义于带该 tag 的 `failover_client_78_07_test.go`），不加 tag 会让 `go vet -tags skip_db_tests ./internal/services/addomain/` 报 `undefined: fakePool78`（编译面回归）。默认构建行为不变。
  - **fixture 三表零副本**：`sys_user` / `sys_dept_ou_mapping` / `sys_dept` 一律复用 `user_ou_service_test.go` 既有包级 const（`createUserTable` / `createDeptOUMappingTable` / `createDeptTable`），不建近似副本（两套副本漂移会让"环境缺列"伪装成业务 bug）。`sys_dept` 不扩列 —— `syncUserAttributes` 读部门只取 `DeptName`，GORM `SELECT *` 对缺失模型列只是零值。
  - **队列表不手写 DDL**：一律直调 `migrations.Migrate215CreateADPushQueue(db)`，三个索引由迁移建（单一 DDL 事实源，避免与模型"不声明 uniqueIndex"红线冲突）。
  - **T-58-W0-02 防御**：`newDrainTestLDAP()` 固定 `dnExistsRes=true` —— 零值 false 会让 `syncUserAttributes` 走"DNExists=false → 清空 ad_dn 后返回 nil"分支，使 code 32 用例假绿。
  - **F-G 处置**：删除 `TestUserHandler_BuildADSyncMap`，保留一条不出现被删函数名的 tombstone 注释说明理由（满足 `grep -c buildADSyncMap` == 0）；覆盖损失为零，依据 `TestPhase57_HasADFieldChanged_Boundaries` 已锁四字段 diff 契约。
  - **F-16 留档**：`TestPhase57_Migration215_BareInsertWithoutID` 加 `t.Skip` —— PG uuid NOT NULL（23502）契约需真实 PG，sqlite 无法复现，v1.22 跟进。
  - **环境**：`gsd-sdk` CLI 在本执行环境不可用（`$HOME/.claude/get-shit-done/bin/gsd-tools.cjs` 不接受 `query <verb>`），故 58-01 的 STATE/ROADMAP/VALIDATION 更新为手工编辑。
  - **本 plan 不标记 QUEUE-03/04/05 为 complete**：58-01 只交付基座，行为断言分别在 Wave 2（58-03）/ Wave 3（58-04）落地。

- Phase 57 审查修复轮决策（2026-10-16/17）：F-01 压制用既有终态 `dead` + `last_error='superseded by disable'`（不新增 `superseded` 状态，避免动 DDL/TargetWhere 两处文本），且压制逻辑在 D-02 配置门控**之前**（安全不变量）；部分索引谓词提为单一事实源 `models.ADPushQueueActivePredicate`（migration_215 与 ON CONFLICT TargetWhere 共用，结构上无法漂移）；F-03 `SKIP_AUTOMIGRATE` 旁路走 `ensureADPushQueue()` 复用 Migrate215 同一 DDL（裸 CreateTable 会缺部分唯一索引 → 42P10）；F-04 索引创建失败 PG/sqlite 双分支 `return err` 阻断启动（与 backfillSysAPIKeysKeyHash 同先例，fail-fast trade 经 verifier 复核判定正确）；F-06 `Update` 从 `tx.Save` 全字段覆盖改为显式字段 `Updates`（nil=不动、显式空串=清空部门，保住 UI 清空能力）；**GAP-01**：`UpdateStatus`（前端列表 Switch 开关，最高频停用入口）同事务入队 disable ——推翻 57-03 期的「UpdateStatus 不入队（D-04 Deferred）」记录，该记录在当时计划内成立但 F-02/GAP-01 后已失效；**GAP-02**：profile 自助路径也入队（信任边界：userID 来自会话无跨用户写面 / 不入队被 user_sync_service 无条件覆盖静默回滚且 GUARD-01 误判 / 执行面与门控同一），DI 用 `addomain.NewADPushEnqueuer()` 零参内联构造无包环
- 57-03 执行期决策：入队调用点固定形态 = 事务闭包内、`return nil` 之前、传 `tx`、失败 `return apperrors.DatabaseError(err)`（禁吞错降级）；`Update` 的 diff 旧值须在 `First` 之后、事务闭包之前做**值拷贝** `oldUserSnapshot`（`user` 是指针，闭包内重新赋值会毁掉 diff 旧值）；HYGIENE-06 实际是**三套** sqlite 环境（非计划列的两套）——`setupUserServiceTestDB`、`setupUserTestDB`(handler 层)、`setupUserCacheTestDB`(userCacheService 内嵌 userService，复用同一写路径)，GAP-02 后加第四套（profile_service_test）第五套（profile_handler_test）；Update 类 fixture 的 `sys_user` 必须补齐 `models.User` 全部列，否则写库报 `no such column` 且让回滚用例假绿；~~`UpdateStatus` 仍不入队（D-04 Deferred）~~ → GAP-01 已推翻，见上
- 57-02 执行期决策：ON CONFLICT 谓词放 `TargetWhere`（非 `Where`）+ `clause.Expr`（非 `clause.Literal`），谓词文本与 migration_215 部分索引文本级一致且无表名限定（先例 `internal/services/asset/reconciliation_detection.go:426`）；入队函数一律显式收 `tx`，禁自持 db / 禁 goroutine；`EnqueueDisable` 先 map 去重保序（PG 同语句二次命中会报 cannot affect row a second time）；测试构造 `models.User` 的 ID 须走 `BaseModel: models.BaseModel{ID: ...}`；`seedADConfigEnabled` 用 `INSERT OR REPLACE` 以支持同用例内切换 sync_enabled/status
- 57-01 执行期决策：迁移编号 215（214 被 ad_login_history 占用）；建表 DDL 双方言分方言（PG `timestamptz` / sqlite `datetime`），索引 DDL 双方言共用；ad_push_queue 进 MigrateModelList（migration_214 禁入是分区父表特例）；迁移在 PG + sqlite 双分支各注册一次，失败仅 `applogger.Errorf` 非阻断
- 方案经三份独立 subagent 审计修订并经用户拍板：时间戳仲裁弃用（GORM 自动刷 updated_at 恒失效）→ 信号守卫；完整 Outbox → ad_push_queue outbox-lite（同事务入队 + in-process cron 排水）；FieldRule 表驱动引擎砍除（代码内映射 struct）；FK 后置可选；对账视图用普通 VIEW 禁用 MV；软删恢复 = ad_dn 校验后恢复（用户选定）
- 排水必须 compare-and-move（闭包内重读 + NormalizeDN 归一化比较），禁用新旧值比较（会激活未验证的 MoveUser 重试分支）
- ErrADTargetNotFound → outbox 终态，防"清 DN→重推"死循环
- Roadmap 依赖约束（三方审计，已落图）：队列基建(57) → GUARD-01(59)；HYGIENE-01 与 QUEUE-04 同在 Phase 58 且 helper 先落地；SAFE(60)/SHADOW(61) 与 QUEUE 相互独立可并行；HYGIENE-03(62) 必须在 SAFE-01(60) 之后

### Blockers

- 无

### Todos

- ~~GSD phase 编号从 57 起~~ ✅ 已落 ROADMAP（Phases 57-62）
- ~~DB migration 编号从 200 起~~ → **修正：214 已被 ad_login_history 占用，Phase 57 = migration_215**（ad_push_queue 表），后续 phase 从 216 顺延（Phase 59 的 VIEW 预计 migration_216）
- ~~Phase plans 数量 TBD~~ ✅ Phase 57 = 3 plans，Phase 58 = 7 plans（7 waves），progress table 已回填
- ~~Phase 58 移交待决项~~ ✅ 58-CONTEXT.md D-01~D-05 已拍板并落 plan：①两条直推路径均删（D-01/D-03，导入换批量入队）②不加 ad_config_id 列、排水时最长后缀唯一匹配（D-02）③F-07~F-16 三档分档复核（D-04）
- 57-HUMAN-UAT.md 三项真环境验证（PG 合并路径 / kill -9 重启 / UAC 533）待人工执行
