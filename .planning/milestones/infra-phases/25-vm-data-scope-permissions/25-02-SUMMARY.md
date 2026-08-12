# Phase 25-02 Summary: 后端数据范围过滤函数

## 完成状态
✅ 完成

## 变更内容

### 文件创建
- `internal/services/vdi/vm_data_scope_filter.go` - 虚拟机数据范围过滤函数

### 文件修改
- `internal/services/vdi/vm_service_impl.go` - 在 ListVMs 方法中集成数据范围过滤

## 实现细节

### Task 1: 创建数据范围过滤函数

**文件**: `vm_data_scope_filter.go`

**导出函数**:
1. `ApplyVMDataScopeFilter(query, userID, dataScope, db)` - 应用5层数据范围过滤
   - DataScopeAll: 不做过滤（全部数据）
   - DataScopeCustom: 通过 sys_role_dept 表过滤自定义部门
   - DataScopeDept: 仅本部门数据
   - DataScopeDeptChild: 本部门及子部门数据
   - DataScopeSelf: 仅本人数据

2. `ApplyBoundUserFilter(query, dataScope)` - 无绑定用户过滤规则
   - 非 DataScopeAll 的查询自动过滤掉 bound_user_id IS NULL 的虚拟机

3. `getChildDepts(db, parentId, deptIds)` - 递归查询所有子部门

**关键设计**:
- 使用 `bound_user_id` 字段过滤（而非 dept_id）
- 基于 sys_user 表的 dept_id 进行子查询过滤
- 错误处理：记录日志并返回空结果（WHERE 1=0）
- 避免 import cycle：使用 *gorm.DB 参数而非 *core.Core

### Task 2: 集成到 Service 层

**文件**: `vm_service_impl.go`

**集成位置**: ListVMs 方法，WHERE 子句之后、query.Count 之前

**实现逻辑**:
```go
// 从上下文获取用户信息（由 DataScopePermission 中间件设置）
userIDVal := ctx.Value("user_id")
dataScopeVal := ctx.Value("data_scope")

// 类型断言并应用过滤
if userIDVal != nil && dataScopeVal != nil {
    userID, ok1 := userIDVal.(string)
    dataScope, ok2 := dataScopeVal.(models.DataScope)
    
    if ok1 && ok2 {
        query = ApplyVMDataScopeFilter(query, userID, dataScope, s.db)
        query = ApplyBoundUserFilter(query, dataScope)
    }
}
```

**向下兼容**:
- 如果 context 中没有 user_id 或 data_scope，跳过过滤（兼容直接 API 调用）
- 如果类型断言失败，跳过过滤

## 验证结果
- ✅ 数据范围过滤函数文件创建成功
- ✅ 所有5种数据范围类型实现正确
- ✅ NULL 值过滤逻辑完整
- ✅ Service 层集成成功
- ✅ 避免 import cycle
- ✅ 代码编译通过

## 下一步依赖
- 25-03-PLAN.md: 路由权限中间件配置（设置 context 值）
