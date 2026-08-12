# Phase 25: 虚拟机数据范围权限配置 - Context

**Gathered:** 2026-06-02
**Status:** Ready for planning

<domain>
## Phase Boundary

为 Phase 22 实现的 VDI 虚拟机管理系统配置细粒度数据范围权限。

**核心目标：**
1. **细粒度操作权限** — 将通用权限 `vdi:vm:operate` 完全分解为独立的操作权限（启动、关机、重启、同步、删除）
2. **数据范围权限控制** — 修改 `bound_user_id` 绑定用户功能，使其用于数据范围识别：
   - 无绑定用户：仅数据范围为"全部"的角色可见
   - 绑定用户张三：张三所在部门和张三本人可见
3. **前端按钮级权限控制** — 根据用户拥有的权限动态显示/隐藏操作按钮

**不包含：**
- 修改 VDI 核心功能（Phase 22 已完成）
- 新增虚拟机操作类型（仅细化现有操作权限）

</domain>

<decisions>
## Implementation Decisions

### 细粒度操作权限分解

#### D-01: 完全分解操作权限
**决策：** 移除通用权限 `vdi:vm:operate`，完全分解为细粒度权限

**细粒度权限列表（5个操作权限）：**
- `vdi:vm:start` — 启动虚拟机
- `vdi:vm:stop` — 关机虚拟机
- `vdi:vm:restart` — 重启虚拟机
- `vdi:vm:sync` — 同步虚拟机状态
- `vdi:vm:delete` — 删除虚拟机

**绑定用户权限（用于数据范围控制，非操作权限）：**
- `vdi:vm:bind` — 绑定用户功能（控制谁能设置绑定关系）

**实施方式：**
- 移除原有的 `vdi:vm:operate` 权限
- 通过菜单迁移脚本新增 5 个细粒度操作权限到 `sys_menu` 表
- 更新路由层的权限中间件配置

### 数据范围权限实现

#### D-02: 参考现有 DataScope 模式
**决策：** 复用现有 `DataScope` 枚举和数据范围过滤机制

**实施方式：**
- 创建独立的数据范围过滤辅助函数 `ApplyDataScopeFilter()`
- 在 VDI Service 层调用该函数进行查询过滤
- 接收查询、用户角色和数据范围作为参数，返回过滤后的查询

#### D-03: 完整的数据范围过滤规则
**决策：** 实现四层数据范围过滤规则

1. **无绑定用户规则：**
   - `bound_user_id IS NULL` 时，仅 `DataScope=1`（全部）的角色可见
   - 其他数据范围的查询将被过滤掉无绑定用户的虚拟机

2. **本人可见规则：**
   - `bound_user_id = 当前用户ID` 时，`DataScope=5`（仅本人）可见
   - 查询条件：`WHERE bound_user_id = ?`

3. **部门可见规则：**
   - `bound_user_id` 所在部门用户，`DataScope=3`（本部门）或 `DataScope=4`（本部门及子部门）可见
   - 通过 `sys_user` 表关联 `dept_id` 进行过滤

4. **自定义部门规则：**
   - `DataScope=2`（自定义）通过角色关联的部门表（`sys_role_dept`）过滤
   - 查询条件：`WHERE bound_user_id IN (SELECT user_id FROM sys_role_dept WHERE role_id = ?)`

#### D-04: 实现层面设计
**决策：** 创建独立的数据范围过滤辅助函数

**函数签名：**
```go
func ApplyDataScopeFilter(query *gorm.DB, userID string, dataScope models.DataScope) *gorm.DB
```

**职责：**
- 根据 `dataScope` 值应用相应的 WHERE 条件
- 处理虚拟机的 `bound_user_id` 字段过滤
- 支持自定义数据范围（通过 `sys_role_dept` 表）

### 后端 API 权限验证

#### D-05: 基于现有模式的验证策略
**决策：** 路由层中间件 + Service 层过滤

**实施方式（参考现有 system 模块）：**
1. **路由层：**
   - 在 VM 路由组添加 `DataScopePermission` 中间件
   - 为单个操作路由添加 `RequirePermissions` 中间件（细粒度权限）

2. **Service 层：**
   - 从上下文获取 `data_scope` 和 `user_id`
   - 调用 `ApplyDataScopeFilter()` 函数进行数据范围过滤

**参考实现（来自 `internal/api/router.go`）：**
```go
users.Use(middleware.DataScopePermission(core))
{
    systemV1.SetupUserRouter(users, core)
}
```

### 前端按钮级权限控制

#### D-06: 使用现有权限系统和动态生成按钮
**决策：** 基于 `authStore.permissions` 动态生成操作按钮

**实施方式：**
1. 使用现有的 `usePermission()` Hook 检查权限
2. 根据用户拥有的权限数组动态生成操作按钮
3. 使用 `authStore.permissions` 作为权限数据来源

**按钮生成逻辑：**
- 遍历权限-按钮映射表
- 检查用户是否拥有对应权限（如 `hasPermission('vdi:vm:start')`）
- 有权限则显示对应按钮，无权限则隐藏

**权限-按钮映射：**
```typescript
const vmOperationButtons = [
  { action: 'start', permission: 'vdi:vm:start', label: '启动', icon: 'PlayCircleOutlined' },
  { action: 'stop', permission: 'vdi:vm:stop', label: '关机', icon: 'StopOutlined' },
  { action: 'restart', permission: 'vdi:vm:restart', label: '重启', icon: 'ReloadOutlined' },
  { action: 'sync', permission: 'vdi:vm:sync', label: '同步', icon: 'SyncOutlined' },
  { action: 'delete', permission: 'vdi:vm:delete', label: '删除', icon: 'DeleteOutlined' },
];
```

### Claude's Discretion

以下方面可以由实现者决定：

1. **数据范围过滤辅助函数的具体位置** — 放在 `internal/services/vdi/` 还是 `pkg/permission/`
2. **权限-按钮映射表的存储方式** — 常量、配置文件、还是数据库
3. **错误消息的具体文案** — 权限不足、数据范围越界等场景
4. **菜单迁移脚本的命名规范** — 遵循现有迁移文件命名模式
5. **前端按钮组件的具体实现** — 使用 Ant Design Button 还是其他组件

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目架构文档
- `docs/项目概述和架构设计.md` — 整体架构
- `docs/开发规范.md` — 开发规范
- `docs/API响应规范.md` — API 响应格式

### Phase 22 上下文（VDI 集成）
- `.planning/phases/22-sangfor-vdi-integration/22-CONTEXT.md` — VDI 系统设计和权限定义
- `.planning/phases/22-sangfor-vdi-integration/22-PATTERNS.md` — VDI 代码模式

### 现有代码文件
- `internal/models/vdi.go` — VDI 虚拟机模型（含 `BoundUserID` 字段）
- `internal/models/base.go` — `DataScope` 枚举定义
- `pkg/middleware/permission.go` — `DataScopePermission` 和 `RequirePermissions` 中间件
- `internal/api/router.go` — 路由权限配置参考（用户模块）
- `internal/core/db/migrations/129_add_vdi_menus.sql` — 现有 VDI 菜单权限定义

### 系统模块参考实现
- `internal/api/v1/system/user_router.go` — 用户路由（参考 DataScopePermission 用法）
- `internal/services/system/role_service.go` — 角色服务（参考数据范围处理）

### 前端参考
- `xingran-react-frontend/src/store/authStore.ts` — 权限存储
- `xingran-react-frontend/src/hooks/usePermission.ts` — 权限检查 Hook（如果存在）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **DataScope 枚举** (`internal/models/base.go`) — 5 种数据范围类型（1=全部, 2=自定义, 3=本部门, 4=本部门及子部门, 5=仅本人）
- **DataScopePermission 中间件** (`pkg/middleware/permission.go`) — 自动获取用户数据范围并存入上下文
- **RequirePermissions 中间件** (`pkg/middleware/permission.go`) — 验证用户是否拥有指定权限
- **VDI 虚拟机模型** (`internal/models/vdi.go`) — 已有 `BoundUserID` 和 `BoundUserName` 字段
- **现有权限系统** — `authStore.permissions` 存储用户权限列表

### Established Patterns
- **路由权限模式** — 在路由组使用 `DataScopePermission`，单个路由使用 `RequirePermissions`
- **Service 层过滤** — 从上下文获取 `data_scope` 和 `user_id`，在查询中应用过滤
- **权限标识符模式** — `vdi:{resource}:{action}` 格式
- **菜单权限管理** — 通过 `sys_menu` 表管理权限和菜单关联

### Integration Points
- **主路由** (`internal/api/router.go`) — 需要为 VM 路由组添加 `DataScopePermission` 中间件
- **VM 路由** (`internal/api/v1/vdi/vm_router.go`) — 需要为单个操作添加 `RequirePermissions` 中间件
- **VDI Service** (`internal/services/vdi/vm_service_impl.go`) — 需要调用数据范围过滤函数
- **菜单迁移** — 需要创建新的迁移脚本添加细粒度权限

### Known Constraints
- VDI 路由当前没有使用任何权限中间件（`vm_router.go` 直接注册路由）
- 现有的 `vdi:vm:operate` 权限需要移除
- 数据范围过滤必须基于 `bound_user_id` 字段
- 必须兼容现有的角色和部门关联表（`sys_role_dept`）

</code_context>

<specifics>
## Specific Ideas

### 数据范围过滤逻辑
```go
// ApplyDataScopeFilter 应用数据范围过滤到虚拟机查询
func ApplyDataScopeFilter(query *gorm.DB, userID string, dataScope models.DataScope) *gorm.DB {
    switch dataScope {
    case models.DataScopeAll:
        // 全部数据：不添加过滤条件
        return query
    case models.DataScopeCustom:
        // 自定义数据：通过 sys_role_dept 表过滤
        return query.Where("bound_user_id IN (SELECT user_id FROM sys_user_role ur JOIN sys_role_dept rd ON ur.role_id = rd.role_id WHERE ur.user_id = ?)", userID)
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
```

### 无绑定用户特殊处理
```go
// 无绑定用户的虚拟机仅对 DataScopeAll 可见
func ApplyBoundUserFilter(query *gorm.DB, dataScope models.DataScope) *gorm.DB {
    if dataScope != models.DataScopeAll {
        // 非 DataScopeAll 的查询自动过滤掉无绑定用户的虚拟机
        return query.Where("bound_user_id IS NOT NULL")
    }
    return query
}
```

### 路由配置示例
```go
// 在 internal/api/router.go 中
vms := authorized.Group("/vms")
vms.Use(middleware.DataScopePermission(core))
{
    vdiV1.SetupVMRouter(vms, core)
}
```

### 前端按钮生成示例
```typescript
// 动态生成操作按钮
const renderOperationButtons = (vm: VDIVM, permissions: string[]) => {
  const buttons = vmOperationButtons.filter(btn => 
    permissions.includes(btn.permission)
  );
  
  return buttons.map(btn => (
    <Button 
      key={btn.action}
      icon={<btn.icon />}
      onClick={() => handleOperation(btn.action, vm)}
    >
      {btn.label}
    </Button>
  ));
};
```

</specifics>

<deferred>
## Deferred Ideas

以下想法不在本期范围：

- **虚拟机组权限** — 基于资源组或分组的权限控制（后续阶段）
- **临时权限提升** — 用户临时申请更高权限的功能（属于权限管理模块）
- **权限审计日志** — 记录权限使用情况（属于审计模块）
- **前端权限配置界面** — 可视化配置权限和角色（属于角色管理模块）

</deferred>

---

*Phase: 25-vm-data-scope-permissions*
*Context gathered: 2026-06-02*
