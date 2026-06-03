---
slug: vdi-sync-auth-token-invalid
status: resolved
trigger: 虚拟机列表页面，对单个虚拟机点击同步按钮时报错
created: 2026-05-28T16:50:00+08:00
updated: 2026-05-28T17:30:00+08:00
---

## Trigger
DATA_START
虚拟机列表页面，对单个虚拟机点击同步按钮时报错
DATA_END

## Symptoms

### Expected Behavior
DATA_START
成功同步虚拟机信息。VDI API 返回最新虚拟机数据并更新数据库。
DATA_END

### Actual Behavior
DATA_START
POST /api/v1/vdi/vm/62202817-7a2a-4c7d-a67e-04c2b11e617c/sync 返回 500 错误
日志显示: "failed to fetch VMs from VDI: API request failed with status 400"
VDI API 错误响应: {"error_code":1101,"error_message":"[AUTH_TOKEN_INVALID]"}
DATA_END

### Error Messages
DATA_START
time="2026-05-28T16:50:17+08:00" level=error msg="VDI 同步失败" action="同步" error="failed to fetch VMs from VDI: API request failed with status 400: \n\r\n \r\n{\"error_code\":1101,\"error_message\":\"[AUTH_TOKEN_INVALID]\"}" path=/api/v1/vdi/vm/62202817-7a2a-4c7d-a67e-04c2b11e617c/sync
ERRO[2026-05-28T16:50:17+08:00] Internal server error client_ip=10.62.10.33 latency=8844 method=POST path=/api/v1/vdi/vm/62202817-7a2a-4c7d-a67e-04c2b11e617c/sync request_body="{}" request_id=mpp95s2m2p4ndk7oc25 status_code=500
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
5. 观察到 500 错误响应
DATA_END

## Current Focus

### Hypothesis
已确认 - AUTH_TOKEN_INVALID 重试逻辑缺失导致同步失败

### Next Action
已完成 - 修复已应用

### Evidence
- VDI AuthManager.Authenticate() 缓存 token 到数据库，过期时间设为 23 小时
- VDI 服务器端可能在 23 小时前就失效了 token（服务器重启、session 过期等）
- ListResourceGroups() 方法有 AUTH_TOKEN_INVALID 重试逻辑（调用 ClearTokenCache + 重新认证）
- ListResourceServers() 和所有其他 API 调用方法缺少相同的重试逻辑
- SyncVMFromVDI -> getClient -> ListResourceServers 触发了无重试的路径

### Eliminated
- 密码解密问题（TestConnection 可以工作说明凭据正确）
- 网络连接问题（能收到 400 响应说明网络畅通）

## Resolution

### Root Cause
VDI API 客户端在收到 AUTH_TOKEN_INVALID (error_code 1101) 错误时，只有 ListResourceGroups() 一个方法实现了"清除缓存 token 并重新认证"的重试逻辑。其他所有 API 调用方法（包括 SyncVMFromVDI 使用的 ListResourceServers）在 token 被 VDI 服务器端失效后直接返回错误，不会尝试重新获取 token。

当 VDI 服务器端的 token 在 23 小时过期前被失效（服务器重启、session 被清除等），客户端仍使用缓存的旧 token 发送请求，导致 400 AUTH_TOKEN_INVALID 错误。

### Fix
在 vdiClientExtendedImpl 中新增 callAPIWithRetry() 方法，集中处理 AUTH_TOKEN_INVALID 的重试逻辑：检测到该错误时自动清除 token 缓存、重新认证、并使用新 token 重试 API 调用。

将所有 API 调用方法（GetVM, ListVMs, GetUserVMs, OperateVM, ConfigIP, RenameVM, BindUser, GetAvailableUsers, ListResourceGroups, ListResources, ListResourceServers）从直接调用 callAPI() 改为调用 callAPIWithRetry()。

移除了 ListResourceGroups() 中之前内联的重复重试逻辑，统一使用新的集中方法。

### Files Changed
- internal/services/vdi/vdi_client_extended.go

## Notes
- VDI API 返回 error_code 1101 表示认证令牌无效
- 需要确认令牌是否已过期、配置错误或未正确传递
- 可能涉及 VDI 配置中的认证凭据设置
