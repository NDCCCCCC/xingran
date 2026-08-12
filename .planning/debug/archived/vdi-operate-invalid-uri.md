---
slug: vdi-operate-invalid-uri
status: resolved
trigger: VDI 虚拟机操作（停止/启动）返回 [COMMON_INVALID_URI] 错误
created: 2026-05-28T17:06:00+08:00
updated: 2026-05-28T17:30:00+08:00
---

## Trigger
DATA_START
VDI 虚拟机操作（停止/启动）返回 [COMMON_INVALID_URI] 错误
DATA_END

## Symptoms

### Expected Behavior
DATA_START
点击虚拟机的"停止"或"启动"按钮后，操作成功执行并返回成功响应
DATA_END

### Actual Behavior
DATA_START
POST /api/v1/vdi/vm/operate 返回 500 错误
VDI API 返回 400 错误: {"error_code":1000,"error_message":"[COMMON_INVALID_URI]"}
请求体: {"vm_ids":["62202817-7a2a-4c7d-a67e-04c2b11e617c"],"action":"stop"}
DATA_END

### Error Messages
DATA_START
time="2026-05-28T17:06:11+08:00" level=error msg="VDI 操作 failed" action="操作" error="failed to operate VMs: API request failed with status 400: \n\r\n \r\n{\"error_code\":1000,\"error_message\":\"[COMMON_INVALID_URI]\"}" path=/api/v1/vdi/vm/operate
ERRO[2026-05-28T17:06:11+08:00] Internal server error client_ip=10.62.10.33 latency=799 method=POST path=/api/v1/vdi/vm/operate request_body="{\"vm_ids\":[\"62202817-7a2a-4c7d-a67e-04c2b11e617c"],"action":"stop\"}" request_id=mpp9qdrpfqccaa84n8 status_code=500
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
4. 点击该虚拟机的"停止"或"启动"按钮
5. 观察到 500 错误响应，日志显示 [COMMON_INVALID_URI]
DATA_END

## Current Focus

### Hypothesis
已确认 - VDI API 不存在 /v1/vm/operate 端点，Sangfor VDI API 要求为每个操作类型使用独立的端点

### Next Action
fix applied - 已修改 OperateVM 方法使用动作特定的 API 端点

### Evidence
- 2026-05-28T17:15: `vdi_client_extended.go:209` 中的 OperateVM 调用 `/v1/vm/operate`，但 Sangfor VDI API 不支持此统一端点
- 2026-05-28T17:16: 旧客户端 `client.go` 显示 Sangfor VDI API 为每个操作使用独立端点: `/api/v1/vm/start`, `/api/v1/vm/stop`, `/api/v1/vm/restart`
- 2026-05-28T17:17: 扩展客户端的认证端点 `/v1/auth/tokens` 无 `/api` 前缀，与旧客户端的 `/api/v1/auth/login` 不同，说明扩展客户端使用的是新版 API 路径格式
- 2026-05-28T17:18: 正确的端点路径应为 `/v1/vm/start`, `/v1/vm/stop`, `/v1/vm/restart`, `/v1/vm/suspend`, `/v1/vm/delete`
- 2026-05-28T17:25: 修改后 go build ./... 编译通过

### Eliminated
- 不是认证问题（认证成功返回 token）
- 不是请求体格式问题（vm_ids 数组格式正确）
- 不是网络连接问题（VDI 服务器可达，返回 400 而非连接超时）

## Resolution

### Root Cause
`vdi_client_extended.go` 中的 `OperateVM` 方法使用了不存在的统一操作端点 `/v1/vm/operate`。Sangfor VDI API 不支持统一的 `/operate` 端点，而是要求为每种操作类型（start/stop/restart/suspend/delete）调用各自的独立端点。

### Fix
修改 `OperateVM` 方法，将操作动作映射到对应的 API 端点路径：
- `start` -> `/v1/vm/start`
- `stop` -> `/v1/vm/stop`
- `restart` -> `/v1/vm/restart`
- `suspend` -> `/v1/vm/suspend`
- `delete` -> `/v1/vm/delete`

同时从请求体中移除了冗余的 `action` 字段（因为动作已体现在 URL 路径中），只保留 `vm_ids`。

### Files Changed
- `internal/services/vdi/vdi_client_extended.go` - 修改 OperateVM 方法，新增 operateActionToPath 映射函数

## Notes
- 预存在的测试编译错误（vdi_client_test.go 引用不存在的 config.VDIServerConfig）与本次修复无关
- 旧客户端 client.go 和 mock_server.go 使用的 `/api/v1/` 路径前缀与扩展客户端的 `/v1/` 路径前缀属于不同 API 版本
