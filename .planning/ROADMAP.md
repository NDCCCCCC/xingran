---
last_updated: 2026-09-06
milestone: v1.30
status: planning
---

# Roadmap: XingRan-Next

## Milestones

- 🚧 **v1.30 V130 缺陷治理 (Defect Remediation)** — Phases 96-101 (roadmap created 2026-09-06) → [workstream ROADMAP（完整 phase 详情）](workstreams/milestone/ROADMAP.md) · [REQUIREMENTS](REQUIREMENTS.md)
- ✅ **v1.29 技术债治理 (Tech Debt Governance)** — Phases 89-95, 7 phases / 26 plans（shipped 2026-09-06）→ [完整 ROADMAP 归档](milestones/v1.29-ROADMAP.md) · [MILESTONE-AUDIT](milestones/v1.29-MILESTONE-AUDIT.md) · [SHIPPED 后深度复查](milestones/v1.29-DEEP-RECHECK.md)
- ✅ **v1.28 前端测试覆盖率** — Phases 82-88（shipped 2026-09-04，阶段性收口 45.13%）→ [frontend-coverage workstream 归档](frontend-coverage-baseline.md)
- ✅ **v1.27 后端覆盖率 ratchet** — Phases 75-81（shipped 2026-08-23）
- ✅ 更早里程碑 — 见 [MILESTONES.md](MILESTONES.md) 与 milestones/ 目录

## Phases

### 🚧 v1.30 V130 缺陷治理 (Phases 96-101, In Progress)

**Milestone Goal:** 修复 v1.29 期间登记的全部 18 项 V130-CANDIDATES 缺陷候选 + 闭环 2 个 deferred 小项（4 个未跟踪测试文件入库决策 + 62-HUMAN-UAT 3 场景）。所有修复属行为变更，每项附回归测试；七 gate 全程不倒退。

- [ ] **Phase 96: 确定性缓存/看板缺陷修复** — CACHEDEF-01..05 + JOBSTAT-01 六项确定性缺陷，每项附回归测试
- [ ] **Phase 97: config_backup 恢复链加固** — V130R-01..03（互斥原子性/实例归属/业务错误码；含 3 个 discuss 设计决策）
- [ ] **Phase 98: 缓存键安全与 base 迁移收尾** — V130R-04..05（列表键防碰撞 + 四包 interface{} 残留迁 base 泛型）
- [ ] **Phase 99: operations 口径统一** — V130R-06..09（Total 软删/换楼乱序/orgId 子部门筛选/分页 clamp 收敛；V130R-09 含 discuss）
- [ ] **Phase 100: 前端契约修复** — V130R-10..12（幽灵方法处置/rpaApi 契约对齐/networkApi 下载链收敛）
- [ ] **Phase 101: 收口** — TESTFILE-01 + UAT62-01..03 + 七 gate 全绿 + v1.30 audit

完整 phase 详情（goal/SC/requirements 映射/依赖图）：[workstreams/milestone/ROADMAP.md](workstreams/milestone/ROADMAP.md)

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

| Phase | Milestone | Plans Complete | Status | Completed |
| ----- | --------- | -------------- | ------ | --------- |
| 96. 确定性缓存/看板缺陷修复 | v1.30 | 0/? | Not started | - |
| 97. config_backup 恢复链加固 | v1.30 | 0/? | Not started | - |
| 98. 缓存键安全与 base 迁移收尾 | v1.30 | 0/? | Not started | - |
| 99. operations 口径统一 | v1.30 | 0/? | Not started | - |
| 100. 前端契约修复 | v1.30 | 0/? | Not started | - |
| 101. 收口（测试文件入库 + 62-UAT + audit） | v1.30 | 0/? | Not started | - |
