---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: AD 双向同步 bug 修复 (AD Bidirectional Sync Bug Fixes)
status: executing
last_updated: "2026-10-16T11:04:55.968Z"
last_activity: 2026-10-16 -- Phase 57 planning complete
progress:
  total_phases: 8
  completed_phases: 0
  total_plans: 3
  completed_plans: 1
  percent: 33
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 Phase 57 plan 01 executed — storage layer of `ad_push_queue` landed
**Last activity**: 2026-10-16 — Phase 57 plan 01 complete（1/3）

## Current Position

Phase: 57 (AD 推送队列基建) — plan 01 done, 02/03 待执行
Plan: 01 ✅（下一：57-02 队列 service）
Status: In progress
Last activity: 2026-10-16 -- 57-01 complete（migration_215 + ADPushQueue 模型 + Wave 0 测试）
Progress: 0/6 phases [░░░░░░░░░░] 33%（1/3 plans）

## Accumulated Context

### Decisions

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
