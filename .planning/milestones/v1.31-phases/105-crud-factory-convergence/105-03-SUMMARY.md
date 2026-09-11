# Phase 105 Plan 03 Summary

- plan: 03
- status: completed
- changes:
  - `holidayCrud` added (`createResourceApi<Holiday>({ basePath: "/duty/holidays" })`)
  - `updateDutyPool` migrated → `dutyPoolCrud.update`
  - `updateHoliday` migrated → `holidayCrud.update`
- D-14 KEEP unchanged: `deleteDutyPool` (line 196-199), `deleteDutySchedule` (line 232-234), `deleteHoliday` (line 278-280)
- WARNING_WHITELIST: `dutyApi: 3` (down from 5)
- regression: type-check + lint + vitest all green
