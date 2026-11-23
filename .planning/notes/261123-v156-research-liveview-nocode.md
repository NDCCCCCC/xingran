---
created: 2026-11-23
topic: v1.56 领域调研（LIVEVIEW / NOCODE / 选择器稳定性）
status: research-input（供 Phase 154/155/156 planner 使用）
sources:
  - https://chromedevtools.github.io/devtools-protocol/
  - https://stackoverflow.com/questions/71437739/page-startscreencast-chrome-devtools-protocol-low-fps-issue
  - https://github.com/puppeteer/puppeteer/issues/478
  - https://thume.ca/2022/05/15/latency-testing-streaming/
  - https://arxiv.org/pdf/2509.12049
  - https://arxiv.org/pdf/2507.22358
  - https://www.robotmk.org/en/blog/web-selectors/
  - https://validate.qa/blog/selector-strategies-data-testid-vs-role-vs-css
  - https://www.uipath.com/rpa/what-is-citizen-development
---

# v1.56 领域调研（决策支撑）

## 1. CDP screencast 的硬约束（决定 LIVEVIEW-01 的参数）

- `Page.startScreencast` **必须逐帧 ACK**（`Page.screencastFrameAck`），不 ACK 会立刻停止发帧。这是节流的天然阀口：前端消费速度即背压。
- 实测经验值普遍偏低：headless 模式下 `startScreencast` 常见 **1–5 fps**，headful 也就 5–10 fps；`nth-frame` 降采样 + 降 quality 是提高 fps 的常规手段。
- 因此 LIVEVIEW-01 的「≥8fps」目标必须靠 **降分辨率（maxWidth 1024/1280）+ jpeg quality 40–60 + nth-frame=2 + ACK 节流** 组合达成，并在 FLEET-03 中实测落数，不能照抄「60fps 远程桌面」的期望。
- 参考延迟量级：远程浏览在 16ms RTT 下仍会叠加 ~67ms 端到端（含渲染与压缩），所以输入注入的**可感知延迟下限约 80–120ms**，UI 需要「已接管」的显式状态而不是追求零延迟。
- 结论：**不做 WebRTC**（要另建编码链路与 STUN/TURN）；CDP 帧流 + ACK 背压已足够「看见 + 接管」，且与既有 `go-rod` 依赖零冲突。

## 2. 人机接管（human-in-the-loop takeover）的成熟模式

- 学术与工程共识：接管的核心不是「能不能注入输入」，而是**控制权转移的状态机 + 情境感知（out-of-the-loop 问题）**。接管前必须让人知道「现在系统在做什么、下一步要做什么」。
- 对应到本里程碑：「画面 + 叠加层（步骤/日志/变量）」是情境感知；`auto/copilot/manual` 是控制权档位；`copilot` 默认（关键节点暂停等确认）符合「一般员工」的信任曲线。
- `Magentic-UI` 一类工作台把「计划可视化 + 逐步确认 + 随时打断」作为一等能力，与本里程碑的 LIVEVIEW-04 + RUNTIME-05 状态机同构，可直接参照其交互骨架而非自创。

## 3. 选择器稳定性阶梯的实证排序

- 业界共识的稳定性从高到低：`data-testid` > `role + accessible name` > `aria-label` > 稳定 `id`/`name` > 可见文本 > 结构路径（nth-child 链） > 绝对 XPath。
- 结构路径与绝对 XPath 在 DOM 微调后失效率最高，应仅作最后兜底；`getByRole` 的争议在于它绑定无障碍语义，但**对自研内网系统**（无 testid 传统）它是目前最可用的稳定锚点。
- 结论（ENGINE-03）：阶梯顺序如上；录制生成的选择器**优先落 testid/role**；命中层级要回传（前端展示「本次靠第几级命中」），这既是自愈信号也是选择器学习的输入。
- 补充：`rod` 的 `Has/HasR/ElementR/Search` 分别对应 CSS / 正则文本 / XPath / 查询 DSL，能覆盖阶梯全部层级，无需引入新依赖。

## 4. 一般员工（citizen developer）的产品形态

- UiPath StudioX / Power Automate Desktop 的共同抽象：**录制 → 可视化步骤列表（自然语言描述）→ 少量参数填写 → 一键运行**；编辑器面向「业务人员」而非「程序员」，动作以业务语义命名（"打开网页"/"填入内容"/"点击按钮"）。
- 结论（NOCODE-04）：三步向导 + 中文步骤描述 + 开箱模板是行业验证过的最小可用形态；不引入流程图/DSL 编辑器（那是给专业开发者的，与目标用户相反）。

## 5. 对本里程碑的直接影响

| REQ | 由调研得出的硬约束 |
|---|---|
| LIVEVIEW-01 | 必做 ACK 节流；fps 目标按「降档可达成」设定；带宽实测入 FLEET-03 |
| LIVEVIEW-02 | 注入延迟 80–120ms 属正常；UI 必须显示接管状态与坐标缩放 |
| LIVEVIEW-03 | `copilot` 默认；控制权转移必须是显式状态机（与 RUNTIME-05 共用） |
| LIVEVIEW-04 | 叠加层是「情境感知」必需品，不是装饰 |
| NOCODE-01/04 | 录制 + 中文步骤描述 + 三步向导；不做流程图编辑器 |
| ENGINE-03 | 7 级阶梯顺序与「优先 testid/role」约定按上述实证排序 |
