---
phase: 56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09
plan: 05
subsystem: portwrite-e2e
tags: [e2e, filetransport, scrapligo, vlan, port-binding, huawei, ruijie, uat-deferral, documentation]
requires:
  - 56-04 (frontend Modals + API wrappers; v1.20.1 milestone frontend complete)
  - 56-03 (backend handler/router/audit; 2 HTTP endpoints)
  - 56-02 (service layer + 4 validators + PortResult.Extra)
  - 56-01 (vendor template 3×2 commands)
  - 54-01 (v1.19 fileTransportExecutor + e2eFixturePath + newTestService + mockCollectionSvc infrastructure)
provides:
  - "11 new TestE2E_* tests via scrapligo FileTransport covering 2 new actions × 2 vendors × variants"
  - "10 new test fixtures (5 Huawei + 5 Ruijie) replaying vendor byte streams"
  - "56-HUMAN-UAT.md site-visit deferral (12 items, verifier_status: human_needed)"
  - "API响应规范.md v1.20.1 section (2 endpoints + schemas + error codes + 3×2 command matrix)"
  - "CHANGELOG.md v1.20.1 entry (newest-first)"
  - "MILESTONES.md v1.20.1 SHIPPED entry"
affects:
  - "internal/services/portwrite/port_write_e2e_test.go"
  - "internal/services/portwrite/testdata/*.fixture"
  - "docs/API响应规范.md"
  - "CHANGELOG.md"
  - ".planning/MILESTONES.md"
  - ".planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md"
tech-stack:
  added: [] # zero new dependencies (go.mod / package.json diff empty)
  patterns:
    - "v1.19 fileTransportExecutor reuse (scrapligo FileTransport replay, no new infrastructure)"
    - "Ruijie fixtures use [Ruijie] system-view prompt (huawei_vrp platform prompt regex excludes spaces)"
    - "seedDeviceVendor helper (vendor override for Ruijie tests; seedPortAndDevice default stays Huawei)"
    - "MAC format locked per vendor CLI: Huawei/H3C = AA-BB-CC-DD-EE-FF; Ruijie = aabb.ccdd.eeff"
    - "v1.19 / v1.18 site-visit UAT deferral pattern (verifier_status: human_needed)"
key-files:
  created:
    - internal/services/portwrite/testdata/huawei_set_access_vlan_success.fixture
    - internal/services/portwrite/testdata/ruijie_set_access_vlan_success.fixture
    - internal/services/portwrite/testdata/huawei_port_binding_add_success.fixture
    - internal/services/portwrite/testdata/huawei_port_binding_add_with_mac_success.fixture
    - internal/services/portwrite/testdata/huawei_port_binding_remove_success.fixture
    - internal/services/portwrite/testdata/huawei_port_binding_remove_with_mac_success.fixture
    - internal/services/portwrite/testdata/ruijie_port_binding_add_success.fixture
    - internal/services/portwrite/testdata/ruijie_port_binding_add_no_mac_success.fixture
    - internal/services/portwrite/testdata/ruijie_port_binding_remove_success.fixture
    - internal/services/portwrite/testdata/ruijie_port_binding_remove_no_mac_success.fixture
    - .planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md
  modified:
    - internal/services/portwrite/port_write_e2e_test.go
    - docs/API响应规范.md
    - CHANGELOG.md
    - .planning/MILESTONES.md
decisions:
  - "Ruijie fixture prompts use [Ruijie] (system-view) instead of [Ruijie-GigabitEthernet 1/0/1] — fileTransportExecutor hardcodes huawei_vrp platform whose prompt regex ^[[\\w.\\-@/:]{1,63}]$ does not match spaces; scrapligo only needs A recognized prompt to advance, not the exact interface-view prompt (Rule 1 fix; real-device prompt with space verified at site-visit)"
  - "Huawei MAC format = AA-BB-CC-DD-EE-FF (per-byte hyphenated) NOT AABB-CCDD-EEFF as PLAN.md fixtures suggested — W1 SUMMARY decision lock + W2 test TestPortBinding_Success confirmed; PLAN.md fixture examples were non-canonical (Rule 1 fix)"
  - "Ruijie DB seed uses GE1/0/1 (normalized short form) — DevicePortStatus BeforeCreate hook normalizes GigabitEthernet1/0/1 -> GE1/0/1 on insert; expandInterfaceName then expands GE1/0/1 -> GigabitEthernet 1/0/1 for the rendered command (with space, Ruijie convention)"
  - "seedDeviceVendor helper added instead of modifying seedPortAndDevice — non-invasive: v1.19 tests keep Huawei default, Ruijie tests call seedDeviceVendor post-seed to UPDATE vendor column"
  - "12 site-visit UAT items (vs plan's 6+) — added Huawei V200R005 vs V200R022+ cross-firmware compatibility (RISK-03) + Ruijie port-security binding MAC column (RISK-02) + H3C user-bind no-static (RISK-01) as explicit verification items"
  - "Frontend type-check + build inherited from W4 (zero frontend changes in 56-05); confirmed vendor-react gzip 774.96 kB <= 776 kB baseline"
metrics:
  duration: "~25 min"
  completed: 2026-07-09
  tasks_complete: 5
  files_changed: 15
  commits: 4
  new_e2e_tests: 11
  new_fixtures: 10
  total_e2e_tests: 21
  vendor_react_gzip_kB: 774.96
---

# Phase 56 Plan 05: E2E + UAT Deferral + Documentation Summary

Closed the v1.20.1 milestone by extending the v1.19 e2e infrastructure with 11 new scrapligo FileTransport tests + 10 new fixtures for the 2 new write actions (`set_access_vlan` + `port_binding`) across Huawei + Ruijie vendors, plus the 56-HUMAN-UAT.md site-visit deferral (12 items) and 3-file documentation sync (API响应规范 / CHANGELOG / MILESTONES). The scrapligo FileTransport integration is now proven for the 2 new actions with the same evidence pattern as v1.19 (mock SSH in CI + real-device UAT deferred to next site visit).

## What Was Built

### Task 1 — 10 e2e fixture files created (commit `5c1e31ec`)
- 5 Huawei fixtures: `huawei_set_access_vlan_success` (3-cmd), `huawei_port_binding_add_success` (IP-only), `huawei_port_binding_add_with_mac_success`, `huawei_port_binding_remove_success`, `huawei_port_binding_remove_with_mac_success`
- 5 Ruijie fixtures: `ruijie_set_access_vlan_success` (3-cmd), `ruijie_port_binding_add_success` (with MAC), `ruijie_port_binding_add_no_mac_success`, `ruijie_port_binding_remove_success`, `ruijie_port_binding_remove_no_mac_success`
- Format follows v1.19 baseline (top-level prompt + screen-length 0 temporary + system-view + per-command response sequence + exit/quit + return)
- Huawei MAC format locked to `AA-BB-CC-DD-EE-FF` (Rule 1 fix — PLAN.md had non-canonical `AABB-CCDD-EEFF`)
- Ruijie prompts use `[Ruijie]` system-view form (Rule 1 fix — fileTransportExecutor hardcodes huawei_vrp platform whose prompt regex excludes spaces)
- Total fixtures: 17 (7 v1.19 + 10 v1.20.1)

### Task 2 — port_write_e2e_test.go extended with 11 new tests (commit `22522e09`)
- `TestE2E_SetAccessVlan_Huawei_Success` — 3-cmd link (interface + port link-type access + port default vlan 100) + INFRA-01 Extra[vlanId]=100 assertion
- `TestE2E_SetAccessVlan_Ruijie_Success` — 3-cmd link (interface + switchport mode access + switchport access vlan 100)
- `TestE2E_PortBinding_Huawei_Add_Success` — IP-only (user-bind static ip-address)
- `TestE2E_PortBinding_Huawei_Add_WithMac_Success` — MAC format conversion (AA:BB:CC:DD:EE:FF → AA-BB-CC-DD-EE-FF)
- `TestE2E_PortBinding_Huawei_Remove_Success` — undo user-bind IP-only
- `TestE2E_PortBinding_Huawei_Remove_WithMac_Success` — undo with MAC
- `TestE2E_PortBinding_Ruijie_Add_Success` — switchport port-security binding aabb.ccdd.eeff IP (Cisco H.H.H MAC format)
- `TestE2E_PortBinding_Ruijie_Add_NoMac_Success` — IP-only binding
- `TestE2E_PortBinding_Ruijie_Remove_Success` — no switchport port-security binding with MAC
- `TestE2E_PortBinding_Ruijie_Remove_NoMac_Success` — IP-only no-binding
- `TestE2E_SetAccessVlan_Huawei_PreStateMatch_NoOp` — pre-state VLAN match short-circuit (fileTransportExecutor not invoked; fixtureProvider panics as guard)
- Each test asserts Status/NoOp/CommandSent substrings + INFRA-01 Extra audit carrier + CollectDevice fire-and-forget refresh
- `seedDeviceVendor` helper added (non-invasive vendor override for Ruijie tests)
- File: 1059 lines (≥900 required); 21 total TestE2E_* (10 v1.19 + 11 v1.20.1)

### Task 3 — 56-HUMAN-UAT.md created (commit `9fbc905c`)
- 107 lines; `verifier_status: human_needed` in frontmatter
- 12 site-visit SSH verification items (≥6 required):
  - Huawei S8700 × 6: set_access_vlan PVID切换 / V200R005 兼容性 / V200R022+ 兼容性 / port_binding add IP-only / add with MAC / remove
  - Ruijie RS8607E × 4: set_access_vlan switchport mode/access / port_binding add with MAC (RISK-02) / add IP-only / remove
  - H3C Comware × 2 (conditional): user-bind no `static` (RISK-01) / set_access_vlan `port access vlan` keyword
- Each item: concrete SSH command + verification step + expected result + owner (现场运维同事)
- References v1.19 54-HUMAN-UAT.md + v1.18 48-HUMAN-UAT.md precedent
- Mock-e2e vs site-visit split documented (CI covers integration chain; site-visit covers firmware/prompt/space quirks)

### Task 4 — 3 documentation files synchronized (commit `ace6d9dc`)
- `docs/API响应规范.md` +v1.20.1 扩展小节: 2 endpoint signatures + `SetAccessVlanRequest`/`PortBindingRequest` schema + 4 sentinel error codes + 2 request examples + `PortResult.Extra` audit carrier spec + 3 vendors × 2 actions command template matrix + MAC format spec
- `CHANGELOG.md` +v1.20.1 entry (newest-first): 8+ bullets covering backend/frontend/testing/docs + Deferred section (12 site-visit + FUTURE-BATCH-05 batch path)
- `.planning/MILESTONES.md` +v1.20.1 SHIPPED entry: 5 waves + 5 key accomplishments + 12 site-visit UAT deferred + risk mitigations
- Zero modification to v1.19 entries (locked and correct)

### Task 5 — Full acceptance gate PASSED (no commit; gate-only task)
- `go build ./...` exit 0
- `go test ./internal/services/portwrite/... -count=1 -timeout 120s` exit 0 (21 e2e + service unit tests all green; 63s)
- `go test ./internal/services/portcollection/... ./internal/api/v1/network/... ./internal/utils/operlog/... -count=1` exit 0 (W1 vendor templates + W3 handlers + Phase 34 operlog regression all green)
- `npm run type-check` exit 0 (run from main repo; worktree frontend unchanged in 56-05)
- `npm run build` exit 0; **vendor-react gzip = 774.96 kB ≤ 776 kB baseline** (zero regression)
- `go.mod` / `go.sum` / `package.json` / `package-lock.json` diffs empty (zero new dependencies)
- Spot-checks: 17 fixtures / 11 new e2e tests / 56-HUMAN-UAT.md (1 verifier_status + 12 items) / API spec 7 matches / CHANGELOG 5 matches / MILESTONES 4 matches

## Verification Results

| Check | Expected | Actual | Status |
|-------|----------|--------|--------|
| `go build ./...` exit | 0 | 0 | PASS |
| portwrite full test suite | exit 0 | exit 0 (63s) | PASS |
| portwrite e2e count | 21 (10 v1.19 + 11 new) | 21 | PASS |
| portcollection tests | exit 0 | exit 0 | PASS |
| network handler tests | exit 0 | exit 0 | PASS |
| operlog regression | exit 0 | exit 0 | PASS |
| npm type-check | exit 0 | exit 0 | PASS |
| npm build | exit 0 | exit 0 (37s) | PASS |
| vendor-react gzip | ≤ 776 kB | 774.96 kB | PASS |
| fixture count | ≥ 13 | 17 | PASS |
| new e2e tests | ≥ 10 | 11 | PASS |
| 56-HUMAN-UAT.md verifier_status | 1 match | 1 | PASS |
| 56-HUMAN-UAT.md unchecked items | ≥ 6 | 12 | PASS |
| API spec v1.20.1 matches | ≥ 3 | 7 | PASS |
| CHANGELOG v1.20.1 matches | ≥ 5 | 5 | PASS |
| MILESTONES v1.20.1 matches | ≥ 3 | 4 | PASS |
| go.mod diff | empty | empty | PASS |
| package.json diff | empty | empty | PASS |
| operlog regression lock | intact | intact | PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Huawei MAC format in fixtures (AABB-CCDD-EEFF → AA-BB-CC-DD-EE-FF)**
- **Found during:** Task 1 (fixture creation)
- **Issue:** PLAN.md Task 1 fixture examples used Huawei MAC format `AABB-CCDD-EEFF` (per-pair hyphenated, grouped 4-4-4). This is non-canonical — the W1 vendor template (`toHuaweiH3CMACFormat` in vendor_port_template.go) and W2 test (`TestPortBinding_Success` asserting `mac-address AA-BB-CC-DD-EE-FF`) both lock the format to `AA-BB-CC-DD-EE-FF` (per-byte hyphenated, 6 pairs). Using the PLAN.md format would have caused the e2e test `CommandSent` assertion to fail.
- **Fix:** Used the canonical `AA-BB-CC-DD-EE-FF` format in all Huawei fixtures. This aligns with W1 decision lock (56-01-SUMMARY.md) and W2 unit test.
- **Files modified:** `huawei_port_binding_add_with_mac_success.fixture`, `huawei_port_binding_remove_with_mac_success.fixture`
- **Commit:** `5c1e31ec`

**2. [Rule 1 - Bug] Fixed Ruijie fixture prompt format (space-containing prompts hang FileTransport)**
- **Found during:** Task 2 (e2e test execution — Ruijie tests timed out)
- **Issue:** PLAN.md Task 1 Ruijie fixtures used `[Ruijie-GigabitEthernet 1/0/1]` (interface-view prompt with space). The `fileTransportExecutor` hardcodes the `huawei_vrp` scrapligo platform, whose configuration prompt regex is `^[[\w.\-@/:]{1,63}]$` — this character class does NOT include space. Consequently scrapligo could not recognize the space-containing prompt as a prompt, and `channel.ReadUntilFuzzy` hung indefinitely waiting for a recognizable prompt.
- **Fix:** Ruijie fixtures use `[Ruijie]` (system-view prompt, no space, matches huawei_vrp regex) as the response to all commands. The rendered command `interface GigabitEthernet 1/0/1` (with space) is still written to FileTransport correctly; scrapligo only needs A recognized prompt to advance, not the exact interface-view prompt. The real-device prompt with space (`[Ruijie-GigabitEthernet 1/0/1]`) is verified at site-visit UAT (RISK-02 item).
- **Files modified:** `ruijie_set_access_vlan_success.fixture`, `ruijie_port_binding_add_success.fixture`, `ruijie_port_binding_add_no_mac_success.fixture`, `ruijie_port_binding_remove_success.fixture`, `ruijie_port_binding_remove_no_mac_success.fixture`
- **Commit:** `22522e09`

**3. [Rule 1 - Bug] Fixed Ruijie DB seed interface name (GigabitEthernet1/0/1 → GE1/0/1)**
- **Found during:** Task 2 (e2e test execution — initial Ruijie test hang diagnosis)
- **Issue:** PLAN.md Task 2 acceptance criteria specified `seedPortAndDevice` interface name `GigabitEthernet 1/0/1` (with space, Ruijie convention). However, `DevicePortStatus` has a `BeforeCreate` GORM hook that normalizes interface names on insert (`NormalizeInterfaceName` collapses full names to short uppercase form). So `GigabitEthernet1/0/1` was stored as `GE1/0/1`. The initial seed value was redundant — the hook normalizes regardless.
- **Fix:** Ruijie tests seed `GE1/0/1` (the normalized form, same as Huawei tests). `expandInterfaceName` then expands `GE1/0/1` → `GigabitEthernet 1/0/1` (with space, Ruijie branch) for the rendered command. This is consistent with the production code path and the Huawei test pattern.
- **Files modified:** `internal/services/portwrite/port_write_e2e_test.go` (5 seed call sites)
- **Commit:** `22522e09`

**4. [Process] Frontend gate inherited from W4 (zero frontend changes in 56-05)**
- **Found during:** Task 5 (acceptance gate)
- **Issue:** The worktree does not have `node_modules` (git worktrees don't copy untracked files). Running `npm run type-check` / `npm run build` from the worktree failed (`tsc not found`). 56-05 makes zero frontend file changes (confirmed via `git diff --name-only 26e10860 HEAD | grep xingran-react-frontend` = empty).
- **Fix:** Ran the frontend gate from the main repo checkout (`D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend`), which has `node_modules` installed. The worktree's committed frontend source is identical to W4's (post-merge base `26e10860`), so the W4 gate state carries forward. Both `npm run type-check` and `npm run build` passed; vendor-react gzip = 774.96 kB ≤ 776 kB baseline.
- **Files affected:** none (56-05 makes zero frontend changes)

## Known Limitations

**Batch path for `set_access_vlan` / `port_binding` deferred to FUTURE-BATCH-05.** Per the W2 design decision and 56-04-SUMMARY.md, the batch path for the 2 new actions is single-port only in v1.20.1. The `BatchWriteRequest` type has 4 optional fields (`VLANID` / `BindOp` / `IPAddress` / `MACAddress`) reserved for future batch support, but `BatchWritePorts` itself does not dispatch to the 2 new actions. BulkWriteDrawer is unchanged. This is tracked as FUTURE-BATCH-05.

**FileTransport cannot fully simulate real-device firmware behavior.** The mock e2e (11 TestE2E_*) proves the integration chain (RenderCommand → executeWrite → SendConfigs → parseConfigError → CollectDevice refresh) works end-to-end. However, FileTransport cannot reproduce: (a) real-device prompt timing / byte encoding differences; (b) cross-firmware command compatibility (Huawei V200R005 vs V200R022+); (c) Ruijie real-device prompts with spaces (`[Ruijie-GigabitEthernet 1/0/1]`). These are deferred to the 12 site-visit UAT items in 56-HUMAN-UAT.md.

## STRIDE Threat Mitigation Summary

All 5 threats from the plan's `<threat_model>` are mitigated:
- **T-56-W5-01** (Tampering — fixture content mismatch): 11 e2e tests assert exact `CommandSent` substrings (`port default vlan 100`, `user-bind static ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF`, `switchport port-security binding aabb.ccdd.eeff 10.62.25.5`, etc.); fixture drift breaks assertions deterministically.
- **T-56-W5-02** (Info Disclosure — sentinel names in docs): API 响应规范.md documents the 4 sentinel error names; these are internal Go identifiers that appear in HTTP response messages and oper_param anyway. No secrets exposed.
- **T-56-W5-03** (Repudiation — site-visit UAT never closed): v1.19 / v1.18 precedent; 现场运维同事 closes items in future site visit; 56-HUMAN-UAT.md is the deferral record.
- **T-56-W5-04** (Tampering — spec drift): 3 documentation files created from the actual code (port_write_handler.go, port_write_service.go, vendor_port_template.go); URL paths / OperType values / error code names / request schemas verified against code.
- **T-56-W5-SC** (Supply chain): `go.mod` / `package.json` diffs empty — zero new dependencies.

## Self-Check: PASSED

**Files verified present:**
- FOUND: `internal/services/portwrite/port_write_e2e_test.go` (1059 lines)
- FOUND: `internal/services/portwrite/testdata/huawei_set_access_vlan_success.fixture`
- FOUND: `internal/services/portwrite/testdata/ruijie_set_access_vlan_success.fixture`
- FOUND: `internal/services/portwrite/testdata/huawei_port_binding_add_success.fixture`
- FOUND: `internal/services/portwrite/testdata/huawei_port_binding_add_with_mac_success.fixture`
- FOUND: `internal/services/portwrite/testdata/huawei_port_binding_remove_success.fixture`
- FOUND: `internal/services/portwrite/testdata/huawei_port_binding_remove_with_mac_success.fixture`
- FOUND: `internal/services/portwrite/testdata/ruijie_port_binding_add_success.fixture`
- FOUND: `internal/services/portwrite/testdata/ruijie_port_binding_add_no_mac_success.fixture`
- FOUND: `internal/services/portwrite/testdata/ruijie_port_binding_remove_success.fixture`
- FOUND: `internal/services/portwrite/testdata/ruijie_port_binding_remove_no_mac_success.fixture`
- FOUND: `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md` (107 lines)
- FOUND: `docs/API响应规范.md` (v1.20.1 section)
- FOUND: `CHANGELOG.md` (v1.20.1 entry)
- FOUND: `.planning/MILESTONES.md` (v1.20.1 entry)

**Commits verified in git log:**
- FOUND: `5c1e31ec` (Task 1 — 10 fixtures)
- FOUND: `22522e09` (Task 2 — 11 e2e tests + Ruijie fixture prompt fix)
- FOUND: `9fbc905c` (Task 3 — 56-HUMAN-UAT.md)
- FOUND: `ace6d9dc` (Task 4 — API spec + CHANGELOG + MILESTONES)

**Test gates verified:**
- `go build ./...` exit 0
- `go test ./internal/services/portwrite/... -count=1` exit 0 (21 e2e + service tests)
- `go test ./internal/services/portcollection/... ./internal/api/v1/network/... ./internal/utils/operlog/... -count=1` exit 0
- `npm run type-check` exit 0
- `npm run build` exit 0; vendor-react gzip 774.96 kB ≤ 776 kB
