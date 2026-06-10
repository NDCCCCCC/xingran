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
**Plans:** 4 plans

**Wave 1 - 基础设施**
- [ ] 25-01-PLAN.md — 数据库迁移：细粒度权限菜单
- [ ] 25-02-PLAN.md — 后端数据范围过滤函数

**Wave 2 - 路由与前端**
- [ ] 25-03-PLAN.md — 路由权限中间件配置
- [ ] 25-04-PLAN.md — 前端动态按钮渲染

**Phase Highlights**:
- 细粒度操作权限分解（5个操作权限 + 1个绑定权限）
- 基于 bound_user_id 的数据范围过滤（5种数据范围）
- 无绑定用户特殊规则（仅 DataScopeAll 可见）
- 前端权限-按钮动态映射
---

### Phase 26: 资产管理模块 ✅

**Goal**: 创建完整的资产管理模块，包含数据库设计、后端 API 和前端页面，支持基于设备序列号 (DEVICESN) 的 Excel 导入/更新功能

**Depends on**: 无（独立新模块）
**Status**: ✅ Completed
**Plans**: 6 plans (3 waves)

**Requirements**:
- **数据模型**: 根据元数据定义创建包含40个字段的资产表结构
- **导入功能**: Excel 批量导入，根据设备序列号判断更新或新增
- **导出功能**: 支持导出当前资产数据到 Excel
- **CRUD 操作**: 完整的增删改查 API
- **前端页面**: 新的"资产管理"菜单下的列表页面

**技术规范**:
- 遵循项目 Handler-Service 架构模式
- 使用 UUID 作为主键
- 支持软删除 (deleted_at)
- 遵循 0=正常, 1=停用的状态值规范
- DeviceSN 作为唯一标识用于 Excel 导入 UpsertKey
- 参考 building/floor/workstation 模式

**Wave 1 - Database & Model**:
- [x] 26-01-PLAN.md — Asset model and database schema

**Wave 2 - Backend Service & API**:
- [x] 26-02-PLAN.md — Asset service layer with UUID validation
- [x] 26-03-PLAN.md — Asset API handlers and routes

**Wave 3 - Excel Import/Export & Frontend**:
- [x] 26-04-PLAN.md — Excel import/export configuration
- [x] 26-05-PLAN.md — Frontend asset list page
- [x] 26-06-PLAN.md — Menu and permission configuration

**Phase Highlights**:
- 📊 Complete asset tracking with 40 fields (device info, user/dept associations, network config, status)
- 🔄 Excel import with DeviceSN as upsert key (update existing, create new)
- 🔗 Automatic department/user resolution via ReferenceResolver
- ✅ UUID validation for foreign keys (dept_id, user_id)
- 📁 Department filtering with sub-department support
- 🎨 Frontend list page with table display and Excel operations

---

### Phase 27: 全局列自定义显示功能 ✅

**Goal**: 为系统中所有列表页面添加可自定义列显示的功能，解决资产列表页面字段过多（43列）的问题

**Depends on**: Phase 26 (资产管理模块 - 资产列表页面作为主要集成目标)
**Status**: ✅ Completed
**Plans**: 1 plan (8 tasks, 1 wave)

**Requirements**:
- **列显示/隐藏**: 用户可以选择显示哪些列
- **列拖拽排序**: 用户可以调整列的显示顺序
- **列宽度自定义**: 用户可以设置列宽度
- **持久化存储**: 配置保存到数据库，永久生效
- **重置功能**: 支持重置为默认配置
- **性能优化**: localStorage 缓存 + 数据库同步，切换列响应 < 200ms

**技术规范**:
- 页面 key 格式: `{module}.{list}` (如 `asset.list`, `user.list`)
- 用户隔离: 基于 user_id 的配置隔离
- 默认配置: 在代码中定义，首次加载自动应用
- 缓存策略: localStorage (5分钟) + 数据库 (永久)

**Wave 1 - Foundation**:
- [x] 27-01-PLAN.md — 数据库、后端 API、前端 Hook 和组件、资产列表集成

**Phase Highlights**:
- 🎯 Global column customization system for all list pages
- 📊 Support for asset list with 43 columns (grouped display)
- 🔄 Automatic save with immediate API call
- 💾 localStorage caching (5 minutes) + database persistence
- 🔒 User-scoped configurations with proper isolation
- 🖥️ Beautiful column selector UI with drag-and-drop
- ⚡ Fast performance (< 200ms response time)

---

### Phase 28: 工位设备关联子表格 ⚠️ Gap Closure Required

**Goal**: 为工位管理页面添加设备关联功能，通过可展开子表格显示该工位的设备，支持手动输入序列号匹配、域控设备同步、资产系统数据同步

**Depends on**: Phase 26 (资产管理模块 - 需要资产数据用于设备匹配)
**Status**: ✅ Complete - All gaps closed
**Plans**: 7 plans (7 completed)

**Requirements**:
- **子表格展示**: 工位列表中每行可展开显示关联设备
- **三个数据来源**:
  - 域控设备: 通过工位绑定用户匹配（最后登录账号）
  - 资产系统: 通过工位绑定用户匹配（责任人）
  - 实际结果: 优先显示，手动维护
- **手动输入**: 支持通过序列号从资产系统匹配设备信息
- **数据同步**: 一键同步域控或资产系统数据到实际结果
- **序列号匹配**: 域控设备同步时通过序列号匹配资产系统详细信息

**技术规范**:
- 前端可展开子表格组件（Ant Design Table expandable）
- 后端设备关联查询 API
- 工位-设备关联存储（ops_workstation_device 表）
- 域控设备查询服务（通过用户名）
- 资产系统设备查询服务（通过责任人）

**Wave 1 - 数据模型与后端服务**:
- [x] 28-01-PLAN.md — 工位设备关联表设计与迁移

**Wave 2 - API 层**:
- [x] 28-03-PLAN.md — 设备关联 API 端点

**Wave 3 - 前端集成**:
- [x] 28-04-PLAN.md — 设备关联子表格组件
- [x] 28-05-PLAN.md — 工位列表页面集成子表格

**Wave 4 - Gap Closure** (UAT 2026-06-10):
- [x] 28-06-PLAN.md — ✅ Gap 1 (Major): 简化手动添加设备模态框为序列号输入 + 自动匹配
- [x] 28-07-PLAN.md — ✅ Gap 2 (Cosmetic): 修复设备来源标签颜色区分

**UAT Diagnosed Gaps**:
1. **Gap 1 (Major)**: 手动添加设备模态框设计不符 - 应该只显示序列号输入字段，后端自动匹配资产信息
2. **Gap 2 (Cosmetic)**: 设备来源标签颜色未区分 - 域控蓝色、资产绿色、手动橙色

**Phase Highlights**:
- 🎯 Workstation-device association with sub-table display
- 🔍 Multi-source device matching (AD, Asset, Manual)
- 🔄 One-click sync from AD or Asset system
- 📊 Serial number based asset information matching
- 👤 User-based device association (bound user)
- 🖥️ Full-featured UI with add/edit/delete/sync operations
- ⚡ Performance optimization with lazy loading

---

    

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
| 25. 虚拟机数据范围权限配置 | - | 0/4 | Planned | - |
| 22A. VDI 基础集成 | v1.12 | 0/5 | Planned | - |
| 22B. VM Agent 服务 | v1.12 | 0/1 | Planned | - |
| 22C. 账号管理与密码 | v1.12 | 0/2 | Planned | - |
| 22D. Web Console 与监控 | v1.12 | 0/2 | Planned | - |
| 26. 资产管理模块 | v1.13 | 6/6 | Complete | 2026-06-08 |
| 27. 全局列自定义显示功能 | v1.14 | 1/1 | Complete | 2026-06-09 |
| 28. 工位设备关联子表格 | v1.15 | 4/6 | Gap Closure | 2026-06-10 |

**Total**: 96 plans completed, 15 plans planned (Phases 13-15), 14 plans planned (Phases 22 + 25), 2 gap closure plans (Phase 28), 127 total plans

**Completion Rate**: 75.6% (96/127) + 2 gap closure plans pending
