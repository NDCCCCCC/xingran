# Phase 48: 网络设备组件序列号采集 (Device Component Serials) - Research

**Researched:** 2026-07-04
**Domain:** Network device component inventory ingestion (multi-SN chassis), asset-model integration, v1.17 reconciliation R2 extension
**Confidence:** HIGH (all decisions locked in CONTEXT.md, real-machine samples verified, no external research needed)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions (D-01 ~ D-14 — DO NOT RECONSIDER)

| ID | Decision | Locked by |
|----|----------|-----------|
| **D-01** | Components ARE `ops_asset` rows; no separate child table | CONTEXT §D-01 |
| **D-02** | Collector is UPDATE-only on `ops_asset` (no INSERT); unmatched → reconciliation anomaly | CONTEXT §D-02 |
| **D-03** | Dual bridge: `parent_asset_id` (self-ref → ops_asset.id) + `source_device_id` (→ sys_network_device.id) | CONTEXT §D-03 |
| **D-04** | If parent switch not in `ops_asset`: `parent_asset_id` stays NULL; `source_device_id` still written; no anomaly | CONTEXT §D-04 |
| **D-05** | Add dedicated `component_type` + `component_slot` columns (NOT AttributeValue JSON); types: `chassis | card | engine | power | fan | transceiver` | CONTEXT §D-05 |
| **D-06** | Component anomaly = new category on `sys_data_reconciliation`; verify `reconciliation_detection.go` supports category extension, else fall back to A-F reuse | CONTEXT §D-06 |
| **D-07** | `asset_service.go` `List()` / `Statistics()` must default `component_type IS NULL` | CONTEXT §D-07 |
| **D-08** | Primary collector path = SNMP ENTITY-MIB **single-GET** (BulkWalk rejected on Huawei S8700); OIDs: `entPhysicalSerialNum(.11) / entPhysicalModelName(.13) / entPhysicalClass(.5) / entPhysicalContainedIn(.4) / entPhysicalName(.7)`; community from `sys_auth_credential.snmp_communities` (NEVER default `public`) | CONTEXT §D-08 |
| **D-09** | CLI supplement: Ruijie `show version` (one-shot chassis + 7 modules); Huawei `display interface transceiver` (transceiver + DDM); Huawei `display device esn` (chassis SN cross-verify); Huawei `display device` (hardware skeleton) | CONTEXT §D-09 |
| **D-10** | Transceivers: only `up` interfaces | CONTEXT §D-10 |
| **D-11** | Ruijie ENTITY-MIB has 352 `temprature*` (typo for temperature) noise nodes under `powerSupply(8)` class — filter by `Name hasPrefix "temprature"` | CONTEXT §D-11 |
| **D-12** | Stack (iStack / IRF / VSU): rebuild ownership via `entPhysicalContainedIn`; engine cards tagged Master/Slave from M1/M2; no stacked devices on this site — interface kept but UAT weak | CONTEXT §D-12 |
| **D-13** | operlog: module "资产管理", OperType 14 (OperTypeSync); non-sensitive fields → plain `operlog.Record()` | CONTEXT §D-13 |
| **D-14** | Depends on v1.17 reconciliation base (Phases 42-46, shipped 2026-07-03); Phase 48 = R2 physical-layer reconciliation's component-SN form | CONTEXT §D-14 |

### Claude's Discretion
- Frontend "从属组件清单" UI (table / card / tree) — UI designer chooses
- TextFSM template field positions / line numbers — already pinned in real samples
- Collect-scheduling cadence — reuse existing `DeviceInfoCollectionService` cron (no incremental collection in v1)

### Deferred Ideas (OUT OF SCOPE for v1)
- **H3C / Maipu** — no devices on site, must wait for other environment
- **Ruijie transceiver DDM TextFSM** — current RS8607E has all SFP slots empty (`transceiver is absent`), but note: 10/47 + 10/48 have populated `Transceiver Type : 10GBASE-SR-SFP+` + `Vendor Serial Number : G1PT549427799/G1PT54942708A` per the actual sample → DDM TextFSM is in fact buildable from existing sample
- **SNMPv3** — v2c sufficient; v3 (auth/priv) only on demand
- **Incremental collection scheduling** — reuse existing cron; no delta collector
- **Board migration event tracking** — board moves between devices not tracked in v1

---

<phase_requirements>
## Phase Requirements

Phase 48 has **no formal REQ-ID** in REQUIREMENTS.md (decision-only phase). All scope is defined by CONTEXT.md D-01 ~ D-14. Implied requirements:

| ID | Source | Behavior |
|----|--------|----------|
| REQ-48-01 | D-01, D-05 | `ops_asset` schema extended with 4 columns (`parent_asset_id` / `source_device_id` / `component_type` / `component_slot`) |
| REQ-48-02 | D-08 | SNMP ENTITY-MIB single-GET path collecting chassis/card/engine/fan SNs per authenticated network device |
| REQ-48-03 | D-09, D-10 | CLI supplement: Ruijie `show version` (board SNs) + Huawei `display interface transceiver` (transceiver SN + DDM) for in-use ports only |
| REQ-48-04 | D-11 | Ruijie ENTITY-MIB noise filter (drop `temprature*` under `powerSupply(8)`) |
| REQ-48-05 | D-02, D-03 | Component-SN → `ops_asset` lookup; on hit UPDATE 4 new columns; on miss emit reconciliation anomaly |
| REQ-48-06 | D-06 | Reconciliation anomalies filed under a new category usable in the conflict-type UI filter |
| REQ-48-07 | D-07 | `asset_service.List/Statistics` exclude `component_type IS NOT NULL` rows by default |
| REQ-48-08 | D-04 | Parent switch absent from `ops_asset` → `parent_asset_id` left NULL, no anomaly |
| REQ-48-09 | D-12 | Stack-device (`entPhysicalContainedIn`) ownership reconstruction hook retained (no UAT coverage) |
| REQ-48-10 | D-13 | operlog entries: "资产管理" / OperTypeSync on every component UPDATE |
| REQ-48-11 | D-14 | Cron integration with existing `DeviceInfoCollectionService` (no new scheduler entry) |
</phase_requirements>

---

## Summary

Phase 48 turns each SN-bearing component of a chassis switch (board / engine / PSU / fan / SFP) into an `ops_asset` row, with `parent_asset_id` pointing back to the chassis row, and emits reconciliation anomalies for SNs the asset system doesn't recognize. The hard parts are ALL already solved: real samples for Huawei S8700 V600R024C00 and Ruijie RS8607E are saved in `templates/samples/` (36 files), the AsyncAPI model and Column-extension pattern is in `internal/models/asset.go` (migration_148), the SNMP client is in `internal/device/snmp_client.go` (with `Get` + `Walk` + `GetBulk`), and the asset-side lookup is one method (`GetByDeviceSN`).

**Three implementation decisions drive the plan:**

1. **`conflict_type` column is `size:2`** (single-char `A`-`F` per `migration_168_reconciliation_tables.go`). **Cannot extend with `G`** without schema change. Component anomalies need either a new sibling column (`recon_category`) or a sibling table. RESEARCH recommends **`recon_category varchar(32)` add column** + drop unique constraint `uniq_recon_asset_type_open` and replace with `(asset_id, conflict_type, recon_category)`. Migration_NNN to land in the same Phase 48 plan.

2. **`asset_service.List()` and `Statistics()` do not currently exclude component rows.** D-07 mandates the default. Plan must add `component_type IS NULL` as a hardcoded where-clause at the **service layer** (not the controller), because the Excel `GetByDeviceSN` path calls service-layer methods that don't pass params and would otherwise need a new flag threaded through 6 call sites.

3. **SNMP must use `Get` (single OID) not `Walk`** — Huawei S8700 rejects GETBULK per RQ-001真机probe. Must loop `entPhysicalSerialNum.<n>` for `n := 1..N` (where N is learned from a single `Walk` on `.0.0` of `entPhysicalDescr` to count entities, then GET the rest one-by-one). Alternative: try `Walk` first, fall back to `Get`-loop on context-deadline. RQ-001 真机shows 627 rows for Ruijie / ~150 rows for Huawei — small, single-GET loop is fine.

**Primary recommendation:** ship as **3 plans** — `48-01` schema + reconciliation category migration + asset-list filter; `48-02` SNMP primary collector + TextFSM CLI supplements; `48-03` reconciliation anomaly generator + cron integration + operlog + frontend dependency view.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `ops_asset` schema extension (4 new cols) + reconciliation `recon_category` col | API / Backend (DB) | — | DDL only — `internal/core/db/migrations/` |
| Component→`ops_asset` lookup + UPSERT/UPDATE path | API / Backend (service) | — | Reuse `assetService.GetByDeviceSN` |
| SNMP ENTITY-MIB single-GET collector | API / Backend | — | `internal/device/` + new `internal/services/component_collector/` |
| TextFSM CLI collectors (per vendor) | API / Backend | — | New collector package, mirrors `portcollection/parser.go` |
| Reconciliation anomaly INSERT | API / Backend | — | Extend `reconciliation_detection.go` or add sibling service |
| Cron integration | API / Backend | — | Hook into `DeviceInfoCollectionService.worker` after `updateDeviceInfo` |
| Asset List/Statistics filter `component_type IS NULL` | API / Backend (service) | Frontend (defensive UI) | Hardcode at service layer per D-07 |
| operlog "资产管理" / OperTypeSync | API / Backend | — | `internal/utils/operlog` |
| "从属组件清单" frontend view | Frontend (React) | — | `xingran-react-frontend/src/pages/operations/assets/` |
| Reconciliation UI filter for component category | Frontend (React) | API / Backend (dict lookup) | Add to `asset_reconciliation_conflict_type` dict as new category (NOT conflict type) |

## Standard Stack

### Core (existing — DO NOT replace)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go gorm/driver/postgres | v1.5.9 | `ops_asset` + `sys_data_reconciliation` schema migration | Already shipped Phase 26 + 42-46; current production stack |
| gosnmp/gosnmp | v1.35.0 | ENTITY-MIB primary path | Real-machine-tested, no churn |
| google/uuid | v1.6.0 | `parent_asset_id` / `source_device_id` (varchar(64)) | Existing convention from `ops_info_points.*_id` patterns |
| net-snmp-style TextFSM (custom) | n/a | CLI output parser | `internal/templates/textfsm.go` + `Clone()` + template cache; do NOT pull in `github.com/cloudflare/go-textfsm` — project uses homegrown |

### Supporting (existing — no new install)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/collectors/asset_collector.go` | (existing) | Pattern reference for per-vendor command dispatch | Mirror for component collector |
| `internal/services/portcollection/parser.go` | (existing) | `getInterfaceCommand(vendor)` style dispatcher | Mirror for component `getCollectorCommand(vendor, kind)` |
| `internal/services/device_info_collection_service.go` | (existing) | Async worker pool (5 workers, queue=1000, recover-on-start) | **Hook into this worker's success path** instead of writing a new scheduler |
| `internal/utils/operlog` | (existing) | Phase 34 reference implementation | New collect events must call `operlog.Record(c, svc, db, "资产管理", operlog.OperTypeSync)` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New `internal/services/component_collector/` | Reuse `internal/collectors/asset_collector.go` | `asset_collector` ships single SN to `network_device.serial_number`; would have to refactor its whole pipeline — too risky. **Recommendation: write a sibling collector, share low-level helpers from `internal/device/`.** |
| Extend `sys_data_reconciliation.conflict_type` (size:2) | Add new `recon_category` column (size:32) | conflict_type is locked at 1 char (A-F). Adding `G`, `H` needs schema change AND unique index rebuild AND dict seed. **Use a sibling column to keep both collision types co-existing.** |
| Build new `DeviceComponentCollectionService` cron | Hook existing `DeviceInfoCollectionService` | Existing service already enqueues per-online-device and recovers on restart. Adding cron would duplicate throttling. **Append one method on existing service.** |
| Generate new `cmd/collect_component_samples/` tool | Resurrect deleted `cmd/snmp_entity_probe/` shell | Real samples already in `templates/samples/` — no need to re-collect. **For UAT debugging, use `redis-cli`-style one-shot bash.** |

**Installation:** No new external deps. Phase 48 is **purely code + 1-2 migration files + 4 TextFSM templates**.

**Version verification** — all existing libs as per `GOPROXY=https://goproxy.cn` lock; no upgrades required. Confirmed via inspection of `go.mod` (no churn in `gosnmp`/`go-textfsm` deps since Phase 43 R2 2026-06-27).

## Package Legitimacy Audit

> This phase installs **NO new external packages**. All work reuses existing `gosnmp` (locked at v1.35.0, in `go.mod` since 2026-04) and the project's homegrown `textfsm` (`internal/templates/textfsm.go`). No `[SLOP]` / `[SUS]` risk.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| (none — no new installs) | — | — | — | — | N/A | N/A |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                        ┌────────────────────────────────────────────┐
                        │           sys_network_device              │
                        │  (cron enqueues DeviceID per online row)   │
                        └────────────────┬───────────────────────────┘
                                         │  existing
                                         ▼
                  ┌────────────────────────────────────────────────┐
                  │  DeviceInfoCollectionService.worker(ctx, devID)  │
                  │   • CollectDeviceInfo() — single chassis SN     │
                  │   • updateDeviceInfo()                          │
                  │   ⮕ NEW: CollectComponentInfo() ← phase 48    │
                  │       (SNMP ENTITY-MIB Get-loop + CLI queries)  │
                  └────────────────┬───────────────────────────────┘
                                   │
                ┌──────────────────┴───────────────────────┐
                │                                          │
                ▼                                          ▼
  ┌──────────────────────────────┐    ┌──────────────────────────────────┐
  │ EntitySourceResolver        │    │ CliSourceResolver                │
  │ • snmp_client.Get(.11/.13/.5│    │ • getCollectorCmd(vendor, kind)  │
  │   /.4/.7) loop 1..N         │    │ • scrapli_wrapper.SendCommand    │
  │ • Community from            │    │ • per-vendor TextFSM parse       │
  │   sys_auth_credential.      │    │ • Huawei: display version/       │
  │   snmp_communities[0]       │    │   device esn/interface           │
  │ • Filter temprature* noise  │    │   transceiver                    │
  │ • Filter Class=8/powerSupply│    │ • Ruijie: show version /show     │
  │ • Group by entPhysicalCont- │    │   interfaces status /show        │
  │   ainedIn → tree of comps   │    │   interface transceiver          │
  └────────────────┬─────────────┘    │   (only up ports)                │
                   │                  └────────────┬─────────────────────┘
                   │                               │
                   └──────────────┬────────────────┘
                                  ▼
                ┌──────────────────────────────────────┐
                │   ComponentSet{Chassis, Components}  │
                │   type Component struct { SN, Type,  │
                │     Slot, Source, Raw }              │
                └────────────────┬─────────────────────┘
                                 │
                                 ▼
              ┌─────────────────────────────────────────┐
              │ ops_asset BulkLookup + WritePath       │
              │ • For each component, single            │
              │   assetService.GetByDeviceSN(sn)        │
              │ • On hit  → UPDATE parent_asset_id,     │
              │   source_device_id, component_type,     │
              │   component_slot (NOT devicesn!)        │
              │ • On miss → emit ReconciliationAnomaly  │
              │   into sys_data_reconciliation with     │
              │   recon_category='component_serial',    │
              │   conflict_type='B' (盘亏) or 'F'       │
              │   (盘盈, missing side) per D-06         │
              └────────────────┬────────────────────────┘
                               │
            ┌──────────────────┴───────────────────────┐
            ▼                                          ▼
   ┌─────────────────────┐                  ┌──────────────────────────┐
   │ ops_asset (updated) │                  │ sys_data_reconciliation  │
   │ • parent_asset_id   │                  │ • recon_category         │
   │ • source_device_id  │                  │ • conflict_type (A-F)    │
   │ • component_type    │                  │ • severity (low/crit)    │
   │ • component_slot    │                  │ • asset_id, raw_snapshot │
   └─────────────────────┘                  └──────────────────────────┘
```

### Recommended Project Structure (Phase 48 addition)

```
internal/
├── services/
│   ├── component_collector/                    (NEW)
│   │   ├── service.go                          // DeviceComponentCollector interface
│   │   ├── snmp_entity_collector.go            // SNMP ENTITY-MIB single-GET
│   │   ├── cli_huawei_collector.go             // Huawei TextFSM
│   │   ├── cli_ruijie_collector.go             // Ruijie TextFSM
│   │   ├── owner_resolver.go                   // entPhysicalContainedIn → tree
│   │   ├── ops_asset_writer.go                 // lookup + UPDATE-only + anomaly
│   │   └── service_test.go                     // unit-test for parser + lookup
│   │
│   └── device_info_collection_service.go       (MODIFY — 1 hook)
│       • after updateDeviceInfo(...) add:
│         s.collectComponentInfo(ctx, device, info)
│
├── device/
│   └── snmp_entity_mib.go                      (NEW) — high-level helpers
│       • WalkEntityIndex() — single Walk on .0.0 to count entities
│       • GetEntityAttrs(idx, attrs...) — batch Get loop
│
├── core/db/migrations/
│   └── migration_NNN_phase_48_ops_asset_components.go  (NEW)
│       • ALTER TABLE ops_asset ADD 4 columns
│       • ALTER TABLE sys_data_reconciliation ADD recon_category varchar(32)
│       • DROP uniq_recon_asset_type_open
│       • CREATE uniq_recon_asset_type_cat_open
│           (asset_id, conflict_type, recon_category) WHERE open
│       • Seed dict asset_reconciliation_recon_category
│           (component_serial / future expansion)
│
└── utils/operlog/
    └── (no change — used as-is per D-13)

templates/                                       (NEW TextFSM templates)
├── ruijie_os_show_version_modules.textfsm       (D-09: parse Slot/Role/ModuleType/SN)
├── huawei_vrp_display_interface_transceiver.textfsm  (Manu. Serial Number + DDM)
├── huawei_vrp_display_device_esn.textfsm        (cross-verify chassis SN)
└── huawei_vrp_display_device_board.textfsm     (already partially exist; finalize)

xingran-react-frontend/src/pages/operations/assets/   (NEW/extended view)
└── components/ComponentList.tsx                 (D-??: subordinate component sheet)
    • Calls /ops/asset/components?parentAssetId=:id
    • Table: ComponentType | Slot | DeviceSN | Model | Source | LastSeen
```

### Pattern 1: SNMP ENTITY-MIB single-GET loop (vs BulkWalk)

**What:** RFC 6933 ENTITY-MIB provides `entPhysicalSerialNum` per entity index. Huawei S8700 V600R024C00 accepts `Get` but rejects `GetBulk` (timed out per RQ-001).

**When to use:** Always for ENTITY-MIB on Huawei. Ruijie accepts both, but single-GET is uniform.

**Example:**
```go
// internal/services/component_collector/snmp_entity_collector.go
const (
    oidEntPhysicalSerialNum   = "1.3.6.1.2.1.47.1.1.1.1.11"
    oidEntPhysicalModelName   = "1.3.6.1.2.1.47.1.1.1.1.13"
    oidEntPhysicalClass       = "1.3.6.1.2.1.47.1.1.1.1.5"
    oidEntPhysicalContainedIn = "1.3.6.1.2.1.47.1.1.1.1.4"
    oidEntPhysicalName        = "1.3.6.1.2.1.47.1.1.1.1.7"
)

// CountPhysicalEntities does one Walk on a SAFE sub-OID to learn entity count N,
// then loops single GETs to pull all attributes. The Walk is bounded by class
// filter (e.g. only class=3,5,6,7,8,9 → avoids 600+ sensor rows on Ruijie).
func (c *EntitySourceResolver) CountEntities(ctx context.Context, classes []int) (map[int]struct{}, error) {
    snmpClient := c.snmpPool.Get(deviceID)
    defer snmpClient.Close()

    indices := map[int]struct{}{}
    err := snmpClient.Walk(oidEntPhysicalClass, func(oid string, val interface{}) bool {
        // val is int64 (parsed Integer) — entPhysicalClass is INTEGER enum
        class, ok := val.(int64)
        if !ok {
            return true // continue
        }
        for _, want := range classes {
            if int(class) == want {
                // extract index from OID suffix ".11.<idx>"
                idx := extractIndexFromOID(oid, oidEntPhysicalClass)
                indices[idx] = struct{}{}
                break
            }
        }
        return true
    })
    if err != nil {
        return nil, fmt.Errorf("walk entPhysicalClass: %w", err)
    }
    return indices, nil
}

// GetEntityAttrs does single GET per (idx, attr) pair.
func (c *EntitySourceResolver) GetEntityAttrs(ctx context.Context, idx int) (EntityAttrs, error) {
    snmpClient := c.snmpPool.Get(deviceID)
    var a EntityAttrs

    // Serial
    if v, err := snmpClient.Get(fmt.Sprintf("%s.%d", oidEntPhysicalSerialNum, idx)); err == nil {
        if s, ok := v.(string); ok { a.Serial = strings.TrimSpace(s) }
    }
    // Model
    if v, err := snmpClient.Get(fmt.Sprintf("%s.%d", oidEntPhysicalModelName, idx)); err == nil {
        if s, ok := v.(string); ok { a.Model = strings.TrimSpace(s) }
    }
    // ContainedIn
    if v, err := snmpClient.Get(fmt.Sprintf("%s.%d", oidEntPhysicalContainedIn, idx)); err == nil {
        if n, ok := v.(int64); ok { a.ContainedIn = int(n) }
    }
    // Name
    if v, err := snmpClient.Get(fmt.Sprintf("%s.%d", oidEntPhysicalName, idx)); err == nil {
        if s, ok := v.(string); ok { a.Name = strings.TrimSpace(s) }
    }
    return a, nil
}
```

**Source:** RFC 6933 ENTITY-MIB v4 + project `internal/device/snmp_client.go:153-252` (Get/GetNext/Walk/GetBulk primitives).

### Pattern 2: Vendor-dispatched TextFSM parsing

**What:** CLI commands differ per vendor. Existing `internal/services/portcollection/parser.go` uses `getInterfaceCommand(vendor)` dispatcher; mirror for components.

**When to use:** Any vendor-command collection. Reuses `internal/templates/textfsm.go` (custom parser, no `external lib`).

**Example:**
```go
// internal/services/component_collector/cli_huawei_collector.go
func getCollectorCommands(vendor models.DeviceVendor, kind string) []string {
    switch vendor {
    case models.VendorHuawei:
        switch kind {
        case "chassis":   return []string{"display device esn"}
        case "transceiver": return []string{"display interface transceiver"}
        // display device / display version already collected by
        // DeviceInfoCollectionService — don't re-run.
        }
    case models.VendorRuijie:
        switch kind {
        case "module": return []string{"show version"}
        case "transceiver":
            return []string{"show interfaces status", "show interface transceiver"}
        }
    }
    return nil
}
```

**Source:** MEMORY `xingran-excel-import-column-position-matching` confirms TextFSM templates use the `internal/templates/` parser; `templates/huawei_vrp_display_device.textfsm` exists for Huawei skeleton.

### Anti-Patterns to Avoid

- **Don't call `Walk` with no class filter.** Ruijie returns 627 rows including 352 temperature-sensor noise + 4 sensor boards + multiple `CN6130` chipsets. RQ-001 confirms all these are non-asset noise. Either filter `Class` during the Walk (per `oidsNtype` formula `MatchType INTEGER { ... }`), or post-filter in Go using `D-11` rule.

- **Don't `INSERT INTO ops_asset` from collector.** D-02 is explicit. The whole point of the architecture is the asset system's source-of-truth stays Excel import + parent lookup; collector only writes relational/metadata columns.

- **Don't extend `conflict_type` with `G`/`H`.** Column is `varchar(2)`. Adding a new category needs either a sibling `recon_category` column (RECOMMENDED) or replacing the dictionary entirely.

- **Don't write a new cron.** D-11 of CLAUDE.md "operlog" + D-14 of CONTEXT.md make it clear: hook into `DeviceInfoCollectionService.worker` after the existing chassis SN write. The new collector runs in the SAME 5-worker async pool.

- **Don't load Ruijie `show hardware`.** Confirmed `% Incomplete command.` on RS8607E. Use `show version` exclusively.

- **Don't trust Huawei `display elabel`.** V600R024C00 outputs `[ArchivesInfo Version]` skeleton with empty `BarCode=`; command retired. Use `display device` skeleton + ENTITY-MIB for actual SNs.

- **Don't fetch real `snmp_community` from environment.** Real-value string lives only in `sys_auth_credential.snmp_communities[]` column. `core.Database.go:786-808` converts `snmp_community` legacy to array — but read `.snmp_communities[0]` only when population confirmed; fall back to credential if empty.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SNMP OID walker | Loop GET per index manually for > 50 entities | `snmp_client.Get` loop WITH class-filtered `Walk` first (Pattern 1) | Conformance to RFC 6933 + matches real-tested Ruijie / Huawei |
| Vendor command dispatch | Hardcoded `if vendor == "huawei"` in service | `getCollectorCommands(vendor, kind)` dispatcher (Pattern 2) | Mirrors `portcollection/parser.go` style — already proven |
| ops_asset lookup by SN | New SELECT query | `assetService.GetByDeviceSN(sn)` (existing) | Single source of truth for SN uniqueness; already used by Excel import |
| OperLog recording | Build a custom logger | `internal/utils/operlog.Record(c, svc, db, "资产管理", operlog.OperTypeSync)` | CLAUDE.md "operlog convention" — mandatory helper, has regression test at `internal/utils/operlog/regression_test.go` |
| Cron scheduling | New robfig/cron entry | Hook into `DeviceInfoCollectionService.worker()` after existing `updateDeviceInfo` | Avoids double-throttling; reuses queue + recovery-on-start |
| TextFSM parsing | Hand-rolled regex on raw CLI output | `internal/templates/textfsm.go` + new `templates/huawei_vrp_display_interface_transceiver.textfsm` etc. | Project's homegrown parser is what `portcollection/` uses; consistent with `getDBFieldName`-style conventions |

**Key insight:** Phase 48 has unusually few "don't hand-roll" items because the architecture decisions were already made during explore (D-01 through D-14). Real complexity is in the **assembling + pipeline orchestration**, not in primitive operations.

## Runtime State Inventory

> **Not a rename / migration phase. Skip data-migration audit.** All new tables and columns default to `NULL` — no existing row changes. Listed for completeness:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `ops_asset` has 0 rows with `component_type` set (column doesn't exist yet) | DDL only; new column default NULL |
| Live service config | `sys_auth_credential.snmp_communities` populated per existing CLI/snmp use | Read-only — needed at runtime by new collector |
| OS-registered state | None — cron is internal `robfig/cron`, no systemd / pm2 entry | None |
| Secrets/env vars | None — `snmp_communities` is DB column already | None |
| Build artifacts | `gosnmp` binary is unchanged; no new deps | None |

**Nothing found in category:** stated explicitly — no DB migration touches pre-existing data.

## Common Pitfalls

### Pitfall 1: Ruijie ENTITY-MIB size + noise
**What goes wrong:** Single `Walk` on `entPhysicalDescr` returns 627 rows. 352 are `temprature*` typo noise nodes under `powerSupply(8)` class (CPU temperature telemetry masquerading as physical entities).
**Why it happens:** Ruijie RGOS extends standard ENTITY-MIB with vendor-specific sensors; OID `.1.3.6.1.2.1.47.1.1.1.1.4.5` (Class 8) is overloaded for telemetry, not only power.
**How to avoid:** Per D-11, the collector filters `Class == powerSupply(8) && Name hasPrefix "temprature"` before parsing. Also filter empty `SerialNumber`.
**Warning signs:** Component counts wildly inflate; `temprature*` or `tempratureSensor*` appearing in component list.

### Pitfall 2: Huawei ENTITY-MIB duplicate entries (module class=5 vs class=fan=9)
**What goes wrong:** Huawei S8700 reports the SAME engine/board TWICE: once as `module(5)` (slot placeholder, SN empty) and once as `fan(9)` (the fan-shaped classification table, SN populated). Treating both as real assets produces duplicates with NULL SN.
**Why it happens:** Huawei ENTITY-MIB implementation dual-entitizes active boards (one in the module tree, one in the class tree) for vendor logic; the populated one lives under class 9.
**How to avoid:** Always prefer the row with non-empty `SerialNum`. If both are populated and disagree, prefer `entPhysicalContainedIn=1` (direct chassis child) over `entPhysicalContainedIn=k` (transitive). RQ-001 table shows the rule.
**Warning signs:** Component count > expected (e.g. 16 instead of 6 for S8700 chassis with 4 LPU + 2 MPU).

### Pitfall 3: HW `display device slot N` returns empty `BarCode=`
**What goes wrong:** Confirmed in `huawei_10_62_25_253_display_device_slot_1.txt` (sample line 8: `BarCode=` is blank). Parser would emit empty SN.
**Why it happens:** V600R024C00 retired `display elabel`; only the bare `ArchivesInfo Version` skeleton remains, with all `BarCode/Item/Description` empty.
**How to avoid:** Use `display device esn` for chassis SN (single line `ESN of chassis 1:NNNN`). For board SNs, use **ENTITY-MIB only**. Do not parse `display device slot N` textfsm output.
**Warning signs:** TextFSM returns record with `Slot=1` but `SN=""`.

### Pitfall 4: `deviceService.GetByDeviceSN` returns nil silently
**What goes wrong:** `asset_service.GetByDeviceSN` returns `(nil, nil)` on record-not-found (line 168-172 of `asset_service.go`). The check pattern `if asset == nil` works, but mixing with `(nil, err)` confuses callers.
**Why it happens:** Intentional — not-found is not an error, and Excel import needs fast existence checks. But collectors must distinguish "not found" (emit anomaly) from "DB error" (retry/crash).
**How to avoid:** Wrap call: `if asset == nil { emit anomaly } else { UPDATE }`. Never treat `<nil, nil>` as a hard error.
**Warning signs:** Anomalies being emitted for SNs that are clearly already in the table (caching issue or stale read).

### Pitfall 5: `display interface transceiver` on a port with NO SFP
**What goes wrong:** For a `down` fiber port, the command enters an interactive prompt or returns nothing parseable. Collecting without filtering (D-10) wastes time / crashes the parser.
**Why it happens:** Huawei / Ruijie only emit `transceiver information:` block when an SFP is plugged; on empty cages the command stalls or returns blank.
**How to avoid:** D-10 first runs `show interfaces status` / `display interface status`, filters by `status=up && type=fiber`, then calls transceiver on that subset only. Confirmed by Ruijie sample: 1/25-1/44 (24 ports) all say `the transceiver is absent!` — must skip these.
**Warning signs:** TextFSM records with `SN=""` or no records returned for many ports.

### Pitfall 6: `conflict_type varchar(2)` hard-coded size
**What goes wrong:** `sys_data_reconciliation.conflict_type` is `size:2`. Adding `G` or `H` for component-category triggers `GORM AutoMigrate` constraint check or PG `value too long for type character varying(2)` on insert.
**Why it happens:** v1.17 originally designed for 6 user/IP/asset conflict types only.
**How to avoid:** Add a sibling column `recon_category varchar(32)` with dictionary `asset_reconciliation_recon_category` (default `'component_serial'`). Use the existing `conflict_type` to capture the high-level A-F shape (apply existing semantics — e.g. "missing declared" maps to `C类-物理有责无` or new code). The combined `(conflict_type, recon_category)` is the filterable dimension. Drop the old partial unique `uniq_recon_asset_type_open` and rebuild as `(asset_id, conflict_type, recon_category) WHERE open`.
**Warning signs:** Plan tries to set `conflict_type='G'` against the existing column.

### Pitfall 7: stack/iStack/IRF in `entPhysicalContainedIn`
**What goes wrong:** Member switch contains chassis of OTHER member via `entPhysicalContainedIn`. Treating one device as the only "root" misses components on stacked slave.
**Why it happens:** With stacking, root chassis contains its local slots, but stacked slaves' chassis appear as children of `0` (logical stack) — entity tree is non-trivial.
**How to avoid:** Per D-12, keep an `OwnerResolver` interface that:
  1. picks the row with `entPhysicalClass=3 (chassis)` AND `entPhysicalContainedIn=0` as the canonical root
  2. all other chassis rows → flagged as "stacked"
  3. ALL components contained in stacked chassis get `parent_asset_id` pointing to the canonical root's `ops_asset.id` (not the stacked chassis's)
  4. emit 1 reconciliation anomaly per stacked chassis row that's missing from `ops_asset`
**Warning signs:** Phase 48 plan doesn't reference `entPhysicalClass=3 + ContainedIn=0` selection logic.

### Pitfall 8: real `snmp_communities` empty for legacy credentials
**What goes wrong:** Migration `database.go:786-808` converts legacy `snmp_community` string to `snmp_communities[]` only for newly-written credentials. Old credentials (pre-Phase-26) may have only the singular column.
**Why it happens:** Legacy schema migration path. Confirmed via `internal/core/db/database.go` snippet in canonical_refs.
**How to avoid:** Collector's community resolution: try `sys_auth_credential.snmp_communities[0]`, else fall back to `snmp_community` legacy column.
**Warning signs:** All SNMP requests return `noSuchObject` for credential rows older than migration 026-148.

## Code Examples

Verified patterns from existing code:

### 1. SNMP single-GET on snmp_client (existing)
```go
// Source: internal/device/snmp_client.go:153-185
func (c *SNMPClient) Get(oid string) (result interface{}, err error) {
    c.mu.RLock(); defer c.mu.RUnlock()
    // ... connectLocked + defer close ...
    resp, err := c.client.Get([]string{oid})
    if err != nil { return nil, fmt.Errorf("SNMP GET失败: %w", err) }
    if len(resp.Variables) == 0 { return nil, fmt.Errorf("没有返回数据") }
    return parseSNMPValue(resp.Variables[0]), nil
}
```
Phase 48 collector uses `c.Get(fmt.Sprintf("%s.%d", oidBase, idx))` in a loop.

### 2. Asset lookup by DeviceSN (existing, will reuse)
```go
// Source: internal/services/operations/asset_service.go:158-176
func (s *assetService) GetByDeviceSN(ctx context.Context, deviceSN string) (*models.Asset, error) {
    if deviceSN == "" { return nil, apperrors.ParamMissing("设备序列号") }
    var asset models.Asset
    err := s.db.WithContext(ctx).
        Where("devicesn = ? AND deleted_at IS NULL", deviceSN).
        First(&asset).Error
    if err == gorm.ErrRecordNotFound { return nil, nil }
    if err != nil { return nil, err }
    return &asset, nil
}
```

### 3. Workstation-attach to ops_asset discovery (existing — portcollection DISPATCH pattern)
```go
// Source: internal/services/portcollection/parser.go
// Pattern: one function per vendorxcommand, dispatched by switch.
func getInterfaceCommand(vendor models.DeviceVendor) []string {
    switch vendor {
    case models.VendorHuawei: return []string{"display interface brief"}
    case models.VendorRuijie: return []string{"show ip interface brief"}
    case models.VendorH3C:    return []string{"display ip interface brief"}
    }
    return nil
}
```

### 4. operlog call convention (existing, CLAUDE-mandated)
```go
// Source: internal/api/v1/system/ad_domain_handler.go (Phase 34 reference)
import "github.com/xingran-next/xingran-go-backend/internal/utils/operlog"

func (h *Handler) SyncSomething(c *gin.Context) {
    // ... business logic ...
    if err := svc.Sync(ctx); err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "资产管理", operlog.OperTypeSync)
    response.Success(c, syncResult)
}
```

### 5. worker hook for existing async pipeline (existing)
```go
// Source: internal/services/device_info_collection_service.go:212-249
func (s *DeviceInfoCollectionService) processTask(ctx context.Context, deviceID string) {
    // ... existing CollectDeviceInfo + updateDeviceInfo ...

    info, err := s.CollectDeviceInfo(ctx, &device)
    if err != nil { ... return }
    s.updateDeviceInfo(&device, info)   // ← already updates sys_network_device

    // ⮕ NEW (Phase 48):
    if err := s.CollectComponentInfo(ctx, &device, info); err != nil {
        applogger.Infof("组件采集失败 [设备=%s]: %v", device.DeviceName, err)
        // do NOT fail the task — chassis update still succeeded
    }

    s.markTaskSuccess(&task, info)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Walk all ENTITY-MIB rows blindly | Walk with class-filter → Get-loop per index | 2026-07-03 RQ-001 probe | 99% noise reduction; Ruijie 627→~16 meaningful rows |
| Reuse `ops_asset` Excel path with INSERT | New UPDATE-only path with `parent_asset_id` | 2026-07-03 user decision | Preserves "声明 vs 实物" layer separation |
| Independent scheduler for components | Hook `DeviceInfoCollectionService.worker` | Architecture pattern since Phase 11 (MAC collector) | Reuses 5-worker pool + queue recovery |
| extend `sys_data_reconciliation.conflict_type` (size:2) | new `recon_category` column | RESEARCH insight 2026-07-04 | Don't break existing A-F semantics / dict seed |

**Deprecated/outdated:**
- `display elabel` / `display elabel slot N` — Huawei V600R024C00 returns skeleton-only, all `BarCode=` empty. **Do not parse.**
- Ruijie `show hardware` — confirmed `% Incomplete command.` on RS8607E. **Do not call.**
- Huawei `display device manufacture-info slot N` — `Too many parameters` on V600R024C00. **Do not call.**
- Direct `INSERT INTO ops_asset` from collector (legacy `asset_collector.go` pattern at `internal/collectors/asset_collector.go` — only writes `sys_network_device.serial_number`, NOT `ops_asset`).

## Assumptions Log

> All claims below tagged `[ASSUMED]` — need user or UAT confirmation before becoming locked.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The existing `sys_data_reconciliation.conflict_type` column constraint is `varchar(2)` — verified via `migration_168_reconciliation_tables.go` default and `SysDataReconciliation` model tag at `internal/models/reconciliation.go:30` (`gorm:"size:2"`). | Common Pitfall #6 | If wrong (e.g. PG generated with type `text` instead), then `G`-style addition is cheap |
| A2 | `sys_auth_credential.snmp_communities` is consistently populated for all row credentials (not just post-Phase 26). | Pitfall #8 | If legacy credentials lack it, collector must fall back to legacy `snmp_community` string column or SNMP `default public` (rejected by D-08 — must add fallback path) |
| A3 | The Ruijie `show interface transceiver` 10/47 + 10/48 records (Ruijie sample shows `Vendor Serial Number : G1PT549427799`) cover enough to write a Ruijie DDM TextFSM template without another sampling run. | Deferred Ideas | If user's second-source environment disagrees on field shape, template breaks on first run |
| A4 | Huawei V600R024C00 single-GET works for ENTITY-MIB across `entPhysicalSerialNum.1..N` (not just sample-tested N≤32) — extrapolation from one chassis. | Pitfall #1/2 | If larger S12700 multi-frame chassis has different behavior, plan needs a discovery pass before UAT |
| A5 | Adding `component_type IS NULL` filter at `assetService.List()` (NOT at the controller) doesn't break Excel import's `GetByDeviceSN` callers — checked that those paths return single asset, no list. | Architectural Map / D-07 | If a list caller exists outside service entry, Excel-rejected components still show in UI |

## Open Questions (RESOLVED — 全部并入 CONTEXT.md D-XX 或 plan task)

1. **Conflict-type semantics for component anomalies** — RESOLVED: D-06 锁定 conflict_type=F + recon_category=component_serial sibling 列方案
   - What we know: `sys_data_reconciliation.conflict_type` is `A-F` (A=match, B=phys no user, C=phys has user, D=mismatch, E=triple mismatch, F=missing).
   - What's unclear: For Phase 48, "盘盈" (实物有账无 = new component observed, no matching `ops_asset` row) doesn't fit A-F cleanly. Similarly "盘亏" (账有实物无 = component declared but collector no longer sees it).
   - Recommendation: treat both as **F类-缺数据** with `recon_category='component_serial'` distinct. Plan must add a sibling dict seed for 2 sub-cases if UI wants them, OR just call them "新增组件" / "消失组件" in the description, using the existing `resolution_note` field for human-readable text.

2. **operlog severity on missing-vs-orphan** — RESOLVED: 48-03 T1 ReconciliationEmitter action 固定 severity='medium'(双方向),severity_override 字段供后期 rule-based 调整
   - What we know: D-06 says component anomalies use existing category.
   - What's unclear: Should "盘盈" (实物有账无) — meaning someone added physical hardware without registering it in the asset system — be flagged as `severity=high` (compliance risk) or `severity=medium`?
   - Recommendation: **`medium` for both directions in v1**. The new `severity_override` field in `SysReconciliationException` (already in model) lets ops add rule-based severity shifts later.

3. **Number of `ops_asset` rows that become "components" after Phase 48** — RESOLVED: D-02 锁"采集只匹配不新建",Q3 audit pre-flight 由 D-02 决策消解(不存在采集自动建组件行,故无 retroactive 标记风险)
   - What we know: For 1 S8700 = chassis + 4×LPU + 2×MPU + 4×PSU + 2×FAN = 13 components; for 1 RS8607E = chassis + 7 board slots + 2 M1/M2 + 3 PSU + 2 fans = ~16 components.
   - What's unclear: Total `ops_asset` rows pre-import = ?. If user already imported components AS main devices in Excel, Phase 48 will mark them as components retroactively — could be confusing.
   - Recommendation: **Audit pre-flight** in Wave 0 (not visible to user): query `SELECT COUNT(*) FROM ops_asset WHERE devicesn IN (chassis_or_card_sn)` for the 2 sample devices' known SNs (12 from RQ-001 table) to estimate impact. If non-zero hits, plan must include a "data review" task before UPDATE.

4. **`component_type` enum persistence** — RESOLVED: 48-01 T1 action 采用 VARCHAR(32) 无 CHECK 约束,字典驱动 asset_component_type
   - What we know: D-05 lists 6 types: `chassis | card | engine | power | fan | transceiver`.
   - What's unclear: VARCHAR (open-ended future types) vs hardcoded enum with PG check constraint?
   - Recommendation: **VARCHAR(32)**, no PG enum type. Reason: dict-driven (`asset_component_type`) lets user add `sled` / `fabric` etc. without schema change. Migration_NNN just adds the column; no CHECK constraint.

5. **`entPhysicalClass=6` (port) handling** — RESOLVED: 48-02 T2a action 采用 Class ∈ {6,8} AND Name 前缀 power* → component_type='power'
   - What we know: Ruijie reports PSU as `class=port(6)` (`port(6) power0 RG-PA600I A82603150300065`), not `class=powerSupply(8)`. Sample table confirms.
   - What's unclear: Should PSU be re-mapped to `component_type='power'` only when `Class=8`, or also when `Class=6 && Name starts with "power"`?
   - Recommendation: **Combine**: Class ∈ {6,8} AND Name has prefix `power*` → `component_type='power'`. Re-checked against sample: `power0` / `power1` / `power2` are RG-PA600I PSUs. Document the rule in TextFSM macro.

## Environment Availability

> Per CLAUDE.md "Common Commands" — `go 1.24` is the standard toolchain. Real machines (华为 S8700 / 锐捷 RS8607E) are NOT in this environment; they are referenced for sample data only.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | migration + collector build | ✓ | 1.24.5 (per CLAUDE.md) | — |
| PostgreSQL 18 | `ops_asset` + `sys_data_reconciliation` schema | ✓ | 18 (per CLAUDE.md) | — (SQLite only for unit tests) |
| gosnmp | SNMP ENTITY-MIB collector | ✓ in go.mod | v1.35.0 | — |
| Internal `templates/textfsm.go` | TextFSM CLI parsing | ✓ | homegrown | — |
| Real Huawei S8700 device | UAT manual verification | ✗ | — | Use `templates/samples/` for parser unit-tests; UAT will revisit at user's site |
| Real Ruijie RS8607E device | UAT manual verification | ✗ | — | Use `templates/samples/` for parser unit-tests |
| Test database with seeded SNMP credentials | Integration test (mock collector) | ✓ | via `internal/services/testutil` (if exists) | Use a local credential record |

**Missing dependencies with no fallback:**
- Real devices for UAT. **This is the biggest risk**: Phase 48 cannot be fully UAT'd in this environment. The plan MUST include a "UAT deferred to site visit" acceptance criterion per D-09/10/11.

**Missing dependencies with fallback:**
- None — code-level dependencies are present.

## Validation Architecture

> Per `.planning/config.json` `workflow.nyquist_validation` not explicitly set to `false` → include section.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) + vitest (frontend) |
| Config file | None — `go test ./...` discovery, frontend has `npm run test` |
| Quick run command | `go test ./internal/services/component_collector/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-48-01 | 4 new cols + recon_category col exist on `ops_asset` / `sys_data_reconciliation` | unit (DB introspection) | `go test ./internal/services/component_collector/... -run TestSchemaMigration` | ❌ Wave 0 |
| REQ-48-02 | ENTITY-MIB single-GET loop parses Huawei/Ruijie sample ENTITY-MIB output correctly | unit (mocked SNMP) | `go test ./internal/services/component_collector/... -run TestSnmpEntityCollector` | ❌ Wave 0 |
| REQ-48-03 | TextFSM templates parse all 5 vendor CLI sample files | unit | `go test ./internal/services/component_collector/... -run TestTextFSMHuaweiTransceiver` | ❌ Wave 0 |
| REQ-48-04 | Ruijie `temprature*` filter drops the 352 noise nodes | unit (fixture injection) | `go test ./internal/services/component_collector/... -run TestRuijieNoiseFilter` | ❌ Wave 0 |
| REQ-48-05 | Component lookup + UPDATE-only behaviour (never INSERT) | unit | `go test ./internal/services/component_collector/... -run TestAssetWriterLookup` | ❌ Wave 0 |
| REQ-48-06 | Reconciliation anomaly row appears with `recon_category='component_serial'` | unit | `go test ./internal/services/component_collector/... -run TestReconciliationAnomaly` | ❌ Wave 0 |
| REQ-48-07 | `assetService.List()` excludes `component_type IS NOT NULL` rows | unit | `go test ./internal/services/operations/... -run TestAssetListExcludesComponents` | ❌ Wave 0 |
| REQ-48-08 | Parent-switch missing → `parent_asset_id` NULL, no anomaly on switch side | unit | `go test ./internal/services/component_collector/... -run TestMissingParentSwitch` | ❌ Wave 0 |
| REQ-48-09 | Stack-mode `entPhysicalContainedIn` tree resolution hook | unit (mock fixture) | `go test ./internal/services/component_collector/... -run TestStackContainment` | ❌ Wave 0 |
| REQ-48-10 | operlog entry written per component UPDATE | unit (logger spy) | `go test ./internal/services/component_collector/... -run TestOperlogRecorded` | ❌ Wave 0 |
| REQ-48-11 | Existing cron hooks collect components (not breaks chassis update path) | regression | `go test ./internal/services/... -run TestDeviceInfoCollection` | ❌ Wave 0 |

**Note:** every REQ requires a new test file. **Wave 0 is 11 tests + 1 fixture file**. Plan must include a `48-00-PLAN.md` for the test scaffold if not present.

### Sampling Rate
- **Per task commit:** `go test ./internal/services/component_collector/...` (component collector only)
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green + frontend `npm run test` + manual UAT site visit deferred

### Wave 0 Gaps

> **REVISION NOTE (gsd-plan-checker INFO 1, 2026-07-04):** The entry `templates/parser/samples_*.txt` (copies of `templates/samples/`) has been **removed** from this list. Plans read `templates/samples/*.txt` directly via `fixtures_loader.go` (no copies). Wave 0 tests are inlined as RED-first TDD in 48-01 T2 / 48-02 T1,T2a,T2b,T2c / 48-03 T1,T2 — no separate `48-00-PLAN.md` is created (gsd-plan-checker WARNING 2). See `48-VALIDATION.md` for the authoritative REQ→test map.

- [ ] `internal/services/component_collector/snmp_entity_collector_test.go` — mocked SNMP for single-GET loop (inlined in 48-02 T2a)
- [ ] `internal/services/component_collector/cli_huawei_collector_test.go` — fixture-driven TextFSM (inlined in 48-02 T2b)
- [ ] `internal/services/component_collector/cli_ruijie_collector_test.go` — fixture-driven TextFSM (inlined in 48-02 T2c)
- [ ] `internal/services/component_collector/ops_asset_writer_test.go` — UPDATE-only assertion (inlined in 48-03 T1)
- [ ] `internal/services/component_collector/owner_resolver_test.go` — containment tree (inlined in 48-02 T1)
- [ ] `internal/services/component_collector/reconciliation_emitter_test.go` — anomaly emission (inlined in 48-03 T1)
- [ ] `internal/services/component_collector/pipeline_test.go` — end-to-end pipeline (inlined in 48-03 T1)
- [ ] `internal/services/operations/asset_listfilter_test.go` — List/Statistics exclude component_type (inlined in 48-01 T2)
- [ ] Migration smoke test (PostgreSQL fixture, validates column existence after apply) — covered by UAT 步骤 2

## Security Domain

> Per CLAUDE.md "Security Considerations": SM2/SM4 are for request encryption; Phase 48 does NOT introduce any new auth or encryption boundary.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | — (no new endpoints; reusing existing `DeviceInfoCollectionService`) |
| V3 Session Management | no | — (cron, not user-driven) |
| V4 Access Control | partial | New component-anomaly UI must reuse existing `ops_asset:list` / `ops_asset:detail` permissions. No new perms |
| V5 Input Validation | yes | CLI raw output → TextFSM template parsing. Templates are the validation, but malformed input must not panic; use template's `^. -> Error` pattern (already standard in `huawei_vrp_display_device.textfsm`) |
| V6 Cryptography | no | — (no new secrets at rest) |

### Known Threat Patterns for this Stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|--------------------|
| CLI parsing injection (e.g. SN contains regex metachars) | Tampering | TextFSM templates use anchored `^...$$` patterns — no eval, no regex side-effects |
| SNMP query amplification | DoS | Single-GET loop bounded by entity count (max ~150 on tested devices); 5-worker pool already throttles |
| Reconciliation alert storm | DoS | D-11 partial unique index `uniq_recon_*` per (asset_id, conflict_type, recon_category) — repeated collector runs produce 1 open anomaly per (asset, type, category) |
| Untrusted `DisplayVersion` exec via collector | Elevation | `device.ExecuteOnDevice` already uses Scrapli SSH + locked credential — no new attack surface |

## Sources

### Primary (HIGH confidence)
- `templates/samples/huawei_10_62_25_253_*.txt` (18 files, ~16 KB total) — Huawei S8700 raw CLI
- `templates/samples/ruijie_10_62_63_21_*.txt` (6 files) — Ruijie RS8607E raw CLI  
- `templates/huawei_vrp_display_device.textfsm` — half-built skeleton template ready to extend
- `templates/ruijie_os_show_version.textfsm` — existing chassis-only version template
- `templates/ruijie_os_show_interfaces_transceiver.textfsm` — existing transceiver template
- `.planning/research/questions.md` (RQ-001, RQ-002) — locked design rationale
- `.planning/notes/260703-network-device-component-serials.md` — full design decision log
- `internal/services/operations/asset_service.go:158-176` — `GetByDeviceSN` interface
- `internal/services/device_info_collection_service.go:212-300` — async worker hooks
- `internal/device/snmp_client.go:153-287` — `Get`/`GetNext`/`Walk`/`GetBulk` API
- `internal/models/asset.go` (whole file) — `ops_asset` extension target
- `internal/models/reconciliation.go` (whole file) — `sys_data_reconciliation` + extension target
- `internal/services/asset/reconciliation_detection.go:79-90` — Layer 3 interface, conflict types
- `internal/services/portcollection/parser.go` — vendor command dispatch pattern
- `internal/core/db/migrations/migration_148_create_ops_asset_table.go` — `ops_asset` original DDL
- `internal/core/db/migrations/migration_168_reconciliation_tables.go` — reconciliation table DDL + `uniq_recon_asset_type_open` partial unique
- `internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go:79-101` — `asset_reconciliation_conflict_type` dict (A-F)

### Secondary (MEDIUM confidence)
- RFC 6933 ENTITY-MIB v4 spec — referenced by Huawei/H3C vendors but not directly cited (vendors cite it)
- Existing MEMORY entries on:
  - `xingran-gorm-sql-constraint-naming-conflict` — affects constraint naming for migration_NNN (use `uni_*_*_*` style for `uniq_recon_asset_type_cat_open`)
  - `xingran-migrations-no-sql-autoloader` — `cmd/main.go` must register `MigrateNNNComponentSerials` in explicit slice
  - `server-side-sort` — `assetService.List()` sort whitelist, must add `component_type` / `parent_asset_id` columns if frontend exposes them
  - `xingran-perm-namespace-split-readonly-page` — if new component-anomaly list page exists, ensure its perms match `ops:asset:*`

### Tertiary (LOW confidence)
- None — no external research performed; all decisions supported by in-repo evidence

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — no new external deps; everything reused from `go.mod` + project history
- Architecture: **HIGH** — locked by CONTEXT.md D-01 ~ D-14 + 6-round explore + 36 real samples
- Pitfalls: **HIGH** — every pitfall has a sample-file citation or code-line citation
- Reconciliation category extension: **MEDIUM** — A5 (sibling column vs extending `conflict_type`) is a derived answer based on column `size:2` constraint; user should sign off before migration_NNN freezes this
- Frontend "从属组件清单" UI shape: **LOW** — explicitly Claude's Discretion (no design pin)
- Stack-mode (iStack/IRF) UAT: **LOW** — no stacked devices on this site, interface kept but not exercised

**Research date:** 2026-07-04
**Valid until:** 2026-07-14 (10 days; no fast-moving deps, but Huawei/Ruijie firmware updates can alter CLI output)

---

## Plan Cut Recommendation

Based on the architecture diagram, dependency graph, and Wave 0 test list, Phase 48 should be split into **3 plans** (across **3 waves**):

- **Wave 1 — `48-01-PLAN.md` Schema + Asset-list filter + Reconciliation category**
  - Migration_NNN: ALTER `ops_asset` (+4 cols) + ALTER `sys_data_reconciliation` (+ recon_category) + DROP/RECREATE partial unique
  - `asset_service.go` `List()` + `Statistics()` add `component_type IS NULL` default filter
  - Model: extend `Asset` struct with the 4 new columns
  - Operlog R&D call sites: confirm wiring
  - Tests: schema migration test, list-filter test, statistic-filter test
  - *No runtime collector code yet.*

- **Wave 2 — `48-02-PLAN.md` Collectors (SNMP + TextFSM)**
  - New `internal/services/component_collector/` package
  - SNMP ENTITY-MIB single-GET loop with Ruijie noise filter (D-11)
  - Vendor-dispatched TextFSM collectors (Huawei display device esn, Ruijie show version, both transceivers)
  - TextFSM templates: `ruijie_os_show_version_modules.textfsm` (new), `huawei_vrp_display_interface_transceiver.textfsm` (new), `huawei_vrp_display_device_esn.textfsm` (new)
  - Unit tests with all 36 sample files as fixtures
  - *Still not wired to cron, no reconciliation emission.*

- **Wave 3 — `48-03-PLAN.md` Pipeline + Reconciliation + Operlog + Frontend view**
  - Hook into `DeviceInfoCollectionService.processTask()` after `updateDeviceInfo`
  - Owner resolver (entPhysicalContainedIn tree)
  - `ops_asset` UPDATE-only writer with `parent_asset_id` / `source_device_id` / `component_type` / `component_slot`
  - Reconciliation anomaly emission (`recon_category='component_serial'`)
  - operlog calls (OperTypeSync, "资产管理")
  - Operlog test confirmation
  - Frontend: `assets/components.tsx` view (read-only)
  - END-to-END integration test against locally-seeded `ops_asset` rows (2 sample devices' SNs)
  - UAT deferred — note in STATE.md that real-device UAT happens at user site visit

**Why not 2 plans:** the schema/migration + dual-path-collector + pipeline integration are 3 disjoint concerns with separate rollback semantics. Collapsing into 2 forces either combined schema+collector commit (medium-loud) or isolating pipeline (split the trust boundary wrong).

**Why not 4+:** the 3 above are atomic. Each is testable independently with the Wave 0 test scaffold.

---

*Researched: 2026-07-04 by Phase 48 research (per CLAUDE.md "operlog convention" and project structure)*
