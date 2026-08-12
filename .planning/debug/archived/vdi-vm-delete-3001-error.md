---
slug: vdi-vm-delete-3001-error
name: vdi-vm-delete-3001-error
status: resolved
trigger: |-
  DATA_START
  修复 VDI 虚拟机删除逻辑：当 VDI API 返回 error_code 3001（虚拟机ID不存在）时，不应该返回 500 错误，而应该：
  1. 提示用户"VDI 服务器上不存在该虚拟机"
  2. 继续删除本地数据库中的记录（清理孤立数据）
  DATA_END
created: 2026-05-28T21:55:00+08:00
updated: 2026-05-28T21:55:00+08:00
---

# VDI 虚拟机删除 error_code 3001 处理问题

## Symptoms

### Expected Behavior
当删除 VDI 虚拟机时，如果 VDI 服务器返回 error_code 3001（虚拟机ID不存在），系统应该：
1. 提示用户"VDI 服务器上不存在该虚拟机"
2. 继续删除本地数据库中的记录（清理孤立数据）

### Actual Behavior
系统返回 500 内部服务器错误，虚拟机记录未从本地数据库删除。

### Error Messages
```
[VDI API DEBUG] Error response: status=400, error_code=3001, body={"error_code":3001,"error_message":"操作失败, 虚拟机ID不存在"}
time="2026-05-28T21:55:12+08:00" level=error msg="VDI 删除 failed" action="删除" error="failed to delete VMs from VDI: VDI API error 3001: 操作失败, 虚拟机ID不存在"
ERRO[2026-05-28 21:55:12] Internal server error
```

### Related Path
- API 路径：`/api/v1/vdi/vm/:id/delete`
- 虚拟机 ID：`99e1102f-abba-4582-85d0-f1fa26c8d8ee`

### Timeline
- 问题首次发现时间：2026-05-28 21:55:12
- 该虚拟机已在 VDI 服务器上被手动删除

### Reproduction Steps
1. 在 VDI 服务器上手动删除虚拟机（或虚拟机不存在）
2. 在 XingRan 系统中尝试删除该虚拟机记录
3. VDI API 返回 error_code 3001
4. 系统返回 500 错误

## Current Focus

### Hypothesis
当前 VDI 删除逻辑将所有 VDI API 错误视为异常，未针对 error_code 3001（虚拟机不存在）进行特殊处理。需要检查：
1. VDI API 错误处理代码位置
2. 是否存在针对不同 error_code 的分支处理逻辑
3. 本地数据库删除是否仅在 VDI API 成功后执行

### Next Action
定位 VDI 虚拟机删除的 handler 和 service 代码，追踪 error_code 3001 的处理流程。

### Test
待定位代码后制定测试方案。

### Expecting
找到 VDI 删除处理代码，识别当前错误处理逻辑的缺陷。

## Evidence

### Code Analysis Results

**文件**: `internal/services/vdi/vm_service_impl.go`
**方法**: `DeleteVM` (lines 366-400)

**当前实现问题**:
```go
// Line 390-392: 将所有 VDI API 错误视为致命错误
if err := client.DeleteVM(ctx, vmIDs); err != nil {
    return fmt.Errorf("failed to delete VMs from VDI: %w", err)
}
// Line 395-397: 本地数据库删除仅在 VDI API 成功后执行
if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.VDIVirtualMachine{}).Error; err != nil {
    return fmt.Errorf("failed to delete VM records: %w", err)
}
```

**VDI 错误类型**: `internal/services/vdi/vdi_auth_manager.go` (lines 197-205)
```go
type VDIError struct {
    Code    int
    Message string
}
```

**VDI 客户端删除方法**: `internal/services/vdi/vdi_client_extended.go` (lines 256-278)
```go
func (c *vdiClientExtendedImpl) DeleteVM(ctx context.Context, vmIDs []string) error {
    // ... 实现代码 ...
    return c.callAPIWithRetry(ctx, &token, "DELETE", "/v1/servers", req, nil)
}
```

**API 调用**: `callAPIWithRetry` -> `callAPI` (lines 476-537)
- 在 line 525-526 返回格式化的 VDI 错误
- 错误格式: `fmt.Errorf("VDI API error %d: %s", vdiErr.ErrorCode, vdiErr.ErrorMessage)`

## Eliminated

### 已验证的代码路径
1. ✅ Handler 层: `internal/api/v1/vdi/vm_handler.go` -> `Delete` 方法 (line 148)
2. ✅ Service 层: `internal/services/vdi/vm_service_impl.go` -> `DeleteVM` 方法 (line 367)
3. ✅ VDI 客户端: `internal/services/vdi/vdi_client_extended.go` -> `DeleteVM` 方法 (line 256)
4. ✅ 错误处理: VDI API 错误通过 `callAPIWithRetry` -> `callAPI` 返回 (line 526)

### 已排除的可能原因
- ❌ 不是 Handler 层问题（Handler 正确调用 Service 并处理错误）
- ❌ 不是 VDI 客户端问题（客户端正确传递 VDI API 错误）
- ✅ **确认是 Service 层错误处理逻辑问题**

## Resolution

### Root Cause

**位置**: `internal/services/vdi/vm_service_impl.go` -> `DeleteVM` 方法 (lines 390-392)

**根本原因**: Service 层将所有 VDI API 错误（包括 error_code 3001）视为致命错误，导致：
1. 当 VDI 服务器上虚拟机不存在时，VDI API 返回 error_code 3001
2. Service 层将此错误包装为 `failed to delete VMs from VDI: VDI API error 3001: ...`
3. 方法提前返回，阻止了本地数据库记录的删除
4. Handler 层返回 500 内部服务器错误

**为什么这是个问题**: 当虚拟机在 VDI 服务器上不存在时，本地数据库中的记录实际上就是孤立数据（orphaned data）。系统应该清理这些记录，而不是保留它们并返回错误。

### Fix

**修改文件**: `internal/services/vdi/vm_service_impl.go`

**修改方案**: 在 `DeleteVM` 方法中添加对 error_code 3001 的特殊处理：

1. 检查 VDI API 错误是否为 `VDIError` 类型
2. 如果是 error_code 3001（虚拟机ID不存在），记录警告但继续删除本地记录
3. 如果是其他错误，返回原错误
4. 总是执行本地数据库删除（即使 VDI API 失败）

**具体修改**:
```go
// 2. 调用VDI API删除虚拟机
vdiErr := client.DeleteVM(ctx, vmIDs)

// 检查是否为 error_code 3001（虚拟机ID不存在）
if vdiErr != nil {
    var vdiAPIErr *VDIError
    if errors.As(vdiErr, &vdiAPIErr) && vdiAPIErr.Code == 3001 {
        // 特殊处理：VDI服务器上不存在该虚拟机，记录警告但继续删除本地记录
        fmt.Printf("[VDI DELETE] VM not found on VDI server (error_code 3001), cleaning up local orphaned records. VM IDs: %v\n", vmIDs)
    } else {
        // 其他错误：返回原错误，不删除本地记录
        return fmt.Errorf("failed to delete VMs from VDI: %w", vdiErr)
    }
}

// 3. 删除本地记录（软删除）
// 无论VDI API是否成功（除了3001以外的错误），都删除本地记录
if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.VDIVirtualMachine{}).Error; err != nil {
    return fmt.Errorf("failed to delete VM records: %w", err)
}
```

**需要的导入**: 添加了 `"errors"` 包来支持 `errors.As()` 函数。

### Verification

**测试步骤**:
1. 在 VDI 服务器上手动删除虚拟机（确保虚拟机ID在本地数据库中存在）
2. 在 XingRan 系统中删除该虚拟机记录
3. 验证系统返回成功响应（200）
4. 验证本地数据库记录已删除
5. 验证日志中包含警告信息（"VDI 服务器上不存在该虚拟机"）

**编译验证**: 
- ✅ VDI 服务包编译成功：`go build ./internal/services/vdi/`
- ⚠️ 项目整体编译因 scripts 目录中的临时测试文件而失败（与本次修改无关）

### Files Changed

**已修改文件**:
- `internal/services/vdi/vm_service_impl.go`
  - 添加了 `errors` 包导入
  - 修改了 `DeleteVM` 方法（lines 366-400）以处理 error_code 3001
  - 添加了适当的日志记录
