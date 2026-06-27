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
- 🚧 **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 (planning, started 2026-06-27)

---

## Phases (v1.17)

### Phase 42: 资产对账观测底座 (R1)

**Goal**: 建立 reconciliation 引擎骨架（物化视图 + 主表 + dashboard），不含告警/例外规则/前端整合。

**Depends on**: 无（独立启动；运维 owner 已确认端口采集率达标）

**Requirements**: RECON-01~07, MONITOR-01, INFRA-01~05

**Success Criteria** (what must be TRUE):

  1. `sys_data_reconciliation` + `sys_reconciliation_exception` 表创建成功，UUID + 软删除 + BaseModel 规范
  2. `reconciliation_normalized` 物化视图 5 分钟增量刷新，包含 asset → port_mac → info_point → workstation → user_id 完整链路
  3. 6 个 Statistics 专用 COUNT 端点（Summary / ByConflictType / BySeverity / HealthTrend / TopUnresolved / ExceptionRuleStats）返回正确
  4. 4 个字典 seed 完整（asset_reconciliation_*）
  5. 8 个 config seed 完整（asset.reconciliation.*）
  6. 6 个 workorder 分类 seed 完整（Type A-F）
  7. 异常列表 admin 页面可正确展示 Type A-F 分布与 Top 10 长期未解决
  8. Dashboard 顶部 5 KPI 卡片、饼图、柱状图、趋势图正确渲染
  9. `go build ./...` + `npm run build` 通过
  10. operlog 完整接入所有写操作

**Plans**: 6 plans

Plans:
- 42-01: 表 + 物化视图 migration + 字典 seed + config seed + workorder 分类 seed
- 42-02: ReconciliationService 核心（Layer 3 冲突检测 + 置信度评分）
- 42-03: ReconciliationExceptionService（IP 段例外规则 CRUD）
- 42-04: ReconciliationStatistics 6 个专用 COUNT 端点
- 42-05: 异常列表 admin 页 + Dashboard 前端（含 queryKeys + cache_key helper）
- 42-06: 路由注册 + operlog 接入 + Cron 注册 + 跨模块权限边界测试

**UI hint**: yes

### Phase 43: 告警 + 工单闭环 (R2)

**Goal**: 将 R1 的"观测"提升为"可行动"——critical/high 自动转工单，WebSocket + SysNotice 推送。

**Depends on**: Phase 42

**Requirements**: WORKORDER-01~02, MONITOR-02~03

**Success Criteria** (what must be TRUE):

  1. critical 异常检出 → 2 分钟内自动创建 workorder（通过 `workorder.BaseService.Create()`）
  2. high 异常检出 → 5 分钟内自动创建 workorder
  3. 工单模板按 Type A-F 差异化（标题、默认分派、SLA、期望动作）
  4. WebSocket 实时推送 critical 异常到 dashboard
  5. SysNotice 写入高危异常
  6. 工单状态变更触发 operlog.Record（OperTypeUpdate / Approve）
  7. 修复回写后 7d 静默期生效（避免循环触发）
  8. 24h 节流规则生效（同一 asset + conflict_type 仅 1 条新异常）
  9. 跨模块权限边界测试通过（无 `asset:reconciliation:list` 时工位页正确隐藏健康度）

**Plans**: 3 plans

Plans:
- 43-01: 异常 → 工单自动转单（critical/high）+ 6 类工单模板
- 43-02: 修复回写 + 7d 静默期 + 24h 节流
- 43-03: WebSocket 推送 + SysNotice + operlog 完整接入

**UI hint**: minimal (扩展现有 dashboard)

### Phase 44: 置信度评分 + IP 段例外 (R3)

**Goal**: 通过 IP 段级别例外规则抑制告警风暴，提供 admin 可视化管理界面。

**Depends on**: Phase 43

**Requirements**: EXCEPTION-01~04

**Success Criteria** (what must be TRUE):

  1. CIDR 格式输入校验（IPv4/IPv6）+ GiST 索引加速查询
  2. 例外规则 CRUD（创建/更新/启用/停用/删除）+ operlog 完整接入
  3. 5 种 actions 多选组合（no_alert/no_notice/no_workorder/skip_severity/silence）
  4. 多条规则取并集（合并效果可视化）
  5. 有效期（expires_at）自动停用 cron 每日清理
  6. 命中测试工具（输入 IP/CIDR 实时返回命中规则）
  7. 例外规则生效统计（按 IP 段聚合）
  8. 告警量比 R2 末期下降 ≥ 60%
  9. Excel 导入/导出例外规则（复用 ExcelConfig 模式）
  10. 命中例外的异常仍记录 reconciliation 表（含 exception_rule_id + applied_actions）

**Plans**: 2 plans

Plans:
- 44-01: 例外规则引擎 + CRUD API + admin 管理页 + 命中测试工具
- 44-02: Excel 导入导出 + Cron 自动清理 + 告警降噪验证

**UI hint**: yes

### Phase 45: 工位详情整合 + 资产详情摘要 (R4)

**Goal**: 把对账健康度嵌入现有 ops/workstation 与 asset/card 页面，提供冲突详情抽屉。

**Depends on**: Phase 44

**Requirements**: INTEGRATE-01~03

**Success Criteria** (what must be TRUE):

  1. 工位详情页 `/ops/workstation/:id` 顶部显示健康度卡片（5 KPI + 趋势 mini chart）
  2. 资产设备子表 + 域控设备子表加入"对账健康"列（行内徽标）
  3. 点击徽标打开 ReconciliationDrawer（冲突摘要 / 历史变更 / 例外规则三 Tab）
  4. 资产详情页 `/asset/card/:id` 顶部摘要块显示对账状态
  5. 跨模块调用性能 ≤ 200ms（service 层直接调用 + 缓存）
  6. 无 `asset:reconciliation:list` 权限时静默隐藏（非 403）
  7. 工位 N+1 查询优化（后端聚合 API）
  8. 修复闭环解耦（修复回写不直接触发对账引擎）
  9. 抽屉"申请例外"按钮预填 IP/类型正确
  10. `npm run build` + `go build ./...` 全通过

**Plans**: 2 plans

Plans:
- 45-01: 工位详情页 + 资产详情页整合 + HealthCard/HealthBadge/ReconciliationDrawer 组件
- 45-02: 跨模块调用 + 权限降级 + N+1 优化 + operlog 接入

**UI hint**: yes

### Phase 46: 半自动修复（可选） (R5)

**Goal**: 高置信度（≥0.9）的 Type B/D 异常走"建议修复"流程（仍需人工确认，不直接覆盖）。

**Depends on**: Phase 45

**Requirements**: （Future, 非必需）

**Success Criteria** (what must be TRUE):

  1. 高置信度异常生成"修复建议"（含建议的 responsible_user_id + 理由）
  2. 人工确认 UI（一键接受 / 拒绝 / 修改建议）
  3. 一键回滚机制（所有修复可回滚到修复前状态）
  4. 误修复率 < 1%（监控 + 告警）
  5. 修复回写完整 operlog + raw_snapshot 留痕
  6. 状态机：建议 → 待确认 → 已接受 / 已拒绝 → 已应用
  7. R5 完成后 v1.17 milestone close

**Plans**: 2 plans (如果启动)

Plans:
- 46-01: 修复建议生成器 + 人工确认 UI
- 46-02: 一键回滚机制 + 误修复监控

**UI hint**: yes (建议独立页面)

### Progress Table

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 42. 资产对账观测底座 (R1) | 0/6 | Planning | — |
| 43. 告警 + 工单闭环 (R2) | 1/3 | In Progress|  |
| 44. 置信度评分 + IP 段例外 (R3) | 0/2 | Pending | — |
| 45. 工位详情整合 (R4) | 0/2 | Pending | — |
| 46. 半自动修复 (R5, 可选) | 0/2 | Pending | — |

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
- ✓ Phase 22A/22B — v1.12 深信服 VDI (6/6 plans)
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

---

*Last updated: 2026-06-26 — Phase 41 Plan 04 Task 1 verification gate PASS (audit-open debug_sessions=0 / verify_phase40.sh [ALL PASS] / validator 100.0%; 29/29 sessions closed; TECH-05+TECH-06 achieved); Task 2 (STATE/ROADMAP/REQUIREMENTS final closure + user final approval + push decision) queued for orchestrator as checkpoint:human-verify*
