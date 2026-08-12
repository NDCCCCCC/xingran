# Architecture Patterns: v1.19 网络设备写命令 (Network Device Write Operations)

**Domain:** Network device SSH write operations (port shutdown/description/dot1x)
**Researched:** 2026-07-06
**Overall confidence:** HIGH

## Executive Summary

v1.19 把网络设备能力从"只读"补成"读写"闭环。写路径必须复用 v1.18 既有的 `DeviceExecutor` + `PooledConnection` + `ScrapliWrapper` 三层抽象,**禁止新建 SSH 池**(避免池子饥饿 TOCTOU、v1.4.0 Queue 竞态等历史债,见 [shutdown-hang-after-port-close]、[device-info-enrichment-zombie-blockage])。写路径与采集路径共享同一个连接引用,Phase 49-02/F-14 已经把 `Acquire/Release` 配对改成 `ReleaseRef`,v1.19 写操作沿用此约定。

写路径与 v1.18 采集的边界仅差一件事:**必须改后触发 `DeviceInfoCollectionService.Enqueue`** 让设备的 `serial_number / model / software_version / uptime` 4 字段重新采集(Shell 命令为载体,不是 SNMP 探测);触发时机是 handler 完整 commit 之后,**不阻塞响应**。operlog 沿用 Phase 34 约定(末尾、`response.Success` 之前)。

**Key Architecture Decisions:**

| 决策 | 选择 | 原因 |
|------|------|------|
| SSH 复用 vs 新建池 | **复用现有连接池**(`PooledConnection.GetConnection`) | 凭据缓存 + 连接复用已稳定;新建池会双池竞争 device 锁 |
| 配置下发 vs 进入配置模式 | **scrapli `SendConfigs` (system-view)** | SendConfig 内置 on-enter/on-exit hook,比手写 vendor 命令序列更安全 |
| 厂商命令映射 | **vendor→template 硬编码 map + 单元测试覆盖** | "落地为先,后续 phase 抽象数据库" (PROJECT.md v1.19 锁定) |
| 批量执行策略 | **串行失败即停(per-port fail-fast)** | 端口配置是命令级事务,部分成功语义会导致后续端口 rollback 困难 |
| 改后采集触发 | **Enqueue 不等待(async-fire-and-forget)** | 采集是 device-info 维度非 per-port 维度,一次会话只触发一次 |
| operlog 粒度 | **per-batch(OperTypeOther)+ 失败项单行(OperTypeStatus)** | batch 跟踪批次审计,失败定位精确到端口 |
| 权限 | **单一 `network:port:write` (全 3 个 op)** | MVP 简化权限矩阵,避免颗粒过细导致运维角色配置成本 |
| 数据库新增 | **无(operlog 兜底)** | 端口配置改动是高频小动作,不进 sys_port_status(那是 snapshot),审计走 sys_oper_log 已够 |
| 前端架构 | **复用 `ports/index.tsx` + 单 Drawer 操作面板** | 不开新页面;Draw 共享列表的选中行 state |

---

## Recommended Architecture

### Component Diagram (NEW in green, MODIFIED in yellow)

```
                                ┌────────────────────────────────┐
   Client (browser)             │ Frontend (NEW)                 │
   ┌────────────────────────┐   │  src/pages/network/ports/      │
   │ Bulk Action Drawer     │──▶│   index.tsx (MOD: 选行 + 触发)│
   │ Single Action Modal    │   │   components/BulkWriteDrawer   │
   │ Result Toast           │   │     (NEW: 串行进度 + 失败点)   │
   └────────────────────────┘   │  src/lib/api/networkApi.ts     │
                                │   (MOD: +writePortShutdown/    │
                                │    writePortDescription/       │
                                │    writePortDot1x batch)       │
                                └────────────────────────────────┘
                                            │ POST /network/ports/write
                                            ▼
                    ┌──────────────────────────────────────────┐
                    │ API: internal/api/v1/network/            │
                    │   port_write_router.go (NEW)              │
                    │   port_write_handler.go (NEW)             │
                    │   - Shutdown / UndoShutdown /             │
                    │     SetDescription / SetDot1x / Batch     │
                    └──────────────────────────────────────────┘
                                            │ 校验权限 network:port:write
                                            ▼
                    ┌──────────────────────────────────────────┐
                    │ Service: internal/services/              │
                    │   network/port_write/                     │
                    │   (NEW package, 落地为先)                 │
                    │                                          │
                    │   port_write_service.go (interface)      │
                    │   port_write_impl.go    (private impl)    │
                    │   templates.go          (vendor→cmd map)  │
                    │   batch.go              (serial orchestr.)│
                    │   result.go             (per-item result) │
                    └──────────────────────────────────────────┘
                                            │ 复用
                                            ▼
   ┌──────────────────────────────────────────────────────────────────┐
   │ 既有资产 (MUST REUSE, 无需新建)                                  │
   │                                                                  │
   │  device.PooledConnection.GetConnection(ctx, device.ID)  ─────┐  │
   │  device.ScrapliWrapper.SendConfigs([]string)              ─┤ │
   │  device.DeviceExecutor                                   ─┤ │
   │                                                                │
   │  core.DeviceInfoCollectionService.Enqueue(deviceID)  ◀── 写后触发 │
   │  operlog.Record / RecordWithBody                       (audit)   │
   │  pkg/middleware.RequirePermissions ["network:port:write"]        │
   │  core.NetworkDeviceService.GetByID / List (前端供应商识别)     │
   └──────────────────────────────────────────────────────────────────┘
```

### Component Boundaries

| 组件 | 文件路径 | 职责 | 与外部通信 |
|------|----------|------|-----------|
| **API router** | `internal/api/v1/network/port_write_router.go` (NEW) | `POST /network/ports/write/shutdown` 等 6 端点 | middleware.RequirePermissions |
| **API handler** | `internal/api/v1/network/port_write_handler.go` (NEW) | 绑定 req → 调 service → operlog.Record | `service.PortWriteService` + `core.OperLogService` |
| **Service interface** | `internal/services/network/port_write/port_write_service.go` (NEW) | 定义 4 个写方法 + 1 个批量方法 | 返回 DTO + error |
| **Service impl** | `internal/services/network/port_write/port_write_impl.go` (NEW) | 调用 scrapli.SendConfigs + 厂商模板 | `device.DeviceExecutor` |
| **Vendor template** | `internal/services/network/port_write/templates.go` (NEW) | `vendor → operation → []string` map | 仅在 service impl 内调用,无外部依赖 |
| **Batch orchestrator** | `internal/services/network/port_write/batch.go` (NEW) | 串行遍历 + 失败短路 + 收集 per-item 结果 | service impl 内部使用 |
| **Result struct** | `internal/services/network/port_write/result.go` (NEW) | `PortWriteResultItem{DeviceID, InterfaceName, Status, Error}` | 返回 handler |
| **Operlog** | `internal/utils/operlog` (既有) | `Record(c, svc, db, "端口配置", OperTypeStatus)` | — |
| **Collector trigger** | `internal/services/device_info_collection_service.go:Enqueue` (既有) | 写后 enqueue deviceID 重新采集 | device_info_collection worker 异步消费 |

---

## Data Flow

### Per-Port Write (single endpoint)

```
[Frontend Drawer] confirmShrink()
    │ POST /network/ports/write/shutdown { deviceId, interfaceName, action: "shutdown" | "undo" }
    ▼
[Router] port_write_router.go:SetupPortWriteRouter
    │ middleware.RequirePermissions(["network:port:write"], core)
    ▼
[Handler] port_write_handler.go:ShutdownPort
    │ c.ShouldBindJSON(&ShutdownPortRequest)
    │   - { deviceId: uuid, interfaceName: "GE0/0/1", action: "shutdown"|"undo" }
    ▼
[Service] port_write_service.go:ShutdownPort
    │ 1. GetByID(deviceId) → NetworkDevice (verify vendor + status=0)
    │ 2. templates.HuaweiCmd("shutdown", iface) → ["system-view", "interface GE0/0/1", "shutdown", "quit", "save"]
    │ 3. DeviceExecutor.ExecuteCustom(ctx, deviceId, func(taskCtx, conn) error {
    │       return conn.wrapper.SendConfigs(cmds)  // scrapligo auto enters/exits system-view
    │    }, 60s)
    │ 4. defer: Enqueue(deviceId)  // 异步 fire-and-forget
    │ 5. return nil
    ▼
[Handler]
    │ operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "端口配置", OperTypeStatus)
    │   // operParam 自动记录 {deviceId, interfaceName, action}
    │ response.Success(c, gin.H{"writtenAt": time.Now()})
    ▼
[Client] Toast.success("端口 shutdown 成功,采集队列已 enqueue")
```

### Batch Write (serial fail-fast)

```
[Frontend BulkWriteDrawer] confirmAll()
    │ POST /network/ports/write/batch { portIds: [...], operation: "shutdown" }
    ▼
[Router]
    ▼
[Handler] BatchShutdown
    │ req.PortIds → service.BatchShutdown(ctx, portIds)
    ▼
[Service] port_write_service.go:BatchShutdown
    │ ports := DB.GetByIDs(portIds)   // 一次查询所有端口,按 device_id 分组
    │ for each device in group(ports by device_id):
    │    result := s.ShutdownPort(ctx, device.ID, ifaces...)
    │    if result.HasError():
    │       // 失败短路:记下 first failed item,标记后续 skipped
    │       append skipped="post-failure" × remainging ports of this device
    │       // 继续遍历下一个 device(单 device fail 不阻断 cross-device)
    │       continue
    │    else:
    │       append result × ports of this device
    │
    │ return BatchResult{
    │   Items: []PortWriteResultItem{...},  // 每端口一项
    │   SuccessCount, FailedCount, SkippedCount,
    │   FirstFailure: first error or nil,
    │ }
    │
    │ 注: enqueue 各 device 只调用一次(每 device 一个)
    ▼
[Handler]
    │ operlog.Record(c, ..., "端口配置", OperTypeBatch,
    │   opts.WithOperParam(map[string]any{
    │     "operation": req.Operation,
    │     "succeeded": result.SuccessCount,
    │     "failed": result.FailedCount,
    │     "skipped": result.SkippedCount,
    │   }))
    │ response.Success(c, result)
    ▼
[Frontend] Drawer 显示进度面板:
    - [√] GE0/0/1-24  成功
    - [×] GE0/0/25    失败: % Unrecognized command(端口shutdown 命令被设备拒绝)
    - [-] GE0/0/26    skipped (短路跳过)
```

### SSH Session Lifecycle

| 问题 | 决策 | 理由 |
|------|------|------|
| 一次 handler 用 1 个 SSH 会话还是 N 个? | **每 device 一个会话语义,所有端口 inline 在该会话执行** | scrapligo `SendConfigs` 自动进/出 system-view,把多端口命令拼接成单一 commit buffer 可减少 SSH 往返。但 v1.19 MVP 选最简:`BatchShutdown` 同 device 内端口用 **1 个会话语义,但 scrapligo SendConfigs 内部每次调 scrapli 仍开 1 通道**;若失败短路即 return |
| 复用池连接还是新建? | **复用 `pool.GetConnection(deviceID)` + `defer conn.ReleaseRef()`** | F-14 (Phase 49) 修复后 Acquire/Release 路径已废弃(GetConnection 内部已 refCount+1,旧调用方会导致 24/20 池满)。v1.19 必须严格 `defer conn.ReleaseRef()` |
| 连接获取失败 vs 命令失败 | **区分两类错误** | GetConnection 失败 = 设备不可达,直接 abort;SendConfig 失败 = 单端口命令错误,可记录 + 继续下一端口(由 BatchFailFast 决策) |
| 失败是否要 rollback 已成功端口? | **MVP 不实现** (PROJECT.md 锁定决策) | 端口配置错误 rollback 需要 pre/post snapshot,超出 MVP scope;v1.19+ follow-up |
| 网络问题(v1.18 shutdown hang) | **复用 v1.18 Stop timeout 8s 兜底** | 已通过 Phase 49 device_info_collection_service.Stop() 兜底 SSH 阻塞,SSH 卡死不再拖死进程退出 |

---

## Patterns to Follow

### Pattern 1: Handler-Service 既有模式 (Phase 34 + CLAUDE.md)

```go
// internal/services/network/port_write/port_write_service.go (NEW, package-level interface)
type PortWriteService interface {
    // 单端口
    ShutdownPort(ctx context.Context, req *ShutdownPortRequest) (*PortWriteResult, error)
    UndoShutdownPort(ctx context.Context, req *ShutdownPortRequest) (*PortWriteResult, error)
    SetPortDescription(ctx context.Context, req *SetDescriptionRequest) (*PortWriteResult, error)
    SetPortDot1x(ctx context.Context, req *SetDot1xRequest) (*PortWriteResult, error)
    // 批量
    BatchShutdown(ctx context.Context, req *BatchPortWriteRequest) (*BatchPortWriteResult, error)
}

// 请求 DTO (操作用 Go 常量标识,不用 string 字符串避免 typo)
type OperationKind int
const (
    OpShutdown OperationKind = iota + 1
    OpUndoShutdown
    OpSetDescription
    OpSetDot1x
)

type ShutdownPortRequest struct {
    DeviceID      string `json:"deviceId"      binding:"required,uuid"`
    InterfaceName string `json:"interfaceName" binding:"required"`
    Action        string `json:"action"        binding:"required,oneof=shutdown undo"`  // 仅 shutdown/undo
}

type SetDescriptionRequest struct {
    DeviceID      string `json:"deviceId"      binding:"required,uuid"`
    InterfaceName string `json:"interfaceName" binding:"required"`
    Description   string `json:"description"   binding:"required,max=240"`  // 华为典型限制 240 字符
}

type SetDot1xRequest struct {
    DeviceID      string `json:"deviceId"      binding:"required,uuid"`
    InterfaceName string `json:"interfaceName" binding:"required"`
    Enabled       bool   `json:"enabled"`
}

type BatchPortWriteRequest struct {
    PortIDs    []string `json:"portIds"    binding:"required,min=1,max=500"`  // 上限防误操作大批量
    Operation  string   `json:"operation"  binding:"required,oneof=shutdown undo description dot1x_enable dot1x_disable"`
    Description string  `json:"description,omitempty"`  // operation=description 时必填,validation 单独处理
}

// 实现结构
type portWriteServiceImpl struct {
    db             *gorm.DB
    deviceExecutor *device.DeviceExecutor
    collectionSvc  *services.DeviceInfoCollectionService  // 改后 enqueue
}

// 构造签名匹配 NetworkExporter 既有模式:
func NewPortWriteService(
    db *gorm.DB,
    deviceExecutor *device.DeviceExecutor,
    collectionSvc *services.DeviceInfoCollectionService,
) PortWriteService {
    return &portWriteServiceImpl{db: db, deviceExecutor: deviceExecutor, collectionSvc: collectionSvc}
}
```

### Pattern 2: Vendor Command Template Map

```go
// internal/services/network/port_write/templates.go (NEW)
//
// PortWriteTemplates 把 (vendor, operation, iface, extra) → 系统视图命令序列。
// 硬编码 map("落地为先,后续 phase 抽象为数据库" — PROJECT.md v1.19 锁定)。
// 单元测试覆盖所有 (vendor, operation) 组合。vendor=None → return nil → service 报 400。

var PortWriteTemplates = map[models.DeviceVendor]map[OperationKind]func(extra string) []string{
    models.VendorHuawei: {
        OpShutdown: func(iface string) []string {
            return []string{
                "system-view",
                "interface " + iface,
                "shutdown",
                "quit",
                "commit",  // huawei 提交配置(commit 必须才生效)
                "quit",
            }
        },
        OpUndoShutdown: func(iface string) []string {
            return []string{
                "system-view",
                "interface " + iface,
                "undo shutdown",
                "quit",
                "commit",
                "quit",
            }
        },
        OpSetDescription: func(_ string) []string { panic("desc needs iface+text") },
        OpSetDot1x: func(_ string) []string { panic("dot1x needs iface+flag") },
    },
    models.VendorH3C: {
        // H3C 与 Huawei 在 system-view 语法 100% 兼容,但 commit 是可选的
        OpShutdown: func(iface string) []string {
            return []string{"system-view", "interface " + iface, "shutdown", "quit", "quit"}
        },
        // ... 略
    },
    models.VendorRuijie: {
        OpShutdown: func(iface string) []string {
            // 锐捷没有 system-view,直接在 config 模式
            return []string{"configure terminal", "interface " + iface, "shutdown", "exit", "exit", "write"}
        },
        // ... 略
    },
}

// 桥接函数(签名规范化):
func ResolveTemplate(vendor models.DeviceVendor, op OperationKind, iface string, extra string) ([]string, error) {
    vendorMap, ok := PortWriteTemplates[vendor]
    if !ok {
        return nil, fmt.Errorf("vendor %s 不支持端口配置", vendor)
    }
    builder, ok := vendorMap[op]
    if !ok {
        return nil, fmt.Errorf("vendor %s 不支持操作 %d", vendor, op)
    }
    // OpSetDescription/OpSetDot1x 需要文本/enable 标志,由 caller 拼成完整 []string
    switch op {
    case OpSetDescription:
        return resolveSetDescription(vendor, iface, extra), nil
    case OpSetDot1x:
        return resolveSetDot1x(vendor, iface, extra == "true"), nil
    default:
        return builder(iface), nil
    }
}
```

**Testing:** `templates_test.go` 覆盖:
- 3 vendors × 4 operations = 12 用例
- 边界:empty interface → panic 验证
- 关键:verifies each command is bound to a specific interface name (无跨端口污染)

### Pattern 3: SSH 执行 + Scragli SendConfigs

```go
// in port_write_impl.go ShutdownPort()
func (s *portWriteServiceImpl) ShutdownPort(ctx context.Context, req *ShutdownPortRequest) (*PortWriteResult, error) {
    // 1. device 存在性 + 在线校验
    var dev models.NetworkDevice
    if err := s.db.WithContext(ctx).Where("id = ?", req.DeviceID).First(&dev).Error; err != nil {
        return nil, fmt.Errorf("设备不存在: %w", err)
    }
    if dev.Status != models.DeviceStatusOnline {
        return nil, fmt.Errorf("设备离线,无法下发")
    }
    if dev.CredentialID == nil || *dev.CredentialID == "" {
        return nil, fmt.Errorf("设备未配置凭证")
    }

    // 2. resolve 命令
    cmds, err := templates.ResolveTemplate(dev.Vendor, OpShutdown, req.InterfaceName, "")
    if err != nil {
        return nil, err
    }

    // 3. SSH 执行(复用既有 pool + DeviceExecutor)
    execErr := s.deviceExecutor.ExecuteCustom(ctx, req.DeviceID,
        func(taskCtx context.Context, conn *device.PooledConnection) error {
            // F-14: defer ReleaseRef()(不是 Release)
            wrapper := conn.GetWrapper()
            if wrapper == nil {
                return fmt.Errorf("连接未就绪")
            }
            // SendConfigs 自动入/退 system-view,scrapligo 1.3.3 内置 on-enter/on-exit hook
            responses, err := wrapper.SendConfigs(cmds)
            if err != nil {
                return err
            }
            // 防御: 任一 response.Failed == true → 视为部分失败
            for _, r := range responses {
                if r.Failed {
                    return fmt.Errorf("配置命令被设备拒绝: %s", strings.TrimSpace(r.Result))
                }
            }
            return nil
        },
        60*time.Second,  // 端口配置常用超时
    )
    if execErr != nil {
        return nil, execErr
    }

    // 4. 改后 enqueue(异步 fire-and-forget,不阻塞)
    defer func() {
        if enqErr := s.collectionSvc.Enqueue(req.DeviceID); enqErr != nil {
            applogger.Warnf("[端口写] 改后采集 enqueue 失败 [device=%s]: %v",
                req.DeviceID, enqErr)
        }
    }()

    return &PortWriteResult{
        DeviceID:      req.DeviceID,
        InterfaceName: req.InterfaceName,
        Status:        "success",
        WrittenAt:     time.Now(),
    }, nil
}
```

### Pattern 4: 批量串行 + 失败短路

```go
// internal/services/network/port_write/batch.go (NEW)
//
// 设计:
//  - 同 device 内多端口共享一个 SSH 会话(sequential within device)
//  - 跨 device 之间无依赖,串行处理(避免 SSH 风暴)
//  - 任一 device 失败 → 记 first failure → 该 device 剩余端口标 skipped
//    但继续处理下一个 device(单 device fail 不阻断 cross-device)
//  - 最终 result 包含:success / failed / skipped 三类计数
func (s *portWriteServiceImpl) BatchShutdown(ctx context.Context, req *BatchPortWriteRequest) (*BatchPortWriteResult, error) {
    // 1. 一次性查出所有端口,按 device 排序,减少 N+1
    var ports []models.DevicePortStatus
    if err := s.db.WithContext(ctx).Where("id IN ?", req.PortIDs).Find(&ports).Error; err != nil {
        return nil, fmt.Errorf("查询端口失败: %w", err)
    }
    if len(ports) == 0 {
        return &BatchPortWriteResult{}, nil
    }

    // 2. 按 deviceID 分组
    byDevice := make(map[string][]models.DevicePortStatus)
    for _, p := range ports {
        byDevice[p.DeviceID] = append(byDevice[p.DeviceID], p)
    }

    result := &BatchPortWriteResult{Items: make([]PortWriteResultItem, 0, len(ports))}

    for deviceID, devicePorts := range byDevice {
        // 同 device 内单端口逐一调用 ShutdownPort()
        // (SendConfigs 内部已经是同一 session,只增加 N 次 SendConfig 调用)
        deviceFailed := false
        for _, p := range devicePorts {
            if deviceFailed {
                // 短路:跳过
                result.Items = append(result.Items, PortWriteResultItem{
                    DeviceID:      deviceID,
                    InterfaceName: p.InterfaceName,
                    Status:        "skipped",
                    ErrorMessage:  "前面端口配置失败,已跳过本端口",
                })
                result.SkippedCount++
                continue
            }

            writeReq := &ShutdownPortRequest{
                DeviceID:      deviceID,
                InterfaceName: p.InterfaceName,
                Action:        req.Operation,  // 兼容 batch 字段
            }
            // 把 OperationKind 翻译成 Action 字符串
            switch req.Operation {
            case "shutdown":
                writeReq.Action = "shutdown"
            case "undo":
                writeReq.Action = "undo"
            }

            res, err := s.ShutdownPort(ctx, writeReq)
            if err != nil {
                deviceFailed = true
                result.Items = append(result.Items, PortWriteResultItem{
                    DeviceID:      deviceID,
                    InterfaceName: p.InterfaceName,
                    Status:        "failed",
                    ErrorMessage:  err.Error(),
                })
                result.FailedCount++
                if result.FirstFailure == nil {
                    result.FirstFailure = &PortWriteResultItem{
                        DeviceID:      deviceID,
                        InterfaceName: p.InterfaceName,
                        Status:        "failed",
                        ErrorMessage:  err.Error(),
                        FailedAt:      time.Now(),
                    }
                }
                continue
            }

            result.Items = append(result.Items, PortWriteResultItem{
                DeviceID:      deviceID,
                InterfaceName: p.InterfaceName,
                Status:        "success",
                WrittenAt:     res.WrittenAt,
            })
            result.SuccessCount++
        }
    }

    return result, nil
}
```

**注意:** batch 内每次 ShutdownPort 都会调 Enqueue — 同 device 多次 enqueue 由 v1.18 D-09 dedup 守护(查 pending/running,有就 return nil)。所以 batch 不会重复入队。详见 [device-info-enrichment-zombie-blockage] dedup 语义。

### Pattern 5: Operlog 双层记录

```go
// 单端口端点
func (h *PortWriteHandler) ShutdownPort(c *gin.Context) {
    var req ShutdownPortRequest
    if !responseHelpers.HandleJSONBinding(c, &req) {
        return
    }
    res, err := h.svc.ShutdownPort(c.Request.Context(), &req)
    if !responseHelpers.HandleServiceError(c, err, "端口 shutdown 失败") {
        return
    }

    // Phase 34 强制约定 + Phase 49 D-13 后续 family: OperTypeStatus
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
        "端口配置", operlog.OperTypeStatus,
        operlog.WithOperParam(map[string]any{
            "operation": "shutdown",
            "deviceId":  req.DeviceID,
            "iface":     req.InterfaceName,
            "writtenAt": res.WrittenAt,
        }))
    response.Success(c, res)
}

// 批量端点
func (h *PortWriteHandler) BatchShutdown(c *gin.Context) {
    var req BatchPortWriteRequest
    if !responseHelpers.HandleJSONBinding(c, &req) {
        return
    }
    res, err := h.svc.BatchShutdown(c.Request.Context(), &req)
    if !responseHelpers.HandleServiceError(c, err, "批量端口配置失败") {
        return
    }

    operlog.Record(c, h.core.OperLogService, h.core.GetDB(),
        "端口配置", operlog.OperTypeBatch,
        operlog.WithOperParam(map[string]any{
            "operation":     req.Operation,
            "totalPorts":    len(req.PortIDs),
            "succeeded":     res.SuccessCount,
            "failed":        res.FailedCount,
            "skipped":       res.SkippedCount,
            "firstFailureAt": firstFailureOrNil(res.FirstFailure),
        }))
    response.Success(c, res)
}
```

### Pattern 6: 前端 BulkWriteDrawer (NEW component)

```tsx
// xingran-react-frontend/src/pages/network/ports/components/BulkWriteDrawer.tsx (NEW)
//
// - 父组件 ports/index.tsx 透传 selectedRowKeys + 操作类型
// - Drawer 显示选中端口预览 + 操作表单(description 输入框 / dot1x 开关 / shutdown 确认)
// - 提交时显示 progress 面板(每端口状态:pending → running → success/failed/skipped)
// - 失败时把 first-failure 显眼展示(skipped 计数明示)
export const BulkWriteDrawer: FC<{
  visible: boolean;
  onClose: () => void;
  selectedPorts: DevicePortStatus[];
  operation: "shutdown" | "undo" | "description" | "dot1x";
}> = ({ visible, onClose, selectedPorts, operation }) => {
  const [submitting, setSubmitting] = useState(false);
  const [results, setResults] = useState<PortWriteResultItem[]>([]);
  // ... form + submit logic via networkApi.batchShutdown etc.

  return (
    <Drawer visible={visible} onClose={onClose} width={680} title={...}>
      <Alert type="warning" message="批量操作串行执行,任一端口失败将跳过该设备后续端口" />
      <Table columns={resultColumns} dataSource={results} pagination={false}
             rowClassName={(r) => r.status === "failed" ? "row-error" : undefined} />
    </Drawer>
  );
};
```

**State management:** 组件 local state (useState/useMemo),不开新 Zustand store。
**API client:** 在 `src/lib/api/networkApi.ts` 中加 4 个 wrapped functions,不让 `ports/index.tsx` 直接 `post(...)`。

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: 新建 SSH 池 / 跳开 PooledConnection

**What:** 直接 `net.Dial("tcp", ip:22)`,绕开连接池实现自己的 SSH 长连接。
**Why bad:**
- 双池竞争 `device.mu` 加锁 → TOCTOU(Phase 49-02 connection_pool.go:266-273 的债)
- scrapligo Queue 竞态(1.3.3 → 1.4.0 双重检查后缓解但仍存在)
- 凭据缓存不一致:池里有连接复用,但新建连会重新拉凭证 → 加密态开销 + 凭据更新延迟
**Instead:** 只用 `core.DeviceExecutor.ExecuteCustom(ctx, deviceID, fn, timeout)`。DeviceExecutor 内部走 `PooledConnection.GetConnection`。

### Anti-Pattern 2: 在 Handler 层拼命令序列

**What:** handler `commands := []string{"system-view", "interface " + iface, "shutdown"}` 直接 SendConfig。
**Why bad:**
- 厂商差异(huawei commit 必填,ruijie write 必填,h3c 不要 commit)散落在 handler 里,不可单测
- 测试组件要 mock GORM + Scragli,而 vendor 逻辑本可以纯 unit-test 不需要 mock
- 后续 v1.19+ 抽象为数据库 / 命令模板 UI 时,迁移路径被迫重构所有 handler
**Instead:** 把 (vendor, operation) → commands 放到 `port_write/templates.go` 内部纯函数,handler 只做 request→template→exec。

### Anti-Pattern 3: batch 并发执行(parallel / errgroup)

**What:** 用 `errgroup` 并发 shutdown 多个端口,任一失败 cancel ctx。
**Why bad:**
- 端口配置是设备级 session 的子操作,真正的并发单元应是 **device**,不是 port
- 并发会触发多 SSH session 连接同一设备 → scrapligo Queue 冲突 + 设备 lockstep 错乱
- PROJECT.md v1.19 锁定决策"串行失败即停"
**Instead:** per-device 串行(同 device 多端口共享 session);跨 device 也串行(MVP 简单优先)。如未来要并发,只在 device 维度并发,不跨端口。

### Anti-Pattern 4: 写成功 → 等采集完(sync wait Enqueue)

**What:** `collectionSvc.RunNow(deviceID)` 同步等待采集完成才响应客户端。
**Why bad:**
- 采集走 SSH 周期通常 5-30 秒,客户端 HTTP 长连接超时或 Nginx 504 风险
- 加剧 SSH 池压力(同时多端口写操作排队等采集)
- v1.18 collector 已迁 async(Enqueue + worker pool),v1.19 不应破坏这个架构
**Instead:** 仅 fire-and-forget `Enqueue(deviceID)`,前端轮询 device 的 `EnrichmentStatus`,或在采集完成事件(WebSocket 或 polling GET /devices/{id}/enrichment-status)刷新端口详情。

### Anti-Pattern 5: 把每个端口当独立 operlog 行

**What:** 每次 ShutdownPort 内部都 `operlog.Record(...)`。
**Why bad:**
- batch 50 端口 → 50 条 operlog(每条独立 transaction 提交)→ sys_oper_log 噪音爆炸
- 前端审计筛选"看今天谁 shutdown 过端口"变成 50 行同 IP / 同 user / 同 action
**Instead:** batch 端点一条 operlog(OperTypeBatch + summary params);单端口端点一条 operlog(OperTypeStatus + 端口细节)。**绝不双层**。

---

## Build Order

按依赖顺序排,每步可独立 ship(契约清晰):

1. **Phase W1 · Templates + Tests (无 DB 无 HTTP)**
   - 落地 `port_write/templates.go` 12 个 vendor × op 组合
   - 落地 `port_write/templates_test.go` 全量覆盖
   - **零依赖**;可独立 PR;后续 4 步都引用其函数
2. **Phase W2 · Service + 单元测试(mocks)**
   - 落地 `port_write_service.go`(interface + impl)
   - 用 mock 的 deviceExecutor / collectionSvc 验证:
     - ShutdownPort happy path (1 端口 + 1 enqueue)
     - ShutdownPort 设备离线 400
     - BatchShutdown 短路(skipped 计数正确)
     - 模板不支持(vendor="cisco")返 error
   - 仍无路由、无前端
3. **Phase W3 · Router + Handler + Operlog**
   - 落地 `port_write_router.go` + `port_write_handler.go`
   - 接到 `network_router.go` 的 `r.Group("/ports/write")` 新组(独立权限 `network:port:write`)
   - 端点:POST /shutdown /undo /description /dot1x /batch(5 端点,MVP 5 即可)
   - migration: sys_menu seed 新菜单项"端口配置";sys_role_menu 注入 admin 角色(`GrantNewMenuToRolesHavingParent` per [migration-grant-new-menu-precision-helper])
4. **Phase W4 · 前端 Drawer + 网络 API 包装**
   - 在 `xingran-react-frontend/src/lib/api/networkApi.ts` 加 4 个 wrapped call
   - `pages/network/ports/index.tsx` 在 toolbar 加"批量操作"按钮(选行后点亮)
   - 新增 `pages/network/ports/components/BulkWriteDrawer.tsx` + 4 个单端 modal
   - 联调:选中端口 → 选操作 → confirm → 显示进度
5. **Phase W5 · E2E + 文档更新**
   - README / docs/API响应规范.md 加端口配置端点签名
   - 跑真机 site-visit UAT(若有真机 S5700 / S5735 可用)— 注意 v1.18 延期到 v1.19+ 的 S8700 / RS8607E 也可借机并发
   - verify document:Phase 49 / W1-W5 result

**Dependencies map:**

```
W1 (templates)
 └─ W2 (service)
     └─ W3 (router/handler)
         └─ W4 (前端)
             └─ W5 (E2E)
```

W2 → W3 之间有一个无形的 contract:W3 拿到的 service 接口签名必须在 W2 写完。

---

## File Paths — Specific to XingRan-Next Codebase

### NEW files (待创建)

```
internal/api/v1/network/port_write_router.go          (~40 LOC, 路由注册 + 权限)
internal/api/v1/network/port_write_handler.go         (~200 LOC, 5 端点 + operlog)
internal/services/network/port_write/port_write_service.go   (~150 LOC, interface + impl struct)
internal/services/network/port_write/templates.go     (~120 LOC, vendor → commands map + resolve 函数)
internal/services/network/port_write/batch.go         (~80 LOC,  串行短路 + 结果聚合)
internal/services/network/port_write/result.go        (~50 LOC,  PortWriteResult / PortWriteResultItem / BatchPortWriteResult)
internal/services/network/port_write/templates_test.go   (~150 LOC, 12 vendor×op 组合 + 边界)
internal/services/network/port_write/port_write_service_test.go   (~250 LOC, mock executor + 串行/失败路径)
internal/core/db/migrations/migration_202_port_write_menu.go   (~50 LOC, sys_menu seed + 精准 GrantNewMenuToRolesHavingParent)

xingran-react-frontend/src/lib/api/networkApi.ts        (MODIFY: +5 wrapped functions)
xingran-react-frontend/src/pages/network/ports/index.tsx                (MODIFY: toolbar + Drawer state)
xingran-react-frontend/src/pages/network/ports/components/BulkWriteDrawer.tsx   (~200 LOC, NEW)
xingran-react-frontend/src/pages/network/ports/components/ShutdownModal.tsx     (~80 LOC,  NEW)
xingran-react-frontend/src/pages/network/ports/components/DescriptionModal.tsx  (~80 LOC,  NEW)
xingran-react-frontend/src/pages/network/ports/components/Dot1xModal.tsx        (~80 LOC,  NEW)
```

### MODIFIED files

```
internal/api/router.go                              (import + SetupNetworkRouter 内挂新组)
internal/api/v1/network/network_router.go           (MOD: 加 SetupPortWriteRouter 内部嵌套 — 或者新主 router 子组)
internal/core/core.go                               (NO MOD: 已在 Phase 48 把 DeviceInfoCollectionService 注入 Core.Fields,直接用)
internal/utils/operlog/operlog.go                   (NO MOD: OperTypeStatus / Batch 已存在)
pkg/middleware/permission.go                        (NO MOD: RequirePermissions 已支持新 perm 字符串)
docs/API响应规范.md                                  (MOD: 加 5 端点 schema)
```

### NEW menu seeding data

```
sys_menu entry (added via migration_202):
  parent_path:  "网络设备"
  path:         "network/ports/write"
  component:    "network/ports/index" (复用同页)
  perms:        "network:port:write"
  menu_type:    "F" (button / 操作按钮)
```

---

## Integration Points Audit (NEW vs MODIFIED vs REUSED)

| 资产 | 状态 | 用途 |
|------|------|------|
| `device.DeviceExecutor` | **REUSED** | `ExecuteCustom` 走 SSH session,不需新建 |
| `device.PooledConnection.GetConnection` | **REUSED** | F-14 后用 `defer conn.ReleaseRef()` 模式,Phase 49-02 修复后用 |
| `device.ScrapliWrapper.SendConfigs` | **REUSED** | scrapligo 1.3.3 内置 on-enter/on-exit system-view,无需手写 |
| `services.DeviceInfoCollectionService.Enqueue` | **REUSED** | v1.18 D-09 dedup 已加,重复 enqueue 安全 no-op |
| `operlog.Record + WithOperParam` | **REUSED** | Phase 34 + Phase 49 D-13 已确立的两套 API |
| `middleware.RequirePermissions` | **REUSED** | 自定义 perm 字符串,无需新增 helper |
| `gorm.DB` (dev/port 模型) | **REUSED** | 已有 `models.NetworkDevice` + `models.DevicePortStatus` |
| `src/lib/api.ts` 的 `post/get` + TokenManager | **REUSED** | 自动 token refresh + 401 拦截 |
| `src/hooks/useTableManager + usePagination` | **REUSED** | 端口列表 page 已用,加 toolbar 按钮即可 |
| `network.NewCredentialHandler` 等既有模式 | **REUSED 模板** | v1.19 new handler/router 完全照抄其结构 |
| `internal/services/network/cache_impl.go` | **BYPASS** | v1.19 写操作不读 cache(写是动作,cache 才是读) |
| `internal/services/network/network_service.go`(如存在) | **EXTENDED?** | 仅若需要列设备 vendor 给前端做预校验 |

---

## Known Risks & Mitigations

### Risk 1: 端口配置命令在不同固件版本差异

**Description:** 华为 V200R003 vs V600R024C00 的 "shutdown" 命令格式略有差异,某些版本须 `undo shutdown` 而非 `shutdown undo`。
**Mitigation:**
- W1 写单测覆盖 3 厂商 × 3 firmware 版本(主分支 / 用户部署版本)
- 通过 mock SSH output wrapper 验证命令正确性
- 失败时不 panic,统一返 error 由前端展示

### Risk 2: 串行 batch 50 端口 × 5秒/端口 = 4分钟超时

**Description:** nginx 默认 60s timeout,batch 50 端口可能超时客户端。
**Mitigation:**
- 前端把 batch 限制 `max=500` 改为 `max=100` / 1 device per call(单 device 内 N 端口才有意义,跨 device 走多次单端口端点)
- 或者前端 polling(WebSocket 或每 5 秒 GET batch status)— MVP 可不做
- 已知不完美但 MVP 锁定(MILESTONES.md v1.19 "操作回滚 / 自动回滚暂不实现")

### Risk 3: 用户跨 device 误触发 deviceA 的写→deviceB 的读

**Description:** enqueue 触发采集后,采集 worker 异步读 deviceB 的数据,而 deviceB 的采集结果是用户立刻想看的。
**Mitigation:**
- 数据隔离:采集针对的是 deviceID(写操作的 deviceA),不会影响 deviceB
- 若前端需要"看到写后的状态",前端轮询 deviceA 的 `EnrichmentStatus` 直到 success
- 这也是 v1.18 已确立的采集架构,前端已有该 polling 模式可直接复用

### Risk 4: scrapli `SendConfigs` 在某些厂商不退出 system-view

**Description:** scrapligo 1.3.3 内置 enter/exit hook 对 huawei_vrp / hp_comware 都验证过,但 ruijie_rjos 的 `configure terminal` 模式与 huawei system-view 语义不完全等价。
**Mitigation:**
- templates.Optimize a test fixture:`MockScrapliFor(Ruijie)` 验证 SendConfig 的最终 prompt 是用户态
- 如 scrape 不退出 → 后续 batch 的第 2 个端口命令进不去 config mode → 第一个端口失败
- 风险已知,登记为 W2 测试用例必跑项,失败即修命令序列或换 scrapligo 版本

---

## Sources

- `D:/code/ClaudeCode/xingran-go-backend/.planning/PROJECT.md` (v1.19 锁定决策)
- `D:/code/ClaudeCode/xingran-go-backend/CLAUDE.md` (Handler-Service 模式 + operlog 约定)
- `internal/services/device_info_collection_service.go` (Enqueue 异步架构)
- `internal/device/scrapli_wrapper.go` (ScrapliWrapper.SendConfigs 实现)
- `internal/device/connection_pool.go` (F-14 修复:ReleaseRef 配对)
- `internal/services/portcollection/collection.go` (既有采集模式参考)
- `internal/services/component_collector/cmd_dispatcher.go` (vendor→commands map 模式参考)
- `internal/services/component_collector/pipeline.go` (Pipeline 异步触发模式)
- `internal/utils/operlog/operlog.go` (Record / RecordWithBody / RecordBackground)
- `internal/api/v1/network/network_router.go` (既有端口路由 + 权限矩阵)
- `internal/api/v1/network/port_handler.go` (端口既有 endpoint 模式参考)
- `internal/api/v1/network/execution_handler.go` (operlog.OperTypeOther 既有使用)
- `.planning/memory/`:device-info-enrichment-zombie-blockage, shutdown-hang-after-port-close, migration-grant-new-menu-precision-helper

---

## Confidence Assessment

| 区域 | Confidence | Notes |
|------|------------|-------|
| SSH 复用 vs 新建 | HIGH | F-14 + Phase 49-02 修复已认定必须复用 |
| 厂商模板硬编码 map | HIGH | PROJECT.md 锁定决策"落地为先,后续 phase 抽象为数据库" |
| 串行失败即停 | HIGH | PROJECT.md v1.19 MVP 锁定 |
| operlog 双层 (per-port + per-batch) | HIGH | Phase 34 强制约定 + 系统既有模式 |
| Enqueue 异步触发 | HIGH | v1.18 已 ship;dedup v1.18 D-09 已加 |
| 前端 BulkWriteDrawer 模式 | MEDIUM | 无既有 batch 操作前端参考(command/executions 是 task 后台扫描),需自创 |
| 权限粒度 (单一) | MEDIUM | PROJECT.md 锁定 "network:port:write" 一档,但 future phase 可能要拆分;预留 perm 字符串可拆 |
| 收集跨设备 batch 与 device lock | MEDIUM | 跨 device batch 阻塞语义未在 code 中验证,W2 写单测验证 |
| Ruijie `write` vs Huawei `commit` 差异 | MEDIUM | 已知但需 W1 templates_test 覆盖 |
| scrapligo SendConfigs 在 Ruijie 退出 system-view | LOW | scrapligo 1.3.3 + ruijie_rjos 平台未在生产实测 sendconfigs 退出行为;W1 mock 测试 + W5 site-visit UAT 验证 |

---

*Last updated 2026-07-06 — v1.19 architecture research complete (5 phases build order: W1 templates → W2 service → W3 router/handler → W4 frontend → W5 E2E).*
