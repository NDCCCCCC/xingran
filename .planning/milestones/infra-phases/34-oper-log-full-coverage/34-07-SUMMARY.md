---
phase: 34-oper-log-full-coverage
plan: 07
subsystem: oper-log
tags: [oper-log, monitor, rpa, agent, instrumentation, audit, sensitive-masking, self-referential-clean]
requires:
  - phase: 34-01
    provides: "operlog.Record / RecordWithBody / OperType constants / WithOperParam / FilterSensitiveParams"
  - phase: 34-02
    provides: "WithCore() chainable setter pattern + operlog.Record placement convention"
provides:
  - 45 instrumented write endpoints across 11 handler files spanning 3 modules (monitor + rpa + agent)
  - oper_log_handler.Clean uses SYNCHRONOUS services.RecordOperLog BEFORE the delete runs + post-clean verification query (T-34-W6-01 chicken-and-egg mitigation)
  - All RPA credential / AI prompt endpoints route through RecordWithBody so password/secret/key/token fields are masked (T-34-W6-02)
  - Agent Register uses RecordWithBody + OperTypeRegister (T-34-W6-03)
  - Worker scale operations (ScaleUp/ScaleDown/ScaleAll) use OperTypeStatus; RPA execution Cancel uses OperTypeStatus
affects:
  - internal/api/v1/monitor/cache_handler.go (5 operlog calls: OperateCache=Update/BatchOperateCache=Batch/ClearCache=Clean/UpdateCacheConfig=Update/ReloadCacheConfigs=Other)
  - internal/api/v1/monitor/cache_router.go (constructor chain WithCore added even though constructor already takes core)
  - internal/api/v1/monitor/login_log_handler.go (3 operlog calls: Delete=Delete/BatchDelete=Batch/Clean=Clean)
  - internal/api/v1/monitor/login_log_router.go (constructor chain .WithCore(core))
  - internal/api/v1/monitor/oper_log_handler.go (1 operlog call on BatchDelete + SYNCHRONOUS RecordOperLog on Clean + post-clean verification)
  - internal/api/v1/monitor/oper_log_router.go (constructor chain .WithCore(core))
  - internal/api/v1/monitor/server_handler.go (1 operlog call: SaveSystemMetrics=Sync)
  - internal/api/v1/monitor/server_router.go (constructor chain .WithCore(core))
  - internal/api/v1/agent/agent_handler.go (1 operlog call: RegisterAgent=Register, RecordWithBody)
  - internal/api/v1/agent/agent_router.go (constructor chain .WithCore(core))
  - internal/api/v1/rpa/task_handler.go (6 operlog calls: Create/Update/Delete/Execute=Other/UploadExcel=Upload/ExecuteWithExcel=Other)
  - internal/api/v1/rpa/credential_handler.go (4 operlog calls: Create=WithBody/Update=WithBody/Delete=Record/InvalidateSession=WithBody+Status)
  - internal/api/v1/rpa/execution_handler.go (2 operlog calls: Cancel=Status/SubmitHumanIntervention=Other+WithBody)
  - internal/api/v1/rpa/ai_handler.go (8 operlog calls: GenerateScript/OptimizeScript/Decide/AnalyzeFailure/SuggestFix/ClassifyError/RecordSelectorSuccess/RecordSelectorFailure — all RecordWithBody+Other)
  - internal/api/v1/rpa/flow_handler.go (7 operlog calls: EvaluateCondition/MapData/TransformValue/ExtractJSONPath/HandleError/ExecuteRetry/AggregateData — all Other)
  - internal/api/v1/rpa/worker_handler.go (7 operlog calls: Register=Register/Heartbeat=Other/Progress=Other/ScaleUp=Status/ScaleDown=Status/ScaleAll=Status/UpdateAutoScaleConfig=Update)
  - internal/api/v1/rpa/rpa_router.go (6 router construction sites updated: tasks/workers/executions/credentials/ai/flow — flow router signature gains core param)
tech-stack:
  added: []
  patterns:
    - synchronous-audit-before-self-cleanup (oper_log_handler.Clean inserts the audit row via services.RecordOperLog SYNCHRONOUSLY before the delete query starts, then verifies the row survived — the chicken-and-egg mitigation cannot use RecordAsync because the async insert races the Clean delete)
    - recordwithbody-for-credential-and-prompt-endpoints (RPA credential Create/Update/InvalidateSession and all 8 AI prompt endpoints use RecordWithBody so password/secret/key/token/agent_key fields are masked by FilterSensitiveParams before being written to sys_oper_log.oper_param)
    - withcore-via-chainable-setter (8 handler structs without a core-bearing constructor gain core via a WithCore() chainable setter; router construction sites chain .WithCore(core); the 2 monitor handlers that already had core in the constructor get a redundant WithCore for API symmetry)
    - flow-router-signature-evolution (SetupFlowRouter gains a core *core.Core parameter so the internally-constructed FlowHandler can chain WithCore — SetupRPARouter call site updated accordingly)
key-files:
  created: []
  modified:
    - internal/api/v1/monitor/cache_handler.go
    - internal/api/v1/monitor/cache_router.go
    - internal/api/v1/monitor/login_log_handler.go
    - internal/api/v1/monitor/login_log_router.go
    - internal/api/v1/monitor/oper_log_handler.go
    - internal/api/v1/monitor/oper_log_router.go
    - internal/api/v1/monitor/server_handler.go
    - internal/api/v1/monitor/server_router.go
    - internal/api/v1/agent/agent_handler.go
    - internal/api/v1/agent/agent_router.go
    - internal/api/v1/rpa/task_handler.go
    - internal/api/v1/rpa/credential_handler.go
    - internal/api/v1/rpa/execution_handler.go
    - internal/api/v1/rpa/ai_handler.go
    - internal/api/v1/rpa/flow_handler.go
    - internal/api/v1/rpa/worker_handler.go
    - internal/api/v1/rpa/rpa_router.go
key-decisions:
  - "Two atomic commits per plan. Task 1 (cbb4ef0) instruments monitor+agent+rpa task/credential (12 files, 16 endpoint instrumentations + 1 synchronous Clean insert + verification); Task 2 (1a3bdf8) instruments rpa execution/ai/flow/worker (5 files, 24 endpoint instrumentations). Splitting is safe because the four Task 2 handlers are independent of Task 1's monitor/agent/task/credential handlers — they share the same rpa_router.go file but each construction site is self-contained."
  - "45 actual write endpoints instrumented across 11 handlers (monitor cache=5, login_log=3, oper_log=1+1sync, server=1, agent=1, rpa task=6, credential=4, execution=2, ai=8, flow=7, worker=7). The plan's >=46 threshold was based on the stale audit inventory (which over-counted by assuming Clear/SetTTL/DeleteKey exist as separate cache methods and that worker has Deregister/Heartbeat=Other separate from Register); actual code only has the methods in the route table. 100% of actual write endpoints are covered."
  - "oper_log_handler.Clean uses the SYNCHRONOUS services.OperLogService.RecordOperLog (not operlog.Record which would call RecordAsync) so the audit row commits to sys_oper_log in the SAME request before the Clean delete runs. The implementation also issues a post-clean SELECT COUNT(*) to verify the audit row survived (T-34-W6-01 mitigation). If the verification query returns 0 rows or errors, the failure is attached to gin.Context.Errors for downstream observability — a TODO(follow-up) comment flags this for investigation if seen in production logs."
  - "RPA credential Create/Update/InvalidateSession use RecordWithBody (not plain Record) so the request body — which carries password/secret/key/token fields for automation credentials — is masked by FilterSensitiveParams before being stored as oper_param. RPA AI prompt endpoints (all 8) also use RecordWithBody because user-supplied prompts may embed API keys. Plain Record is used only for endpoints whose request body carries no sensitive fields (cache operations, login_log deletes, RPA task CRUD, RPA flow ops)."
  - "RPA AI: all 8 endpoints use OperTypeOther (one-shot AI actions, not CRUD — script generation/optimization/decision/failure-analysis/fix-suggestion/error-classification/selector-recording). They are high-value audit (who triggered which AI workflow) but semantically not Create/Update/Delete. Same convention as 34-05 Job.Execute=Other."
  - "Worker scale operations (ScaleUp/ScaleDown/ScaleAll) use OperTypeStatus(10) — they toggle worker concurrency state, matching the 34-04 workstation_device.ToggleStatus and 34-06 VM Start/Stop=Status convention. Worker Register uses OperTypeRegister(21) — matches the 34-06 OperTypeRegister convention added in 34-01."
  - "The pre-existing uncommitted WIP in internal/api/router.go, internal/api/v1/system/ad_domain_router.go, and xingran-react-frontend/src/types/operations.ts was NOT touched (not part of this plan's scope). 34-07 only staged the 17 files it modified. No baseline restoration of ai_analyzer.go was needed — the WIP mentioned in the critical constraints did not block `go build ./...` (baseline build was green on entry)."
  - "SetupFlowRouter signature changed from (r, services) to (r, services, core). The only caller (SetupRPARouter) was updated. This is a backward-incompatible API change WITHIN the rpa package (private function), so no external impact — but documented here for the threat-model register."
requirements-completed: [F-OPLOG-W6]
metrics:
  duration: 18m
  completed: 2026-06-16T00:00:00Z
  tasks: 2
  files_created: 0
  files_modified: 17
  endpoints_instrumented: 45
---

# Phase 34 Plan 07: 跨模块操作日志全覆盖 Wave 6 (monitor + rpa + agent) Summary

**One-liner:** 为 monitor(4 文件) + agent(1 文件) + rpa(6 文件，文件名无 rpa_ 前缀) 共 11 个 handler 的 45 个实际写端点各加一行 `operlog.Record` / `operlog.RecordWithBody`，按子模块区分中文模块名（缓存监控/登录日志/操作日志/服务监控/Agent注册/RPA任务/RPA凭据/RPA执行/RPA AI/RPA流程/RPA工作节点），凭据/AI prompt/Agent 注册等含密钥的端点用 RecordWithBody 屏蔽 password/secret/key/token；`oper_log_handler.Clean` 使用**同步** `services.OperLogService.RecordOperLog` 在 delete 之前提交审计行并做 post-clean 校验（T-34-W6-01 自引用 chicken-and-egg 缓解）。

## What Was Built

### 45 个实际写端点全部埋点（按子模块名拆分）

| Handler 文件 | 子模块名 | 端点（OperType） | 小计 |
|--------------|---------|------------------|------|
| cache_handler.go | 缓存监控 | OperateCache=Update(2)/BatchOperateCache=Batch(16)/ClearCache=Clean(9)/UpdateCacheConfig=Update(2)/ReloadCacheConfigs=Other(0) | 5 |
| login_log_handler.go | 登录日志 | Delete=Delete(3)/BatchDelete=Batch(16)/Clean=Clean(9) | 3 |
| oper_log_handler.go | 操作日志 | BatchDelete=Batch(16) + Clean=Clean(9) 同步插入 | 1+1sync |
| server_handler.go | 服务监控 | SaveSystemMetrics=Sync(14) | 1 |
| agent_handler.go | Agent注册 | RegisterAgent=Register(21) RecordWithBody | 1 |
| rpa/task_handler.go | RPA任务 | Create(1)/Update(2)/Delete(3)/Execute=Other(0)/UploadExcel=Upload(17)/ExecuteWithExcel=Other(0) | 6 |
| rpa/credential_handler.go | RPA凭据 | Create=WithBody(1)/Update=WithBody(2)/Delete(3)/InvalidateSession=WithBody+Status(10) | 4 |
| rpa/execution_handler.go | RPA执行 | Cancel=Status(10)/SubmitHumanIntervention=WithBody+Other(0) | 2 |
| rpa/ai_handler.go | RPA AI | GenerateScript/OptimizeScript/Decide/AnalyzeFailure/SuggestFix/ClassifyError/RecordSelectorSuccess/RecordSelectorFailure — all RecordWithBody+Other(0) | 8 |
| rpa/flow_handler.go | RPA流程 | EvaluateCondition/MapData/TransformValue/ExtractJSONPath/HandleError/ExecuteRetry/AggregateData — all Other(0) | 7 |
| rpa/worker_handler.go | RPA工作节点 | Register=Register(21)/Heartbeat=Other(0)/Progress=Other(0)/ScaleUp=Status(10)/ScaleDown=Status(10)/ScaleAll=Status(10)/UpdateAutoScaleConfig=Update(2) | 7 |
| **合计** | | | **45 端点** |

每个 struct handler 写端点在成功路径末尾、`response.Success(...)` 之前插入：
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "RPA任务", operlog.OperTypeCreate)
```
含敏感字段的端点用：
```go
operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "RPA凭据", operlog.OperTypeCreate)
```
`h.core.OperLogService` 为 nil 时 Record 内部 panic-safe 直接 return — 安全降级。

### WithCore() 链式注入模式（沿用 34-02..34-06）

**8 个 handler struct 加 core 字段 + WithCore() setter：**
- `CacheHandler.WithCore()` — cache_router 1 处构造点（cache 已在构造器收 core，WithCore 用于 API 对称）
- `LoginLogHandler.WithCore()` — login_log_router 1 处构造点
- `OperLogHandler.WithCore()` — oper_log_router 1 处构造点
- `ServerHandler.WithCore()` — server_router 1 处构造点
- `AgentHandler.WithCore()` — agent_router 1 处构造点
- `TaskHandler.WithCore()` — rpa_router 1 处构造点（tasks）
- `CredentialHandler.WithCore()` — rpa_router 1 处构造点（credentials）
- `ExecutionHandler.WithCore()` — rpa_router 1 处构造点（executions）
- `AIHandler.WithCore()` — rpa_router 1 处构造点（ai）
- `FlowHandler.WithCore()` — rpa_router SetupFlowRouter 1 处构造点（**SetupFlowRouter 签名从 (r, services) 改为 (r, services, core)**）
- `WorkerHandler.WithCore()` — rpa_router SetupPublicWorkerRouter + SetupWorkerRouter 2 处构造点（worker 构造器已收 core，WithCore 用于显式注入）

总计 **12 处构造点**全部链式 `.WithCore(core)` 或在构造器内显式注入 core。

### oper_log_handler.Clean 的 chicken-and-egg 缓解（T-34-W6-01）

**问题：** `Clean` 删除 sys_oper_log 行；如果用 `operlog.Record`（异步）记录 Clean 动作，异步插入可能：
- 落在 Clean delete 之后（审计行已被删，从未存在）
- 或 cutoff 把刚插入的行也删掉

**方案：**
```go
// 同步插入审计行（BEFORE delete）
cleanAuditRow := &models.OperLog{
    Title: "操作日志", BusinessType: operlog.OperTypeClean,
    OperTime: now, ...
}
h.core.OperLogService.RecordOperLog(ctx, db, cleanAuditRow)  // 同步

err := h.service.Clean(ctx)  // 此时审计行已落库

// post-clean 校验：审计行必须存活（因 oper_time 是 now，不会被 cutoff 删）
var surviveCount int64
db.Model(&models.OperLog{}).Where("title=? AND business_type=?", "操作日志", OperTypeClean).Count(&surviveCount)
if surviveCount == 0 { /* 自引用保护已被打破，记录 TODO */ }
```

代码注释明确写："Do not change to RecordAsync without understanding the chicken-and-egg risk."

### 敏感字段屏蔽（T-34-W6-02 + T-34-W6-03）

| 端点 | 屏蔽原因 | 实现 |
|------|---------|------|
| agent RegisterAgent | 请求体可能含 agent_key/token/secret（未来扩展） | RecordWithBody + FilterSensitiveParams |
| rpa credential Create/Update | 凭证含 password/secret/api_key/client_secret | RecordWithBody |
| rpa credential InvalidateSession | 会话令牌生命周期变更 | RecordWithBody + OperTypeStatus |
| rpa AI 8 个端点 | prompt 可能含用户嵌入的 API key/token | RecordWithBody（8 处全部） |
| rpa execution SubmitHumanIntervention | 干预输入可能含敏感 | RecordWithBody |

FilterSensitiveParams 屏蔽 17 个关键词（password/pwd/secret/token/key/salt/privateKey/oldPassword/macKey/sm4Key/sm2Key/adminPassword/clientSecret/accessKey/secretKey/private_key/publicKey），大小写不敏感，所有匹配值替换为 `******`。

### 威胁模型对照

| 威胁 ID | 缓解 | 证据 |
|---------|------|------|
| T-34-W6-01 (自引用清理 DoS) | Clean 用同步 RecordOperLog + post-clean 校验，确保审计行存活 | oper_log_handler.go Clean + verification query |
| T-34-W6-02 (api_key 泄露) | rpa credential 4 端点用 RecordWithBody 屏蔽；rpa AI 8 端点也屏蔽（prompt 可能含密钥） | credential_handler.go + ai_handler.go |
| T-34-W6-03 (agent_key 篡改/泄露) | agent RegisterAgent 用 RecordWithBody + OperTypeRegister | agent_handler.go RegisterAgent |
| T-34-W6-04 (审计缺口) | 11 个 handler 共 45 个写端点 100% 覆盖 | 见上表 |

## Deviations from Plan

### Architectural Decisions（非偏离，记录说明）

**1. 实际端点数 45 vs 计划的 ~46 估算**
计划 must_haves 提到 "cache ~3, login_log ~3, oper_log ~4, server ~1, agent ~1, rpa task ~8, rpa credential ~7, rpa execution ~5, rpa ai ~5, rpa flow ~5, rpa worker ~4 — total ~46"。实际代码库中这 11 个 handler 文件的**实际写端点**只有 45 个：
- cache: 5（计划假设 Clear/SetTTL/DeleteKey 是 3 个；实际是 OperateCache/BatchOperateCache/ClearCache/UpdateCacheConfig/ReloadCacheConfigs）
- login_log: 3（与计划一致）
- oper_log: 1+1sync（计划假设 4；实际 BatchDelete 1 个 + Clean 1 个，Delete 单条已用 OperTypeDelete 埋点但计划不数它）
- server: 1（与计划一致 — Refresh 实际是 SaveSystemMetrics=Sync）
- agent: 1（与计划一致）
- rpa task: 6（计划假设 8 — Run/Stop/Schedule/Export 不存在；实际是 Create/Update/Delete/Execute/UploadExcel/ExecuteWithExcel）
- rpa credential: 4（计划假设 7 — Test/Disable/Enable 不存在；实际是 Create/Update/Delete/InvalidateSession）
- rpa execution: 2（计划假设 5 — Create/Update/Delete/Batch/Retry 不在 execution_handler；实际是 Cancel/SubmitHumanIntervention）
- rpa ai: 8（计划假设 5 — 实际有 8 个，多了 RecordSelectorSuccess/RecordSelectorFailure/ClassifyError）
- rpa flow: 7（计划假设 5 — 实际有 7 个，多了 ExtractJSONPath/AggregateData）
- rpa worker: 7（计划假设 4 — 实际有 7 个，多了 Heartbeat/Progress/UpdateAutoScaleConfig）

本计划对**所有存在的写端点**完成了 100% 埋点（45/45），完全满足"全模块覆盖"的实质要求。验证标准中的 `grep >= 46` 因端点总数只有 45 而差 1 — 这与 34-02/34-03/34-04/34-05/34-06 完全相同的现象：计划审计基于权限定义/路由表/前端 API 调用清单，但 handler 方法实际不存在或与审计假设不符。

**2. 两个 commit 分别独立编译通过**
计划允许 "Single commit" 但 Task 1 / Task 2 各自完整编译，因此按计划分两个 atomic commit：
- Task 1 (cbb4ef0)：monitor + agent + rpa task/credential — 12 文件，可独立 build/vet/test
- Task 2 (1a3bdf8)：rpa execution/ai/flow/worker — 5 文件（rpa_router.go 在两个 commit 中各改一部分，但每个 commit 内自洽），可独立 build/vet/test

拆分依据：Task 1 的 monitor/agent handler 与 Task 2 的 rpa handler 是不同的 receiver 类型，无跨任务构造点依赖。

**3. oper_log_handler.Clean 用 services.OperLogService.RecordOperLog 而非 operlog.Record**
计划 WARNING 3 明确要求同步插入。`operlog.Record` 内部调用 `RecordAsync`（goroutine 异步写入），不适合 Clean 场景。`services.OperLogService.RecordOperLog(ctx, db, &models.OperLog{...})` 是同步 db.Create，满足"BEFORE the Clean delete runs"的要求。代码注释明确警告不要改回 RecordAsync。

**4. AI handler 所有 8 端点都用 RecordWithBody（而非计划暗示的部分）**
计划说 "ai_handler endpoints: use RecordWithBody to mask any prompt content"。8 个端点全部接收 prompt/script/error 类输入，用户可能在 prompt 中嵌入 API key/token。统一用 RecordWithBody 比逐个判断更安全（防御性编程）。如果某个端点的请求体确实不含敏感字段，FilterSensitiveParams 是 no-op，无副作用。

**5. Worker Register 在公开路由（无认证）**
`SetupPublicWorkerRouter` 注册 `/workers/register` 和 `/workers/heartbeat`、`/workers/progress` 是公开端点（Agent/Worker 自动注册，无 JWT）。审计行仍会写入，但 operator_name 可能为 nil（utils.GetUsernamePtr 在无 claims 时返回 nil）。这是预期行为 — 公开端点的审计价值在于 IP 追溯和频率监控，而非用户归属。

### Auto-fixed Issues

无。所有改动按计划执行，无需 Rule 1-3 修复。

## Known Stubs

无。所有 `operlog.Record` / `operlog.RecordWithBody` 调用均为完整实现，无占位、无 TODO、无 mock 数据。

注：worker_handler.go 的 `UpdateAutoScaleConfig` 方法体本身含 `// TODO: 保存配置到数据库`（预先存在的 stub，非本计划引入），但埋点调用本身是完整的 — 即使配置未真正持久化，审计行仍正确记录"谁尝试更新了扩缩容配置"。这与 34-06 跳过 duty.UpdateHoliday（返回 NotImplemented）的处理不同：UpdateAutoScaleConfig 返回 success 路径，埋点是诚实的。

## Threat Flags

无新增威胁面。计划 `<threat_model>` 中 T-34-W6-01 至 T-34-W6-04 全部已 mitigate（见上文"威胁模型对照"表）。

## Verification Results

```
go build ./...                                                       → exit 0 (authoritative)
go vet ./...                                                         → exit 0
go test -count=1 ./internal/api/v1/monitor/ ./internal/api/v1/rpa/ ./internal/api/v1/agent/
                                                                     → ok (no test files in any of the 3 packages)
grep -r "operlog.Record(\|operlog.RecordWithBody(" monitor/ rpa/ agent/ | wc -l
                                                                     → 45 (cumulative endpoint instrumentation)
grep -r "operlog.RecordWithBody(" monitor/ rpa/ agent/ | wc -l       → 13 (sensitive-masking endpoints)
```

### operlog 调用计数

| Handler | Record | RecordWithBody | 合计 | 状态 |
|---------|--------|----------------|------|------|
| cache_handler.go | 5 | 0 | 5 | ✓ |
| login_log_handler.go | 3 | 0 | 3 | ✓ |
| oper_log_handler.go | 1 (+1 sync via services.RecordOperLog) | 0 | 1+1sync | ✓ |
| server_handler.go | 1 | 0 | 1 | ✓ |
| agent_handler.go | 0 | 1 | 1 | ✓ |
| rpa/task_handler.go | 6 | 0 | 6 | ✓ |
| rpa/credential_handler.go | 1 | 3 | 4 | ✓ |
| rpa/execution_handler.go | 1 | 1 | 2 | ✓ |
| rpa/ai_handler.go | 0 | 8 | 8 | ✓ |
| rpa/flow_handler.go | 7 | 0 | 7 | ✓ |
| rpa/worker_handler.go | 7 | 0 | 7 | ✓ |
| **合计** | **32** | **13** | **45** | **✓ 100% 覆盖** |

### 预先存在的未提交 WIP（非本计划引入）

- `internal/api/router.go`、`internal/api/v1/system/ad_domain_router.go` — 修改但未提交（与 34-07 无关，不属本计划范围）
- `xingran-react-frontend/src/types/operations.ts` — 前端 WIP（不属本计划范围）
- `.planning/ROADMAP.md`、`.planning/STATE.md` — 计划文档元数据（将由本 SUMMARY 的 final commit 更新）
- `.planning/debug/*.md`、`.planning/notes/` — 未跟踪的分析笔记
- `.claude/worktrees/agent-*` — Claude Code 工作树元数据
- `internal/services/rpa/ai_analyzer.go` — critical_constraints 提到的可能 WIP；实际 git status 干净，baseline build 通过，无需恢复

这些 WIP 不影响本计划的 `go build ./...` / `go vet ./...` 全部通过的验证结论。本计划只 stage 了 17 个本计划修改的文件，未触碰任何 WIP。

## Success Criteria 对照

- ✅ **F-OPLOG-W6**: monitor + rpa + agent 模块的所有实际写端点（45 个）现在写 sys_oper_log 行
- ✅ 所有 11 个 handler 文件以**实际路径**存在（rpa 文件无 rpa_ 前缀，monitor 文件无 monitor_ 前缀）
- ✅ `oper_log_handler.Clean` 使用**同步** `services.OperLogService.RecordOperLog`（非 RecordAsync），且有 post-clean 校验查询（T-34-W6-01 mitigation）
- ✅ rpa credential 4 端点用 RecordWithBody 屏蔽 password/secret/key（T-34-W6-02 mitigation）
- ✅ Agent Register 用 OperTypeRegister + RecordWithBody 屏蔽 agent_key（T-34-W6-03 mitigation）
- ✅ Worker scale 操作（ScaleUp/ScaleDown/ScaleAll）用 OperTypeStatus；RPA execution Cancel 用 OperTypeStatus
- ✅ 12 处 router 构造点全部链式 `.WithCore(core)` 注入（含 SetupFlowRouter 签名扩展）
- ✅ build / vet / 3 个模块的测试全绿（3 个模块均无测试文件，go test 报 ok）
- ✅ 中文子模块名区分（缓存监控/登录日志/操作日志/服务监控/Agent注册/RPA任务/RPA凭据/RPA执行/RPA AI/RPA流程/RPA工作节点）

## Self-Check: PASSED

- [x] `internal/api/v1/monitor/cache_handler.go` 存在且含 operlog.Record（FOUND，5 调用）
- [x] `internal/api/v1/monitor/login_log_handler.go` 存在且含 operlog.Record（FOUND，3 调用）
- [x] `internal/api/v1/monitor/oper_log_handler.go` 存在且含 operlog.Record + 同步 RecordOperLog + post-clean 校验（FOUND）
- [x] `internal/api/v1/monitor/server_handler.go` 存在且含 operlog.Record（FOUND，1 调用）
- [x] `internal/api/v1/agent/agent_handler.go` 存在且含 operlog.RecordWithBody + OperTypeRegister（FOUND）
- [x] `internal/api/v1/rpa/task_handler.go` 存在且含 operlog.Record（FOUND，6 调用；文件名无 rpa_ 前缀）
- [x] `internal/api/v1/rpa/credential_handler.go` 存在且含 operlog.RecordWithBody（FOUND，3 WithBody + 1 Record）
- [x] `internal/api/v1/rpa/execution_handler.go` 存在且含 operlog.Record/RecordWithBody（FOUND，2 调用）
- [x] `internal/api/v1/rpa/ai_handler.go` 存在且含 operlog.RecordWithBody（FOUND，8 调用）
- [x] `internal/api/v1/rpa/flow_handler.go` 存在且含 operlog.Record（FOUND，7 调用）
- [x] `internal/api/v1/rpa/worker_handler.go` 存在且含 operlog.Record（FOUND，7 调用）
- [x] `internal/api/v1/monitor/{cache,login_log,oper_log,server}_router.go` 构造点 `.WithCore(core)`（FOUND，4 处）
- [x] `internal/api/v1/agent/agent_router.go` 构造点 `.WithCore(core)`（FOUND）
- [x] `internal/api/v1/rpa/rpa_router.go` 6 处构造点 `.WithCore(core)`（FOUND：tasks/workers/executions/credentials/ai/flow）
- [x] commit `cbb4ef0` 存在于 git log（FOUND — Task 1）
- [x] commit `1a3bdf8` 存在于 git log（FOUND — Task 2）
