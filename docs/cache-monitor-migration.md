# 缓存监控模块迁移文档

## 迁移概述

本次迁移将缓存监控模块从旧的函数式Handler模式迁移到新的结构体Handler + Service层模式。

## 迁移时间

2026-01-22

## 迁移内容

### 源文件（已删除）

- `internal/api/v1/cache_handler.go` - 请求/响应类型定义
- `internal/api/v1/cache_config_handler.go` - 缓存配置管理Handler
- `internal/api/v1/cache_operation_handler.go` - 缓存操作Handler
- `internal/api/v1/cache_query_handler.go` - 缓存查询Handler
- `internal/api/v1/cache_stats_handler.go` - 缓存统计Handler
- `internal/api/v1/cache_test_handler.go` - 缓存测试Handler
- `internal/api/v1/cache_helpers.go` - 辅助函数

### 目标文件（新建）

- `internal/api/v1/monitor/cache_handler.go` - 整合的缓存Handler（结构体模式）
- `internal/api/v1/monitor/cache_router.go` - 缓存路由配置
- `internal/api/v1/monitor/cache_helpers.go` - 辅助函数（移至monitor包）

## 架构变更

### 旧架构（函数式Handler）

```go
// 函数式Handler
func getCacheList(core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 处理逻辑
    }
}

// 路由注册
func RegisterCacheRoutes(router *gin.RouterGroup, core *core.Core) {
    router.POST("/cache/list", getCacheList(core))
}
```

### 新架构（结构体Handler + Service）

```go
// 结构体Handler
type CacheHandler struct {
    core                 *core.Core
    db                   *gorm.DB
    cacheConfigService   *services.CacheConfigService
}

func NewCacheHandler(core *core.Core, cacheConfigService *services.CacheConfigService) *CacheHandler {
    return &CacheHandler{
        core:               core,
        db:                 core.DB.GetDB(),
        cacheConfigService: cacheConfigService,
    }
}

func (h *CacheHandler) GetCacheList(c *gin.Context) {
    // 处理逻辑
}

// 路由注册
func SetupCacheRouter(r *gin.RouterGroup, core *core.Core) {
    cacheConfigService := services.NewCacheConfigService(core.DB.GetDB())
    cacheHandler := NewCacheHandler(core, cacheConfigService)

    r.POST("/cache/list", cacheHandler.GetCacheList)
}
```

## API端点

所有缓存监控相关的API端点保持不变，路径均为 `/monitor/cache/*`：

### 缓存信息
- `POST /monitor/cache/list` - 获取缓存列表
- `GET /monitor/cache/:key` - 获取缓存详情

### 缓存操作
- `POST /monitor/cache/operate` - 操作缓存（get/set/del/exists/expire/ttl）
- `POST /monitor/cache/batch` - 批量操作缓存
- `POST /monitor/cache/clear` - 清空缓存

### 缓存统计
- `POST /monitor/cache/stats/list` - 获取缓存统计
- `POST /monitor/cache/monitor` - 获取缓存监控数据

### 缓存导出
- `POST /monitor/cache/export` - 导出缓存数据

### 缓存配置管理
- `GET /monitor/cache/config` - 获取缓存配置列表
- `PUT /monitor/cache/config` - 更新缓存配置
- `POST /monitor/cache/config/reload` - 重新加载缓存配置

### 测试端点
- `GET /monitor/cache/test` - 测试缓存功能

## 路由配置变更

### 主路由器 (`internal/api/router.go`)

```go
// 新增import
import monitorV1 "github.com/ruoyi-next/ruoyi-go-backend/internal/api/v1/monitor"

// 监控模块路由配置
monitor := r.Group("/monitor")
monitor.Use(middleware.JWTAuth(core.JWTManager))
{
    v1.SetupMonitorRouter(monitor, core)
    // 使用新架构：结构体Handler + Service层
    monitorV1.SetupCacheRouter(monitor, core)
}
```

### 监控路由器 (`internal/api/v1/monitor_router.go`)

移除了 `RegisterCacheRoutes` 函数的调用，因为缓存路由现在通过 `monitorV1.SetupCacheRouter` 注册。

## 依赖关系

### Service层

缓存Handler使用 `services.CacheConfigService` 来管理缓存配置：

```go
// 缓存配置服务已存在于 internal/services/cache_config_service.go
type CacheConfigService struct {
    db      *gorm.DB
    configs map[string]time.Duration
    mu      sync.RWMutex
}
```

### Core组件

Handler通过 `core.Core` 访问以下组件：
- `core.Cache` - 缓存接口
- `core.DB` - 数据库接口
- `core.DeptService` - 部门服务（用于测试）

## 测试验证

编译验证通过：
```bash
cd D:\CODE\ClaudeCode\ruoyi-go-backend
go build -o ruoyi-backend.exe ./cmd/main.go
```

生成的可执行文件：
- `ruoyi-backend.exe` (65MB)

## 注意事项

1. **API兼容性**：所有API端点路径保持不变，确保前端无需修改
2. **辅助函数**：`getCachesFromCacheWithLevel` 等辅助函数已移至 `monitor/cache_helpers.go`
3. **类型定义**：请求/响应类型定义已移至 `monitor/cache_handler.go`
4. **Service复用**：继续使用现有的 `services.CacheConfigService`，未创建新的Service

## 参考实现

本次迁移参考了已完成的网络模块（`internal/api/v1/network/`）的实现模式：
- 结构体Handler模式
- Service层分离
- 统一路由注册方式

## 后续工作

缓存监控模块已成功迁移至新架构，与系统中的其他模块（网络、操作日志、登录日志等）保持一致的架构风格。
