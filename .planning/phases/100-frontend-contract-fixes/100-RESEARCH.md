# Phase 100: 前端契约修复 - Research

**Researched:** 2026-09-06
**Domain:** 前端 `src/lib` API 契约层与后端 gin 路由注册表对齐（rpaApi 全族对账 / vdiApi 幽灵方法 / networkApi 下载链收敛）
**Confidence:** HIGH（本报告所有关键结论均在本次会话中以 file:line 实测验证，非训练记忆）

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions（D-100-1..8，逐字复制自 100-CONTEXT.md）

**V130R-10 幽灵方法处置**
- **D-100-1:** 用户标准「不考虑向后兼容」→ rpaApi 8 处 / vdiApi 2 处工厂 spread 幽灵方法**直接删除**（不保留空壳）。删除前逐一确认零调用方（grep 调用面）；有调用方的幽灵方法升格为 V130R-11 对账项处理。
- **D-100-2:** 守卫升级——`apiFactory.invariants.test.ts` 对 rpaApi/vdiApi 锁定清理后方法集（KEEP baseline 更新为「后端路由存在」的实测集合），防回增。

**V130R-11 rpaApi 全族契约对账**
- **D-100-3:** 对账方法：scriptApi/scheduleApi/variableApi/templateApi/notificationApi/statisticsApi 全族逐一列清单——方法名/HTTP 动词/路径 → 对照后端 `internal/api/v1/` rpa 路由注册表 → 三分类：**alive**（路由存在，保留）/ **dead**（路由不存在且零调用方，删除）/ **mismatch**（路由不存在但有调用方——逐个裁决：调用方同删，或确有产品语义则后端补路由；默认删，补路由需在 RESEARCH 中给出明确证据）。
- **D-100-4:** 对账清单落盘为 phase 工件 `RECONCILIATION.md`（方法/动词/路径/后端路由/裁决/理由 六列），作为 SUMMARY 附件归档。
- **D-100-5:** 守卫：invariants 测试 rpaApi 档锁定清理后基线（warning tier 计数 = 清理后实测值），死方法回增即红。

**V130R-12 下载链收敛**
- **D-100-6:** `src/lib/api/networkApi.ts` 内嵌 axios 下载链全部收敛到权威 `src/lib/download.ts`（blobAxios + downloadFile/downloadFilePost），禁止平行实现。
- **D-100-7:** `downloadFilePost` 补 JSON 错误体检测：响应 `content-type` 含 `application/json` 时解析错误体并 throw（不再把 200+JSON 错误响应存成 .xlsx）；附回归测试（mock blob/json 两种响应）。
- **D-100-8:** `apiFactory.invariants.test.ts` 的 `readdirSync` 改递归扫描（`readdirSync` with `recursive: true` 或手工栈），覆盖子目录中的 `*Api.ts`。

**全局（用户长程标准适用）**
- 死方法/死链一律删除，不保留兼容导出
- 前端三 gate 硬约束：`npm run lint` 0 errors（1389 warnings 存量不倒退）、`npm run type-check` 通过、`npm run test` 0 失败（554 文件 / 3800 tests 基线）；覆盖率 45/45 dirs gate 不倒退
- 后端若需补路由（D-100-3 mismatch 裁决为补），走后端既有 Handler-Service 模式 + operlog 写操作约定

### Claude's Discretion
（CONTEXT.md 无显式 discretion 段；D-100-3 mismatch 逐项裁决由 planner 按「默认删、补路由需明确证据」标准执行）

### Deferred Ideas (OUT OF SCOPE)
None — 三项全部在 scope 内。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| V130R-10 | rpaApi.ts（8 处）/vdiApi.ts（2 处）工厂 spread 幽灵方法处置，附回归测试 | §1 完整清单：8+2 个 spread 站点逐一定位（file:line）、54+6 个工厂注入无路由方法实例、零生产调用方实测 |
| V130R-11 | rpaApi 全族契约对账——补路由或裁剪死方法，对账清单落盘，附守卫 | §2 全族 116 方法完整对账表（含 vdiApi 34 方法）、后端路由真相源 registry、19 alive / 97 dead / 6 mismatch 分类与裁决证据 |
| V130R-12 | networkApi 下载链收敛 + downloadFilePost JSON 检测 + invariants 递归扫描，附回归测试 | §3 四份平行实现 file:line、download.ts 权威 API 面、readdirSync 递归化影响面（EXPECTED_FILES 13→15） |
</phase_requirements>

## Summary

Phase 100 的三专项经全量源码对账后，实际形态比 REQUIREMENTS 的概述**更大也更简单**：rpaApi.ts 全族 116 个方法中只有 **19 个有后端路由**，且**整个 rpaApi.ts 没有任何生产页面消费者**——RPA 三个页面（tasks/executions/workers）全部直接 `import { post } from "@/lib/api"` 内联调用（tasks/index.tsx:18、executions/index.tsx:18、workers/index.tsx:39），rpaApi 的引用方只有它自己的 2 个测试文件 + download.test.ts（executionApi.downloadReport）+ 2 个页面测试的空转 `vi.mock`（mock 的方法名如 `getRPATaskList` 在 rpaApi 中根本不存在，属 Phase 88 遗留防御性 mock）。因此 V130R-11 的「mismatch 裁决」在前端侧几乎不存在——**所有无路由方法都是 dead 而非 mismatch**，唯一的 mismatch 有调用方案件在 **vdiApi**（`/vdi/vms/operate` 与 accounts 子资源族，有真实页面调用）。

后端 `/rpa` 路由真相源（`internal/api/v1/rpa/rpa_router.go`，挂载于 `router.go:962-968` JWT 组）只注册了 6 组：tasks(8) / workers-auth(7) + workers-public(3) / executions(9) / credentials(7) / ai(11) / flow(8)。**script/schedule/variable/template/notification/statistics 六个路由组在前后端任何位置都不存在**（全仓 `internal/` grep 实证）。

V130R-12 侧，networkApi.ts 持有第 2 份 blobAxios（:14-33）+ 第 2 份 triggerBrowserDownload（:183-192）+ 自有 JSON 错误嗅探启发式（`size<1024 && type含json`），且 MACHistoryPage.tsx:377-382 还有第 4 份内联 a/click 下载链；权威 `download.ts` 的 `downloadFilePost`（:79-92）确实无任何 JSON 错误体检测（仅 status 码检查）。invariants 测试的 `readdirSync(LIB_DIR)`（:125）非递归，导致 `src/lib/api/` 子目录的 networkApi.ts 与 macHeatmapApi.ts **双双逃过扫描**——递归化后 EXPECTED_FILES 需 13→15。

**Primary recommendation:** 按「先 vdi 补路由裁决（唯一需要用户确认的产品语义项）→ rpaApi 大裁剪 → vdiApi ghost/mismatch 处置 → networkApi 收敛 → 三重守卫升级（keys 基线 + 递归扫描 + downloadFilePost 回归）」顺序拆 plan；rpaApi 端态建议整文件评估删除（零生产消费者），至少裁至 19 alive 方法集。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| API 客户端方法集与后端路由对齐 | 前端 `src/lib` | 后端 router 注册表（真相源） | 契约以**后端已注册路由**为准；前端删除不触后端 |
| `/vdi/vms/operate` 生产批量操作 | 后端 `vm_router.go`（1 行补注册） | 前端 vdiApi | handler/service/测试已存在（vm_handler.go:204），只缺生产注册——补路由优于删功能 |
| VM accounts 子资源 | 前端删除（连同页面 Tab） | — | 后端 handler/service 均不存在，运行时本就 404；补齐 = 新功能开发，超出缺陷治理授权 |
| 文件下载传输链 | 前端 `src/lib/download.ts`（单一权威） | — | Phase 94 D-04 既定权威；networkApi 平行实现收敛 |
| 契约防回增守卫 | 前端 `apiFactory.invariants.test.ts` | — | D-12 双档 AST 扫描的既有扩展点 |

---

## 1. 幽灵方法完整清单（V130R-10）

### 1.1 「8 处 / 2 处」的精确定义（实测）

「8 处 / 2 处」= **createResourceApi spread 站点数**（v1.29-DEEP-RECHECK 前端域 Warning 原文：「rpa/vdi spread 新增后端不存在的 batch/statistics/searchOptions（潜伏 404 面）」——Phase 94 把 5 方法私有工厂提升为 8 方法权威工厂后，spread 实例**新增**了 batch/statistics/searchOptions 三方法，这些方法在后端无对应路由）：

**rpaApi.ts 8 个 spread 站点**（`createResourceApi` 调用行号实测）：

| # | 内部变量 | basePath | 行号 | 后端路由组存在？ |
|---|---------|----------|------|----------------|
| 1 | `taskCrudApi` | /rpa/tasks | rpaApi.ts:70 | 存在（rpa_router.go:47-58） |
| 2 | `scriptCrud` | /rpa/scripts | rpaApi.ts:132 | **不存在**（全仓无 /rpa/scripts） |
| 3 | `workerCrudApi` | /rpa/workers | rpaApi.ts:172 | 存在（rpa_router.go:61-80 + 公开组 :10-20） |
| 4 | `executionCrudApi` | /rpa/executions | rpaApi.ts:259 | 存在（rpa_router.go:83-96） |
| 5 | `scheduleCrudApi` | /rpa/schedules | rpaApi.ts:327 | **不存在** |
| 6 | `variableCrudApi` | /rpa/variables | rpaApi.ts:393 | **不存在** |
| 7 | `templateCrudApi` | /rpa/templates | rpaApi.ts:442 | **不存在** |
| 8 | `notificationCrudApi` | /rpa/notifications | rpaApi.ts:545 | **不存在** |

**vdiApi.ts 2 个 spread 站点**：`vmCrud`（vdiApi.ts:31，/vdi/vms）+ `vdiServerCrud`（vdiApi.ts:142，/vdi/servers）。

### 1.2 工厂注入幽灵方法逐站点清单（54 + 6 实例）

工厂 8 方法 = list/get/create/update/delete/batch/statistics/searchOptions。下表列**后端无路由**的工厂方法（alive 的不列）：

**rpaApi（54 个幽灵工厂方法实例，全部零调用方——含测试在内都无人调用 batch/statistics/searchOptions）：**

| 站点 | 幽灵工厂方法 | 后端缺失路由 |
|------|-------------|-------------|
| taskApi | batch, statistics, searchOptions (3) | POST /rpa/tasks/batch、/statistics、/dropdown-options 均未注册 |
| scriptApi | **全部 8 个** | 无 /rpa/scripts 组 |
| workerApi | get, create, update, delete, batch, searchOptions (6) | POST /rpa/workers/:id、POST /rpa/workers、/:id/update、/:id/delete、/batch、/dropdown-options 均未注册（statistics ✓ 存在 :66） |
| executionApi | create, update, delete, batch, searchOptions (5) | POST /rpa/executions、/:id/update、/:id/delete、/batch、/dropdown-options 均未注册（list/get/statistics ✓ :84-87） |
| scheduleApi | **全部 8 个** | 无 /rpa/schedules 组 |
| variableApi | **全部 8 个** | 无 /rpa/variables 组 |
| templateApi | **全部 8 个** | 无 /rpa/templates 组 |
| notificationApi | **全部 8 个** | 无 /rpa/notifications 组 |

**vdiApi（6 个幽灵工厂方法实例，全部零调用方）：**

| 站点 | 幽灵工厂方法 | 后端缺失路由 |
|------|-------------|-------------|
| vmApi | batch, statistics, searchOptions (3) | POST /vdi/vms/batch、/statistics、/dropdown-options 未注册（vm_router.go:12-44 无此三路由） |
| vdiServerApi | batch, statistics, searchOptions (3) | POST /vdi/servers/batch、/statistics、/dropdown-options 未注册（vdi_server_router.go:14-19 无） |

**调用面验证结论（D-100-1 前置条件全部满足）：** 对 `vmApi.\w+` / `vdiServerApi.\w+` / rpaApi 全部导出对象的全 src grep 实测——**batch/statistics/searchOptions 在全部 60 个 spread 实例上没有任何调用点（含测试）**。rpaApi 六个整族对象（script/schedule/variable/template/notification/statistics）的引用方只有自身测试文件。`vi.mock("@/lib/rpaApi")`（tasks/workers 页面测试 :16-22）mock 的是不存在的方法名，rpaApi 被删空后这些 factory mock 依然合法（vitest factory mock 不解析真实模块），但属死重，建议顺手清理。

### 1.3 vdiApi 的 6 个「非工厂」mismatch 方法（**唯一有生产调用方的案件**）

| 方法 | 前端路径 | 后端状态 | 生产调用方 | 分类 |
|------|---------|---------|-----------|------|
| `vmApi.operate` | POST /vdi/vms/operate | **handler 存在但未注册生产路由**（vm_handler.go:204 `Operate`；vm_router.go 无 `/operate`） | 无直接调用方 | mismatch（弱） |
| `vmApi.batchOperate` | POST /vdi/vms/operate（复用同路径，:83-88） | 同上 | **VirtualMachineList/index.tsx:546**（批量开关机/快照 UI） | **mismatch（强）** |
| `vmApi.listAccounts` | POST /vdi/vms/:vmId/accounts | handler/service 均不存在（vdi 包 grep `Account` 零命中） | VirtualMachineDetail/index.tsx:62 | mismatch（建议删） |
| `vmApi.createAccount` | 同上 | 同上 | VirtualMachineDetail/index.tsx:86 | mismatch（建议删） |
| `vmApi.resetAccountPassword` | POST /vdi/vms/:vmId/accounts/:accountId/reset_password | 同上 | VirtualMachineDetail/index.tsx:108 | mismatch（建议删） |
| `vmApi.deleteAccount` | POST /vdi/vms/:vmId/accounts/:accountId/delete | 同上 | VirtualMachineDetail/index.tsx:126 | mismatch（建议删） |

**operate 案件补路由证据（D-100-3「明确证据」标准）：** (a) `VMHandler.Operate` 已实现（vm_handler.go:204-227）且 handler 测试已覆盖成功/bad-json/service-error 三路径（vm_handler_test.go:384-407）；(b) service 层 `OperateVMs` 完整存在（vm_service_impl.go:789、vdi_client_extended.go:209-235 action 映射）；(c) 前端批量操作是 VirtualMachineList 活跃产品功能（:546 真实调用）；(d) 补注册 = vm_router.go 加 1 行 `r.POST("/operate", middleware.RequirePermissions([]string{"vdi:vm:edit"}, core), vmHandler.Operate)`（权限粒度比照同文件 :23-29 的电源操作）。**替代方案**：不补路由，前端 batchOperate 改为循环调用已注册的 `/start` `/stop` `/restart` 三路由（vm_router.go:27-29）——语义等价但失去原子性，或整组删除（批量操作功能死亡）。**倾向补路由**，需 planner/用户确认。

**accounts 案件建议：** 删 4 方法 + VirtualMachineDetail 的 accounts Tab UI（:36-40, :58-134, :137+）。该功能**运行时本就 404**（后端零实现），删除是移除已坏功能而非功能回退；补齐需从 VDI client 到 handler 的全链路新开发，超出缺陷治理 scope（ROADMAP D-01「不引入新业务功能」）。

---

## 2. rpaApi 全族契约对账清单（V130R-11，核心交付物）

### 2.1 后端 /rpa 路由真相源 registry（rpa_router.go 全量，file:line 实测）

挂载：`router.go:953-955`（公开组 `/rpa` → SetupPublicWorkerRouter）+ `router.go:962-968`（JWT 组 `/rpa` → SetupRPARouter）。前端 axios baseURL = `VITE_API_BASE_URL || "/api/v1"`，故 wire 路径 = `/api/v1/rpa/...`。

```
tasks（rpa_router.go:47-58）    : POST /list, ""(create), /:id, /:id/update, /:id/delete, /:id/execute, /upload-excel, /:id/execute-with-excel
workers 公开（:10-20）          : POST /register, /:id/heartbeat, /progress
workers 认证（:61-80）          : POST /list, /statistics, /:id/scale-up, /:id/scale-down, /scale-all, GET+POST /autoscale/config
executions（:83-96）            : POST /list, /statistics, /:id, /:id/cancel, /:id/logs, GET /:id/download, GET /:id/batch-report, GET+POST /:id/human-intervention
credentials（:143-154）         : POST /list, ""(create), /:id, /:id/update, /:id/delete, /sessions/list, /sessions/:id/invalidate
ai（:99-118）                   : POST /generate, /optimize, /decide, /analyze-failure, /suggest-fix, /classify-error, /selector/{record-success,record-failure,best,score,alternatives}
flow（:121-140）                : POST /evaluate-condition, /map-data, /transform-value, /extract-jsonpath, /aggregate-data, /handle-error, /execute-retry
```

**不存在**（`internal/` 全仓 `Group("/scripts|schedules|variables|templates|notifications|statistics")` + `.POST("/scripts|..."` 双重 grep 零命中，仅命中 workorder/duty/network 等其他模块的同名路由）：`/rpa/scripts`、`/rpa/schedules`、`/rpa/variables`、`/rpa/templates`、`/rpa/notifications`、`/rpa/statistics`、`/rpa/tasks/{batch,statistics,dropdown-options,cancel,duplicate,executions,validate-script,export,import}`、`/rpa/workers/{get,create,update,delete,batch,dropdown-options,online,offline,:id/restart,:id/progress,:id/statistics}`、`/rpa/executions/{create,update,delete,batch,dropdown-options,:id/stream-logs,:id/screenshots,:id/report,:id/retry}`、`/rpa/ai/{explain,capture-state}`。

### 2.2 全族对账总表（116 方法 → 19 alive / 97 dead / 0 前端 mismatch）

**关键事实先行：rpaApi.ts 的生产消费者为零**（§2.3）。表中「调用方」= src/ 内 `rpaApi.` 或具名导入的调用点（排除 rpaApi.ts 自身）；RPA 页面经**内联 post 直调**同类端点者单独标注于「页面内联对照」列——这些直调与 rpaApi 无关，删除 rpaApi 方法不影响页面。

#### taskApi（15 方法：6 alive / 9 dead）

| 方法 | 动词+路径 | 后端路由 | 调用方 | 分类 |
|------|----------|---------|--------|------|
| list | POST /rpa/tasks/list | ✓ :48 | 仅 rpaApi.test.ts:29 | alive（页面走内联 post :65） |
| get | POST /rpa/tasks/:id | ✓ :50 | 仅测试 | alive |
| create | POST /rpa/tasks | ✓ :49 | 仅测试（页面内联 :139） | alive |
| update | POST /rpa/tasks/:id/update | ✓ :51 | 仅测试（页面内联 :136） | alive |
| delete | POST /rpa/tasks/:id/delete | ✓ :52 | 仅测试（页面内联 :98） | alive |
| execute | POST /rpa/tasks/:id/execute | ✓ :53 | 仅测试（页面内联 :118） | alive |
| batch | POST /rpa/tasks/batch | ✗ | 零 | **dead** |
| statistics | POST /rpa/tasks/statistics | ✗ | 零 | **dead** |
| searchOptions | POST /rpa/tasks/dropdown-options | ✗ | 零 | **dead** |
| cancelExecution | POST /rpa/tasks/:id/cancel | ✗ | 仅测试 :64 | **dead** |
| duplicate | POST /rpa/tasks/:id/duplicate | ✗ | 仅测试 :66 | **dead** |
| executions | POST /rpa/tasks/:id/executions | ✗ | 仅测试 :69 | **dead** |
| validateScript | POST /rpa/tasks/validate-script | ✗ | 仅测试 :76 | **dead** |
| export | POST /rpa/tasks/:id/export | ✗ | 仅测试 :105 | **dead** |
| import | POST /rpa/tasks/import | ✗ | 仅测试 :107 | **dead** |

#### scriptApi（10 方法：0 alive / 10 dead）— 整族删除

| 方法 | 路径 | 后端 | 调用方 | 分类 |
|------|------|------|--------|------|
| list/get/create/update/delete/batch/statistics/searchOptions | /rpa/scripts/* | 无组 | 仅 rpaApi.test.ts:90-99 + batch56 测试 | **全 dead** |
| testAction | POST /rpa/scripts/test-action | ✗ | 仅测试 :81 | **dead** |
| format | POST /rpa/scripts/format | ✗ | 仅测试 :78 | **dead** |

#### workerApi（15 方法：4 alive / 11 dead）

| 方法 | 路径 | 后端 | 调用方 | 分类 |
|------|------|------|--------|------|
| list | POST /rpa/workers/list | ✓ :64 | 仅测试（页面内联 :114） | alive |
| statistics（override，:211-230） | 无 id→POST /rpa/workers/statistics ✓ :66；有 id→POST /rpa/workers/:id/statistics ✗ | 半活 | 仅测试（页面内联 :67） | alive（id 分支 dead，收窄签名或删分支） |
| register | POST /rpa/workers/register | ✓ :15 公开 | 仅测试 :47 | alive*（*Worker 节点端点，前端 admin bundle 无使用语义，见 Open Questions Q4） |
| heartbeat | POST /rpa/workers/:id/heartbeat | ✓ :18 公开 | 仅测试 :50 | alive*（同上） |
| progress | POST /rpa/workers/:id/progress | ✗（后端为无 id 的 /progress :19） | 零 | **dead** |
| get | POST /rpa/workers/:id | ✗ | 零 | **dead** |
| create/update/delete/batch/searchOptions | /rpa/workers/* | ✗ | 零 | **dead** |
| getOnline | POST /rpa/workers/online | ✗ | 零 | **dead** |
| offline | POST /rpa/workers/:id/offline | ✗ | 零 | **dead** |
| restart | POST /rpa/workers/:id/restart | ✗ | 零 | **dead** |

#### executionApi（14 方法：5 alive / 9 dead）

| 方法 | 路径 | 后端 | 调用方 | 分类 |
|------|------|------|--------|------|
| list | POST /rpa/executions/list | ✓ :84 | 仅测试（页面内联 :80） | alive |
| get | POST /rpa/executions/:id | ✓ :87 | 零 | alive |
| statistics | POST /rpa/executions/statistics | ✓ :86 | 仅测试（页面内联 :42） | alive |
| cancel | POST /rpa/executions/:id/cancel | ✓ :88 | 仅 batch56 测试 | alive |
| logs | POST /rpa/executions/:id/logs | ✓ :89 | 仅测试（页面内联 ExecutionDetailModal.tsx:54） | alive |
| create/update/delete/batch/searchOptions | /rpa/executions/* | ✗ | 零 | **dead** |
| streamLogs | POST /rpa/executions/:id/stream-logs | ✗ | 零 | **dead** |
| screenshots | POST /rpa/executions/:id/screenshots | ✗ | 零 | **dead** |
| downloadReport | POST /rpa/executions/:id/report?format= | ✗（后端是 GET /:id/download :90，动宾皆异） | **仅 download.test.ts:225,241** | **dead**（测试同删/改） |
| retry | POST /rpa/executions/:id/retry | ✗ | 零 | **dead** |

#### scheduleApi（14 方法：0 alive / 14 dead）— 整族删除
list/get/create/update/delete/batch/statistics/searchOptions + activate(:339)/pause(:346)/disable(:353)/runNow(:360)/validateCron(:367)/nextRunTime(:378)。无 /rpa/schedules 组；调用方仅 rpaApi.test.ts:114-124 + batch56。

#### variableApi（12 方法：0 alive / 12 dead）— 整族删除
工厂 8 + getGlobal(:405)/getByTask(:412)/batchSet(:419)/decrypt(:426)。无 /rpa/variables 组；调用方仅测试。

#### templateApi（13 方法：0 alive / 13 dead）— 整族删除
工厂 8 + categories(:454)/useTemplate(:461)/rate(:468)/favorite(:475)/unfavorite(:483)。无 /rpa/templates 组；调用方仅测试。

#### aiApi（6 方法：4 alive / 2 dead）

| 方法 | 路径 | 后端 | 分类 |
|------|------|------|------|
| generateScript | POST /rpa/ai/generate | ✓ :101 | alive（唯一潜在生产消费者 AIScriptEditor.tsx:98 是**注释掉**的代码） |
| optimizeScript | POST /rpa/ai/optimize | ✓ :102 | alive |
| decide | POST /rpa/ai/decide | ✓ :105 | alive |
| analyzeFailure | POST /rpa/ai/analyze-failure | ✓ :108 | alive |
| explainScript | POST /rpa/ai/explain | ✗ | **dead** |
| captureState | POST /rpa/ai/capture-state | ✗ | **dead** |

#### notificationApi（12 方法：0 alive / 12 dead）— 整族删除
工厂 8 + enable(:559)/disable(:566)/test(:573)/getGlobal(:580)。无 /rpa/notifications 组；调用方仅测试。注意与 KEEP 例外 `notificationConfigApi`（src/lib/notificationConfigApi.ts，system 模块）完全无关。

#### statisticsApi（5 方法：0 alive / 5 dead）— 整族删除
overview(:594)/tasks(:607)/workers(:623)/executions(:644)/trends(:662)。无 /rpa/statistics 组；调用方仅测试。

#### 聚合导出 `rpaApi`（rpaApi.ts:671-682）
10 个子对象（task/script/worker/execution/schedule/variable/template/ai/notification/statistics）。清理后若 6 族删除 → 剩 4 键；rpaApi.test.ts:192-208「暴露 10 个子 API」断言必须同步改。

### 2.3 端态建议（planner 决策点）

- **方案 A（激进，推荐评估）**：rpaApi 生产消费者为零 → 整文件删除 + download.test.ts 摘除 executionApi 块 + 2 个页面测试的 vi.mock 清理。19 个 alive 方法全部无生产调用方，保留它们只是「能对上路由的死代码」。invariants 锁定目标变为「文件不存在」或空导出。
- **方案 B（保守，符合 D-100-2 字面）**：裁剪至 19 alive 方法集（task 6 / worker 4 / execution 5 / ai 4），invariants keys 基线锁 19 方法。D-100-2 措辞「KEEP baseline 更新为后端路由存在的实测集合」与 B 直接对应。
- 两方案都满足 CONTEXT.md 全局标准「死方法一律删除」；差异仅在 alive-but-zero-caller 方法是否保留。**若无用户裁决，按 B 执行**（D-100-2/5 的守卫措辞以「清理后方法集」为锚，B 是最小解释成本路径）。

### 2.4 vdiApi 对账表（34 方法：22 alive / 6 ghost / 6 mismatch）

| 方法 | 后端路由 | 调用方 | 分类 |
|------|---------|--------|------|
| vmApi.list | ✓ vm_router.go:18 | VirtualMachineList:191 | alive |
| vmApi.get | ✓ :22 | VirtualMachineDetail:47 | alive |
| vmApi.create | ✓ :21 | VirtualMachineList:523,781 | alive |
| vmApi.update | ✓ :23 | 零生产（仅 vdiApi.test.ts:40） | alive |
| vmApi.delete | ✓ :24 | VirtualMachineList:567 | alive |
| vmApi.bindUser | ✓ :32 | VirtualMachineList:631 | alive |
| vmApi.unbindUser | ✓ :33 | 零 | alive |
| vmApi.sync | ✓ :36 | VirtualMachineList:585 | alive |
| vmApi.listResourceGroups / listResources | ✓ :19-20 | List:663,673 | alive |
| vmApi.listVTPPlatforms / listRunPositions / listStorages / listNetworks | ✓ :40-43 | List:148-151,323-343,684-712 | alive |
| vmApi.batch / statistics / searchOptions | ✗ | 零 | **ghost（D-100-1 直接删）** |
| vmApi.operate | ✗ 未注册（handler 在） | 零直接 | **mismatch（§1.3）** |
| vmApi.batchOperate | ✗ 未注册（handler 在） | List:546 | **mismatch（§1.3，补路由首选）** |
| vmApi.listAccounts / createAccount / resetAccountPassword / deleteAccount | ✗（handler/service 均无） | Detail:62,86,108,126 | **mismatch（建议删+Tab 同删）** |
| vdiServerApi.list | ✓ vdi_server_router.go:14 | VDIServerConfig:43 + List:137,469,485,654 | alive |
| vdiServerApi.get | ✓ :16 | 零生产 | alive |
| vdiServerApi.create / update / delete / testConnection | ✓ :15,17-19 | VDIServerConfig:99,112,130,144 | alive |
| vdiServerApi.batch / statistics / searchOptions | ✗ | 零 | **ghost（直接删）** |

---

## 3. networkApi 下载链（V130R-12）

### 3.1 现存平行实现清单（4 份下载链，file:line 实测）

| # | 位置 | 内容 | 与权威的差异 |
|---|------|------|-------------|
| 1 | `src/lib/api/networkApi.ts:14-33` | 私有 `blobAxios`（axios.create + 5min timeout + 请求拦截器注入 token）+ **硬编码默认头 `Content-Type: application/json`**（:17-19） | 权威 download.ts:23-34 无默认 Content-Type 头；其余等价 |
| 2 | `networkApi.ts:183-192` | 私有 `triggerBrowserDownload` | 与权威 download.ts:55-64 逐行等价的重复实现 |
| 3 | `networkApi.ts:142-176`（exportMACHistory，GET）+ `:206-239`（batchExport，POST） | blob 请求 + **自有 JSON 错误嗅探**（`blob.size < 1024 && blob.type 含 "json"` → 解析 message/msg 并 throw，:156-166 / :218-228）+ 自有 Content-Disposition 文件名提取（:167-174 / :229-236） | 权威链**完全没有** JSON 检测；嗅探启发式（size<1024）比 D-100-7 的 content-type 判据弱（>1KB 的 JSON 错误体漏网） |
| 4 | `src/pages/network/mac/history/MACHistoryPage.tsx:377-382` | exportMACHistory 返回 `{blob, filename}` 后，页面**再手写** createObjectURL/a.click/revokeObjectURL 链 | 第 4 份触发链；收敛后应由 download.ts 全托管 |

### 3.2 权威 API 面（src/lib/download.ts，file:line）

- `blobAxios: AxiosInstance`（:23-26）：baseURL `VITE_API_BASE_URL || "/api/v1"`，timeout 300000，**无响应拦截器**（Blob 不可 JSON 解包），请求拦截器异步注入 Bearer（:28-34）
- `extractFilenameFromBlobResponse(response, defaultFilename)`（:37-52）：Content-Disposition 正则 + decodeURIComponent（decodeURIComponent 对坏编码可抛 URIError——deep-recheck Info 已在案，不在本相 scope）
- `triggerBrowserDownload(blob, filename)`（:55-64）
- `downloadFile(url, filename)`（:67-75）：GET + responseType blob，仅 `status` 码 2xx 检查，**无 JSON 错误体检测**
- `downloadFilePost(url, body, defaultFilename)`（:79-92）：POST + responseType blob，仅 status 检查，**无 JSON 错误体检测** ← **D-100-7 修改点**

### 3.3 收敛映射建议（D-100-6）

| 现有 | 收敛后 |
|------|--------|
| networkApi 私有 blobAxios（:14-33） | 删除，`import { blobAxios } from "../download"` |
| networkApi 私有 triggerBrowserDownload（:183-192） | 删除，用权威版 |
| `batchExport`（:206-239） | 内部改走 `downloadFilePost` 链或 blobAxios+权威 helpers；**保留函数签名**（返回实际 filename，9 个页面消费者）——注意 batchExport 需要「返回 filename」语义，`downloadFilePost` 返回 void，收敛时要么扩展 downloadFilePost 返回 filename，要么 batchExport 保留薄壳（blobAxios 请求 + 权威 extract/trigger）。**注意**：D-100-6 说「下载链收敛」而非「函数删除」，薄壳（私有 axios 实例消失、触发/文件名提取全走权威）即达标 |
| `exportMACHistory`（:142-176） | 同上；返回 `{blob, filename}` 契约由 MACHistoryPage.tsx:372 消费——若改用 triggerBrowserDownload 全托管则页面 :377-382 手写链同删（行为变化：从「返回 blob」变「直接触发下载」，页面改动 1 处，属「消费者调用点随对账结果同步处理」授权范围） |
| MACHistoryPage.tsx:377-382 手写触发链 | 随 exportMACHistory 收敛删除 |

### 3.4 downloadFilePost JSON 检测设计（D-100-7）

- 判据：`response.headers["content-type"]` 含 `application/json` → `response.data`（Blob）`.text()` → `JSON.parse` → throw `Error(errBody.message || ...)`。优先生效于 `response.status < 300` 分支**之前**（后端错误封装常配 200 或 4xx + JSON body；axios 默认 4xx 会 reject，但 `validateStatus` 未定制时 200+JSON 错误体是主漏网场景）
- 后端错误 envelope 参照：`{code, message, data, ...}`（CLAUDE.md API Response Format）——解析 `message` 字段
- 回归测试落点：`src/lib/download.test.ts`（既有 mock 基建完整：axios create 工厂 mock :26-46、blobAxios 打桩 :57-61）——新增 2 用例：① content-type=application/json + 错误体 → rejects.toThrow；② content-type=流 + blob → 正常触发下载（防检测误伤）
- 参照先例：networkApi 现有 CR-01 嗅探测试若存在于 networkApi.test.ts——实测该文件**无** export/batchExport 的 JSON 嗅探用例（networkApi.test.ts 只测 post 类函数），收敛后嗅探逻辑随迁移自然消失

### 3.5 invariants 递归扫描（D-100-8）

- 现状：`apiFactory.invariants.test.ts:124-128` `listApiFiles()` 用 `readdirSync(LIB_DIR)` **非递归**枚举 `src/lib` 顶层；`:239-241` 断言 `files` 与写死的 `EXPECTED_FILES`（:72-86，13 项）逐项相等
- 逃逸者：`src/lib/api/networkApi.ts` + `src/lib/api/macHeatmapApi.ts`（glob `src/lib/**/*Api*.ts` 实测仅此两文件在子目录）
- 递归化改动面：① `readdirSync(LIB_DIR, { recursive: true })`（Node ≥20.1 支持；本机 Node v24.19.0 ✓）——注意 Windows 下返回路径含 `\` 分隔符，**排序/比较前需统一 `replaceAll(sep, "/")`**，否则 EXPECTED_FILES 等值断言 Windows-only 红；② EXPECTED_FILES 13→15（`api/networkApi.ts`、`api/macHeatmapApi.ts`）；③ WARNING_WHITELIST 补 2 个显式登记（两文件实测 **0 命中**：networkApi 的 write* 包装全是 plain-literal URL 不含 CRUD 后缀、export/batchExport 是多语句 blobAxios 体不满足单 return 口径；macHeatmapApi 单函数 plain-literal URL）——invariants 档位规则「每个非硬档文件都必须显式登记」要求 0 也必须登记
- 语义影响：递归后 networkApi 进入扫描面 = D-100-6 收敛后被扫描守卫覆盖（逃逸口封堵的真正意义）

---

## 4. Test/Lint Impact Assessment

### 4.1 现存测试与 rpaApi/vdiApi/networkApi 的耦合（删除方法会编译失败的文件）

| 测试文件 | 耦合内容 | 处置 |
|---------|---------|------|
| `src/lib/rpaApi.test.ts`（209 行） | 锁定全部 dead 端点 URL（:30-188）+「10 个子 API」结构断言（:192-208） | 重写为清理后契约基线（方案 B：19 方法 keys + alive URL；方案 A：整文件删除） |
| `src/lib/__tests__/rpaApi.batch56.unit.test.ts`（83 行） | 调用 scriptApi/scheduleApi/variableApi/templateApi/notificationApi/statisticsApi 的 list 等（:37-82）——被删对象 | 随族删除重写或整文件删除 |
| `src/lib/download.test.ts` | :54 导入 executionApi；:217-250 downloadReport 两用例（POST /rpa/executions/:id/report） | downloadReport 判 dead 后摘除该 describe 块 + import；**同时**此处是 D-100-7 新用例的落点 |
| `src/lib/vdiApi.test.ts`（157 行） | operate/batchOperate（:50-64）、accounts 族（:118-135） | 按 §1.3 裁决同步：补路由则保留并加权限无关断言；删 accounts 则摘 :118-135 |
| `src/pages/vdi/VirtualMachineDetail/__tests__/index.render.test.tsx` | `vi.mocked(vmApi.listAccounts)`（:60,68） | accounts 删除则同步摘 mock 与相关用例 |
| `src/pages/vdi/VirtualMachineList/__tests__/index.render.test.tsx` | 仅 mock vmApi.list（:48,67） | 不受影响（batchOperate 删/补路由均不触） |
| `src/pages/operations/rpa/{tasks,workers}/__tests__/index.test.tsx` | `vi.mock("@/lib/rpaApi", factory)`（:16-22），factory 方法名在 rpaApi 中**不存在**（getRPATaskList 等） | 空转 mock，rpaApi 变更后依然合法；建议顺手清理（非必须） |
| `src/lib/api/__tests__/networkApi.test.ts` + `networkApiBatch.unit.test.ts` | 只测 post 类函数（query/write*/bundle），不触下载链 | 收敛不动 post 函数则零影响；若删 exportMACHistory 需 grep——实测两文件**无** export 用例，零影响 |
| `src/lib/apiFactory.invariants.test.ts` | HARD_ALLOWED（rpaApi:0 期望不变——删除不产生新模板）/ EXPECTED_FILES / WARNING_WHITELIST | D-100-2/5/8 的主改造点（§3.5） |

### 4.2 前端 gate 命令（精确）

```bash
cd xingran-react-frontend
npm run lint          # eslint . —— 0 errors / 1389 warnings 基线（存量只许降）
npm run type-check    # tsc --noEmit -p tsconfig.app.json
npx vitest run        # 554 files / 3800 tests 基线（npm run test 在 TTY 下是 watch 模式，CI 用 run）
# 覆盖率 gate（45 dirs ratchet）：
npm run test:coverage
bash ../.github/scripts/check-frontend-coverage.sh coverage/coverage-final.json ../.coverage-fe-floors
# 后端（仅当 §1.3 补路由裁决为「补」）：
cd .. && go build ./... && go test ./internal/api/v1/... ./internal/services/vdi/...
```

**覆盖率 ratchet 风险**：`.coverage-fe-floors` 中 `lib 87.2` / `pages/vdi 30.4` / `pages/operations 39.4` 为下限 ratchet。删除**被测试覆盖**的死代码（rpaApi 测试覆盖其全部方法）会同时缩分子/分母，lib 实测值方向不可预判——删除后**必须实跑 coverage gate**；若 lib 跌破 87.2，随删的 rpaApi.test.ts/batch56 测试文件也在 include 口径内（测试文件计入 src/**），净效应需实测。floors 只升不降（D-07），跌破即需补测或调整删除粒度。

## 5. Architecture Patterns

### System Architecture Diagram（对账数据流）

```
前端调用面                       后端真相源
─────────                       ─────────
RPA 页面(tasks/executions/workers)──post()内联──▶ /api/v1/rpa/{tasks,executions,workers}/...  ✓ 已对齐
rpaApi.ts (116 方法) ──┬─ 19 方法 ──────────────▶ 已注册路由（无生产消费者）
                       └─ 97 方法 ──▶ ✗ 404（无路由）──▶ V130R-10/11: 删 97（0 mismatch in rpa）
VDI 页面(vmApi 22 方法) ──▶ /api/v1/vdi/vms|servers/... ✓ 已对齐
vmApi.batchOperate ──▶ /vdi/vms/operate ──▶ handler 在但未注册 ──▶ 裁决点：补路由 1 行（推荐）/删
vmApi.accounts×4 ──▶ /vdi/vms/:id/accounts ──▶ 后端零实现（现状 404）──▶ 删 + Detail Tab 同删
networkApi 下载链(×4 份) ──▶ 收敛 ──▶ download.ts（blobAxios/downloadFile/Post + JSON 检测新增）
invariants AST 扫描(readdirSync 非递归) ──▶ 漏 src/lib/api/*.ts ──▶ 递归化 13→15 文件
```

### Pattern 1: keys 基线锁（D-100-2/D-100-5 落地形态）

AST 扫描管「手写 CRUD 模板」，方法集回增要用**对象 keys 等值断言**管：

```typescript
// 追加到 apiFactory.invariants.test.ts（或新文件 rpaVdiContract.test.ts）
// Source: D-100-2「KEEP baseline 更新为后端路由存在的实测集合」
import { taskApi, workerApi, executionApi, aiApi } from "./rpaApi";
import { vmApi, vdiServerApi } from "./vdiApi";

const POST_CLEANUP_BASELINE: Record<string, string[]> = {
  taskApi: ["create", "delete", "execute", "get", "list", "update"], // 全部有后端路由（§2.2）
  workerApi: ["heartbeat", "list", "register", "statistics"],
  executionApi: ["cancel", "get", "list", "logs", "statistics"],
  aiApi: ["analyzeFailure", "decide", "generateScript", "optimizeScript"],
  vmApi: [/* 清理后实测集 */],
  vdiServerApi: ["create", "delete", "get", "list", "testConnection", "update"],
};
// 每个对象：expect(Object.keys(obj).sort()).toEqual(baseline) —— 新增方法即红
```

### Pattern 2: gin 生产路由注册（若 operate 补路由裁决为「补」）

```go
// internal/api/v1/vdi/vm_router.go —— 比照同文件电源操作权限粒度（:27-29）
r.POST("/operate", middleware.RequirePermissions([]string{"vdi:vm:edit"}, core), vmHandler.Operate)
```
handler 已含 swagger 注解（vm_handler.go:203）；vdi 组已有 OperLogMiddleware 组级挂载（router.go:924）；rpa 包的 route-count smoke 测试用 `>=` 断言（router_public_test.go:270,289），加路由不红；vdi 包无 route-count 锁（grep 实证）。

### Anti-Patterns to Avoid

- **保留空壳方法**：D-100-1 明令禁止；删就删导出 + 类型 + 测试三件套
- **为「对账完整性」给 dead 方法补后端路由**：rpaApi 97 个 dead 方法对应的功能（脚本管理/调度/变量/模板/通知/统计）后端从未实现——补路由 = 凭空造 6 个模块，直接违反 ROADMAP D-01「不引入新业务功能」
- **在 invariants 里用 grep 式字符串匹配做 keys 锁**：AST/运行时对象 keys 均可，正则扫源码会对注释/类型名误报
- **把 networkApi 的 size<1024 嗅探搬进 download.ts**：D-100-7 锁定 content-type 判据；size 启发式是弱实现，收敛时以强判据替换而非搬运

## 6. Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| POST 文件下载 | 每模块私有 axios 实例 + 手写 a/click 链 | `download.ts` blobAxios + downloadFile/Post（Phase 94 D-04 权威） | token 注入/超时/文件名提取已统一；4 份平行实现正是本相要消除的缺陷 |
| 200+JSON 错误体识别 | blob.size 阈值启发式 | content-type 判据（D-100-7 锁定） | >1KB 错误体漏网；content-type 是协议级信号 |
| 递归目录枚举 | 手写递归栈 | `readdirSync(dir, { recursive: true })` | Node ≥20.1 原生（本机 v24.19）；手工栈徒增 Windows sep 分隔符 bug 面 |
| 方法集防回增 | 每次人工 grep 调用面 | invariants keys 基线等值断言 | 双向锁（回增/私删都红），与既有 hard/warning tier 同构 |

**Key insight:** 契约漂移的根因是「前端方法集没有机器可查的后端路由对照物」。本相的三重守卫（keys 基线 + 递归扫描 + RECONCILIATION.md 台账）把对照物固化为测试资产，而不是靠一次人工对账。

## 7. Runtime State Inventory

> 本相为前端 lib 层删除/收敛 + 可能的 1 行后端路由注册，无重命名/数据迁移。逐类排查结论：

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None——删除的 API 方法不落库；无缓存键/集合名涉及（后端 /rpa/scripts 等路由从未存在，无历史数据） | none |
| Live service config | None——前端无 n8n/调度类外部注册；menu 表若配置了已删页面的路由由现有页面路由承载（RPA 三页面保留，不受影响） | none |
| OS-registered state | None——纯 web 前端 | none |
| Secrets/env vars | None——不新增/删除环境变量（VITE_API_BASE_URL 沿用） | none |
| Build artifacts | None——Vite 无持久构建产物耦合；`node_modules/` 存在于仓库根（untracked，与 worktree junction 地雷相关，本相不触） | none |

## 8. Common Pitfalls

### Pitfall 1: 测试与实现必须同一提交内原子删除
**What goes wrong:** 删 `scriptApi` 导出但 rpaApi.batch56.unit.test.ts 仍 import → `npm run type-check` / vitest 编译期即红。
**Why it happens:** 死方法的引用面一半在测试文件里（rpaApi 97 个 dead 方法中 20+ 个仅被测试引用）。
**How to avoid:** 每个删除 task 的验收命令必须含 `npx vitest run src/lib` + `npm run type-check`；测试改造条目已列于 §4.1，逐文件进 plan。
**Warning signs:** `TS2305: Module '"./rpaApi"' has no exported member`。

### Pitfall 2: Windows 路径分隔符击穿 EXPECTED_FILES 等值断言
**What goes wrong:** `readdirSync(recursive)` 在 Windows 返回 `api\networkApi.ts`，与字面量 `"api/networkApi.ts"` 比较失败——本地红、CI（Linux）绿或反之。
**How to avoid:** 枚举后立即 `path.normalize` 统一为 `/` 再 sort/比较；等值断言用 POSIX 形式字面量。
**Warning signs:** invariants 测试「范围恰为 13/15 个文件」用例平台相关失败。

### Pitfall 3: 覆盖率 ratchet 反向击穿
**What goes wrong:** 删除被测试高覆盖的死代码后 `lib` 目录实测覆盖率跌破 87.2 下限，coverage gate 红。
**How to avoid:** 见 §4.2 风险段——删除 plan 内置 `test:coverage` + check 脚本步骤；先删后测，跌破则把 rpaApi 存活方法的契约测试（方案 B 的 rpaApi.test.ts 重写版）计入分母平衡。
**Warning signs:** `check-frontend-coverage.sh` 输出 `lib 86.x < 87.2`。

### Pitfall 4: 名字相近对象误删/误改
**What goes wrong:** `notificationApi`（rpaApi.ts:552，整族 dead）与 KEEP 例外 `notificationConfigApi`（src/lib/notificationConfigApi.ts，system 模块、工厂 KEEP 名单在册）名字相近；`statisticsApi`（rpa）与各模块 `/statistics` 路由；`workerApi.statistics` 覆盖了工厂 statistics。
**How to avoid:** 所有 grep 带 `/rpa/` 或文件路径 scope；§2 表已按对象逐一定位。
**Warning signs:** diff 触及 notificationConfigApi.ts / noticeApi.ts。

### Pitfall 5: downloadReport 的测试锁与 downloadFilePost 复用混淆
**What goes wrong:** 删 executionApi.downloadReport 时误删 `downloadFilePost` 本身（它在 rpaApi 内的唯一消费者就是 downloadReport）。
**How to avoid:** `downloadFilePost` 有 opsApi.ts:265（excelApi.export）+ :561（asset export）两个活消费者 + D-100-7 要在其上新增检测——**函数保留**，只删 rpaApi 的调用点。
**Warning signs:** opsApi excel 导出用例红。

### Pitfall 6: vi.mock 空转工厂在模块删除后的行为差异
**What goes wrong:** 若 rpaApi.ts 整文件删除（方案 A），页面测试 `vi.mock("@/lib/rpaApi", factory)` 因真实模块不存在且无人 import 而成为纯噪音——vitest 不会报错（factory mock 不解析真实模块），但会误导后续读者以为存在依赖。
**How to avoid:** 方案 A 执行时顺手摘除两处 vi.mock 块（tasks/workers 页面测试 :16-22）。
**Warning signs:** 无（纯可读性问题）。

## 9. Code Examples

### 后端路由存在性对照（对账方法论示例）

```bash
# 前端方法 URL 提取（rpaApi.ts 内全部模板/字面量 URL）
grep -n "post[<(]\`*[\"']/rpa/" xingran-react-frontend/src/lib/rpaApi.ts
# 后端注册表提取（gin 路由注册行）
grep -rn 'r\.\(POST\|GET\)("' internal/api/v1/rpa/rpa_router.go
# 全仓否定验证（六组路由确实无处注册）
grep -rn 'Group("/\(scripts\|schedules\|variables\|templates\|notifications\|statistics\)")' internal/
```

### downloadFilePost JSON 检测（D-100-7 目标形态）

```typescript
// src/lib/download.ts:79 扩展（示意，最终以 plan 为准）
export async function downloadFilePost(
  url: string, body: unknown, defaultFilename: string
): Promise<void> {
  const response = await blobAxios.post<Blob>(url, body, { responseType: "blob" });
  if (response.status < 200 || response.status >= 300) {
    throw new Error(`下载失败: ${defaultFilename}`);
  }
  // D-100-7: 200 + application/json = 后端把错误体伪装成下载
  const contentType = String(response.headers["content-type"] ?? "");
  if (contentType.includes("application/json")) {
    const text = await response.data.text();
    let message = "下载失败";
    try { message = (JSON.parse(text) as { message?: string }).message || message; } catch { /* 非法 JSON 保留默认 */ }
    throw new Error(message);
  }
  const filename = extractFilenameFromBlobResponse(response, defaultFilename);
  triggerBrowserDownload(response.data, filename);
}
```

## 10. State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 各模块私有 createCrudApi（5 方法） | `src/lib/apiFactory.ts` createResourceApi（8 方法）单一权威 | v1.29 Phase 94 | spread 实例新增 batch/statistics/searchOptions → 幽灵面（本相 V130R-10 清理对象） |
| opsApi 内联 POST-blob 下载链 | `download.ts` blobAxios + downloadFile/Post | Phase 94 D-04 | networkApi 的第 2 份实现成为违建（V130R-12 收敛对象） |
| gin 静态+参数同级路由 panic | gin ≥1.7 支持静态优先混合匹配 | 上游 | 后端 /tasks 下 `/list`、`/upload-excel`（静态）与 `/:id`（参数）共存合法；前端 `validate-script` 仍因未注册而 404 |

**Deprecated/outdated:**
- `rpaApi.test.ts` 的「契约测试」定位：它锁的是**前端单方面声明的 URL**，而非前后端契约——本相重写后应锁「清理后实测存活集」
- 页面测试对 `@/lib/rpaApi` 的 vi.mock：mock 的方法名（getRPATaskList 等）自始不存在，属 Phase 88 时代误植

## 11. Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | V130R-10「8 处/2 处」= 8/2 个 createResourceApi spread 站点（与代码实测精确吻合：rpaApi 8 个 createResourceApi 调用、vdiApi 2 个） | §1.1 | 低——即使原意是「8 个方法」，§1.2 已给出逐方法全清单，覆盖任意解释 |
| A2 | rpaApi 端态默认取方案 B（裁至 19 alive 方法）而非整文件删除；方案 A 需用户/planner 追加裁决 | §2.3 | 低——两方案均满足 D-100 全部条目，差异仅 alive-but-zero-caller 方法去留 |
| A3 | `/vdi/vms/operate` 裁决倾向「补路由」；accounts 族裁决倾向「删」。二者均为 mismatch 默认删规则（D-100-3）的例外建议，**需 plan/用户确认** | §1.3 | 中——补路由扩大后端攻击面（新增 1 个写端点），删 accounts 移除用户可见（已坏）功能 Tab |
| A4 | 测试基线「554 文件 / 3800 tests」「lint 1389 warnings」沿用 CONTEXT.md 记载，未在本会话实跑复核 | §4.2 | 低——gate 是「不倒退」，以执行时实跑值为准 |
| A5 | frontend axios baseURL 前缀为 `/api/v1`，与后端挂载组前缀一致（由 download.ts:24、networkApi.ts:15 与 api_v1_tail 测试三方互证） | §2.1 | 极低 |

## 12. Open Questions (RESOLVED 2026-09-06 — 裁决见 100-CONTEXT.md「RESEARCH 裁决补充」)

1. **rpaApi.ts 端态：整文件删除（A）还是裁至 alive 集（B）？** — **RESOLVED: B（D-100-9）**，裁至 alive 集（后按 D-100-12 再减 register/heartbeat = 17 端态）；页面内联 post 记 observed debt。
2. **`/vdi/vms/operate` 补路由 vs 删批量操作？** — **RESOLVED: 补路由（D-100-10）**，后端实现齐全仅缺注册，属接线缺陷修复非新功能。
3. **VM accounts Tab 删除的用户确认** — **RESOLVED: 删（D-100-11）**，运行时 100% 404 保留即反模式；OVR 台账随 RECONCILIATION.md 登记。
4. **workerApi.register/heartbeat（alive、零 UI 调用方、Worker 节点专用公开端点）去留？** — **RESOLVED: 前端方法删除（D-100-12）**，后端公开路由保留不动。

## 13. Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | 前端 gate 全部命令 | ✓ | v24.19.0 | — |
| vitest | 单测/回归测试 | ✓（node_modules 就绪） | ^4.0.18（package.json） | — |
| typescript | type-check + invariants AST 扫描 | ✓ | ~5.9.3（devDependencies，零新装） | — |
| Go | 后端补路由（仅 A2/Q2 裁决为补时） | ✓ | go1.24.5 windows/amd64 | — |
| npm legacy-peer-deps | 干净安装复现 | ✓（.npmrc 在档，直接 import 包必须显式进 package.json——本相零新装，无影响） | — | — |

**Missing dependencies with no fallback:** None。

## 14. Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.0.18（jsdom, globals, testTimeout 15s, maxWorkers 4） |
| Config file | `xingran-react-frontend/vitest.config.ts`（coverage include `src/**/*.{ts,tsx}`，gate 移交外部脚本） |
| Quick run command | `cd xingran-react-frontend && npx vitest run src/lib/apiFactory.invariants.test.ts src/lib/download.test.ts src/lib/rpaApi.test.ts src/lib/vdiApi.test.ts` |
| Full suite command | `cd xingran-react-frontend && npx vitest run`（基线 554 files / 3800 tests） |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| V130R-10 | 幽灵工厂方法（batch/statistics/searchOptions 等 54+6 实例）不再出现在 rpa/vdi 对象上 | unit（keys 基线锁，新增断言） | `npx vitest run src/lib/apiFactory.invariants.test.ts` | ❌ Wave 0（新增 describe 块/文件） |
| V130R-11 | rpaApi 清理后方法集 == RECONCILIATION.md 实测存活集；死方法回增即红 | unit（keys 基线锁 + rpaApi.test.ts 重写为存活契约） | `npx vitest run src/lib/rpaApi.test.ts` | ⚠️ 存在但需重写（锁的是 dead URL） |
| V130R-12① | networkApi 下载链不再持有私有 blobAxios/trigger 副本（D-100-6） | unit（invariants 递归扫描 + 可选 source-string 断言） | `npx vitest run src/lib/apiFactory.invariants.test.ts` | ❌ Wave 0（递归化 + EXPECTED_FILES 15） |
| V130R-12② | downloadFilePost 对 200+JSON 错误体 throw、对 blob 正常下载（D-100-7） | unit（mock blobAxios.post 两分支） | `npx vitest run src/lib/download.test.ts` | ⚠️ 存在，新增 2 用例 |
| V130R-12③ | invariants 递归枚举 == 15 文件（含 api/ 子目录 2 文件，各显式登记 0） | unit | `npx vitest run src/lib/apiFactory.invariants.test.ts` | ⚠️ 存在，EXPECTED_FILES 扩容 |
| 全局 | 七 gate 不倒退 | gate | `npm run lint && npm run type-check && npx vitest run` + coverage 脚本（§4.2） | ✓ 既有 |

### Sampling Rate
- **Per task commit:** quick run（上述 4 文件）+ `npm run type-check`
- **Per wave merge:** `npx vitest run` 全量 + lint（warnings ≤1389）
- **Phase gate:** 全量 vitest + coverage 脚本（lib 87.2 / pages/vdi 30.4 / pages/operations 39.4 floors）+ 若含后端补路由则 `go build ./... && go test ./internal/...`

### Wave 0 Gaps
- [ ] invariants keys 基线断言（V130R-10/11 守卫）——新增于 apiFactory.invariants.test.ts 或独立 `src/lib/rpaVdiContract.test.ts`
- [ ] download.test.ts 的 JSON/blob 双分支用例（V130R-12②）
- [ ] invariants 递归化改造本身（EXPECTED_FILES 13→15 + Windows sep 归一）——它同时是 V130R-12③ 的交付与其余守卫的运行前提
- 框架安装：无需（vitest/typescript 已在 devDependencies）

## 15. Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | —（不触认证） |
| V3 Session Management | no | — |
| V4 Access Control | **yes（条件）** | 仅当 Q2 裁决补路由：`middleware.RequirePermissions(["vdi:vm:edit"])` 比照 vm_router.go:23-29 粒度，**禁止裸注册**（vm_handler_test.go 的本地测试路由不带权限，照抄即事故） |
| V5 Input Validation | no（新增面） | 后端 Operate handler 已有 gin binding；前端纯删除 |
| V6 Cryptography | no | — |

### Known Threat Patterns for 本相改动面

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 幽灵方法 ≠ 安全漏洞但扩大信息暴露面（API 面泄露后端不存在的能力图谱） | Information Disclosure | 全量删除（D-100 全局标准「不保留兼容导出」） |
| 补路由端点无权限校验（照抄 handler 测试的裸注册） | Elevation of Privilege | RequirePermissions 强制 + vdi 组 OperLogMiddleware 组级审计已在位（router.go:924） |
| 200+JSON 错误体存成 .xlsx（用户拿到伪造成功文件） | Tampering / 防欺骗 | D-100-7 content-type 检测 + 双分支回归测试 |
| Worker register/heartbeat 公开端点包装器残留在 admin bundle | Information Disclosure | 可选删除（Open Question Q4；路由本身是设计内公开端点，不受本相影响） |

## 16. Sources

### Primary (HIGH confidence — 本会话 file:line 实测)
- `xingran-react-frontend/src/lib/rpaApi.ts`（全 683 行）/ `vdiApi.ts`（173 行）/ `apiFactory.ts` / `apiFactory.invariants.test.ts` / `download.ts` / `api/networkApi.ts`（487 行）/ `api/macHeatmapApi.ts`
- `internal/api/v1/rpa/rpa_router.go`（全 155 行）/ `internal/api/v1/vdi/vm_router.go` / `vdi_server_router.go` / `vm_handler.go`（handler 清单 grep）/ `internal/api/router.go`（:920-968 挂载段）
- `internal/api/v1/network/network_router.go:237`（batch-export）/ `mac_history_router.go:23-29`（history 路由）
- 测试文件：`rpaApi.test.ts` / `__tests__/rpaApi.batch56.unit.test.ts` / `download.test.ts` / `vdiApi.test.ts` / `api/__tests__/networkApi*.test.ts` / 页面测试 5 个
- 消费者 grep 全量：`rpaApi|vmApi\.|vdiServerApi\.|from.*vdiApi|/rpa/|downloadFilePost\(|exportMACHistory|batchExport`
- `.planning/phases/100-frontend-contract-fixes/100-CONTEXT.md` / `.planning/workstreams/milestone/ROADMAP.md` / `.planning/REQUIREMENTS.md` / `.planning/milestones/v1.29-DEEP-RECHECK.md`
- `package.json` / `vitest.config.ts` / `.npmrc` / `.coverage-fe-floors` / `.github/scripts/check-frontend-coverage.sh`（存在性）

### Secondary (MEDIUM confidence)
- 无（未使用 WebSearch——本相全部结论可源码实测，无需外部资料）

### Tertiary (LOW confidence)
- None

## 17. Package Legitimacy Audit

本相**零新增外部包**（递归扫描用 Node 原生 `readdirSync`，AST 用既有 devDependencies 的 typescript，无任何 npm install 步骤）。协议触发了但无对象——上表按空处理，无需 planner 插入 checkpoint。

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Metadata

**Confidence breakdown:**
- 对账清单（§1/§2/§3）: HIGH — 每行均 file:line 实测，后端否定性结论经全仓双重 grep（Group 注册 + 直接 POST 注册）互证
- 端态建议（§2.3/§12）: MEDIUM — 方案选择含产品语义判断（A2/Q1-Q4），需 planner/用户裁决
- Pitfalls/Validation: HIGH — 基于本仓既有测试基建与 floors ratchet 机制实测

**Research date:** 2026-09-06
**Valid until:** 2026-10-06（30 天；若 main 分支 rpaApi.ts / vm_router.go / download.ts 有后续提交需复核行号）
