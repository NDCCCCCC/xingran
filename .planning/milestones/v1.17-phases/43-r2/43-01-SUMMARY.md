---
phase: 43-r2
plan: 01
subsystem: asset-reconciliation
tags: [feat, cron, workorder, migration, R2]
dependency_graph:
  requires: [Migrate168 (sys_data_reconciliation.workorder_id), BaseService.Create (Phase 23), ConfigService.GetByKey (Phase 39)]
  provides: [ReconciliationWorkorderTemplate 5 B-F, ReconciliationWorkorderService.CreateWorkorderFromException, reconciliation:createWorkorderCritical cron @every 2m, reconciliation:createWorkorderHigh cron @every 5m, sys_config:assignee_role_map + sla_minutes_by_severity, 6 category description backfill]
  affects: [internal/scheduler (R2 cron registration), internal/services/workorder (template), internal/services/asset (workorder service), internal/core/db (Migrate171+172 registration)]
tech_stack:
  added: []
  patterns: [count-then-insert sys_config seed (migration_165/169 style), partial UPDATE idempotency for description backfill, 12-step CreateWorkorderFromException flow, logrus.Errorf-only failure path (no SysNotice), workorder.BaseService.Create as SYSTEM submitter for T-43-01 mitigation]
key_files:
  created:
    - internal/services/workorder/reconciliation_template.go (5 templates B/C/D/E/F + GetTemplate + AllTemplates)
    - internal/services/asset/reconciliation_workorder.go (ReconciliationWorkorderService + CreateWorkorderFromException 12-step)
    - internal/core/db/migrations/migration_171_reconciliation_workorder_assignee_role.go (2 sys_config seeds)
    - internal/core/db/migrations/migration_172_reconciliation_workorder_templates_seed.go (6 category description backfill)
  modified:
    - internal/scheduler/reconciliation_tasks.go (RegisterTask switch add 2 case + createWorkorderBySeverity helper + reconJobs add 2 entries + jobNameToInvokeTarget add 2 mapping)
    - internal/core/db/database.go (append Migrate171 + Migrate172 registration lines after Migrate170)
decisions:
  - "DefaultPriority: B/C=High(2), D/F=Medium(1), E=Low(0) — E 类三方不一致通常是历史遗留,优先级最低"
  - "submitterID='SYSTEM' for all R2 workorders — T-43-01 mitigation, workorder.status=Pending 等待人工接管,避免双重系统行为"
  - "Soft failure path: missing config / unparseable JSON / no user with role => assigneeID=nil, workorder still created — 避免单条 cron 失败级联"
  - "severityToSLAMinutes hardcoded (30/240/1440/10080) matches sys_config.sla_minutes_by_severity default — service does NOT read sys_config for SLA, only writes description text"
  - "reconJobs LIMIT 50(critical) / LIMIT 30(high) per cron tick — 避免单次 cron 周期堆积,预留 1-2min 给单条转单流程"
  - "Migration 172 idempotent UPDATE: WHERE description IS NULL OR = R1 default value — 避免覆盖 admin 手动修改"
  - "6 category description 与 reconciliation_template.go DescriptionLines 内容完全一致 — 后续修改需同步两处"
  - "Migration 171 role_id=1/2/3 是占位 — 运维按实际 sys_role.id 修改,默认无用户匹配时 assigneeID=nil"
metrics:
  duration: ~12 min
  completed_date: 2026-06-28
  tasks: 2
  files_created: 4
  files_modified: 2
  commits: 2
  lines_added: ~702
---

# Phase 43 Plan 01: R2 critical/high 自动转工单 + 6 类工单模板 + sys_config role 映射 Summary

## One-liner
critical/high 异常自动转工单(2 个新 cron @every 2m/5m)+ 5 类 B-F 工单模板差异化(D-A2-04)+ type→role 映射从 sys_config JSONB 读取(D-A2-02)+ SLA 按 severity 分级(D-A2-03),为 Phase 43 R2 闭环的核心第一步,符合 WORKORDER-01 ROADMAP SC 1+2+3。

## What Built

### Task 1: 模板 + 转单 service
- **`internal/services/workorder/reconciliation_template.go`** (190 lines)
  - 5 个 B-F 模板常量:`TemplateBType`/`TemplateCType`/`TemplateDType`/`TemplateEType`/`TemplateFType`
  - 每个含:ConflictType / TypePrefix / AssigneeRoleKey (asset_owner / ops_owner / responsible_owner) / 5 句中文 DescriptionLines / DefaultPriority (B/C=High(2), D/F=Medium(1), E=Low(0))
  - `reconciliationTemplatesByType` map + `GetTemplate(conflictType)` O(1) 查询(返回 nil 表示 A 类或未知)
  - `AllTemplates()` 给 admin UI 用
- **`internal/services/asset/reconciliation_workorder.go`** (210 lines)
  - `ReconciliationWorkorderService` + `NewReconciliationWorkorderService(db)`
  - `CreateWorkorderFromException(ctx, exceptionID)` 12 步流程:SELECT 异常 → SELECT category → 取模板 → 读 sys_config → JSONB 反查 role_id → 取 sys_user → 解析 raw_snapshot 取 asset_code → D-A2-01 标题模板 → 拼接 description(含 SLA) → `workorder.NewBaseService(s.db).Create(ctx, req, "SYSTEM")` → UPDATE workorder_id → return
  - `severityToSLAMinutes(severity)` 30/240/1440/10080 for critical/high/medium/low
  - `AssigneeRoleMap` type→role 映射表
  - 失败仅 `logrus.Errorf`,不写 SysNotice(D-A1-03)

### Task 2: cron 续写 + 2 migration + database.go 注册
- **`internal/scheduler/reconciliation_tasks.go`** (续写,+60 lines)
  - `RegisterTask` switch 新增 2 case: `createWorkorderCritical` / `createWorkorderHigh`
  - `createWorkorderBySeverity(ctx, db, woSvc, severity, limit)` helper:SELECT WHERE severity=? AND deleted_at IS NULL AND resolved_at IS NULL AND workorder_id IS NULL ORDER BY detected_at ASC LIMIT (50/30) → for each 调 woSvc.CreateWorkorderFromException,失败 logrus.Errorf + continue
  - `reconJobs` slice 追加 2 条:`对账-自动转工单critical`(@every 2m)/`对账-自动转工单high`(@every 5m)
  - `jobNameToInvokeTarget` 追加 2 mapping
- **`internal/core/db/migrations/migration_171_reconciliation_workorder_assignee_role.go`** (75 lines)
  - Seed 2 个 sys_config:
    - `asset.reconciliation.workorder.assignee_role_map` = `{"asset_owner":1,"ops_owner":2,"responsible_owner":3}`(role_id 1/2/3 占位)
    - `asset.reconciliation.workorder.sla_minutes_by_severity` = `{"critical":30,"high":240,"medium":1440,"low":10080}`
  - config_type='Y', is_system=1, idempotent count-then-insert(migration_165/169 风格)
- **`internal/core/db/migrations/migration_172_reconciliation_workorder_templates_seed.go`** (105 lines)
  - 补种 6 个 sys_workorder_category description:
    - A 类:"物理+责任人有且一致 健康无需动作"
    - B-F 类:5 句中文,与 reconciliation_template.go DescriptionLines 内容完全一致
  - 幂等 UPDATE:WHERE description IS NULL OR = R1 默认值,避免覆盖 admin 手动修改
- **`internal/core/db/database.go`** (+8 lines)
  - 在 Migrate170 之后追加 2 行 Migrate171 + Migrate172 注册

## Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| go build ./... exit 0 | PASS | `go build ./...` returns no output |
| 5 B-F templates + GetTemplate | PASS | `reconciliation_template.go:120-141` 定义 5 个 var,`reconciliationTemplatesByType` map 索引 |
| CreateWorkorderFromException | PASS | `reconciliation_workorder.go:90-180` 12 步流程完整 |
| scheduler 2 case + 2 sys_job | PASS | `grep createWorkorderCritical\|createWorkorderHigh` 命中 `reconciliation_tasks.go:71,80,226,228`;reconJobs slice 含 `对账-自动转工单critical`/@every 2m + `对账-自动转工单high`/@every 5m |
| database.go 注册 Migrate171+172 | PASS | `grep Migrate171\|Migrate172` 命中 `database.go:472,476` |
| migration_171 seed 2 sys_config | PASS | `grep assignee_role_map\|sla_minutes_by_severity` 命中 `migration_171_reconciliation_workorder_assignee_role.go:42,48` |
| migration_172 补种 6 category description | PASS | 6 个 templates[A/B/C/D/E/F] 在 `migration_172:53-94` |
| CLAUDE.md Status Convention | PASS | 全代码使用 0=启用 1=停用;priority 用 0=Low/1=Medium/2=High/3=Urgent (WorkOrderPriority) |
| operlog 约定 | PASS | `workorder.BaseService.Create` 内部已写 sys_oper_log(T-43-02 mitigation);本 service 不重复调用 |
| xingran-migrations-no-sql-autoloader | PASS | 2 个 migration 都是 .go 函数 + 在 database.go 显式注册 |
| 失败仅 logrus | PASS | 所有失败路径用 `logrus.Errorf` 或 `applogger.Errorf`,无 SysNotice 写入 |

## Deviations from Plan

None — plan executed exactly as written.

## Auth Gates

None — 计划无外部认证依赖,所有 DB 操作使用现有 db handle。

## Known Stubs

无 — 所有代码路径完整,无 TODO/FIXME/placeholder。

注:工单模板 DescriptionLines / Migration 172 seed description 与 reconciliation_template.go 的 5 句中文内容一致;若运维修改模板,需同步修改两处。这是显式同步约定,非 stub。

## Threat Flags

无新安全面。所有威胁已在 plan frontmatter 评估并 mitigation:
- T-43-01 (转单越权):submitterID="SYSTEM" + workorder status=Pending ✓
- T-43-02 (转单无审计):BaseService.Create 内部写 sys_oper_log ✓
- T-43-03 (大量异常 DoS):uniq_recon_asset_type_open 唯一索引 + LIMIT 50/30 ✓
- T-43-04 (JSONB 解析):Unmarshal 失败回退 nil,软失败 ✓
- T-43-SC (新依赖):无新依赖 ✓

## Followup Notes

### 后续 plans 依赖
- **43-02**:修复回写 + 静默期重检测(用本 plan 的 workorder_id 关联)
- **43-03**:WS 推送 + SysNotice + "标记已解决" UI(用本 plan 的 workorder_id + raw_snapshot)

### UAT 验证项(用户执行)
- 启动 backend → 检查 sys_job 表有 2 条 R2 cron(`对账-自动转工单critical`/@every 2m + `对账-自动转工单high`/@every 5m)
- 检查 sys_config 表有 2 条 seed(`asset.reconciliation.workorder.assignee_role_map` + `.sla_minutes_by_severity`)
- 手动 INSERT 一条 critical 异常(`severity='critical', workorder_id IS NULL, deleted_at IS NULL, resolved_at IS NULL`)→ 等待 2min → 验证 sys_workorder 有新工单 + sys_data_reconciliation.workorder_id 已回写
- 验证工单 description 含 SLA minutes(如 critical="## SLA: 30m")
- 验证 sys_user.role_id 1/2/3 是占位 → 默认 assigneeID=nil,工单创建后由人工分配

### 配置项调整建议
- `asset.reconciliation.workorder.assignee_role_map` 的 role_id 1/2/3 是占位,运维按实际 sys_role.id 修改(B/C/D=asset_owner, E=ops_owner, F=responsible_owner)
- 若运维调整 role_id,需在 admin 参数配置页修改 config_value 后,下个 cron 周期生效

## Self-Check: PASSED

- `internal/services/workorder/reconciliation_template.go` exists ✓
- `internal/services/asset/reconciliation_workorder.go` exists ✓
- `internal/scheduler/reconciliation_tasks.go` updated ✓
- `internal/core/db/migrations/migration_171_reconciliation_workorder_assignee_role.go` exists ✓
- `internal/core/db/migrations/migration_172_reconciliation_workorder_templates_seed.go` exists ✓
- `internal/core/db/database.go` updated ✓
- Commits `008621d0` (Task 1) + `94c5501b` (Task 2) exist in git log ✓
- `go build ./...` exit 0 ✓