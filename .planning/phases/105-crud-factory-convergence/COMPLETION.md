# Phase 105 COMPLETION

**Phase:** 105 | **Status:** COMPLETED | **Date:** 2026-09-08
**Goal:** 前端 CRUD 收敛 apiFactory — adDomainApi / knowledgeApi / dutyApi / workorderApi 19 处手写五件套迁移 `createResourceApi`

---

## Summary

4 plans (105-01 ~ 105-04) in 2 waves, all completed green.

| Plan | File | Changes | WARNING_WHITELIST |
|------|------|---------|-----------------|
| 105-01 | adDomainApi.ts | `groupCrud` + `userCrud` added; `updateADGroup` → `groupCrud.update`; `updateADUser` → `userCrud.update` | adDomainApi: 1 (was 3) |
| 105-02 | knowledgeApi.ts | `tagCrud` added; `updateKnowledgeArticle` → `articleCrud.update`; `updateKnowledgeTag` → `tagCrud.update` | knowledgeApi: 3 (was 5) |
| 105-03 | dutyApi.ts | `holidayCrud` added; `updateDutyPool` → `dutyPoolCrud.update`; `updateHoliday` → `holidayCrud.update` | dutyApi: 3 (was 5) |
| 105-04 | workorderApi.ts | `updateWorkOrder` → `orderCrud.update`; `updatePeriodicTemplate` → `periodicCrud.update` | workorderApi: 4 (was 6) |

---

## Invariants Baseline (apiFactory.invariants.test.ts)

```
adDomainApi:   1  (down from 3)  — deleteADConfig KEEP
knowledgeApi:   3  (down from 5)  — deleteKnowledgeArticle/category/tag KEEP
dutyApi:       3  (down from 5)  — deleteDutyPool/schedule/holiday KEEP
workorderApi:  4  (down from 6)  — deleteWorkOrder/category/PeriodicTemplate KEEP
```

D-14 单参 delete 契约（`post('/resource/${id}/delete')` 保持不变）全部保留。

---

## Regression Gates

| Gate | Result |
|------|--------|
| `npm run type-check` | ✅ 0 errors |
| `npm run lint` | ✅ 0 errors (1378 warnings pre-existing) |
| `vitest --run src/lib/apiFactory.invariants.test.ts` | ✅ 10/10 passed |

---

## Files Modified

- `xingran-react-frontend/src/lib/adDomainApi.ts`
- `xingran-react-frontend/src/lib/knowledgeApi.ts`
- `xingran-react-frontend/src/lib/dutyApi.ts`
- `xingran-react-frontend/src/lib/workorderApi.ts`
- `xingran-react-frontend/src/lib/apiFactory.invariants.test.ts`

---

## Next Step

Phase 106 — 前端映射统一与类型卫生（FEMAP-01~03 + TS-01~02）
