# Phase 54: W5 — E2E + Real-Device UAT + Documentation - Pattern Map

**Mapped:** 2026-07-07
**Files analyzed:** 10 (3 new Go test/fixture/helper + 1 new planning doc + 1 new CHANGELOG + 4 doc edits + 1 state edit)
**Analogs found:** 10 / 10 (10 strong matches, 0 no-analog)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/portwrite/port_write_e2e_test.go` (new) | test (Go service-layer e2e) | request-response (service → mock SSH) | `internal/services/portwrite/port_write_service_test.go` | exact (same package, same target service, replaces mock with FileTransport) |
| `internal/device/e2e_helpers.go` 或 `internal/device/testing_helpers.go` (new) | utility (test-only export factory) | request-response (construct *PooledConnection for e2e) | `internal/device/connection_pool.go:29-36` (PooledConnection struct) + `internal/device/scrapli_wrapper.go:93-153` (NewScrapliWrapper factory) | role-match (composes existing structs from same package) |
| `internal/services/portwrite/testdata/*.fixture` (new, 6-8 files) | config (test fixture / device IO byte-stream) | file-I/O (scrapligo FileTransport reads bytes) | `C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\driver\network\test-fixtures\send-configs-simple.txt` | role-match (Cisco → Huawei VRP rewrite) |
| `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` (new) | config (UAT deferral tracking) | request-response (planning artifact) | `.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md` | exact (CONTEXT D-08 锁定复刻) |
| `docs/API响应规范.md` (modify, insert 端口写小节) | config (docs) | request-response | same file `### 批量操作响应` (line 184-208) | exact (same file, 新增小节) |
| `docs/安全和认证设计（国密）.md` (modify, SC#3 加密行为文档化) | config (docs) | request-response | same file `## 4. 数据加密` (line 913) + `configs/config.yaml:88-117` (exclude_paths 实证) | exact (same file) |
| `CHANGELOG.md` (new, project root) | config (docs) | request-response | `.planning/MILESTONES.md` `## v1.18` entry (line 3-25) | role-match (相近 milestone-entry 格式；项目首次建 CHANGELOG) |
| `README.md` (modify, 核心特性段) | config (docs) | request-response | same file `## 核心特性` line 7-21 (`- **网络设备纳管**：Scrapli (SSH/Telnet) + SNMP + TextFSM 模板解析，支持端口采集、MAC 历史、LLDP 拓扑` line 12) | exact (same file, 同段落补能力描述) |
| `.planning/MILESTONES.md` (modify, 加 v1.19 条目) | config (docs) | request-response | same file `## v1.18` entry (line 3-25) | exact (复刻 v1.18 格式) |
| `.planning/STATE.md` (modify, deferred 表 50→54) | config (docs) | request-response | same file `### v1.19 自身 deferred items` table (line 179-185) + line 183 `50-HUMAN-UAT.md` 字面 | exact (单行替换) |

---

## Pattern Assignments

### `internal/services/portwrite/port_write_e2e_test.go` (test, service-layer e2e)

**Analog:** `internal/services/portwrite/port_write_service_test.go` (Phase 51, same package)

**关键差异（planner 必读）：** Phase 51 mock (`mockDeviceExecutor.ExecuteCustom` line 88-96) **不调用 fn**（注释 line 394 确认）；e2e 版 `fileTransportExecutor.ExecuteCustom` **必须调用 fn**，让 `portWriteServiceImpl.executeWrite` 内 `wrapper.SendConfigs(cmds)` → `parseConfigError` 闭包链真正跑（line 184-204）。这是 Phase 54 独有增量价值，planner 必须在 plan action 显式锁此差异。

**Imports pattern** (复刻 Phase 51 测试 + 加 scrapligo transport/network/options)：

```go
// Source: port_write_service_test.go:3-18 (现有 testify + sqlite imports) +
//         scrapligo v1.4.0 driver_test.go:78-120 prepareDriver (FileTransport imports)
package portwrite

import (
    "context"
    "path/filepath"
    "testing"
    "time"

    "github.com/xingran-next/xingran-go-backend/internal/device"
    "github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
    "github.com/scrapli/scrapligo/driver/network"
    "github.com/scrapli/scrapligo/driver/options"
    "github.com/scrapli/scrapligo/platform"
    "github.com/scrapli/scrapligo/transport"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)
```

**fileTransportExecutor 类型骨架** (实现 `portWriteExecutor` 接口，调 fn)：

```go
// Source: scrapligo driver/network/driver_test.go:78-136 prepareDriver +
//         port_write_service.go:66-68 portWriteExecutor 接口签名
type fileTransportExecutor struct {
    fixturePath string // testdata/huawei_*.fixture 绝对路径
}

func (e *fileTransportExecutor) ExecuteCustom(
    ctx context.Context,
    deviceID string,
    fn func(context.Context, *device.PooledConnection) error,
    timeout time.Duration,
) error {
    // 用 platform.NewPlatform 加载 huawei_vrp.yaml 自动获取 privilege levels + prompt patterns
    // (参 scrapli_wrapper.go:131-138 现有 platform.NewPlatform 调用模式)
    p, err := platform.NewPlatform(
        "huawei_vrp",
        "dummy-host",
        options.WithTransportType(transport.FileTransport),
        options.WithFileTransportFile(e.fixturePath),
        options.WithTransportReadSize(1),  // scrapligo 测试惯例（Pitfall #1）
        options.WithReadDelay(0),
    )
    // ...
    d, _ := p.GetNetworkDriver()
    _ = d.Open()
    defer d.Close()

    // 关键：planner 必须用 device.NewPooledConnectionForE2E(见下个文件)
    // 拿到一个 pc 让 fn(ctx, pc) 真正执行 SendConfigs → parseConfigError
    pc := device.NewPooledConnectionForE2E(...) // ← 见 device 包 helper
    return fn(ctx, pc)  // ← Phase 51 mock 不做的事；e2e 核心价值
}
```

**核心 e2e 测试骨架** (复刻 `TestShutdown_Success` line 334-356)：

```go
// Source: port_write_service_test.go:334-356 (TestShutdown_Success 复刻 setup)
func TestE2E_Shutdown_Huawei_HappyPath(t *testing.T) {
    db := newTestDB(t)
    seedPortAndDevice(t, db, "port-1", "device-1", "GE0/0/1", "up", false, "")

    fixturePath, _ := filepath.Abs("testdata/huawei_shutdown_success.fixture")
    exec := &fileTransportExecutor{fixturePath: fixturePath}
    mockColl := new(mockCollectionSvc)
    svc := newTestService(exec, mockColl, db)  // ← 复用 Phase 51 工厂

    mockColl.On("Enqueue", "device-1").Return(nil)

    result, err := svc.Shutdown(context.Background(), "port-1", "e2e-test")

    assert.NoError(t, err)
    assert.Equal(t, "succeeded", result.Status)
    assert.False(t, result.NoOp)
    assert.Contains(t, result.CommandSent, "shutdown")
    mockColl.AssertExpectations(t)
}
```

**复用要点 (从 Phase 51 测试直接 import / 复用)**:
- `newTestDB(t)` (line 294-300): sqlite `:memory:` + AutoMigrate `{NetworkDevice, DevicePortStatus}` — e2e 直接复用，不重写
- `seedPortAndDevice(t, db, portID, deviceID, iface, adminStatus, dot1x, desc)` (line 304-331): DB seed 工厂，e2e 直接复用
- `newTestService(exec, coll, db)` (line 282-288): 把 `portWriteServiceImpl{db, deviceExecutor, collectionSvc}` 私有字段注入模式直接复用（e2e 把 exec 换成 `&fileTransportExecutor{...}`）
- `mockCollectionSvc` (line 99-106): Enqueue 单方法 mock — e2e 也直接复用（PortWriteService 依赖 `portWriteCollectionSvc` 接口）

**关键偏差点 (planner 必须考虑)**:
- e2e 测试需要 vendor=Huawei 的 device 行（`seedPortAndDevice` 内部已 `models.VendorHuawei`，但 e2e batch 用例多端口同设备时需确保 `device_id` 一致）
- FileTransport `Read` 在 content 耗尽时 `select{}` 死锁（scrapligo `transport/file.go:65-80`），fixture 必须完整覆盖 prompt + 回显命令 + 输出，否则 e2e 测试 60s 超时挂死（**Pitfall #1 from RESEARCH**）
- e2e 测试**不要触碰 operlog 常量**（SC#7 回归守护 `internal/utils/operlog/regression_test.go:45-71` 已锁 25 个常量值）
- `singlePortTimeout = 30 * time.Second`（service 常量 line 54），e2e 用例断言 `mockExec.On("ExecuteCustom", ..., singlePortTimeout)` 时需 import 该常量（同包，直接引用）

---

### `internal/device/e2e_helpers.go` (utility, test-only export factory)

**Analog:** `internal/device/connection_pool.go:29-36` (PooledConnection struct) + `internal/device/scrapli_wrapper.go:93-153` (NewScrapliWrapper factory)

**核心约束 (RESEARCH Open Question #1 + A1 假设, planner 必须在 Wave 0 解决):**

`device.PooledConnection` 是 concrete struct（非 interface），字段 `wrapper *ScrapliWrapper` / `refCount int32` / `lastUsed time.Time` / `deviceID string` / `mu *sync.Mutex` / `pool *DeviceConnectionPool` **多为小写私有**（`connection_pool.go:29-36`）。e2e 测试在 `portwrite` 包外（即 `port_write_e2e_test.go` 在 `portwrite` 包但 PooledConnection 在 `device` 包）**不能直接 `&device.PooledConnection{wrapper: ...}` 构造**。

**`PooledConnection` 字段定义** (connection_pool.go:28-36):

```go
// PooledConnection 池化连接，带有引用计数和设备级锁
type PooledConnection struct {
    wrapper  *ScrapliWrapper
    refCount int32       // 引用计数（原子操作）
    lastUsed time.Time   // 最后使用时间
    deviceID string      // 设备ID
    mu       *sync.Mutex // 设备级互斥锁（指针类型，用于共享）
    pool     *DeviceConnectionPool
}
```

**现有公开访问器** (connection_pool.go:153-156) — 仅返回 wrapper，无构造器：

```go
// GetWrapper 获取底层 wrapper
func (pc *PooledConnection) GetWrapper() *ScrapliWrapper {
    return pc.wrapper
}
```

**现有 NewScrapliWrapper 工厂签名** (scrapli_wrapper.go:93-153) — 接受 device/user/pwd/protocol，**不接受 transport option**（这是 RESEARCH 确证的 landmine，硬编码 SSH/Telnet）：

```go
// NewScrapliWrapper 创建scrapligo封装实例
func NewScrapliWrapper(device *models.NetworkDevice, username, password string, protocolType models.ProtocolType) (*ScrapliWrapper, error) {
    // ...
    platformName := PlatformName(device.Vendor)  // ← Huawei 映射 "huawei_vrp" (line 67-80)
    opts := []util.Option{
        options.WithAuthUsername(username),
        options.WithAuthPassword(password),
        options.WithAuthNoStrictKey(),
    }
    // 根据协议选择 transport (硬编码 SSH/Telnet，无 FileTransport 路径)
    // ...
    p, err := platform.NewPlatform(platformName, device.IPAddress, opts...)
    // ...
    return &ScrapliWrapper{driver: d, device: device, state: StateInitializing, ...}, nil
}
```

**推荐的 helper 工厂签名** (planner 在新文件实现，推荐命名 `e2e_helpers.go`，公开导出 `ForE2E` 后缀以与 production 用法物理隔离)：

```go
// internal/device/e2e_helpers.go
package device

import (
    "github.com/scrapli/scrapligo/driver/network"
)

// NewPooledConnectionForE2E 构造一个仅用于 e2e 测试的 *PooledConnection，
// 接受一个已 Open 的 *network.Driver（典型来源：scrapligo FileTransport + fixture）。
//
// 与 production GetConnection 路径物理隔离（无 connection pool / 无 refCount 协调 / 无 deviceLock），
// 仅满足 portWriteExecutor.ExecuteCustom 的 fn 签名 fn(ctx, *PooledConnection) error。
//
// 使用方负责在测试结束后调 driver.Close()（PooledConnection.Close 在 e2e 路径下是 noop）。
func NewPooledConnectionForE2E(d *network.Driver) *PooledConnection {
    return &PooledConnection{
        wrapper:  newScrapliWrapperForE2E(d),
        refCount: 1,
        deviceID: "e2e-dummy-device",
        mu:       &sync.Mutex{},
        pool:     nil, // e2e 不接入 pool
    }
}

// newScrapliWrapperForE2E 构造一个最小化 *ScrapliWrapper，仅暴露 SendConfigs。
// 不调 OpenContext（driver 已由 caller Open），不启动 initDone 轮询。
func newScrapliWrapperForE2E(d *network.Driver) *ScrapliWrapper {
    return &ScrapliWrapper{
        driver:   d,
        device:   nil,
        state:    StateReady, // 跳过 OpenContext 初始化阶段
        closing:  make(chan struct{}),
        initDone: make(chan struct{}),
    }
}
```

**复用要点**:
- `ScrapliWrapper.SendConfigs(configs []string) ([]*Response, error)` (scrapli_wrapper.go:594-616) — e2e 通过此方法跑真实 scrapligo channel layer
- `ScrapliWrapper.acquireOp/releaseOp` (scrapli_wrapper.go:245-269) — SendConfigs 内部用，e2e helper 设 `state=StateReady` 即可让 acquireOp 通过
- `PlatformName(VendorHuawei)` 返回 `"huawei_vrp"` (scrapli_wrapper.go:67-80) — e2e fixture 的 prompt pattern 必须匹配此 platform YAML

**关键偏差点 (planner 注意)**:
- 此文件是**唯一**需要在 production 包 (`internal/device`) 加 test helper 的文件；CONTEXT D-01 拒绝"在 device 包加 functional option WithTransport()"，但 D-02 暗示 "researcher 确认注入路径" → RESEARCH 结论是 "不接受"，故 planner 必须用此 helper 路径或在 fn 签名层做 narrow interface（破坏 production 签名，违反 validation only，**不推荐**）
- 命名建议：`NewPooledConnectionForE2E` 而非 `NewPooledConnectionForTest`（前者更明确语义；后者易被误用为通用 unit test 工厂）
- helper 文件**不需要 build tag**（`//go:build e2e`），因为 Go 包导出本身不区分 build tag；用命名 (`ForE2E` 后缀) + 文档注释物理隔离即可

---

### `internal/services/portwrite/testdata/*.fixture` (config, 6-8 个 Huawei VRP 字节流)

**Analog:** `C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\driver\network\test-fixtures\send-configs-simple.txt`

**Analog 内容 (Cisco IOSXE 风格, scrapligo 自带)**:

```
C3560CX#
C3560CX#configure terminal
C3560CX(config)#
C3560CX(config)#interface loopback1
C3560CX(config-if)#no interface loopback1
C3560CX(config)#
```

**Huawei VRP platform YAML prompt patterns** (从 `assets/platforms/huawei_vrp.yaml:8,16` 直接读取) — **这是 fixture 字节流必须匹配的正则**：

```yaml
# Source: C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.4.0/assets/platforms/huawei_vrp.yaml
privilege-levels:
  exec:
    pattern: '(?im)^<[\w.\-@/:]{1,63}>$'        # ← 注意是 <...> 不是裸 Huawei>
  configuration:
    pattern: '(?im)^[[\w.\-@/:]{1,63}]$'        # ← 注意是 [...] 不是 Huawei]
    escalate: 'system-view'                      # ← exec → configuration 的命令
    deescalate: 'quit'                           # ← configuration → exec 的命令
failed-when-contains:                            # ← scrapligo 自动把这些当失败
  - 'Error: Unrecognized command'
  - 'Error: Wrong parameter'
  - 'Error:Ambiguous command'
  - 'Error:Too many parameters'
  - 'Error:Incomplete command'
```

**Huawei VRP fixture 模板** (改写 send-configs-simple.txt 为 `<Huawei>` / `[Huawei]` / `[Huawei-GE0/0/1]`)：

```
# testdata/huawei_shutdown_success.fixture
# 模拟 Huawei VRP 设备返回字节流
# cmds 渲染 (vendor_port_template.go:49): shutdown action → ["shutdown"]
# scrapligo SendConfig 自动进 system-view → 下发 shutdown → 退回 exec
<Huawei>
<Huawei>system-view
[Huawei]interface GE0/0/1
[Huawei-GE0/0/1]shutdown
[Huawei-GE0/0/1]quit
[Huawei]return
<Huawei>
```

**RenderCommand 输出对照** (vendor_port_template.go:48-69，决定 fixture 必须回显的命令序列)：

| Action | RenderCommand (Huawei) 输出 cmds | fixture 必须包含的回显 |
|--------|----------------------------------|----------------------|
| shutdown | `["shutdown"]` | `[Huawei-GE0/0/1]shutdown` |
| undo_shutdown | `["undo shutdown"]` | `[Huawei-GE0/0/1]undo shutdown` |
| description | `["interface GE0/0/1", "description uplink"]` (2 cmds) | `[Huawei]interface GE0/0/1` + `[Huawei-GE0/0/1]description uplink` |
| dot1x_enable | `["dot1x enable"]` | `[Huawei-GE0/0/1]dot1x enable` |
| dot1x_disable | `["undo dot1x enable"]` | `[Huawei-GE0/0/1]undo dot1x enable` |

**device_rejected fixture 示例** (parse_error.go:73-82 rejectionMarkers)：

```
# testdata/huawei_device_rejected.fixture
<Huawei>
<Huawei>system-view
[Huawei]interface GE0/0/1
[Huawei-GE0/0/1]% Error: Unrecognized command found at '^'.
[Huawei-GE0/0/1]
```

**复用要点**:
- scrapligo FileTransport `Read` 一次返回 1 字节（`transport/file.go:65-80`），fixture 是设备 → scrapligo 的字节流，**包含 prompt + 回显命令**
- fixture **必须按 cmds 顺序预写回显命令**（Pitfall #4 from RESEARCH）：service 通过 RenderCommand 已知 cmds，fixture 必须匹配
- `screen-length 0 temporary` 命令在 `network-on-open` 自动发送（huawei_vrp.yaml:33-34）— fixture 第一行 `<Huawei>` 之后可能需要回显此命令，planner 第一个 task 应跑 1 个 happy path 验证是否需要

**关键偏差点**:
- **不要复用 `transport/test-fixtures/`** (RESEARCH Pitfall #3 已确证)：该目录只有 SSH 密钥对 `dumbserver` + `dumbserver.pub`，无设备 IO fixture。CONTEXT D-01 Claude's Discretion "优先复用" 路径不通。
- `failed-when-contains` 列表（`Error: Unrecognized command` 等）会被 scrapligo 自动标记 `Failed=true` → parseConfigError 第 2 步 `resp.Failed == true → WriteErrorTransport`（parse_error.go:109-111）。**device_rejected 测试用例**要么绕过 `failed-when-contains`（用 `% Error:` 而非 `Error:` 开头），要么期望 TransportError 而非 DeviceRejected —— planner 必须在 fixture 设计阶段决定走哪条路径
- fixture 路径建议 `testdata/*.fixture`（Go 测试惯例），文件名 `huawei_<action>_<scenario>.fixture`（如 `huawei_shutdown_success.fixture` / `huawei_description_success.fixture` / `huawei_device_rejected.fixture`）
- planner 第一个 task 应是"用 1 个 happy path 验证 platform 加载 + channel strip prompt 正确"（RESEARCH Open Question #2），验证可行后再扩展到 6-8 个

---

### `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` (config, UAT deferral tracking)

**Analog:** `.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md` (v1.18 先例)

**Frontmatter 字段** (复刻 48-HUMAN-UAT.md:1-22)：

```yaml
---
status: partial
phase: 54-w5-e2e-real-device-uat-documentation
source:
  - .planning/phases/54-w5-e2e-real-device-uat-documentation/54-VERIFICATION.md  # planner 需新建或调整
started: 2026-07-07T...
updated: 2026-07-07T...
milestone: v1.19
verifier_status: human_needed
verifier_score: "<score from automated gates + 6 项 site-visit UAT deferred (informational)>"
automated_gates:
  go_build: PASSED
  go_test_portwrite_e2e: PASSED (N tests, FileTransport replay)
  go_test_operlog_regression: PASSED (25 OperType + 11 keyword 不回归)
  go_test_full_suite: PASSED
  npm_build: PASSED
  tsc_type_check: PASSED
  config_yaml_encryption_grep: PASSED (写端点不在 exclude_paths)
---
```

**Tests (site-visit deferred) 结构** (复刻 48-HUMAN-UAT.md:44-61)：

```markdown
### 1. 真机 Huawei S5700/S5735 shutdown 命令实测
**expected:** service.Shutdown 在真机上成功执行 system-view → interface → shutdown → quit
**result:** [pending] — 推迟到现场访问
**why_human:** 无真机；FileTransport e2e 仅覆盖 1 厂商字节流回放
**addressed_in:** 下次现场访问（运维同事携带 Huawei 设备接入）

### 2. 真机 H3C VRP 同源命令差异验证
... (类似结构)

### 6. WR-02 观察：现场运维 custom-reason 使用频率
**expected:** 记录现场 1 周内 PortWriteModal "其他..." custom-reason 输入次数
**result:** [pending] — 推迟到现场访问
**why_human:** 决定 Phase 55 WR-02 修复路径（高频→修，低频→wontfix）
**addressed_in:** 下次现场访问（兑现 STATE.md Phase 55 WR-02 决策依赖闭环）
```

**Summary 表** (复刻 48-HUMAN-UAT.md:64-73)：

```markdown
| Metric | Count |
|--------|-------|
| total | 7 |   ← 6 SSH verification + 1 WR-02 观察条目 (D-09)
| passed | 0 |
| issues | 0 |
| pending | 7 |
| skipped | 0 |
| blocked | 0 |
```

**Owner + 关联声明** (复刻 48-HUMAN-UAT.md:78-87)：

```markdown
## Owner
现场访问时由运维同事携带 Huawei/H3C/Ruijie 设备接入，跑 6 端点 + UI 实测。
Site visit 完成后回写本文件 (将 `[pending]` 改为 `pass`/`fail` + 实测详情)，并通知 owner 关闭此 UAT。

## 关联声明
- `.planning/STATE.md` §Deferred Items 表（D-08 同步 50→54）
- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-VERIFICATION.md` human_verification section
- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-RESEARCH.md` §Environment Availability
- `.planning/PROJECT.md` §"Current Milestone: v1.19"（真机 UAT 推迟决策来源）
```

**复用要点**:
- 整个文档**几乎逐行复刻** 48-HUMAN-UAT.md，只改：phase 号 (48→54)、milestone (v1.18→v1.19)、具体测试项（设备组件序列号 → 端口写命令）、Summary total (3→7)
- D-09 新增 WR-02 观察条目（48-HUMAN-UAT.md 没有），planner 必须加这一项

**关键偏差点**:
- 设备型号写 **"待现场运维确认"**（不要写死 S5700/S5735，RESEARCH D-09 提示 v1.18 memory 显示现场有 S8700/RS8607E，实际型号以现场为准）
- automated_gates 清单由 planner 在 phase 末尾填**实际跑过的命令结果**（不是预测）
- SC#4 字面路径 `50-port-write-network-ports-planned/50-HUMAN-UAT.md` 是占位名（CONTEXT D-11），实际放 54 目录

---

### `docs/API响应规范.md` (modify, 新增"网络设备端口写操作"小节)

**Analog:** same file `### 批量操作响应` (line 184-208)

**Analog 现有结构** (line 184-208，新增小节仿此风格)：

```markdown
### 批量操作响应
```json
{
    "code": 0,
    "message": "批量操作完成",
    "data": {
        "success_count": 8,
        "failed_count": 2,
        "failed_items": [
            {
                "index": 3,
                "item": {"id": "item3"},
                "error": "数据不存在"
            }
        ]
    },
    "timestamp": 1766380800,
    "request_id": "req_1234567890"
}
```
```

**端口写小节建议位置**: 插在 `### 异步任务响应` (line 210) 之前（紧跟批量操作响应之后），保持"特殊场景响应"分类下的连贯性。

**新增小节骨架** (6 端点签名 + PortResult / BatchResult schema)：

```markdown
### 网络设备端口写操作

端口写操作支持 5 个单端口端点 + 1 个批量端点，所有端点位于 `/network/ports/write/*` 下，
要求 `network:port:write` 权限 + SM2+SM4 请求体加密。

#### 单端口写端点

| 路径 | 方法 | 操作类型 | operlog OperType |
|------|------|---------|-----------------|
| `/network/ports/write/shutdown` | POST | 关闭端口 | OperTypeStatus (10) |
| `/network/ports/write/undo-shutdown` | POST | 启用端口 | OperTypeStatus (10) |
| `/network/ports/write/description` | POST | 设置描述 | OperTypeUpdate (2) |
| `/network/ports/write/dot1x-enable` | POST | 启用 802.1X | OperTypeStatus (10) |
| `/network/ports/write/dot1x-disable` | POST | 停用 802.1X | OperTypeStatus (10) |

**请求体 schema** (port_write_handler.go:54-63)：

```json
{
    "portId": "uuid-string",         // 必填，目标端口 UUID
    "description": "uplink-to-core",  // 可选，仅 description 端点使用（≤80 字符）
    "reason": "操作原因"              // 可选，UI-02 操作原因，后端仅记录不校验
}
```

**响应 data** (port_write_service.go:34-42 PortResult)：

```json
{
    "code": 0,
    "message": "success",
    "data": {
        "portId": "uuid-string",
        "action": "shutdown",
        "status": "succeeded",         // "succeeded" | "failed" | "skipped"
        "noOp": false,                 // PORT-06 已处目标态时为 true
        "currentState": "admin_down",  // NoOp 时填，可选
        "error": "",                   // failed 时填
        "commandSent": "shutdown"      // 审计真相源，不脱敏
    }
}
```

#### 批量写端点

`POST /network/ports/write/batch` — 同设备多端口同操作，serial fail-fast。

**请求体 schema** (port_write_service.go:45-50 BatchWriteRequest)：

```json
{
    "deviceId": "uuid-string",
    "action": "shutdown",              // 5 种 PortAction 之一
    "portIds": ["uuid-1", "uuid-2"],  // 1-50 个
    "description": "uplink"            // 可选，仅 action=description
}
```

**响应 data** (batch_orchestrator.go:14-19 BatchResult，三数组同时存在)：

```json
{
    "code": 0,
    "data": {
        "succeeded": [ /* PortResult[] */ ],
        "failed":    [ /* PortResult[] */ ],
        "skipped":   [ /* PortResult[] — PORT-06 NoOp */ ]
    }
}
```

任一端口 transport/device_rejected 错误触发 fail-fast（剩余端口不执行，不进任何数组）。
```

**复用要点**:
- OperType 常量值 (10/2) 直接引用 `internal/utils/operlog/operlog.go:36-66`，不重写
- PortResult / BatchResult schema 字段名直接引用 `port_write_service.go:34-50` + `batch_orchestrator.go:14-19`
- 路由路径引用 `port_write_router.go:42-47`（6 个 kebab 路径）

**关键偏差点**:
- 现有"批量操作响应"用 `success_count/failed_count/failed_items` 结构（line 184-208），端口写 batch 用 `succeeded/failed/skipped` 三数组（service 实际返回结构）。**不要硬套现有格式**，planner 显式说明端口写 batch 是"三数组"语义（PORT-06 skipped 不存在于通用批量操作）。

---

### `docs/安全和认证设计（国密）.md` (modify, SC#3 加密行为文档化)

**Analog:** same file `## 4. 数据加密` (line 913) + `configs/config.yaml:88-117` (exclude_paths 实证)

**config.yaml exclude_paths 实证** (line 88-100) — **SC#3 文档化的事实依据**：

```yaml
# configs/config.yaml:88-100 (已 VERIFIED)
request_encryption:
  enabled: true
  exclude_paths:
    - "/api/v1/system/auth/public-key"  # 公钥接口必须排除
    - "/api/v1/system/auth/test-sm2"
    - "/api/v1/upload/*"
    - "/api/v1/captcha/*"
    - "/api/v1/rpa/workers/register"
    - "/api/v1/rpa/workers/*/heartbeat"
    - "/api/v1/rpa/workers/progress"
    # 注意：/network/ports/write/* 不在列表 → 写端点保持 SM2+SM4 加密
```

**建议新增小节位置**: `## 4. 数据加密` 之后或 `### 4.1 敏感数据加密` 子段下，新增 `### 4.x 网络设备写端点加密行为` 小节。

**新增小节骨架** (区分 SSH 加密层 vs HTTP 加密层 — RESEARCH Pitfall #5)：

```markdown
### 4.x 网络设备写端点加密行为

**关键区分：两层加密正交**

| 加密层 | 协议 | 路径 | 配置位置 |
|-------|------|------|---------|
| SSH transport 加密 | 后端 → 网络设备（scrapligo） | 不经过 HTTP 中间件 | scrapligo driver 内部（StandardTransport / TelnetTransport） |
| HTTP 请求体 SM2+SM4 加密 | 前端 → 后端（Gin 中间件） | `/network/ports/write/*` | `configs/config.yaml: request_encryption.exclude_paths` |

**写端点保持 HTTP 加密（不豁免）**

网络设备端口写操作（shutdown / undo-shutdown / description / dot1x-enable / dot1x-disable / batch）
是敏感操作（影响生产网络），HTTP 请求体 `{portId, reason}` 保持 SM2+SM4 加密。

实证：`configs/config.yaml: request_encryption.exclude_paths` 列表不含 `/network/ports/write/*`，
中间件 `pkg/middleware/encryption.go` 对写端点正常加解密请求体。

**与 SSH 加密无关**

SSH 是后端 (scrapligo) 与网络设备之间的协议层加密，由 scrapligo StandardTransport
（CBC + CTR 模式兼容老/新设备）自管，与前端到后端的 HTTP SM2+SM4 加密正交。
```

**复用要点**:
- exclude_paths 实证由 `configs/config.yaml:88-100` 提供，文档**引用而非重写**
- 中间件路径引用 `pkg/middleware/encryption.go`（planner 实际 grep 确认文件名）

**关键偏差点**:
- **不要按 SC#3 字面写"写端点不加 SM2+SM4"**（CONTEXT D-04 锁定保持加密，RESEARCH Pitfall #5 警告）。文档化时必须明确"保持加密"。
- 中间件匹配逻辑（filepath.Match + `/*` 通配）planner 可在 grep 实际中间件源码后补充说明

---

### `CHANGELOG.md` (new, project root)

**Analog:** `.planning/MILESTONES.md` `## v1.18` entry (line 3-25)

**Analog v1.18 entry 结构** (MILESTONES.md:3-25)：

```markdown
## v1.18 网络设备硬件清单 (Device Component Serials) — ✅ SHIPPED 2026-07-04

**Phases**: 1 (Phase 48) | **Plans**: 3 (48-01 / 48-02 / 48-03) | **Waves**: 3 (...)

**Delivered**: 实现"一机多序列号"——... (一段话概述)

**Key Accomplishments**:
1. ✅ **Schema 落地 (48-01 / Wave 1)** — ...
2. ✅ **组件收集器包 (48-02 / Wave 2)** — ...
...

**Comms**:
- N commits 主仓 + N merge 嵌套
- 14/14 D-id 全覆盖
- 0 critical regression
- N 文件 / N insertions

**Known deferred items at close**: ...
```

**CHANGELOG.md 推荐结构** (Keep a Changelog 风格 + 项目本地化)：

```markdown
# Changelog

本项目所有显著变更记录于此文件。格式参考 [Keep a Changelog](https://keepachangelog.com/)，
版本号遵循 [Semantic Versioning](https://semver.org/)。

## [v1.19] - 2026-07-XX

### Added — 网络设备写命令（端口 shutdown/dot1x/description）

- **后端**: 6 个写端点（5 单端口 + 1 batch），路径 `/network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch}`，要求 `network:port:write` 权限
- **服务层**: `PortWriteService` (5 方法 + batch)，3 厂商模板渲染（Huawei/H3C/Ruijie）+ pre-state NoOp 检测（PORT-06）+ detached 30min context 批量 fail-fast
- **审计**: `sys_port_write_audit` 表（前后值真相源）+ operlog 全覆盖（CONV-01..04：shutdown→OperTypeStatus / description→OperTypeUpdate / batch→OperTypeBatch）
- **前端**: PortWriteModal + BulkWriteDrawer + ColumnConfig + 通知配置 UI
- **测试**: service 层 e2e（FileTransport 回放）+ operlog 25 OperType regression guard

### Phases
- Phase 50 (W1): 厂商模板单元测试 + vendor action/command map
- Phase 51 (W2): PortWriteService + batch orchestrator + mock tests
- Phase 52 (W3): router/handler/operlog/permission migration
- Phase 53 (W4): 前端 UI
- Phase 54 (W5): e2e + 真机 UAT 推迟 + 文档化

### Deferred
- 真机 SSH 写命令验证 → `54-HUMAN-UAT.md` site visit
- 3 厂商 e2e fixture 全覆盖 → v1.19.x+
- 跨固件版本命令差异 → follow-up

## [v1.18] - 2026-07-04

（可选补 v1.18 一行总结，详见 `.planning/MILESTONES.md`）
```

**复用要点**:
- 直接引用 MILESTONES.md v1.18 entry 的"Phases / Plans / Waves / Delivered / Key Accomplishments / Comms / Known deferred items" 字段结构
- 项目本地化：用 Keep a Changelog 的 Added/Changed/Deprecated/Removed/Fixed/Security 分类（更标准），同时保留 Phases/Deferred 项目特有字段

**关键偏差点**:
- README.md head 有 `<!-- generated-by: gsd-doc-writer -->` 标记（README line 1）— **CHANGELOG.md 不加此标记**（CONTEXT D-05 锁定独立 CHANGELOG 避免生成器冲突）
- v1.18 是否补：CONTEXT D-05 默认 v1.19 起记，planner 可选补 v1.18 一行（参 MILESTONES.md v1.18 段）

---

### `README.md` (modify, 核心特性段)

**Analog:** same file `## 核心特性` line 7-21, specifically line 12 (`- **网络设备纳管**：...`)

**Analog 现有行** (line 12)：

```markdown
- **网络设备纳管**：Scrapli (SSH/Telnet) + SNMP + TextFSM 模板解析，支持端口采集、MAC 历史、LLDP 拓扑
```

**修改后建议** (在原行后追加"端口写命令"能力)：

```markdown
- **网络设备纳管**：Scrapli (SSH/Telnet) + SNMP + TextFSM 模板解析，支持端口采集、MAC 历史、LLDP 拓扑、端口写命令（shutdown / undo shutdown / description / dot1x 启停）+ 批量配置 + 完整审计（sys_port_write_audit）
```

**复用要点**:
- 在同一行扩展，不新增 bullet（保持"核心特性"列表紧凑）
- 关键词覆盖：端口写命令 + 批量配置 + 审计（对应 SC#5 要求）

**关键偏差点**:
- **不要改 README 顶部 `<!-- generated-by: gsd-doc-writer -->` 标记**（line 1）— 但 planner 可以在标记下方手动编辑"核心特性"段（CONTEXT D-05 提示：生成器覆盖风险主要在"版本历史"段，核心特性段相对稳定；planner 实际编辑后可观察生成器行为）
- 同时检查 line 35 `scrapligo v1.3.3` 是否需改为 `v1.4.0`（D-10 纠正）— 这属于文档事实更新，planner 可顺手改

---

### `.planning/MILESTONES.md` (modify, 加 v1.19 条目)

**Analog:** same file `## v1.18` entry (line 3-25)

**Analog v1.18 完整结构** (line 3-25)：

```markdown
## v1.18 网络设备硬件清单 (Device Component Serials) — ✅ SHIPPED 2026-07-04

**Phases**: 1 (Phase 48) | **Plans**: 3 (48-01 / 48-02 / 48-03) | **Waves**: 3 (schema → collectors → pipeline+operlog+frontend)

**Delivered**: 实现"一机多序列号"——... (一段话概述)

**Key Accomplishments**:

1. ✅ **Schema 落地 (48-01 / Wave 1)** — ...
2. ✅ **组件收集器包 (48-02 / Wave 2)** — ...
3. ✅ **Pipeline + 审计 (48-03 / Wave 3)** — ...
4. ✅ **Index + 查询不污染主表** — ...
5. ✅ **前端 ComponentListTab** — ...
6. ✅ **真机 UAT 推迟声明(为 P0 完成让路)** — ...

**Comms**:
- 6 commits 主仓 + 13 merge 嵌套 ...
- 14/14 D-id 全覆盖 ...
- 0 critical regression ...
- 44 文件 / 4,712 insertions(+57 deletions)

**Known deferred items at close**: ...

---
```

**v1.19 条目建议结构** (复刻 v1.18，填 v1.19 实际数据)：

```markdown
## v1.19 网络设备写命令 (Port Write Operations) — ✅ SHIPPED 2026-07-XX

**Phases**: 5 (Phases 50-54) | **Plans**: 5-7 (50-01 / 51-01 / 52-01 / 53-01 / 54-01) | **Waves**: 5 (W1 模板 → W2 服务 → W3 HTTP+审计+权限 → W4 前端 → W5 e2e+UAT+文档)

**Delivered**: 网络设备端口写命令全栈实现——3 厂商（Huawei/H3C/Ruijie）× 5 操作（shutdown / undo shutdown / description / dot1x enable / dot1x disable）+ batch 批量 + 完整审计（sys_port_write_audit + operlog 25 OperType）+ 权限隔离（network:port:write）+ 前端 PortWriteModal/BulkWriteDrawer + service 层 e2e（scrapligo FileTransport 回放）+ 真机 UAT 推迟文档化。

**Key Accomplishments**:

1. ✅ **W1 厂商模板 (Phase 50)** — vendorPortTemplate map + RenderCommand + Huawei/H3C VRP 同源 + Ruijie RGOS Cisco 风格 dot1x
2. ✅ **W2 PortWriteService (Phase 51)** — 6 方法 service + BatchWritePorts fail-fast + detached 30min context + parseConfigError 5 步优先级 + PORT-06 pre-state NoOp + mock tests (mockDeviceExecutor)
3. ✅ **W3 HTTP + 审计 + 权限 (Phase 52)** — 6 kebab 端点 + 组级 RequirePermissions([network:port:write]) + sys_port_write_audit 表 + operlog CONV-01..04 + Phase Path C migration
4. ✅ **W4 前端 UI (Phase 53)** — PortWriteModal + BulkWriteDrawer + ColumnConfig + 通知配置 + Nyquist validation
5. ✅ **W5 e2e + UAT + 文档 (Phase 54)** — service 层 e2e（FileTransport replay 补 Phase 51 fn 闭包漏洞）+ 54-HUMAN-UAT.md 真机推迟 + API/加密/CHANGELOG/README/MILESTONES 文档化 + 全量回归绿灯

**Comms**:
- N commits 主仓 + N merge 嵌套（planner 填实际）
- N/N D-id 全覆盖
- 0 critical regression（go test ./... + npm build + type-check + operlog regression 全绿）
- N 文件 / N insertions

**Known deferred items at close**: 6 项 SSH 真机 verification + 1 项 WR-02 观察 → 54-HUMAN-UAT.md site visit；HTTP handler 层 e2e / 3 厂商 fixture 全覆盖 / BATCH-05 实时进度 → v1.19.x+

---
```

**复用要点**:
- 复刻 v1.18 entry 的所有字段：Phases/Plans/Waves/Delivered/Key Accomplishments (编号)/Comms/Known deferred items
- 位置：插在 `## v1.18` (line 3) **之前**（最新版本在上，符合 MILESTONES.md 现有倒序：v1.18 → v1.17 → v1.3 → v1.2 → v1.1 → v1.0）

**关键偏差点**:
- 实际 commit 数 / 文件数 / D-id 覆盖率 planner 在 phase 末尾填**实际数据**
- "SHIPPED [date]" 由 phase 完成日填

---

### `.planning/STATE.md` (modify, deferred 表 50→54)

**Analog:** same file `### v1.19 自身 deferred items` table (line 179-185)

**Analog 现有表** (line 179-185)：

```markdown
### v1.19 自身 deferred items (W5 记录)

| Item | Category | Deferred to | Owner |
|------|----------|-------------|-------|
| 真机 SSH 写命令验证 (Huawei S5700/S5735 + H3C + Ruijie RS8607E 各 shutdown + description + dot1x) | device_needed | 50-HUMAN-UAT.md site visit | 现场运维 |
| 跨固件版本命令差异 (Huawei V200R005 vs V600R024C00) | device_needed | follow-up | 现场运维 |
| Real-device SSH 往返延迟测量 (batching + per-port timeout calibration) | device_needed | follow-up | 现场运维 |
```

**修改后** (line 183 单行替换 `50-HUMAN-UAT.md` → `54-HUMAN-UAT.md`)：

```markdown
| 真机 SSH 写命令验证 (Huawei S5700/S5735 + H3C + Ruijie RS8607E 各 shutdown + description + dot1x) | device_needed | 54-HUMAN-UAT.md site visit | 现场运维 |
```

**复用要点**:
- 单字符替换：`50-HUMAN-UAT.md` → `54-HUMAN-UAT.md`（CONTEXT D-08 同步要求）
- 不动其他行

**关键偏差点**:
- 同时检查 STATE.md §Critical Pitfalls / §Known Risks 是否还有 v1.19 相关条目需在 phase 末尾更新（planner 在 verify-work 阶段 review）

---

## Shared Patterns

### Phase 51 测试基建（直接复用）

**Source:** `internal/services/portwrite/port_write_service_test.go:282-331`
**Apply to:** `internal/services/portwrite/port_write_e2e_test.go`

```go
// newTestService 构造最小可用 *portWriteServiceImpl（line 282-288）
func newTestService(exec portWriteExecutor, coll portWriteCollectionSvc, db *gorm.DB) *portWriteServiceImpl {
    return &portWriteServiceImpl{
        db:             db,
        deviceExecutor: exec,
        collectionSvc:  coll,
    }
}

// newTestDB 构造内存 sqlite + AutoMigrate（line 294-300）
func newTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    assert.NoError(t, err)
    assert.NoError(t, db.AutoMigrate(&models.NetworkDevice{}, &models.DevicePortStatus{}))
    return db
}

// seedPortAndDevice 创建一个测试设备 + 一个端口行（line 304-331）
// 多次调用同一 deviceID 时只创建 device 一次（避免 ip_address UNIQUE 冲突）
func seedPortAndDevice(t *testing.T, db *gorm.DB, portID, deviceID, interfaceName, adminStatus string, dot1xEnabled bool, description string)
```

**Apply pattern:** e2e 测试**不重写**这三个工厂，直接在同包内调用（Go 同包测试可见性）。e2e 把 `mockDeviceExecutor` 替换为 `fileTransportExecutor`，其他不变。

---

### operlog OperType 常量映射（API 文档 + CHANGELOG 引用）

**Source:** `internal/utils/operlog/operlog.go:36-66` + regression_test.go:45-71
**Apply to:** `docs/API响应规范.md` (端口写小节) + `CHANGELOG.md` (v1.19 entry) + `internal/services/portwrite/port_write_e2e_test.go` (e2e 不触碰常量)

```go
// Source: internal/utils/operlog/operlog.go:36-66 (VERIFIED)
OperTypeOther   = 0
OperTypeCreate  = 1
OperTypeUpdate  = 2  // ← description action
OperTypeDelete  = 3
// ...
OperTypeStatus  = 10 // ← shutdown / undo-shutdown / dot1x-enable / dot1x-disable
// ...
OperTypeBatch   = 16 // ← batch action
// ...
OperTypeUnlock  = 24 // 总计 25 个常量，值固定不可重排
```

**Apply pattern:** API 文档小节的 OperType 列直接引用这些常量名 + 值（如 `OperTypeStatus (10)`），不重新定义。CHANGELOG 提到 CONV-01..04 映射时引用常量名。e2e 测试**不直接断言 OperType 值**（由 `regression_test.go` 守护）。

---

### scrapligo FileTransport + platform.NewPlatform 工厂模式

**Source:** `C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\driver\network\driver_test.go:78-136` (prepareDriver) + `internal/device/scrapli_wrapper.go:131-144` (项目现有 platform 调用)
**Apply to:** `internal/services/portwrite/port_write_e2e_test.go` (fileTransportExecutor.ExecuteCustom 内 driver 构造)

```go
// Source: scrapligo driver/network/driver_test.go:83-120 prepareDriver (FileTransport + custom priv levels)
d, err := network.NewDriver(
    "dummy",
    options.WithTransportType(transport.FileTransport),
    options.WithFileTransportFile(resolveFile(t, payloadFile)),
    options.WithTransportReadSize(1),  // scrapligo 测试惯例
    options.WithReadDelay(0),
    // ...privilege levels (Huawei 由 platform YAML 自动加载，可不显式传)
)

// 项目现有 platform.NewPlatform 调用 (scrapli_wrapper.go:131-144)
p, err := platform.NewPlatform(
    platformName,         // "huawei_vrp" for Huawei (scrapli_wrapper.go:67-80 PlatformName)
    device.IPAddress,
    opts...,              // 含 transport / auth / file options
)
d, err := p.GetNetworkDriver()
```

**Apply pattern:** e2e 用 `platform.NewPlatform("huawei_vrp", "dummy-host", fileTransportOpts...)` 加载华为 platform YAML（自动获取 prompt patterns + privilege levels），比 scrapligo 自身 prepareDriver 显式传 priv levels 更简洁。

---

### UAT 推迟文档三件套（frontmatter + Tests + Summary）

**Source:** `.planning/phases/48-device-component-serials-planned/48-HUMAN-UAT.md`
**Apply to:** `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md` (CONTEXT D-08 锁定逐行复刻)

```markdown
---
status: partial
phase: <phase-dir-name>
milestone: <vX.Y>
verifier_status: human_needed
automated_gates: { ... }
---

## 自动化闸门（已 PASS）
## Tests (site-visit, N 项 deferred)
### 1. <test name>
**expected:** ...
**result:** [pending]
**why_human:** ...
**addressed_in:** 下次现场访问
## Summary (table: total/passed/issues/pending/skipped/blocked)
## Gaps
## Owner
## 关联声明
```

**Apply pattern:** 整个 54-HUMAN-UAT.md 文档结构逐行复刻 48 模板，只改 phase 号 / milestone / 具体测试项 / total count。

---

## No Analog Found

无。所有 10 个文件均有强匹配 analog（exact 或 role-match）。

| File | Reason |
|------|--------|
| (none) | — |

注：`CHANGELOG.md` 项目首次建无完全匹配 analog，但 `.planning/MILESTONES.md` v1.18 entry 提供了相近的"milestone-entry"格式（role-match），planner 在 Keep a Changelog 标准格式基础上融合 MILESTONES.md 字段结构即可。

---

## Metadata

**Analog search scope:**
- `internal/services/portwrite/` (Phase 51 全套: service / batch / parse_error / test)
- `internal/device/` (connection_pool / scrapli_wrapper / executor)
- `internal/services/portcollection/` (vendor_port_template + tests — Huawei cmds 真实来源)
- `internal/api/v1/network/` (port_write_router / port_write_handler — API 文档来源)
- `internal/utils/operlog/` (operlog.go + regression_test.go — OperType 常量来源)
- `configs/config.yaml` (line 88-117 encryption exclude_paths 实证)
- `docs/API响应规范.md` + `docs/安全和认证设计（国密）.md` (现有文档结构)
- `.planning/MILESTONES.md` + `.planning/STATE.md` + `.planning/phases/48-*/48-HUMAN-UAT.md` (文档模板)
- `README.md` (核心特性段)
- `C:\Users\CPIC\go\pkg\mod\github.com\scrapli\scrapligo@v1.4.0\` (transport/file.go + driver/network/driver_test.go prepareDriver + driver/network/test-fixtures/send-configs-simple.txt + assets/platforms/huawei_vrp.yaml)

**Files scanned:** 18 source files + 4 planning docs + 4 scrapligo v1.4.0 reference files

**Pattern extraction date:** 2026-07-07

**Key landmines surfaced for planner attention:**
1. **PooledConnection 私有字段** (RESEARCH A1 / Open Question #1) — 必须在 `internal/device/` 加 test helper（`NewPooledConnectionForE2E`），planner Wave 0 第一 task 解决
2. **Huawei VRP prompt patterns** 是 `<Huawei>` / `[Huawei]` 不是 `Huawei>` / `Huawei]` (来自 `huawei_vrp.yaml:8,16`) — fixture 字节流必须匹配
3. **`failed-when-contains` 自动标记 Failed=true** (`huawei_vrp.yaml:23-29`) — device_rejected 测试用例需用 `% Error:` 而非 `Error:` 开头避开，或期望 TransportError
4. **`transport/test-fixtures/` landmine** (RESEARCH Pitfall #3) — 只有 SSH 密钥对，无设备 IO fixture，全部 fixture 手写
5. **README.md generated-by 标记** (line 1) — CHANGELOG.md 独立文件不加此标记，避免生成器冲突

## PATTERN MAPPING COMPLETE
