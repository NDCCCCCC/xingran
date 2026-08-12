# Phase 50: W1 — Vendor Templates + Unit Tests - Context

**Gathered:** 2026-07-06
**Status:** Ready for planning
**Source:** v1.19 init decisions (see `.planning/PROJECT.md` Current Milestone 段) + `.planning/REQUIREMENTS.md` SSH-01/05 + `.planning/ROADMAP.md` Phase 50 段

<domain>
## Phase Boundary

锁定 vendor → action → command 契约（**纯 Go map + 单测，零外部依赖**）作为 Phase 51-53 的稳定底座。

**In scope**:
- `internal/services/portcollection/vendor_port_template.go` 新建：3 厂商 × 5 操作 = 15 个硬编码命令模板
- `vendor_port_template_test.go` 同包：12+ 单元测试覆盖每个 (vendor, action) 组合 + 负向用例（未支持 vendor、未知 action、参数缺失）
- 公共导出函数 `RenderCommand(vendor, action, params) ([]string, error)`
- 命名类型 `PortAction`（Go const）与 `PortTemplateParams`（struct）

**Out of scope**:
- SSH 连接管理（Phase 51 PortWriteService）
- HTTP handler / router / operlog（Phase 52）
- 前后端 UI（Phase 53）
- 真实设备验证（Phase 54 mock SSH e2e + 现场 UAT 推迟）
- Maipu 厂商（无设备，后续 phase）
- Cisco IOS-XE/XR（已锁后续 phase，FUTURE-08）
- 数据库抽象（v1.19 init 锁定为"硬编码 Go map，落地为先"）

</domain>

<decisions>
## Implementation Decisions

### D-01: 厂商与操作范围（v1.19 init locked）
- 厂商 3 个：Huawei VRP / H3C Comware / Ruijie RGOS
- 操作 5 个：shutdown / undo shutdown / description / dot1x enable / dot1x disable
- 总计 15 个 (vendor, action) 模板
- 源：`.planning/PROJECT.md` §"锁定决策 (v1.19 init)" + `.planning/REQUIREMENTS.md` PORT-01..05

### D-02: 文件位置 + 包归属
- 文件：`internal/services/portcollection/vendor_port_template.go`
- 包：`portcollection`（与现有 `template_cache.go` 同包，但语义不同——`template_cache.go` 是 TextFSM 模板缓存，Phase 50 是厂商命令模板）
- 公共导出：`RenderCommand`、`PortAction`、`PortTemplateParams`
- 源：`.planning/ROADMAP.md` Phase 50 §Success Criteria #1

### D-03: `PortAction` 命名风格 = PascalCase 常量 + snake_case 字符串值
- 例：`ActionShutdown PortAction = "shutdown"`、`ActionDot1xEnable PortAction = "dot1x_enable"`
- 兼顾 Go 代码可读（`switch action { case ActionShutdown }`） + 审计日志可读（operlog `action: shutdown` 接近厂商术语）
- 影响 Phase 52 operlog 字段取值（CONV-01..04 已锁 OperType 映射，但 action 字符串走 PortAction 值）

### D-04: `PortTemplateParams` 显式 struct
```go
type PortTemplateParams struct {
    InterfaceName string // 必填，shutdown/undo/description/dot1x 都需要
    Description   string // 仅 description action 用；其他 action 忽略
}
```
- 强类型 + 字段文档化
- Phase 51 service 调用方传参清晰：`RenderCommand(v, ActionShutdown, PortTemplateParams{InterfaceName: "GE0/0/1"})`
- 后续 phase 加新字段（如 VLAN）不破坏签名

### D-05: 所有 action 统一返回 `[]string` 形态
- shutdown / undo shutdown：`[]string{"shutdown"}` 长度 1
- description（多命令）：`[]string{"interface GE0/0/1", "description uplink"}` 长度 2
- dot1x enable/disable Huawei/H3C（VRP 同源）：长度 1
- dot1x enable/disable Ruijie（Cisco 风格需进 interface view）：长度 2
- Phase 51 service 用 `for i, cmd := range cmds { ... }` 串行 SendConfig，失败时 `i` 即可定位哪条命令被拒

### D-06: 参数校验位置
- `RenderCommand` 内部校验：InterfaceName 非空（所有 action 必填）
- `RenderCommand` 内部校验：description action 时 Description 非空 + ≤ 80 字符（与 `device_port_status.Description` size:500 中预留 80 的 UI 字符约定对齐）
- 校验失败返回 sentinel error（`ErrEmptyInterfaceName` / `ErrDescriptionTooLong` / `ErrDescriptionEmpty`），Phase 51 service 翻译为 HTTP 400
- Description 内容转义：暂不强制（scrapli 文本透传，设备端解析；后续 phase 如发现注入风险再加）

### D-07: 锐捷 dot1x 走 Cisco 风格进 interface view
- Ruijie dot1x enable：`["interface GigabitEthernet0/0/1", "dot1x port-control auto"]`
- Ruijie dot1x disable：`["interface GigabitEthernet0/0/1", "no dot1x port-control"]`
- 接口名归一化：调用方负责（`device_port_status.go:70` BeforeCreate hook 已强制归一为短名 → 锐捷进 view 需还原全称，**Phase 50 接受传入全称或短名 + 通过 `GetCommandForVendor`-like helper 转换**；Phase 51 实施时如冲突则改 D-07）
- 注：此为预判，planner 需在 Phase 51 时核实 `scrapligo` assets 期望的接口名形态

### D-08: Huawei / H3C 命令同源
- shutdown：`shutdown`
- undo shutdown：`undo shutdown`
- description：`interface <name>` + `description <text>`（H3C 也用 "description" 而非 Cisco 的 "description"）
- dot1x enable：`dot1x enable`
- dot1x disable：`undo dot1x enable`
- 区别仅在 scrapligo platform 配置文件（已有 `huawei_vrp` / `hp_comware`）

### D-09: 未支持 vendor / 未知 action 错误策略
- 未支持 vendor（如 Cisco、Maipu）：返回 `ErrUnsupportedVendor`，包含 vendor 名
- 未知 action：返回 `ErrUnknownAction`，包含 action 字符串
- 阶段边界对齐：`.planning/REQUIREMENTS.md` Out of Scope 列已显式排除 Cisco/自动回滚/dry-run

### Claude's Discretion

- **测试用例组织形式**：12+ 单测具体编排（表驱动 vs 子测试 `t.Run`）— `internal/services/operations/reference_resolver_test.go` 风格为参考，planner 选其一
- **测试 golden 数据存放**：内联 table（_test.go 内）vs `testdata/vendor_port_template_golden.json` — Phase 50 仅 15 用例，倾向内联（与 TESTING.md 现状一致，**不引入新约定**）
- **sentinel error 定义位置**：`vendor_port_template.go` 内部 `var Err... = errors.New(...)` 还是单独 `errors.go` — 倾向同文件（保持 1 文件 1 关注点）
- **PortAction 是否暴露 String() 方法**：Phase 52 operlog 记录 action 字段时是否需要 — 倾向暴露（`func (a PortAction) String() string` 用值即可），便于日志

</decisions>

<canonical_refs>
## Canonical References

**下游 agent (planner / researcher) 必须先读这些。**

### 网络设备模型与 SSH 底层
- `internal/models/network_device.go` — `DeviceVendor` 枚举（huawei/h3c/ruijie/maipu）+ `NetworkDevice` 字段
- `internal/models/device_port_status.go` — `InterfaceName` 归一化 hook（line 70）+ `Description size:500`
- `internal/device/scrapli_wrapper.go` — `SendConfig` / `SendConfigs` 调用契约（line 567/594）+ `PlatformName` 厂商映射（line 67）+ `GetCommandForVendor` read-only 模式参考（line 682）
- `internal/device/connection_pool.go` — 连接池契约（Phase 51 实施时使用，本 phase 暂无依赖）

### 端口采集参考架构（同包 sibling）
- `internal/services/portcollection/template_cache.go` — 同包现有 TextFSM 模板缓存（与 Phase 50 关注点不同，仅作"同包共存"参考）
- `internal/services/portcollection/parser.go` — vendor 命令分发模式（`getInterfaceCommand` 等）
- `internal/services/portcollection/collection.go` — 采集 service 参考（vendor 分发 + 批量执行结构）

### v1.19 锁定决策
- `.planning/PROJECT.md` §"锁定决策 (v1.19 init)" — 5 条 v1.19 init 决策
- `.planning/REQUIREMENTS.md` — 36 项 v1.19 MVP 需求（SSH/PORT/BATCH/AUDIT/UI/PERM/INFRA/CONV 8 类）
- `.planning/ROADMAP.md` Phase 50 段 — 5 条 Success Criteria
- `.planning/STATE.md` §"Critical Pitfalls → Mitigation Map" — 7 项 v1.19 pitfall 与 phase 对应

### 测试规范
- `.planning/codebase/TESTING.md` — testify/assert + `//go:build !skip_db_tests` 模式（Phase 50 是纯 Go map 无需 DB tag）
- `internal/services/operations/reference_resolver_test.go` — 单测风格参考（行 1-55）
- `internal/services/operations/building_service_test.go` — 同包多个 _test.go 共存示例

### 操作日志
- `internal/utils/operlog/` — operlog helper（Phase 50 不直接调用，但常量值需保持兼容：OperTypeStatus/OperTypeUpdate/OperTypeBatch）
- `.planning/PROJECT.md` §"操作日志记录约定" — OperType 25 常量集（值固定不可重排）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `scrapli_wrapper.go:SendConfig(string) (*Response, error)` — Phase 51 service 直接调用 Phase 50 模板的每个字符串；Phase 50 不调用
- `scrapli_wrapper.go:SendConfigs([]string) ([]*Response, error)` — Phase 51 批量执行 `[]string` 用，与 Phase 50 模板返回形态天然对齐（D-05 决策依据）
- `scrapli_wrapper.go:PlatformName(vendor) string` — vendor → scrapligo platform 名映射（huawei_vrp / hp_comware / ruijie_rjos），Phase 51 使用
- `scrapli_wrapper.go:GetCommandForVendor(vendor, commandType) string` — read-only 厂商命令查询模式（line 682），Phase 50 模板可视为其写命令孪生兄弟
- `models.NetworkDevice.Vendor` — DeviceVendor 类型（已存在 string 别名），Phase 50 公共函数第一参数直接用此类型

### Established Patterns
- **Go const 命名**：项目惯例 PascalCase + 字符串值（如 `internal/constants/` 下的常量）
- **错误返回**：service 层 `return fmt.Errorf("...: %w", err)` 包装；sentinel error 用 `errors.New`（如 `pkg/errors/errors.go`）
- **testify 断言**：`assert.NoError` / `assert.Equal` / `assert.Error` / `assert.Contains`（见 TESTING.md）
- **包内多 _test.go**：参考 `internal/services/operations/` 同包 8 个 _test.go 共存

### Integration Points
- **同包导入路径**：`github.com/xingran-next/xingran-go-backend/internal/services/portcollection`
- **下游消费方**（Phase 51）：
  - `internal/services/portwrite/port_write_service.go`（新建，Phase 51）— 调 `portcollection.RenderCommand`
  - `internal/services/portwrite/batch_orchestrator.go`（新建，Phase 51）— 同上
- **同包不冲突**：`template_cache.go` 走 `templates.FSM` 类型；Phase 50 走 `[]string` 类型，关注点隔离

</code_context>

<specifics>
## Specific Ideas

### 命令模板骨架（D-01~D-08 综合）

```go
package portcollection

import (
    "errors"
    "fmt"
    "strings"

    "github.com/xingran-next/xingran-go-backend/internal/models"
)

// PortAction 端口写操作类型
type PortAction string

const (
    ActionShutdown      PortAction = "shutdown"
    ActionUndoShutdown  PortAction = "undo_shutdown"
    ActionDescription   PortAction = "description"
    ActionDot1xEnable   PortAction = "dot1x_enable"
    ActionDot1xDisable  PortAction = "dot1x_disable"
)

func (a PortAction) String() string { return string(a) }

// PortTemplateParams 模板渲染参数
type PortTemplateParams struct {
    InterfaceName string // 所有 action 必填
    Description   string // 仅 ActionDescription 使用；其他 action 忽略
}

// Sentinel errors
var (
    ErrUnsupportedVendor      = errors.New("portcollection: vendor not supported for write operations")
    ErrUnknownAction          = errors.New("portcollection: unknown port action")
    ErrEmptyInterfaceName     = errors.New("portcollection: interface name is required")
    ErrDescriptionEmpty       = errors.New("portcollection: description is required for description action")
    ErrDescriptionTooLong     = errors.New("portcollection: description exceeds 80 characters")
)

// vendorPortTemplate 内部映射: vendor × action → 命令序列
var vendorPortTemplate = map[models.DeviceVendor]map[PortAction]func(PortTemplateParams) ([]string, error){
    models.VendorHuawei: {
        ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return []string{"shutdown"}, nil },
        ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return []string{"undo shutdown"}, nil },
        ActionDescription:  renderH3CDescription, // 复用 H3C,两者同源
        ActionDot1xEnable:  func(p PortTemplateParams) ([]string, error) { return []string{"dot1x enable"}, nil },
        ActionDot1xDisable: func(p PortTemplateParams) ([]string, error) { return []string{"undo dot1x enable"}, nil },
    },
    models.VendorH3C: {
        // 同 Huawei
    },
    models.VendorRuijie: {
        ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return []string{"shutdown"}, nil },
        ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return []string{"no shutdown"}, nil },
        ActionDescription:  renderRuijieDescription, // Cisco 风格
        ActionDot1xEnable:  renderRuijieDot1xEnable,  // Cisco 风格进 view
        ActionDot1xDisable: renderRuijieDot1xDisable, // 同上
    },
}

// RenderCommand 公共导出
func RenderCommand(vendor models.DeviceVendor, action PortAction, params PortTemplateParams) ([]string, error) {
    if params.InterfaceName == "" {
        return nil, ErrEmptyInterfaceName
    }
    vendorMap, ok := vendorPortTemplate[vendor]
    if !ok {
        return nil, fmt.Errorf("%w: %s", ErrUnsupportedVendor, vendor)
    }
    render, ok := vendorMap[action]
    if !ok {
        return nil, fmt.Errorf("%w: %s (vendor: %s)", ErrUnknownAction, action, vendor)
    }
    return render(params)
}

func renderH3CDescription(p PortTemplateParams) ([]string, error) {
    if p.Description == "" {
        return nil, ErrDescriptionEmpty
    }
    if len(p.Description) > 80 {
        return nil, fmt.Errorf("%w: %d > 80", ErrDescriptionTooLong, len(p.Description))
    }
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        fmt.Sprintf("description %s", p.Description),
    }, nil
}

func renderRuijieDescription(p PortTemplateParams) ([]string, error) {
    if p.Description == "" {
        return nil, ErrDescriptionEmpty
    }
    if len(p.Description) > 80 {
        return nil, fmt.Errorf("%w: %d > 80", ErrDescriptionTooLong, len(p.Description))
    }
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        fmt.Sprintf("description %s", p.Description),
    }, nil
}

func renderRuijieDot1xEnable(p PortTemplateParams) ([]string, error) {
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        "dot1x port-control auto",
    }, nil
}

func renderRuijieDot1xDisable(p PortTemplateParams) ([]string, error) {
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        "no dot1x port-control",
    }, nil
}
```

### 测试骨架（12+ 用例覆盖 15 模板 + 负向）

```go
// vendor_port_template_test.go
package portcollection

import (
    "errors"
    "testing"

    "github.com/xingran-next/xingran-go-backend/internal/models"
    "github.com/stretchr/testify/assert"
)

func TestRenderCommand_VendorActionMatrix(t *testing.T) {
    tests := []struct {
        name     string
        vendor   models.DeviceVendor
        action   PortAction
        params   PortTemplateParams
        expected []string
    }{
        // Huawei (5)
        {"huawei_shutdown", models.VendorHuawei, ActionShutdown,
         PortTemplateParams{InterfaceName: "GE0/0/1"}, []string{"shutdown"}},
        {"huawei_undo_shutdown", models.VendorHuawei, ActionUndoShutdown,
         PortTemplateParams{InterfaceName: "GE0/0/1"}, []string{"undo shutdown"}},
        {"huawei_description", models.VendorHuawei, ActionDescription,
         PortTemplateParams{InterfaceName: "GE0/0/1", Description: "uplink"},
         []string{"interface GE0/0/1", "description uplink"}},
        {"huawei_dot1x_enable", models.VendorHuawei, ActionDot1xEnable,
         PortTemplateParams{InterfaceName: "GE0/0/1"}, []string{"dot1x enable"}},
        {"huawei_dot1x_disable", models.VendorHuawei, ActionDot1xDisable,
         PortTemplateParams{InterfaceName: "GE0/0/1"}, []string{"undo dot1x enable"}},
        // H3C (5) - 与 Huawei 同源
        {"h3c_shutdown", models.VendorH3C, ActionShutdown,
         PortTemplateParams{InterfaceName: "GE0/0/1"}, []string{"shutdown"}},
        // ... 其余 4 个 H3C
        // Ruijie (5) - Cisco 风格
        {"ruijie_shutdown", models.VendorRuijie, ActionShutdown,
         PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"}, []string{"shutdown"}},
        {"ruijie_undo_shutdown", models.VendorRuijie, ActionUndoShutdown,
         PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"}, []string{"no shutdown"}},
        {"ruijie_description", models.VendorRuijie, ActionDescription,
         PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1", Description: "uplink"},
         []string{"interface GigabitEthernet0/0/1", "description uplink"}},
        {"ruijie_dot1x_enable", models.VendorRuijie, ActionDot1xEnable,
         PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"},
         []string{"interface GigabitEthernet0/0/1", "dot1x port-control auto"}},
        {"ruijie_dot1x_disable", models.VendorRuijie, ActionDot1xDisable,
         PortTemplateParams{InterfaceName: "GigabitEthernet0/0/1"},
         []string{"interface GigabitEthernet0/0/1", "no dot1x port-control"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := RenderCommand(tt.vendor, tt.action, tt.params)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}

func TestRenderCommand_EmptyInterfaceName(t *testing.T) {
    _, err := RenderCommand(models.VendorHuawei, ActionShutdown, PortTemplateParams{InterfaceName: ""})
    assert.ErrorIs(t, err, ErrEmptyInterfaceName)
}

func TestRenderCommand_UnsupportedVendor(t *testing.T) {
    _, err := RenderCommand(models.DeviceVendor("cisco"), ActionShutdown, PortTemplateParams{InterfaceName: "GE0/0/1"})
    assert.ErrorIs(t, err, ErrUnsupportedVendor)
}

func TestRenderCommand_UnknownAction(t *testing.T) {
    _, err := RenderCommand(models.VendorHuawei, PortAction("bogus"), PortTemplateParams{InterfaceName: "GE0/0/1"})
    assert.ErrorIs(t, err, ErrUnknownAction)
}

func TestRenderCommand_DescriptionEmpty(t *testing.T) {
    _, err := RenderCommand(models.VendorHuawei, ActionDescription, PortTemplateParams{InterfaceName: "GE0/0/1", Description: ""})
    assert.ErrorIs(t, err, ErrDescriptionEmpty)
}

func TestRenderCommand_DescriptionTooLong(t *testing.T) {
    longDesc := strings.Repeat("x", 81)
    _, err := RenderCommand(models.VendorHuawei, ActionDescription, PortTemplateParams{InterfaceName: "GE0/0/1", Description: longDesc})
    assert.ErrorIs(t, err, ErrDescriptionTooLong)
}
```

</specifics>

<deferred>
## Deferred Ideas

- **Maipu 厂商模板**：本环境无设备，模板留待其他环境（v1.19 OUT-OF-SCOPE）
- **Cisco IOS-XE/XR 模板**：FUTURE-08
- **Description 内容转义**：当前透传（scrapli 文本模式），如发现注入风险再加（D-06 备注）
- **接口名短名/全称归一化**：D-07 备注中提及，Phase 51 实施时核实 scrapligo 期望形态后再回头改 D-07
- **数据库抽象模板**：v1.19 init 锁定"硬编码 Go map 落地为先"，后续 phase 抽象为数据库表（与 v1.19 整体战略一致）

### Reviewed Todos (not folded)
- `operlog-exclude-paths.md`（score 0.2） — RPA 心跳日志白名单，与 Phase 50 范围无关，归 Phase 52 operlog 决策处理

---

*Phase: 50-w1-vendor-templates-unit-tests-vendor-action-command-map*
*Context gathered: 2026-07-06 via /gsd:discuss-phase 50 (轻量讨论 + Claude discretion, 匹配 Phase 3 / Phase 48 模式)*
