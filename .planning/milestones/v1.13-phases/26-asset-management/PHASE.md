# Phase 26: 资产管理模块

**Phase Number**: 26
**Status**: ✅ Completed
**Milestone**: v1.13

## Goal

创建完整的资产管理模块，包含数据库设计、后端 API 和前端页面，支持基于设备序列号 (DEVICESN) 的 Excel 导入/更新功能。

## Requirements

### 核心功能
- **数据模型**: 根据元数据定义创建包含40个字段的资产表结构
- **导入功能**: Excel 批量导入，根据设备序列号判断更新或新增
- **导出功能**: 支持导出当前资产数据到 Excel
- **CRUD 操作**: 完整的增删改查 API
- **前端页面**: 新的"资产管理"菜单下的列表页面

### 技术规范
- 遵循项目 Handler-Service 架构模式
- 使用 UUID 作为主键
- 支持软删除 (deleted_at)
- 遵循 0=正常, 1=停用的状态值规范
- 参考 building/floor/workstation 模式

## Depends On

- 无（独立新模块）

## Plans

### Wave 1 - Database & Model (1 plan)
- [26-01-PLAN.md](./26-01-PLAN.md) — Asset model and database schema

### Wave 2 - Backend Service & API (2 plans)
- [26-02-PLAN.md](./26-02-PLAN.md) — Asset service layer with UUID validation ✅
- [26-03-PLAN.md](./26-03-PLAN.md) — Asset API handlers and routes ✅

### Wave 3 - Excel Import/Export & Frontend (3 plans)
- [26-04-PLAN.md](./26-04-PLAN.md) — Excel import/export configuration ✅
- [26-05-PLAN.md](./26-05-PLAN.md) — Frontend asset list page ✅
- [26-06-PLAN.md](./26-06-PLAN.md) — Menu and permission configuration ✅

## Execution Order

```
Wave 1: Model + Database
  └─> 26-01 (Asset Model + Migration)

Wave 2: Backend Implementation (depends on 26-01)
  └─> 26-02 (Service Layer)
  └─> 26-03 (API Handlers + Routes)

Wave 3: Frontend & Configuration (depends on 26-03)
  └─> 26-04 (Excel Config)
  └─> 26-05 (Frontend Page)
  └─> 26-06 (Menu + Permissions)
```

## Phase Highlights

**6 plans in 3 waves**:

- Wave 1: Database schema with 40 fields, indexes, and constraints
- Wave 2: Complete backend CRUD with UUID validation and department/user resolution
- Wave 3: Excel import/export with DeviceSN as upsert key, frontend list page, and menu integration

**Key Features**:

- DeviceSN as unique identifier for Excel import/update
- Automatic department/user resolution via ReferenceResolver
- UUID validation for foreign keys (dept_id, user_id)
- Department filtering with sub-department support
- Excel import with PartialUpdate for incremental updates
- Comprehensive asset tracking with 40 fields

## Files

- Phase directory: `.planning/phases/26-asset-management/`
- Plans: 6 PLAN.md files (26-01 through 26-06)
- ROADMAP.md: Phase 26 entry to be updated
