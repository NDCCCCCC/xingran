# Phase 99 Deferred Items

## Out-of-scope discoveries (99-04 full-suite gate, 2026-09-07)

`go test ./...` after 99-04 shows exactly 3 failing tests, all **proven pre-existing**
(reproduced in a clean worktree at c6c170a + the parallel session's uncommitted WIP,
i.e. without any 99-04 commit). All 99-04-touched packages pass. Not fixed per scope
boundary — these belong to other phases' in-flight work:

1. `internal/api/v1/network` TestBackupHandler_Restore/mutual_exclusion_rejected —
   expects 400, service returns 409. Cause: uncommitted
   `internal/services/config_restore_task_service.go` (V130R-03 D-04 conflict→409
   semantics, WIP); handler test not yet updated by that workstream.
2. `internal/services/knowledge` TestKnowledgeService_GetKnowledgeArticle_CacheMiss_Success —
   cached value nil after miss. Related to parallel cache-key escape WIP
   (`cache_keys.go` EscapeCacheKeyValue + committed `user_cache_impl.go` callers;
   clean c6c170a does not even compile without the uncommitted cache_keys.go).
3. `internal/services/workorder` TestWorkOrderCacheService_GetStatistics_Empty —
   same cache-get-nil pattern as (2).

Evidence commands: reconstructed pre-state via `git worktree add --detach` at c6c170a
+ `git diff HEAD -- <9 WIP files> | git apply` + copy untracked
`pkg/response/business_error.go`; all 3 tests FAIL there identically.
