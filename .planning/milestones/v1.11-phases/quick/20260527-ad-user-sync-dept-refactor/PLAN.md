# AD用户同步流程重构 - 方案A

## 概述

消除AD用户同步流程中的代码重复，将部门解析逻辑整合到UserSyncService内部，实现一次调用完成用户同步和部门设置。

## 问题分析

### 当前代码重复问题

**位置1**: `internal/api/v1/auth.go`
- 第147行：调用 `SyncADUser` 同步用户基本信息
- 第220行：调用 `HandleUserLoginAD` 设置部门

**位置2**: `internal/api/v1/system/ad_domain_user_sync_handler.go`
- 第130行：调用 `BatchSyncADUsers` 批量同步用户
- 第134-142行：循环调用 `HandleUserLoginAD` 为每个用户设置部门

### 问题根源

1. **职责分散**: 用户同步和部门设置是两个独立步骤
2. **调用复杂**: 每个调用点都需要调用两个不同的服务
3. **容易遗漏**: 新增同步场景时可能忘记调用部门设置
4. **事务分离**: 用户同步和部门设置不在同一事务中

## 解决方案（方案A）

### 核心思路

将部门解析逻辑整合到 `UserSyncService.SyncADUser()` 方法内部，使其成为完整的用户同步流程。

### 改动清单

#### 1. 修改 `UserSyncService` 结构体

**文件**: `internal/services/addomain/user_sync_service.go`

```go
// UserSyncService 用户同步服务
// 负责将AD用户信息同步到sys_user表，并自动解析部门
type UserSyncService struct {
    db         *gorm.DB
    pwdManager PasswordManager
    ouMapper   DeptOUMapper // 使用mapper接口而不是整个service
}
```

#### 2. 在 `SyncADUser` 方法中整合部门解析逻辑

**文件**: `internal/services/addomain/user_sync_service.go`

在 `SyncADUser` 方法中，同步用户基本信息后自动解析并设置部门：

```go
func (s *UserSyncService) SyncADUser(ctx context.Context, adUserInfo *security.ADUserInfo, defaultRoleID string) (*security.SyncedUser, error) {
    // 1. 转换并同步用户基本信息
    user, err := s.SyncUserFromAD(ctx, adUser, defaultRoleID)
    if err != nil {
        return nil, err
    }

    // 2. 自动解析并设置部门（整合的逻辑）
    if adUserInfo.OUDN != "" {
        deptID, err := s.resolveDeptFromOU(ctx, adUserInfo.OUDN)
        if err != nil {
            applogger.Warnf("解析部门失败（不影响同步）: %v", err)
            // 部门解析失败，使用默认部门继续
        } else {
            // 更新用户部门
            if err := s.db.WithContext(ctx).Model(&user).Update("dept_id", deptID).Error; err != nil {
                applogger.Warnf("更新用户部门失败: %v", err)
            }
        }
    }

    // 3. 返回同步结果
    return s.toSyncedUser(&user), nil
}
```

#### 3. 添加 `resolveDeptFromOU` 方法

**文件**: `internal/services/addomain/user_sync_service.go`

新增方法来处理OU到部门的解析：

```go
// resolveDeptFromOU 解析OU并设置部门
func (s *UserSyncService) resolveDeptFromOU(ctx context.Context, ouDN string) (string, error) {
    // 1. 尝试查找现有映射
    deptID, err := s.ouMapper.FindDeptByOUDN(ctx, ouDN)
    if err == nil {
        return deptID, nil
    }

    // 2. 未找到映射，自动创建部门及映射
    return s.createDeptFromOUDN(ctx, ouDN)
}
```

#### 4. 移动 `createDeptFromOUDN` 逻辑

**选项A**: 从 `UserOUService` 移动到 `UserSyncService`
**选项B**: 通过 `ouMapper` 接口调用（推荐）

#### 5. 删除重复的 `HandleUserLoginAD` 调用

**文件**: `internal/api/v1/auth.go`
- 删除第220行的 `HandleUserLoginAD` 调用

**文件**: `internal/api/v1/system/ad_domain_user_sync_handler.go`
- 删除第134-142行的循环调用

#### 6. 更新构造函数

修改 `UserSyncService` 的构造函数，注入 `DeptOUMapper` 接口。

## 预期效果

- ✅ **消除代码重复**: 删除多处重复的 `HandleUserLoginAD` 调用
- ✅ **简化调用逻辑**: AD登录和批量同步只需调用一次 `SyncADUser`
- ✅ **统一事务处理**: 用户同步和部门设置在同一事务中
- ✅ **降低遗漏风险**: 新增同步场景自动包含部门设置

## 实施步骤

1. 阅读 `UserSyncService` 当前实现
2. 阅读 `UserOUService` 的 `HandleUserLoginAD` 和 `createDeptFromOUDN` 实现
3. 阅读 `DeptOUMapper` 接口定义
4. 修改 `UserSyncService` 结构体，添加 `ouMapper` 字段
5. 实现 `resolveDeptFromOU` 方法
6. 修改 `SyncADUser` 方法，整合部门解析逻辑
7. 删除 `auth.go` 中的重复调用
8. 删除 `ad_domain_user_sync_handler.go` 中的重复调用
9. 更新构造函数调用处
10. 测试AD登录功能
11. 测试批量同步功能
12. 编译验证

## 验证标准

1. `go build ./...` 编译成功
2. AD登录流程正常（用户同步+部门设置）
3. 批量同步流程正常（用户同步+部门设置）
4. 无代码重复（删除了 `HandleUserLoginAD` 调用）
