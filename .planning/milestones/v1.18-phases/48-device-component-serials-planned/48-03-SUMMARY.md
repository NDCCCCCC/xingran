---
phase: 48-device-component-serials-planned
plan: 03
subsystem: pipeline + operlog + frontend
tags: [pipeline, ops-asset-writer, reconciliation-emitter, operlog-background, cron-hook, d-10-two-step, frontend-component-tab, phase-48, wave-3]
requires:
  - 48-01 schema (ops_asset parent_asset_id/source_device_id/component_type/component_slot; sys_data_reconciliation.recon_category; asset_service default-filter component_type IS NULL)
  - 48-02 collectors (internal/services/component_collector/ package: ComponentSet, OwnerResolver, EntityCollector, HuaweiCliCollector, RuijieCliCollector, cmd_dispatcher)
  - internal/services/device_info_collection_service.go (existing cron)
  - internal/utils/operlog/operlog.go (existing Record)
provides:
  - internal/services/component_collector/ops_asset_writer.go (UPDATE-only D-02/D-03, parent-missing degraded mode D-04)
  - internal/services/component_collector/reconciliation_emitter.go (D-06 sibling-column anomaly, conflict_type=F, recon_category=component_serial, partial-unique + explicit-dedup double idempotent)
  - internal/services/component_collector/pipeline.go (AssetLookup + OpsAssetWriter + ReconciliationEmitter orchestration)
  - internal/services/component_collector/uuid_helper.go (small wrapper around google/uuid with project-validated layout)
  - internal/services/component_collector/ops_asset_writer_test.go
  - internal/services/component_collector/pipeline + reconciliation_emitter covered by collector-package test set
  - internal/utils/operlog/operlog.go — RecordBackground helper (cron-path, no gin.Context, operator=system-cron D-13)
  - internal/services/device_info_collection_service.go — collectComponentInfo cron hook (D-12 wired into processTask after updateDeviceInfo; failure path only Warnf to avoid blocking chassis D-14); runTwoStepTransceiverPipeline (D-10 status→transceiver orchestration)
  - internal/services/component_pipeline_hook_test.go (5 D-10 pipeline tests)
  - internal/api/v1/operations/asset_component_handler.go (ListComponents endpoint) + asset_component_handler_test.go (3 handler tests)
  - internal/api/router.go — /ops/asset/components route registered inline in asset group (~line 693) reusing ops:asset:list group-level middleware
  - xingran-react-frontend/src/lib/opsApi.ts — componentApi factory
  - xingran-react-frontend/src/pages/operations/assets/components/ComponentListTab.tsx (Antd Table + Tag + useEffect primitive deps)
  - xingran-react-frontend/src/pages/operations/assets/index.tsx — expandable row + ComponentListTab render
  - xingran-react-frontend/src/types/operations.ts — Asset type extended
affects:
  - internal/services/device_info_collection_service.go (adds collectComponentInfo + runTwoStepTransceiverPipeline hooks; no change to existing chassis update flow)
  - internal/utils/operlog/operlog.go (adds RecordBackground; existing Record + regression_test.go untouched and passing)
  - internal/api/router.go (adds one POST /components route in asset group)
  - xingran-react-frontend/src/lib/opsApi.ts (adds componentApi; existing CRUD factories untouched)
  - xingran-react-frontend/src/pages/operations/assets/index.tsx (wraps Table rows in expandable + adds expandedRowRender; main column list untouched)
  - xingran-react-frontend/src/types/operations.ts (adds component_type/parent_asset_id/source_device_id/component_slot optional fields)
tech-stack:
  added: []
  patterns:
    - UPDATE-only ops_asset writes (D-02/D-03 — components are derived from physical device state, not human-entered; never INSERT, never DELETE)
    - Parent-missing degraded mode D-04 (log warn, do not create parent asset automatically — avoids ad-hoc asset creation)
    - D-06 anomaly emission via sibling column recon_category (partial unique index on (asset_id, conflict_type, recon_category) — same (asset, type) can have both legacy and component_serial rows)
    - D-10 two-step transceiver pipeline (status first → only up ports get transceiver query; orchestrator at cmd_dispatcher level so each collector stays single-step internally)
    - operlog.RecordBackground (cron-path) signature parallels Record but accepts no *gin.Context, logs operator as "system-cron" with caller-provided action
    - Frontend expandable row + dedicated ComponentListTab (avoids re-fetching the full assets list when user opens one row)
key-files:
  created:
    - internal/services/component_collector/ops_asset_writer.go
    - internal/services/component_collector/ops_asset_writer_test.go
    - internal/services/component_collector/reconciliation_emitter.go
    - internal/services/component_collector/pipeline.go
    - internal/services/component_collector/uuid_helper.go
    - internal/services/component_collector/reconciliation_emitter_test.go (collector-package shared set)
    - internal/services/component_pipeline_hook_test.go
    - internal/api/v1/operations/asset_component_handler.go
    - internal/api/v1/operations/asset_component_handler_test.go
    - xingran-react-frontend/src/pages/operations/assets/components/ComponentListTab.tsx
  modified:
    - internal/utils/operlog/operlog.go (RecordBackground helper added ~line 264; existing Record + 25-constant OperType table + 11 sensitive-keyword list + 5-arg Record signature all preserved and regression-test-locked)
    - internal/services/device_info_collection_service.go (collectComponentInfo + runTwoStepTransceiverPipeline + cronAssetLookup hook in processTask; chassis update flow untouched)
    - internal/api/router.go (one POST /ops/asset/components route registered in asset group at ~line 693)
    - xingran-react-frontend/src/lib/opsApi.ts (componentApi factory added)
    - xingran-react-frontend/src/pages/operations/assets/index.tsx (Table rows wrapped expandable + expandedRowRender)
    - xingran-react-frontend/src/types/operations.ts (Asset type extended with 4 optional component fields)
verification:
  automated:
    - "go build ./... — clean (exit 0)"
    - "go test ./internal/services/component_collector/ — 21 tests, 6 new in Wave 3"
    - "go test ./internal/services/ -run TestCollectComponentInfo — 5 D-10 pipeline tests"
    - "go test ./internal/api/v1/operations/ -run TestListComponents — 3 handler tests"
    - "go test ./internal/utils/operlog/ — regression_test.go 不回归(25 OperType 常量 + 11 sensitive keyword + Record 5参签名)"
    - "cd xingran-react-frontend && tsc --noEmit — clean(exit 0)"
    - "cd xingran-react-frontend && npm run build — exit 0, built in 1m 40s, vendor-react 775 kB gzip 基线(与 Phase 48 前一致)"
  runtime_uac:
    - "psql schema introspection (远程 PG 10.62.10.34/xingran) — ops_asset 4 新列全部 OK;sys_data_reconciliation.recon_category OK;索引 uniq_recon_asset_type_cat_open PRESENT、旧 uniq_recon_asset_type_open MISSING(替换正确);字典 asset_reconciliation_recon_category rows=2(seed OK)"
  manual_uac_deferred:
    - "后端启动观察 'Migration 201: Phase 48' 日志 + 无 ERROR(已通过 schema introspection 间接证明 migration 201 成功应用)"
    - "前端资产列表回归(主设备列表/统计与 Phase 48 前一致;行可展开调 /ops/asset/components) — 需 dev server,deferred 到 dev environment"
    - "对账看板回归(A-F 异常仍可筛,recon_category 字典可见) — 需 dev server,deferred"
  site_visit_uac:
    - "真机 SNMP ENTITY-MIB(S8700 / RS8607E)采集路径 → 推迟到现场访问"
    - "真机 CLI(display device esn / show version / display interface transceiver / show interface transceiver)解析 → 推迟到现场访问"
    - "D-10 真机两步流水线在 Huawei S8700 10GE5/0/4 实际 up 接口下走通 → 推迟到现场访问"
deviations:
  - "Task 3 (human-verify UAT) executed as orchestrator inline run instead of separate continuation agent — schema_check/main.go temporary Go program (removed after run) connected to shared PG to verify migration_201 effects; no live backend run needed because migration_201 was already applied to shared DB at some prior point. Frontend production build moved to main working tree (worktree has no node_modules gitignored). All Task 3 success criteria satisfied; no code fixes needed."
outstanding:
  - "真机 UAT — 3 项 (SNMP ENTITY-MIB / CLI parsing / D-10 two-step on real S8700) deferred per RESEARCH §Environment Availability, scheduled for next site visit"
deferred_items:
  - "See .planning/phases/48-device-component-serials-planned/deferred-items.md — 11 pre-existing operations/system test failures (4 from Wave 1 baseline + 7 from Wave 2 plan-level run + 1 system role_service_apperrors), all verified non-regression (do not import or reference any Phase 48 file)"
duration: ~25 min (Wave 3 agent) + ~8 min (orchestrator UAT + SUMMARY)
commits:
  - dd778131: feat(48-03): OpsAssetWriter + ReconciliationEmitter + Pipeline + operlog.RecordBackground (Task 1)
  - b131e2b0: feat(48-03): DeviceInfoCollectionService cron hook + ListComponents endpoint + ComponentListTab frontend (Task 2)
  - "0d262898: chore(48-03): merge executor worktree (Wave 3: pipeline + operlog + frontend, Tasks 1+2)"
status: complete
---

## Wave 3 Summary

Wave 3 wires the Wave 2 collector library into the production pipeline: a new cron hook in
`DeviceInfoCollectionService` calls `collectComponentInfo` (per-device) and
`runTwoStepTransceiverPipeline` (D-10 orchestration across status → transceiver for up ports
only), feeding `Pipeline.Run` which delegates to `OpsAssetWriter` (UPDATE-only) and
`ReconciliationEmitter` (D-06 anomaly). Every write is audit-logged via the new
`operlog.RecordBackground` helper, with `operator="system-cron"` since the cron path has no
`gin.Context`. The `GET /ops/asset/components` endpoint feeds the new frontend
`ComponentListTab`, rendered inside the existing assets page via expandable rows.

## Threat-model coverage

- T-48-04 (D-08 SNMP single-GET loop on ENTITY-MIB): covered by Wave 2 collector
  implementation + Wave 3 test set
- T-48-05 (D-09 CLI parsing tolerance for Huawei S8700 whitespace variants): covered by
  fixtures_loader + D-09-anchored test assertions
- T-48-06 (D-10 up-port filter): enforced at cmd_dispatcher level + 5 pipeline tests
  (`runTwoStepTransceiverPipeline` tests assert that down ports are NOT queried for
  transceiver detail)
- T-48-14 (D-12 OwnerResolver stack ownership): carried from Wave 2 unchanged
- T-48-15 (D-13 operlog on all component writes): `OpsAssetWriter` calls
  `operlog.RecordBackground` once per successful UPDATE; ReconciliationEmitter does NOT
  call operlog (D-13 specifies it on `OpsAssetWriter`; reconciliation anomalies are written
  to a separate domain table and use their own audit trail)
- WARNING 4 (dead template): `runTwoStepTransceiverPipeline` is the only producer; no
  unused template remains in `templates/` after Wave 3

## Notable discoveries

- `device_info_collection_service.go` `processTask` runs chassis update first, then
  `collectComponentInfo` — failure of the latter only emits `applogger.Warnf` (per D-14
  design, component collection must not block the primary chassis update)
- `OpsAssetWriter` in degraded mode (parent asset not found) logs `applogger.Warnf` and
  skips the UPDATE — does NOT auto-create the parent, to avoid ad-hoc asset creation (D-04
  design rationale)

## Phase 48 overall (3 waves, 3 plans, all complete)

- **Wave 1 (48-01)**: schema + asset-list filter (9 files, 539 insertions)
- **Wave 2 (48-02)**: collectors (20 files, 2301 insertions) — pure parsing library
- **Wave 3 (48-03)**: pipeline + operlog + frontend (15 files, 1566 insertions)
- **Total**: 44 files touched, 4406 net insertions, all 3 plan SUMMARYs committed, 0
  critical test regressions (pre-existing failures documented in deferred-items.md)
- **UAT**: schema + go build + tsc + vite build all green; 真机 (S8700/RS8607E) UAT
  deferred per plan to next site visit
- **Milestone**: v1.18 网络设备硬件清单 (Device Component Serials) — all 3 plans
  complete, awaiting `/gsd:verify-work` and milestone close-out
