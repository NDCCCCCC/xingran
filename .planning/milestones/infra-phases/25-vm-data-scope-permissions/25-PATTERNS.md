# Phase 25: 虚拟机数据范围权限配置 - Pattern Map

**Mapped:** 2025-06-02
**Files analyzed:** 7 (4 backend + 2 frontend + 1 migration)
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/api/v1/vdi/vm_router.go` | router | request-response | `internal/api/v1/system/user_router.go` | exact |
| `internal/services/vdi/vm_data_scope_filter.go` | utility | transform | `pkg/middleware/permission.go` (ApplyDataScope) | role-match |
| `internal/services/vdi/vm_service_impl.go` (modify) | service | request-response | `internal/services/system/user_service.go` | partial |
| `internal/core/db/migrations/131_add_vdi_granular_permissions.sql` | migration | batch | `internal/core/db/migrations/129_add_vdi_menus.sql` | exact |
| `xingran-react-frontend/src/pages/vdi/VirtualMachineList/vmOperationButtons.ts` | utility | config | N/A (new pattern) | no-analog |
| `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` (modify) | component | request-response | `xingran-react-frontend/src/store/authStore.ts` | partial |
| `internal/api/router.go` (modify) | router | request-response | `internal/api/router.go` (user routes) | exact |

## Pattern Assignments

### `internal/api/v1/vdi/vm_router.go` (router, request-response)

**Analog:** `internal/api/v1/system/user_router.go`

**Router pattern with DataScopePermission middleware** (lines 133-147 from router.go):
```go
// 用户管理
users := authorized.Group("/users")
users.Use(middleware.RequirePermissions([]string{
    string(permission.UserList),
    string(permission.UserAdd),
    string(permission.UserEdit),
    string(permission.UserView),
}, core))
// 添加数据权限中间件
users.Use(middleware.DataScopePermission(core))
{
    // 新架构：结构体Handler + Service层
    systemV1.SetupUserRouter(users, core)
}
```

**Current VM Router pattern** (vm_router.go lines 1-37):
```go
// SetupVMRouter 设置虚拟机路由
// 始终注册路由，使用动态客户端
func SetupVMRouter(r *gin.RouterGroup, core *core.Core) {
    // 创建服务实例，传入nil客户端（服务层会动态查找）
    vmService := vdiServices.NewVMServiceWithDynamicClient(core.GetDB())
    vmHandler := NewVMHandler(vmService, core.GetDB())

    // 注册路由
    r.POST("/list", vmHandler.List)
    r.POST("/resource-groups", vmHandler.ListResourceGroups)
    r.POST("/resources", vmHandler.ListResources)
    r.POST("", vmHandler.Create)
    r.POST("/:id", vmHandler.GetByID)
    r.POST("/:id/update", vmHandler.Update)
    r.POST("/:id/delete", vmHandler.Delete)

    // VDI特定操作（调用VDI API）
    r.POST("/operate", vmHandler.Operate)
    r.POST("/:id/bind_user", vmHandler.BindUser)
    r.POST("/:id/unbind_user", vmHandler.UnbindUser)
    r.POST("/:id/sync", vmHandler.SyncFromVDI)
    r.POST("/sync-all", vmHandler.SyncAll)
}
```

**Required changes:** Add `DataScopePermission` middleware to VM routes in `internal/api/router.go` and add `RequirePermissions` middleware to individual operation routes.

---

### `internal/services/vdi/vm_data_scope_filter.go` (utility, transform)

**Analog:** `pkg/middleware/permission.go` (ApplyDataScope function)

**ApplyDataScope pattern** (permission.go lines 253-317):
```go
// ApplyDataScope 应用数据权限过滤
func ApplyDataScope(db *gorm.DB, userID string, dataScope models.DataScope, core *core.Core, deptField string) *gorm.DB {
    // 验证字段名安全性，防止SQL注入
    if !isValidDataScopeField(deptField) {
        return db.Where("1=0") // 无效字段名，返回空结果
    }

    switch dataScope {
    case models.DataScopeAll:
        // 全部数据权限，不做过滤
        return db
    case models.DataScopeCustom:
        // 自定义数据权限，查询用户可访问的部门
        var deptIds []string
        core.DB.GetDB().Raw(`
            SELECT DISTINCT rd.dept_id
            FROM sys_user_role ur
            INNER JOIN sys_role_dept rd ON ur.role_id = rd.role_id
            WHERE ur.user_id = ?
        `, userID).Scan(&deptIds)

        if len(deptIds) > 0 {
            return db.Where(deptField+" IN ?", deptIds)
        }
        return db.Where("1=0") // 没有权限访问任何数据
    case models.DataScopeDept:
        // 本部门数据权限，查询用户所属部门
        var deptId string
        if err := core.DB.GetDB().Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId).Error; err != nil {
            if err != gorm.ErrRecordNotFound {
                // 非记录未找到错误，记录数据库错误日志
                applogger.Errorf("Failed to query user dept for data scope filtering (user_id=%s): %v", userID, err)
            }
            return db.Where("1=0")
        }
        if deptId != "" {
            return db.Where(deptField+" = ?", deptId)
        }
        return db.Where("1=0")
    case models.DataScopeDeptChild:
        // 本部门及子部门数据权限
        var deptId string
        if err := core.DB.GetDB().Raw("SELECT dept_id FROM sys_user WHERE id = ?", userID).Scan(&deptId).Error; err != nil {
            if err != gorm.ErrRecordNotFound {
                // 非记录未找到错误，记录数据库错误日志
                applogger.Errorf("Failed to query user dept for data scope filtering (user_id=%s): %v", userID, err)
            }
            return db.Where("1=0")
        }
        if deptId != "" {
            // 查询本部门及所有子部门
            var childDeptIds []string
            childDeptIds = append(childDeptIds, deptId)
            getChildDepts(core, deptId, &childDeptIds)

            return db.Where(deptField+" IN ?", childDeptIds)
        }
        return db.Where("1=0")
    case models.DataScopeSelf:
        // 仅本人数据权限
        return db.Where(deptField+" = ?", userID)
    default:
        return db.Where("1=0") // 默认无权限
    }
}
```

**VM-specific adaptation needed:** 
- Replace `deptField` with `bound_user_id` field filtering
- Add NULL value handling for `bound_user_id IS NULL` case
- Use subqueries to filter by `bound_user_id` department membership

---

### `internal/services/vdi/vm_service_impl.go` (service, request-response)

**Analog:** `internal/services/vdi/vm_service_impl.go` (ListVMs method)

**Current ListVMs pattern** (vm_service_impl.go lines 554-641):
```go
// ListVMs 获取虚拟机列表
func (s *vmServiceImpl) ListVMs(ctx context.Context, req *ListVMRequest) (*PageResult, error) {
    // 检查是否有启用的VDI服务器配置
    var serverCount int64
    if err := s.db.WithContext(ctx).Model(&models.VDIServer{}).Where("status = 0").Count(&serverCount).Error; err != nil {
        return nil, fmt.Errorf("failed to check VDI servers: %w", err)
    }

    // 如果没有启用的VDI服务器，返回空列表并提示
    if serverCount == 0 {
        return &PageResult{
            List:     []VDIVMDTO{},
            Total:    0,
            Page:     req.Page,
            PageSize: req.PageSize,
        }, nil
    }

    // 获取VDI客户端
    client, err := s.getClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get VDI client: %w", err)
    }

    // 检查本地数据库是否有虚拟机数据
    var localCount int64
    if err := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{}).Count(&localCount).Error; err != nil {
        return nil, fmt.Errorf("failed to check local VMs: %w", err)
    }

    // 如果本地数据为空，从VDI服务器同步数据
    if localCount == 0 {
        if err := s.syncVMsFromVDI(ctx, client); err != nil {
            return nil, fmt.Errorf("failed to sync VMs from VDI: %w", err)
        }
    }

    // 设置默认分页参数
    if req.Page <= 0 {
        req.Page = 1
    }
    if req.PageSize <= 0 || req.PageSize > 100 {
        req.PageSize = 10
    }

    // 构建查询
    query := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{})

    // 添加过滤条件
    if req.Name != "" {
        query = query.Where("name LIKE ?", "%"+req.Name+"%")
    }
    if req.VdiServerID != "" {
        query = query.Where("vdi_server_id = ?", req.VdiServerID)
    }
    if req.ResourceID != "" {
        query = query.Where("resource_id = ?", req.ResourceID)
    }
    if req.PowerState != "" {
        query = query.Where("power_state = ?", req.PowerState)
    }

    // 获取总数
    var total int64
    if err := query.Count(&total).Error; err != nil {
        return nil, fmt.Errorf("failed to count VMs: %w", err)
    }

    // 分页查询
    var vms []models.VDIVirtualMachine
    offset := (req.Page - 1) * req.PageSize
    if err := query.Offset(offset).Limit(req.PageSize).Find(&vms).Error; err != nil {
        return nil, fmt.Errorf("failed to list VMs: %w", err)
    }

    // 转换为DTO
    dtos := make([]VDIVMDTO, len(vms))
    for i, vm := range vms {
        dtos[i] = *s.toDTO(&vm)
    }

    return &PageResult{
        List:     dtos,
        Total:    total,
        Page:     req.Page,
        PageSize: req.PageSize,
    }, nil
}
```

**Required modification:** Add data scope filtering after line 615 (after building query):
```go
// 从上下文获取用户信息
userID := ctx.Value("user_id").(string)
dataScope := ctx.Value("data_scope").(models.DataScope)

// 应用数据范围过滤
query = ApplyVMDataScopeFilter(query, userID, dataScope, s.core)
```

---

### `internal/core/db/migrations/131_add_vdi_granular_permissions.sql` (migration, batch)

**Analog:** `internal/core/db/migrations/129_add_vdi_menus.sql`

**Menu migration pattern** (129_add_vdi_menus.sql lines 1-23):
```sql
-- VDI 虚拟机管理模块菜单数据
-- 虚拟机管理一级菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440001', '虚拟机管理', NULL, 5, 'vdi', 'Layout', 'M', '1', '0', 'vdi:visit', 'CloudServerOutlined', '虚拟机管理目录', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 虚拟机列表菜单
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440002', '虚拟机列表', '770e8400-e29b-41d4-a716-446655440001', 1, 'vdi/vm', 'vdi/VirtualMachineList/index', 'C', '1', '0', 'vdi:vm:list', 'DesktopOutlined', '虚拟机列表菜单', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

-- 虚拟机列表按钮
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at, deleted_at) VALUES
('770e8400-e29b-41d4-a716-446655440003', '虚拟机查询', '770e8400-e29b-41d4-a716-446655440002', 1, '', '', 'F', '1', '0', 'vdi:vm:query', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440004', '虚拟机新增', '770e8400-e29b-41d4-a716-446655440002', 2, '', '', 'F', '1', '0', 'vdi:vm:add', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440005', '虚拟机修改', '770e8400-e29b-41d4-a716-446655440002', 3, '', '', 'F', '1', '0', 'vdi:vm:edit', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440006', '虚拟机删除', '770e8400-e29b-41d4-a716-446655440002', 4, '', '', 'F', '1', '0', 'vdi:vm:remove', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440007', '虚拟机操作', '770e8400-e29b-41d4-a716-446655440002', 5, '', '', 'F', '1', '0', 'vdi:vm:operate', '#', '开关机、重启等操作', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440008', '配置IP', '770e8400-e29b-41d4-a716-446655440002', 6, '', '', 'F', '1', '0', 'vdi:vm:config', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440009', '重命名', '770e8400-e29b-41d4-a716-446655440002', 7, '', '', 'F', '1', '0', 'vdi:vm:rename', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440010', '绑定用户', '770e8400-e29b-41d4-a716-446655440002', 8, '', '', 'F', '1', '0', 'vdi:vm:bind', '#', '', NOW(), NOW(), NULL),
('770e8400-e29b-41d4-a716-446655440011', '同步状态', '770e8400-e29b-41d4-a716-446655440002', 9, '', '', 'F', '1', '0', 'vdi:vm:sync', '#', '从VDI服务器同步虚拟机状态', NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;
```

**Required pattern:** 
1. Delete existing `vdi:vm:operate` permission (line 18 above)
2. Add 5 granular permissions: `vdi:vm:start`, `vdi:vm:stop`, `vdi:vm:restart`, `vdi:vm:sync`, `vdi:vm:delete`
3. Use `ON CONFLICT (id) DO NOTHING` for idempotency
4. Use UUID generation for menu IDs (pattern: `770e8400-e29b-41d4-a716-4466554400XX`)

---

### `xingran-react-frontend/src/pages/vdi/VirtualMachineList/vmOperationButtons.ts` (utility, config)

**Analog:** N/A (new pattern, based on TypeScript interface patterns)

**TypeScript interface pattern** (from types/system.ts lines 12-34):
```typescript
/**
 * 用户类型
 */
export interface User {
  id: string;
  username: string;
  nickname?: string;
  employeeNo?: string;
  email?: string;
  phone?: string;
  avatar?: string;
  gender: 0 | 1 | 2;
  status: Status;
  deptId?: string;
  deptName?: string;
  deptFullName?: string; // 完整部门路径（从二级开始）
  roles: string[];
  roleIds?: string[];
  permissions: string[];
  isAdmin?: boolean;
  dataScope?: string;
  loginIp?: string;
  loginTime?: string;
  createTime: string;
  updateTime: string;
}
```

**VM operation buttons pattern** (new file):
```typescript
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

---

### `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` (component, request-response)

**Analog:** `xingran-react-frontend/src/store/authStore.ts` (permissions access pattern)

**AuthStore permissions pattern** (authStore.ts lines 12-34, 77-78):
```typescript
interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  menusLoaded: boolean; // 登录时是否已加载菜单
  initialized: boolean; // 是否已尝试从存储恢复
}

export interface User {
  // ... other fields
  permissions: string[];
  // ... other fields
}
```

**Dynamic button rendering pattern** (from RESEARCH.md lines 271-305):
```typescript
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

---

### `internal/api/router.go` (router, request-response)

**Analog:** `internal/api/router.go` (VDI routes section, lines 133-147)

**Existing VDI route pattern** (search for VDI routes in router.go):
```go
// VDI management routes
vdi := authorized.Group("/vdi")
{
    vdiV1.SetupVMRouter(vms, core)
}
```

**Required modification:** Wrap VDI routes with DataScopePermission middleware:
```go
// VDI management routes
vms := authorized.Group("/vdi/vms")
vms.Use(middleware.DataScopePermission(core))
{
    vdiV1.SetupVMRouter(vms, core)
}
```

---

## Shared Patterns

### Authentication/Authorization Middleware
**Source:** `pkg/middleware/permission.go`
**Apply to:** All VDI route configurations

**DataScopePermission middleware** (lines 207-237):
```go
// DataScopePermission 数据权限中间件
func DataScopePermission(core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, ok := getUserIDAsString(c)
        if !ok {
            response.Error(c, response.ErrUnauthorized, "用户未认证")
            c.Abort()
            return
        }

        // 超级管理员直接通过
        if isSuperAdmin(core, userID) {
            c.Next()
            return
        }

        // 获取用户最大数据权限范围
        dataScope, err := getUserMaxDataScope(core, userID)
        if err != nil {
            response.Error(c, response.ErrServerError, "获取数据权限失败")
            c.Abort()
            return
        }

        // 将数据权限信息存储到上下文
        c.Set("data_scope", dataScope)
        c.Set("user_id", userID)

        c.Next()
    }
}
```

**RequirePermissions middleware** (lines 149-176):
```go
// RequirePermissions 需要多个权限中的任意一个
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, ok := getUserIDAsString(c)
        if !ok {
            response.Error(c, response.ErrUnauthorized, "用户未认证")
            c.Abort()
            return
        }

        // 超级管理员直接通过
        if isSuperAdmin(core, userID) {
            c.Next()
            return
        }

        // 检查是否有任意一个权限
        for _, permission := range permissions {
            if checkUserPermission(core, userID, permission) {
                c.Next()
                return
            }
        }

        response.Error(c, response.ErrForbidden, "没有访问权限")
        c.Abort()
    }
}
```

---

### DataScope Enum Definition
**Source:** `internal/models/base.go`
**Apply to:** All data scope filtering logic

**DataScope enum** (lines 69-78):
```go
// DataScope 数据范围枚举
type DataScope int

const (
    DataScopeAll       DataScope = 1 // 全部数据
    DataScopeCustom    DataScope = 2 // 自定义数据
    DataScopeDept      DataScope = 3 // 本部门数据
    DataScopeDeptChild DataScope = 4 // 本部门及子部门数据
    DataScopeSelf      DataScope = 5 // 仅本人数据
)
```

---

### Context Value Extraction Pattern
**Source:** `pkg/middleware/permission.go` (getUserIDAsString function)
**Apply to:** Service layer data scope filtering

**Context value extraction** (lines 24-35):
```go
// getUserIDAsString 安全地从context获取user_id
func getUserIDAsString(c *gin.Context) (string, bool) {
    userID, exists := c.Get("user_id")
    if !exists {
        return "", false
    }
    userIDStr, ok := userID.(string)
    if !ok {
        return "", false
    }
    return userIDStr, true
}
```

**Service layer context extraction** (from RESEARCH.md lines 326-342):
```go
// 从上下文获取用户信息
userID := ctx.Value("user_id").(string)
dataScope := ctx.Value("data_scope").(models.DataScope)

// 构建查询
query := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{})

// 应用数据范围过滤
query = ApplyVMDataScopeFilter(query, userID, dataScope, s.core)
```

---

### GORM Query Building Pattern
**Source:** `internal/services/vdi/vm_service_impl.go` (ListVMs method)
**Apply to:** All VDI service list queries

**Query building pattern** (lines 599-627):
```go
// 构建查询
query := s.db.WithContext(ctx).Model(&models.VDIVirtualMachine{})

// 添加过滤条件
if req.Name != "" {
    query = query.Where("name LIKE ?", "%"+req.Name+"%")
}
if req.VdiServerID != "" {
    query = query.Where("vdi_server_id = ?", req.VdiServerID)
}
if req.ResourceID != "" {
    query = query.Where("resource_id = ?", req.ResourceID)
}
if req.PowerState != "" {
    query = query.Where("power_state = ?", req.PowerState)
}

// 获取总数
var total int64
if err := query.Count(&total).Error; err != nil {
    return nil, fmt.Errorf("failed to count VMs: %w", err)
}

// 分页查询
var vms []models.VDIVirtualMachine
offset := (req.Page - 1) * req.PageSize
if err := query.Offset(offset).Limit(req.PageSize).Find(&vms).Error; err != nil {
    return nil, fmt.Errorf("failed to list VMs: %w", err)
}
```

---

### UUID Generation Pattern
**Source:** `internal/models/base.go` (BeforeCreate hook)
**Apply to:** All menu migration scripts

**UUID generation** (lines 10-27):
```go
import (
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
    ID        string         `gorm:"type:uuid;primary_key" json:"id"`
    CreatedAt time.Time      `json:"createdAt"`
    UpdatedAt time.Time      `json:"updatedAt"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
    CreatedBy string         `gorm:"size:64" json:"createdBy"`
    UpdatedBy string         `gorm:"size:64" json:"updatedBy"`
    Version   int            `json:"version"`
}

// BeforeCreate GORM钩子 - 创建前
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
    if b.ID == "" {
        b.ID = uuid.New().String()
    }
    return nil
}
```

**Migration script UUID pattern** (use fixed UUIDs for reproducibility):
```sql
-- Use pattern: 770e8400-e29b-41d4-a716-4466554400XX
-- XX = sequential number (20, 21, 22, 23, 24 for new permissions)
```

---

### Frontend Permission-Based Rendering Pattern
**Source:** `xingran-react-frontend/src/store/authStore.ts`
**Apply to:** All frontend components that need permission-based UI

**Permission store access** (from authStore.ts):
```typescript
// User interface includes permissions array
export interface User {
  permissions: string[];
  // ... other fields
}

// Access via useAuthStore hook
const { user } = useAuthStore();
const permissions = user?.permissions || [];
```

**Permission-based filtering** (from RESEARCH.md):
```typescript
// Filter buttons based on permissions
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
```

---

## No Analog Found

No files without analogs. All new files have close matches in the existing codebase.

## Metadata

**Analog search scope:**
- Backend: `internal/api/v1/`, `internal/services/`, `internal/models/`, `pkg/middleware/`, `internal/core/db/migrations/`
- Frontend: `xingran-react-frontend/src/store/`, `xingran-react-frontend/src/types/`, `xingran-react-frontend/src/pages/`

**Files scanned:** 15 Go files, 5 TypeScript files, 2 SQL migration files

**Pattern extraction date:** 2025-06-02

**Key pattern insights:**
1. **Router middleware pattern**: Use `DataScopePermission` at route group level, `RequirePermissions` at individual route level
2. **Data scope filtering**: Service layer extracts context values and applies field-specific filtering
3. **VM-specific adaptation**: Must handle `bound_user_id` field (nullable) instead of `dept_id`
4. **Frontend permissions**: Access via `authStore.user.permissions` array, use `includes()` for checking
5. **Migration scripts**: Use fixed UUIDs with sequential pattern, `ON CONFLICT` for idempotency
