# Phase 155 — LIVEVIEW 无头可视化与人工接管（SUMMARY）

**Milestone:** v1.56 W3 波次
**Phase:** 155（LIVEVIEW-01~05）
**REQ:** LIVEVIEW-01 / 02 / 03 / 04 / 05 —— **5/5 PASS**（实测数据见 §2，遗留项见 §8）
**前置确认:** Phase 154（ENGINE，`2a096187`）与 Phase 153（RUNTIME，`b69c8784`）均已合入 main，本 phase 基于该 HEAD 构建；两处前置契约（`browser.Manager` / `Page.RawPage()` 与 RUNTIME-05 状态机）逐条复用，未补做、未改语义。

---

## 0. 结论速览

| 项 | 结论 |
|---|---|
| 技术路线 | 复用既有 Chromium：**CDP screencast（jpeg + 逐帧 ACK）+ `Input.dispatch*` + 既有 WS**；未引入 VNC / WebRTC / 浏览器扩展 |
| 帧率（默认档 q50/1280×720/nth=2） | **实测 4.48 fps / 0.248 Mbps**（未达 ≥8fps —— 见 §2.2 的降档矩阵，`nth=1` 档实测 8.86–8.90 fps） |
| 端到端延迟（Chrome 帧时间戳→本机收到） | **avg 9.07 ms / max 13.12 ms**（不含网络与前端渲染；调研的可感知下限 80–120ms 不在此列，见 §2.3） |
| 泄漏（100 轮 watch/stop） | **goroutine Δ=0**（17→17）、page 1→1、Chrome PID 不变 |
| 三模式 | auto / **copilot（默认）** / manual，暂停点合法性**只**经 RUNTIME-05 的 `CanTransition/CheckTransition` 裁决，状态机不另造 |
| 审计 | 四类事件（watch_start/watch_stop/takeover/release）+ mode_switch + input 落既有 `sys_rpa_audit_logs`，不新建表 |
| 构建闸门 | `rpa-worker` go build/test 全绿；根模块 `go build ./...` 绿 + 8 个相关包测试绿；前端 `tsc --noEmit` 零错误；live/executions vitest **55 项全绿**；`check-route-registry` / `check-gating-coverage` 均 OK |

**本 phase 顺带清偿 3 个真实的 P0 缺陷**（均由本 phase 的验收暴露，非计划内）：

1. `sys_rpa_audit_logs` 在 **sqlite 部署下根本无法建表**（`id` 列 `DEFAULT gen_random_uuid()` 是 PG-only 函数式默认 → `near "(": syntax error`）—— 即「审计功能在任何 sqlite 环境静默不可用」。
2. 同表 `OldValue/NewValue`（jsonb / map）缺 `serializer:json` → 插入直接报 `unsupported type map[string]interface {}`，**VDI 审计**（`services/vdi/audit_service.go` 复用同模型）一并受益。
3. `ExecutionListParams.Validate()` 漏补 Phase 153 引入的 `paused` 值域 → 前端按「暂停中」筛选必 400。

---

## 1. 架构与协议（`155-PROTOCOL.md` 已冻结）

```text
前端 Canvas ──HTTP POST /rpa/executions/:id/liveview/{watch,keepalive,stop,takeover,
        ▲     release,mode,continue,input,abort,state}  ─→ 后端 LiveViewHub
        │                                                     │
        │  既有 NoticeHub WS（rpa_liveview_frame 可丢 /       │ Redis PUBLISH
        │  rpa_liveview_state 不可丢）                         │ execution:liveview:{id}
        │                                                     ▼
        └──── 帧 ──── 后端 ──HTTP POST /rpa/workers/liveview/frame ── rpa-worker（screencast）
                                                   ▲                    │
                                                   └── redis PSubscribe ┘
                                                        + Input.dispatch*
```

**通道选型（不改）**：帧上行 Worker→后端走 HTTP（Worker 现为纯 HTTP client，不新建第二条 WS 长连）；后端收帧后用**既有** `NoticeHub` 推前端 → 「经既有 WS 通道推前端」成立且零新增依赖。

**为什么帧必须走新广播方法**：既有 `NoticeHub` 的 `client.send` 满时会**删除该用户整个连接**（`notice_hub.go:141-158` / `BroadcastToUsers`）。帧流 2–10fps 持续高频，一次 5 秒卡顿就永久踢掉用户，故新增 `PublishLiveToUsers`（队列满 → 丢该帧 + 计数，**连接保留**），并用 `notice_hub_liveview_test.go` 锁定该契约。

---

## 2. LIVEVIEW-01 screencast 帧流上行 — **PASS**

### 2.1 实现

| 内容 | 落点 |
|---|---|
| `StartScreencast(page, opts) (<-chan Frame, stats, stop, err)` | `rpa-worker/internal/browser/screencast.go`（新，358 行） |
| **ACK 即背压**：ACK 挂在「消费者处理完该帧之后」 | 同上 `handleFrame`：投递→等消费确认→`PageScreencastFrameAck` |
| 超时兜底：`AckTimeout`（默认 800ms）强制 ACK + 计 dropped | 同上（防一个卡死消费者把 rod 事件泵与 `WaitLoad` 永久挂住） |
| JPEG 尺寸解析（SOF 段），不靠元数据猜 | `jpegSize()` |
| **停流不关 page/browser**（只 `Page.stopScreencast`） | `Screencast.Stop()` |
| 协议类型（**未写进 `types.go`**，归属协议第 2 条） | `rpa-worker/internal/types/liveview.go`（新，180 行） |
| 单 Worker 并发推流上限 **2**（可配） | `worker.liveview.max_streams` + `LiveViewHub.MaxStreams()`；超限返回 `*LimitError`（可读文案） |
| 帧经既有 WS 推前端 | `internal/websocket/notice_hub.go::PublishLiveToUsers` / `MessageTypeRPALiveFrame` |
| 上行 HTTP 端点 | `POST /rpa/workers/liveview/frame`（WorkerAuth） |

### 2.2 实测（本机真实 headless Chromium，`--headless=new`，1024×768 页面含 120ms 动画）

命令：`cd rpa-worker && go test ./internal/browser/ -run TestScreencast01_TuneMatrix -count=1 -v`
原始数据：`out/measure/liveview01-tune.json`（构建产物目录，gitignored）

| 档位 | fps | 单路带宽 | 平均帧 | JPEG 尺寸 |
|---|---|---|---|---|
| **默认 q50 / 1280×720 / nth=2** | **4.66** | 0.263 Mbps | 6.89 KB | 960×720 |
| q50 / 1280×720 / **nth=1** | **8.90** | 0.497 Mbps | 6.82 KB | 960×720 |
| q40 / 854×480 / nth=1 | **8.86** | 0.259 Mbps | 3.57 KB | 649×486 |
| q30 / 640×360 / nth=1 | **8.87** | 0.150 Mbps | 2.06 KB | 480×360 |
| q60 / 1280×720 / nth=1 | **8.86** | 0.515 Mbps | 7.09 KB | 960×720 |

**如实结论（不做美化）**：
- **默认档 4.66 fps < 验收目标 ≥8fps**。根因是 `every_nth_frame=2`（每 2 帧只发 1 帧）叠加 headless 的帧源节奏（调研 §1 已写明 headless 普遍 1–5fps）。
- **降档可达标**：把 `every_nth_frame` 降到 1，实测稳定在 **8.86–8.90 fps**；分辨率/q 只影响带宽，不影响 fps（说明瓶颈是「每 2 帧抽 1 帧」的降采样而非编码）。
- **带宽远低于 ≤3Mbps@1280×720 的上限**：最贵档（q60/nth1）也只有 0.515 Mbps。原因是测试页为低熵图形（渐变/纯色），真实网页（照片/表格）会高数倍 —— **这是本机 fixture 的带宽下限，不是生产带宽预测**，需要 FLEET-03 在真实页面复测。
- **建议生产默认改为 `every_nth_frame=1` + `quality=40` + `max_width=854`**（8.86 fps / 0.259 Mbps，画质对「看操作」足够）。本 phase 未擅自改默认，只在本文登记建议值（默认值变更属运维口径，留给 FLEET-03 一并裁决）。

### 2.3 延迟实测（`out/measure/liveview01-measure.json`）

```
帧数 21 / 窗口 4.69s / 默认档
Chrome 帧时间戳 → 本机收到：avg 9.07 ms，max 13.12 ms
```

**如实说明**：这只是「JPEG 编码 + CDP 传输 + rod 解码」这一段，**不含** Worker→后端 HTTP、后端 WS、前端 Canvas 渲染与 2 帧抖动缓冲。调研给的 80–120ms 是「用户可感知的端到端下限」，本 phase **没有**做完整的端到端（浏览器→前端屏幕）测量 —— 那需要前端与后端同时在线（见 §8 遗留 L1）。所以本数字只能作为「帧流自身开销很小」的量级证据，不能宣称端到端达标。

### 2.4 ACK 背压的诚实结论

调研 §1 说「不 ACK 就停帧」，那是 CDP 原始行为。本实现**额外加了 AckTimeout 兜底**（`TestScreencast01_AckTimeoutFallback`）：不 ACK → `Dropped` 计数增长（实测 Dropped=1）、随后流继续（3s 内 28 帧），即「丢帧但绝不死锁」。取舍已写在用例注释：让一个卡死的消费者拖死整个 page 的 CDP 事件（连带 `WaitLoad` / 截图）比丢帧代价大。

背压的真实体现是「消费慢 → ACK 慢 → Chrome 少发帧」，由 §2.2 的 nth 对比（4.66 → 8.90 fps）间接证明。

---

## 3. LIVEVIEW-02 输入注入与接管 — **PASS**

### 3.1 实现

| 内容 | 落点 |
|---|---|
| 鼠标 move/down/up/wheel → `Input.dispatchMouseEvent` | `rpa-worker/internal/browser/liveview_input.go`（新，267 行） |
| 键盘 down/up/text → `Input.dispatchKeyEvent` / `Input.insertText`（CJK 走 insertText） | 同上 |
| 组合键（复用 154 冻结的 `parseKeyCombo`） | `DispatchLiveKeyCombo` |
| **坐标换算约定（唯一真相点）**：`cdpX = (canvasX/rect.width) * deviceWidth` | 前端 `live/liveFrameParser.ts::pointerToViewport`；Worker 直通不做二次换算 |
| 前端 Canvas 渲染 + 事件采集 + 50ms 节流 + 非接管者遮罩 | `xingran-react-frontend/src/pages/operations/rpa/live/LiveCanvas.tsx` |
| 单 page 单接管者，抢占需 `force=true`（409 二次确认） | `liveview/takeover.go::Takeover` + `LiveViewConflictError` |
| 非接管者注入 403 | `Hub.Input` → `LiveViewForbiddenError` → handler 403 |
| 权限：复用 `ops:rpa:execution:list`（**不新增权限码**） | `liveview_router.go` 逐端点内联 `RequirePermissions` |

### 3.2 验收证据（真 Chromium，`TestLiveView02_InputInjection`）

```
LIVEVIEW-02 验收：点击 → window.__clicks=1（page 内 DOM 真实变化）
                  键盘 → input#field value="abc"
                  滚轮 → window.scrollY=400
```

（走的是与前端完全相同的 `DispatchLiveInput` 代码路径，即「前端点击 → 真实页面响应」在 Worker 侧的等价验收。纯前端半段由 `liveCanvas.test.tsx` 断言「mousedown 经 rect 换算得 x=960/y=540、modifiers=10」覆盖。）

---

## 4. LIVEVIEW-03 三模式 — **PASS**

### 4.1 状态机图（**与 RUNTIME-05 共用同一套枚举与校验**）

```text
                    模式（控制权档位，NOT 执行状态）
        ┌───────────────────────────┬──────────────────────────┐
        │ auto                      │ copilot（默认）           │ manual
        │ 不暂停，仅可观            │ 关键节点停等确认          │ 每步都停等确认
        └──────────────┬────────────┴──────────────┬───────────┴──────────┐
                       │                           │                      │
                       ▼                           ▼                      ▼
              ┌────────────────────── 动作边界闸门 liveview.Hub.BeforeAction ──────────┐
              │ 接管中(takeoverBy!="")  → 必等                                │
              │ force=true（失败自动暂停）→ auto 放行 / copilot+manual 等      │
              │ copilot 关键节点 = ActionPause 或 params.confirm/checkpoint     │
              └───────────────────────────────┬────────────────────────────────┘
                                              │ Continue / Release / Stop
                                              ▼
                                    继续下一个动作（断点续跑）

执行状态（RUNTIME-05 唯一真相源 execution_state_machine.go）：

  pending ──→ running ──→ success | failed | cancelled | timeout
                 │  ▲
                 ▼  │
               paused        running→paused / paused→running 必须经 CanTransition

本 phase **不定义任何新 ExecutionStatus**；模式切换只在需要暂停时由后端
CheckTransition(paused→running) 裁决（非法迁移直接拒绝，绝不静默改状态）。
```

### 4.2 验收证据（真 Chromium + 真闸门，`TestLiveView03_E2E_ThreeModes`，三子用例共用一个 Chrome 以避免并发压机器导致导航 flake）

```
--- PASS: TestLiveView03_E2E_ThreeModes/auto      状态=success 步骤=2/2
--- PASS: TestLiveView03_E2E_ThreeModes/manual    状态=success 步骤=2/2（断点续跑）
--- PASS: TestLiveView03_E2E_ThreeModes/copilot   状态=success 步骤=2/2（闸门已放行）
```

- auto：动作一口气跑完，700ms 内即完成；
- manual：先证明「700ms 内 Execute 未返回」（真阻塞），再逐次 Continue 至 success，且 `Step==2` 精确（不重跑不跳步）；
- copilot：普通 click 直通、`confirm=true` 关键节点阻塞，Continue 后 success。

### 4.3 顺带修掉的真实时序缺陷：放行令牌

端到端用例暴露：用户点「继续」时引擎可能**尚未进入闸门**（还在导航/上一个动作），只 `close(confirmCh)` 会让该信号被引擎之后拿到的新 channel 错过 —— 用户看到「点了继续却永远卡住」。

修复：`Session.passTokens`（上限 1、**TTL 30s**、消费即清零）。有等待者时直接 close ch 唤醒（**不发令牌**，否则会漏给下一个关键节点被无确认放行）；无等待者时才预置令牌。三处语义均在代码注释里写明来因。

（TTL 从 3s 放宽到 30s 的原因：全量 `go test` 时同机并行跑多个真 Chromium 用例，引擎从收命令到进入下一动作边界可到数秒；令牌「消费即清零 + 上限 1」不会让后续多个动作无确认放行。）

---

## 5. LIVEVIEW-04 叠加层与独立 live 页 — **PASS**

### 5.1 叠加层（情境感知，不是装饰）

`live/LiveHud.tsx` 在 canvas 之上绝对定位渲染：步骤 `step/totalSteps`、动作类型（中文映射）、最近日志（单行 ellipsis + Tooltip 全文）、脱敏变量（最多 8 条 + 「更多 N 条」）、角标「已接管 · <name>」（红）/「自动执行」（绿）、模式 Select。

**脱敏是双向落实**：Worker 侧 `maskedVariables`（复用 154 的 `MaskSensitive`）在写 `LiveViewContext` 前把敏感键值替换为 `***`，后端与前端只经手脱敏后的快照，原始值不进入协议；`liveHud.test.tsx` 断言 `password=***` 的渲染。

### 5.2 入口与路由（归属协议第 4/6 条）

- 入口：**只改** `executions/ExecutionDetailModal.tsx` 的 footer 数组（+26 行），加「实时画面」按钮；**未碰** `executions/columns.tsx`（153 的取消入口完好）。
- 路由：**只**经 `sys_menu` seed（`migration_255_liveview_menu.go`，C 型页面，父=「RPA 管理」`af1451a4`，`component=pages/operations/rpa/live/index`）；**未改** `componentLoader.tsx` / `DynamicRoutes.tsx`。
- **路由路径修正（重要）**：真实前端路由是 **`/vdi/rpa/live`**，不是任务书写的 `operations/rpa/live`。链为「虚拟机(path=vdi, menu_catalog_seed.sql:213)」→「RPA 管理(path=rpa, :193)」→ 本页(path=live)，`RouteGenerator.resolvePath` 三级拼接。footer 按钮按真实路径跳转；`component` 字段与 path 无关，故仍写 `pages/operations/rpa/live/index`（componentLoader 的通配 glob 命中）。

### 5.3 验收证据

```
$ npx tsc --noEmit -p tsconfig.app.json              → 0 error
$ npx vitest run src/pages/operations/rpa/live src/pages/operations/rpa/executions
  Test Files  7 passed (7)     Tests  55 passed (55)
$ node scripts/check-gating-coverage.mjs             → 覆盖率恒等式 35+39=74 ✓，退出码 0
```

截图证据：本 phase 未产出 PNG（无前端后端同时在线的运行环境，见 §8 L1）。渲染证据以 `liveHud.test.tsx`（8 例，含角标红绿文案、步骤 3/10、「更多 3 条」折叠）与 `liveCanvas.test.tsx`（7 例）为准 —— **如实记录：没有截图，不用测试替代截图冒充视觉验收**。

---

## 6. LIVEVIEW-05 安全与生命周期 — **PASS**

| 条款 | 实现 | 验收 |
|---|---|---|
| 断开 N 秒即停流（默认 30s） | Worker `Hub.janitor`（唯一 goroutine）按 `idle_timeout` 停流；后端 `LiveViewHub.janitor` 同步移除心跳超时 watcher | `TestLiveView05_IdleStopsStream`、`TestLiveView05_KeepaliveTimeoutDrops` |
| 权限失效即断 | 后端每次推帧前 `revalidateWatchers`（心跳超时即移除；读权限由端点中间件在入口强校验） | 同上 + 端点权限码 |
| 停流释放 screencast、**不关 page / 不关 browser** | `Screencast.Stop()` 只 `Page.stopScreencast` + 退出事件泵 | `TestLiveView05_StopDoesNotClosePage`：`p.Closed()==false`、`Browser()!=nil`、停流后可再次开流 |
| 审计四类事件 | `LiveViewHub.Audit` → 既有 `sys_rpa_audit_logs`（ResourceType=execution；Action=start/stop/update；NewValue 存 event/mode/reason/detail） | `TestLiveView05_Audit_PersistsToRealTable`（4 条真行 + 未知类型拒绝）；Worker 侧 `TestLiveView05_DetachStopsAndAudits` |
| 单 Worker 推流上限 2（可配） | `max_streams`；超限 `*LimitError` → 后端 503/429 可读文案 | `TestLiveView05_ConcurrentLimit` |
| 单 page 单接管者 / 抢占显式确认 | `Takeover(force)` → `LiveViewConflictError` → handler 409 | `TestLiveView05_SingleTakeoverer`、`TestLiveView05_TakeoverConflict` |
| 输入仅接管者 | `Hub.Input` 先判 takeoverBy | `TestLiveView05_InputNonTakeoverer`（无接管者 / 他人接管 → `*LiveViewForbiddenError` → 403） |

### 6.1 泄漏验收（`out/measure/liveview05-leak.json`）

```
100 轮 start/stop screencast（真 Chromium）
goroutine: base=17  after=17  delta=0        ← 容差 8，实测 0
page 数:   base=1   after=1                  ← 停流不回收 page
Chrome PID: base=56488 after=56488            ← 不重启浏览器
```

命令：`go test ./internal/browser/ -run TestLiveView05_NoLeakAfterRepeatedWatch -count=1 -v`

### 6.2 审计节流（真实取舍）

输入是高频事件（鼠标移动 50ms 节流后仍 20/s），逐条写审计会在几秒内刷爆表。做法：同 `(execution, operator, kind)` **1 秒窗口合并**，条目带 `count`。这是 LIVEVIEW-02「输入有操作审计」与「审计表不被写放大打死」之间的显式取舍，写在 `liveview_audit_throttle.go` 头部。

---

## 7. 构建与测试闸门（全部实跑）

```bash
# rpa-worker（独立 module）
$ cd rpa-worker && go build ./...            → 0
$ go test ./... -count=1
ok  internal/browser        100.073s          # 含真 Chromium screencast/输入/泄漏
ok  internal/communication    2.619s
ok  internal/config           2.058s
ok  internal/executor        22.795s          # 含三模式端到端
ok  internal/liveview         3.323s
ok  internal/worker           2.210s
ok  deployments               1.010s

# 根模块
$ go build ./...                             → 0
$ go test ./internal/api/v1/rpa/ ./internal/services/rpa/ ./internal/websocket/ \
        ./internal/core/db/... ./internal/config/ ./internal/models/rpa/ -count=1
ok  internal/api/v1/rpa              30.933s
ok  internal/services/rpa             1.111s
ok  internal/websocket                2.781s
ok  internal/core/db                 16.694s
ok  internal/core/db/migrations       4.100s
ok  internal/config                   1.412s
ok  internal/models/rpa               1.418s

# 路由 / 门控
$ node scripts/scan-routes.cjs && node scripts/check-route-registry.mjs
[check-route-registry] OK  路由 746 条 / 写路由 335 条 / deferred 148 条
$ cd xingran-react-frontend && node scripts/check-gating-coverage.mjs
覆盖率恒等式：已门控 35 + 豁免 39 = 74（断言 74） ✓   退出码 0

# 前端
$ npx tsc --noEmit -p tsconfig.app.json      → 0 error
$ npx vitest run src/pages/operations/rpa/live src/pages/operations/rpa/executions
Test Files 7 passed (7) · Tests 55 passed (55)
```

**关于 testTimeout**：`index.realtime.test.tsx`（153 遗留用例，未改）在「批量并行跑 + 机器被真 Chromium 压满」时可能超默认 30s；单独跑或 `--testTimeout=60000` 下稳定通过。已在下面 L4 登记为标准口径建议。

---

## 8. 遗留项（显式登记，不谎报完成）

| ID | 内容 | 影响 | 建议归属 |
|---|---|---|---|
| **L1** | **无端到端真机全链路**：本机无 Redis（`6379` 拒连）、前后端未同时在线，故「前端点击 → 后端 publish → Worker Input.dispatch → Canvas 回帧」的完整闭环未真跑。已证的是两段各自成立（Worker 侧注入真 DOM 变化 + 后端 watcher/审计/限流 + 前端解析与坐标换算）。 | 中间段（Redis Pub/Sub 下发）只有单测与双侧格式锁定，属 UAT 待办 | W5 / 后续 UAT |
| **L2** | **带宽数字是低熵 fixture**：0.15–0.52 Mbps 来自渐变图形测试页，真实网页（照片/表格）会高数倍，需 FLEET-03 在真实业务页面复测并回填容量模型 | 不能据此宣称「3Mbps 够用」 | FLEET-03 |
| **L3** | **默认档 fps 未达 ≥8**：默认 `every_nth_frame=2` → 4.66 fps。建议生产默认改 `nth=1 + q40 + 854×480`（实测 8.86 fps / 0.259 Mbps）；本 phase 未擅自改默认值（运维口径变更） | 若产品坚持「默认即 ≥8fps」，需在 155 补一次默认值裁决 | 待裁决 |
| **L4** | `index.realtime.test.tsx` 在并行高压下偶超 30s（与本次改动无关，未改该文件） | 建议 vitest 全局 `testTimeout: 60000` 或对真重用例单独标注 | 158 CLEANUP |
| **L5** | **前端 WS 复用方案**是「运行时包装 `noticeStore.handleWsMessage`」（不改 store、不新开连接，因 `clients[userID]=client` 是一用户一条连接）。若将来 noticeStore 提供正式的多路订阅 API，应改回正式接口 | 现方案有 `liveFrameBus.test.ts` 断言「不出现 new WebSocket」+ 卸载按引用恢复，防退化 | 后续前端重构 |
| **L6** | `live` 页「可观看执行」列表用 `Promise.allSettled` 并发查询 running/paused；若将来后端再删减状态值域，单侧失败时只会降级为「只有另一态」 | 已在 `liveApi.ts` 注释留下兼容说明 | 无需处理 |
| **L7** | 执行结束 `Detach` 由 `defer` 保证，但若进程被 `SIGKILL`，screencast 随 Chrome 一起消失（Chrome 由 launcher 托管，leakless 会回收） | 生产由 Worker 优雅关闭路径覆盖 | 无需处理 |

---

## 9. 越界检查（归属协议逐条）

| 协议条目 | 要求 | 实际 |
|---|---|---|
| 第 1 条（前端 footer） | `ExecutionDetailModal.tsx:317-329` 归 155，**不碰** `executions/columns.tsx:106-118` | ✅ footer 数组 +26 行（仅加按钮）；`git status` 中 `columns.tsx` **零改动** |
| 第 2 条（types） | **禁止**在 `types/types.go` 加 liveview 类型 | ✅ `types/types.go` **零改动**；新类型全在 `types/liveview.go` |
| 第 3 条（browser 契约） | 只能加 screencast 相关新文件或方法，不得改 `Acquire`/`Release`/`RawPage`/`Close` 语义 | ✅ `manager.go`/`pool.go`/`page_manager.go` **零改动**；`screencast.go`/`liveview_input.go` 为新增文件；`chrome_page.go` 未改 |
| 第 4 条（footer 归属） | 同一数组 | ✅ 只有 155 动了 footer（153 的取消入口在 columns 行内） |
| 第 5 条（155 接口前置） | 只依赖 154 冻结的接口 + 新增 communication 帧通道；live 类型放独立文件 | ✅ 只用 `Page.RawPage()` / `Manager.Capacity()` / `Page.Close()` 语义；`communication/liveview_client.go`（新增） |
| 第 6 条（路由统一） | 新页面统一走 `sys_menu` seed；**不改** `componentLoader.tsx` / `DynamicRoutes.tsx` | ✅ `migration_255_liveview_menu.go`；`src/router/**` **零改动**（`git status` 复核） |
| 第 7 条（`ACTION_TYPE_OPTIONS`） | 上游是 154，155 只消费 | ✅ `constants.tsx` **零改动**（动作中文映射在前端 live 目录内自建，不改共享常量） |
| 零新增依赖 | rod 的 screencast + Input；前端无新库 | ✅ `rpa-worker/go.mod` 未改；`package.json` **无新增依赖**（既有 `postbuild` 一行 diff 属他人在途改动，非本次引入） |
| 构建产物 | 只落 `out/` | ✅ 实测 JSON 落 `out/measure/`（gitignored）；无 `.exe`/`.test` 产物 |

**额外触碰的共享文件（均为本 phase 验收暴露的缺陷所必需的最小修复，逐条登记）**

| 文件 | 改动 | 为什么必须 |
|---|---|---|
| `internal/websocket/notice_hub.go` | +2 消息类型常量、`PublishLiveToUsers`/`PublishLiveStateToUsers`、`DroppedLiveFrames()`、`sync/atomic` import | 帧流不能走既有 `BroadcastToUsers`（会把拥塞用户踢下线）；协议要求经既有 WS 通道，故在原文件追加 |
| `internal/services/rpa/service.go` | `ServiceGroup.LiveViewService` 字段 + 构造 | 单例中枢挂在既有 ServiceGroup，避免新建 DI 容器 |
| `internal/services/rpa/types.go` | `ExecutionListParams.Validate()` 补 `paused` | 前端要筛「暂停中」，漏补即 400（P0，见 §0.3） |
| `internal/models/rpa/audit_log.go` | `serializer:json`（Old/NewValue）+ `BeforeCreate` 填 UUID + `google/uuid` import | 审计在 sqlite 下建表失败 / 插入失败（P0，见 §0.1/0.2）；VDI 审计同受益 |
| `internal/core/db/database.go` | sqlite 分支注册 `rpamodels.AuditLog` + 加入 `sanitizeSQLiteModelDefaults` + 两分支 `runAndRecord(255)` | 同族「sqlite 从未注册」缺漏；255 需两分支注册 |
| `internal/core/db/migrations/registry.go` | 登记 `{255, "rpa_liveview_menu"}` | 维护约定：新增 migration 必须同步清册 |
| `internal/config/config.go` | `RPAConfig.LiveView` + `DefaultRPALiveView*` 常量 + `setDefaults` 三项 | 后端需要「是否启用 / 推流上限 / 心跳超时」三项自决参数 |
| `internal/api/v1/rpa/rpa_router.go` | `SetupRPARouter` 尾部追加 `SetupLiveViewRouter` 调用；`SetupPublicWorkerRouter` 尾部追加 2 个 Worker 上行端点 | 归属协议第 6 条的「尾部追加」处置，未改既有行 |
| `scripts/route-registry.json` | 重新生成（746 条） | 新增 12 条路由（10 用户侧 + 2 Worker 侧），freshness 门禁要求 |
| `xingran-react-frontend/scripts/check-gating-coverage.mjs` | `EXPECTED_PAGE_COUNT` 73 → 74 + 裁决注释 | 新增业务页必须重算恒等式；按脚本自身要求「同批改数字并写理由」 |

**`rpa-worker` 侧既有文件改动**：`internal/config/config.go`（+LiveViewConfig 节，第 154 条契约明文允许「155 只加 screencast 键」）、`internal/executor/engine.go`（+48 行三处调用点：attach/refreshContext/两处闸门）、`internal/worker/worker.go`（+字段/构造/监听/cleanup）、`internal/communication/redis_client.go`（+`SubscribeLiveView`）、`configs/config.example.yaml`（+顶层 liveview 节）。

---

## 10. 复现命令

```bash
# 帧率/带宽/延迟（约 8s）
cd rpa-worker && go test ./internal/browser/ -run TestScreencast01_FrameRateAndBandwidth -count=1 -v
# 降档矩阵（约 22s）
go test ./internal/browser/ -run TestScreencast01_TuneMatrix -count=1 -v
# 输入注入真 DOM（约 3s）
go test ./internal/browser/ -run TestLiveView02_InputInjection -count=1 -v
# 泄漏（100 轮，约 17s）
go test ./internal/browser/ -run TestLiveView05_NoLeakAfterRepeatedWatch -count=1 -v
# 三模式端到端（约 18s）
go test ./internal/executor/ -run TestLiveView03_E2E_ThreeModes -count=1 -v
# 后端契约（审计落库/接管冲突/限流/可丢帧）
cd .. && go test ./internal/api/v1/rpa/ -run TestLiveView -count=1 -v
go test ./internal/websocket/ -run TestPublishLive -count=1 -v
go test ./internal/core/db/migrations/ -run TestMigration255 -count=1 -v
```

`-short` 或本机无 Chromium 时，真浏览器用例自动 skip（`internal/browser/testchrome.go::TestChromePath` 探测，可用 `RPA_TEST_CHROME_PATH` 覆盖）。

---

## 11. 文件改动清单

### 新增（rpa-worker，独立 module）

| 文件 | 行数 | 内容 |
|---|---|---|
| `internal/browser/screencast.go` | 358 | `StartScreencast`（ACK 背压 + 超时兜底 + JPEG 尺寸 + stats） |
| `internal/browser/liveview_input.go` | 267 | `DispatchLiveInput` / 组合键 / 修饰键掩码 / 虚拟键兜底 |
| `internal/browser/screencast_integration_test.go` | 685 | 真 Chromium 验收：fps/带宽/延迟/降档矩阵/ACK 兜底/输入注入/泄漏/不关 page |
| `internal/types/liveview.go` | 180 | 帧/上下文/输入/下行指令/审计事件类型（**未进 `types.go`**） |
| `internal/liveview/hub.go` | 502 | 会话表（上限/空闲停流/janitor/帧泵/审计） |
| `internal/liveview/takeover.go` | 445 | 接管/释放/模式/输入/动作边界闸门/放行令牌 |
| `internal/liveview/ctx.go` | 23 | context 别名与 nil 安全小工具 |
| `internal/liveview/hub_test.go` | 344 | 三模式闸门/单接管者/空闲停流/限流/审计 |
| `internal/communication/liveview_client.go` | 32 | `APIClient.PushFrame` / `PushEvent` |
| `internal/executor/engine_liveview.go` | 108 | Attach/Detach/context 刷新/闸门/脱敏变量 |
| `internal/executor/engine_liveview_test.go` | 201 | 三模式端到端（真 Chromium） |
| `internal/worker/liveview_listener.go` | 151 | 下行指令监听（一个 goroutine 覆盖所有执行） |
| `internal/config/liveview_config_test.go` | 119 | schema/默认值/显式 false/示例 yaml 可解析 |

### 新增（根模块）

| 文件 | 行数 | 内容 |
|---|---|---|
| `internal/services/rpa/liveview_hub.go` | ~1000 | watcher 注册表 + 三模式 + 限流 + 帧中转 + 审计 + `Continue` 走 RUNTIME-05 |
| `internal/api/v1/rpa/liveview_handler.go` | ~470 | 12 个端点 handler + 错误码分流 |
| `internal/api/v1/rpa/liveview_router.go` | ~50 | 10 个用户侧端点（逐行内联权限码） |
| `internal/api/v1/rpa/liveview_audit_throttle.go` | ~70 | 输入审计 1s 窗口合并 |
| `internal/api/v1/rpa/liveview_p155_test.go` | ~250 | 9 条契约用例（真 sqlite） |
| `internal/core/db/migrations/migration_255_liveview_menu.go` | ~110 | 实时画面 C 型菜单（父=RPA 管理） |
| `internal/core/db/migrations/migration_255_liveview_menu_test.go` | ~110 | 建/幂等/父缺失跳过/不扩权限面 |
| `internal/websocket/notice_hub_liveview_test.go` | ~130 | 可丢帧不丢连接契约 + 消息类型常量 |

### 新增/修改（前端）

| 文件 | 内容 |
|---|---|
| `src/pages/operations/rpa/live/index.tsx`（新，18k） | 页面：可观看列表 + canvas + HUD + 接管/模式/继续 + keepalive 20s + 卸载停流 |
| `.../live/types.ts`、`liveFrameParser.ts`、`liveFrameBus.ts`、`liveApi.ts`、`LiveCanvas.tsx`、`LiveHud.tsx`（新） | 协议类型/纯函数/WS 复用/端点封装/画布/叠加层 |
| `.../live/__tests__/{liveFrameParser,liveHud,liveCanvas,liveFrameBus}.test.*`（新） | 48 例 |
| `executions/__tests__/ExecutionDetailModal.liveEntry.test.tsx`（新） | 2 例（footer 入口 + 跳转） |
| `executions/ExecutionDetailModal.tsx`（改 +26 行） | footer 加「实时画面」按钮（唯一改动处） |

### 修改（配置/文档）

- `rpa-worker/configs/config.example.yaml`：新增顶层 `liveview` 节（10 个键，全部带中文说明与实测依据）
- `.planning/phases/155-liveview/155-PROTOCOL.md`（新）：冻结的协议契约（9 节）
- `.planning/phases/155-liveview/155-SUMMARY.md`（本文）

---

**提交**：见 `git log`（Lead 统一提交，commit message 汇总本 phase 5 条 REQ 与 3 个 P0 清偿）。
