# Phase 3: 信息点导入设备端口配置 - Context

**Gathered:** 2026-04-16
**Status:** Ready for planning

<domain>
## Phase Boundary

在 Excel 导入配置中为信息点（infoPoint）添加"所属设备"和"所属端口"两个可选列。导入时通过设备名称匹配 `sys_network_device.device_name` 写入 `device_id`，通过端口名称匹配 `sys_device_port_status.interface_name` 写入 `port_id`。

**Scope:** 仅修改 `excel_config.go` 的 `infoPoint` 配置，添加两列定义。无模型变更、无迁移、无前端改动。

</domain>

<decisions>
## Implementation Decisions

### 匹配策略
- **D-01:** 设备名称精确匹配 `sys_network_device.device_name`（与 workstation 中 deptName/userName 的 Reference 模式一致）
- **D-02:** 端口名称精确匹配 `sys_device_port_status.interface_name`
- **D-03:** 两个字段均为可选（Required: false），匹配失败留空不阻断导入
- **D-04:** 不做端口与设备的级联验证（Out of Scope）

### Claude's Discretion
- Reference 配置格式、列顺序、Header 命名等细节由 Claude 决定
- 端口匹配使用全局查找（不限设备），因 Reference 模式不支持 DependsOn 级联到 device_id → port_id

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Excel 导入架构
- `internal/services/operations/excel_config.go` — 列映射配置，infoPoint 部分是本阶段的修改目标
- `internal/services/operations/reference_resolver.go` — 名称→ID 解析器，已有所有需要的方法

### 数据模型
- `internal/models/operations/infopoint.go` — 信息点模型（已有 DeviceID/DeviceName/PortID/PortName 字段）
- `internal/models/network_device.go` — 网络设备模型（表名 sys_network_device，名称字段 device_name）
- `internal/models/device_port_status.go` — 设备端口模型（表名 sys_device_port_status，端口字段 interface_name）

### 参考模式（已完成的类似实现）
- `excel_config.go` 中 workstation 配置的 `deptName`/`userName` 列 — 同样的 Reference 模式

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `ExcelColumn.Reference` 字段：配置 `"table.field"` 格式即可自动触发名称→ID 解析
- `ReferenceResolver`：已支持批量解析（`ResolveBatch`），按 Reference 类型分组查询
- `OpsInfoPoint` 模型：已有 `DeviceID`/`PortID` 字段（`*string`，可为空）

### Established Patterns
- workstation 配置中 `deptName` → `Reference: "sys_dept.dept_name", DBField: "dept_id"` 是直接的参考模式
- 可选字段：不设 `Required: true`，匹配失败时 reference_resolver 不写入结果，batch_upserter 会跳过该字段

### Integration Points
- `excel_config.go` 第 197-213 行：`infoPoint` 配置块，在 `Columns` 数组中添加新列即可
- 无需修改 `reference_resolver.go`、`excel_service.go`、`batch_upserter.go` 或任何前端代码

</code_context>

<specifics>
## Specific Ideas

No specific requirements — standard Reference pattern application. The implementation is mechanically adding two ExcelColumn entries to the existing infoPoint config.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-信息点导入设备端口配置*
*Context gathered: 2026-04-16*
