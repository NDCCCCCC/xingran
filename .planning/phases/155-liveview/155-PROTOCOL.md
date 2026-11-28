# Phase 155 LIVEVIEW 协议契约（冻结 · 供实现与验收共用）

> 冻结时间 2026-11-24。任何一侧改动必须同步本文件 + 另一侧 owner 确认。
> 归属：Worker（rpa-worker）／后端（internal/services/rpa + api/v1/rpa）／前端（pages/operations/rpa/live）。

## 0. 通道选型（不改）

- 画面帧：**Worker --HTTP POST--> 后端 --既有 NoticeHub WS--> 前端 Canvas**
  不新建 Worker→后端 WS 长连（Worker 现为纯 HTTP client；第二条长连会引入重连/鉴权双套生命周期）。
- 下行输入/控制：**前端 --HTTP POST--> 后端 --Redis PubSub--> Worker --CDP Input.dispatch\***
- 不引入 WebRTC / VNC / 浏览器扩展（Out of Scope 明文）。

## 1. 消息类型常量

| 位置 | 常量 | 值 |
|---|---|---|
| backend `internal/websocket` | `MessageTypeRPALiveFrame` | `rpa_liveview_frame` |
| backend `internal/websocket` | `MessageTypeRPALiveState` | `rpa_liveview_state` |
| worker `types` | `LiveViewPattern` | `execution:liveview:*` |
| 两端 | `LiveViewChannel(id)` | `execution:liveview:{executionId}` |

WS 信封复用既有 `NoticeMessage`：`Type` 透传上述类型，`Content` 是下表 JSON 的字符串。
**注意**：`NoticeMessage.ExecutionID`/`NoticeID` 不填业务 ID（前端从 Content 解析）。

## 2. Worker → 后端（HTTP，WorkerAuth，`X-Worker-Token`）

### `POST /rpa/workers/liveview/frame`
体 = `types.LiveViewUpstreamFrame`：

```json
{
  "frame": {
    "executionId": "uuid", "seq": 12, "timestamp": 1700000000123,
    "data": "<base64 jpeg>", "byteSize": 51234,
    "width": 1280, "height": 720,
    "deviceWidth": 1920, "deviceHeight": 1080,
    "scrollOffsetX": 0, "scrollOffsetY": 0, "pageScaleFactor": 1
  },
  "context": { "executionId": "...", "taskName": "...", "workerId": "...", "workerName": "...",
    "step": 3, "totalSteps": 10, "actionType": "click", "actionDesc": "...",
    "status": "running", "mode": "copilot", "takeoverBy": "", "takeoverAt": 0,
    "lastLog": "...", "variables": {"user":"alice","password":"***"}, "startedAt": 1700000000000 }
}
```

后端行为：把 frame 转 `rpa_liveview_frame` 推给该执行的 watcher（只推 watcher，不是全员广播）；
`context` 转 `rpa_liveview_state` 推给 watcher。**帧必须可丢**（后端 WS 队列满时丢帧不丢连接，
见 §6），context 不可丢。

### `POST /rpa/workers/liveview/event`
体 = `types.LiveViewAuditEvent`：

```json
{ "executionId":"uuid", "event":"watch_start|watch_stop|takeover|release|mode_switch|input",
  "operator":"alice", "reason":"", "mode":"copilot", "detail":"", "timestamp":1700000000000 }
```

后端行为：写 `sys_rpa_audit_logs`（`ResourceType=execution`，`ResourceID=executionId`，
`Action="start"` 用于 watch_start / `"stop"` 用于 watch_stop / `"update"` 用于 takeover|release|mode_switch|input，
`NewValue` 存 `{"event":..., "mode":..., "reason":..., "detail":...}`，`OperatorName=operator`）。
写失败 500（审计不可静默丢弃，这是安全项）。

## 3. 前端 → 后端（HTTP，`ops:rpa:execution:list` 读权限 + 数据范围）

| 方法 路径 | 体 | 说明 |
|---|---|---|
| `POST /rpa/executions/:id/liveview/watch` | `{}` | 开始观看；返回 `{mode, maxStreams, streaming}`；超限 429 + 可读 message |
| `POST /rpa/executions/:id/liveview/keepalive` | `{}` | 心跳（20s 一次），刷新闲置计时 |
| `POST /rpa/executions/:id/liveview/stop` | `{reason}` | 停流 |
| `POST /rpa/executions/:id/liveview/takeover` | `{force:bool}` | 接管；已被占用且 force=false → 409「已被他人接管」 |
| `POST /rpa/executions/:id/liveview/release` | `{reason}` | 释放接管 |
| `POST /rpa/executions/:id/liveview/mode` | `{mode:"auto"|"copilot"|"manual"}` | 切模式；非法 400 |
| `POST /rpa/executions/:id/liveview/continue` | `{reason}` | 放行被暂停的动作 |
| `POST /rpa/executions/:id/liveview/input` | `LiveInputCommand` (§4) | 输入注入；非接管者 403 |
| `GET /rpa/executions/:id/liveview/state` | - | 叠加层情境 + 推流统计快照 |

所有端点写审计（除 `state`/`keepalive`）。

## 4. 下行输入体（`LiveInputCommand`）

```json
{ "executionId":"uuid", "kind":"mouse_move|mouse_down|mouse_up|mouse_wheel|key_down|key_up|text",
  "x": 100.5, "y": 200.25, "button":"left|right|middle|none", "clickCount":1,
  "modifiers":0, "deltaX":0, "deltaY":0,
  "key":"a", "code":"KeyA", "text":"", "virtualKey":65,
  "operator":"alice", "timestamp":1700000000000 }
```

- `x`/`y` 是 **viewport CSS 像素**（前端已完成画布→viewport 换算）。
- `modifiers` 位掩码：Alt=1, Ctrl=2, Meta=4, Shift=8。
- `key`/`code`/`virtualKey` 由前端从 DOM KeyboardEvent 原样取；Worker 仅在 `virtualKey=0` 时按 key 名兜底。

## 5. 后端 → Worker（Redis PubSub `execution:liveview:*`）

体 = `types.LiveViewCommand`：

```json
{ "action":"watch|stop|input|takeover|release|mode|keepalive|continue|abort",
  "executionId":"uuid", "operator":"alice", "mode":"copilot",
  "input": { …§4… }, "timestamp":1700000000000 }
```

Worker 用 `PSubscribe("execution:liveview:*")` 一次订阅全部执行（与 `execution:cancel:*` 同族先例）。
频道名取 `execution:liveview:{executionId}`，Worker 从 channel 反解 executionId。

## 6. 背压与可丢语义（LIVEVIEW-01/05 的核心）

- **帧 ACK 即背压**：Worker 的 screencast 逐帧 ACK，ACK 在「上报完成后」才发 →
  上报慢 = ACK 慢 = Chrome 少发帧。超时 `AckTimeout`（默认 800ms）强制 ACK 并计 dropped。
- **后端 → 前端可丢帧**：既有 `NoticeHub` 的 `client.send` 满时**会 close 整个连接**（
  见 `notice_hub.go:141-158`），帧流不能走这条路径。必须新增
  `NoticeHub.PublishLiveFrameToUsers(userIDs, payload)`：队列满时**丢弃该帧**并计数，
  绝不 close 连接。`rpa_liveview_state`（叠加层）不可丢，走正常路径。
- **限流**：单 Worker 推流上限 `worker.liveview.max_streams`（默认 2）。超限 watch 返回
  可读错误（后端 429）。
- **空闲停流**：Worker 侧 `IdleTimeout`（默认 30s）无下行指令即停流；后端侧同样在
  `IdleTimeout` 后把该 watcher 标记为已断开并停发帧（双保险，防止 Worker 侧 janitor 失效）。

## 7. 安全（LIVEVIEW-05）

- 帧与输入只对有 `ops:rpa:execution:list` 且数据范围可见该执行的人开放（复用既有读权限，
  不新增权限码；`watch/keepalive/stop/state/continue/input/takeover/release/mode` 全部挂该码）。
- 同一 execution/page 同时只允许一个接管者；抢占需 `force=true`（前端二次确认后发送）。
- 权限失效即断流：后端每次帧推送前校验 watcher 仍持有该执行读权限（缓存 5s），失效即移除 watcher。
- 审计四类事件必写：`watch_start` / `watch_stop` / `takeover` / `release`（外加 `mode_switch` / `input`）。

## 8. 生命周期硬语义（154-SUMMARY §7.4 继承）

- `stopScreencast` **只**停流；**不关 page、不关 browser**。
- 执行结束 → `Detach` → 停流；page 由 `Manager.Release` 回收。
- Worker `Shutdown` → `liveview.Hub.Close()` 停所有流；**不代替** `Manager.Close()`。

## 9. 前端契约

- 画布渲染：`<canvas>` + `new Image()` 逐帧绘制（`createImageBitmap` 可选，避免旧浏览器兼容问题）。
- 缩放映射：
  ```
  cdpX = (canvasX / canvasDisplayWidth)  * frame.deviceWidth
  cdpY = (canvasY / canvasDisplayHeight) * frame.deviceHeight
  ```
  其中 `canvasDisplayWidth/Height` 是 CSS 显示尺寸（`getBoundingClientRect()`），
  不是 canvas 的 `width/height` 属性（后者是 2x DPR 存储分辨率）。
- 叠加层：步骤 `step/totalSteps`、动作类型、最近日志、脱敏变量、`已接管 <name>` / `自动执行` 角标。
- 入口：`ExecutionDetailModal.tsx:317-329` 的 footer 加「实时画面」按钮（归属协议第 4 条）。
- 路由：**只**通过 `sys_menu` seed migration 注册（协议第 6 条），不改 `componentLoader.tsx` /
  `DynamicRoutes.tsx`。页面路径 `pages/operations/rpa/live/index`。
