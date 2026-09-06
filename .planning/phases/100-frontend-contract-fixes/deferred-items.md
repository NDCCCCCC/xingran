# Phase 100 — Deferred / Out-of-Scope Items

## 2026-09-07（100-02 Task 1 后端 gate 期间发现）

- **[out-of-scope, parallel-agent] `internal/api/v1/network` `TestBackupHandler_Restore/mutual_exclusion_rejected` 失败（期望 400 实际 409）。**
  根因在工作树中 Phase 97 恢复链加固的在途改动（`pkg/response/business_error.go` 新增 + `pkg/response/handler_helpers.go` 修改，均非本 executor 触碰），互斥冲突错误码 400→409 后该测试未同步。与本计划 vdi 改动零关联（本计划仅改 `internal/api/v1/vdi/vm_router.go` / `vm_handler.go` + 新增 `vm_router_test.go`；`go test ./internal/api/v1/vdi/...` 全绿）。归 Phase 97 执行方收口。
