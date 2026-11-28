# Phase W6 SUMMARY — 部署前置补齐（REC-01 / REC-02 / REC-03）

| 项 | 值 |
|---|---|
| **Phase** | W6（v1.56 里程碑收尾波次，非 31 REQ 主线） |
| **执行日期** | 2026-11-25 |
| **前置确认** | `git log --oneline -8`：158（`b9f29e14`）与收尾修正（`d4147207`）均在 main ✅ |
| **状态口径** | `code-complete · verification-pending` —— **未 SHIPPED**（端到端真机链路登记 UAT，见 §6） |
| **通道设计** | `.planning/phases/w6-rec-and-schedules/W6-PLAN.md`（先设计后动手，硬约束遵守） |
| **分支** | `main` |

---

## 0. 一句话结论

三条 REQ 全部代码侧落地且有本机实证：

- **REC-01**：前端 → 后端 → Worker → 真 Chromium 的完整录制/拾取通道已打通，真 Chromium 端到端 PASS（3 个真实操作 → 4 条动作 + 4 条中文步骤 + 上行产出验证；拾取 7 级阶梯回传；并发上限可读拒绝）。
- **REC-02**：调度管理前端页完整交付（列表 / 简单选项+高级折叠 / 启停 / 删除二次确认 / migration 258 双方言幂等）。
- **REC-03**：前端 ActionType 与 Go 侧 26 值同源，三方一致性测试（Go ↔ 后端镜像 ↔ 前端枚举 ↔ stepLabels）缺一即红，穷尽 switch 把双源风险压到编译期。

---

## 1. REC-01 录制/拾取 HTTP 端点接线 — ✅ PASS

### 1.1 通道设计（详见 W6-PLAN.md §2，要点复述）

| 决策 | 内容 |
|---|---|
| **帧通道复用 155 liveview** | 录制会话的画面帧走既有 `execution:liveview:{sessionId}` 频道 + `rpa_liveview_frame` WS 消息 —— 帧的语义（Worker 浏览器画面 + viewport 坐标系 + 可丢广播 + ACK 背压）与 liveview 完全相同，另立一套必然产生第二套坐标系约定 |
| **会话通道独立** | `recording:session:{sessionId}` 频道（Worker `PSubscribe("recording:session:*")`）—— 录制不是「一次执行」：没有 execution 行、不进执行状态机、不写执行审计，硬塞 execution 命名空间会污染 158 刚清完的假数据 |
| **上行不重开 WS** | Worker 保持纯 HTTP client，新增 3 个上行端点挂 `WorkerAuth`（与 155 同款处置） |
| **并发约束在 Worker 裁决** | `worker.recording.max_sessions`（默认 2，复用 155「并发推流上限」配置风格）；后端只展示上限与可读错误 |
| **超时三层** | 前端心跳 20s / 后端 janitor 闲置 5 分钟 / Worker 兜底 janitor 回收孤儿 page（不关 browser，遵 154 硬语义） |
| **权限不扩面** | 用户侧端点全部挂既有 `ops:rpa:task:add`（录制是「做任务的手段」，产物存进任务脚本）—— 与 155「不新增权限码」同款裁定 |

### 1.2 交付物

**Worker（rpa-worker）**
| 文件 | 内容 |
|---|---|
| `internal/types/recording.go`（新） | 协议类型（频道名 / 指令 / 状态 / 产出 / 拾取）。**归属协议第 4 条：独立文件，`types.go` 零改动** |
| `internal/recording/hub.go`（新） | 会话管理：`Start`（Acquire page + 导航 + Recorder.Start）/ `Stop`（Stop → BuildRecording → 上行 → Release）/ `Pick`（PickInFrames 7 级阶梯 → 上行）/ `Keepalive` / `Abort` / 兜底 janitor |
| `internal/communication/recording_client.go`（新） | `ReportStatus / ReportResult / ReportPick` 三个 HTTP 上行 |
| `internal/communication/redis_client.go` | 追加 `SubscribeRecording`（PSubscribe 模式订阅） |
| `internal/config/config.go` | 追加 `RecordingConfig`（`worker.recording.*`，默认 enabled=true / max_sessions=2 / idle_timeout=5min）+ `applyRecordingDefaults` |
| `internal/worker/recording_listener.go`（新） | 指令分派循环 + 状态周期上报（与 liveview_listener 同族：单 goroutine 覆盖全会话） |
| `internal/worker/worker.go` | 接线：initComponents 构造 hub、Start 起两个 goroutine、cleanup Close |
| `internal/recording/hub_e2e_test.go`（新） | **真 Chromium 端到端验收**（§1.4） |

**后端（根模块）**
| 文件 | 内容 |
|---|---|
| `internal/services/rpa/recording_service.go`（新） | `RecordingHub`：会话表 + 状态机（created→starting→recording→stopped/failed/expired/aborted）+ Redis 下行 + 闲置回收 janitor + **产出落库**（`saveActionsToTask` 走既有 `TaskService.Update` 写回任务脚本） |
| `internal/services/rpa/service.go` | ServiceGroup 挂 `RecordingService`（cache/noticeHub/db/taskSvc 注入） |
| `internal/api/v1/rpa/recording_handler.go`（新） | 用户侧 6 端点（start/stop/pick/keepalive/abort/get）+ Worker 上行 3 端点；operlog 埋点「RPA录制」 |
| `internal/api/v1/rpa/rpa_router.go` | **只在 `SetupRPARouter` 尾部追加** `SetupRecordingRouter(...)` 一行 + `SetupPublicWorkerRouter` 尾部追加 3 个 Worker 上行端点（归属协议第 1 条：既有行零改动） |

**前端**
| 文件 | 内容 |
|---|---|
| `recording/types.ts`（新） | 会话状态 / 7 级阶梯 / 产出类型（与后端同构） |
| `recording/recordingApi.ts`（新） | 6 个 API 封装（走 `@/lib/api`，禁裸 axios） |
| `recording/RecordingPanel.tsx`（新） | 录制面板：start → 复用 155 `LiveCanvas` 画帧（sessionId 在帧通道扮演 executionId）→ 心跳+轮询快照 → 拾取浮层（7 级逐级可用性）→ stop → `toFrontendActions` 回填 |
| `recording/__tests__/recordingPanel.test.ts`（新） | 回填映射 + 终态判定（7 用例） |
| `tasks/index.tsx` | wizard 的 `onOpenRecorder` **从指引文案改为真按钮** → 打开录制面板 Modal → 产物经 `initialActions` 回填向导（不回退 156 交互） |

### 1.3 关键实现裁定

- **产出落库在后端**：Worker 停止时把 actions 通过 HTTP 上行（权威结果），后端 `ReportResult` 收到后 (a) 存会话快照（前端轮询取回填向导）；(b) 若 start 时指定了 `taskId`，直接写回该任务的 `script` 列 —— 满足 REQ 原文「停止 → 产出 actions JSON 存为该 task 的脚本」。
- **录制画面只可看**：员工在真实页面操作（录制器采集真实事件），画布不做输入注入 —— 与 155 的「接管才可注入」语义清晰分层。
- **重复 stop 幂等**：`stopOnce` 保证只产出一次；重复调用返回缓存结果，前端可安全重复点「完成录制」。

### 1.4 验收（真 Chromium，非模拟）

`rpa-worker/internal/recording/hub_e2e_test.go` 三条，全部 **PASS**：

```
=== RUN   TestRecordingHub_RealChromium_ThreeActions
    录制产出 8 个事件 → 4 个动作
      步骤 1: 打开网页
      步骤 2: 在「请输入金额」填入内容
      步骤 3: 选择「住宿费」
      步骤 4: 点击按钮「提交」
--- PASS: TestRecordingHub_RealChromium_ThreeActions (2.86s)

=== RUN   TestRecordingHub_RealChromium_PickTierLadder
    拾取到 <button> 文本="提交" 推荐="text=提交报销单"（第 2 级，评分 90）
--- PASS: TestRecordingHub_RealChromium_PickTierLadder (1.92s)

=== RUN   TestRecordingHub_SessionLimit
    超限错误文案: 本机同时进行的录制已达上限 1 个，请先结束其中一个再试
--- PASS: TestRecordingHub_SessionLimit (1.68s)
```

断言逐条覆盖 REQ 验收口径：
1. **真 Chromium**（`TestChromePath` 探测真机 Chrome；headless 真实启动）；
2. **3 个真实动作**（真实鼠标键盘事件：填金额 / 选下拉 / 点提交 —— 不 evaluate 改 DOM）；
3. **停止后产出中文步骤**（4 条中文步骤，与动作等长、零技术术语）;
4. **直接可执行**：产出 schema 与 `types.Action` 一致（recorder 构造约束），156 的回放先例（`TestRecorder_RealChromium_LoginFillSubmit`，仍 PASS）已验证同 schema 动作可回放执行；
5. **上行验证**：Upstream 恰收到 1 次产出上报（sessionId/动作数一致）+ 1 次拾取上报（7 级阶梯非空）；
6. **拾取 7 级阶梯**：`PickInFrames` 按坐标命中 `<button>`，阶梯逐级可用性回传，推荐级评分 90；
7. **并发上限**：超限返回可读中文错误（员工能看懂）。

---

## 2. REC-02 调度管理前端页 — ✅ PASS

### 2.1 交付物（`pages/operations/rpa/schedules/`，新建目录）

| 文件 | 内容 |
|---|---|
| `index.tsx` | 列表页：筛选（名称/状态）+ 表格 + 新建按钮 |
| `columns.tsx` | 列：名称 / 关联任务（显示任务名非 id）/ 运行方式（中文）/ 启停 Switch / 最近触发 / 下次触发 / 操作 |
| `modals/ScheduleModal.tsx` | 新建/编辑弹窗 |
| `cronPresets.ts` | 简单选项 ↔ cron 纯函数换算（可逆） |
| `api.ts` | 后端 7 端点封装（走 `@/lib/api`） |
| `__tests__/schedules.test.tsx` | **24 个用例**（换算互逆 / 列渲染 / 表单行为） |

### 2.2 员工友好设计

| 员工看到 | 实际提交 |
|---|---|
| 「每天 09:00」 | `0 9 * * *` |
| 「每周一 09:00」 | `0 9 * * 1` |
| 「每隔 15 分钟」 | `*/15 * * * *` |
| （默认折叠的）高级设置 | cron 原样 |

- cron 表达式藏在默认折叠的「高级设置」里；三个简单选项覆盖 REQ 要求的「每天/每周/间隔分钟数」。
- 删除走二次确认；启停即时生效（调 `POST /:id/status`）。
- 权限码接 `ops:rpa:schedule:list/add/edit/remove`（migration 257 已播种，本 phase 只消费不扩面）。

### 2.3 路由 migration

`migration_258_rpa_schedule_menu.go`（参照 255/256 先例）：
- C 型页面菜单「定时调度」（`component=pages/operations/rpa/schedules/index`，父=「RPA 管理」，order=18，固定 UUID `a1b2c3d4-0258-4000-8000-000000000258`）；
- 幂等（先查后建）+ 父菜单缺失 WARN 非阻断 + 无方言专属 DDL（双方言幂等）；
- `registry.go` 登记 `{258, "rpa_schedule_menu"}` + `database.go` **PG/sqlite 两分支注册**；
- **不改** `componentLoader.tsx` / `DynamicRoutes.tsx`（归属协议第 6 条）；
- migration 测试 PASS（幂等 / 父菜单缺失跳过 / 指向 schedules 页面 / 不含权限码）。

---

## 3. REC-03 前端 ActionType 枚举对齐 — ✅ PASS

### 3.1 改动

| 文件 | 改动 |
|---|---|
| `types/rpa.ts` | `ActionType` 补齐至 **26 个**（去掉 Go 侧不存在的 `press`，新增 evaluate/waitFor/close/loop/pause/condition/autologin/assert/doubleClick/rightClick/drag/keyboard/script）；**常量数组是真相源**：`RPA_ACTION_TYPES`（`as const`）→ `type ActionType = (typeof RPA_ACTION_TYPES)[number]`，类型与值不可能再分叉 |
| `constants.tsx` | `ACTION_TYPE_TEXT_MAP` 类型收紧为 `Record<ActionType, string>`（编译期强制 26 全覆盖）；`ACTION_TYPE_OPTIONS` 从手写 9 项改为**遍历 RPA_ACTION_TYPES 派生**（消除「下拉选不到多数动作」的漂移） |
| `wizard/stepLabels.ts` | `HUMAN_VERBS` 收紧为 `Record<ActionType, string>`（键集编译期 == 26 枚举）；`describeAction` 字符串分派改为**穷尽 switch**（default 分支 never 断言，漏 case 编译失败）—— 落实过程中穷尽断言立即抓到漏网的 `drag` case，机制有效 |
| `live/liveFrameParser.ts` | 运行时标签表补 `keyboard`（Go 侧同源）；`press` 保留为历史兼容键（仅旧数据显示，不作为新动作来源） |

### 3.2 三方一致性测试（缺一即红）

**Go 侧** `internal/services/rpa/action_type_consistency_test.go`（4 用例 PASS）：
- 读 `rpa-worker/internal/types/types.go` 抽 26 常量 ↔ 后端镜像 `workerActionTypes` ↔ 前端 `types/rpa.ts` 枚举数组，三方双向比对；
- 断言 Go 集合恰 26 个、前端枚举无重复、每个动作中文名非空且不含 ASCII 字母。

**前端侧** `wizard/__tests__/actionTypeConsistency.test.ts`（8 用例 PASS）：
- `RPA_ACTION_TYPES` 长度 === 26；
- `HUMAN_VERBS` 键集 === 枚举集；
- `ACTION_TYPE_TEXT_MAP` 覆盖全部 26 个；
- 中文名全中文无 ASCII。

**证据**：
```
=== RUN   TestActionTypeConsistency_WorkerActionTypesMirror
--- PASS: TestActionTypeConsistency_WorkerActionTypesMirror (0.00s)
=== RUN   TestActionTypeConsistency_GoSetIsExactly26
--- PASS: TestActionTypeConsistency_GoSetIsExactly26 (0.00s)
=== RUN   TestWorkerActionTypes_MirrorsWorkerTypesGo
    动作枚举同源校验通过：worker 26 个，后端镜像 26 个
--- PASS: TestWorkerActionTypes_MirrorsWorkerTypesGo (0.00s)
```

---

## 4. 闸门验证结果（本机）

| 闸门 | 命令 | 结果 |
|---|---|---|
| 根模块构建 | `go build ./...` | ✅ exit 0 |
| Worker 构建 | `cd rpa-worker && go build ./...` | ✅ exit 0 |
| 后端 RPA 包测试 | `go test ./internal/api/v1/rpa/ ./internal/services/rpa/` | ✅ ok（api 23.7s 含既有契约冒烟） |
| 迁移测试 | `go test ./internal/core/db/migrations/` | ✅ ok（含 258 新测试） |
| Worker 全量测试 | `cd rpa-worker && go test ./...` | ✅ 8 包全 ok（含 recording 3 条真 Chromium） |
| 前端类型检查 | `npx tsc --noEmit -p tsconfig.app.json` | ✅ 0 error |
| 前端 RPA 测试 | `npx vitest run src/pages/operations/rpa/` | ✅ **18 files / 169 tests passed**（含新增 recording 7 + schedules 24 + 枚举一致性 8） |
| 前端生产构建 | `npm run build` | ✅ built in 38.75s |
| 路由注册表 | `node scripts/scan-routes.cjs` | ✅ 重生成：总路由 **753 → 762**（+9：recordings 6 + workers/recording 3），写路由 340，deferred 153 |

**本次新增测试统计**：Go 侧 4（枚举一致性）+ 5（migration 258）+ 3（真 Chromium 端到端）；前端 7（录制回填）+ 24（调度页）+ 8（枚举一致性）= **51 个**。

---

## 5. 归属协议越界检查

| 协议条 | 要求 | 实际 | 结论 |
|---|---|---|---|
| 1 | `rpa_router.go` 只在 `SetupRPARouter` 尾部追加 | 尾部追加 `SetupRecordingRouter` 1 行 + `SetupPublicWorkerRouter` 尾部追加 3 个 Worker 上行端点；既有行零改动 | ✅ |
| 2 | 不改 154 冻结的 `browser/manager.go` / `page_manager.go`；recorder/picker 只加不改 | `manager.go` / `page_manager.go` / `pool.go` / `recorder.go` / `picker.go` **git diff 均为空**；新会话管理放独立包 `internal/recording/` | ✅ |
| 3 | 不回退 152 AIScriptEditor 真调用 / 155 live 画布 / 156 wizard 交互 | `AIScriptEditor.tsx` 仅 1 处类型适配（`ACTION_TYPE_TEXT_MAP[d.type as ActionType]`，随 REC-03 类型收紧，调用逻辑未动）；`live/*` 只读复用（仅 `liveFrameParser.ts` 标签表按 REC-03 授权修改）；wizard 交互零改动（仅接线 `onOpenRecorder`） | ✅ |
| 4 | worker 新会话类型放独立文件 | `types/recording.go`；`types.go` 零改动 | ✅ |
| 5 | 零新增 Go/npm 依赖 | `go.mod` / `package.json` 零新增（`google/uuid` 为既有依赖） | ✅ |
| 6 | 构建产物只落 `out/` | Go 构建不落盘；前端 dist 为既有行为 | ✅ |
| 硬约束 | 通道设计先写后动手 | `W6-PLAN.md` 先落（含频道命名/schema/超时取舍）再实现 | ✅ |
| 硬约束 | 员工可见文案全中文零技术术语 | 录制面板/调度页文案全中文；cron 藏在默认折叠的高级设置 | ✅ |

---

## 6. verification-pending 项清单（未验收，不得当已完成）

| # | 条目 | 类别 | 阻塞原因 / 解锁路径 |
|---|---|---|---|
| 1 | **REC-01 全链路真机 UAT**：浏览器里点录制 → 录 3 个动作 → 停止 → 步骤列表出现 3 条中文步骤 → 直接可执行。本机已验证 Worker 侧真 Chromium 链路（§1.4）与前端单元层；**浏览器↔后端↔Worker 三进程联动**（Redis 下行 + 帧回传 + 轮询回填）需要起完整环境跑一遍 | UAT | 需同时起后端（PG/sqlite + Redis）+ Worker + 前端 dev server 的完整本机环境 |
| 2 | REC-01 拾取浮层的**浏览器端交互验收**（悬浮 → 7 级阶梯浮层 → 点击写入动作 selector）在真浏览器里的操作体验 | UAT | 同上；浮层数据链路已由真机测试覆盖（7 级阶梯回传断言） |
| 3 | REC-02 页面**真连后端 CRUD** + 建 1 分钟后的调度 → 页面可见 next_run_time → 到点 executions 出现 `trigger_type=schedule` 记录。后端到点触发已由 158 本机真跑（158-SUMMARY §2.3）；前端页↔后端联调需起完整环境 | UAT | 同上 |
| 4 | `ops:rpa:schedule:*` 四码 + `ops:rpa:task:add`（录制端点）的**角色授权实测**（非超管持码角色能否正常操作） | 验证类 | 需真实角色/菜单授权环境（158 §6 #5 既有项延续） |
| 5 | migration 258 在**生产 PG** 上执行（menu seed 落库） | 验证类 | 本环境无生产 PG 写授权（158 §6 #1 同款） |
| 6 | 录制会话在**多 Worker 集群**下的定向投递（start 指定 workerId 的路由正确性） | 能力边界 | 单机部署形态无此场景；集群部署时验证 |
| 7 | 既有项：158 §6 #6（多实例调度重复触发收敛）、#8（tests/e2e exit 127 环境问题） | 既有 | 原样延续，非本 phase 引入 |

---

## 7. REQ 结论

| REQ | 结论 | 关键证据 |
|---|---|---|
| REC-01 录制/拾取端点接线 | **PASS**（Worker/后端/前端三层代码 + 真 Chromium 会话层端到端） | 真 Chromium：3 真实操作 → 4 动作+中文步骤+上行验证 / 拾取 7 级阶梯（推荐 text=提交报销单，评分 90）/ 并发上限可读拒绝；通道设计先行（W6-PLAN §2）；路由尾追加零改行 |
| REC-02 调度管理前端页 | **PASS** | schedules 页 6 文件 + 24 测试；三简单选项+cron 高级折叠；启停/删除二次确认/立即运行；migration 258 双方言幂等 + 两处注册 + registry 登记；权限码复用 257 播种 |
| REC-03 ActionType 枚举对齐 | **PASS** | 前端 26 值同源（常量数组为真相源）；HUMAN_VERBS/ACTION_TYPE_TEXT_MAP 编译期全覆盖；穷尽 switch 抓到漏网 drag；三方一致性测试 Go 4 + 前端 8 全 PASS；tsc 0 error + build 成功 |

**整体状态**：`code-complete · verification-pending` —— 不标 SHIPPED；§6 共 7 项验证待办（其中 3 项为本 phase 的浏览器端 UAT）。
