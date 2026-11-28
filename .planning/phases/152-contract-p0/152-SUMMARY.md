# Phase 152 — CONTRACT P0 契约修复（SUMMARY）

**Milestone:** v1.56 W1 波次  
**Branch:** `gsd/phase-152-contract`（worktree: `D:/CODE/ClaudeCode/wt-152`，未 merge 到 main）  
**REQ:** CONTRACT-01 ~ CONTRACT-06（6/6 PASS）  
**基线:** `.planning/REQUIREMENTS.md` 立项基线 D1/D2/D3/D4/D5/D15/D16

---

## 0. 优先级取值域裁定（CONTRACT-03 要求二选一）

**裁定：后端放宽到 0-100，前端不动，不做 DB 迁移。**

| 项 | 内容 |
|---|---|
| 选择 | 后端 `TaskPriority` 语义放宽为 0-100（`services/rpa/types.go:87 TaskPriorityMax = 100`） |
| 理由 1 | 前端 `EditModal.tsx:299` 已是 `InputNumber min=0 max=100 initialValue=50`（0-100 滑块语义），改前端要重做交互与历史数据映射 |
| 理由 2 | DB 列是 `SMALLINT`（`102_add_rpa_tables.sql:15 priority SMALLINT`），0-100 完全在 SMALLINT 域内，**无需迁移** |
| 理由 3 | 任务优先级是「排序权重」而非四档分类；0-3 枚举无法表达 50=中等这种直觉语义 |
| 兼容 | `TaskPriorityLow/Medium/High/Urgent`（0/1/2/3）常量保留，仍是 0-100 域内的合法子集，旧数据零影响 |
| 校验 | Create/Update 均校验 `0 <= priority <= 100`，越界 400 且消息含 `0-100` |

---

## CONTRACT-01 凭证链路端到端可用 — **PASS**

### 改动文件

| 文件 | 行 | 改动 |
|---|---|---|
| `internal/api/v1/rpa/rpa_router.go` | 29（`SetupPublicWorkerRouter`）<br>61（`SetupRPARouter`） | `NewServiceGroup(..., nil, ...)` → `NewServiceGroup(..., core.SM4Cipher, ...)`（第 5 参 passwordCipher）。两处均加 D1 说明注释 |
| `internal/services/rpa/credential_service.go` | 82 | 新增 `ErrCipherNotConfigured = errors.New("凭证加密器未配置")` |
| 同上 | 85 | 新增 `requireCipher()` nil 守卫（同时覆盖接口未赋值与 typed-nil） |
| 同上 | 421 `encryptCredentialData` / 453 `encryptSessionData` / 499 `decryptString` / 513 `decryptExtraData` / 531 `decryptSessionData` | 5 个方法全部加 nil 守卫；`decryptString` / `decryptExtraData` / `decryptSessionData` 签名改为返回 `error`（原来是静默回退/void） |
| 同上 | 150-160 / 188 / 233 / 267-283 / 305 | 调用点改为错误传播（不再有 nil 解引用路径） |
| `internal/core/db/database.go` | 917-918 | **sqlite 分支补注册 `RPACredential` / `RPASession`**（此前从未注册 → 全新 sqlite 库凭证 CRUD 直接 `no such table`） |
| `internal/api/v1/rpa/handler_helpers.go` | 117 / 125 | 新增 `currentUserID` / `currentDeptID`：优先读 `user_id`（JWT 中间件实际写入的键），回落 `userId`。历史读 `userId` 恒空 → 凭证归属失真 |

### 验收证据

```
$ go test -count=1 ./internal/api/v1/rpa/ -run "TestContractP0_Credential" -v
=== RUN   TestContractP0_CredentialCRUD_SQLiteRealTables
--- PASS: TestContractP0_CredentialCRUD_SQLiteRealTables (0.83s)
=== RUN   TestContractP0_CredentialNilCipher_ReadableError
--- PASS: TestContractP0_CredentialNilCipher_ReadableError (0.76s)
```

正向（`contract_p0_smoke_test.go:227`）：httptest + `db.NewDatabase(sqlite)` + **真实 `AutoMigrate()`** + 真实 `crypto.NewSM4Cipher`，走 `SetupRPARouter` 装配的真实 `ServiceGroup`：

| 请求 | 断言 |
|---|---|
| `POST /rpa/credentials` | 200；DB 里 `username_encrypted`/`password_encrypted`/`extra_data_encrypted` **均非明文** |
| `POST /rpa/credentials/{id}` | 200；返回解密后的 `username=alice` |
| `POST /rpa/credentials/list` | 200；total=1 |
| `POST /rpa/credentials/{id}/update` | 200；DB 中密码重新加密 |
| 会话创建（`CredentialService.CreateSession`）+ `POST /rpa/credentials/sessions/{id}/invalidate` | 会话 token 加密落库；invalidate 200 |
| `POST /rpa/credentials/{id}/delete` | 200；删除后详情非 200 且 message=「凭证不存在」 |

负向（`contract_p0_smoke_test.go:313`）：**真实 service** 传 `nil` cipher（非 mock）：

| 请求 | 断言 |
|---|---|
| `POST /rpa/credentials` | 非 200，message 含 **「凭证加密器未配置」**（无 panic） |
| `POST /rpa/credentials/list`（预置一行密文，强制走解密循环） | 非 200，message 含 **「凭证加密器未配置」** |

> 说明：`response.Error(c, http.StatusNotFound, ...)` 在 `pkg/response/response.go:157-162` 的 `int` 分支恒映射 HTTP 400，属全局既有行为，不在本 Phase 修复（改动影响全模块），故详情用例断言「非 200 + message 明确」。

---

## CONTRACT-02 三张列表状态筛选前后端同构 — **PASS**

### 改动文件

| 文件 | 行 | 改动 |
|---|---|---|
| `internal/services/rpa/types.go` | 61 | `TaskListParams.Status` `*int` → **`*rpamodels.TaskStatus`**（0=启用/1=停用，与模型枚举同构） |
| 同上 | 170 | `WorkerListParams.Status` `*int` → **`*string`**（模型 `Worker.Status` 是 `WorkerStatus string`） |
| 同上 | 193 | `ExecutionListParams.Status` `*int` → **`*string`**（模型 `Execution.Status` 是 `string`） |
| 同上 | 70 / 174 / 197 | 三个 `Validate()`：非法枚举返回可读中文（列出合法取值） |
| `internal/api/v1/rpa/task_handler.go` | 54 | List 接入 `params.Validate()` |
| `internal/api/v1/rpa/worker_handler.go` | 80 / 83 | List 接入 `Validate()` + `params.Status = normalizeOptionalStringFilter(...)` |
| `internal/api/v1/rpa/execution_handler.go` | 56 / 59 | 同上 |
| `internal/api/v1/rpa/handler_helpers.go` | 36 | `describeBindError`：binding 类型错误翻译成「字段 X 类型不匹配：期望 …，收到 …」 |
| 同上 | 54 | `validateListParams`：枚举域校验统一 400 出口 |
| 同上 | 104 | `normalizeOptionalStringFilter`：`status:""` 归一化为 nil（前端 allowClear 清空语义）。**未改** `worker_service.go:331` / `execution_service.go:135` |
| `xingran-react-frontend/.../tasks/index.tsx` | 34-37 | `TASK_SEARCH_STATUS_OPTIONS` 由执行态字符串（pending/running/completed/failed/cancelled）改为**任务态数字 0/1（启用/停用）** |
| `xingran-react-frontend/.../constants.tsx` | 37-52 | `TASK_STATUS_OPTIONS` 0草稿/1启用/2禁用 → **0启用/1停用**，文本/颜色映射同步 |
| 同上 | 63-83 | `EXECUTION_STATUS_OPTIONS` / TEXT_MAP / COLOR_MAP **补 `timeout`（已超时）** |

### 验收证据

```
$ go test -count=1 ./internal/api/v1/rpa/ -run "TestContractP0_(Task|Worker|Execution)ListStatusBinding" -v
--- PASS: TestContractP0_TaskListStatusBinding (0.99s)
--- PASS: TestContractP0_WorkerListStatusBinding (1.61s)
--- PASS: TestContractP0_ExecutionListStatusBinding (1.39s)
```

真实 sqlite 表 + 真实 List 服务（无 mock service）：

| 请求体 | 期望 | 实测 |
|---|---|---|
| `tasks/list {status:0}` | 200，只命中启用 | PASS（total=1） |
| `tasks/list {status:1}` | 200，只命中停用 | PASS（total=1） |
| `tasks/list {status:null}` | 200，不过滤 | PASS（total=2） |
| `tasks/list {status:"pending"}`（旧前端误传） | **400 可读** | PASS，message=`请求参数错误: 字段 status 类型不匹配：期望 rpa.TaskStatus，收到 string（…）` |
| `tasks/list {status:9}` | **400 可读** | PASS，message 含 `0（启用）` |
| `workers/list {status:"online"}` | 200，只命中在线 | PASS（total=1） |
| `workers/list {status:""}` | 200，不过滤 | PASS（total=2） |
| `workers/list {status:null}` | 200，不过滤 | PASS（total=2） |
| `workers/list {status:"yelling"}` | **400 可读** | PASS，message 含 `online` |
| `executions/list {status:"pending"}` | 200，命中 1 | PASS |
| `executions/list {status:"timeout"}` | 200，命中 1 | PASS（本次补进前端枚举的取值） |
| `executions/list {status:""}` / `{status:null}` | 200，不过滤 | PASS（total=2） |
| `executions/list {status:"completed"}` | **400 可读** | PASS，message 含 `pending` |

前端（`__tests__/constants.test.tsx`，12 tests 全绿）：

```
TASK_STATUS_OPTIONS.map(o => o.value) === [0, 1]        ✓ 且 typeof === "number"
getTaskStatusText(0) === "启用" / (1) === "停用"          ✓
EXECUTION_STATUS_OPTIONS 覆盖 pending/running/success/failed/cancelled/timeout  ✓
EXECUTION_STATUS_OPTIONS 不含 "completed"                ✓
```

---

## CONTRACT-03 任务搜索与字段无损 — **PASS**

### 改动文件

| 文件 | 行 | 改动 |
|---|---|---|
| `internal/services/rpa/task_service.go` | 192 | `task_name LIKE` → **`name LIKE`**（真实列；`Task.TaskName` 的 gorm tag 是 `column:name`）。D3 |
| 同上 | 97 | `Create` 补 `Tags: req.Tags`（原 req 有字段但从未赋值 → 标签死字段）。D5 |
| 同上 | 104-115 | `Create` 后按请求值补偿 priority（GORM 对 `default:1` 的零值会替换成默认值，导致显式 `priority:0` 被写成 1） |
| 同上 | 144 | `if req.MaxRetry >= 0`（恒真，未传会把 retry_count 清零）→ `if req.MaxRetry > 0` |
| 同上 | 148 / 151 | `if req.Status > 0` / `if req.Priority > 0` → **指针语义** `if req.Status != nil` / `if req.Priority != nil`（区分「未传」与「传 0」）。D5 |
| 同上 | 157 | `Update` 补 `updates["tags"]`（空串视为未传，不误清空） |
| `internal/services/rpa/types.go` | 110-150 | `UpdateTaskRequest.Status/Priority` `int` → **`*int`**；`ID` 去掉 `binding:"required"`（handler 总是用路径参数覆盖，旧 tag 让前端不传 id 的正常更新 400）；新增 `CreateTaskRequest.Validate()` / `UpdateTaskRequest.Validate()` |
| `xingran-react-frontend/.../tasks/modals/EditModal.tsx` | 132 | 新建默认 `status: 0`（原 `"pending"` 执行态字符串 → 后端 400） |
| 同上 | 149 | 编辑回填 `status` 用数字原样回填（含 0），不再 `=== 0 ? "pending" : "disabled"` |
| 同上 | 299 | 优先级 label 改「优先级(0-100)」，域 0-100（与后端 `TaskPriorityMax` 对齐） |
| 同上 | 313-316 | 新增「任务状态」表单项（`TASK_STATUS_OPTIONS`，0=启用/1=停用） |

### 验收证据（建→查→改→查 逐字段 round-trip）

```
$ go test -count=1 ./internal/api/v1/rpa/ -run "TestContractP0_TaskRoundTrip" -v
--- PASS: TestContractP0_TaskRoundTrip_TagsAndZeroStatus (1.35s)
```

`contract_p0_smoke_test.go:503`，真实 sqlite 表：

| 步骤 | 请求/断言 | 结果 |
|---|---|---|
| 建 | `POST /tasks {name, tags:"alpha,beta", priority:0, status:0, script:[…]}` | 200，返回 id |
| 查 1 | `POST /tasks/{id}` | `tags="alpha,beta"` ✓（旧实现恒空）、`priority=0` ✓ |
| 搜 | `POST /tasks/list {name:"round-trip"}` | **200**，total=1 ✓（旧实现 500 `no such column: task_name`） |
| 改 1 | `POST /tasks/{id}/update {status:1, priority:7, tags:"gamma"}` | 200 |
| 查 2 | 详情 | `status=1` ✓ `priority=7` ✓ `tags="gamma"` ✓ |
| 改 2 | `POST /tasks/{id}/update {status:0, priority:0}` | 200 |
| 查 3 | 详情 | **`status=0` ✓ `priority=0` ✓**（旧实现 `>0` 判定永远写不回 0） |
| 改 3（未传 status） | `POST /tasks/{id}/update {name:"…renamed"}` | 200；查：name 已改，**status 仍保留 0** ✓（nil ≠ 0） |
| 越界 | `POST /tasks/{id}/update {priority:101}` | **400**，message 含 `0-100` ✓ |
| 软删 | `POST /tasks/{id}/delete` | 200；再按名搜索 total=0 ✓ |

---

## CONTRACT-04 选择器学习表打通 — **PASS**

### 改动文件

| 文件 | 行 | 改动 |
|---|---|---|
| `internal/services/rpa/selector_learner.go` | 84 | `func (SelectorSuccessRecord) TableName() string { return "sys_rpa_selector_success" }`。D4 |
| 同上 | 116 | `func (SelectorFailureRecord) TableName() string { return "sys_rpa_selector_failure" }` |
| 同上 | 60 / 74 | ID tag 去掉 PG-only 的 `default:gen_random_uuid()`（sqlite 建表即 `near "(": syntax error`） |
| 同上 | 94 / 121 | 新增 `BeforeCreate` 在应用层补 UUID（PG/sqlite 行为一致） |
| `scripts/dbprovision/main.go` | 105-106 | migrateList 注册 `&svcrpa.SelectorSuccessRecord{}` / `&svcrpa.SelectorFailureRecord{}` |
| `internal/core/db/database.go` | 907-908 | sqlite 分支 AutoMigrate 注册两模型 |
| 同上 | 917-918 | （顺带修 CONTRACT-01 缺表）sqlite 注册 `RPACredential` / `RPASession` |
| `internal/core/db/database_test.go` | 210 | `TestNewDatabaseSQLite` 补断言 `sys_rpa_selector_success` / `sys_rpa_selector_failure` 两表存在 |
| `internal/services/rpa/selector_learner_learn_test.go` | 28-48 | **删除**手写 `CREATE TABLE selector_success_records` / `selector_failure_records` / `sys_rpa_executions` DDL，改 `db.AutoMigrate(&Execution{}, &SelectorSuccessRecord{}, &SelectorFailureRecord{})` + `HasTable` / `TableName()` 断言 |
| `internal/services/rpa/ai_selector_excel_test.go` | 299-310 | 同上：`newSelectorTestDB` 改真实 AutoMigrate |
| 同上 | 406 / 410 | raw INSERT 表名改为真实 `sys_rpa_selector_success` / `sys_rpa_selector_failure` |

### 验收证据

```
$ go test -count=1 ./internal/services/rpa/ -run "TestSelectorLearner|TestLearnFromExecution" -v
--- PASS: TestSelectorLearner_RecordAndGetBest
--- PASS: TestSelectorLearner_NilResultNotCached
--- PASS: TestSelectorLearner_ScoreAndAlternatives
--- PASS: TestLearnFromExecution_ConsumesSelectorEvents
--- PASS: TestLearnFromExecution_FailureThreshold
--- PASS: TestLearnFromExecution_Errors

$ go test -count=1 ./internal/core/db/ -run TestNewDatabaseSQLite -v
--- PASS: TestNewDatabaseSQLite (1.26s)   # 含两表 HasTable 断言
```

`contract_p0_smoke_test.go:595` 与 `:626`，真实 sqlite 全量 AutoMigrate 后：

| 断言 | 结果 |
|---|---|
| `HasTable(&SelectorSuccessRecord{})` / `(&SelectorFailureRecord{})` | true ✓ |
| `TableName()` == `sys_rpa_selector_success` / `sys_rpa_selector_failure` | ✓ |
| `sqlite_master` 中**无** `selector_success_records` 影子表 | count=0 ✓ |
| 直接 `db.Create(...)` 两模型（不经 service） | 成功 ✓，`ID` 由 BeforeCreate 补 UUID ✓ |

**11 个 `/rpa/ai/selector/*` + AI 端点各一条真库用例**（`:626`，逐条 200 + data 非空）：

| # | 端点 | 断言要点 |
|---|---|---|
| 1 | `POST /rpa/ai/selector/record-success` | 200；真表 count=1 |
| 2 | 再次 `record-success` | 200；`success_count` 递增到 2（真表 upsert 可写） |
| 3 | `POST /rpa/ai/selector/record-failure` | 200；`sys_rpa_selector_failure` count=1 |
| 4 | `POST /rpa/ai/selector/best` | 200；`selector=="#login"`（从真表选出） |
| 5 | `POST /rpa/ai/selector/score` | 200；`data.score` 非空 |
| 6 | `POST /rpa/ai/selector/alternatives` | 200；`data.alternatives` 非空 |
| 7 | `POST /rpa/ai/generate` | 200（真实 AI provider = httptest 外部边界替身） |
| 8 | `POST /rpa/ai/optimize` | 200 |
| 9 | `POST /rpa/ai/decide` | 200 |
| 10 | `POST /rpa/ai/analyze-failure` | 200 |
| 11 | `POST /rpa/ai/suggest-fix` | 200 |
| 12 | `POST /rpa/ai/classify-error` | 200 |

> AI provider 用 `httptest.NewServer` 属**外部 HTTP 依赖**边界替身，不是内部 schema/service 替身；`SetupAIRouter` 注册 11 路由由既有 `router_public_test.go:TestSetupAIRouter_Smoke` 断言。

---

## CONTRACT-05 前端不再用 mock 冒充 — **PASS**

### 改动文件（AIScriptEditor — D15）

| 文件 | 行 | 改动 |
|---|---|---|
| `xingran-react-frontend/.../tasks/modals/AIScriptEditor.tsx` | 43-46 | **删除** `mockGeneratedActions`（写死的 5 条演示动作）常量 |
| 同上 | 65 | 新增 `AI_GENERATE_TIMEOUT_MS = 35000`（前端超时预算；后端 AIClient 自身 30s） |
| 同上 | 78-92 | 新增 `mapAIActions(dtos)`：后端 `ScriptAction` DTO → 前端 `Action`；未知 `type` 按 `ACTION_TYPE_OPTIONS` 白名单丢弃 |
| 同上 | 94 | 新增 `GenerateErrorKind = "failed" \\| "timeout" \\| "empty"` 三态 |
| 同上 | 117-160 | `handleGenerate`：**删除 `setTimeout(1500)`**，改 `api.post("/rpa/ai/generate", {description}, {signal})`；AbortSignal + 35s 超时中断 |
| 同上 | 136-146 | 空结果态（200 但 `actions` 为空或全部被白名单过滤）→ `kind:"empty"` |
| 同上 | 152-176 | 失败/超时分支分别产出 `kind:"failed"`（含后端 message）/ `kind:"timeout"` |
| 同上 | 178-190 | `useEffect` 关闭时中断在途请求，避免卸载后 setState |
| 同上 | 322-352 | 渲染 `data-testid="ai-generate-error"` 三态告警 + 「重试生成」按钮 |
| 同上 | 355-364 | 渲染 `data-testid="ai-generate-explanation"`（AI 说明） |

### 改动文件（执行页实时化 — D16）

| 文件 | 行 | 改动 |
|---|---|---|
| `xingran-react-frontend/.../executions/index.tsx` | 12 | `import { useRPAProgress } from "@/hooks/useRPAProgress"` |
| 同上 | 109-145 | **删除 `setInterval(…, 5000)` running 轮询块**；改为 `useRPAProgress({onProgress,onCompleted,onFailed})` 订阅 WS 事件驱动刷新；仅保留 **30s 低频断线兜底**（且仅在存在 running 时） |

### 新增测试

| 文件 | 用例 | 断言要点 |
|---|---|---|
| `tasks/modals/__tests__/AIScriptEditor.realapi.test.tsx` | 4 | ① 端点 `/rpa/ai/generate` 收到真实请求（url + description 精确匹配），且在**不推进虚拟时钟**下拿到动作步骤（证明结果不来自 setTimeout）；② 失败态可读错误 + 重试按钮；③ 空结果态（200 + actions:[]）；④ 超时态（AbortError） |
| `executions/__tests__/index.realtime.test.tsx` | 2 | ① 存在 running 记录时推进 12s 虚拟时钟，列表请求仍只 1 次（旧 5s 轮询会 ≥3 次）；② WS 进度消息经 `noticeStore.handleWsMessage` 派发后列表请求次数 ≥2（实时链路成立） |

### 验收证据

```
$ npx vitest run --pool=threads src/pages/operations/rpa/tasks/modals/__tests__/AIScriptEditor.realapi.test.tsx
 ✓ (4 tests) 5918ms
   ✓ 发出真实 HTTP 请求（url + description），且不依赖 setTimeout 假延迟  1958ms
   ✓ 失败态：后端报错时展示可读错误并允许重试  1774ms
   ✓ 空结果态：200 但 actions 为空时给出警告而非假装成功  1089ms
   ✓ 超时态：请求 AbortError 时提示超时（而非笼统失败）  1091ms

$ npx vitest run --pool=threads src/pages/operations/rpa/executions/__tests__/index.realtime.test.tsx
 ✓ (2 tests) 4121ms
   ✓ 存在 running 任务时也不以 5 秒间隔轮询列表  3560ms
   ✓ WebSocket 进度消息到达时刷新列表（实时链路成立）  558ms
```

### 红绿变异验证（证明测试真能挡回归）

把 `executions/index.tsx` 兜底间隔从 `30000` 改回旧值 `5000` 后：

```
× 存在 running 任务时也不以 5 秒间隔轮询列表 3548ms
AssertionError: expected "vi.fn()" to be called 1 times, but got 3 times
Tests  1 failed | 1 passed (2)
```

已还原为 30000。

---

## CONTRACT-06 契约回归闸门 — **PASS**

### 改动文件

| 文件 | 行 | 内容 |
|---|---|---|
| `internal/api/v1/rpa/contract_p0_smoke_test.go` | 新增，730 行 | httptest + sqlite **真表**（`db.NewDatabase` + 真实 `AutoMigrate()`）+ **真实 service**（`rpa.NewServiceGroup` + 真实 `core.SM4Cipher`），覆盖 CONTRACT-01~05 对应请求。**无一处手写 CREATE TABLE、无 mock service** |
| `AGENTS.md` | 55-65 | 「测试目录与文件命名（强制）」段新增约束：**禁止在 handler 测试里用自建表 / mock service 掩盖真实 schema** |

### 8 个冒烟用例（逐条 PASS）

```
$ go test -count=1 ./internal/api/v1/rpa/ -run "TestContractP0" -v
=== RUN   TestContractP0_CredentialCRUD_SQLiteRealTables
--- PASS: TestContractP0_CredentialCRUD_SQLiteRealTables (0.83s)      # CONTRACT-01 正向
=== RUN   TestContractP0_CredentialNilCipher_ReadableError
--- PASS: TestContractP0_CredentialNilCipher_ReadableError (0.76s)    # CONTRACT-01 负向
=== RUN   TestContractP0_TaskListStatusBinding
--- PASS: TestContractP0_TaskListStatusBinding (0.99s)                # CONTRACT-02 任务列表
=== RUN   TestContractP0_WorkerListStatusBinding
--- PASS: TestContractP0_WorkerListStatusBinding (1.61s)              # CONTRACT-02 Worker 列表
=== RUN   TestContractP0_ExecutionListStatusBinding
--- PASS: TestContractP0_ExecutionListStatusBinding (1.39s)           # CONTRACT-02 执行列表
=== RUN   TestContractP0_TaskRoundTrip_TagsAndZeroStatus
--- PASS: TestContractP0_TaskRoundTrip_TagsAndZeroStatus (1.35s)      # CONTRACT-03
=== RUN   TestContractP0_SelectorTables_RealMigrationThenWrite
--- PASS: TestContractP0_SelectorTables_RealMigrationThenWrite (1.71s) # CONTRACT-04
=== RUN   TestContractP0_AIAndSelectorEndpoints_RealDB
--- PASS: TestContractP0_AIAndSelectorEndpoints_RealDB (1.81s)        # CONTRACT-04 11 端点
PASS
ok  github.com/xingran-next/xingran-go-backend/internal/api/v1/rpa  10.692s
```

### 纳入 CI

`go test ./internal/api/v1/rpa/` 天然覆盖（无 build tag、无 `-short` 跳过、无需外部服务 —— sqlite 是纯 Go 驱动，AI provider 是进程内 httptest）。

### AGENTS.md 新增约束原文（摘要）

- handler 层契约用例必须走 **httptest + 真实数据库（`db.NewDatabase` → `AutoMigrate()`，或真实迁移后的表）**，不得 `db.Exec("CREATE TABLE ...")` 手写表结构让用例变绿；
- 不得用 mock service（`mockTaskService` / `mockCredentialService` 等）作为**契约验收**替身，mock 只用于 handler 分支/错误映射的纯单元用途；
- 唯一允许的外部替身是**外部 HTTP 依赖**（AI provider、第三方 API），用 `httptest.NewServer`；
- 权威样板指向 `internal/api/v1/rpa/contract_p0_smoke_test.go`。

---

## 全量验收命令与结果

```
$ go build ./...
BUILD_EXIT=0

$ go test -count=1 ./internal/api/v1/rpa/ ./internal/services/rpa/ ./internal/models/rpa/
ok   github.com/xingran-next/xingran-go-backend/internal/api/v1/rpa        7.683s
ok   github.com/xingran-next/xingran-go-backend/internal/services/rpa     0.762s
ok   github.com/xingran-next/xingran-go-backend/internal/models/rpa       0.434s

$ go test -count=1 ./internal/core/db/          # database.go 改动回归
ok   github.com/xingran-next/xingran-go-backend/internal/core/db         10.109s

$ go build ./scripts/dbprovision/
OK

$ cd xingran-react-frontend && npx tsc --noEmit -p tsconfig.app.json
TSC_EXIT=0

$ npx vitest run --pool=threads src/pages/operations/rpa
Test Files  9 passed (9)
Tests       42 passed (42)
```

> vitest 备注：`maxWorkers=4` 并发下 `tasks/__tests__/columns.test.tsx` 偶发 45s 冷启动超时（既有 flake，与本次改动无关 —— 排除本次新增的两个测试文件后仍能复现一次、复跑通过；`--maxWorkers=1` 串行 42/42 稳定通过）。不以改动测试超时阈值的方式掩盖。

---

## 归属协议自检（越界检查）

| 协议 | 约束 | 实际 diff 区间 | 结论 |
|---|---|---|---|
| 1 | `task_service.go` 只碰 `:81-200`，禁止 `:243+` | `+96,104,141,147,154,189,192`（全部 < 200）；`:243` 之后**零改动** | ✅ |
| 2 | `rpa_router.go` 只碰 `:22-52`，禁止尾部加路由 | 仅 `@@ -24 +24,6 @@` 与 `@@ -51 +56,6 @@`（两处 NewServiceGroup）；`SetupRPARouter` 尾部零新增路由 | ✅ |
| 3 | `executions/index.tsx` 只碰轮询块 `:105-135`，禁止 `:135+` | diff 区间 `+109..145`，仅轮询块与 import；`:135` 之后 handler 区（`handleViewDetail` 起）零改动 | ✅ |
| 4 | `constants.tsx` 的 `ACTION_TYPE_OPTIONS:166-176` 不要动 | diff 仅 `+32/+37/+43/+49/+54/+63/+73/+83`；`ACTION_TYPE_OPTIONS` 区间零改动 | ✅ |

`git status --short` 共 26 项（23 M + 3 ??），全部属本 Phase 文件面：

- 后端：`rpa_router.go`、`credential_handler.go`、`task_handler.go`、`worker_handler.go`、`execution_handler.go`、`handler_helpers.go`、`contract_p0_smoke_test.go`(新)、`credential_service.go`、`task_service.go`、`types.go`、`selector_learner.go`、两个 selector 测试、`database.go`、`database_test.go`、`dbprovision/main.go`、`AGENTS.md`
- 前端：`constants.tsx`、`tasks/index.tsx`、`EditModal.tsx`、`AIScriptEditor.tsx`、`executions/index.tsx`、`constants.test.tsx`、两个新测试文件

**未触碰** Phase 153（`task_service.go:243+`、`executions/index.tsx:135+`）、Phase 154（`constants.tsx ACTION_TYPE_OPTIONS`）、Phase 158（`rpa_router.go` 尾部）的任何一行。

---

## 遗留项

| # | 项 | 归属 | 说明 |
|---|---|---|---|
| L1 | `pkg/response.toAppError` 的 `int` 分支恒映射 HTTP 400 | 全局（非本 Phase） | `response.Error(c, http.StatusNotFound, ...)` 实际返回 400。影响全模块所有「传 int 状态码」的 handler。本 Phase 不改（改动面过大），已在 CONTRACT-01 用例注释登记 |
| L2 | `POST /rpa/credentials/sessions/list` 仍是空桩（`credential_handler.go:ListSessions` 返回 `{list:[],total:0}`） | 后续 Phase | 本次只验了「会话创建（服务层）+ 失效（HTTP）」，会话列表未实装；不在 CONTRACT-01 六条缺陷范围 |
| L3 | `credential_service_test.go` 仍用 `db.Exec(CREATE TABLE sys_rpa_credentials …)` 手写表 | 后续清理 | 属**服务层**单测（非 handler 契约闸门），新约束只强制 handler 层；且这些表已在 sqlite AutoMigrate 注册，后续可统一改 AutoMigrate |
| L4 | `GetCredentialForExecution` 的 `ORDER BY last_used_at DESC NULLS LAST` 是 PG 语法 | 后续 Phase | sqlite 下该端点会报错；CONTRACT-01 验收路径未覆盖此函数（执行链路属 Phase 153 RUNTIME-02） |
| L5 | `AIScriptEditor` 未做「AI 不可用 → 降级录制入口」 | Phase 156（NOCODE-04） | 本 Phase 只做真实调用与三态提示；降级入口按 ROADMAP 归 156 |
| L6 | vitest `maxWorkers=4` 下 `columns.test.tsx` 冷启动 flake | 既有 | 非本次引入，已记录复现条件与串行验证结果 |

