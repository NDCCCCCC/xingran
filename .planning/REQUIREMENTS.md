# Requirements: v1.2 可配置仪表盘生产级改造

## Overview

**Milestone**: v1.2 可配置仪表盘生产级改造（仿 Zabbix）
**Goal**: 实现可配置仪表盘系统的核心数据获取机制、前端交互、实时刷新，达到生产级可用状态
**Scope**: 仅改进 `/dashboard-system`，保留 `/monitor/dashboard` 固定监控仪表盘不变

---

## WDG-01: Widget 数据获取机制

**Category**: 后端核心
**Priority**: P0 - 阻塞其他功能

### Requirements

- [ ] **WDG-01a**: 实现 `GetWidgetData` 方法
  - 从配置的 API 端点获取数据
  - 支持 GET/POST 方法
  - 支持请求参数和请求体
  - 权限验证：用户必须有该端点的访问权限

- [ ] **WDG-01b**: 实现 `GetBatchWidgetData` 批量获取
  - 并行获取多个 Widget 数据
  - 单个失败不影响其他 Widget
  - 返回每个 Widget 的数据或错误

- [ ] **WDG-01c**: 支持数据转换（transform）
  - 表达式求值（如 `data.items[0].value`）
  - 聚合函数：sum、avg、count、min、max
  - 排序：asc、desc
  - 限制结果数量：limit

- [ ] **WDG-01d**: 支持三种数据源类型
  - **API**: HTTP 请求获取数据
  - **WebSocket**: 实时推送数据
  - **Static**: 静态配置数据

### Success Criteria

1. 用户可以配置 Widget 从任意有权限的 API 获取数据
2. 数据转换配置生效，返回转换后的结果
3. 批量获取时单个失败不影响整体
4. 三种数据源类型均可正常工作

---

## FE-02: 前端未完成功能

**Category**: 前端交互
**Priority**: P0 - 核心用户体验

### Requirements

- [ ] **FE-02a**: 实现 Widget 选择器组件
  - 展示可用 Widget 类型列表
  - 支持搜索和分类筛选
  - 点击后添加到仪表盘
  - 自动分配位置（避免重叠）

- [ ] **FE-02b**: 实现仪表盘设置面板
  - 编辑名称、描述
  - 设置可见范围（私有/部门/全局）
  - 配置刷新间隔
  - 保存后立即生效

- [ ] **FE-02c**: 实现模板预览功能
  - 展示模板的 Widget 布局
  - 显示示例数据
  - 支持从模板创建仪表盘

- [ ] **FE-02d**: 完善 Widget 编辑/删除交互
  - 编辑 Widget 配置（数据源、显示样式）
  - 删除确认对话框
  - 拖拽调整位置和大小

### Success Criteria

1. 用户可以通过选择器添加任意类型 Widget
2. 仪表盘设置修改后正确保存
3. 模板预览展示准确，创建流程顺畅
4. Widget 编辑/删除操作响应正确

---

## RT-03: 实时数据刷新

**Category**: 数据同步
**Priority**: P1 - 生产必备

### Requirements

- [ ] **RT-03a**: 实现 WebSocket 连接管理
  - 建立连接、断开、自动重连
  - 心跳检测
  - 订阅/取消订阅频道

- [ ] **RT-03b**: 实现基于刷新间隔的轮询
  - 每个仪表盘可配置刷新间隔
  - 后台定时刷新数据
  - 页面可见时恢复刷新

- [ ] **RT-03c**: 实现缓存策略
  - 相同请求短时间内复用缓存
  - 缓存失效机制
  - 手动刷新绕过缓存

- [ ] **RT-03d**: 前端实时数据订阅
  - WebSocket 消息处理
  - 数据更新触发 UI 重渲染
  - 断线重连后恢复订阅

### Success Criteria

1. WebSocket 连接稳定，断线能自动重连
2. 轮询按配置间隔正确执行
3. 缓存有效减少重复请求
4. 数据更新实时反映到 UI

---

## UX-04: 用户体验优化

**Category**: 用户体验
**Priority**: P2 - 锦上添花

### Requirements

- [ ] **UX-04a**: Widget 拖拽布局优化
  - 网格对齐
  - 碰撞检测和自动避让
  - 拖拽预览

- [ ] **UX-04b**: 仪表盘加载状态
  - 初始加载骨架屏
  - Widget 加载中状态
  - 错误重试机制

- [ ] **UX-04c**: 数据为空友好提示
  - 无数据时显示引导
  - 配置错误提示
  - 权限不足提示

- [ ] **UX-04d**: 响应式布局适配
  - 桌面端多列布局
  - 平板端自适应
  - 移动端单列布局

### Success Criteria

1. 拖拽流畅，布局自动调整
2. 加载状态清晰，错误可恢复
3. 空状态有明确引导
4. 各设备尺寸显示正常

---

## Out of Scope

- 仪表盘系统统一 — 保留两套系统
- 复杂图表类型（3D、地图）
- 告警规则配置
- Zabbix 告警集成
- 数据导出功能

---

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| WDG-01a | - | pending |
| WDG-01b | - | pending |
| WDG-01c | - | pending |
| WDG-01d | - | pending |
| FE-02a | - | pending |
| FE-02b | - | pending |
| FE-02c | - | pending |
| FE-02d | - | pending |
| RT-03a | - | pending |
| RT-03b | - | pending |
| RT-03c | - | pending |
| RT-03d | - | pending |
| UX-04a | - | pending |
| UX-04b | - | pending |
| UX-04c | - | pending |
| UX-04d | - | pending |

---
*Created: 2026-04-20*
