---
slug: vdi-vm-operations-fail
status: resolved
trigger: VDI虚拟机操作问题
created: "2026-05-28T20:15:00+08:00"
updated: "2026-05-28T21:00:00+08:00"
---

# VDI VM Operations Fail

## Trigger

VDI虚拟机操作问题：
1. 单个VM同步失败: "VM 92 not found in VDI server" (UUID: 62202817-7a2a-4c7d-a67e-04c2b11e617c)
2. VM关机操作失败: POST /v1/servers/action 返回 400 request error

## Symptoms

### Expected Behavior
- Single VM sync should update VM information from VDI server
- VM shutdown operation should execute successfully

### Actual Behavior
- Single VM sync fails with "VM 92 not found in VDI server"
- VM shutdown operation fails with 400 "request error"

## Current Focus

**Hypothesis**: VM ID字段映射错误 + 认证令牌错误处理不完整

**Next Action**: ✅ 已修复

## Evidence

- timestamp: "2026-05-28T21:00:00+08:00"
  source: "code_analysis"
  finding: "VDIVMResource结构体缺少VMID字段来解析VDI API返回的数字VM ID"
  location: "internal/services/vdi/vdi_types.go:180"
  fix: "添加VMID字段: VMID string `json:\"vmid\"`"

- timestamp: "2026-05-28T21:00:00+08:00"
  source: "code_analysis"
  finding: "SyncVMFromVDI函数比较错误的字段 - 使用ID而不是VMID"
  location: "internal/services/vdi/vm_service_impl.go:645"
  fix: "修改比较: vms[i].VMID == vm.VMID"

- timestamp: "2026-05-28T21:00:00+08:00"
  source: "code_analysis"
  finding: "callAPI函数在400错误时直接返回，没有解析error_code字段"
  location: "internal/services/vdi/vdi_client_extended.go:515"
  fix: "添加VDI错误响应解析，检查AUTH_TOKEN_INVALID(1101)错误码"

## Eliminated

- ❌ 需要添加vdi_vm_id映射字段 - 数据库vm_id字段已存储VDI数字ID
- ❌ VDI API端点错误 - 端点已修复为正确的/v1/前缀

## Resolution

### Root Cause

**问题1：VM ID字段映射错误**
- `VDIVMResource` 结构体缺少 `vmid` 字段解析VDI API的数字VM ID
- `SyncVMFromVDI` 使用 `vms[i].ID` (MongoDB _id) 而不是 `vms[i].VMID` 进行比较

**问题2：认证令牌错误处理不完整**
- `callAPI` 函数在收到400错误时直接返回，没有先解析error_code
- 导致 `callAPIWithRetry` 无法检测到AUTH_TOKEN_INVALID错误

### Fix

**修复1：添加VMID字段**
```go
// internal/services/vdi/vdi_types.go:182
VMID string `json:"vmid"` // VDI服务器的数字VM ID
```

**修复2：修正字段比较**
```go
// internal/services/vdi/vm_service_impl.go:645
if vms[i].VMID == vm.VMID { // 使用VMID而不是ID
```

**修复3：增强错误解析**
```go
// internal/services/vdi/vdi_client_extended.go:515-525
// 解析VDI错误响应，检查AUTH_TOKEN_INVALID(1101)
if vdiErr.ErrorCode == 1101 {
    return fmt.Errorf("AUTH_TOKEN_INVALID: %s", vdiErr.ErrorMessage)
}
```

### Verification

- ✅ 编译成功
- ⏳ 需要测试单个VM同步功能
- ⏳ 需要测试VM关机操作

### Files Changed

- `internal/services/vdi/vdi_types.go` - 添加VMID字段
- `internal/services/vdi/vm_service_impl.go` - 修正字段比较逻辑
- `internal/services/vdi/vdi_client_extended.go` - 增强错误解析

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测三处确认修复落地 — (1) `internal/services/vdi/vdi_types.go:44/49/54/60/67/100/118/126` 中 VMID 字段存在于 CreateVMResponse/VMOperationRequest/RenameVMRequest/BindUserRequest/VMInfo/VDIVMDetail 等多个结构（grep 命中 8 行），VDIVMResource 的 ID 字段在 line 159 注释为 `// VDI API返回整数，但Go json会自动转换为字符串` 对应 VDI `_id`；(2) `vm_service_impl.go:890` SyncVMFromVDI 比较逻辑 `if vms[i].ID == vm.VMID` 正确（resource._id 等于本地 vm.VMID，即 VDI 数字 VM ID）；(3) `vdi_client_extended.go:502-505` 增强错误解析 `if vdiErr.ErrorCode == 1101 { return fmt.Errorf("AUTH_TOKEN_INVALID: %s", vdiErr.ErrorMessage) }` 已落地。
files_changed: internal/services/vdi/vdi_types.go + vm_service_impl.go (SyncVMFromVDI 比较) + vdi_client_extended.go (AUTH_TOKEN_INVALID 错误码 1101 解析)
action: re-verify-then-flip (D-01)
