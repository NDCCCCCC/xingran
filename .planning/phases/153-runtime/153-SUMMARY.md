# Phase 153 — RUNTIME 执行闭环（SUMMARY）

**Milestone:** v1.56 W2 波次
**Branch:** `gsd/phase-153-runtime`（worktree: `D:/CODE/ClaudeCode/wt-153`，未 merge）
**REQ:** RUNTIME-01 ~ RUNTIME-05（5/5 PASS）
**前置确认:** Phase 152 已合入 main（`2a0843b4 merge: Phase 152 CONTRACT P0 契约修复`），另见 Phase 154 已合入（`2a096187`），本 phase 基于 HEAD 构建。

---

## RUNTIME-01 Worker 归属真实回写 — **PASS**

| 改动 | 位置 |
|---|---|
| `WorkerProgressRequest` 加 `WorkerID` 字段（worker 端 ProgressReport 本就携带 `workerId`，后端类型此前缺失） | `internal/services/rpa/types.go` |
| `WorkerHeartbeatRequest` 加 `RunningExecutionIDs []string`（心跳通道补归属）；worker 端 `sendHeartbeat` 填当前运行中执行 | `types.go` / `rpa-worker/internal/types/types.go` / `worker.go` |
| `attributeExecution()`：仅当 `execution.worker_id` 为空时回写（幂等 + 并发安全：首次归属后不再覆盖） | `worker_service.go` |
| `resolveWorker()`：按业务 worker_id 反查 `sys_rpa_workers`，回写 `worker_id`（**列 type:uuid → 存 Worker 主键 ID**）+ `worker_name` | 同上 |
| **WS 推送体读新值**：Progress 更新 DB 后重新 `First` 读执行记录再构建 `ProgressUpdate`（旧实现读写入前快照，worker 归属恒空） | 同上 |

**语义裁定**：`sys_rpa_executions.worker_id` 是 uuid 列（PG 下存业务 worker_id 如 "worker-001" 会报类型错误），故 DB 存 Worker 主键 uuid；WS 消息体与 DB 一致。按业务 worker_id 筛选可 join `sys_rpa_workers`。

**验收证据**（`TestWorkerService_Progress_AttributionWriteBack` / `TestWorkerService_Heartbeat_AttributeRunning`，真实 sqlite 双表）：

```
--- PASS: TestWorkerService_Progress_AttributionWriteBack
  DB: worker_id='worker-uuid-1', worker_name='node-1'（非空）
  WS 推送体: u.WorkerID='worker-uuid-1', u.WorkerName='node-1'（新值，非旧快照）
  幂等: 第二次上报 wk-other 不覆盖已有归属
--- PASS: TestWorkerService_Heartbeat_AttributeRunning
```

## RUNTIME-02 会话注入真实生效 — **PASS**

| 改动 | 位置 |
|---|---|
| `message.Variables["__sessionData"] = message.SessionData`（顶层字段保留双写兼容）；Variables nil 安全初始化 | `task_service.go publishTaskToRedis` |
| `extractTargetSystem` 重写：URL host 规则映射（剥 scheme/port/userinfo → 首段子域名即系统名，`erp.company.com→erp`、大小写不敏感、`www` 跳过）；裸 IP / localhost / 无法解析 → `"default"`；空 URL → `"unknown"` | `task_service.go` |
| `resolveUserDeptID()`：按 userID 查 `sys_user.dept_id` 真实部门（查询失败降级空串，不阻断发布；替代原 `deptID := ""` 恒空） | 同上 |
| `applySessionData` 从 Debug 占位改为**真实注入**（经 `Page.RawPage()` 消费 browser.Manager 冻结契约，未改 browser/*）：① cookies → `rod.Page.SetCookies`（domain 缺省按 targetURL 补、Expires 转 TimeSinceEpoch）；② localStorage → `EvalOnNewDocument`（文档创建时写入，SPA 首载即登录态）+ 已加载时 `Eval` 即时补注；③ token → localStorage 常见键（token/accessToken）+ **patch `window.fetch`** 为同源请求附加 `Authorization: Bearer` | `rpa-worker/internal/executor/engine.go` |
| Execute 主流程：Acquire 后、导航前，`task.Variables[VariableSessionData]` 存在即注入（autologin 动作路径同样生效） | 同上 |

**验收证据**（`TestRuntime02_ApplySessionData_LoggedInState`，**真 Chromium + httptest fixture**，2.67s）：

```
--- PASS: TestRuntime02_ApplySessionData_LoggedInState
  fixture 页面 JS 自证：cookie 读出 session_user=alice、localStorage 读出
  token=test-token-123 → auth-state=logged-in（断言只看页面副作用）
```

## RUNTIME-03 取消可中断 — **PASS**

| 改动 | 位置 |
|---|---|
| 后端 `Cancel` 新签名 `Cancel(ctx, id, operator, reason)`：状态机裁决 + logs 留痕（谁/何时/原因，`FormatLog` 同格式追加） | `execution_service.go` |
| `PublishExecutionCancel` → 频道 `execution:cancel:{id}`，payload 含 operator/reason/timestamp（审计）；publish 尽力而为（Redis 不可用仅告警，DB 终态不回滚） | `redis_publish.go` / `execution_handler.go Cancel` |
| worker 端订阅：`RedisClient.SubscribeExecutionCancel`（`PSubscribe("execution:cancel:*")`，与 scaleCommandListener 同族）+ `Worker.cancelCommandListener` + `activeCancels` 注册表 | `communication/redis_client.go` / `worker/worker.go` |
| `executeTask` 登记 `registerCancel(executionID, cancel)` / 退出注销；心跳携带 runningExecutionIDs | `worker/worker.go` |
| 引擎取消检查点：`executeAction` 入口 + `Execute` 每个动作边界 + 重试循环边界，各至少一次/动作；取消后 `createCancelledResult`（状态恒 cancelled，**绝不以 success 收尾**）；`defer browserManager.Release(page)` 保证 page 释放 | `executor/engine.go` |
| 前端：`columns.tsx` 行内取消按钮（仅 pending/running/paused 显示，danger）+ `index.tsx handleCancelExecution`（Modal.confirm 二次确认 + TextArea 原因 → `POST /cancel {reason}`）；**未触碰 ExecutionDetailModal footer**（归属协议第 4 条） | `executions/columns.tsx` / `executions/index.tsx` |

**验收证据**：

```
--- PASS: TestRuntime03_Execute_CancelReleasesPage（真 Chromium, 2.10s）
  执行 wait(5s)+click 任务，300ms 后 cancel ctx → cancelled 终态返回；
  随后 Acquire 5s 超时内立即可用（page 已释放，无泄漏）
--- PASS: TestRuntime03_ExecuteAction_ContextCancelled（-short 可跑）
--- PASS: TestRuntime03_CreateCancelledResult
--- PASS: TestRuntime03_Cancel_Handler_SQLiteRealTables（httptest + 真实 sqlite）
  pending→cancel 200 + 终态留痕含操作者；success 终态取消被拒（「无法取消/终态不可覆盖」）
--- PASS: TestRuntime03_Cancel_DegradedWhenCacheUnsupported（Redis 不可用降级 200）
--- PASS: TestRuntime03_Cancel_RealRedis_PublishesCommand（条件化，见「遗留」）
```

## RUNTIME-04 人工干预通道打通 — **PASS**

| 改动 | 位置 |
|---|---|
| **publish 落点锁定 `SubmitHumanIntervention`（归属协议第 3 条）**：CreateHumanInterventionEvent + ProcessHumanIntervention 之后经 `h.core.Cache` 发布 `worker:human-input:{workerId}:{executionId}`（复用 worker 端 `subscribeForHumanInput` 订阅端同频道同 payload 形状）；**未改 excel_service.go 一行** | `execution_handler.go` |
| workerId 兜底：query 缺失时按执行归属 uuid 反查 `sys_rpa_workers.worker_id` 业务键（干预频道键） | 同上 `resolveWorkerBusinessID` |
| cache 未注入 / publish 失败 → 明确 500（「人工干预通道未配置」「人工干预下发失败」）——干预未到达 Worker 时前端必须知情，不静默 | 同上 |
| 干预类型：`resume`（输入写任务变量，控制字段 action/reason/executionId/operator 过滤不入业务变量）/ `skip`（跳过等待继续）/ `abort`（`ErrExecutionAborted` 哨兵 → Execute 捕获 → cancelled 终态） | `engine.go executePause` |
| 事件 ID 修复：`HumanInterventionEvent.BeforeCreate` 补应用层 UUID（此前恒空串，PG 靠建表默认兜底，sqlite/GORM 显式插入第二次必 UNIQUE 冲突）。**独立新文件**，不触碰 excel_service.go（158 重写区） | `services/rpa/human_intervention_id.go`（新） |
| sqlite AutoMigrate 补注册 `sys_rpa_human_interventions`（此前从未注册 → 全新 sqlite 库干预落库 `no such table`，与 152 补凭证表同因同解，零漂移原则） | `internal/core/db/database.go` |

**验收证据**：

```
--- PASS: TestRuntime04_SubmitIntervention_PublishTarget（httptest + 真实 sqlite）
  cache 未注入 → 500 +「人工干预通道未配置」；事件已落 DB（审计留痕）
  cache 无 Redis 后端 → 500 +「人工干预下发失败」（不静默）
--- PASS: TestRuntime04_AbortSentinel（abort 哨兵 errors.Is 判定）
--- PASS: TestRuntime02_ApplySessionData（pause 动作依赖的 executePause 注入链路）
--- PASS: TestRuntime04_HumanInput_RealRedis_PublishesAction（条件化，见「遗留」）
```

> 全链路「pause → 前端提交确认 → worker 恢复」的真 Redis 端到端用例已写成条件化集成测试（`runtime_p153_redis_test.go`，`REDIS_ADDR` 可达时自动真跑，校验频道与 payload 形状）。本机无 Redis 且无 docker（6379 ECONNREFUSED）→ 本轮 skip。**零新增依赖**约束下不引入 miniredis；publish/subscribe 两侧的频道格式、payload 形状由两侧用例分别锁定（worker 侧 `subscribeForHumanInput` 解析 map、后侧 publish map），格式漂移会被任一侧用例捕获。

## RUNTIME-05 执行状态机显式化 — **PASS**

**唯一真相源**：`internal/services/rpa/execution_state_machine.go`（新文件）。

### 合法迁移表

```
pending  → running | cancelled                （未领取即取消）
running  → success | failed | cancelled | timeout | paused
paused   → running | cancelled                （人工恢复 / 暂停中被取消）
终态 success/failed/cancelled/timeout 无出边；from==to 为幂等重放（接受，不写入）
```

| 改动 | 位置 |
|---|---|
| `CanTransition` / `CheckTransition`（区分幂等接受与非法拒绝）/ `ExecutionStatusTerminal` | `execution_state_machine.go`（新） |
| 模型补 `RPAExecutionStatusPaused = "paused"`（列 size:20 无需迁移） | `models/rpa/execution.go` |
| `worker_service.Progress` 状态写入走状态机：非法迁移返回明确 error（**不静默覆盖终态**）；终态幂等重放只更新进度字段 | `worker_service.go` |
| `execution_service.Cancel` 判断收敛到状态机（旧三字符串手写比较删除） | `execution_service.go` |

**验收证据**（全迁移矩阵：7×7=49 状态对全覆盖，合法 9 + 非法 32 + 自环 7）：

```
--- PASS: TestStateMachine_LegalTransitions（合法 9 条逐一验证 + 计数一致）
--- PASS: TestStateMachine_IllegalTransitions（其余状态对全部拒绝/幂等，零漏网）
--- PASS: TestStateMachine_TerminalNotOverwritten（4 终态 × 6 目标全拒，cancelled 不被 success 覆盖）
--- PASS: TestStateMachine_IdempotentReplayCancelled（幂等重放）
--- PASS: TestWorkerService_Progress_TerminalNotOverwritten（服务层集成：cancelled→success 报「非法状态迁移」）
```

---

## 构建与测试（全部实跑）

```
$ go build ./...                                   # PASS（worktree 内）
$ go test -count=1 ./internal/api/v1/rpa/ ./internal/services/rpa/
ok  internal/api/v1/rpa      11.742s
ok  internal/services/rpa    0.735s
$ cd rpa-worker && go build ./... && go test ./... -count=1
ok  internal/browser        38.998s   （含真 Chromium 用例）
ok  internal/communication   2.387s
ok  internal/config          2.231s
ok  internal/executor       17.730s   （含真 Chromium 取消/会话注入验收）
ok  internal/worker          2.031s
$ cd xingran-react-frontend && npm run type-check   # PASS（tsc --noEmit 零错误）
```

## 归属协议遵守情况（越界检查）

| 协议条目 | 结果 |
|---|---|
| 1. `task_service.go` 只碰 `:243+` | ✅ 仅改 `publishTaskToRedis` 主体（:304+）、`extractTargetSystem`（:398+）、新增 `resolveUserDeptID`；`:81-200`（Create/Update/List）零改动（文件头 import 区加 `net`/`strings`，属函数区外） |
| 2. `execution_handler.go` Cancel + 干预 publish | ✅ 均落此处（协议第 2/3 条指定落点） |
| 3. `excel_service.go` 不许碰 | ✅ 零改动；干预事件 ID 钩子放独立新文件 `human_intervention_id.go` |
| 4. `ExecutionDetailModal.tsx` footer 不许碰 | ✅ 零改动；取消入口落 `columns.tsx` 行内 actions + `index.tsx` handler |
| 5. `rpa-worker/internal/browser/*` 只消费不改 | ✅ 零改动；会话注入经 `Page.RawPage()` 冻结出口 |

**额外触碰的共享文件（均不在协议禁列，且为必要最小增量）**：
- `internal/core/db/database.go`：sqlite AutoMigrate 补注册干预事件表（152 同区先例）
- `internal/models/rpa/execution.go`：补 `Paused` 状态常量（RUNTIME-05 必需）
- `pkg/cache/client_export.go`（新，纯增量 `ExportClient()`：跨包断言未导出方法恒失败，取消/干预通道需要原生 Pub/Sub 出口）
- 既有测试文件的签名适配（`Cancel` 新参数）与按 RUNTIME-02/05 新语义的断言修正

## 遗留项

1. **真 Redis 端到端未实跑**（本机 6379 不可达、无 docker；零新增依赖禁 miniredis）——条件化用例已就位，`REDIS_ADDR` 可达的环境 `go test ./internal/api/v1/rpa/ -run RealRedis` 即真跑。publish/subscribe 双侧格式分别锁定。
2. **前端取消入口的 e2e 未写**（RUNTIME-03 前端验收以行内入口 + API 调用 + 后端全链路为证据；vitest 组件测试建议随 W5 收尾统一补）。
3. **Execution 的 worker_id 存 uuid 的筛选语义**：前端「按 Worker 筛选」传业务 worker_id 与 uuid 列不一致（存量问题，两处口径历史上从未对齐过——本 phase 至少保证了 DB 值稳定真实）；建议 158 收尾统一（join 或列语义迁移）。
4. `Engine.currentExecutionID/currentStep` 等共享字段在 N 并发下互相覆盖（**存量问题**，154 遗留），本 phase 的 pause/screenshot 上报沿用该字段未扩大问题面；建议单独立项改造为 per-execution 上下文。
