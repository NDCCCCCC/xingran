---
title: 加解密类型不匹配批量修复
status: pending
created: 2026-05-27
updated: 2026-05-27
quick_id: 260527-mpv
slug: crypto-type-fix
---

# 加解密类型不匹配批量修复

## 目标
按优先级修复审查报告中发现的所有 `*crypto.SM4Cipher` 类型不匹配问题，统一改为 `addomain.PasswordCipher` 接口。

## 背景
审查报告 `260527-crypto-type-audit` 发现 3 处与 `connection_pool.go` 相同的类型不匹配问题，导致服务初始化时传入 `nil`，无法进行密码加解密。

## 修复清单（按优先级）

### 🔴 高优先级 #1: RPA 凭证服务
**文件**:
- `internal/services/rpa/credential_service.go`
- `internal/services/rpa/service.go`

**修改**:
1. 字段类型: `sm4 *crypto.SM4Cipher` → `passwordCipher addomain.PasswordCipher`
2. 构造函数参数类型同步修改
3. Core 初始化: `nil` → `c.SM4Cipher`

### 🔴 高优先级 #2: 授权凭证服务
**文件**:
- `internal/services/auth_credential_service.go`
- `internal/api/v1/network/network_router.go`
- `internal/api/v1/network/network_export_handler.go`
- `internal/api/v1/network/batch_export_helper.go`

**修改**:
1. 字段类型: `sm4Cipher *crypto.SM4Cipher` → `passwordCipher addomain.PasswordCipher`
2. 构造函数参数类型同步修改
3. Router 初始化: `nil` → `core.SM4Cipher` (3 处)

### 🟡 中优先级 #3: 设备监控服务
**文件**:
- `internal/services/device_monitor_service.go`

**修改**:
1. 构造函数参数类型: `sm4Cipher *crypto.SM4Cipher` → `passwordCipher addomain.PasswordCipher`
2. Core 初始化: `nil` → `c.SM4Cipher`

## 执行策略

**按优先级分批修复**:
1. 先修复高优先级 #1（RPA 凭证服务）
2. 再修复高优先级 #2（授权凭证服务）
3. 最后修复中优先级 #3（设备监控服务）

**每个优先级**:
1. 修改服务文件（类型定义）
2. 修改初始化文件（传入正确的 cipher）
3. 运行 `go build ./...` 验证
4. 提交 atomic commit

## 预期结果
- ✅ 所有服务统一使用 `addomain.PasswordCipher` 接口
- ✅ 所有初始化调用传入 `c.SM4Cipher` 而非 `nil`
- ✅ 编译成功，无类型错误
- ✅ 密码加解密功能恢复正常
