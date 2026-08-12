## Summary

**refactor(14): consolidate macEventMeta + dedupe page files (W2/W5/W6/W7)**

Wave 3 of Phase 14 gap closure. Single-source-of-truth refactor + 4 WARNING-level UX issues resolved.

### Commit
- **Hash**: `b46f176`
- **Message**: `refactor(14): consolidate macEventMeta + dedupe page files (W2/W5/W6/W7)`
- **Files changed**: 9 (3 modified components, 2 new components, 2 new pages, 2 re-pointed index.tsx, 2 deleted, 1 modified API) — git auto-detected history.tsx → history/MACHistoryPage.tsx and trajectory.tsx → trajectory/TrajectoryPage.tsx as renames

### Key changes

**W7 (IN-04) — Event metadata single source of truth**
- NEW `xingran-react-frontend/src/components/network/macEventMeta.ts` exports `EVENT_COLORS`, `EVENT_ICON`, `EVENT_LABEL`, `EVENT_TAG_COLOR`, and `MACEventType` type
- `MACTrajectoryChart.tsx`: deleted local `EVENT_COLORS` (lines 42-47), imports from `macEventMeta` (with `as MACEventType` cast to align `TrajectoryNode.eventType` to canonical type)
- `MACEventsTimeline.tsx`: deleted local `EVENT_COLORS` + `EVENT_META` (lines 29-48); now imports `EVENT_COLORS`, `EVENT_ICON`, `EVENT_LABEL`, `EVENT_TAG_COLOR` from `macEventMeta`; uses explicit per-field lookups (`EVENT_ICON[eventType]`, `EVENT_TAG_COLOR[eventType]`, etc.) with a `?? 'appeared'` fallback
- `MACHistoryPage.tsx` (was `history.tsx`): deleted local `EVENT_TAG_COLORS` + `EVENT_TYPE_LABELS` (lines 71-82), imports `EVENT_LABEL` and `EVENT_TAG_COLOR` from `@/components/network/macEventMeta`
- `components/network/index.ts` barrel updated to re-export all 4 maps + `MACEventType` type

**W5 (WR-04) — Timeline error state uses shared component**
- `MACEventsTimeline.tsx` error branch (was lines 119-125) replaced `<Empty description="加载失败" image={Empty.PRESENTED_IMAGE_SIMPLE} />` with `<ErrorAlertWithRetry error={error as Error} onRetry={() => { void refetch(); }} />`
- `useQuery` destructure extended to `const { data: events, isLoading, error, refetch } = useQuery({...})`

**W6 (WR-03) — Truncation warning in getMACEvents**
- `getMACEvents` in `lib/api/networkApi.ts` now destructures `result.data.list` and `result.data.total`; if `total > list.length` (i.e. backend has more events than `pageSize: 100` returned), emits `console.warn(\`[getMACEvents] 事件被截断:total=${total}, returned=${list.length}。前端 pageSize=100 上限。考虑添加 sort 扩展或扩大分页。\`)`. Preserves `slice().sort()` ORDER BY DESC fallback.

**W2 (CR-05) — Dedupe page files into directory structure**
- `pages/network/mac/history.tsx` → `pages/network/mac/history/MACHistoryPage.tsx` (preserves fix-03 `copyMAC` async + `message.success/error` feedback AND URL effect with `searchParams` dep + `Object.keys(initial).length > 0` guard)
- `pages/network/mac/trajectory.tsx` → `pages/network/mac/trajectory/TrajectoryPage.tsx` (verbatim copy — trajectory page never had local `EVENT_COLORS`/`EVENT_LABEL`, so no consolidation needed)
- `pages/network/mac/history/index.tsx` updated to `export { default } from './MACHistoryPage'`
- `pages/network/mac/trajectory/index.tsx` updated to `export { default } from './TrajectoryPage'`
- Old `history.tsx` and `trajectory.tsx` deleted (git auto-detected as renames at 97% and 99% similarity)

### Verification commands + outputs

```bash
$ cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit -p .
EXIT: 0

$ test -f src/components/network/macEventMeta.ts && echo PASS
PASS
$ test -f src/pages/network/mac/history/MACHistoryPage.tsx && echo PASS
PASS
$ test -f src/pages/network/mac/trajectory/TrajectoryPage.tsx && echo PASS
PASS
$ test ! -f src/pages/network/mac/history.tsx && echo PASS
PASS
$ test ! -f src/pages/network/mac/trajectory.tsx && echo PASS
PASS

$ grep -l "from './macEventMeta'\|from '@/components/network/macEventMeta'" \
    src/components/network/MACEventsTimeline.tsx \
    src/components/network/MACTrajectoryChart.tsx \
    src/pages/network/mac/history/MACHistoryPage.tsx
3 files match (count check passes)

$ grep -c "EVENT_TAG_COLORS\|EVENT_TYPE_LABELS" \
    src/pages/network/mac/history/MACHistoryPage.tsx
0 (no local copies remain)

$ grep -c "ErrorAlertWithRetry" src/components/network/MACEventsTimeline.tsx
2 (1 import + 1 usage)

$ grep -n "console\.warn.*截断\|console\.warn.*truncation\|console\.warn.*getMACEvents" \
    src/lib/api/networkApi.ts
122:    console.warn(`[getMACEvents] 事件被截断:total=${total}, returned=${list.length}。前端 pageSize=100 上限。考虑添加 sort 扩展或扩大分页。`);

$ grep -c "EVENT_COLORS = {" src/components/network/MACTrajectoryChart.tsx
0 (local literal removed)
```

### Deviations from plan

None. All 7 action items executed exactly as specified. Cast `as MACEventType` added in MACTrajectoryChart when reading `node.eventType` to align the local `TrajectoryNode.eventType` union with the canonical `MACEventType` exported by `macEventMeta.ts`. Cast `as keyof typeof EVENT_TAG_COLOR` and `as keyof typeof EVENT_LABEL` added in MACHistoryPage where `record.eventType` is a generic `string` (the API type is loose) — same defensive pattern, just slightly different syntax to satisfy TS Record lookup.

### Follow-ups

- E2E smoke (out of band, per plan): start dev server, navigate to `/network/mac/history`, expand a row, verify timeline loads; trigger error in DevTools throttle, verify `ErrorAlertWithRetry` renders with retry button. Not run during this autonomous plan execution.
- Vite glob (`src/pages/**/*.tsx`) auto-picks the new `history/index.tsx` and `trajectory/index.tsx` re-exports — no router changes needed.
- Fix-05b mobile card view final styling (referenced in MACHistoryPage header comment) still pending per upstream `14-05b` plan.
