# 深信服桌面云集成 - 研究报告

## 1. API分析

### 1.1 虚拟机管理API（12个核心接口）

根据深信服桌面云开放平台API文档（V1.2），虚拟机管理模块包含以下接口：

| 序号 | 接口名称 | HTTP方法 | 端点路径 | 功能描述 |
|------|----------|----------|----------|----------|
| 1 | 操作虚拟机(批量) | POST | /v1/vm/operate | 批量开机、关机、重启、休眠等操作 |
| 2 | 虚拟机删除(批量) | DELETE | /v1/vm | 批量删除虚拟机 |
| 3 | 虚拟机配置IP | POST | /v1/vm/config_ip | 批量设置虚拟机IP地址 |
| 4 | 获取虚拟机可关联用户 | GET | /v1/vm/:id/available_users | 查询可关联到虚拟机的用户列表 |
| 5 | 虚拟机关联策略 | POST | /v1/vm/:id/bind_policy | 为虚拟机绑定策略组 |
| 6 | 虚拟机批量修改区域 | POST | /v1/vm/batch_modify_area | 批量修改虚拟机所属区域 |
| 7 | 虚拟机关联Windows补丁配置 | POST | /v1/vm/bind_patch_config | 关联Windows更新补丁配置 |
| 8 | 获取资源下所有虚拟机 | GET | /v1/resource/:id/vms | 获取指定资源组下的所有虚拟机 |
| 9 | 获取虚拟机详情 | GET | /v1/vm/:id | 获取指定虚拟机的详细信息 |
| 10 | 修改虚拟机名称 | PUT | /v1/vm/:id/name | 修改虚拟机名称 |
| 11 | 虚拟机关联用户 | POST | /v1/vm/:id/bind_user | 将虚拟机关联到用户 |
| 12 | 查询用户关联的虚拟机列表 | GET | /v1/user/:id/vms | 查询用户关联的所有虚拟机 |

### 1.2 资源管理API（4.3节）

- 新建资源组
- 编辑/删除资源组
- 新建独享桌面资源
- 获取资源列表
- **新建虚拟机** - 创建新的虚拟机实例

### 1.3 认证机制

**API认证流程：**

```
POST /v1/auth/tokens
{
  "auth": {
    "name": "admin",
    "password": "password"
  }
}

响应:
{
  "error_code": 0,
  "error_message": "",
  "data": {
    "token": {
      "tenant_id": 0,
      "auth_token": "eyJ0eXAiOiJKV1QiLCJhbGc..."
    }
  }
}
```

**后续请求需要在HTTP头中携带：**
```
Auth-Token: {auth_token}
```

### 1.4 响应格式

所有API响应遵循统一格式：

```json
{
  "error_code": 0,        // 0表示成功
  "error_message": "",
  "data": {               // 业务数据
    // ...
  }
}
```

## 2. 现有架构分析

### 2.1 项目结构

```
internal/
├── api/v1/
│   ├── operations/       # 参考：类似的CRUD模块
│   │   ├── building_handler.go
│   │   ├── building_router.go
│   │   └── ...
│   └── vdi/              # 新增：虚拟桌面管理模块
│       ├── vm_handler.go
│       ├── vm_router.go
│       └── ...
├── services/
│   ├── operations/       # 参考：业务逻辑层
│   └── vdi/              # 新增：VDI业务逻辑
│       ├── vm_service.go
│       ├── vdi_client.go # 深信服API客户端
│       └── ...
├── models/               # 新增：VDI数据模型
│   ├── virtual_machine.go
│   ├── vdi_resource.go
│   └── vdi_user_binding.go
└── config/
    └── vdi_config.go     # 新增：VDI配置
```

### 2.2 可复用的模式

**opsApi.ts 模式（前端）：**
- 统一的CRUD API封装
- 支持分页、筛选、排序
- 自动错误处理

**Handler-Service模式（后端）：**
- Handler：HTTP请求处理
- Service：业务逻辑封装
- Repository：数据访问（GORM）

## 3. 数据模型设计

### 3.1 虚拟机表（sys_vdi_vm）

```go
type VDIVirtualMachine struct {
    Base
    VMID           string    `gorm:"type:varchar(100);uniqueIndex" json:"vm_id"`           // 深信服VM ID
    Name           string    `gorm:"type:varchar(200);not null" json:"name"`               // 虚拟机名称
    ResourceID     string    `gorm:"type:varchar(100);index" json:"resource_id"`          // 资源组ID
    Status         int       `gorm:"type:int;default:0" json:"status"`                    // 0=正常, 1=停用
    PowerState     string    `gorm:"type:varchar(50)" json:"power_state"`                 // 电源状态：running/stopped/suspended
    IPAddress      string    `gorm:"type:varchar(50)" json:"ip_address"`                  // IP地址
    MACAddress     string    `gorm:"type:varchar(50)" json:"mac_address"`                 // MAC地址
    OSType         string    `gorm:"type:varchar(50)" json:"os_type"`                     // 操作系统类型
    CPU            int       `json:"cpu"`                                                 // CPU核心数
    Memory         int       `json:"memory"`                                              // 内存(MB)
    Disk           int       `json:"disk"`                                                // 磁盘(GB)
    BoundUserID    *string   `gorm:"type:varchar(100)" json:"bound_user_id"`              // 绑定用户ID
    BoundUserName  *string   `gorm:"type:varchar(200)" json:"bound_user_name"`            // 绑定用户名
    PolicyGroupID  *string   `gorm:"type:varchar(100)" json:"policy_group_id"`            // 策略组ID
    LastSyncAt     *time.Time `json:"last_sync_at"`                                       // 最后同步时间
    VdiServerID    string    `gorm:"type:varchar(100);index" json:"vdi_server_id"`        // VDI服务器ID
}
```

### 3.2 VDI服务器配置表（sys_vdi_server）

```go
type VDIServer struct {
    Base
    Name            string `gorm:"type:varchar(200);not null" json:"name"`
    Endpoint        string `gorm:"type:varchar(500);not null" json:"endpoint"`         // API端点
    Username        string `gorm:"type:varchar(100);not null" json:"username"`
    PasswordEncrypted string `gorm:"type:varchar(500);not null" json:"-"`              // SM4加密
    TenantID        int    `json:"tenant_id"`
    AuthToken       string `gorm:"type:varchar(1000)" json:"-"`                        // 缓存的token
    TokenExpiry     *time.Time `json:"-"`                                              // token过期时间
    Status          int    `gorm:"type:int;default:0" json:"status"`                   // 0=正常, 1=停用
}
```

### 3.3 资源组表（sys_vdi_resource_group）

```go
type VDIResourceGroup struct {
    Base
    ResourceGroupID string `gorm:"type:varchar(100);uniqueIndex" json:"resource_group_id"`
    Name            string `gorm:"type:varchar(200);not null" json:"name"`
    VdiServerID     string `gorm:"type:varchar(100);index" json:"vdi_server_id"`
    Type            string `gorm:"type:varchar(50)" json:"type"`                       // 独享桌面/池桌面
    Status          int    `gorm:"type:int;default:0" json:"status"`
}
```

## 4. 集成方案设计

### 4.1 VDI客户端封装

```go
// internal/services/vdi/vdi_client.go
type VDIClient interface {
    // 认证
    Authenticate(ctx context.Context) (string, error)
    
    // 虚拟机操作
    OperateVM(ctx context.Context, vmIDs []string, action string) error
    DeleteVM(ctx context.Context, vmIDs []string) error
    ConfigIP(ctx context.Context, req []VMIPConfig) error
    RenameVM(ctx context.Context, vmID, newName string) error
    
    // 查询
    GetVM(ctx context.Context, vmID string) (*VDIVMDetail, error)
    ListVMs(ctx context.Context, resourceID string) ([]VDIVMSummary, error)
    GetUserVMs(ctx context.Context, userID string) ([]VDIVMSummary, error)
    
    // 关联操作
    BindUser(ctx context.Context, vmID, userID string) error
    BindPolicy(ctx context.Context, vmID, policyID string) error
    GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error)
}
```

### 4.2 服务层设计

```go
// internal/services/vdi/vm_service.go
type VMService interface {
    // 虚拟机CRUD
    CreateVM(ctx context.Context, req *CreateVMRequest) (*VDIVirtualMachine, error)
    GetVM(ctx context.Context, id string) (*VDIVMDTO, error)
    ListVMs(ctx context.Context, req *ListVMRequest) (*PageResult, error)
    UpdateVM(ctx context.Context, id string, req *UpdateVMRequest) error
    DeleteVM(ctx context.Context, ids []string) error
    
    // 虚拟机操作
    OperateVM(ctx context.Context, ids []string, action VMPowerAction) error
    BatchConfigIP(ctx context.Context, req []VMIPConfigRequest) error
    RenameVM(ctx context.Context, id, newName string) error
    
    // 用户关联
    BindUser(ctx context.Context, vmID, userID string) error
    UnbindUser(ctx context.Context, vmID string) error
    
    // 同步
    SyncVMFromVDI(ctx context.Context, vmID string) error
    SyncAllVMs(ctx context.Context, serverID string) error
}
```

## 5. 前端设计

### 5.1 页面结构

```
src/pages/vdi/
├── VirtualMachineList.tsx     # 虚拟机列表
├── VirtualMachineDetail.tsx   # 虚拟机详情
├── ResourceGroupList.tsx      # 资源组管理
└── VDIServerConfig.tsx        # VDI服务器配置
```

### 5.2 API客户端

```typescript
// src/lib/vdiApi.ts
export const vdiApi = {
  // 虚拟机
  list: (params: ListParams) => post('/vdi/vm/list', params),
  get: (id: string) => post(`/vdi/vm/${id}`),
  operate: (ids: string[], action: string) => post('/vdi/vm/operate', { ids, action }),
  configIP: (configs: VMIPConfig[]) => post('/vdi/vm/config_ip', configs),
  rename: (id: string, name: string) => post(`/vdi/vm/${id}/rename`, { name }),
  
  // 资源组
  resourceGroups: {
    list: (params) => post('/vdi/resource-groups/list', params),
  }
};
```

## 6. 实施建议

### 6.1 分阶段实施

**Phase 1: 基础集成（22-sangfor-vdi-integration）**
- VDI客户端封装
- 数据模型创建
- 虚拟机列表查询
- 虚拟机详情查询
- 基础CRUD操作

**Phase 2: 高级操作（未来阶段）**
- 批量操作（开关机、配置IP）
- 用户关联
- 策略管理
- 监控和告警

**Phase 3: 优化和完善（未来阶段）**
- 缓存优化
- 性能优化
- 用户体验优化

### 6.2 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 深信服API变更 | 高 | 版本化客户端封装，抽象层隔离 |
| Token过期处理 | 中 | 自动刷新机制，错误重试 |
| 批量操作超时 | 中 | 异步任务队列，进度追踪 |
| 网络不稳定 | 中 | 重试机制，超时控制 |
| 数据一致性 | 高 | 本地缓存+定期同步 |

### 6.3 优先级排序

**P0（必须有）：**
1. VDI服务器配置管理
2. 认证机制
3. 虚拟机列表查询
4. 虚拟机详情查询
5. 虚拟机基本操作（开关机、重启）

**P1（重要）：**
1. 虚拟机创建
2. 虚拟机删除
3. IP配置
4. 用户关联

**P2（可选）：**
1. 策略管理
2. 高级监控
3. 批量操作优化
4. 资源组管理
