# Phase 106 Plan 04 — eslint-disable reason suffix

## Status: DONE

## Changes

### Task 1 — App.tsx line 35
- File: `xingran-react-frontend/src/App.tsx`
- Before: `// eslint-disable-next-line react-hooks/exhaustive-deps`
- After: `// eslint-disable-next-line react-hooks/exhaustive-deps -- reason: allMenus.length is intentionally the only trigger; routeConfigManager.initialize is stable and should not re-run on menu metadata changes`

### Task 2 — TargetSelector.tsx line 91
- File: `xingran-react-frontend/src/components/TargetSelector.tsx`
- Before: `// eslint-disable-next-line react-hooks/exhaustive-deps`
- After: `// eslint-disable-next-line react-hooks/exhaustive-deps -- reason: targetType is the only trigger; dependent callbacks (loadDepts/loadUsers) are stable useCallback refs`

### Task 3 — fix-suggestion/index.tsx line 118
- File: `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx`
- Before: `// eslint-disable-next-line react-hooks/exhaustive-deps`
- After: `// eslint-disable-next-line react-hooks/exhaustive-deps -- reason: initializing form with filterValues from URL params; no reactive deps needed (setFieldsValue is stable)`

### Task 4 — exceptions/index.tsx line 166
- File: `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx`
- Before: `// eslint-disable-next-line react-hooks/exhaustive-deps`
- After: `// eslint-disable-next-line react-hooks/exhaustive-deps -- reason: initializing form with filterValues from URL params; no reactive deps needed (setFieldsValue is stable)`

## Verification

- `grep -n "eslint-disable"` confirmed all four `eslint-disable-next-line react-hooks/exhaustive-deps` lines now end with `-- reason: ...`
- `npx eslint ... --quiet` passed with 0 errors
