# 错误处理使用指南

## 概述

本包提供了统一的错误处理和错误码体系，用于XingRan-Next项目的所有模块。

## 基础概念

### 错误码分类

错误码按模块划分，每个模块占用1000个码段：

- **0**: 成功
- **1000-1999**: 系统级通用错误
- **2000-2999**: 用户权限模块
- **3000-3999**: 运维管理模块
- **4000-4999**: 调度任务模块
- **5000-5999**: 工单模块
- **6000-6999**: 监控模块
- **7000-7999**: 网络设备模块
- **8000-8999**: 知识库模块
- **9000-9999**: 值班管理模块

## 使用方法

### 1. 基础用法

#### 创建新错误

```go
import apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"

// 使用预定义的错误
err := apperrors.UserNotFound()

// 使用自定义消息
err := apperrors.New(apperrors.CodeUserNotFound, "用户不存在")
```

#### 包装错误

```go
// 包装底层错误
if err := db.First(&user).Error; err != nil {
    return apperrors.Wrap(err, apperrors.CodeDatabaseError, "查询用户失败")
}

// 包装并指定HTTP状态码
if err := db.First(&user).Error; err != nil {
    return apperrors.WrapWithHTTPStatus(err, apperrors.CodeUserNotFound, http.StatusNotFound, "用户不存在")
}
```

### 2. 便捷函数

每个错误类型都有对应的便捷函数，可以直接调用：

```go
// 用户相关
apperrors.UserNotFound()
apperrors.UserExists()
apperrors.UserNotFoundWithID("123")
apperrors.UserExistsWithUsername("admin")

// 角色相关
apperrors.RoleNotFound()
apperrors.RoleHasUsers()

// 部门相关
apperrors.DeptNotFound()
apperrors.DeptHasChildren()

// 楼宇相关
apperrors.BuildingNotFound()
apperrors.BuildingOrgInvalid()
apperrors.BuildingOrgInvalidWithMsg("关联的组织ID格式不正确")

// 数据库错误
apperrors.DatabaseError(err)

// 参数错误
apperrors.ParamError()
apperrors.ParamMissing("username")
apperrors.ParamInvalid("email")
```

### 3. 添加上下文信息

错误可以携带额外的上下文信息，方便调试和日志记录：

```go
err := apperrors.UserNotFound().
    WithContext("user_id", "123").
    WithContext("username", "testuser")

// 或者批量添加
err := apperrors.UserNotFound().WithContexts(map[string]interface{}{
    "user_id": "123",
    "username": "testuser",
    "request_id": "abc-123",
})
```

### 4. 错误判断

```go
// 检查是否是AppError
if apperrors.IsAppError(err) {
    appErr := apperrors.GetAppError(err)
    // 处理应用错误
}

// 获取错误码
code := apperrors.GetErrorCode(err)

// 错误码判断
if appErr := apperrors.GetAppError(err); appErr != nil {
    switch appErr.Code {
    case apperrors.CodeUserNotFound:
        // 处理用户不存在
    case apperrors.CodeDatabaseError:
        // 处理数据库错误
    }
}
```

### 5. 在Service中使用

```go
func (s *userService) GetUser(ctx context.Context, id string) (*models.User, error) {
    var user models.User
    if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, apperrors.UserNotFoundWithID(id)
        }
        return nil, apperrors.DatabaseError(err)
    }
    return &user, nil
}

func (s *userService) CreateUser(ctx context.Context, req *CreateUserRequest) error {
    // 检查用户名是否已存在
    var count int64
    if err := s.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
        return apperrors.DatabaseError(err)
    }
    if count > 0 {
        return apperrors.UserExistsWithUsername(req.Username)
    }

    // 创建用户...
    return nil
}
```

### 6. 在Handler中使用

```go
func (h *UserHandler) GetUser(c *gin.Context) {
    id := c.Param("id")

    user, err := h.userService.GetUser(c.Request.Context(), id)
    if err != nil {
        // response包会自动处理AppError类型
        response.Error(c, err)
        return
    }

    response.Success(c, user)
}
```

## 错误码规范

### 添加新错误码

1. 确定错误所属模块和码段
2. 在`pkg/errors/codes.go`中添加错误码常量
3. 在`DefaultMessage()`方法中添加对应的错误消息
4. 在`pkg/errors/errors.go`中添加便捷构造函数（可选）

### 示例：添加新的错误码

```go
// 1. 在codes.go中定义常量
const (
    CodeCustomError ErrorCode = 3015 // 自定义错误
)

// 2. 在DefaultMessage()中添加消息
func (c ErrorCode) DefaultMessage() string {
    // ...
    case CodeCustomError:
        return "自定义错误消息"
    // ...
}

// 3. 在errors.go中添加便捷函数（可选）
func CustomError() *AppError {
    return New(CodeCustomError, CodeCustomError.DefaultMessage())
}

func CustomErrorWithMsg(msg string) *AppError {
    return New(CodeCustomError, msg)
}
```

## 最佳实践

### 1. 错误粒度

- 使用具体的错误码，而不是通用的错误码
- 例如：使用`CodeUserNotFound`而不是`CodeRecordNotFound`

### 2. 错误消息

- 优先使用预定义的错误消息
- 需要时使用自定义消息提供更多信息
- 例如：`UserNotFoundWithID(id)`比`UserNotFound()`提供更多信息

### 3. 错误包装

- 总是包装底层错误，保留错误链
- 使用`Wrap()`而不是直接返回新错误
- 这样可以使用`errors.Is()`和`errors.As()`进行错误检查

### 4. 错误处理

- 在Service层使用AppError
- 在Handler层使用response.Error()自动转换为HTTP响应
- 不要在Service层直接处理HTTP相关逻辑

### 5. 上下文信息

- 对于需要调试的错误，添加上下文信息
- 不要在错误消息中包含敏感信息（如密码、token等）

## 迁移指南

### 从旧的错误处理迁移

**之前：**
```go
if err != nil {
    return fmt.Errorf("用户不存在")
}
```

**之后：**
```go
if err != nil {
    return apperrors.UserNotFound()
}
```

**之前：**
```go
if err := db.First(&user).Error; err != nil {
    return fmt.Errorf("查询失败: %w", err)
}
```

**之后：**
```go
if err := db.First(&user).Error; err != nil {
    return apperrors.DatabaseError(err)
}
```

## 完整示例

```go
package service

import (
    "context"
    apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
    "gorm.io/gorm"
)

type UserService struct {
    db *gorm.DB
}

func (s *UserService) UpdateUserEmail(ctx context.Context, userID, email string) error {
    // 1. 验证参数
    if email == "" {
        return apperrors.ParamMissing("email")
    }

    // 2. 检查用户是否存在
    var user models.User
    if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return apperrors.UserNotFoundWithID(userID)
        }
        return apperrors.DatabaseError(err)
    }

    // 3. 检查邮箱是否被其他用户使用
    var count int64
    if err := s.db.Model(&models.User{}).
        Where("email = ? AND id != ?", email, userID).
        Count(&count).Error; err != nil {
        return apperrors.DatabaseError(err)
    }
    if count > 0 {
        return apperrors.New(apperrors.CodeUserExists, "邮箱已被其他用户使用")
    }

    // 4. 更新邮箱
    if err := s.db.Model(&user).Update("email", email).Error; err != nil {
        return apperrors.DatabaseError(err)
    }

    // 5. 清除缓存
    // s.cache.Invalidate(userID)

    return nil
}
```
