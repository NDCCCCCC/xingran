---
last_updated: 2026-07-06
update_trigger: start of milestone v1.19 (network device write operations)
previous_update: 2026-07-04 after v1.18 milestone close (Phase 48 shipped: 1 phase / 3 plans / 3 waves)
---

# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15; Phase 13 gap closure retroactive 2026-06-26)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联** — Phase 28 (shipped 2026-06-10)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.15 部门物理位置映射** — Phase 39 (shipped 2026-06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 (shipped 2026-07-03) — see [milestones/v1.17-ROADMAP.md](milestones/v1.17-ROADMAP.md)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 (shipped 2026-07-04) — see [milestones/v1.18-ROADMAP.md](milestones/v1.18-ROADMAP.md)
- 🚧 **v1.19 网络设备写命令 (Network Device Write Operations)** — Phases 50-54 (planning — see below)

---

## Phases (v1.19)

- [ ] **Phase 50** — W1: Vendor Templates + Unit Tests
- [ ] **Phase 51** — W2: PortWriteService + Batch Orchestrator + Mock Tests
- [ ] **Phase 52** — W3: Router/Handler/Operlog/Permission/Migration
- [ ] **Phase 53** — W4: Frontend Drawer + Progress Dialog + API Wrappers
- [ ] **Phase 54** — W5: E2E + Real-Device UAT + Documentation

### Phase 50: W1 — Vendor Templates + Unit Tests (Vendor → Action → Command Map)

**Goal**: Lock the vendor→action→command contract before any HTTP/DB wiring. Lock one entry per `(vendor, action)` pair across Huawei VRP / H3C Comware / Ruijie RGOS, with table-driven unit tests that catch vendor syntax drift early.

**Depends on**: Nothing (pure Go map + tests; zero external dependencies)

**Requirements**: SSH-05 (vendor→action→command map), SSH-01 (SendConfig call signature contract)

**Success Criteria** (what must be TRUE):

  1. 3 vendors × 5 actions = 15 templates implemented in `internal/services/portcollection/vendor_port_template.go` (Huawei VRP, H3C Comware, Ruijie RGOS × shutdown / undo shutdown / description / dot1x enable / dot1x disable)
  2. 12+ unit tests in `vendor_port_template_test.go` covering each (vendor, action) combination, including negative cases (missing vendor, unsupported action)
  3. `PortAction` and `PortTemplateParams` types defined as exported named types; `RenderCommand(vendor, action, params) ([]string, error)` public function signature locked
  4. Huawei and H3C share command shape (VRP heritage); Ruijie uses Cisco-style with `dot1x port-control auto` / `no dot1x port-control`
  5. `go test ./internal/services/portcollection/...` exits 0 with all vendor template tests passing

**Plans**: 1 plan (50-01-PLAN.md) — vendor template skeleton + table-driven tests + RenderCommand contract

**UI hint**: no

### Phase 51: W2 — PortWriteService + Batch Orchestrator + Mock Tests

**Goal**: Implement the service layer with SSH connection lifecycle management and serial fail-fast batch semantics, fully unit-testable via mocks (no real SSH).

**Depends on**: Phase 50 (vendor templates contract locked)

**Requirements**: SSH-02 (parse transport_error vs device_rejected), SSH-03 (per-port Acquire/ReleaseRef), SSH-04 (detached context, 30min timeout), SSH-06 (DeviceCredentialHelper), PORT-06 (pre-state check admin_status / dot1x_enabled), BATCH-02 (serial fail-fast), BATCH-03 (partial result struct), BATCH-04 (maxBatchSize=50 cap), AUDIT-04 (collectionSvc.Enqueue after success)

**Success Criteria** (what must be TRUE):

  1. `PortWriteService` interface + private `portWriteServiceImpl` + DI (`db`, `deviceExecutor`, `collectionSvc`)
  2. Per-port `Acquire`/`ReleaseRef` cycle (NOT per-batch) wrapped via `defer conn.ReleaseRef()` to avoid pool starvation
  3. Detached context via `context.WithTimeout(context.Background(), 30*time.Minute)` for batch operations, avoiding `Core.Close()` 30s shutdown deadline
  4. `parseConfigError(result)` distinguishes `transport_error` from `device_rejected` via `% Error:` / `Unrecognized command` / `Illegal` markers
  5. `maxBatchSize = 50` hard cap enforced in service layer; >50 returns validation error before any SSH traffic
  6. Batch result struct: `{succeeded: []PortResult, failed: []PortResult, skipped: []PortResult}` with per-port status + failure reason
  7. Service tests use mocks (`mockDeviceExecutor`, `mockCollectionSvc`) — no real SSH calls
  8. `AUDIT-04`: `collectionSvc.Enqueue(deviceID)` invoked within 1-2s after successful port write (fire-and-forget)

**Plans**: 1-2 plans (51-01-PLAN.md service skeleton + batch orchestrator; 51-02-PLAN.md mock tests if split)

**UI hint**: no

### Phase 52: W3 — Router/Handler/Operlog/Permission/Migration

**Goal**: HTTP wiring with audit logging, permission gating, and menu seed. Wire all 6 endpoints under `/network/ports/write` group; create `sys_port_write_audit` table; seed "端口配置" submenu under "端口管理" parent and grant to existing parent-associated roles.

**Depends on**: Phase 51 (service contract stable)

**Requirements**: AUDIT-01 (operlog.Record before response.Success), AUDIT-02 (operlog records device_id/port_id/action/description/operator/result_status), AUDIT-03 (sys_port_write_audit table), PERM-01 (NetworkPortWrite constant), PERM-02 (RequirePermissions middleware on all 6 endpoints), PERM-03 (menu seed + GrantNewMenuToRolesHavingParent), INFRA-01 (audit table migration), INFRA-02 (route registration /network/ports/write), INFRA-03 (cache_keys for port write results), CONV-01 (shutdown/undo → OperTypeStatus), CONV-02 (description → OperTypeUpdate), CONV-03 (dot1x → OperTypeStatus), CONV-04 (batch → OperTypeBatch), PORT-01 (shutdown endpoint), PORT-02 (undo shutdown endpoint), PORT-03 (description endpoint), PORT-04 (dot1x enable endpoint), PORT-05 (dot1x disable endpoint), BATCH-01 (batch endpoint)

**Success Criteria** (what must be TRUE):

  1. 6 endpoints registered (5 single-port + 1 batch) under `/network/ports/write` group in `port_write_router.go` and wired in `network_router.go`
  2. `RequirePermissions(["network:port:write"])` middleware applied to all 6 endpoints
  3. `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口管理", ...)` called before `response.Success(...)` on each endpoint with OperType mapping per CONV-01/02/03/04
  4. `NetworkPortWrite = "network:port:write"` constant added to `pkg/permission/config.go` (distinct from `NetworkPortQuery`)
  5. `sys_port_write_audit` table created via `migration_2XX_port_write_audit.go` with `(device_id, port_id, created_at)` index, all 11 columns (device_id/port_id/action/before_value/after_value/command_sent/device_response/status/failure_reason/operator/created_at)
  6. `sys_menu` seed in migration adds "端口配置" child menu under existing "端口管理" parent, calls `GrantNewMenuToRolesHavingParent(db, "端口管理", newMenuID)` to precisely grant only parent-associated roles (antd父子联动 visual trap 防御)
  7. `pkg/permission.NetworkPortWrite` referenced from `port_write_handler.go` and not hardcoded string
  8. `go build ./...` exits 0; `go test ./...` exits 0 (operlog regression lock intact, no Phase 51 mock test regression)

**Plans**: 2 plans

Plans:
**Wave 1**

- [x] 52-01-PLAN.md — Wave 1: NetworkPortWrite 常量 + PortWriteAudit model(Path X) + cache_keys 占位 + 6 handler + /network/ports/write/* 路由组 + 组级 2-arg RequirePermissions + network_router.go 注册 (PORT-01..05, BATCH-01, PERM-01/02, AUDIT-01/02, CONV-01..04, INFRA-02/03) — Shipped 2026-07-07 (3 commits: 44980bea / ec7a34ad / 88af87f7)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 52-02-PLAN.md — Wave 2 (depends on 52-01): PortWriteAudit 加入 database.go AutoMigrate + postgres 分支显式调 Migrate202PortWriteAudit + menu_grant_helpers.go 幂等授权 helper + migration_202_port_write_audit.go(复合索引 + 菜单 seed + helper 调用) (AUDIT-03, INFRA-01, PERM-03) — Shipped 2026-07-07 (4 commits: e6911127 / 6b833b80 / 8dd54d16 / d75e58af)

**UI hint**: no

### Phase 53: W4 — Frontend Drawer + Progress Dialog + API Wrappers

**Goal**: User-facing UI for single-port operations + batch operations. 5 single-port modals with required reason input; BulkWriteDrawer with progress bar + failure list; permission-gated API wrappers.

**Depends on**: Phase 52 (HTTP API contract stable)

**Requirements**: UI-01 (port list/detail entry points with action menu), UI-02 (single-port confirm modal 5-200 char reason), UI-03 (BulkWriteDrawer with progress bar + failure list), UI-04 (antd Toast success/failure + audit log link), UI-05 (disable refresh during batch), UI-06 (6 wrapped POST functions in networkApi.ts)

**Success Criteria** (what must be TRUE):

  1. 5 single-port operation modals (shutdown / undo shutdown / description / dot1x enable / dot1x disable) all require "操作原因" input with 5-200 char validation
  2. `BulkWriteDrawer.tsx` provides selected ports list + operation type selector + progress bar (X/Y ports) + failure detail panel showing per-port error reason
  3. `src/lib/api/networkApi.ts` exports 6 wrapped POST functions: `writeShutdown` / `writeUndoShutdown` / `writeDescription` / `writeDot1xEnable` / `writeDot1xDisable` / `batchWritePorts`
  4. Permission gating via `useAuthStore` — UI entries hidden if user lacks `network:port:write` permission
  5. antd Toast feedback on completion (success or failure) with "查看审计日志" link to `sys_oper_log` filtered by module=`端口管理`
  6. Port list "刷新" button disabled (loading state) during batch operation to avoid race with `Enqueue` triggered collection
  7. `npm run type-check` exits 0; `npm run build` exits 0 without vendor bundle regression > 50 kB gzip (vendor-react baseline 776 kB gzip from Phase 48 reference)

**Plans**: 1-2 plans (53-01-PLAN.md API wrappers + modals; 53-02-PLAN.md BulkWriteDrawer if split)

**UI hint**: yes (frontend modals, BulkWriteDrawer with progress, port list action menu)

### Phase 54: W5 — E2E + Real-Device UAT + Documentation

**Goal**: Validate v1.19 end-to-end against a mock SSH server (always runnable), defer real-device UAT to the next site visit (per v1.18 Phase 48 precedent), and update project documentation.

**Depends on**: Phase 50, 51, 52, 53 (all shipped)

**Requirements**: All 28+ v1.19 MVP requirements (validation only — no new functionality; deferred real-device UAT recorded in `50-HUMAN-UAT.md` per v1.18 48-HUMAN-UAT.md pattern)

**Success Criteria** (what must be TRUE):

  1. Mock SSH server e2e test (`port_write_e2e_test.go` using a `scrapligo` Connect target against an in-process mock sshd) exercises all 5 single-port + 1 batch endpoint paths, exits 0
  2. `docs/API响应规范.md` updated with the 6 new endpoint signatures, request schemas, and response shapes
  3. `docs/安全和认证设计（国密）.md` (or equivalent) confirms write endpoints respect encryption exemption list (no SM2+SM4 wrap on SSH-derived paths)
  4. `.planning/phases/50-port-write-network-ports-planned/50-HUMAN-UAT.md` created with sections marked `pending` for the 6 SSH transport verifications (Huawei S5700/S5735 shutdown + description + dot1x, H3C equivalent, Ruijie RS8607E equivalent), explicitly deferred to next site visit with owner = 现场运维同事
  5. README + CHANGELOG updated with v1.19 entry; milestone entry in `.planning/MILESTONES.md`
  6. `go test ./...` + `npm run build` + `npm run type-check` all exit 0 from clean checkout
  7. No regression to operlog regression_test.go (25 OperType constants, 11 sensitive keywords, Record signature)

**Plans**: 1 plan (54-01-PLAN.md — mock SSH e2e + docs update + UAT deferral)

**UI hint**: no

### Progress Table

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 50. W1 Vendor Templates + Unit Tests | 1/1 | Complete    | 2026-07-06 |
| 51. W2 PortWriteService + Batch + Mock Tests | 1/1 | Complete   | 2026-07-06 |
| 52. W3 Router/Handler/Operlog/Permission/Migration | 2/2 | Complete   | 2026-07-07 |
| 53. W4 Frontend Drawer + Progress Dialog + API Wrappers | 0/1-2 | Not started | — |
| 54. W5 E2E + Real-Device UAT + Documentation | 0/1 | Not started | — |

---

## Phases (v1.18)

<details>
<summary>✅ v1.18 网络设备硬件清单 (Phase 48) — SHIPPED 2026-07-04</summary>

### Phase 48: 网络设备组件序列号采集 (Device Component Serials) — 3/3 plans, shipped 2026-07-04

**Goal**: 支持"一机多序列号"——将网络设备的板卡/引擎卡、电源、风扇、光模块各自的序列号**作为资产设备纳入资产系统 `ops_asset`**（每个组件一条记录，`DeviceSN`=组件序列号），保存组件对交换机/路由的从属关系并在前端展示。

**Depends on**: **v1.17 对账底座（Phase 42-46）** — Phase 48 是 R2 物理层对账的组件序列号形态，需对账引擎先就位

**Requirements**: 14 D-ids (D-01~D-14) 覆盖,源自 48-CONTEXT.md,无正式 REQ-ID

**Success Criteria**: 16 项全部通过自动化 UAT(schema introspection / go build / 21 collector tests / 5 D-10 流水线测试 / 3 handler tests / 5 operlog regression / tsc / vite build)。3 项 site-visit 真机 UAT(S8700/RS8607E)按 RESEARCH §Environment Availability 显式推迟到下次现场访问。

**Plans**: 3 plans

Plans:
**Wave 1**

- [x] 48-01-PLAN.md — Schema 迁移(migration_201) + Asset/Reconciliation model 扩展 + asset_service List/Statistics 默认 component_type IS NULL 过滤 (D-01, D-05, D-06, D-07)

**Wave 2** (depends on 48-01)

- [x] 48-02-PLAN.md — component_collector 包(SNMP ENTITY-MIB 单 GET 收集器 + 锐捷 temprature* 过滤 + 华为 dual-class 去重 + OwnerResolver stack 接口 + 4 TextFSM 模板 + CLI 收集器) (D-08, D-09, D-10, D-11, D-12)

**Wave 3** (depends on 48-01 + 48-02)

- [x] 48-03-PLAN.md — OpsAssetWriter(UPDATE-only + D-04 父缺失降级) + ReconciliationEmitter(conflict_type=F + recon_category=component_serial) + Pipeline + DeviceInfoCollectionService cron hook + operlog + 后端 list 端点 + 前端组件清单 Tab + UAT 推迟声明 (D-02, D-03, D-04, D-06, D-07, D-13, D-14)

**UI hint**: yes (前端组件清单 Tab,asset 详情 expandable)

See [milestones/v1.18-ROADMAP.md](milestones/v1.18-ROADMAP.md) for archived details. 3 项 site-visit UAT 待现场闭环,详见 `.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md`。

</details>

## Phases (v1.17)

<details>
<summary>✅ v1.17 资产对账 (Phases 42-46) — SHIPPED 2026-07-03</summary>

- [x] Phase 42: 资产对账观测底座 (R1) — 5/6 plans (42-03 skeleton superseded by R3)
- [x] Phase 43: 告警 + 工单闭环 (R2) — 3/3 plans
- [x] Phase 44: 置信度评分 + IP 段例外 (R3) — 2/2 plans
- [x] Phase 45: 工位详情整合 (R4) — 2/2 plans
- [x] Phase 46: 半自动修复 (R5, 可选) — 2/2 plans (UAT 9/10 PASS)

See [milestones/v1.17-ROADMAP.md](milestones/v1.17-ROADMAP.md) for details.

</details>

### Progress Table

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 42. 资产对账观测底座 (R1) | 5/6 | Complete | 2026-06-27 |
| 43. 告警 + 工单闭环 (R2) | 3/3 | Complete    | 2026-06-27 |
| 44. 置信度评分 + IP 段例外 (R3) | 2/2 | Complete    | 2026-06-28 |
| 45. 工位详情整合 (R4) | 2/2 | Complete | 2026-06-28 |
| 46. 半自动修复 (R5, 可选) | 2/2 | Complete | 2026-07-03 |
| 47. 修復資產對賬系統根因 | 2/2 | Complete | 2026-07-03 |

---

## Archive: Pre-v1.17 Phase History (Phases 1-46 preserved)

<details>
<summary>Earlier phase history (v1.0–v1.16) preserved for reference</summary>

### Phase Summary (cumulative)

- ✓ Phases 1-11 — v1.0–v1.4 基础功能 (27/27 plans)
- ✓ Phases 12-15 — v1.5 MAC地址历史数据管理 (26/26 plans)
- ✓ Phase 16 — v1.6 API密钥管理 (10/10 plans)
- ✓ Phase 17 — v1.7 加密配置同步 (6/6 plans)
- ✓ Phase 18 — v1.8 登录端点加密 (4/4 plans)
- ✓ Phases 19-20 — v1.9 AD域控集成 (11/11 plans)
- ✓ Phase 21 — v1.10 网络设备权限 (1/1 plan)
- ✓ Phases 22A/22B — v1.12 深信服 VDI (6/6 plans)
- ✓ Phase 23 — v1.11 AD组自动同步 (6/7 plans, 23-13 superseded)
- ✓ Phase 24 — UUID一致性优化 (2/2 plans)
- ✓ Phase 25 — VM 数据范围权限 (5/5 plans)
- ✓ Phase 26 — v1.13 资产管理 (6/6 plans)
- ✓ Phase 27 — v1.14 全局列自定义 (1/1 plan)
- ✓ Phase 28 — v1.15 工位设备关联 (4/4 plans)
- ✓ Phase 30 — 前端性能优化 (5/5 plans, 25/25 must-haves)
- ✓ Phase 31 — P0 收尾 (5/5 plans, F-14 + F-17)
- ✓ Phase 32 — v1.14 P1 重构 + P2 架构优化 (7/7 waves, 15 P1 + 8 P2)
- ✓ Phase 33 — Vercel React Best Practices (4/4 waves, 26 defects)
- ✓ Phase 34 — 操作日志全模块集成 (11/10 plans, 100% coverage 298/298)
- ✓ Phase 37 — 前端部门选择组件统一收敛 (6/6 plans, 10 migration points)
- ✓ Phase 38 — AD 账号池统一 (4/4 plans)
- ✓ Phase 39 — v1.15 部门物理位置映射 (8/8 plans)
- ✓ Phases 40-41 — v1.16 技术债清理 (8 plans)
- ✓ Phases 42-46 — v1.17 资产对账 (14/16 plans, 42-03 superseded)
- ✓ Phase 47 — 资产对账根因修复 (2/2 plans)
- ✓ Phase 48 — v1.18 网络设备组件 (3/3 plans)
- ✓ Phase 49 — v1.18 gap closure (2/2 plans)

**Total shipped**: 142 plans across 39 phases (Phases 1-49), 8 gap-closure plans

</details>

---

*Last updated: 2026-07-06 — v1.19 网络设备写命令 roadmap drafted; Phases 50-54 planned in 5-wave build order (W1 templates → W2 service → W3 HTTP+audit+perm → W4 frontend → W5 e2e+UAT+docs). Awaiting `/gsd:plan-phase 50` to begin execution.*
