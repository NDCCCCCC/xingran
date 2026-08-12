# Quick Task: 移除自动映射功能，在OU页面添加用户组管理

## 目标
1. 移除域用户组页面的"批量自动映射"功能
2. 清理相关的自动映射后端代码
3. 在组织单元(OU)页面添加用户组管理功能，类似关联部门信息

## 待办事项

### 前端修改
- [ ] 从域用户组页面移除"批量自动映射"按钮
- [ ] 移除相关API调用（autoMapAllDepartments）
- [ ] 从adDomainApi.ts移除autoMapAllDepartments函数
- [ ] 在OU页面添加"关联用户组"功能
  - 参考关联部门信息的UI模式
  - 显示当前OU关联的用户组列表
  - 支持添加/移除关联关系

### 后端修改
- [ ] 移除AutoMapAllDepartments相关的路由和handler
- [ ] 从dept_group_mapping_service.go移除AutoMapAllDepartments函数
- [ ] 保留CreateMapping、UpdateMapping、DeleteMapping等手动映射功能
- [ ] 添加根据OU查询关联用户组的API

### 数据库
- [ ] 保留sys_dept_group_mapping表（手动映射使用）
- [ ] 可能需要添加OU到用户组的直接关联表

## 清理的文件/代码
- internal/services/addomain/dept_group_mapping_service.go中的AutoMapAllDepartments函数
- internal/api/v1/system/ad_domain_router.go中的automap路由
- xingran-react-frontend/src/pages/ad-domain/groups/index.tsx中的自动映射按钮
- xingran-react-frontend/src/lib/adDomainApi.ts中的autoMapAllDepartments函数
