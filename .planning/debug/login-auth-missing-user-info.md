---
name: login-auth-missing-user-info
status: fixed
trigger: 调查并修复登录失败bug: 前端报错"认证成功但用户信息缺失"(errorHandler.ts:220)，后端返回500错误。用户名: chenchao-076，请求路径: /api/v1/system/auth/login
created: 2026-05-26
updated: 2026-05-26
labels:
  - bug
  - login
  - authentication
  - error-500
---

# Debug Session: login-auth-missing-user-info

## Symptoms

### Expected Behavior
正常登录跳转到首页

### Actual Behavior
后端日志发给你了，不要问我

### Error Messages
```
前端: "认证成功但用户信息缺失" (errorHandler.ts:220)
后端: ERRO[2026-05-26 13:16:22] Internal server error
status_code=500
```

### Timeline
最近才开始出现

### Reproduction
不确定，之前好像出现过类似情况

### Context
- 用户名: chenchao-076
- 请求路径: /api/v1/system/auth/login
- 请求体包含加密密码

## Current Focus

### Hypothesis
AD域控认证成功但无法获取完整用户信息时，authResult.User为nil，导致login handler在第198行检查时报错"认证成功但用户信息缺失"并返回500错误。

### Next Action
实施修复：在login handler中处理NeedsSync=true且User=nil的情况

### Test
待设计测试

### Expecting
AD认证成功但用户同步失败时，仍能完成登录流程（可能需要降级处理）

## Evidence

### 2026-05-26 13:30:00 - Initial Analysis
- **Source**: Code examination of `/internal/api/v1/auth.go:198-201`
- **Finding**: Exact error message found at line 199: `response.Error(c, response.ErrServerError, "认证成功但用户信息缺失")`
- **Trigger**: `if authResult.User == nil` check after successful authentication
- **Impact**: Returns 500 error even though authentication succeeded

### 2026-05-26 13:35:00 - AD Authenticator Analysis
- **Source**: `/internal/core/security/ad_authenticator.go`
- **Finding**: AD authenticator returns `User: nil` in multiple scenarios:
  1. Lines 89-96: Admin connection fails after user auth success
  2. Lines 103-110: Admin bind fails
  3. Lines 117-124: User search fails
  4. Lines 133-138: User sync fails
  5. Lines 160-165: No user sync service configured
- **Common Pattern**: All cases return successful auth (`error: nil`) but with `User: nil` and `NeedsSync: true`
- **User Impact**: Users get 500 error instead of successful login when AD infrastructure has issues

### 2026-05-26 13:40:00 - Authentication Flow Analysis
- **Source**: `/internal/api/v1/auth.go:170-201`
- **Flow**:
  1. Line 170: `authResult, err := authenticator.Authenticate(ctx, authReq)`
  2. Lines 171-195: Error handling for auth failures (wrong password, user disabled, etc.)
  3. Lines 198-201: **BUG** - assumes successful auth always includes User info
  4. Lines 204-208: Queries full user from DB using `authResult.User.ID`
- **Root Cause**: Handler doesn't handle the `NeedsSync: true, User: nil` case properly

### 2026-05-26 13:45:00 - Root Cause Identification
- **Root Cause**: Login handler at `/internal/api/v1/auth.go:198-201` assumes `authResult.User` is always non-nil after successful authentication
- **Why it happens**: AD authenticator returns `User: nil, NeedsSync: true` when:
  - AD admin connection fails after user bind succeeds
  - User sync service fails to create/update local user record
  - User search in AD fails after successful authentication
- **Impact**: Users who successfully authenticate against AD but encounter infrastructure issues get 500 error instead of login success
- **Factory Setup**: Auth factory properly sets user sync service via `SetUserSyncer()` at `/internal/core/core.go:528`
- **Specialist Hint**: backend (Go authentication handler)

## Eliminated

## Resolution

### Root Cause
Login handler (`internal/api/v1/auth.go:198-201`) doesn't handle the case where AD authentication succeeds but user sync fails, resulting in `authResult.User == nil` and a 500 error.

### Fix
Modify the login handler to handle `authResult.User == nil` scenarios:
1. When `NeedsSync: true` and `User: nil`, attempt to query existing user from database by username
2. If user exists in DB, proceed with login using existing user record
3. If user doesn't exist, return a more specific error message indicating user sync is required
4. Log warnings for AD infrastructure issues but don't fail login if user already exists locally

### Verification
需要重启后端服务并测试：
1. 使用AD账号(chenchao-076)尝试登录
2. 验证能正常跳转到首页
3. 检查后端日志确认使用了本地用户数据
4. 验证修复不影响正常本地账号登录

### Files Changed
- `internal/api/v1/auth.go:197-221` - 添加NeedsSync且User==nil的处理逻辑