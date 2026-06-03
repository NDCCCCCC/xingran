---
slug: ad-password-encryption-decryption-errors
status: resolved
trigger: 请系统性检查AD域控密码加解密逻辑，检查是否有重复的，未使用的代码逻辑
created: 2026-05-27
updated: 2026-05-27
session_type: bug
---

# Debug Session: AD Password Encryption/Decryption Logic Review

## Symptoms

### Expected Behavior
AD密码加解密逻辑应该是统一、无重复的，所有代码都被正确使用。密码的加密、保存、解密整个流程应该能正常工作，支持AD域控连接。

### Actual Behavior
现在密码的加密，保存，解密，整个流程中存在错误，导致无法连接AD域控，连接测试失败。需要系统性检查：
1. 是否有重复的加解密逻辑实现
2. 是否有未使用的加解密代码
3. 加解密流程中的错误

### Error Messages
SM4加密器始终为nil，导致密码加密/解密始终使用AES-legacy回退，且ADAuthenticator中的解密逻辑完全不工作

### Timeline
- 2026-05-27: 通过代码审查确定根因

### Reproduction
1. 创建AD域配置（密码会被AES-legacy加密）
2. 尝试连接测试（addomain包可解密，但ADAuthenticator不能）
3. 使用AD登录认证（密码解密失败，无法绑定管理员）

## Current Focus

**Hypothesis:** initSM4Cipher()始终返回nil，导致全局SM4加密器未初始化，ADAuthenticator没有注入SM4加密器
**Test:** 代码审查验证
**Expecting:** initSM4Cipher应创建真正的SM4加密器并传递到所有消费者
**Next Action:** 修复完成
**Reasoning Checkpoint:** 根因确认，修复已应用

## Evidence

- 2026-05-27: `initSM4Cipher()` 在 core.go:110-113 始终返回 nil（注释说"SM4 cipher 由 addomain 包内部管理"但实际从未管理）
- 2026-05-27: `AuthStrategyFactory.GetAuthenticator()` 创建 ADAuthenticator 时从未调用 SetSM4Cipher()
- 2026-05-27: `ADAuthenticator.decryptPassword()` 使用 `*crypto.SM4Cipher` 类型，与 `addomain.PasswordCipher` 接口不兼容
- 2026-05-27: 存在两套独立的 decryptPassword: addomain包(SM4+AES回退) 和 ADAuthenticator(仅SM4)
- 2026-05-27: `CryptoSM4Cipher` 字段在 Core 结构体中声明但从未初始化或使用（死代码）

## Eliminated

- 无其他潜在根因

## Resolution
**Root Cause:** `initSM4Cipher()` in core.go 硬编码返回 nil，导致 SM4 加密器从未被创建。这产生了连锁影响：(1) addomain 包的全局密码始终为 nil，只能用 AES-legacy 加解密；(2) ADAuthenticator 从未收到 SM4 加密器，其私有的 decryptPassword 在 cipher 为 nil 时直接返回加密字符串原文；(3) ADAuthenticator 的解密逻辑不包含 AES-legacy 回退，与 addomain 包的实现不一致。另外 CryptoSM4Cipher 是声明但从未使用的死代码。
**Fix:**
1. core.go: 修复 initSM4Cipher() 使用 crypto.NewSM4Cipher() 创建真正的 SM4 加密器；删除死代码 CryptoSM4Cipher 字段；在 initAuthFactory() 中将 SM4Cipher 传递给 AuthStrategyFactory
2. auth_strategy_factory.go: 新增 sm4Cipher 字段，NewAuthStrategyFactory 使用 variadic 参数接收；在创建 ADAuthenticator 后调用 SetPasswordCipher 注入加密器
3. ad_authenticator.go: 将 sm4Cipher 字段类型从 *crypto.SM4Cipher 改为 addomain.PasswordCipher 接口；将 SetSM4Cipher 改为 SetPasswordCipher；decryptPassword 方法先尝试本地 cipher，再回退到 addomain.DecryptPassword（含完整 SM4+AES 回退链）
**Verification:** go build ./... 编译通过，无错误
**Files Changed:**
- internal/core/core.go (修复 initSM4Cipher, 删除 CryptoSM4Cipher, 传递 cipher 给 AuthFactory)
- internal/core/security/auth_strategy_factory.go (接收并注入 SM4 cipher)
- internal/core/security/ad_authenticator.go (使用 PasswordCipher 接口, 统一解密逻辑)
