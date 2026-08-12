# Phase 54: W5 — E2E + Real-Device UAT + Documentation - Research

**Researched:** 2026-07-07
**Domain:** Go service-layer e2e testing (scrapligo FileTransport replay) + project documentation + UAT site-visit deferral
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Mock SSH e2e 技术方案（SC#1）**

- **D-01: scrapligo `transport.NewFileTransport()` + 预录制 fixture 回放**（非自建 sshd / 非扩展现有 mock）。scrapligo v1.4.0 无公开 Mock API；用 transport 包公开 `NewFileTransport()` 作"文件回放 transport"，从预录制 fixture 读取设备 IO 字节序列。**不自建 in-process sshd**、**不扩展现有 mockDeviceExecutor**（Phase 51 mock 不调 fn）。
- **D-02: service 层 e2e（非 HTTP handler 层）**。直接调 `PortWriteService.ExecutePortWrite` / `ExecuteBatch`，补 Phase 51 `mockDeviceExecutor` 未调 fn 的闭包漏洞。**不做 HTTP handler 层 e2e**（Phase 52 零测试基建，cold start 工作量过大）。
- **D-03: happy path + 关键错误路径，1 厂商（Huawei）验证链路**。happy path：5 single + 1 batch；错误路径 4 类（transport_error / device_rejected / batch fail-fast / PORT-06 skipped）；fixture ~6-8 个 Huawei。**不做 3 厂商 fixture 全覆盖**。

**文档更新（SC#2 / SC#3 / SC#5）**

- **D-04: SC#3 加密语义 = 写端点保持 SM2+SM4 加密，不改 config**。`config.yaml` `request_encryption.exclude_paths` **不包含** `/network/ports/write/*` → 写端点当前已加密。SC#3 字面 "no SM2+SM4 wrap on SSH-derived paths" 措辞误导；SC#3 重解释为 "确认写端点正确加密 + 文档化加密行为"。**不加入 exclude_paths**。
- **D-05: 新建 `CHANGELOG.md`（项目根），从 v1.19 起记**。**不合并进 README**（避免 `<!-- generated-by: gsd-doc-writer -->` 覆盖）。
- **D-06: `docs/API响应规范.md` 新增"网络设备端口写操作"小节**（5 single + 1 batch 端点的 request schema + response shape）。
- **D-07: README 更新能力描述 + `MILESTONES.md` v1.19 条目**。

**UAT 推迟 + Phase 55 协调（SC#4）**

- **D-08: UAT 文件 = `54-HUMAN-UAT.md`，放 phase 54 目录**（非 SC#4 字面的 phase 50 目录）。文件路径：`.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`。结构复刻 `48-HUMAN-UAT.md`。**同步更新**：`.planning/STATE.md` §Deferred Items 表 "50-HUMAN-UAT.md" → "54-HUMAN-UAT.md"。
- **D-09: UAT 文档加 WR-02 "custom-reason 使用频率" 观察条目**，兑现 Phase 55 WR-02 修复决策依赖闭环。

**事实纠正**

- **D-10: scrapligo 实际版本 v1.4.0**（非 ROADMAP/REQUIREMENTS 写的 v1.3.3）。`go.mod` 已确认 `v1.4.0`。
- **D-11: SC#4 路径 `50-port-write-network-ports-planned/` 是占位名**。实际 phase 50 目录 = `50-w1-vendor-templates-unit-tests-vendor-action-command-map`；UAT 文件实际放 phase 54 目录（D-08）。

### Claude's Discretion

- **fixture 来源与格式**：优先复用 scrapligo `transport/test-fixtures/` 现成 fixture 改造（**RESEARCH 发现 landmine — 见 Open Questions #3**）；`% Error` / `Unrecognized command` / 连接失败等错误场景手写 fixture 补充；fixture 字节序列精确度由 researcher 查 scrapligo fixture 格式文档确认
- **API 响应规范小节具体位置**：D-06 给了"新增小节"方向，具体插在"批量操作响应"(line 184) 之后还是"特殊场景响应"末尾，planner 按文档连贯性定
- **CHANGELOG 是否补 v1.18**：D-05 默认 v1.19 起记，planner 可选补 v1.18 一行（参考 MILESTONES.md v1.18 段）
- **UAT 文档 automated_gates 清单**：复刻 48 模式列出本 phase 跑过的自动化闸门（go test / npm build / type-check / operlog regression 实际结果）
- **e2e 测试 DeviceExecutor FileTransport 注入点**：researcher 确认 `device.DeviceExecutor` 如何接受 transport option（构造函数参数 vs functional option），planner 据此设计测试 setup

### Deferred Ideas (OUT OF SCOPE)

- 真机 SSH 写命令验证（Huawei/H3C/Ruijie 各 shutdown + description + dot1x）→ 54-HUMAN-UAT.md site visit
- HTTP handler 层 e2e（gin test engine + 6 路由全打通）→ v1.19.x+
- 3 厂商（Huawei/H3C/Ruijie）e2e fixture 全覆盖 → v1.19.x+
- 跨固件版本命令差异（Huawei V200R005 vs V600R024C00）→ follow-up
- Real-device SSH 往返延迟测量 / batch timeout 标定 → follow-up
- BATCH-05 批量实时进度（SSE/WS）→ v1.19.x
- `sys_port_write_audit` 详情查看 UI → v1.19.x+
- Phase 55 技术债修复（WR-02/IN-01/IN-02/CR-02/HealthCard）→ Phase 55 独立 phase
</user_constraints>

<phase_requirements>
## Phase Requirements

Phase 54 = **validation only**（Phase 50-53 已 ship 全部 36 v1.19 MVP 需求）。本 phase 不实现新需求，而是为已 ship 需求补 e2e 测试 + 文档化 + 真机 UAT 推迟记录。

| Requirement ID | Phase 50-53 落地点 | Phase 54 验证手段 |
|----------------|---------------------|--------------------|
| SSH-01 (SendConfig contract) | Phase 50 vendor templates | E2E fixture 回放 SendConfigs 真实链路 |
| SSH-02 (transport_error vs device_rejected 解析) | Phase 51 parseConfigError | E2E 4 类错误路径断言 |
| SSH-03 (per-port Acquire/ReleaseRef) | Phase 51 executeWithRetry | E2E 通过 fn 闭包触发 wrapper.SendConfigs |
| SSH-04 (detached 30min context) | Phase 51 BatchWritePorts | Phase 51 已测（TestBatchWritePorts_DetachedContext）；Phase 54 不重复 |
| SSH-05 (vendor → template map) | Phase 50 RenderCommand | Phase 50 unit tests 已覆盖；Phase 54 E2E 验证 1 厂商（Huawei）真实链路 |
| SSH-06 (DeviceCredentialHelper) | Phase 51 fallback 路径 | E2E 通过 FileTransport 绕过凭证 |
| PORT-01..PORT-05 (5 单端口操作) | Phase 52 6 端点 | E2E 5 single happy path + 4 类错误路径 |
| PORT-06 (pre-state check NoOp) | Phase 51 checkPreState | E2E PORT-06 skipped 错误路径 |
| BATCH-01..BATCH-04 (batch 编排) | Phase 51 BatchWritePorts | E2E batch happy path + fail-fast |
| BATCH-05 (进度反馈) | Phase 53 BulkWriteDrawer | Phase 53 已 ship；本 phase 不验证（已在 53 unit test） |
| AUDIT-01..AUDIT-04 (operlog + sys_port_write_audit) | Phase 52 | SC#7 operlog regression guard；audit 表已在 Phase 52 验证 |
| UI-01..UI-06 | Phase 53 | SC#6 npm build + type-check 绿灯 |
| PERM-01..PERM-03 | Phase 52 | 不在本 phase 验证范围（Phase 52 unit test 已覆盖） |
| INFRA-01..INFRA-03 | Phase 52 | SC#6 go test ./... 绿灯 |
| CONV-01..CONV-04 (OperType 映射) | Phase 52 | SC#7 operlog 25 OperType 常量不回归 |
</phase_requirements>

## Summary

Phase 54 是 v1.19 网络设备写命令里程碑的收尾验证 phase。不新增功能，只做三件事：(1) 补 service 层 e2e 测试（FileTransport 回放）覆盖 Phase 51 mock 遗漏的 fn 闭包链路；(2) 真机 UAT 推迟文档化（复刻 v1.18 Phase 48 模式）；(3) 文档更新（API 响应规范 + 加密行为 + CHANGELOG + README + MILESTONES.md）+ 全量回归绿灯。

本研究对 CONTEXT.md 的 3 个开放问题给出确证回答，并发现 1 个 CONTEXT 隐含假设的**重要 landmine**：scrapligo `transport/test-fixtures/` 目录**只含一对 SSH 密钥**（`dumbserver` + `dumbserver.pub`），**没有任何预录制设备 IO fixture**。所有 fixture 必须手写。CONTEXT D-01 "Claude's Discretion 优先复用 scrapligo test-fixtures" 的"复用"路径不通——必须改为参考 scrapligo **driver/network/test-fixtures/** 的 Cisco 风格 fixture 模式手写 Huawei VRP fixture。

第二项关键发现：**DeviceExecutor → DeviceTaskScheduler → DeviceConnectionPool → createConnection 链路不接受 transport 注入**。`createConnection` 内部硬编码 `NewScrapliWrapper` / `NewScrapliWrapperWithPort`，强制走真实 SSH/Telnet。e2e 不能通过现有 `*device.DeviceExecutor` API 注入 FileTransport —— 必须在 service 层注入一个**自定义 portWriteExecutor 实现**，它内部直接构造 `network.NewDriver(..., WithTransportType(FileTransport), WithFileTransportFile(...))` 跑真实 scrapligo SendConfigs 链路（绕过现有 connection pool，但不绕过 service 编排逻辑）。

**Primary recommendation:** e2e 测试通过实现一个 `fileTransportExecutor` 类型（实现 `portWriteExecutor.ExecuteCustom` 接口），其 fn 闭包内直接持有 scrapligo `*network.Driver`（用 FileTransport 配置）+ 调 `driver.SendConfigs(cmds)` + 把响应转回 `*device.Response`，从而让 service 层 `executeWithRetry` → fn → SendConfigs → parseConfigError 闭包链真正执行。fixture 全部手写（参考 scrapligo `driver/network/test-fixtures/send-configs-simple.txt` 模式：prompt + 回显命令 + config-mode prompt 字节流）。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| E2E test fixture replay | Test tier (Go testing) | — | FileTransport 在 transport 层从文件读字节流；service 层通过 portWriteExecutor 接口注入；不污染 production 代码 |
| Vendor command rendering | Service tier (`portcollection.RenderCommand`) | — | 已在 Phase 50 落地；E2E 通过它生成 cmds，不重复实现 |
| SSH transport abstraction | scrapligo library (`transport.File`) | — | FileTransport 实现 transport.Implementation 接口；e2e 替换标准 SSH transport |
| operlog regression guard | Util tier (`internal/utils/operlog`) | — | 25 OperType 常量 + 11 敏感关键词 + Record 5 参签名；e2e + 文档改动不得触碰 |
| Documentation | Docs tier (`docs/`, root `CHANGELOG.md`, `README.md`) | — | 6 端点签名 + 加密行为 + milestone 条目；遵循现有生成器约束（README 头部 generated-by 标记） |
| UAT deferral tracking | Planning tier (`.planning/`) | — | 54-HUMAN-UAT.md 复刻 48 模式；STATE.md 同步更新 |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| scrapli/scrapligo | v1.4.0 | FileTransport + network.Driver 作 e2e mock 底座 | go.mod 已锁定；项目唯一 scrapligo 依赖；FileTransport 是 scrapligo 官方测试 transport `[VERIFIED: go.mod + scrapligo v1.4.0 source transport/file.go:14]` |
| stretchr/testify | v1.11.1 | assert + mock；e2e 断言 PortResult/BatchResult | 项目既有测试栈；Phase 51 mock 同栈 `[VERIFIED: go.sum + port_write_service_test.go:14]` |
| testing (stdlib) | Go 1.24 | `go test` runner | 项目唯一 Go test runner `[VERIFIED: CLAUDE.md]` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| gorm.io/driver/sqlite | v1.5.4 | 内存 sqlite `:memory:` 测试 DB | e2e service 层测试复用 Phase 51 `newTestDB(t)` 模式 `[VERIFIED: port_write_service_test.go:16]` |
| xuri/excelize/v2 | v2.10.0 | （本 phase 不直接使用） | — |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| FileTransport (回放) | golang.org/x/crypto/ssh 自建 sshd | CONTEXT D-01 已拒绝：实现 config-mode 交互状态机工作量最大，MVP overkill |
| FileTransport (回放) | 扩展现有 mockDeviceExecutor 加 fake PooledConnection | CONTEXT D-01 已拒绝：不测 scrapligo 字节解析层；扩展点价值低于 FileTransport |
| service 层 e2e | HTTP handler 层 e2e (gin test engine) | CONTEXT D-02 已拒绝：Phase 52 handler 零测试基建 cold start |
| 1 厂商 fixture | 3 厂商 (Huawei/H3C/Ruijie) 全覆盖 | CONTEXT D-03 已拒绝：fixture 脆性 ×3；厂商差异留真机 UAT |

**Installation:**

无新依赖。本 phase 不引入任何外部包（仅使用 go.mod 已有的 scrapligo / testify / sqlite）。

```bash
# 验证既有依赖（无需 install）
go mod verify
grep scrapligo go.mod  # 应输出 github.com/scrapli/scrapligo v1.4.0
```

**Version verification:** `[VERIFIED]` 在研究过程中通过 `go.mod` 直接读取确认：
- `github.com/scrapli/scrapligo v1.4.0`（D-10 锁定）
- `github.com/stretchr/testify v1.11.1`（port_write_service_test.go:14 import 路径与项目其他测试同栈）

## Package Legitimacy Audit

> 本 phase **不安装任何外部包**——所有依赖在 Phase 50-53 已落地。Package Legitimacy Gate 协议第 1-4 步均不适用（无可审计的新包）。
>
> 现有依赖已在 `.planning/phases/50-w1-vendor-templates-unit-tests-vendor-action-command-map/50-RESEARCH.md` 等前序 phase 完成审计。本 phase 视为继承已通过状态。

| Package | Registry | Status |
|---------|----------|--------|
| scrapli/scrapligo@v1.4.0 | Go modules | Inherited (Phase 50 audited) |
| stretchr/testify@v1.11.1 | Go modules | Inherited (project-wide) |
| gorm.io/driver/sqlite@v1.5.4 | Go modules | Inherited (Phase 51 audited) |

*本 phase 未触发 slopcheck / npm view / cargo search 流程（零新增包）。*

## Architecture Patterns

### System Architecture Diagram

```
e2e Test Setup                    Service Layer (被测对象)              scrapligo (真实链路)
─────────────────                ──────────────────────               ────────────────────
                                                                          
 fileTransportExecutor  ←─实现─  portWriteExecutor (interface)         
       │                                                                 
       │  ExecuteCustom(ctx, deviceID, fn, timeout)                      
       ├─────────────────────────→  svc.Shutdown / BatchWritePorts      
                                       │                                 
                                       │  executeWrite                   
                                       │  ├─ RenderCommand (Phase 50)   
                                       │  │   → cmds []string            
                                       │  ├─ checkPreState (PORT-06)    
                                       │  └─ deviceExecutor.ExecuteCustom(ctx, devID, fn, timeout)
                                       │              │                  
                                       │              │ fn closure      
                                       │              ▼                  
                                       │         (fileTransportExecutor)
                                       │              │                  
                                       │              │ Open scrapligo *network.Driver
                                       │              │ with FileTransport + fixture file
                                       │              ▼                  
                                       │         driver.SendConfigs(cmds)  ← 真实 scrapligo 解析
                                       │              │                  
                                       │              │ transport.File.Read() 
                                       │              │ reads 1 byte at a time
                                       │              │ from fixture file
                                       │              ▼                  
                                       │         *Response {Result, Failed}  
                                       │              │                  
                                       │         fn returns → service 拿 lastResp
                                       │                                 
                                       │  parseConfigError(lastResp)     
                                       │  ├─ nil → success               
                                       │  ├─ Failed=true → transport_err 
                                       │  ├─ "% Error:" → device_reject  
                                       │  └─ "timeout"/"EOF" → transport 
                                       │                                 
                                       └─→ PortResult / BatchResult      
                                                                          
 Assert: PortResult.Status == "succeeded" / "failed" / "skipped"         
 Assert: BatchResult.{Succeeded,Failed,Skipped} 数组长度                  
```

文件 → 实现映射：

| 文件 | 实现职责 |
|------|----------|
| `internal/services/portwrite/port_write_e2e_test.go`（新建） | e2e 测试主体：fileTransportExecutor + 5 happy path + 4 错误路径 + batch |
| `internal/services/portwrite/testdata/*.fixture`（新建） | 手写 fixture：Huawei VRP 设备 IO 字节流（prompt + 命令回显 + config-mode 切换） |
| `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md`（新建） | UAT 推迟文档（复刻 48 模式） |
| `docs/API响应规范.md`（改） | 新增"网络设备端口写操作"小节 |
| `docs/安全和认证设计（国密）.md`（改） | SC#3 文档化写端点加密行为 |
| `CHANGELOG.md`（新建） | v1.19 entry |
| `README.md`（改） | 网络设备纳管能力描述 |
| `.planning/MILESTONES.md`（改） | v1.19 milestone 条目 |
| `.planning/STATE.md`（改） | deferred 表 50→54 同步 |

### Recommended Project Structure

```
internal/services/portwrite/
├── port_write_service.go          # 已 ship (Phase 51) — executeWrite + writeSinglePort
├── batch_orchestrator.go          # 已 ship (Phase 51) — BatchWritePorts (fail-fast)
├── parse_error.go                 # 已 ship (Phase 51) — parseConfigError
├── pre_state_check.go             # 已 ship (Phase 51) — PORT-06 NoOp
├── port_write_service_test.go     # 已 ship (Phase 51) — mockDeviceExecutor (不调 fn)
├── port_write_e2e_test.go         # 【新建】fileTransportExecutor + e2e 用例
└── testdata/                      # 【新建】fixture 文件
    ├── huawei_shutdown_success.fixture
    ├── huawei_undo_shutdown_success.fixture
    ├── huawei_description_success.fixture
    ├── huawei_dot1x_enable_success.fixture
    ├── huawei_dot1x_disable_success.fixture
    ├── huawei_batch_success.fixture
    ├── huawei_device_rejected.fixture   # "% Error: Unrecognized command"
    └── huawei_transport_error.fixture   # 模拟 connection refused / EOF

.planning/phases/54-w5-e2e-real-device-uat-documentation/
├── 54-CONTEXT.md
├── 54-RESEARCH.md (本文件)
├── 54-HUMAN-UAT.md                 # 【新建】UAT 推迟文档
└── 54-01-PLAN.md (planner 产出)

docs/
├── API响应规范.md                   # 【改】新增端口写操作小节
└── 安全和认证设计（国密）.md          # 【改】SC#3 加密行为文档化

CHANGELOG.md                         # 【新建】项目根
README.md                            # 【改】核心特性段
.planning/MILESTONES.md              # 【改】v1.19 条目
.planning/STATE.md                   # 【改】deferred 表 50→54
```

### Pattern 1: FileTransport Executor 注入（D-02 实现路径）

**What:** e2e 测试通过实现 `portWriteExecutor.ExecuteCustom` 接口的自定义类型 `fileTransportExecutor`，在 fn 闭包外构造带 FileTransport 的 scrapligo `*network.Driver`，闭包内调 `driver.SendConfigs(cmds)`，把响应转回 `*device.Response`。

**When to use:** 唯一可行的 service 层 e2e mock 路径（DeviceExecutor → ConnectionPool 链路不接受 transport 注入，必须绕过）。

**Why not extend DeviceExecutor:** `DeviceConnectionPool.createConnection` (line 335-423) 硬编码 `NewScrapliWrapper` / `NewScrapliWrapperWithPort`，强制走真实 SSH/Telnet + 凭证 + 设备可达性检查。无 functional option 或构造参数可注入 FileTransport。改造生产代码（加 `WithTransport` option）违反 CONTEXT "validation only / 不新增功能" 边界。

**Example:**

```go
// Source: 基于 scrapligo v1.4.0 driver/network/driver_test.go:78-136 prepareDriver 模式
// + 项目 portWriteExecutor 接口 (port_write_service.go:67)
//
// fileTransportExecutor 实现 portWriteExecutor 接口（不绕过 fn）：
//   - Open scrapligo *network.Driver 用 FileTransport + fixture 文件
//   - ExecuteCustom 内部构造 wrapper，调 fn(ctx, pc) 让 service 闭包真正跑 SendConfigs
//   - 转换 scrapligo Response → device.Response 喂回 service.parseConfigError

type fileTransportExecutor struct {
    fixturePath string                 // testdata/huawei_*.fixture 绝对路径
    platform    string                 // "huawei_vrp"
    privilegeOverride *network.PrivilegeLevels // 可选：自定义 priv levels
}

func (e *fileTransportExecutor) ExecuteCustom(
    ctx context.Context,
    deviceID string,
    fn func(context.Context, *device.PooledConnection) error,
    timeout time.Duration,
) error {
    // 关键：用 FileTransport 构造 scrapligo driver
    d, err := network.NewDriver(
        "dummy-device",
        options.WithTransportType(transport.FileTransport),
        options.WithFileTransportFile(e.fixturePath),
        options.WithTransportReadSize(1),         // scrapligo 自身测试惯例
        options.WithReadDelay(0),
        // Huawei VRP privilege levels（参考 scrapligo platform huawei_vrp.yaml）
        // 或用 platform.NewPlatform("huawei_vrp.yaml", ...) 自动加载
    )
    if err != nil {
        return fmt.Errorf("e2e: create driver: %w", err)
    }

    if err := d.Channel.Open(); err != nil {
        return fmt.Errorf("e2e: open channel: %w", err)
    }
    defer d.Channel.Close()

    // 构造 wrapper（最小化：只暴露 SendConfigs 给 fn）
    wrapper := newFileTransportWrapper(d)
    pc := &device.PooledConnection{Wrapper: wrapper}  // 注意：需绕过 PooledConnection 私有字段

    // 关键：调用 fn（service 提供的闭包）— 这是 Phase 51 mock 不做的事
    return fn(ctx, pc)
}
```

> **LANDMINE — planner 必须考虑：** `device.PooledConnection` 是 concrete struct 不是 interface，且字段（`wrapper`、`refCount`、`mu` 等）多为私有。e2e 测试不能在 `portwrite` 包外构造它。**3 个可选路径**（planner 选定）：
> 1. **改 portWriteExecutor fn 签名为更窄接口**（最小侵入）：把 `fn func(context.Context, *device.PooledConnection) error` 改为 `fn func(context.Context, interface{ SendConfigs([]string) ([]*device.Response, error) }) error` — 但这会破坏 production DeviceExecutor 兼容性。
> 2. **在 device 包加测试 helper**（`internal/device/test_helpers.go`，build tag `//go:build test_helpers` 或 export test）：暴露 `NewPooledConnectionForTest(wrapper *ScrapliWrapper)` —— 但 production 包加 test helper 不优雅。
> 3. **创建内部接口 narrow down**（推荐）：在 `portwrite` 包内定义 `portWriteConn` 接口（仅暴露 `SendConfigs`），让 production `*device.PooledConnection` 与 e2e mock 都满足它。但这要改 service 内 fn 签名 → production 代码改动，违反 "validation only" 边界。
> 4. **最务实路径（推荐 planner 评估）**：在 `port_write_e2e_test.go` 用 `//go:build e2e` tag 隔离 + 直接在 `portwrite` 包内（同包测试可以访问私有字段）构造 `*device.PooledConnection` —— 但 `PooledConnection` 在 `device` 包，跨包不能访问私有字段。
>
> **推荐路径（planner 决定）：** 路径 1 或 2 折中——在 device 包加一个 `NewPooledConnectionForE2E(wrapper *ScrapliWrapper) *PooledConnection` 公开工厂（与 production 用法隔离）。或路径 3 引入 `portWriteConn` 接口但只在 fn 闭包内使用（不破坏 production service 签名，因为接口是 fn 参数类型 — 但会破坏 `portWriteExecutor.ExecuteCustom` 签名）。
>
> **CONTEXT D-02 已暗示这是 planner 必须解决的核心问题**："researcher 确认 device.DeviceExecutor 如何接受 transport option" —— 研究结论：**不接受**，必须在 service 层注入 mock executor。planner 必须为 PooledConnection 跨包构造问题选定一个解法。

### Pattern 2: Fixture 文件格式（手写 Huawei VRP 字节流）

**What:** scrapligo FileTransport 把整个 fixture 文件 `io.ReadAll` 进内存，然后 `Read(n int)` **一次返回 1 个字节**（line 75-79）——fixture 是设备 IO 字节流的完整回放，包含 prompts、命令回显、config-mode 切换、输出。

**When to use:** 所有 happy path / device_rejected fixture。transport_error 不需要 fixture（直接让 driver.Open 失败或返回 connection refused 文本）。

**Example（参考 scrapligo `driver/network/test-fixtures/send-configs-simple.txt`）:**

```
Huawei>SYS
Huawei]system-view
Huawei]interface GE0/0/1
Huawei-GE0/0/1]shutdown
Huawei-GE0/0/1]quit
Huawei]return
Huawei>
```

**关键约束：**
- **fixture 必须包含 prompt + 回显命令**：scrapligo channel 读取时会把 prompt（`Huawei>`）和回显命令（`sys`）都读到 Response.Result 里，最终 channel 层会 strip prompt 但保留 result body。
- **config-mode 进入/退出由 platform YAML 定义**：Huawei VRP 的 system-view / interface view prompt pattern 在 `huawei_vrp.yaml`（scrapligo assets）。e2e 依赖该 YAML 正确解析 `]` 后缀的 config-mode prompt。
- **ReadSize=1**：scrapligo 测试惯例 `options.WithTransportReadSize(1)` —— FileTransport 一次读 1 字节，channel 层负责拼装。
- **Huawei VRP platform 文件名**：`huawei_vrp.yaml`（参 `device.PlatformName` line 69 + scrapligo assets/platforms/huawei_vrp.yaml）。
- **错误场景 fixture 示例**（`% Error`）：fixture 主体可包含 `% Error: Unrecognized command found at '^'.` 字面文本，scrapligo 把它读进 Response.Result，service.parseConfigError 命中 `rejectionMarkers` 第一条 `% Error:`。

**Source:** `[VERIFIED: scrapligo v1.4.0 transport/file.go:64-80 (Read 1 byte impl) + driver/network/driver_test.go:78-120 prepareDriver + driver/network/test-fixtures/send-configs-simple.txt content]`

### Anti-Patterns to Avoid

- **ANTIPATTERN: 扩展现有 mockDeviceExecutor 加 fake PooledConnection** — 即便加 fake PooledConnection，也不测 scrapligo 字节解析层（fixture → channel → Response）。CONTEXT D-01 已拒绝。Phase 51 mock 的核心价值是测 service 编排（已被 Phase 51 测过），核心漏洞是 fn 不被调用（FileTransport 才能补）。
- **ANTIPATTERN: 在 device 包加 functional option WithTransport()** — production 代码改动违反 "validation only / 不新增功能" 边界。本 phase 仅在测试代码注入 mock。
- **ANTIPATTERN: HTTP handler 层 e2e** — Phase 52 handler 零测试基建，gin test engine + mock Core（DB/Cache/OperLogService/CollectionSvc）全套 cold start 工作量过大。SC#1 字面 "endpoint paths" 已由 CONTEXT D-02 降级为 service 公开方法路径。
- **ANTIPATTERN: 自建 in-process sshd** — golang.org/x/crypto/ssh 实现华为 config-mode 交互状态机工作量最大，FileTransport 已能满足"in-process mock sshd"语义。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Mock SSH transport | 自建 sshd / 自定义 transport | `transport.NewFileTransport()` + `options.WithFileTransportFile()` | scrapligo v1.4.0 官方测试 transport；从文件回放设备 IO；scrapligo 自身测试栈使用 `[VERIFIED]` |
| SendConfigs 链路模拟 | 在 mock 里手写 Response 对象 | 让 FileTransport 真实跑 channel 层解析 | Phase 51 mock 已这么做但不调 fn；FileTransport 才测 fn 内 SendConfigs + channel strip prompt + parseConfigError 全链路 |
| operlog 常量断言 | 在 e2e 测试里重复断言 25 OperType | 复用 `internal/utils/operlog/regression_test.go` 已有断言 | 该文件已 pin 25 常量值 + 11 敏感关键词；SC#7 只需 `go test ./internal/utils/operlog/` 绿灯 |
| UAT 文档结构 | 从零设计 frontmatter 字段 | 复刻 `48-HUMAN-UAT.md` 完整结构 | v1.18 已验证的 site-visit deferral 模板；CONTEXT D-08 锁定 |
| scrapligo driver 构造 | 手写 privilege levels / prompt pattern | 用 `platform.NewPlatform("huawei_vrp.yaml", ...)` 自动加载 | scrapligo 自带华为 VRP platform YAML；手写易错 |

**Key insight:** scrapligo FileTransport 是**唯一能同时满足 SC#1 字面 "in-process mock sshd" 语义 + 不绕过 fn 闘包 + 不依赖真机**的方案。其他所有路径要么绕过 fn（Phase 51 mock 现状），要么需要真机（违反 SC#1 in-process）。

## Runtime State Inventory

> Phase 54 是**新功能验证 phase**（新建 e2e 测试 + 文档），**不是 rename / refactor / migration phase**。本节按 protocol 要求显式回答（即便大部分为 None）。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — 本 phase 不改 schema、不改业务数据。`sys_port_write_audit` 表在 Phase 52 已落地，本 phase 不动 | None |
| Live service config | None — `config.yaml` `request_encryption.exclude_paths` 不修改（D-04 锁定不加 exclude）。仅文档化现状 | None |
| OS-registered state | None — 不涉及 Task Scheduler / pm2 / launchd 注册 | None |
| Secrets/env vars | None — 不修改 SM4_KEY / DB_PASSWORD 等环境变量 | None |
| Build artifacts | None — e2e 测试是新增源文件；CHANGELOG/README 是文档；不产生需要 reinstall 的 egg-info / 全局包 | None |

**Nothing found in any category — verified by reading CONTEXT.md §In scope + §Out of scope + §Deferred Ideas。**

## Common Pitfalls

### Pitfall 1: FileTransport `Read` 在内容耗尽时 `select{}` 死锁
**What goes wrong:** FileTransport.Read 在 `len(t.content) == 0` 时调 `select {}`（无 case 永久阻塞）—— 这是 scrapligo 故意设计的"读完后挂起读 goroutine"行为（避免 race detector 派对）。如果 fixture 字节序列不够覆盖完整 channel IO（prompt + 命令 + 输出 + 退出 prompt），scrapligo channel 层会卡在等待下个 prompt 永久阻塞，e2e 测试超时挂死。
**Why it happens:** scrapligo channel 与 FileTransport 是 byte-stream 交互：channel 写命令（被 FileTransport `Write` 收集到 `Writes` slice 不做任何回放），然后 channel 读 prompt（FileTransport 从 `content` 切片返回字节）。如果 fixture 漏写一个 prompt 或一个回显命令，channel 解析 prompt pattern 失败 → 一直读 → FileTransport content 空 → `select{}` 死锁。
**How to avoid:** fixture 必须严格按"设备 IO 完整会话"组织：每条命令前有 prompt，命令后有 prompt + 输出。参考 scrapligo `send-configs-simple.txt` 模式：每行都是设备返回的字节流（包括 prompt + 回显命令 + 后续 prompt）。fixture 写完后用 `-count=1 -timeout=60s` 跑一次，卡住超过 30s 说明 fixture 漏字节。
**Warning signs:** `go test` 卡住不退出；`panic: test timed out`；`goroutine ... [select (no cases)]` 在 goroutine dump 中。

### Pitfall 2: PooledConnection 私有字段阻碍 e2e mock 注入
**What goes wrong:** `device.PooledConnection` 是 concrete struct 不是 interface；其字段 `wrapper *ScrapliWrapper`、`refCount int32`、`mu *sync.Mutex`、`pool *DeviceConnectionPool` 多为私有（小写）。e2e 测试在 `portwrite` 包外（即 `port_write_e2e_test.go` 在 `portwrite` 包，但 PooledConnection 在 `device` 包）无法直接构造 `*device.PooledConnection{wrapper: ...}`。
**Why it happens:** CONTEXT D-02 假设 DeviceExecutor 接受 transport option，实际研究确认 `DeviceConnectionPool.createConnection` (line 335-423) 硬编码 `NewScrapliWrapper`，不接受 transport 注入。e2e 必须绕过 DeviceExecutor → 直造 mock portWriteExecutor，但 mock 的 fn 闭包要调 `pc.GetWrapper()` 拿 `*ScrapliWrapper` —— `ScrapliWrapper` 也是 concrete struct，私有字段 `driver *network.Driver`。
**How to avoid:** planner 在 3-4 个解法中选一（见 Architecture Patterns Pattern 1 LANDMINE 块）。推荐路径：在 device 包加 `NewPooledConnectionForE2E(wrapper *ScrapliWrapper) *PooledConnection` 公开工厂；或更彻底——把 `portWriteExecutor` 接口的 fn 参数从 `*device.PooledConnection` 改为 narrower interface（但破坏 production service 签名，违反 validation only 边界）。
**Warning signs:** 编译错误 `cannot assign to field wrapper (not exported)`；或 `cannot use e.fileTransportConn (type ...) as type *device.PooledConnection`。

### Pitfall 3: scrapligo `test-fixtures/` 目录 landmine（CONTEXT D-01 隐含假设错误）
**What goes wrong:** CONTEXT.md Claude's Discretion 写 "优先复用 scrapligo `transport/test-fixtures/` 现成 fixture 改造"。**实测该目录只有 SSH 密钥对**（`dumbserver` + `dumbserver.pub`），**没有任何设备 IO fixture**。预录制 fixture 在另一处：`driver/network/test-fixtures/`，且是 Cisco IOSXE 风格（`C3560CX#`），**无 Huawei VRP fixture**。
**Why it happens:** scrapligo 项目结构里 `transport/test-fixtures/` 是 transport 层（system/standard SSH）functional 测试用的 SSH server 凭证；replay fixture 在 network driver 层。
**How to avoid:** 全部 fixture 手写。参考 `driver/network/test-fixtures/send-configs-simple.txt` 模式（Cisco 风格），改写为 Huawei VRP（`<Huawei>` prompt + `system-view` + `[Huawei]` + `[Huawei-GE0/0/1]` 等）。预估每 fixture ~50-200 字节，6-8 个共 ~1KB。
**Warning signs:** 跑 e2e 时 fixture 加载报 "no such file"；或测试预期华为 prompt 实际读到 Cisco prompt（fixture 复用错地方）。

### Pitfall 4: FileTransport `Write` 不回放（只收集）
**What goes wrong:** 假设 FileTransport 是双向回放（既读又写），实际 `Write(b []byte) error` 只把字节 append 到 `Writes [][]byte` slice 不做任何处理。这意味着 fixture **只模拟设备返回的字节**（设备 → scrapligo），不模拟 scrapligo 发送的字节（scrapligo → 设备，命令本身）。scrapligo channel 层自己负责把命令字符 write 出去，然后从 transport Read 等 prompt 回显——但 FileTransport 不会把 write 的字节回灌到 read。
**Why it happens:** FileTransport 设计假设 fixture 已经包含命令回显（设备 echo 命令到自己的输出流）。所以 fixture 必须在 prompt 之后手动写上 scrapligo 即将发送的命令（这就是为什么 scrapligo 自身 fixture 看起来 prompt + 命令 + prompt + 输出 交替）。
**How to avoid:** 写 fixture 时**预先知道 cmds 序列**（service 通过 RenderCommand 已确定），手动按命令顺序在 fixture 里 prompt 之后插入回显命令文本。若 cmds 改变（service 改模板），fixture 必须同步改 —— 这是 fixture 脆性根源。
**Warning signs:** channel 层报"无法识别 prompt"；或 SendConfigs 返回 `*Response{Result: ""}` （空结果）—— 说明 fixture 没回显命令，channel 没读到 prompt 之间的内容。

### Pitfall 5: SC#3 字面 "no SM2+SM4 wrap on SSH-derived paths" 措辞误导
**What goes wrong:** 文档化时若按字面写"写端点不加 SM2+SM4"，会引导后续运维误把 `/network/ports/write/*` 加入 exclude_paths —— 实际 CONTEXT D-04 锁定保持加密。
**Why it happens:** SC#3 作者混淆了两个加密层：SSH 是后端→设备协议（scrapligo SSH 连设备）；SM2+SM4 是 HTTP 请求体加密（前端→后端）。二者正交。
**How to avoid:** 在 `docs/安全和认证设计（国密）.md` 明确区分两层："SSH transport 加密（scrapligo 自管）" vs "HTTP 请求体 SM2+SM4 加密（中间件层）"；写端点保持 HTTP 加密，与 SSH 加密无关。**确认 config.yaml `request_encryption.exclude_paths` 不含 `/network/ports/write/*`**（已实证 line 91-99，D-04 锁定）。
**Warning signs:** 文档 review 时若有人提议 "把写端点加入 exclude_paths"，立即对照 D-04 拒绝。

## Code Examples

### Example 1: fileTransportExecutor 骨架（推荐 planner 实现）

```go
// Source: scrapligo v1.4.0 driver/network/driver_test.go:78-136 prepareDriver + 
// 项目 portWriteExecutor 接口 (internal/services/portwrite/port_write_service.go:66-68)
package portwrite

import (
    "context"
    "fmt"
    "time"

    "github.com/scrapli/scrapligo/driver/network"
    "github.com/scrapli/scrapligo/driver/options"
    "github.com/scrapli/scrapligo/platform"
    "github.com/scrapli/scrapligo/transport"
    "github.com/xingran-next/xingran-go-backend/internal/device"
)

// fileTransportExecutor 实现 portWriteExecutor，使用 scrapligo FileTransport 跑真实 SendConfigs。
// 关键：与 Phase 51 mockDeviceExecutor 不同，本 executor 真正调用 fn 闭包。
type fileTransportExecutor struct {
    fixturePath string  // testdata/huawei_*.fixture 绝对路径
}

func (e *fileTransportExecutor) ExecuteCustom(
    ctx context.Context,
    deviceID string,
    fn func(context.Context, *device.PooledConnection) error,
    timeout time.Duration,
) error {
    // 用 platform 自动加载 Huawei VRP privilege levels / prompt patterns
    p, err := platform.NewPlatform(
        "huawei_vrp.yaml",  // scrapligo assets/platforms/huawei_vrp.yaml
        "dummy-host",
        options.WithTransportType(transport.FileTransport),
        options.WithFileTransportFile(e.fixturePath),
        options.WithTransportReadSize(1),
        options.WithReadDelay(0),
    )
    if err != nil {
        return fmt.Errorf("e2e: create platform: %w", err)
    }

    d, err := p.GetNetworkDriver()
    if err != nil {
        return fmt.Errorf("e2e: get network driver: %w", err)
    }

    if err := d.Open(); err != nil {
        return fmt.Errorf("e2e: open driver: %w", err)  // → service translateErr → transport_error
    }
    defer d.Close()

    // LANDMINE: PooledConnection 私有字段问题，planner 选定解法
    // 选项 A: device.NewPooledConnectionForE2E(scrapliWrapper)
    // 选项 B: 改 portWriteExecutor fn 参数为 narrower interface
    pc := device.NewPooledConnectionForE2E(newE2EScrapliWrapper(d))

    // 关键：调用 fn 让 service.executeWrite 的闭包真正跑 SendConfigs
    return fn(ctx, pc)
}

// newE2EScrapliWrapper 构造一个最小化 *device.ScrapliWrapper，只暴露 SendConfigs。
// 注意：ScrapliWrapper.driver 字段私有，需通过 device 包公开工厂或反射注入。
func newE2EScrapliWrapper(d *network.Driver) *device.ScrapliWrapper {
    // planner 选定：device.NewScrapliWrapperFromDriver(d) 或类似公开工厂
    // 见 Architecture Patterns Pattern 1 LANDMINE 4 路径
    return device.NewScrapliWrapperFromDriverForE2E(d)
}
```

### Example 2: e2e happy path 测试骨架

```go
// Source: 参考 port_write_service_test.go:334 TestShutdown_Success + scrapligo prepareDriver 模式
package portwrite

import (
    "context"
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

// TestE2E_Shutdown_Huawei_HappyPath 真实链路：service → fn → scrapligo SendConfigs → FileTransport
func TestE2E_Shutdown_Huawei_HappyPath(t *testing.T) {
    db := newTestDB(t)
    seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

    fixturePath, _ := filepath.Abs("testdata/huawei_shutdown_success.fixture")
    exec := &fileTransportExecutor{fixturePath: fixturePath}
    mockColl := new(mockCollectionSvc)
    svc := newTestService(exec, mockColl, db)
    ctx := context.Background()

    mockColl.On("Enqueue", "device-1").Return(nil)

    result, err := svc.Shutdown(ctx, "port-1", "e2e-test")

    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "succeeded", result.Status)
    assert.False(t, result.NoOp)
    assert.Contains(t, result.CommandSent, "shutdown")

    mockColl.AssertExpectations(t)
}
```

### Example 3: Huawei VRP fixture 内容（参考 scrapligo send-configs-simple.txt 改写）

```
# testdata/huawei_shutdown_success.fixture
# 模拟 Huawei VRP 设备返回的字节流（service 通过 RenderCommand(Huawei, shutdown, ...) 会下发 cmds=["shutdown"]）
# scrapligo SendConfigs 会自动进 system-view → interface view → 下发 shutdown → return
# fixture 必须覆盖完整 config-mode 会话
<Huawei>
<Huawei>system-view
[Huawei]interface GE0/0/1
[Huawei-GE0/0/1]shutdown
[Huawei-GE0/0/1]quit
[Huawei]return
<Huawei>
```

### Example 4: operlog regression 不回归断言（复用既有）

```bash
# Source: SC#7 — 不需要写新断言，复用 internal/utils/operlog/regression_test.go
# 这条命令在 SC#6 / SC#7 验证闸门中执行
go test ./internal/utils/operlog/ -run 'TestOperType|TestRecordSignature|TestFilterSensitive' -v
# 期望：25 OperType 常量值稳定 + Record 5 参签名 + 11 敏感关键词不回归
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| scrapli (Python) + subprocess (ScrapliGo 早期) | scrapligo native Go (v1.4.0) | v1.4.0 (2024+) | 直接 Go API，无 Python 依赖；FileTransport 是 native Go 测试 transport |
| Java XingRan (sys_menu 1:1 移植) | Go 版独立实现 + Meta JSONB | xingran-go-backend 项目初始化 | Go 版 sys_menu schema 与 Java 不同；命令差异参考 STATE.md memory `xingran-menu-no-java-fields` |
| Phase 51 mockDeviceExecutor 不调 fn | FileTransport 跑真实 scrapligo SendConfigs | Phase 54 (本 phase) | 补 parseConfigError + BatchResult 编排在集成层的验证 |

**Deprecated/outdated:**
- ROADMAP / REQUIREMENTS 写 "scrapli v1.3.3"：D-10 已纠正为 v1.4.0（go.mod 锁定）
- SC#4 路径 `50-port-write-network-ports-planned/`：D-11 已纠正，实际 phase 50 目录是 `50-w1-vendor-templates-unit-tests-vendor-action-command-map`，UAT 文件放 phase 54 目录（D-08）

## Assumptions Log

> 所有标 `[ASSUMED]` 的 claim 必须 planner / discuss-phase 在执行前与用户确认。

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `device.NewPooledConnectionForE2E` 公开工厂（或等价路径）需要新增到 `internal/device/` 包 | Architecture Patterns Pattern 1 LANDMINE | 若 planner 选路径 3（narrower interface），需改 production service 签名，违反 "validation only" 边界。需用户确认是否接受在 device 包加 test helper |
| A2 | scrapligo `huawei_vrp.yaml` platform 文件能正确解析华为 config-mode prompt `[Huawei-GE0/0/1]` | Pattern 2 | 若 platform YAML prompt pattern 不匹配 fixture，channel 卡死（Pitfall #1）。需 planner 先用 `platform.NewPlatform("huawei_vrp.yaml", ...)` + 简单 fixture 跑通一次验证 |
| A3 | fixture 手写工作量 ~6-8 个 × 50-200 字节可接受 | Standard Stack Alternatives | 若实际华为 prompt 复杂度超预期，fixture 数量可能膨胀到 10-12 个，工作量增加 50%。planner 可考虑先用 1 个 happy path 验证可行后再补全 |
| A4 | Phase 51 `newTestDB` / `seedPortAndDevice` 可直接复用 | Code Examples Example 2 | 若 sqlite `:memory:` 与 PostgreSQL 在 dot1x_enabled bool / NetworkDevice schema 行为有差异，e2e 可能误报。但 Phase 51 已用同模式，风险低 |

**风险评估总览：** A1 是最高风险假设 —— PooledConnection 私有字段问题是 FileTransport 路径的唯一架构阻塞点。planner 必须在 Wave 0 之前与用户确认是否接受在 device 包加 test helper（最小侵入），或选择其他路径。

## Open Questions

### 1. **PooledConnection 私有字段注入路径**（最高优先级，A1）
- **What we know:** `*device.PooledConnection` 字段私有（wrapper / refCount / mu / pool），跨包无法直接构造；`portWriteExecutor.ExecuteCustom` 的 fn 参数签名锁定为 `func(context.Context, *device.PooledConnection) error`。
- **What's unclear:** planner 应在 4 个解法中选哪个（见 Architecture Patterns Pattern 1 LANDMINE）。
- **Recommendation:** planner 在 PLAN.md Wave 0 列出 4 路径 trade-off，由 discuss-phase 或用户最终确认。推荐路径：device 包加 `NewPooledConnectionForE2E(wrapper *ScrapliWrapper) *PooledConnection` 公开工厂（最小侵入 + 不破坏 production 签名 + 测试代码与 production 用法物理隔离）。

### 2. **fixture 字节序列精确度验证**
- **What we know:** scrapligo 自身 fixture（`send-configs-simple.txt`）是 Cisco IOSXE 风格；华为 VRP 风格需手写。
- **What's unclear:** 华为 VRP platform YAML 的 prompt pattern 具体正则；fixture 必须精确匹配否则 channel 卡死。
- **Recommendation:** planner 第一个 task 是"用 1 个 happy path fixture 验证 platform 加载 + channel strip prompt 正确"，验证可行后再扩展到 6-8 个 fixture。可参考 `C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\platform\assets\platforms\huawei_vrp.yaml`（若存在）。

### 3. **transport/test-fixtures/ landmine 已确证（不再是 open question）**
- **What we know:** 该目录只含 SSH 密钥对，无设备 IO fixture。CONTEXT D-01 Claude's Discretion 路径不通。
- **Recommendation:** planner 必须显式说明"全部 fixture 手写"，并按 scrapligo `driver/network/test-fixtures/send-configs-simple.txt` 模式改写为 Huawei VRP 风格。

## Environment Availability

> Phase 54 依赖外部库（scrapligo / testify）但都在 go.mod 已锁定；不依赖运行时服务（DB / Redis / 真机 SSH）。

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| scrapligo v1.4.0 source | FileTransport API 确认 + huawei_vrp.yaml platform 加载 | ✓ | v1.4.0 (go.mod) | — |
| Go 1.24 toolchain | e2e test 编译运行 | ✓ | go1.24.5 (CLAUDE.md) | — |
| testify v1.11.1 | assert + mock 断言 | ✓ | v1.11.1 (go.sum) | — |
| sqlite `:memory:` | Phase 51 newTestDB 模式 | ✓ | gorm.io/driver/sqlite v1.5.4 | — |
| PostgreSQL | 本 phase **不需要**（e2e 用 sqlite） | — | — | — |
| Redis | 本 phase **不需要**（e2e 不测 cache layer） | — | — | — |
| 真机 SSH 设备 | SC#4 真机 UAT → 54-HUMAN-UAT.md 推迟 | ✗ | — | FileTransport + 现场访问 owner |
| Node.js / npm | SC#6 npm run build + type-check | ✓ | 见 xingran-react-frontend/package.json | — |

**Missing dependencies with no fallback:** None — 所有测试依赖在 go.mod 已锁定，运行时依赖（PG/Redis）本 phase 不需要，真机 SSH 通过 site-visit deferral 路径处理。

**Missing dependencies with fallback:** 真机 SSH 设备（无 fallback，但 SC#4 已显式 deferral 到 54-HUMAN-UAT.md，不阻塞 phase 完成）。

## Validation Architecture

> `workflow.nyquist_validation: true` 已确认（`.planning/config.json`）。本节为 planner 提供 Nyquist Dimension 8 测试映射。

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + `stretchr/testify v1.11.1`（assert + mock）+ `gorm.io/driver/sqlite v1.5.4`（in-memory DB） |
| Config file | 无（Go 标准 `go test`，无配置文件） |
| Quick run command | `go test ./internal/services/portwrite/ -run "TestE2E_" -count=1 -timeout=60s` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SC#1 / SSH-01..05 | FileTransport 真实链路 SendConfigs（5 single + 1 batch happy path） | e2e (service + scrapligo) | `go test ./internal/services/portwrite/ -run "TestE2E_.*_HappyPath" -count=1 -timeout=60s` | ❌ Wave 1 创建 |
| SC#1 / SSH-02 transport_error | fixture/connection 失败 → WriteErrorTransport | e2e error path | `go test ./internal/services/portwrite/ -run "TestE2E_.*TransportError" -count=1` | ❌ Wave 1 创建 |
| SC#1 / SSH-02 device_rejected | `% Error: Unrecognized command` fixture → WriteErrorDeviceRejected | e2e error path | `go test ./internal/services/portwrite/ -run "TestE2E_.*DeviceRejected" -count=1` | ❌ Wave 1 创建 |
| SC#1 / BATCH-02 fail-fast | 第 2 端口失败 → Succeeded=[1] Failed=[1] Skipped=[]（剩余 break） | e2e batch | `go test ./internal/services/portwrite/ -run "TestE2E_Batch_.*FailFast" -count=1` | ❌ Wave 1 创建 |
| SC#1 / PORT-06 skipped | pre-state 匹配 → NoOp=true Status="skipped" | e2e NoOp path | `go test ./internal/services/portwrite/ -run "TestE2E_.*NoOp" -count=1` | ❌ Wave 1 创建（PORT-06 已在 Phase 51 unit test 覆盖，e2e 不必重复，但若 fixture 简单可补） |
| SC#2 API 文档 | 6 端点签名 + schema 文档化 | manual (docs) | `grep -c "网络设备端口写操作" docs/API响应规范.md` (期望 ≥1) | ❌ Wave 2 创建 |
| SC#3 加密行为 | 写端点保持 SM2+SM4 加密文档化 + config.yaml 实证 | manual (docs) + config grep | `grep -c "/network/ports/write" configs/config.yaml` (期望 0 = 未在 exclude_paths) | ✅ config.yaml 已实证；❌ Wave 2 docs 改 |
| SC#4 UAT 推迟 | 54-HUMAN-UAT.md 6 项 SSH verification + WR-02 观察条目，全部 pending | manual (planning) | `test -f .planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md && grep -c "pending" 54-HUMAN-UAT.md` (期望 ≥6) | ❌ Wave 2 创建 |
| SC#5 文档更新 | README + CHANGELOG + MILESTONES.md v1.19 entry | manual (docs) | `test -f CHANGELOG.md && grep -c "v1.19" CHANGELOG.md README.md .planning/MILESTONES.md` | ❌ Wave 2 创建（CHANGELOG 新建） |
| SC#6 全量回归 | go test ./... + npm build + type-check 三绿 | automated gate | `go test ./... && (cd xingran-react-frontend && npm run build && npm run type-check)` | ✅ infra exists；❌ phase 末跑 |
| SC#7 operlog regression | 25 OperType + 11 keyword + Record 5 参 不回归 | unit regression | `go test ./internal/utils/operlog/ -run "TestOperType\|TestRecordSignature\|TestFilterSensitive" -count=1` | ✅ Exists (`internal/utils/operlog/regression_test.go`) |

### Sampling Rate

- **Per task commit:** `go test ./internal/services/portwrite/ -count=1 -timeout=60s`（quick：只跑 portwrite 包，包含 Phase 51 mock + Phase 54 e2e）
- **Per wave merge:** `go test ./... -count=1`（full Go suite）+ `cd xingran-react-frontend && npm run build && npm run type-check`（前端不回归）
- **Phase gate:** 三绿全过才能进入 `/gsd:verify-work`：(1) `go test ./...` exit 0；(2) `cd xingran-react-frontend && npm run build` exit 0；(3) `cd xingran-react-frontend && npm run type-check` exit 0；(4) `go test ./internal/utils/operlog/ -run "TestOperType\|TestRecordSignature\|TestFilterSensitive"` exit 0

### Wave 0 Gaps

- [ ] `internal/services/portwrite/port_write_e2e_test.go` — Wave 1 新建；覆盖 REQ-SC#1 / SSH-01..05 / PORT-01..06 / BATCH-01..04 e2e 路径
- [ ] `internal/services/portwrite/testdata/*.fixture` — Wave 1 新建 6-8 个 Huawei VRP fixture 文件
- [ ] **（可能需要）** `internal/device/e2e_helpers.go` 或类似公开工厂 — Wave 0 第一 task 解决 PooledConnection 私有字段注入（A1 假设）—— 见 Open Questions #1
- [ ] `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` — Wave 2 新建 UAT 推迟文档
- [ ] `docs/API响应规范.md` 改 + `docs/安全和认证设计（国密）.md` 改 + `CHANGELOG.md` 新建 + `README.md` 改 + `.planning/MILESTONES.md` 改 + `.planning/STATE.md` 改 — Wave 2 文档批次

*(Framework install：无 —— go.mod 已锁定所有 Go 依赖；前端 framework 在 Phase 53 已 audit。)*

## Security Domain

> `security_enforcement` 在 config.json 无显式 key，按 protocol "absent = enabled" 处理。但 Phase 54 是 **validation + docs** phase，不改任何安全相关代码 / 中间件 / 凭证处理 —— 仅文档化现状（D-04 锁定不改 config.yaml exclude_paths）。本节按 ASVS 简化列出适用项。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 本 phase 不动认证代码 |
| V3 Session Management | no | 本 phase 不动 session |
| V4 Access Control | yes (文档化) | PERM-01..03 已在 Phase 52 落地 `network:port:write` 权限；本 phase 仅文档化（API 规范、CHANGELOG） |
| V5 Input Validation | yes (文档化) | request schema 在 Phase 52 handler 已 bind+validate；本 phase 仅文档化（API 响应规范） |
| V6 Cryptography | yes (实证文档化) | SC#3 文档化：写端点保持 SM2+SM4 HTTP 请求体加密（D-04 锁定不加 exclude_paths）；config.yaml:91-99 实证 exclude_paths 不含 `/network/ports/write/*` `[VERIFIED: configs/config.yaml:88-117]` |
| V7 Logging | yes (regression guard) | SC#7 operlog 25 OperType + 11 敏感关键词 regression_test.go 不回归；sys_port_write_audit 真相源（Phase 52 落地）本 phase 不动 |

### Known Threat Patterns for v1.19 写端点

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 写端点误入 exclude_paths → request body 裸传 | Information disclosure | D-04 锁定不加 exclude_paths；docs 文档化加密行为；config grep 闸门（`grep /network/ports/write config.yaml` = 0） |
| operlog 脱敏关键词漏词 → 敏感字段进 sys_oper_log | Information disclosure | SC#7 regression_test.go 锁定 11 关键词不回归；sys_port_write_audit 作为未脱敏真相源（旁路） |
| fixture 中误含真机 IP / 凭证 | Information disclosure (test artifact) | fixture 用 `dummy-host` 占位；不写真 IP；e2e 用 sqlite `:memory:` 无真实数据 |

## Sources

### Primary (HIGH confidence)

- **scrapligo v1.4.0 source code** (`C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\`) — `transport/file.go:14-87` NewFileTransport / File.Open/Read/Write 完整实现；`driver/options/transportfile.go:10-22` WithFileTransportFile option；`driver/options/generic.go:14-38` WithTransportType 4 类型枚举含 FileTransport；`transport/factory.go:24-93` NewTransport 工厂；`driver/network/driver_test.go:78-136` prepareDriver 测试模式；`driver/network/test-fixtures/send-configs-simple.txt` Cisco 风格 fixture 范例
- **go.mod** — `github.com/scrapli/scrapligo v1.4.0`（D-10 锁定）；`github.com/stretchr/testify v1.11.1`；`gorm.io/driver/sqlite v1.5.4`
- **项目源码** — `internal/services/portwrite/port_write_service.go`（service 签名 + executeWrite fn 闭包）；`internal/services/portwrite/port_write_service_test.go:78-96` mockDeviceExecutor 不调 fn（Phase 51 现状）；`internal/services/portwrite/batch_orchestrator.go` fail-fast；`internal/services/portwrite/parse_error.go` parseConfigError 5 步；`internal/services/portwrite/pre_state_check.go` PORT-06；`internal/services/portcollection/vendor_port_template.go` RenderCommand + Huawei 模板；`internal/device/executor.go` DeviceExecutor.ExecuteCustom；`internal/device/connection_pool.go:335-423` createConnection 硬编码 NewScrapliWrapper（注入阻塞点）；`internal/device/scrapli_wrapper.go:93-217` NewScrapliWrapper/WithPort；`internal/utils/operlog/regression_test.go:45-71` 25 OperType 锁定
- **configs/config.yaml:88-117** — `request_encryption.exclude_paths` 实证不含 `/network/ports/write/*`（D-04 锁定）
- **.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md** — UAT 推迟模板（frontmatter + automated_gates + Tests site-visit + Summary + Owner + 关联声明）
- **.planning/config.json** — `workflow.nyquist_validation: true`（启用 Nyquist）

### Secondary (MEDIUM confidence)

- **CONTEXT.md D-01..D-11** — 11 项 locked decisions（用户提供，本研究验证 + 发现 1 处 landmine）
- **ROADMAP.md / REQUIREMENTS.md** — 36 v1.19 MVP 需求 mapping（已 ship Phase 50-53，本 phase validation only）

### Tertiary (LOW confidence)

- 无 — 本研究所有结论由源码 / 配置 / CONTEXT.md 直接验证，无 WebSearch-only 来源

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — go.mod / 源码直接读取验证；scrapligo v1.4.0 FileTransport API 完整阅读
- Architecture: HIGH — service / device / scrapligo 三层调用链通读；PooledConnection 注入阻塞点确证
- Pitfalls: HIGH — 5 项 pitfall 均来自源码 / 配置直接观察，非推测
- Open Questions: 1 项需用户确认（A1 PooledConnection 注入路径），3 项 planner 可自主决策（A2/A3/A4）

**Research date:** 2026-07-07
**Valid until:** 2026-08-06（30 天稳定期；scrapligo v1.4.0 / Go 1.24 / Phase 50-53 落地代码均稳定）
