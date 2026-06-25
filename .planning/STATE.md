---
gsd_state_version: 1.0
milestone: v1.16
milestone_name: 技术债清理 (Tech-Debt Cleanup)
status: planning
last_updated: "2026-06-25T08:30:00.000Z"
last_activity: 2026-06-25
progress:
  total_phases: 1
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

**Project**: RuoYi-Next 运维管理系统
**Created**: 2026-04-16
**Status**: Planning v1.16 — ROADMAP drafted, ready to plan Phase 40
**Last activity**: 2026-06-25 — v1.16 ROADMAP created (Phase 40: tech-debt-cleanup)

## Current Position

Phase: Phase 40 (tech-debt-cleanup)
Plan: —
Status: Ready to plan Phase 40
Last activity: 2026-06-25 — v1.16 roadmap drafted, 5 TECH requirements mapped to Phase 40

## Project Reference

**Core value**:
- 端到端运维可观测与审计能力:每个写操作产生可追溯记录(who/when/what/from-where/before-after-state),敏感字段自动脱敏
- Excel 驱动的批量管理(导入/导出含 dept-name → UUID 自动解析)
- 单一可信 IT 资产系统

**v1.16 目标**: 把 v1.15 累积的 22 个 deferred debug 会话 + 6 个 audit 数据质量问题集中处理,让系统回到"无遗留债务"状态,audit-open 中 debug_sessions < 5

## Phase Status (v1.16)

| Phase | Name | Status | Progress |
|-------|------|--------|----------|
| 40 | Tech-Debt Cleanup | 📋 Planning ready | 0/TBD plans |

## Completed Milestones

- ✅ **v1.0 工位导入部门/用户关联** (2026-04-16) — 2 phases, 7 plans
- ✅ **v1.1 信息点导入设备端口关联** (2026-04-16) — 1 phase, 1 plan
- ✅ **v1.2 可配置仪表盘生产级改造** (2026-04-21) — 4 phases, 11 plans
- ✅ **v1.3 技术债清理** (2026-04-27) — 3 phases, 9 plans
- ✅ **v1.4 MAC地址采集优化** (2026-05-09) — 1 phase, 4 plans
- ✅ **v1.5 MAC地址历史数据管理** (2026-06-15) — 4 phases, 26 plans
- ✅ **v1.6 API密钥管理系统** (2026-05-19) — 1 phase, 10 plans
- ✅ **v1.7 前后端加密配置同步** (2026-05-20) — 1 phase, 6 plans
- ✅ **v1.8 登录端点加密增强** (2026-05-21) — 1 phase, 4 plans
- ✅ **v1.9 AD域控集成扩展** (2026-05-24) — 2 phases, 11 plans
- ✅ **v1.10 网络设备权限修复** (2026-05-24) — 1 phase, 1 plan
- ✅ **v1.11 AD组自动同步系统** (2026-05-26) — 1 phase (23), 18 plans
- ✅ **v1.13 资产管理模块** (2026-06-08) — 1 phase (26), 6 plans
- ✅ **v1.14 全局列自定义** (2026-06-09) — 1 phase (27), 1 plan
- ✅ **v1.15 工位设备关联** (2026-06-10) — 1 phase (28), 4 plans
- ✅ **Phase 39 工位部门物理位置映射** (2026-06-25) — 1 phase (39), 8 plans

## Session History

| Date | Action | Phase |
|------|--------|-------|
| 2026-04-16 | Project initialized | — |
| ... (Phases 1-39 shipped, see full history in archive) | | |
| 2026-06-22 | Phase 37 完成 (前端部门选择组件统一收敛) - 6/6 plans | 37 |
| 2026-06-23 | Phase 38 Plan 01 完成 (Wave 1 前置 DI 脚手架) - 8 服务 struct 注入 AccountPool | 38 |
| 2026-06-25 | Milestone v1.16 started — ROADMAP drafted (Phase 40: tech-debt-cleanup) | 40 |

## Quick Tasks Completed (recent)

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260623-batch-sync | 用户导入后 AD 批量同步（BatchSyncUsersToAD）：复用单 LDAP 连接 + errgroup 信号量 | 2026-06-23 | - | [260623-batch-sync-optimize](./quick/20260623-batch-sync-optimize/) |
| 260622-lv9 | 系统用户 Excel 导入功能 + 性别默认值改保密 + 导入后触发 AD 域同步 | 2026-06-22 | 5d872a5,cc2a77d,7d40b32,4758d36 | [260622-lv9-xlsx-email](./quick/260622-lv9-xlsx-email/) |
| 260622-u8n | 从 sys_dept.leader 批量同步用户 AD manager 属性 | 2026-06-22 | - | [260622-u8n-ad-manager-leader-ad](./quick/260622-u8n-ad-manager-leader-ad/) |

## Accumulated Context

### v1.16 Tech-Debt Cleanup 范围

**22 个 deferred debug 会话** (frontmatter `deferred_to: v1.16-tech-debt`):
- 16 个 `root_cause_found`(AD 域控 6 + VDI 2 + infopoint/captcha/permission/api-key 等 8)
- 5 个 `awaiting_human_verify`(workstation-device-build-errors、workstation-expand-device-load-empty、reset-theme-hardcoded-colors、user-filter-lost-after-edit、vite-hmr-token-blank-page)
- 1 个 `root-cause-found`(apikey-route-path-duplication,连字符变体状态值)

**6 个 audit 数据质量问题**:
- `login-400-bad-request` / `ops-asset-constraint-uni-ops-asset-devicesn-not-exist` — 缺标准 YAML frontmatter
- `knowledge-base` — 非 debug 会话,需移出 audit 范围
- `info-point-type-null-import` / `sys-mac-filter-rules-relation-does-not-exist` — 实际已 resolved,frontmatter 需统一

**验收指标**:
- `gsd-sdk query audit-open` → `debug_sessions < 5`(从当前 57 大幅下降)
- `.planning/debug/*.md` → 100% 标准 YAML frontmatter

### Roadmap Evolution

- ✅ v1.0–v1.15 完成 (Phases 1-39, 137 plans + 8 gap-closure)
- 🚧 v1.16 规划中: Tech-Debt Cleanup (Phase 40, 5 TECH requirements)

### Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| 22 个 debug 会话修复可能引入回归 | Medium | 按 root_cause_found / awaiting_human_verify 分批,每批独立 commit,逐步验证 |
| 5 个 awaiting_human_verify 需在 dev 环境复现 | Low | dev 环境启动后逐个 case 复现,验证修复有效 |
| `knowledge-base` 改名可能影响其他工具 | Low | 优先选择 `skip_audit: true` 注释,避免破坏引用 |

## Next Steps

**当前选项**:

1. **规划 Phase 40 详细 plan**
   - 建议 3-wave 拆分:
     - **Wave 1**: 16 个 root_cause_found 修复 + 1 个 root-cause-found (TECH-01 + TECH-03,17 项代码修复 + commit)
     - **Wave 2**: 5 个 awaiting_human_verify dev 环境复现验证 (TECH-02,5 项需在 dev 启动)
     - **Wave 3**: 6 个 audit 数据质量整理 + 验收指标核验 (TECH-04 + TECH-05,frontmatter 修复 + 脚本验证)

2. **运行 `/gsd:plan-phase 40`** 进入详细 plan 阶段

**推荐**: Wave 1 优先(纯代码修复,可独立提交),Wave 2 需 dev 环境 + 用户配合复现

## Session Continuity

Last session: 2026-06-25T08:30:00.000Z
Stopped at: v1.16 ROADMAP created, Phase 40 ready to plan
Resume file: `.planning/ROADMAP.md`

**Active Phase:** 40 (tech-debt-cleanup) — 5 TECH requirements — 0 plans — ready for `/gsd:plan-phase 40`
