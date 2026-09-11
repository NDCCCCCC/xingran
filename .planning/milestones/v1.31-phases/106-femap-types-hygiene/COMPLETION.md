# Phase 106 COMPLETION

**Phase:** 106 | **Status:** COMPLETED | **Date:** 2026-09-08
**Goal:** 前端展示映射收敛共享常量 + 类型卫生（FEMAP-01~03 + TS-01~02）

---

## Summary

4 plans (106-01 ~ 106-04) in 2 waves, all completed green.

| Plan | Wave | Changes | Requirements |
|------|------|---------|-------------|
| 106-01 | 1 | `constants/status.ts` 新增 6 个常量；fixStatusColor/fixStatusLabel 迁出；修复 `rolled_back: "magenta"` → `"orange"`；`StatusOption.value` 拓宽为 `number \| string` | FEMAP-01, FEMAP-02 |
| 106-02 | 1 | server-rooms/MAC history/MAC list 三页面内联 Tag 替换为集中常量 | FEMAP-02, FEMAP-03 |
| 106-03 | 2 | `as any` 逐处收窄（login error narrow cast / assets params Record / floors justification comment） | TS-01 |
| 106-04 | 2 | 4 处 bare `eslint-disable` 补 `-- reason` 后缀 | TS-02 |

---

## Key Fixes

- **`rolled_back` 颜色不一致**: `FixSuggestionDetailDrawer.tsx` 用 `"magenta"`，`index.tsx` 用 `"orange"` → 统一为 `"orange"` via `FIX_STATUS_TAG_CONFIG`
- **`StatusOption.value` 类型冲突**: `number` 类型键无法容纳 string-keyed FixStatus/MAC type → 拓宽为 `number | string`

---

## New Constants Added to `constants/status.ts`

| Constant | Type | Values |
|----------|------|--------|
| `FIX_STATUS_OPTIONS` | `StatusOption[]` | pending/accepted/rejected/applied/rolled_back/failed |
| `FIX_STATUS_TAG_CONFIG` | `Record<string, ...>` | gold/blue/default/green/orange/red |
| `MAC_HISTORY_STATUS_OPTIONS` | `StatusOption[]` | 0=正常, 1=停用 |
| `MAC_HISTORY_STATUS_TAG_CONFIG` | `StatusTagConfig` | 0→green, 1→red |
| `MAC_TYPE_OPTIONS` | `StatusOption[]` | dynamic/static/secure |
| `MAC_TYPE_TAG_CONFIG` | `Record<string, ...>` | blue/green/orange |

---

## Regression Gates

| Gate | Result |
|------|--------|
| `npm run type-check` | ✅ 0 errors |
| `npm run lint` | ✅ 0 errors (1377 warnings pre-existing) |
| `vitest apiFactory.invariants.test.ts` | ✅ 10/10 passed |

---

## Files Modified

- `xingran-react-frontend/src/constants/status.ts`
- `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx`
- `xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx`
- `xingran-react-frontend/src/pages/operations/server-rooms/index.tsx`
- `xingran-react-frontend/src/pages/network/mac/history/MACHistoryPage.tsx`
- `xingran-react-frontend/src/pages/network/mac/index.tsx`
- `xingran-react-frontend/src/pages/login/index.tsx`
- `xingran-react-frontend/src/pages/operations/assets/index.tsx`
- `xingran-react-frontend/src/pages/operations/floors/utils.ts`
- `xingran-react-frontend/src/App.tsx`
- `xingran-react-frontend/src/components/TargetSelector.tsx`
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx`
- `xingran-react-frontend/src/pages/system/user/constants.ts` (StatusOption widening)

---

## Next Step

Phase 107 — TODO 清零 + nilness 排查（22 处非测试 TODO 逐项决策 + NIL-01 根因闭环）
