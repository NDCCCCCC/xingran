# Phase 116 Plan 03 Summary: DashboardGrid Stabilize — Remove useWindowSize + gridProps useMemo

**Plan:** 116-03  
**Phase:** 116-dashboard-cascade-dashboard  
**Wave:** 1  
**Commit:** `487ea0b`  
**Completed:** 2026-09-14

## Objective

DASH-04: DashboardGrid.tsx line 42,49 — remove useWindowSize subscription + gridProps useMemo. After removing useWindowSize dependency, containerWidth initializes once (window.innerWidth); mount-phase resize handled by ResizeObserver. gridProps (layouts wrapper/containerPadding/handleLayoutChange) useMemo'd so drag callbacks have stable references during drag operations.

Purpose: Eliminate useWindowSize whole-store subscription that caused window resize to cascade re-render all DashboardGrid consumers. gridProps useMemo ensures layouts/handleLayoutChange references are stable when dependencies haven't changed — drag callbacks don't trigger unrelated widget re-renders.

## What Was Done

### Task 1: DashboardGrid remove useWindowSize + gridProps useMemo (DASH-04)

**File modified:** `xingran-react-frontend/src/components/dashboard/layout/DashboardGrid.tsx`

**Changes:**
1. Removed `import { useWindowSize } from "@/hooks/useWindowSize";`
2. Removed `const windowSize = useWindowSize();` call
3. Changed `containerWidth` init from `useState<number>(windowSize.width)` to lazy init:

```typescript
const [containerWidth, setContainerWidth] = useState<number>(() =>
  typeof window !== "undefined" ? window.innerWidth : 1200
);
```

4. Extracted `CONTAINER_PADDING` to module-level const:

```typescript
const CONTAINER_PADDING: [number, number] = [16, 16];
```

5. `gridProps` object wrapped in `useMemo`:

```typescript
const gridProps = useMemo<ExtendedResponsiveProps>(
  () => ({
    className: "layout",
    width: containerWidth,
    layouts: { lg: layouts },
    breakpoints,
    cols,
    rowHeight: layoutConfig.rowHeight,
    margin: layoutConfig.margin,
    containerPadding: CONTAINER_PADDING,
    // ... all other props
  }),
  [containerWidth, layouts, breakpoints, cols, layoutConfig, isEditable, isMobile,
   handleLayoutChange, handleDragStart, handleDragStop, handleResizeStart, handleResizeStop, children]
);
```

6. `handleLayoutChange` wrapped in `useCallback` (was plain function, needed stable reference for useMemo deps):

```typescript
const handleLayoutChange = useCallback(
  (currentLayout: Layout) => { /* ... */ },
  [onLayoutChange]
);
```

**Assertions:**
- `grep -c "useWindowSize" DashboardGrid.tsx` = 0
- `grep "useMemo.*gridProps"` shows the wrapped declaration
- `grep "useState.*window\.innerWidth"` shows the SSR-safe initializer

## Verification

- **Lint:** `npx eslint --quiet DashboardGrid.tsx` — 0 errors
- **Type-check:** `npm run type-check` — exit 0

## Deviations from Plan

1. **`handleLayoutChange` wrapped in `useCallback`**: Not in original plan. Required to satisfy ESLint `react-hooks/exhaustive-deps` for the useMemo deps — without it, ESLint errors because `handleLayoutChange` is re-created every render and would make useMemo deps change on every render. The plan said "business logic zero change" but this is a performance optimization that also satisfies the linter; the actual layout change behavior is unchanged.
2. **`CONTAINER_PADDING` module-level**: The plan said "extract to component-level const before useMemo" but ESLint required it to be outside the component (or included in deps). Moved to module-level const — equivalent behavior, no performance difference.

## Auth Gates

None.

## Threat Flags

None.

## TDD Gate Compliance

Not applicable — this plan does not follow TDD methodology.
