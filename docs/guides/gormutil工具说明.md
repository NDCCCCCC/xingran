# GORM工具包使用指南

本工具包位于`pkg/gormutil`，用于解决GORM中的N+1查询问题，提供预加载、JOIN构建和批量查询等功能。

## 组件概览

### 1. PreloadBuilder - 预加载构建器

用于管理GORM的Preload操作，支持链式调用和条件预加载。

**使用示例：**

```go
import "github.com/xingran-next/xingran-go-backend/pkg/gormutil"

// 简单预加载
preloader := gormutil.NewPreloadBuilder()
preloader.Add("Dept")
preloader.Add("Roles")
db := preloader.Apply(query)

// 带条件的预加载（Add 返回 *PreloadBuilder，WithCondition 定义在 *PreloadItem 上，需用闭包 AddFunc 注册）
preloader := gormutil.NewPreloadBuilder()
preloader.AddFunc(func(db *gorm.DB) *gorm.DB {
    return db.Preload("Roles", "status = ?", 0)  // 直接调用 GORM Preload
})
db := preloader.Apply(query)
```

### 2. JoinBuilder - JOIN构建器

用于构建复杂的SQL JOIN查询，支持INNER JOIN、LEFT JOIN、RIGHT JOIN。

**使用示例：**

```go
import "github.com/xingran-next/xingran-go-backend/pkg/gormutil"

// 创建JOIN构建器
builder := gormutil.NewJoinBuilder(db.Model(&User{}))

// 添加LEFT JOIN
builder.LeftJoin("sys_department", "sys_department.id = users.dept_id")

// 添加更多条件和字段
builder.Select("users.*", "sys_department.dept_name").
    Where("users.status = ?", 0).
    Order("users.created_at DESC").
    Limit(10)

// 执行查询
var users []User
result := builder.Find(&users)
```

### 3. ResultMapper - 结果映射工具

提供切片映射、过滤、分组等函数式编程工具。

**使用示例：**

```go
import "github.com/xingran-next/xingran-go-backend/pkg/gormutil"

// 提取ID列表
userIDs := gormutil.ExtractIDs(users, func(u User) string {
    return u.ID
})

// 按字段分组
userRoleMap := gormutil.GroupBy(userRoles, func(ur UserRole) string {
    return ur.UserID
})

// 切片映射
roleNames := gormutil.MapSlice(roles, func(r Role) string {
    return r.RoleName
})

// 切片过滤
activeUsers := gormutil.FilterSlice(users, func(u User) bool {
    return u.Status == 0
})

// 构建ID映射
roleMap := gormutil.ToIDMap(roles, func(r Role) string {
    return r.ID
})
```

## N+1查询优化实例

### 优化前（存在N+1问题）

```go
// 查询用户列表
db.Find(&users)

// 循环查询每个用户的角色（N+1问题）
for _, user := range users {
    db.Preload("Roles").First(&user) // 每个用户一次查询！
}
```

### 优化后（使用JOIN一次查询）

```go
// 使用JOIN一次性查询用户角色和角色信息
type UserRoleResult struct {
    UserID   string
    RoleID   string
    RoleName string
}

var results []UserRoleResult
db.Table("sys_user_role").
    Select("sys_user_role.user_id, sys_user_role.role_id, sys_role.role_name").
    Joins("INNER JOIN sys_role ON sys_user_role.role_id = sys_role.id").
    Where("sys_user_role.user_id IN ?", userIDs).
    Find(&results)

// 使用工具函数分组
userRoleMap := gormutil.GroupBy(results, func(r UserRoleResult) string {
    return r.UserID
})
```

## 性能对比

假设查询100个用户及其角色：

| 方法 | 查询次数 | 说明 |
|------|----------|------|
| 未优化（N+1） | 1 + 100 + 100 = 201 | 1次用户 + 100次用户角色 + 100次角色 |
| 优化后（预加载） | 3 | 1次用户 + 1次用户角色 + 1次角色 |
| 优化后（JOIN） | 1 | 使用JOIN一次查询所有数据 |

## 最佳实践

1. **优先使用Preload**：对于简单的关联关系，使用Preload最方便
2. **批量查询用JOIN**：对于多对多关系，使用JOIN可以减少查询次数
3. **使用工具函数**：利用ResultMapper的工具函数简化代码
4. **注意索引**：确保JOIN的字段有索引，否则性能反而下降
5. **监控SQL**：使用GORM的Debug模式查看实际执行的SQL

```go
// 查看实际SQL
db.Debug().Preload("Roles").Find(&users)
```

## 注意事项

1. Preload会执行单独的SQL查询，适合关联数据不多的情况
2. JOIN适合数据量大、需要复杂条件的场景
3. 使用JOIN时注意字段名冲突，可以使用别名或Select指定字段
4. 对于超大数据集，考虑分页或游标分页

## 文件说明

- `preload_helper.go` - 预加载构建器
- `join_builder.go` - JOIN构建器
- `batch_loader.go` - 批量加载器
- `result_mapper.go` - 结果映射工具

这些工具可以组合使用，灵活应对各种查询场景。
