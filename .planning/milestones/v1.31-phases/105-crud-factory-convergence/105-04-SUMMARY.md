# Phase 105 Plan 04 — SUMMARY

## Plan
- 04

## Status
- completed

## Changes

| Function | Before | After |
|---|---|---|
| `updateWorkOrder` | `post("/workorder/orders/${id}/update", data)` | `orderCrud.update(id, data)` |
| `updatePeriodicTemplate` | `post("/workorder/periodic/templates/${id}/update", data)` | `periodicCrud.update(id, data)` |

## D-14 KEEP (unchanged)
- `deleteWorkOrder` — single-param `post(url)` contract locked by existing tests
- `deleteWorkOrderCategory` — single-param `post(url)` contract
- `deletePeriodicTemplate` — single-param `post(url)` contract

## WARNING_WHITELIST update
- `workorderApi`: 6 → **4**
- Comment: `deleteWorkOrder/category/PeriodicTemplate 单参直调 KEEP（D-14）；orders update/periodic update 已委托`

## Regression
- `npm run type-check` — exit 0
- `npm run lint` — exit 0 (1378 warnings, 0 errors, all pre-existing)
- `npm run test -- --run src/lib/apiFactory.invariants.test.ts` — 10 passed

## Files modified
- `xingran-react-frontend/src/lib/workorderApi.ts`
- `xingran-react-frontend/src/lib/apiFactory.invariants.test.ts`
