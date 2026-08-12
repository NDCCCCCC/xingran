# Phase 14: 前端与用户体验 - Context DRAFT (待续)

**Gathered:** 2026-06-14
**Status:** 草稿 (DRAFT) — 上下文耗尽暂停,等待下一会续接
**Context 来源:** gsd:discuss-phase 14 (已运行 ~65%)

<domain>
## Phase Boundary (从 ROADMAP.md, 已锁定)

基于 Phase 12 (数据模型与采集集成) 与 Phase 13 (查询层与轨迹) 的后端能力,补齐 MAC 地址历史数据管理的完整前端 UX — 包括查询列表页、轨迹可视化页、Excel 导出、菜单与权限注册、与网络设备/工位模块的联动入口,以及移动端响应式适配。

**Scope (REQUIREMENTS.md 锁定)**:
- UI-01: 实现 MAC 历史查询页面
- UI-02: 实现数据导出功能
- UI-04: 实现历史事件时间线组件

**Out of scope (Phase 13 已交付, Phase 14 复用)**:
- UI-03: MAC 轨迹可视化 ECharts Gantt — 已通过 Phase 13-04 交付 `MACTrajectoryChart.tsx`, Phase 14 集成而不重写

**已规划 5 个子 PLAN (从 ROADMAP 详细小节)**:
- 14-01: MAC 历史查询主列表页 (时间筛选、MAC/设备/端口筛选、分页、操作列)
- 14-02: 轨迹可视化页 UX 增强 (ECharts Gantt 交互、停留时长热力、时间范围预设导出)
- 14-03: 菜单/权限/路由注册 (与 sys_menu 同步, 与现有网络设备模块并列)
- 14-04: Excel 导出与批量操作 (后端 export 接口 + 前端 download 触发)
- 14-05: 移动端响应式 + 空状态/错误状态打磨

</domain>

<decisions>
## 已知锁定决策 (从 ROADMAP/REQUIREMENTS/前序 Phase 继承,无需再问)

### D-01: Phase 14 = 纯前端 phase
**来源**: ROADMAP.md "Depends on: Phase 12, Phase 13"
后端 API 全部就位,本阶段不动后端代码 (除 14-04 提到的 export 接口可能是新加)。

### D-02: 复用 Phase 13 已交付资产
**来源**: Phase 13-04 SUMMARY + ROADMAP 详细小节
- `MACTrajectoryChart.tsx` 组件 — 不重写
- `pages/network/mac/trajectory/index.tsx` — 集成而非新建

### D-03: 复用既有前端架构模式
**来源**: Phase 30 CONTEXT + CLAUDE.md
- 状态管理: Zustand (authStore, layoutStore 等)
- 数据获取: React Query 5 (Phase 30 推广后标准)
- UI 库: Ant Design 6
- 列表管理: `useTableManager`, `useColumnConfig` hooks
- API 调用: `lib/api.ts` 包装函数 (非 raw axios)
- 实时更新: `useRealtimeUpdates`, `useWidgetPolling` hooks (按需)

### D-04: 菜单与权限规范
**来源**: 现有 sys_menu 模式 + Phase 27/28 经验
- 权限点格式: `network:mac:list/query/export`
- 与"网络设备"模块并列 (parent menu = network)

### D-05: 移动端策略
**来源**: 既有 `HybridLayout` + Ant Design 6 Grid
- 桌面端: 表格视图
- 移动端: 卡片视图 (参考 operations/buildings 既有模式)
- 响应式断点: Ant Design 6 默认 (xs/sm/md/lg/xl)

</decisions>

<gray_areas_to_discuss>
## 下一会需讨论的 Gray Areas (从 REQUIREMENTS 反推, 待续)

按 GSD discuss-phase 工作流,以下 gray areas 需要在下一会澄清后再写正式 CONTEXT.md:

### GA-01: 列表页查询模式
- 分页 vs 无限滚动 vs 虚拟滚动? (与 Phase 30 资产列表虚拟滚动一致 vs 简单分页)
- 时间范围选择器形式: Ant Design `RangePicker` vs 快捷预设 (1h/24h/7d/30d/自定义)

### GA-02: 时间线组件形态
- UI-04 "历史事件时间线" 是独立组件 (跨页面复用) 还是查询页内嵌子组件?
- 视觉: 垂直时间线 (Ant Design Timeline) vs 水平滚动 vs ECharts Scatter?

### GA-03: 导出格式与触发
- UI-02 "数据导出" 是仅 Excel 还是同时支持 CSV/PDF?
- 导出范围: 仅当前查询条件 vs 提供"全量导出" 选项?
- 触发方式: 列表页工具栏按钮 vs 操作列每行"导出该 MAC"

### GA-04: 与工位/网络设备模块的联动
- 入口位置: 网络设备详情页"查看 MAC 历史"按钮 vs 工位详情页关联入口 vs 仅主菜单?
- URL 跳转时是否携带 query 参数 (deviceId, portName)?

### GA-05: 空数据/错误/加载状态
- 空数据: 引导用户去采集页? 还是仅静态提示?
- 加载状态: Skeleton vs Spin?
- 错误状态: 错误码提示 vs 通用 "查询失败"?

### GA-06: 厂商识别 (UI-04 子能力? 或者是单独需求)
- OUI 库使用: 前端维护映射表 vs 后端接口?
- 缓存策略?

</gray_areas_to_discuss>

<codebase_context>
## Scout 状态: 未完成 (下一会续)

`.planning/codebase/` 存在 7 个 map:
- ARCHITECTURE.md, CONCERNS.md, CONVENTIONS.md, INTEGRATIONS.md, STACK.md, STRUCTURE.md, TESTING.md

按 scout-codebase.md 的 "UI / frontend" phase 类型 → 应读 CONVENTIONS.md, STRUCTURE.md, STACK.md

下一会启动 scout 时直接 read 这三个 map 即可。
</codebase_context>

<canonical_refs>
## 外部参考 (MANDATORY, 下一会验证完整性)

- `.planning/ROADMAP.md` — Phase 14 详细小节 (本会 commit `dbf358b`)
- `.planning/REQUIREMENTS.md` — UI-01/02/04 需求
- `.planning/STATE.md` — Phase 13 已完成, Phase 14 Planned
- `.planning/phases/13-query-layer-trajectory/13-04-SUMMARY.md` — Phase 13 轨迹页交付
- `.planning/phases/30-js/30-CONTEXT.md` — 性能优化经验, 借鉴
- `xingran-react-frontend/src/pages/network/mac/` — Phase 13 已建页面
- `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` — 复用组件
- `docs/开发规范.md` — 前端开发规范 (CLAUDE.md 引用)
</canonical_refs>

<deferred>
## 已记录的 Deferred Ideas (从历史/REQUIREMENTS 收集, 本 phase 不做)

(暂无 — 下一会 discuss 阶段补充)
</deferred>

---

## 下一会续接指南

```bash
# /clear 后, 满上下文重启
/gsd:discuss-phase 14
```

GSD discuss-phase 工作流检测到:
- `has_context: false` (CONTEXT-DRAFT.md 不算正式 CONTEXT.md) → 不会触发 "Update" 提示
- 目录下 `14-CONTEXT-DRAFT.md` 存在, 人工用 Read 工具读取后, 直接进入 "load_prior_context" / "scout_codebase" 步骤

或跳过 discuss 直接 plan:
```bash
/gsd:plan-phase 14 --skip-research   # 不做 researcher
/gsd:plan-phase 14                    # 默认 (含 researcher)
```

---

**草稿状态**: 已写 65% 上下文, scout/gray areas/CONTEXT 模板填充待续。
