---
phase: quick
plan: 260527-gdc
subsystem: ad-auth
tags: [bugfix, sm4, encryption, ad-domain]
dependency_graph:
  requires: []
  provides: [bugfix-ad-password-encryption]
  affects: [ad-authentication, ad-sync, ad-config]
tech_stack:
  added: []
  patterns: [PasswordCipher-interface, decrypt-fallback-chain]
key_files:
  created: []
  modified:
    - internal/core/core.go
    - internal/core/security/ad_authenticator.go
    - internal/core/security/auth_strategy_factory.go
    - internal/services/addomain/config.go
decisions:
  - "使用 addomain.PasswordCipher 接口替代具体 *crypto.SM4Cipher 类型，解耦加密器实现"
  - "保留 AES-legacy 回退链以确保向后兼容已存储的旧格式密码"
metrics:
  duration: 36s
  completed: 2026-05-27
---

# Phase quick Plan 260527-gdc: AD Password Encryption Git Commit Summary

修复 initSM4Cipher() 硬编码返回 nil 导致 AD 域控密码加解密完全失效的问题，并提交到版本控制。

## Changes Made

### Task 1: 验证编译并提交 AD 密码加解密修复

**Commit:** cc2963b

**4 files modified (72 insertions, 21 deletions):**

| File | Changes |
|------|---------|
| `internal/core/core.go` | `initSM4Cipher()` 使用 `crypto.NewSM4Cipher()` 替代硬编码 nil；删除未使用的 `CryptoSM4Cipher` 字段；`initAuthFactory()` 将 SM4Cipher 传递给 AuthStrategyFactory |
| `internal/core/security/ad_authenticator.go` | `sm4Cipher` 类型改为 `PasswordCipher` 接口；`decryptPassword` 含 SM4 + AES-legacy 回退链；添加认证流程调试日志 |
| `internal/core/security/auth_strategy_factory.go` | `NewAuthStrategyFactory` 接受可选 `PasswordCipher` 参数；在创建 AD 和 Hybrid 认证器时注入 cipher |
| `internal/services/addomain/config.go` | `TestConnection` 添加连接测试调试日志 |

## Verification

- Build check: Pre-existing `SetupADGroupMappingRouter` error (unrelated, out of scope)
- Commit verified: `cc2963b` with exactly 4 files changed
- No unexpected file deletions
- No untracked files left behind

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None - no new security surface introduced. The changes fix existing broken encryption functionality.

## Self-Check: PASSED

- FOUND: internal/core/core.go
- FOUND: internal/core/security/ad_authenticator.go
- FOUND: internal/core/security/auth_strategy_factory.go
- FOUND: internal/services/addomain/config.go
- FOUND: cc2963b (commit)
