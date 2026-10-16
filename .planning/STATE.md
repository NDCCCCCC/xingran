---
gsd_state_version: 1.0
milestone: v1.21
milestone_name: AD 双向同步 bug 修复
status: planning
stopped_at: ""
last_updated: "2026-10-16T00:00:00.000Z"
last_activity: 2026-10-16 — Milestone v1.21 INITIATED (三方 subagent 审计裁决修订方案: outbox-lite 队列 + 信号守卫 + compare-and-move; 11 项 bug 全修)
progress:
  total_phases: 0
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: v1.21 milestone planning — requirements → roadmap
**Last activity**: 2026-10-16 — 前置闭环: 1641 个未提交文件分 5 组提交合入 main (ced2d75b..31c7eee8) 并推送 origin; v1.21 里程碑启动

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-10-16 — Milestone v1.21 started

## Accumulated Context

### Decisions
- 方案经三份独立 subagent 审计修订并经用户拍板：时间戳仲裁弃用（GORM 自动刷 updated_at 恒失效）→ 信号守卫；完整 Outbox → ad_push_queue outbox-lite（同事务入队 + in-process cron 排水）；FieldRule 表驱动引擎砍除（代码内映射 struct）；FK 后置可选；对账视图用普通 VIEW 禁用 MV；软删恢复 = ad_dn 校验后恢复（用户选定）
- 排水必须 compare-and-move（闭包内重读 + NormalizeDN 归一化比较），禁用新旧值比较（会激活未验证的 MoveUser 重试分支）
- ErrADTargetNotFound → outbox 终态，防"清 DN→重推"死循环

### Blockers
- 无

### Todos
- GSD phase 编号从 57 起（v1.20 止于 Phase 56）
- DB migration 编号从 200 起
