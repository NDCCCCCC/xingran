---
status: reference
type: knowledge-base
note: 不是 debug session —— 已解决会话的模式参考库，供 gsd-debugger 检索。audit-open 应跳过。
---

# GSD Debug Knowledge Base

Resolved debug sessions. Used by `gsd-debugger` to surface known-pattern hypotheses at the start of new investigations.

---

## adlogin-ou-test-fail-20260818 — Dead test stub asserting on un-wired gin.New() produces 404
- **Date:** 2026-08-18
- **Error patterns:** TestADLoginWithOUProcessing, Should be true, 404, gin.New, authType, expectedStatus, w.Code
- **Root cause:** Test file at `internal/api/v1/auth/auth_handler_test.go` is a dead-code stub since initial commit. It constructs `gin.New()` without registering any handler for `/login`, so Gin returns default 404, which fails the assertion `w.Code == 200 || w.Code == 401`. The directory contains only the test file (no `auth_handler.go`); the real handler lives at `internal/api/v1/auth.go` (package `v1`) at route `POST /system/auth/login` (not `/login`). Test payload uses `authType` field while real `LoginRequest` uses `authMode` — confirms stub was never wired to production.
- **Fix:** Delete the dead test file and the now-empty `internal/api/v1/auth/` directory. No production code touched.
- **Files changed:** internal/api/v1/auth/auth_handler_test.go, internal/api/v1/auth/
---

## prettier-format-check-fail-20260823 — CI format:check fails on files from manually edited migration commit that skipped prettier
- **Date:** 2026-08-23
- **Error patterns:** prettier --check, format:check, Code style issues found in 2 files, [warn] src/types/global.d.ts, [warn] tsconfig.app.json, exit code 1, npm run format:check
- **Root cause:** Manual TS 7 migration commit (16a26e1, PR #3) edited `xingran-react-frontend/tsconfig.app.json` and `src/types/global.d.ts` without running `prettier --write` before committing. Two violations: `files` array left multi-line (prettier collapses it — total length under printWidth) and a `}// 行尾注释` no-space brace-comment style in global.d.ts. CI `prettier --check .` full-repo scan surfaced both only after earlier CI steps (test/staticcheck) were fixed in PR #4, so the latent debt appeared "new".
- **Fix:** `npx prettier --write src/types/global.d.ts tsconfig.app.json` (prettier 3.9.6, same lockfile as CI). Pure format diff, no semantic change. Verified: full-repo format:check / type-check / lint all exit 0.
- **Lesson:** When CI failures are sequential (fails fast at first step), later steps (format:check) may hide additional latent failures — expect to re-run full CI pipeline locally (`npm run format:check && npm run type-check && npm run lint`) after any dependency-bump or manual-migration commit.
- **Files changed:** xingran-react-frontend/src/types/global.d.ts, xingran-react-frontend/tsconfig.app.json
---
