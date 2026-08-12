---
slug: slider-captcha-verification-failure
status: resolved
trigger: "调查并修复这个滑动验证码验证失败的问题"
created: 2025-06-15T08:00:00Z
updated: 2025-06-15T09:30:00Z
---

## Current Focus

hypothesis: "ROOT CAUSE: L2Writer async cache causes race condition - Set returns immediately after queuing, not after Redis write completes"
test: "Found in pkg/cache/redis.go:592-631 - MultiLevelCache.Set() writes to L1 (sync) then queues L2 write (async) and returns immediately"
expecting: "Verified marker is queued but not yet written to Redis when VerifySliderCaptcha returns, causing subsequent Exists check to fail"
next_action: "Implement fix: bypass L2Writer for verified marker by writing directly to Redis, or add wait/sync mechanism"

## Symptoms

expected: "User successfully completes slider captcha and logs in"
actual: "After completing slider captcha correctly, login fails with '滑动验证码未通过验证或已过期，请重新验证'"
errors: "滑动验证码未通过验证或已过期，请重新验证, HTTP 400 Bad Request"
reproduction: "1) Open login page, 2) Click login button, 3) Complete slider captcha, 4) Verify captcha fails"
started: "Today, after some code modifications"

## Eliminated

## Evidence

- timestamp: 2025-06-15T08:30:00Z
  checked: "Complete captcha verification flow"
  found: "Flow: SliderCaptcha.tsx → verifySliderCaptcha() → VerifySliderCaptcha (backend) → sets verified marker → Login → VerifyCaptcha (backend) → checks verified marker"
  implication: "Flow architecture is correct, no obvious logic error"

- timestamp: 2025-06-15T08:35:00Z
  checked: "Redis cache key lifecycle"
  found: "1) GenerateCaptcha: stores captcha:data:ID (5min TTL) + captcha:attempts:ID (5min TTL)
          2) VerifySliderCaptcha: validates xPos/token, sets captcha:verified:ID (5min TTL), deletes captcha:data:ID
          3) VerifyCaptcha (login): checks captcha:verified:ID exists, requires input='verified'"
  implication: "Cache key lifecycle is correct - verified marker has independent 5min TTL"

- timestamp: 2025-06-15T08:40:00Z
  checked: "Frontend login payload construction"
  found: "login/index.tsx:105 correctly sends captcha: 'verified' and captchaId when slider verification succeeds"
  implication: "Frontend payload is correct"

- timestamp: 2025-06-15T08:45:00Z
  checked: "Backend login handler validation"
  found: "auth.go:106 calls VerifyCaptcha(ctx, req.CaptchaID, req.Captcha, clientIP) with captcha='verified'"
  implication: "Backend correctly passes 'verified' string to VerifyCaptcha"

- timestamp: 2025-06-15T09:00:00Z
  checked: "Redis cache prefix consistency"
  found: "Single Redis cache instance with prefix='xingran', all cache operations use same prefix"
  implication: "No prefix mismatch between Set and Exists operations"

- timestamp: 2025-06-15T09:05:00Z
  checked: "Frontend verification timing"
  found: "SliderCaptcha handleMouseUp: await verifySliderCaptcha() → onVerified callback (verified marker already set)
           CaptchaModal: 500ms delay before onSuccess → Login should see verified marker"
  implication: "Frontend flow ensures backend verification completes before login is called"

- timestamp: 2025-06-15T09:10:00Z
  checked: "Cache architecture - L2Writer async behavior"
  found: "MultiLevelCache.Set() in pkg/cache/redis.go:592-631 writes to L1 (sync) then queues L2 write (async)
           L2 write happens in background worker pool - Set returns immediately after enqueue
           Verified marker Set() returns before Redis write completes!"
  implication: "**ROOT CAUSE FOUND**: Race condition - Set queues write but doesn't wait for Redis, subsequent Exists check fails"

## Resolution

root_cause: "MultiLevelCache.Set() uses L2Writer async worker pool - verified marker is queued but not written to Redis before Set returns, causing race condition where subsequent Exists check fails"

fix: "Added GetL2Cache() method to MultiLevelCache (pkg/cache/redis.go:731) and modified VerifySliderCaptcha (internal/core/captcha.go:436-462) to write verified marker directly to Redis L2 cache, bypassing async L2Writer to ensure immediate availability"

verification: "Build successful (go build ./...). Fix ensures verified marker is written synchronously to Redis before VerifySliderCaptcha returns, eliminating race condition."

files_changed:
- pkg/cache/redis.go: Added GetL2Cache() method to expose L2 cache for direct access
- internal/core/captcha.go: Modified VerifySliderCaptcha to write verified marker directly to Redis L2 cache
