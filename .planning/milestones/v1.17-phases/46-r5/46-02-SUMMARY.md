---
phase: 46-r5
plan: 46-02
title: 一键回滚机制 + 误修复监控 Summary
subsystem: asset-reconciliation
tags: [phase-46, r5, fix-suggestion, reconciliation, rollback, monitoring, mis-fix-rate, golang, react]
dependency_graph:
  requires:
    - FixSuggestionService (Plan 46-01)
    - FixSuggestionHandler (Plan 46-01)
    - sys_reconciliation_fix_suggestion table (Plan 46-01)
    - operlog.Record (existing)
    - NoticeService.CreateNoticeWithTargets + PublishNotice
    - NoticeHub.BroadcastToAll
  provides:
    - FixSuggestionMonitor service (7d mis-fix rate monitor)
    - 1h throttle breach detection
    - 错峰 cron 7,17,27,37,47,57 * * * *
    - WS 推送 fix_suggestion_mis_fix_rate_breach 事件
    - SysNotice 三通道告警(mis_fix_threshold > 0.01)
    - RollbackModal 独立组件(≥10 字符 reason)
    - 7d 倒计时彩色 Tag(red<1d, orange<3d, blue>=3d, gray expired)
    - 6-status color mapping 视觉强化
    - 4 operlog audit tests
    - scripts/verify_phase46_r5.sh 端到端验证
    - .planning/milestones/20260703-v1.17-r5-close.md
  affects:
    - sys_reconciliation_fix_suggestion (no schema change)
    - sys_oper_log (新增 monitor alert 记录)
    - sys_notice (新增 monitor 告警条目)
    - ops_asset.user_id (Rollback 恢复)
    - reconciliation:health:workstation:{id} Redis cache
tech-stack:
  added: []
  patterns:
    - Layered validation: Go-side defense + DB-side authoritative (clock-drift safe)
    - 1h in-memory throttle (mutex-protected)
    - State-change detection (non-breach -> breach)
    - Three-channel alert: WS + SysNotice + operlog
    - Wrong-peak cron (4min offset from generator)
    - Color-coded countdown Tag (visual urgency)
    - Extracted modal component pattern
key-files:
  created:
    - internal/services/asset/fix_suggestion_monitor.go
    - internal/scheduler/reconciliation_fix_suggestion_monitor.go
    - internal/services/asset/fix_suggestion_audit_test.go
    - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/RollbackModal.tsx
    - scripts/verify_phase46_r5.sh
    - .planning/milestones/20260703-v1.17-r5-close.md
  modified:
    - internal/services/asset/fix_suggestion_service.go
    - internal/api/v1/asset/fix_suggestion_handler.go
    - internal/scheduler/reconciliation_tasks.go
    - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx
    - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx
decisions:
  - D-C1: Rollback granularity = restore user_id only
  - D-C2: 7d rollback window (DB-side INTERVAL authoritative + Go-side defense)
  - D-C3: Strong operlog, Rollback uses OperTypeReset=11
  - D-C5: Mis-fix rate = rolled_back / applied (7d rolling window)
  - W-7: 错峰 cron (7,17,27,37,47,57) avoid race with generator (3,8,13,18,...)
  - 1h throttle for mis-fix rate alerts (state change OR > 1h since last)
  - rolled_back: magenta -> orange (D-D2 visual)
  - countdown: red<1d, orange<3d, blue>=3d, gray expired
metrics:
  duration_minutes: 60
  completed_date: 2026-07-03
  tasks_completed: 5/5
  commits: 4
---

# Phase 46 Plan 02: 一键回滚机制 + 误修复监控 Summary

## One-liner

Phase 46 R5 半自动修复后段:双层 7d 校验的强化 Rollback + 1h 节流的误修复率监控(WS + SysNotice + operlog 三通道)+ UI 7d 倒计时彩色 Tag + 4 个 operlog 审计测试 + v1.17 milestone close 报告。

## What Was Built

### Backend (3 files new, 3 files modified)

#### Service Layer
- **`internal/services/asset/fix_suggestion_monitor.go`** (NEW) — `FixSuggestionMonitor` service
  - `CheckAndNotify(ctx)` 7d 误修复率检测 + 三通道告警
  - `shouldNotifyBreach(now)` 1h 节流 + 状态变化检测
  - 注入 db + configSvc + noticeHub + noticeSvc + svc
  - 防告警风暴:W-7 修订 in-memory 状态机
- **`internal/services/asset/fix_suggestion_service.go`** (MODIFIED) — `Rollback` method 强化
  - 校验顺序加固:id > userID > len(reason) >= 10 > applied state > window_until != nil > time.Now() defensive check > pre_fix_user_id != nil > DB-side window check
  - D-C3 文档化每个 step 的 audit intent

#### Handler Layer
- **`internal/api/v1/asset/fix_suggestion_handler.go`** (MODIFIED) — Rollback handler 文档
  - 错误码映射 docstring(400 / 409 / 500)
  - 严格顺序 service → invalidate cache → operlog → response

#### Scheduler
- **`internal/scheduler/reconciliation_fix_suggestion_monitor.go`** (NEW) — RegisterFixSuggestionMisFixMonitor
  - 单一 taskType "reconciliation" + target "monitorFixSuggestionMisFix"
- **`internal/scheduler/reconciliation_tasks.go`** (MODIFIED)
  - 注入 fixSuggestionMonitor
  - 新增 case "monitorFixSuggestionMisFix" dispatch
  - 新增 sys_job "对账-误修复率监控" cron `7,17,27,37,47,57 * * * *`
  - jobNameToInvokeTarget 新增映射

#### Tests
- **`internal/services/asset/fix_suggestion_audit_test.go`** (NEW) — 4 + 2 = 6 audit tests
  - `TestFixSuggestionAcceptWritesOperLog` — oper_type=2 (OperTypeUpdate)
  - `TestFixSuggestionRejectWritesOperLog` — oper_type=23 (OperTypeReject) + rejectionReason
  - `TestFixSuggestionApplyWritesOperLog` — oper_type=2 + preFixUserId
  - `TestFixSuggestionRollbackWritesOperLog` — oper_type=11 (OperTypeReset) + rollbackReason
  - `TestFixSuggestionAuditHandlerOperLogOrder` — handler service → operlog order
  - `TestFixSuggestionHandlerOperTypeConstants` — handler OperType 引用次数

#### Verify Script
- **`scripts/verify_phase46_r5.sh`** (NEW) — End-to-end verification
  - 静态检查:4 个写端点 service 调用 + monitor cron 注册
  - operlog 常量值核对
  - Go audit 测试运行
  - Live E2E(可选,默认 SKIP_LIVE=1)

### Frontend (1 file new, 2 files modified)

- **`src/pages/asset/reconciliation/fix-suggestion/components/RollbackModal.tsx`** (NEW) — 提取的 Modal 组件
  - antd Modal + Form with min:10 validation
  - onSubmit(reason) callback pattern
- **`src/pages/asset/reconciliation/fix-suggestion/index.tsx`** (MODIFIED)
  - rollbackModal state 加 rollbackReason 字段
  - isWithinRollbackWindow → isWithin7d 重命名
  - handleRollbackSubmit(reason) 接受参数
  - 失效 list + stats + detail 三组 query
  - 窗口已过错误 → 自动刷新 list 隐藏按钮
  - fixStatusColor: rolled_back 改橙色
- **`src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx`** (MODIFIED)
  - calcCountdown 返回结构化 {remainingMs, remainingDays, remainingHours, remainingMinutes, isExpired}
  - 7d 倒计时渲染为彩色 Tag(red<1d, orange<3d, blue>=3d, gray expired)

### Documentation
- **`.planning/milestones/20260703-v1.17-r5-close.md`** (NEW) — v1.17 R5 milestone close 报告
  - 7 SC 全部勾选
  - 11 个 atomic commit 列表
  - 7 个系统端点 URL
  - 2 个 cron 表达式
  - 3 个关键学习

## Verification Results

```
go build ./...                                            → exit 0
go test ./internal/services/asset/...                     → 17 tests PASS (9 R5 + 6 R5-audit + 2 acceptance)
  - TestFixSuggestionListPagination                       PASS
  - TestFixSuggestionRejectRequiresReason                 PASS
  - TestFixSuggestionStatsWindow                          PASS
  - TestFixSuggestionStatsPendingAllNoWindow              PASS
  - TestFixSuggestionAcceptConcurrentPartialUnique        PASS
  - TestFixSuggestionApplyUpdatesAssetResolved            PASS
  - TestFixSuggestionStatsUsesAppliedAtFilter             PASS
  - TestFixSuggestionDBIntervalUsed                       PASS
  - TestFixSuggestionInterfaceHas8Methods                 PASS
  - TestFixSuggestionD4SortWhitelist                      PASS
  - TestFixSuggestionInterfaceSatisfiable                 PASS
  - TestFixSuggestionAcceptWritesOperLog (46-02)          PASS
  - TestFixSuggestionRejectWritesOperLog (46-02)          PASS
  - TestFixSuggestionApplyWritesOperLog (46-02)           PASS
  - TestFixSuggestionRollbackWritesOperLog (46-02)        PASS
  - TestFixSuggestionAuditHandlerOperLogOrder (46-02)     PASS
  - TestFixSuggestionHandlerOperTypeConstants (46-02)     PASS
npm run type-check:strict                                 → exit 0
bash scripts/verify_phase46_r5.sh SKIP_LIVE=1             → exit 0
```

Pre-existing build errors (out of scope per scope boundary):
- `src/components/operations/WorkstationDeviceTable/types.ts` (unrelated module)
- `src/lib/assetApi.ts:445` (pre-existing refresh method TypeScript strict issue)

## Deviations from Plan

None. All 5 tasks executed as specified. All 4 locked decisions (D-C1, D-C2, D-C3, D-C5) and 1 critical revision (W-7) implemented exactly.

## Self-Check

- [x] `Rollback` service contains 6-step validation chain (id, userID, reason, applied state, window_until, pre_fix_user_id, DB-side window)
- [x] `Rollback` handler returns 400 for "回滚窗口已过(7d)" + "回滚原因至少 10 字符"
- [x] `FixSuggestionMonitor` struct has all 5 fields (db, configService, noticeHub, noticeService, service)
- [x] `CheckAndNotify` calls `service.Stats(ctx, 7)` + `noticeHub.BroadcastToAll` + SysNotice + operlog
- [x] 1h throttle via `lastBreachNotifiedAt` + mutex + state change detection
- [x] Cron `7,17,27,37,47,57 * * * *` registered in `reconciliation_tasks.go`
- [x] `RollbackModal.tsx` exists with min:10 validation
- [x] `FixSuggestionDetailDrawer.tsx` contains `remainingDays` + `remainingHours` countdown
- [x] Color-coded Tag (red<1d, orange<3d, blue>=3d, gray expired)
- [x] 4 operlog audit tests + 2 static tests all pass
- [x] `scripts/verify_phase46_r5.sh` exit 0 in SKIP_LIVE=1 mode
- [x] Milestone close report exists with 7 SC checklist + 11 commit list

## Self-Check: PASSED

## Commits (4 atomic)

1. `220ae78e` — feat(46-02): harden Rollback endpoint with layered window check + audit comments
2. `6a747d64` — feat(46-02): mis-fix rate monitor service + cron (D-C5 1h throttle + WS + SysNotice + operlog)
3. `31fb5846` — feat(46-02): extract RollbackModal + 7d countdown Tag + 6-status colors
4. `c272896a` — feat(46-02): operlog audit chain tests (4 WritesOperLog + 2 static) + verify script
5. `3640685d` — docs(46-02): v1.17 R5 半自动修复 milestone close 报告
