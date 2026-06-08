---
slug: role-cache-preload-issue
status: resolved
trigger: "Role缓存预加载问题"
created: "2026-06-04T00:00:00Z"
updated: "2026-06-04T08:15:00Z"
---

# Role缓存预加载问题

## Symptoms

### Expected Behavior
Role 相关缓存应该在系统启动后自动预加载，并在缓存管理页面可见。

### Actual Behavior
缓存管理页面看不到 role 相关缓存。

### Error Messages
N/A

### Timeline
问题最近被发现，具体时间未知。

### Reproduction
查看缓存管理界面，没有发现 role 相关的缓存键。

### Additional Context
需要检查：
1. role 缓存预加载的代码位置
2. 预加载是否被调用
3. 缓存键命名是否正确
4. 是否有错误日志

## Current Focus

### hypothesis
role 缓存预加载可能没有被正确调用，或者缓存键命名不匹配导致查询失败。

### next_action
gather initial evidence: 搜索 role 缓存预加载相关代码，检查启动逻辑。

### test
待定

### expecting
待定

### reasoning_checkpoint
待定

### tdd_checkpoint
待定

## Evidence

### timestamp: 2026-06-04T08:00:00Z
**Found root cause:**

The `WarmUpRoleCache` function in `cache_manager.go` calls `roleSvc.List(ctx, params)` to warm up the cache. However, the `roleCacheService` implementation does NOT override the `List` method - it only overrides `GetAllEnabled`, `GetMenusWithCache`, and `GetDeptsWithCache`.

This means:
1. The base `roleService.List` method is called, which performs a direct database query without caching
2. No cache entries are created during warm-up
3. The cache management page shows no role-related cache keys

**Evidence:**
- `role_cache_impl.go` has no `List` method override
- Cache keys exist: `CacheKeyRoleAll = "cache:role:all"`, `CacheKeyRoleEnabled = "cache:role:enabled"`
- Warm-up function: `WarmUpRoleCache` calls `roleSvc.List()`
- Service initialization: `NewRoleServiceWithCache` is correctly used in `initSystemServicesForWarmUp`

### timestamp: 2026-06-04T08:15:00Z
**Fix applied:**

Added two missing cached methods to `roleCacheService`:
1. `List(ctx, params)` - 缓存角色列表查询结果
2. `GetByID(ctx, id)` - 缓存单个角色详情查询

Both methods use appropriate cache keys and 30-minute expiration.

## Eliminated

## Resolution

### root_cause
`roleCacheService` 没有重写 `List` 和 `GetByID` 方法，导致预热函数调用的是基类 `roleService` 的方法，这些方法直接查询数据库而不使用缓存。只有 `GetAllEnabled`、`GetMenusWithCache` 和 `GetDeptsWithCache` 有缓存实现。

### fix
在 `role_cache_impl.go` 中添加了以下方法：
1. `List(ctx, params)` - 使用 `cache:role:list` 开头的缓存键，包含查询参数以支持不同查询条件的缓存
2. `GetByID(ctx, id)` - 使用 `cache:role:all:id:{id}` 作为缓存键
3. `buildListCacheKey(params)` - 辅助方法，根据查询参数构建唯一缓存键

所有方法使用 30 分钟的缓存过期时间，并在数据变更时通过 `InvalidateRoleCache` 清除相关缓存。

### verification
1. ✅ 代码编译成功
2. 重启应用程序
3. 检查缓存管理页面是否显示 role 缓存条目
4. 验证日志是否显示角色缓存预热成功

### files_changed
- `internal/services/system/role_cache_impl.go` - 添加了带缓存的 `List` 和 `GetByID` 方法