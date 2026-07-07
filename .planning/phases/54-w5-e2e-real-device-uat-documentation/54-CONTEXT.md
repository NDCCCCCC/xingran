# Phase 54: W5 — E2E + Real-Device UAT + Documentation - Context

**Gathered:** 2026-07-07
**Status:** Ready for planning
**Source:** ROADMAP.md Phase 54 段（7 条 Success Criteria）+ REQUIREMENTS.md（36 MVP 全映射，Phase 54 = validation only）+ STATE.md §Critical Pitfalls + §Deferred Items + PROJECT.md §v1.19 + Phase 50/51/52/53 CONTEXT/SUMMARY + 48-HUMAN-UAT.md（v1.18 推迟先例）+ scrapligo v1.4.0 源码 scout（transport/file.go NewFileTransport）+ config.yaml 加密豁免实证

<domain>
## Phase Boundary

Phase 54 是 **v1.19 网络设备写命令里程碑的收尾验证 phase**——不新增任何功能（Phase 50-53 已 ship 全部 36 MVP 需求），只做三件事：

1. **Mock SSH e2e 测试**：补上 Phase 51 `mockDeviceExecutor` 遗漏的真实链路——让 service 层 `executeWithRetry` 的 fn 闭包（`wrapper.SendConfigs` → `parseConfigError` → BatchResult 编排）真正执行。Phase 51 mock 不调 fn（`port_write_service_test.go:394` 注释确认），parseConfigError 对真实 scrapligo Response 的解析、batch fail-fast 语义、PORT-06 skipped 在集成层从未被验证。
2. **真机 UAT 推迟文档化**：复刻 v1.18 `48-HUMAN-UAT.md` 模式，把 6 项 SSH transport verification（Huawei/H3C/Ruijie × shutdown/description/dot1x）显式推迟到下次现场访问；加 WR-02 观察条目兑现 Phase 55 依赖闭环。
3. **文档更新 + 全量回归绿灯**：API 响应规范补 6 端点 / SC#3 加密行为文档化 / 新建 CHANGELOG.md / README 能力更新 / MILESTONES.md v1.19 条目 + `go test ./...` + `npm run build` + `npm run type-check` + operlog regression 全绿。

**In scope**:
- `internal/services/portwrite/port_write_e2e_test.go` 新建：service 层 e2e，scrapligo `transport.NewFileTransport()` + 预录制 fixture 回放，覆盖 5 single + 1 batch happy path + 4 类错误路径（transport_error / device_rejected / batch fail-fast / PORT-06 skipped），1 厂商 Huawei 验证链路
- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` 新建：复刻 48-HUMAN-UAT.md 结构，6 项 SSH verification + WR-02 custom-reason 观察条目，全部标 `pending` 推迟现场
- `docs/API响应规范.md` 改：新增 6 端点（5 single + 1 batch）签名 + request schema + response shape 小节
- `docs/安全和认证设计（国密）.md` 改：文档化写端点 SM2+SM4 加密行为（确认**不**在 exclude_paths，保持加密）
- `CHANGELOG.md` 新建（项目根）：v1.19 entry（可选补 v1.18）
- `README.md` 改：网络设备纳管能力描述补"端口写命令（shutdown/undo/description/dot1x）"
- `.planning/MILESTONES.md` 改：新增 v1.19 milestone 条目（复刻 v1.18 格式：Phases/Plans/Tasks/Delivered/Key Accomplishments）
- `.planning/STATE.md` 改：deferred 表 "50-HUMAN-UAT.md" → "54-HUMAN-UAT.md"（D-08 同步）

**Out of scope**:
- 真机 SSH 写命令实测 → 54-HUMAN-UAT.md site visit（推迟到下次现场，owner = 现场运维同事）
- HTTP handler 层 e2e（gin test engine + 6 路由全打通）→ v1.19.x+（Phase 52 handler 零测试基建是 cold start，工作量大）
- 3 厂商（Huawei/H3C/Ruijie）e2e fixture 全覆盖 → v1.19.x+（fixture 字节序列脆性 ×3，厂商差异留真机 UAT）
- 跨固件版本命令差异（Huawei V200R005 vs V600R024C00）→ follow-up（现场验证）
- Real-device SSH 往返延迟测量 / batch timeout 标定 → follow-up
- BATCH-05 批量实时进度（SSE/WS）→ v1.19.x（53-CONTEXT 已 deferred）
- `sys_port_write_audit` 详情查看 UI → v1.19.x+（53-CONTEXT 已 deferred）
- Phase 55 技术债修复（WR-02/IN-01/IN-02/CR-02/HealthCard）→ Phase 55 独立 phase（本 phase 仅在 UAT 文档加 WR-02 观察条目驱动其决策）

</domain>

<decisions>
## Implementation Decisions

### Mock SSH e2e 技术方案（SC#1）

- **D-01: scrapligo `transport.NewFileTransport()` + 预录制 fixture 回放（非自建 sshd / 非扩展现有 mock）**
  - scrapligo v1.4.0 **无公开 Mock API**（无 `NewMock`/`WithMock`/`MockDriver`，`find ... -iname "*mock*"` 空），但 transport 包有公开 `NewFileTransport()`（`transport/file.go:14`）——这是 scrapligo 官方"文件回放"transport，从预录制 fixture 读取设备 IO 字节序列，无需真 sshd / 无端口 / CI 友好
  - 跑**真实 scrapligo SendConfigs 全链路**（厂商模板渲染 → config-mode 进入/退出 → Response 解析），fixture 文件 = "mock sshd 响应流"，满足 SC#1 字面 "in-process mock sshd" 语义
  - **不**自建 in-process sshd（golang.org/x/crypto/ssh）：实现 config-mode 交互状态机工作量最大，MVP 阶段 overkill
  - **不**扩展现有 `mockDeviceExecutor`：Phase 51 mock 不调 fn（绕过 SendConfigs），扩展它即便加 fake PooledConnection 也不测 scrapligo 字节解析层，价值低于 FileTransport
  - fixture 来源（planner discretion）：优先复用 scrapligo 自带 `transport/test-fixtures/` 改造，`% Error` / `Unrecognized command` 等错误场景手写补充

### e2e 覆盖深度与层级（SC#1）

- **D-02: service 层 e2e（非 HTTP handler 层）**
  - 直接调 `PortWriteService.ExecutePortWrite` / `ExecuteBatch` + 注入 FileTransport 的 `DeviceExecutor`
  - 验证 service 编排 → SendConfigs → parseConfigError → BatchResult 全链路（**补 Phase 51 fn 闘包漏洞**——核心价值）
  - SC#1 字面 "all 5 single-port + 1 batch endpoint paths" 的 "endpoint paths" 语义降级为 **service 公开方法路径**
  - **不**做 HTTP handler 层 e2e：Phase 52 handler 零测试基建（TESTING.md 确认"所有模块 API handlers 无测试覆盖"），gin test engine + mock Core 全套依赖（DB/Cache/OperLogService/CollectionSvc）cold start 工作量过大；HTTP 契约正确性已由 Phase 52 落地 + Phase 53 wrapper 对齐保证

- **D-03: happy path + 关键错误路径，1 厂商（Huawei）验证链路**
  - happy path：5 single（shutdown / undo shutdown / description / dot1x enable / dot1x disable 各 1 成功）+ 1 batch（同设备多端口同操作成功）
  - 关键错误路径（4 类，补 STATE.md Pitfall #1 / SSH-02 漏洞）：
    1. `transport_error`：连接失败 / 超时 / EOF（parseConfigError → `WriteErrorTransport`）
    2. `device_rejected`：`% Error` / `Unrecognized command` / `Illegal` 标记（parseConfigError → `WriteErrorDeviceRejected`）
    3. batch fail-fast：第二条端口失败立即停止（BATCH-02），返回 `{succeeded:[首条], failed:[第二条], skipped:[剩余]}`
    4. PORT-06 skipped：端口已处目标态（如已 shutdown 再 shutdown），返回 `NoOp=true / Status="skipped"`
  - **1 厂商 Huawei 验证链路**：fixture ~6-8 个；厂商命令差异（华为/H3C VRP 同源 vs 锐捷 Cisco-style `dot1x port-control auto` / `no dot1x port-control`）留真机 UAT 验证
  - **不**做 3 厂商 fixture 全覆盖：fixture 字节序列脆性 ×3（~18-24 个），维护成本高，且真机 UAT 必覆盖厂商差异

### 文档更新（SC#2 / SC#3 / SC#5）

- **D-04: SC#3 加密语义 = 写端点保持 SM2+SM4 加密，不改 config**
  - config.yaml `request_encryption.exclude_paths`（line 91-99）**不包含** `/network/ports/write/*` → 写端点当前**已加密**（前端 SM2+SM4 加密 request body → 后端解密）
  - SC#3 字面 "no SM2+SM4 wrap on SSH-derived paths" 措辞**误导**：SSH 是后端→设备协议，与 HTTP 请求体加密正交；写端点是敏感操作（shutdown/dot1x），应保持加密
  - SC#3 重解释为 **"确认写端点正确加密 + 文档化加密行为"**：在 `docs/安全和认证设计（国密）.md` 文档化写端点加密行为（不豁免），不改 config.yaml / 不加 migration
  - **不**加入 exclude_paths：敏感操作裸传 `{portId, reason}` 降低安全性，与等保合规趋势相左

- **D-05: 新建 `CHANGELOG.md`（项目根），从 v1.19 起记**
  - SC#5 字面要求 "README + CHANGELOG updated"，但 `CHANGELOG.md` **当前不存在**
  - 新建独立 `CHANGELOG.md`：v1.19 entry（网络设备写命令：3 厂商 × 5 操作 + batch + 审计 + 权限 + 前端 Drawer），可选补 v1.18（网络设备硬件清单）
  - **不**合并进 README：README.md head 有 `<!-- generated-by: gsd-doc-writer -->` 标记，手改"版本历史"段易被下次 `/gsd:docs-update` 覆盖；独立 CHANGELOG 避免生成器冲突
  - 项目已 v1.19 / 39 phases / 142+ plans，CHANGELOG 是合理项目治理

- **D-06: `docs/API响应规范.md` 新增"网络设备端口写操作"小节**
  - 现有结构已有"批量操作响应"(line 184)、"特殊场景响应"等段，6 端点签名新增小节接入
  - 覆盖：5 single 端点（`/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable}`）+ 1 batch 端点（`/network/ports/write/batch`）的 request schema（`{portId, reason?}` / `{deviceId, action, portIds, description?}`）+ response shape（PortResult / BatchResult 三数组）
  - 具体格式（小节位置 / 代码块风格）planner 按现有段惯例定

- **D-07: README 更新能力描述 + `MILESTONES.md` v1.19 条目**
  - README.md "核心特性" → "网络设备纳管"项补"端口写命令（shutdown / undo shutdown / description / dot1x 启停）+ 批量配置 + 完整审计"
  - `.planning/MILESTONES.md` 新增 v1.19 条目，复刻 v1.18 格式：`## v1.19 网络设备写命令 — ✅ SHIPPED [date]` + Phases(50-54)/Plans/Tasks/Delivered/Key Accomplishments

### UAT 推迟 + Phase 55 协调（SC#4）

- **D-08: UAT 文件 = `54-HUMAN-UAT.md`，放 phase 54 目录（非 SC#4 字面的 phase 50 目录）**
  - 文件路径：`.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`
  - 理由：**产出者存放**——Phase 54 产出此 UAT 文件。v1.18 先例 `48-HUMAN-UAT.md` 放 48 目录，因 48 兼"主交付 + 产出者"双角色；v1.19 拆 5 phase（50-54）后，产出 UAT 的是 54
  - SC#4 字面路径 `50-port-write-network-ports-planned/50-HUMAN-UAT.md` 是**占位名 + 命名漂移**：实际 phase 50 目录是 `50-w1-vendor-templates-unit-tests-vendor-action-command-map`（W1 wave 命名，非 milestone 主交付）
  - 文件名遵循"文件名 phase 号 = 目录 phase 号"先例（48-HUMAN-UAT.md ↔ 48 目录）
  - **同步更新**：`.planning/STATE.md` §Deferred Items 表中 "50-HUMAN-UAT.md site visit" → "54-HUMAN-UAT.md site visit"
  - 结构复刻 `48-HUMAN-UAT.md`：frontmatter（status: partial / verifier_status: human_needed / automated_gates）+ 自动化闸门清单 + Tests(site-visit deferred) 逐项 + Summary 表 + Owner + 关联声明

- **D-09: UAT 文档加 WR-02 "custom-reason 使用频率" 观察条目**
  - 6 项 SSH transport verification（SC#4 方向）之外，新增 1 项观察条目："WR-02 观察：记录现场运维使用 custom-reason（'其他...'）输入操作原因的频率"
  - 兑现 STATE.md 声明的依赖闭环："Phase 55 WR-02（PortWriteModal custom-reason validator 签名修复）修复决策由 W5 UAT 观察"
  - 观察结果驱动 Phase 55 决策：custom-reason 高频 → 修 WR-02（53-REVIEW.md IN 阶修复）；低频 → 标 wontfix
  - 6 项 SSH verification 设备型号（SC#4 写 Huawei S5700/S5735）标 **"待现场运维确认"**（复刻 48 pending 模式；v1.18 memory 显示现场有 S8700/RS8607E，实际型号以现场为准）

### 事实纠正（待 planner/researcher 锁定）

- **D-10: scrapligo 实际版本 v1.4.0（非 ROADMAP/REQUIREMENTS 写的 v1.3.3）**
  - `go.mod`: `github.com/scrapli/scrapligo v1.4.0`
  - ROADMAP Phase 50/51 段、REQUIREMENTS SSH-01 写 "v1.3.3" 是旧值；planner 以 v1.4.0 为准（FileTransport API 在 v1.4.0 `transport/file.go:14`）

- **D-11: SC#4 路径 `50-port-write-network-ports-planned/` 是占位名**
  - 实际 phase 50 目录 = `50-w1-vendor-templates-unit-tests-vendor-action-command-map`；UAT 文件实际放 phase 54 目录（D-08）

### Claude's Discretion

- **fixture 来源与格式**：优先复用 scrapligo `transport/test-fixtures/` 现成 fixture 改造，`% Error` / `Unrecognized command` / 连接失败等错误场景手写 fixture 补充；fixture 字节序列精确度由 researcher 查 scrapligo fixture 格式文档确认
- **API 响应规范小节具体位置**：D-06 给了"新增小节"方向，具体插在"批量操作响应"(line 184) 之后还是"特殊场景响应"末尾，planner 按文档连贯性定
- **CHANGELOG 是否补 v1.18**：D-05 默认 v1.19 起记，planner 可选补 v1.18 一行（参考 MILESTONES.md v1.18 段）
- **UAT 文档 automated_gates 清单**：复刻 48 模式列出本 phase 跑过的自动化闸门（go test / npm build / type-check / operlog regression 实际结果）
- **e2e 测试 DeviceExecutor FileTransport 注入点**：researcher 确认 `device.DeviceExecutor` 如何接受 transport option（构造函数参数 vs functional option），planner 据此设计测试 setup

### Folded Todos
None — cross_reference_todos 发现 2 个弱匹配（score 0.6）pending todo，均跨 milestone 无关，未 fold（见 Reviewed Todos）。

</decisions>

<canonical_refs>
## Canonical References

**下游 agent (planner / researcher) 必须先读这些。**

### v1.19 落地契约（e2e 被测对象）
- `.planning/phases/51-w2-portwriteservice-batch-orchestrator-mock-tests/51-CONTEXT.md` — D-10..D-18，PortWriteService 签名 + BatchResult 三数组 + fail-fast 语义 + detached 30min context
- `.planning/phases/51-w2-portwriteservice-batch-orchestrator-mock-tests/51-01-SUMMARY.md` — service 最终形状
- `internal/services/portwrite/port_write_service.go` — `ExecutePortWrite` / `ExecuteBatch` 公开方法（e2e 入口）+ `portWriteExecutor.ExecuteCustom` 接口 + `executeWithRetry` fn 闭包（含 SendConfigs + parseConfigError，Phase 51 未测）
- `internal/services/portwrite/parse_error.go` — `parseConfigError` 5 步优先级解析（transport_error vs device_rejected），e2e 错误路径验证目标
- `internal/services/portwrite/pre_state_check.go` — PORT-06 pre-state check（admin_status / dot1x_enabled → NoOp/skipped）
- `internal/services/portwrite/batch_orchestrator.go` — BATCH-02 serial fail-fast + BATCH-03 partial result 编排
- `internal/services/portwrite/port_write_service_test.go` — Phase 51 现有 mock（`mockDeviceExecutor` 不调 fn，line 394 注释），e2e 测试的对照基线 + mock 注入模式参考

### Phase 52 HTTP 契约（API 文档来源）
- `.planning/phases/52-w3-router-handler-operlog-permission-migration/52-CONTEXT.md` — D-01..D-16，6 端点 + Path C audit + 权限
- `.planning/phases/52-w3-router-handler-operlog-permission-migration/52-01-SUMMARY.md` — 6 端点最终形状 + PortWriteRequest struct
- `internal/api/v1/network/port_write_handler.go` — 6 handler，request body shape（`{portId, description?, reason?}` / batch `{deviceId, action, portIds, description?}`）+ sentinel→HTTP 翻译
- `internal/api/v1/network/port_write_router.go` — 6 kebab 路径（`/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}`），API 文档 URL 来源

### scrapligo FileTransport（e2e mock 技术）
- `C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\transport\file.go:14` — `NewFileTransport()` 公开 API（fixture 回放 transport）
- `C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\transport\test-fixtures\` — scrapligo 自带 fixture 样本（复用改造来源）
- `go.mod` — `github.com/scrapli/scrapligo v1.4.0`（D-10 实际版本，非 v1.3.3）

### v1.18 UAT 推迟先例（54-HUMAN-UAT.md 模板）
- `.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md` — 复刻模板：frontmatter（status: partial / verifier_status: human_needed / automated_gates）+ Tests(site-visit deferred) + Summary 表 + Owner + 关联声明
- `.planning/phases/48-device-component-serials-planned/48-VERIFICATION.md` — human_verification section（UAT 推迟的 verifier 记录方式）

### 加密豁免实证（SC#3）
- `configs/config.yaml:88-117` — `request_encryption.exclude_paths`（写端点 `/network/ports/write/*` **不在**列表 = 当前加密中）+ `response_encryption` 配置
- `pkg/middleware/`（encryption 中间件）— 确认 exclude_paths 匹配逻辑（filepath.Match + `/*` 通配）
- `docs/安全和认证设计（国密）.md` — SC#3 文档化目标文件

### 文档更新目标
- `docs/API响应规范.md` — SC#2 目标（现有结构：批量操作响应 line 184 / 特殊场景响应 / 错误响应规范）
- `README.md` — SC#5 目标（head `<!-- generated-by: gsd-doc-writer -->`，核心特性段"网络设备纳管"项）
- `.planning/MILESTONES.md` — SC#5 目标（v1.18 条目格式参考）

### v1.19 锁定决策
- `.planning/PROJECT.md` §"Current Milestone: v1.19" — init 决策（device_id 直连 / 厂商 map / OperType / 权限隔离 / sys_port_write_audit 真相源 / 真机 UAT 推迟）
- `.planning/REQUIREMENTS.md` — 36 MVP 需求（SSH/PORT/BATCH/AUDIT/UI/PERM/INFRA/CONV），Phase 54 = validation only
- `.planning/ROADMAP.md` Phase 54 段 — 7 条 Success Criteria（SC#4 路径占位名 D-11 纠正；SC#1 v1.3.3 D-10 纠正；SC#3 加密语义 D-04 纠正）
- `.planning/STATE.md` — §Critical Pitfalls（Pitfall #1 SSH-02 transport vs device_rejected）+ §Deferred Items（v1.19 自身 deferred 3 项 + Phase 48 真机 UAT deferred 先例）+ §Known Risks（Mock SSH e2e 不能完全覆盖真机固件怪癖）

### Phase 55 依赖（WR-02 协调）
- `.planning/phases/55-phase-53-leftover-sweep/`（Phase 55 目录，0 plans TBD）— WR-02 / IN-01 / IN-02 / CR-02 / HealthCard 技术债，WR-02 修复决策由本 phase UAT 观察驱动（D-09）
- `.planning/ROADMAP.md` Phase 55 段 — WR-02 描述（PortWriteModal/BulkWriteDrawer custom-reason validator 签名修复）

### 测试基建参考
- `.planning/codebase/TESTING.md` — 测试栈（Go testing + testify v1.11.1 / Vitest）+ 现有测试覆盖现状（API handlers 零覆盖 / No mocks / No e2e）+ Gaps
- `internal/utils/operlog/regression_test.go` — SC#7 回归守护（25 OperType 常量 / 11 敏感关键词 / Record 5 参签名），e2e + 文档改动后必须仍绿

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **scrapligo `transport.NewFileTransport()`**（v1.4.0 公开 API）— e2e mock transport 底座，从 fixture 文件回放设备 IO，无真 sshd
- **`mockDeviceExecutor`**（port_write_service_test.go:78）— Phase 51 现有 testify mock，实现 `portWriteExecutor.ExecuteCustom`；e2e 可参考其注入模式但需改为调 fn + FileTransport
- **`device.DeviceExecutor`**（`internal/device/`）— PortWriteService 的依赖，e2e 需注入 FileTransport 配置的 DeviceExecutor（注入点 researcher 确认）
- **testify v1.11.1**（assertions + mock）— Go 测试栈，e2e 断言用 `assert.Equal` / `assert.NoError`
- **`operlog.Record` regression lock**（regression_test.go）— SC#7 回归基线，e2e 不应触碰 operlog 常量
- **`48-HUMAN-UAT.md`** 结构 — 54-HUMAN-UAT.md 直接复刻（frontmatter + automated_gates + Tests + Summary + Owner）

### Established Patterns
- **fixture-driven 测试**：scrapligo 自身 test-fixtures 模式（回放预录制 IO），无真机依赖
- **service 层 + mock 依赖注入**：Phase 51 `NewPortWriteService(db, deviceExecutor, collectionSvc)` 工厂模式，e2e 替换 deviceExecutor 为 FileTransport 版
- **UAT 推迟文档化**：v1.18 Phase 48 先例（site-visit item 标 `[pending]` + `why_human` + `addressed_in` 三字段）
- **里程碑收尾文档三件套**：README 能力 + CHANGELOG entry + MILESTONES.md 条目（v1.18 ship 模式）
- **operlog OperType 映射**：CONV-01..04（shutdown/undo/dot1x→OperTypeStatus / description→OperTypeUpdate / batch→OperTypeBatch），API 文档 + CHANGELOG 引用

### Integration Points
- `internal/services/portwrite/port_write_e2e_test.go`（新建）— e2e 测试文件，与 port_write_service_test.go 同包
- `internal/services/portwrite/`（testdata 子目录）— fixture 文件存放（`testdata/*.fixture` 或随 scrapligo fixture 惯例）
- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`（新建）— UAT 推迟文档
- `docs/API响应规范.md`（改）— 新增端口写操作小节
- `docs/安全和认证设计（国密）.md`（改）— SC#3 加密行为文档化
- `CHANGELOG.md`（新建，项目根）— v1.19 entry
- `README.md`（改）— 核心特性段
- `.planning/MILESTONES.md`（改）— v1.19 条目
- `.planning/STATE.md`（改）— deferred 表 50→54 HUMAN-UAT 同步

</code_context>

<specifics>
## Specific Ideas

- **D-01 FileTransport = "mock sshd 响应流"**：SC#1 字面 "in-process mock sshd" 的语义由 FileTransport 满足——fixture 文件就是预录制的 sshd 响应字节流，scrapligo 跑真实 SendConfigs 但 transport 层从文件读而非真 socket。无需起 sshd 进程，CI 友好。
- **D-02 service 层 e2e 的核心价值 = 补 Phase 51 fn 闭包漏洞**：Phase 51 `mockDeviceExecutor.ExecuteCustom` 不调 fn（line 394 注释），service 内 `executeWithRetry` → `wrapper.SendConfigs(cmds)` → `parseConfigError` 闭包链从未执行。FileTransport e2e 让 fn 真正跑，这是 Phase 54 相对 Phase 51 的独有增量价值——不是重复测 service 编排，而是首次测 fn 内的 SendConfigs + parseConfigError 集成。
- **D-04 SC#3 "SSH-derived paths" 措辞误导**：SSH 是后端→设备协议（scrapligo SSH 连设备），SM2+SM4 是 HTTP 请求体加密（前端→后端），二者正交。SC#3 作者把二者混淆，实际写端点应保持 HTTP 加密（敏感操作）。文档化时明确这个区分。
- **D-08 产出者存放原则**：v1.18 的 48-HUMAN-UAT.md 放 48 目录是因为 48 兼主交付+产出者；v1.19 拆 5 phase 后，UAT 文件的产出者是 54（收尾验证 phase），故放 54 目录。命名 `54-HUMAN-UAT.md` 保持"文件名 phase 号 = 目录 phase 号"一致性。
- **D-09 WR-02 观察闭环**：STATE.md 明确"Phase 55 WR-02 修复决策由 W5 UAT 观察"，本 phase 在 UAT 文档加观察条目是兑现该声明——否则 Phase 55 planner 无依据决定修/不修。custom-reason 高频→修，低频→wontfix。

</specifics>

<deferred>
## Deferred Ideas

- **真机 SSH 写命令验证**（Huawei S5700/S5735 + H3C + Ruijie RS8607E 各 shutdown + description + dot1x）→ 54-HUMAN-UAT.md site visit，owner = 现场运维同事，下次现场访问
- **HTTP handler 层 e2e**（gin test engine + 6 路由全打通 + mock Core 依赖）→ v1.19.x+：Phase 52 handler 零测试基建是 cold start，需先建 handler 测试基础设施
- **3 厂商 e2e fixture 全覆盖**（Huawei/H3C/Ruijie × 5 操作）→ v1.19.x+：fixture 字节序列脆性 ×3，厂商命令差异由真机 UAT 必覆盖
- **跨固件版本命令差异**（Huawei V200R005 vs V600R024C00）→ follow-up：现场多版本固件验证
- **Real-device SSH 往返延迟测量 / batch per-port timeout 标定**→ follow-up：现场用真机延迟数据校准 30min detached context / per-port timeout
- **`sys_port_write_audit` 详情查看 UI** → v1.19.x+（53-CONTEXT 已 deferred）
- **BATCH-05 批量实时进度（SSE/WS）** → v1.19.x（53-CONTEXT D-05 indeterminate spinner 是 MVP 过渡）
- **fixture 自动录制工具**（从真机录制一次 → 回放）→ v1.19.x+：现场录制后回填 fixture，减少手写脆性

### Reviewed Todos (not folded)
cross_reference_todos 发现 2 个弱匹配（score 0.6）pending todo，均跨 milestone 无关，reviewed 但未 fold：
- `.planning/todos/pending/operlog-exclude-paths.md`（"operlog.exclude_paths 配置驱动白名单 / RPA 心跳日志污染"）— v1.17 之前的 RPA 心跳日志问题，与 v1.19 端口写操作日志（CONV-01..04）无关，Phase 54 的 operlog 是写操作记录非 exclude_paths 配置
- `.planning/todos/pending/v1.17-reconciliation-decisions.md`（"v1.17 资产对账决策追踪"）— 已 ship 的 v1.17 milestone 决策追踪，与 v1.19 网络设备写命令无关

</deferred>

---

*Phase: 54-w5-e2e-real-device-uat-documentation*
*Context gathered: 2026-07-07*
</content>
</invoke>
