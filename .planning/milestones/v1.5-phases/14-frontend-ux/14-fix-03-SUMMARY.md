## One-liner

Closed 4 React anti-patterns (B3/W1/W3/W4) across 3 frontend files with ref guard, type narrowing, user feedback, and proper effect deps — TypeScript 0 errors.

## Key changes

- **`ErrorAlertWithRetry.tsx`** — B3/CR-03 fix
  - Added `useRef` to React import (line 15).
  - Added `ranRef` (guards duplicate `code === 1007` triggers across renders) and `cancelledRef` (guards unmount before redirect).
  - Replaced useEffect to early-return on non-1007 / already-handled code, catch logout errors, and only redirect on `!cancelledRef.current`.
  - Cleanup function `() => { cancelledRef.current = true; }` wired into effect.
- **`EmptyStateWithAction.tsx`** — W1/CR-04 fix
  - `showAction` now uses `Boolean(actionLabel) && typeof actionPath === 'string' && actionPath.length > 0`.
  - `<Link to={actionPath}>` no longer needs the `as string` cast — type narrowing makes it redundant.
  - `grep -c "as string"` returns 0 (cast removed).
- **`history.tsx`** — W3/WR-07 + W4/WR-01 fixes
  - `copyMAC` now `async`, wraps `navigator.clipboard.writeText` in try/catch with `message.success(\`已复制 ${mac}\`)` on success and `message.error(...)` on failure (including a fallback for `Clipboard API unavailable` and non-Error throws).
  - Call site `onClick={() => copyMAC(...)}` wrapped to `onClick={() => { void copyMAC(...); }}` to preserve fire-and-forget at the JSX layer.
  - URL useEffect: removed `// eslint-disable-next-line react-hooks/exhaustive-deps`, added `searchParams` and `form` to deps so re-navigation re-reads URL; added `Object.keys(initial).length > 0` guard before `form.setFieldsValue(...)` to avoid AntD warning on empty initial state.
  - `handleExport` and `networkApi.ts` left untouched (fix-02 scope).

## Verification commands + outputs

```
$ cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit -p .
(no output, exit 0)

$ grep -n "useRef" xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx
15:import { useEffect, useRef, type FC } from 'react';
82:  const ranRef = useRef<number | null>(null);
83:  const cancelledRef = useRef(false);
(>= 2 satisfied: 3 matches)

$ grep -n "ranRef" xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx
80,82,86,87 (>= 1 satisfied)

$ grep -n "cancelledRef.current = true" xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx
98:      cancelledRef.current = true;
(>= 1 satisfied)

$ grep -n "typeof actionPath" xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx
34:    typeof actionPath === 'string' &&
(>= 1 satisfied)

$ grep -c "as string" xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx
0
(MUST be 0 satisfied)

$ grep -n "message.success.*已复制\|message.error.*复制失败" xingran-react-frontend/src/pages/network/mac/history.tsx
411:      message.success(`已复制 ${mac}`);
413:      message.error(err instanceof Error ? `复制失败:${err.message}` : '复制失败,请手动复制');
(>= 2 satisfied)

$ grep -c "eslint-disable.*react-hooks/exhaustive-deps" xingran-react-frontend/src/pages/network/mac/history.tsx
0
(MUST be 0 satisfied)

$ grep -n "searchParams, form\|searchParams,form" xingran-react-frontend/src/pages/network/mac/history.tsx
153:  }, [searchParams, form]);
(>= 1 satisfied)
```

## Deviations

None. Plan executed exactly as specified. All action items, all acceptance criteria, all verification commands met.

## Follow-ups

- The W4 effect now re-reads on every `searchParams` reference change (every URL mutation). In practice, only programmatic `setSearchParams` calls and nav re-mounts trigger it; re-navigation from other pages already remounts the component and gets a fresh read. No infinite-loop risk because the effect does not call `setSearchParams`.
- Recommend a smoke test: visit `/network/mac/history?deviceId=<uuid>&portName=GigabitEthernet0/0/1&mac=aa:bb:cc:dd:ee:ff&startTime=2026-06-15T00:00:00Z&endTime=2026-06-15T23:59:59Z` and confirm the form pre-fills with normalized MAC, device ID, port name, and custom range.
- Recommend a smoke test: click the `CopyOutlined` button in mobile card view and confirm `message.success` toast appears with the MAC value.

## Commit

`1104d472f6f16d413a6f34b6f2362c9205dae9bb` — 3 files changed, 37 insertions(+), 14 deletions(-).
