# Go interface{} 类型安全迁移指南

**版本**: 1.1
**日期**: 2026-01-31
**更新**: 2026-02-01
**状态**: ✅ operations 和 system 模块迁移已完成

---

## 目录

- [概述](#概述)
- [为什么需要迁移](#为什么需要迁移)
- [迁移示例](#迁移示例)
- [分步迁移指南](#分步迁移指南)
- [常见问题](#常见问题)
- [迁移检查清单](#迁移检查清单)

---

## 概述

本文档提供了将 Go Service 层从 `map[string]interface{}` 参数迁移到类型安全请求结构体的详细指南。

### 当前模式

```go
// Service 接口
type BuildingService interface {
    List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
}

// Handler
func (h *BuildingHandler) List(c *gin.Context) {
    var params map[string]interface{}
    c.ShouldBindJSON(&params)
    result, err := h.service.List(c.Request.Context(), params)
    // ...
}

// Service 实现
func (s *buildingService) applyFilters(query *gorm.DB, params map[string]interface{}) *gorm.DB {
    if name, ok := params["name"].(string); ok && name != "" {  // ❌ 类型断言
        query = query.Where("name LIKE ?", "%"+name+"%")
    }
    if status := extractIntParam(params, "status", -1); status >= 0 {
        query = query.Where("status = ?", status)
    }
    return query
}
```

### 目标模式

```go
// 请求结构体
type BuildingListRequest struct {
    PaginationParams  // 嵌入分页
    StatusRequest     // 嵌入状态筛选
    Name      string  `json:"name"`
    OrgID     string  `json:"orgId"`
}

// Service 接口
type BuildingService interface {
    List(ctx context.Context, req BuildingListRequest) (*PageResult, error)
}

// Handler
func (h *BuildingHandler) List(c *gin.Context) {
    var req requests.BuildingListRequest
    c.ShouldBindJSON(&req)  // ✅ 直接绑定到结构体
    result, err := h.service.List(c.Request.Context(), req)
    // ...
}

// Service 实现
func (s *buildingService) applyFilters(query *gorm.DB, req BuildingListRequest) *gorm.DB {
    if req.Name != "" {  // ✅ 无需类型断言
        query = query.Where("name LIKE ?", "%"+req.Name+"%")
    }
    if req.HasStatus() {  // ✅ 使用辅助方法
        query = query.Where("status = ?", req.GetStatus(0))
    }
    return query
}
```

---

## 为什么需要迁移

### 优点

1. **类型安全**: 编译时检查，减少运行时错误
2. **IDE 支持**: 自动补全、重命名、查找引用
3. **代码可读性**: 清晰的参数定义，无需猜测
4. **易于维护**: 添加新字段只需修改结构体
5. **文档化**: 请求结构体即文档

### 缺点

1. **初始工作量大**: 需要定义多个请求结构体
2. **学习曲线**: 团队需要适应新模式
3. **兼容性**: 需要确保不影响现有 API

---

## 迁移示例

已创建的示例文件：

| 文件 | 说明 |
|------|------|
| `internal/api/v1/operations/requests/common.go` | 通用请求结构体（分页、批量操作、状态筛选） |
| `internal/api/v1/operations/requests/building_requests.go` | 楼宇模块请求结构体 |
| `internal/api/v1/operations/requests/floor_requests.go` | 楼层模块请求结构体 |
| `internal/api/v1/operations/requests/workstation_requests.go` | 工位模块请求结构体 |
| `internal/services/operations/building_service_typesafe.go` | 类型安全的 Service 示例 |
| `internal/api/v1/operations/building_handler_typesafe.go` | 类型安全的 Handler 示例 |

---

## 分步迁移指南

### 步骤 1: 创建请求结构体

在 `internal/api/v1/{module}/requests/` 目录下创建请求结构体文件。

#### 基础结构

```go
// common.go - 通用请求结构体
package requests

import "github.com/xingran-next/xingran-go-backend/internal/constants"

// PaginationParams 分页参数
type PaginationParams struct {
    Current  int `json:"current"`
    PageSize int `json:"pageSize"`
}

func (p *PaginationParams) GetPagination() (current, pageSize int) {
    // ... 应用默认值和限制
}

// StatusRequest 状态筛选
type StatusRequest struct {
    Status *int `json:"status"`
}

func (s *StatusRequest) HasStatus() bool {
    return s.Status != nil
}

func (s *StatusRequest) GetStatus(defaultValue int) int {
    if s.Status != nil {
        return *s.Status
    }
    return defaultValue
}
```

#### 模块特定结构

```go
// building_requests.go
package requests

type BuildingListRequest struct {
    PaginationParams  // 嵌入分页
    StatusRequest     // 嵌入状态筛选
    Name       string `json:"name"`
    OrgID      string `json:"orgId"`
}
```

### 步骤 2: 更新 Service 接口

```go
// 旧接口
type BuildingService interface {
    List(ctx context.Context, params map[string]interface{}) (*PageResult, error)
}

// 新接口
type BuildingService interface {
    List(ctx context.Context, req requests.BuildingListRequest) (*PageResult, error)
}
```

### 步骤 3: 更新 Service 实现

```go
// 旧实现
func (s *buildingService) List(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
    query := s.db.WithContext(ctx).Model(&operations.OpsBuilding{})
    query = s.applyFilters(query, params)
    // ...
}

func (s *buildingService) applyFilters(query *gorm.DB, params map[string]interface{}) *gorm.DB {
    if name, ok := params["name"].(string); ok && name != "" {
        query = query.Where("name LIKE ?", "%"+name+"%")
    }
    if status := extractIntParam(params, "status", -1); status >= 0 {
        query = query.Where("status = ?", status)
    }
    return query
}

// 新实现
func (s *buildingService) List(ctx context.Context, req requests.BuildingListRequest) (*PageResult, error) {
    query := s.db.WithContext(ctx).Model(&operations.OpsBuilding{})
    query = s.applyFilters(query, req)
    // ...
}

func (s *buildingService) applyFilters(query *gorm.DB, req requests.BuildingRequest) *gorm.DB {
    if req.Name != "" {  // 无需类型断言
        query = query.Where("name LIKE ?", "%"+req.Name+"%")
    }
    if req.HasStatus() {  // 使用辅助方法
        query = query.Where("status = ?", req.GetStatus(0))
    }
    return query
}
```

### 步骤 4: 更新 Handler

```go
// 旧 Handler
func (h *BuildingHandler) List(c *gin.Context) {
    var params map[string]interface{}
    if !handleJSONBinding(c, &params) {
        return
    }
    result, err := h.service.List(c.Request.Context(), params)
    // ...
}

// 新 Handler
func (h *BuildingHandler) List(c *gin.Context) {
    var req requests.BuildingListRequest
    if !handleJSONBinding(c, &req) {
        return
    }
    result, err := h.service.List(c.Request.Context(), req)
    // ...
}
```

### 步骤 5: 删除旧代码

迁移完成后，可以删除：
- `extractIntParam`, `extractStringParam` 等辅助函数
- 旧的 Service 接口定义
- `_typesafe.go` 示例文件（重命名替换原文件）

---

## 常见问题

### Q1: 可选参数如何处理？

使用指针类型：

```go
type BuildingListRequest struct {
    Name   string  `json:"name"`   // 必填（空字符串表示不筛选）
    Status *int    `json:"status"` // 可选（nil 表示不筛选）
}

// 使用
if req.Status != nil {
    query = query.Where("status = ?", *req.Status)
}
```

### Q2: 如何复用模型作为请求？

对于简单的 CRUD，可以直接复用模型：

```go
// 创建/更新直接使用模型
func (h *BuildingHandler) Create(c *gin.Context) {
    var building operations.OpsBuilding  // 直接使用模型
    c.ShouldBindJSON(&building)
    // ...
}
```

### Q3: 如何处理默认值？

在请求结构体中提供方法：

```go
func (r *BuildingListRequest) GetPagination() (current, pageSize int) {
    if r.Current < 1 {
        current = constants.DefaultCurrent
    } else {
        current = r.Current
    }
    // ...
    return current, pageSize
}
```

### Q4: 如何兼容现有 API？

两种方式：

1. **渐进式迁移**：保留旧接口，新接口使用类型安全
```go
type BuildingService interface {
    // 旧接口（标记为废弃）
    // Deprecated: Use List with BuildingListRequest
    ListOld(ctx context.Context, params map[string]interface{}) (*PageResult, error)

    // 新接口
    List(ctx context.Context, req requests.BuildingListRequest) (*PageResult, error)
}
```

2. **适配器模式**：旧接口调用新接口
```go
func (s *buildingService) ListOld(ctx context.Context, params map[string]interface{}) (*PageResult, error) {
    req := convertMapToRequest(params)
    return s.List(ctx, req)
}
```

---

## 迁移检查清单

迁移每个模块时，确保：

- [ ] 创建请求结构体文件 `requests/{module}_requests.go`
- [ ] 定义所有需要的请求结构体
- [ ] 更新 Service 接口签名
- [ ] 更新 Service 实现
- [ ] 更新 Handler 方法
- [ ] 删除不再使用的辅助函数（`extractIntParam` 等）
- [ ] 运行测试确保功能正常
- [ ] 更新相关文档

---

## 待迁移模块清单

根据代码分析，以下模块需要迁移：

### operations 模块 ✅ **已完成**
- [x] Building
- [x] Floor
- [x] Workstation
- [x] ServerRoom
- [x] RoomDevice
- [x] InfoPoint
- [x] DedicatedLine
- [x] FloorPlanText
- [x] Door
- [x] Wall

### system 模块 ✅ **已完成**
- [x] User
- [x] Role
- [x] Menu
- [x] Department
- [x] Post
- [x] Dict
- [x] Config
- [x] Notice

### 其他模块（待迁移）
- [ ] workorder 模块
- [ ] duty 模块
- [ ] network 模块
- [ ] scheduler 模块
- [ ] knowledge 模块

### monitor 模块（Handler interface 模式迁移）
- [x] server_handler.go - 服务器监控（已迁移到 interface 模式）
- [x] oper_log_handler.go - 操作日志（已迁移到 interface 模式）
- [x] login_log_handler.go - 登录日志（已迁移到 interface 模式）
- [x] cache_handler.go - 缓存监控（已迁移到 interface 模式）

---

## 总结

类型安全的请求结构体模式是 Go 项目的最佳实践。通过本次迁移，我们将：

1. ✅ **operations 模块**已完成迁移（10 个子模块）
2. ✅ **system 模块**已完成迁移（8 个核心模块）
3. ✅ **monitor 模块** Handler 已迁移到 interface 模式
4. 消除所有类型断言带来的运行时风险
5. 提升代码可读性和可维护性
6. 改善 IDE 开发体验
7. 为后续功能迭代打下良好基础

### 已完成工作

- ✅ 创建通用请求结构体 (`internal/api/v1/operations/requests/common.go`)
- ✅ **operations 模块**: 10 个子模块完整迁移
- ✅ **system 模块**: 8 个核心模块完整迁移
- ✅ **monitor 模块**: Handler interface 模式迁移（4 个 handler）
  - ServerHandler - 服务器监控服务
  - OperLogHandler - 操作日志服务
  - LoginLogHandler - 登录日志服务
  - CacheHandler - 缓存监控服务

### 迁移文件位置

**operations 模块**:
- `internal/api/v1/operations/requests/` - 请求结构体目录
- `internal/services/operations/*_service.go` - 已迁移为类型安全版本
- `internal/api/v1/operations/*_handler.go` - 已迁移为类型安全版本

**system 模块**:
- `internal/api/v1/system/requests/` - 请求结构体目录
- `internal/services/system/*_service.go` - 已迁移为类型安全版本
- `internal/api/v1/system/*_handler.go` - 已迁移为类型安全版本

**monitor 模块** (interface 模式):
- `internal/services/monitor/cache_service.go` - CacheService 接口
- `internal/services/monitor/server_service.go` - ServerService 接口
- `internal/services/monitor/oper_log_service.go` - OperLogService 接口
- `internal/services/monitor/login_log_service.go` - LoginLogService 接口
- `internal/api/v1/monitor/*_handler.go` - 使用 interface 模式

### 下一步

建议采用渐进式迁移策略，继续迁移其他模块：
1. workorder 模块（工单管理）
2. duty 模块（值班管理）
3. network 模块（网络设备管理）
4. scheduler 模块（定时任务）
5. knowledge 模块（知识库）
