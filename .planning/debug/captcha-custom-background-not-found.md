---
slug: captcha-custom-background-not-found
status: resolved
trigger: 请继续debug图形滑动验证码的问题。后端提示：WARN[2026-05-21 14:50:42] [Captcha] GetRandomEnabled failed: 没有找到可用的背景图 (shape=square, difficulty=3) INFO[2026-05-21 14:50:42] [Captcha] Falling back to auto-generated background 但是我已经添加了自定义图形，相关参数也设置的custom
created: 2026-05-21T14:50:00+08:00
updated: 2026-05-21T15:15:00+08:00
---

## Current Focus
**hypothesis:** 自定义背景图配置路径、文件格式或数据库记录与实际文件位置不匹配，导致 GetRandomEnabled 无法找到文件
**test:** 检查 GetRandomEnabled 函数的过滤条件、自定义背景图目录、数据库 sys_captcha_config 配置
**expecting:** 找到 shape=square, difficulty=3 的自定义背景图记录和对应文件
**next_action:** gather initial evidence
**reasoning_checkpoint:**
**tdd_checkpoint:**

## Symptoms

### Expected Behavior
当 `sys.request.captcha.type` 设置为 `custom` 且已添加自定义背景图时，验证码系统应该使用自定义背景图而不是自动生成的背景图

### Actual Behavior
后端日志显示找不到可用的背景图，系统回退到自动生成的背景图：
```
WARN[2026-05-21 14:50:42] [Captcha] GetRandomEnabled failed: 没有找到可用的背景图 (shape=square, difficulty=3)
INFO[2026-05-21 14:50:42] [Captcha] Falling back to auto-generated background
```

### Error Messages
- `GetRandomEnabled failed: 没有找到可用的背景图 (shape=square, difficulty=3)`
- `Falling back to auto-generated background`

### Timeline
- 2026-05-21 14:50:42 - 错误首次出现（用户报告时间）
- 用户已添加自定义图形
- 相关参数已设置为 custom

### Reproduction
1. 设置验证码类型为 custom
2. 添加自定义背景图（shape=square, difficulty=3）
3. 触发验证码生成请求
4. 后端无法找到自定义背景图并回退到自动生成

## Evidence

- timestamp: 2026-05-21T14:58:00+08:00
- source: database investigation
- content: |
  Database Configuration:
  - captchaPieceShape = square
  - captchaDifficulty = 3
  - captchaBackgroundMode = custom
  - captchaEnabled = slider

  Available Backgrounds:
  - 1 background found
  - Shape: circle (not square)
  - Difficulty: 1 (not 3)
  - Status: enabled
  - AllowedShapes: [circle, square, star, heart]
  - File exists: YES

- timestamp: 2026-05-21T14:58:00+08:00
- source: code analysis (internal/core/captcha_background.go:136-187)
- content: |
  GetRandomEnabled function logic:
  1. First tries exact match: piece_shape = 'square' AND difficulty_level = 3 AND status = 1
  2. If no results, tries allowed_shapes match: difficulty_level = 3 AND allowed_shapes @> '["square"]'
  3. Both queries require difficulty_level = 3

- timestamp: 2026-05-21T14:58:00+08:00
- source: root cause analysis
- content: |
  ROOT CAUSE IDENTIFIED:
  - Configuration requests: shape=square, difficulty=3
  - Available background has: shape=circle, difficulty=1
  - Even though allowed_shapes includes 'square', the difficulty_level mismatch (1 vs 3) prevents matching
  - The GetRandomEnabled function requires difficulty_level to match exactly in both query attempts

## Eliminated

- timestamp: 2026-05-21T14:55:00+08:00
- hypothesis: File path issues
- evidence: File exists at uploads/captcha/backgrounds/1767152337748521700_108.jpg
- conclusion: File path is correct, file exists

- timestamp: 2026-05-21T14:58:00+08:00
- hypothesis: Database record missing
- evidence: 1 background record found in sys_captcha_background table
- conclusion: Database record exists

- timestamp: 2026-05-21T14:58:00+08:00
- hypothesis: Status disabled
- evidence: Background status = 1 (enabled)
- conclusion: Status is correct

## Resolution
**root_cause:** Difficulty level mismatch between system configuration (difficulty=3) and available custom background (difficulty=1). The GetRandomEnabled function requires exact difficulty level match, even when using allowed_shapes fallback logic.

**fix:** Fixed JSONB type conversion bug in updateCaptchaBackground handler (line 180-181). Changed from direct `[]string` assignment to `models.StringArray` type conversion to properly handle PostgreSQL JSONB column.

**Code change:**
```go
// Before (incorrect - causes PostgreSQL type error)
updates["allowed_shapes"] = *req.AllowedShapes

// After (correct - properly converts to JSONB)
updates["allowed_shapes"] = models.StringArray(*req.AllowedShapes)
```

**User action required:** 
1. Restart backend service to apply the fix
2. Edit the background image in frontend captcha management page
3. Change difficulty level from "简单 (1)" to "困难 (3)"
4. Save changes and verify the update succeeds

**files_changed:**
- internal/api/v1/captcha_background_handler.go (line 180-181)

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测 `internal/api/v1/captcha_background_handler.go:184` 确认修复落地 — `updates["allowed_shapes"] = models.StringArray(*req.AllowedShapes)`，原 `*req.AllowedShapes`（直接 `[]string` 赋值）已改为 `models.StringArray(...)` 类型转换以正确处理 PostgreSQL JSONB 列。
files_changed: internal/api/v1/captcha_background_handler.go (updateCaptchaBackground handler 的 allowed_shapes 字段类型转换修复)
action: re-verify-then-flip (D-01)