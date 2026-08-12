# Phase 51: W2 — PortWriteService + Batch Orchestrator + Mock Tests - Context

**Gathered:** 2026-07-06
**Status:** Ready for planning
**Source:** v1.19 init decisions (see `.planning/PROJECT.md` Current Milestone 段) + `.planning/REQUIREMENTS.md` SSH-02/03/04/06 + PORT-06 + BATCH-02/03/04 + AUDIT-04 段 + `.planning/ROADMAP.md` Phase 51 段 + Phase 50 CONTEXT 决策

<domain>
## Phase Boundary

锁定 service 层（**PortWriteService + Batch Orchestrator**，单文件双方法 + 工厂函数）的 SSH 连接生命周期、错误解析、批量失败语义，确保 100% mockable / 单测可覆盖，零真实 SSH 流量。

**In scope**:
- `internal/services/portwrite/port_write_service.go` 新建：service 接口 `PortWriteService` + 私有 `portWriteServiceImpl` + DI (`db`, `deviceExecutor`, `collectionSvc`)
- 5 个单端口方法：`Shutdown(ctx, portID, operator)` / `UndoShutdown(ctx, portID, operator)` / `SetDescription(ctx, portID, desc, operator)` / `EnableDot1x(ctx, portID, operator)` / `DisableDot1x(ctx, portID, operator)`
- 1 个批量方法：`BatchWritePorts(ctx, batchReq, operator) (*BatchResult, error)`
- `internal/services/portwrite/batch_orchestrator.go` 同包：批量串行 fail-fast 循环 + partial result 聚合逻辑
- `internal/services/portwrite/parse_error.go`（或同文件）：`parseConfigError(*Response) error` 区分 `transport_error` vs `device_rejected`，通过 `% Error:` / `Unrecognized command` / `Illegal` marker 识别
- `internal/services/portwrite/pre_state_check.go`（或同方法）：`CheckPortPreState(ctx, portID, action)` 从 `sys_device_port_status` 读 admin_status / dot1x_enabled
- `internal/services/portwrite/cache_keys.go`：若需要共享 cache_keys（与 portcollection 同名空间是否独立见下 D-14）
- `*_test.go` 配套：单测 + 表驱动 mock + parseConfigError marker 内联表驱动

**Out of scope**:
- HTTP handler / router / operlog.Record / NetworkPortWrite permission constant / migration — Phase 52
- 前端 BulkWriteDrawer / modal / API wrappers — Phase 53
- 真机验证 / e2e — Phase 54（mock SSH 即可 end-to-end 覆盖）
- 模板抽象为数据库表 — v1.19 init 锁定为"硬编码 Go map 落地为先"，后续 phase 抽象
- Maipu / Cisco / 速率 / VLAN 等其它操作 — Out of Scope (v1.19 REQUIREMENTS.md 表)
- 数据库 `sys_port_write_audit` 表 schema — Phase 52-02 migration 落地，**但 schema 已在本 phase 接口签名中预位**（`status` 字段需含 `succeeded / failed / skipped` 枚举 — 见 D-08）

</domain>

<decisions>
## Implementation Decisions

### D-10: SSH 连接生命周期 = DeviceExecutor.ExecuteCustom 包装
- 选用 `DeviceExecutor.ExecuteCustom(ctx, deviceID, fn, timeout)` 是 project 现成模式：批量 50 端口 = 1 次 connection lifecycle + 内部循环 `SendConfig`/`SendConfigs` per port
- 同 device 池缓存：池按 deviceID 复用，所以"per-port Acquire/ReleaseRef 字面"和"per-batch 共用连接"在正确性上等价；但 ExecuteCustom 多一层 scheduler 提供的 timeout / 重试 / panic recover 保护
- `fn(ctx, *PooledConnection) error` 内：调用 `pc.GetWrapper().SendConfig(cmd)` 或 `SendConfigs(cmds)`，**端口循环之间不调 ReleaseRef**（fn 跳出 = scheduler 自动释放）
- 单端口接口（5 个）也走 ExecuteCustom — fn 内只发 1 条命令便退出，per-port 粒度的 timeout 可在 ExecuteCustom 的 `timeout` 参数独立控制

### D-11: per-port SSH 超时
- ExecuteCustom 的 `timeout` 参数：单端口默认 30s（与 DeviceExecutor.DefaultExecutionConfig.Timeout 对齐），批量接口默认 60s（50 端口 × ~1s + SSH 延迟余量）
- timeout 越大越要小心：Core.Close() 30s deadline 已被 detached context 兜住（D-12），不会撞默认 30s 截止
- 若端口真实命令延迟超过 60s（罕见 Huawei S5735），需视现场调整 — 提为后续 Phase 优化项

### D-12: 批量 detached context = 30min background context
- 批量接口进入 service 后第一行立即 `ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)`，**完全脱离 HTTP request context**
- 同步 GORM queries 也用 detached ctx（`s.db.WithContext(ctx)`）
- `defer cancel()` 排空 goroutine（即便 30min 超时也需 cancel 释放 timer）
- 避免 PROJECT.md "Critical Pitfalls #5: Batch execution exceeds 30s Core.Close deadline" — 这个是 S5700 系列固件 bulk write 实测可能 ~5min/批次的关键前提

### D-13: PORT-06 pre-state 数据源 = DB 读 sys_device_port_status
- 单端口接口（5 个）：`s.db.WithContext(ctx).Where("id = ?", portID).First(&port)` 取 `admin_status` + `dot1x_enabled`
- 批量接口（1 个）：`s.db.WithContext(ctx).Where("device_id = ? AND id IN ?", deviceID, portIDs).Find(&ports)` 批量取，1 次 DB round-trip 拿全部
- 已匹配逻辑：
  - `Shutdown` 当 `admin_status == "down"` → skipped
  - `UndoShutdown` 当 `admin_status == "up"` → skipped
  - `EnableDot1x` 当 `dot1x_enabled == true` → skipped
  - `DisableDot1x` 当 `dot1x_enabled == false` → skipped
  - `SetDescription` 文本比对：DB.Description == req.Description → skipped（避免无变更触发设备侧回写 diff 噪声）
- DB 行不存在（端口"消失"——尚未被 portcollection cron 采集或已 delete）：fallback 跳过 pre-state 检查直接下发，避免误报

### D-14: `skipped` 数组本 phase 填充
- 批量接口 `BatchResult.Skipped []PortResult` 填 PORT-06 pre-state 已匹配的端口
- 单端口接口 NoOp 返回 `(&PortResult{PortID: ..., Action: ..., NoOp: true, CurrentState: ...}, nil)` — handler 据此决定走 200 + operlog "无需操作" 还是 4xx 错误（建议 200 — 与 batch 语义一致）
- BATCH-03 partial result 形状 = `{Succeeded: []PortResult, Failed: []PortResult, Skipped: []PortResult}` 三数组共存，**单字段都有**

### D-15: skipped 路径完整审计覆盖
- 单一 skipped 都进 `sys_port_write_audit`（即使 Phase 52 才建表）：表 `status` 枚举字段需含 `succeeded / failed / skipped` 三值（D-08 锁定 Phase 52 migration schema 预位）
- operlog 全记：用 `operlog.OperTypeStatus`(=10) 一致语义（与"动作未发生"业务含义最贴近）；不引入新 OperType
- single port 路径：写 1 条 operlog + 1 条 audit；batch 路径：每个端口（N 个）写 1 条 operlog + 1 条 audit（n×2 总计）

### D-16: parseConfigError marker 优先级 = 顺序扫描
- 函数形态：`func parseConfigError(resp *device.Response) error`
- 扫描规则（顺序匹配，命中即返回）：
  1. 空响应 / `resp.Result == ""` + `resp.Err != nil` → `&WriteError{Kind: TransportError, Cause: resp.Err}`
  2. 包含 `% Error:` / `% Input error` / `Error: ` 前缀 → `device_rejected` + 提取错误行原文
  3. 包含 `Unrecognized command` / `Unknown command` / `Illegal` / `Invalid` / `Wrong parameter` → `device_rejected`
  4. 其他非空响应（含 OK / `Info:` / 空 Result） → `nil`（成功）
  5. 包含 `connection refused` / `timeout` / `EOF` / `i/o timeout` → `transport_error`
- 实现：私有 `const` 字符串切片或一个 `var rejectionMarkers = []string{...}` 表，单测表驱动覆盖（见 D-18）

### D-17: 批量 fail-fast 语义
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

### D-18: 单测深度 = mock + inline 表驱动 marker 测试
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

### Folded Todos
None — 两个 reviewed todos 不属于本 phase 范围。

</decisions>

<canonical_refs>
## Canonical References

**下游 agent (planner / researcher) 必须先读这些。**

### v1.19 锁定决策
- `.planning/PROJECT.md` §"Current Milestone: v1.19 网络设备写命令" — 5 条 v1.19 init 决策（device_id 直连 / 硬编码 vendor→template map / 3 厂商 MVP / Enqueue 复用 v1.18 / OperType 映射 / network:port:write 权限隔离 / sys_port_write_audit 真相源 / 串行 fail-fast / 真机 UAT 推迟）
- `.planning/REQUIREMENTS.md` SSH/PORT/BATCH/AUDIT/UI/PERM/INFRA/CONV 8 类 36 项 + 10 项 FUTURE + 12 项 Out of Scope
- `.planning/ROADMAP.md` Phase 51 段 — 8 条 Success Criteria
- `.planning/STATE.md` §"Critical Pitfalls → Mitigation Map" — 7 项 v1.19 pitfall 与 phase 对应（pitfall 1/5/6/7 均与 Phase 51 相关）
- `.planning/phases/50-w1-vendor-templates-unit-tests-vendor-action-command-map/50-CONTEXT.md` — D-01..D-09 模板契约 + PortAction/PortTemplateParams/sentinel errors

### Phase 50 落地契约（已写完，本 phase 消费）
- `internal/services/portcollection/vendor_port_template.go` — `RenderCommand(vendor, action, params) ([]string, error)` 公共 API（3 厂商 × 5 action = 15 模板）
- `internal/services/portcollection/vendor_port_template_test.go` — 12+ 单测覆盖全模板 + 负向用例

### SSH / 连接池 / 执行器（直接调用的基础设施）
- `internal/device/scrapli_wrapper.go:567` — `SendConfig(string) (*Response, error)`（单命令）
- `internal/device/scrapli_wrapper.go:594` — `SendConfigs([]string) ([]*Response, error)`（多命令）
- `internal/device/scrapli_wrapper.go:666` — `Response` struct（`Result` + `Err` 字段）
- `internal/device/scrapli_wrapper.go:67` — `PlatformName(vendor) string`
- `internal/device/connection_pool.go:159` — `DeviceConnectionPool.GetConnection(ctx, deviceID) (*PooledConnection, error)`（本 phase 不直接调，走 D-10 ExecuteCustom）
- `internal/device/connection_pool.go:97-108` — `PooledConnection.ReleaseRef()` 警告：deprecated `Release()` 会 panic
- `internal/device/executor.go:32-197` — `DeviceExecutor.ExecuteCustom(ctx, deviceID, fn func(context.Context, *PooledConnection) error, timeout time.Duration) error` — **D-10 选用**
- `internal/device/executor.go:101` — `ExecuteMultipleOnDevice(ctx, deviceID, commands []string, stripPrompt, ...)` — 备选（D-10 不选）

### 凭据解析（SSH-06）
- `internal/services/device_credential_helper.go:13-90` — `DeviceCredentialHelper.GetDeviceCredential(ctx, device) (*AuthCredential, error)` + `GetCredentialByID(ctx, credID)`

### 改后采集触发（AUDIT-04）
- `internal/services/device_info_collection_service.go:131-133` — `Enqueue(deviceID string) error` fire-and-forget 入队

### 端口采集参考架构（同包已有）
- `internal/services/portcollection/collection.go:20-95` — `CollectionService` 同模式 service（DI = db + executor）+ `CollectDevice(ctx, deviceID) (*CollectionResult, error)`
- `internal/services/portcollection/template_cache.go` — 同包 sibling 不冲突参考

### 端口状态数据（PORT-06 pre-state 读取源 — D-13）
- `internal/models/device_port_status.go:35` — `AdminStatus string` 字段 + 注释
- `internal/models/device_port_status.go:46` — `Dot1xEnabled bool` 字段
- `internal/models/device_port_status.go:50` — `PortSecurityEnabled bool`（本 phase 不读，留扩展位）

### 核心 DI（注入点）
- `internal/core/core.go` — `Core` struct 含 `DeviceExecutor *device.DeviceExecutor` + `DeviceInfoCollectionService *DeviceInfoCollectionService` + DB
- `internal/services/portwrite/setup.go`（新建）— 类似 `internal/services/operations/excel_service.go` 的 `NewPortWriteService(db, executor, collectionSvc)`

### 操作日志约定（Phase 52 落地，本 phase 接口需兼容）
- `internal/utils/operlog/operlog.go` — `Record(c, svc, db, module, operType)`
- `.planning/PROJECT.md` §"操作日志记录约定" — `OperTypeStatus` (=10) / `OperTypeUpdate` (=2) / `OperTypeBatch` (=16) / `OperTypeOther` (=0) 常量
- **本 phase 服务层不直接调 operlog**，handler 层（Phase 52）负责，本 phase 仅在 result struct 上标注 what-to-record

### 测试规范
- `.planning/codebase/TESTING.md` — testify/assert + `//go:build !skip_db_tests` 模式（本 phase 单测纯 mock，无 DB tag）
- `internal/services/operations/cache_invalidator_test.go:11` — `testify/mock` 使用惯例
- `internal/services/portcollection/vendor_port_template_test.go`（Phase 50 已落地）— 表驱动单测风格

### 网络设备模型
- `internal/models/network_device.go` — `DeviceVendor` 枚举（huawei/h3c/ruijie/maipu）+ `NetworkDevice` 字段
- `internal/models/device_port_status.go:35/46/50` — D-13 读取的字段

### 服务接口模式（参考）
- `internal/services/operations/building_service.go` — 标准 service interface + 私有 impl + 工厂函数（DI 通过构造参数）
- `internal/services/system/user_service.go` — 同模式（CLAUDE.md §Go Code Patterns）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `device.DeviceExecutor.ExecuteCustom` (executor.go:152-190) — Phase 51 主力，D-10 选用；fn 内可调 `pc.GetWrapper().SendConfig/SendConfigs`
- `device.ScrapliWrapper.SendConfig(string)` 与 `SendConfigs([]string)` (scrapli_wrapper.go:567/594) — Phase 51 直接调，返回 `*Response{Result, Err}` 用作 D-16 parseConfigError 输入
- `device.DeviceConnectionPool.GetConnection(ctx, deviceID)` (connection_pool.go:226) — 本 phase 不直接调（D-10 走 Executor），但 pool 内部仍被 scheduler 复用
- `services.DeviceCredentialHelper.GetDeviceCredential(ctx, device)` (device_credential_helper.go:24) — SSH-06 凭据解析，service 层调用前一行
- `services.DeviceInfoCollectionService.Enqueue(deviceID)` (device_info_collection_service.go:133) — AUDIT-04 1-2s fire-and-forget 触发
- `portcollection.RenderCommand` (Phase 50 落地) — 5 action × 3 vendor = 15 模板渲染函数，本 phase 入口必调
- `models.DevicePortStatus.AdminStatus` + `Dot1xEnabled` — D-13 DB 直读零 SSH 开销

### Established Patterns
- **Service interface**：项目惯例 = `type XxxService interface { Method1(...) ... }` + `type xxxServiceImpl struct { ... }` + `func NewXxxService(...) XxxService`（CLAUDE.md §Go Code Patterns + operations/building_service.go）
- **DI 构造**：service 不读 Core，直接接受构造参数；router 层（Phase 52）从 `core.Core` 取依赖注入
- **错误返回**：service 层 `return fmt.Errorf("...: %w", err)` 包装；sentinel error 用 `errors.New`（`pkg/errors`）
- **testify/assert + testify/mock**：标准单测栈（TESTING.md）
- **mock 模式**：`MockCacheProvider` 模式（cache_invalidator_test.go:14-60）—— `mock.Mock` 嵌入 + `func (m *MockX) Method(...) 返回值 { args := m.Called(...); return args.X(0), args.Error(1) }`
- **上下文分离**：批量 / 后台任务用 `context.WithTimeout(context.Background(), ...)` 脱钩 HTTP（mac_history_partition.go / mac_collection_service.go 范例）

### Integration Points
- **新建包**：`github.com/xingran-next/xingran-go-backend/internal/services/portwrite/`
- **下游消费方（Phase 52）**：
  - `internal/api/v1/network/port_write_handler.go`（新建）— 6 个端点 handler，调 `PortWriteService.Xxx`
  - `internal/api/v1/network/port_write_router.go`（新建）— `SetupPortWriteRouter(r *gin.RouterGroup, core *core.Core)` 在 `network_router.go` 内注册
  - Phase 52-02 migration 加 `sys_port_write_audit` 表（D-08 / D-15 schema 预位）
- **同项目协作**：`internal/services/portcollection/` Phase 50 兄弟包 + `internal/services/component_collector/` v1.18 兄弟 — 三者互不依赖但共存于 ops 管理链路

</code_context>

<specifics>
## Specific Ideas

### service 接口骨架（D-10 + D-14 + D-17 综合）

```go
package portwrite

import (
    "context"
    "errors"
    "fmt"

    "github.com/xingran-next/xingran-go-backend/internal/device"
    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
    "gorm.io/gorm"
)

// Sentinel errors
var (
    ErrBatchTooLarge     = errors.New("portwrite: batch exceeds max size of 50")
    ErrEmptyBatch        = errors.New("portwrite: batch is empty")
    ErrMixedDevices      = errors.New("portwrite: batch contains ports from different devices")
    ErrPortNotFound      = errors.New("portwrite: port not found")
    ErrWriteFailed       = errors.New("portwrite: write operation failed")
    ErrTransportError    = errors.New("portwrite: SSH transport error")
)

// Action 端口写操作（与 Phase 50 PortAction re-export 或别名）
type Action = portcollection.PortAction

// PortResult 单端口写结果
type PortResult struct {
    PortID       string `json:"portId"`
    Action       Action  `json:"action"`
    Status       string `json:"status"`       // "succeeded" | "failed" | "skipped"
    NoOp         bool   `json:"noOp"`
    CurrentState string `json:"currentState,omitempty"` // skipped 时填当前 admin_status 等
    Error        string `json:"error,omitempty"`
    CommandSent  string `json:"commandSent,omitempty"` // 未脱敏 — audit 用
}

// BatchResult 批量结果
type BatchResult struct {
    Succeeded []PortResult `json:"succeeded"`
    Failed    []PortResult `json:"failed"`
    Skipped   []PortResult `json:"skipped"`
}

// BatchWriteRequest 批量写请求
type BatchWriteRequest struct {
    DeviceID  string   `json:"deviceId"`
    Action    Action   `json:"action"`
    PortIDs   []string `json:"portIds"`
    Description string `json:"description,omitempty"` // 仅 ActionDescription
}

// PortWriteService 端口写 service 接口
type PortWriteService interface {
    Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error)
    EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    BatchWritePorts(ctx context.Context, req BatchWriteRequest, operator string) (*BatchResult, error)
}

// 单端口入参合并：~SetDescription 多一个 desc 参数，5 个方法签名差异小 — 这是 typed methods 风格 vs generic 风格的选择（倾向 typed，对齐 operlog OperType 5 种不同调用）

const maxBatchSize = 50

// 私有 impl
type portWriteServiceImpl struct {
    db            *gorm.DB
    deviceExecutor *device.DeviceExecutor
    collectionSvc *services.DeviceInfoCollectionService // 后台采集触发（AUDIT-04）
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

### parseConfigError 形态（D-16）

```go
type WriteErrorKind int

const (
    WriteErrorNone WriteErrorKind = iota
    WriteErrorTransport
    WriteErrorDeviceRejected
)

type WriteError struct {
    Kind    WriteErrorKind
    Cause   error
    Message string
}

func (e *WriteError) Error() string { ... }
func (e *WriteError) Unwrap() error { return e.Cause }

var transportMarkers = []string{"connection refused", "timeout", "EOF", "i/o timeout"}
var rejectionMarkers = []string{"% Error:", "% Input error", "Error: ", "Unrecognized command", "Unknown command", "Illegal", "Invalid", "Wrong parameter"}

func parseConfigError(resp *device.Response) error {
    if resp == nil { return &WriteError{Kind: WriteErrorTransport, Message: "nil response"} }
    if resp.Err != nil { return &WriteError{Kind: WriteErrorTransport, Cause: resp.Err} }
    if resp.Result == "" { return nil }  // 真空 = 成功
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
    return nil  // OK / Info / 空 = 成功
}
```

### 单测骨架（D-18）

```go
// port_write_service_test.go（同目录）
package portwrite

import (
    "context"
    "errors"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// mockDeviceExecutor —— 嵌入 mock.Mock
type mockDeviceExecutor struct {
    mock.Mock
}
func (m *mockDeviceExecutor) ExecuteCustom(ctx context.Context, deviceID string, fn func(context.Context, *device.PooledConnection) error, timeout time.Duration) error {
    args := m.Called(ctx, deviceID, timeout)
    fnArg := args.Get(2)  // 视场景可调 fn
    if fnArg != nil {
        if err := fn(ctx, nil); err != nil { return err }
    }
    return args.Error(0)
}

// mockCollectionSvc
type mockCollectionSvc struct { mock.Mock }
func (m *mockCollectionSvc) Enqueue(deviceID string) error { ... }

// TestParseConfigError 表驱动
func TestParseConfigError(t *testing.T) {
    tests := []struct {
        name string
        resp *device.Response
        wantKind WriteErrorKind
        wantErr bool
    }{
        {"huawei percent error", &device.Response{Result: "% Error: Unrecognized command found at '^'."}, WriteErrorDeviceRejected, true},
        {"h3c wrong parameter", &device.Response{Result: "% Wrong parameter found at '^'."}, WriteErrorDeviceRejected, true},
        {"ruijie unrecognized", &device.Response{Result: "Unrecognized command"}, WriteErrorDeviceRejected, true},
        {"huawei ok", &device.Response{Result: "Info: configuration succeeded"}, WriteErrorNone, false},
        {"empty ok", &device.Response{Result: ""}, WriteErrorNone, false},
        {"nil response", nil, WriteErrorTransport, true},
        {"err set", &device.Response{Err: errors.New("i/o timeout")}, WriteErrorTransport, true},
        {"transport in text", &device.Response{Result: "connection refused"}, WriteErrorTransport, true},
        // ... 10+ 用例
    }
    for _, tt := range tests { t.Run(tt.name, ...) }
}

// TestBatchWritePorts_FailFast
// TestBatchWritePorts_ExceedsMax
// TestShutdown_NoOpWhenAlreadyDown
// ...
```

</specifics>

<deferred>
## Deferred Ideas

### Reviewed Todos (not folded)
- `operlog-exclude-paths.md` (score 0.4, area=general) — RPA 心跳日志白名单。Phase 51 service 层不直接调 operlog，归属 Phase 52-01 handler 决策
- `v1.17-reconciliation-decisions.md` (score 0.2, area=general) — v1.17 资产对账已闭环于 Phase 46-47，与 Phase 51 完全无关

### Phase 51 内未触达 scope creep（列出避免误归）
- **per-port 端 operator 记录字段**：service 不存 operator；handler（Phase 52）从 gin context 提取注入即可
- **rate-limit 单端口接口防 DoS**：UI 层 disabled 已足够（UI-05）
- **混合设备 batch 自动拆分**：抛 `ErrMixedDevices` 拒绝，让客户端拆
- **operlog.Record 调用时机与脱敏兼容**：本 phase 仅设计 result struct signal；具体 Record 字段序列由 Phase 52 handler 决定
- **real-fixture 文件 (`testdata/write_errors/*.txt`)**：Phase 54 site-visit UAT 后再有真实样本采集
- **设备组策略（按 label/role 选端口）**：超出 batch 范围，留未来 phase
- **写命令 dry-run 模式**：v1.19 Out of Scope (FUTURE-04)
- **跨厂商命令抽象**：vendor→template map 硬编码已锁定（FUTURE 范围）

### Suggested follow-up phase candidates (creator discretion，仅作下一轮 v1.19+ seed 备选)
- 自动回滚（snapshot before + reverse on failure）— FUTURE-09
- 多用户并发写互斥（同一端口同时 1 operator 可写）— FUTURE-10
- 操作历史 UI（点击端口看历史 operlog）— FUTURE-05

---

*Phase: 51-w2-portwriteservice-batch-orchestrator-mock-tests*
*Context gathered: 2026-07-06 via /gsd:discuss-phase 51 (4 gray areas × 2 round-trips, 9 selections)*
</content>
</invoke>