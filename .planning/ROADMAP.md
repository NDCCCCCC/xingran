# Roadmap: 工位导入部门/用户关联功能

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (已完成)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- 🚧 **v1.12 深信服桌面云集成** — Phases 22A-22D (部分完成)

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
<summary>✅ v1.5 MAC地址历史数据管理 (Phases 12-15) — SHIPPED 2026-06-15</summary>

- [x] Phase 12: 数据模型与采集集成 (5/5 plans) — completed 2026-05-11
- [x] Phase 13: 查询层与轨迹 (6/6 plans) — completed 2026-06-13
- [x] Phase 14: 前端与用户体验 (10/10 plans) — completed 2026-06-15
- [x] Phase 15: 性能优化 (5/5 plans) — completed 2026-06-15

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

### Phase 14: 前端与用户体验 📋

**Goal**: 基于 Phase 12 (数据模型与采集集成) 与 Phase 13 (查询层与轨迹) 的后端能力,补齐 MAC 地址历史数据管理的完整前端 UX — 包括查询列表页、轨迹可视化页、Excel 导出、菜单与权限注册、与网络设备/工位模块的联动入口,以及移动端响应式适配
**Depends on**: Phase 12 (数据模型与采集集成), Phase 13 (查询层与轨迹) — 后端 API 必须先就位
**Status**: 📋 Planned (待规划, 0/6 plans)

**预期子计划 (6 plans, 14-05 拆分为 14-05a + 14-05b)**:

- [ ] 14-01-PLAN.md — MAC 历史查询主列表页 (时间筛选、MAC/设备/端口筛选、分页、操作列)
- [ ] 14-02-PLAN.md — 轨迹可视化页 UX 增强 (ECharts Gantt 交互、停留时长热力、时间范围预设导出)
- [ ] 14-03-PLAN.md — 菜单/权限/路由注册 (与 sys_menu 同步,与现有网络设备模块并列展示)
- [ ] 14-04-PLAN.md — Excel 导出与批量操作 (后端 export 接口 + 前端 download 触发)
- [ ] 14-05a-PLAN.md — 共享三态组件抽取 (EmptyStateWithAction + ErrorAlertWithRetry,Wave 2)
- [ ] 14-05b-PLAN.md — 列表/轨迹页改造 + 网络设备联动入口 (Wave 3,保留 14-04 导出按钮)

**Phase Highlights**:

- 🎨 复用 Phase 13 已交付的 `MACTrajectoryChart.tsx`,不重写可视化核心
- 🔍 时间范围筛选复用 `useColumnConfig` 与现有 `useTableManager` hooks
- 📊 列表页与轨迹页共享 MAC 输入,实现"查询→可视化"单步跳转
- 🧭 菜单注册与网络设备模块并列,权限点 `network:mac:list/query/export`
- 📱 移动端使用 Ant Design 6 Grid + 卡片视图,继承现有 `HybridLayout` 侧边栏折叠行为
- ♻️ 严格遵守 React Query 缓存键规范,避免重复请求(参考 Phase 30 经验)

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

### Phase 30: 前端性能优化(消除瀑布流、包大小、重渲染、JS性能)

**Goal:** 基于 Vercel React 最佳实践审计，对 RuoYi-Next 前端做系统化性能优化：消除瀑布流、减小包大小、修复重渲染、提升 JS 性能。
**Requirements**:

- 首屏 JS ≤ 500KB（gzip）
- 首屏路由 LCP ≤ 2.5s
- vendor chunk 独立拆分：react / antd / echarts / three
- 4 个重库按需加载：three.js / echarts / xlsx / @uiw/react-md-editor
- 字典查询全局缓存 (React Query)
- 资产列表（43 列）启用虚拟滚动
- 5 条 ESLint 性能规则

**Depends on:** Phase 29
**Plans:** 5 plans (5 waves) ✅ ALL COMPLETE 2026-06-13

Plans:
**Wave 1**

- [x] 30-01-PLAN.md — Wave 1: 基础设施 (Vite visualizer、manualChunks、500KB 阈值、基线)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 30-02-PLAN.md — Wave 2: 重库按需加载 (three/echarts/xlsx/md-editor 懒加载)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 30-03-PLAN.md — Wave 3: 查询层 (useDict、QueryClient 默认值、useTableQuery)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 30-04-PLAN.md — Wave 4: 渲染层 (虚拟滚动、React.memo、5 条 ESLint 规则) ✅ 2026-06-13

**Wave 5** *(gap closure after verification — 23/25 must-haves)*

- [x] 30-05-PLAN.md — Wave 5: 缺口闭合 (useDict/useTableQuery/Widget/BaseEditModal 消费者接 + D-16 ESLint 降级 + gap-6 文档化) ✅ 2026-06-13 — **phase 30 fully shipped (25/25)**

### Phase 31: P0 收尾:连接池 lifecycle + Config API 重设计 ✅ COMPLETE 2026-06-13

**Goal:** 完成 20260612 审查报告剩余 2 项 P0 finding(F-14 connection_pool 引用计数与 lifecycle 重做、F-17 ConfigUpdateRequest 添加 ConfigKey 字段并重设计校验逻辑)
**Requirements:** F-14, F-17
**Depends on:** Phase 30
**Plans:** 5 plans, all complete

Plans:
- [x] 31-01-PLAN.md — F-14 GetConnection 内部 refCount +1 (commit cf41672)
- [x] 31-02-PLAN.md — F-14 caller 适配 4 处 (commit 2f770bf)
- [x] 31-03-PLAN.md — F-14 设计修订 ReleaseRef API + 6 个并发测试 (commit e464d72)
- [x] 31-04-PLAN.md — F-17 ConfigUpdateRequest + Update 校验 (commit e7b2b41)
- [x] 31-05-PLAN.md — F-17 前端类型同步 + SUMMARY (commit be7c40d)
- [x] UAT: 2 passed, 4 skipped (runtime), 0 issues (commit 470f1a9)

**Outcome:** P0 finding 100% 覆盖 (20 完整 + 2 部分 = 22/22),
28 RED baseline 7 轮 audit-fix + 1 phase 累计 0 净增加。

### Phase 32: v1.14 P1 重构与 P2 架构优化

**Goal:** 清理 20260612 后端代码审查报告中**所有非 P0 的 P1（47 项, 精选 15 项）与 P2（80+ 项）**问题。聚焦"安全加固 / 并发与一致性 / 业务逻辑修订 / 架构债与测试覆盖"四大方向，将系统从"功能完整、有隐患"推进到"生产可信、可维护"。

**Requirements**:

#### 🔒 P1 安全加固（7 项）
- [P1-S1] `pkg/crypto/sm2_jwt.go:222-266` — 手写 JWT 解析校验 `alg` 头，杜绝算法混淆攻击
- [P1-S2] `pkg/crypto/request_encryption.go:88-103` — timestamp 窗口从 ±300s 收紧到 ±60s（或可配置）
- [P1-S3] `pkg/crypto/nonce_storage.go` — 启用 `cleanupExpiredNonces` 定时清理，防止内存无限增长
- [P1-S4] `pkg/middleware/permission.go:106-118` — 取消子菜单自动继承父菜单权限，修复权限提升漏洞
- [P1-S5] `internal/core/security/password.go:25` — PBKDF2 迭代从 1000 提升到 ≥600000（OWASP 推荐）+ 平滑迁移
- [P1-S6] `internal/core/security/password.go:145-166` — `GenerateRandomPassword` 改用 `crypto/rand` 无偏置算法
- [P1-S7] `internal/services/operations/excel_handler.go:67-72` — Excel 文件 magic number 校验，不只校验后缀

#### ⚡ P1 并发与一致性（6 项）
- [P1-C1] `internal/services/addomain/sync.go:35` — AD 同步系列方法加 `singleflight` 互斥，防止 scheduler+手动触发并发
- [P1-C2] `internal/services/addomain/group_sync_service.go:336-376` — `handleDeletedGroups` 加阈值保护（如单次删除 ≥30% 视为 LDAP 故障，拒删）
- [P1-C3] `internal/services/addomain/dept_ou_mapper.go:60-100` — `UpsertMapping` 收敛为单事务，避免 delete/insert 中间崩溃
- [P1-C4] `internal/collectors/port_collector.go:353-381` — 端口状态批量插入（`clause.OnConflict`），保护连接池
- [P1-C5] `internal/websocket/notice_hub.go` — 添加 `readPump` + ping/pong 心跳，探测并清理僵尸连接
- [P1-C6] `internal/services/operations/excel_service.go:905-906` — `validateUniqueness` 改批量 `IN` 查询，消除 N+1

#### 🐛 P1 业务逻辑（2 项）
- [P1-B1] `internal/services/system/config_service.go:91-99` — `sys.request.encryption.enabled` 修改后立即失效缓存，无需重启
- [P1-B2] `internal/services/system/user_service.go:404` — 移除 `List` 中 `buildDepartmentPaths` 的重复调用

#### 🛠️ P2 架构债（精选 8 类）
- [P2-A1] `internal/core/core.go` — 拆分 god struct（25+ 字段）为 `CoreServices` / `CoreInfra` 两个组合根
- [P2-A2] 统一缓存键体系 — `cache_keys.go` 与 `data_cache_service.go` 两套定义合并到 `cache_keys.go`
- [P2-A3] 消除 `user_service.go` 与 `user_service_optimized.go` 并存 — 删除旧版，router 切到 optimized
- [P2-A4] 修复迁移文件编号冲突（027/028/029/030/031/036 多文件重号）— 重命名 + 注释来源
- [P2-A5] 统一错误处理 — `role_service` 等 `fmt.Errorf` 收敛到 `apperrors`，Handler 层一处映射
- [P2-A6] AD 模块关键路径单元测试补足（`ldap_client` Connect/Bind/Search、`group_sync`、`user_ou_service`），并删除/补全 `stripBaseDN_test.go`、`dept_ou_mapper_test.go` 空壳
- [P2-A7] `internal/device/` — Scrapli / Python subprocess 引入 process group + Wait + 僵尸进程定时清理
- [P2-A8] Excel 导入全量包裹事务（含 `processThreeLevelDepartments` 子流程），失败可回滚

**Depends on:** Phase 31（P0 收尾 — F-14 connection_pool lifecycle + F-17 ConfigUpdateRequest）
**Plans:** TBD (建议 7 plans，按 Wave 切分；运行 `/gsd-plan-phase 32` 细化)

**建议 Wave 拆分**:

- **Wave 1 — P1 安全加固快赢**: P1-S2 / P1-S3 / P1-S4 / P1-S7（无迁移、低风险）
- **Wave 2 — P1 密码强度迁移**: P1-S1 / P1-S5 / P1-S6（含密码哈希平滑迁移、需测试覆盖）
- **Wave 3 — P1 并发与一致性**: P1-C1 ~ P1-C6（多服务联动，需端到端验证）
- **Wave 4 — P1 业务逻辑收尾**: P1-B1 / P1-B2
- **Wave 5 — P2 架构债（重构）**: P2-A1 / P2-A2 / P2-A3 / P2-A5（架构调整，集中爆破）
- **Wave 6 — P2 迁移与子进程**: P2-A4 / P2-A7 / P2-A8
- **Wave 7 — P2 测试覆盖**: P2-A6

**Success Criteria**:

- 所有 15 项 P1 + 8 类 P2 在 PR 中可追溯到具体 commit
- `go vet ./...` 0 警告（消化 20260612 报告里 5 处 vet 警告）
- 关键路径测试覆盖率 ≥70%（AD/JWT/密码模块）
- 复审通过 — 与 20260612 报告同维度回审，所有列出问题闭环

**Source**: `.planning/reviews/20260612-backend-code-review.md`

Plans:
- [ ] TBD (运行 `/gsd-plan-phase 32` 生成详细 plan 文件)

### Phase 33: 前端 Vercel React Best Practices 修复 - 修复审查报告 20260613 中的 26 个性能/质量缺陷

**Goal:** [To be planned]
**Requirements**: TBD
**Depends on:** Phase 32
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 33 to break down)

### Phase 34: 操作日志全模块集成 📋

**Goal**: 为 RuoYi-Next 后端所有 309 个业务写操作端点集成操作日志记录，覆盖率从 2.9%（9/309）提升到 100%（309/309）。基于 AD 域控参考实现（`recordOperLog` 模式），按 7 个 Wave 分批实施，确保敏感字段（password/token/key）自动脱敏、异步写入不影响响应。

**Background**: 2026-06-15 审计发现：
- ✅ 基础设施完整（sys_oper_log 表 + OperLog 模型 + OperLogService + 查询 Handler/Router）
- ⚠️ `OperLogMiddleware` 已挂载到 9 个业务路由组但实际从未生效（因 handler 未调用 `SetOperLogInfo`）
- ✅ AD 域控的 `ad_domain_handler.go` 通过 `recordOperLog` 实现 9 处有效记录（唯一参考实现）
- ❌ 其他 300 个写操作端点未集成

**Depends on:** Phase 32（中间件架构依赖）
**Source:** `.planning/notes/260615-oper-log-coverage-audit.md`

**Requirements**:

- **基础设施增强**:
  - 将 `recordOperLog` helper 从 `internal/api/v1/system/helper.go` 迁移到共享包（`internal/utils/` 或 `pkg/operlog/`）
  - 扩展 `OperType` 业务类型常量（Grpc/Import/Export 等更精细）
  - 增强 `FilterSensitiveParams` 黑名单（增加 salt/privateKey/oldPassword/macKey/sm4Key 等）
  - helper 函数支持 `costTime`、`errorMsg`、`operParam` 附加字段

- **业务侧集成** (按 Wave 分批):
  - Wave 1 — system 核心: user / role / dept / menu / dict / post（47 端点）
  - Wave 2 — system 周边: notice / apikey / config / profile / settings / files（25 端点）
  - Wave 3 — operations 全部: 楼宇/楼层/工位/工位设备/机房/资产等（63 端点）
  - Wave 4 — network 全部: 设备/凭据/模板/命令/执行/备份/拓扑/MAC/端口（53 端点）
  - Wave 5 — vdi + workorder + duty + knowledge + scheduler（58 端点）
  - Wave 6 — monitor + rpa + agent（46 端点）
  - Wave 7 — 其他 system 子模块: dashboard / column-config / my-notices / notification-config / ou-group-mapping（8 端点）

- **端到端验证**:
  - 编写脚本枚举所有 309 个写操作端点
  - 逐个触发后查询 sys_oper_log 验证记录成功
  - 抽样检查 title / businessType 字段准确性
  - 验证敏感字段过滤（password 应显示为 `******`）
  - 验证异步写入性能（不应阻塞响应）

- **文档与回归**:
  - 更新 `docs/开发规范.md` 添加"新增 handler 必须调用 recordOperLog"约定
  - 补充单元测试覆盖 helper 函数边界情况
  - 删除中间件方案（保留中间件作为兜底安全网，handler 主动调用 recordOperLog 提供精确语义）

**Plans:** 4/10 plans executed

Plans:
**Wave 1 — 基础设施 (Foundation)**

- [x] 34-01-PLAN.md — Wave 1: 共享 operlog 包 (Record 可变参 + RecordWithBody + 24 OperType + 17 关键词 FilterSensitiveParams + 委托层改造) ✅ 2026-06-15 (commits 5f73691, 14a238d)

**Wave 2 — system 核心** *(依赖 Wave 1)*

- [x] 34-02-PLAN.md — system 核心: user / role / dept / menu / dict / post 集成 operlog.Record / RecordWithBody (31 写端点埋点 + WithCore() 链式注入 + SM4 兼容烟雾测试 TestResetPassword_SM4MiddlewareCompat) ✅ 2026-06-15 (commits ffd8bae, 6867d3e)

**Wave 2 — system 外围** *(依赖 Wave 1)*

- [x] 34-03-PLAN.md — system 外围: notice / apikey / config / profile / settings / file 集成 operlog.Record / RecordWithBody (23 写端点埋点 + WithCore() 链式注入 + apikey Create RecordWithBody 遮蔽 api_secret + profile ChangePassword 显式 WithOperParam 遮蔽 oldPassword/newPassword + 上传端点 multipart-safe {filename,size}) ✅ 2026-06-15 (commits d72fd0b, a47ec65)
- [x] 34-04-PLAN.md — operations 模块: building / floor / workstation / workstation_device / server_room / room_device / dedicated_line / infopoint / wall / door / floor_plan_text / excel / asset 集成 operlog.Record (56 写端点埋点 + WithCore() 链式注入 + Excel 导入只记录 filename+size+inserted/updated/failed 缓解 T-34-W3-01 + Excel 导出 FilterSensitiveParams + 模板下载 OperTypeDownload + workstation_device SyncAD/SyncAsset 用 OperTypeSync + SetPrimary 用 OperTypeGrant) ✅ 2026-06-16 (commit d7f8903)

**Success Criteria**:
- sys_oper_log 表有效记录数对应 309 个端点，**覆盖率 ≥ 95%**（允许少量内部管理端点豁免）
- 抽样 30 个端点验证 title 与业务语义匹配，businessType 与操作类型匹配
- 密码/密钥类字段全部脱敏为 `******`
- 操作日志写入不阻塞 API 响应（P95 < 50ms）
- 每个 Wave 独立 commit 便于回滚

**Phase Highlights**:
- 📊 全模块操作日志覆盖从 2.9% 提升到 95%+
- 🔒 敏感字段自动脱敏
- ⚡ 异步写入，不影响业务性能
- 🎯 与 AD 域控参考实现保持一致
- 🔄 分 7 Wave 渐进式集成，风险可控

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
| 13. 查询层与轨迹 | v1.5 | 6/6 | Complete | 2026-06-13 |
| 14. 前端与用户体验 | v1.5 | 10/10 | Complete | 2026-06-15 |
| 15. 性能优化 | v1.5 | 5/5 | Complete | 2026-06-15 |
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
| 34. 操作日志全模块集成 | - | 4/10 | In Progress|  |

**Total**: 122 plans completed, 8 gap closure plans, 24 plans planned (Phases 22C + 22D + 25 + 34), 154 total plans

**Completion Rate**: 84.7% (122/144) + 8 gap closure plans shipped
