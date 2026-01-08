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
| `cache.user.list` | 用户列表缓存时间 | 10 | 5-60 | 用户列表数据的缓存时间 |
| `cache.user.byid` | 用户详情缓存时间 | 30 | 5-120 | 用户详情数据的缓存时间 |
| `cache.menu.tree` | 菜单树缓存时间 | 30 | 5-120 | 菜单树结构数据的缓存时间 |
| `cache.menu.router` | 菜单路由缓存时间 | 30 | 5-120 | 菜单路由数据的缓存时间 |

### 使用方式

**1. 通过 API 修改缓存时间**

```bash
# 获取所有缓存配置
GET /monitor/cache/config

# 更新缓存配置
PUT /monitor/cache/config
{
  "key": "cache.dept.tree",
  "value": 60  # 设置为 60 分钟
}

# 重新加载配置（使修改生效）
POST /monitor/cache/config/reload
```

**2. 通过参数管理页面修改**

1. 登录系统管理后台
2. 进入"系统管理" → "参数管理"
3. 搜索 `cache.` 开头的配置项
4. 修改配置值（单位：分钟）
5. 保存后自动生效

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

- **多级缓存**：L1（内存缓存）+ L2（Redis缓存）
- **统一管理**：所有缓存服务在 Core 中统一初始化和管理
- **自动序列化**：支持 JSON 序列化/反序列化
- **缓存失效**：支持按键删除和按模式批量删除
- **高性能**：读多写少场景性能提升显著

### 架构层次

```
Handler (dept_handler.go)
    ↓ 使用
Core.DeptService (已初始化的单例)
    ↓ 包含
DataCacheService (通用缓存服务)
    ↓ 使用
Core.Cache (多级缓存: L1内存 + L2 Redis)
```

## 已实现的缓存服务

### 1. 部门缓存服务 (DeptService)

**位置**：`internal/services/dept_service.go`

**缓存键**：
- `dept:tree` - 部门树缓存
- `dept:list` - 部门列表缓存
- `dept:tree:select` - 部门树选择器缓存

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

**使用示例**：
```go
// 在 Handler 中使用
func getDeptTree(core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        depts, err := core.DeptService.GetDeptTreeWithCache(c.Request.Context(), false)
        if err != nil {
            response.Error(c, response.ErrServerError, err.Error())
            return
        }
        response.Success(c, depts)
    }
}
```

### 2. 角色缓存服务 (RoleCacheService)

**位置**：`internal/services/role_cache_service.go`

**缓存键**：
- `role:menus:{roleId}` - 角色菜单权限缓存
- `role:menu:tree:{roleId}` - 角色菜单树缓存
- `role:depts:{roleId}` - 角色部门权限缓存

**缓存时间**：30 分钟

**主要方法**：
```go
// 获取角色菜单权限（带缓存）
func (s *RoleCacheService) GetRoleMenusWithCache(ctx context.Context, roleID string) ([]models.Menu, error)

// 获取角色菜单树（带缓存）
func (s *RoleCacheService) GetRoleMenuTreeWithCache(ctx context.Context, roleID string) ([]models.Menu, error)

// 获取角色部门权限（带缓存）
func (s *RoleCacheService) GetRoleDeptsWithCache(ctx context.Context, roleID string) ([]*models.Department, error)

// 清除角色缓存
func (s *RoleCacheService) InvalidateRoleCache(ctx context.Context, roleID string) error
```

### 3. 字典缓存服务 (DictCacheService)

**位置**：`internal/services/dict_cache_service.go`

**缓存键**：
- `dict:type` - 字典类型缓存
- `dict:data:type:{dictType}` - 按类型查询字典数据缓存

**缓存时间**：30 分钟

**主要方法**：
```go
// 获取所有字典类型（带缓存）
func (s *DictCacheService) GetDictTypesWithCache(ctx context.Context) ([]*models.DictType, error)

// 获取字典数据（带缓存）
func (s *DictCacheService) GetDictDataByTypeWithCache(ctx context.Context, dictType string) ([]*models.DictData, error)

// 清除字典缓存
func (s *DictCacheService) InvalidateDictCache(ctx context.Context) error
```

## 缓存失效机制

### 自动失效

当数据发生变化时，缓存会自动失效。以部门服务为例：

**创建部门时**：
```go
// dept_handler.go
func createDept(core *core.Core) gin.HandlerFunc {
    // ...
    // 使用 DeptService 创建部门（会自动清除缓存）
    if err := core.DeptService.CreateDept(c.Request.Context(), &dept); err != nil {
        response.Error(c, response.ErrServerError, "创建部门失败")
        return
    }
}
```

**更新部门时**：
```go
// dept_handler.go
func updateDept(core *core.Core) gin.HandlerFunc {
    // ...
    // 提交事务后清除缓存
    if err := core.DeptService.InvalidateDeptCache(c.Request.Context()); err != nil {
        fmt.Printf("清除部门缓存失败: %v\n", err)
    }
}
```

**删除部门时**：
```go
// dept_handler.go
func deleteDept(core *core.Core) gin.HandlerFunc {
    // ...
    // 删除后清除缓存
    if err := core.DeptService.InvalidateDeptCache(c.Request.Context()); err != nil {
        fmt.Printf("清除部门缓存失败: %v\n", err)
    }
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
Service 调用 InvalidateDeptCache()
    ↓
删除相关缓存键
    ↓
下次请求时重新查询数据库并缓存
```

### 清除的缓存键

部门服务清除的缓存键包括：
- `dept:tree` - 部门树缓存
- `dept:list` - 部门列表缓存
- `dept:tree:select` - 部门树选择器缓存
- `dept:tree:all` - 包含所有状态的部门树缓存

## 为新模块添加缓存

### 步骤 1：定义缓存键常量

在 `internal/services/data_cache_service.go` 中添加新模块的缓存键：

```go
const (
    // 现有的缓存键...
    CacheKeyDeptTree         = "dept:tree"

    // 新增：用户相关缓存键
    CacheKeyUserList         = "user:list"
    CacheKeyUserByID         = "user:id"

    // 新增：菜单相关缓存键
    CacheKeyMenuTree         = "menu:tree"
)

// GetUserByIDKey 获取用户缓存键
func GetUserByIDKey(userID string) string {
    return fmt.Sprintf("%s:%s", CacheKeyUserByID, userID)
}
```

### 步骤 2：创建带缓存的服务

创建新的服务文件：

```go
// internal/services/user_service.go

package services

import (
    "context"
    "fmt"
    "time"

    "github.com/ruoyi-next/ruoyi-go-backend/internal/models"
    "gorm.io/gorm"
)

// UserService 用户服务（带缓存）
type UserService struct {
    db        *gorm.DB
    dataCache *DataCacheService
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, dataCache *DataCacheService) *UserService {
    return &UserService{
        db:        db,
        dataCache: dataCache,
    }
}

// GetUserByIDWithCache 根据ID获取用户（带缓存）
func (s *UserService) GetUserByIDWithCache(ctx context.Context, userID string) (*models.User, error) {
    cacheKey := GetUserByIDKey(userID)

    var user models.User

    // 使用 GetOrSet 实现 Cache-Aside 模式
    err := s.dataCache.GetOrSet(ctx, cacheKey, &user, 30*time.Minute, func() (interface{}, error) {
        return s.queryUserByID(userID)
    })

    if err != nil {
        return nil, fmt.Errorf("获取用户失败: %w", err)
    }

    return &user, nil
}

// queryUserByID 从数据库查询用户
func (s *UserService) queryUserByID(userID string) (*models.User, error) {
    var user models.User
    if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
        return nil, fmt.Errorf("查询用户失败: %w", err)
    }
    return &user, nil
}

// InvalidateUserCache 清除用户相关缓存
func (s *UserService) InvalidateUserCache(ctx context.Context, userID string) error {
    if err := s.dataCache.Delete(ctx, GetUserByIDKey(userID)); err != nil {
        return fmt.Errorf("删除用户缓存失败: %w", err)
    }

    if err := s.dataCache.Delete(ctx, CacheKeyUserList); err != nil {
        return fmt.Errorf("删除用户列表缓存失败: %w", err)
    }

    return nil
}
```

### 步骤 3：注册服务到 Core

修改 `internal/core/core.go`：

```go
// 1. 添加服务字段
type Core struct {
    // ... 其他字段

    // 数据缓存服务
    DataCacheService        *services.DataCacheService
    DeptService             *services.DeptService
    UserService             *services.UserService  // 新增
}

// 2. 在 Init() 中初始化
func (c *Core) Init() error {
    // ... 其他初始化代码

    // 初始化缓存服务
    if c.Cache != nil {
        c.DataCacheService = services.NewDataCacheService(c.Cache)
        c.DeptService = services.NewDeptService(c.GetDB(), c.DataCacheService)
        c.UserService = services.NewUserService(c.GetDB(), c.DataCacheService)  // 新增
        log.Println("数据缓存服务初始化完成")
    }

    return nil
}
```

### 步骤 4：在 Handler 中使用

```go
func getUserInfo(core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.Param("id")

        // 使用缓存服务
        user, err := core.UserService.GetUserByIDWithCache(c.Request.Context(), userID)
        if err != nil {
            response.Error(c, response.ErrServerError, err.Error())
            return
        }

        response.Success(c, user)
    }
}

func updateUser(core *core.Core) gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... 更新逻辑

        // 清除缓存
        if err := core.UserService.InvalidateUserCache(c.Request.Context(), userID); err != nil {
            log.Printf("清除用户缓存失败: %v", err)
        }

        response.Success(c, "更新成功")
    }
}
```

## 缓存策略建议

### 1. 读多写少数据（如：部门、菜单、字典）

**适用场景**：配置数据、基础数据

**缓存时间**：30 分钟

**示例**：
```go
err := s.dataCache.GetOrSet(ctx, cacheKey, &result, 30*time.Minute, queryFunc)
```

### 2. 读多写多数据（如：用户列表）

**适用场景**：经常变动的列表数据

**缓存时间**：5-10 分钟

**示例**：
```go
err := s.dataCache.GetOrSet(ctx, cacheKey, &result, 5*time.Minute, queryFunc)
```

### 3. 统计数据（如：在线用户数）

**适用场景**：实时统计数据

**缓存时间**：1-5 分钟

**策略**：只缓存，不自动查询

```go
value, _ := s.dataCache.Get(ctx, "online:users")
```

### 4. 热点数据（如：当前用户信息）

**适用场景**：频繁访问的单条数据

**缓存时间**：1 小时

**示例**：
```go
err := s.dataCache.GetOrSet(ctx, cacheKey, &user, 1*time.Hour, queryFunc)
```

## 缓存失效策略

### 1. 创建数据时

```go
func (s *UserService) CreateUser(ctx context.Context, user *models.User) error {
    if err := s.db.Create(user).Error; err != nil {
        return err
    }

    // 清除列表缓存
    s.dataCache.Delete(ctx, CacheKeyUserList)

    return nil
}
```

### 2. 更新数据时

```go
func (s *UserService) UpdateUser(ctx context.Context, user *models.User) error {
    if err := s.db.Save(user).Error; err != nil {
        return err
    }

    // 清除该用户的缓存
    s.InvalidateUserCache(ctx, user.ID)

    return nil
}
```

### 3. 删除数据时

```go
func (s *UserService) DeleteUser(ctx context.Context, userID string) error {
    if err := s.db.Delete(&models.User{}, "id = ?", userID).Error; err != nil {
        return err
    }

    // 清除该用户的缓存
    s.InvalidateUserCache(ctx, userID)

    return nil
}
```

### 4. 批量操作时

```go
func (s *UserService) BatchUpdateUsers(ctx context.Context, users []models.User) error {
    if err := s.db.Save(&users).Error; err != nil {
        return err
    }

    // 清除所有用户相关缓存
    s.InvalidateAllUserCache(ctx)

    return nil
}
```

## 缓存管理

### 查看缓存数据

访问缓存管理页面：`/monitor/cache`

可以查看：
- 所有缓存键
- 缓存值
- 缓存类型
- TTL（过期时间）
- 缓存大小

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

## 注意事项

### 1. 避免缓存穿透

对于不存在的数据，也要缓存空值：

```go
func (s *UserService) GetUserByIDWithCache(ctx context.Context, userID string) (*models.User, error) {
    cacheKey := GetUserByIDKey(userID)

    var user models.User
    err := s.dataCache.GetOrSet(ctx, cacheKey, &user, 30*time.Minute, func() (interface{}, error) {
        u, err := s.queryUserByID(userID)
        if err == gorm.ErrRecordNotFound {
            // 缓存空对象，避免缓存穿透
            return &models.User{}, nil
        }
        return u, err
    })

    // 检查是否为空对象
    if user.ID == "" {
        return nil, gorm.ErrRecordNotFound
    }

    return &user, nil
}
```

### 2. 避免缓存雪崩

为不同的缓存键设置不同的过期时间：

```go
// 添加随机偏移量，防止同时失效
randomOffset := time.Duration(rand.Intn(300)) * time.Second
ttl := 30*time.Minute + randomOffset

err := s.dataCache.GetOrSet(ctx, cacheKey, &result, ttl, queryFunc)
```

### 3. 缓存预热

系统启动时预加载热点数据：

```go
// 在 Core.Init() 中添加
func (c *Core) warmUpCache(ctx context.Context) {
    log.Println("开始预热缓存...")

    // 预加载部门树
    if _, err := c.DeptService.GetDeptTreeWithCache(ctx, false); err != nil {
        log.Printf("预热部门缓存失败: %v", err)
    }

    log.Println("缓存预热完成")
}
```

### 4. 监控缓存命中率

定期检查缓存命中率，优化缓存策略：

```go
// 在缓存服务中添加统计
type CacheStats struct {
    Hits   int64
    Misses int64
}

func (s *DataCacheService) GetHitRate() float64 {
    total := s.Hits + s.Misses
    if total == 0 {
        return 0
    }
    return float64(s.Hits) / float64(total) * 100
}
```

## 性能对比

### 无缓存 vs 有缓存

| 操作 | 无缓存 | 有缓存 | 提升 |
|------|--------|--------|------|
| 获取部门树 | ~50ms | ~2ms | 25x |
| 获取角色菜单 | ~30ms | ~2ms | 15x |
| 获取字典数据 | ~20ms | ~1ms | 20x |

### 压力测试结果

- **QPS 提升**：从 200 提升到 2000+
- **响应时间**：P99 从 100ms 降到 10ms
- **数据库负载**：降低 80%

## 最佳实践

1. **读多写少用缓存**：配置数据、基础数据优先使用缓存
2. **写多读少慎用缓存**：频繁变动的数据不推荐使用缓存
3. **合理设置 TTL**：根据数据变化频率设置合适的过期时间
4. **及时失效缓存**：数据更新时立即清除相关缓存
5. **监控缓存效果**：定期检查缓存命中率和性能指标
6. **避免过度缓存**：不是所有数据都需要缓存，合理选择

## 故障排查

### 缓存不生效

1. 检查 Redis 连接是否正常
2. 检查缓存键是否正确
3. 检查缓存时间是否设置过短
4. 检查是否及时清除缓存

### 性能没有提升

1. 检查缓存命中率
2. 检查查询条件是否过于动态
3. 检查缓存时间设置是否合理
4. 检查是否存在缓存穿透

### 内存占用过高

1. 检查缓存数据量是否过大
2. 检查是否有过期的缓存未清理
3. 考虑使用 LRU 淘汰策略
4. 调整缓存时间

## 相关文件

- 缓存服务：`internal/services/data_cache_service.go`
- 部门服务：`internal/services/dept_service.go`
- 角色服务：`internal/services/role_cache_service.go`
- 字典服务：`internal/services/dict_cache_service.go`
- 核心管理：`internal/core/core.go`
- 缓存接口：`pkg/cache/cache.go`
- Redis 实现：`pkg/cache/redis.go`
