---
phase: quick
slug: enhance-ou-management-with-dept-mapping
created: 2026-05-25
status: planning
---

## Objective

增强现有 OU 管理页面，添加部门关联功能，避免重复开发。在现有只读 OU 管理页面基础上，添加可编辑的部门关联面板，实现 XingRan 部门与 AD OU 的双向映射管理。

## Scope

**In Scope:**
- 后端 API：获取/更新 OU-部门映射关系
- 前端 OU 管理页面增强：添加部门关联面板
- 部门选择器组件（树形结构）
- 映射状态显示（synced/pending/failed）
- 集成到现有 OU 管理页面

**Out of Scope:**
- 部门管理页面增强（阶段2）
- 实时同步状态推送
- 批量关联功能
- 同步冲突解决

## Implementation Plan

### Task 1: 后端 API 端点

**文件:** `internal/api/v1/addomain/ou_mapping_handler.go` (新建)
**文件:** `internal/api/v1/addomain/ou_mapping_router.go` (新建)

创建 OU-部门映射的 API 端点：

```go
// GET /api/v1/ad/ou/:ouDn/dept-mapping - 获取 OU 关联的部门
// POST /api/v1/ad/ou/:ouDn/dept-mapping - 更新 OU 部门关联
// 参数: { "deptId": "uuid" }
```

### Task 2: 前端 API 集成

**文件:** `xingran-react-frontend/src/lib/adDomainApi.ts`

添加新的 API 函数：

```typescript
// 获取 OU 关联的部门信息
export function getOUDeptMapping(ouDn: string): Promise<BaseResponse<DeptMapping>>

// 更新 OU 部门关联
export function updateOUDeptMapping(ouDn: string, deptId: string): Promise<BaseResponse<void>>
```

### Task 3: 部门选择器组件

**文件:** `xingran-react-frontend/src/components/system/DeptSelector.tsx` (新建)

创建部门树选择器组件，支持：
- 树形结构显示
- 搜索功能
- 单选模式
- 异步加载子节点

### Task 4: OU 管理页面增强

**文件:** `xingran-react-frontend/src/pages/ad-domain/ous/index.tsx`

在现有 OU 用户列表下方添加部门关联面板：

```tsx
<Card title="关联部门信息" style={{ marginTop: 16 }}>
  {/* 显示当前映射 */}
  {/* 部门选择器 */}
  {/* 更新按钮 */}
  {/* 同步状态显示 */}
</Card>
```

## Key Dependencies

- 现有 OU 管理页面：`src/pages/ad-domain/ous/index.tsx`
- 现有 OU 服务：`internal/services/addomain/ou.go`
- 现有部门 OU 映射器：`internal/services/addomain/dept_ou_mapper.go`
- 现有部门 API：`src/lib/api.ts` (部门树相关)

## Success Criteria

1. ✅ 后端 API 能正确查询和更新 `sys_dept_ou_mapping` 表
2. ✅ OU 管理页面显示当前关联的部门信息
3. ✅ 用户可以通过部门选择器修改关联关系
4. ✅ 更新后实时反映在界面上
5. ✅ 代码编译通过，无新增 lint 错误

## Risk Mitigation

- **现有页面兼容性:** 在现有页面添加组件，不破坏现有功能
- **API 性能:** 部门树数据使用缓存，避免重复查询
- **数据一致性:** 使用事务确保映射关系的完整性
