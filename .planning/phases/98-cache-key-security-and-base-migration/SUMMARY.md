---
phase: 98
phase_name: 缓存键安全与 base 迁移收尾
status: completed
completed_at: 2026-09-06
plans:
  - plan_id: 98-01
    status: completed
    commit: 5fd062e
    description: V130R-04 列表缓存键防碰撞修复
  - plan_id: 98-02
    status: completed
    commit: 9a40341
    description: V130R-05 四包 interface{} GetOrSet 迁 base.GetOrSetJSON[T]
requirements:
  - V130R-04: COMPLETE
  - V130R-05: COMPLETE
---

# Phase 98 Summary: 缓存键安全与 base 迁移收尾

## Delivered

### V130R-04: 列表缓存键防碰撞（Plan 98-01）
- **commit**: `5fd062e`
- **Fix**: `EscapeCacheKeyValue()` 对所有用户输入参数值转义 `:` → `%3A`
- **Files**: `system/user_cache_impl.go`（7 字段）+ `system/role_cache_impl.go`（2 字段）
- **Regression test**: `user_cache_key_collision_test.go` — `Username="bob:status:1"` vs `Username="bob"`+`Status=1` 生成不同键 ✓

### V130R-05: 四包 interface{} GetOrSet 迁 base.GetOrSetJSON[T]（Plan 98-02）
- **commit**: `9a40341`
- **Migration**: 11 处全部迁移

| Package | Method | Type |
|---------|--------|------|
| duty | GetTodayDuty | `[]services.TodayDutyMember` |
| duty | GetMonthlyDutySchedule | `map[string][]services.TodayDutyMember` |
| duty | GetHolidayList | `[]models.Holiday` |
| knowledge | GetKnowledgeArticle | `*models.KnowledgeArticle` |
| knowledge | GetKnowledgeCategoryList | `[]models.KnowledgeCategory` |
| knowledge | GetAllTags | `[]models.KnowledgeTag` |
| network | GetDeviceStatistics | `map[string]interface{}` |
| network | GetDevicesByDept | `[]models.NetworkDevice` |
| network | GetDevicesByCredential | `[]models.NetworkDevice` |
| workorder | GetMyPending | custom struct `{List, Total}` |
| workorder | GetStatistics | `*Statistics` |

- **cache_invariants_92_test**: warning 档全清零（DUTY 0 / KNOWLEDGE 0 / NETWORK 0 / WORKORDER 0）
- **Note**: `getExpiration` 方法保留（作为 TTL 解析委托点，尚未收敛到 base）

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | ✓ PASS |
| `TestCacheKeyCollision` | ✓ PASS |
| `TestNoInterfaceGetOrSetResidue` | ✓ PASS（warning 档全 0） |
| `go test ./internal/services/system/...` | ✓ PASS |
| Pre-existing workorder test failure | 已知，与迁移无关 |

## Phase Progress

| Phase | Status |
|-------|--------|
| Phase 96 确定性缓存/看板缺陷修复 | ✓ Completed |
| Phase 97 config_backup 恢复链加固 | ✓ Completed |
| **Phase 98 缓存键安全与 base 迁移收尾** | **✓ Completed** |
| Phase 99 operations 口径统一 | Pending |
| Phase 100 前端契约修复 | Pending |
| Phase 101 收口 | Pending |

**Milestone**: 2/6 phases complete
