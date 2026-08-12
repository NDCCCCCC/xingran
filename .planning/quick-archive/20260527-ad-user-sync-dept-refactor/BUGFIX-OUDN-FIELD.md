# 批量同步部门解析失败 - Bug 修复报告

## 问题描述

**症状**: 批量同步6个AD用户时，部门解析全部失败（6个用户中有4个失败，2个部分成功）

**根本原因**: `BatchSyncADUsers` 方法在转换为 `security.ADUserInfo` 时，**遗漏了 `OUDN` 字段的赋值**

### 问题分析

通过日志分析发现：
- 用户同步成功，但没有看到任何部门解析相关的日志
- 数据库中 `ad_ou_dn` 字段有值，但部门没有被设置

### 问题代码

**文件**: `internal/services/system/user_sync_service.go` (第265-274行)

```go
// ❌ 错误代码 - OUDN 字段缺失
for _, adUser := range users {
    adUserInfo := &security.ADUserInfo{
        UserDN:      adUser.UserDN,
        // 缺少 OUDN 字段！
        Username:    adUser.Username,
        DisplayName: adUser.DisplayName,
        ...
    }
}
```

### 修复方案

**文件**: `internal/services/system/user_sync_service.go`

```go
// ✅ 修复后代码 - 添加 OUDN 字段
for _, adUser := range users {
    adUserInfo := &security.ADUserInfo{
        UserDN:      adUser.UserDN,
        OUDN:        adUser.OuDn, // 重要：传递 OU DN 用于部门解析
        Username:    adUser.Username,
        DisplayName: adUser.DisplayName,
        ...
    }
}
```

## 修复验证

### 编译验证
```bash
go build ./...
```
✅ 编译成功，无错误

### 预期效果
修复后，批量同步流程会：
1. 正确传递 OUDN 字段到 SyncADUser
2. 自动调用 resolveDeptFromOU 解析部门
3. 输出部门解析日志（成功或失败）
4. 为用户设置正确的部门ID

## 影响范围

**影响功能**:
- AD用户批量同步 (`/api/v1/ad-domain/users/batch-sync`)
- 部门自动解析和设置

**不影响**:
- AD登录功能（已验证正常工作）
- 单个用户同步

## 测试建议

1. 重新同步之前的6个用户，验证部门是否正确设置
2. 检查日志输出是否包含部门解析相关信息
3. 验证不同OU层级的部门创建是否正常

## 相关文件

- `internal/services/system/user_sync_service.go` - 批量同步逻辑
- `internal/api/v1/system/ad_domain_user_sync_handler.go` - API 端点
- `internal/core/security/authenticator.go` - ADUserInfo 结构体定义

## 修复时间

2026-05-27 14:50:00
