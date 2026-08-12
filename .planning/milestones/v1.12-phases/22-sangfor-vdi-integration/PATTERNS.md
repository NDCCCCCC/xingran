# 深信服桌面云集成 - 架构模式参考

## 1. 可复用的项目模式

### 1.1 opsApi.ts 模式（前端CRUD封装）

**位置:** `xingran-react-frontend/src/lib/opsApi.ts`

这个文件提供了统一的CRUD API工厂模式，非常适合用于VDI模块的前端API封装。

**核心模式:**

```typescript
// 统一的API创建工厂
function createCrudApi<T>(baseUrl: string) {
  return {
    list: (params: ListParams) => post(`${baseUrl}/list`, params),
    get: (id: string) => post(`${baseUrl}/${id}`),
    create: (data: T) => post(`${baseUrl}`, data),
    update: (id: string, data: Partial<T>) => post(`${baseUrl}/${id}/update`, data),
    delete: (id: string) => post(`${baseUrl}/${id}/delete`)
  };
}

// 使用示例
export const buildingApi = createCrudApi<Building>('/ops/building');
export const floorApi = createCrudApi<Floor>('/ops/floor');
```

**VDI模块应用:**

```typescript
// src/lib/vdiApi.ts
export const vmApi = createCrudApi<VirtualMachine>('/vdi/vm');
export const resourceGroupApi = createCrudApi<ResourceGroup>('/vdi/resource-groups');
export const vdiServerApi = createCrudApi<VDIServer>('/vdi/servers');
```

### 1.2 Handler-Service 模式（后端）

**标准结构:**

```
internal/api/v1/{module}/
├── {entity}_handler.go      # HTTP处理
├── {entity}_router.go       # 路由配置
└── dto.go                   # 数据传输对象

internal/services/{module}/
├── {entity}_service.go      # 服务接口
├── {entity}_service_impl.go # 服务实现
└── repository.go            # 数据访问（可选）
```

**Handler 模板:**

```go
// internal/api/v1/vdi/vm_handler.go
type VMHandler struct {
    vmService vdi.VMService
}

func NewVMHandler(vmService vdi.VMService) *VMHandler {
    return &VMHandler{vmService: vmService}
}

func (h *VMHandler) List(c *gin.Context) {
    var req vdi.ListVMRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "请求参数错误")
        return
    }

    result, err := h.vmService.ListVMs(c.Request.Context(), &req)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }

    response.Success(c, result)
}
```

**Router 模板:**

```go
// internal/api/v1/vdi/vm_router.go
func SetupVMRouter(r *gin.RouterGroup, core *core.Core) {
    vmService := vdi.NewVMService(core.GetDB(), core.Cache)
    vmHandler := vdi.NewVMHandler(vmService)

    r.POST("/list", vmHandler.List)
    r.POST("", vmHandler.Create)
    r.POST("/:id", vmHandler.GetByID)
    r.POST("/:id/update", vmHandler.Update)
    r.POST("/:id/delete", vmHandler.Delete)

    // VDI特定操作
    r.POST("/operate", vmHandler.Operate)
    r.POST("/config-ip", vmHandler.ConfigIP)
}
```

### 1.3 Service 接口模式

```go
// internal/services/vdi/vm_service.go
type VMService interface {
    // CRUD
    CreateVM(ctx context.Context, req *CreateVMRequest) (*VDIVirtualMachine, error)
    GetVM(ctx context.Context, id string) (*VDIVMDTO, error)
    ListVMs(ctx context.Context, req *ListVMRequest) (*PageResult, error)
    UpdateVM(ctx context.Context, id string, req *UpdateVMRequest) error
    DeleteVM(ctx context.Context, ids []string) error

    // VDI特定操作
    OperateVM(ctx context.Context, ids []string, action VMPowerAction) error
    ConfigIP(ctx context.Context, req []VMIPConfigRequest) error
}

type vmServiceImpl struct {
    db        *gorm.DB
    cache     cache.Cache
    vdiClient VDIClient
}

func NewVMService(db *gorm.DB, cache cache.Cache) VMService {
    return &vmServiceImpl{
        db:    db,
        cache: cache,
        // vdiClient 从配置初始化
    }
}
```

## 2. 文件命名和组织惯例

### 2.1 后端文件命名

```
# Handler层
{entity}_handler.go          # 主handler文件
{entity}_router.go           # 路由配置
dto.go                       # 请求/响应DTO

# Service层
{entity}_service.go          # 服务接口定义
{entity}_service_impl.go     # 服务实现
{entity}_cache_service.go    # 缓存服务（如需要）

# Model层
{entity}.go                  # 数据模型定义
{entity}_sync.go             # 同步逻辑（如需要）
```

### 2.2 前端文件命名

```
src/pages/vdi/
├── VirtualMachine/
│   ├── index.tsx            # 列表页
│   ├── Detail.tsx           # 详情页
│   └── components/          # 子组件
│       ├── VMTable.tsx
│      ── VMOpsButtons.tsx
│       └── ...
├── ResourceGroup/
│   └── index.tsx
└── components/              # 共享组件
    └── VDIStatusTag.tsx

src/types/vdi.ts             # 类型定义
src/lib/vdiApi.ts            # API客户端
```

## 3. 代码模板和示例

### 3.1 批量操作模式

参考 Excel 导入的批量处理模式：

```go
// internal/services/operations/excel_service.go
type BatchResult struct {
    Success int
    Failed  int
    Errors  []BatchError
}

func (s *ExcelService) BatchUpsert(ctx context.Context, items []Item) *BatchResult {
    result := &BatchResult{}

    for _, item := range items {
        if err := s.processItem(ctx, item); err != nil {
            result.Failed++
            result.Errors = append(result.Errors, BatchError{
                Row:  item.Row,
                Error: err.Error(),
            })
        } else {
            result.Success++
        }
    }

    return result
}
```

**VDI应用:**

```go
// internal/services/vdi/vm_service.go
func (s *vmServiceImpl) BatchOperateVM(ctx context.Context, req []VMOperateRequest) *BatchResult {
    result := &BatchResult{}

    for _, r := range req {
        if err := s.vdiClient.OperateVM(ctx, r.VMID, r.Action); err != nil {
            result.Failed++
            result.Errors = append(result.Errors, BatchError{
                VMID:  r.VMID,
                Error: err.Error(),
            })
        } else {
            result.Success++
        }
    }

    return result
}
```

### 3.2 错误处理模式

```go
// 使用 pkg/errors 的错误包装
if err := ValidateVMID(vmID); err != nil {
    return fmt.Errorf("invalid VM ID: %w", err)
}

// Handler层转换为HTTP响应
if err != nil {
    if errors.Is(err, ErrVMNotFound) {
        response.Error(c, http.StatusNotFound, "虚拟机不存在")
        return
    }
    response.Error(c, http.StatusInternalServerError, err.Error())
    return
}
```

### 3.3 分页查询模式

```go
type ListVMRequest struct {
    Page       int    `json:"page"`
    PageSize   int    `json:"pageSize"`
    Name       string `json:"name"`
    Status     *int   `json:"status"`
    ResourceID string `json:"resourceId"`
}

func (s *vmServiceImpl) ListVMs(ctx context.Context, req *ListVMRequest) (*PageResult, error) {
    query := s.db.WithContext(ctx).Model(&VDIVirtualMachine{})

    // 筛选条件
    if req.Name != "" {
        query = query.Where("name LIKE ?", "%"+req.Name+"%")
    }
    if req.Status != nil {
        query = query.Where("status = ?", *req.Status)
    }

    // 分页
    var total int64
    query.Count(&total)

    var vms []VDIVirtualMachine
    if err := query.Offset((req.Page - 1) * req.PageSize).
        Limit(req.PageSize).
        Find(&vms).Error; err != nil {
        return nil, err
    }

    return &PageResult{
        List:     vms,
        Total:    total,
        Page:     req.Page,
        PageSize: req.PageSize,
    }, nil
}
```

### 3.4 缓存模式

```go
// 使用 CacheProvider 接口
func (s *vmServiceImpl) GetVM(ctx context.Context, id string) (*VDIVMDTO, error) {
    var result VDIVMDTO

    // 尝试从缓存获取
    cacheKey := fmt.Sprintf("vdi:vm:%s", id)
    if err := s.cache.Get(ctx, cacheKey, &result); err == nil {
        return &result, nil
    }

    // 从数据库查询
    vm, err := s.getVMFromDB(ctx, id)
    if err != nil {
        return nil, err
    }

    // 转换为DTO
    result = s.toDTO(vm)

    // 缓存结果（5分钟）
    s.cache.Set(ctx, cacheKey, result, 5*time.Minute)

    return &result, nil
}
```

## 4. 需要避免的反模式

### 4.1 ❌ 直接在Handler中调用外部API

```go
// 错误示例
func (h *VMHandler) Operate(c *gin.Context) {
    // 直接HTTP调用深信服API
    resp, err := http.Post("...")
}
```

**✅ 正确做法:**

```go
// 通过Service层封装
func (h *VMHandler) Operate(c *gin.Context) {
    err := h.vmService.OperateVM(c.Request.Context(), req)
}
```

### 4.2 ❌ 硬编码配置

```go
// 错误示例
const VDI_ENDPOINT = "https://vdi.example.com/api"
```

**✅ 正确做法:**

```go
// 从配置文件读取
type VDIConfig struct {
    Servers []VDIServerConfig `yaml:"servers"`
}
```

### 4.3 ❌ 忽略错误处理

```go
// 错误示例
func (s *vmServiceImpl) DeleteVM(id string) {
    s.db.Delete(&VDIVirtualMachine{}, id)
    // 没有错误处理
}
```

**✅ 正确做法:**

```go
func (s *vmServiceImpl) DeleteVM(ctx context.Context, id string) error {
    if err := s.db.Delete(&VDIVirtualMachine{}, id).Error; err != nil {
        return fmt.Errorf("failed to delete VM: %w", err)
    }
    return nil
}
```

### 4.4 ❌ 前端直接使用axios

```typescript
// 错误示例
const data = await axios.get('/api/vdi/vm/list')
```

**✅ 正确做法:**

```typescript
// 使用封装的API函数
import { vmApi } from '@/lib/vdiApi';
const result = await vmApi.list({ page: 1, pageSize: 10 });
```

## 5. 配置管理

### 5.1 VDI服务器配置

```yaml
# configs/config.yaml
vdi:
  servers:
    - name: "生产环境"
      endpoint: "https://vdi-prod.example.com"
      username: "admin"
      password: "${VDI_PASSWORD_PROD}"  # 环境变量
      tenant_id: 0
      enabled: true
    - name: "测试环境"
      endpoint: "https://vdi-test.example.com"
      username: "admin"
      password: "${VDI_PASSWORD_TEST}"
      tenant_id: 1
      enabled: true

  # 缓存配置
  cache:
    vm_ttl: 300          # 虚拟机缓存时间（秒）
    resource_ttl: 600    # 资源组缓存时间

  # 超时配置
  timeout:
    connect: 10s         # 连接超时
    request: 30s         # 请求超时
```

### 5.2 配置结构体

```go
// internal/config/vdi_config.go
type VDIConfig struct {
    Servers []VDIServerConfig `yaml:"servers" validate:"required"`
    Cache   VDICacheConfig    `yaml:"cache"`
    Timeout VDITimeoutConfig  `yaml:"timeout"`
}

type VDIServerConfig struct {
    Name     string `yaml:"name" validate:"required"`
    Endpoint string `yaml:"endpoint" validate:"required,url"`
    Username string `yaml:"username" validate:"required"`
    Password string `yaml:"password" validate:"required"`
    TenantID int    `yaml:"tenant_id"`
    Enabled  bool   `yaml:"enabled"`
}
```

## 6. 测试模式

### 6.1 单元测试

```go
// internal/services/vdi/vm_service_test.go
func TestVMService_CreateVM(t *testing.T) {
    // 使用 mock VDI client
    mockClient := &MockVDIClient{}
    service := NewVMService(db, cache, mockClient)

    // 测试用例
    req := &CreateVMRequest{
        Name:       "test-vm",
        ResourceID: "res-1",
    }

    vm, err := service.CreateVM(context.Background(), req)

    assert.NoError(t, err)
    assert.NotNil(t, vm)
    assert.Equal(t, "test-vm", vm.Name)
}
```

### 6.2 集成测试

```go
// tests/integration/vdi_test.go
func TestVDIIntegration_Authenticate(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    client := NewVDIClient(testConfig)
    token, err := client.Authenticate(context.Background())

    assert.NoError(t, err)
    assert.NotEmpty(t, token)
}
```
