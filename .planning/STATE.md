---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)
status: in_progress
last_updated: "2026-10-19T10:45:00.000Z"
last_activity: 2026-10-19 -- Phase 58 Wave 3（58-04）执行完成：compare-and-move 幂等 OU 移动 + F-A 三处载体回写 ad_dn + F-B 配置作用域 OU 查询 + OU 路径 code 32 终态 + 终态不被 markPushed 覆盖
progress:
  total_phases: 8
  completed_phases: 1
  total_plans: 10
  completed_plans: 7
  percent: 70
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 Phase 58 executing (wave 3/6 done — 58-01 / 58-02 / 58-03 / 58-04 完成，58-05 待执行)
**Last activity**: 2026-10-19 — execute-phase 58 plan 04：compare-and-move 幂等 OU 移动（NormalizeDN 比较 → 一致跳过 MoveUser）+ F-A 三处载体（drainer/moveUserToNewOU/moveSingleUserToOU）同时回写 ad_dn 与 ad_ou_dn + F-B 配置作用域 OU 查询（不跨域）+ OU 路径 code 32 转 dead + 终态在 markPushed 之前短路；28 个用例全绿，全仓 `go test ./...` exit 0，Phase 57 sentinel 绿

## Current Position

Phase: 58 (排水器与 compare-and-move 推送) — Executing
Plan: 58-05-PLAN.md 待执行（Wave 4：cron 接线 + sys_config 单旋钮 + 禁 sys_job）
Status: Phase 58 Wave 3 完成（58-04 已交付 compare-and-move 执行面：`compareAndMoveOU` 返回 `(terminal, err)`、目标 OU 走 `FindOUDNByDeptIDForConfig`、F-A 三处载体全部双写 ad_dn）
Last activity: 2026-10-19 -- 58-04 三个 task 落地（`44b4b958` / `f00f41b1` / `c89b4835`）
Progress: 1/6 phases [█░░░░░░░░] 17%（7/10 plans）

## Accumulated Context

### Decisions

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
