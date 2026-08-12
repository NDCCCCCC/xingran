# Phase 09: 后端代码优化 - Pattern Map

**映射时间:** 2026-04-27
**分析文件数:** 11 个新/修改文件
**找到类比:** 9 / 11

## File Classification

| 新/修改文件 | 角色 | 数据流 | 最近类比 | 匹配质量 |
|------------|------|--------|----------|----------|
| `internal/services/dashboard_service.go` | service | CRUD | `internal/services/system/dashboard_service.go` | 待删除 - 死代码 |
| `internal/services/settings_service.go` | service | CRUD | `internal/services/system/settings_service.go` | 待删除 - 死代码 |
| `internal/core/core.go` | config | request-response | `internal/core/core.go` | exact - 字段清理 |
| `internal/api/v1/ws_notice_handler.go` | handler | websocket | `internal/api/v1/ws_notice_handler.go` | exact - 安全修复 |
| `internal/scheduler/cron.go` | service | event-driven | `internal/scheduler/cron.go` | exact - 并发安全已实现 |
| `internal/api/v1/auth.go` | handler | request-response | `internal/api/v1/auth.go` | exact - 错误日志添加 |
| `internal/api/v1/ws_notice_handler_test.go` | test | unit | `internal/core/security/password_test.go` | pattern-match |
| `internal/scheduler/cron_test.go` | test | unit | `internal/scheduler/ad_sync_tasks_test.go` | exact |
| `internal/api/v1/auth_test.go` | test | unit | `internal/core/security/password_test.go` | pattern-match |

## Pattern Assignments

### 1. 死代码删除验证模式 (CODE-02a)

**目标文件:** `internal/services/system/dashboard_service.go`, `internal/services/system/settings_service.go`

**验证模式 (bash 工具链):**
```bash
# Step 1: 检查 dashboard_service 外部引用
grep -r "DashboardService" --include="*.go" internal/ | grep -v "internal/services/system/dashboard_service.go"

# Step 2: 检查 settings_service 外部引用
grep -r "SettingsService" --include="*.go" internal/ | grep -v "internal/services/system/settings_service.go"

# Step 3: 验证构建
go build ./...

# Step 4: 运行测试
go test ./...
```

**源位置:** `.planning/phases/09-backend-cleanup/09-RESEARCH.md:238-251`

**删除确认标准:**
- grep 零外部引用（排除自身文件）
- `go build ./...` 构建成功
- `go test ./...` 测试通过

---

### 2. Core 结构渐进式清理模式 (CODE-02b)

**目标文件:** `internal/core/core.go`

**Core 字段分组 (按依赖关系):**

**组 1 - 独立设备相关字段 (无内部依赖):**
- `DeviceManager *device.Manager` - 行 52
- `NetworkDeviceService *services.NetworkDeviceService` - (不存在, 可直接删除引用)

**组 2 - 设备监控相关 (有依赖关系):**
- `DeviceMonitorService *services.DeviceMonitorService` - 行 56
- `DeviceDiscoveryService *services.DeviceDiscoveryService` - 行 54
- `DeviceInfoCollectionService *services.DeviceInfoCollectionService` - 行 55

**组 3 - 缓存预热相关 (仅初始化期间使用):**
- `MetricsCacheService *MetricsCacheService` - 行 50
- `CaptchaService *CaptchaService` - 行 58
- `CaptchaBackgroundService *CaptchaBackgroundService` - 行 59
- `OperLogService services.OperLogService` - 行 60
- `TokenBlacklistService services.TokenBlacklistService` - 行 61

**组 4 - API 元数据:**
- `APIMetadata *config.APIMetadataConfig` - 行 73
- `APIEndpointService *services.APIEndpointService` - 行 74

**验证模式 (渐进式删除):**
```bash
# Step 1: 检查字段外部引用
grep -r "core\.DeviceManager" --include="*.go" | grep -v "^internal/core/core.go"
grep -r "core\.NetworkDeviceService" --include="*.go" | grep -v "^internal/core/core.go"

# Step 2: 删除字段
# [编辑 core.go, 删除对应行]

# Step 3: 验证构建
go build ./...

# Step 4: 提交
git commit -m "refactor(core): remove DeviceManager field"
```

**源位置:**
- Core 结构定义: `internal/core/core.go:43-75`
- 12 个死字段分析: `.planning/phases/09-backend-cleanup/09-CONTEXT.md:33-34`

**缓存预热服务转换模式:**
```go
// 当前: Core 字段存储
type Core struct {
    UserService     system.UserService
    RoleService     system.RoleService
    // ...
}

// 修改后: 本地变量 (在 Init() 方法内)
func (c *Core) Init() error {
    // 仅在初始化期间使用的服务转换为局部变量
    userService := system.NewUserService(c.GetDB(), c.Cache, c.PwdManager)
    roleService := system.NewRoleService(c.GetDB(), c.Cache)

    // 执行缓存预热
    warmUpServices := &warmUpServices{
        UserService: userService,
        RoleService: roleService,
    }
    warmUpServices.warmUpCache(context.Background())

    // 局部变量在方法结束后自动释放
    return nil
}
```

**源位置:** `internal/core/core.go:32-39` (warmUpServices 定义)

---

### 3. WebSocket CheckOrigin 安全修复模式 (CODE-02c)

**目标文件:** `internal/api/v1/ws_notice_handler.go`

**当前实现 (行 24-54):**
```go
CheckOrigin: func(r *http.Request) bool {
    if allowAll {
        return true
    }

    origin := r.Header.Get("Origin")
    if origin == "" {
        return true // 非浏览器客户端无Origin头
    }

    // 允许同源请求
    host := r.Header.Get("Host")
    if host == "" {
        host = r.Host
    }
    if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
        return true
    }

    // 允许 localhost（开发环境）和配置的域名
    if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
        return true
    }
    for _, allowed := range allowedOrigins {
        if origin == allowed || strings.HasPrefix(origin, allowed) {
            return true
        }
    }

    return false
}
```

**修复建议 (添加安全日志):**
```go
CheckOrigin: func(r *http.Request) bool {
    if allowAll {
        // 生产环境警告
        applogger.Warnf("WebSocket CheckOrigin 允许所有来源（开发模式）")
        return true
    }

    origin := r.Header.Get("Origin")
    if origin == "" {
        return true // 非浏览器客户端
    }

    // 允许同源请求
    host := r.Header.Get("Host")
    if host == "" {
        host = r.Host
    }
    if strings.HasPrefix(origin, "http://"+host) || strings.HasPrefix(origin, "https://"+host) {
        return true
    }

    // 允许 localhost（开发环境）
    if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1") {
        return true
    }

    // 检查白名单
    for _, allowed := range allowedOrigins {
        if origin == allowed || strings.HasPrefix(origin, allowed) {
            return true
        }
    }

    // 记录拒绝的来源（安全审计）
    applogger.Warnf("WebSocket 连接被拒绝: origin=%s, client_ip=%s",
        origin, r.RemoteAddr)
    return false
}
```

**源位置:** `internal/api/v1/ws_notice_handler.go:16-56`

**日志模式:** 使用 `applogger.Warnf` 记录安全相关事件

---

### 4. GlobalDeviceMonitorService 并发安全验证模式 (CODE-02c)

**目标文件:** `internal/scheduler/cron.go`

**已实现模式 (Phase 8 引入, 行 899-1035):**
```go
// GlobalDeviceMonitorService 全局设备监控服务
// 遵循 Go 最佳实践：使用 sync.RWMutex 保护全局变量
var (
    GlobalDeviceMonitorService        DeviceMonitorService
    GlobalDeviceMonitorServiceMu      sync.RWMutex
)

// SetDeviceMonitorService 设置设备监控服务（线程安全）
func SetDeviceMonitorService(service DeviceMonitorService) {
    GlobalDeviceMonitorServiceMu.Lock()
    defer GlobalDeviceMonitorServiceMu.Unlock()
    GlobalDeviceMonitorService = service
}

// GetDeviceMonitorService 获取设备监控服务（线程安全）
func GetDeviceMonitorService() DeviceMonitorService {
    GlobalDeviceMonitorServiceMu.RLock()
    defer GlobalDeviceMonitorServiceMu.RUnlock()
    return GlobalDeviceMonitorService
}
```

**源位置:** `internal/scheduler/cron.go:899-1035`

**验证任务:** 确保所有调用点都使用 mutex 保护的方法
- ✅ `SetDeviceMonitorService()` - 已加锁
- ✅ `GetDeviceMonitorService()` - 已加锁
- ⚠️ 需检查: 所有调用 `GetDeviceMonitorService()` 的地方是否正确处理 nil

**安全使用模式:**
```go
svc := GetDeviceMonitorService()
if svc == nil {
    return fmt.Errorf("device monitor service not initialized")
}
// 使用 svc...
```

**参考模式:** `.planning/phases/08-snmp-panic/08-CONTEXT.md` (并发安全模式)

---

### 5. 登录错误日志添加模式 (CODE-02c)

**目标文件:** `internal/api/v1/auth.go`

**当前实现 (行 409-432):**
```go
// recordLoginLog 记录登录日志
func recordLoginLog(c *gin.Context, core *core.Core, username string, status int, msg string) {
    clientIP := utils.GetClientIP(c)
    userAgent := c.Request.UserAgent()

    browser, os := parseUserAgent(userAgent)

    loginLog := models.LoginLog{
        Username:      username,
        IPAddr:        clientIP,
        LoginLocation: nil,
        Browser:       &browser,
        OS:            &os,
        Status:        status,
        Msg:           &msg,
        LoginTime:     time.Now(),
    }

    go func() {
        if err := core.DB.GetDB().Create(&loginLog).Error; err != nil {
            applogger.Warnf("记录登录日志失败: %v", err)
        }
    }()
}
```

**修复建议 (已有错误日志, 需验证日志级别):**
```go
// 当前已使用 applogger.Warnf, 符合要求
// 可能的改进: 添加更多上下文信息

go func() {
    if err := core.DB.GetDB().Create(&loginLog).Error; err != nil {
        applogger.Errorf("记录登录日志失败 (user: %s, ip: %s, status: %d): %v",
            username, clientIP, status, err)
    }
}()
```

**源位置:** `internal/api/v1/auth.go:409-432`

**日志级别选择:**
- `Errorf`: 关键操作失败 (登录日志记录失败)
- `Warnf`: 非关键操作失败 (当前使用)
- 建议升级为 Errorf 以便于监控和告警

---

## Test Pattern Assignments

### 6. WebSocket CheckOrigin 测试模式

**目标文件:** `internal/api/v1/ws_notice_handler_test.go` (新建)

**参考测试:** `internal/core/security/password_test.go`

**测试模式:**
```go
package v1

import (
    "net/http"
    "net/url"
    "testing"

    "github.com/stretchr/testify/assert"
)

// TestWebSocketUpgrader_CheckOrigin 测试 CheckOrigin 验证逻辑
func TestWebSocketUpgrader_CheckOrigin(t *testing.T) {
    tests := []struct {
        name          string
        allowedOrigins []string
        origin        string
        expected      bool
    }{
        {
            name:          "允许所有来源",
            allowedOrigins: []string{"*"},
            origin:        "http://evil.com",
            expected:      true,
        },
        {
            name:          "同源请求允许",
            allowedOrigins: []string{},
            origin:        "http://localhost:9000",
            expected:      true, // Host 匹配
        },
        {
            name:          "localhost 允许",
            allowedOrigins: []string{},
            origin:        "http://localhost:3000",
            expected:      true,
        },
        {
            name:          "白名单匹配",
            allowedOrigins: []string{"https://example.com"},
            origin:        "https://example.com",
            expected:      true,
        },
        {
            name:          "白名单不匹配拒绝",
            allowedOrigins: []string{"https://example.com"},
            origin:        "https://evil.com",
            expected:      false,
        },
        {
            name:          "空 Origin 允许 (非浏览器)",
            allowedOrigins: []string{},
            origin:        "",
            expected:      true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            upgrader := newWebSocketUpgrader(tt.allowedOrigins)

            req := &http.Request{
                Header: http.Header{},
                Host:   "localhost:9000",
            }
            if tt.origin != "" {
                req.Header.Set("Origin", tt.origin)
            }

            result := upgrader.CheckOrigin(req)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**测试框架:** Go testing + testify/assert

**源位置:** `internal/core/security/password_test.go:1-50` (测试结构参考)

---

### 7. GlobalDeviceMonitorService 并发测试模式

**目标文件:** `internal/scheduler/cron_test.go` (新建)

**参考测试:** `internal/scheduler/ad_sync_tasks_test.go`

**测试模式:**
```go
package scheduler

import (
    "sync"
    "testing"

    "github.com/stretchr/testify/assert"
)

// TestGlobalDeviceMonitorService_ConcurrentAccess 测试并发访问
func TestGlobalDeviceMonitorService_ConcurrentAccess(t *testing.T) {
    // 保存原有状态
    oldService := GlobalDeviceMonitorService
    defer func() {
        SetDeviceMonitorService(oldService)
    })

    // 创建 mock service
    mockService := &mockDeviceMonitorService{}
    SetDeviceMonitorService(mockService)

    // 并发读写
    done := make(chan bool, 100)
    for i := 0; i < 50; i++ {
        go func() {
            svc := GetDeviceMonitorService()
            assert.NotNil(t, svc)
            done <- true
        }()
    }

    for i := 0; i < 50; i++ {
        go func() {
            SetDeviceMonitorService(mockService)
            done <- true
        }()
    }

    // 等待所有 goroutine 完成
    for i := 0; i < 100; i++ {
        <-done
    }

    // 验证最终状态
    finalService := GetDeviceMonitorService()
    assert.NotNil(t, finalService)
}

// mockDeviceMonitorService mock 实现
type mockDeviceMonitorService struct{}

func (m *mockDeviceMonitorService) CheckDeviceStatus(ctx context.Context) error {
    return nil
}

// TestGlobalDeviceMonitorService_NilSafe 测试 nil 安全
func TestGlobalDeviceMonitorService_NilSafe(t *testing.T) {
    // 设置为 nil
    SetDeviceMonitorService(nil)

    // 获取应该返回 nil 而非 panic
    svc := GetDeviceMonitorService()
    assert.Nil(t, svc)
}
```

**测试框架:** Go testing + testify/assert + -race detector

**并发验证命令:**
```bash
go test -race -run TestGlobalDeviceMonitorService_ConcurrentAccess ./internal/scheduler/
```

**源位置:** `internal/scheduler/ad_sync_tasks_test.go:42-74` (并发测试模式参考)

---

### 8. 登录错误日志测试模式

**目标文件:** `internal/api/v1/auth_test.go` (新建)

**参考测试:** `internal/core/security/password_test.go`

**测试模式:**
```go
package v1

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/xingran-next/xingran-go-backend/internal/core"
)

// TestLogin_ErrorLogging 测试登录错误日志记录
func TestLogin_ErrorLogging(t *testing.T) {
    // 设置 Gin 为测试模式
    gin.SetMode(gin.TestMode)

    // 创建测试路由
    router := gin.New()
    router.POST("/login", login(&core.Core{}))

    // 测试登录失败场景
    loginReq := LoginRequest{
        Username: "nonexistent",
        Password: "wrongpassword",
    }

    body, _ := json.Marshal(loginReq)
    req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // 验证响应
    assert.Equal(t, http.StatusBadRequest, w.Code)

    // 验证日志记录（需要捕获日志输出或使用 mock logger）
    // 这里需要实际的日志捕获机制或依赖集成的日志系统
}

// TestRecordLoginLog_ErrorHandling 测试登录日志错误处理
func TestRecordLoginLog_ErrorHandling(t *testing.T) {
    // 测试日志记录失败时的处理
    // 需要 mock Core.DB.GetDB() 返回错误

    // 这里需要更多测试基础设施支持
    // 可以考虑使用 testify/mock 或 gomock
}
```

**测试框架:** Go testing + testify/assert + httptest

**日志捕获模式:** 使用 buffer 重定向 logger 输出进行验证

**源位置:** `internal/core/security/password_test.go:9-33` (基础测试结构参考)

---

## Shared Patterns

### 1. 死代码验证流程 (所有删除操作)

**模式:** grep 验证 + go build 验证 + go test 验证

**应用到:**
- DashboardService 删除
- SettingsService 删除
- Core 字段删除 (每组)

**验证命令序列:**
```bash
# 1. 检查外部引用
grep -r "SymbolName" --include="*.go" internal/ | grep -v "self_file"

# 2. 构建验证
go build ./...

# 3. 测试验证
go test ./...

# 4. 竞态检测 (可选)
go test -race ./...
```

**源位置:** `.planning/phases/09-backend-cleanup/09-RESEARCH.md:238-251`

---

### 2. 日志记录模式 (所有错误处理)

**模式:** 使用 `applogger` 包统一日志记录

**日志级别选择:**
- `Errorf`: 关键操作失败 (登录日志、数据库连接)
- `Warnf`: 非关键操作失败 (可降级的服务)
- `Infof`: 正常操作记录 (服务启动、配置加载)

**日志格式:**
```go
applogger.Errorf("操作失败 (context_key: %s): %v", value, err)
```

**应用到:**
- WebSocket CheckOrigin 拒绝日志
- 登录日志记录失败日志
- Core 初始化失败日志

**源位置:** `internal/api/v1/auth.go:428-430` (当前日志使用)

---

### 3. 并发安全模式 (全局变量访问)

**模式:** RWMutex 保护全局变量

**应用到:**
- GlobalDeviceMonitorService (已实现)
- GlobalDeviceInfoCollectionService (已实现)

**实现模板:**
```go
var (
    GlobalService    ServiceType
    GlobalServiceMu  sync.RWMutex
)

func SetGlobalService(service ServiceType) {
    GlobalServiceMu.Lock()
    defer GlobalServiceMu.Unlock()
    GlobalService = service
}

func GetGlobalService() ServiceType {
    GlobalServiceMu.RLock()
    defer GlobalServiceMu.RUnlock()
    return GlobalService
}
```

**源位置:** `internal/scheduler/cron.go:899-1035` (已验证实现)

---

### 4. 测试文件组织模式

**模式:** 测试文件与源文件同目录, 命名为 `xxx_test.go`

**测试结构:**
```go
package package_name

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

// TestFunctionName 测试功能
func TestFunctionName(t *testing.T) {
    // Arrange
    // Act
    // Assert
}

// BenchmarkFunctionName 基准测试
func BenchmarkFunctionName(b *testing.B) {
    // 测试性能
}
```

**应用到:**
- `ws_notice_handler_test.go`
- `cron_test.go`
- `auth_test.go`

**参考文件:**
- `internal/core/security/password_test.go`
- `internal/scheduler/ad_sync_tasks_test.go`

---

## No Analog Found

| 文件 | 角色 | 数据流 | 原因 |
|------|------|--------|------|
| (无) | - | - | 所有文件都找到了类比或参考模式 |

## Metadata

**Analog 搜索范围:**
- `internal/core/core.go` - Core 结构定义和初始化
- `internal/api/v1/ws_notice_handler.go` - WebSocket 处理模式
- `internal/scheduler/cron.go` - 全局变量并发安全模式
- `internal/api/v1/auth.go` - 认证处理和日志模式
- `internal/core/security/password_test.go` - 单元测试模式
- `internal/scheduler/ad_sync_tasks_test.go` - 并发测试模式

**文件扫描数:** 15
**模式提取日期:** 2026-04-27

**关键技术决策:**
1. 死代码删除必须经过三重验证 (grep + build + test)
2. Core 字段清理按依赖关系分组, 每组独立提交
3. 并发安全使用 Phase 8 验证的 RWMutex 模式
4. 测试覆盖核心安全修复路径
