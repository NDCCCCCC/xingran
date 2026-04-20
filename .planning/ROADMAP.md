# Roadmap: 工位导入部门/用户关联功能

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- 🔄 **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (in progress)

## Phases

<details>
<summary>✅ v1.0 工位导入部门/用户关联 (Phases 1-2) — SHIPPED</summary>

- [x] Phase 1: 后端导入配置 (3/3 plans)
- [x] Phase 2: 验证与导出 (4/4 plans)

</details>

<details>
<summary>✅ v1.1 信息点导入设备端口关联 (Phase 3) — SHIPPED 2026-04-16</summary>

- [x] Phase 3: 信息点导入设备端口配置 (1/1 plan)

</details>

---

## v1.2 可配置仪表盘生产级改造

### Phase 4: Widget 数据获取机制

**Goal:** 实现后端 Widget 数据获取核心功能，支持从 API 端点获取数据并进行转换处理

**Depends on:** —

**Requirements:**
- WDG-01a: 实现 GetWidgetData 方法
- WDG-01b: 实现 GetBatchWidgetData 批量获取
- WDG-01c: 数据转换配置支持
- WDG-01d: 三种数据源类型支持

**Success Criteria:**
1. 用户可以配置 Widget 从指定 API 端点获取数据
2. 支持批量获取多个 Widget 数据（性能优化）
3. 数据转换表达式可以正确求值（如 `data.items[0].value`）
4. API、WebSocket、Static 三种数据源均可正常工作
5. 权限过滤生效，用户只能访问有权限的端点

**Plans:**
- [x] 04-01-PLAN.md — 实现 GetWidgetData 核心逻辑，支持 API/Static/WebSocket 三种数据源
- [x] 04-02-PLAN.md — 实现数据转换管道，使用 JSONata 表达式引擎
- [x] 04-03-PLAN.md — 实现批量获取和缓存，使用 errgroup 并行获取

---

### Phase 5: 前端交互完善

**Goal:** 实现仪表盘编辑所需的前端交互组件，让用户可以配置和管理仪表盘

**Depends on:** Phase 4

**Requirements:**
- FE-02a: Widget 选择器组件
- FE-02b: 仪表盘设置面板
- FE-02c: 模板预览功能
- FE-02d: Widget 编辑/删除交互

**Success Criteria:**
1. 用户可以通过 Widget 选择器添加新 Widget 到仪表盘
2. 用户可以修改仪表盘名称、描述、权限范围、刷新间隔
3. 用户可以预览模板效果后再决定是否使用
4. 用户可以编辑 Widget 配置（数据源、显示样式）
5. 用户可以删除 Widget 并确认

**Plans:**
- [ ] 05-01: 实现 Widget 选择器
- [ ] 05-02: 实现仪表盘设置面板
- [ ] 05-03: 实现模板预览和 Widget 编辑

---

### Phase 6: 实时数据刷新

**Goal:** 实现仪表盘数据的实时更新机制，支持 WebSocket 推送和轮询两种模式

**Depends on:** Phase 4, Phase 5

**Requirements:**
- RT-03a: WebSocket 连接管理
- RT-03b: 基于刷新间隔的轮询机制
- RT-03c: 缓存策略
- RT-03d: 前端实时数据订阅

**Success Criteria:**
1. WebSocket 连接可以正常建立、断开、自动重连
2. 用户配置的刷新间隔生效，数据定时更新
3. 相同请求在短时间内不会重复发送（缓存生效）
4. 前端可以订阅数据变化并实时更新 UI
5. 网络断开时有友好提示，恢复后自动重连

**Plans:**
- [ ] 06-01: 实现 WebSocket 连接管理
- [ ] 06-02: 实现轮询和缓存机制
- [ ] 06-03: 前端数据订阅集成

---

### Phase 7: 用户体验优化

**Goal:** 优化仪表盘交互体验，处理边界情况，实现响应式布局

**Depends on:** Phase 5, Phase 6

**Requirements:**
- UX-04a: Widget 拖拽布局优化
- UX-04b: 加载状态和错误处理
- UX-04c: 数据为空时的友好提示
- UX-04d: 响应式布局适配

**Success Criteria:**
1. Widget 拖拽时自动对齐网格，不会重叠
2. 数据加载时显示 loading 状态，错误时显示错误信息
3. Widget 数据为空时显示友好提示而非空白
4. 仪表盘在桌面、平板、移动端均有良好显示效果
5. 所有交互有适当的动画过渡

**Plans:**
- [ ] 07-01: 优化拖拽布局和响应式
- [ ] 07-02: 完善加载状态和错误处理

---

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. 后端导入配置 | v1.0 | 3/3 | Complete | — |
| 2. 验证与导出 | v1.0 | 4/4 | Complete | — |
| 3. 信息点导入设备端口配置 | v1.1 | 1/1 | Complete | 2026-04-16 |
| 4. Widget 数据获取机制 | v1.2 | 3/3 | Complete | 2026-04-20 |
| 5. 前端交互完善 | v1.2 | 0/3 | Not Started | — |
| 6. 实时数据刷新 | v1.2 | 0/3 | Not Started | — |
| 7. 用户体验优化 | v1.2 | 0/2 | Not Started | — |
