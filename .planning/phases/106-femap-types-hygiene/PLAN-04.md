---
phase: 106
plan: "04"
type: execute
wave: 2
depends_on: ["01"]
files_modified:
  - xingran-react-frontend/src/App.tsx
  - xingran-react-frontend/src/components/TargetSelector.tsx
  - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx
  - xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx
autonomous: true
requirements_addressed: [TS-02]
---

<objective>
Add a `-- reason` suffix to all bare `eslint-disable-next-line react-hooks/exhaustive-deps` comments that lack justification. These four files are the confirmed bare-disable locations identified in the Phase 106 research. Audit all modified files to ensure no new bare disables are introduced.
</objective>

<tasks>

## Prerequisites

Plan 01 must be executed first (wave 2 depends on wave 1).

## Step 1 — `App.tsx` line 35

Read `src/App.tsx` around line 35.

**Before:**
```tsx
useEffect(() => {
  if (allMenus.length > 0) {
    routeConfigManager.initialize(allMenus);
  }
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [allMenus.length]);
```

**After:**
```tsx
useEffect(() => {
  if (allMenus.length > 0) {
    routeConfigManager.initialize(allMenus);
  }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- reason: allMenus.length is intentionally the only trigger; routeConfigManager.initialize is stable and should not re-run on menu metadata changes
}, [allMenus.length]);
```

## Step 2 — `TargetSelector.tsx` line 91

Read `src/components/TargetSelector.tsx` around line 91.

**Before:**
```tsx
// eslint-disable-next-line react-hooks/exhaustive-deps
}, [targetType]);
```

**After:**
```tsx
// eslint-disable-next-line react-hooks/exhaustive-deps -- reason: targetType is the only trigger; dependent callbacks (loadDepts/loadUsers) are stable useCallback refs
}, [targetType]);
```

## Step 3 — `fix-suggestion/index.tsx` line 146

Read `src/pages/asset/reconciliation/fix-suggestion/index.tsx` around line 146.

**Before:**
```tsx
useEffect(() => {
  form.setFieldsValue(filterValues);
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);
```

**After:**
```tsx
useEffect(() => {
  form.setFieldsValue(filterValues);
  // eslint-disable-next-line react-hooks/exhaustive-deps -- reason: initializing form with filterValues from URL params; no reactive deps needed (setFieldsValue is stable)
}, []);
```

## Step 4 — `exceptions/index.tsx` line 166

Read `src/pages/asset/reconciliation/exceptions/index.tsx` around line 166.

**Before:**
```tsx
useEffect(() => {
  form.setFieldsValue(filterValues);
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);
```

**After:**
```tsx
useEffect(() => {
  form.setFieldsValue(filterValues);
  // eslint-disable-next-line react-hooks/exhaustive-deps -- reason: initializing form with filterValues from URL params; no reactive deps needed (setFieldsValue is stable)
}, []);
```

## Step 5 — Verify no new bare disables

After making all changes, run:

```bash
cd xingran-react-frontend
grep -n "eslint-disable" src/App.tsx src/components/TargetSelector.tsx src/pages/asset/reconciliation/fix-suggestion/index.tsx src/pages/asset/reconciliation/exceptions/index.tsx
```

Confirm all `eslint-disable-next-line` lines end with `-- reason`.

Run ESLint on the four files:
```bash
cd xingran-react-frontend
npx eslint src/App.tsx src/components/TargetSelector.tsx src/pages/asset/reconciliation/fix-suggestion/index.tsx src/pages/asset/reconciliation/exceptions/index.tsx --quiet
```

</tasks>

<success_criteria>
- All four `eslint-disable-next-line react-hooks/exhaustive-deps` comments have a trailing `-- reason` suffix with a meaningful justification
- No new bare eslint-disable comments are introduced in these four files
- ESLint quiet run passes with no new errors
</success_criteria>

<output>
Create SUMMARY.md on completion
</output>
