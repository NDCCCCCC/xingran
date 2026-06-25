---
slug: ad-dept-sync-handler-test-args
status: resolved
trigger: 'Cluster 2/5: ad_dept_sync_handler_test.go - NewADDeptSyncHandler 缺参'
created: 2026-06-12
updated: 2026-06-12
---

# Cluster 2: ad_dept_sync_handler_test.go - NewADDeptSyncHandler 缺参

## Symptoms

`go vet` 报告 3 处错误（line 19, 43, 58），全部同模式：
```
not enough arguments in call to NewADDeptSyncHandler
  have (nil)
  want (*gorm.DB, *addomain.DeptToADSyncService)
```

## 实际代码 (ad_dept_sync_handler_test.go)
```go
handler := NewADDeptSyncHandler(nil)  // line 19, 43, 58 — 都缺第 2 参
```

## 当前真实签名 (ad_dept_sync_handler.go:19)
```go
func NewADDeptSyncHandler(db *gorm.DB, syncService *addomain.DeptToADSyncService) *ADDeptSyncHandler
```

## Initial Hypothesis

测试文件是早期写的，handler 构造函数后来被 refactor 加了第 2 参 `syncService`，但测试文件没同步更新。属于"refactor 后未同步更新测试"经典场景。

## Current Focus

- **hypothesis:** 测试文件需要把 3 处 `NewADDeptSyncHandler(nil)` 改为 `NewADDeptSyncHandler(nil, nil)`。
- **next_action:** 用 Edit 工具做精确替换（`replace_all: true` 因为 3 处模式完全相同）。
- **test:** `go vet ./internal/api/v1/system/` 退出码 0；`go test ./internal/api/v1/system/` 跑一遍相关测试。
- **expecting:** 3 个错误全部消失；测试仍能通过（因为 handler 内部逻辑有 nil 检查 / 测试本身不真正调用 service）。
- **blind_spots:** ① handler 内部是否会 nil-deref 同步 service？需看 handler.go 全文；② 路由 line 44 有误 `"sync\dept-status:configId"` —— 看起来是反斜杠 + 冒号，不是 `/` —— 也可能有问题，但 vet 没说，先不动。
- **tdd_checkpoint:** 测试本身的语义是否正确（用 nil 调用）待验证；如需可加 mock。

## Evidence

<!-- 时序追加新证据 -->

- 2026-06-12T<run>: 读取 `ad_dept_sync_handler_test.go`，确认 3 处 `NewADDeptSyncHandler(nil)` 位于 line 19、43、58，与 brief 一致。
- 2026-06-12T<run>: 读取 `ad_dept_sync_handler.go:19`，确认签名 `NewADDeptSyncHandler(db *gorm.DB, syncService *addomain.DeptToADSyncService)`，3 个 handler 方法均会解引用 `h.db` 或 `h.syncService`。
- 2026-06-12T<run>: 用 `Edit replace_all=true` 将 3 处 `NewADDeptSyncHandler(nil)` 替换为 `NewADDeptSyncHandler(nil, nil)`。
- 2026-06-12T<run>: `go vet ./internal/api/v1/system/` —— 本 Cluster 的 3 个 `NewADDeptSyncHandler` 错误已消失（grep 无匹配，exit=1）；但 `user_handler_test.go:19:3` 仍报 `declared and not used: router`，属于 Cluster 外预先存在的问题，阻塞整个包编译，本次不动。
- 2026-06-12T<run>: `go test ./internal/api/v1/system/ -run "TestSyncDeptToADHandler|TestGetDeptSyncStatus|TestTriggerDeptSync" -v` 因包级 build failure 未跑成（`user_handler_test.go` 阻塞），不是本 Cluster 引入。

## Eliminated

- ❌ 路由 line 44 反斜杠问题：实际文件是 `/sync/dept-status/:configId`（forward slash + colon，标准 Gin param 语法），brief 描述 `/sync\dept-status:configId` 为误报，**false alarm**。
- ❌ `router := gin.New()` 无中间件问题：与同包其他测试风格一致，非问题。

## Resolution

- root_cause: `NewADDeptSyncHandler` 构造函数后来被 refactor 增加第 2 参 `syncService *addomain.DeptToADSyncService`，但 3 处测试调用仍只传 1 个 `nil` 参数。
- fix: 用 `Edit replace_all=true` 将 3 处 `NewADDeptSyncHandler(nil)` 改为 `NewADDeptSyncHandler(nil, nil)`，仅此而已。
- verification: 本 Cluster 的 3 个 `NewADDeptSyncHandler` 错误已消除（grep 无匹配）。`go vet` 包级仍失败，但失败位置在 `user_handler_test.go:19:3`（declared and not used: router），属 Cluster 外问题，不计入本 Cluster。
- files_changed:
  - `internal/api/v1/system/ad_dept_sync_handler_test.go` — 3 行替换（line 19, 43, 58）。

## Side Findings (DO NOT FIX — record only)

1. `go vet` / `go test` 阻塞在 `internal/api/v1/system/user_handler_test.go:19:3: declared and not used: router` —— 属另一 Cluster 或独立问题，不在本次范围。
2. Brief 中提到的 line 44 `"/sync\dept-status:configId"` 反斜杠是 brief 描述错误；实际文件是 `/sync/dept-status/:configId`（标准 Gin param 语法），**false alarm，无需修改**。
3. 测试中 `router := gin.New()` 无中间件：与同包风格一致，不算问题。
4. Nil-deref 风险（仅记录，不修）：
   - `SyncDeptToAD`（line 37）：`h.syncService.SyncDeptStructureToAD(...)` —— 传 `(nil, nil)` 时 service 为 nil，调用即 panic。
   - `GetDeptSyncStatus`（line 53）：`h.db.Where(...).First(...)` —— 传 `(nil, nil)` 时 db 为 nil，GORM chain panic。
   - `TriggerDeptSync`（line 86）：`h.syncService.SyncDeptStructureToAD(...)` 在 goroutine 中 —— 同上 panic。
   - 现状：3 个测试本就在请求-格式层面断言（assert body contains / status code 200|400），**handler 在绑定失败前**会先 panic，但测试是否真的命中 panic 路径取决于请求是否走到 service 调用。当前测试用例的请求（`adConfigId: "config-1"`）能通过 `ShouldBindJSON` 校验并进入 service 调用分支 → **理论存在 panic 风险**。但本 Cluster 范围仅"修编译错误"，不动测试语义；如后续需要稳定通过测试，应改为注入 mock service / mock db。
