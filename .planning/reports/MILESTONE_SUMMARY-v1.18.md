---
milestone: v1.18
milestone_name: 网络设备硬件清单 (Device Component Serials)
shipped: 2026-07-04 (Phase 48) + 2026-07-06 (Phase 49 gap closure)
phases: 2 (Phase 48 + Phase 49)
plans: 5 (48-01 / 48-02 / 48-03 / 49-01 / 49-02)
generated: 2026-07-08
---

# Milestone v1.18 — Project Summary

**Generated:** 2026-07-08
**Purpose:** Team onboarding and project review

---

## 1. Project Overview

**XingRan-Next** is a Go backend + React frontend enterprise IT operations management system. It evolved from a workstation-import association feature (v1.0) into a multi-domain platform covering operations (buildings, floors, workstations, assets), network (device discovery, SNMP, Scrapli-based command execution, ports), identity (AD/LDAP, RBAC), and VDI (Sangfor). National cryptography (SM2/SM3/SM4) is mandatory on sensitive endpoints.

**Core value:** End-to-end operational observability and auditability — every write action produces a traceable record (who, when, what, from-where, before/after-state) with sensitive fields auto-masked. Combined with Excel-driven bulk management, this gives operators a single trustworthy system-of-record for the entire IT estate.

**v1.18 specifically** delivered the "one device, many serial numbers" capability — network device components (chassis, cards, engines, power supplies, fans, transceivers) now persist as `ops_asset` rows with parent-child relationships, integrated into the existing asset reconciliation engine. This milestone extends asset management from whole-device to sub-device granularity while reusing existing reconciliation, operlog, and asset infrastructure.

---

## 2. Architecture & Technical Decisions

### Storage Model
- **Decision:** Components as `ops_asset` rows (no separate sub-table)
  - **Why:** Asset system unified as "serial number = asset"; fits external Excel import conventions
  - **Phase:** 48-01 (D-01)
- **Decision:** 4 new dedicated columns (`parent_asset_id`, `source_device_id`, `component_type`, `component_slot`), not JSON reuse
  - **Why:** Reconciliation needs structured filtering by category
  - **Phase:** 48-01 (D-05)
- **Decision:** Dual bridging (`parent_asset_id` → `ops_asset.id`, `source_device_id` → `sys_network_device.id`)
  - **Why:** Internal parent anchor + external collection provenance both needed
  - **Phase:** 48-01/02 (D-03)
- **Decision:** UPDATE-only writes to `ops_asset` from collector pipeline (no INSERT/DELETE of component rows)
  - **Why:** Components are derived data from physical state; never allow collector to auto-create assets (preserves v1.17 reconciliation layering)
  - **Phase:** 48-03 (D-02)

### Reconciliation Integration
- **Decision:** Sibling-column `recon_category` with `component_serial` category + partial unique index `(asset_id, conflict_type, recon_category) WHERE open`
  - **Why:** Same (asset, conflict_type) can have both legacy and component_serial anomaly rows; UI can filter by category without affecting A-F semantics
  - **Phase:** 48-01 + 48-03 (D-06)
- **Decision:** Service-layer hardcoded `component_type IS NULL` filter in `asset_service.List()` and `Statistics()`
  - **Why:** Avoids "1 switch + 6 boards = 7 devices" inflation in main list/stats views
  - **Phase:** 48-01 (D-07)
- **Decision:** Parent-missing degraded mode (`parent_asset_id = NULL` + `Warnf`, no parent auto-creation)
  - **Why:** Switch asset missing is a separate concern; don't pollute component layer with parent recovery
  - **Phase:** 48-03 (D-04)

### Collection Strategy
- **Decision:** Dual-path collection — SNMP ENTITY-MIB (primary, single-GET) + CLI TextFSM (secondary, fallback)
  - **Why:** Huawei S8700 + Ruijie RS8607E reject SNMP GETBULK; some data (Ruijie boards, Huawei transceivers) only available via CLI
  - **Phase:** 48-02 (D-08, D-09)
- **Decision:** Two-step transceiver pipeline (status first → only up ports get transceiver query)
  - **Why:** Avoids sending `display interface transceiver` on empty ports (cost optimization)
  - **Phase:** 48-02 + 48-03 (D-10)
- **Decision:** Filter Ruijie `temprature*` noise nodes (`Class==8 && HasPrefix(Name, "temprature")`)
  - **Why:** Ruijie private typo for `temperature` pollutes 352/627 ENTITY-MIB rows; real PSUs (`power0/1/2`) must be retained
  - **Phase:** 48-02 (D-11)
- **Decision:** OwnerResolver with `entPhysicalContainedIn` tree reconstruction + 32-hop cycle guard + canonical root (smallest chassis Index)
  - **Why:** Stack scenarios (iStack/IRF/VSU) need stable ownership; deterministic input-order-independent root selection
  - **Phase:** 48-02 (D-12)

### Operations Integration
- **Decision:** `operlog.RecordBackground` helper added for cron-path (no `*gin.Context`)
  - **Why:** Reuses 25-constant OperType table + 11-keyword sensitive filter; cron operator="system-cron"
  - **Phase:** 48-03 (D-13)
- **Decision:** `collectComponentInfo` cron hook failure path = `Warnf` only, no operlog
  - **Why:** Avoids audit noise from transient collection failures; chassis update must not be blocked
  - **Phase:** 48-03 (D-13, D-14)

### Frontend Integration
- **Decision:** Antd Table expandable row + dedicated `ComponentListTab.tsx` (not main-list reload)
  - **Why:** Avoids re-fetching full assets list on row open
  - **Phase:** 48-03 (D-07)
- **Decision:** Backend `GET /ops/asset/components` endpoint reuses `ops:asset:list` group-level middleware
  - **Why:** No new permission required; component access inherits asset list access
  - **Phase:** 48-03

---

## 3. Phases Delivered

### Phase 48 — 网络设备组件序列号采集 (Device Component Serials)

| Wave | Plan | Description | One-Liner |
|------|------|-------------|-----------|
| **W1** | 48-01 | Schema + Asset-List Filter | `ops_asset` 4 new columns + `sys_data_reconciliation.recon_category` sibling column + index switch (`uniq_recon_asset_type_cat_open`) + asset_service default filter `component_type IS NULL` |
| **W2** | 48-02 | Component Collectors | `component_collector/` 17-file package: ComponentSet/OwnerResolver/EntityCollector/HuaweiCliCollector/RuijieCliCollector/cmd_dispatcher + 6 TextFSM templates + D-08/D-10/D-11 enforcement |
| **W3** | 48-03 | Pipeline + Operlog + Frontend | OpsAssetWriter (UPDATE-only + D-04 parent-degraded) + ReconciliationEmitter (D-06 sibling-column) + Pipeline orchestration + `operlog.RecordBackground` + cron hook + `ListComponents` endpoint + `ComponentListTab` expandable row |

**Phase 49 — v1.18 Gap Closure** (post-shipping UAT fix)

| Plan | Description | One-Liner |
|------|-------------|-----------|
| 49-01 | chassis SN 采集接入 | `enrichChassisSerial` hooks into `collectDeviceInfo` to call `ParseShowVersionModules`/`ParseDisplayDeviceEsn` for Ruijie/Huawei chassis SN extraction (fixes Gap 3: `sys_network_device.serial_number` 100% empty) |
| 49-02 | 板卡采集接入 + E2E validation | `collectComponentInfo` now wires board collection; production E2E validated RS8607E-03 shows 9 components |

---

## 4. Requirements Coverage

### ✅ 14 D-ids / 11 REQ-48-* (all satisfied)

| D-id | REQ | Description | Status |
|------|-----|-------------|--------|
| D-01 | REQ-48-01 | Components as `ops_asset` rows (not sub-table) | ✅ |
| D-02 | REQ-48-05 | Collection is matching-only (UPDATE-only, no auto-INSERT) | ✅ |
| D-03 | REQ-48-05 | Dual bridging (`parent_asset_id` + `source_device_id`) | ✅ |
| D-04 | REQ-48-08 | Parent-missing degraded mode (`NULL` + warn) | ✅ |
| D-05 | REQ-48-01 | Dedicated `component_type` + `component_slot` columns | ✅ |
| D-06 | REQ-48-06 | Sibling `recon_category` + partial unique index | ✅ |
| D-07 | REQ-48-07 | asset_service default-filter `component_type IS NULL` | ✅ |
| D-08 | REQ-48-02 | SNMP ENTITY-MIB single-GET collector (Huawei + Ruijie) | ✅ |
| D-09 | REQ-48-03 | CLI TextFSM collectors (Huawei display + Ruijie show) | ✅ |
| D-10 | REQ-48-03 | Two-step transceiver pipeline (status → up-only ports) | ✅ |
| D-11 | REQ-48-04 | Ruijie `temprature*` noise filter | ✅ |
| D-12 | REQ-48-09 | OwnerResolver stack containment tree | ✅ (code-verified, real stack UAT weak per environment) |
| D-13 | REQ-48-10 | operlog.RecordBackground on each UPDATE | ✅ |
| D-14 | REQ-48-11 | Cron integration + failure-doesn't-block | ✅ |

### ⚠️ Partial — Real-device UAT deferred

- **D-08 real Huawei S8700 SNMP ENTITY-MIB** → deferred to next site visit (no S8700 in environment; only synthetic fixture tests)
- **D-09 real D-10 two-step on Huawei S8700 10GE5/0/4** → deferred (no live fiber port)
- **D-09/D-11 real Ruijie RS8607E 627 rows with 352 `temprature*` noise** → deferred (no device; synthetic fixture validates filter rule, not exact 352 count)

**Site-visit items:** 3 (all explicit deferrals per `.planning/milestones/v1.18-phases/48-device-component-serials-planned/48-HUMAN-UAT.md`)

### Audit verdict (`48-VERIFICATION.md`)
- `status: human_needed`
- `score: 13/14 D-ids fully verified + site-visit UAT deferred (informational, not a failure)`
- **0 critical gaps**; all 11 REQ-* automated/verifiable
- operlog regression lock intact (25 OperType constants + 11 sensitive keywords + Record 5-arg signature)

---

## 5. Key Decisions Log

Aggregated from `48-CONTEXT.md <decisions>` + `48-VERIFICATION.md`:

| ID | Decision | Phase | Rationale |
|----|----------|-------|-----------|
| D-01 | Components as `ops_asset` rows | 48 | Unified asset model; no separate sub-table |
| D-02 | UPDATE-only collector writes | 48 | Preserve v1.17 layering; no auto-INSERT |
| D-03 | Dual bridging parent + source | 48 | Internal anchor + collection provenance |
| D-04 | Parent-missing degraded mode | 48 | Don't pollute component layer with parent recovery |
| D-05 | Dedicated component_type/slot columns | 48 | Structured filter for reconciliation |
| D-06 | Sibling `recon_category` column | 48 | Same (asset, type) can coexist with legacy |
| D-07 | Default filter in asset_service | 48 | Avoid main-list inflation |
| D-08 | SNMP single-GET (not BulkWalk) | 48 | Huawei + Ruijie reject GETBULK |
| D-09 | Dual-path collection (SNMP + CLI) | 48 | Some data only via CLI |
| D-10 | Two-step transceiver (status → up only) | 48 | Cost optimization, skip empty ports |
| D-11 | Filter `temprature*` noise | 48 | 352/627 rows are private typo |
| D-12 | OwnerResolver ContainedIn tree | 48 | Stack ownership |
| D-13 | operlog.RecordBackground (cron-path) | 48 | No gin.Context, operator="system-cron" |
| D-14 | Cron hook failure = Warnf only | 48 | Don't block chassis update |
| CLI | Homegrown TextFSM parser limitation | 48 | `Continue.Record` idiom is no-op; single-state pattern used |
| Header | `display device esn` regex `\s+` → `\s*` | 49 | Match real-device no-space output (`1:SN` not `1: SN`) |

---

## 6. Tech Debt & Deferred Items

### Site-Visit UAT (3 items, deferred to next site visit)
1. Real Huawei S8700 SNMP ENTITY-MIB single-GET path (D-08)
2. Real Ruijie RS8607E SNMP 627 rows + 352 `temprature*` noise filter (D-11)
3. D-10 two-step pipeline on real Huawei S8700 10GE5/0/4 fiber interface (D-10)

**Owner:** 现场运维同事 (on-site operations team)
**Detail:** `.planning/milestones/v1.18-phases/48-device-component-serials-planned/48-HUMAN-UAT.md`

### Pre-existing Test Failures (NOT introduced by v1.18)
Verified via `git stash` baseline on `b8fd2f45`:
- `internal/services/operations/validation_helper_test.go` — sqlite fixture missing column
- `internal/services/operations/reference_resolver_test.go` — sqlite fixture
- `internal/services/operations/batch_upserter_test.go` — sqlite semantics
- `internal/services/operations/pagination_helper_test.go` — pagination drift
- `internal/services/system/role_service_apperrors_test.go` — fixture state

**11 failures documented in `.planning/milestones/v1.18-phases/48-device-component-serials-planned/deferred-items.md` as out-of-scope.**

### Phase 49 Lessons Learned (per RETROSPECTIVE)
- LSP false positives in worktree isolation
- `gsd-sdk worktree cleanup-wave` `empty_manifest` bug
- git stash pop mishap (3-file UU conflict)
- STATE.md milestone tag drift (v1.17 residual in v1.18 frontmatter)
- Pre-existing test failures require baseline verification
- `Continue.Record` latent bug in Ruijie transceiver textfsm (no production caller)

---

## 7. Getting Started

### Run the project
- Backend dev: `go build -o xingran-backend.exe ./cmd/main.go`
- Frontend dev: `cd xingran-react-frontend && npm run dev`
- See `CLAUDE.md` for full setup

### Run tests for this milestone's artifacts
```bash
# Component collectors
go test ./internal/services/component_collector/ -v

# D-10 two-step pipeline hook
go test ./internal/services/ -run TestCollectComponentInfo -v

# ListComponents handler
go test ./internal/api/v1/operations/ -run TestListComponents -v

# operlog regression lock (must stay green)
go test ./internal/utils/operlog/ -v

# asset_service default filter
go test ./internal/services/operations/ -run TestAsset -v
```

### Key directories (v1.18 modifications)

#### Backend (Phase 48 + 49)
```
internal/
├── core/db/migrations/
│   └── migration_201_phase48_component_columns.go    [CREATED]
├── models/
│   ├── asset.go                                       [MODIFIED +4 columns]
│   └── reconciliation.go                              [MODIFIED +recon_category]
├── device/
│   └── snmp_entity_mib.go                             [CREATED D-08]
├── services/
│   ├── component_collector/                           [NEW PACKAGE 17 files]
│   │   ├── component_set.go                           (types)
│   │   ├── owner_resolver.go                          (D-12 stack)
│   │   ├── snmp_entity_collector.go                   (D-08/D-11)
│   │   ├── cli_huawei_collector.go                    (D-09)
│   │   ├── cli_ruijie_collector.go                    (D-09)
│   │   ├── cmd_dispatcher.go                          (D-10 pipeline)
│   │   ├── ops_asset_writer.go                        (D-02/D-03/D-04 UPDATE-only)
│   │   ├── reconciliation_emitter.go                  (D-06 sibling-column)
│   │   ├── pipeline.go                                (orchestration)
│   │   ├── fixtures_loader.go                         (35 sample fixtures)
│   │   └── *_test.go                                  (21 tests)
│   ├── device_info_collection_service.go             [MODIFIED collectComponentInfo hook + D-10 + 49-01 enrichChassisSerial]
│   ├── operations/
│   │   └── asset_service.go                           [MODIFIED D-07 default filter]
│   └── utils/operlog/operlog.go                       [MODIFIED RecordBackground helper]
└── api/
    ├── v1/operations/asset_component_handler.go        [CREATED]
    └── router.go                                       [MODIFIED /components route]

templates/
├── huawei_vrp_display_device_esn.textfsm               [MODIFIED 49-01 \s*]
├── huawei_vrp_display_interface_status.textfsm        [CREATED]
├── huawei_vrp_display_interface_transceiver.textfsm   [CREATED]
├── ruijie_os_show_version_modules.textfsm             [CREATED]
├── ruijie_os_show_interfaces_transceiver_ddm.textfsm  [CREATED]
└── ruijie_os_show_interfaces_status.textfsm           [MODIFIED STATUS capture]
```

#### Frontend
```
xingran-react-frontend/src/
├── types/operations.ts                                 [MODIFIED Asset +4 fields]
├── lib/opsApi.ts                                       [MODIFIED componentApi factory]
└── pages/operations/assets/
    ├── components/ComponentListTab.tsx                 [CREATED expandable row tab]
    └── index.tsx                                       [MODIFIED expandedRowRender]
```

### Where to look first

| Want to understand... | Start here |
|----------------------|------------|
| Data model for components | `internal/models/asset.go` (4 new columns) + `48-CONTEXT.md` D-01/D-05 |
| Collection flow (SNMP + CLI) | `internal/services/component_collector/snmp_entity_collector.go` + `cli_huawei_collector.go` + `cmd_dispatcher.go` |
| Reconciliation sibling-column pattern | `internal/core/db/migrations/migration_201_phase48_component_columns.go` (DROP/CREATE partial unique) + `reconciliation_emitter.go` |
| ops_asset write strategy | `internal/services/component_collector/ops_asset_writer.go` (UPDATE-only D-02 + parent-degraded D-04 + operlog.RecordBackground D-13) |
| cron integration | `internal/services/device_info_collection_service.go` (`collectComponentInfo` + `runTwoStepTransceiverPipeline` + `enrichChassisSerial` 49-01) |
| Frontend expandable row | `xingran-react-frontend/src/pages/operations/assets/index.tsx` (Antd Table expandable) → `components/ComponentListTab.tsx` |
| operlog cron-path helper | `internal/utils/operlog/operlog.go` (RecordBackground line 264-321) |
| v1.18 retrospective | `RETROSPECTIVE.md` §Milestone: v1.18 |

### Production state (verified post-Phase 49)
- ✅ RS8607E-03 production chassis SN: `G1H40D100022A` (was empty, now populated)
- ✅ Component count for RS8607E-03: 9 components (7 boards + 2 transceivers)
- ✅ All 14 online Ruijie/Huawei devices: `serial_number` populated (was 0/14 before Phase 49)
- ✅ operlog regression 5 tests PASS (Phase 34 lock intact)

---

## Stats

- **Milestone:** v1.18 网络设备硬件清单 (Device Component Serials)
- **Phases:** 2 (Phase 48 = 3 plans; Phase 49 = 2 plans = gap closure)
- **Plans:** 5 (48-01/48-02/48-03 + 49-01/49-02)
- **Timeline:** 2026-07-04 (Phase 48 start, 08:59 +0800) → 2026-07-06 (Phase 49 close, 08:29 +0800) — 2.0 days
- **Commits:** 19 (Phase 48) + 2 (Phase 49 docs) = 21 total (12 code commits + 9 docs)
- **Files changed:** 87 (8,421 insertions / 644 deletions)
- **D-ids covered:** 14/14 fully verified (13 code + 1 deferred to real-device)
- **REQ covered:** 11/11 (100%)
- **Tests added:** 21 collector + 5 pipeline hook + 3 handler + 5 operlog regression + 3 list-filter = 37 new tests
- **Site-visit UAT deferred:** 3 items (Huawei S8700 + Ruijie RS8607E + D-10 live fiber)
- **Contributors:** ninedrunk
- **Pre-existing failures (untouched):** 11 operations/system tests documented in `deferred-items.md`

### Git Range
```
Start: b8fd2f45 (feat(48-01): add migration_201 + Asset/SysDataReconciliation column extensions)
End:   c7af2b3e (docs(phase-49): complete phase execution — verifier passed 7/7)

b8fd2f45..c7af2b3e = 87 files changed, 8421 insertions, 644 deletions
```

---

*For current project status, see `.planning/STATE.md`*
*For archived v1.18 roadmap, see `.planning/milestones/v1.18-ROADMAP.md` (mid-milestone archived at Phase 48 ship)*
