---
created: 2026-11-23
topic: v1.56 Phase 152-158 并行性实证裁决（文件集交集）
status: decision（已回写 ROADMAP 依赖拓扑）
method: 3 个只读 subagent 分别扫描后端 / rpa-worker / 前端，全部结论带 文件:行号 取证
related:
  - .planning/ROADMAP.md
  - .planning/REQUIREMENTS.md
---

# v1.56 Phase 并行性裁决（实证）

> 判据：**同文件同函数（或同 export 同一字面量数组）= 硬冲突（不可并行）**；**同文件不同函数/不同 export = 软冲突（理论上可并行，但 GSD execute-phase 的 `files_modified` 重叠检测会强制降级为串行）**；**无交集 = 可并行**。

## 1. 三路扫描结论摘要

### 后端（152 / 153 / 158 + 155/156/157 的根模块侧）

| pair | 共享文件 | 共享符号 | 风险 | 依据 |
|---|---|---|---|---|
| 152×153 | `internal/services/rpa/task_service.go` | 152 碰 `Create:81`/`Update:106`/`List:157`；153 碰 `publishTaskToRedis:243`/`extractTargetSystem:372` | 软（函数不相交） | 行号不重叠 |
| 152×158 | `internal/api/v1/rpa/rpa_router.go` | 152 改 `NewServiceGroup:24,:51`；158 尾部追加 Schedule 路由 | 软 | 同文件不同函数 |
| 153×158 | `internal/services/rpa/excel_service.go`（**条件性**） | 153 的 RUNTIME-04 publish 若落 `ProcessHumanIntervention:331`；158 重写批量报告 `:219-284` | 软，可规避 | 153 把 publish 落 `execution_handler.go:238`（`h.core.Cache` 可达）则**零交集** |
| 152×157 | 无 | — | 无交集 | 152 不碰 `scaling_service` / `RPAScalingConfig` |
| 153×157 | 无 | — | 无交集 | 157 的 provider 键在**根模块** `internal/config/config.go:259`，rpa-worker Go 代码 157 基本不碰 |

**152 的实际必需写入面不含** `execution_service.go` / `worker_service.go`（两者 List 用通用 `*params.Status` 解引用，改类型即自动生效）。

### rpa-worker（153 / 154 / 155 / 157）

| pair | 共享文件 | 共享符号 | 风险 | 依据 |
|---|---|---|---|---|
| **154×155** | `browser/pool.go`、`chrome_page.go`、`page_manager.go`、`executor/engine.go`、`worker/worker.go`、`config/config.go` | `Page` 接口 `pool.go:14` **无 `GetPage` 出口**，155 必需的 `*rod.Page` 恰在 154 重写的文件里；`Engine.Execute:55` 的 Acquire 路径与接管暂停点同区域；`Worker` struct `:38` 两侧都加字段 | **硬冲突** | 三处同文件同函数 |
| 153×154 | `executor/engine.go`、`worker/worker.go`、`communication/*` | 153 碰 `applySessionData:715`/`executePause:425`/`subscribeForHumanInput:496`；154 碰 `executeAction:209`；worker.go 153 加取消订阅、154 改 `initComponents:152` | 软（函数不相交，但同文件） | 行号不重叠 |
| 154×157 | `rpa-worker/configs/config.yaml`（`max_pages:50` vs `max_concurrency:6`） | 键不同但语义被 ENGINE-01 合并 | 软 | 157 的 CI 漂移校验须兼容「旧键保留+告警」中间态 |
| 155×157 | `configs/config.yaml`、`deployments/docker-compose.yml` | 无共享 Go 符号 | 软 | 改动区隔明显 |

**动作缺口实证**（ENGINE-02 的直接输入）：

```
types.go 常量 17 个（:50-66）；engine.go case 12 个（:221-320）
差集 A（常量已声明、switch 无 case → 执行即 unsupported）：
  upload:58  download:59  evaluate:60  close:62  condition:65
差集 B（常量与 case 双缺）：
  assert hover doubleClick rightClick drag keyboard switchTab iframe
可复用原语：ChromePage.Evaluate:236 / PageManager.Evaluate:103
```

### 前端（152 / 153 / 155 / 156）

| pair | 共享文件 | 共享符号 | 风险 | 依据 |
|---|---|---|---|---|
| **153×155** | `executions/ExecutionDetailModal.tsx` | footer 数组 `:317-329` —— 取消按钮与「实时画面」入口**同一个数组字面量同一 render 内插入** | **硬冲突** | 同数组 |
| 152×153 | `executions/index.tsx`、`types/rpa.ts` | 152 独占轮询块 `:105-135`；153 独占 handler 区 `:135+`；types 不同段 | 软 | 函数级归属可拆 |
| 152×155 | `types/rpa.ts` | 152 改既有枚举段；155 EOF 追加 | 软 | 不同段 |
| **152×156** | `tasks/modals/AIScriptEditor.tsx` | 152 改 `handleGenerate:90`/删 `mockGeneratedActions:43`；156 加「AI 不可用降级为录制」入口 —— **同组件相邻区域且语义依赖 152 的真实失败态** | **软偏硬** | 建议串行 |
| 153×156 | `types/rpa.ts` 不同段 | 无符号重叠 | 近乎无交集 | — |
| **155×156** | `live/*` 画布（156 的拾取器落其上）、`types/rpa.ts` 双 EOF 追加 | 结构依赖 + 追加段竞争 | 软 + **结构依赖** | 155 必须先于 156 |

**前端路由机制实证**：`src/router/componentLoader.tsx:36` 的 `import.meta.glob("/src/pages/**/{index,detail}.tsx")` 会自动收集 `live/index.tsx` 与 `wizard/index.tsx`，**目录本身无需手工注册**；但「有没有路由可达」取决于后端 `sys_menu` 节点 —— 若无菜单，需走 `DynamicRoutes.tsx` 静态导入 + glob 排除的 notice/detail 先例，此时 155 与 156 会共享 `componentLoader.tsx` + `DynamicRoutes.tsx` 构成硬冲突。

## 2. 裁决：推荐执行波次

```text
W1  152 ∥ 154          （P0 契约收拾 vs Worker 引擎重写；零文件交集，仅前端 constants.tsx/types 不同 export 软碰）
W2  153 ∥ 157          （执行闭环 vs VDI 承载；后端零交集、worker 侧 157 不碰 Go）
W3  155                （依赖 154 的 page 池与 RawPage 出口 + 153 的暂停/接管状态机）
W4  156                （依赖 155 的画布拾取器 + 154 的选择器阶梯）
W5  158                （收尾：依赖 153 状态机、155 审计、157 provider 配置）
```

与 ROADMAP 原拓扑的差异与理由：

| 原 ROADMAP | 实证裁决 | 理由 |
|---|---|---|
| `152 → 153 ∥ 154` | **`152 ∥ 154`** | 154 是 worker-only，与 152 的后端/前端写面零交集；把 154 提前一个波次可让 155 的阻塞期整体前移 |
| `155 → 156 → 157` | **`153 ∥ 157` 提前** | 157 只改 `scaling_service`/`docker_client`/根 `RPAScalingConfig`/部署产物，与 153 完全无交集；157 的 FLEET-02 参数表需 154 的配置语义定稿，故放 W2（154 已合入） |
| `153 ∥ 154` 相邻 | **拆到 W1/W2** | 两者在 `engine.go`/`worker.go` 同文件（软冲突），GSD 的 `files_modified` 重叠检测会强制串行；分波后整段无冲突 |

## 3. 必须补的 PLAN 边界协议（否则并行会退化为串行或被 merge 冲突拖垮）

1. **`task_service.go` 归属**：152 独占 `:81-200`（Create/Update/List），153 只碰 `:243+`（publishTaskToRedis/extractTargetSystem）。152 先行合入，153 rebase。
2. **`rpa_router.go` 归属**：152 独占 `:22-52`（两处 NewServiceGroup）；158 只在 `SetupRPARouter` 尾部追加 Schedule 组（`SetupScheduleRouter`），不改既有行。
3. **RUNTIME-04 publish 落点**：定为 `execution_handler.go:238 SubmitHumanIntervention`（用 `h.core.Cache`），**不落** `excel_service.go` —— 这是 153 与 158 解耦的唯一手段。
4. **前端 footer 归属**：取消按钮划给 153 但落在 `executions/columns.tsx:106-118` 行内 actions + `executions/index.tsx:135` handler（**不进 Modal footer**）；「实时画面」入口划给 155 落 Modal footer `:317-329`。二者因此不再同数组。
5. **155 的接口前置**：154 必须在 W1 内先落并冻结 `browser.Manager` 契约（`Acquire(ctx) (Page, error)` / `Release` / `Page.RawPage() *rod.Page` / 「Close 只关 page」语义），155 只依赖该接口 + 新增 `internal/communication` 帧通道；`types.go` 的 live 消息类型放独立文件（如 `types/liveview.go`）避免与 154 的动作常量同文件。
6. **路由策略统一**：155 与 156 的新页面**统一走后端 `sys_menu` seed migration**（各自动各自 migration 文件），前端零 router 改动 —— 否则两相共享 `componentLoader.tsx` + `DynamicRoutes.tsx`。
7. **`ACTION_TYPE_OPTIONS` 归属**：扩充的真实上游是 **154**（ENGINE-02），156 只能消费，不得作为第三写方。

## 4. ROADMAP 文件面的漏项（PLAN 阶段必须补齐）

| Phase | 漏登的文件/符号 | 依据 |
|---|---|---|
| 153 | 前端 `executions/columns.tsx:106-118`（行内取消入口）、`executions/index.tsx:135`（handler） | RUNTIME-03「前端执行页提供取消入口」 |
| 155 | 前端 `executions/ExecutionDetailModal.tsx:317-329`（实时画面入口） | LIVEVIEW-04「可从执行详情跳转」 |
| 156 | 前端 `tasks/index.tsx:212-223`（新增按钮改开向导）、`tasks/modals/EditModal.tsx:391`（模板/转录制）、`tasks/modals/AIScriptEditor.tsx`（降级录制入口） | NOCODE-04 / NOCODE-03 |
| 157 | provider 键在**根模块** `internal/config/config.go:259 RPAScalingConfig`，**不是** rpa-worker 的 config | scan-worker 取证 |
| 158 | `rpa_router.go` 的 Schedule CRUD 路由注册（原文「CRUD API」推不出可绕开 router） | scan-backend 取证 |

## 5. 机制层限制（必须知道）

- 当前 `.planning/config.json` 的 `parallelization: false` —— **GSD 默认串行执行同层 wave**。
- GSD 的并行机制是**相位内（intra-phase）按 plan 的 wave**，`execute-phase.md:553-580` 会在同一 wave 内先做 `files_modified` 重叠检测，任意两个 plan 共享一个文件就**整波降级为串行**。
- **GSD 没有跨 phase 并行机制**（无 `--phase 152,154` 之类）。上面 W1/W2 的「∥」要靠操作层实现：要么开两个会话各自跑 `/gsd-execute-phase`（风险是无 merge 保护，必须靠 1~7 的边界协议 + 分文件），要么先把两个 phase 的 plan 合并进同一波（不建议，会污染 phase 边界）。
- 推荐做法：**W1 与 W2 用两个独立会话并行跑，W3~W5 串行**；每个会话按第 3 节的归属协议只写自己的文件集，完成后由 Lead 做一次全量 `go build ./... && go test ./... && cd rpa-worker && go build ./...` 与 `tsc` 复核。
