# Phase 97: config_backup 恢复链加固 - Context

**Gathered:** 2026-09-06
**Status:** Ready for planning

<domain>
## Phase Boundary

修复 config_backup 恢复链三项缺陷（V130R-01/02/03），含 3 个 discuss 设计决策。所有修复附回归测试，七 gate 不倒退。

Phase 97 范围外：gzip 压缩/解压（Phase 93 已实现）、RestoreConfig 下发内核（Phase 93 已实现）、任务状态机（Phase 93 已实现）。

</domain>

<decisions>
## Implementation Decisions

### V130R-01 — 超时互斥原子化
- **D-01: 分段子 context 两段预算。** `runRestore` 内部分两段 context：
  - 阶段① 恢复前备份：`context.WithTimeout(context.Background(), backupTimeout)`（短 TTL）
  - 阶段② RestoreConfig 下发：`context.WithTimeout(context.Background(), restoreTimeout)`（长 TTL）
  - 任一段超时 cancel，下发 goroutine 检测 `ctx.Done()` 立即停止
  - ctx cancel 后下发 goroutine 真正停止，不依赖 `ExecuteCustom` 内部超时
  - `constants.RestoreConfigTimeout` 拆分为 `RestoreBackupTimeout`（30s）和 `RestoreConfigExecTimeout`（5min）
- **D-02: 背对背双任务窗口消除。** 超时后任务 fail，下发 goroutine 检测 ctx.Done() 立即退出；下一个请求可重新发起，无需等 running 态自然结束。grace period 由 V130R-02 的 grace period 覆盖，不在此处置。

### V130R-02 — 多实例归属过滤
- **D-03: grace period 容忍方案。** `RecoverStaleRunningTasks` 查询条件改为：
  ```sql
  WHERE status IN ('pending', 'running')
    AND updated_at < now() - interval '10 minutes'
  ```
  - `grace_period = 2 × RestoreConfigTimeout`（约 10 分钟）
  - 崩溃后 10 分钟内的 running 任务不自动收敛，等待自愈或人工介入
  - pending 态照常收敛（无 goroutine 认领，是真孤儿）
  - 注释：A5 discretion 方案①，状态机自洽性收口，不违 Phase 93 D-20

### V130R-03 — 业务错误码语义化
- **D-04: 新建 `pkg/response/BusinessError` 类型。**
  ```go
  // pkg/response/business_error.go
  type BusinessError struct {
      HTTPStatus int    // 409 / 400 / ...
      Code      int    // 业务错误码（如 409001）
      Message   string
  }
  func (e *BusinessError) Error() string { return e.Message }
  ```
- `HandleServiceError`（`pkg/response/handler_helpers.go`）识别 `*BusinessError`：
  ```go
  if be, ok := err.(*response.BusinessError); ok {
      Error(c, be.HTTPStatus, be.Message)
      return false
  }
  ```
- `StartRestore` 返回错误改用 `BusinessError`：
  - `"该设备存在进行中的恢复任务"` → `&BusinessError{HTTPStatus: 409, Code: 409001, Message: "..."}`
  - `"备份不属于目标设备"` → `&BusinessError{HTTPStatus: 400, Code: 400001, Message: "..."}`
- 其他模块复用：任何 service 层需要语义化错误码均可复用 `BusinessError`，无需新建 error 类型

### 回归测试纪律（v1.30 D-02 细化）
- **D-05: 行为级测试。** 每项修复断言完整行为链：
  - V130R-01：超时场景（backup ctx 超时 / restore ctx 超时）→ 任务终态 failed + 下发 goroutine 真正停止
  - V130R-02：多实例 grace period 过滤（10 分钟内 running 不收敛 / pending 正常收敛）
  - V130R-03：`BusinessError` → 对应 HTTP status 码（409 / 400），非 500

</decisions>

<canonical_refs>
## Canonical References

### 缺陷登记与来源链
- `.planning/REQUIREMENTS.md` — V130R-01/02/03 需求定义与 v1.30 锁定决策 D-01..D-05
- `.planning/workstreams/milestone/ROADMAP.md` §Phase 97 — Goal / Success Criteria / Notes
- `.planning/milestones/v1.29-DEEP-RECHECK.md` §config_backup 域 — 超时互斥提前释放 / RecoverStaleRunningTasks 多实例误杀 / 业务冲突错误统一 500 三项 Warning

### Phase 93 先例与既有实现
- `.planning/milestones/v1.29-phases/93-config-backup-todo-p2/93-RESEARCH.md` — RestoreConfig 异步模式 / ExecuteCustom / 互斥 D-08 先例 / A5 三种 grace period 方案
- `internal/services/config_restore_task_service.go` — StartRestore(:67-110) / RecoverStaleRunningTasks(:347-358) 现状
- `internal/api/v1/network/backup_handler.go:268-292` — Restore handler，HandleServiceError 统一 500

### 项目约定（CLAUDE.md 相关章节）
- `CLAUDE.md` §Config Backup Restore Convention — gzipCompress/Decompress 唯一入口 / RestoreConfig 唯一入口 / 恢复必须异步
- `CLAUDE.md` §Compilation & Build Verification — 每步改码后 `go build ./...`
- `pkg/response/handler_helpers.go` — HandleServiceError 当前实现（统一 500）

### 代码参照（修复现场）
- `internal/services/config_restore_task_service.go:106` — 现有单一 RestoreConfigTimeout budget
- `internal/services/config_restore_task_service.go:347-358` — RecoverStaleRunningTasks 现状（无 grace period 过滤）
- `pkg/response/handler_helpers.go:21-27` — HandleServiceError 当前实现
- `pkg/response/response.go:87-98` — Error 函数与 toAppError 转换逻辑

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `config_restore_task_service.go:133-163` — StartRestore 互斥检查模式（直接复用 BusinessError 返回路径）
- `config_restore_task_service.go:347-358` — RecoverStaleRunningTasks 注释记录了 A5 三方案，修改点明确
- `pkg/response/response.go:87-98` — toAppError 支持 error 接口扩展，新增 BusinessError 只需在 handler 层识别

### Established Patterns
- Phase 93 D-34 四态状态机（pending → running → success/failed）是既定事实
- Phase 93 D-08 同设备互斥用 pending/running IN 查询，已实现
- Handler-Service 模式 + operlog.Record 在 success path 末尾

### Integration Points
- V130R-03 BusinessError 需要 `config_restore_task_service.go` 的 StartRestore 返回类型改造
- HandleServiceError 识别 BusinessError 需要改 `pkg/response/handler_helpers.go`
- V130R-02 grace period 改 `config_restore_task_service.go:350` 一行 WHERE 条件

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

无

</deferred>

---

*Phase: 97-config-backup-restore-chain-hardening*
*Context gathered: 2026-09-06*
