# Milestone v1.17 — Project Summary

**Generated:** 2026-07-08
**Status:** SHIPPED 2026-07-03 (archived in `.planning/milestones/v1.17-{ROADMAP,REQUIREMENTS}.md`)
**Purpose:** Team onboarding and project review

---

## 1. Project Overview

**XingRan-Next** is a Go backend + React frontend enterprise IT operations management system.
The v1.17 milestone adds an **Asset Reconciliation** engine: an automated observability layer
that detects drift between three sources of asset truth — *physical* (port→MAC chain),
*declared* (`sys_workstation.user_id`), and *AD managed_by* (`description`-style host record).
It drives an alert → workorder → 7-day rollback loop, exposed both to ops staff (a health-score
card on workstation detail pages) and to a semi-automated fix-suggestion workbench.

**Why it matters:** Before v1.17, drift was found by accident ("工单里有人报错"). Now operators
get a dashboard, deterministic reconcile logic, and a safe one-click rollback path — so a wrong
fix is recoverable inside a 7-day window.

**Headline shipped features:**

1. **Three-way reconciliation engine** with Type A-F classification (健康 / 无主 / 不一致 /
   未上线 / 幽灵 / AD 不一致), backed by a materialized view (`reconciliation_normalized`,
   5-min incremental refresh).
2. **IP-segment exception rules** — CIDR-based (e.g. `192.168.0.0/16`) with 5-action unions
   (no_alert / no_notice / no_workorder / skip_severity / silence) + `expires_at` auto-disable
   + a live "命中测试" tool.
3. **Alert + workorder closure loop** — critical/high anomalies auto-create workorders; frontend
   gets WebSocket push + `sys_notice` write; 7-day silence-after-resolve + 24-hour throttle
   prevent alert storms / regeneration loops.
4. **Workstation-detail-page integration** — health-score card on top (5 KPIs + mini trend),
   inline reconciliation badge in nested asset/AD sub-tables, click-to-open 3-tab
   `ReconciliationDrawer`.
5. **Optional R5 半自动修复** — 6-state machine, partial unique index
   `uniq_fix_suggestion_pending_per_exception`, 7-day one-click rollback with DB- and
   Go-side window check, mis-fix-rate monitor with 1-hour throttle, 5-KPI + 8-column admin page.

---

## 2. Architecture & Technical Decisions

### Locked decisions (per ROADMAP)

- **Observe-only strategy.** Engine never mutates `sys_workstation.user_id` or any business
  table; fixes are explicit user-driven actions. Why: drift provenance must survive in
  `raw_snapshot` for post-mortem.
- **Permission namespace:** `asset:reconciliation:*`. Why: slot alongside `ops:building:*`,
  `network:port:*`, etc. — keeps the existing RBAC matrix authoritative.
- **API surface:** `/asset/reconciliation/*` (list/exception/fix-suggestion/stats/etc.).
- **Cross-module call:** `ops/workstation` → `asset/reconciliation` invoked at *service layer*
  with permission downgrade. Why: middleware-level cross-module calls are hard to test;
  service call preserves cron-friendly context.
- **菜单归属：** 资产管理 / 数据质量 (asset reconciliation + 3 二级菜单).
- **Owner 合并:** 运维 + 资产 + 权限 = 同一人 (单签,无需双签).

### Implementation patterns established

- **6 dedicated `Statistics` COUNT endpoints** (Summary / ByConflictType / BySeverity /
  HealthTrend / TopUnresolved / ExceptionRuleStats). Why: list endpoints cap at MaxPageSize=100,
  which would silently truncate dashboard totals.
- **`operlog.Record` covers all reconciler writes.** Module constants
  (`"对账异常"` / `"修复建议"`) registered in the operlog package to keep audit lines
  queryable.
- **`RawSnapshot` is JSONB**, frozen at detection time. Why: every later decision (fix,
  rollback, silence) can refer to the original evidence for "what changed since".
- **`uniqueness fix pending-per-exception` partial unique index** uses GORM-friendly naming
  (`uniq_*_*`) while the SQL helper uses raw `CREATE UNIQUE INDEX IF NOT EXISTS`.
  Why: see the GORM-vs-SQL naming-conflict memory note.
- **Materialized view refresh uses a 5-min cron** with a `recoverMaterializedViews()`
  helper that runs ahead of `ALTER TYPE` AutoMigrate statements, to avoid the
  "GORM AutoMigrate blocked by matview" trap documented in project memory.

### Cross-phase reuse audit (v0.4, captured at R1 launch)

- ✅ Reused: 字典/参数/operlog/Handler-Service/CacheProvider/Excel/UUID/Status/前端 hooks/
  opsApi/ECharts/UI 库 (13 items).
- ⚠️ Partial reuse: 字典枚举值/参数 seed/Excel config/operlog module 常量 (4 items).
- ❌ New: cache_key helper + Statistics 端点 + queryKeys + operlog module + HealthScore
  函数 + 路由注册 + Cron 注册 (7 items, deliberately net-new).

---

## 3. Phases Delivered

| Phase | Name | Status | One-Liner |
|-------|------|--------|-----------|
| 42 (R1) | 资产对账观测底座 | ✅ Complete (5/5 plans, 42-03 skeleton superseded by R3) | Materialized view + reconciliation tables + 6 Statistics endpoints + dashboard 5 KPI + operlog coverage |
| 43 (R2) | 告警 + 工单闭环 | ✅ Complete (3/3 plans) | critical/high auto-workorder + WebSocket + SysNotice + 7-day silence |
| 44 (R3) | 置信度评分 + IP 段例外 | ✅ Complete (2/2 plans) | CIDR rules + 5 actions + expiry + 命中测试; verifier 10/10 |
| 45 (R4) | 工位详情整合 | ✅ Complete (2/2 plans) | 健康度 card + ReconciliationDrawer 3 Tab + 行内健康徽标 |
| 46 (R5) | 半自动修复 (optional) | ✅ Complete (2/2 plans, UAT 9/10 PASS) | 6-state machine + 7d 回滚 + 误修复率告警三通道 + 修复建议 admin 页 |
| 47 (post) | v1.18 root-cause fix for infoPoint / Layer-3 / port_status drift | ✅ Complete | (See PROJECT.md; ships as part of v1.17 close-out) |

---

## 4. Requirements Coverage

Per `.planning/milestones/v1.17-REQUIREMENTS.md` — all **30+ requirement IDs** shipped.
Summary by category:

- ✅ **RECON-01..07** (engine core, 7 IDs): Type A-F classification, reverse user_id inference,
  confidence scoring (physical +0.5 / declared +0.3 / AD +0.2), 5-min materialized view,
  raw_snapshot capture, 7-day silence, `sys_config`-driven behavior.
- ✅ **EXCEPTION-01..04** (4 IDs): CIDR rules, 5-action union, expiry auto-disable, 命中测试.
- ✅ **WORKORDER-01..02** (2 IDs): critical/high auto-create + operlog coverage.
- ✅ **MONITOR-01..03** (3 IDs): 6 Statistics endpoints, WebSocket critical-push, SysNotice
  write.
- ✅ **INTEGRATE-01..03** (3 IDs): workstation page top-card health, sub-table badge,
  ReconciliationDrawer 3-tab open.
- ✅ **AUDIT-01..02** (2 IDs): `operlog.Record`/`RecordWithBody` on all write endpoints,
  raw_snapshot JSONB.
- ✅ **INFRA-01..05** (5 IDs): 4 dict / 8 config / 6 workorder-category seeds, 8 cache key
  constants, 9 queryKeys.
- ✅ **SC1-SC7** (R5, 7 IDs): generator cron, accept/reject UI, 7-day rollback, mis-fix-rate
  monitor <1%, operlog chain, 6-state machine, R5 closes milestone.

**UAT result:** 9/10 PASS in R5; 1 skipped (`Test 10 权限非 admin 端到端`, requires test
account setup). Non-blocking.

**Deferred / known gaps:**

| Gap | Phase | Severity | Resolution |
|-----|-------|----------|------------|
| 42-03 SUMMARY 缺失 (replaced by Phase 44) | R1 | documentation | R3 supersedes skeleton; close-out notes document this |
| Test 10 (权限非 admin e2e) skipped | R5 | testing | Need test account; bookkeep into future UAT cycle |
| Cron `7,17,27,37,47,57 * * * *` 缺年份字段 | R5 | bug | Ops SQL fix (UPDATE `sys_job` add `0` prefix) |
| Apply `resolution_method` 列不存在 | R5 | bug | Code fix `87d5fc82` |
| Rollback `pre_fix_user_id == nil` 过度防御 | R5 | bug | Code fix `87d5fc82` |

All on-deck bugs were fixed before milestone close; documentation gaps are non-blocking.

---

## 5. Key Decisions Log

(From archived ROADMAP + 44-PLAN / 46-PLAN context — selected material decisions.)

| ID | Decision | Why | Phase |
|----|----------|-----|-------|
| D-A1 | Observe-only 策略 (only `sys_data_reconciliation` + `sys_reconciliation_exception`, never mutate business tables) | 故障溯源需要 `raw_snapshot` 长期冻结 | 42 (R1) |
| D-A2 | Materials view = `reconciliation_normalized`, 5-min cron refresh + DROP-dependent-MV helper | GORM AutoMigrate 与物化视图冲突;helper 解锁 safe re-create | 42 |
| D-A3 | 6 Statistics COUNT 端点 (no `list.length`) | MaxPageSize=100 会把 dashboard 总数钳制 | 42 |
| D-A4 | Fix suggestion generator cron `@every 5m`, trigger condition = `confidence>=threshold AND conflict_type='B' AND workorder_id IS NULL AND resolved_at IS NULL` | 唯一性 + 不重复触发 + 不打断已用工单 | 46 (R5) |
| D-B1 | 6-state machine (pending/accepted/rejected/applied/rolled_back/failed) | 与 修复/拒绝/应用/回滚/异常 五条用户路径对齐 | 46 |
| D-B2 | partial unique index `uniq_fix_suggestion_pending_per_exception` (only on `status='pending'`) | 防并发 Accept 同一个 exception_id | 46 |
| D-B3 | Apply 同步写 `resolved_at` on `sys_data_reconciliation` | 阻止 generator 重新产生循环 | 46 |
| D-B4 | 一键回滚 7d 窗口 = 双层校验 (Go-side `before` + DB-side `INTERVAL`) | 防 Go-side clock-drift / 数据库时间错位 | 46 |
| D-C1 | Rollback 只回滚 `user_id` (不是 component / status) | 最小破坏性变更;其他字段可能已有其他 fix | 46 |
| D-C5 | 误修复率 cron `7,17,27,37,47,57 * * * *` + 1h 节流 (`lastBreachNotifiedAt` 状态机) | 慢变量(7d 窗口)不需要 10min 6次推送;1h 是信息增量合理上限 | 46 |
| D-D1 | Operlog OperType mapping: Apply → `OperTypeBatch`(?), Rollback → `OperTypeReset=11` | 区分 status-change vs reset-action | 46 |
| (cross) | `operlog.Record` 强约定 — handler module 调用前是 success-path 必经 | Phase 34 invariant lock (25 OperType + 11 sensitive keywords regression_test.go) | 42, 43, 46 |

---

## 6. Tech Debt & Deferred Items

### Hardened-debt ledger at milestone close

The R5 phase explicitly hardened **5 design risks** that had been ACCEPTED_AS_KNOWN_LIMITATION
in earlier R-rounds. These are *added* defenses, not removed debt:

1. **partial unique index** — concurrent-accept invariant now DB-enforced
   (`uniq_fix_suggestion_pending_per_exception`).
2. **two-layer (Go-side + DB-side) rollback window** — `INTERVAL '7 days'` defense even when
   the Go code computes `Now() - before` incorrectly.
3. **resolved_at synchronously written** in Apply — generator can no longer regenerate.
4. **`OperTypeReset = 11`** assigned to rollback — distinguishes rollback from apply.
5. **Mis-fix-rate throttle (1h `lastBreachNotifiedAt`)** — only one WS+SysNotice+operlog
   event chain per breach, not one per cron tick.

### Known limitations handed to v1.18+

- **Mis-fix-rate frontend subscription** — currently only backend WS push. v1.18 candidate
  to extend `useReconciliationWebSocket` hook with browser-side threshold sliders.
- **R5 effectiveness dashboard** — applied → 7-day rolling re-rollback rate trend not yet
  surfaceable.
- **No automated test for `Test 10` (non-admin E2E)** — needs a test account fixture.

### Lessons fed forward (from RETROSPECTIVE.md cross-milestone trends)

- **Pre-existing test failure baseline was re-used**: documented baseline run pre/post
  Phase 42-46 to avoid being blamed for the 3 sets of long-standing failures
  (`tests/integration/login_encryption_test.go`, `internal/services/operations/...`).
- **Cross-milestone pattern confirmed**: `pg_indexes` introspection replaces
  `DROP IF EXISTS` (GORM doesn't honor IF EXISTS) — applied again in v1.17 migration 199
  for the partial unique index.
- **operlog module 中文常量 forced** — `"对账异常"`/`"修复建议"` 模块名 in operlog
  chain keeps audit grep stable across services.

---

## 7. Getting Started

### Entry points for new contributors

| Activity | Path | Notes |
|----------|------|-------|
| **Run backend** | `cmd/main.go` | `go run ./cmd/main.go`; reads `configs/config.yaml` |
| **Run frontend** | `xingran-react-frontend/` | `npm install && npm run dev` (port 4000) |
| **V1.17 模型** | `internal/models/data_reconciliation.go` | `DataReconciliation` + `ReconciliationException` + R5 `FixSuggestion` |
| **V1.17 服务入口** | `internal/services/asset/reconciliation_service.go` + `fix_suggestion_service.go` | Handler-Service pattern; private impl + constructor |
| **V1.17 HTTP 端点** | `internal/api/v1/asset/reconciliation_handler.go` + `fix_suggestion_handler.go` | `POST /asset/reconciliation/*` |
| **V1.17 cron** | `internal/scheduler/reconciliation_tasks.go` + `fix_suggestion_tasks.go` | 物化视图 refresh + generator + monitor |
| **V1.17 前端页面** | `src/pages/asset/reconciliation/` | List + Drawer + ExceptionRule 管理 + Fix Suggestion admin |
| **V1.17 前端类型** | `src/types/reconciliation.ts` | Mirror backend JSON tag camelCase |
| **Tests** | `internal/services/asset/*_test.go` (5 layers); `internal/api/v1/asset/...test.go`; `xingran-react-frontend/src/pages/asset/.../*.test.tsx` | 5-layer test set per RETROSPECTIVE |
| **Run v1.17 tests** | `go test ./internal/services/asset/... ./internal/api/v1/asset/...` + `npx vitest run src/pages/asset/` | All gates green |

### Where to look first (ordered)

1. **`.planning/milestones/v1.17-ROADMAP.md`** — full phase breakdown, key decisions locked.
2. **`.planning/milestones/v1.17-REQUIREMENTS.md`** — every requirement ID with traceability
   back to phase + plan.
3. **`internal/services/asset/reconciliation_service.go`** — the core engine.
4. **`internal/core/db/migrations/migration_1XX_reconciliation*.go`** — schema evolution.
5. **`internal/scheduler/reconciliation_tasks.go`** — the cron entrypoint.

### Key architectural invariants to respect (Do Not Break)

- **`operlog.Record` MUST be called before `response.Success` on all write paths.**
  Regression lock: `internal/utils/operlog/regression_test.go` (25 OperType constants +
  11 sensitive keywords + Record 5-arg signature). Failing this lock = CRITICAL regression.
- **Never write to `sys_workstation.user_id` automatically.** R5 Apply is the only path
  allowed (and it goes through `OperTypeReset`-style audit).
- **`raw_snapshot` is immutable.** Once written, do not UPDATE the JSONB — it is the audit
  chain's evidence.
- **Reconciliation generator locks:** trigger condition is
  `confidence>=threshold AND conflict_type='B' AND workorder_id IS NULL AND resolved_at IS NULL`.
  Adding new conditions = requires decision ID logged in current milestone ROADMAP.

---

## Stats

- **Timeline:** 2026-06-27 (R1 planning start) → 2026-07-03 (milestone close). 7 days
  wall-clock.
- **Phases:** 6 (R1/R2/R3/R4/R5 + Phase 47 root-cause fix). All complete.
- **Plans:** 14 (R1=5 + R2=3 + R3=2 + R4=2 + R5=2; 42-03 skeleton superseded by R3).
- **Atomic commits:** 17 on `worktree-phase47-discuss` branch + 1 in-session fix
  (`87d5fc82`) + 2 docs commits = **20 total** for v1.17.
- **Files added (post-v1.17 baseline):**
  - Backend: 7 recon files (model + 3 migrations + 3 services)
  - Backend: 2 (handler + router)
  - Frontend: 4 (api + queryKeys + page + drawer + modal)
  - + Phase 47 root-cause fixes
- **Files in repo at v1.17 close:** ~485 Go (backend) + ~506 TS/TSX (frontend, +1 page
  for fix-suggestion).
- **Requirement IDs shipped:** **30+** requirements across 8 categories (RECON-7,
  EXCEPTION-4, WORKORDER-2, MONITOR-3, INTEGRATE-3, AUDIT-2, INFRA-5, R5 SC1-7).
- **Test result at close:** R5 UAT 9/10 PASS + 1 SKIP (test account fixture pending).
- **Contributors:** see `git log --since=2026-06-27 --until=2026-07-04 --format=%ae | sort -u`
  (single primary author + assist reviewers).
- **Crosses:** BUILD-OK (`go build ./...`), LINT-OK (`go vet ./...`), zero migrations
  conflict warnings at archive time.

---

*Generated by `/gsd:milestone-summary v1.17`. Source of truth:
`.planning/milestones/v1.17-ROADMAP.md`, `v1.17-REQUIREMENTS.md`,
`20260703-v1.17-r5-close.md`, plus `STATE.md` & `RETROSPECTIVE.md` cross-references.*
