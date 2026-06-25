# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15)
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
- 🚧 **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phase 40 (planning)

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

**Plans**: TBD

**UI hint**: no

### Progress Table

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 40. Tech-Debt Cleanup | 1/3 | In Progress|  |

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

*Last updated: 2026-06-25 — Milestone v1.16 (Tech-Debt Cleanup) planning complete*
