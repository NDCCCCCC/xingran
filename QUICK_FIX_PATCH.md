# 补丁说明：修复 AD 域用户批量同步问题

## 问题1：dept_code 包含空格
**位置**：`internal/services/system/user_sync_service.go` 第460-472行

**原代码**：
```go
// 3. 从右到左处理部门层级（跳过域名部分）
var deptNames []string
for _, part := range ouParts {
    if !strings.Contains(part, "DC=") {
        // 提取 OU= 后面的名称
        name := strings.TrimPrefix(part, "OU=")
        deptNames = append(deptNames, name)
    }
}
```

**修复后**：
```go
// 3. 从右到左处理部门层级（跳过域名部分）
var deptNames []string
for _, part := range ouParts {
    if !strings.Contains(part, "DC=") {
        // 提取 OU= 后面的名称，并清理所有空格
        name := strings.TrimSpace(strings.TrimPrefix(part, "OU="))
        // 去除所有内部空格
        name = strings.ReplaceAll(name, " ", "")
        if name != "" {
            deptNames = append(deptNames, name)
        }
    }
}
```

## 问题2：部门创建时使用复杂的 deptCode 生成
**位置**：`internal/services/system/user_sync_service.go` 第486-514行

**原代码**：
```go
if err == gorm.ErrRecordNotFound {
    // 生成唯一的 DeptCode
    deptCode := s.generateUniqueDeptCode(ctx, deptName)
    
    // 部门不存在，创建新部门
    newDept := models.Department{
        DeptName:  deptName,
        DeptCode:  deptCode,  // 问题：可能带空格
        ...
    }
}
```

**修复后**：
```go
if err == gorm.ErrRecordNotFound {
    // 部门不存在，创建新部门
    // 直接使用 deptName 作为 dept_code（已清理空格，保证一致性）
    newDept := models.Department{
        DeptName:  deptName,
        DeptCode:  deptName,  // dept_code 与 dept_name 一致（已清理空格）
        ...
    }
}
```

## 问题3：处理软删除用户
**位置**：`internal/services/system/user_sync_service.go` 第52-86行

**添加**：在双重查找逻辑后添加第三层查找
```go
if errors.Is(err, gorm.ErrRecordNotFound) {
    // 3. 检查是否有被软删除的同名用户
    err := tx.Unscoped().Where("username = ? AND deleted_at IS NOT NULL", adUser.Username).First(&user).Error
    if err == nil {
        // 找到被软删除的用户，恢复并补充AD信息
        return s.restoreDeletedUserWithADInfo(tx, &user, adUser, defaultRoleID)
    }
    
    if errors.Is(err, gorm.ErrRecordNotFound) {
        // 4. 用户确实不存在，创建新用户
        return s.createNewUser(tx, &user, adUser, defaultRoleID)
    }
    
    return fmt.Errorf("查询用户失败: %w", err)
}
```

## 快速修复步骤

1. 打开文件：`internal/services/system/user_sync_service.go`

2. 找到第460-472行（resolveDeptFromOU 函数中的 OU 解析部分），修改为清理空格版本

3. 找到第486-514行（部门创建部分），移除 `generateUniqueDeptCode` 调用，直接使用 `deptName`

4. 找到第52-86行（SyncADUser 函数中的查找逻辑），添加软删除用户处理

5. 添加新函数 `restoreDeletedUserWithADInfo`（在第212行之后）

6. 重新编译：`go build -o ruoyi-backend.exe ./cmd/main.go`

7. 重启服务并测试
