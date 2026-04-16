# 工位导入部门/用户关联功能

## What This Is

为 RuoYi-Next 运维管理系统的工位（Workstation）Excel 导入功能添加"所属部门"和"所属用户"两个可选字段。导入时通过部门名称匹配 `sys_dept`、通过用户名匹配 `sys_user`，建立工位与部门、用户的关联关系，并在前端展示。

## Core Value

工位导入时能自动关联部门和用户，避免手动逐条配置，提升批量管理效率。

## Requirements

### Validated

<!-- From existing codebase -->

- ✓ 工位 CRUD（创建、读取、更新、删除）— 现有
- ✓ Excel 批量导入/导出工位 — 现有（`internal/services/operations/excel_service.go`）
- ✓ 工位关联 `org_id`（部门 UUID）— 现有字段，导入时通过部门名称匹配
- ✓ 前端工位列表/详情页面 — 现有
- ✓ 部门表 `sys_dept` 含 `dept_name` 字段 — 现有
- ✓ 用户表 `sys_user` 含 `user_name` 字段 — 现有

### Active

- [ ] Excel 导入模板添加"所属部门"和"所属用户"两个可选列
- [ ] 导入时通过部门名称模糊/精确匹配 `sys_dept`，将匹配到的 `dept_id` 写入工位的 `org_id`
- [ ] 导入时通过用户名精确匹配 `sys_user`，将匹配到的 `user_id` 写入工位的新关联字段
- [ ] 工位模型添加用户关联字段（如 `user_id`），数据库迁移
- [ ] 工位列表 API 返回部门名称和用户姓名（join 或关联查询）
- [ ] 前端工位列表展示部门名称和用户姓名列
- [ ] 前端工位详情展示部门名称和用户姓名
- [ ] 匹配失败时（部门名或用户名不存在）：有就关联，没有就留空，不阻断导入

### Out of Scope

- 导入时自动创建不存在的部门或用户 — 超出导入功能职责范围
- 批量修改已有工位的部门/用户关联 — 本期仅关注导入场景
- 工位与用户的多对多关系 — 当前只需一对一关联

## Context

### 技术环境
- **后端**: Go 1.24 + Gin + GORM + PostgreSQL
- **前端**: React 19 + TypeScript + Ant Design 6 + Zustand
- **Excel**: excelize v2 用于导入导出
- **现有导入流程**: `excel_service.go` → `excel_config.go`（列映射）→ `reference_resolver.go`（名称→UUID）→ `batch_upserter.go`（批量写入）

### 关键现有代码
- Excel 导入服务: `internal/services/operations/excel_service.go`
- Excel 列配置: `internal/services/operations/excel_config.go`
- 名称→ID 解析: `internal/services/operations/reference_resolver.go`
- 批量写入: `internal/services/operations/batch_upserter.go`
- 工位模型: `internal/models/` 中的 workstation 相关文件
- 工位 Handler: `internal/api/v1/operations/`
- 前端工位页面: `ruoyi-react-frontend/src/pages/operations/`
- 前端导入组件: `ruoyi-react-frontend/src/components/operations/`
- 前端 opsApi: `ruoyi-react-frontend/src/lib/opsApi.ts`

### 已知模式
- `org_id` 已通过 `reference_resolver.go` 支持部门名称→UUID 匹配，可复用此模式
- 工位导入配置在 `excel_config.go` 中以结构化方式定义列映射

## Constraints

- **Tech Stack**: 必须沿用现有架构模式（Handler-Service、opsApi、excel_config）
- **Backward Compatible**: 新字段可选，不影响现有工位数据和导入流程
- **UUID Foreign Keys**: `org_id` 和 `user_id` 必须是有效 UUID，参考 `building_service.go` 的 `validateOrg()` 模式
- **Status Convention**: 遵循 0=正常, 1=停用 的惯例
- **Response Format**: 使用 `response.Success()` / `response.Error()` 包装

## Current Milestone: v1.1 信息点导入设备/端口关联

**Goal:** 为信息点 Excel 导入添加"所属设备"和"所属端口"两个可选列

**Target features:**
- 导入配置添加"所属设备"列，通过设备名称匹配 `sys_network_device.name` → `device_id`
- 导入配置添加"所属端口"列，通过端口名称匹配 `sys_device_port_status.port_name` → `port_id`
- 两个字段可选，匹配失败留空不阻断导入

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 用户关联字段: 在 workstation 表添加 `user_id` 列 | 与现有 `org_id` 模式一致，一对一关联 | — Pending |
| 用户匹配方式: 通过 `user_name` 精确匹配 | 用户名唯一且稳定，用户要求 | — Pending |
| 部门匹配方式: 通过 `dept_name` 精确匹配 | 与现有 `reference_resolver` 模式一致 | — Pending |
| 匹配失败策略: 留空不阻断 | 用户明确要求可选关联 | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-16 — Milestone v1.1 started*
