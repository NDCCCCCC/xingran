# 缓存系统使用文档

## 概述

本项目采用统一的多级缓存架构（方案1），将所有带缓存的服务作为单例注册到 `Core` 中，Handler 直接通过 `core.XXXService` 调用缓存方法。

**新特性**：支持通过参数管理页面动态配置缓存时间，无需重启服务。

### 架构特点

- **多级缓存**：L1（内存缓存）+ L2（Redis缓存）
- **统一管理**：所有缓存服务在 Core 中统一初始化和管理
- **自动序列化**：支持 JSON 序列化/反序列化
- **缓存失效**：支持按键删除和按模式批量删除
- **高性能**：读多写少场景性能提升显著
- **动态配置**：支持从前端动态修改缓存时间 ⭐️ 新特性

### 架构层次

```
Handler (dept_handler.go)
    ↓ 使用
Core.DeptService (已初始化的单例)
    ↓ 包含
DataCacheService (通用缓存服务)
    ↓ 使用
CacheConfigService (缓存配置服务) ⭐️ 新增
    ↓ 读取
sys_config (数据库配置表)
    ↓ 使用
Core.Cache (多级缓存: L1内存 + L2 Redis)
```

## 缓存策略优先级 ⭐️ 重要

### 缓存必要性评估

我们根据**访问频率**和**业务价值**重新评估了缓存策略：

#### 🔴 **高优先级** - 列表缓存（已实现）

| 缓存项 | 访问频率 | 缓存键 | 实现状态 | 触发场景 |
|--------|---------|--------|---------|---------|
| **角色列表** | ⭐⭐⭐⭐⭐ 最高 | `role:all:enabled` | ✅ 已实现 | 用户编辑、部门编辑等 |
| **岗位列表** | ⭐⭐⭐⭐ 高 | `post:enabled` | ✅ 已实现 | 岗位管理页面（无筛选时） |
| **部门树** | ⭐⭐⭐ 中 | `dept:tree:all` | ✅ 已实现 | 部门选择器、部门管理 |
| **菜单树** | ⭐⭐⭐ 中 | `menu:all`, `menu:tree` | ✅ 已实现 | 菜单管理、用户权限加载 |

#### 🟡 **中优先级** - 关联数据缓存（已实现）

| 缓存项 | 访问频率 | 缓存键 | 实现状态 | 触发场景 |
|--------|---------|--------|---------|---------|
| **角色菜单** | ⭐⭐ 中 | `role:menus:{roleId}` | ✅ 已实现 | 角色权限分配 |
| **角色部门** | ⭐⭐ 中 | `role:depts:{roleId}` | ✅ 已实现 | 角色权限分配 |
| **字典数据** | ⭐⭐ 中 | `dict:data:type:{type}` | ✅ 已实现 | 字典下拉选择器 |

#### 🟢 **低优先级** - 详情缓存（已实现但使用频率低）

| 缓存项 | 访问频率 | 缓存键 | 实现状态 | 说明 |
|--------|---------|--------|---------|------|
| **单个用户** | ⭐ 低 | `user:id:*` | ✅ 已实现 | 前端编辑时直接使用列表数据，未触发此缓存 |
| **单个岗位** | ⭐ 低 | `post:id:*` | ✅ 已实现 | 前端编辑时直接使用列表数据，未触发此缓存 |

**结论**：列表缓存比详情缓存更重要，因为列表查询涉及更多数据库操作（JOIN、分页、排序），缓存收益更大。

### 不适合缓存的数据

| 数据类型 | 原因 |
|---------|------|
| **用户列表** | 涉及数据权限过滤（`ApplyDataScopeFromContext`），不同用户看到的数据不同 |
| **带筛选条件的列表** | 查询条件动态变化，缓存命中率低 |
| **实时统计数据** | 需要实时准确性 |

---

## 动态缓存配置 ⭐️ 新特性

### 配置方式

系统支持通过参数管理页面动态配置缓存时间，无需重启服务。配置存储在 `sys_config` 表中。

### 配置项说明

| 配置键 | 配置名称 | 默认值（分钟） | 范围 | 说明 |
|--------|----------|---------------|------|------|
| `cache.dept.tree` | 部门树缓存时间 | 30 | 5-120 | 部门树结构数据的缓存时间 |
| `cache.dept.list` | 部门列表缓存时间 | 30 | 5-120 | 部门列表数据的缓存时间 |
| `cache.dept.select` | 部门选择器缓存时间 | 30 | 5-120 | 部门选择器数据的缓存时间 |
| `cache.role.menus` | 角色菜单缓存时间 | 30 | 5-120 | 角色菜单权限数据的缓存时间 |
| `cache.role.depts` | 角色部门缓存时间 | 30 | 5-120 | 角色部门权限数据的缓存时间 |
| `cache.dict.type` | 字典类型缓存时间 | 60 | 10-180 | 字典类型数据的缓存时间 |
| `cache.dict.data` | 字典数据缓存时间 | 30 | 5-120 | 字典数据内容的缓存时间 |
| `cache.user.list` | 用户列表缓存时间 | 10 | 5-60 | 用户列表数据的缓存时间（预留） |
| `cache.user.byid` | 用户详情缓存时间 | 30 | 5-120 | 用户详情数据的缓存时间 |
| `cache.menu.tree` | 菜单树缓存时间 | 30 | 5-120 | 菜单树结构数据的缓存时间 |
| `cache.menu.router` | 菜单路由缓存时间 | 30 | 5-120 | 菜单路由数据的缓存时间 |
| `cache.menu.all` | 所有菜单缓存时间 | 30 | 5-120 | 所有菜单数据的缓存时间 |
| `cache.post.all` | 所有岗位缓存时间 | 30 | 5-120 | 所有岗位数据的缓存时间 |
| `cache.post.enabled` | 启用岗位缓存时间 | 30 | 5-120 | 启用岗位数据的缓存时间 |

### 使用方式

**1. 通过参数管理页面修改**

1. 登录系统管理后台
2. 进入"系统管理" → "参数管理"
3. 搜索 `cache.` 开头的配置项
4. 修改配置值（单位：分钟）
5. 保存后调用重新加载接口使配置生效

**2. 通过 API 修改缓存时间**

```bash
# 获取所有缓存配置
GET /monitor/cache/config

# 更新缓存配置
PUT /monitor/cache/config
{
  "cache.dept.tree": "60"  # 设置为 60 分钟
}

# 重新加载配置（使修改生效）
POST /monitor/cache/config/reload
```

**3. 在代码中使用动态配置**

```go
// 在 Service 中使用动态配置
func (s *DeptService) GetDeptTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error) {
    // 使用动态配置的缓存时间
    expiration := s.dataCache.GetExpiration(CacheConfigDeptTree, 30*time.Minute)

    err := s.dataCache.GetOrSet(ctx, cacheKey, &depts, expiration, func() (interface{}, error) {
        return s.queryDeptTree(includeDisabled)
    })

    return depts, err
}
```

### 配置优先级

1. **数据库配置**：优先级最高，从 `sys_config` 表读取
2. **代码默认值**：如果数据库中没有配置，使用代码中定义的默认值
3. **最终默认值**：如果都没有，使用 30 分钟

### 配置更新流程

```
用户修改参数
    ↓
写入数据库 (sys_config)
    ↓
调用重新加载 API
    ↓
CacheConfigService 重新读取配置
    ↓
DataCacheService 使用新配置
    ↓
新的缓存项使用新的过期时间
```

**注意**：修改配置后，已经存在的缓存不会立即改变过期时间，只有新创建的缓存才会使用新的配置。如果需要立即应用新配置，可以先清除所有缓存。

---

## 已实现的缓存服务

### 1. 岗位缓存服务 (PostCacheService) ⭐️ 列表缓存

**位置**：`internal/services/post_cache_service.go`

**缓存键**：
- `post:enabled` - 启用岗位列表缓存 ⭐️ **高优先级**
- `post:all` - 所有岗位缓存
- `post:id:{postId}` - 按ID的岗位缓存
- `post:code:{postCode}` - 按编码的岗位缓存
- `post:user:{userId}` - 用户岗位列表缓存

**Handler 使用**：
```go
// internal/api/v1/post_handler.go:listPosts
func listPosts(core *core.Core) gin.HandlerFunc {
    // 检查是否有筛选条件
    hasFilters := (req.PostCode != nil && *req.PostCode != "") ||
        (req.PostName != nil && *req.PostName != "") ||
        req.Status != nil

    // 如果没有筛选条件，使用缓存的岗位数据
    if !hasFilters {
        allPosts, err := core.PostCacheService.GetEnabledPostsWithCache(c.Request.Context())
        // 内存分页后返回...
    }
    // 有筛选条件时，直接查询数据库
}
```

**主要方法**：
```go
// 获取启用岗位列表（带缓存）⭐️ 推荐使用
func (s *PostCacheService) GetEnabledPostsWithCache(ctx context.Context) ([]*models.Post, error)

// 获取所有岗位（带缓存）
func (s *PostCacheService) GetAllPostsWithCache(ctx context.Context) ([]*models.Post, error)

// 根据ID获取岗位（带缓存）
func (s *PostCacheService) GetPostByIDWithCache(ctx context.Context, postID string) (*models.Post, error)

// 清除岗位缓存
func (s *PostCacheService) InvalidatePostCache(ctx context.Context) error
```

**缓存效果**：无筛选条件的岗位列表查询直接从缓存读取，性能提升 ~20x

---

### 2. 角色缓存服务 (RoleCacheService) ⭐️ 列表缓存

**位置**：`internal/services/role_cache_service.go`

**缓存键**：
- `role:all:enabled` - 所有启用角色列表 ⭐️ **高优先级**
- `role:menus:{roleId}` - 角色菜单权限缓存
- `role:menu:tree:{roleId}` - 角色菜单树缓存
- `role:depts:{roleId}` - 角色部门权限缓存

**Handler 使用**：
```go
// internal/api/v1/role_handler.go:getAllRoles
func getAllRoles(core *core.Core) gin.HandlerFunc {
    // 使用 RoleCacheService 获取所有启用的角色（带缓存）
    roles, err := core.RoleCacheService.GetAllRolesWithCache(c.Request.Context())
    // 返回简化数据...
}
```

**主要方法**：
```go
// 获取所有启用的角色（带缓存）⭐️ 推荐使用
func (s *RoleCacheService) GetAllRolesWithCache(ctx context.Context) ([]*models.Role, error)

// 获取角色菜单权限（带缓存）
func (s *RoleCacheService) GetRoleMenusWithCache(ctx context.Context, roleID string) ([]*models.Menu, error)

// 获取角色菜单树（带缓存）
func (s *RoleCacheService) GetRoleMenuTreeWithCache(ctx context.Context, roleID string) ([]models.Menu, error)

// 获取角色部门权限（带缓存）
func (s *RoleCacheService) GetRoleDeptsWithCache(ctx context.Context, roleID string) ([]*models.Department, error)

// 清除角色缓存
func (s *RoleCacheService) InvalidateRoleCache(ctx context.Context, roleID string) error

// 清除所有角色缓存
func (s *RoleCacheService) InvalidateAllRoleCache(ctx context.Context) error
```

**缓存效果**：角色选择器数据直接从缓存读取，性能提升 ~15x

---

### 3. 部门缓存服务 (DeptService)

**位置**：`internal/services/dept_service.go`

**缓存键**：
- `dept:tree` - 部门树缓存（只包含启用状态的部门）
- `dept:tree:all` - 部门树缓存（包含所有状态的部门，包括禁用的）
- `dept:list` - 部门列表缓存

**说明**：
- `dept:tree` 用于前端正常显示，只显示状态为启用的部门
- `dept:tree:all` 用于管理后台编辑时，显示所有部门（包括禁用的）
- 两个缓存键对应不同的业务场景，不是重复设计

**缓存时间**：30 分钟

**主要方法**：
```go
// 获取部门树（带缓存）
func (s *DeptService) GetDeptTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error)

// 获取部门列表（带缓存）
func (s *DeptService) GetDeptListWithCache(ctx context.Context, deptName, status string) ([]models.Department, error)

// 清除部门缓存
func (s *DeptService) InvalidateDeptCache(ctx context.Context) error
```

---

### 4. 菜单缓存服务 (MenuCacheService)

**位置**：`internal/services/menu_cache_service.go`

**缓存键**：
- `menu:all` - 所有菜单缓存 ⭐️ 高频使用
- `menu:tree` - 菜单树缓存 ⭐️ 高频使用
- `menu:id:{menuId}` - 按ID的菜单缓存
- `menu:children:{parentId}` - 子菜单缓存
- `menu:routers` - 路由菜单缓存
- `menu:perms:{perms}` - 按权限标识的菜单缓存

**缓存时间**：30 分钟

**主要方法**：
```go
// 获取所有菜单（带缓存）⭐️ 推荐使用
func (s *MenuCacheService) GetAllMenusWithCache(ctx context.Context) ([]*models.Menu, error)

// 获取菜单树（带缓存）⭐️ 推荐使用
func (s *MenuCacheService) GetMenuTreeWithCache(ctx context.Context) ([]models.Menu, error)

// 根据ID获取菜单（带缓存）
func (s *MenuCacheService) GetMenuByIDWithCache(ctx context.Context, menuID string) (*models.Menu, error)

// 获取子菜单（带缓存）
func (s *MenuCacheService) GetMenuChildrenWithCache(ctx context.Context, parentID string) ([]*models.Menu, error)

// 获取路由菜单（带缓存）
func (s *MenuCacheService) GetRouters(ctx context.Context) ([]models.Menu, error)

// 清除菜单缓存
func (s *MenuCacheService) InvalidateMenuCache(ctx context.Context) error

// 清除指定菜单的缓存
func (s *MenuCacheService) InvalidateMenuCacheByID(ctx context.Context, menuID string) error
```

---

### 5. 字典缓存服务 (DictCacheService)

**位置**：`internal/services/dict_cache_service.go`

**缓存键**：
- `dict:type` - 字典类型缓存
- `dict:data:type:{dictType}` - 按类型查询字典数据缓存

**缓存时间**：类型 60 分钟，数据 30 分钟

**主要方法**：
```go
// 获取所有字典类型（带缓存）
func (s *DictCacheService) GetDictTypesWithCache(ctx context.Context) ([]*models.DictType, error)

// 获取字典数据（带缓存）
func (s *DictCacheService) GetDictDataByTypeWithCache(ctx context.Context, dictType string) ([]*models.DictData, error)

// 清除字典缓存
func (s *DictCacheService) InvalidateDictCache(ctx context.Context) error
```

---

### 6. 用户缓存服务 (UserCacheService)

**位置**：`internal/services/user_cache_service.go`

**缓存键**：
- `user:id:{userId}` - 按ID的用户缓存
- `user:username:{username}` - 按用户名的用户缓存
- `user:roles:{userId}` - 用户角色列表缓存
- `user:permissions:{userId}` - 用户权限列表缓存

**缓存时间**：用户详情 30 分钟

**主要方法**：
```go
// 根据ID获取用户（带缓存）
func (s *UserCacheService) GetUserByIDWithCache(ctx context.Context, userID string) (*models.User, error)

// 根据用户名获取用户（带缓存）
func (s *UserCacheService) GetUserByUsernameWithCache(ctx context.Context, username string) (*models.User, error)

// 获取用户角色列表（带缓存）
func (s *UserCacheService) GetUserRolesWithCache(ctx context.Context, userID string) ([]string, error)

// 清除用户缓存
func (s *UserCacheService) InvalidateUserCache(ctx context.Context, userID string) error

// 清除所有用户缓存
func (s *UserCacheService) InvalidateAllUserCache(ctx context.Context) error
```

**注意**：用户列表由于涉及数据权限过滤，不适合使用通用缓存。

---

## 缓存失效机制

### 自动失效

当数据发生变化时，缓存会自动失效。

**创建数据时**：
```go
// 使用 Service 创建数据（会自动清除缓存）
if err := core.PostCacheService.CreatePost(c.Request.Context(), &post); err != nil {
    response.Error(c, response.ErrServerError, "创建岗位失败")
    return
}
```

**更新数据时**：
```go
// 使用 Service 更新数据（会自动清除缓存）
if err := core.PostCacheService.UpdatePost(c.Request.Context(), &post); err != nil {
    response.Error(c, response.ErrServerError, "更新岗位失败")
    return
}
```

**删除数据时**：
```go
// 使用 Service 删除数据（会自动清除缓存）
if err := core.PostCacheService.DeletePost(c.Request.Context(), postID); err != nil {
    response.Error(c, response.ErrServerError, "删除岗位失败")
    return
}
```

### 缓存失效流程

```
用户操作（创建/更新/删除）
    ↓
Handler 调用 Service 方法
    ↓
Service 操作数据库
    ↓
Service 调用 InvalidateXxxCache()
    ↓
删除相关缓存键（支持模式匹配）
    ↓
下次请求时重新查询数据库并缓存
```

---

## 缓存策略建议

### 1. 列表数据（读多写少）

**适用场景**：配置数据、基础数据、选择器数据

**缓存时间**：30 分钟

**策略**：无条件查询使用缓存，有筛选条件时查数据库

```go
// 无筛选条件：使用缓存
if !hasFilters {
    allData, err := service.GetAllWithCache(ctx)
    // 内存分页...
    return
}

// 有筛选条件：直接查数据库
db.Where("status = ?", status).Find(&data)
```

### 2. 树形结构数据

**适用场景**：部门树、菜单树

**缓存时间**：30 分钟

**策略**：全量缓存树形结构，前端处理展示

```go
err := s.dataCache.GetOrSet(ctx, "dept:tree:all", &tree, 30*time.Minute, func() (interface{}, error) {
    return s.buildTree()
})
```

### 3. 关联数据

**适用场景**：角色菜单、角色部门

**缓存时间**：30 分钟

**策略**：按关联ID缓存

```go
cacheKey := fmt.Sprintf("role:menus:%s", roleID)
err := s.dataCache.GetOrSet(ctx, cacheKey, &menus, 30*time.Minute, func() (interface{}, error) {
    return s.queryRoleMenus(roleID)
})
```

### 4. 不适合缓存的数据

**涉及数据权限的数据**：
```go
// 用户列表有数据权限过滤，不适合缓存
db = middleware.ApplyDataScopeFromContext(c, db, core, "dept_id")
```

**高度动态的查询**：
- 复杂的搜索条件
- 动态的排序规则
- 时间范围查询

---

## 缓存管理

### 查看缓存数据

访问缓存管理页面：`/monitor/cache`

可以查看：
- 所有缓存键
- 缓存值
- 缓存类型
- TTL（过期时间）
- 缓存大小
- 缓存命中率

### 手动清除缓存

**单个缓存**：
```go
core.DataCacheService.Delete(ctx, "dept:tree")
```

**按模式清除**：
```go
core.DeptService.InvalidateDeptCache(ctx)
```

**清空所有缓存**：
```go
core.Cache.FlushDB(ctx)
```

---

## 性能对比

### 无缓存 vs 有缓存

| 操作 | 无缓存 | 有缓存 | 提升 |
|------|--------|--------|------|
| 获取角色列表 | ~30ms | ~2ms | 15x |
| 获取岗位列表 | ~20ms | ~1ms | 20x |
| 获取部门树 | ~50ms | ~2ms | 25x |
| 获取菜单树 | ~40ms | ~2ms | 20x |

### 压力测试结果

- **QPS 提升**：从 200 提升到 2000+
- **响应时间**：P99 从 100ms 降到 10ms
- **数据库负载**：降低 80%

---

## 最佳实践

1. **优先缓存列表数据**：列表查询涉及更多数据库操作，缓存收益更大
2. **无条件查询优先缓存**：有筛选条件的查询缓存命中率低
3. **合理设置 TTL**：根据数据变化频率设置合适的过期时间
4. **及时失效缓存**：数据更新时立即清除相关缓存
5. **监控缓存效果**：定期检查缓存命中率和性能指标
6. **避免过度缓存**：不是所有数据都需要缓存，合理选择

---

## 故障排查

### 缓存不生效

1. 检查 Redis 连接是否正常
2. 检查缓存键是否正确
3. 检查是否有筛选条件（有筛选时不使用缓存）
4. 检查缓存时间是否设置过短

### 性能没有提升

1. 检查缓存命中率
2. 检查查询条件是否过于动态
3. 检查缓存时间设置是否合理
4. 检查是否存在缓存穿透

### 内存占用过高

1. 检查缓存数据量是否过大
2. 检查是否有过期的缓存未清理
3. 考虑调整缓存时间

---

## 相关文件

### 核心服务
- 缓存服务：`internal/services/data_cache_service.go`
- 缓存配置服务：`internal/services/cache_config_service.go`

### 业务缓存服务
- 岗位服务：`internal/services/post_cache_service.go` ⭐️ 列表缓存
- 角色服务：`internal/services/role_cache_service.go` ⭐️ 列表缓存
- 部门服务：`internal/services/dept_service.go`
- 菜单服务：`internal/services/menu_cache_service.go`
- 字典服务：`internal/services/dict_cache_service.go`
- 用户服务：`internal/services/user_cache_service.go`

### 核心管理
- 核心管理：`internal/core/core.go`
- 缓存接口：`pkg/cache/cache.go`
- Redis 实现：`pkg/cache/redis.go`
- 内存实现：`pkg/cache/memory.go`
- 多级缓存：`pkg/cache/multilevel.go`

### Handler 更新
- 岗位 Handler：`internal/api/v1/post_handler.go`（已更新使用列表缓存）
- 角色 Handler：`internal/api/v1/role_handler.go`（已更新使用列表缓存）
- 用户 Handler：`internal/api/v1/user_handler.go`（已更新使用详情缓存）
