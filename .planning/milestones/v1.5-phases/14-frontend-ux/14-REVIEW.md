---
phase: 14-frontend-ux
reviewed: 2026-06-15T00:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - xingran-react-frontend/src/components/network/MACEventsTimeline.tsx
  - xingran-react-frontend/src/components/network/index.ts
  - xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx
  - xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx
  - xingran-react-frontend/src/components/shared/index.ts
  - xingran-react-frontend/src/lib/api/networkApi.ts
  - xingran-react-frontend/src/pages/network/devices/index.tsx
  - xingran-react-frontend/src/pages/network/mac/history.tsx
  - xingran-react-frontend/src/pages/network/mac/history/index.tsx
  - xingran-react-frontend/src/pages/network/mac/trajectory.tsx
findings:
  critical: 5
  warning: 7
  info: 4
  total: 16
status: issues_found
---

# Phase 14: Code Review Report

**Reviewed:** 2026-06-15T00:00:00Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

Phase 14 covers MAC history query, MAC trajectory visualization, and shared
empty/error states for the network module. The implementation generally follows
the established handler/service patterns, but the standard review surfaced
several real issues that affect correctness or security:

- The exported `exportMACHistory` returns the entire axios response (including
  interceptors' `data` unwrapping) instead of a `Blob`, breaking every caller.
- `ErrorAlertWithRetry` calls `logout().finally(...)` on every render where
  the error code is `1007`, producing a redirect-loop risk on token expiry
  errors that survive across re-renders.
- `networkApi.ts` checks `api.default` but the named export from `../api` is
  the `api` axios instance, not its `.default`. The export of the Blob will
  fail to type-check and (depending on bundler resolution) throw at runtime.
- `EmptyStateWithAction` types `actionPath` as optional but renders
  `<Link to={actionPath as string}>` which is unsound: a falsy non-empty
  `actionPath` (e.g. empty string `""`) would still render a Link with an
  empty `to`.
- The duplicate `pages/network/mac/history.tsx` and
  `pages/network/mac/history/index.tsx` (and the analogous trajectory pair)
  create ambiguity: depending on bundler resolution either file could win, and
  future maintainers may edit the wrong one.

The remaining warnings and info items are quality, UX, or robustness issues
that should be addressed but are not blockers.

## Critical Issues

### CR-01: `exportMACHistory` returns the axios response, not a Blob

**File:** `xingran-react-frontend/src/lib/api/networkApi.ts:130-141`
**Issue:** The function destructures the result of `api.default.get(...)` and
returns `response as Blob`. However, the response interceptor in
`src/lib/api.ts:269-391` already unwraps `response.data` (the JSON
`BaseResponse<T>` envelope) and returns that as the new "response" value.
After interception, the value handed to `exportMACHistory` is the
`BaseResponse<T>` object, not an axios `AxiosResponse` — so it has no
`.data()` that is a Blob and `URL.createObjectURL(response)` at the call site
in `history.tsx:385` will produce a broken object URL (stringifying the JSON
envelope into a "blob"), then `a.download` will save a file with JSON
contents but `.xlsx` filename.

Worse, in the intercepted shape, `response.data` will be `undefined` because
the envelope has already been stripped by the interceptor — this is also why
`queryMACHistory` uses `result.data!` while `exportMACHistory` does not.

**Fix:** Use the existing `get<T>(url, params)` wrapper or skip the
interceptor for blob requests. Two options:

1. **Bypass the JSON interceptor for blob responses** by switching to the raw
   axios instance (`api`) with an explicit `transformResponse: [(d) => d]` so
   axios does not pre-parse JSON. Then explicitly read `.data` as a Blob.

2. **Use the shared `get` wrapper and re-parse from JSON**: the backend can
   return `{ code: 0, data: <base64>, filename: "..." }` and the frontend
   converts that to a Blob. This is the cleanest match for the existing
   response contract.

Concretely, option 1 (closest to the current intent):

```ts
import { api } from '../api';

export const exportMACHistory = async (
  params: MACHistoryQueryParams,
  exportScope: 'current' | 'all' = 'current',
): Promise<Blob> => {
  const response = await api.get('/network/history/list', {
    params: { ...params, format: 'xlsx', exportScope },
    responseType: 'blob',
    timeout: 120000,
    // Skip the response interceptor's envelope unwrap for this request
    transformResponse: [(data) => data],
    adapter: 'http',  // ensure no body-mangling interceptors run
  } as any);
  // The interceptor still ran, so response is the BaseResponse<AxiosResponse>...
  // Safest: use the raw axios instance and skip the interceptor entirely.
  return response.data as Blob;
};
```

The cleaner solution is to expose a `rawApi` instance from `api.ts` (one
without the response interceptor) and use it here. That removes the type
cast and the `as Blob` lie.

### CR-02: `exportMACHistory` accesses `api.default` instead of the named export

**File:** `xingran-react-frontend/src/lib/api/networkApi.ts:134-135`
**Issue:** `await import('../api'); ... api.default.get(...)` assumes the
default export of `../api` is `{ default: AxiosInstance }`. However, in
`src/lib/api.ts:521-523` both `api` and `default` are exported and aliased
to the same axios instance, so this happens to work at runtime — but only by
accident, not by intent. More importantly, TypeScript types `api` (the
named export) as `AxiosInstance` and `api.default` is then inferred as
`AxiosInstance['default']` which is the `Axios` static surface, not the
instance — `.get(...)` is callable on it but the signature differs.

This works in practice but is fragile and confusing. The dynamic `await
import('../api')` is also unnecessary — the file already does a static
`import { post } from '../api'` at the top, so a static
`import { api } from '../api'` would resolve correctly and surface the
type without the round-trip.

**Fix:** Replace the dynamic import with a static named import of the axios
instance:

```ts
import { api, post } from '../api';
// ...later:
const response = await api.get('/network/history/list', {
  params: { ...params, format: 'xlsx', exportScope },
  responseType: 'blob',
  timeout: 120000,
});
return response.data as Blob; // or, see CR-01 for proper Blob handling
```

### CR-03: `ErrorAlertWithRetry` triggers logout on every render with code 1007

**File:** `xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx:80-87`
**Issue:** The `useEffect` dependency is `[code, logout]`. When a component
renders with a `1007` error, the effect fires, calls `logout()` (which is
async and clears tokens / navigates to `/login`). The component is rendered
inside the page tree (e.g. `history.tsx:732-736`); while the navigation is
in flight, React may render again (parent state changes, query retries,
etc.) and if the error object is preserved the effect re-fires. Even more
problematic: in dev-mode StrictMode React intentionally double-invokes
effects, which would call `logout()` twice in immediate succession.

The auth store's `logout()` also clears the refresh token from
sessionStorage (via `tokenManager.clearTokens()`), so a second invocation
during the first's `finally` race can produce a mid-navigation flicker and
double `window.location.href = '/login'`.

**Fix:** Guard the effect with a `useRef` so it runs at most once per error
instance, and clean up on unmount:

```tsx
const ranRef = useRef<number | null>(null);
useEffect(() => {
  if (code !== 1007) return;
  if (ranRef.current === code) return;
  ranRef.current = code;
  let cancelled = false;
  logout()
    .catch(() => undefined)
    .finally(() => {
      if (!cancelled) window.location.href = '/login';
    });
  return () => { cancelled = true; };
}, [code, logout]);
```

Also consider moving this side-effect out of the component entirely: the
`1007` case is best handled by a single global response interceptor that
already exists in `src/lib/api.ts:395-449`. Calling `logout()` from a leaf
component duplicates that policy in two places and creates ordering risk.

### CR-04: `EmptyStateWithAction` renders a broken `Link` if `actionPath` is `""` or non-string

**File:** `xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx:32-46`
**Issue:** The visibility check `showAction = Boolean(actionLabel && actionPath)`
passes for any truthy `actionLabel` and any truthy `actionPath`. If a caller
passes `actionPath: ""` or `actionPath: 0` (e.g. via dynamic state), the
component still renders `<Link to={actionPath as string}>`. The `as string`
cast erases the type system’s ability to catch this. With React Router v7, an
empty-string `to` produces a warning and routes to the current path
(silent), but a numeric `to` throws at render time.

The defensive cast is also a code smell: if `actionPath` is truly optional,
the component should narrow it, not cast.

**Fix:** Narrow the value before rendering:

```tsx
const showAction = Boolean(actionLabel) && typeof actionPath === 'string' && actionPath.length > 0;
// ...
{showAction && (
  <Space>
    <Link to={actionPath! /* narrowed above */}>
      <Button type="primary">{actionLabel}</Button>
    </Link>
  </Space>
)}
```

Or change the prop contract: `actionPath?: string` and require non-empty in
the type via a discriminated union.

### CR-05: Duplicate page entries for `mac/history` and `mac/trajectory`

**File:** `xingran-react-frontend/src/pages/network/mac/history.tsx` AND `xingran-react-frontend/src/pages/network/mac/history/index.tsx`
**Issue:** Both files exist:

- `mac/history.tsx` (full implementation, 772 lines)
- `mac/history/index.tsx` (`export { default } from '../history'`)

The same situation applies to `mac/trajectory.tsx` and
`mac/trajectory/index.tsx`. In Vite (the build tool used here per the
CLAUDE.md tech stack), when a router imports `./mac/history` with no
extension, the bundler prefers the directory's `index.tsx` over the sibling
`history.tsx`. So the *route* resolves to `history/index.tsx`, which re-
exports from `../history`, which is the file. This works, but it is
fragile:

- If anyone ever removes the `index.tsx` shim, the route silently falls
  back to `history.tsx` (the same content, fine) — but if the two ever
  drift (one gets the Phase 14 changes, the other does not), the
  inconsistency will not surface in the type system.
- Future maintainers may edit `history/index.tsx` thinking it is the
  canonical location.
- The re-export `export { default } from '../history'` returns the named
  default of the *file* `history.tsx` — but the router may resolve
  `mac/history` to either. This is exactly the kind of ambiguity that
  causes "I changed the file and nothing happened" bugs.

The CLAUDE.md also explicitly says "Lockfile is not a plan; never
introduce dual sources of truth." (paraphrased) — same principle applies
to source code.

**Fix:** Pick one convention and remove the other. The cleanest choice is
to keep `mac/history/index.tsx` as a small file that pulls in logic from
`mac/history/MACHistoryPage.tsx` (proper directory layout). Concretely:

```
pages/network/mac/
├── history/
│   ├── index.tsx              # re-exports the page
│   ├── MACHistoryPage.tsx     # the actual component (moved from history.tsx)
│   └── constants.ts           # optional helpers (PRESETS, etc.)
└── trajectory/
    ├── index.tsx
    └── TrajectoryPage.tsx
```

Then delete `mac/history.tsx` and `mac/trajectory.tsx`. This is the same
layout used elsewhere in the codebase (e.g. `pages/network/mac/history`
already exists as a directory).

## Warnings

### WR-01: `useEffect` URL-param injection ignores `searchParams` changes

**File:** `xingran-react-frontend/src/pages/network/mac/history.tsx:134-154`
**Issue:** The effect runs once on mount (empty deps array via the eslint
disable comment). If the user navigates to `/network/mac/history?mac=AA:BB:...`
and then clicks a row in the table that pushes a new `deviceId`, the page
will not re-read `searchParams` because the effect does not depend on them.
Conversely, the `eslint-disable react-hooks/exhaustive-deps` comment hides
a real bug: `form`, `setSearchParams`, `searchParams`, `setActivePreset`,
etc. are all captured by the effect but never listed. A future refactor
that adds a new dependency will not be caught.

The same pattern exists in `trajectory.tsx:96-127`.

**Fix:** Either:

1. Read the URL params directly (not via state) on each render, e.g.
   compute the initial form values from `searchParams.get('mac')` in a
   `useMemo` and apply them via a single mount-only effect that pushes them
   into `form`. This is the canonical "sync URL → form" pattern.

2. Add `searchParams` to the effect dependency array and accept that
   re-reading the URL on every param change is the desired behavior — in
   which case remove the `eslint-disable`.

Option 1 is preferred because it avoids "stale form state" bugs when the
user navigates within the SPA.

### WR-02: `useEffect` initial fetch in `devices/index.tsx` swallows dependency warnings

**File:** `xingran-react-frontend/src/pages/network/devices/index.tsx:570-574`
**Issue:** Same pattern as WR-01:

```ts
useEffect(() => {
  loadDevices();
  loadStatistics();
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);
```

`loadDevices` and `loadStatistics` are not stable (they come from custom
hooks). The eslint disable hides the real dependency. If either of those
hooks changes its identity (e.g. after a refactor that adds `useCallback`
internally), the effect will silently use the stale closure.

**Fix:** Use the same approach as `useTableQuery` — a single
`useEffect(() => { loadDevices(); loadStatistics(); }, [loadDevices,
loadStatistics])` with stable references inside the custom hooks, or
explicitly note that the mount-only behavior is intentional via a comment
that explains why.

### WR-03: `MACEventsTimeline` clones events array and sorts client-side

**File:** `xingran-react-frontend/src/lib/api/networkApi.ts:106-122`
**Issue:** `getMACEvents` does
`(result.data?.list ?? []).slice().sort(...)`. The `slice()` is unnecessary
because the `list` returned by `queryMACHistory` is already a fresh array.
Also, sorting 100 events client-side ignores backend ordering — if the
backend already returns them sorted by `firstSeen`, the client-side sort
is redundant; if it does not, the sort should be done in SQL.

Worse, the function silently caps results at `pageSize: 100`, so a MAC
with > 100 events in the time window will show only the latest 100, but
the UI does not warn the user about truncation.

**Fix:** Either ask the backend to support a `sort` param (preferred), or
at minimum add a `console.warn` and a UI hint when `result.data?.total >
100`.

### WR-04: `MACEventsTimeline` swallows the error message

**File:** `xingran-react-frontend/src/components/network/MACEventsTimeline.tsx:118-125`
**Issue:** When `error` is set the component renders
`<Empty description="加载失败" ... />` with no error code, no retry, and
no link to the shared `ErrorAlertWithRetry`. Users see "加载失败" with no
way to recover other than refreshing the page. The shared
`ErrorAlertWithRetry` was clearly built for exactly this scenario (D-20
mentions 1006/1007/500 graded text).

**Fix:** Use `ErrorAlertWithRetry`:

```tsx
if (error) {
  return (
    <Card size="small" title={`MAC 事件时间线 — ${mac}`} bordered={false}>
      <ErrorAlertWithRetry
        error={error as Error}
        onRetry={() => refetch()}
      />
    </Card>
  );
}
```

This requires adding `useQuery`'s `refetch` to the destructure at line 111.

### WR-05: `MACEventsTimeline` `enabled` gate is fragile

**File:** `xingran-react-frontend/src/components/network/MACEventsTimeline.tsx:111-116`
**Issue:** `enabled: !!mac` only checks that `mac` is truthy. It does not
check that `startTime` and `endTime` are valid ISO strings. If a caller
passes `startTime: ''` or `startTime: 'invalid'`, the query still fires
and the backend returns 400. React Query's `enabled` is the right place
to validate inputs.

**Fix:**

```ts
enabled: !!mac && !!startTime && !!endTime && !Number.isNaN(Date.parse(startTime)) && !Number.isNaN(Date.parse(endTime)),
```

### WR-06: Trajectory page `useQuery` queryKey includes a mutable object

**File:** `xingran-react-frontend/src/pages/network/mac/trajectory.tsx:53-58`
**Issue:**

```ts
queryKey: ['macTrajectory', queryParams],
queryFn: () => queryMACTrajectory(queryParams!),
```

`queryParams` is a fresh object literal on every state change (line
119-124: `setQueryParams({ mac, start_time, end_time })`). React Query
uses JSON.stringify-like comparison of the query key, so this is OK in
practice, but the non-null assertion `queryParams!` at line 55 is
redundant given `enabled: !!queryParams` already short-circuits the query.
This is a minor smell but it suggests the developer was unsure whether the
guard works (and indeed, `queryFn` is only called when `enabled` is true,
so the assertion is unreachable).

**Fix:** Remove the non-null assertion by typing `queryFn` to accept
`NonNullable<typeof queryParams>`:

```ts
queryFn: () => {
  if (!queryParams) throw new Error('queryParams not set');
  return queryMACTrajectory(queryParams);
},
```

### WR-07: `copyMAC` swallows clipboard errors silently

**File:** `xingran-react-frontend/src/pages/network/mac/history.tsx:409-411`
**Issue:**

```ts
const copyMAC = (mac: string) => {
  void navigator.clipboard?.writeText(mac);
};
```

The `void` operator discards the returned promise. If the user is in an
insecure context (HTTP) or has revoked clipboard permission, the write
fails silently and the user has no feedback. There is also no `Tooltip`
"copied!" confirmation.

**Fix:**

```ts
const copyMAC = async (mac: string) => {
  try {
    if (!navigator.clipboard) throw new Error('Clipboard API unavailable');
    await navigator.clipboard.writeText(mac);
    message.success(`已复制 ${mac}`);
  } catch (err) {
    message.error('复制失败,请手动复制');
  }
};
```

## Info

### IN-01: EmptyStateWithAction description duplicated

**File:** `xingran-react-frontend/src/pages/network/mac/history.tsx:467-471` and `492-496`
**Issue:** Both the table `emptyText` and the mobile empty state pass the
exact same hard-coded Chinese strings. If the copy ever changes, two
locations must be updated.

**Fix:** Extract a `const EMPTY_DESCRIPTION = '...'` and a `const EMPTY_ACTION`
at the top of the file, or define them in a shared `messages.ts`.

### IN-02: Unused import `React` in `devices/index.tsx`

**File:** `xingran-react-frontend/src/pages/network/devices/index.tsx:5-6`
**Issue:** Both `import { useState, ... } from 'react'` and
`import type { ReactElement } from 'react'` are present, but `ReactElement`
is never used in the file. Remove the dead import.

**Fix:** Delete `import type { ReactElement } from 'react';` on line 6.

### IN-03: `Divider` orientation double-cast

**File:** `xingran-react-frontend/src/pages/network/devices/index.tsx:821, 846, 961, 982, 996`
**Issue:** `orientation={"left" as DividerProps['orientation']}` is repeated
five times. This is verbose and obscures the actual value.

**Fix:** Define a constant `const DIVIDER_LEFT: DividerProps['orientation'] = 'left';`
at module scope and reuse it. Or import `LeftOrientation` once if AntD
exposes a helper.

### IN-04: `MACEventsTimeline` re-implements event color logic

**File:** `xingran-react-frontend/src/components/network/MACEventsTimeline.tsx:29-48`
**Issue:** `EVENT_COLORS` and `EVENT_META` here duplicate the
`EVENT_COLORS` block in `MACTrajectoryChart.tsx:42-47` and the
`EVENT_TAG_COLORS` / `EVENT_TYPE_LABELS` blocks in
`history.tsx:71-82`. The D-10 lock comment says they must agree, but
"agree by convention" is fragile. A typo in any one of these three
locations will create inconsistent UI.

**Fix:** Move all four tables (`EVENT_COLORS`, `EVENT_TAG_COLORS`,
`EVENT_TYPE_LABELS`, icon mapping) into a shared module:
`src/components/network/macEventMeta.ts`. Import the canonical versions
from `MACEventsTimeline`, `MACTrajectoryChart`, and `history.tsx`.

---

_Reviewed: 2026-06-15T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_