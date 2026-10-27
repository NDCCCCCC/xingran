# Quick Task: 创建OU-组直接关联功能，删除部门-组映射

## 目标
1. 删除所有部门-组映射功能（sys_dept_group_mapping 相关）
2. 创建OU-组直接关联功能
3. 在OU页面添加"关联用户组"管理UI

## 待办事项

### 后端 - 删除部门-组映射
- [x] 删除 `internal/models/dept_group_mapping.go` 模型
- [x] 删除 `internal/services/addomain/dept_group_mapping_service.go` 服务
- [x] 删除 `internal/api/v1/system/ad_group_mapping_router.go` 路由
- [x] 从 `internal/api/v1/system/ad_domain_router.go` 移除映射相关路由
- [x] 从前端删除部门-组映射页面 `src/pages/ad-domain/group-mapping/`

### 后端 - 创建OU-组映射
- [x] 创建 `OUGroupMapping` 模型
- [x] 创建 OU-组映射服务 `ou_group_mapping_service.go`
- [x] 创建 API handler 和路由
- [x] 实现以下API：
  - POST `/ou-group-mappings` - 创建关联
  - GET `/ou-group-mappings` - 查询列表
  - GET `/ou-group-mappings/:id` - 获取详情
  - PUT `/ou-group-mappings/:id` - 更新
  - DELETE `/ou-group-mappings/:id` - 删除
  - GET `/ou/:ouDn/group-mappings` - 获取OU的所有关联

### 前端 - OU页面UI改进
- [x] 删除"关联部门信息"卡片和相关功能
- [x] 添加"关联用户组"卡片，显示：
  - 已关联的用户组列表
  - 同步状态（OU成员同步部门组成员）
  - "修改关联"按钮
- [x] 创建用户组选择模态框
- [x] 添加前端API函数

### 数据库迁移
- [x] 创建 ou_group_mapping 表
- [ ] 可选：删除 sys_dept_group_mapping 表（或保留用于历史数据）

## UI 设计参考

### 关联用户组卡片
```
┌─────────────────────────────────────────┐
│ 关联用户组                   [修改关联]  │
├─────────────────────────────────────────┤
│ 已关联组：                                │
│   • CXHUB-人力资源部                     │
│   • CXHUB-财务部                         │
│                                          │
│ 同步状态：                                │
│   OU成员 → 部门 → 用户组               │
│   状态：[已启用/未启用]                   │
│   最后同步：2025-05-27 10:00            │
└─────────────────────────────────────────┘
```
