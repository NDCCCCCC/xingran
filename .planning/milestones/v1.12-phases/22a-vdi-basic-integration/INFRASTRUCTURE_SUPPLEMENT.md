# Phase 22A 基础设施补充计划

**目标**: 基于**现有基础设施**补充 VDI 模块所需的生产级要素

**原则**: 复用现有实现，避免重复开发

---

## 1. 现有基础设施清单

### ✅ 已有（可复用）

| 组件 | 位置 | 说明 |
|------|------|------|
| **错误处理** | `pkg/errors/errors.go` | 完整的 `AppError` 系统，支持错误码、HTTP 状态映射 |
| **操作日志** | `pkg/middleware/oper_log.go` | `OperLogMiddleware` 自动记录，支持异步写入 |
| **审计日志模型** | `internal/models/rpa/audit_log.go` | `AuditLog` 结构，含 OldValue/NewValue JSONB |
| **测试框架** | 标准库 + testify + httptest | `auth_integration_test.go` 提供参考模式 |
| **密码加密** | `internal/services/addomain/utils.go` | AES-128-GCM 加密/解密函数 |

### ❌ 缺失（需补充）

| 组件 | 缺失内容 | 优先级 |
|------|----------|--------|
| **VDI Mock Server** | 模拟深信服 VDI API 响应 | P0 |
| **VDI 专用错误码** | 扩展 `pkg/errors` 添加 VDI 特定错误 | P0 |
| **审计日志服务** | VDI 审计日志记录器 | P1 |
| **VDI Client 管理** | 连接池/单例模式 | P1 |
| **Prometheus Metrics** | VDI 操作指标暴露 | P2 |

---

## 2. P0 必须补充

### 2.1 VDI Mock Server

**目的**: 模拟深信服 VDI API，无需真实 VDI 服务器

**位置**: `internal/services/vdi/mock_server.go`

**实现**（基于项目 httptest 模式）:

```go
package vdi

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "sync"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/require"
)

// VDIAPIMock 模拟 VDI API 服务器
type VDIAPIMock struct {
    server    *httptest.Server
    router    *gin.Engine
    
    // Mock 数据存储
    vms       map[string]*VDIVMDetail
    vmsMu     sync.RWMutex
    
    // 模拟状态
    shouldFail bool
    callCount  map[string]int
    callCountMu sync.RWMutex
}

// NewVDIAPIMock 创建 Mock VDI 服务器
func NewVDIAPIMock(t *require.Assertions) *VDIAPIMock {
    gin.SetMode(gin.TestMode)
    
    mock := &VDIAPIMock{
        router:    gin.New(),
        vms:       make(map[string]*VDIVMDetail),
        callCount: make(map[string]int),
    }
    
    mock.setupRoutes()
    mock.server = httptest.NewServer(mock.router)
    
    return mock
}

// setupRoutes 设置 Mock API 路由（完整覆盖 VDI API）
func (m *VDIAPIMock) setupRoutes() {
    api := m.router.Group("/v1")
    
    // 认证
    api.POST("/auth/tokens", m.mockAuth)
    
    // 虚拟机操作
    api.POST("/vm", m.mockCreateVM)
    api.DELETE("/vm", m.mockDeleteVM)
    api.POST("/vm/operate", m.mockOperateVM)
    api.POST("/vm/config_ip", m.mockConfigIP)
    api.PUT("/vm/:id/name", m.mockRenameVM)
    api.POST("/vm/:id/bind_user", m.mockBindUser)
    api.GET("/vm/:id", m.mockGetVM)
    api.GET("/resource/:id/vms", m.mockListVMs)
    api.GET("/vm/:id/available_users", m.mockAvailableUsers)
}

// mockAuth 模拟认证
func (m *VDIAPIMock) mockAuth(c *gin.Context) {
    m.incrementCall("auth")
    
    if m.shouldFail {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error_code":    2001,
            "error_message": "认证失败",
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "error_code":    0,
        "error_message": "",
        "data": gin.H{
            "token": gin.H{
                "tenant_id":  0,
                "auth_token": "mock-auth-token-" + randomString(32),
            },
        },
    })
}

// mockCreateVM 模拟创建虚拟机
func (m *VDIAPIMock) mockCreateVM(c *gin.Context) {
    m.incrementCall("create_vm")
    
    var req CreateVMRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error_code":    1001,
            "error_message": "参数错误",
        })
        return
    }
    
    vm := &VDIVMDetail{
        VMID:       "mock-vm-" + randomString(8),
        Name:       req.Name,
        ResourceID: req.ResourceID,
        Status:     0,
        PowerState: "stopped",
        CPU:        req.CPU,
        Memory:     req.Memory,
        Disk:       req.Disk,
    }
    
    m.vmsMu.Lock()
    m.vms[vm.VMID] = vm
    m.vmsMu.Unlock()
    
    c.JSON(http.StatusOK, gin.H{
        "error_code":    0,
        "error_message": "",
        "data":          vm,
    })
}

// mockDeleteVM 模拟删除虚拟机
func (m *VDIAPIMock) mockDeleteVM(c *gin.Context) {
    m.incrementCall("delete_vm")
    
    var req DeleteVMRequest
    c.ShouldBindJSON(&req)
    
    for _, vmID := range req.VMIDs {
        m.vmsMu.Lock()
        delete(m.vms, vmID)
        m.vmsMu.Unlock()
    }
    
    c.JSON(http.StatusOK, gin.H{
        "error_code": 0,
        "error_message": "",
        "data": req.VMIDs,
    })
}

// ... 其他 API 方法实现类似

// GetEndpoint 返回 Mock 服务器地址
func (m *VDIAPIMock) GetEndpoint() string {
    return m.server.URL
}

// GetVMCount 获取 VM 数量
func (m *VDIAPIMock) GetVMCount() int {
    m.vmsMu.RLock()
    defer m.vmsMu.RUnlock()
    return len(m.vms)
}

// SetShouldFail 设置是否模拟失败
func (m *VDIAPIMock) SetShouldFail(shouldFail bool) {
    m.shouldFail = shouldFail
}

// GetCallCount 获取 API 调用次数
func (m *VDIAPIMock) GetCallCount(apiName string) int {
    m.callCountMu.RLock()
    defer m.callCountMu.RUnlock()
    return m.callCount[apiName]
}

// incrementCall 增加 API 调用计数
func (m *VDIAPIMock) incrementCall(apiName string) {
    m.callCountMu.Lock()
    defer m.callCountMu.Unlock()
    m.callCount[apiName]++
}

// Close 关闭 Mock 服务器
func (m *VDIAPIMock) Close() {
    if m.server != nil {
        m.server.Close()
    }
}

func randomString(length int) string {
    // 简单实现
    return "random"
}
```

**测试示例**:

```go
// internal/services/vdi/vdi_client_test.go
func TestVDIClient_Authentication(t *testing.T) {
    mock := NewVDIAPIMock(require.New(t))
    defer mock.Close()
    
    cfg := config.VDIServerConfig{
        Endpoint: mock.GetEndpoint(),
        Username: "admin",
        Password: "password",
    }
    
    client := NewVDIClient(nil, "test-server", cfg)
    token, err := client.Authenticate(context.Background())
    
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
    assert.Equal(t, 1, mock.GetCallCount("auth"))
}

func TestVDIClient_CreateVM(t *testing.T) {
    mock := NewVDIAPIMock(require.New(t))
    defer mock.Close()
    
    // ... 测试创建虚拟机
}
```

### 2.2 VDI 专用错误码

**位置**: `pkg/errors/errors.go`（末尾追加）

```go
// ============================================================================
// VDI 模块错误码
// ============================================================================

// VDI 错误码定义
const (
    CodeVDIServerNotFound     ErrorCode = 54001 // VDI 服务器不存在
    CodeVDIServerExists       ErrorCode = 54002 // VDI 服务器已存在
    CodeVDIApiFailed          ErrorCode = 54003 // VDI API 调用失败
    CodeVDIAuthFailed          ErrorCode = 54004 // VDI 认证失败
    CodeVDITokenExpired        ErrorCode = 54005 // VDI Token 过期
    CodeVMNotFound             ErrorCode = 54006 // 虚拟机不存在
    CodeVMExists               ErrorCode = 54007 // 虚拟机已存在
    CodeVMOperationFailed      ErrorCode = 54008 // 虚拟机操作失败
    CodeVMInconsistentState    ErrorCode = 54009 // 虚拟机状态不一致
)

func (c ErrorCode) DefaultHTTPStatus() int {
    switch {
    case c >= 54001 && c <= 54100:
        return http.StatusInternalServerError // VDI 错误默认 500
    default:
        return http.StatusInternalServerError
    }
}

func (c ErrorCode) DefaultMessage() string {
    switch c {
    case CodeVDIServerNotFound:
        return "VDI 服务器不存在"
    case CodeVDIServerExists:
        return "VDI 服务器已存在"
    case CodeVDIApiFailed:
        return "VDI API 调用失败"
    case CodeVDIAuthFailed:
        return "VDI 认证失败"
    case CodeVDITokenExpired:
        return "VDI Token 过期"
    case CodeVMNotFound:
        return "虚拟机不存在"
    case CodeVMExists:
        return "虚拟机已存在"
    case CodeVMOperationFailed:
        return "虚拟机操作失败"
    case CodeVMInconsistentState:
        return "虚拟机状态不一致"
    default:
        return "服务器内部错误"
    }
}

// VDI 便捷错误函数
func VDIServerNotFound(id string) *AppError {
    return New(CodeVDIServerNotFound, fmt.Sprintf("VDI 服务器不存在: %s", id))
}

func VDIServerExists() *AppError {
    return New(CodeVDIServerExists, CodeVDIServerExists.DefaultMessage())
}

func VDIApiFailed(err error) *AppError {
    return Wrap(err, CodeVDIApiFailed, CodeVDIApiFailed.DefaultMessage())
}

func VDIAuthFailed() *AppError {
    return New(CodeVDIAuthFailed, CodeVDIAuthFailed.DefaultMessage())
}

func VMNotFound(vmID string) *AppError {
    return New(CodeVMNotFound, fmt.Sprintf("虚拟机不存在: %s", vmID))
}

func VMOperationFailed(operation string, err error) *AppError {
    return Wrap(err, CodeVMOperationFailed, fmt.Sprintf("虚拟机%s失败", operation))
}

func VMInconsistentState(vmID string) *AppError {
    return New(CodeVMInconsistentState, fmt.Sprintf("虚拟机状态不一致: %s", vmID))
}
```

---

## 3. P1 强烈建议

### 3.1 VDI 审计日志服务

**位置**: `internal/services/vdi/audit_service.go`

```go
package vdi

import (
    "context"
    "time"
    
    "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
    "gorm.io/gorm"
)

// AuditService VDI 审计日志服务
type AuditService struct {
    db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
    return &AuditService{db: db}
}

// LogVMOperation 记录虚拟机操作
func (s *AuditService) LogVMOperation(ctx context.Context, op AuditOperation) error {
    log := &rpa.AuditLog{
        ResourceType: rpa.ResourceTypeTask, // 复用 RPA 的类型，或新增 VDI 专用
        ResourceID:   op.ResourceID,
        Action:       op.Action,
        OldValue:     op.OldValue,
        NewValue:     op.NewValue,
        OperatorID:   op.OperatorID,
        OperatorName: op.OperatorName,
        IPAddress:    op.IPAddress,
        Result:       rpa.AuditResultSuccess,
        CreatedAt:    time.Now(),
    }
    
    if op.Error != nil {
        log.Result = rpa.AuditResultFailed
        log.ErrorMessage = op.Error.Error()
    }
    
    return s.db.WithContext(ctx).Create(log).Error
}

// VDI 审计操作类型
type VDIAction string

const (
    VDIActionCreateVM      VDIAction = "create_vm"
    VDIActionDeleteVM      VDIAction = "delete_vm"
    VDIActionStartVM       VDIAction = "start_vm"
    VDIActionStopVM        VDIAction = "stop_vm"
    VDIActionRestartVM     VDIAction = "restart_vm"
    VDIActionConfigIP      VDIAction = "config_ip"
    VDIActionRenameVM      VDIAction = "rename_vm"
    VDIActionBindUser      VDIAction = "bind_user"
    VDIActionUnbindUser    VDIAction = "unbind_user"
    VDIActionSyncVM        VDIAction = "sync_vm"
)

type VMOperation struct {
    ResourceID   string
    Action       VDIAction
    OldValue     map[string]interface{}
    NewValue     map[string]interface{}
    OperatorID   string
    OperatorName string
    IPAddress    string
    Error        error
}
```

### 3.2 VDI Client 管理器

**位置**: `internal/services/vdi/client_manager.go`

```go
package vdi

import (
    "sync"
    
    "github.com/xingran-next/xingran-go-backend/internal/config"
)

// ClientManager VDI 客户端管理器（单例模式）
type ClientManager struct {
    clients sync.Map // serverID -> *VDIClient
}

var globalClientManager *ClientManager
var clientManagerOnce sync.Once

// GetClientManager 获取全局客户端管理器
func GetClientManager() *ClientManager {
    clientManagerOnce.Do(func() {
        globalClientManager = &ClientManager{}
    })
    return globalClientManager
}

// GetOrCreateClient 获取或创建 VDI 客户端
func (m *ClientManager) GetOrCreateClient(
    db *gorm.DB,
    serverID string,
    cfg config.VDIServerConfig,
) VDIClient {
    // 先尝试从缓存获取
    if client, ok := m.clients.Load(serverID); ok {
        return client.(VDIClient)
    }
    
    // 创建新客户端
    client := NewVDIClient(db, serverID, cfg)
    
    // 存入缓存（如果其他人已创建，使用已存在的）
    actual, _ := m.clients.LoadOrStore(serverID, client)
    
    return actual.(VDIClient)
}

// RemoveClient 移除客户端（用于测试或服务器删除）
func (m *ClientManager) RemoveClient(serverID string) {
    m.clients.Delete(serverID)
}
```

---

## 4. 集成到 Phase 22A 计划

### 4.1 修改 Wave 1 (22-01-PLAN.md)

在 `Task 1: 创建VDI数据模型` 后添加：

```markdown
### Task 1.5: 添加 VDI 专用错误码

**Files**: `pkg/errors/errors.go`

**Action**:
1. 在文件末尾追加 VDI 错误码定义（见 INFRASTRUCTURE_SUPPLEMENT.md §2.2）
2. 添加便捷错误函数
3. 更新 ErrorCode 枚举

**Verify**:
```bash
go build ./pkg/errors/
```

**Done**:
- [ ] VDI 错误码定义完成
- [ ] 错误码 HTTP 状态映射正确
- [ ] 编译无错误
```

### 4.2 修改 Wave 2 (22-02-PLAN.md)

在 `Task 4: 实现VDIClient` 后添加：

```markdown
### Task 2.5: 创建 VDI Mock Server

**Files**: `internal/services/vdi/mock_server.go`

**Action**:
1. 创建 VDIAPIMock 结构（见 INFRASTRUCTURE_SUPPLEMENT.md §2.1）
2. 实现所有 11 个 VDI API 端点的 Mock 响应
3. 添加测试辅助方法（GetEndpoint, GetCallCount, SetShouldFail）

**Verify**:
```bash
go test -v -run TestVDIAPIMock ./internal/services/vdi/
```

**Done**:
- [ ] Mock Server 实现完成
- [ ] 所有 VDI API 端点有 Mock 响应
- [ ] 测试辅助方法可用
- [ ] 单元测试通过

### Task 2.6: 编写 VDI Client 单元测试

**Files**: `internal/services/vdi/vdi_client_test.go`

**Action**:
1. 使用 VDIAPIMock 测试认证
2. 使用 VDIAPIMock 测试虚拟机操作
3. 测试错误处理和重试逻辑
4. 目标覆盖率 >80%

**Verify**:
```bash
go test -v -cover ./internal/services/vdi/
```

**Done**:
- [ ] 认证测试通过
- [ ] 虚拟机操作测试通过
- [ ] 错误处理测试通过
- [ ] 测试覆盖率 >80%
```

### 4.3 修改 Wave 3 (22-03-PLAN.md)

在 `Task 4: 实现VDI服务器服务` 后添加：

```markdown
### Task 3.5: 实现 VDI 审计日志服务

**Files**: `internal/services/vdi/audit_service.go`

**Action**:
1. 创建 AuditService 结构（见 INFRASTRUCTURE_SUPPLEMENT.md §3.1）
2. 实现 LogVMOperation 方法
3. 在所有 VM 操作中添加审计点

**集成示例**:
```go
func (s *vmServiceImpl) DeleteVM(ctx context.Context, ids []string) error {
    // 记录操作开始
    audit.LogVMOperation(ctx, VMOperation{
        ResourceID: strings.Join(ids, ","),
        Action:     VDIActionDeleteVM,
        OldValue:   map[string]interface{}{"ids": ids},
        OperatorID: getOperatorID(ctx),
    })
    
    // ... 原有逻辑
}
```

**Verify**:
```bash
go test -v -run TestAuditService ./internal/services/vdi/
```

**Done**:
- [ ] AuditService 实现完成
- [ ] 所有 VM 操作添加审计点
- [ ] 审计日志正确写入数据库
```

在 `Task 4: 实现VDI服务器服务` 后添加：

```markdown
### Task 3.6: 实现 VDI Client 管理器

**Files**: `internal/services/vdi/client_manager.go`

**Action**:
1. 创建 ClientManager 结构（见 INFRASTRUCTURE_SUPPLEMENT.md §3.2）
2. 实现单例模式
3. 更新 Router 使用 ClientManager

**Router 修改**:
```go
func SetupVMRouter(r *gin.RouterGroup, core *core.Core) {
    // 使用 ClientManager
    clientManager := vdi.GetClientManager()
    
    // 从配置获取第一个服务器
    if len(core.Config.VDI.Servers) == 0 {
        return
    }
    
    serverCfg := core.Config.VDI.Servers[0]
    vdiClient := clientManager.GetOrCreateClient(core.GetDB(), "default", serverCfg)
    
    // ... 原有逻辑
}
```

**Verify**:
```bash
go test -v -run TestClientManager ./internal/services/vdi/
```

**Done**:
- [ ] ClientManager 实现完成
- [ ] 单例模式工作正常
- [ ] Router 正确使用 ClientManager
```

### 4.4 修改 Wave 4 (22-04-PLAN.md)

在路由配置中添加操作日志中间件支持：

```markdown
### Task 5 补充: 配置 VDI 操作日志

**Files**: `pkg/middleware/oper_log.go`, `internal/api/router.go`

**Action**:
1. 在 `DefaultOperLogConfig()` 的 LogPaths 中添加:
   ```go
   "/vdi/vm",
   "/vdi/servers",
   ```

2. VDI Handler 中设置操作日志信息:
   ```go
   func (h *VMHandler) Create(c *gin.Context) {
       middleware.SetOperLogInfo(c, "创建虚拟机", 1, "创建")
       // ... 原有逻辑
   }
   ```

**Verify**:
```bash
# 启动服务器，创建虚拟机，检查 sys_oper_log 表
```

**Done**:
- [ ] VDI 路径添加到操作日志配置
- [ ] VDI Handler 设置操作日志信息
- [ ] 操作日志正确记录
```

---

## 5. P2 可选优化（不阻塞执行）

### 5.1 Prometheus Metrics

**位置**: `internal/api/v1/vdi/metrics.go`

```go
package vdi

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    vmOperationsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "vdi_vm_operations_total",
            Help: "Total number of VM operations",
        },
        []string{"operation", "server_id", "status"},
    )
    
    vdiAPIDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "vdi_api_duration_seconds",
            Help:    "VDI API call duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"endpoint"},
    )
)

// recordVMOperation 记录虚拟机操作指标
func recordVMOperation(operation, serverID string, status string) {
    vmOperationsTotal.WithLabelValues(operation, serverID, status).Inc()
}
```

### 5.2 健康检查端点

**位置**: `internal/api/v1/vdi/health_handler.go`

```go
func (h *VDIHandler) HealthCheck(c *gin.Context) {
    serverID := c.Query("server_id")
    
    // 检查数据库连接
    if err := h.db.DB().Ping(); err != nil {
        c.JSON(503, gin.H{"status": "down", "reason": "database"})
        return
    }
    
    // 检查 VDI API 连接
    if err := h.serverService.TestConnection(c.Request.Context(), serverID); err != nil {
        c.JSON(503, gin.H{"status": "down", "reason": "vdi_api", "error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"status": "up"})
}
```

---

## 6. 执行顺序

### Week 1: 基础设施

```
Day 1-2: 错误码 + Mock Server
├── 修改 pkg/errors/errors.go
├── 创建 internal/services/vdi/mock_server.go
└── 编写基础单元测试

Day 3-4: 审计日志 + ClientManager
├── 创建 internal/services/vdi/audit_service.go
├── 创建 internal/services/vdi/client_manager.go
└── 集成到现有计划

Day 5: 测试验证
├── 运行完整测试套件
├── 验证覆盖率 >80%
└── 修复问题
```

### Week 2: 执行 Phase 22A

```
/gsd-execute-phase 22a-vdi-basic-integration
```

---

## 7. 验证清单

**执行前**:
- [ ] VDI Mock Server 可用
- [ ] VDI 错误码定义完成
- [ ] 审计日志服务就绪
- [ ] ClientManager 实现完成

**执行中**:
- [ ] 每个 Wave 完成后运行 `go test ./...`
- [ ] 每个 Wave 完成后验证 `go build ./...`
- [ ] 审计日志正确记录

**完成后**:
- [ ] 所有单元测试通过
- [ ] 测试覆盖率 >80%
- [ ] 操作日志可查询
- [ ] 审计日志可查询
- [ ] 前端可正常调用

---

## 8. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| Mock Server 与真实 API 不一致 | 测试无效 | 中 | Wave 2 完成后连接真实 VDI 验证 |
| 审计日志影响性能 | 响应变慢 | 低 | 异步写入，复用现有 OperLogMiddleware |
| ClientManager 并发问题 | 数据竞争 | 低 | 使用 sync.Map，经过充分测试 |

---

**补充完成时间**: 2026-05-25
**下一步**: 开始执行基础设施补充，然后执行 Phase 22A
