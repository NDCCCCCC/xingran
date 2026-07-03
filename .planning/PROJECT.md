---
last_updated: 2026-06-27
update_trigger: start of milestone v1.17 (asset reconciliation)
previous_update: 2026-06-26 after v1.16 milestone close (Phase 40/41 + 13 gap closure shipped)
---

## Current Milestone: v1.17 资产对账 (Asset Reconciliation)

**Goal:** 建立"实物层 vs 声明层"对账引擎，使资产系统数据保持与实际情况相符。

**Target features / 工作池:**
- 多源数据冲突检测（物理链路 vs 系统责任人 vs AD managed_by）
- Observe-only 策略 + 告警驱动人工修复（不动业务表）
- IP 段级别例外规则（5 种 actions：no_alert/no_notice/no_workorder/skip_severity/silence）
- 工位详情页整合对账健康度（顶部卡片 + 行内徽标 + 抽屉）
- 自动转工单闭环（critical/high 异常）
- 半自动修复建议（高置信度，R5 可选）

**5 phases (42-46):**
- Phase 42 / R1: 观测底座（物化视图 + reconciliation 表 + dashboard + Statistics 端点）
- Phase 43 / R2: 告警 + 工单闭环
- Phase 44 / R3: 置信度评分 + IP 段例外规则 + 例外管理页 + 命中测试
- Phase 45 / R4: 工位详情整合 + 资产详情摘要块
- Phase 46 / R5: 半自动修复（可选）

**关键决策 (locked):**
- 策略：Observe-only（不修改业务表，仅记录 + 告警 + 人工修复）
- 菜单归属：资产管理 / 数据质量（资产对账 + 3 个二级菜单）
- 权限命名：asset:reconciliation:*
- API 路由：/asset/reconciliation/*
- 跨模块调用：ops/workstation → asset/reconciliation（service 层直接 + 权限降级）
- Owner 合并：运维 + 资产 + 权限 同一人（无需双签，单人签字）

**复用审计结论 (v0.4):**
- ✅ 已复用 13 项：字典/参数/operlog/Handler-Service/CacheProvider/Excel/UUID/Status/前端 hooks/opsApi/ECharts/UI库
- ⚠️ 部分复用 4 项：字典枚举值/参数 seed/Excel config/operlog module 常量
- ❌ 必须新增 7 项：cache_key helper/Statistics 端点/queryKeys/operlog module/HealthScore 函数/路由注册/Cron 注册

**R1 启动门槛 (✅ 已满足):**
- 端口采集覆盖率达标（运维 owner 确认）
- T1-T7 基础设施调查完成
- v0.3 架构 + v0.4 复用审计 + 跨模块权限声明已落盘

**决策记录 (.planning/notes/):**
- asset-reconciliation-strategy.md (v0.3 + v0.4)
- 260627-reconciliation-reuse-audit.md
- 260627-cross-module-permission.md
- 260627-port-coverage-audit.md

**Seed:** .planning/seeds/asset-reconciliation-v1.17.md

**Todo:** .planning/todos/pending/v1.17-reconciliation-decisions.md (T1-T30)

# RuoYi-Next 运维管理系统

## What This Is

RuoYi-Nect is a Go backend + React frontend enterprise IT operations
management system. It has evolved from a workstation-import
association feature (v1.0) into a multi-domain platform covering:

- **Operations domain**: buildings, floors, workstations, server rooms,
  info-points, dedicated lines, assets (CRUD + Excel import/export
  with geocoding).
- **Network domain**: device discovery, SNMP, Scrapli-based command
  execution, credential vault, topology, MAC history, port collection.
- **Identity domain**: AD/LDAP integration (login, OU-dept mapping,
  group auto-sync), user/role/dept/menu management, RBAC.
- **VDI domain** (v1.11): Sangfor VDI virtual desktop management with
  VM CRUD, power operations, user binding, server config.
- **Audit domain** (v1.15 / Phase 34): full-coverage operation logging
  for all 309 write endpoints with sensitive-keyword masking.

National cryptography (SM2/SM3/SM4) is mandatory for in-transit and
at-rest data on sensitive endpoints.

## Core Value

End-to-end operational observability and auditability: every write
action in the system produces a traceable record (who, when, what,
from-where, before/after-state) with sensitive fields auto-masked.
Combined with Excel-driven bulk management (import/export with
auto-resolution of references like dept-name → UUID), this gives
operators a single trustworthy system-of-record for the entire IT
estate.

## Requirements

### Validated

- ✓ 工位 CRUD（创建、读取、更新、删除）— 现有
- ✓ Excel 批量导入/导出工位 — 现有
- ✓ 工位关联 `org_id`（部门 UUID）— 现有字段，导入时通过部门名称匹配
- ✓ 前端工位列表/详情页面 — 现有
- ✓ 部门表 `sys_dept` 含 `dept_name` 字段 — 现有
- ✓ 用户表 `sys_user` 含 `user_name` 字段 — 现有
- ✓ Excel 导入模板添加"所属部门"和"所属用户"两个可选列 — v1.0
- ✓ 导入时通过部门名称/用户名匹配，写入对应 ID — v1.0
- ✓ 匹配失败留空不阻断导入 — v1.0
- ✓ 信息点导入配置添加"所属设备"列 (`deviceName` → `device_id`) — v1.1
- ✓ 信息点导入配置添加"所属端口"列 (`portName` → `port_id`) — v1.1
- ✓ 两个新列均为可选，空值不阻断导入 — v1.1
- ✓ 模板下载包含设备和端口示例值 — v1.1
- ✓ Widget 数据获取机制 (GetWidgetData/GetBatchWidgetData) — v1.2
- ✓ 前端交互完善 (Widget 选择器、仪表盘设置、模板预览) — v1.2
- ✓ 实时数据刷新 (WebSocket 连接、轮询、缓存) — v1.2
- ✓ 用户体验优化 (拖拽布局、加载状态、响应式) — v1.2
- ✓ SNMP panic 修复 (panic 恢复 + 并发安全 + 连接验证) — v1.3
- ✓ 后端代码优化 (Core 清理 + 安全修复) — v1.3
- ✓ 批量导出功能 (多实体 ZIP 打包下载) — v1.3
- ✓ **v1.16 技术债清理** (Tech-Debt Cleanup) — Phases 40/41 — 2026-06-26 (22+29 个 deferred debug session + 6 audit 文件规范化)
- ✓ **v1.17 资产对账** (Asset Reconciliation) — Phases 42-46 — 2026-07-03 (observe-only 对账引擎 + 工单闭环 + IP 段例外 + 工位详情整合 + 半自动修复建议 + UAT 9/10 PASS)
  - R1 观测底座：物化视图 + Statistics 端点 + operlog 全覆盖
  - R2 告警+工单：critical/high 自动转工单 + WebSocket + SysNotice + 7d 静默期
  - R3 IP 段例外：CIDR + 5 actions + 命中测试 + Excel 导入导出
  - R4 工位详情：健康度卡片 + ReconciliationDrawer 3 Tab
  - R5 半自动修复：6 状态机 + 7d 回滚 + 误修复率告警三通道
- ✓ **Phase 47 根因修复** (infoPoint drift / Layer3 UPSERT / port_status 漂移 / MAC parser 校验) — 2026-07-03

### Active

## Current Milestone: v1.15 工位设备关联 (Phase 28) — ✅ SHIPPED 2026-06-10

Most recent shipped milestone. Since v1.15, additional work has
shipped in **unlabeled phases** (30, 31, 32, 33, 34) covering
frontend perf, P0 closure, P1/P2 review fixes, Vercel React
best-practices, and operlog full coverage. See "Active" section
for the v1.16 planning state.

## Active: v1.18 (planning) — Phase 48 网络设备组件序列号采集

The v1.18 scope targets the **Device Component Serials** feature:
one-network-device-many-component-serials — chassis/card/engine/power/fan/transceiver
each as a separate `ops_asset` record (`DeviceSN` = component SN),
preserving the parent-child topology between component and switch/router.

**Candidate work**:
- **R1**: Extend `Asset` (`ops_asset`) model + migration: `parent_asset_id` (self-FK),
  `source_device_id` (FK to `sys_network_device.id`), `component_type` (chassis/card/engine/power/fan/transceiver),
  `component_slot`
- **R2**: Dual-path collection (based on 2026-07-03 real-device probes)
  - **Main (SNMP ENTITY-MIB)**: Single GET (NOT BulkWalk; HW/Ruijie reject) pull
    `entPhysicalSerialNum` / `entPhysicalModelName` / `entPhysicalClass` / `entPhysicalContainedIn`
    - Huawei S8700: chassis / module / fan / business card + engine
    - Ruijie RS8607E: chassis / fan / power / M1-M2 main control, business card slot 1-5 placeholder (SN via CLI)
    - Required filter: Ruijie `powerSupply(8)` but Name like `temprature*` sensor pseudo-entity
      (typo "temprature", Ruijie private extension noise, full table 352 must filter)
  - **CLI supplement**:
    - Ruijie `show version`: business card (slot 1-5) SN
    - Huawei `display interface transceiver`: transceiver SN + DDM (not via SNMP)
    - Huawei `display device esn`: device SN (cross-verify with SNMP)
    - Huawei `display device` / `display device board`: hardware list skeleton
- **R3**: Only collect in-use / up transceivers
- **R4**: Match component SN against `ops_asset` by `DeviceSN`; only UPDATE not INSERT (实物/声明 分层)
- **R5**: Display component list under asset/device detail page (grouped by `parent_asset_id`)
- **R6**: Generate `sys_data_reconciliation` anomalies for unmatched SNs (新 category "组件序列号")

**Prerequisites** (mostly done):
- RQ-001 真机采样 (real-device probes) ~80% complete as of 2026-07-03
- 锐捷 ENTITY-MIB real-device probe ✓ (2026-07-03)
- 锐捷光模块 DDM TextFSM 另一设备采样 (remaining)
- H3C/迈普 另环境采样 (remaining)

**Dependency**: Requires v1.17 对账引擎 (Phase 42-46) ✓ SHIPPED 2026-07-03

See `.planning/ROADMAP.md` Phase 48 section for details.

### Completed

#### v1.3 技术债清理 — ✅ Completed 2026-04-27

**SNMP-01: SNMP Panic 修复**
- [x] SNMP-01a: 实现 panic 恢复包装器
- [x] SNMP-01b: 实现 RWMutex 并发安全机制
- [x] SNMP-01c: 实现连接就绪性验证

**CODE-02: 后端代码优化**
- [x] CODE-02a: 系统性扫描死代码（无死代码可删除）
- [x] CODE-02b: 清理 Core 结构（删除 2 字段，文档化 12 字段）
- [x] CODE-02c: 修复安全问题（WebSocket、并发安全、错误日志）

**EXPORT-03: 网络设备导出**
- [x] EXPORT-03a: 注册导出路由 ✅
- [x] EXPORT-03b: 前端集成 ✅
- [x] EXPORT-03c: 厂商特定格式 ✅
- [x] D-04: 实现批量导出功能 ✅

#### v1.2 可配置仪表盘生产级改造（仿 Zabbix）— ✅ Completed 2026-04-21

**WDG-01: Widget 数据获取机制**
- [x] WDG-01a: 实现 `GetWidgetData` 方法
- [x] WDG-01b: 实现 `GetBatchWidgetData` 批量获取
- [x] WDG-01c: 支持数据转换配置（transform）
- [x] WDG-01d: 支持三种数据源类型（API、WebSocket、Static）

**FE-02: 前端未完成功能**
- [x] FE-02a: 实现 Widget 选择器组件
- [x] FE-02b: 实现仪表盘设置面板
- [x] FE-02c: 实现模板预览功能
- [x] FE-02d: 完善 Widget 编辑/删除交互

**RT-03: 实时数据刷新**
- [x] RT-03a: 实现 WebSocket 连接管理
- [x] RT-03b: 实现基于刷新间隔的轮询机制
- [x] RT-03c: 实现缓存策略
- [x] RT-03d: 前端实时数据订阅和更新

**UX-04: 用户体验优化**
- [x] UX-04a: Widget 拖拽布局优化
- [x] UX-04b: 仪表盘加载状态和错误处理
- [x] UX-04c: 数据为空时的友好提示
- [x] UX-04d: 响应式布局适配

### Out of Scope

- 导入时自动创建不存在的部门或用户 — 超出导入功能职责范围
- 批量修改已有工位的部门/用户关联 — 本期仅关注导入场景
- 工位与用户的多对多关系 — 当前只需一对一关联
- 设备/端口多对多关系 — 当前只需一对一
- 导入时自动创建不存在的设备或端口 — 超出导入功能职责
- 端口与设备的级联验证（确认端口属于指定设备）— 复杂度高，用户未要求
- **仪表盘系统统一** — 保留两套系统：固定监控仪表盘（/monitor/dashboard）和可配置仪表盘（/dashboard-system）
- 复杂图表类型（3D 图表、地图）— 超出本期范围
- 告警规则配置 — 属于监控系统，非仪表盘核心
- Zabbix 告警集成 — 超出本期范围

## Context

### 技术环境
- **后端**: Go 1.24 + Gin + GORM + PostgreSQL
- **前端**: React 19 + TypeScript + Ant Design 6 + Zustand
- **Excel**: excelize v2 用于导入导出
- **现有导入流程**: `excel_service.go` → `excel_config.go`（列映射）→ `reference_resolver.go`（名称→UUID）→ `batch_upserter.go`（批量写入）
- **仪表盘**: 自定义 Widget 系统，支持 StatCard、Chart、Table、List、Progress 类型
- **两套仪表盘**:
  - `/monitor/dashboard` — 固定监控仪表盘（保留不变）
  - `/dashboard-system` — 可配置仪表盘（v1.2 改进目标）
- **批量导出**: ZIP 打包多个 Excel 文件，支持 9 个实体类型

### Current state (2026-06-16)

- **Shipped**: v1.0 through v1.15 (and Phase 22 VDI 22A/22B partial for v1.11) —
  18 phases, 133 plans completed, ≈85% of total plan count.
- **Phases 30 / 31 / 32 / 33 / 34**: unlabeled (no v1.x tag in ROADMAP
  milestones list) but fully shipped. Phase 34 included a full code
  review with 1 critical security finding (sensitiveKeys did not
  match camelCase field names) and 7 atomic fix commits.
- **Codebase size**: 485 Go files (backend), 506 TS/TSX files
  (frontend) per `CLAUDE.md`.
- **Test coverage**: regression_test.go for operlog locks 25 OperType
  constants, 18 mandatory sensitive keywords, Record signature, keyword
  stability. 14 subtests in TestFilterSensitiveParams + 5 edge cases.
- **Build status** (post-Phase 34 review closure): `go build ./...`
  clean, full operlog test suite PASS, system test suite PASS.

### Known technical debt (post-v1.15)

- VDI 22C (账号管理) and 22D (Web Console) — listed in ROADMAP as
  v1.12 sub-phases but not started. Blocked on 22A/22B production
  observability data.
- Phase 25 (VM 数据范围权限) — depends on Phase 22, also not started.
- `operlog_e2e_verify.sh` live-DB section gated on `SKIP_E2E=1` by
  default — should be promoted to a CI gate.
- `regression_test.go:146` — `for i := 0; i < 5; i++` can be
  modernized to `for i := range 5` (Go 1.22+). Cosmetic, single line.
- 103 open items in `gsd-sdk query audit-open` (most are stale
  `root_cause_found` debug sessions from May 2026).

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 用户关联字段: 在 workstation 表添加 `user_id` 列 | 与现有 `org_id` 模式一致，一对一关联 | ✓ Good |
| 用户匹配方式: 通过 `user_name` 精确匹配 | 用户名唯一且稳定 | ✓ Good |
| 部门匹配方式: 通过 `dept_name` 精确匹配 | 与现有 `reference_resolver` 模式一致 | ✓ Good |
| 匹配失败策略: 留空不阻断 | 用户明确要求可选关联 | ✓ Good |
| Device Reference: `sys_network_device.device_name` | 设备名是用户可见的标识字段 | ✓ Good |
| Port Reference: `sys_device_port_status.interface_name` | 接口名是网络设备端口的唯一标识 | ✓ Good |
| Port 无 DependsOn: 全局查找不限设备 | 用户未要求级联验证 | ✓ Good |
| 配置驱动模式: 只改 excel_config.go + excel_service.go | 最小改动，复用现有管道 | ✓ Good |
| 保留两套仪表盘: 固定 + 可配置 | 用户明确要求保留固定监控仪表盘 | ✓ Good |
| SNMP panic 修复: 三层防护（恢复+并发+验证） | 防御式编程，确保稳定性 | ✓ Good |
| Core 清理: 文档化而非删除 | 大多数字段有外部依赖 | ✓ Good |
| TDD 方法: RED→GREEN→REFACTOR | 确保测试覆盖和安全修复质量 | ✓ Good |
| 批量导出: ZIP 格式打包多个 Excel | 用户需求，简化下载流程 | ✓ Good |
| VDI 集成: 拆分 22A→22D 4 子阶段 (P1) | 渐进式集成降低风险 | ✓ Good (22A/22B done; 22C/22D pending) |
| operlog 共享包: 提取到 `internal/utils/operlog` (P34) | 避免 handler module 间的循环依赖 | ✓ Good |
| OperType 25 常量集 (P34) | 锁定审计语义，不再用 OperTypeOther 兜底 | ✓ Good |
| OperTypeUnlock = 24 新增 (P34 review) | 解锁是独立审计动作 | ✓ Good |
| sensitiveKeys 34 条 camelCase/snake_case 全覆盖 (P34 review) | 修复 substring-search 不匹配 camelCase 字段的 critical bug | ✓ Good (CR-001) |
| 工位设备关联子表格: Ant Design Table expandable (P28) | 复用现有 Table 组件 | ✓ Good |

## Constraints

- **Tech Stack**: 必须沿用现有架构模式（Handler-Service、opsApi、excel_config）
- **Backward Compatible**: 新字段可选，不影响现有数据和导入流程
- **UUID Foreign Keys**: 关联字段必须是有效 UUID
- **Status Convention**: 遵循 0=正常, 1=停用 的惯例
- **Response Format**: 使用 `response.Success()` / `response.Error()` 包装
- **Widget 数据获取**: 必须支持权限过滤，用户只能访问有权限的 API 端点
- **保留固定仪表盘**: `/monitor/dashboard` 保持不变，仅改进 `/dashboard-system`
- **SNMP 稳定性**: 必须保证高并发场景下不崩溃
- **测试覆盖**: 安全修复必须有单元测试验证

## Evolution

This document evolves at phase transitions and milestone boundaries.

## Validated — v1.0 through v1.15 (summary)

The full Validated section listing all shipped requirements is preserved
in `.planning/REQUIREMENTS.md` history. High-level milestones:

- ✓ v1.0 工位导入部门/用户关联 (2026-04-16) — 2 phases, 7 plans
- ✓ v1.1 信息点导入设备端口关联 (2026-04-16) — 1 phase, 1 plan
- ✓ v1.2 可配置仪表盘生产级改造 (2026-04-21) — 4 phases, 11 plans
- ✓ v1.3 技术债清理 (2026-04-27) — 3 phases, 9 plans
- ✓ v1.4 MAC地址采集优化 (2026-05-09) — 1 phase, 4 plans
- ✓ v1.5 MAC地址历史数据管理 (2026-06-15) — 4 phases, 26 plans (REQ archive: `.planning/milestones/v1.5-REQUIREMENTS.md`)
- ✓ v1.6 API密钥管理系统 (2026-05-19) — 1 phase, 10 plans
- ✓ v1.7 前后端加密配置同步 (2026-05-20) — 1 phase, 6 plans
- ✓ v1.8 登录端点加密增强 (2026-05-21) — 1 phase, 4 plans
- ✓ v1.9 AD域控集成扩展 (2026-05-24) — 2 phases, 11 plans
- ✓ v1.10 网络设备权限修复 (2026-05-24) — 1 phase, 1 plan
- ✓ v1.11 深信服桌面云集成 (VDI 22A+22B partial) (2026-06-02) — 1 phase (22), 6 plans
- ✓ v1.13 资产管理模块 (Phase 26) (2026-06-08) — 1 phase, 6 plans
- ✓ v1.14 全局列自定义 (Phase 27) (2026-06-09) — 1 phase, 1 plan
- ✓ v1.15 工位设备关联 (Phase 28) (2026-06-10) — 1 phase, 4 plans
- ✓ Phase 30 前端性能优化 (unlabeled, 2026-06-13) — 5 plans
- ✓ Phase 31 P0 收尾 (2026-06-13) — 5 plans
- ✓ Phase 32 v1.14 P1/P2 (unlabeled, 2026-06-13) — 7 waves, 20 commits
- ✓ Phase 33 Vercel React Best Practices (unlabeled, 2026-06-13) — 4 waves
- ✓ Phase 34 操作日志全模块集成 + review (2026-06-16) — 11 plans + 7 review-fix commits
- ✓ Phase 39 工位部门物理位置映射 (workstation dept-location alias) (2026-06-25) — 8 plans + UAT 修复 commits
- ✓ Phase 44 R3 IP 例外规则引擎 (v1.17 资产对账) (2026-06-28) — 2 plans, CIDR GiST 引擎 + Layer 3.5 拦截 + 例外 CRUD/命中测试 + cron 过期清理 + 转单 no_workorder 过滤 + 降噪基线 + Excel 导入导出; VERIFICATION 10/10 passed, 1 CR-03 gap-closure fix (baseline 路由权限), 4 硬化债 BLOCKER 跟踪

---
*Last updated: 2026-07-03 after v1.17 milestone close + Phase 47 root cause fixes shipped + v1.18 Phase 48 planning*
