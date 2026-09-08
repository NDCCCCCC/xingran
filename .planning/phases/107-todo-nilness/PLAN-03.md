---
phase: 107
plan: "03"
type: execute
wave: 2
depends_on: []
files_modified:
  - internal/api/v1/monitor/login_log_handler.go
autonomous: true
requirements_addressed: [TODO-03]
---

<objective>
Implement the UnlockUser handler stub in login_log_handler.go. The stub currently returns success without actually unlocking anything. Implement the Redis cache key deletion to truly unlock a brute-force-locked user.
</objective>

<tasks>

## TASK-1: Verify lockout key pattern

**Research finding confirmed:**

- Lockout key format: `constants.LoginLockKeyFormat = "login:lock:%s"`
- Defined in: `pkg/constants/cache.go:14`
- Set by: `CaptchaService.RecordLoginFailure` (internal/core/captcha.go:512) when failed login count reaches `LoginMaxRetry` threshold
- Checked by: `CaptchaService.CheckLoginLock` (internal/core/captcha.go:491)
- The `LoginLogHandler` already has access to `core.Cache` (inherited from embedded core field)

## TASK-2: Implement UnlockUser

**File:** `internal/api/v1/monitor/login_log_handler.go`

**Current stub (lines 109-120):**
```go
func (h *LoginLogHandler) UnlockUser(c *gin.Context) {
    username := c.Param("username")
    if username == "" {
        response.Error(c, apperrors.ParamMissing("用户名"))
        return
    }

    // TODO: 实现解锁用户逻辑（如从Redis中删除锁定状态）

    response.Success(c, gin.H{"message": "解锁成功"})
}
```

**Replace the TODO comment with the implementation:**

```go
// 使用 core.Cache 删除登录锁定键
lockKey := fmt.Sprintf(constants.LoginLockKeyFormat, username)
_ = h.core.Cache.Delete(c.Request.Context(), lockKey)
```

**Key design decisions:**
1. Use `constants.LoginLockKeyFormat` for the key pattern — single source of truth
2. Use `c.Request.Context()` for the delete call — proper context propagation
3. Ignore the error from `Cache.Delete` — if the key doesn't exist (already unlocked), that's fine; we want the user unlocked regardless
4. The `response.Success` call remains after the delete — returns "解锁成功" to the admin

**Imports needed:**
- `"fmt"` — for `fmt.Sprintf`
- `"github.com/xingran-next/xingran-go-backend/pkg/constants"` — for `LoginLockKeyFormat`

Verify these are already imported in the file before adding.

</tasks>

<success_criteria>
- `go build ./...` passes — handler compiles cleanly
- `go test ./...` passes — no regressions
- UnlockUser now actually deletes the Redis lockout key before returning success
- The unlock operation is idempotent (safe to call even if not locked)
</success_criteria>

<output>
Part of combined SUMMARY.md after Wave 1 + Wave 2 complete
</output>
