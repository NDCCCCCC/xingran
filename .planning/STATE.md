---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)
status: executing
last_updated: "2026-10-17T00:00:00.000Z"
last_activity: 2026-10-17 -- Phase 57 verified（human_needed: 3 项真环境人工验证留档 UAT）+ marked complete
progress:
  total_phases: 8
  completed_phases: 1
  total_plans: 3
  completed_plans: 3
  percent: 17
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 Phase 57 complete (3/3) — queue storage + enqueue service + 4 条写路径同事务接线（含 GAP-01 UpdateStatus / GAP-02 profile）
**Last activity**: 2026-10-17 — code review F-01~F-06 修复 + verifier GAP-01/02/03 闭环 + VERIFICATION human_needed（3 项 UAT 留档）

## Current Position

Phase: 58 (排水器与 compare-and-move 推送) — 待规划
Plan: Phase 57 ✅ 全部关闭（3/3 plans + 2 轮审查修复 + 验证收口；下一：/gsd:discuss-phase 58 或 /gsd:plan-phase 58）
Status: Phase 57 Complete
Last activity: 2026-10-17 -- VERIFICATION.md 判定 human_needed（代码侧全 VERIFIED；PG 实机合并/kill -9 重启/UAC 533 三项入 57-HUMAN-UAT.md）
Progress: 1/6 phases [█░░░░░░░░] 17%（3/3 plans）

## Accumulated Context

### Decisions

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
- ~~Phase plans 数量 TBD~~ ✅ Phase 57 = 3 plans，progress table 已回填
- Phase 58 移交待决项（详见 ROADMAP Phase 58 段）：①两条遗留直推路径（user_handler.go:194 + user_import_handler.go:114）与排水器双推冲突须处置 ②队列表无 ad_config_id 列的取舍 ③F-07~F-16 复核
- 57-HUMAN-UAT.md 三项真环境验证（PG 合并路径 / kill -9 重启 / UAC 533）待人工执行
