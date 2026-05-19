# Roadmap: 工位导入部门/用户关联功能

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- 🚧 **v1.5 MAC地址历史数据管理** — Phases 12-15 (in progress)
- 📋 **v1.6 API密钥管理系统** — Phase 16 (planned)

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
<summary>🚧 v1.5 MAC地址历史数据管理 (Phases 12-15) — IN PROGRESS</summary>

- [x] Phase 12: 数据模型与采集集成 (5/5 plans) — completed 2026-05-11
- [ ] Phase 13: 查询层与轨迹
- [ ] Phase 14: 前端与用户体验
- [ ] Phase 15: 性能优化

</details>

---

<details>
<summary>📋 v1.6 API密钥管理系统 (Phase 16) — PLANNED</summary>

- [ ] Phase 16: API密钥管理 (10 plans) — planned

</details>

---

## Phase Details

### Phase 12: 数据模型与采集集成
**Goal**: 建立历史数据基础设施，最小侵入集成到现有采集流程
**Depends on**: Phase 11 (v1.4 MAC采集优化)
**Requirements**: DATA-01, DATA-02, QUERY-01
**Success Criteria** (what must be TRUE):
  1. 采集周期运行时，MAC地址变更自动记录到历史表（appeared/disappeared/moved/vlan_changed）
  2. 用户可按设备ID和接口名查询历史MAC地址变化，支持时间范围和分页
  3. 历史表使用月度分区存储，自动清理超过保留期的数据（默认90天）
  4. 历史记录包含设备名称快照，设备删除后历史数据仍然可读
  5. 查询性能满足：90天内查询<2秒，数据库增长率<10%总MAC数/采集周期
**Plans**:
- [x] 12-01-PLAN.md — 创建历史数据模型和变更检测（Wave 1）— completed 2026-05-11
- [x] 12-02-PLAN.md — 实现分区管理和自动清理（Wave 2）— completed 2026-05-11
- [x] 12-03-PLAN.md — 创建历史数据查询服务（Wave 3）— completed 2026-05-11
- [ ] 12-04-PLAN.md — 修复MAC历史路由注册位置（Gap closure, Wave 4）— UAT Gap 2 (blocker)
- [ ] 12-05-PLAN.md — 修复清理任务注册时机（Gap closure, Wave 4）— UAT Gap 0 (major)

### Phase 13: 查询层与轨迹
**Goal**: 提供强大的历史查询能力和MAC轨迹追踪功能
**Depends on**: Phase 12
**Requirements**: QUERY-02, QUERY-03, QUERY-04, UI-03
**Success Criteria** (what must be TRUE):
  1. 用户可按MAC地址查询跨设备/端口的移动路径，按时间顺序展示迁移历史
  2. 系统自动计算MAC地址在每个端口的停留时长（last_seen - first_seen）
  3. 查询结果中显示MAC地址厂商信息（基于OUI识别），无法识别显示"Unknown"
  4. 前端使用ECharts时间轴图表可视化MAC移动路径，支持交互式缩放和拖拽
  5. 支持按MAC地址或端口聚合统计连接时长，识别长期占用端口的设备
**Plans**: TBD
**UI hint**: yes

### Phase 14: 前端与用户体验
**Goal**: 完善前端交互，提供直观的历史数据探索体验
**Depends on**: Phase 13
**Requirements**: UI-01, UI-02, UI-04
**Success Criteria** (what must be TRUE):
  1. 用户可在MAC历史查询页面选择设备、端口和时间范围，表格分页显示查询结果
  2. 用户可将查询结果导出为Excel文件，导出包含所有查询列（限制最多30天数据）
  3. 历史事件时间线组件按时间顺序展示所有变更事件，不同事件类型使用不同图标
  4. 用户可点击时间线事件查看详情，变更类型标签使用不同颜色区分
  5. 页面提供快捷时间选项（最近24小时、7天、30天、90天），提升查询效率
**Plans**: TBD
**UI hint**: yes

### Phase 15: 性能优化
**Goal**: 优化大规模数据场景下的性能，确保生产环境稳定性
**Depends on**: Phase 14
**Requirements**: PERF-01, PERF-02, PERF-03, PERF-04
**Success Criteria** (what must be TRUE):
  1. 系统使用复合索引和BRIN索引优化查询，90天查询<2秒，1年查询<5秒
  2. 系统自动为常用查询创建物化视图，每5分钟刷新，查询优先使用物化视图
  3. 热点查询结果缓存在Redis中，缓存键包含查询参数哈希，过期时间5分钟
  4. 用户可查看端口使用热力图，颜色深浅表示使用密度，支持按时间范围筛选
  5. 分区裁剪机制自动跳过无关分区，查询性能不受历史数据总量影响
**Plans**: TBD
**UI hint**: yes

### Phase 16: API 密钥管理
**Goal**: 实现完整的 API 密钥管理系统，支持密钥的 CRUD 操作、自动生成、权限控制、使用日志审计和前端管理页面
**Depends on**: 无（独立功能）
**Requirements**: 独立功能需求
**Success Criteria** (what must be TRUE):
  1. 用户可以创建、查看、编辑、删除 API 密钥
  2. 密钥自动生成（rec_ + 64位hex），仅创建时完整显示一次
  3. 密钥验证支持格式检查、过期验证、IP白名单验证
  4. 速率限制根据作用域动态调整（read: 30/min, write: 100/min, admin: 200/min）
  5. 使用日志异步记录，支持查询和统计分析
  6. 前端管理页面支持密钥脱敏显示、复制、查看日志等操作
**Plans**: 10 plans

Plans:
- [x] 16-01-PLAN.md — 创建数据模型和数据库迁移（Wave 1）— completed 2026-05-19
- [ ] 16-02-PLAN.md — 实现服务层 - 基础 CRUD（Wave 2）
- [ ] 16-02a-PLAN.md — 实现服务层 - 使用日志和统计（Wave 2）
- [ ] 16-03-PLAN.md — 创建处理器和路由（Wave 3）
- [ ] 16-04-PLAN.md — 实现认证中间件和速率限制器（Wave 4）
- [ ] 16-05a-PLAN.md — 创建前端类型定义和 API 客户端（Wave 5）
- [ ] 16-05b-PLAN.md — 创建前端管理页面组件（Wave 5）
- [ ] 16-06-PLAN.md — 后端测试（Wave 6）
- [ ] 16-07-PLAN.md — 前端测试（Wave 7）
- [ ] 16-08-PLAN.md — 集成测试和覆盖率报告（Wave 8）

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
| 12. 数据模型与采集集成 | v1.5 | 3/5 | Gap Closure | - |
| 13. 查询层与轨迹 | v1.5 | 0/5 | Planned | - |
| 14. 前端与用户体验 | v1.5 | 0/5 | Planned | - |
| 15. 性能优化 | v1.5 | 0/5 | Planned | - |
| 16. API 密钥管理 | v1.6 | 1/10 | In Progress | 2026-05-19 |

---

**Total**: 16 phases, 34 plans complete, 2 gap closure plans, 20 plans planned (5 each for phases 13-15), 10 plans planned (phase 16), 66 total
