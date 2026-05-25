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
- 📋 **v1.11 深信服桌面云集成** — Phases 22A-22D (planned)

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
<summary>📋 v1.11 深信服桌面云集成 (Phases 22A-22D) — PLANNED</summary>

- [🔄] Phase 22A: VDI 基础集成 (5 plans) — 12-15h (2/5 plans completed)
- [ ] Phase 22B: VM Agent 服务 (1 plan) — 6-8h
- [ ] Phase 22C: 账号管理与密码 (2 plans) — 7-9h
- [ ] Phase 22D: Web Console 与监控 (2 plans) — 6-8h

**依赖关系**: 22A → 22B → 22C → 22D

**说明**: 原 Phase 22 拆分为 4 个子阶段，每个阶段可独立执行，上下文更充足

</details>

---

## Phase Details

### Phase 12: 数据模型与采集集成 ✅
**Goal**: 建立历史数据基础设施，最小侵入集成到现有采集流程
**Depends on**: Phase 11 (v1.4 MAC采集优化)
**Requirements**: DATA-01, DATA-02, QUERY-01
**Status**: ✅ Completed 2026-05-11
**Plans**:
- [x] 12-01-PLAN.md — 创建历史数据模型和变更检测
- [x] 12-02-PLAN.md — 实现分区管理和自动清理
- [x] 12-03-PLAN.md — 创建历史数据查询服务
- [x] 12-04-PLAN.md — 修复MAC历史路由注册位置
- [x] 12-05-PLAN.md — 修复清理任务注册时机

### Phase 13: 查询层与轨迹 📋
**Goal**: 提供强大的历史查询能力和MAC轨迹追踪功能
**Depends on**: Phase 12
**Requirements**: QUERY-02, QUERY-03, QUERY-04, UI-03
**Status**: 📋 Planned
**Plans**: TBD (5 plans expected)
**UI hint**: yes

### Phase 14: 前端与用户体验 📋
**Goal**: 完善前端交互，提供直观的历史数据探索体验
**Depends on**: Phase 13
**Requirements**: UI-01, UI-02, UI-04
**Status**: 📋 Planned
**Plans**: TBD (5 plans expected)
**UI hint**: yes

### Phase 15: 性能优化 📋
**Goal**: 优化大规模数据场景下的性能，确保生产环境稳定性
**Depends on**: Phase 14
**Requirements**: PERF-01, PERF-02, PERF-03, PERF-04
**Status**: 📋 Planned
**Plans**: TBD (5 plans expected)
**UI hint**: yes

### Phase 16: API 密钥管理 ✅
**Goal**: 实现完整的 API 密钥管理系统，支持密钥的 CRUD 操作、自动生成、权限控制、使用日志审计和前端管理页面
**Depends on**: 无（独立功能）
**Status**: ✅ Completed 2026-05-19
**Plans**: 10/10 complete

Plans:
- [x] 16-01-PLAN.md — 创建数据模型和数据库迁移
- [x] 16-02-PLAN.md — 实现服务层 - 基础 CRUD
- [x] 16-02a-PLAN.md — 实现服务层 - 使用日志和统计
- [x] 16-03-PLAN.md — 创建处理器和路由
- [x] 16-04-PLAN.md — 实现认证中间件和速率限制器
- [x] 16-05a-PLAN.md — 创建前端类型定义和 API 客户端
- [x] 16-05b-PLAN.md — 创建前端管理页面组件
- [x] 16-06-PLAN.md — 后端测试
- [x] 16-07-PLAN.md — 前端测试
- [x] 16-08-PLAN.md — 集成测试和覆盖率报告
- [x] 16-09-PLAN.md — 测试问题修复
- [x] 16-10-PLAN.md — 测试完成

**Coverage**: 后端 81.1%, 前端 85%

### Phase 17: 前后端加密配置同步 ✅
**Goal**: 实现前后端加密配置的运行时同步，支持请求和响应加密的统一配置管理
**Depends on**: 无（独立功能）
**Status**: ✅ Completed 2026-05-20
**Plans**: 6/6 complete

Plans:
- [x] 17-01-PLAN.md — 创建后端公共加密配置端点
- [x] 17-02-PLAN.md — 实现前端加密配置服务与缓存
- [x] 17-03-PLAN.md — 集成Token刷新流程与手动刷新工具
- [x] 17-04-PLAN.md — 启用响应加密与数据库配置同步
- [x] 17-05-PLAN.md — 实现前端响应解密与UI刷新
- [x] 17-06-PLAN.md — 测试与文档

### Phase 18: 登录端点请求体加密 ✅
**Goal**: 为登录端点 `/system/auth/login` 启用 SM2+SM4 混合请求体加密，实现三层加密保护
**Depends on**: Phase 17 (前后端加密配置同步)
**Status**: ✅ Completed 2026-05-21
**Plans**: 4/4 complete

Plans:
- [x] 18-01-PLAN.md — 配置文件修改（移除登录端点排除/黑名单）
- [x] 18-02-PLAN.md — 单元测试（后端+前端加密逻辑测试）
- [x] 18-03-PLAN.md — 集成测试（E2E测试、性能测试、安全测试）
- [x] 18-04-PLAN.md — 部署和验证（部署文档、安全文档、最终验证）

### Phase 19: AD域控账号登录功能 ✅
**Goal**: 在现有AD域控管理基础上实现域控账号登录功能，包括：策略模式认证系统、初次登录自动同步用户信息、参数管理页面AD登录开关、前端SM4+SM2加密集成
**Depends on**: Phase 18
**Status**: ✅ Completed 2026-05-21
**Plans**: 6/6 complete

Plans:
- [x] 19-01-PLAN.md — 创建AD认证策略和配置系统
- [x] 19-02-PLAN.md — 实现AD认证策略和登录逻辑
- [x] 19-03-PLAN.md — 实现参数管理页面AD登录开关
- [x] 19-04-PLAN.md — 前端登录界面SM4+SM2加密集成
- [x] 19-05-PLAN.md — 后端和前端单元测试
- [x] 19-06-PLAN.md — 集成测试和文档

### Phase 20: AD域控OU与部门映射 ✅
**Goal**: 实现系统部门与AD域控OU的双向映射功能，包括定时同步、用户登录部门设置、用户修改AD同步
**Depends on**: Phase 19 (AD域控账号登录功能)
**Status**: ✅ Completed 2026-05-24
**Plans**: 5/5 complete

Plans:
- [x] 20-01-PLAN.md — 创建数据模型和基础服务
- [x] 20-02-PLAN.md — 实现部门定时同步功能
- [x] 20-03-PLAN.md — 实现用户登录部门设置
- [x] 20-04-PLAN.md — 实现用户修改AD同步
- [x] 20-05-PLAN.md — 测试与验证

### Phase 21: 网络设备管理权限泄露修复 ✅
**Goal**: 修复网络设备路由组权限泄露问题，将OR逻辑的权限中间件改为细粒度的子路由权限控制
**Depends on**: 无（安全修复）
**Status**: ✅ Completed 2026-05-24
**Plans**: 1/1 complete

Plans:
- [x] 21-01-PLAN.md — 修复网络设备权限泄露（移除OR逻辑权限，实现细粒度子路由权限）

**Security Impact**: 修复了用户拥有单个网络权限即可访问所有网络端点的严重安全漏洞

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
| 22A. VDI 基础集成 | v1.11 | 0/5 | Planned | - |
| 22B. VM Agent 服务 | v1.11 | 0/1 | Planned | - |
| 22C. 账号管理与密码 | v1.11 | 0/2 | Planned | - |
| 22D. Web Console 与监控 | v1.11 | 0/2 | Planned | - |

**Total**: 65 plans completed, 15 plans planned (Phases 13-15), 10 plans planned (Phase 22), 90 total plans

**Completion Rate**: 72% (65/90)
