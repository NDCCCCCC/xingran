---
phase: 14-frontend-ux
plan: fix-02
wave: 2
gap_closure: true
gap_refs: [B2, CR-01]
subsystem: network-mac-history
status: complete
commit: 8b98dac
---

# 14-fix-02 SUMMARY

Rewrote `exportMACHistory` to use `fetch()` + `getAccessToken()` + `response.blob()`, bypassing the `src/lib/api.ts` response interceptor that unwrapped `{code, message, data}` envelopes and corrupted real Excel bytes into a JSON object. Closes CR-01 + B2 from `14-VERIFICATION.md`.

## Files modified

| Path | Lines |
|------|-------|
| `xingran-react-frontend/src/lib/api/networkApi.ts` | +40 / -14 |
| `xingran-react-frontend/src/pages/network/mac/history.tsx` | +1 / -4 (handleExport only) |

## Key changes

### 1. `networkApi.ts` — `exportMACHistory` rewrite
- Added import: `import { getAccessToken } from '@/utils/authHelpers';`
- New signature returns `Promise<{ blob: Blob; filename: string }>` (was `Promise<Blob>`)
- Uses native `fetch()` + `URLSearchParams` to build the GET query, skipping `undefined`/`null`/`''`
- Auth via `Authorization: Bearer ${token}` from `getAccessToken()` (TokenManager-aware)
- CR-01 error deserialization branch: if `blob.size < 1024 && blob.type.includes('json')`, parse text and throw with `errBody.message || errBody.msg`
- Filename extraction from `Content-Disposition` header (regex match + `decodeURIComponent`); fallback `mac_history_${exportScope}_${Date.now()}.xlsx`
- Old `import('../api') + api.default.get + responseType: 'blob'` implementation completely removed (0 occurrences remaining)

### 2. `history.tsx` — `handleExport` call site update
- Replaced `const blob = await exportMACHistory(...)` with `const { blob, filename } = await exportMACHistory(...)`
- Removed the synthetic `const filename = \`mac_history_${exportScope}_${ts}.xlsx\`;` line (now uses server-provided filename via Content-Disposition)
- Preserved `URL.createObjectURL(blob)` + `a.download = filename` + `a.click()` + `URL.revokeObjectURL(url)` exactly as required by D-15
- Preserved the existing `try/catch` + `message.success/error` + `finally setExporting(false)` flow

## Verification commands & outputs

```bash
$ cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend
$ npx tsc --noEmit -p .
# EXITCODE=0  ✓

$ grep -c "api.default.get" src/lib/api/networkApi.ts
0  ✓ (required: 0)

$ grep -c "fetch.*history/list" src/lib/api/networkApi.ts
1  ✓ (required: >= 1)

$ grep -c "getAccessToken" src/lib/api/networkApi.ts
3  ✓ (required: >= 1; 1 import + 1 call + 1 in comment)

$ grep -c "URLSearchParams" src/lib/api/networkApi.ts
1  ✓ (required: >= 1)

$ grep -c "content-disposition\|Content-Disposition" src/lib/api/networkApi.ts
2  ✓ (required: >= 1; 1 lookup + 1 regex literal)

$ grep -c "blob.size < 1024" src/lib/api/networkApi.ts
1  ✓ (required: >= 1)

$ grep -c "URL.createObjectURL" src/pages/network/mac/history.tsx
1  ✓ (required: >= 1; D-15 preserved)

$ grep -n "exportMACHistory\|URL.createObjectURL" src/pages/network/mac/history.tsx
38:import { queryMACHistory, exportMACHistory } from '@/lib/api/networkApi';
377:        const { blob, filename } = await exportMACHistory(
382:        const url = URL.createObjectURL(blob);
# 3 lines  ✓ (required: >= 2)
```

## Deviations from plan

None. Plan executed exactly as specified.

## Follow-ups

- **14-fix-03** (next plan) will also touch `history.tsx` to replace inline `Alert`/`Empty` with `EmptyStateWithAction` / `ErrorAlertWithRetry`. This plan was surgically scoped to `handleExport` only to avoid merge conflicts.
- **End-to-end smoke** is out-of-band per plan: requires backend running, login, navigate to `/network/mac/history`, click "导出当前查询" → verify `mac_history_current_<ts>.xlsx` opens in Excel. Not executed in this autonomous run.
- The `import { post } from '../api';` line at top of `networkApi.ts` is preserved (still needed by `queryMACTrajectory`, `queryMACHistory`, `getMACEvents`). Only the dynamic `await import('../api')` inside the old `exportMACHistory` was removed.

## Commit

```
8b98dac fix(14): rewrite exportMACHistory to fetch + Blob (CR-01)
```

Local-only (not pushed, no PR opened) per plan instructions.
