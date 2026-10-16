---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)
status: executing
last_updated: "2026-10-16T19:58:00.000Z"
last_activity: 2026-10-16 -- Phase 57 plan 02 complete（2/3）
progress:
  total_phases: 8
  completed_phases: 0
  total_plans: 3
  completed_plans: 3
  percent: 100
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 Phase 57 complete (3/3) — queue storage + enqueue service + user_service transaction wiring
**Last activity**: 2026-10-16 — Phase 57 plan 03 complete（3/3，Phase 57 收官）

## Current Position

Phase: 58 (排水器与 compare-and-move 推送) — 待规划
Plan: 57-03 ✅（Phase 57 全部 3 个 plan 完成；下一：/gsd:plan-phase 58）
Status: Phase 57 Complete
Last activity: 2026-10-16 -- 57-03 complete（user_service 三处事务内入队接线 + 三套 sqlite 环境补表 + TestPhase57_ 20 例全绿）
Progress: 1/6 phases [█░░░░░░░░] 17%（3/3 plans）

## Accumulated Context

### Decisions

- 57-03 执行期决策：入队调用点固定形态 = 事务闭包内、`return nil` 之前、传 `tx`、失败 `return apperrors.DatabaseError(err)`（禁吞错降级）；`Update` 的 diff 旧值须在 `First` 之后、事务闭包之前做**值拷贝** `oldUserSnapshot`（`user` 是指针，闭包内重新赋值会毁掉 diff 旧值）；HYGIENE-06 实际是**三套** sqlite 环境（非计划列的两套）——`setupUserServiceTestDB`、`setupUserTestDB`(handler 层)、`setupUserCacheTestDB`(userCacheService 内嵌 userService，复用同一写路径)；Update 类 fixture 的 `sys_user` 必须补齐 `models.User` 全部列，否则 `tx.Save` 报 `no such column` 且让回滚用例假绿；`UpdateStatus` 仍不入队（D-04 Deferred），`user_handler.go` 旧推送路径保留至 Phase 58
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
- Phase plans 数量 TBD（plan-phase 时确定），roadmap progress table 待回填
