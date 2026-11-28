# Phase W6 — 部署前置补齐（录制接线 / 调度页 / 枚举对齐）

| 项 | 值 |
|---|---|
| **Phase** | W6（v1.56 里程碑收尾波次，非 31 REQ 主线） |
| **执行日期** | 2026-11-25 |
| **需求映射** | REC-01（录制/拾取 HTTP 端点接线）、REC-02（调度管理前端页）、REC-03（前端 ActionType 枚举对齐） |
| **前置依赖** | Phase 155（`70405142`）/ 156（`63dfbc14`）/ 158（`b9f29e14`）+ 收尾修正（`d4147207`）**均已合入 main**，开工前 `git log --oneline -8` 已确认 |
| **状态口径** | `code-complete · verification-pending` —— **未 SHIPPED** |
| **分支** | `main` |

---

## 0. 本 phase 的定位

152–158 已交付的 31 条 REQ 里，有两处「后端好了、前端断头」：

| 缺口 | 来源 | 本 phase 对应 REQ |
|---|---|---|
| 录制器 / 拾取器是**完整库能力**，但无任何 HTTP 端点把它接到前端 | 156-SUMMARY §5 遗留 #1 | REC-01 |
| `sys_rpa_schedules` 后端 CRUD + 到点触发已闭环，但**无列表/新建/启停 UI** | 158-SUMMARY §6 #9 | REC-02 |
| 前端 `ActionType` 仅 13 值，Go 侧已 26 值；156 用字符串分派规避，**双源风险仍在** | 156-SUMMARY §5 遗留 #2 | REC-03 |

目标用户不变：**不会写代码的一般员工**。验收标准是「他能独立完成」，不是「功能存在」。

---

## 1. 归属协议（152–158 已合入，本 phase 仍守）

| # | 约束 | 本 phase 的落实方式 |
|---|---|---|
| 1 | `rpa_router.go`：**只在 `SetupRPARouter` 尾部追加**，不改既有行 | 追加 `SetupRecordingRouter(r.Group("/recordings"), ...)` 一行；`SetupRPARouter` 既有 7 行零改动 |
| 2 | **不得改** 154 冻结的 `browser/manager.go` / `page_manager.go` 核心契约 | 录制/拾取会话**只读**使用 `Manager.Acquire/Release` 与 `Page.RawPage()`；`recorder.go` / `picker.go` **零改动**（其 API 已够用），只新增 `rpa-worker/internal/recording/` 独立包 |
| 3 | **不得回退** 152 的 AIScriptEditor 真调用、155 的 live 画布结构、156 的 wizard 交互 | AIScriptEditor 不碰；155 的 `live/*` 结构**只读复用**（画布组件被 REC-01 前端引用，不修改）；156 的 wizard 交互保留，向导**只增**录制入口与回填能力 |
| 4 | worker 侧新会话类型放**独立文件**，不进 `types.go` | 新增 `rpa-worker/internal/types/recording.go`（`types.go` 零改动） |
| 5 | 零新增 Go/npm 依赖 | 录制通道复用既有 `redis/go-redis/v9`、`gorilla/websocket`；前端复用既有 antd / dnd-kit / vitest |
| 6 | 构建产物只落 `out/` | 本 phase 不产出二进制；`go build ./...` 不落盘 |

---

## 2. REC-01 通道设计（**先设计，后动手**）

### 2.1 问题陈述

Worker 侧已有两个**完整**的库能力：

| 能力 | 位置 | 现状 |
|---|---|---|
| 页面内录制器 | `rpa-worker/internal/browser/recorder.go`（1037 行） | `NewRecorder(page, log)` → `Install/Start/Events/Stop` → `BuildRecording(events)` → `RecordingResult{Actions, Steps, StartURL}`，产出 schema 与 `types.Action` 完全一致 |
| 元素拾取器 | `rpa-worker/internal/browser/picker.go`（332 行） | `NewPicker(page, log)` → `PickAt/PickBySelector/PickInFrames` → `PickedElement{Found, Tag, Text, Tiers(7级), Recommended, RecommendedTier, Frame}` |

两者都以 `*rod.Page` 为入口，都需要**一个活着的 page**。缺的是：**从「前端点了录制」到「Worker 拿到一个 page 并把结果送回前端」的完整链路**。

### 2.2 关键设计取舍：与 155 liveview 帧通道的关系

155 已建立一条**下行 Redis / 上行 HTTP / 前端 WS** 的三段式通道（`155-PROTOCOL.md`）：

```
下行：前端 --HTTP--> 后端 --Redis PubSub execution:liveview:{id}--> Worker --CDP Input.dispatch*-->
上行：Worker --HTTP POST /rpa/workers/liveview/frame--> 后端 --NoticeHub WS--> 前端 Canvas
```

**REC-01 是否复用这条通道？分两个维度分别裁定：**

#### 维度 A：帧（画面）通道 → **复用 liveview 的帧通道，不另开新通道**

- 录制会话需要把「Worker 上的浏览器画面」投到前端（员工要看着画面操作，才能录出真实动作）。
- 155 的帧通道已经把最硬的两个问题解决了：(a) `NoticeHub.PublishLiveToUsers` 是**可丢帧广播**（队列满丢帧不 close 连接）；(b) 帧 ACK 背压闭环在 Worker 侧。
- 另开一条 `recording:frame` 通道 = 重写一遍这两件事，且必然产生第二套坐标系约定（155-PROTOCOL §9 的 canvas→viewport 换算）——**这是最典型的双源漂移温床**。
- **裁定：复用。** 录制会话在 liveview 侧以「一个伪 executionId 的 liveview 会话」存在，画面照走 `rpa_liveview_frame`。

#### 维度 B：会话生命周期通道 → **另立独立频道，不复用 `execution:liveview:*`**

- liveview 频道的 key 是 `executionID`，其语义强绑定「一次 RPA 执行」：后端 `LiveViewHub` 的会话表以 execution 为键、`Continue` 会去查 `sys_rpa_executions` 的状态机、审计写 `ResourceType=execution`。
- 录制会话**不是一次执行**：它没有 execution 行、没有 task 脚本、它的产物是「一份 actions JSON 给某个 task 用」。硬塞进 execution 命名空间会导致：
  - 录制会话被写进 `sys_rpa_executions`（假数据，158 刚清完一轮假数据）；
  - `Continue` 的状态机校验对录制无意义；
  - 审计表混入「不存在 execution 的操作记录」。
- **裁定：另立 `recording:session:{sessionId}` 频道**，只承载录制自己的生命周期指令；**画面帧仍走 liveview 的 `execution:liveview:{sessionId}` 频道**（sessionId 与 executionId 同形，Worker 侧从频道名反解，两侧共用同一套反解规则）。

> 结论一句话：**帧通道复用 liveview（画面语义完全相同：都是「Worker 浏览器画面」）���会话通道独立（生命周期语义完全不同：录制产脚本，执行产执行记录）。**

### 2.3 Redis 频道命名

| 频道 | 方向 | 模式 | 承载 |
|---|---|---|---|
| `recording:session:{sessionId}` | 后端 → Worker | Worker 侧 `PSubscribe("recording:session:*")` | 录制会话生命周期指令（start / stop / pick / keepalive / abort） |
| `execution:liveview:{sessionId}` | 后端 → Worker | 复用 155 的 `LiveViewPattern` | 该录制会话的**画面 + 输入注入**（watch / input / takeover / keepalive） |
| `execution:cancel:*` | 后端 → Worker | 复用 153 的既有模式 | **不用于**录制（录制无执行可取消；停录用 `recording:session` 的 stop） |

命名与既有前缀族一致：`execution:*`（执行域）/ `recording:*`（录制域）/ `worker:*`（Worker 域）。**不使用 `rpa:` 前缀**——既有 Redis 频道里 `rpa:task:pending` 是 Stream 名而非 Pub/Sub 频道，混用会让运维按前缀检索时误判。

### 2.4 消息 schema

#### 2.4.1 下行：录制会话指令（`recording:session:{sessionId}`）

载体类型：`rpa-worker/internal/types/recording.go`（**新增独立文件**，`types.go` 零改动）。

```jsonc
{
  "action": "start",          // start | stop | pick | keepalive | abort
  "sessionId": "uuid",
  "workerId": "worker-abc",  // 目标 Worker（start 时指定，其余可空）
  "operator": "张三",         // 操作者展示名
  "targetUrl": "https://...", // start：起始页面 URL（可空 = Worker 空白页起步）
  "pickX": 0,                // pick：画布 CSS 像素 X（viewport 坐标系，与 155 一致）
  "pickY": 0,                // pick：画布 CSS 像素 Y
  "timestamp": 1766380800000
}
```

`action` 语义表：

| action | 何时发 | Worker 侧动作 |
|---|---|---|
| `start` | 用户点「开始录制」 | Acquire 一个 page → 导航到 `targetUrl` → `NewRecorder(page)` → `Install()` + `Start()` → 同时把该 page 登记进 liveview 侧（供画面与输入注入） |
| `stop` | 用户点「停止录制」 | `recorder.Stop()` 取事件 → `BuildRecording` → **把 actions JSON 通过 HTTP 上行给后端** → 释放 page |
| `pick` | 用户在画布上悬浮点击元素 | `NewPicker(page)` → `PickInFrames(x, y)` → **把 7 级阶梯结果通过 HTTP 上行给后端** |
| `keepalive` | 前端心跳（默认 20s） | 刷新会话闲置计时 |
| `abort` | 会话异常/前端强制退出 | 释放 page、清理会话 |

#### 2.4.2 上行：Worker → 后端（HTTP，`WorkerAuth`，与 155 同款处置）

| 端点 | 体 | 说明 |
|---|---|---|
| `POST /rpa/workers/recording/result` | `{sessionId, actions[], steps[], startUrl, eventCount}` | 停止后的产出（**权威结果**），后端据此写任务脚本 |
| `POST /rpa/workers/recording/pick` | `{sessionId, found, tag, text, tiers[], recommended, recommendedTier, recommendedScore, frame}` | 拾取结果（7 级阶梯原样回传） |
| `POST /rpa/workers/recording/status` | `{sessionId, status, eventCount, message}` | 状态变化（started / recording / stopped / failed），供前端显示「已录 N 个动作」 |

> **注意**：上行**不重开 WS**。与 155 一致，Worker 是纯 HTTP client（`communication.APIClient`）；上行只加 3 个端点，全部挂 `middleware.WorkerAuth`（Worker 不是用户，无 RBAC 主体）。

#### 2.4.3 后端 → 前端（复用既有 NoticeHub WS，不新建连接）

| 消息类型 | 内容 | 可丢性 |
|---|---|---|
| `rpa_liveview_frame`（155 既有） | 录制会话的画面帧（`executionId` 位放 `sessionId`） | **可丢**（走 `PublishLiveToUsers`） |
| `rpa_liveview_state`（155 既有） | 叠加层情境 | 不可丢 |
| `rpa_recording_status`（**新增**） | `{sessionId, status, eventCount, message}` | 不可丢（走 `BroadcastToUsers`） |
| `rpa_recording_result`（**新增**） | `{sessionId, actions[], steps[], startUrl}` | 不可丢 |
| `rpa_recording_pick`（**新增**） | `{sessionId, ...7 级阶梯}` | 不可丢 |

新增的三个类型只在 `internal/websocket/notice_hub.go` 的常量块**追加**（不碰 `PublishLiveToUsers` 等既有方法），前端在 `liveFrameBus.ts` **同款**的运行时挂接机制里**分流**（复用 `installLiveFrameTap` 的引用计数模式，新增一个 tap，不新建 WS 连接——否则第二次注册会覆盖同一 userID 的连接，把通知铃铛打死，这是 155-PROTOCOL 与 liveFrameBus.ts 头注释里已登记的坑）。

### 2.5 会话生命周期

```
前端                后端                     Worker（Redis + HTTP）
 │  start             │                          │
 ├──────────────────>│ 生成 sessionId            │
 │                   ├──── Redis recording:session:{id} ───>│ Acquire page
 │                   │                                  │ 导航 + Install + Start
 │                   │<─── HTTP status(started) ────────┤
 │<── WS rpa_recording_status(started) ─┤                │
 │                   │                                  │
 │  （用户在画面上操作，输入经 liveview 通道注入 Worker）    │
 │                   │                                  │ recorder 采集事件
 │                   │<─── HTTP status(recording, N) ────┤（节流上报）
 │<── WS rpa_recording_status(recording) ┤                │
 │                   │                                  │
 │  stop             │                          │
 ├──────────────────>├──── Redis recording:session:{id} ───>│ recorder.Stop()
 │                   │                                  │ BuildRecording → actions
 │                   │<─── HTTP result(actions) ─────────┤
 │  ← 写 task.script │                                  │ Release page
 │  ← 返回 actions   │                          │
 │<── WS rpa_recording_result ─────────────┤                │
 │  → 填入向导第 2 步步骤列表                │                │
```

**关键点**：停止时**由 Worker 产出 actions、后端落库**（`sys_rpa_tasks.script`），而不是前端回传自己拼的 actions —— 前端回传会引入「前端被篡改」的可能，且与「产出 actions JSON 存为该 task 的脚本」的 REQ 原文一致。后端落库成功后**再**把 actions 推给前端用于回填向导，两端一致。

### 2.6 超时清理（三层，缺一不可）

| 层 | 超时 | 触发 | 后果 |
|---|---|---|---|
| 前端 | 心跳 20s（与 155 一致）；页面卸载即 `stop` | 前端 | 无 |
| 后端 | 会话闲置 `recording.idle_timeout`（默认 5 分钟） | 后端 janitor（唯一 goroutine，同 155 风格） | 下发 `abort` 给 Worker + 删会话 |
| Worker | 收到 `abort` 或 `stop` 后立即释放 page；Worker 侧另有 60s 兜底 janitor 清理孤儿 page | Worker | 释放 page（**不关 browser**，遵 154 硬语义） |

**孤儿 page 是本设计最大的资源风险**：录制会话在 Worker 侧持有的是一个真实 Chromium page（占内存）。若后端崩溃 / Redis 断连，Worker 侧会话会悬挂。故 Worker 侧必须自带兜底清理。

### 2.7 并发约束

- **单 Worker 同时只允许 N 个录制会话**，N 由 `worker.recording.max_sessions` 配置（默认 2，与 155 的 `max_streams` 默认值同量级）。
- **裁决位置在 Worker**（与 155 一致：真正的名额在 Worker 侧裁决，后端只展示上限与给可读错误文案）。Worker 侧超限返回 `LimitError`（可读中文），后端转 **429**。
- 后端侧 `rpa.recording.max_sessions` 只用于展示与前置校验（避免用户点了才开始被拒）。
- 录制会话占用 Worker 的 `Manager` page 名额，故 N ≤ `worker.max_concurrency`（配置加载期校验并给出可读告警，不静默取小值）。

### 2.8 权限

| 端点组 | 权限码 | 理由 |
|---|---|---|
| 用户侧录制端点 | `ops:rpa:task:add`（ANY-of，读写同一码） | 录制是「做任务的手段」，产物最终存进某个 task 的脚本；不新增权限码（与 155「不新增权限码」同款裁定） |
| Worker 上行端点 | `middleware.WorkerAuth` | Worker 不是用户，无 RBAC 主体（与 155 同款） |

### 2.9 前端交互

1. wizard 第 1 步「录一段操作」卡片 → 点「下一步」或卡片本身 → **打开录制面板**（Modal 内嵌，**不是死按钮**）。
2. 录制面板：复用 155 的 `LiveCanvas` 画布组件（**只读复用，不修改**），叠加「正在录制 · 已录 N 个动作」指示条 + 「停止录制」按钮。
3. **拾取**：在画布上悬浮/点击元素 → 后端 `pick` → Worker 回 7 级阶梯 → 前端浮层逐级显示「可用 / 不可用」→ 用户选一级 → 写入动作的 selector。
4. 停止 → 后端落库 → 返回 actions → **直接填进向导第 2 步的步骤列表**（中文步骤名由 156 的 `stepLabels.describeAction` 转换，REC-03 后枚举同源）。

---

## 3. REC-02 调度管理前端页设计

### 3.1 页面结构

`xingran-react-frontend/src/pages/operations/rpa/schedules/`（新建目录）：

| 文件 | 职责 |
|---|---|
| `index.tsx` | 列表页（筛选栏 + 表格 + 新建按钮） |
| `columns.tsx` | 列定义（名称 / 关联任务 / 运行方式 / 启停 / 最近触发 / 下次触发 / 操作） |
| `modals/ScheduleModal.tsx` | 新建/编辑弹窗 |
| `cronPresets.ts` | 简单选项 → cron 表达式换算 + 高级折叠里的 cron 解析 |
| `api.ts` | 后端调用封装（走 `@/lib/api` 的 `post`，禁裸 axios） |
| `__tests__/schedules.test.tsx` | vitest 用例 |

### 3.2 员工可见文案与「隐藏 cron」

| 员工看到的 | 实际提交 |
|---|---|
| 「每天 09:00」 | `0 9 * * *` |
| 「每周一 09:00」 | `0 9 * * 1` |
| 「每隔 15 分钟」 | `*/15 * * * *` |
| 「高级」折叠里的 cron 表达式 | 原样提交（默认折叠，员工不需要看到） |

后端 `schedule_service.ComputeNextRunTime` 已支持 5 位与 6 位表达式、非法 cron 返回可读错误 —— 前端只需保证生成合法表达式。

### 3.3 路由

**只通过 `sys_menu` seed migration 注册**（归属协议第 6 条，与 255/256 同款），新增 `migration_258_rpa_schedule_menu.go`：
- 1 个 C 型页面菜单「定时调度」（`component=pages/operations/rpa/schedules/index`），父 = 「RPA 管理」`af1451a4-650c-4ed1-a2b7-b58769dc3920`，order=18。
- **不新增 F 型按钮**：`ops:rpa:schedule:list/add/edit/remove` 四码已由 migration 257 播种，本 phase 只复用，不扩权限面。
- 双方言幂等 + `database.go` 两处注册 + `registry.go` 登记（255/256/257 同款）。
- **不改** `componentLoader.tsx` / `DynamicRoutes.tsx`。

---

## 4. REC-03 ActionType 枚举对齐设计

### 4.1 现状

- Go 侧 `rpa-worker/internal/types/types.go`：**26 个** `ActionType` 常量。
- 前端 `src/types/rpa.ts`：**13 个**（缺 evaluate / waitFor / close / loop / pause / condition / autologin / assert / doubleClick / rightClick / drag / keyboard / switchTab / script），且含一个 Go 侧**不存在**的 `press`。
- 156 用 `stepLabels.ts` 的字符串分派规避 → **双源风险**（前端枚举与 Go 常量各写一份，谁漏了都不知道）。

### 4.2 裁定：单一真相源 + 三方对齐测试

| 层 | 职责 |
|---|---|
| Go `rpa-worker/internal/types/types.go` | **唯一真相源**（26 个常量） |
| 前端 `src/types/rpa.ts` | 镜像声明（26 个字面量联合类型），**由测试锁死与 Go 一致** |
| 前端 `wizard/stepLabels.ts` | **消费方**：`HUMAN_VERBS` 的键集必须覆盖全部 26 个（缺一即红） |

**消除双源**：把 `stepLabels.ts` 里手写的 26 个 `HUMAN_VERBS` 键改为**从共享枚举派生**（`Record<ActionType, string>` 类型层面强制全覆盖），字符串 switch 改为**对共享枚举 exhaustiveness 检查**（`switch` 的 `default` 分支用 `never` 断言，漏 case 即编译失败）。

### 4.3 三方一致性测试

参照 worker 侧 `engine_action_coverage_test.go` 的「枚举 ↔ 分派全覆盖」思路，**三方对齐，缺一即红**：

| 测试 | 位置 | 断言 |
|---|---|---|
| Go 侧同源测试 | `internal/services/rpa/action_type_consistency_test.go` | 直接读 `rpa-worker/internal/types/types.go` 抽 26 个常量 → 断言与**前端 `types/rpa.ts` 的联合类型字面量集合**、与**后端 `workerActionTypes` 镜像**三者**双向相等** |
| 前端侧消费覆盖测试 | `xingran-react-frontend/src/pages/operations/rpa/wizard/__tests__/actionTypeConsistency.test.ts` | 断言前端枚举集合 == stepLabels `HUMAN_VERBS` 键集 == Go types.go 抽出的常量集（三方） |

Go 侧测试是**跨仓库读源文件**的（`internal/services/rpa/` 上溯到仓库根读 `rpa-worker/internal/types/types.go` 与 `xingran-react-frontend/src/types/rpa.ts`）—— 与 156 的 `TestWorkerActionTypes_MirrorsWorkerTypesGo` 同款做法（读文件失败则 `t.Skip`，不误红）。

---

## 5. 交付与闸门

| 闸门 | 命令 |
|---|---|
| 根模块构建 | `go build ./...` |
| 根模块 RPA 包测试 | `go test ./internal/api/v1/rpa/ ./internal/services/rpa/` |
| Worker 构建 + 测试 | `cd rpa-worker && go build ./... && go test ./...` |
| 前端类型检查 | `cd xingran-react-frontend && npx tsc --noEmit -p tsconfig.app.json` |
| 前端测试 | `cd xingran-react-frontend && npx vitest run src/pages/operations/rpa/` |
| 前端构建 | `cd xingran-react-frontend && npm run build` |
| 路由注册表 | `node scripts/scan-routes.cjs`（新增端点后重生成） |

**真 Chromium 验收**（REC-01）：Worker 侧起真 Chromium → 录制 3 个真实动作 → 停止 → 断言产出 3 条动作 → 回放 → 服务端真实收到数据。**严禁 fake**。

---

## 6. 风险登记

| # | 风险 | 缓解 |
|---|---|---|
| 1 | 录制会话在 Worker 侧持有真实 page，异常时泄漏 | Worker 侧兜底 janitor（§2.6）；stop/abort 必释放 |
| 2 | 帧通道复用 liveview → 录制会话的「伪 executionId」可能污染执行记录 | 录制会话**不写** `sys_rpa_executions`；后端 liveview hub 的 `Continue` 状态机查询对录制 session 无副作用（无 execution 行 → 查不到即跳过）；审计写 `ResourceType=execution` 的路径**仅在有 execution 时启用**，录制会话走独立的 `recording:*` 审计字段 |
| 3 | REC-03 改 `types/rpa.ts` 可能影响既有前端代码 | 该文件当前 ActionType 13 值；补齐是**加法**（`press` 保留为兼容别名，但 Go 侧无此值 → 测试断言时按「Go 侧集合 ⊆ 前端集合 且 前端多出的仅为显式白名单别名」处理） |
| 4 | REC-02 依赖 158 的调度后端，若 158 有缺陷则页面同步坏 | 158 已本机真跑到点触发（158-SUMMARY §2.3），后端可信；页面层只做 CRUD |
| 5 | 前端路由若误改 `componentLoader.tsx` 会与 155/156 冲突 | 只走 sys_menu seed migration，**不改**两个路由文件 |
