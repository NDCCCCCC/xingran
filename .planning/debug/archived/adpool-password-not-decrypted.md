---
slug: adpool-password-not-decrypted
status: resolved
trigger: phase 36有一个ad域控管理员账号密码加解密的问题，请排查并修复。
created: 2026-06-23
updated: 2026-06-23
session_type: bug
---

# Debug Session: AD 账号池密码读取未解密（Phase 36）

## Symptoms

### Expected Behavior
Phase 36 账号池（`sys_ad_service_accounts.password_ciphertext`）中的账号密码应能正常用于
LDAP bind，支撑 AD 登录认证、AD 同步等所有依赖账号池的链路。

### Actual Behavior
账号池账号创建后连接 AD 始终失败（LDAP error 49 invalid credentials），账号陆续进入熔断
（status=2）；AD 域账号登录系统失败；AD 同步任务失败。三症状并发。

### Error Messages
LDAP bind 失败（error 49），连锁触发 AD 端账号锁定（data 775）与本地熔断（status=2）。

### Timeline
- 2026-06-23: 排查发现，根因为 Phase 36 账号池路径引入时遗漏解密步骤。

### Reproduction
1. 创建账号池账号（密码被 `core.SM4Cipher.Encrypt()` 加密存入 password_ciphertext）
2. 触发 AD 登录认证（ad_authenticator → FailoverClient.PickFirstConnect）
3. FailoverClient 遍历账号 → tryBindAttempts 直接用密文 bind → error 49 → MarkFailure
4. 3 次失败后全部熔断；AD 端可能锁定该账号（data 775）

## Current Focus

**Hypothesis:** `ldap_client.go:tryBindAttempts` 账号池分支直接用 `c.account.PasswordCiphertext`
（SM4 密文）做 LDAP bind，从未解密；而单管理员路径（config.AdminPassword）所有 caller 都先 decryptPassword。
**Test:** grep 证实 PasswordCiphertext 全项目仅 ldap_client.go:140 一处读取且未解密；
AdminPassword 所有 caller（9 处）均解密；用户确认三症状并发。
**Expecting:** tryBindAttempts 内部对账号池密文 decryptPassword 后 bind。
**Next Action:** 修复完成 + 回归测试通过。
**Reasoning Checkpoint:** 根因确认，修复+测试已验证。

## Evidence

- 2026-06-23: 写入路径 `ad_account_handler.go:104/148` 用 `core.SM4Cipher.Encrypt()` 加密（密文入库）。
- 2026-06-23: 读取路径 `ldap_client.go:140` `password = c.account.PasswordCiphertext` 直接用密文 bind，
  代码注释自述漏洞："注意：调用方需在传入前完成密码解密（caller 负责）"——但 FailoverClient 从未解密。
- 2026-06-23: 对照单管理员路径 `config.AdminPassword`，9 处 caller 全部在使用前 decryptPassword：
  ad_authenticator.go:99、user.go×4、sync.go:90、group_sync_service.go×2、dept_sync_tasks.go×2、
  config.go:205、user_ad_sync_service.go×3。账号池路径零解密，显失对称。
- 2026-06-23: `PasswordCiphertext` grep 全项目仅 ldap_client.go:140 一处读取，修复覆盖完整。
- 2026-06-23: 同步任务（NewLDAPClient(config) 不带 account）走单管理员路径，解密正确，
  同步失败系连锁锁定（data 775）而非加解密 bug。
- 2026-06-23: 加密/解密用同一 SM4 cipher 实例（core.SM4Cipher 同时被 SetADSM4Cipher 设为全局），
  round-trip 兼容，`TestAccountPoolPasswordRoundTrip` PASS 证实。

## Eliminated

- ❌ SM4 cipher 未初始化（历史会话 ad-password-encryption-decryption-errors / sm4-encryptor-not-initialized
  已修复 initSM4Cipher 返回 nil + ADAuthenticator 注入；本次 core.go:118/134 初始化正常）
- ❌ 单管理员路径加解密 bug（9 处 caller 均正确解密）
- ❌ 同步任务加解密 bug（user_ad_sync_service.go 三处均正确解密，注释已预判 775）
- ❌ SM4 密钥配置问题（round-trip 测试通过）

## Resolution

**Root Cause:** Phase 36 引入账号池（`sys_ad_service_accounts.password_ciphertext`）时，写入路径
用 SM4 加密，但读取使用路径（`LDAPClient.tryBindAttempts`）直接把密文当明文做 LDAP bind。
与历史已修复的单管理员路径（`sys_ad_config.admin_password`）是**两条独立的密码字段/路径**，
历史修复未覆盖账号池路径。LDAP bind 需明文密码，传密文必然 error 49，导致 FailoverClient
遍历池中所有账号全部失败 → MarkFailure 累加 → 熔断；连锁触发 AD 端账号锁定（data 775），
波及同名单管理员账号使同步任务也失败。代码注释已自述契约漏洞（"caller 负责解密"）但
FailoverClient 从未履行。

**Fix:**
1. `internal/services/addomain/ldap_client.go` `tryBindAttempts`：账号池分支改为
   `password = decryptPassword(c.account.PasswordCiphertext)`，将脆弱的"caller 负责"
   契约改为内部自动解密（FailoverClient 遍历多账号极易遗漏，内部解密更鲁棒），
   与单管理员路径 decryptPassword 处理一致。
2. `internal/services/addomain/ldap_client_test.go`：新增两个回归测试
   - `TestAccountPoolPasswordRoundTrip`：锁定 handler 加密↔tryBindAttempts 解密的闭环契约
   - `TestDecryptPassword_InvalidCiphertext_ReturnsEmpty`：守护 F-03 安全策略（解密失败不回退明文）

**Verification:**
- `go build ./...` 通过（BUILD_EXIT=0）
- `go test ./internal/services/addomain/` 整包通过（7.453s，含 2 个新测试 PASS，无回归）

**Files Changed:**
- internal/services/addomain/ldap_client.go（tryBindAttempts 账号池分支解密 + 注释更新）
- internal/services/addomain/ldap_client_test.go（+2 回归测试）

**遗留提醒：** 修复前若 AD 端账号已被锁定（data 775），代码修复后仍需在 AD 端手动解锁或
等待自动解锁，否则登录/同步会持续失败到解锁为止。
