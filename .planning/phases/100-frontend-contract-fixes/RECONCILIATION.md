# Phase 100 — rpaApi 全族契约对账台账（RECONCILIATION）

**Phase:** 100-frontend-contract-fixes（Plan 100-01, V130R-10/V130R-11）
**日期:** 2026-09-06（执行落盘 2026-09-07）
**裁决依据:** D-100-3（三分类 alive/dead/mismatch）、D-100-9（端态裁至存活集，方案 B）、D-100-12（worker register/heartbeat 前端方法删除）
**后端真相源:** `internal/api/v1/rpa/rpa_router.go`（公开组挂载 `internal/api/router.go:953-955`，JWT 组挂载 `:962-968`；前端 wire 路径 = `/api/v1/rpa/...`）
**对账总量:** 116 方法 → 17 alive（保留）/ 97 dead（删除）/ 2 alive-but-deleted（register/heartbeat，D-100-12）

**关键事实:** rpaApi.ts 生产消费者为零——RPA 三页面全部内联 `post` 直调（tasks/index.tsx:18、executions/index.tsx:18、workers/index.tsx:39）；rpa 族内 mismatch 为零（所有无路由方法均无调用方，全部判 dead）。

## 六列对账表

裁决列取值：`alive→保留` / `dead→删除` / `alive-but-deleted→前端删除（D-100-12）`。

### taskApi（15 方法：6 alive / 9 dead）

| 方法 | 动词 | 路径 | 后端路由（rpa_router.go） | 裁决 | 理由 |
|------|------|------|--------------------------|------|------|
| list | POST | /rpa/tasks/list | ✓ :48 | alive→保留 | 路由注册 |
| get | POST | /rpa/tasks/:id | ✓ :50 | alive→保留 | 路由注册 |
| create | POST | /rpa/tasks | ✓ :49 | alive→保留 | 路由注册 |
| update | POST | /rpa/tasks/:id/update | ✓ :51 | alive→保留 | 路由注册 |
| delete | POST | /rpa/tasks/:id/delete | ✓ :52 | alive→保留 | 路由注册 |
| execute | POST | /rpa/tasks/:id/execute | ✓ :53 | alive→保留 | 路由注册（手写保留） |
| batch | POST | /rpa/tasks/batch | ✗ | dead→删除 | 工厂幽灵，零调用方 |
| statistics | POST | /rpa/tasks/statistics | ✗ | dead→删除 | 工厂幽灵，零调用方 |
| searchOptions | POST | /rpa/tasks/dropdown-options | ✗ | dead→删除 | 工厂幽灵，零调用方 |
| cancelExecution | POST | /rpa/tasks/:id/cancel | ✗ | dead→删除 | 无路由，仅测试引用 |
| duplicate | POST | /rpa/tasks/:id/duplicate | ✗ | dead→删除 | 无路由，仅测试引用 |
| executions | POST | /rpa/tasks/:id/executions | ✗ | dead→删除 | 无路由，仅测试引用 |
| validateScript | POST | /rpa/tasks/validate-script | ✗ | dead→删除 | 无路由，仅测试引用 |
| export | POST | /rpa/tasks/:id/export | ✗ | dead→删除 | 无路由，仅测试引用 |
| import | POST | /rpa/tasks/import | ✗ | dead→删除 | 无路由，仅测试引用 |

### scriptApi（10 方法：0 alive / 10 dead）— 整族删除

| 方法 | 动词 | 路径 | 后端路由 | 裁决 | 理由 |
|------|------|------|----------|------|------|
| list | POST | /rpa/scripts/list | ✗ 无 /rpa/scripts 组 | dead→删除 | 路由组全仓不存在（双重 grep 实证） |
| get | POST | /rpa/scripts/:id | ✗ | dead→删除 | 同上 |
| create | POST | /rpa/scripts | ✗ | dead→删除 | 同上 |
| update | POST | /rpa/scripts/:id/update | ✗ | dead→删除 | 同上 |
| delete | POST | /rpa/scripts/:id/delete | ✗ | dead→删除 | 同上 |
| batch | POST | /rpa/scripts/batch | ✗ | dead→删除 | 工厂幽灵 |
| statistics | POST | /rpa/scripts/statistics | ✗ | dead→删除 | 工厂幽灵 |
| searchOptions | POST | /rpa/scripts/dropdown-options | ✗ | dead→删除 | 工厂幽灵 |
| testAction | POST | /rpa/scripts/test-action | ✗ | dead→删除 | 手写死方法 |
| format | POST | /rpa/scripts/format | ✗ | dead→删除 | 手写死方法 |

### workerApi（15 方法：4 alive / 11 dead，其中 register/heartbeat 按 D-100-12 前端删除）

| 方法 | 动词 | 路径 | 后端路由（rpa_router.go） | 裁决 | 理由 |
|------|------|------|--------------------------|------|------|
| list | POST | /rpa/workers/list | ✓ :64 | alive→保留 | 路由注册 |
| statistics（无参分支） | POST | /rpa/workers/statistics | ✓ :66 | alive→保留 | 路由注册；D-100-9 收窄为无参形态 |
| statistics（id 分支） | POST | /rpa/workers/:id/statistics | ✗ | dead→删除 | 路由未注册，分支随收窄移除 |
| register | POST | /rpa/workers/register | ✓ :15（公开组） | alive-but-deleted→前端删除（D-100-12） | 路由存在但属 Worker 节点专用端点；RPA Worker 是独立进程直连后端 HTTP，不经 admin bundle；后端公开路由保留不动 |
| heartbeat | POST | /rpa/workers/:id/heartbeat | ✓ :18（公开组） | alive-but-deleted→前端删除（D-100-12） | 同上 |
| progress | POST | /rpa/workers/:id/progress | ✗（后端为无 id 的 /progress :19） | dead→删除 | 动宾均不匹配，零调用方 |
| get | POST | /rpa/workers/:id | ✗ | dead→删除 | 工厂幽灵，零调用方 |
| create | POST | /rpa/workers | ✗ | dead→删除 | 工厂幽灵 |
| update | POST | /rpa/workers/:id/update | ✗ | dead→删除 | 工厂幽灵 |
| delete | POST | /rpa/workers/:id/delete | ✗ | dead→删除 | 工厂幽灵 |
| batch | POST | /rpa/workers/batch | ✗ | dead→删除 | 工厂幽灵 |
| searchOptions | POST | /rpa/workers/dropdown-options | ✗ | dead→删除 | 工厂幽灵 |
| getOnline | POST | /rpa/workers/online | ✗ | dead→删除 | 手写死方法 |
| offline | POST | /rpa/workers/:id/offline | ✗ | dead→删除 | 手写死方法 |
| restart | POST | /rpa/workers/:id/restart | ✗ | dead→删除 | 手写死方法 |

### executionApi（14 方法：5 alive / 9 dead）

| 方法 | 动词 | 路径 | 后端路由（rpa_router.go） | 裁决 | 理由 |
|------|------|------|--------------------------|------|------|
| list | POST | /rpa/executions/list | ✓ :84 | alive→保留 | 路由注册 |
| get | POST | /rpa/executions/:id | ✓ :87 | alive→保留 | 路由注册 |
| statistics | POST | /rpa/executions/statistics | ✓ :86 | alive→保留 | 路由注册（工厂 pick） |
| cancel | POST | /rpa/executions/:id/cancel | ✓ :88 | alive→保留 | 路由注册（手写保留） |
| logs | POST | /rpa/executions/:id/logs | ✓ :89 | alive→保留 | 路由注册（手写保留） |
| create | POST | /rpa/executions | ✗ | dead→删除 | 工厂幽灵，零调用方 |
| update | POST | /rpa/executions/:id/update | ✗ | dead→删除 | 工厂幽灵 |
| delete | POST | /rpa/executions/:id/delete | ✗ | dead→删除 | 工厂幽灵 |
| batch | POST | /rpa/executions/batch | ✗ | dead→删除 | 工厂幽灵 |
| searchOptions | POST | /rpa/executions/dropdown-options | ✗ | dead→删除 | 工厂幽灵 |
| streamLogs | POST | /rpa/executions/:id/stream-logs | ✗ | dead→删除 | 手写死方法 |
| screenshots | POST | /rpa/executions/:id/screenshots | ✗ | dead→删除 | 手写死方法 |
| downloadReport | POST | /rpa/executions/:id/report?format= | ✗（后端为 GET /:id/download :90，动宾皆异） | dead→删除 | 无路由；其 downloadFilePost 消费点随删，download.ts 函数本身保留（活消费者 opsApi excel/asset export） |
| retry | POST | /rpa/executions/:id/retry | ✗ | dead→删除 | 手写死方法 |

### scheduleApi（14 方法：0 alive / 14 dead）— 整族删除

| 方法 | 动词 | 路径 | 后端路由 | 裁决 | 理由 |
|------|------|------|----------|------|------|
| list/get/create/update/delete/batch/statistics/searchOptions | POST | /rpa/schedules/* | ✗ 无 /rpa/schedules 组 | dead→删除 | 路由组全仓不存在；调用方仅测试 |
| activate | POST | /rpa/schedules/:id/activate | ✗ | dead→删除 | 同上 |
| pause | POST | /rpa/schedules/:id/pause | ✗ | dead→删除 | 同上 |
| disable | POST | /rpa/schedules/:id/disable | ✗ | dead→删除 | 同上 |
| runNow | POST | /rpa/schedules/:id/run-now | ✗ | dead→删除 | 同上 |
| validateCron | POST | /rpa/schedules/validate-cron | ✗ | dead→删除 | 同上 |
| nextRunTime | POST | /rpa/schedules/:id/next-run | ✗ | dead→删除 | 同上 |

### variableApi（12 方法：0 alive / 12 dead）— 整族删除

| 方法 | 动词 | 路径 | 后端路由 | 裁决 | 理由 |
|------|------|------|----------|------|------|
| list/get/create/update/delete/batch/statistics/searchOptions | POST | /rpa/variables/* | ✗ 无 /rpa/variables 组 | dead→删除 | 路由组全仓不存在；调用方仅测试 |
| getGlobal | POST | /rpa/variables/global | ✗ | dead→删除 | 同上 |
| getByTask | POST | /rpa/variables/task/:taskId | ✗ | dead→删除 | 同上 |
| batchSet | POST | /rpa/variables/batch-set | ✗ | dead→删除 | 同上 |
| decrypt | POST | /rpa/variables/:id/decrypt | ✗ | dead→删除 | 同上 |

### templateApi（13 方法：0 alive / 13 dead）— 整族删除

| 方法 | 动词 | 路径 | 后端路由 | 裁决 | 理由 |
|------|------|------|----------|------|------|
| list/get/create/update/delete/batch/statistics/searchOptions | POST | /rpa/templates/* | ✗ 无 /rpa/templates 组 | dead→删除 | 路由组全仓不存在；调用方仅测试 |
| categories | POST | /rpa/templates/categories | ✗ | dead→删除 | 同上 |
| useTemplate | POST | /rpa/templates/:id/use | ✗ | dead→删除 | 同上 |
| rate | POST | /rpa/templates/:id/rate | ✗ | dead→删除 | 同上 |
| favorite | POST | /rpa/templates/:id/favorite | ✗ | dead→删除 | 同上 |
| unfavorite | POST | /rpa/templates/:id/unfavorite | ✗ | dead→删除 | 同上 |

### aiApi（6 方法：4 alive / 2 dead）

| 方法 | 动词 | 路径 | 后端路由（rpa_router.go） | 裁决 | 理由 |
|------|------|------|--------------------------|------|------|
| generateScript | POST | /rpa/ai/generate | ✓ :101 | alive→保留 | 路由注册 |
| optimizeScript | POST | /rpa/ai/optimize | ✓ :102 | alive→保留 | 路由注册 |
| decide | POST | /rpa/ai/decide | ✓ :105 | alive→保留 | 路由注册 |
| analyzeFailure | POST | /rpa/ai/analyze-failure | ✓ :108 | alive→保留 | 路由注册 |
| explainScript | POST | /rpa/ai/explain | ✗ | dead→删除 | 无路由，零调用方 |
| captureState | POST | /rpa/ai/capture-state | ✗ | dead→删除 | 无路由，零调用方 |

### notificationApi（12 方法：0 alive / 12 dead）— 整族删除

注意：与 KEEP 例外 `src/lib/notificationConfigApi.ts`（system 模块，工厂 KEEP 名单在册）完全无关，本族删除不触及该文件。

| 方法 | 动词 | 路径 | 后端路由 | 裁决 | 理由 |
|------|------|------|----------|------|------|
| list/get/create/update/delete/batch/statistics/searchOptions | POST | /rpa/notifications/* | ✗ 无 /rpa/notifications 组 | dead→删除 | 路由组全仓不存在；调用方仅测试 |
| enable | POST | /rpa/notifications/:id/enable | ✗ | dead→删除 | 同上 |
| disable | POST | /rpa/notifications/:id/disable | ✗ | dead→删除 | 同上 |
| test | POST | /rpa/notifications/:id/test | ✗ | dead→删除 | 同上 |
| getGlobal | POST | /rpa/notifications/global | ✗ | dead→删除 | 同上 |

### statisticsApi（5 方法：0 alive / 5 dead）— 整族删除

| 方法 | 动词 | 路径 | 后端路由 | 裁决 | 理由 |
|------|------|------|----------|------|------|
| overview | POST | /rpa/statistics/overview | ✗ 无 /rpa/statistics 组 | dead→删除 | 路由组全仓不存在；调用方仅测试 |
| tasks | POST | /rpa/statistics/tasks | ✗ | dead→删除 | 同上 |
| workers | POST | /rpa/statistics/workers | ✗ | dead→删除 | 同上 |
| executions | POST | /rpa/statistics/executions | ✗ | dead→删除 | 同上 |
| trends | POST | /rpa/statistics/trends | ✗ | dead→删除 | 同上 |

## 端态（17 存活方法）

文件锚点：`xingran-react-frontend/src/lib/rpaApi.ts`（Phase 100 裁剪后）

| 对象 | 方法 | 后端路由（rpa_router.go） |
|------|------|--------------------------|
| taskApi（6） | list, get, create, update, delete（工厂 pick）, execute（手写） | :48-53 |
| workerApi（2） | list（工厂 pick）, statistics（手写，无参收窄版） | :64 / :66 |
| executionApi（5） | list, get, statistics（工厂 pick）, cancel, logs（手写） | :84-89 |
| aiApi（4） | generateScript, optimizeScript, decide, analyzeFailure | :101-108 |
| 聚合 `rpaApi` | 4 键 { task, worker, execution, ai } | — |

守卫：`apiFactory.invariants.test.ts` keys 基线（D-100-2/D-100-5）+ `rpaApi.test.ts` 存活契约测试（17 方法 URL+动词断言）双向锁定；回增/私删即红。

## 类型处置说明

死方法/死族专属类型（Script/Schedule/Variable/Template/NotificationConfig/WorkerRegisterRequest/WorkerHeartbeatRequest/ExecutionProgress/AIScriptExplain*/CaptureState*/RPAStatistics 等）逐一 grep 后：凡被存活代码（RPA 页面 Task/Execution/Worker/Action/ExecutionLog、useRPAProgress/noticeStore 的 RPAProgressMessage、types/rpa.ts 内部类型网如 `Task.script?: Action[] | Script`）引用的一律保留于 `src/types/rpa.ts`（该文件不在本计划 files_modified 范围）。rpaApi.ts 内的 import 块与 7 个 `*ListSearchParams` 导出接口（全仓零消费者）已随裁剪收缩/删除。

## observed-debt（登记不修，D-100-9 纪律）

- RPA 三页面绕过 apiFactory/xxxApi 层内联 `post` 直调（既有约定债）：`src/pages/operations/rpa/tasks/index.tsx:18`、`src/pages/operations/rpa/executions/index.tsx:18`、`src/pages/operations/rpa/workers/index.tsx:39`。rpaApi 端态保留后页面是否迁移到 xxxApi 层属后续 phase 决策，本相不顺手扩 scope。
- 后端 `/rpa/workers/register`、`/rpa/workers/:id/heartbeat` 公开路由保留（设计内 Worker 节点端点，D-100-12 明示不动）；admin bundle 不再携带这两个端点的前端包装器。
