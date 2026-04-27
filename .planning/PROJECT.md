# 工位导入部门/用户关联功能

## What This Is

为 RuoYi-Next 运维管理系统的 Excel 导入功能添加可选关联字段。v1.0 实现了工位导入时通过部门/用户名称自动匹配并关联；v1.1 扩展到信息点导入，支持设备名/端口名自动匹配；v1.2 实现可配置仪表盘系统生产级改造（仿 Zabbix）。

## Core Value

Excel 导入时自动关联关联实体（部门、用户、设备、端口），避免手动逐条配置，提升批量管理效率。可配置仪表盘系统提供类似 Zabbix 的监控可视化能力，支持自定义 Widget、实时数据刷新、权限控制。

## Requirements

### Validated

- ✓ 工位 CRUD（创建、读取、更新、删除）— 现有
- ✓ Excel 批量导入/导出工位 — 现有
- ✓ 工位关联 `org_id`（部门 UUID）— 现有字段，导入时通过部门名称匹配
- ✓ 前端工位列表/详情页面 — 现有
- ✓ 部门表 `sys_dept` 含 `dept_name` 字段 — 现有
- ✓ 用户表 `sys_user` 含 `user_name` 字段 — 现有
- ✓ Excel 导入模板添加"所属部门"和"所属用户"两个可选列 — v1.0
- ✓ 导入时通过部门名称/用户名匹配，写入对应 ID — v1.0
- ✓ 匹配失败留空不阻断导入 — v1.0
- ✓ 信息点导入配置添加"所属设备"列 (`deviceName` → `device_id`) — v1.1
- ✓ 信息点导入配置添加"所属端口"列 (`portName` → `port_id`) — v1.1
- ✓ 两个新列均为可选，空值不阻断导入 — v1.1
- ✓ 模板下载包含设备和端口示例值 — v1.1

### Active

## Current Milestone: v1.3 技术债清理

**目标:** 修复已知问题、优化代码结构、完善功能

**目标功能:**
- SNMP panic 修复 — 修复 scrapligo transport 在 SNMP Ping 时的崩溃问题
- 后端代码优化 — 清理死代码、修复安全问题、简化 Core 结构
- 网络设备导出 — 实现网络设备配置导出功能

### Completed

#### v1.2 可配置仪表盘生产级改造（仿 Zabbix）— ✅ Completed 2026-04-21

**WDG-01: Widget 数据获取机制**
- [x] **WDG-01a**: 实现 `GetWidgetData` 方法，支持从配置的 API 端点获取数据
- [x] **WDG-01b**: 实现 `GetBatchWidgetData` 批量获取多个 Widget 数据
- [x] **WDG-01c**: 支持数据转换（transform）配置：表达式求值、聚合、排序、限制
- [x] **WDG-01d**: 支持三种数据源类型：API、WebSocket、Static

**FE-02: 前端未完成功能**
- [x] **FE-02a**: 实现 Widget 选择器组件（添加 Widget 到仪表盘）
- [x] **FE-02b**: 实现仪表盘设置面板（名称、描述、权限、刷新间隔）
- [x] **FE-02c**: 实现模板预览功能
- [x] **FE-02d**: 完善 Widget 编辑/删除交互

**RT-03: 实时数据刷新**
- [x] **RT-03a**: 实现 WebSocket 连接管理（连接、断开、重连）
- [x] **RT-03b**: 实现基于刷新间隔的轮询机制
- [x] **RT-03c**: 实现缓存策略（避免重复请求）
- [x] **RT-03d**: 前端实时数据订阅和更新

**UX-04: 用户体验优化**
- [x] **UX-04a**: Widget 拖拽布局优化（网格对齐、碰撞检测）
- [x] **UX-04b**: 仪表盘加载状态和错误处理
- [x] **UX-04c**: 数据为空时的友好提示
- [x] **UX-04d**: 响应式布局适配（桌面/平板/移动端）

### Out of Scope

- 导入时自动创建不存在的部门或用户 — 超出导入功能职责范围
- 批量修改已有工位的部门/用户关联 — 本期仅关注导入场景
- 工位与用户的多对多关系 — 当前只需一对一关联
- 设备/端口多对多关系 — 当前只需一对一
- 导入时自动创建不存在的设备或端口 — 超出导入功能职责
- 端口与设备的级联验证（确认端口属于指定设备）— 复杂度高，用户未要求
- **仪表盘系统统一** — 保留两套系统：固定监控仪表盘（/monitor/dashboard）和可配置仪表盘（/dashboard-system）
- 复杂图表类型（3D 图表、地图）— 超出本期范围
- 告警规则配置 — 属于监控系统，非仪表盘核心
- Zabbix 告警集成 — 超出本期范围

## Context

### 技术环境
- **后端**: Go 1.24 + Gin + GORM + PostgreSQL
- **前端**: React 19 + TypeScript + Ant Design 6 + Zustand
- **Excel**: excelize v2 用于导入导出
- **现有导入流程**: `excel_service.go` → `excel_config.go`（列映射）→ `reference_resolver.go`（名称→UUID）→ `batch_upserter.go`（批量写入）
- **仪表盘**: 自定义 Widget 系统，支持 StatCard、Chart、Table、List、Progress 类型
- **两套仪表盘**: 
  - `/monitor/dashboard` — 固定监控仪表盘（保留不变）
  - `/dashboard-system` — 可配置仪表盘（本期改进目标）

### 当前状态
- Shipped v1.0 + v1.1, 3 phases, all complete
- 配置驱动模式已验证：添加新关联列只需修改 `excel_config.go` 和 `excel_service.go`
- ReferenceResolver 管道支持任意表+字段匹配
- 可配置仪表盘基础架构已搭建，核心数据获取机制未实现

### 已知技术债
- `batch_upserter_test.go` 测试需要活跃 DB 连接，CI 环境下会 panic
- `dashboard_service.go` GetWidgetData/GetBatchWidgetData 返回未实现错误
- 前端 LayoutToolbar 多个 TODO 未实现

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 用户关联字段: 在 workstation 表添加 `user_id` 列 | 与现有 `org_id` 模式一致，一对一关联 | ✓ Good |
| 用户匹配方式: 通过 `user_name` 精确匹配 | 用户名唯一且稳定 | ✓ Good |
| 部门匹配方式: 通过 `dept_name` 精确匹配 | 与现有 `reference_resolver` 模式一致 | ✓ Good |
| 匹配失败策略: 留空不阻断 | 用户明确要求可选关联 | ✓ Good |
| Device Reference: `sys_network_device.device_name` | 设备名是用户可见的标识字段 | ✓ Good |
| Port Reference: `sys_device_port_status.interface_name` | 接口名是网络设备端口的唯一标识 | ✓ Good |
| Port 无 DependsOn: 全局查找不限设备 | 用户未要求级联验证 | ✓ Good |
| 配置驱动模式: 只改 excel_config.go + excel_service.go | 最小改动，复用现有管道 | ✓ Good |
| 保留两套仪表盘: 固定 + 可配置 | 用户明确要求保留固定监控仪表盘 | ✓ Good |

## Constraints

- **Tech Stack**: 必须沿用现有架构模式（Handler-Service、opsApi、excel_config）
- **Backward Compatible**: 新字段可选，不影响现有数据和导入流程
- **UUID Foreign Keys**: 关联字段必须是有效 UUID
- **Status Convention**: 遵循 0=正常, 1=停用 的惯例
- **Response Format**: 使用 `response.Success()` / `response.Error()` 包装
- **Widget 数据获取**: 必须支持权限过滤，用户只能访问有权限的 API 端点
- **保留固定仪表盘**: `/monitor/dashboard` 保持不变，仅改进 `/dashboard-system`

## Evolution

This document evolves at phase transitions and milestone boundaries.

---
*Last updated: 2026-04-27 starting v1.3 milestone*
