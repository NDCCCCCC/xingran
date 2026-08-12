---
slug: user-sync-service-test-args
status: investigating
trigger: 'Side finding 1: user_sync_service_test.go - 构造函数缺参 + ADUser 未定义 + 返回值数量'
created: 2026-06-12
updated: 2026-06-12
---

# Side Finding 1: user_sync_service_test.go

## Symptoms

`go test -c -o /dev/null ./internal/services/system/` 报告 10+ 错误：

### 错误 A: NewUserSyncService 缺参
```
:27, :59, :105, :135: not enough arguments in call to NewUserSyncService
  have (*gorm.DB)
  want (*gorm.DB, PasswordManager, *addomain.DeptOUmapper)
```

### 错误 B: ADUser 未定义
```
:29, :73, :108: undefined: addomain.ADUser
```

### 错误 C: 返回值数量
```
:37, :81, :116: assignment mismatch: 1 variable but service.SyncUserFromAD returns 2 values
```

## Initial Hypothesis

测试是早期写的，`user_sync_service` 后来 refactor：
- `NewUserSyncService` 增加了 `PasswordManager` 和 `*addomain.DeptOUmapper` 参数
- `addomain.ADUser` 类型被重命名/移走
- `SyncUserFromAD` 改为返回 2 值（可能是 `(*result, error)` 或类似）

## Current Focus

- **hypothesis:** 3 类问题都是测试与新 API 不匹配；测试需要传 nil + 改类型 + 加 error 接收。
- **next_action:** 
  1. 找 `addomain.ADUser` 当前名（可能是 `addomain.User` / `addomain.ADUserInfo` 等）
  2. 看 `user_sync_service.SyncUserFromAD` 真实签名
  3. 修复 3 类错误
- **test:** `go test -c -o /dev/null ./internal/services/system/` 中 user_sync_service_test.go 错误清零
- **expecting:** 同 cluster 4 一样，nil 包装 + 可能加 t.Skip 防 panic
- **scope:** 用户说"全保留测试"，所以不能删测试函数

## Side findings
- (本文件之外的其它 side finding 不在本 session 范围)

## Evidence

- 2026-06-12: 找到 `NewUserSyncService(db *gorm.DB, pwdManager PasswordManager, ouMapper *addomain.DeptOUmapper)` 签名
- 2026-06-12: 找到 `SyncUserFromAD(ctx, adUser *ADUserInfoForSync, defaultRoleID string) (*models.User, error)`
- 2026-06-12: 找到 `ADUserInfoForSync` (本地类型，在 user_sync_service.go 第 18 行)，字段: UserDN/OuDn/Username/DisplayName/Email/Phone/Mobile/Title/Department
- 2026-06-12: 找到 `PasswordManager interface` 在 `internal/services/system/user_service.go:15`
- 2026-06-12: nil-deref 风险点：line 96 `s.pwdManager.HashPassword("123456")` 无 nil 检查；但 setupTestDBForSync 内 t.Skip 保证不会执行

## Eliminated

- ~~ADUser 类型在 addomain 包中需重命名~~ -> 实际是 `ADUserInfoForSync` 在 system 包内
- ~~需要 mock pwdManager/ouMapper 接口实现~~ -> t.Skip 已经保证代码不可达，传 nil 安全

## Resolution

- **root_cause:** 测试文件基于旧 API 编写，三个独立的 API 不匹配：
  1. `NewUserSyncService` 后增 `PasswordManager` 和 `*addomain.DeptOUmapper` 参数
  2. AD 用户类型从 `addomain.ADUser` 改为本地 `ADUserInfoForSync`
  3. `SyncUserFromAD` 改为返回 `(*models.User, error)` 双值
- **fix:**
  1. `NewUserSyncService(db)` -> `NewUserSyncService(db, nil, nil)` (PasswordManager / DeptOUmapper 用 nil 占位)
  2. `addomain.ADUser` -> `ADUserInfoForSync`（同包，移除 addomain import）
  3. `err := service.SyncUserFromAD(...)` -> `_, err := service.SyncUserFromAD(...)`
- **verification:**
  - `go test -c -o /dev/null ./internal/services/system/` 0 错误
  - `go build ./...` 0 错误
  - 6 个测试函数全部保留并 SKIP（`setupTestDBForSync` 内部 t.Skip 保护）
- **files_changed:** `internal/services/system/user_sync_service_test.go`
- **t.Skip 决策:** 不需要新增 t.Skip —— 原文件 line 16 已有 `t.Skip("测试数据库配置未实现")`，所有测试函数第一行就调用 `setupTestDBForSync`，会立即 skip，不会触达 nil-deref 代码路径
## Resolution

- root_cause:
- fix:
- verification:
- files_changed:
