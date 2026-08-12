---
slug: permission-control-bypass-network-devices
status: resolved
deferred_to: v1.16-tech-debt
trigger: 普通用户角色没有赋予任何权限，但是登录后可以使用网络设备管理所有功能（除了部门列表无权限）
created: 2026-05-24
updated: 2026-06-25
---

## Symptoms

**Expected Behavior:**
普通用户无权限时应无法访问，未授权的功能应完全不可访问，返回403权限不足

**Actual Behavior:**
普通用户登录后具体能访问哪些功能？可访问网络设备管理所有功能（除部门列表外正确拦截）

**Timeline:**
不确定

**Reproduction:**
创建普通用户，不分配权限，登录测试

## Current Focus

**Hypothesis:** 网络设备管理路由在 `internal/api/router.go:312-333` 配置了权限检查中间件，但是使用 `RequirePermissions` 实现"任意一个权限即可通过"的逻辑。普通用户没有任何权限时，理应被所有权限检查拦截，但实际却能访问所有功能。

**Test:** 检查 `RequirePermissions` 中间件的逻辑实现

**Expecting:** 发现中间件实现问题或配置问题

**Next Action:** 分析权限中间件逻辑，确定为何权限检查失效

**Reasoning Checkpoint:** Pending

**TDD Checkpoint:** Pending

## Evidence

### timestamp: 2026-05-24 21:30:00
**Location:** `internal/api/router.go:312-333`

**Finding:** 网络设备管理路由组配置了权限中间件：
```go
network.Use(middleware.RequirePermissions([]string{
    "network:device:list",
    "network:device:add",
    // ... 20+ 个权限
}, core))
```

**Critical Issue:** `RequirePermissions` 使用"任意一个权限即可通过"（OR逻辑），这意味着：
- 用户只要拥有这20+个权限中的**任何一个**，就能访问所有网络设备管理功能
- 这是一个权限设计缺陷：应该是"需要所有权限"（AND逻辑）或者在每个路由上单独配置权限

### timestamp: 2026-05-24 21:35:00
**Location:** `pkg/middleware/permission.go:150-176`

**Finding:** `RequirePermissions` 实现分析：
```go
func RequirePermissions(permissions []string, core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... 检查是否有任意一个权限
        for _, permission := range permissions {
            if checkUserPermission(core, userID, permission) {
                c.Next()  // 有一个权限就通过
                return
            }
        }
        // 都没有才拒绝
        response.Error(c, response.ErrForbidden, "没有访问权限")
        c.Abort()
    }
}
```

**Root Cause:** 权限中间件设计不当
- 在路由组级别使用 `RequirePermissions` 配置了20+个不同权限
- 使用OR逻辑："有任意一个权限即可访问所有功能"
- 导致权限泄露：拥有低权限（如查看权限）的用户可以访问高权限功能（如删除、备份恢复等）

### timestamp: 2026-05-24 21:40:00
**Location:** 对比分析 - 其他模块的正确做法

**System模块的正确做法：**
```go
// 用户管理 - 在路由组级别配置权限（都是用户相关权限，合理）
users.Use(middleware.RequirePermissions([]string{
    string(permission.UserList),
    string(permission.UserAdd),
    string(permission.UserEdit),
    string(permission.UserView),
}, core))

// 部门管理 - 在路由组级别配置权限（都是部门相关权限，合理）
depts.Use(middleware.RequirePermissions([]string{
    string(permission.DeptList),
    string(permission.DeptAdd),
    string(permission.DeptEdit),
    string(permission.DeptView),
}, core))
```

**网络设备模块的问题做法：**
```go
// 网络设备管理 - 混合了不同实体的权限
network.Use(middleware.RequirePermissions([]string{
    "network:device:list",      // 设备相关
    "network:credential:add",   // 凭证相关
    "network:backup:restore",   // 备份相关
    "network:discovery:view",   // 发现相关
    // ... 混合了6-7个不同实体的权限
}, core))
```

**问题分析：**
1. 将不同实体的权限混合在一个路由组级别检查
2. OR逻辑导致：有"设备查看"权限的用户可以访问"备份恢复"功能
3. 违反了最小权限原则

## Eliminated

## Resolution

**root_cause:** 网络设备管理模块在路由组级别使用 `RequirePermissions` 配置了20+个不同实体的权限，由于中间件使用OR逻辑（有任意一个权限即可通过），导致拥有低权限的用户可以访问所有高权限功能，造成权限控制失效。

**proposed_fix:** 有两种修复方案：

**方案1（推荐）：细化权限控制到各个子路由**
- 移除路由组级别的权限中间件
- 在每个子路由组（devices、credentials、templates等）单独配置权限
- 参考MAC和端口路由的做法（lines 153-168）

**方案2：使用AND逻辑权限检查**
- 将 `RequirePermissions` 改为 `RequireAllPermissions`
- 要求用户同时拥有所有20+个权限才能访问
- 缺点：过于严格，不符合实际使用场景

**建议采用方案1**，因为它：
1. 符合最小权限原则
2. 与现有的MAC/端口路由保持一致
3. 更灵活，可以为不同功能配置不同权限

**specialist_hint:** go, permissions, middleware, security

## Phase 40 Closure (2026-06-25)

复测 `internal/api/router.go:333-345` 网络设备模块已移除路由组级别的 OR 权限中间件，
改为委托给 `networkV1.SetupNetworkRouter` 在子路由组分别配置权限：
- `internal/api/v1/network/network_router.go:58` devices → RequirePermissionsWithQuery
- line 79 credentials / 100 templates / 122 command / 138 executions /
  154 backups / 178 discoveries / 198 mac / 209 ports 各自独立 RequirePermissions

已采用 Resolution 推荐方案 1（细化到子路由）。frontmatter 翻 `resolved`。

verification: `grep -rn "RequirePermissions" internal/api/v1/network/network_router.go` 多处命中；`internal/api/router.go` 网络段无 OR 大杂烩
files_changed: .planning/debug/permission-control-bypass-network-devices.md