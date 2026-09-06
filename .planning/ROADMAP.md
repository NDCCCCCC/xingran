---
last_updated: 2026-09-07
milestone: v1.30
status: shipped
---

# Roadmap: XingRan-Next

## Milestones

- ✅ **v1.30 V130 缺陷治理 (Defect Remediation)** — Phases 96-101，SHIPPED 2026-09-07 → [归档 ROADMAP](milestones/v1.30-ROADMAP.md) · [AUDIT](milestones/v1.30-MILESTONE-AUDIT.md) · [REQUIREMENTS](milestones/v1.30-REQUIREMENTS.md)
- ✅ **v1.29 技术债治理 (Tech Debt Governance)** — Phases 89-95, 7 phases / 26 plans（shipped 2026-09-06）→ [完整 ROADMAP 归档](milestones/v1.29-ROADMAP.md) · [MILESTONE-AUDIT](milestones/v1.29-MILESTONE-AUDIT.md) · [SHIPPED 后深度复查](milestones/v1.29-DEEP-RECHECK.md)
- ✅ **v1.28 前端测试覆盖率** — Phases 82-88（shipped 2026-09-04，阶段性收口 45.13%）→ [frontend-coverage workstream 归档](frontend-coverage-baseline.md)
- ✅ **v1.27 后端覆盖率 ratchet** — Phases 75-81（shipped 2026-08-23）
- ✅ 更早里程碑 — 见 [MILESTONES.md](MILESTONES.md) 与 milestones/ 目录

## Phases

<details>
<summary>✅ v1.30 V130 缺陷治理 (Phases 96-101) — SHIPPED 2026-09-07</summary>

**Milestone Goal:** 修复 v1.29 期间登记的全部 18 项 V130-CANDIDATES 缺陷候选 + 闭环 2 个 deferred 小项（4 个未跟踪测试文件入库决策 + 62-HUMAN-UAT 3 场景）。所有修复属行为变更，每项附回归测试；七 gate 全程不倒退。

- [x] Phase 96: 确定性缓存/看板缺陷修复 (3/3 plans) — 2026-09-06
- [x] Phase 97: config_backup 恢复链加固 (3/3 plans，2026-09-07 recovery 落库 2dd46a3)
- [x] Phase 98: 缓存键安全与 base 迁移收尾 (2/2 plans) — 2026-09-06
- [x] Phase 99: operations 口径统一 (6/6 plans 含 99-06 gap closure) — 2026-09-07
- [x] Phase 100: 前端契约修复 (3/3 plans，100-VERIFICATION 6/6 SC) — 2026-09-07
- [x] Phase 101: 收口——测试文件入库 + 62-UAT + audit (2/2 plans；UAT62-01/02 待人工，runbook 就绪) — 2026-09-07

完整 phase 详情（goal/SC/requirements 映射/依赖图）：[milestones/v1.30-ROADMAP.md](milestones/v1.30-ROADMAP.md)

</details>

<details>
<summary>✅ v1.29 技术债治理 (Phases 89-95) — SHIPPED 2026-09-06</summary>

- [x] Phase 89: PAGINATION 常量集中化 (3/3 plans) — 2026-09-04
- [x] Phase 90: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化 (4/4 plans) — 2026-09-04
- [x] Phase 91: CRUD 复用 base.Repository[T] (4/4 plans) — 2026-09-04
- [x] Phase 92: 缓存层三处架构统一 (4/4 plans) — 2026-09-05
- [x] Phase 93: config_backup 三处 TODO 闭环 (6/6 plans) — 2026-09-05
- [x] Phase 94: 前端 API 工厂化 (3/3 plans) — 2026-09-06
- [x] Phase 95: v1.28 SHIP 收口 + v1.29 closeout + audit (2/2 plans) — 2026-09-06

完整 phase 详情（goal/SC/requirements 映射/wave 编排）：[milestones/v1.29-ROADMAP.md](milestones/v1.29-ROADMAP.md)

</details>

## Progress

当前无进行中 milestone。上一 milestone v1.30 已 SHIPPED（6 phases / 21 plans / 22 requirements，见 [MILESTONES.md](MILESTONES.md)）。

人工验证台账（v1.30 遗留，待用户执行）：

| 项 | 内容 | 就绪资产 |
|----|------|---------|
| UAT62-01 | Migrate176 R1/R2→R5 真实 PG 就地升级 | [UAT-RUNBOOK §1](phases/101-closeout-uat-audit/UAT-RUNBOOK.md) |
| UAT62-02 | Advisory lock 双实例迁移保护 | [UAT-RUNBOOK §2](phases/101-closeout-uat-audit/UAT-RUNBOOK.md) |
| FE-1..3 | VDI 批量操作 e2e / 浏览器下载体验 / VM Tab 视觉走查 | [100-VERIFICATION](milestones/v1.30-phases/100-frontend-contract-fixes/100-VERIFICATION.md) |
