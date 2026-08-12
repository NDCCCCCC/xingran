# Research Summary — v1.19 网络设备写命令 (Network Device Write Operations)

**Project:** XingRan-Next (xingran-go-backend)
**Domain:** Enterprise IT operations / network device management (SSH write path)
**Researched:** 2026-07-06
**Confidence:** HIGH (architecture/integration) / MEDIUM (vendor-specific write quirks)
**Replaces:** v1.5 SUMMARY.md (MAC 地址历史数据管理)

---

## Executive Summary

v1.19 closes the "read+write" loop on XingRan-Next's network device management system. v1.18 (shipped 2026-07-04, Phase 48) added SSH-based read collection via `scrapli/scrapligo v1.3.3`; v1.19 adds the **mutating counterpart** — Web-driven port config push (shutdown/undo, description, dot1x enable/disable) against Huawei VRP, H3C Comware, and Ruijie RGOS devices, with a re-collection trigger and full audit trail.

**Critical insight:** this is a **composition milestone, not an infrastructure milestone**. All required SSH write primitives are already in place from v1.18:
- `ScrapliWrapper.SendConfig` (`internal/device/scrapli_wrapper.go:567`)
- `ConfigExecutionService` persistence shape (`internal/services/config_execution_service.go:277`)
- `DeviceInfoCollectionService.Enqueue` (the v1.18 hook v1.19 calls after success)
- `operlog.Record/RecordWithBody` (Phase 34)
- `RequirePermissions` middleware (`pkg/middleware/permission.go`)

**Zero new Go modules required.** The work is concentrated in three new files (a service, a vendor→template map, a handler) plus targeted edits to routing, permission constants, and a new `network:port:write` namespace.

**Primary risks** are NOT "SSH works" (it does) but **semantic correctness and auditability under failure**:
- SSH transport success ≠ device acceptance (must parse `% Error:` markers)
- Port writes need before/after state snapshots for compliance
- operlog's 11 sensitive-keyword filter will over-mask legitimate port descriptions
- Batch operations must detach from HTTP request context to survive `Core.Close()`'s 30s shutdown deadline

**Recommended approach:** thin, opinionated milestone with hardcoded vendor templates (per PROJECT.md locked decision "落地为先"), serial fail-fast batch semantics, a new `sys_port_write_audit` table for unmasked audit detail, and 5 build-ordered phases (W1 templates → W2 service → W3 router/handler → W4 frontend → W5 E2E).

---

## Stack Additions

**Zero new Go module dependencies** (HIGH confidence). Composable from existing v1.18 primitives:

| Component | Where | v1.19 Usage |
|-----------|-------|-------------|
| `scrapli/scrapligo v1.3.3` | SSH/Telnet | `SendConfig` already exercised by `ConfigExecutionService` |
| `ConfigExecutionService` | `internal/services/config_execution_service.go:277` | Persistence shape reference |
| `DeviceInfoCollectionService.Enqueue(deviceID)` | v1.18 hook | Async re-collection trigger after write success (1-2s latency, v1.18 D-09 dedup is safety net) |
| `DeviceCredentialHelper` | `internal/services/device_credential_helper.go` | Resolve device → credential with default fallback |
| `operlog.Record / RecordWithBody / RecordBackground` | `internal/utils/operlog` | Per-port `Record`; batch-level `Record` with summary params |
| `pkg/permission/config.go` | `pkg/permission` | Add `NetworkPortWrite = "network:port:write"` |
| `portcollection` package | `internal/services/portcollection` | Owns port domain; v1.19's `PortWriteService` lives here or as sibling |

**Frontend:** zero new deps. Reuses antd 6.1 Table actions, Modal/Drawer, `@/lib/api.ts` wrapped POST, React Query 5.90.12 mutations.

**Optional Scrapli upgrade:** v1.4.x picks up queue-panic mitigation + `GetPrompt() (string, error)` signature. Low risk, can be separate commit.

---

## Feature Categories

### Table Stakes (P1, required for v1.19 launch) — HIGH confidence

- 二次确认弹窗 with required "操作原因" (5-200 chars)
- 操作结果回传 — toast + SSH output preview
- 操作审计日志 — module name + `OperTypeStatus` / `OperTypeUpdate` / `OperTypeBatch`
- 权限控制 `network:port:write` (NEW, separate from `network:port:query`)
- 单端口精确操作 (shutdown/undo_shutdown/description/dot1x_enable/dot1x_disable)
- 多端口批量操作 (serial + fail-fast) — multi-select + Drawer + progress
- 改后采集触发 — `Enqueue(deviceID)` after success
- 失败点定位 — per-port error array in batch result
- 前置状态校验 — read `admin_status`/`dot1x_enabled`, warn if already target
- 超时/中断可见 — default 30s per-port, UI countdown

### Differentiators (P2, v1.19.x follow-up)

- 命令预览 (Command Preview) — show vendor→CLI string in collapsed panel
- 操作历史查看 (Operation History per Port) — link to `sys_oper_log` filtered by module
- 批量冲突解决 — `skipped` array for already-target-state ports
- 设备不可达预检 — 1s ping before commit
- dry-run 模式 — render but don't send

### Anti-Features (explicitly out of scope)

- **自动回滚 (Automatic Rollback)** — reverse-op planning has too many failure modes
- **定时写操作 (Scheduled Writes)** — overnight ops lack response; use work-order system
- **多用户并发写仲裁 (Concurrent Write Arbitration)** — last-write-wins acceptable for MVP
- **写命令中转执行 (Local Buffer & Replay)** — retry of `shutdown` is dangerous
- **跨厂商同命令 (Cross-vendor Unified Command)** — precision loss; vendor→template map is locked
- **实时写命令流 (Live Command Stream)** — scrapligo is request/response, not stream
- **AI 智能推荐 (AI Recommend)** — write ops require deterministic commands
- **改前快照/回滚点 (Pre-change Snapshot)** — use existing backup system separately
- **写入中的取消 (Cancel Mid-execution)** — only effective before SSH send

---

## Architecture Integration (HIGH confidence)

**Component boundaries** (5-phase build order, strict dependencies):

```
internal/
├── api/v1/network/
│   ├── port_write_router.go          (NEW, ~40 LOC)
│   ├── port_write_handler.go         (NEW, ~200 LOC)
│   └── network_router.go             (MODIFY — append /ports/write group)
├── services/portcollection/
│   ├── port_write_service.go         (NEW — interface + impl + DI)
│   ├── vendor_port_template.go       (NEW — map[Vendor][Action]fn)
│   └── batch.go                      (NEW — serial fail-fast orchestrator)
├── core/db/migrations/
│   └── migration_NNN_port_write_menu.go  (NEW — sys_menu + GrantNewMenuToRolesHavingParent)
pkg/permission/config.go              (MODIFY — NetworkPortWrite constant)
xingran-react-frontend/src/
├── lib/api/networkApi.ts             (MODIFY — 5 wrapped functions)
└── pages/network/ports/components/
    └── BulkWriteDrawer.tsx           (NEW, ~200 LOC)
```

**Data flow (per-port write)**:
```
Frontend Drawer → POST /network/ports/write/{shutdown|...} → Router (RequirePermissions)
  → Handler binds request → service.WritePort(ctx, req)
  → Service: load device+credential → render vendor command
  → pool.GetConnection + defer conn.ReleaseRef() (F-14 fixed pattern)
  → wrapper.SendConfig → parse response for `% Error:` markers
  → on success → collectionSvc.Enqueue(deviceID) (fire-and-forget)
  → Handler → operlog.Record(... OperTypeStatus/Update ...)
  → response.Success
```

**Batch flow** (serial + fail-fast):
- `BatchWritePorts` iterates ports
- On first failure: return partial result `{succeeded: [...], failed: [...], skipped: [...]}`
- Per-port `defer ReleaseRef()` to avoid pool starvation
- Per-port timeout 30s
- **Detached context** with 30min deadline (avoid Core.Close 30s trap)
- 50-port hard cap (Pitfall 11)
- `Enqueue` called per device (v1.18 D-09 dedup prevents queue duplication)
- Operlog: ONE row with `OperTypeBatch` + summary params — NOT per-port

**Vendor command template shape** (hardcoded Go map, locked):
- Huawei: `interface {iface}\n shutdown` / `undo shutdown` / `description {text}` / `dot1x enable` / `undo dot1x enable`
- H3C: identical to Huawei (shared VRP heritage)
- Ruijie: Cisco-style — `interface {iface}\n shutdown` / `no shutdown` / `description {text}` / `dot1x port-control auto` / `no dot1x port-control`

---

## Critical Pitfalls (Top 7)

| # | Pitfall | Mitigation |
|---|---------|------------|
| 1 | **SSH transport success ≠ device acceptance** — `SendConfig` returns result text but does NOT parse `% Error:` / `Unrecognized command` / `Illegal` markers | Add `parseConfigError(result)` to distinguish `transport_error` from `device_rejected`. Without this, operlog shows "shutdown succeeded" but port is unchanged |
| 2 | **No before/after state snapshot for audit** — `sys_oper_log.oper_param` is write-only | New `sys_port_write_audit` table with `BeforeValue` / `AfterValue` / `CommandSent` / `DeviceResponse` / `Status` |
| 3 | **Vendor command syntax subtlety breaks silently** — H3C accepts `GE0/0/1` shorthand while Huawei may require `GigabitEthernet0/0/1`; Ruijie uses `configure terminal` not `system-view`; dot1x keywords differ | Hardcoded `vendor→template map` with **per-vendor unit tests** is the locked mitigation |
| 4 | **Operlog over-masking hides audit value** — 11 mandatory sensitive keywords match on JSON key name; `description` like `DMZ-port-for-key-service` gets masked to `******` | `sys_port_write_audit.CommandSent` is the unmasked source of truth; `sys_oper_log` records only high-level summary |
| 5 | **Batch execution exceeds 30s Core.Close deadline** — `Core.Close()` 30s shutdown can fire mid-batch, leaving devices in half-configured state | Detach from `c.Request.Context()`, use `context.WithTimeout(context.Background(), 30*time.Minute)` |
| 6 | **Connection pool exhaustion** — 50-port batch held for 5 minutes holds 1/50 of pool; v1.18 cron collection starts failing with "连接池已满" | Per-port `Acquire/ReleaseRef` cycle (NOT per-batch), `defer conn.ReleaseRef()` after each port |
| 7 | **No batch size cap** — operator can select 1000 ports; 1000 × 2s = 33min, exceeds 30min context | Hard cap `maxBatchSize = 50`, soft warning at 20; maxConcurrentBatches per operator |

---

## Implications for Roadmap

### 5-Phase Build Order (recommended)

| Phase | Goal | Files | Addresses |
|-------|------|-------|-----------|
| **50** W1 — Vendor Templates + Unit Tests | Zero-dep contract | `vendor_port_template.go` + 12+ tests | FEATURES P1 vendor→command map; PITFALLS #3 |
| **51** W2 — PortWriteService + Mock Tests | Service layer with DI + mocks | `port_write_service.go` + `batch.go` + tests | FEATURES P1 backend; PITFALLS #1, #7 |
| **52** W3 — Router/Handler/Operlog/Permission/Migration | HTTP + audit | `port_write_router.go` + `port_write_handler.go` + migration | FEATURES P1 permissions/operlog; PITFALLS #5, #9, #16, #17 |
| **53** W4 — Frontend Drawer + Progress Dialog + API Wrappers | User-facing UI | `BulkWriteDrawer.tsx` + 5 wrapped API fns | FEATURES P1 frontend; PITFALLS #8, #13 |
| **54** W5 — E2E + Real-Device UAT + Documentation | Validation | Updated `docs/API响应规范.md` + `50-HUMAN-UAT.md` | All 4 prior phases validated on real devices |

**Phase ordering rationale:**
- Templates first (W1) — load-bearing contract, zero external deps, catches vendor syntax early
- Service before HTTP (W2 before W3) — service signature is what HTTP binds to; mocks are cheapest
- Backend before frontend (W3 before W4) — API contract must be stable
- E2E last (W5) — only after W1-W4 land can real-device UAT be planned efficiently

### Research Flags by Phase

- **W1 (Templates)**: standard pattern (Go map + table-driven tests). Skip research.
- **W2 (Service)**: Handler-Service pattern established (Phase 34). Skip research.
- **W3 (Router/Handler)**: Needs research on operlog module name conflict + menu attachment point.
- **W4 (Frontend)**: Light research on novel batch progress UI (antd Drawer + Table progress pattern).
- **W5 (E2E UAT)**: Depends on actual S5700/S5735/RS8607E hardware availability. Site visit unblocks.

### Gaps to Address

1. **Operlog module name conflict** — recommend `"端口管理"` (matches existing `port_handler.go`; new module name pollutes audit search)
2. **Menu attachment point** — under existing "端口管理" parent menu to avoid menu sprawl
3. **Vendor command coverage for firmware variants** — MVP locks one entry per `(vendor, action)`; resolve via real-device UAT
4. **Batch resume semantics** — defer to v1.19.x; MVP returns `failed_ports` array, operator manually re-issues
5. **Auto-rollback policy** — PROJECT.md explicitly out of scope. `sys_port_write_audit.BeforeValue` enables manual revert
6. **Critical port policy** — defer to v1.19+; MVP uses single `network:port:write` perm

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| **Stack** | HIGH | v1.18 read path + Scrapli verified; `SendConfig` already in use by `ConfigExecutionService` |
| **Features** | HIGH (P1) / MEDIUM (vendor templates) | operlog + permission + DeviceExecutor + Scrapli all v1.18-shipped; vendor syntax needs real-device UAT |
| **Architecture** | HIGH | 5-phase build order mirrors ARCHITECTURE.md; SSH reuse locked by F-14 fix; operlog follows Phase 34 conventions |
| **Pitfalls** | HIGH (general) / MEDIUM (vendor-specific) | Codebase-verified pitfalls reliable; vendor firmware variants need UAT |

**Overall: HIGH.** Architecture, stack, and risk surface well-understood. MEDIUM-LOW confidence points concentrated in vendor-specific syntax, only resolvable on real devices.

---

## Sources

**Primary (HIGH confidence):**
- `PROJECT.md` — v1.19 locked decisions (vendor scope, hardcoded templates, serial fail-fast, post-write enqueue)
- `CLAUDE.md` — Handler-Service pattern, operlog 25 OperType constants, status convention, response format
- `internal/device/scrapli_wrapper.go:567` — `SendConfig` verified
- `internal/services/config_execution_service.go:277` — existing SendConfig caller pattern
- `internal/services/device_info_collection_service.go:133` — `Enqueue(deviceID)` interface
- `internal/services/device_credential_helper.go:24-40` — device→credential resolver
- `internal/utils/operlog/operlog.go` — `Record` / `RecordWithBody` / `RecordBackground`
- `pkg/permission/config.go:147-198` — existing `NetworkPortQuery` / `NetworkCommandExecute`
- `internal/api/v1/network/network_router.go:206-214` — existing port group + perm matrix
- `internal/api/v1/network/port_handler.go:96-111` — read-side handler pattern to mirror
- `.planning/notes/migration-grant-new-menu-precision-helper.md` — `GrantNewMenuToRolesHavingParent`
- `.planning/notes/device-info-enrichment-zombie-blockage.md` — v1.18 Enqueue dedup
- `.planning/notes/shutdown-hang-after-port-close.md` — 8s shutdown timeout pattern

**Secondary (MEDIUM confidence):**
- v1.18 research notes
- Vendor command syntax knowledge (Huawei VRP / H3C Comware / Ruijie RGOS) — needs UAT validation
- Scrapligo v1.4.0 release notes (queue-panic mitigation)

**Tertiary (LOW confidence):**
- Specific firmware version differences (V200R005 vs V600R024C00)
- Ruijie S2960 dot1x syntax — assumed from Ruijie collector pattern
- Production SSH round-trip times for batch sizing

---

*Research completed: 2026-07-06*
*Ready for roadmap: yes*