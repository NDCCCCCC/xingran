# Deferred Items — Phase 39

Out-of-scope pre-existing issues discovered during plan execution. NOT fixed per executor scope boundary rule.

## 39-05 — Pre-existing ESLint issues in opsApi.ts (NOT introduced by this plan)

Discovered during Plan 39-05 Task 2 verification (`npx eslint` on modified files).

| Line | Severity | Rule | Description |
|------|----------|------|-------------|
| opsApi.ts:281 | warning | @typescript-eslint/no-unsafe-assignment | Unsafe assignment of `any` value in `extractFilenameFromBlobResponse` |
| opsApi.ts:298 | warning | @typescript-eslint/no-unsafe-assignment | Unsafe assignment of `any` value in `triggerBrowserDownload` region |
| opsApi.ts:491 | error | @typescript-eslint/no-unused-vars | `maxAge` parameter in `geocodeWithCache` is assigned but never used |

**Context:** All three issues are in pre-existing functions (`extractFilenameFromBlobResponse`, `triggerBrowserDownload`, `geocodeWithCache`) — NOT in the new `DeptOption` / `LocationAlias` / `locationAliasApi` / `workstationApi.deptOptions` code added by 39-05.

**Action required:** Separate cleanup ticket. Do NOT bundle into Phase 39 (scope boundary).
