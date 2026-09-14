---
phase: 106
plan: "03"
type: execute
wave: 2
depends_on: ["01"]
files_modified:
  - xingran-react-frontend/src/pages/login/index.tsx
  - xingran-react-frontend/src/pages/operations/assets/index.tsx
  - xingran-react-frontend/src/pages/operations/floors/utils.ts
autonomous: true
requirements_addressed: [TS-01]
---

<objective>
Narrow or justify the three production (non-test) `as any` occurrences in the frontend codebase. login/index.tsx uses an unknown error that needs a proper type guard. assets/index.tsx passes params to a typed export function — the cast masks a type mismatch that should be resolved with a proper type. floors/utils.ts has an acceptable JSON-parse fallback that needs a justification comment.
</objective>

<tasks>

## Prerequisites

Plan 01 must be executed first (wave 2 depends on wave 1).

## Step 1 — `login/index.tsx` — narrow `error as any`

Read `src/pages/login/index.tsx` around line 28.

**Before (lines 26–38):**
```typescript
function extractLoginErrorMessage(error: unknown): string {
  if (!error) return "登录失败，请重试";
  const anyError = error as any;
  const respData = anyError?.response?.data;
  if (respData && typeof respData === "object") {
    if (typeof respData.message === "string" && respData.message) return respData.message;
    if (typeof respData.msg === "string" && respData.msg) return respData.msg;
  }
  if (typeof anyError?.message === "string" && anyError.message) {
    return anyError.message;
  }
```

**After:**
```typescript
function extractLoginErrorMessage(error: unknown): string {
  if (!error) return "登录失败，请重试";
  // Type-narrow unknown error via duck-typed property access.
  // The error shape is: { response?: { data?: { message?: string; msg?: string } }; message?: string }
  const respData = (error as { response?: { data?: Record<string, unknown> } })?.response?.data;
  if (respData && typeof respData === "object") {
    if (typeof respData.message === "string" && respData.message) return respData.message;
    if (typeof respData.msg === "string" && respData.msg) return respData.msg;
  }
  if (typeof (error as { message?: unknown })?.message === "string") {
    return (error as { message: string }).message;
  }
```

The key changes:
1. Remove `const anyError = error as any` — no longer needed.
2. Narrow `error` via explicit typed casts at each use site, using a minimal duck-type interface inline.
3. The final `anyError?.message` branch uses `(error as { message?: unknown })?.message` for the string check, then `(error as { message: string }).message` for the return — both are narrow casts, not `as any`.

## Step 2 — `assets/index.tsx` — define proper export params type

Read `src/pages/operations/assets/index.tsx` around line 255.

The call is:
```typescript
await assetApi.excel.export(params as any);
```

Where `params` is built from a spread of `searchValues` plus pagination:
```typescript
const params = {
  current: 1,
  pageSize: 10000,
  ...(searchValues as Record<string, unknown>),
};
```

**Fix:** Change the cast to use `Record<string, unknown>` consistently, which is what the API accepts:

```typescript
await assetApi.excel.export({ ...params });
```

Since `params` already has `current` and `pageSize` as `number`, and `searchValues` is spread as `Record<string, unknown>`, the resulting object is `Record<string, unknown>` — no `as any` needed if the type is declared explicitly:

```typescript
const params: Record<string, unknown> = {
  current: 1,
  pageSize: 10000,
  ...searchValues,
};
```

Then the call becomes:
```typescript
await assetApi.excel.export(params);
```

This removes the `as any` cast entirely.

## Step 3 — `floors/utils.ts` — add justification comment

Read `src/pages/operations/floors/utils.ts` around line 75.

**Before:**
```typescript
// eslint-disable-next-line @typescript-eslint/no-explicit-any
return value as any;
```

**After:**
```typescript
// acceptable: JSON.parse fallback for untyped cache data — T is inferred from call site
// eslint-disable-next-line @typescript-eslint/no-explicit-any
return value as any;
```

The `// eslint-disable-next-line` comment should remain; the new comment explains why `as any` is acceptable in this specific case (cache deserialization with typed call-site inference).

</tasks>

<success_criteria>
- `login/index.tsx` — no `as any` remaining in extractLoginErrorMessage; uses typed narrow casts
- `assets/index.tsx` — no `as any` in the asset export call; params typed as `Record<string, unknown>`
- `floors/utils.ts` — the `as any` at line 75 has an inline justification comment
- All TypeScript compiles with no new errors
</success_criteria>

<output>
Create SUMMARY.md on completion
</output>
