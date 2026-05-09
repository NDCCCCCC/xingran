# Roadmap: 工位导入部门/用户关联功能

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)

## Phases

<details>
<summary>✅ v1.0 工位导入部门/用户关联 (Phases 1-2) — SHIPPED 2026-04-16</summary>

- [x] Phase 1: 后端导入配置 (3/3 plans) — completed 2026-04-16
- [x] Phase 2: 验证与导出 (4/4 plans) — completed 2026-04-16

</details>

<details>
<summary>✅ v1.1 信息点导入设备端口关联 (Phase 3) — SHIPPED 2026-04-16</summary>

- [x] Phase 3: 信息点导入设备端口配置 (1/1 plan) — completed 2026-04-16

</details>

<details>
<summary>✅ v1.2 可配置仪表盘生产级改造 (Phases 4-7) — SHIPPED 2026-04-21</summary>

- [x] Phase 4: Widget 数据获取机制 (3/3 plans) — completed 2026-04-20
- [x] Phase 5: 前端交互完善 (3/3 plans) — completed 2026-04-21
- [x] Phase 6: 实时数据刷新 (3/3 plans) — completed 2026-04-21
- [x] Phase 7: 用户体验优化 (2/2 plans) — completed 2026-04-21

</details>

<details>
<summary>✅ v1.3 技术债清理 (Phases 8-10) — SHIPPED 2026-04-27</summary>

- [x] Phase 8: SNMP Panic 修复 (3/3 plans) — completed 2026-04-27
- [x] Phase 9: 后端代码优化 (5/5 plans) — completed 2026-04-27
- [x] Phase 10: 网络设备导出集成 (1/1 plan) — completed 2026-04-27

</details>

---

<details>
<summary>🚧 v1.4 MAC地址采集优化 (Phase 11) — IN PROGRESS</summary>

- [ ] Phase 11: MAC地址采集优化 - 过滤设备间互联端口 (4/4 plans) — planned 2026-05-09

**Plans:**
- [ ] 11-01-PLAN.md — LLDP discovery service implementation
- [ ] 11-02-PLAN.md — MAC filtering enhancement with LLDP
- [ ] 11-03-PLAN.md — Configurable filter rules and API
- [ ] 11-04-PLAN.md — Comprehensive testing and validation

**Wave Structure:**
- Wave 1 (parallel): 11-01 (LLDP service), 11-02 (MAC filtering)
- Wave 2 (parallel): 11-03 (Filter rules API), 11-04 (Testing)

**Requirements:**
- MAC-01: LLDP discovery identifies uplink ports
- MAC-02: MAC filtering excludes LLDP neighbor ports
- MAC-03: MAC filtering respects count threshold
- MAC-04: LLDP failure doesn't block MAC collection
- MAC-05: Configurable filtering rules work

</details>

---

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. 后端导入配置 | v1.0 | 3/3 | Complete | 2026-04-16 |
| 2. 验证与导出 | v1.0 | 4/4 | Complete | 2026-04-16 |
| 3. 信息点导入设备端口配置 | v1.1 | 1/1 | Complete | 2026-04-16 |
| 4. Widget 数据获取机制 | v1.2 | 3/3 | Complete | 2026-04-20 |
| 5. 前端交互完善 | v1.2 | 3/3 | Complete | 2026-04-21 |
| 6. 实时数据刷新 | v1.2 | 3/3 | Complete | 2026-04-21 |
| 7. 用户体验优化 | v1.2 | 2/2 | Complete | 2026-04-21 |
| 8. SNMP Panic 修复 | v1.3 | 3/3 | Complete | 2026-04-27 |
| 9. 后端代码优化 | v1.3 | 5/5 | Complete | 2026-04-27 |
| 10. 网络设备导出集成 | v1.3 | 1/1 | Complete | 2026-04-27 |
| 11. MAC地址采集优化 - 过滤设备间互联端口 | v1.4 | 0/4 | Planning | - |

---

**Total**: 11 phases, 31 plans, 1 in planning 🚧
