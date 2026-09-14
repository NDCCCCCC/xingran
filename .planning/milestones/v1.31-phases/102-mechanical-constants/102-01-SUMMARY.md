# Phase 102 Plan 01 Summary: CACHE-01 captcha键收敛

**Plan:** 102-01
**Phase:** 102-mechanical-constants
**Completed:** 2026-09-07
**Commits:** 3

## One-liner

captcha域6种键格式收敛pkg/constants/cache.go具名常量，16个直接位点全部引用常量。

## Objective

CACHE-01: captcha 16个直接Sprintf键位点（6种键格式）收敛为pkg/constants/cache.go具名格式常量并全部引用，行为等价。

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | RED→GREEN: 注册6个captcha键格式常量+等价快照测试 | d28cac7 | pkg/constants/cache.go, pkg/constants/cache_102_test.go |
| 2 | captcha.go 13个直接位点替换为常量引用 | 2f520a0 | internal/core/captcha.go |
| 3 | captcha_background.go 3个直接位点替换（派生式保留） | 7dcf828 | internal/core/captcha_background.go |

## Commits

- **d28cac7** `test(102-01): add RED test for captcha cache key equivalence` — RED测试+6常量追加
- **2f520a0** `fix(102-01): replace captcha.go 13 inline key sites with constants` — 13位点替换
- **7dcf828** `fix(102-01): replace captcha_background.go 3 inline key sites with constants` — 3位点+import

## Key Files Created/Modified

| File | Change |
|------|--------|
| `pkg/constants/cache.go` | 新增6常量（CaptchaRateLimit/Data/AttemptsKeyFormat + LoginFail/CaptchaBgList/CaptchaCachePoolPrefixFormat） |
| `pkg/constants/cache_102_test.go` | 新建等价比快照测试TestCaptchaCacheKeyEquivalence |
| `internal/core/captcha.go` | 13个Sprintf位点改引用constants（rateLimitKey/storageKey/attemptsKey/failKey） |
| `internal/core/captcha_background.go` | 3个Sprintf位点改引用constants（bg:list/cache:pool）+ 新增constants import |

## Verification Results

- `go build ./...` — 退出码0
- `go test ./pkg/constants/ -run TestCaptchaCacheKeyEquivalence -count=1` — ok
- 内联键字面量清零（grep `"captcha:` / `"login:fail:` 计数0）
- 派生构造保留（`poolPrefix + ":counter"` 2处、`fmt.Sprintf("%s:%d", poolPrefix` 2处）

## TTL锁定机制披露（D-102-5①）

TTL常量化属Deferred，本相不改。TTL不变由两道机制锁定：
1. **diff硬门** — git diff无time.Duration/Expire行变更
2. **既有回归测试网** — captcha_78_01_test.go/captcha_background_78_01_test.go/core_74_08_test.go内嵌键字面量，键值不变自然绿

## Deviation from Plan

无偏差。全部16个直接位点已替换，6个常量已注册，等价快照测试绿。

## Threat Flags

无新增安全面。

## Self-Check

- [x] pkg/constants/cache.go包含6常量（值与interfaces块逐字符一致）
- [x] go test TestCaptchaCacheKeyEquivalence绿
- [x] cache_102_test.go期望列为纯字面量（无constants自引用）
- [x] grep "captcha:和"login:fail:计数0
- [x] 派生构造保留（poolPrefix+":counter"/fmt.Sprintf("%s:%d", poolPrefix各2处）
- [x] go build ./...退出码0
- [x] git log d28cac7/2f520a0/7dcf828存在
