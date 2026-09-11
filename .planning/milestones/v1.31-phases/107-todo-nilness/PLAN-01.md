---
phase: 107
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/api/v1/rpa/worker_handler.go
  - internal/api/v1/rpa/credential_handler.go
  - internal/services/rpa/error_handling.go
  - internal/services/rpa/selector_learner.go
  - internal/services/rpa/task_service.go
  - internal/services/system/config_service.go
  - internal/core/db/init_data.go
  - internal/core/core.go
  - internal/services/device_discovery_service.go
  - internal/agent/server/handlers.go
  - internal/services/rpa/data_mapper.go
autonomous: true
requirements_addressed: [TODO-02, TODO-03, TODO-04, NIL-01]
---

<objective>
Batch delete backend TODO comments and fix NIL-01 nilness root cause. Pure comment removal — no functional code changes except NIL-01 fix.
</objective>

<tasks>

## TASK-1: Delete TODO comments in RPA worker_handler.go

**File:** `internal/api/v1/rpa/worker_handler.go`

- **Line 260**: Delete the line `// TODO: 从数据库或配置获取自动扩缩容配置`
  - Keep the hardcoded `AutoScaleConfig{...}` struct return — valid stub
- **Line 279**: Delete the line `// TODO: 保存配置到数据库`
  - Keep `operlog.Record(...)` and `successMsg(...)` calls — functional code remains

Both are comment-only removals. No code logic changes.

</tasks>

<tasks>

## TASK-2: Delete TODO comment in RPA credential_handler.go

**File:** `internal/api/v1/rpa/credential_handler.go`

- **Line 154**: Delete the line `// TODO: 实现 ListSessions 方法`
  - Keep `success(c, gin.H{"list": []interface{}{}, "total": 0})` — valid empty-list stub

</tasks>

<tasks>

## TASK-3: Delete TODO comment block in RPA error_handling.go

**File:** `internal/services/rpa/error_handling.go`

- **Lines 311-314**: Delete the entire comment block:
  ```
  // TODO: 实现实际的回滚逻辑
  // 1. 获取已完成的步骤
  // 2. 按相反顺序执行补偿动作
  // 3. 更新执行状态
  ```
- Keep `return nil` at line 316 — functional no-op, acceptable rollback stub

</tasks>

<tasks>

## TASK-4: Verify and delete TODO in RPA selector_learner.go

**File:** `internal/services/rpa/selector_learner.go`

- **Line 367**: The line reads `// 计算最近使用得分（30天内）` — this is a normal algorithm comment, NOT a placeholder TODO. No action needed.
- **Line 235**: The code is `rec := &SelectorRecommendation{...}` with no TODO comment. Confirmed by reading the file — no actual TODO exists at lines 235 or 367.
- **Researcher finding stands**: Both comments are normal business logic descriptions, not placeholder TODOs. No changes to this file.

</tasks>

<tasks>

## TASK-5: Delete TODO in RPA task_service.go

**File:** `internal/services/rpa/task_service.go`

- **Line 296**: Delete `// TODO: 从上下文获取部门ID`
  - Keep `deptID := ""` — valid fallback, acceptable simplification

</tasks>

<tasks>

## TASK-6: Delete entire RefreshCache method in config_service.go

**File:** `internal/services/system/config_service.go`

- **Lines 256-259**: Delete the entire `RefreshCache` method:
  ```go
  func (s *configService) RefreshCache(ctx context.Context) error {
      // TODO: 实现缓存刷新逻辑
      return nil
  }
  ```
- Method is never called anywhere in the codebase — confirmed dead code by RESEARCH
- Delete the blank line before `// toStringPtrStr` at line 261 as well

</tasks>

<tasks>

## TASK-7: Delete commented createCaptchaBackgroundMenus in init_data.go

**File:** `internal/core/db/init_data.go`

- **Lines 637-648+**: Delete the entire commented-out function block:
  ```go
  // createCaptchaBackgroundMenus 创建验证码背景图管理菜单
  // TODO: 此函数未使用，如需要启用验证码背景图功能，请取消注释并调用此函数
  /*
  func createCaptchaBackgroundMenus(db *gorm.DB) error {
      ...
  }
  */
  ```
- Read file to confirm full extent of commented block before deleting
- Never called — dead code

</tasks>

<tasks>

## TASK-8: Delete TODO comment in core.go

**File:** `internal/core/core.go`

- **Line 912**: Delete `// TODO: 当 core.Core 接入 Redis 后启用；当前传 nil 不影响主流程`
- Keep the `if err := accountPool.StartHotReload(context.Background()); err != nil { ... }` block — functional code handles the "not wired" case correctly already

</tasks>

<tasks>

## TASK-9: Delete TODO in device_discovery_service.go

**File:** `internal/services/device_discovery_service.go`

- **Lines 662-663**: Delete:
  ```
  // TODO: 实际实现中需要从临时表或缓存中获取发现的设备
  // 目前返回空列表
  ```
- Keep `return []*DiscoveredDevice{}, nil` — valid stub for discovery scan returning no results

</tasks>

<tasks>

## TASK-10: Delete TODO and unused assignment in agent handlers.go

**File:** `internal/agent/server/handlers.go`

- **Line 304**: Delete `// TODO: 实现动态更新 agent_id 和 vm_id 的逻辑`
- **Line 306**: Delete `_ = req // 占位: 当前实现始终使用 authenticator 配置;SA9003 抑制`
- Keep the `if req.AgentID != "" && req.VMID != "" {` condition and its closing brace — the condition body becomes empty but the structure remains (or remove the entire if block if body is empty)

</tasks>

<tasks>

## TASK-11: Fix NIL-01 in data_mapper.go

**File:** `internal/services/rpa/data_mapper.go`

- **Line 332**: Change:
  ```go
  if value == nil || value == "" {
  ```
  To:
  ```go
  if value == "" {
  ```
- **Rationale**: Line 239 (`if value == nil { return nil, nil }`) already returns early for nil. The `value == nil` part at line 332 is impossible. Only `value == ""` is meaningful.
- This eliminates the impossible nilness-check flagged by static analysis.

</tasks>

<success_criteria>
- All listed TODO comment lines removed (or entire dead methods deleted)
- NIL-01 condition fixed: `if value == ""` only
- `go build ./...` passes (backend compiles cleanly)
- `go test ./...` passes (no regressions)
- No functional code logic changed (stubs kept, only comments/methods removed)
</success_criteria>

<output>
Create SUMMARY.md on completion of all tasks
</output>
