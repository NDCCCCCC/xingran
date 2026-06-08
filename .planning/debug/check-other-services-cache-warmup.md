---
slug: check-other-services-cache-warmup
status: resolved
trigger: "检查其他服务缓存预热问题"
created: "2026-06-04T00:00:00Z"
updated: "2026-06-04T00:00:00Z"
---

# 检查其他服务缓存预热问题

## Symptoms

### Expected Behavior
所有系统服务（dept、user、menu、dict、post 等）的缓存预热应该正常工作，缓存管理页面应显示所有相关缓存键。

### Actual Behavior
role 服务发现缺少 List 和 GetByID 缓存实现，需要检查其他服务是否存在类似问题。

### Error Messages
N/A

### Timeline
在修复 role 缓存问题后发现，需要检查其他服务是否有相同问题。

### Reproduction
检查所有 *_cache_impl.go 文件和 WarmUp 函数调用的方法是否匹配。

### Additional Context
需要检查：
1. 所有 *_cache_impl.go 文件（dept、user、menu、dict、post）
2. WarmUp 函数调用的方法列表
3. 缓存服务是否实现了这些方法

## Current Focus

### hypothesis
UserService 缓存实现缺少 List() 方法的缓存覆盖，导致 WarmUpUserCache 调用的 List() 方法不使用缓存。

### next_action
investigate: 检查 UserService 接口的 List 方法在 userCacheService 中是否有缓存实现。

### test
运行应用后检查缓存管理页面，确认预热后是否存在缓存键。

### expecting
userCacheService 没有实现 List() 方法，而是继承了 userService 的非缓存版本。

### reasoning_checkpoint
待定

### tdd_checkpoint
待定

## Evidence

- timestamp: "2025-06-04T10:00:00Z"
  source: "cache_manager.go 分析"
  finding: |
    WarmUp 函数定义了5个服务的预热函数：
    - WarmUpUserCache: 调用 userSvc.List()
    - WarmUpRoleCache: 调用 roleSvc.List()
    - WarmUpMenuCache: 调用 menuSvc.GetTree()
    - WarmUpDeptCache: 调用 deptSvc.GetTree()
    - WarmUpPostCache: 调用 postSvc.List()

- timestamp: "2025-06-04T11:00:00Z"
  source: "user_cache_impl.go 详细分析"
  finding: |
    userCacheService 结构体：
    - 嵌入了 *userService
    - 只实现了缓存相关方法：GetByIDWithCache, GetByUsernameWithCache, GetRolesWithCache, GetPermissionsWithCache
    - 只重写了带缓存失效的方法：Create, Update, Delete, BatchDelete, UpdateStatus, ResetPassword
    - **没有实现 List() 方法**
    - 因此 WarmUpUserCache 调用的是 userService 的原生 List() 方法，不使用缓存

- timestamp: "2025-06-04T11:05:00Z"
  source: "role_cache_impl.go 分析"
  finding: |
    roleCacheService 实现了：
    - List() 方法（带缓存）✅
    - GetByID() 方法（带缓存）✅

- timestamp: "2025-06-04T11:10:00Z"
  source: "department_cache_impl.go 分析"
  finding: |
    departmentCacheService 实现了：
    - GetTree() 方法（带缓存）✅

- timestamp: "2025-06-04T11:15:00Z"
  source: "menu_cache_impl.go 分析"
  finding: |
    menuCacheService 实现了：
    - GetTree() 方法（带缓存）✅

- timestamp: "2025-06-04T11:20:00Z"
  source: "post_cache_impl.go 分析"
  finding: |
    postCacheService 实现了：
    - List() 方法（带缓存）✅

- timestamp: "2025-06-04T12:00:00Z"
  source: "编译验证"
  finding: |
    修复后的 user_cache_impl.go 成功通过编译检查：
    - 添加了 List() 方法，使用缓存包装
    - 添加了 buildListCacheKey() 辅助方法
    - 正确处理指针类型参数（Username, Status, DeptID）
    - 缓存时间设置为10分钟（使用 CacheConfigUserList 配置）

## Eliminated

- ❌ role 服务已实现 List 和 GetByID 缓存
- ❌ menu 服务已实现 GetTree 缓存  
- ❌ dept 服务已实现 GetTree 缓存
- ❌ post 服务已实现 List 缓存

## Resolution

### root_cause
UserService 缓存实现（userCacheService）缺少 List() 方法的缓存覆盖。当 WarmUpUserCache 调用 userSvc.List() 时，实际执行的是基础 userService 的 List() 方法，该方法直接查询数据库而不使用缓存。这导致用户缓存预热功能失效。

### fix
在 userCacheService 中实现 List() 方法，使用缓存包装基础 userService 的 List() 方法。参考 roleCacheService.List() 的实现模式。

实现细节：
- 添加了 List() 方法，调用基础 userService.List() 并缓存结果
- 实现了 buildListCacheKey() 方法，根据查询参数构建唯一缓存键
- 缓存键格式：user:list:username:{username}:status:{status}:dept:{deptId}:page:{current}:size:{pageSize}
- 缓存时间：10分钟（使用 CacheConfigUserList 配置，默认值）
- 正确处理指针类型参数，避免空指针引用

### verification
✅ 修复通过编译检查（go build ./...）
⏳ 需要运行应用程序并验证：
1. 缓存预热后，缓存管理页面显示 user:list 相关缓存键
2. 用户列表查询性能改善
3. 缓存失效机制正常工作（Create/Update/Delete 后清除列表缓存）

### files_changed
- internal/services/system/user_cache_impl.go（添加 List() 和 buildListCacheKey() 方法）
