# Phase 97: config_backup 恢复链加固

**Goal:** 修复 config_backup 恢复链三项缺陷（V130R-01/02/03），含 3 个 discuss 设计决策。所有修复附回归测试，七 gate 不倒退。

**Phase Boundary:** Phase 97 范围外：gzip 压缩/解压（Phase 93 已实现）、RestoreConfig 下发内核（Phase 93 已实现）、任务状态机（Phase 93 已实现）。

---

## Phase Goal

**As a** system operator, **I want to** have atomic restore timeout enforcement, multi-instance safe task recovery, and semantic HTTP error codes for restore conflicts, **so that** config restore operations are reliable under concurrent instances and timeout scenarios, and error responses distinguish business conflicts (409/400) from server errors (500).

---

## Requirements Addressed

- **V130R-01** (V130R-01): `config_restore_task_service.go:106` — 超时路径互斥原子化
- **V130R-02** (V130R-02): `config_restore_task_service.go:347-358` — RecoverStaleRunningTasks 实例归属过滤
- **V130R-03** (V130R-03): `backup_handler.go:283-284` — 业务错误码语义化

---

## Multi-Source Coverage Audit

| Source | Item | Covered By |
|--------|------|------------|
| GOAL (ROADMAP phase goal) | 恢复链三项缺陷修复 | V130R-01/02/03 tasks |
| REQ (V130R-01/02/03) | 超时互斥原子化 | Task 1 (V130R-01) |
| REQ (V130R-02) | 多实例归属过滤 grace period | Task 2 (V130R-02) |
| REQ (V130R-03) | BusinessError 类型 + HandleServiceError 识别 | Task 3 (V130R-03) |
| RESEARCH (97-CONTEXT.md) | D-01/D-02/D-03/D-04/D-05 | All tasks |
| CONTEXT (D-01..D-05) | Locked decisions implementation | All tasks |

---

## Must-Haves

### truths
1. V130R-01: 恢复前备份阶段超时（30s）后，下发 goroutine 检测 `ctx.Done()` 立即退出，任务置 failed；RestoreConfig 下发阶段超时（5min）同样立即退出
2. V130R-01: 背对背新恢复请求不再出现双任务窗口（下一请求可立即发起）
3. V130R-02: RecoverStaleRunningTasks 在 10 分钟 grace period 内的 running 任务不被收敛；pending 任务照常收敛
4. V130R-03: `StartRestore` 返回"进行中"错误时 HTTP 状态码为 409；返回"不属于设备"错误时为 400；非 500

### artifacts
- `pkg/response/business_error.go` — `BusinessError` struct with `HTTPStatus`, `Code`, `Message` fields and `Error()` method
- `pkg/response/handler_helpers.go` — `HandleServiceError` recognizes `*BusinessError`, returns its HTTPStatus
- `pkg/constants/timeouts.go` — `RestoreBackupTimeout` (30s) + `RestoreConfigExecTimeout` (5min) replace `RestoreConfigTimeout` (10min)
- `internal/services/config_restore_task_service.go` — `runRestore` uses two-phase contexts; `RecoverStaleRunningTasks` uses grace period filter

### key_links
- From `backup_handler.go:283` → `HandleServiceError` → recognizes `*BusinessError` → returns HTTPStatus 409/400
- From `config_restore_task_service.go:106` (old `RestoreConfigTimeout`) → split into two timeouts applied at lines ~165 (backup ctx) and ~187 (restore ctx)

---

## VALIDATION.md Not Applicable

This is a defect remediation phase (Phase 97) with locked discuss decisions (D-01 through D-05). No open research questions remain after discuss-phase. Each V130R requirement describes a deterministic, testable behavior:

- V130R-01: Timeout worker exit — verifiable via goroutine termination test
- V130R-02: Grace period filter — verifiable via timestamp-based test cases
- V130R-03: HTTP status codes — verifiable via HTTP response assertion

The regression tests for each plan serve as the validation evidence. No separate VALIDATION.md artifact is required.

---

## Threat Model (STRIDE)

| Threat ID | Category | Component | Disposition | Mitigation |
|-----------|----------|-----------|------------|------------|
| T-97-01 | Denial of Service | config_restore_task_service.go | mitigate | Two-phase ctx budgets prevent indefinite resource blocking |
| T-97-02 | Elevation of Privilege | backup_handler.go | mitigate | BusinessError 409/400 semantics prevent error confusion |
| T-97-03 | Spoofing | config_restore_task_service.go:StartRestore | mitigate | DeviceID validation in StartRestore checks backup belongs to target device |
| T-97-04 | Tampering | config_restore_task_service.go:RecoverStaleRunningTasks | mitigate | Grace period filter prevents premature status overwrites across instances |
| T-97-05 | Repudiation | config_restore_task_service.go | accept | Context cancellation is internal; operlog records task lifecycle separately |
| T-97-06 | Information Disclosure | config_restore_task_service.go:StartRestore | mitigate | BusinessError.Message fields are user-facing but contain no secrets; device config content never exposed in error responses |
| T-97-SC | Supply Chain | new pkg/response/business_error.go | accept | No external dependencies — pure Go struct |

---

## Wave Structure

| Wave | Plans | Autonomous | Dependencies |
|------|-------|------------|--------------|
| 1 | 97-01 (V130R-01 timeouts) | yes | none |
| 2 | 97-02 (V130R-02 grace period) | yes | depends on 97-01 |
| 2 | 97-03 (V130R-03 BusinessError) | yes | depends on 97-01 |

**Rationale:** All three plans touch `config_restore_task_service.go` (different functions within the same file). 97-01 adds new constants and refactors `runRestore` (lines ~106-112, ~163-177, ~187). 97-02 modifies `RecoverStaleRunningTasks` (lines ~347-358). 97-03 modifies `StartRestore` error returns (lines ~73-86). 97-02 and 97-03 can run in parallel after 97-01 completes since they touch different functions within the same file.

---

## Plans

### Plan 97-01: V130R-01 超时互斥原子化

**files_modified:** `pkg/constants/timeouts.go`, `internal/services/config_restore_task_service.go`

**depends_on:** []

**wave:** 1

**autonomous:** true

**requirements:** [V130R-01]

```markdown
<objective>
将 RestoreConfigTimeout 拆分为两段 context 预算（备份 30s + 下发 5min），任一段超时 goroutine 检测 ctx.Done() 立即停止，消除背对背双任务窗口。
</objective>

<context>
@pkg/constants/timeouts.go
@internal/services/config_restore_task_service.go:106 (line 106 current RestoreConfigTimeout)
@internal/services/config_restore_task_service.go:163-177 (backup phase CreateBackup)
@internal/services/config_restore_task_service.go:187 (RestoreConfig call)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Split RestoreConfigTimeout into two constants</name>
  <files>pkg/constants/timeouts.go</files>
  <action>
在 timeouts.go 中将 `RestoreConfigTimeout = 10 * time.Minute` 替换为两段预算常量：

```go
// RestoreBackupTimeout is the context timeout for the pre-restore backup phase
// (CreateBackup in runRestore, Phase 97 V130R-01 D-01).
RestoreBackupTimeout = 30 * time.Second

// RestoreConfigExecTimeout is the context timeout for the RestoreConfig下发 phase
// in runRestore (Phase 97 V130R-01 D-01). Full config push can be hundreds of
// lines line-by-line, far exceeding single-command CommandExecTimeout.
RestoreConfigExecTimeout = 5 * time.Minute
```
注释说明与旧 `RestoreConfigTimeout` 的关系，供后续迁移参考。
  </action>
  <verify>grep -c "RestoreBackupTimeout\|RestoreConfigExecTimeout" pkg/constants/timeouts.go</verify>
  <done>timeouts.go contains both new constants; RestoreConfigTimeout removed</done>
</task>

<task type="auto">
  <name>Task 2: Refactor runRestore into two-phase contexts</name>
  <files>internal/services/config_restore_task_service.go</files>
  <action>
修改 `config_restore_task_service.go` 中 `StartRestore` 和 `runRestore`，实现两段独立 context（per D-01）：

**BEFORE (lines ~106-112, goroutine receives runCtx):**
```go
// StartRestore 启动异步恢复任务
func (s *ConfigRestoreTaskService) StartRestore(ctx context.Context, deviceID string, backupID string) error {
    runCtx, runCancel := context.WithTimeout(context.Background(), constants.RestoreConfigTimeout)
    go s.runRestore(runCtx, runCancel, deviceID, backupID)
    return nil
}
```

**AFTER (goroutine internally manages two-phase budgets, StartRestore returns immediately):**
```go
func (s *ConfigRestoreTaskService) StartRestore(ctx context.Context, deviceID string, backupID string) error {
    go s.runRestore(context.Background(), deviceID, backupID) // goroutine owns its own contexts
    return nil
}
```

**runRestore internal two-phase (lines ~126, ~163-177, ~187):**
```go
func (s *ConfigRestoreTaskService) runRestore(baseCtx context.Context, deviceID string, backupID string) {
    // Phase 1: Pre-restore backup (30s budget)
    backupCtx, backupCancel := context.WithTimeout(context.Background(), constants.RestoreBackupTimeout)
    defer backupCancel()

    if backupCtx.Err() != nil {
        s.failTask(...)
        return
    }
    if err := s.backupSvc.CreateBackup(backupCtx, ...); err != nil {
        if backupCtx.Err() != nil {
            s.failTask(...)
            return
        }
        // ... handle other errors
    }

    // Phase 2: Config restore (5min budget)
    restoreCtx, restoreCancel := context.WithTimeout(context.Background(), constants.RestoreConfigExecTimeout)
    defer restoreCancel()

    if restoreCtx.Err() != nil {
        s.failTask(...)
        return
    }
    if err := s.executor.RestoreConfig(restoreCtx, ...); err != nil {
        if restoreCtx.Err() != nil {
            s.failTask(...)
            return
        }
        // ... handle other errors
    }
}
```

关键实现要点（per D-01）：
- 两段之间无共享 budget，任一超时各自独立取消
- goroutine 在 ctx.Done() 检测后真正退出，不依赖 ExecuteCustom 内部超时
- backupCancel() 在阶段②开始前调用以释放阶段①资源
  </action>
  <verify>go build ./internal/services/config_restore_task_service.go 2>&1 | head -20</verify>
  <done>runRestore uses two distinct context phases; neither exceeds 5min for restore or 30s for backup</done>
</task>

<task type="auto">
  <name>Task 3: Add regression tests for V130R-01</name>
  <files>internal/services/config_restore_task_service_97_01_test.go</files>
  <action>
创建 `internal/services/config_restore_task_service_97_01_test.go`，测试两项超时场景：

1. **BackupCtxTimeout**: Mock `backupSvc.CreateBackup` 使其阻塞超过 RestoreBackupTimeout，验证：
   - 任务终态为 failed
   - error_message 包含超时语义
   - RestoreConfig 未被调用

2. **RestoreCtxTimeout**: Mock `executor.RestoreConfig` 使其阻塞超过 RestoreConfigExecTimeout，验证：
   - 任务终态为 failed
   - 下发 goroutine 在 ctx.Done() 检测后真正退出（不继续推送命令）
   - 背对背新请求可立即发起（无 running 态残留锁死）

使用 `time寒冷` 冻结时间或 `gomock` 注入延迟。测试文件命名符合项目惯例 `*_97_01_test.go`。
  </action>
  <verify>go test -v -run "V130R.01" ./internal/services/ -count=1 2>&1 | tail -20</verify>
  <done>Two timeout scenarios each assert task final state and goroutine termination</done>
</task>

</tasks>

<verification>
go build ./... && go test -v -run "V130R.01" ./internal/services/ -count=1
</verification>
```

---

### Plan 97-02: V130R-02 多实例归属过滤

**files_modified:** `internal/services/config_restore_task_service.go`

**depends_on:** [97-01]

**wave:** 2

**autonomous:** true

**requirements:** [V130R-02]

```markdown
<objective>
在 RecoverStaleRunningTasks 中加入 10 分钟 grace period 过滤，使多实例重启时不误杀其他实例在途任务。pending 态照常收敛，running 态等待自愈。
</objective>

<context>
@internal/services/config_restore_task_service.go:347-358 (RecoverStaleRunningTasks current impl)
@internal/services/config_restore_task_service.go:347 (注释记录 A5 三方案)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add grace period filter to RecoverStaleRunningTasks</name>
  <files>internal/services/config_restore_task_service.go</files>
  <action>
修改 `RecoverStaleRunningTasks`（:347-358）的 WHERE 条件，加入 grace period 过滤。

将：
```go
Where("status IN (?)", []models.RestoreTaskStatus{
    models.RestoreTaskStatusPending,
    models.RestoreTaskStatusRunning,
})
```

改为（per D-03 grace_period = 2 x RestoreConfigTimeout = 10 分钟）：
```go
Where("status IN (?) AND updated_at < ?", []models.RestoreTaskStatus{
    models.RestoreTaskStatusPending,
    models.RestoreTaskStatusRunning,
}, time.Now().Add(-10*time.Minute))
```

同时更新注释（A5 三方案已选定方案①，grace period 容忍）：
- pending 照常收敛（无 goroutine 认领，是真孤儿）
- running 等待自愈或人工介入（grace period 内不收敛）
  </action>
  <verify>go build ./internal/services/config_restore_task_service.go 2>&1 | head -10</verify>
  <done>RecoverStaleRunningTasks filters by 10-minute grace period; running tasks within grace period are not converged</done>
</task>

<task type="auto">
  <name>Task 2: Add regression tests for V130R-02</name>
  <files>internal/services/config_restore_task_service_97_02_test.go</files>
  <action>
创建 `internal/services/config_restore_task_service_97_02_test.go`，测试多实例 grace period 过滤：

1. **GracePeriodRunningNotConverged**: 构造一个 updated_at 5 分钟前的 running 任务，调用 RecoverStaleRunningTasks，验证该任务仍为 running（未收敛）

2. **GracePeriodExpiredRunningConverged**: 构造一个 updated_at 15 分钟前的 running 任务，调用 RecoverStaleRunningTasks，验证该任务被置为 failed

3. **PendingAlwaysConverged**: 无论 grace period 内外，pending 任务都收敛（pending 无 goroutine 认领，是真孤儿）

使用 GORM 的 `Update` 直接构造旧时间戳的测试数据，或通过 `mock` 注入。
  </action>
  <verify>go test -v -run "V130R.02" ./internal/services/ -count=1 2>&1 | tail -15</verify>
  <done>Three test cases verify grace period behavior for running and pending tasks</done>
</task>

</tasks>

<verification>
go build ./... && go test -v -run "V130R.02" ./internal/services/ -count=1
</verification>
```

---

### Plan 97-03: V130R-03 业务错误码语义化

**files_modified:** `pkg/response/business_error.go`, `pkg/response/handler_helpers.go`, `internal/services/config_restore_task_service.go`, `internal/api/v1/network/backup_handler.go`

**depends_on:** [97-01]

**wave:** 2

**autonomous:** true

**requirements:** [V130R-03]

```markdown
<objective>
新建 BusinessError 类型，使 HandleServiceError 能识别并进行中(409)/不属于设备(400)语义化返回，不再统一 500。
</objective>

<context>
@pkg/response/handler_helpers.go:21-27 (HandleServiceError current impl)
@pkg/response/response.go:87-98 (Error function and toAppError)
@internal/services/config_restore_task_service.go:73-74 (同设备错误)
@internal/services/config_restore_task_service.go:86 (进行中错误)
@internal/api/v1/network/backup_handler.go:283-284 (Restore handler)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create BusinessError type in pkg/response</name>
  <files>pkg/response/business_error.go</files>
  <action>
新建 `pkg/response/business_error.go`（per D-04）：

```go
package response

// BusinessError carries semantic HTTP status and business error code for
// recoverable business conflicts (e.g., "resource in use", "does not belong").
// HandleServiceError recognizes *BusinessError and returns its HTTPStatus
// instead of the default 500.
type BusinessError struct {
    HTTPStatus int    // 409 Conflict / 400 Bad Request / etc.
    Code       int    // Business error code (e.g., 409001, 400001)
    Message    string // Human-readable message
}

func (e *BusinessError) Error() string { return e.Message }
```

文件头注释说明复用规则：任何 service 层需要语义化错误码均可复用此类型。
  </action>
  <verify>test -f pkg/response/business_error.go && grep -c "BusinessError" pkg/response/business_error.go</verify>
  <done>pkg/response/business_error.go exists with BusinessError struct and Error() method</done>
</task>

<task type="auto">
  <name>Task 2: Update HandleServiceError to recognize BusinessError</name>
  <files>pkg/response/handler_helpers.go</files>
  <action>
修改 `pkg/response/handler_helpers.go` 中 `HandleServiceError`（:21-27），在 `if err != nil {` 分支最前面加入 BusinessError 类型识别：

```go
func HandleServiceError(c *gin.Context, err error, operation string) bool {
    if err != nil {
        // D-04: BusinessError carries semantic HTTP status
        if be, ok := err.(*response.BusinessError); ok {
            Error(c, be.HTTPStatus, operation+"失败: "+be.Message)
            return false
        }
        Error(c, http.StatusInternalServerError, operation+"失败: "+err.Error())
        return false
    }
    return true
}
```

注意：需要 import `github.com/xingran-next/xingran-go-backend/pkg/response`（已存在），但函数内部使用 `response.BusinessError` 需避免自身递归——在 `handler_helpers.go` 所在包内 `response` 是包名，BusinessError 在同包可通过类型断言直接访问。
  </action>
  <verify>go build ./pkg/response/handler_helpers.go 2>&1 | head -5</verify>
  <done>HandleServiceError returns BusinessError.HTTPStatus (409/400) instead of 500</done>
</task>

<task type="auto">
  <name>Task 3: Update StartRestore to return BusinessError</name>
  <files>internal/services/config_restore_task_service.go</files>
  <action>
修改 `config_restore_task_service.go` 中 `StartRestore` 的两处错误返回（per D-04）：

1. `:73-74` 同设备错误（备份不属于目标设备）：
```go
if backup.DeviceID != deviceID {
    return nil, &response.BusinessError{
        HTTPStatus: 400,
        Code:       400001,
        Message:    "备份不属于目标设备，仅限恢复到备份源设备",
    }
}
```

2. `:86` 进行中恢复任务冲突：
```go
return nil, &response.BusinessError{
    HTTPStatus: 409,
    Code:       409001,
    Message:    "该设备存在进行中的恢复任务",
}
```

3. 同样修改 :100 唯一索引冲突的返回（`isDuplicateActiveRestoreErr` 命中时）：
```go
if isDuplicateActiveRestoreErr(err) {
    return nil, &response.BusinessError{
        HTTPStatus: 409,
        Code:       409001,
        Message:    "该设备存在进行中的恢复任务",
    }
}
```

确保 import 了 `response` 包。
  </action>
  <verify>go build ./internal/services/config_restore_task_service.go 2>&1 | head -10</verify>
  <done>StartRestore returns BusinessError with correct HTTPStatus for both conflict cases</done>
</task>

<task type="auto">
  <name>Task 4: Add regression tests for V130R-03</name>
  <files>internal/services/config_restore_task_service_97_03_test.go</files>
  <action>
创建 `internal/services/config_restore_task_service_97_03_test.go`，测试 BusinessError 语义化返回：

1. **DeviceMismatchReturns400**: Mock backupSvc.GetBackupByID 返回不同 deviceID 的备份，调用 StartRestore，验证返回的 error 是 `*response.BusinessError` 且 HTTPStatus=400 且 Code=400001

2. **DuplicateRestoreReturns409**: Mock DB 使 `isDuplicateActiveRestoreErr` 命中，调用 StartRestore，验证返回的 error 是 `*response.BusinessError` 且 HTTPStatus=409 且 Code=409001

同时在 `pkg/response/handler_helpers_test.go` 中添加或扩展测试：
3. **HandleServiceErrorWithBusinessError400**: 传入 `&response.BusinessError{HTTPStatus: 400, Code: 400001, Message: "test"}`，验证 `c.JSON` 的 status code 是 400
4. **HandleServiceErrorWithBusinessError409**: 传入 `&response.BusinessError{HTTPStatus: 409, Code: 409001, Message: "test"}`，验证 `c.JSON` 的 status code 是 409
5. **HandleServiceErrorWithRegularError**: 传入普通 error，验证 status code 是 500（非 400/409）
  </action>
  <verify>go test -v -run "V130R.03" ./internal/services/ ./pkg/response/ -count=1 2>&1 | tail -20</verify>
  <done>BusinessError maps to correct HTTP status (409/400); regular errors remain 500</done>
</task>

</tasks>

<verification>
go build ./... && go test -v -run "V130R.03" ./internal/services/ ./pkg/response/ -count=1
</verification>
```

---

## Success Criteria

1. **V130R-01**: 超时后 worker 不再继续向设备推送命令，背对背新恢复请求不再出现双任务窗口
2. **V130R-02**: RecoverStaleRunningTasks 在多实例/滚动重启下不误杀其他实例在途任务（grace period 10min）
3. **V130R-03**: 「存在进行中恢复任务」 -> 409, 「备份不属于目标设备」 -> 400, 不再统一返回 500
4. **回归纪律**: 每项修复附 2-3 个行为级测试；七 gate（go build / go test / coverage / lint / type-check / diff coverage）全程不倒退

---

## Output

Create `.planning/phases/97-config-backup-restore-chain-hardening/97-01-SUMMARY.md` after completing Plan 97-01
Create `.planning/phases/97-config-backup-restore-chain-hardening/97-02-SUMMARY.md` after completing Plan 97-02
Create `.planning/phases/97-config-backup-restore-chain-hardening/97-03-SUMMARY.md` after completing Plan 97-03
