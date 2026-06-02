# Roadmap: 工位导入部门/用户关联功能

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- 🚧 **v1.5 MAC地址历史数据管理** — Phases 12-15 (部分完成)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- 📋 **v1.12 深信服桌面云集成** — Phases 22A-22D (planned)

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

<details>
<summary>✅ v1.4 MAC地址采集优化 (Phase 11) — SHIPPED 2026-05-09</summary>

- [x] Phase 11: MAC地址采集优化 - 过滤设备间互联端口 (4/4 plans) — completed 2026-05-09

</details>

---

<details>
<summary>🚧 v1.5 MAC地址历史数据管理 (Phases 12-15) — PARTIALLY COMPLETE</summary>

- [x] Phase 12: 数据模型与采集集成 (5/5 plans) — completed 2026-05-11
- [ ] Phase 13: 查询层与轨迹 (0/5 plans) — planned
- [ ] Phase 14: 前端与用户体验 (0/5 plans) — planned
- [ ] Phase 15: 性能优化 (0/5 plans) — planned

</details>

---

<details>
<summary>✅ v1.6 API密钥管理系统 (Phase 16) — SHIPPED 2026-05-19</summary>

- [x] Phase 16: API密钥管理 (10/10 plans) — completed 2026-05-19

</details>

<details>
<summary>✅ v1.7 前后端加密配置同步 (Phase 17) — SHIPPED 2026-05-20</summary>

- [x] Phase 17: 前后端加密配置同步 (6/6 plans) — completed 2026-05-20

</details>

---

<details>
<summary>✅ v1.8 登录端点加密增强 (Phase 18) — SHIPPED 2026-05-21</summary>

- [x] Phase 18: 登录端点请求体加密 (4/4 plans) — completed 2026-05-21

</details>

---

<details>
<summary>✅ v1.9 AD域控集成扩展 (Phases 19-20) — SHIPPED 2026-05-24</summary>

- [x] Phase 19: AD域控账号登录功能 (6/6 plans) — completed 2026-05-21
- [x] Phase 20: AD域控OU与部门映射 (5/5 plans) — completed 2026-05-24

</details>

---

<details>
<summary>✅ v1.10 网络设备权限修复 (Phase 21) — SHIPPED 2026-05-24</summary>

- [x] Phase 21: 网络设备管理权限泄露修复 (1/1 plan) — completed 2026-05-24

</details>

---

<details>
<summary>✅ v1.11 AD组自动同步系统 (Phase 23) — SHIPPED 2026-05-26</summary>

**Goal**: 实现AD组自动同步功能，将系统部门与AD组建立映射关系，实现部门成员自动成为对应组的成员，并确保每个成员只能属于一个组。

**Depends on**: Phase 20 (AD域控OU与部门映射)

**Plans**: 18 total plans

**Wave 1 - Foundation**:
- [x] 23-08-PLAN.md — 添加MemberOUDN配置字段
- [x] 23-09-PLAN.md — 部门-组映射数据模型

**Wave 2 - Service Layer**:
- [x] 23-10-PLAN.md — 部门-组映射服务
- [x] 23-12-PLAN.md — AD组管理服务

**Wave 3 - Core Sync Logic**:
- [x] 23-11-PLAN.md — 成员同步服务-核心同步逻辑

**Wave 4 - Advanced Features**:
- [x] 23-14-PLAN.md — 变更处理和排他性成员关系

**Wave 5 - Scheduler Integration**:
- [x] 23-13-PLAN.md — 定时组同步任务注册

**Wave 6 - API Layer**:
- [x] 23-06-PLAN.md — API接口层-AD组同步HTTP端点

**Wave 7 - Configuration Management**:
- [x] 23-15-PLAN.md — 配置管理-组同步系统参数

**Wave 8 - Frontend UI**:
- [x] 23-16-PLAN.md — 前端UI-部门-组映射管理页面
- [x] 23-17-PLAN.md — 前端UI-同步监控页面
- [x] 23-18-PLAN.md — 前端UI-批量自动映射页面

**Phase Highlights**:
- 🔄 Complete automated group synchronization system
- 📊 Real-time sync monitoring and logging
- 🎯 Automatic department-to-group mapping (cxhub-{dept} naming)
- 🔒 Exclusive group membership enforcement
- ⚙️ Flexible configuration management
- 🖥️ Full-featured UI with mapping management, monitoring, and batch operations

</details>

---

### Phase 22: 深信服桌面云集成 📋
**Goal**: 集成深信服虚拟桌面基础设施（VDI），实现虚拟机管理、服务器配置、用户关联和同步功能
**Depends on**: 无（独立功能）
**Status**: 📋 Planned (拆分为 4 个子阶段)

**Phase 22A: VDI 基础集成** (5 plans)
- [ ] 22-01-PLAN.md — VDI数据模型与配置基础
- [ ] 22-02-PLAN.md — VDI API客户端与认证
- [ ] 22-03-PLAN.md — VDI服务层实现
- [ ] 22-04-PLAN.md — VDI后端API层
- [ ] 22-05-PLAN.md — VDI前端UI实现

**Phase 22B: VM Agent 服务** (1 plan)
- [ ] 22-06-PLAN.md — VM Agent 服务集成

**Phase 22C: 账号管理与密码** (2 plans)
- [ ] 22-07-PLAN.md — VDI账号管理
- [ ] 22-08-PLAN.md — VDI密码管理

**Phase 22D: Web Console 与监控** (2 plans)
- [ ] 22-09-PLAN.md — Web Console 集成
- [ ] 22-10-PLAN.md — VDI监控功能

**依赖关系**: 22A → 22B → 22C → 22D

**Phase Highlights**:
- 🔐 Production-grade security: AES-128-GCM password encryption
- 🔄 Complete VDI API integration: All operations call real Sangfor VDI endpoints
- 📊 Comprehensive VM management: CRUD, power operations, IP config, rename, user binding
- 🚀 Batch operations: Support for bulk VM operations with VDI API calls
- 🎨 Full-featured UI: Virtual machine list, server configuration, operation buttons
- 🔄 Data synchronization: VM status sync from VDI to local database

### Phase 24: UUID类型一致性优化

**Goal:** 统一代码库中的 UUID 类型处理模式，确保新代码遵循一致的规范。本阶段不涉及大规模重构已有表，而是建立清晰的规范指导未来开发，并通过文档和测试确保规范得到遵守。

**Requirements**:
- 统一 UUID 生成策略为 Go 侧生成
- 保持 Go 层使用 string 类型（向后兼容）
- 明确外部系统 UUID 字段处理方式
- 建立迁移脚本的编写规范
- 补充单元测试确保规范执行
- 更新开发规范文档

**Depends on:** 无（独立规范整理工作）
**Plans:** 2 plans

**Wave 1 - 文档与规范**:
- [x] 24-01-PLAN.md — 开发规范文档更新
- [x] 24-02-PLAN.md — 单元测试覆盖

**Phase Highlights**:
- 📚 Complete UUID handling standards documentation
- 🧪 Test coverage for UUID generation and validation
- 📝 Clear migration script patterns
- 🚫 Anti-pattern warnings for developers

### Phase 25: 虚拟机数据范围权限配置

**Goal:** 为虚拟机列表配置细粒度权限，包括启动、关机、重启、同步、删除等操作；修改绑定用户功能以支持数据范围权限识别

**Requirements**:
- 各种操作（启动、关机、重启、同步、删除）需要单独配置权限
- 绑定用户功能仅用于数据范围权限识别：
  - 无绑定用户：仅数据范围为"全部"的角色可见
  - 绑定用户张三：张三所在部门和张三本人可见

**Depends on:** Phase 22 (深信服桌面云集成)
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 25 to break down)

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
| 11. MAC地址采集优化 - 过滤设备间互联端口 | v1.4 | 4/4 | Complete | 2026-05-09 |
| 12. 数据模型与采集集成 | v1.5 | 5/5 | Complete | 2026-05-11 |
| 13. 查询层与轨迹 | v1.5 | 0/5 | Planned | - |
| 14. 前端与用户体验 | v1.5 | 0/5 | Planned | - |
| 15. 性能优化 | v1.5 | 0/5 | Planned | - |
| 16. API 密钥管理 | v1.6 | 10/10 | Complete | 2026-05-19 |
| 17. 前后端加密配置同步 | v1.7 | 6/6 | Complete | 2026-05-20 |
| 18. 登录端点请求体加密 | v1.8 | 4/4 | Complete | 2026-05-21 |
| 19. AD域控账号登录功能 | v1.9 | 6/6 | Complete | 2026-05-21 |
| 20. AD域控OU与部门映射 | v1.9 | 5/5 | Complete | 2026-05-24 |
| 21. 网络设备管理权限泄露修复 | v1.10 | 1/1 | Complete | 2026-05-24 |
| 23. AD组自动同步系统 | v1.11 | 18/18 | Complete | 2026-05-26 |
| 24. UUID类型一致性优化 | - | 2/2 | Complete | 2026-05-27 |
| 25. 虚拟机数据范围权限配置 | - | 0/0 | Planned | - |
| 22A. VDI 基础集成 | v1.12 | 0/5 | Planned | - |
| 22B. VM Agent 服务 | v1.12 | 0/1 | Planned | - |
| 22C. 账号管理与密码 | v1.12 | 0/2 | Planned | - |
| 22D. Web Console 与监控 | v1.12 | 0/2 | Planned | - |

**Total**: 85 plans completed, 15 plans planned (Phases 13-15), 10 plans planned (Phase 22), 110 total plans

**Completion Rate**: 77% (85/110)
