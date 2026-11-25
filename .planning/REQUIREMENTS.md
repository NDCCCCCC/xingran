# Requirements: RuoYi-Next 运维管理系统

**Defined:** 2026-11-23
**Milestone:** v1.56 RPA 员工自助自动化 (Self-Service RPA for General Staff)
**Core Value:** 让不会写代码的一般员工能把重复的浏览器流程变成自动任务，并且随时能看见、能接管、能改；同时让这套东西在一台台 VDI 虚拟机上可预测地扩容。

## 立项基线（2026-11-23 全模块审计实证）

对 `internal/{api/v1,services,models}/rpa` + `rpa-worker/` + 前端 `pages/operations/rpa` 逐文件审计、`go build`/`go test` 实跑、定向探针实测（探针文件已删）后确认的**必然失败路径**（均为 100% 复现，非偶发）：

| # | 缺陷 | 实证位置 | 影响 |
|---|---|---|---|
| D1 | `NewServiceGroup` 第 5 参传 `nil`（passwordCipher） | `rpa_router.go:24,51`；探针 panic: nil pointer | 凭证创建/更新必然 500 |
| D2 | 三列表 `Status *int` vs 前端字符串 | 探针 `cannot unmarshal string into ... status of type int` | 任务/Worker/执行列表状态筛选必然 400 |
| D3 | 任务名筛选用 `task_name`（真列是 `name`） | 探针 `no such column: task_name` | 任务名搜索必然 500 |
| D4 | 选择器模型无 `TableName()`，真表是 `sys_rpa_selector_success` | 探针 `no such table: selector_success_records` | 11 个 AI 选择器端点 + `LearnFromExecution` 必然 500 |
| D5 | 任务 Create/Update 丢 `Tags`；`Status>0`/`Priority>0` 判定 | `task_service.go:84-132` | 标签死字段；无法改回停用/优先级 0 |
| D6 | 执行记录从不写 `worker_id` | `execution_service.go:50`、`worker_service.go:222` | Worker 列恒空、按 Worker 筛选无效 |
| D7 | 会话放消息顶层，Worker 只读 `Variables["__sessionData"]` | `task_service.go:287` vs `engine.go:626` | 会话复用永不生效 |
| D8 | `applySessionData` 全是 `logger.Debug` 占位 | `engine.go:730,740` | cookie/localStorage/token 均未注入 |
| D9 | `extractTargetSystem` 恒返回 `"default"`；`deptID` 传空 | `task_service.go:372`、`:296` | 按系统匹配凭证、部门共享凭证失效 |
| D10 | 取消只改 DB，无 Redis 通道；Worker 无订阅 | `execution_service.go:Cancel` | 取消后被 Worker 覆盖回 success |
| D11 | `pause` 等 `worker:human-input:*`，后端只写表不 publish | `engine.go:499` vs `excel_service.go` | 任何 pause 必然超时失败 |
| D12 | 批量报告硬编码 `100/95/5`；创建/更新为空函数 | `excel_service.go:GetBatchExecutionReport` | 批量功能全是假数据 |
| D13 | 自动扩缩容配置 GET 返回常量、POST 只记 operlog | `worker_handler.go:358-390` | 配置不可持久化 |
| D14 | `WorkerHandler` 每实例 new redis.Client | `worker_handler.go:36-45` | 连接泄漏（公开组+主组各一） |
| D15 | 前端 AI 编辑器是 `setTimeout(1500)` + mock | `AIScriptEditor.tsx:100-101` | 后端 11 个 AI 端点零调用 |
| D16 | `useRPAProgress` 零调用方 | 全仓 grep | 实时进度未接线，执行页靠轮询 |
| D17 | 每 page 启动独立 Chrome 进程 | `chrome_page.go:MustLaunch()` | `max_pages` 语义错位，内存线性爆炸 |
| D18 | `sys_rpa_schedules` 有表有模型，零服务/零 API 读写 | `database.go:894` 注释自认 | 定时执行 RPA 事实上不可用 |
| D19 | `flow_handler/data_mapper/error_handling` 为死代码 | `NewFlowHandler` 等无调用方 | 维护面虚高 |
| D20 | Worker `types.go` 声明 upload/download/evaluate/condition 等，`engine.go` 无 case | switch 缺失 | 前端动作选项执行即 `unsupported action type` |

## 用户三条新方向 → 本里程碑落点

| 用户方向 | 落点 |
|---|---|
| 「一般员工可以使用其方便的做一些繁琐的重复的流程化任务，操作够简单」 | NOCODE 全部 + CONTRACT（先修 500/panic，否则谈不上可用） |
| 「满足一般浏览器的所有操作」 | ENGINE-01~05（动作全集 + 选择器阶梯 + 真实浏览器验收） |
| 「无头浏览器可视化页面…渲染一个页面出来展示现状，用户随时接管浏览器和 rpa 协作」 | LIVEVIEW-01~05（screencast 帧流 + 输入注入 + 三模式 + 叠加层 + 审计） |
| 「VDI 虚拟机管理…一 VM 多容器 vs 一容器多浏览器」 | FLEET-01~03 + `notes/261123-rpa-worker-capacity-model.md` |

**承载模型结论（结构性已定，数字待 FLEET-03 实测回填）:** 默认 **一 VM 一 Worker 容器 + 容器内 1 Chromium + N page（N = `worker.max_concurrency`）**；扩容粒度 **VM 不是容器**（由 VDI API 编排）；「一 VM 多容器」仅保留给多租户/多浏览器版本/滚动升级，且要求 VM ≥32GB、每容器 shm ≤1GB。

## v1.56 Requirements

### CONTRACT — P0 契约修复（先让已存在的功能真的能用）

- [ ] **CONTRACT-01**: 凭证链路端到端可用 —— 两处 `NewServiceGroup` 注入 `core.SM4Cipher`；`CredentialService` 对 nil cipher 显式报错而非 panic；凭证 CRUD + 会话创建/失效在 sqlite 与 PG 均通过真实 HTTP 层用例（不用服务层 fake cipher 顶替验收）。
- [ ] **CONTRACT-02**: 三张列表状态筛选前后端同构 —— `TaskListParams.Status` 改用任务枚举（0/1）、`WorkerListParams.Status` 与 `ExecutionListParams.Status` 改用状态字符串；前端任务页筛选值改为任务态（启用/停用）而非执行态字符串；`"pending"`/`"online"`/`""`/`null` 边界各有用例，非法枚举返回 400 且错误可读。
- [ ] **CONTRACT-03**: 任务搜索与字段无损 —— 任务名筛选改用真实列 `name`；Create/Update 保存 `tags`；`Status`/`Priority` 用指针语义区分「未传」与「传 0」；前后端优先级取值域统一（迁移或适配二选一，PLAN 中给出裁定）；建→查→改→查 逐字段 round-trip 断言。
- [ ] **CONTRACT-04**: 选择器学习表打通 —— 两模型补 `TableName()` 指向 `sys_rpa_selector_success`/`sys_rpa_selector_failure`；加入 `scripts/dbprovision` 与 sqlite AutoMigrate 清单；删除测试里手写 `CREATE TABLE selector_success_records` 的掩盖式 DDL，改为断言「真实迁移后可直接写入」；11 个 `/rpa/ai/selector/*` 端点各一条真库用例。
- [ ] **CONTRACT-05**: 前端不再用 mock 冒充 —— `AIScriptEditor` 实调 `POST /rpa/ai/generate`（含失败态/超时态/空结果态）；`useRPAProgress` 接入执行页替换轮询；两者均有组件测试断言「发出真实请求」而非 `setTimeout`。
- [ ] **CONTRACT-06**: 契约回归闸门 —— 新增 `internal/api/v1/rpa` HTTP 层冒烟（httptest + sqlite 真表，非 mock service），覆盖 6 条 P0 缺陷对应请求；纳入 CI 硬门禁；并把「禁止在 handler 测试里用自建表/mock 掩盖真实 schema」写入 `AGENTS.md` 或等价文档。

### RUNTIME — 执行闭环（让任务真的能跑完、能停、能干预）

- [ ] **RUNTIME-01**: 执行记录写 Worker 归属 —— Worker 首次上报进度/心跳时按 `worker_id` 反查 Worker 回写 execution 的 `worker_id`/`worker_name`（并发安全 + 幂等）；Worker 列、按 Worker 筛选、WebSocket 消息三处均有用例。
- [ ] **RUNTIME-02**: 会话与凭证真正生效 —— 后端把有效会话注入 `Variables["__sessionData"]`（顶层字段保留兼容）；Worker `applySessionData` 真正 `page.SetCookies` + localStorage 注入 + token 头注入；`extractTargetSystem` 改为可配置映射（任务/凭证显式 `targetSystem` 优先、其次 URL host 规则）；`deptID` 从执行上下文取真实值。
- [ ] **RUNTIME-03**: 取消可中断 —— 新增 Redis 通道 `worker:cancel:{workerId}:{executionId}`（或等价 stream 指令），Worker 订阅后 cancel 该任务 ctx 并上报 `cancelled` 终态；后端 `Cancel` 发布指令且对「已终态」「Worker 离线」返回明确语义；前端执行页提供取消入口（二次确认 + 权限码）。
- [ ] **RUNTIME-04**: 人工干预双向闭环 —— Worker `pause` 上报后写入 `sys_rpa_human_interventions`；后端 `SubmitHumanIntervention` 校验 `action ∈ resume/skip/abort` 后 publish 到 `worker:human-input:{workerId}:{executionId}`；Worker 收到后按 action 继续/跳过/中止；超时与「用户未响应」UI 可区分；全链路有真 Redis（miniredis 或等价）集成用例。
- [ ] **RUNTIME-05**: 执行状态机收敛 —— 显式状态机（pending→running→(paused)→success/failed/cancelled/timeout），非法迁移拒绝并记 WARN；`cancelled` 不得被后续 `success` 覆盖；`timeout` 由 Worker 侧 `TaskTimeout` 主动上报；状态迁移矩阵单测 + 幂等重放用例。

### ENGINE — 执行引擎与「一般浏览器所有操作」

- [ ] **ENGINE-01**: 浏览器承载改造 —— `Pool` 语义从「Chrome 进程池」改为「page 池」：`browser.Manager` 持有 1 个 `rod.Browser`、按需 `NewPage`；`Page.Close()` 只关 page，Chrome 崩溃时整池重建；`worker.max_concurrency` 与 `browser.max_pages` 合并为单一「并发页面数」配置（旧键保留兼容并告警）；N page 下 RSS 增长基线写入验收。
- [ ] **ENGINE-02**: 动作全集（浏览器常用操作零缺口）—— 实现并通过真实 Chromium 用例：`upload`（`Element.SetFiles`）、`download`（`Browser.WaitDownload` + 产物归档）、`evaluate`（JS 执行与返回值入变量）、`assert`（断言 + 失败即终止策略）、`hover`、`doubleClick`、`rightClick`、`drag`（含 HTML5 drag 兼容策略）、`keyboard`（组合键/快捷键）、`scroll`（元素内滚动）、`condition`（真条件分支，不再是空 case）、`close`（关闭当前 page/标签）、`switchTab`、`iframe`（进入/退出 frame）；未实现动作提交时被静态断言拦截（`types.go` 枚举与 `engine.go` switch 一致性测试）。
- [ ] **ENGINE-03**: 选择器阶梯与自愈 —— 元素定位按 `data-testid > role+name > aria-label > 稳定属性(id/name) > 文本 > 结构路径 > XPath` 阶梯尝试（`rod` 的 `Has/HasR/ElementR/Search`）；失败自动尝试下一级并记录命中层级；连续失败达阈值触发选择器学习告警（复用 `RecordFailure`）；阶梯评分单测 + 真实页面命中用例；约定「录制生成的选择器优先落 testid/role」。
- [ ] **ENGINE-04**: 变量与数据流 —— 变量作用域（任务级/循环项级/会话级）显式化；`${var}` 与嵌套 `a.b[0].c` 支持；变量未定义按策略（严格失败 / 空串，可配）；extract 结果回写变量并可被后续动作引用；敏感变量（凭证）在日志与上报中脱敏（复用 operlog 敏感键机制）。
- [ ] **ENGINE-05**: 真实浏览器验收基线 —— `rpa-worker` 建立可重复集成测试：容器内 Chromium 起真实页面（httptest 或内嵌 fixture），覆盖 ENGINE-02 全部动作 + ENGINE-03 阶梯 + 失败重试；`rpa-worker` 的 `go test ./...` 从「仅 browser 包有测试」变为 executor/worker/communication 均有真实用例；CI 以 `-short` 跳过重型用例但保留轻量兜底。

### LIVEVIEW — 无头浏览器可视化与人工接管

- [ ] **LIVEVIEW-01**: 实时画面帧流 —— Worker 对执行中 page 启 `Page.startScreencast`（`format=jpeg`、`quality`/`maxWidth` 可配、按客户端 ACK 节流），帧经 WebSocket 推前端；单路带宽与帧率有实测基线（默认目标 ≥8fps / ≤3Mbps@1280×720，可降档）；无执行时停流（不空转烧 CPU）。
- [ ] **LIVEVIEW-02**: 输入注入与接管 —— 前端画面上的点击/移动/滚轮/键盘事件映射为 CDP `Input.dispatchMouseEvent`/`dispatchKeyEvent` 注入同一 page；坐标系缩放正确（画布尺寸 ↔ viewport）；接管期间 RPA 动作暂停，释放后恢复；输入注入有权限码与操作审计（谁在何时接管了哪个执行）。
- [ ] **LIVEVIEW-03**: 三种协作模式 —— `auto`（全自动）、`copilot`（关键节点暂停等确认，默认推荐）、`manual`（完全人工）；模式可在任务级设置并在执行中切换（切换记审计与时间线事件）；copilot 的暂停点支持「动作级标记 + 失败自动暂停」两种来源；状态机与 RUNTIME-05 共用同一套枚举。
- [ ] **LIVEVIEW-04**: 画面叠加层与独立页 —— 画面上叠加步骤序号/总步数、动作类型、最近日志、关键变量（脱敏后）、「已接管/自动执行」角标；前端有独立「RPA 实时画面」页（`pages/operations/rpa/live` 或等价），可从执行详情跳转，支持多执行并行观看时的列表与切换。
- [ ] **LIVEVIEW-05**: 安全与资源边界 —— 画面流与输入注入仅对「有权查看该执行」的用户开放（复用 `ops:rpa:execution:list` + 数据范围）；同一 page 同一时刻只允许一个接管者（抢占需显式确认）；用户断开后 N 秒内停流；单 Worker 同时推流路数上限可配（默认 2），超限返回可读错误而非静默降级。

### NOCODE — 员工零门槛

- [ ] **NOCODE-01**: 页面内录制即生成 —— `Page.evalOnNewDocument` 注入监听器捕获 click/input/change/submit/select/navigate/key，按时间与去抖聚类成动作序列；录制过程生成中文「可读步骤描述」，结束即产出可直接保存的任务草稿；不依赖浏览器插件（零分发成本）。
- [ ] **NOCODE-02**: 元素拾取器 —— 在 LIVEVIEW 画面上「点选元素」直接生成选择器（按 ENGINE-03 阶梯生成候选并展示置信度），用户可一键换用次选；拾取结果写回录制/编辑中的任务；iframe 内也可用。
- [ ] **NOCODE-03**: 自然语言生成落真 —— 前端 AI 编辑器接 `POST /rpa/ai/generate`（CONTRACT-05），后端返回的 `actions` 必须通过 `types.go` 动作枚举校验（非法动作服务端拒绝并给出可读原因）；生成结果可「一键转录制/一键编辑」；AI 不可用时降级为「录制」入口而非报错死路。
- [ ] **NOCODE-04**: 三步向导 + 开箱模板 —— 新建任务向导固定三步：① 选模板或「录一段」 ② 选运行时机（手动/定时/立即一次） ③ 选凭证与通知；内置 ≥6 个开箱模板（登录取数、表单批量填报、报表下载、跨系统搬数、定时巡检、文件上传），模板带中文说明与示例数据；向导全流程有前端测试；「一般员工」路径的点击次数与字段数在 PLAN 中给出明确上限并断言。

### FLEET — 承载、容量与 VDI 编排

- [ ] **FLEET-01**: 扩缩容 provider 化 —— `ScalingService` 的 Docker 依赖抽象为 provider 接口：`ContainerProvider`（本地/单机，保留现 mock 供测试）与 `VDIVMProvider`（生产，走 `internal/services/vdi` 的 `CreateVM/StartVM/StopVM/ListVMs/GetVMInfo`）；按 `rpa.scaling.provider` 选择；provider 能力差异显式声明（如 VDI 不支持容器级缩容）。
- [ ] **FLEET-02**: 「一 VM 一容器一 Chromium」落地 —— 提供 VDI 镜像构建与首启脚本：预装 Chromium + worker + 单容器 systemd 单元；容器内 `max_concurrency = N`、shm/内存按 `notes/261123-rpa-worker-capacity-model.md` 参数表设置；docker-compose 与 VDI 部署两条路径各一份权威配置，CI 校验关键参数不漂移。
- [ ] **FLEET-03**: 容量实测矩阵回填 —— 在真实 VDI VM（8GB/4vCPU 与 16GB/8vCPU 各一档）跑压测：N = 1/2/4/8 并发下的 P95 执行时延、Chrome 崩溃率、OOM 率、单路 LIVEVIEW 带宽；实测数字回填承载模型笔记的「待实测」处并给出推荐默认 `max_concurrency`；实测不可执行时**必须显式登记为 UAT 待办**，不得用推算数字充数。

### CLEANUP — 收尾真相化

- [ ] **CLEANUP-01**: 假数据清零 —— 批量报告真实落库（`sys_rpa_executions.batch_report` 等既有列）+ 逐项状态更新；自动扩缩容配置持久化到 `sys_config` 并热生效；`GetBatchExecutionReport` / `UpdateAutoScaleConfig` 各有「写入后读取一致」用例；删除所有硬编码假结果路径。
- [ ] **CLEANUP-02**: 定时调度接通 —— `sys_rpa_schedules` 接入 `internal/scheduler`：CRUD API + 启动时注册 cron job（`rpa_task` + `task_id`）+ 下次/上次运行时间回写 + 停用即摘除 job；调度触发与手动触发共用同一执行入口且 `trigger_type=schedule`；有「到点真触发」集成用例（可用秒级 cron）。
- [ ] **CLEANUP-03**: 死代码与泄漏清理 —— `flow_handler.go`（7 端点）/`data_mapper.go`/`error_handling.go` 三选一：删除，或明确保留为 service 内部能力并加使用方与豁免说明；`WorkerHandler` 复用 `core.Cache` 的 Redis 客户端（不再每实例 `redis.NewClient`）；`docs/RPA系统完成状态报告.md` 等失真文档更新为与代码一致；`grep` 断言「无未使用导出符号」「无每实例 Redis 连接」纳入 CI 或 lint。

## Out of Scope

| 项 | 理由 |
|---|---|
| 浏览器插件式录制（Chrome Extension） | 分发与签名成本高；`evalOnNewDocument` 页内注入已覆盖录制需求（NOCODE-01） |
| VNC / WebRTC / 独立远程桌面栈 | 与「复用同一 Chromium」目标冲突；CDP screencast + 输入注入足够，零新增重量级依赖 |
| 桌面（非浏览器）自动化（Win32/键鼠宏） | 目标用户诉求与部署形态（Linux 无头 VM）不支持 |
| 移动端 App / 小程序 | Web-first，本期不做 |
| 多租户隔离（每租户独立 Worker 池/配额计费） | 需组织模型与计费体系，超出本期；FLEET 只保留「一 VM 多容器」技术选项 |
| AI Agent 全自主（无人值守长链路决策） | 目标用户是一般员工，copilot 优先；AI 只做生成/修复建议 |
| 存量「v1.56 候选」代码的 UAT 追认（备份恢复/设置页收敛） | 缺真实环境证据，按 `notes/SHIPPED-DEFINITION.md` 只登记不追认，不混入本期 REQ |
| 79 条人工验证跟踪表 | 维持 0 勾选口径，不因本里程碑改写 |

## Constraints (inherited + v1.56 新增)

- **Handler-Service 分层**：handler 只做绑定/校验/响应，业务在 service；`response.Success()/Error()` 包装。
- **operlog 写操作**：写操作 success path 在 `response.Success` 前 `operlog.Record`；敏感端点 `RecordWithBody`；新增 RPA 写端点同样受 `AUDIT-01` 静态门禁约束。
- **双方言**：SQLite + PG 双通过；migration 双方言幂等、`database.go` 两处注册（PG + sqlite 分支）、`registry.go` 234+ 不进 ActiveMigrations；`.golangci.yml` 的 forbidigo 方言 token 规则不得新增违规。
- **RBAC**：新端点逐端点挂权限码（读 ANY-of / 写 strict），并在 `pkg/permission/config.go` + F 型播种 migration 同步；前端按钮按权限码门控。
- **零新增重量级依赖**：LIVEVIEW 只用 `go-rod` 已有的 `proto.PageStartScreencast` / `Input.*`；前端只用既有 WebSocket + Canvas，不引入 WebRTC 库。
- **Worker 双副本边界**：只在根副本 `rpa-worker/` 演进；`xingran-react-frontend/rpa-worker/` 维持可编译骨架，不新增业务。
- **禁 mock 验收**：不允许用「服务层 fake」或「测试内自建表」冒充端到端验收；真实浏览器与真实 schema 是验收前提。
- **构建产物只落 `out/`**；前端 `tsc` / `lint:ci` / vitest 全绿；后端 `go build ./...` / `go test ./...` 全绿。

## Traceability（v1.56 ROADMAP 已生成 · Phases 152-158）

| Requirement | Phase | Status |
|-------------|-------|--------|
| CONTRACT-01 | Phase 152 | Pending |
| CONTRACT-02 | Phase 152 | Pending |
| CONTRACT-03 | Phase 152 | Pending |
| CONTRACT-04 | Phase 152 | Pending |
| CONTRACT-05 | Phase 152 | Pending |
| CONTRACT-06 | Phase 152 | Pending |
| RUNTIME-01 | Phase 153 | Pending |
| RUNTIME-02 | Phase 153 | Pending |
| RUNTIME-03 | Phase 153 | Pending |
| RUNTIME-04 | Phase 153 | Pending |
| RUNTIME-05 | Phase 153 | Pending |
| ENGINE-01 | Phase 154 | Pending |
| ENGINE-02 | Phase 154 | Pending |
| ENGINE-03 | Phase 154 | Pending |
| ENGINE-04 | Phase 154 | Pending |
| ENGINE-05 | Phase 154 | Pending |
| LIVEVIEW-01 | Phase 155 | Pending |
| LIVEVIEW-02 | Phase 155 | Pending |
| LIVEVIEW-03 | Phase 155 | Pending |
| LIVEVIEW-04 | Phase 155 | Pending |
| LIVEVIEW-05 | Phase 155 | Pending |
| NOCODE-01 | Phase 156 | Pending |
| NOCODE-02 | Phase 156 | Pending |
| NOCODE-03 | Phase 156 | Pending |
| NOCODE-04 | Phase 156 | Pending |
| FLEET-01 | Phase 157 | Pending |
| FLEET-02 | Phase 157 | Pending |
| FLEET-03 | Phase 157 | Pending |
| CLEANUP-01 | Phase 158 | code-complete · verification-pending（2026-11-25，证据 `.planning/phases/158-cleanup/158-SUMMARY.md`） |
| CLEANUP-02 | Phase 158 | code-complete · verification-pending（2026-11-25，本机真跑到点触发，证据 `.planning/phases/158-cleanup/158-SUMMARY.md`） |
| CLEANUP-03 | Phase 158 | code-complete · verification-pending（2026-11-25，删除 2885 行 + 文档真相化，证据 `.planning/phases/158-cleanup/158-SUMMARY.md`） |

**Coverage:**
- v1 requirements: 31 total
- Mapped to phases: 31
- Unmapped: 0

---
*Requirements defined: 2026-11-23*
*Last updated: 2026-11-23 after v1.56 milestone initiation*
