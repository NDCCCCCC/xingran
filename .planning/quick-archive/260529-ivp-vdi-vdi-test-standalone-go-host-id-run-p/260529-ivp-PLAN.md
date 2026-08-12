---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/services/vdi/vdi_client_extended.go
  - internal/services/vdi/vm_service.go
  - internal/services/vdi/vm_service_impl.go
  - internal/api/v1/vdi/vm_handler.go
  - internal/api/v1/vdi/vm_router.go
  - xingran-react-frontend/src/types/vdi.ts
  - xingran-react-frontend/src/lib/vdiApi.ts
  - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
autonomous: true
requirements:
  - VDI虚拟机创建完整参数实现

must_haves:
  truths:
    - "后端VDI客户端可以获取运行位置、存储位置、网络接口列表"
    - "后端CreateVM调用真实VDI API POST /v1/servers，包含host.id/run_position.id/storage/network/disk参数"
    - "host.id从运行位置的father_id派生，run_position.id根据id==father_id规则确定"
    - "前端创建表单包含运行位置、存储、网络接口的级联下拉选择"
    - "选择资源后自动加载运行位置、存储、网络选项"
  artifacts:
    - path: "internal/services/vdi/vdi_client_extended.go"
      provides: "GetRunPositions, GetStorages, GetNetworks, CreateServer 方法"
    - path: "internal/services/vdi/vm_service_impl.go"
      provides: "完整的CreateVM实现，调用VDI API创建虚拟机"
    - path: "internal/api/v1/vdi/vm_handler.go"
      provides: "ListRunPositions, ListStorages, ListNetworks handler"
    - path: "xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx"
      provides: "创建表单增加运行位置/存储/网络下拉"
  key_links:
    - from: "vm_handler.go"
      to: "vm_service.go"
      via: "handler calls service methods"
      pattern: "vmService\\.(ListRunPositions|ListStorages|ListNetworks|CreateVM)"
    - from: "vm_service_impl.go"
      to: "vdi_client_extended.go"
      via: "service calls client VDI API methods"
      pattern: "client\\.(GetRunPositions|GetStorages|GetNetworks|CreateServer)"
    - from: "VirtualMachineList/index.tsx"
      to: "/vdi/vm/run-positions"
      via: "前端调用后端API获取运行位置"
      pattern: "vmApi\\.listRunPositions"
---

<objective>
实现VDI虚拟机创建的完整参数传递：运行位置(host.id/run_position.id逻辑)、存储位置、网络接口、VTP平台ID。

Purpose: 当前CreateVM是模拟实现(生成假的vm-id)，需要替换为真实的VDI API调用 POST /v1/servers，包含host.id/father_id派生逻辑、存储/网络/磁盘选择参数。参考scripts/vdi_test_standalone.go的CreateServer方法实现。

Output: 后端3个新API端点 + 真实CreateVM实现 + 前端创建表单4个新级联下拉选择器
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
@scripts/vdi_test_standalone.go

<interfaces>
<!-- VDI Client Extended Interface - methods to add -->
From internal/services/vdi/vdi_client_extended.go:
```go
type VDIClientExtended interface {
    // ... existing methods ...
    // NEW methods to add:
    GetRunPositions(ctx context.Context, vtpID int) ([]RunPosition, error)
    GetStorages(ctx context.Context, vtpID int) ([]Storage, error)
    GetNetworks(ctx context.Context, vtpID int) ([]Network, error)
    CreateServer(ctx context.Context, req CreateServerRequest) (*CreateServerResponse, error)
}
```

From internal/services/vdi/vdi_types.go - types already defined:
```go
type RunPosition struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    FatherID string `json:"father_id"`
}

type Storage struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Type   string `json:"type"`
    Total  string `json:"total"`
    Avail  string `json:"avail"`
    Shared int    `json:"shared"`
    Status int    `json:"status"`
}

type Network struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Mode string `json:"mode"`
}

type CreateServerRequest struct {
    Resource    ResourceInfo `json:"resource"`
    Host        HostInfo     `json:"host"`
    RunPosition PositionInfo `json:"run_position"`
    Disk        DiskInfo     `json:"disk"`
    Storage     StorageInfo  `json:"storage"`
    Network     NetworkInfo  `json:"network"`
    Servers     ServerCount  `json:"servers"`
}
```

From internal/services/vdi/vm_service.go - CreateVMServiceRequest to update:
```go
type CreateVMServiceRequest struct {
    Name        string `json:"name"`
    ResourceID  string `json:"resource_id"`
    VdiServerID string `json:"vdi_server_id"`
    CPUNumber   int    `json:"cpu_number"`
    CPUCore     int    `json:"cpu_core"`
    Memory      int    `json:"memory"`
    Disk        int    `json:"disk"`
}
```

From scripts/vdi_test_standalone.go - host.id / run_position.id derivation logic (lines 523-555):
```go
// host.id = father_id
hostID := fatherID
// run_position.id rule:
// id == father_id -> empty string (root node = host itself)
// id != father_id -> use id (child node = specific run position)
if id == fatherID {
    finalRunPositionID = ""
} else {
    finalRunPositionID = id
}
```

From internal/models/vdi.go - VDIServer model:
```go
type VDIServer struct {
    BaseModel
    Name              string `gorm:"type:varchar(200);not null"`
    Endpoint          string `gorm:"type:varchar(500);not null"`
    Username          string `gorm:"type:varchar(100);not null"`
    PasswordEncrypted string `gorm:"type:varchar(500);not null"`
    TenantID          int    `json:"tenant_id"` // This is the vtp_id!
    AuthToken         string `gorm:"type:varchar(1000)"`
    TokenExpiry       *time.Time
    LastSyncTime      *time.Time `gorm:"column:last_sync_time"`
    Status            int    `gorm:"type:int;default:0"`
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: 后端 - VDI客户端添加资源查询和创建方法 + Service层实现完整CreateVM</name>
  <files>
    internal/services/vdi/vdi_client_extended.go
    internal/services/vdi/vm_service.go
    internal/services/vdi/vm_service_impl.go
    internal/api/v1/vdi/vm_handler.go
    internal/api/v1/vdi/vm_router.go
  </files>
  <action>
**1. vdi_client_extended.go - 添加4个新方法到VDIClientExtended接口和实现:**

在 `VDIClientExtended` 接口中添加:
```go
GetRunPositions(ctx context.Context, vtpID int) ([]RunPosition, error)
GetStorages(ctx context.Context, vtpID int) ([]Storage, error)
GetNetworks(ctx context.Context, vtpID int) ([]Network, error)
CreateServer(ctx context.Context, req CreateServerRequest) (*CreateServerResponse, error)
```

实现这些方法，参考 vdi_test_standalone.go 的调用方式:
- `GetRunPositions`: GET `/v1/run_position?vtp_id={vtpID}`, 响应在 `data.run` 数组中 (注意RunPositionResponse结构已在vdi_types.go中定义，但解析时data是嵌套对象)
- `GetStorages`: GET `/v1/storages?vtp_id={vtpID}`, 响应在 `data.storages` 数组中
- `GetNetworks`: GET `/v1/networks?vtp_id={vtpID}`, 响应在 `data.networks` 数组中
- `CreateServer`: POST `/v1/servers`, 请求体为 `CreateServerRequest` 结构体 (已在vdi_types.go中定义), 响应为 `CreateServerResponse`

每个方法使用已有的 `callAPIWithRetry` 方法。

**2. vm_service.go - 更新CreateVMServiceRequest和接口:**

在 `CreateVMServiceRequest` 中添加缺失的VDI创建参数:
```go
type CreateVMServiceRequest struct {
    Name           string `json:"name" validate:"required"`
    ResourceID     string `json:"resource_id" validate:"required"`
    VdiServerID    string `json:"vdi_server_id" validate:"required"`
    CPUNumber      int    `json:"cpu_number" validate:"min=1,max=64"`
    CPUCore        int    `json:"cpu_core" validate:"min=1,max=128"`
    Memory         int    `json:"memory" validate:"min=512,max=131072"`
    Disk           int    `json:"disk" validate:"min=20,max=2000"`
    RunPositionID  string `json:"run_position_id"`   // 运行位置ID
    StorageID      string `json:"storage_id"`         // 存储位置ID
    NetworkID      string `json:"network_id"`         // 网络接口ID
    DiskID         string `json:"disk_id"`             // 个人盘ID (通常等于storageID)
    VtpID          int    `json:"vtp_id"`              // VTP平台ID
    Count          int    `json:"count"`               // 创建数量，默认1
}
```

在 `VMService` 接口中添加:
```go
ListRunPositions(ctx context.Context, vdiServerID string, vtpID int) ([]RunPosition, error)
ListStorages(ctx context.Context, vdiServerID string, vtpID int) ([]Storage, error)
ListNetworks(ctx context.Context, vdiServerID string, vtpID int) ([]Network, error)
```

**3. vm_service_impl.go - 实现完整的CreateVM和3个资源查询方法:**

`CreateVM` 方法替换模拟实现，改为调用真实VDI API:
1. 验证VDI服务器存在
2. 获取VDI客户端
3. 如果前端传了RunPositionID，获取运行位置列表，根据RunPositionID找到对应位置
4. 应用 host.id / run_position.id 派生逻辑 (关键！参考standalone脚本):
   - `hostID = position.FatherID` (host.id 取 father_id)
   - `runPositionID`: 如果 `position.ID == position.FatherID` 则为空字符串，否则取 `position.ID`
5. 构造 `CreateServerRequest`，resource.id 为前端传入的 ResourceID (转int)
6. 调用 `client.CreateServer()`
7. 检查响应 error_code，返回创建结果(包含task_id和server_id)
8. 不再在本地数据库创建虚拟机记录（VDI创建是异步操作，虚拟机通过同步获取）

实现 `ListRunPositions`、`ListStorages`、`ListNetworks`:
- 获取VDI客户端
- 调用对应client方法
- 返回结果

**4. vm_handler.go - 添加3个新handler:**

```go
// ListRunPositions 查询运行位置
func (h *VMHandler) ListRunPositions(c *gin.Context) {
    var req struct {
        VdiServerID string `json:"vdi_server_id"`
        VtpID       int    `json:"vtp_id"`
    }
    // bind, validate vtp_id required, call service
}

// ListStorages 查询存储位置
func (h *VMHandler) ListStorages(c *gin.Context) { /* same pattern */ }

// ListNetworks 查询网络接口
func (h *VMHandler) ListNetworks(c *gin.Context) { /* same pattern */ }
```

**5. vm_router.go - 注册3个新路由:**

在路由注册中添加:
```go
r.POST("/run-positions", vmHandler.ListRunPositions)
r.POST("/storages", vmHandler.ListStorages)
r.POST("/networks", vmHandler.ListNetworks)
```

注意: vtp_id 来自 VDIServer.TenantID 字段。前端需要从VDI服务器配置中读取tenant_id作为vtp_id传入。
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./...</automated>
  </verify>
  <done>
    - VDIClientExtended接口和实现包含GetRunPositions/GetStorages/GetNetworks/CreateServer方法
    - CreateVMServiceRequest包含RunPositionID/StorageID/NetworkID/DiskID/VtpID/Count字段
    - VMService接口包含ListRunPositions/ListStorages/ListNetworks方法
    - CreateVM调用真实VDI API POST /v1/servers，包含host.id派生逻辑
    - vm_handler.go包含ListRunPositions/ListStorages/ListNetworks handler
    - vm_router.go注册了/vm/run-positions、/vm/storages、/vm/networks路由
    - go build ./... 编译通过
  </done>
</task>

<task type="auto">
  <name>Task 2: 前端 - 创建表单添加运行位置/存储/网络级联选择器</name>
  <files>
    xingran-react-frontend/src/types/vdi.ts
    xingran-react-frontend/src/lib/vdiApi.ts
    xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
  </files>
  <action>
**1. vdi.ts - 更新类型定义:**

在 `CreateVMRequest` 中添加新字段:
```typescript
export interface CreateVMRequest {
  name: string;
  resource_id: string;
  vdi_server_id: string;
  cpu_number?: number;
  cpu_core?: number;
  memory?: number;
  disk?: number;
  run_position_id?: string;
  storage_id?: string;
  network_id?: string;
  disk_id?: string;
  vtp_id?: number;
  count?: number;
}
```

添加新类型:
```typescript
export interface RunPosition {
  id: string;
  name: string;
  father_id: string;
}

export interface VDIStorage {
  id: string;
  name: string;
  type: string;
  total: string;
  avail: string;
}

export interface VDINetwork {
  id: string;
  name: string;
  mode: string;
}
```

注意: Storage和Network类型名可能与antd冲突，使用VDIStorage/VDINetwork前缀。

**2. vdiApi.ts - 添加3个新API方法:**

在vmApi对象中添加:
```typescript
listRunPositions: (vdiServerId: string, vtpId: number) =>
    post('/vdi/vm/run-positions', { vdi_server_id: vdiServerId, vtp_id: vtpId }),

listStorages: (vdiServerId: string, vtpId: number) =>
    post('/vdi/vm/storages', { vdi_server_id: vdiServerId, vtp_id: vtpId }),

listNetworks: (vdiServerId: string, vtpId: number) =>
    post('/vdi/vm/networks', { vdi_server_id: vdiServerId, vtp_id: vtpId }),
```

**3. VirtualMachineList/index.tsx - 创建表单添加级联下拉:**

在已有的资源(resource)下拉之后，添加4个新的级联下拉选择器。级联关系:
- 选择VDI服务器 -> 获取vtp_id(从VDIServer.tenant_id)
- 选择资源 -> 自动加载运行位置、存储位置、网络接口

具体步骤:
a) 添加state: `runPositions`, `storages`, `networks` (对应类型数组)
b) 添加 Form.useWatch 监听 `resource_id` 变化
c) 当 `resource_id` 变化且selectedServerId存在时:
   - 从vdiServers数组中找到选中的服务器，获取其tenant_id作为vtp_id
   - 并行调用3个API: listRunPositions, listStorages, listNetworks
   - 将结果分别设置到state
d) 在创建模态框表单中，在资源选择器之后、名称输入之前，添加:
   - 运行位置下拉 (Select, 数据来自runPositions, 显示 name)
   - 存储位置下拉 (Select, 数据来自storages, 显示 name + avail信息)
   - 个人盘下拉 (Select, 数据复用storages, 默认选中与存储位置相同)
   - 网络接口下拉 (Select, 数据来自networks, 显示 name + mode)
   - 创建数量 (InputNumber, 默认1, min 1, max 10)
e) 清理逻辑: 当resource变化时，清空运行位置/存储/网络选项
f) 创建时将所有选择值传入CreateVMRequest

UI细节:
- 存储位置选项: `{name} ({avail}MB可用 / {type})`
- 运行位置选项: `{name}`
- 网络接口选项: `{name} ({mode})`
- 当资源未选择时，4个下拉框disabled
- 个人盘选择变更时自动同步disk_id字段
- 用antd的Grid或Space布局让表单更紧凑
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit 2>&1 | head -30</automated>
  </verify>
  <done>
    - vdi.ts包含RunPosition/VDIStorage/VDINetwork类型和更新的CreateVMRequest
    - vdiApi.ts包含listRunPositions/listStorages/listNetworks方法
    - 创建表单包含运行位置/存储/网络/个人盘/创建数量字段
    - 选择资源后自动加载运行位置/存储/网络选项(级联)
    - vtp_id从VDI服务器配置的tenant_id自动获取
    - 创建提交时完整参数传递到后端
    - TypeScript编译无错误
  </done>
</task>

</tasks>

<verification>
1. `go build ./...` 编译通过
2. `npx tsc --noEmit` TypeScript检查通过
3. 检查VDIClientExtended接口包含4个新方法
4. 检查vm_router.go包含3个新路由
5. 检查前端创建表单包含新的下拉选择器
</verification>

<success_criteria>
- 后端4个新VDI API方法实现(GetRunPositions/GetStorages/GetNetworks/CreateServer)
- 后端3个新HTTP端点(POST /vdi/vm/run-positions, /storages, /networks)
- CreateVM调用真实VDI API POST /v1/servers，包含host.id/father_id派生逻辑
- 前端创建表单包含运行位置/存储/网络/个人盘级联下拉选择
- go build 和 tsc 编译通过
</success_criteria>

<output>
After completion, create `.planning/quick/260529-ivp-vdi-vdi-test-standalone-go-host-id-run-p/260529-ivp-SUMMARY.md`
</output>
