# Plan 18-01 Summary: Configuration Updates

## Completed
2026-05-21

## Objective
Enable SM2+SM4 hybrid request body encryption for the login endpoint `/system/auth/login` to achieve three-layer encryption protection (HTTPS + SM2+SM4 request body + SM2 password field encryption).

## Changes Made

### Backend Configuration (configs/config.yaml)

**Request Encryption exclude_paths - Before:**
```yaml
exclude_paths:
  - "/api/v1/system/auth/public-key"
  - "/api/v1/system/auth/test-sm2"
  - "/api/v1/system/auth/login"       # ← REMOVED
  - "/api/v1/upload/*"
  - "/api/v1/captcha/*"
  - "/api/v1/rpa/workers/register"
  - "/api/v1/rpa/workers/*/heartbeat"
  - "/api/v1/rpa/workers/progress"
```

**Request Encryption exclude_paths - After:**
```yaml
exclude_paths:
  - "/api/v1/system/auth/public-key"
  - "/api/v1/system/auth/test-sm2"
  # 登录接口已移除 - 启用请求体加密 (Phase 18)
  - "/api/v1/upload/*"
  - "/api/v1/captcha/*"
  - "/api/v1/rpa/workers/register"
  - "/api/v1/rpa/workers/*/heartbeat"
  - "/api/v1/rpa/workers/progress"
```

**Response Encryption exclude_paths - Before:**
```yaml
exclude_paths:
  - "/api/v1/system/auth/public-key"
  - "/api/v1/system/auth/test-sm2"
  - "/api/v1/system/auth/login"       # ← REMOVED
  - "/api/v1/upload/*"
  - "/api/v1/captcha/*"
  - "/api/v1/rpa/workers/register"
  - "/api/v1/rpa/workers/*/heartbeat"
  - "/api/v1/rpa/workers/progress"
```

**Response Encryption exclude_paths - After:**
```yaml
exclude_paths:
  - "/api/v1/system/auth/public-key"
  - "/api/v1/system/auth/test-sm2"
  # 登录接口已移除 - 启用响应体加密 (Phase 18)
  - "/api/v1/upload/*"
  - "/api/v1/captcha/*"
  - "/api/v1/rpa/workers/register"
  - "/api/v1/rpa/workers/*/heartbeat"
  - "/api/v1/rpa/workers/progress"
```

### Frontend Configuration (xingran-react-frontend/src/lib/api.ts)

**ENCRYPTION_BLACKLIST - Before:**
```typescript
const ENCRYPTION_BLACKLIST: string[] = [
  '/system/auth/login',              // ← REMOVED
  '/system/auth/public-key',
  '/system/auth/captcha',
  '/system/auth/encryption-config',
  '/upload',
];
```

**ENCRYPTION_BLACKLIST - After:**
```typescript
const ENCRYPTION_BLACKLIST: string[] = [
  // 登录接口已移除 - 启用请求体加密 (Phase 18)
  '/system/auth/public-key',
  '/system/auth/captcha',
  '/system/auth/encryption-config',
  '/upload',
];
```

## Verification

### Configuration Consistency Check
✅ Backend config.yaml: login removed from BOTH exclude_paths
✅ Frontend api.ts: login removed from ENCRYPTION_BLACKLIST
✅ Required endpoints remain excluded: public-key, captcha, upload
✅ require_encryption: false (backward compatibility maintained)

### Key-Links Verification
✅ `xingran-react-frontend/src/lib/api.ts` → `/system/auth/login` via `shouldEncryptRequest()`: No longer blacklisted
✅ `configs/config.yaml` → `/api/v1/system/auth/login` via `exclude_paths`: No longer excluded

## Expected Behavior Changes

1. **Frontend**: Login requests to `/system/auth/login` will now be encrypted with SM2+SM4
   - Request body structure changes from plaintext to `{ encrypted: true, data: "...", sm4Key: "...", iv: "...", timestamp: ..., nonce: ... }`
   - X-Request-Encrypted header will be present

2. **Backend**: RequestDecryption middleware will decrypt login requests before handler processes them
   - No code changes required in login handler
   - Middleware restores plaintext before handler sees request

3. **Response**: Login responses will be encrypted with SM4 using the same key
   - X-Response-Encrypted header will be present
   - Frontend response interceptor will decrypt automatically

## Backward Compatibility

✅ **Maintained**: `require_encryption: false` ensures old clients sending plaintext requests still work

✅ **Gradual Migration**: New clients encrypt, old clients use plaintext - both work

✅ **Future**: Can set `require_encryption: true` to enforce encryption for all clients

## Security Improvements

**Three-Layer Encryption Active:**
1. **Layer 1**: HTTPS/TLS (transport layer) - Always enabled
2. **Layer 2**: SM2+SM4 request/response body encryption (application layer) - **NOW ENABLED**
3. **Layer 3**: SM2 password field encryption (field level) - Always enabled

**Defense-in-Depth Benefits:**
- Protection against TLS termination attacks
- Prevention of deep packet inspection
- Credentials not visible in server logs
- MITM resistance even if TLS is compromised

## Commit
```
feat(18-01): remove login endpoint from encryption exclusion

Remove /api/v1/system/auth/login from both request and response
encryption exclude_paths to enable SM2+SM4 hybrid encryption for
the login endpoint.

Backend changes:
- Remove login from request_encryption.exclude_paths
- Remove login from response_encryption.exclude_paths
- Keep required exclusions: public-key, test-sm2, upload, captcha

Frontend changes:
- Remove /system/auth/login from ENCRYPTION_BLACKLIST
- Maintain backward compatibility with require_encryption: false

This enables three-layer encryption protection:
1. HTTPS/TLS (transport layer)
2. SM2+SM4 request body encryption (application layer)
3. SM2 password field encryption (field level)

Phase 18 Plan 01 - Configuration Updates
```

## Next Steps
Proceed to Plan 18-02: Unit Tests
- Create backend unit tests for encrypted login
- Create frontend unit tests for request encryption interceptor
- Achieve >80% test coverage for encryption-related code paths
