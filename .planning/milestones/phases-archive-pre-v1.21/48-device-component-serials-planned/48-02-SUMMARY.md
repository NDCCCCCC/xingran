---
phase: 48-device-component-serials-planned
plan: 02
subsystem: component_collector (in-memory parsing only; no DB writes)
tags: [snmp, entity-mib, textfsm, cli-parsing, d-10-up-filter, d-11-temprature, d-12-stack, phase-48, wave-2]
requires:
  - 48-01 schema (parent_asset_id / source_device_id / component_type / component_slot on ops_asset; recon_category on sys_data_reconciliation) — shipped Wave 1
  - internal/device/snmp_client.go (Get/Walk API) — existing
  - internal/templates/textfsm.go (homegrown parser) — existing
provides:
  - internal/services/component_collector/ package (ComponentSet / Component / EntityRow / OwnedComponent types + OwnerResolver + EntityCollector + HuaweiCliCollector + RuijieCliCollector + GetCollectorCommands dispatcher + fixtures_loader)
  - internal/device/snmp_entity_mib.go (exported OID base constants + SNMPGetter interface + CountPhysicalEntitiesByClass + GetEntityAttrs helpers)
  - 6 new TextFSM templates (huawei_vrp_display_device_esn / huawei_vrp_display_interface_status / huawei_vrp_display_interface_transceiver / ruijie_os_show_version_modules / ruijie_os_show_interfaces_transceiver_ddm + modified ruijie_os_show_interfaces_status with STATUS capture)
  - Pure parsing API: ComponentSet in, ComponentSet out; no DB writes (Wave 3 wires pipeline)
affects:
  - templates/ruijie_os_show_interfaces_status.textfsm (added STATUS captured variable; previously discarded as \S+ — D-10 enabler)
  - No runtime side effects (no cron hook, no DB writer — pure library code)
tech-stack:
  added: []
  patterns:
    - SNMP single-GET loop on ENTITY-MIB (D-08 — Huawei S8700 rejects GetBulk)
    - Class-filtered Walk to bound the GET loop and avoid ~600 sensor rows on Ruijie
    - Homegrown textfsm parser (internal/templates/textfsm.go) — single-state Record-on-final-field pattern (Continue.Record idiom is a no-op in this parser, discovered during TDD RED)
    - Stub SNMPGetter interface for test injection without live gosnmp
    - D-10 two-step transceiver pipeline (status → transceiver) enforced at cmd_dispatcher level
key-files:
  created:
    - internal/services/component_collector/component_set.go
    - internal/services/component_collector/owner_resolver.go
    - internal/services/component_collector/owner_resolver_test.go
    - internal/services/component_collector/snmp_entity_collector.go
    - internal/services/component_collector/snmp_entity_collector_test.go
    - internal/services/component_collector/fixtures_loader.go
    - internal/services/component_collector/cli_huawei_collector.go
    - internal/services/component_collector/cli_huawei_collector_test.go
    - internal/services/component_collector/cli_ruijie_collector.go
    - internal/services/component_collector/cli_ruijie_collector_test.go
    - internal/services/component_collector/cmd_dispatcher.go
    - internal/device/snmp_entity_mib.go
    - templates/huawei_vrp_display_device_esn.textfsm
    - templates/huawei_vrp_display_interface_status.textfsm
    - templates/huawei_vrp_display_interface_transceiver.textfsm
    - templates/ruijie_os_show_version_modules.textfsm
    - templates/ruijie_os_show_interfaces_transceiver_ddm.textfsm
  modified:
    - templates/ruijie_os_show_interfaces_status.textfsm (added STATUS captured variable for D-10 filter)
decisions:
  - Export ENTITY-MIB OID base constants (OidEntPhysical*) so tests and the collector can build per-index OIDs without re-declaring them; lowercase aliases retained for internal use
  - Define SNMPGetter as an interface (not *device.SNMPClient directly) so unit tests inject stub clients — parseSNMPValue is package-private so a public type switch is the only testable surface
  - Single-state Record-on-final-field pattern for TextFSM templates — the homegrown parser's Continue.Record idiom is a no-op (the parser only recognises Continue / Record / Error / <StateName> transitions); discovered during the Huawei transceiver TDD RED phase
  - Huawei dual-class dedup is name-prefix based (first whitespace token) because Huawei reports the same engine as "LSG7SRUEX1C0 5" (Class=5 placeholder) + "LSG7SRUEX1C0 5 (Master)" (Class=9 populated)
  - Class=9 with non-"fan" name maps to ComponentTypeEngine (not ComponentTypeFan) — Huawei reports board engines under fan(9) per Pitfall 2; the dual-class row IS the engine, not a literal fan
  - Ruijie PSU workaround (Open Question 5): Class=6/port with Name prefix "power" is mapped to ComponentTypePower alongside the standard Class=8/powerSupply mapping
  - OwnerResolver canonical root = smallest chassis Index (deterministic, input-order-independent); 32-hop walk cap guards against ContainedIn cycles
metrics:
  duration: ~20min
  completed: 2026-07-04
  tasks: 4/4
  files-touched: 17 (16 created + 1 modified)
---

# Phase 48 Plan 02: Component Collectors (SNMP + CLI) Summary

Wave 2 of Phase 48 — pure parsing collectors that produce `ComponentSet` from SNMP ENTITY-MIB and CLI output. No DB writes, no cron hook — Wave 3 (48-03) wires the pipeline.

## What Was Built

**Package `internal/services/component_collector/`** (new — pure library code):

- **component_set.go** — type definitions: `Component`, `ComponentSet`, `EntityRow`, `OwnedComponent` plus `ComponentType*` and `Source*` string constants (D-05 enumeration: chassis/card/engine/power/fan/transceiver; Source enumeration: snmp/cli-huawei/cli-ruijie)
- **owner_resolver.go** — `OwnerResolver.ResolveOwnership` implements D-12 entPhysicalContainedIn tree walk: deterministic canonical-root selection (smallest chassis Index), stack-member flagging for slave chassis, cycle guard (32-hop cap + visited-set), orphan marking for missing parents
- **snmp_entity_collector.go** — `EntityCollector.Collect` implements D-08 single-GET pipeline: class-filtered Walk → 5 single GETs per index → D-11 temprature* filter (Class==8 && Name hasPrefix "temprature") → Huawei dual-class dedup (Pitfall 2) → OwnerResolver tree → ComponentSet
- **cli_huawei_collector.go** — `HuaweiCliCollector` parses `display device esn` (chassis SN cross-verify, tolerates V600R024C00 retired-command error path — Pitfall 3), `display interface status` (D-10 up-filter), `display interface transceiver` (Manu. Serial Number with D-10 up-filter post-parse)
- **cli_ruijie_collector.go** — `RuijieCliCollector` parses `show version` (one-shot chassis + 7 module slots), `show interfaces status` (D-10 up-filter), `show interface transceiver` (Vendor SN + DDM Bias/Tx/Rx with D-10 up-filter)
- **cmd_dispatcher.go** — `GetCollectorCommands(vendor, kind)` vendor dispatcher; transceiver kind returns D-10 two-step pipeline (`status` before `transceiver`) for both Huawei and Ruijie; nil for out-of-scope vendors (H3C/Maipu)
- **fixtures_loader.go** — `LoadFixture` + `CountFixtures` (dynamic `filepath.Glob` — never hardcodes the current 35-file count per RESEARCH WARNING 3)

**`internal/device/snmp_entity_mib.go`** (new):
- Exported OID base constants (`OidEntPhysicalSerialNum` / `OidEntPhysicalModelName` / `OidEntPhysicalClass` / `OidEntPhysicalContainedIn` / `OidEntPhysicalName`)
- `EntityAttrs` struct, `SNMPGetter` interface (minimal subset of `*SNMPClient` for test injection)
- `CountPhysicalEntitiesByClass` — single class-filtered Walk to bound the GET loop
- `GetEntityAttrs` — five single GETs per index with type-switch on `interface{}` (parseSNMPValue is package-private in snmp_client.go)

**TextFSM templates** (6 new + 1 modified):
- `templates/huawei_vrp_display_device_esn.textfsm` — chassis ESN; tolerates "Error: Unrecognized command" retired path
- `templates/huawei_vrp_display_interface_status.textfsm` — INTERFACE + STATUS for D-10 pre-filter
- `templates/huawei_vrp_display_interface_transceiver.textfsm` — INTERFACE + Manu. Serial Number per transceiver block
- `templates/ruijie_os_show_version_modules.textfsm` — state-machine parser: Start captures System serial number (chassis), transitions to Modules state to extract Slot/ModuleType/SN per module
- `templates/ruijie_os_show_interfaces_transceiver_ddm.textfsm` — Vendor SN + DDM Bias/Tx/Rx Power (referenced by `ParseTransceiverDDM` — WARNING 4 dead-code mitigation)
- `templates/ruijie_os_show_interfaces_status.textfsm` (MODIFIED) — added STATUS captured variable (previously discarded as `\S+`) so D-10 up-filter can be applied Go-side

## Commits

- `94cf234e` — feat(48-02): ComponentSet types + OwnerResolver (D-12 stack ownership) — Task 1
- `a9ccc3cc` — feat(48-02): SNMP ENTITY-MIB collector + fixtures_loader (D-08/D-11/D-12) — Task 2a
- `4d0316b0` — feat(48-02): Huawei CLI collector + 3 TextFSM templates (D-09/D-10) — Task 2b
- `0fa4941a` — feat(48-02): Ruijie CLI collector + cmd_dispatcher + D-10/D-09/DDM (WARNING 4) — Task 2c

## Verification

Per the plan's `<verification>` block:

- `go build ./...` — clean (exit 0)
- `go test ./internal/services/component_collector/ -v` — **15 tests PASS**:
  - 4 OwnerResolver tests (single-chassis, stack mode, orphan/cycle, canonical-root-picks-first)
  - 4 SNMP EntityCollector tests (Ruijie temprature filter, real-PSU retention, Huawei dual-class dedup, fixtures glob count)
  - 4 Huawei CLI tests (display device esn error+success paths, interface status, transceiver real-fixture 2 SFPs, D-10 down-port filter)
  - 4 Ruijie tests (show version modules 1 chassis + 7 slots, interfaces status, transceiver DDM WARNING 4, D-10 down-port filter) + GetCollectorCommands dispatcher (D-10 pipeline ordering)
- `go test ./internal/device/` — PASS
- `go test ./internal/services/portcollection/` — PASS (no regression)
- `go test ./internal/scheduler/... ./internal/websocket/...` — PASS
- Pre-existing operations/system failures verified NOT caused by Phase 48-02 (logged to deferred-items.md)

## TDD Gate Compliance

All 4 tasks marked `tdd="true"`. Each task followed RED → GREEN:

- **Task 1 RED:** owner_resolver_test.go failed to compile (undefined types) → **GREEN:** component_set.go + owner_resolver.go implemented, 4/4 tests PASS
- **Task 2a RED:** snmp_entity_collector_test.go failed to compile (undefined `device.OidEntPhysical*` constants + `NewEntityCollector`) → **GREEN:** snmp_entity_mib.go (exported OIDs) + snmp_entity_collector.go implemented, 4/4 tests PASS (2 tests needed expectation correction mid-GREEN: real-fixture component count arithmetic + Huawei dual-class row is engine not fan)
- **Task 2b RED:** cli_huawei_collector_test.go failed to compile (undefined `NewHuaweiCliCollector`) → **GREEN:** cli_huawei_collector.go + 3 templates implemented; transceiver template needed 3 iterations (Continue.Record is a no-op in this parser — switched to single-state Record-on-final-field pattern; `\s+` after colon fixed to `\s*` to match real sample byte layout)
- **Task 2c RED:** cli_ruijie_collector_test.go failed to compile (undefined `NewRuijieCliCollector` + `GetCollectorCommands`) → **GREEN:** cli_ruijie_collector.go + cmd_dispatcher.go + 3 templates implemented, 5/5 tests PASS first try

Per execute-plan.md TDD gate rules: RED phase verified for every task ✓; GREEN commit present for every task ✓. Refactor step not needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed two test-expectation errors during Task 2a GREEN**
- **Found during:** Task 2a GREEN run
- **Issue:** (a) `TestEntityCollectorRuijieFiltersTemprature` expected 5 components but the fixture (2 PSU + 1 fan + 1 module) actually produces 4 — arithmetic error in the test, not the implementation. (b) `TestEntityCollectorHuaweiDualClassDedup` expected the dual-class row to map to `ComponentTypeFan`, but per RESEARCH Pitfall 2 the dual-class row IS the engine (Class=9 with non-"fan" Name → `ComponentTypeEngine`); the test expectation was semantically wrong.
- **Fix:** Corrected the test arithmetic (5→4) and changed the dual-class test to expect `ComponentTypeEngine` (with a comment explaining why). No production code change.
- **Files modified:** `internal/services/component_collector/snmp_entity_collector_test.go`
- **Commit:** `a9ccc3cc`

**2. [Rule 1 - Bug] TextFSM transceiver template — fixed two parser-contract bugs**
- **Found during:** Task 2b GREEN run
- **Issue:** (a) Initial template used the `Continue.Record` idiom copied from `templates/ruijie_os_show_interfaces_transceiver.textfsm`, but the homegrown `internal/templates/textfsm.go` parser only recognises `Continue` / `Record` / `Error` / `<StateName>` transitions — `Continue.Record` is a no-op (this also means the existing ruijie template was effectively broken for multi-record extraction, but it had no production caller). (b) The SN regex used `\s+` after the colon, but the real Huawei sample has `:8000012000082` with no space after the colon.
- **Fix:** Switched to single-state template with `-> Record` on the SN line (variables persist across lines because the parser doesn't reset `CurrentRecord` between matches within the same state); changed `\s+` to `\s*` after the colon.
- **Files modified:** `templates/huawei_vrp_display_interface_transceiver.textfsm`
- **Commit:** `4d0316b0`

### Out-of-Scope Discoveries

Pre-existing operations + system test failures logged to `.planning/phases/48-device-component-serials-planned/deferred-items.md`. None reference any Phase 48-02 file. Categories: sqlite fixture drift (operations validators/reference resolver), pagination-helper constant drift, batch-upserter sqlite semantics, role-service-apperrors fixture state.

## Threat Mitigation (per plan's `<threat_model>`)

| Threat | Mitigation (per plan) | Implemented |
|--------|-----------------------|-------------|
| T-48-04 Tampering (CLI SN regex metachars) | TextFSM templates anchored `^...$$`; no eval/regex side-effects | ✓ All 6 templates use `^...$$` anchoring |
| T-48-05 DoS (627-row ENTITY-MIB) | Class-filtered Walk bounds GET loop; <=150 entities; 5 single GETs per index | ✓ `snmpEntityClasses` filter + `CountPhysicalEntitiesByClass` |
| T-48-06 Info Disclosure (temprature* filter drops real PSU) | Filter condition `Class==8 && HasPrefix("temprature")`; real PSU `power0/power1` retained | ✓ `TestEntityCollectorRuijieFiltersTemprature` + `TestEntityCollectorPreservesRealPowerSupply` |
| T-48-07 Elevation (SSH credential leak) | Reuse existing connection pool; no new credential path | ✓ No SSH code in this plan — Wave 3 wires pipeline |
| T-48-14 Info Disclosure (D-10 down-port leak) | `TestHuaweiTransceiverFiltersDownPorts` + `TestRuijieTransceiverFiltersDownPorts` hard tests | ✓ Both implemented + PASS; cmd_dispatcher enforces two-step pipeline ordering |
| T-48-15 Tampering (Ruijie _ddm.textfsm dead code — WARNING 4) | `TestRuijieCliParseTransceiverDDM` exercises `_ddm.textfsm` end-to-end | ✓ Test asserts Vendor SN + DDM evidence in `Component.Raw` |
| T-48-SC Tampering (no new packages) | N/A — no installs | ✓ `go.mod` unchanged |

## Known Stubs

None — this plan produces pure parsing types and templates. No UI render paths, no data sources wired to empty props. All `Component` instances produced by tests carry real fixture SNs.

## Threat Flags

None — no new endpoints, auth paths, or schema changes at trust boundaries. The plan introduces only in-memory parsing code; the DB writer / reconciliation emitter / cron hook all land in Wave 3 (48-03).

## Manual / UAT Coverage (deferred)

Per VALIDATION.md, the following cannot be automated and require user site visit:

| Item | Verification |
|------|--------------|
| Real Huawei S8700 SNMP ENTITY-MIB single-GET path | User site visit — `EntityCollector.Collect` against live device, observe ComponentSet |
| Real Ruijie RS8607E SNMP ENTITY-MIB (627 rows incl. 352 temprature* noise) | User site visit — verify D-11 filter reduces to <=20 components |
| Real Huawei `display interface transceiver` on 10GE5/0/4 up port | User site visit — D-10 two-step pipeline against live device |
| Real Ruijie `show interface transceiver` on TenGigabitEthernet 1/47 | User site visit — D-10 + DDM extraction against live device |

## Self-Check: PASSED

Files verified (all created files exist on disk):

- FOUND: internal/services/component_collector/component_set.go
- FOUND: internal/services/component_collector/owner_resolver.go
- FOUND: internal/services/component_collector/owner_resolver_test.go
- FOUND: internal/services/component_collector/snmp_entity_collector.go
- FOUND: internal/services/component_collector/snmp_entity_collector_test.go
- FOUND: internal/services/component_collector/fixtures_loader.go
- FOUND: internal/services/component_collector/cli_huawei_collector.go
- FOUND: internal/services/component_collector/cli_huawei_collector_test.go
- FOUND: internal/services/component_collector/cli_ruijie_collector.go
- FOUND: internal/services/component_collector/cli_ruijie_collector_test.go
- FOUND: internal/services/component_collector/cmd_dispatcher.go
- FOUND: internal/device/snmp_entity_mib.go
- FOUND: templates/huawei_vrp_display_device_esn.textfsm
- FOUND: templates/huawei_vrp_display_interface_status.textfsm
- FOUND: templates/huawei_vrp_display_interface_transceiver.textfsm
- FOUND: templates/ruijie_os_show_version_modules.textfsm
- FOUND: templates/ruijie_os_show_interfaces_transceiver_ddm.textfsm
- FOUND: templates/ruijie_os_show_interfaces_status.textfsm (modified)

Commits verified:

- FOUND: 94cf234e (Task 1)
- FOUND: a9ccc3cc (Task 2a)
- FOUND: 4d0316b0 (Task 2b)
- FOUND: 0fa4941a (Task 2c)
