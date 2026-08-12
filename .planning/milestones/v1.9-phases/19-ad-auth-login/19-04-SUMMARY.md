# Phase 19 Plan 04: Frontend Auth Mode Selector and SM4 Password Encryption Summary

---
phase: 19
plan: 04
subsystem: authentication
tags: [frontend, auth-mode, sm4-ecb, login, radio-selector, ad-auth]
dependency_graph:
  requires: [19-03]
  provides: [AuthMode type, AuthModeOption interface, SM4 password encryption, auth mode selector UI]
  affects: [xingran-react-frontend/src/types/auth.ts, xingran-react-frontend/src/utils/sm4.ts, xingran-react-frontend/src/pages/login/index.tsx, xingran-react-frontend/src/store/authStore.ts]
tech_stack:
  added: []
  patterns: [SM4-ECB password encryption, Auth mode state management]
key_files:
  created: []
  modified:
    - xingran-react-frontend/src/types/auth.ts
    - xingran-react-frontend/src/utils/sm4.ts
    - xingran-react-frontend/src/pages/login/index.tsx
    - xingran-react-frontend/src/pages/login/login.css
    - xingran-react-frontend/src/store/authStore.ts
decisions:
  - Used SM4-ECB mode for password encryption (simpler than CBC, no IV needed)
  - Login page generates session SM4 key client-side via crypto.getRandomValues (T-19-11)
  - authStore still wraps login data with getEncryptedLoginRequest for SM2 encryption layer
  - fetchSM4KeyForPassword documented as requiring backend endpoint not yet implemented
  - Auth mode selector uses card-style Radio items with icons and descriptions
metrics:
  duration: 14m
  completed: 2026-05-21
  tasks_total: 5
  tasks_completed: 5
  files_created: 0
  files_modified: 5
  commit_count: 5
---

## One-liner

Frontend auth mode selector (local/ad/hybrid) with SM4-ECB password encryption, styled Radio.Group UI, and authMode propagation through login flow to backend.

## Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Update frontend type definitions | ce85766 | xingran-react-frontend/src/types/auth.ts |
| 2 | Enhance SM4 password encryption utility | 92cb80f | xingran-react-frontend/src/utils/sm4.ts |
| 3 | Update login page component | 6b76344 | xingran-react-frontend/src/pages/login/index.tsx |
| 4 | Update authStore for authMode support | 88ef7c0 | xingran-react-frontend/src/store/authStore.ts |
| 5 | Add auth mode UI optimization | 24b80b9 | xingran-react-frontend/src/pages/login/index.tsx, login.css |

## What Was Built

### AuthMode Type Definitions (`types/auth.ts`)
- `AuthMode` type: `'local' | 'ad' | 'hybrid'`
- `AuthModeOption` interface with label, value, and optional description
- Extended `LoginRequest` with optional `authMode?: AuthMode` field

### SM4 Password Encryption (`utils/sm4.ts`)
- `encryptPasswordWithSM4(password, keyHex)`: Encrypts password using SM4-ECB mode
- `generateSessionKey()`: Generates 32-char hex SM4 key via `generateSM4Key()` which uses `crypto.getRandomValues` (T-19-11 mitigation)
- `fetchSM4KeyForPassword()`: Placeholder for fetching SM4 key from backend `/api/v1/system/auth/sm4-key` endpoint (not yet implemented server-side)

### Login Page (`pages/login/index.tsx`)
- Auth mode selector with `Radio.Group` containing three options with icons and descriptions
- Password encrypted via `encryptPasswordWithSM4` before being sent to authStore
- `authMode` and `encryptedPassword: false` flags included in login request
- Captcha logic preserved unchanged

### Auth Mode Selector CSS (`pages/login/login.css`)
- Card-style radio items with padding, border, and border-radius
- Hover state: blue border + light blue background
- Selected state: blue border + slightly darker blue background
- Radio button aligned to top for multi-line content

### AuthStore (`store/authStore.ts`)
- `authMode?: AuthMode` added to `AuthState` interface
- Login method extracts `authMode` from credentials (defaults to 'local')
- `authMode` passed to backend API in login request
- `authMode` saved to store state on login success
- `authMode` reset to `undefined` on logout

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] fetchSM4KeyForPassword cannot use SM2 decrypt from client side**
- **Found during:** Task 2
- **Issue:** Plan called `sm2.decrypt(result.data.encryptedKey)` but the SM2 module only exports `decryptWithSM2` (async) which requires a private key. Frontend only holds the SM2 public key, not the private key.
- **Fix:** Adjusted `fetchSM4KeyForPassword` to accept `result.data.keyHex` directly from backend (protected by HTTPS). Documented the limitation in code comments. The function is available for future backend endpoint integration.
- **Files modified:** xingran-react-frontend/src/utils/sm4.ts
- **Commit:** 92cb80f

**2. [Rule 3 - Blocking] Tab/space mismatch in authStore.ts edits**
- **Found during:** Task 4
- **Issue:** authStore.ts uses tab indentation, causing Edit tool failures when matching strings with space-based indentation.
- **Fix:** Used Write tool to rewrite the complete file with correct tab indentation.
- **Files modified:** xingran-react-frontend/src/store/authStore.ts
- **Commit:** 88ef7c0

## Threat Model Compliance

| Threat ID | Mitigation | Status |
|-----------|-----------|--------|
| T-19-10 | Backend validates authMode values | Frontend sends only local/ad/hybrid; backend validation in Wave 2 |
| T-19-11 | Secure SM4 key generation | Implemented: generateSessionKey uses crypto.getRandomValues |
| T-19-12 | Password encrypted before transport | Implemented: SM4-ECB encrypts password before sending, plus SM2 layer in authStore |
| T-19-13 | SM4 encryption performance | Accepted: SM4-ECB is fast, no noticeable UX impact |

## Known Stubs

| Stub | File | Reason |
|------|------|--------|
| fetchSM4KeyForPassword | xingran-react-frontend/src/utils/sm4.ts | Backend endpoint `/api/v1/system/auth/sm4-key` not yet implemented. Function is complete but will fail at runtime until backend is ready. Currently login uses client-side generated session key instead. |

## Threat Flags

None - all new surface (auth mode parameter, SM4 encryption) is covered by the plan's threat model.

## Self-Check: PASSED

---

**Execution completed:** 2026-05-21
**Duration:** 14 minutes
**Executor:** Claude (GSD Execute Phase - Parallel Worktree)
