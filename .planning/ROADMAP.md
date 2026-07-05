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

---

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

### Phase 47: 修复資產對賬系統根因 (infoPoint 配置、Layer 3 引擎、port_status 漂移、parser 校驗) — 2/2 plans, shipped 2026-07-03 (independent of v1.17)

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

### Progress Table

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 42. 资产对账观测底座 (R1) | 5/6 | Complete | 2026-06-27 |
| 43. 告警 + 工单闭环 (R2) | 3/3 | Complete    | 2026-06-27 |
| 44. 置信度评分 + IP 段例外 (R3) | 2/2 | Complete    | 2026-06-28 |
| 45. 工位详情整合 (R4) | 2/2 | Complete | 2026-06-28 |
| 46. 半自动修复 (R5, 可选) | 2/2 | Complete | 2026-07-03 |
| 47. 修复資產對賬系統根因 | 2/2 | Complete | 2026-07-03 |

---

## Phases (v1.16)

### Phase 40: Tech-Debt Cleanup (技术债清理)

**Goal**: 处理 v1.15 累积的 22 个 deferred debug 会话 + 6 个 audit 数据质量问题，让系统回到"无遗留债务"状态，audit-open 中 debug_sessions < 5。

**Depends on**: 无（独立清理工作，可独立验证）

**Requirements**: TECH-01, TECH-02, TECH-03, TECH-04, TECH-05

**Success Criteria** (what must be TRUE):

  1. `.planning/debug/` 中 16 个 `root_cause_found` 会话已通过代码修复 + commit 闭环，frontmatter `status: resolved` 可被 audit 工具正确识别
  2. `.planning/debug/` 中 5 个 `awaiting_human_verify` 会话(workstation-device-build-errors、workstation-expand-device-load-empty、reset-theme-hardcoded-colors、user-filter-lost-after-edit、vite-hmr-token-blank-page)经 dev 环境复现验证后,frontmatter `status: resolved`
  3. `.planning/debug/apikey-route-path-duplication.md`(`root-cause-found` 状态)随 TECH-01 一并处理完成,frontmatter `status: resolved`
  4. `login-400-bad-request` 与 `ops-asset-constraint-uni-ops-asset-devicesn-not-exist` 两个 debug 会话补齐标准 YAML frontmatter(status / slug / updated / session_type / related 字段齐全)
  5. `knowledge-base` 文件明确移出 audit 范围(添加 `skip_audit: true` 注释或改名),不被 audit-open 误计
  6. `info-point-type-null-import` 与 `sys-mac-filter-rules-relation-does-not-exist` frontmatter 统一,audit 工具能正确识别其 resolved 状态
  7. `gsd-sdk query audit-open` 输出 `debug_sessions < 5`(从当前 57 下降)
  8. `scripts/validate_debug_frontmatter.sh`(或等价校验脚本)对 `.planning/debug/*.md` 全量扫描通过率 = 100%

**Plans**: 3/3 complete (40-01 / 40-02 / 40-03)

**UI hint**: no

### Phase 41: Debug Session Gap Closure (debug_sessions 29 → < 5)

**Goal**: 闭环 Phase 40 遗留的 `audit-open` debug_sessions 缺口 —— Phase 40 完成 22 个 deferred session + 6 audit 文件规范化,但仍有 29 个 scope 外历史 open session(investigating 10 / debug_complete 5 / fixed 3 / verifying 2 / fix_applied 2 / diagnosed 2 / 其他 5)。本 phase 把 debug_sessions 降到 < 5,达成 TECH-05 验收。

**Depends on**: Phase 40(其批量 frontmatter 修复 + validator/verify 工具链就位)

**Requirements**: TECH-06(支撑 TECH-05 的 debug_sessions < 5 部分)

**Success Criteria** (what must be TRUE):

  1. ~12 个"实质已完成态"session(debug_complete / fixed / fix_applied / applied / fixed_pending_restart)经核查修复确实落地后,frontmatter 翻 `status: resolved`
  2. 剩余 ~17 个真正在途 session(investigating / verifying / diagnosed / root_cause_identified / checkpoint_reached)逐个处理:能修则修 + 验证 → resolved;陈年/重复/不适用者标记 won't-fix 或 `skip_audit: true` 移出 audit 计数
  3. `gsd-sdk query audit-open` 输出 `debug_sessions < 5`
  4. `bash scripts/verify_phase40.sh` 两条标准全 PASS(exit 0)
  5. 每个被翻/won't-fix 的 session 在其 `.md` Resolution 章节追加 Phase 41 Closure 证据(D-07 沿用)
  6. `go build ./...` / `npm run build` 全程退出码 0(若有代码修复)

**Plans**: 4 plans

Plans:
**Wave 1**

- [x] 41-01-PLAN.md — Cohort A 12 个'实质已完成态'session 复测-后-翻(D-01) — 2026-06-26 complete, 12 fix(41) commits, validator 100%
- [x] 41-02-PLAN.md — Cohort B 前端集群 7 个 session 混合修+wontfix(D-02) — 2026-06-26 complete, 2 fix(41) + 4 docs(41) commits, 4 with skip_audit (css-tree-vendor-chunk-tdz + login-md-vendor-crash + react-vendor-activity-tdz + device-modal-serial-auto-match), 1 HELD for browser verify (hardcoded-blue-theme-bypass), validator 100%, npm run build 0
- [x] 41-03-PLAN.md — Cohort B 后端/认证集群 10 个 session 混合修+wontfix(D-02) — 2026-06-26 complete, 1 fix(41) + 9 docs(41) commits, 2 with skip_audit (ad-admin-lockout-recurrence + interface-to-any-conversion-hint), validator 100%, go build 0

**Wave 2**

- [x] 41-04-PLAN.md — 验收关门:audit-open<5 + verify_phase40.sh 双 PASS(D-03/04/05) — 2026-06-26 complete (Task 1: three-piece suite ALL PASS, 29/29 sessions closed, TECH-05+TECH-06 achieved; Task 2: STATE/ROADMAP/REQUIREMENTS updated, v1.16 milestone closed, user approved, push deferred)

**Phase 41 Plans**: 4/4 complete (2026-06-26)

**UI hint**: no

### Progress Table

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 40. Tech-Debt Cleanup | 3/3 | Complete   | 2026-06-25 |
| 41. Debug Session Gap Closure | 4/4 | Complete    | 2026-06-26 |

---

## Archive: Pre-v1.16 Phase History

<details>
<summary>Earlier phase history (v1.0–v1.15) preserved for reference</summary>

The full history of Phases 1-39 is preserved below. See also:

- `.planning/PROJECT.md` (Validated section) for v1.0–v1.15 requirement history
- `.planning/MILESTONES.md` for shipped-milestone summaries
- `.planning/phases/` for per-phase plan files

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

**Total shipped**: 137 plans across 26 phases, 8 gap-closure plans

</details>

### Phase 47 (Root Cause Fixes): 修复資產對賬系統根因 — SHIPPED 2026-07-03 (independent of v1.17)

**Goal:** 根除 Phase 45 R5 reconciliation 系统中识别出的 4 个数据完整性问题:① infoPoint.port_id 指向错设备的 port_status（drift 引发）② Layer 3 检测引擎只 INSERT 不 UPDATE 既有 conflict，导致 124 个 D 异常卡住无法重新分类 ③ sys_device_port_status.device_id 在不同设备之间漂移 ④ MAC parser 接受非 MAC 文本（如 "Flags:"、"total"、"#"，导致垃圾数据入库）。

**Requirements**:

- R1: 4F001 工位 infoPoint 4D217 指向 RS8607E-03 的 GE2/25 端口（port_id=230e8ef7）
- R2: sys_data_reconciliation 所有卡住的 open 异常能在 5 分钟内被 Layer 3 cron 重新分类
- R3: Layer 3 检测改为 UPSERT 语义（UPDATE if exists else INSERT），不再吃 SQLSTATE 23505
- R4: sys_device_port_status 行 id 与 device_id 一一对应，永不漂移
- R5: parseRuijiePortSecurityLine 加 MAC 格式校验（12位 hex），过滤 #/flags/total 等噪声

**Depends on:** Phase 46
**Plans:** 2 plans (47-01 / 47-02) — both shipped 2026-07-03

### Phase 49: v1.18 组件采集 gap closure

**Goal:** 修复 v1.18 组件采集链路 3 个连锁缺口,使 chassis 资产展开「从属组件清单」Tab 能正确显示板卡/引擎卡组件。修复顺序:Gap 3(chassis SN 采集写回 `sys_network_device.serial_number`)→ Gap 2(板卡组件 `parent_asset_id` 通过 `ops_asset.devicesn = sys_network_device.serial_number` 关联 chassis 资产)→ Gap 1(`collectComponentInfo` 接入已写好但未调用的 `ParseShowVersionModules` / `ParseDisplayDeviceEsn`)。端到端验证目标:资产 `CX-WH-WH-04F-FL-RS8607E-03`(devicesn=G1M9140000175)展开显示 6 条板卡组件。

**Requirements**: TBD (源自 49-CONTEXT.md 的 3 个连锁缺口 + 端到端验证目标,无正式 REQ-ID)
**Depends on:** Phase 48
**Plans:** 1/2 plans executed

Plans:
**Wave 1**

- [x] 49-01-PLAN.md — Gap 3 chassis SN 解析器接入,回填 sys_network_device.serial_number(ruijie/huawei 在线设备)

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 49-02-PLAN.md — Gap 1+2 collectComponentInfo 接入板卡+chassis 解析器,合并 ComponentSet 喂 Pipeline.Run + E2E 验证

---

*Last updated: 2026-07-04 — Phase 48 v1.18 SHIPPED (3/3 plans, 14/14 D-ids, 3 site-visit UAT 推迟到下次现场); v1.18 archived to `milestones/v1.18-ROADMAP.md` + `MILESTONES.md`. Awaiting next milestone (v1.19).*
