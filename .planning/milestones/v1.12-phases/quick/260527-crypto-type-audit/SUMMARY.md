---
title: 加解密类型一致性审查报告
status: complete
created: 2026-05-27
updated: 2026-05-27
quick_id: 260527-mln
slug: crypto-type-audit
---

# 加解密类型一致性审查报告

## 审查目标
排查所有加解密相关的逻辑，检查是否存在类似 `connection_pool.go` 的类型不匹配问题。

## 审查结果
**发现 3 处严重的类型不匹配问题**，与刚才修复的 `connection_pool.go` 问题相同模式。

---

## 问题清单

### 🔴 问题 #1: RPA 凭证服务
**文件**: `internal/services/rpa/credential_service.go`
**位置**: 第 61 行、第 66 行

**问题描述**:
```go
type credentialServiceImpl struct {
    sm4   *crypto.SM4Cipher  // ❌ 具体类型
}

func NewCredentialService(db *gorm.DB, sm4 *crypto.SM4Cipher, cache CredentialCache)
```

**初始化位置**: `internal/core/core.go:627`
```go
rpaServices := rpa.NewServiceGroup(c.GetDB(), c.Config, c.NoticeHub, c.Cache, nil)
```
**问题**: 传入 `nil`，导致凭证加密/解密失败

---

### 🔴 问题 #2: 授权凭证服务
**文件**: `internal/services/auth_credential_service.go`
**位置**: 第 15 行、第 19 行

**问题描述**:
```go
type AuthCredentialService struct {
    sm4Cipher *crypto.SM4Cipher  // ❌ 具体类型
}

func NewAuthCredentialService(db *gorm.DB, sm4Cipher *crypto.SM4Cipher)
```

**初始化位置**:
- `internal/api/v1/network/network_router.go:25`
- `internal/api/v1/network/network_export_handler.go:124`
- `internal/api/v1/network/batch_export_helper.go:191`

**问题**: 所有调用点都传入 `nil`，导致网络设备凭证无法解密

---

### 🔴 问题 #3: 设备监控服务
**文件**: `internal/services/device_monitor_service.go`
**位置**: 第 50 行

**问题描述**:
```go
func NewDeviceMonitorService(db *gorm.DB, sm4Cipher *crypto.SM4Cipher, config *DeviceMonitorConfig)
```

**初始化位置**: `internal/core/core.go:312`
```go
c.DeviceMonitorService = services.NewDeviceMonitorService(c.GetDB(), nil, services.DefaultDeviceMonitorConfig())
```
**问题**: 传入 `nil`，可能影响设备监控功能

---

## 根本原因

与 `connection_pool.go` 问题相同：

1. **类型不一致**: 服务使用 `*crypto.SM4Cipher` 具体类型，而 Core 提供 `addomain.PasswordCipher` 接口
2. **初始化传入 nil**: 由于类型不兼容，初始化时传入 `nil` 而不是 `c.SM4Cipher`
3. **运行时失败**: 所有密码解密操作失败，因为加密器未初始化

---

## 修复方案

### 统一修复策略

**将所有服务改为使用 `addomain.PasswordCipher` 接口**：

```go
// 修改前
sm4Cipher *crypto.SM4Cipher

// 修改后
passwordCipher addomain.PasswordCipher
```

### 需要修改的文件

| 文件 | 修改内容 |
|------|----------|
| `internal/services/rpa/credential_service.go` | 字段类型 + 构造函数参数 |
| `internal/services/rpa/service.go` | 构造函数参数 |
| `internal/services/auth_credential_service.go` | 字段类型 + 构造函数参数 |
| `internal/services/device_monitor_service.go` | 构造函数参数 |
| `internal/core/core.go` | 初始化调用（nil → c.SM4Cipher） |
| `internal/api/v1/network/network_router.go` | 初始化调用（nil → core.SM4Cipher） |
| `internal/api/v1/network/network_export_handler.go` | 初始化调用（nil → core.SM4Cipher） |
| `internal/api/v1/network/batch_export_helper.go` | 初始化调用（nil → core.SM4Cipher） |

---

## 影响范围

| 服务 | 影响 | 严重性 |
|------|------|--------|
| RPA 凭证服务 | RPA 任务无法使用凭证加密存储 | 🔴 高 |
| 授权凭证服务 | 网络设备凭证无法解密 | 🔴 高 |
| 设备监控服务 | 可能影响设备监控密码解密 | 🟡 中 |

---

## 建议优先级

1. **立即修复**: RPA 凭证服务、授权凭证服务（影响核心功能）
2. **尽快修复**: 设备监控服务（预防性修复）
3. **代码审查**: 检查其他可能的类型不匹配

---

## 预防措施

1. **类型安全**: 统一使用 `addomain.PasswordCipher` 接口，避免直接使用 `*crypto.SM4Cipher`
2. **编译检查**: `go build ./...` 无法检测运行时 `nil` 问题，需要代码审查
3. **依赖注入**: 确保 Core 初始化时所有依赖正确传递
4. **文档规范**: 在 Core.go 中添加注释说明 SM4Cipher 的类型和用法
