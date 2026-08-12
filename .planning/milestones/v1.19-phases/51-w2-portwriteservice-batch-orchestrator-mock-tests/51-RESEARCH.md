# Phase 51: W2 — PortWriteService + Batch Orchestrator + Mock Tests - Research

**Researched:** 2026-07-06
**Domain:** Network device write service layer (SSH + scrapligo + GORM)
**Confidence:** HIGH

## Summary

Phase 51 introduces the service layer between the Phase 50 vendor template contract and the Phase 52 HTTP/audit/permission handlers. The work splits into three concerns: (1) a `PortWriteService` interface + private impl that owns SSH write operations for single ports, (2) a batch orchestrator that drives a serial fail-fast loop over 1-50 ports, and (3) comprehensive mock-based unit tests covering normal paths, error parsing, fail-fast, pre-state skipping, and detached context behavior.

The codebase already provides the necessary infrastructure: `device.DeviceExecutor.ExecuteCustom` for per-task lifecycle with timeout + panic recovery + pool reuse, `device.ScrapliWrapper.SendConfig/SendConfigs` for actual SSH command dispatch, `services.DeviceCredentialHelper` for SSH-06 credential resolution, `services.DeviceInfoCollectionService.Enqueue` for AUDIT-04 post-write refresh, and `models.DevicePortStatus` for PORT-06 pre-state checks. The pre-existing `MockCacheProvider` pattern in `cache_invalidator_test.go:14` plus the vendor template test style in `vendor_port_template_test.go` lock the testing approach.

Key technical decisions are already pinned in CONTEXT.md (D-10..D-18). The remaining open work is purely mechanical: implementing the service against those constraints, building the batch loop, writing the error parser with table-driven tests, and verifying the full mockable surface. No new external dependencies are required — the project already has `testify/mock` vendored for similar service-level unit tests.

**Primary recommendation:** Build the service as a thin orchestration layer (parse → render → execute → audit) that depends only on the `device.DeviceExecutor` interface (concrete pointer type for now, with mockable method signatures aligned to `ExecuteCustom`), `services.DeviceInfoCollectionService` (for `Enqueue`), and a GORM `*gorm.DB` for pre-state lookups. Define `portWriteServiceImpl` as a struct that holds these deps and exposes the 6-method interface. Put the batch orchestrator in a separate file in the same package so the parser (`parseConfigError`) and helpers can be unit-tested independently from the SSH-side flows.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### D-10: SSH 连接生命周期 = DeviceExecutor.ExecuteCustom 包装
- 选用 `DeviceExecutor.ExecuteCustom(ctx, deviceID, fn, timeout)` 是 project 现成模式：批量 50 端口 = 1 次 connection lifecycle + 内部循环 `SendConfig`/`SendConfigs` per port
- 同 device 池缓存：池按 deviceID 复用，所以"per-port Acquire/ReleaseRef 字面"和"per-batch 共用连接"在正确性上等价；但 ExecuteCustom 多一层 scheduler 提供的 timeout / 重试 / panic recover 保护
- `fn(ctx, *PooledConnection) error` 内：调用 `pc.GetWrapper().SendConfig(cmd)` 或 `SendConfigs(cmds)`，**端口循环之间不调 ReleaseRef**（fn 跳出 = scheduler 自动释放）
- 单端口接口（5 个）也走 ExecuteCustom — fn 内只发 1 条命令便退出，per-port 粒度的 timeout 可在 ExecuteCustom 的 `timeout` 参数独立控制

#### D-11: per-port SSH 超时
- ExecuteCustom 的 `timeout` 参数：单端口默认 30s（与 DeviceExecutor.DefaultExecutionConfig.Timeout 对齐），批量接口默认 60s（50 端口 × ~1s + SSH 延迟余量）
- timeout 越大越要小心：Core.Close() 30s deadline 已被 detached context 兜住（D-12），不会撞默认 30s 截止
- 若端口真实命令延迟超过 60s（罕见 Huawei S5735），需视现场调整 — 提为后续 Phase 优化项

#### D-12: 批量 detached context = 30min background context
- 批量接口进入 service 后第一行立即 `ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)`，**完全脱离 HTTP request context**
- 同步 GORM queries 也用 detached ctx（`s.db.WithContext(ctx)`）
- `defer cancel()` 排空 goroutine（即便 30min 超时也需 cancel 释放 timer）
- 避免 PROJECT.md "Critical Pitfalls #5: Batch execution exceeds 30s Core.Close deadline" — 这个是 S5700 系列固件 bulk write 实测可能 ~5min/批次的关键前提

#### D-13: PORT-06 pre-state 数据源 = DB 读 sys_device_port_status
- 单端口接口（5 个）：`s.db.WithContext(ctx).Where("id = ?", portID).First(&port)` 取 `admin_status` + `dot1x_enabled`
- 批量接口（1 个）：`s.db.WithContext(ctx).Where("device_id = ? AND id IN ?", deviceID, portIDs).Find(&ports)` 批量取，1 次 DB round-trip 拿全部
- 已匹配逻辑：
  - `Shutdown` 当 `admin_status == "down"` → skipped
  - `UndoShutdown` 当 `admin_status == "up"` → skipped
  - `EnableDot1x` 当 `dot1x_enabled == true` → skipped
  - `DisableDot1x` 当 `dot1x_enabled == false` → skipped
  - `SetDescription` 文本比对：DB.Description == req.Description → skipped（避免无变更触发设备侧回写 diff 噪声）
- DB 行不存在（端口"消失"——尚未被 portcollection cron 采集或已 delete）：fallback 跳过 pre-state 检查直接下发，避免误报

#### D-14: `skipped` 数组本 phase 填充
- 批量接口 `BatchResult.Skipped []PortResult` 填 PORT-06 pre-state 已匹配的端口
- 单端口接口 NoOp 返回 `(&PortResult{PortID: ..., Action: ..., NoOp: true, CurrentState: ...}, nil)` — handler 据此决定走 200 + operlog "无需操作" 还是 4xx 错误（建议 200 — 与 batch 语义一致）
- BATCH-03 partial result 形状 = `{Succeeded: []PortResult, Failed: []PortResult, Skipped: []PortResult}` 三数组共存，**单字段都有**

#### D-15: skipped 路径完整审计覆盖
- 单一 skipped 都进 `sys_port_write_audit`（即使 Phase 52 才建表）：表 `status` 枚举字段需含 `succeeded / failed / skipped` 三值（D-08 锁定 Phase 52 migration schema 预位）
- operlog 全记：用 `operlog.OperTypeStatus`(=10) 一致语义（与"动作未发生"业务含义最贴近）；不引入新 OperType
- single port 路径：写 1 条 operlog + 1 条 audit；batch 路径：每个端口（N 个）写 1 条 operlog + 1 条 audit（n×2 总计）

#### D-16: parseConfigError marker 优先级 = 顺序扫描
- 函数形态：`func parseConfigError(resp *device.Response) error`
- 扫描规则（顺序匹配，命中即返回）：
  1. 空响应 / `resp.Result == ""` + `resp.Err != nil` → `&WriteError{Kind: TransportError, Cause: resp.Err}`
  2. 包含 `% Error:` / `% Input error` / `Error: ` 前缀 → `device_rejected` + 提取错误行原文
  3. 包含 `Unrecognized command` / `Unknown command` / `Illegal` / `Invalid` / `Wrong parameter` → `device_rejected`
  4. 其他非空响应（含 OK / `Info:` / 空 Result） → `nil`（成功）
  5. 包含 `connection refused` / `timeout` / `EOF` / `i/o timeout` → `transport_error`
- 实现：私有 `const` 字符串切片或一个 `var rejectionMarkers = []string{...}` 表，单测表驱动覆盖（见 D-18）

#### D-17: 批量 fail-fast 语义
- 进入 batch 时校验：`len(req.PortIDs) > 50` → 返回 `ErrBatchTooLarge`（无 SSH 流量）；`len(req.PortIDs) == 0` → 返回 `ErrEmptyBatch`
- 内部循环：
  ```go
  for _, portID := range req.PortIDs {
      result, err := s.writeSinglePortInBatch(ctx, portID, req.Action, ...)
      switch {
      case err == nil && !result.NoOp:
          succeeded = append(succeeded, result)
      case result.NoOp:
          skipped = append(skipped, result)
      case err != nil && isTransportError(err):
          failed = append(failed, result)
          break  // fail-fast: transport 错立即停
      case err != nil && isDeviceRejected(err):
          failed = append(failed, result)
          break  // fail-fast: device_rejected 也停
      }
  }
  ```
- 失败点回传：`BatchResult.Failed[0].Error == "<具体原因>"` 让前端知道在哪里断
- 剩余未执行端口不进任何数组（视为"未尝试"，前端可二次发起）

#### D-18: 单测深度 = mock + inline 表驱动 marker 测试
- **service 测试**：用 `testify/mock`（项目惯例，`internal/services/operations/cache_invalidator_test.go:11` 已使用）实现 `mockDeviceExecutor` + `mockCollectionSvc`
  - `mockDeviceExecutor.ExecuteCustom` 返回预设 `*PooledConnection` 包裹的 fake `*ScrapliWrapper`，后者 `SendConfig` 返回预设 `*Response`
  - 覆盖：5 单端口方法正常路径 + 5 单端口错误路径（transport_error + device_rejected） + batch 正常 + batch fail-fast 两种 + batch 超 50 拒绝 + batch pre-state skip 全部归 skipped 数组 + single port DB 行缺失 fallback + audit/operlog 触发次数验证
- **parseConfigError 边界测试**（同 `_test.go` inline 表驱动）：10+ 用例覆盖 D-16 marker 优先级，包括：华为 `% Error:`、H3C `% Wrong parameter`、锐捷 `Unrecognized command`、空响应、超时字符串 mixed-in 等
- 不引 `testdata/` 目录（本 phase 无真机记错误响应样本），Phase 54 site-visit UAT 后如有需要再补 `testdata/write_errors/*.txt` + 真实 fixture 测试

### Claude's Discretion
- **sentinel errors 定义位置**：`port_write_service.go` 内 `var Err... = errors.New(...)` 还是单独 `errors.go` — 倾向同文件（保持 1 文件 1 关注点，与 Phase 50 D-09 同风格）
- **设备名校验逻辑**：批量接口的 `portID` 列表是否需要校验"全部同 device"——若 PASS 则单 connection 高效，错混不同 device 需拒绝或多个 batch 子循环。倾向强制要求 batch 内全同 device（错混返回 `ErrMixedDevices`）
- **RateLimiter 集成**：单端口接口是否需要 rate-limit 防 DoS（运维人员多次点击）——倾向不引，依赖前端按钮 disabled（UI-05）
- **PortStatus upsert 紧耦合**：单端口 write 成功后是否写一条 `sys_port_write_audit` placeholder audit row 即时反馈给后续 collector — **不写**，audit 表由 Phase 52 AUDIT-03 建表，service 仅返回 result，handler + Phase 52 migration 落表
- **mock struct 位置**：`port_write_service_test.go` 同目录内 inline mock vs `internal/services/portwrite/mocks_test.go` 单独文件 — 倾向同 `_test.go` 文件，遵循 cache_invalidator_test.go 风格

### Deferred Ideas (OUT OF SCOPE)
- operlog-exclude-paths.md (Phase 52 决策)
- v1.17 reconciliation (闭环)
- per-port 端 operator 记录字段（service 不存，handler 提取）
- rate-limit 单端口接口（UI 层防 DoS 足够）
- 混合设备 batch 自动拆分（抛 ErrMixedDevices 让客户端拆）
- operlog.Record 调用时机与脱敏兼容（handler 决策）
- real-fixture testdata/write_errors/*.txt（Phase 54 site-visit 后）
- 设备组策略（按 label/role 选端口）
- 写命令 dry-run 模式
- 跨厂商命令抽象
- 自动回滚
- 多用户并发写互斥
- 操作历史 UI
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SSH-02 | 解析 SendConfig 返回结果中的 `% Error:` / `Unrecognized command` / `Illegal` 标记，区分 `transport_error` 与 `device_rejected` | D-16 parseConfigError 形态 + 10+ 表驱动单测 |
| SSH-03 | 复用 PooledConnection.GetConnection + defer conn.ReleaseRef() 模式管理 SSH 连接生命周期 | D-10 通过 DeviceExecutor.ExecuteCustom 包装；连接池 GetConnection 已被 scheduler 内部调用 |
| SSH-04 | 批处理中使用脱离 HTTP 请求上下文的 30 分钟超时 context | D-12 detached context `context.WithTimeout(context.Background(), 30*time.Minute)` |
| SSH-06 | 通过 DeviceCredentialHelper 解析 device_id → 凭据 | D-10 service 调用前 `DeviceCredentialHelper.GetDeviceCredential(ctx, device)` |
| PORT-06 | 写命令执行前读取端口当前 admin_status / dot1x_enabled，对已处于目标状态的端口提示"无需操作" | D-13 DB 直读 sys_device_port_status；D-14 NoOp 路径填 Skipped |
| BATCH-02 | 以串行 + 失败即停（fail-fast）方式执行批量操作 | D-17 batch orchestrator fail-fast 循环 |
| BATCH-03 | 返回部分执行结果 {succeeded, failed, skipped} | D-14 三数组 + D-17 分类填充 |
| BATCH-04 | 限制批量操作最大端口数为 50 | D-17 入口校验 `len(req.PortIDs) > 50` → ErrBatchTooLarge |
| AUDIT-04 | 写操作成功 1-2 秒后通过 DeviceInfoCollectionService.Enqueue 触发采集 | D-15 service 成功路径 `collectionSvc.Enqueue(deviceID)` 异步触发 |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Vendor template resolution | Backend (portcollection) | — | Phase 50 落地 `RenderCommand(vendor, action, params)` 已在 `internal/services/portcollection/`，本 phase 直接消费 |
| SSH command dispatch | Backend (device) | — | `device.ScrapliWrapper.SendConfig/SendConfigs` 是项目唯一下行通道 |
| Connection lifecycle | Backend (device.DeviceExecutor) | Backend (device.DeviceConnectionPool) | D-10 锁定 ExecuteCustom 包装，pool 内部仍被 scheduler 复用 |
| Credential lookup | Backend (services.DeviceCredentialHelper) | — | SSH-06 显式调用 `GetDeviceCredential(ctx, device)`，不在 service 层重写 |
| Pre-state check | Backend (portwrite service + GORM) | — | D-13 直接读 `sys_device_port_status` 拿 admin_status / dot1x_enabled；不引缓存（PORT-06 只 2 字段、频次低） |
| Error parsing | Backend (portwrite) | — | D-16 私有 `parseConfigError` 解析 `*device.Response`；不入公共 API（handler 不需要） |
| Batch fail-fast loop | Backend (portwrite batch_orchestrator) | — | D-17 串行循环 + 错误分类 + 立即 break |
| Post-write collection enqueue | Backend (portwrite service) | Backend (DeviceInfoCollectionService) | D-15 service 成功路径调 `collectionSvc.Enqueue(deviceID)`（fire-and-forget） |
| Operlog | Backend (handler / operlog) | — | D-15 service **不**直接调 operlog（避免与 handler 重复）；handler 层在 Phase 52 落表 |
| Audit table write | Backend (handler) | — | D-15 service **不**写 `sys_port_write_audit`（表 Phase 52 才建），仅 result struct 携带 audit 所需字段 |
| HTTP binding | Frontend / handler | — | Phase 52 范围，本 phase 不触达 |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/xingran-next/xingran-go-backend/internal/device` | (project) | `DeviceExecutor.ExecuteCustom` + `ScrapliWrapper.SendConfig/SendConfigs` | 唯一 SSH 下行通道，含 scheduler 提供的 timeout/panic recovery/pool 复用 |
| `github.com/xingran-next/xingran-go-backend/internal/services/portcollection` | (project) | `RenderCommand(vendor, action, params)` 渲染 15 个 (vendor, action) 命令模板 | Phase 50 落地，service 零 SSH 流量直接消费 |
| `github.com/xingran-next/xingran-go-backend/internal/services` | (project) | `DeviceCredentialHelper` + `DeviceInfoCollectionService` | SSH-06 凭据解析 + AUDIT-04 采集触发，零新依赖 |
| `gorm.io/gorm` | v1.30.5 | DB 读 `sys_device_port_status` (PORT-06 pre-state) | 项目惯例 ORM |
| `github.com/stretchr/testify` | (vendored) | `assert` + `mock` 单测栈 | `cache_invalidator_test.go` + `vendor_port_template_test.go` 已使用 |
| `errors` / `fmt` (stdlib) | Go 1.24 | sentinel error + Errorf 包装 | 项目惯例 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `context` (stdlib) | Go 1.24 | 批量 detached 30min context (D-12) + HTTP request context propagation | 所有 service 入口首参 `ctx context.Context` |
| `strings` (stdlib) | Go 1.24 | `parseConfigError` 大小写不敏感 marker 扫描 | 仅 parseConfigError 内部使用 |
| `time` (stdlib) | Go 1.24 | `time.Duration` 用于 `ExecuteCustom` timeout 参数 (D-11) + `context.WithTimeout` | 常量定义 + 超时注入 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `DeviceExecutor.ExecuteCustom` | `DeviceConnectionPool.GetConnection` 直接调 + 手动 defer ReleaseRef | ExecuteCustom 多 scheduler 提供的 timeout / panic recovery；D-10 锁定 |
| `context.WithTimeout(context.Background(), 30*time.Minute)` (D-12) | 沿用 HTTP `c.Request.Context()` | PROJECT.md Pitfall #5：Core.Close 30s 截止会切断批量 — detached 是唯一规避 |
| 表驱动 `parseConfigError` marker 表 | 正则表达式 | D-16 锁定顺序扫描 + 私有 const 切片；正则易引入匹配爆炸 |
| 5 个独立单端口方法签名 | 单一 `WritePort(ctx, portID, action, desc, op)` | typed methods 对齐 operlog OperType 5 种调用（CONV-01..03 + 02 + 03），handler 翻译清晰 |

**No new packages required.** The Phase 51 implementation reuses the existing dependency tree end-to-end.

**Version verification:** `go.mod` already pins:
- `gorm.io/gorm v1.30.5`
- `github.com/stretchr/testify` (in vendor — see `go.sum`)
- `github.com/xingran-next/xingran-go-backend/internal/device`, `internal/services`, `internal/services/portcollection` (project)

## Package Legitimacy Audit

> Phase 51 installs **zero external packages**. All dependencies are project-internal (`internal/device`, `internal/services`, `internal/services/portcollection`) or already-vendored (`testify`, `gorm`). No slopcheck run is required.

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| (none) | — | — | — | — | n/a | No new external deps |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none
**Packages gated by `checkpoint:human-verify`:** none

## Architecture Patterns

### System Architecture Diagram

```
[HTTP Request (Phase 52)]
        ↓
[Gin Router] → [Handler] → [PortWriteService interface]
                                  ↓ (5 single + 1 batch)
                       ┌──────────────────────────────┐
                       │ portWriteServiceImpl         │
                       │                              │
                       │  1. CheckPortPreState        │ ←── GORM: sys_device_port_status
                       │     (single: First, batch:   │     (D-13 admin_status /
                       │      Find IN ?)              │      dot1x_enabled)
                       │                              │
                       │  2. GetDeviceCredential      │ ←── services.DeviceCredentialHelper
                       │     (SSH-06)                 │     (lookup device_id → cred)
                       │                              │
                       │  3. DeviceExecutor.          │ ←── device.DeviceExecutor (per-port 30s)
                       │     ExecuteCustom(ctx,       │     scheduler 内 GetConnection
                       │     deviceID, fn, timeout)   │     + defer ReleaseRef
                       │     fn 内 SendConfig/SendConfigs
                       │                              │
                       │  4. parseConfigError(resp)   │ ←── pure Go: marker 顺序扫描
                       │     → WriteError{Transport,  │     区分 transport vs device_rejected
                       │        DeviceRejected}       │
                       │                              │
                       │  5. Audit/Enqueue            │ ←── DeviceInfoCollectionService.Enqueue
                       │     (AUDIT-04, success only) │     (1-2s 后台采集, fire-and-forget)
                       └──────────────────────────────┘
                                  ↓
                        [PortResult / BatchResult]
                                  ↓
[Handler → Response] (Phase 52)
```

### Recommended Project Structure

```
internal/services/portwrite/
├── port_write_service.go      # PortWriteService interface + portWriteServiceImpl + NewPortWriteService
│                              # 5 single methods + BatchWritePorts
├── batch_orchestrator.go      # 批量 fail-fast 循环 + 校验（maxBatchSize=50, 同 device, 非空）
├── parse_error.go             # parseConfigError + WriteError type + marker consts
├── pre_state_check.go         # CheckPortPreState(ctx, portID, action) → NoOp result
├── setup.go                   # (可选) DI helper 包装 NewPortWriteService
└── port_write_service_test.go # 单测：mock executor + mock collectionSvc + DB mock
                               # 覆盖 5 single + 1 batch 正常 + 错误 + skipped 路径
```

**说明：** 单文件拆分 vs 一文件 6 方法 = 看复杂度。若 `parseConfigError` + `WriteError` 总行数 > 80，独立 `parse_error.go` 更利于单测（table-driven 测试只 import 必要部分）。若 < 50 行，可并入 `port_write_service.go` 末尾的 `// error parsing` 段。**倾向 D-18 + Claude discretion 选独立 `parse_error.go`**。

### Pattern 1: Service Interface + Private Impl

**What:** `type PortWriteService interface { 6 methods }` + `type portWriteServiceImpl struct { db, deviceExecutor, collectionSvc }` + `func NewPortWriteService(db, executor, svc) PortWriteService` 工厂。

**When to use:** 任何 5+ 方法集合的 service 入口（CLAUDE.md §Go Code Patterns + `operations/building_service.go` 范例）。

**Example:**
```go
// File: internal/services/portwrite/port_write_service.go
package portwrite

import (
    "context"
    "errors"
    "fmt"

    "github.com/xingran-next/xingran-go-backend/internal/device"
    "github.com/xingran-next/xingran-go-backend/internal/services"
    "github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
    "gorm.io/gorm"
)

// Sentinel errors
var (
    ErrBatchTooLarge = errors.New("portwrite: batch exceeds max size of 50")
    ErrEmptyBatch    = errors.New("portwrite: batch is empty")
    ErrMixedDevices  = errors.New("portwrite: batch contains ports from different devices")
    ErrPortNotFound  = errors.New("portwrite: port not found")
)

// Action 是 Phase 50 PortAction 的 alias，避免双 import 路径冲突
type Action = portcollection.PortAction

// PortResult 单端口写结果
type PortResult struct {
    PortID       string `json:"portId"`
    Action       Action `json:"action"`
    Status       string `json:"status"`       // "succeeded" | "failed" | "skipped"
    NoOp         bool   `json:"noOp"`
    CurrentState string `json:"currentState,omitempty"`
    Error        string `json:"error,omitempty"`
    CommandSent  string `json:"commandSent,omitempty"`
}

const maxBatchSize = 50

// PortWriteService 端口写 service 接口
type PortWriteService interface {
    Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error)
    EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    BatchWritePorts(ctx context.Context, req BatchWriteRequest, operator string) (*BatchResult, error)
}

type portWriteServiceImpl struct {
    db             *gorm.DB
    deviceExecutor *device.DeviceExecutor
    collectionSvc  *services.DeviceInfoCollectionService
}

func NewPortWriteService(
    db *gorm.DB,
    deviceExecutor *device.DeviceExecutor,
    collectionSvc *services.DeviceInfoCollectionService,
) PortWriteService {
    return &portWriteServiceImpl{
        db: db, deviceExecutor: deviceExecutor, collectionSvc: collectionSvc,
    }
}
```

### Pattern 2: parseConfigError 顺序 marker 扫描（D-16）

**What:** 私有函数 + `WriteError` 类型 + transport/rejection marker 表。优先级扫描，命中即返回。

**When to use:** 解析 scrapligo `*Response{Result, Err}` 时区分 transport vs device_rejected 错误。

**Example:**
```go
// File: internal/services/portwrite/parse_error.go
package portwrite

import (
    "errors"
    "fmt"
    "strings"

    "github.com/xingran-next/xingran-go-backend/internal/device"
)

// WriteErrorKind 写命令错误分类
type WriteErrorKind int

const (
    WriteErrorNone WriteErrorKind = iota
    WriteErrorTransport
    WriteErrorDeviceRejected
)

// WriteError 写命令错误
type WriteError struct {
    Kind    WriteErrorKind
    Cause   error
    Message string
}

func (e *WriteError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("write error (%s): %s: %v", errorKindName(e.Kind), e.Message, e.Cause)
    }
    return fmt.Sprintf("write error (%s): %s", errorKindName(e.Kind), e.Message)
}

func (e *WriteError) Unwrap() error { return e.Cause }

func errorKindName(k WriteErrorKind) string {
    switch k {
    case WriteErrorTransport:
        return "transport"
    case WriteErrorDeviceRejected:
        return "device_rejected"
    default:
        return "none"
    }
}

var transportMarkers = []string{"connection refused", "timeout", "EOF", "i/o timeout"}
var rejectionMarkers = []string{
    "% Error:", "% Input error", "Error: ",
    "Unrecognized command", "Unknown command",
    "Illegal", "Invalid", "Wrong parameter",
}

// parseConfigError 按优先级解析 SendConfig 返回。
// 1) nil resp → TransportError
// 2) resp.Err != nil → TransportError (Cause = resp.Err)
// 3) resp.Result == "" → nil (真空 = 成功)
// 4) 顺序匹配 transportMarkers → TransportError
// 5) 顺序匹配 rejectionMarkers → DeviceRejected
// 6) 其他（含 "Info:" / "OK"） → nil
func parseConfigError(resp *device.Response) error {
    if resp == nil {
        return &WriteError{Kind: WriteErrorTransport, Message: "nil response"}
    }
    if resp.Err != nil {
        return &WriteError{Kind: WriteErrorTransport, Cause: resp.Err}
    }
    if resp.Result == "" {
        return nil
    }
    lower := strings.ToLower(resp.Result)
    for _, m := range transportMarkers {
        if strings.Contains(lower, strings.ToLower(m)) {
            return &WriteError{Kind: WriteErrorTransport, Message: resp.Result}
        }
    }
    for _, m := range rejectionMarkers {
        if strings.Contains(resp.Result, m) {
            return &WriteError{Kind: WriteErrorDeviceRejected, Message: resp.Result}
        }
    }
    return nil
}

// isTransportError / isDeviceRejected 给 batch orchestrator 用
func isTransportError(err error) bool {
    var we *WriteError
    if errors.As(err, &we) {
        return we.Kind == WriteErrorTransport
    }
    return false
}

func isDeviceRejected(err error) bool {
    var we *WriteError
    if errors.As(err, &we) {
        return we.Kind == WriteErrorDeviceRejected
    }
    return false
}
```

### Pattern 3: ExecuteCustom 包装 + detached context（D-10 + D-12）

**What:** 单端口方法走 `ExecuteCustom` per-port 30s timeout；批量方法首行 `context.WithTimeout(context.Background(), 30*time.Minute)` detached 后内部调 ExecuteCustom per-port 60s timeout。

**When to use:** 任何需要 SSH 写操作且不希望被 HTTP / Core.Close 截止时间切断的 service 入口。

**Example:**
```go
// 单端口入口
func (s *portWriteServiceImpl) Shutdown(ctx context.Context, portID, operator string) (*PortResult, error) {
    return s.writeSinglePort(ctx, portID, portcollection.ActionShutdown, "", operator)
}

func (s *portWriteServiceImpl) writeSinglePort(
    ctx context.Context, portID string, action Action, desc, operator string,
) (*PortResult, error) {
    // 1. 加载端口元数据（含 device_id + vendor）
    var port models.DevicePortStatus
    if err := s.db.WithContext(ctx).Where("id = ?", portID).First(&port).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            // 端口"消失" — 跳过 pre-state, 直接下发
            return s.executeWrite(ctx, portID, "", action, desc, operator, "")
        }
        return nil, fmt.Errorf("query port: %w", err)
    }

    // 2. PORT-06 pre-state check
    if noopResult := s.checkPreState(&port, action, desc); noopResult != nil {
        return noopResult, nil
    }

    // 3. ExecuteCustom (30s timeout, per-port)
    return s.executeWrite(ctx, port.ID, port.DeviceID, action, desc, operator, port.InterfaceName)
}

func (s *portWriteServiceImpl) executeWrite(
    ctx context.Context, portID, deviceID string, action Action, desc, operator, interfaceName string,
) (*PortResult, error) {
    // 加载设备元数据
    var device models.NetworkDevice
    if err := s.db.WithContext(ctx).Where("id = ?", deviceID).First(&device).Error; err != nil {
        return nil, fmt.Errorf("query device: %w", err)
    }

    // 渲染命令
    cmds, err := portcollection.RenderCommand(device.Vendor, action, portcollection.PortTemplateParams{
        InterfaceName: interfaceName,
        Description:   desc,
    })
    if err != nil {
        return nil, err
    }

    var lastResp *device.Response
    executeErr := s.deviceExecutor.ExecuteCustom(ctx, deviceID, func(ctx context.Context, pc *device.PooledConnection) error {
        wrapper := pc.GetWrapper()
        responses, err := wrapper.SendConfigs(cmds)
        if err != nil {
            return err
        }
        if len(responses) > 0 {
            lastResp = responses[len(responses)-1]
        }
        return nil
    }, singlePortTimeout) // 30s per D-11

    // 解析结果
    if executeErr != nil {
        return &PortResult{
            PortID: portID, Action: action, Status: "failed",
            Error: executeErr.Error(),
            CommandSent: strings.Join(cmds, " | "),
        }, executeErr
    }
    if parseErr := parseConfigError(lastResp); parseErr != nil {
        return &PortResult{
            PortID: portID, Action: action, Status: "failed",
            Error: parseErr.Error(),
            CommandSent: strings.Join(cmds, " | "),
        }, parseErr
    }

    // 成功: 触发后台采集
    if s.collectionSvc != nil {
        _ = s.collectionSvc.Enqueue(deviceID)
    }
    return &PortResult{
        PortID: portID, Action: action, Status: "succeeded",
        CommandSent: strings.Join(cmds, " | "),
    }, nil
}

const (
    singlePortTimeout = 30 * time.Second  // D-11
    batchPortTimeout  = 60 * time.Second  // D-11
    batchDetachedTimeout = 30 * time.Minute // D-12
)
```

### Pattern 4: Batch Orchestrator Fail-Fast（D-17）

**What:** 入口校验（max=50, min=1, 同 device） → 批量查 pre-state → 逐 port serial loop with fail-fast break。

**When to use:** 批量写操作入口。

**Example:**
```go
// File: internal/services/portwrite/batch_orchestrator.go
package portwrite

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// BatchWriteRequest 批量写请求
type BatchWriteRequest struct {
    DeviceID    string   `json:"deviceId"`
    Action      Action   `json:"action"`
    PortIDs     []string `json:"portIds"`
    Description string   `json:"description,omitempty"`
}

// BatchResult 批量结果（D-14 + BATCH-03）
type BatchResult struct {
    Succeeded []PortResult `json:"succeeded"`
    Failed    []PortResult `json:"failed"`
    Skipped   []PortResult `json:"skipped"`
}

func (s *portWriteServiceImpl) BatchWritePorts(
    ctx context.Context, req BatchWriteRequest, operator string,
) (*BatchResult, error) {
    // D-12: detached context 30min
    ctx, cancel := context.WithTimeout(context.Background(), batchDetachedTimeout)
    defer cancel()

    // D-17: 入口校验
    if len(req.PortIDs) == 0 {
        return nil, ErrEmptyBatch
    }
    if len(req.PortIDs) > maxBatchSize {
        return nil, fmt.Errorf("%w: got %d", ErrBatchTooLarge, len(req.PortIDs))
    }
    if req.DeviceID == "" {
        return nil, errors.New("portwrite: deviceId is required")
    }

    // D-13: 批量 pre-state 查询 (1 DB round-trip)
    var ports []models.DevicePortStatus
    if err := s.db.WithContext(ctx).
        Where("device_id = ? AND id IN ?", req.DeviceID, req.PortIDs).
        Find(&ports).Error; err != nil {
        return nil, fmt.Errorf("query ports: %w", err)
    }
    preStateMap := make(map[string]models.DevicePortStatus, len(ports))
    for _, p := range ports {
        preStateMap[p.ID] = p
    }

    result := &BatchResult{
        Succeeded: []PortResult{},
        Failed:    []PortResult{},
        Skipped:   []PortResult{},
    }

    // D-17: serial fail-fast loop
    for _, portID := range req.PortIDs {
        port, exists := preStateMap[portID]
        if !exists {
            // 端口"消失" — 直接下发（D-13 fallback）
            writeResult, err := s.executeWrite(ctx, portID, req.DeviceID, req.Action, req.Description, operator, "")
            if err != nil {
                result.Failed = append(result.Failed, *writeResult)
                break // fail-fast
            }
            if writeResult.NoOp {
                result.Skipped = append(result.Skipped, *writeResult)
                continue
            }
            result.Succeeded = append(result.Succeeded, *writeResult)
            continue
        }

        // pre-state check
        if noopResult := s.checkPreState(&port, req.Action, req.Description); noopResult != nil {
            result.Skipped = append(result.Skipped, *noopResult)
            continue
        }

        // execute
        writeResult, err := s.executeWrite(ctx, port.ID, port.DeviceID, req.Action, req.Description, operator, port.InterfaceName)
        if err != nil {
            result.Failed = append(result.Failed, *writeResult)
            if isTransportError(err) || isDeviceRejected(err) {
                break // fail-fast
            }
            // 其它 error (e.g. DB transient) 也立即停 — 与 D-17 fail-fast 语义一致
            break
        }
        if writeResult.NoOp {
            result.Skipped = append(result.Skipped, *writeResult)
            continue
        }
        result.Succeeded = append(result.Succeeded, *writeResult)
    }
    return result, nil
}
```

### Pattern 5: Pre-state Check（D-13 + D-14）

**What:** 单端口 + 批量统一走 `checkPreState` 函数。匹配则返回 `&PortResult{NoOp: true, ...}`，不匹配返回 `nil`（继续执行）。

**Example:**
```go
// File: internal/services/portwrite/pre_state_check.go
package portwrite

import (
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// checkPreState 返回 NoOp PortResult if action 不需执行；返回 nil if 需要下发。
func (s *portWriteServiceImpl) checkPreState(
    port *models.DevicePortStatus, action Action, desc string,
) *PortResult {
    switch action {
    case portcollection.ActionShutdown:
        if port.AdminStatus == "down" {
            return &PortResult{PortID: port.ID, Action: action, Status: "skipped", NoOp: true, CurrentState: "admin_down"}
        }
    case portcollection.ActionUndoShutdown:
        if port.AdminStatus == "up" {
            return &PortResult{PortID: port.ID, Action: action, Status: "skipped", NoOp: true, CurrentState: "admin_up"}
        }
    case portcollection.ActionEnableDot1x:
        if port.Dot1xEnabled {
            return &PortResult{PortID: port.ID, Action: action, Status: "skipped", NoOp: true, CurrentState: "dot1x_enabled"}
        }
    case portcollection.ActionDot1xDisable:
        if !port.Dot1xEnabled {
            return &PortResult{PortID: port.ID, Action: action, Status: "skipped", NoOp: true, CurrentState: "dot1x_disabled"}
        }
    case portcollection.ActionDescription:
        if port.Description == desc {
            return &PortResult{PortID: port.ID, Action: action, Status: "skipped", NoOp: true, CurrentState: "description_match"}
        }
    }
    return nil
}
```

### Pattern 6: Mock-based Unit Test with testify/mock

**What:** 嵌入式 `mock.Mock` + `Called`/`Return` + `AssertExpectations` 模式（cache_invalidator_test.go 风格）。

**When to use:** 任何 service 入口需要 mock DeviceExecutor + DB + CollectionService 的场景。

**Example:**
```go
// File: internal/services/portwrite/port_write_service_test.go
package portwrite

import (
    "context"
    "errors"
    "testing"
    "time"

    "github.com/xingran-next/xingran-go-backend/internal/device"
    "github.com/xingran-next/xingran-go-backend/internal/services"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

// mockDeviceExecutor 嵌入 mock.Mock 实现 service.DeviceExecutor 接口
// 注意: Phase 51 service 直接用 *device.DeviceExecutor (concrete pointer),
//       mock 通过定义一个等价的 interface + 调整 service 字段类型实现
type mockDeviceExecutor struct {
    mock.Mock
}

// 若 service 字段类型改为 interface, 此处签名 = interface 定义
func (m *mockDeviceExecutor) ExecuteCustom(
    ctx context.Context, deviceID string,
    fn func(context.Context, *device.PooledConnection) error,
    timeout time.Duration,
) error {
    args := m.Called(ctx, deviceID, timeout)
    // 触发 fn 回调
    if fn != nil {
        if err := fn(ctx, nil); err != nil {
            return err
        }
    }
    return args.Error(0)
}

type mockCollectionSvc struct {
    mock.Mock
}

func (m *mockCollectionSvc) Enqueue(deviceID string) error {
    args := m.Called(deviceID)
    return args.Error(0)
}

func TestParseConfigError(t *testing.T) {
    tests := []struct {
        name     string
        resp     *device.Response
        wantKind WriteErrorKind
        wantErr  bool
    }{
        {"huawei_percent_error", &device.Response{Result: "% Error: Unrecognized command found at '^'."}, WriteErrorDeviceRejected, true},
        {"h3c_wrong_parameter", &device.Response{Result: "% Wrong parameter found at '^'."}, WriteErrorDeviceRejected, true},
        {"ruijie_unrecognized", &device.Response{Result: "Unrecognized command"}, WriteErrorDeviceRejected, true},
        {"huawei_ok_info", &device.Response{Result: "Info: configuration succeeded"}, WriteErrorNone, false},
        {"huawei_ok_empty", &device.Response{Result: ""}, WriteErrorNone, false},
        {"nil_response", nil, WriteErrorTransport, true},
        {"err_set", &device.Response{Err: errors.New("i/o timeout")}, WriteErrorTransport, true},
        {"connection_refused_text", &device.Response{Result: "connection refused"}, WriteErrorTransport, true},
        {"timeout_text", &device.Response{Result: "i/o timeout while writing"}, WriteErrorTransport, true},
        {"eof_text", &device.Response{Result: "unexpected EOF"}, WriteErrorTransport, true},
        {"illegal_param", &device.Response{Result: "Illegal parameter value at '^'."}, WriteErrorDeviceRejected, true},
        {"unknown_command", &device.Response{Result: "Unknown command"}, WriteErrorDeviceRejected, true},
        {"info_only", &device.Response{Result: "Info: This command will take effect after save"}, WriteErrorNone, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := parseConfigError(tt.resp)
            if tt.wantErr {
                assert.Error(t, err)
                var we *WriteError
                if assert.True(t, errors.As(err, &we)) {
                    assert.Equal(t, tt.wantKind, we.Kind)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestBatchWritePorts_ExceedsMax(t *testing.T) {
    // 构造 51 个 portID
    portIDs := make([]string, 51)
    for i := range portIDs {
        portIDs[i] = fmt.Sprintf("port-%d", i)
    }
    svc := &portWriteServiceImpl{db: nil} // BatchWritePorts 入口校验不需 DB
    result, err := svc.BatchWritePorts(context.Background(), BatchWriteRequest{
        DeviceID: "dev-1", Action: ActionShutdown, PortIDs: portIDs,
    }, "test-op")
    assert.ErrorIs(t, err, ErrBatchTooLarge)
    assert.Nil(t, result)
}

// 更多测试: TestShutdown_Success / TestShutdown_TransportError / TestShutdown_NoOpAlreadyDown
// TestBatchWritePorts_FailFast_Transport / TestBatchWritePorts_FailFast_DeviceRejected
// TestBatchWritePorts_AllSkipped / TestBatchWritePorts_Success
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SSH 连接管理 | 自实现 connection pool / 多线程调度 | `device.DeviceExecutor.ExecuteCustom` | 内置 pool 复用 + scheduler timeout + panic recovery + scrapligo race mitigation (`acquireOp`/`releaseOp`) — D-10 |
| Scrapli 命令发送 | 自实现 SSH 包 | `device.ScrapliWrapper.SendConfig`/`SendConfigs` | 已封装 `opMu` 锁 + EOF/connection-error 状态标记 + GetPrompt 序列化 — Phase 50 链路 |
| 厂商命令渲染 | 在 service 写 `if vendor == "huawei" { ... }` | `portcollection.RenderCommand(vendor, action, params)` | Phase 50 已锁 + 12+ 单测覆盖 15 模板 + sentinel errors 完备 — 重新写等于抛弃测试 |
| 凭据解析 | service 内写 SQL 取 `sys_auth_credential` | `services.DeviceCredentialHelper.GetDeviceCredential(ctx, device)` | SSH-06 锁定调用 + 已有 fallback 默认凭证逻辑 + SM4 解密集成 |
| 写后采集触发 | service 写 cron / goroutine 调度 | `services.DeviceInfoCollectionService.Enqueue(deviceID)` | AUDIT-04 + 已有 dedup 逻辑（pending/running 不重复入队）+ worker pool 复用 |
| 端口预状态查询 | service 写复杂 SQL + GORM join | `s.db.Where("id = ?", portID).First(&port)` 直接读 model | `models.DevicePortStatus` 已含 `AdminStatus` + `Dot1xEnabled` + GORM soft delete 集成 |
| 测试 mock 框架 | 手写 stub 函数 / 自实现 mock | `testify/mock` (`Mock` 嵌入 + `Called` + `Return` + `AssertExpectations`) | `cache_invalidator_test.go` 已在用 + 标准 Go 测试生态 |

**Key insight:** Phase 51 的 service 层本质是一个**调度 / 编排**层（parse → render → execute → audit），不引入新概念。所有"复杂度"都已被前序 phase 锁定到具体 helper（`RenderCommand`、`ExecuteCustom`、`Enqueue`、`GetDeviceCredential`）。Phase 51 的产出 = 把这些 helper 按 CONTEXT.md D-10..D-18 的契约串起来 + 写测试覆盖。**不要在 service 层重新实现任何底层能力。**

## Runtime State Inventory

> Phase 51 creates the service layer. No data migration; no env var rename; no OS-registered state.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — `sys_port_write_audit` 表 Phase 52 才建（INFRA-01） | None in this phase |
| Live service config | None — service 由 Phase 52 handler 在 router 中显式注入；不读 sys_config | None |
| OS-registered state | None — 无 Windows Task Scheduler / pm2 / systemd 注册项 | None |
| Secrets/env vars | None — service 不读新 env var；凭据解析仍走 DeviceCredentialHelper（已有 SM4 cipher） | None |
| Build artifacts | None — 仅新增 `.go` 文件；`go build ./...` 不需要数据迁移 | None |

**Nothing found in any category** — service layer is pure Go code with zero runtime footprint beyond what Phase 50 / device layer already established.

## Common Pitfalls

### Pitfall 1: 批量接口未脱离 HTTP context → Core.Close 30s 切断
**What goes wrong:** 批量接口若用 `ctx := c.Request.Context()` 继承 HTTP context，S5700 系列固件 bulk write 实测可能耗时 5min，被 `core.Core.Close()` 30s 截止切断 → 部分端口写一半状态不可知
**Why it happens:** 开发者惯性 `func(ctx context.Context, ...)` 直接转发
**How to avoid:** D-12 锁定首行 `ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute); defer cancel()`
**Warning signs:** 单测中 context 5min 后未触发 `ctx.Done()` 路径；或 `core.Close()` 后批量仍在跑

### Pitfall 2: per-port Acquire/ReleaseRef 误用导致死锁
**What goes wrong:** 在 `ExecuteCustom` 的 fn 内手工调 `pc.ReleaseRef()` → scheduler 仍持锁 → 下一端口等锁 → 30min 超时
**Why it happens:** D-10 锁定 fn 跳出 = scheduler 自动释放；二次释放触发 refCount panic 或 unlock-of-unlocked-mutex（参见 `connection_pool.go:97-108` 警告）
**How to avoid:** fn 内部**只调** `pc.GetWrapper().SendConfig/SendConfigs`，不调 `Acquire`/`Release`/`ReleaseRef`
**Warning signs:** 单测中 refCount 永远不归零；`negative refCount` panic

### Pitfall 3: parseConfigError 顺序颠倒 — device_rejected 误报为 transport_error
**What goes wrong:** 若先扫 transportMarkers 再扫 rejectionMarkers，"% Error: timeout"（device_rejected 含 "timeout" 字样）会被错判为 transport_error → BATCH-02 fail-fast 误停
**Why it happens:** 优先级意识缺失，transport 标记词（"timeout"/"EOF"）与设备错误行字符串有交集
**How to avoid:** D-16 锁定 5 步顺序：nil → Err set → 空 Result → transport markers → rejection markers → 默认成功
**Warning signs:** 单测覆盖 10+ marker，强制每个 marker 输入验证其 kind；尤其 `% Error: timeout` 边界用例

### Pitfall 4: 批量 maxBatchSize 校验放错位置 → SSH 已连接后才发现
**What goes wrong:** 先 `ExecuteCustom` 获取连接再校验 `len(req.PortIDs) > 50` → 浪费一次 SSH handshake + 占用 pool slot
**Why it happens:** 校验顺序倒置
**How to avoid:** D-17 锁定入口三连校验：empty → exceeds → deviceID，**全部在 ExecuteCustom 之前**返回
**Warning signs:** 监控显示 Enqueue 后 pool 短暂满；用户传 100 端口时 SSH 流量从 100 个端口全发后才发现超限

### Pitfall 5: NoOp 路径未返回 PortResult，handler 误判 4xx
**What goes wrong:** pre-state 已匹配时 service 返回 `(nil, nil)` → handler 走 `response.Error(c, 500, "no result")` → 用户看到 500 但实际设备无问题
**Why it happens:** D-14 明确"返回 NoOp PortResult"易被忽视
**How to avoid:** `checkPreState` 命中时始终返回 `&PortResult{..., NoOp: true, Status: "skipped"}`，不返回 nil
**Warning signs:** 单测覆盖每 action × pre-state-matched 组合；handler UAT（Phase 52）显示已 up 端口 undo_shutdown 报 500

### Pitfall 6: Enqueue 在 ExecuteCustom 失败后仍触发
**What goes wrong:** `executeWrite` 返回 error 但 `collectionSvc.Enqueue(deviceID)` 仍被调用 → 触发一轮无意义的采集（设备侧命令全失败 + 本地 DB 无变化）
**Why it happens:** Enqueue 与 error 检查顺序倒置
**How to avoid:** `if executeErr != nil { return ..., executeErr }` 在 Enqueue 之前
**Warning signs:** 系统日志显示 1 个设备连续触发多次 Enqueue 但无任何写命令成功；采集任务表堆积

### Pitfall 7: Vendor 不支持时返回模糊错误
**What goes wrong:** `RenderCommand` 返回 `ErrUnsupportedVendor: cisco` 但 service 包装后丢失 vendor 信息 → handler 返回 "设备不支持" 不告诉前端是哪个 vendor
**Why it happens:** service 用 `fmt.Errorf("...: %w", err)` 但未 echo 关键字段
**How to avoid:** 错误链路 `errors.Is(err, portcollection.ErrUnsupportedVendor)` 显式传递；handler 翻译 422 + 设备 vendor 名
**Warning signs:** Phase 54 真机 UAT 时 Maipu/Cisco 设备写命令报 "internal error" 而非 "vendor not supported"

## Code Examples

Verified patterns from official sources / project code:

### Common Operation 1: `ScrapliWrapper.SendConfigs` 返回值
```go
// Source: internal/device/scrapli_wrapper.go:594
func (w *ScrapliWrapper) SendConfigs(configs []string) ([]*Response, error) {
    // ...
    for _, cfg := range configs {
        r, err := w.driver.SendConfig(cfg)
        if err != nil {
            return responses, fmt.Errorf("发送配置 '%s' 失败: %w", cfg, err)
        }
        responses = append(responses, &Response{
            Result:   r.Result,
            Started:  r.StartTime,
            Finished: r.EndTime,
            Failed:   r.Failed != nil,
        }, ...)
    }
    return responses, nil
}
```
**注意：** `SendConfigs` 中途失败时**返回累计 responses**（不是 nil），调用方应看 `len(responses) < len(configs)` 判定是否部分成功。但对 Phase 51：5 个 action 渲染的命令都**强相关**（description 是 "interface + description" 两条，dot1x ruijie 是 "interface + dot1x" 两条），任一失败即整体失败，所以单端口 path 直接 `lastResp = responses[len(responses)-1]`（或失败时 err 即含失败位置）。

### Common Operation 2: `DeviceExecutor.ExecuteCustom` 签名
```go
// Source: internal/device/executor.go:152
func (e *DeviceExecutor) ExecuteCustom(
    ctx context.Context,
    deviceID string,
    executeFunc func(context.Context, *PooledConnection) error,
    timeout time.Duration,
) error {
    // Submit task to scheduler, wait for callback or timeout
    // fn 内部获取的 *PooledConnection 由 scheduler 管理, fn 跳出 = scheduler 释放
}
```
**调用契约：**
- `deviceID` = `models.NetworkDevice.ID`（UUID string）
- `fn` 内 `pc.GetWrapper()` 拿 `*ScrapliWrapper`，**fn 跳出后 pc 不可再用**
- `timeout` 0 表示不超时（默认行为）
- `timeout + deviceExecutorTimeoutBuffer`（1 min）作为 scheduler 等待上限

### Common Operation 3: `DeviceInfoCollectionService.Enqueue` dedup
```go
// Source: internal/services/device_info_collection_service.go:131
func (s *DeviceInfoCollectionService) Enqueue(deviceID string) error {
    // 检查 pending/running, 有就 return nil (dedup)
    // 否则创建 DeviceEnrichmentTask + 入队
}
```
**Phase 51 触发点：** 单端口/批量成功路径末尾；失败路径**不**触发（D-15 + Pitfall 6 防范）。Enqueue 是 fire-and-forget，service 不等其完成。

### Common Operation 4: `DeviceCredentialHelper.GetDeviceCredential`
```go
// Source: internal/services/device_credential_helper.go:24
func (h *DeviceCredentialHelper) GetDeviceCredential(ctx context.Context, device *models.NetworkDevice) (*models.AuthCredential, error)
```
**Phase 51 调用点：** 实际**不直接调用** — `ExecuteCustom` 内部 scheduler 已通过 `pool.GetConnection` 取凭证并解密；service 层无需重复。如果 Phase 52 handler 决定在调用 service 前显式校验凭证存在性，那是 handler 决策，不影响 Phase 51 service 内部。**结论：service 内无需单独调 `GetDeviceCredential`。**

### Common Operation 5: `models.DevicePortStatus` 字段
```go
// Source: internal/models/device_port_status.go:31
type DevicePortStatus struct {
    ID            string `gorm:"type:uuid;primary_key"`
    DeviceID      string `gorm:"type:uuid;not null;uniqueIndex:uniq_device_interface,priority:1"`
    InterfaceName string `gorm:"size:100;not null;uniqueIndex:uniq_device_interface,priority:2"`
    AdminStatus   string `gorm:"size:20"` // "up" | "down" | "testing" | ...
    OperStatus    string `gorm:"size:20"`
    Description   string `gorm:"size:500"`
    Dot1xEnabled  bool   `gorm:"default:false"`
    // ...
}
```
**Phase 51 读取字段：** `AdminStatus` + `Dot1xEnabled` + `Description`（pre-state 三种 action 各一字段）。**不读** `OperStatus` / `VLAN` / `Speed` 等。

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 散在 cron 调 SSH 写命令 | Service 层抽象 + vendor 模板 + handler 翻译 HTTP | Phase 50 + 51 | 3 厂商统一接口；前/后端可独立测试 |
| 每个写命令自己写 vendor switch | `portcollection.RenderCommand` 派发 | Phase 50 | 模板集中在 1 个 map，单测覆盖语法漂移 |
| 批量 = `for { single() }` 在 handler 层 | 独立 `BatchWritePorts` 走 detached 30min context | Phase 51 | Core.Close 30s 不再误切；fail-fast 业务可测 |
| 写后人工触发 / 等待下次 cron | 成功路径立即 `Enqueue` 触发后台采集 | Phase 51 (AUDIT-04) | 1-2s 内 audit 真相源更新 |
| operlog 模糊记录 | Phase 52 起 CONV-01..04 锁定 OperType 映射（status/update/batch） | Phase 52 | 审计可按 verb 聚合；本 phase service 不直接调 operlog |

**Deprecated/outdated:**
- **DeviceConnectionPool.GetConnection 直接调** + 手动 `defer ReleaseRef()`：F-14 之后（2026-07-06 修复）已被 `DeviceExecutor.ExecuteCustom` 取代，service 不应再直接调 pool（参见 `connection_pool.go:97-108` "F-14 fix" 注释）
- **vendor 命令 switch 在 service**（如 `if vendor == "huawei" { commands = []string{"shutdown"} }`）：Phase 50 已锁走 `RenderCommand` 派发

## Assumptions Log

> All decisions in CONTEXT.md are locked (D-10..D-18) and verified against project code. No `[ASSUMED]` claims remain for the planner to validate — every architectural choice traces to either a pre-existing infrastructure file or a CONTEXT.md decision.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| (none) | All decisions verified | — | — |

**If this table is empty:** All architectural choices in this research were verified against the project codebase (CONTEXT.md canonical refs, `internal/device/executor.go`, `internal/services/portcollection/vendor_port_template.go`, `internal/models/device_port_status.go`, `internal/services/device_info_collection_service.go`, `internal/services/device_credential_helper.go`, `internal/core/core.go`).

## Open Questions

1. **mockDeviceExecutor 与 service.DeviceExecutor 接口的契合度**
   - What we know: 当前 `portWriteServiceImpl.deviceExecutor` 字段是 `*device.DeviceExecutor`（concrete pointer）
   - What's unclear: testify/mock 不能 mock concrete type 的方法 — service 字段类型需改为 interface 才可单测
   - Recommendation: **引入一个 `portWriteExecutor` interface 在 service 包内**，service 字段类型从 `*device.DeviceExecutor` 改为 `portWriteExecutor`；`*device.DeviceExecutor` 天然满足该 interface（结构方法嵌入）；`mockDeviceExecutor` 在 _test.go 显式实现该 interface
   - 实施时确认: 验证 `*device.DeviceExecutor` 暴露 `ExecuteCustom(ctx, deviceID, fn, timeout) error` 方法（已确认 executor.go:152）

2. **mockCollectionSvc 与 `*services.DeviceInfoCollectionService` 的关系**
   - What we know: `*services.DeviceInfoCollectionService.Enqueue(deviceID string) error` 是 public 方法
   - What's unclear: 同上 — concrete pointer 不能 mock
   - Recommendation: 同上 — service 字段类型改为 `portWriteCollectionSvc` interface（只暴露 `Enqueue(string) error`）

3. **GORM DB 在单测中如何处理？**
   - What we know: `*gorm.DB` 是 struct，`WithContext` + `Where` + `First`/`Find` 链式调用
   - What's unclear: 单测中是否需要 mock DB？或 sqlmock 库？
   - Recommendation: **D-18 单测矩阵不需要 mock DB**（因为测试焦点在 SSH 错误解析 + fail-fast 行为，不在 SQL 查询逻辑）。GORM 查询走真实 sqlite in-memory 或完全绕过（pre-state 已匹配 / 已不匹配场景用 `nil` portID + 直接调 executeWrite，DB 返回 ErrRecordNotFound → fallback 直接下发 — 这路径刚好覆盖 D-13 "端口消失" 行为）
   - 备选: 若需要严格 DB 单测，引入 `go-sqlmock` 或 `DATA-DOG/go-sqlmock` — 但这超出 D-18 范围，倾向**不引**

4. **service 字段类型从 concrete pointer 改 interface 后，Phase 52 router 注入是否要变？**
   - What we know: `NewPortWriteService(db, deviceExecutor, collectionSvc)` 工厂函数接受 `*device.DeviceExecutor`
   - What's unclear: 改 interface 后签名是否变
   - Recommendation: **保持** `NewPortWriteService` 接收 `*device.DeviceExecutor`（concrete），在工厂内部赋值给 `portWriteExecutor` interface 字段。这样 router 无需变（仍传 `*device.DeviceExecutor`）

## Environment Availability

> Phase 51 needs Go toolchain + the existing project source tree. No new external runtime dependencies.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.24 toolchain | Build + test | Yes (go.mod) | 1.24 | — |
| testify (mock + assert) | Unit tests | Yes (vendored) | vendored | — |
| gorm v1.30.5 | Pre-state DB query | Yes (vendored) | 1.30.5 | — |
| internal/device, internal/services, internal/services/portcollection | Service impl | Yes (project source) | — | — |
| PostgreSQL / SQLite (live DB) | NOT required for Phase 51 unit tests (pure mock) | Yes (infra) | — | — |
| SSH device pool | NOT required for Phase 51 unit tests (mock executor) | n/a | — | — |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** None.

**Step 2.6 conclusion:** No external environment work needed. The phase is pure Go service code + mock-based unit tests; all dependencies live in the repo or are vendored.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `testing` (stdlib) + `github.com/stretchr/testify/assert` + `github.com/stretchr/testify/mock` |
| Config file | None — Go test convention; `go test ./...` auto-discovers |
| Quick run command | `go test ./internal/services/portwrite/... -v -count=1` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SSH-02 | parseConfigError 区分 transport vs device_rejected | unit (table-driven) | `go test ./internal/services/portwrite/... -run TestParseConfigError -v` | ❌ Wave 0 |
| SSH-03 | per-port Acquire/ReleaseRef 借 ExecuteCustom 复用 | unit (mock executor) | `go test ./internal/services/portwrite/... -run TestShutdown -v` | ❌ Wave 0 |
| SSH-04 | detached 30min context 不被 HTTP ctx 切断 | unit (mock) | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_DetachedContext` | ❌ Wave 0 |
| SSH-06 | 凭据解析（间接通过 ExecuteCustom，service 不直调） | n/a | (不写单独单测 — 由 device 层覆盖) | n/a |
| PORT-06 | pre-state NoOp 路径 | unit | `go test ./internal/services/portwrite/... -run TestPreState -v` | ❌ Wave 0 |
| BATCH-02 | fail-fast 立即停 | unit | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_FailFast -v` | ❌ Wave 0 |
| BATCH-03 | partial result 三数组 | unit | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_PartialResult -v` | ❌ Wave 0 |
| BATCH-04 | maxBatchSize=50 cap | unit | `go test ./internal/services/portwrite/... -run TestBatchWritePorts_ExceedsMax -v` | ❌ Wave 0 |
| AUDIT-04 | success 路径触发 Enqueue | unit (mock collectionSvc) | `go test ./internal/services/portwrite/... -run TestShutdown_TriggersEnqueue -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/services/portwrite/... -count=1`
- **Per wave merge:** `go test ./...` + `go vet ./...`
- **Phase gate:** Full suite green + 100% of TestParseConfigError + TestShutdown/TestUndoShutdown/.../TestBatchWritePorts_* subtests passing

### Wave 0 Gaps
- [ ] `internal/services/portwrite/port_write_service.go` — 6-method service interface + impl
- [ ] `internal/services/portwrite/batch_orchestrator.go` — BatchWriteRequest + BatchResult + 批量循环
- [ ] `internal/services/portwrite/parse_error.go` — parseConfigError + WriteError + markers
- [ ] `internal/services/portwrite/pre_state_check.go` — checkPreState 单端口 + 批量统一函数
- [ ] `internal/services/portwrite/port_write_service_test.go` — 单测（含 10+ parseConfigError 用例）
- [ ] Service 字段类型调整: `*device.DeviceExecutor` → `portWriteExecutor` interface（仅暴露 ExecuteCustom）
- [ ] Service 字段类型调整: `*services.DeviceInfoCollectionService` → `portWriteCollectionSvc` interface（仅暴露 Enqueue）

## Security Domain

> Phase 51 is service-layer only. No HTTP exposure yet (Phase 52 handles that). Security at this layer is about:
> 1. Avoiding accidental credential leak in error messages
> 2. Ensuring parseConfigError doesn't expose internal SSH details to non-error paths
> 3. Following D-15 principle: no operlog/audit table write (handler owns that)

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No (service doesn't auth) | n/a — Phase 52 handler uses `RequirePermissions(["network:port:write"])` middleware |
| V3 Session Management | No | n/a |
| V4 Access Control | No (Phase 52 scope) | n/a |
| V5 Input Validation | **Yes** | InterfaceName + Description 由 `RenderCommand` 校验（Phase 50 sentinel errors）；service 层只透传 |
| V6 Cryptography | No (service 不加解密) | n/a — credentials by DeviceCredentialHelper 内部 SM4 解密；description 透传 scrapli |
| V7 Error Handling | **Yes** | `WriteError.Error()` 不含密码/凭据；parseConfigError 提取的 Message 是设备原文（`% Error: ...`）非 SSH 内部 |
| V9 Logging | Yes (partial) | service 不写 operlog/audit（D-15）— 避免泄露到错误日志 |
| V12 File & Resource | Yes (partial) | detached context 30min 兜底；defer cancel 释放 timer |

### Known Threat Patterns for this Stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|--------------------|
| Description 注入 SSH 命令 (T-50-01) | Tampering | scrapli 文本模式透传；设备 CLI parser 兜底；Phase 50 D-06 deferred — Phase 51 不在 service 加 escape（保持 RenderCommand 输出原貌） |
| ErrInternal 泄露 SSH 内部栈 (T-51-01) | Information Disclosure | `WriteError.Error()` 仅含 `Kind` + 设备原文 + `Cause`（std 错误）；不含 stack trace；handler 翻译时再脱敏 |
| Batch fail-fast DOS via 100 端口 (T-51-02) | Denial of Service | D-17 入口 maxBatchSize=50 cap 硬上限；前端 UI-05 软提示 20 |
| 端口消失误报 (T-51-03) | Repudiation | D-13 ErrRecordNotFound fallback 直接下发；无静默吞错 |
| Enqueue 风暴 (T-51-04) | Denial of Service | 失败路径不调 Enqueue（Pitfall 6）；Enqueue 内部 dedup（pending/running 跳过） |
| operlog 重复记录 (T-51-05) | Repudiation | D-15 service 不调 operlog；Phase 52 handler 单点记录；不双写 |

## Sources

### Primary (HIGH confidence)
- `internal/device/executor.go:152-190` — `DeviceExecutor.ExecuteCustom` 签名 + scheduler 行为
- `internal/device/scrapli_wrapper.go:567-616` — `SendConfig`/`SendConfigs` 返回 `*Response{Result, Err}` + 错误捕获
- `internal/device/connection_pool.go:97-108` — F-14 修复 + `ReleaseRef` 警告（不调 Release）
- `internal/services/portcollection/vendor_port_template.go:75-91` — `RenderCommand(vendor, action, params) ([]string, error)` 公共 API
- `internal/services/portcollection/vendor_port_template_test.go` — 17+ 单测表驱动风格参考
- `internal/services/device_credential_helper.go:24-40` — `GetDeviceCredential(ctx, device)` 凭据解析契约
- `internal/services/device_info_collection_service.go:131-163` — `Enqueue(deviceID)` fire-and-forget 触发契约 + dedup
- `internal/services/operations/cache_invalidator_test.go:14-66` — testify/mock 嵌入式 mock 模式参考
- `internal/services/portcollection/collection.go:88-95` — `CollectionService` 标准 service interface + DI 构造模式
- `internal/models/device_port_status.go:31-57` — `AdminStatus` + `Dot1xEnabled` + `Description` 字段定义
- `internal/models/network_device.go:14-22` — `DeviceVendor` 枚举 (Huawei/H3C/Ruijie/Maipu)
- `internal/core/core.go:283-289` — `DeviceInfoCollectionService` + `DeviceExecutor` 在 Core 中初始化
- `internal/core/core_services.go:18-46` — `CoreServices` struct 暴露 device services 字段
- `internal/utils/operlog/operlog.go:51-67` — `OperTypeStatus=10` / `OperTypeUpdate=2` / `OperTypeBatch=16` 常量
- `.planning/phases/51-w2-portwriteservice-batch-orchestrator-mock-tests/51-CONTEXT.md` — D-10..D-18 全部锁定决策
- `.planning/phases/50-w1-vendor-templates-unit-tests-vendor-action-command-map/50-01-SUMMARY.md` — Phase 50 交付契约

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` — SSH-02/03/04/06 + PORT-06 + BATCH-02/03/04 + AUDIT-04 需求定义
- `.planning/STATE.md` — Critical Pitfalls #5 (Core.Close 30s 切断) + #6 (Connection pool exhaustion) + #7 (No batch size cap)
- `.planning/ROADMAP.md` Phase 51 — 8 条 Success Criteria (本 phase 满足其 #1..#4)

### Tertiary (LOW confidence)
- (无 — 所有关键决策均已通过源码 + 上下文化决策锁验证)

## Metadata

**Confidence breakdown:**
- **Standard stack:** HIGH — 所有依赖已在 `go.mod` + `internal/` 中验证存在
- **Architecture:** HIGH — 5 锁定决策 (D-10..D-14) 对应 5 个已存在的源码锚点（`ExecuteCustom` / `Enqueue` / `RenderCommand` / `models.DevicePortStatus` / `Detached context`）
- **Pitfalls:** HIGH — 7 个 pitfall 全部对应 CONTEXT.md 锁定决策或已知 F-14 / Pitfall #5 类陷阱
- **Patterns:** HIGH — 5 个 pattern 全部基于项目现成代码（service interface / `parseConfigError` / `ExecuteCustom` / `batch_orchestrator` / `testify/mock`）
- **Mock testing:** MEDIUM-HIGH — 唯一待验证点是 service 字段类型改为 interface 后 `*device.DeviceExecutor` 是否仍满足（method set 验证 = HIGH，但单测写起来是否自然 = MEDIUM）

**Research date:** 2026-07-06
**Valid until:** 2026-08-06 (30 days — Phase 51 不引新外部依赖，源码锚点稳定)
