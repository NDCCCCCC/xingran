# Phase 25-03 Summary: 路由权限中间件配置

## 完成状态
✅ 完成

## 变更内容

### 文件修改
- `internal/api/router.go` - 为 VDI 路由添加 DataScopePermission 中间件
- `internal/api/v1/vdi/vm_router.go` - 拆分 /operate 路由为细粒度权限路由
- `internal/api/v1/vdi/vm_handler.go` - 添加 StartVM、StopVM、RestartVM 处理方法

## 实现细节

### Task 1: 添加 DataScopePermission 中间件

**修改**: `internal/api/router.go`

**变更内容**:
1. VDI 路由组路径从 `/vdi/vm` 改为 `/vdi/vms`
2. 在 vms 路由组添加 `DataScopePermission(core)` 中间件
3. 从权限列表中删除 `vdi:vm:operate`（已被拆分）

**关键实现**:
```go
// 虚拟机管理
vms := vdi.Group("/vms")
vms.Use(middleware.DataScopePermission(core))  // 新增：设置 context 值
vms.Use(middleware.RequirePermissions([]string{
    "vdi:vm:list",
    "vdi:vm:add",
    "vdi:vm:edit",
}, core))
{
    vdiV1.SetupVMRouter(vms, core)
}
```

### Task 2: 拆分 /operate 路由

**修改**: `internal/api/v1/vdi/vm_router.go`

**新增路由**:
- `POST /start` - 启动虚拟机，权限 `vdi:vm:start`
- `POST /stop` - 关闭虚拟机，权限 `vdi:vm:stop`
- `POST /restart` - 重启虚拟机，权限 `vdi:vm:restart`

**细粒度权限路由**:
```go
// VDI 电源操作 — 使用细粒度权限，拆分为独立路由
r.POST("/start", middleware.RequirePermissions([]string{"vdi:vm:start"}, core), vmHandler.StartVM)
r.POST("/stop", middleware.RequirePermissions([]string{"vdi:vm:stop"}, core), vmHandler.StopVM)
r.POST("/restart", middleware.RequirePermissions([]string{"vdi:vm:restart"}, core), vmHandler.RestartVM)

// 用户绑定操作
r.POST("/:id/bind_user", middleware.RequirePermissions([]string{"vdi:vm:bind"}, core), vmHandler.BindUser)
r.POST("/:id/unbind_user", middleware.RequirePermissions([]string{"vdi:vm:bind"}, core), vmHandler.UnbindUser)

// 同步操作
r.POST("/:id/sync", middleware.RequirePermissions([]string{"vdi:vm:sync"}, core), vmHandler.SyncFromVDI)
r.POST("/sync-all", middleware.RequirePermissions([]string{"vdi:vm:sync"}, core), vmHandler.SyncAll)

// 删除操作
r.POST("/:id/delete", middleware.RequirePermissions([]string{"vdi:vm:delete"}, core), vmHandler.Delete)
```

**权限映射**:
| 路由 | 权限 |
|------|------|
| /start | vdi:vm:start |
| /stop | vdi:vm:stop |
| /restart | vdi:vm:restart |
| /bind_user | vdi:vm:bind |
| /unbind_user | vdi:vm:bind |
| /sync | vdi:vm:sync |
| /sync-all | vdi:vm:sync |
| /:id/delete | vdi:vm:delete |

### Task 3: 添加独立操作处理方法

**修改**: `internal/api/v1/vdi/vm_handler.go`

**新增方法**:
1. `StartVM(c *gin.Context)` - 固定动作为 `VMPowerOn`
2. `StopVM(c *gin.Context)` - 固定动作为 `VMPowerOff`
3. `RestartVM(c *gin.Context)` - 固定动作为 `VMPowerRestart`

**实现模式**:
```go
func (h *VMHandler) StartVM(c *gin.Context) {
    var req vdiServices.VMOperateRequest
    if !handleJSONBinding(c, &req) {
        return
    }
    // 固定动作为 start（不由客户端提供）
    req.Action = vdiServices.VMPowerOn
    
    if !handleServiceError(c, h.vmService.OperateVM(c.Request.Context(), &req), "启动虚拟机") {
        return
    }
    response.Success(c, nil)
}
```

**关键设计**:
- action 字段由 Handler 设置，不由客户端提供
- 消除路由层 OR 逻辑需求
- Swagger 文档完整

## 架构决策

### 符合 Architectural Responsibility Map
- **路由层**: 验证细粒度操作权限
- **中间件层**: 设置数据范围上下文值
- **Handler 层**: 将路由映射到服务操作
- **Service 层**: 应用数据范围过滤

### 消除复杂权限逻辑
- 旧方案: `/operate` 路由 + 复杂 OR 权限 (`vdi:vm:start OR vdi:vm:stop OR ...`)
- 新方案: 独立路由 + 单一权限 (`/start` 仅 `vdi:vm:start`)

## 验证结果
- ✅ router.go 路由路径改为 /vdi/vms
- ✅ DataScopePermission 中间件添加成功
- ✅ /operate 路由已删除
- ✅ 3个独立电源操作路由创建
- ✅ 绑定/解绑用户使用 vdi:vm:bind 权限
- ✅ 同步操作使用 vdi:vm:sync 权限
- ✅ 3个 Handler 方法添加成功
- ✅ Swagger 文档完整
- ✅ 代码编译通过

## 下一步依赖
- 25-04-PLAN.md: 前端动态按钮渲染（使用新权限标识符）
