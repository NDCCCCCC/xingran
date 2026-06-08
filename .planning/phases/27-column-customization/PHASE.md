---
phase: 27
name: 全局列自定义显示功能
status: planning
created: 2026-06-08
---

# Phase 27: 全局列自定义显示功能

## 概述

为系统中所有列表页面添加可自定义列显示的功能，解决资产列表页面字段过多（43列）的问题。用户可以选择显示哪些列、调整列顺序、设置列宽度，配置保存到数据库后永久生效。

## 业务价值

1. **提升用户体验**: 用户可根据需要隐藏不常用列，突出关键信息
2. **提高工作效率**: 自定义列布局，快速定位重要数据
3. **全局复用**: 一次性开发，所有列表页面受益
4. **持久化存储**: 用户配置永久保存，跨会话保持

## 核心需求

### 功能需求
- [x] 列显示/隐藏控制
- [x] 列拖拽排序
- [x] 列宽度自定义（可选）
- [x] 配置保存到数据库
- [x] 支持重置为默认配置
- [x] localStorage 缓存提升性能

### 非功能需求
- **性能**: 列切换响应时间 < 200ms
- **兼容性**: 支持所有现有列表页面（用户、角色、部门、资产、楼宇等）
- **可扩展性**: 新增列表页面时无需修改核心代码

## 技术范围

### 1. 数据库层
新建 `sys_user_column_config` 表：

```sql
CREATE TABLE sys_user_column_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    page_key VARCHAR(100) NOT NULL,        -- 页面标识 (如 'asset.list', 'user.list')
    column_key VARCHAR(100) NOT NULL,      -- 列标识 (如 'devicesn', 'deviceModelName')
    display_order INTEGER NOT NULL,        -- 显示顺序
    visible BOOLEAN DEFAULT true,          -- 是否可见
    width INTEGER,                         -- 列宽度（像素）
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    UNIQUE(user_id, page_key, column_key)
);

CREATE INDEX idx_user_column_config_user_page ON sys_user_column_config(user_id, page_key);
```

### 2. 后端层
新增 API 接口：

```
GET    /api/v1/system/column-config/:page_key      -- 获取用户列配置
POST   /api/v1/system/column-config                 -- 保存/更新列配置
DELETE /api/v1/system/column-config/:page_key      -- 重置为默认配置
```

### 3. 前端层
- **ColumnSelector 组件**: 列选择器（Checkbox 组）
- **ColumnConfigManager**: 配置管理 Hook
- **localStorage 缓存**: 减少服务器请求
- **集成到 Table 组件**: 自动应用配置

### 4. 默认配置
每个页面需定义默认列配置（JSON 或代码中定义）：

```typescript
const defaultColumnConfig = {
  'asset.list': [
    { key: 'devicesn', visible: true, order: 1, width: 150 },
    { key: 'deviceModelName', visible: true, order: 2, width: 120 },
    // ... 其他列
  ],
  'user.list': [
    { key: 'username', visible: true, order: 1, width: 120 },
    // ...
  ]
}
```

## 约束条件

- **现有页面兼容**: 不能破坏现有列表页面的默认显示
- **权限隔离**: 用户只能管理自己的列配置
- **数据迁移**: 首次使用时自动应用默认配置

## 验收标准

1. ✅ 用户可以显示/隐藏任意列
2. ✅ 用户可以拖拽调整列顺序
3. ✅ 配置保存后刷新页面依然生效
4. ✅ 重置功能恢复默认配置
5. ✅ 支持资产列表（43列）及其他所有列表页面
6. ✅ 切换列时性能流畅（< 200ms）

## 依赖关系

- 依赖现有: `sys_user` 表（用户关联）
- 依赖现有: Ant Design Table 组件
- 无前置阶段依赖
