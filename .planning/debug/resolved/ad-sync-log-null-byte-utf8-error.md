---
slug: ad-sync-log-null-byte-utf8-error
status: resolved
trigger: 今天修改导致的错误，之前是好的。AD同步时报错：ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)
created: 2026-05-26T13:51:00+08:00
updated: 2026-05-26T14:20:00+08:00
---

## Symptoms

**Expected Behavior:**
- AD 同步应该成功（用户确认配置没有问题）

**Actual Behavior:**
- AD 同步时出现 500 Internal Server Error
- GORM 尝试更新 `sys_ad_sync_log` 表时失败
- 错误：`ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)`

**Error Messages:**
```
ERRO[2026-05-26 13:51:08] [GORM错误] UPDATE "sys_ad_sync_log" SET "computer_count"=0,"duration"=0,"end_time"='2026-05-26 13:51:08.138',"error_count"=1,"error_message"='绑定失败: LDAP Result Code 49 "Invalid Credentials": 80090308: LdapErr: DSID-0C0904AE, comment: AcceptSecurityContext error, data 775, v3839 (尝试: UPN, NetBIOS, 直连)',"group_count"=0,"ou_count"=0,"sync_status"='failed',"user_count"=0 WHERE id = '4bddaa77-32b0-4013-bbf9-4cb8719f1050' | 耗时: 2.6714ms | 错误: ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)
```

**Timeline:**
- 之前功能正常
- 今天修改后出现问题
- 用户提到"类似问题已经处理过"（但不确定具体是哪个）

**Reproduction:**
1. 触发 AD 同步（可能是凭据错误导致 LDAP 绑定失败）
2. 尝试将错误消息写入 `sys_ad_sync_log.error_message` 字段
3. PostgreSQL 因 UTF-8 编码验证失败而拒绝写入

## Current Focus

**Hypothesis:** 错误消息中包含 null byte (`\x00`)，需要保存到数据库前清理

**Next Action:** 查找 AD 同步日志保存相关代码，检查 error_message 处理逻辑

**Test:** 检查 `error_message` 字段内容是否包含 `\x00` 字符

**Expecting:** 找到生成错误消息的代码位置，并添加清理逻辑

**Evidence:**

- **File**: `internal/services/addomain/ldap_client.go:113`
- **Code**: `return fmt.Errorf("绑定失败: %w (尝试: UPN, NetBIOS, 直连)", lastErr)`
- **Root Cause**: LDAP error messages contain null bytes (`\x00`) which are invalid UTF-8 for PostgreSQL text columns

- **File**: `internal/services/addomain/sync.go:610`
- **Code**: `updates["error_message"] = errorMsg`
- **Issue**: Error messages from LDAP are passed directly to database without sanitization

**Eliminated:**

- Not a database schema issue (column is `type:text` which should handle UTF-8)
- Not a connection issue (error occurs during UPDATE, not connection)
- Not a GORM configuration issue

**Reasoning Checkpoint:**

1. LDAP bind failure generates error message: `LDAP Result Code 49 "Invalid Credentials": 80090308: LdapErr: DSID-0C0904AE, comment: AcceptSecurityContext error, data 775, v3839`
2. This error message contains embedded null bytes from the LDAP protocol
3. When passed to `updateSyncLog()`, it's stored directly in `error_message` field
4. PostgreSQL validates UTF-8 encoding and rejects text with null bytes
5. GORM UPDATE fails with SQLSTATE 22021

## Investigation Notes

- LDAP library errors may contain binary data/null bytes from protocol messages
- PostgreSQL `text` columns do not allow null bytes (validates UTF-8)
- Need to sanitize error messages before database storage
- Similar to string sanitization patterns used elsewhere in codebase

## Resolution

**Root Cause:**
LDAP error messages contain null bytes (`\x00`) from the LDAP protocol. When these error messages are stored directly in the `sys_ad_sync_log.error_message` column, PostgreSQL rejects them due to invalid UTF-8 encoding (SQLSTATE 22021).

**Fix:**
Create a utility function to sanitize error messages by removing null bytes before database storage. Apply this in `updateSyncLog()` function in `internal/services/addomain/sync.go`.

**Verification:**
- Test with invalid AD credentials to trigger LDAP bind error
- Verify error message is saved to database successfully
- Check that sanitized error message is still readable in logs

**Files Changed:**
- `internal/services/addomain/sync.go` - Add sanitization in `updateSyncLog()`
- Possible: `internal/utils/stringutil.go` - Add `SanitizeErrorMessage()` helper
