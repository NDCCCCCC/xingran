# 缓存系统使用文档

## 概述

本项目采用基于 `CacheProvider` 接口的模块化缓存架构，将缓存服务作为 System 模块的一部分，通过接口解耦实现灵活的缓存策略。

**架构特点**：
- **接口解耦**：通过 `CacheProvider` 接口实现依赖倒置
- **服务组合**：缓存服务组合基础服务，避免代码重复
- **多级缓存**：L1（内存缓存）+ L2（Redis缓存）
- **动态配置**：支持通过参数管理页面动态配置缓存时间，无需重启服务
- **全量缓存 + 内存筛选**：提高缓存命中率，减少数据库查询

### 架构层次

```
Handler (role_handler.go)
    ↓ 使用
RoleService (接口)
    ↓ 实现
roleCacheService (组合 roleService + CacheProvider)
    ↓ 使用
CacheProvider (接口)
    ↓ 实现
DataCacheServiceAdapter (适配 DataCacheService)
    ↓ 使用
Core.Cache (多级缓存: L1内存 + L2 Redis)
    ↓ 读取
CacheConfigService (缓存配置服务) - 动态 TTL
    ↓ 读取
sys_config (数据库配置表)
```

### 新架构优势

| 特性 | 说明 |
|------|------|
| **接口解耦** | 通过 `CacheProvider` 接口，可轻松替换缓存实现 |
| **可测试性** | 提供 `NoOpCacheProvider` 空实现，方便单元测试 |
| **服务组合** | 缓存服务组合基础服务，继承所有 CRUD 方法 |
| **统一配置** | 通过 `CacheServiceBase` 统一管理缓存配置 |
| **通用工具** | `cache_utils.go` 提供通用的缓存失效函数 |

---

## 缓存服务实现

### 核心：CacheProvider 接口

**位置**：`internal/services/system/cache_provider.go`

```go
// CacheProvider 缓存提供者接口
// 由 system 模块外部实现，用于解耦缓存逻辑
type CacheProvider interface {
    // GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
    GetOrSet(
        ctx context.Context,
        key string,
        dest interface{},
        expiration time.Duration,
        query func() (interface{}, error),
    ) error

    // Delete 删除缓存
    Delete(ctx context.Context, key string) error

    // DeleteByPattern 根据模式删除缓存
    DeleteByPattern(ctx context.Context, pattern string) error
}

// NoOpCacheProvider 空缓存提供者（用于无缓存场景或测试）
type NoOpCacheProvider struct{}
```

### 缓存服务基础类

**位置**：`internal/services/system/cache_utils.go`

```go
// CacheServiceBase 缓存服务基础结构
type CacheServiceBase struct {
    Config *services.CacheConfigService
}

// GetExpiration 获取缓存过期时间（通用方法）
func (b *CacheServiceBase) GetExpiration(configKey string, defaultVal time.Duration) time.Duration

// InvalidateCacheByPattern 根据模式列表失效缓存（通用方法）
func InvalidateCacheByPattern(ctx context.Context, cache CacheProvider, patterns []string, module string)

// InvalidateCacheByKey 根据键列表失效缓存（通用方法）
func InvalidateCacheByKey(ctx context.Context, cache CacheProvider, keys []string, module string)
```

---

## 缓存策略优先级

### 缓存必要性评估

我们根据**访问频率**和**业务价值**重新评估了缓存策略：

#### 🔴 **高优先级** - 列表缓存（已实现）

| 缓存项 | 访问频率 | 缓存键 | 实现状态 | 触发场景 |
|--------|---------|--------|---------|---------|
| **角色列表** | ⭐⭐⭐⭐⭐ 最高 | `role:enabled` | ✅ 已实现 | 用户编辑、部门编辑等 |
| **岗位列表** | ⭐⭐⭐⭐ 高 | `post:enabled` | ✅ 已实现 | 岗位管理页面（无筛选时） |
| **部门树** | ⭐⭐⭐ 中 | `dept:tree` | ✅ 已实现 | 部门选择器、部门管理 |
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
| **单个用户** | ⭐ 低 | `user:id` | ✅ 已实现 | 前端编辑时直接使用列表数据，未触发此缓存 |
| **单个岗位** | ⭐ 低 | `post:id` | ✅ 已实现 | 前端编辑时直接使用列表数据，未触发此缓存 |

**结论**：列表缓存比详情缓存更重要，因为列表查询涉及更多数据库操作（JOIN、分页、排序），缓存收益更大。

### 不适合缓存的数据

| 数据类型 | 原因 |
|---------|------|
| **用户列表** | 涉及数据权限过滤（`ApplyDataScopeFromContext`），不同用户看到的数据不同 |
| **带筛选条件的列表** | 查询条件动态变化，缓存命中率低 |
| **实时统计数据** | 需要实时准确性 |

---

## 动态缓存配置

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
// internal/services/system/role_cache_impl.go
func (s *roleCacheService) GetAllEnabled(ctx context.Context) ([]*models.Role, error) {
    cacheKey := services.CacheKeyRoleEnabled
    var result []*models.Role

    // 使用动态配置的缓存时间
    expiration := s.GetExpiration(services.CacheConfigRoleMenus, 30*time.Minute)

    err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
        return s.roleService.GetAllEnabled(ctx)
    })

    return result, err
}
```

### 配置优先级

1. **数据库配置**：优先级最高，从 `sys_config` 表读取
2. **代码默认值**：如果数据库中没有配置，使用代码中定义的默认值
3. **最终默认值**：如果都没有，使用 30 分钟

---

## 已实现的缓存服务

### 1. 角色缓存服务 (RoleService) ⭐️ 列表缓存

**位置**：`internal/services/system/role_cache_impl.go`

**架构**：
```go
type roleCacheService struct {
    *roleService              // 组合基础服务
    cache       CacheProvider // 使用接口解耦
    CacheServiceBase          // 统一的缓存配置
}
```

**缓存键**：
- `role:enabled` - 所有启用角色列表 ⭐️ **高优先级**
- `role:menus:{roleId}` - 角色菜单权限缓存
- `role:depts:{roleId}` - 角色部门权限缓存

**主要方法**：
```go
// 获取所有启用的角色（带缓存）⭐️ 推荐使用
func (s *roleCacheService) GetAllEnabled(ctx context.Context) ([]*models.Role, error)

// 获取角色菜单权限（带缓存）
func (s *roleCacheService) GetMenusWithCache(ctx context.Context, roleID string) ([]models.Menu, error)

// 获取角色部门权限（带缓存）
func (s *roleCacheService) GetDeptsWithCache(ctx context.Context, roleID string) ([]models.Department, error)

// 失效角色缓存
func (s *roleCacheService) InvalidateRoleCache(ctx context.Context, roleID string) error

// 创建/更新/删除（自动失效缓存）
func (s *roleCacheService) Create(ctx context.Context, req *RoleCreateRequest) error
func (s *roleCacheService) Update(ctx context.Context, req *RoleUpdateRequest) error
func (s *roleCacheService) Delete(ctx context.Context, id string) error
```

**缓存效果**：角色选择器数据直接从缓存读取，性能提升 ~15x

---

### 2. 岗位缓存服务 (PostService) ⭐️ 列表缓存

**位置**：`internal/services/system/post_cache_impl.go`

**架构**：
```go
type postCacheService struct {
    *postService              // 组合基础服务
    cache       CacheProvider // 使用接口解耦
    CacheServiceBase          // 统一的缓存配置
}
```

**缓存键**：
- `post:enabled` - 启用岗位列表缓存 ⭐️ **高优先级**
- `post:all` - 所有岗位缓存

**特色功能**：全量缓存 + 内存筛选

```go
// List 查询岗位列表（全量缓存+内存筛选版本）
func (s *postCacheService) List(ctx context.Context, params PostListParams) (*PageResult, error) {
    // 1. 获取全量缓存
    allPosts, err := s.GetAllWithCache(ctx)

    // 2. 内存筛选
    filtered := s.filterPosts(allPosts, params)

    // 3. 内存分页
    paged, total := paginate(filtered, params.Current, params.PageSize)

    return &PageResult{...}, nil
}
```

**主要方法**：
```go
// 获取所有岗位（带缓存）
func (s *postCacheService) GetAllWithCache(ctx context.Context) ([]*models.Post, error)

// 获取启用的岗位（带缓存）
func (s *postCacheService) GetEnabledWithCache(ctx context.Context) ([]*models.Post, error)

// 查询列表（内存筛选版）
func (s *postCacheService) List(ctx context.Context, params PostListParams) (*PageResult, error)

// 失效岗位缓存
func (s *postCacheService) InvalidatePostCache(ctx context.Context) error

// 创建/更新/删除（自动失效缓存）
func (s *postCacheService) Create(ctx context.Context, req *PostCreateRequest) error
func (s *postCacheService) Update(ctx context.Context, req *PostUpdateRequest) error
func (s *postCacheService) Delete(ctx context.Context, id string) error
```

**缓存效果**：无筛选条件的岗位列表查询直接从缓存读取，性能提升 ~20x

---

### 3. 字典类型缓存服务 (DictTypeService)

**位置**：`internal/services/system/dict_cache_impl.go`

**架构**：
```go
type dictTypeCacheService struct {
    *dictTypeService          // 组合基础服务
    cache           CacheProvider
    CacheServiceBase
}
```

**缓存键**：
- `dict:type` - 所有字典类型缓存

**主要方法**：
```go
// 获取所有字典类型（带缓存）
func (s *dictTypeCacheService) GetAllWithCache(ctx context.Context) ([]*models.DictType, error)

// 查询列表（内存筛选版）
func (s *dictTypeCacheService) List(ctx context.Context, params DictTypeListParams) (*PageResult, error)

// 失效缓存
func (s *dictTypeCacheService) invalidateCache(ctx context.Context)
```

**缓存时间**：60 分钟（字典类型变化较少）

---

### 4. 字典数据缓存服务 (DictDataService)

**位置**：`internal/services/system/dict_cache_impl.go`

**架构**：
```go
type dictDataCacheService struct {
    *dictDataService          // 组合基础服务
    cache           CacheProvider
    CacheServiceBase
}
```

**缓存键**：
- `dict:data:type:{dictType}` - 按类型查询字典数据缓存

**主要方法**：
```go
// 根据类型获取字典数据（带缓存）
func (s *dictDataCacheService) GetByTypeWithCache(ctx context.Context, dictType string) ([]*models.DictData, error)

// 失效缓存
func (s *dictDataCacheService) invalidateCache(ctx context.Context, dictType string)
```

**缓存时间**：30 分钟

---

### 5. 部门缓存服务 (DepartmentService)

**位置**：`internal/services/system/department_cache_impl.go`

**缓存键**：
- `dept:tree` - 部门树缓存（只包含启用状态的部门）
- `dept:tree:all` - 部门树缓存（包含所有状态的部门，包括禁用的）

**说明**：
- `dept:tree` 用于前端正常显示，只显示状态为启用的部门
- `dept:tree:all` 用于管理后台编辑时，显示所有部门（包括禁用的）

**主要方法**：
```go
// 获取部门树（带缓存）
func (s *departmentCacheService) GetTreeWithCache(ctx context.Context, includeDisabled bool) ([]*models.Department, error)

// 失效缓存
func (s *departmentCacheService) InvalidateCache(ctx context.Context) error
```

---

### 6. 菜单缓存服务 (MenuService)

**位置**：`internal/services/system/menu_cache_impl.go`

**缓存键**：
- `menu:all` - 所有菜单缓存 ⭐️ 高频使用
- `menu:tree` - 菜单树缓存 ⭐️ 高频使用

**主要方法**：
```go
// 获取所有菜单（带缓存）⭐️ 推荐使用
func (s *menuCacheService) GetAllWithCache(ctx context.Context) ([]*models.Menu, error)

// 获取菜单树（带缓存）⭐️ 推荐使用
func (s *menuCacheService) GetTreeWithCache(ctx context.Context) ([]models.Menu, error)

// 失效缓存
func (s *menuCacheService) InvalidateCache(ctx context.Context) error
```

---

## 缓存失效机制

### 自动失效

当数据发生变化时，缓存会自动失效。

**创建数据时**：
```go
func (s *roleCacheService) Create(ctx context.Context, req *RoleCreateRequest) error {
    if err := s.roleService.Create(ctx, req); err != nil {
        return err
    }
    return s.InvalidateRoleCache(ctx, "")
}
```

**更新数据时**：
```go
func (s *roleCacheService) Update(ctx context.Context, req *RoleUpdateRequest) error {
    if err := s.roleService.Update(ctx, req); err != nil {
        return err
    }
    return s.InvalidateRoleCache(ctx, req.ID)
}
```

**删除数据时**：
```go
func (s *roleCacheService) Delete(ctx context.Context, id string) error {
    if err := s.roleService.Delete(ctx, id); err != nil {
        return err
    }
    return s.InvalidateRoleCache(ctx, id)
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
使用通用工具函数删除缓存
    ↓
下次请求时重新查询数据库并缓存
```

### 通用缓存失效函数

**位置**：`internal/services/system/cache_utils.go`

```go
// 根据模式列表失效缓存（通用方法）
func InvalidateCacheByPattern(ctx context.Context, cache CacheProvider, patterns []string, module string)

// 根据键列表失效缓存（通用方法）
func InvalidateCacheByKey(ctx context.Context, cache CacheProvider, keys []string, module string)
```

---

## 缓存策略建议

### 1. 列表数据（读多写少）

**适用场景**：配置数据、基础数据、选择器数据

**缓存时间**：30 分钟

**策略**：全量缓存 + 内存筛选

```go
// 获取全量缓存
allPosts, err := s.GetAllWithCache(ctx)

// 内存筛选
filtered := s.filterPosts(allPosts, params)

// 内存分页
paged, total := paginate(filtered, params.Current, params.PageSize)
```

### 2. 树形结构数据

**适用场景**：部门树、菜单树

**缓存时间**：30 分钟

**策略**：全量缓存树形结构，前端处理展示

```go
err := s.cache.GetOrSet(ctx, "dept:tree:all", &tree, expiration, func() (interface{}, error) {
    return s.buildTree()
})
```

### 3. 关联数据

**适用场景**：角色菜单、角色部门

**缓存时间**：30 分钟

**策略**：按关联ID缓存

```go
cacheKey := services.GetRoleMenusKey(roleID)
err := s.cache.GetOrSet(ctx, cacheKey, &result, expiration, func() (interface{}, error) {
    return s.queryMenus(ctx, roleID)
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
core.Cache.Delete(ctx, "dept:tree")
```

**按模式清除**：
```go
InvalidateCacheByPattern(ctx, cache, []string{"role:*"}, "ROLE")
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

### 1. 使用接口而非具体实现

```go
// ✅ 正确 - 使用接口
type roleCacheService struct {
    cache CacheProvider  // 接口解耦
}

// ❌ 错误 - 直接依赖具体实现
type roleCacheService struct {
    dataCache *DataCacheService  // 强耦合
}
```

### 2. 优先缓存列表数据

列表查询涉及更多数据库操作，缓存收益更大

### 3. 全量缓存 + 内存筛选

对于有筛选条件的查询，缓存全量数据后在内存中筛选

### 4. 及时失效缓存

数据更新时立即清除相关缓存

### 5. 合理设置 TTL

根据数据变化频率设置合适的过期时间

### 6. 避免过度缓存

不是所有数据都需要缓存，合理选择

---

## 故障排查

### 缓存不生效

1. 检查 Redis 连接是否正常
2. 检查缓存键是否正确
3. 检查是否有筛选条件（有筛选时可能不使用缓存）
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

### System 模块缓存服务（新架构）
- CacheProvider 接口：`internal/services/system/cache_provider.go`
- 缓存工具：`internal/services/system/cache_utils.go`
- 角色缓存：`internal/services/system/role_cache_impl.go` ⭐️ 列表缓存
- 岗位缓存：`internal/services/system/post_cache_impl.go` ⭐️ 列表缓存
- 字典缓存：`internal/services/system/dict_cache_impl.go`
- 部门缓存：`internal/services/system/department_cache_impl.go`
- 菜单缓存：`internal/services/system/menu_cache_impl.go`

### Legacy 缓存服务（**已迁移完成，历史记录**）

Legacy 单文件缓存服务（每个实体一个独立文件）已**全部移除**，项目统一使用 System 模块的新 `CacheProvider` 架构。下列文件已删除，新代码不应再创建：

- ~~`internal/services/role_cache_service.go`~~
- ~~`internal/services/post_cache_service.go`~~
- ~~`internal/services/menu_cache_service.go`~~
- ~~`internal/services/dict_cache_service.go`~~
- ~~`internal/services/user_cache_service.go`~~
- ~~`internal/services/dept_service.go`~~

如需新增模块的缓存，请参考上方"System 模块缓存服务"的模式，使用 `CacheProvider` 接口。

### 核心管理
- 核心管理：`internal/core/core.go`
- 缓存接口：`pkg/cache/cache.go`
- Redis 实现：`pkg/cache/redis.go`
- 内存实现：`pkg/cache/memory.go`
- 多级缓存：`pkg/cache/redis.go`（`MultiLevelCache` 结构体所在）

### API 层
- 缓存管理 API：`internal/api/v1/monitor/cache_handler.go`
- 缓存管理路由：`internal/api/v1/monitor/cache_router.go`
