---
slug: vdi-sync-json-unmarshal-error
status: resolved
trigger: VDI 虚拟机同步时 JSON 解析失败 - is_enable_group_policy 字段类型不匹配
created: 2026-05-28T17:06:00+08:00
updated: 2026-05-28T17:30:00+08:00
---

## Trigger
DATA_START
VDI 虚拟机同步时 JSON 解析失败 - is_enable_group_policy 字段类型不匹配
DATA_END

## Symptoms

### Expected Behavior
DATA_START
同步成功完成。VDI API 响应正确解析并更新虚拟机信息。
DATA_END

### Actual Behavior
DATA_START
POST /api/v1/vdi/vm/62202817-7a2a-4c7d-a67e-04c2b11e617c/sync 返回 500 错误
日志显示: "failed to fetch VMs from VDI: decode response failed: json: cannot unmarshal bool into Go struct field VDIVMResource.data.data.is_enable_group_policy of type string"
VDI API 返回布尔值，但 Go 结构体 VDIVMResource 中的 is_enable_group_policy 字段定义为 string 类型
DATA_END

### Error Messages
DATA_START
time="2026-05-28T17:06:16+08:00" level=error msg="VDI 同步 failed" action="同步" error="failed to fetch VMs from VDI: decode response failed: json: cannot unmarshal bool into Go struct field VDIVMResource.data.data.is_enable_group_policy of type string" path=/api/v1/vdi/vm/62202817-7a2a-4c7d-a67e-04c2b11e617c/sync
ERRO[2026-05-28T17:06:16+08:00] Internal server error client_ip=10.62.10.33 latency=1224 method=POST path=/api/v1/vdi/vm/62202817-7a2a-4c7d-a67e-04c2b11e617c/sync request_body="{}" request_id=mpp9qhs8pntcisosdx status_code=500
DATA_END

### Timeline
DATA_START
不确定 - 不清楚这个功能之前是否正常工作过
DATA_END

### Reproduction Steps
DATA_START
1. 登录系统
2. 进入虚拟机列表页面
3. 找到任意虚拟机
4. 点击该虚拟机的"同步"按钮
5. 观察到 500 错误响应，日志显示 JSON 解析失败
DATA_END

## Current Focus

### Hypothesis
已确认

### Next Action
已修复

### Evidence
- 2026-05-28T17:30: 检查 `internal/services/vdi/vdi_types.go` 第214行，`IsEnableGroupPolicy` 字段定义为 `string` 类型
- 2026-05-28T17:30: VDI API 返回 `is_enable_group_policy` 为 bool 类型（从错误日志确认）
- 2026-05-28T17:30: 搜索整个 `internal/` 目录，`IsEnableGroupPolicy` 仅在结构体定义处引用，无下游代码访问该字段
- 2026-05-28T17:30: 检查 `saveOrUpdateVM` 方法，确认该字段从未被持久化到数据库或用于业务逻辑
- 2026-05-28T17:30: 修复后 `go build ./...` 编译通过

### Eliminated
无

## Resolution

### Root Cause
`VDIVMResource` 结构体中 `IsEnableGroupPolicy` 字段类型定义为 `string`，但深信服 VDI API 实际返回的是 `bool` 类型值。`json.Unmarshal` 在反序列化时因类型不匹配而失败，导致同步操作返回 500 错误。

### Fix
将 `internal/services/vdi/vdi_types.go` 第214行 `IsEnableGroupPolicy` 的类型从 `string` 改为 `bool`。
该字段仅在结构体定义处引用，无下游代码依赖其类型，因此修改无副作用。

变更文件: `internal/services/vdi/vdi_types.go` (1行)
变更内容: `IsEnableGroupPolicy string` -> `IsEnableGroupPolicy bool`

### Verification
- `go build ./...` 编译通过
- 该字段无下游使用，不影响其他功能

## Notes
- JSON 解析错误: cannot unmarshal bool into string
- 问题字段: VDIVMResource.data.data.is_enable_group_policy
- VDI API 返回布尔值，Go 结构体期望字符串类型
- 同一结构体中 `IsClient` 字段已正确使用 `bool` 类型（第211行），说明 VDI API 中 bool 字段是常见的
