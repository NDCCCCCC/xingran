---
slug: encryption-config-not-working
status: resolved
created: 2026-05-21T00:00:00Z
updated: 2026-05-21T09:00:00Z
trigger: 经过去测试，我发现设置sys.request.encryption.enabled为true或者false貌似没有区别，根本没有生效！请检查！
---

# Debug Session: 加密配置不生效

## Trigger

经过测试，我发现设置sys.request.encryption.enabled为true或者false貌似没有区别，根本没有生效！请检查！

## Symptoms

### Expected Behavior
- 设置为 false 时，请求和响应都不应该加密（无加密 header）

### Actual Behavior
- 无论设置什么，请求始终加密
- 修改配置前后行为完全一样

### Configuration Method
- 通过参数管理页面 UI 修改
- 直接修改数据库（UPDATE sys_config）

### Post-Configuration Actions
- 重启了后端服务
- 刷新了前端页面

## Current Focus

**Status:** ROOT CAUSE FOUND
**Hypothesis:** Frontend API client is not using dynamic encryption configuration
**Next Action:** Fix frontend to use dynamic configuration
**Test:** Modify config and verify requests stop being encrypted
**Expecting:** Requests should not have X-Request-Encrypted header when config is false

**Reasoning Checkpoint:** ROOT CAUSE IDENTIFIED - Frontend hardcoded ENABLE_REQUEST_ENCRYPTION=true

## Evidence

- timestamp: 2026-05-21T08:47:49Z - Investigation started
- timestamp: 2026-05-21T08:47:49Z - **ROOT CAUSE FOUND**: Frontend API client (`xingran-react-frontend/src/lib/api.ts:38`) has hardcoded `ENABLE_REQUEST_ENCRYPTION = true` instead of using dynamic configuration from backend
- timestamp: 2026-05-21T08:47:49Z - **Backend is working correctly**: `pkg/middleware/request_decryption.go` has proper DB integration with cache refresh
- timestamp: 2026-05-21T08:47:49Z - **Frontend service exists but unused**: `xingran-react-frontend/src/services/encryptionConfig.ts` provides `getCachedEncryptionConfig()` but not called by API client
- timestamp: 2026-05-21T08:47:49Z - **Helper functions available**: `xingran-react-frontend/src/utils/authHelpers.ts` has `getEncryptionConfigStatus()` but not integrated into request interceptor
- timestamp: 2026-05-21T09:00:00Z - **FIX APPLIED**: Integrated dynamic encryption configuration into frontend API client

## Eliminated

- Backend configuration reading: Working correctly (getConfigFromDB with cache)
- Middleware cache refresh: Working correctly (RefreshEncryptionConfigCache called on update)
- Database configuration storage: Working correctly (config update logic is correct)
- Frontend encryption service: Service exists and works, just not integrated

## Resolution

- **Root Cause:** Frontend API client (`xingran-react-frontend/src/lib/api.ts:38`) has hardcoded `ENABLE_REQUEST_ENCRYPTION = true` and ignores backend dynamic configuration. The `initEncryptionConfig` function was removed to prevent infinite loops, but no proper integration was implemented.

- **Fix Applied (2026-05-21T09:00:00Z):**
  
  1. **Added encryption config endpoint to blacklist** (`api.ts:48`)
     - Added `/system/auth/encryption-config` to `ENCRYPTION_BLACKLIST`
     - Prevents infinite loop (config endpoint itself won't be encrypted)
  
  2. **Re-implemented `initEncryptionConfig()` function** (`api.ts:96-113`)
     - Imports `getCachedEncryptionConfig` from encryptionConfig service
     - Fetches encryption status from backend on app startup
     - Updates `ENABLE_REQUEST_ENCRYPTION` variable dynamically
     - Includes error handling with fail-safe default to `true`
  
  3. **Added `refreshEncryptionConfig()` export** (`api.ts:115-120`)
     - Clears cache and reloads config from backend
     - For use by config management UI after updates
  
  4. **Enabled `initEncryptionConfig()` call in main.tsx** (`main.tsx:14`)
     - Uncommented the function call that was previously disabled
     - App now fetches encryption config on startup before rendering

- **Verification:** 
  - ✅ TypeScript compilation successful
  - ⏳ Pending user testing: Restart frontend and verify requests respect database config

- **Files Changed:**
  - `xingran-react-frontend/src/lib/api.ts` - Re-implemented dynamic config integration
  - `xingran-react-frontend/src/main.tsx` - Enabled config initialization on startup

## Specialist Review

Specialist: typescript-expert
Review: The issue is a classic frontend configuration integration problem. The backend Phase 17 implementation is complete and working correctly, but the frontend integration is incomplete. The fix should maintain the existing API client architecture while adding dynamic configuration support.