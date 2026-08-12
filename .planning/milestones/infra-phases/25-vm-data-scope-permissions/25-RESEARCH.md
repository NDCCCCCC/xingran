# Phase 25: 虚拟机数据范围权限配置 - Research

**Researched:** 2026-06-02
**Domain:** VDI 虚拟机细粒度权限控制
**Confidence:** HIGH

## Summary

Phase 25 为 Phase 22 实现的 VDI 虚拟机管理系统配置细粒度数据范围权限。研究现有代码发现项目已具备完整的权限基础设施：RBAC 权限模型（`sys_menu` 表）、数据范围枚举（`DataScope`）、权限中间件（`DataScopePermission`、`RequirePermissions`）、以及基于 `bound_user_id` 的虚拟机绑定用户字段。核心工作是在现有框架上实现虚拟机专属的数据范围过滤逻辑和细粒度操作权限。

**Primary recommendation:** 复用现有权限中间件和数据范围过滤机制，为 VDI 模块创建独立的数据范围过滤函数，通过菜单迁移脚本添加细粒度操作权限，前端基于 `authStore.permissions` 动态生成操作按钮。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 细粒度权限验证 | API / Backend | Frontend | 路由层中间件验证操作权限，Service 层过滤数据范围 |
| 数据范围过滤 | API / Backend | — | Service 层根据用户角色和部门过滤虚拟机数据 |
| 按钮级权限控制 | Browser / Client | — | 前端根据用户权限数组动态显示/隐藏操作按钮 |
| 菜单权限管理 | Database / Storage | — | 通过 `sys_menu` 表管理权限定义和角色关联 |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Gin Router | 1.10.0 | HTTP 路由和中间件 | 项目现有框架，提供 `DataScopePermission` 和 `RequirePermissions` 中间件 |
| GORM | 1.30.5 | ORM 数据访问 | 项目现有 ORM，支持 `ApplyDataScope` 查询过滤 |
| DataScope Enum | — | 数据范围类型定义 | 现有枚举（1=全部, 2=自定义, 3=本部门, 4=本部门及子部门, 5=仅本人） |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| React | 19.2 | 前端 UI 框架 | 动态生成操作按钮，基于权限显示/隐藏 |
| Zustand | 5.0 | 状态管理 | `authStore.permissions` 存储用户权限列表 |
| Ant Design | 6.1 | UI 组件库 | Button、Icon 组件实现操作按钮 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 复用 `ApplyDataScope` | 自定义数据范围过滤函数 | 需要处理 `bound_user_id` 特殊逻辑（NULL 值、部门关联） |

**Installation:**
无需安装新依赖，使用项目现有技术栈。

**Version verification:** 所有库已在项目中验证使用。

## Architecture Patterns

### System Architecture Diagram

```
用户请求虚拟机列表
    ↓
认证中间件（JWT Token 验证）
    ↓
DataScopePermission 中间件
    ├─ 从上下文获取 user_id
    ├─ 查询用户最大数据范围（MAX(r.data_scope)）
    └─ 存储到上下文：c.Set("data_scope", dataScope)
    ↓
VM Handler.List()
    ↓
VM Service.ListVMs()
    ├─ 从上下文获取 data_scope 和 user_id
    ├─ 调用 ApplyVMDataScopeFilter()
    │   ├─ DataScopeAll: 无过滤
    │   ├─ DataScopeCustom: 通过 sys_role_dept 过滤
    │   ├─ DataScopeDept: 通过用户部门过滤
    │   ├─ DataScopeDeptChild: 通过部门树过滤
    │   ├─ DataScopeSelf: bound_user_id = user_id
    │   └─ ApplyBoundUserFilter(): 非 DataScopeAll 过滤 NULL 值
    └─ 返回过滤后的虚拟机列表
    ↓
响应包装（response.Success）
    ↓
前端接收数据
    ↓
动态生成操作按钮
    ├─ 遍历 vmOperationButtons 配置
    ├─ 检查 authStore.permissions.includes(btn.permission)
    └─ 有权限则显示按钮，无权限则隐藏
```

### Recommended Project Structure
```
internal/
├── api/v1/vdi/
│   ├── vm_router.go          # 添加 DataScopePermission 和 RequirePermissions 中间件
│   └── vm_handler.go         # 保持现有逻辑，无需修改
├── services/vdi/
│   ├── vm_service_impl.go     # 调用 ApplyVMDataScopeFilter
│   └── vm_data_scope_filter.go  # 新增：虚拟机数据范围过滤函数
└── core/db/migrations/
    └── 131_add_vdi_granular_permissions.sql  # 新增：细粒度权限菜单

xingran-react-frontend/src/
├── pages/vdi/VirtualMachineList/
│   ├── index.tsx             # 添加 renderOperationButtons 函数
│   └── vmOperationButtons.ts  # 新增：权限-按钮映射配置
└── utils/
    └── permissionHelpers.ts   # 新增：usePermission Hook（如果需要）
```

### Pattern 1: 路由层权限中间件配置
**What:** 在路由组添加 `DataScopePermission` 中间件，为单个操作添加 `RequirePermissions` 中间件
**When to use:** 需要验证用户操作权限和数据范围访问权限时
**Example:**
```go
// Source: internal/api/router.go
vms := authorized.Group("/vms")
vms.Use(middleware.DataScopePermission(core))
{
    vdiV1.SetupVMRouter(vms, core)
}

// Source: internal/api/v1/vdi/vm_router.go
r.POST("/operate", middleware.RequirePermissions([]string{"vdi:vm:start"}, core), vmHandler.Operate)
r.POST("/:id/sync", middleware.RequirePermissions([]string{"vdi:vm:sync"}, core), vmHandler.SyncFromVDI)
```

### Pattern 2: 数据范围过滤函数
**What:** 独立的数据范围过滤函数，处理虚拟机 `bound_user_id` 字段的特殊逻辑
**When to use:** Service 层需要根据用户角色和部门过滤虚拟机数据时
**Example:**
```go
// Source: internal/services/vdi/vm_data_scope_filter.go
func ApplyVMDataScopeFilter(query *gorm.DB, userID string, dataScope models.DataScope, core *core.Core) *gorm.DB {
    // 先应用基础数据范围过滤
    query = applyBaseDataScope(query, userID, dataScope, core)
    
    // 再应用无绑定用户特殊规则
    return ApplyBoundUserFilter(query, dataScope)
}

func ApplyBoundUserFilter(query *gorm.DB, dataScope models.DataScope) *gorm.DB {
    if dataScope != models.DataScopeAll {
        return query.Where("bound_user_id IS NOT NULL")
    }
    return query
}
```

### Anti-Patterns to Avoid
- **硬编码权限检查**: 不要在 Handler 中直接检查权限，应使用中间件
- **SQL 拼接**: 不要使用字符串拼接构建 SQL，使用 GORM 链式调用防止注入
- **忽略 NULL 值**: 不要忘记处理 `bound_user_id IS NULL` 的特殊情况
- **前端权限隐藏即安全**: 不要仅在前端隐藏按钮，后端必须验证权限

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 权限中间件 | 自定义认证和权限验证逻辑 | `middleware.DataScopePermission` 和 `middleware.RequirePermissions` | 现有中间件已处理 JWT 验证、超级管理员判断、权限查询 |
| 数据范围过滤 | 手动编写复杂的 JOIN 查询 | 复用 `ApplyDataScope` 模式，扩展 `bound_user_id` 逻辑 | 现有模式已处理部门树、自定义部门等复杂场景 |
| 前端权限检查 | 手动在组件中判断权限 | 基于 `authStore.permissions` 动态生成按钮 | 权限集中管理，避免散落的判断逻辑 |

**Key insight:** 虚拟机权限系统的核心是数据范围过滤逻辑的特殊性（基于 `bound_user_id` 而非 `dept_id`），但权限验证、菜单管理、前端按钮控制等机制已完整存在，无需重新实现。

## Common Pitfalls

### Pitfall 1: 忽略 `bound_user_id IS NULL` 的特殊处理
**What goes wrong:** 无绑定用户的虚拟机对所有角色可见，违反"仅 DataScopeAll 可见"的规则
**Why it happens:** 直接复用 `ApplyDataScope` 函数，未添加 NULL 值过滤逻辑
**How to avoid:** 在数据范围过滤后，额外调用 `ApplyBoundUserFilter()` 处理 NULL 值
**Warning signs:** 测试时发现非管理员角色能看到未绑定用户的虚拟机

### Pitfall 2: 细粒度权限路由配置错误
**What goes wrong:** 用户拥有 `vdi:vm:operate` 通用权限，但无法执行具体操作（启动/关机等）
**Why it happens:** 迁移脚本删除了通用权限但路由未更新为细粒度权限
**How to avoid:** 确保路由配置与菜单权限一致，测试每个操作的权限验证
**Warning signs:** 权限不足错误 403 Forbidden

### Pitfall 3: 前端权限按钮映射遗漏
**What goes wrong:** 前端显示所有操作按钮，点击后才提示权限不足
**Why it happens:** `vmOperationButtons` 配置不完整或未正确检查权限
**How to avoid:** 完整定义所有 5 个操作权限的按钮配置，测试每个权限的按钮显示
**Warning signs:** UI 显示的操作按钮与后端权限不匹配

### Pitfall 4: 部门数据范围过滤 SQL 错误
**What goes wrong:** 部门数据范围查询返回空结果或错误
**Why it happens:** `sys_user` 和 `sys_dept` 表关联错误，或未考虑部门树结构
**How to avoid:** 复用现有 `ApplyDataScope` 函数的部门过滤逻辑，通过子查询验证 `bound_user_id` 所属部门
**Warning signs:** 部门角色无法看到应该可见的虚拟机

## Code Examples

Verified patterns from official sources:

### 细粒度权限路由配置
```go
// Source: pkg/middleware/permission.go + internal/api/router.go
// 在路由组添加数据权限中间件
vms := authorized.Group("/vms")
vms.Use(middleware.DataScopePermission(core))
{
    vdiV1.SetupVMRouter(vms, core)
}

// 在单个路由添加细粒度权限验证
r.POST("/operate", middleware.RequirePermissions([]string{"vdi:vm:start"}, core), vmHandler.Operate)
r.POST("/:id/sync", middleware.RequirePermissions([]string{"vdi:vm:sync"}, core), vmHandler.SyncFromVDI)
```

### 虚拟机数据范围过滤函数
```go
// Source: internal/services/vdi/vm_data_scope_filter.go (新增)
package vdi

import (
    "github.com/xingran-next/xingran-go-backend/internal/core"
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "gorm.io/gorm"
)

// ApplyVMDataScopeFilter 应用虚拟机数据范围过滤
func ApplyVMDataScopeFilter(query *gorm.DB, userID string, dataScope models.DataScope, core *core.Core) *gorm.DB {
    switch dataScope {
    case models.DataScopeAll:
        // 全部数据：不添加过滤条件
        return query
    case models.DataScopeCustom:
        // 自定义数据：通过 sys_role_dept 表过滤 bound_user_id
        return query.Where("bound_user_id IN (SELECT u.id FROM sys_user u INNER JOIN sys_user_role ur ON u.id = ur.user_id INNER JOIN sys_role_dept rd ON ur.role_id = rd.role_id WHERE ur.user_id = ?)", userID)
    case models.DataScopeDept:
        // 本部门数据：通过用户部门过滤
        return query.Where("bound_user_id IN (SELECT id FROM sys_user WHERE dept_id = (SELECT dept_id FROM sys_user WHERE id = ?))", userID)
    case models.DataScopeDeptChild:
        // 本部门及子部门数据：通过部门树过滤
        return query.Where("bound_user_id IN (SELECT id FROM sys_user WHERE dept_id IN (SELECT id FROM sys_dept WHERE FIND_IN_SET((SELECT dept_id FROM sys_user WHERE id = ?), ancestors)))", userID)
    case models.DataScopeSelf:
        // 仅本人数据
        return query.Where("bound_user_id = ?", userID)
    default:
        return query
    }
}

// ApplyBoundUserFilter 应用无绑定用户过滤规则
func ApplyBoundUserFilter(query *gorm.DB, dataScope models.DataScope) *gorm.DB {
    if dataScope != models.DataScopeAll {
        // 非 DataScopeAll 的查询自动过滤掉无绑定用户的虚拟机
        return query.Where("bound_user_id IS NOT NULL")
    }
    return query
}
```

### 前端权限-按钮映射配置
```typescript
// Source: xingran-react-frontend/src/pages/vdi/VirtualMachineList/vmOperationButtons.ts (新增)
import { PlayCircleOutlined, StopOutlined, ReloadOutlined, SyncOutlined, DeleteOutlined } from '@ant-design/icons';

export interface VMOprationButton {
  action: 'start' | 'stop' | 'restart' | 'sync' | 'delete';
  permission: string;
  label: string;
  icon: React.ReactNode;
}

export const vmOperationButtons: VMOprationButton[] = [
  { action: 'start', permission: 'vdi:vm:start', label: '启动', icon: <PlayCircleOutlined /> },
  { action: 'stop', permission: 'vdi:vm:stop', label: '关机', icon: <StopOutlined /> },
  { action: 'restart', permission: 'vdi:vm:restart', label: '重启', icon: <ReloadOutlined /> },
  { action: 'sync', permission: 'vdi:vm:sync', label: '同步', icon: <SyncOutlined /> },
  { action: 'delete', permission: 'vdi:vm:delete', label: '删除', icon: <DeleteOutlined /> },
];
```

### 前端动态生成操作按钮
```typescript
// Source: xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx (修改)
import { useAuthStore } from '@/store/authStore';
import { vmOperationButtons } from './vmOperationButtons';

const VirtualMachineList: React.FC = () => {
  const { user } = useAuthStore();
  const permissions = user?.permissions || [];

  // 动态生成操作按钮
  const renderOperationButtons = (vm: VirtualMachine) => {
    return vmOperationButtons
      .filter(btn => permissions.includes(btn.permission))
      .map(btn => (
        <Button
          key={btn.action}
          icon={btn.icon}
          onClick={() => handleOperation(btn.action, vm)}
        >
          {btn.label}
        </Button>
      ));
  };

  // 在表格列中使用
  const columns: ColumnsType<VirtualMachine> = [
    // ... 其他列
    {
      title: '操作',
      key: 'action',
      render: (_, vm) => <Space>{renderOperationButtons(vm)}</Space>,
    },
  ];
};
```

### 菜单迁移脚本（添加细粒度权限）
```sql
-- Source: internal/core/db/migrations/131_add_vdi_granular_permissions.sql (新增)

-- 1. 删除通用操作权限 vdi:vm:operate
DELETE FROM sys_menu WHERE perms = 'vdi:vm:operate';

-- 2. 添加细粒度操作权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440020', '启动虚拟机', '770e8400-e29b-41d4-a716-446655440002', 10, '', '', 'F', '1', '0', 'vdi:vm:start', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440021', '关机虚拟机', '770e8400-e29b-41d4-a716-446655440002', 11, '', '', 'F', '1', '0', 'vdi:vm:stop', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440022', '重启虚拟机', '770e8400-e29b-41d4-a716-446655440002', 12, '', '', 'F', '1', '0', 'vdi:vm:restart', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440023', '删除虚拟机', '770e8400-e29b-41d4-a716-446655440002', 13, '', '', 'F', '1', '0', 'vdi:vm:delete', '#', '', NOW(), NOW(), NULL);

-- 注意：vdi:vm:sync 已存在（770e8400-e29b-41d4-a716-446655440011），无需新增
```

### Service 层调用数据范围过滤
```go
// Source: internal/services/vdi/vm_service_impl.go (修改)
func (s *vmServiceImpl) ListVMs(ctx context.Context, req *ListVMsRequest) (*ListVMsResponse, error) {
    // ... 现有代码

    // 从上下文获取用户信息
    userID := ctx.Value("user_id").(string)
    dataScope := ctx.Value("data_scope").(models.DataScope)

    // 构建查询
    query := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{})

    // 应用数据范围过滤
    query = ApplyVMDataScopeFilter(query, userID, dataScope, s.core)

    // ... 继续现有查询逻辑
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 通用操作权限 `vdi:vm:operate` | 细粒度权限 `vdi:vm:start/stop/restart/sync/delete` | Phase 25 | 权限控制更精细，可单独分配操作权限 |
| 无数据范围过滤 | 基于 `bound_user_id` 的数据范围过滤 | Phase 25 | 虚拟机数据权限与部门和用户绑定关系关联 |
| 前端硬编码按钮显示 | 基于权限动态生成按钮 | Phase 25 | 权限变更自动反映到 UI，减少维护成本 |

**Deprecated/outdated:**
- `vdi:vm:operate` 通用权限：将被细粒度权限替代，需要删除并迁移角色权限配置

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `sys_menu` 表使用 `perms` 字段存储权限标识符 | Standard Stack | 如果字段名不同，菜单迁移脚本需要调整 |
| A2 | `authStore.permissions` 存储用户权限字符串数组 | Architecture Patterns | 如果存储格式不同，前端权限检查需要调整 |
| A3 | 现有 `ApplyDataScope` 函数适用于部门字段 | Common Pitfalls | 如果不适用，需要重新实现部门过滤逻辑 |

## Open Questions

1. **角色权限迁移策略**
   - What we know: 现有角色可能已分配 `vdi:vm:operate` 通用权限
   - What's unclear: 如何迁移现有角色权限到细粒度权限（默认授予全部 5 个权限？还是由管理员手动配置？）
   - Recommendation: 在迁移脚本中为拥有 `vdi:vm:operate` 的角色自动添加全部 5 个细粒度权限，避免权限丢失

2. **前端权限缓存策略**
   - What we know: `authStore` 使用 Zustand persist 中间件持久化权限
   - What's unclear: 权限变更后（管理员修改角色权限）前端是否需要刷新或重新登录？
   - Recommendation: 现有系统未实现权限实时刷新，需用户重新登录生效

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| PostgreSQL | 数据库迁移和权限查询 | ✓ | 18.1 | — |
| Redis | 权限缓存 | ✓ | 7.4 | — |
| Gin Router | 路由和中间件 | ✓ | 1.10.0 | — |
| GORM | ORM 数据访问 | ✓ | 1.30.5 | — |
| React | 前端框架 | ✓ | 19.2 | — |
| Zustand | 状态管理 | ✓ | 5.0 | — |

**Missing dependencies with no fallback:**
- 无

**Missing dependencies with fallback:**
- 无

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + React Vitest |
| Config file | 无配置文件（测试与源码同目录） |
| Quick run command | `go test ./internal/services/vdi/` |
| Full suite command | `go test ./... && cd xingran-react-frontend && npm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-01 | 细粒度权限分解 | integration | `go test -run TestGranularPermissions ./internal/api/v1/vdi/` | ❌ Wave 0 |
| D-02 | 数据范围过滤 | unit | `go test -run TestApplyVMDataScopeFilter ./internal/services/vdi/` | ❌ Wave 0 |
| D-03 | 无绑定用户规则 | unit | `go test -run TestApplyBoundUserFilter ./internal/services/vdi/` | ❌ Wave 0 |
| D-05 | 后端权限验证 | integration | `go test -run TestRequirePermissions ./internal/api/v1/vdi/` | ❌ Wave 0 |
| D-06 | 前端按钮级权限 | unit | `npm test -- --run testPermissionButtons` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/services/vdi/`
- **Per wave merge:** `go test ./... && cd xingran-react-frontend && npm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/services/vdi/vm_data_scope_filter_test.go` — 数据范围过滤函数测试
- [ ] `internal/api/v1/vdi/vm_router_test.go` — 路由权限中间件测试
- [ ] `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.test.tsx` — 前端按钮权限测试
- [ ] Framework install: 无需安装 — 项目已配置测试框架

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Access Control | yes | 现有 RBAC 权限系统 + 数据范围过滤 |
| V2 Authentication | yes | JWT Token 认证（已有） |
| V3 Session Management | yes | 双 Token 机制（已有） |
| V4 Access Control | yes | `RequirePermissions` 中间件验证操作权限 |
| V5 Input Validation | yes | GORM 参数化查询防止 SQL 注入 |

### Known Threat Patterns for VDI 权限系统

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 权限提升 | Tampering | `RequirePermissions` 中间件强制验证，Handler 层不处理权限逻辑 |
| 数据越界访问 | Information Disclosure | `DataScopePermission` 中间件 + `ApplyVMDataScopeFilter` 过滤数据范围 |
| SQL 注入 | Tampering | GORM 参数化查询，禁止字符串拼接 SQL |
| NULL 值绕过 | Information Disclosure | `ApplyBoundUserFilter` 强制过滤 `bound_user_id IS NULL` |
| 前端权限绕过 | Tampering | 后端强制验证权限，前端权限控制仅用于 UI 体验 |

## Sources

### Primary (HIGH confidence)
- [Context7 library ID] - 未使用（基于项目现有代码研究）
- [Official docs URL] - 项目内文档：`docs/项目概述和架构设计.md`、`docs/开发规范.md`
- [项目源码] - `internal/models/base.go`、`pkg/middleware/permission.go`、`internal/models/vdi.go`

### Secondary (MEDIUM confidence)
- [项目源码验证] - `internal/api/v1/vdi/vm_router.go`、`internal/services/vdi/vm_service_impl.go`、`xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`
- [现有菜单迁移] - `internal/core/db/migrations/129_add_vdi_menus.sql`

### Tertiary (LOW confidence)
- [WebSearch] - 未使用（所有信息来自项目源码和文档）

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 基于项目现有技术栈，无需新依赖
- Architecture: HIGH - 现有权限中间件和数据范围过滤机制已验证
- Pitfalls: HIGH - 基于现有代码分析，识别了 `bound_user_id` NULL 值等关键风险点

**Research date:** 2026-06-02
**Valid until:** 30 days（项目技术栈稳定，权限系统模式已验证）
