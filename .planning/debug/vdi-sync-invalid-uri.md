---
slug: vdi-sync-invalid-uri
status: investigating
trigger: VDI虚拟机同步失败 - 使用错误的API端点导致 [COMMON_INVALID_URI]
created: 2026-05-28T18:45:00+08:00
---

## Trigger
DATA_START
VDI虚拟机同步失败 - 使用错误的API端点导致 [COMMON_INVALID_URI]
DATA_END

## Symptoms

### Expected Behavior
DATA_START
VDI虚拟机同步任务应该成功连接到VDI服务器，获取资源组列表，然后同步虚拟机数据。
DATA_END

### Actual Behavior
DATA_START
VDI同步任务失败，日志显示调用 `/api/v1/resources_group` 端点返回400错误和 `[COMMON_INVALID_URI]`。

错误日志：
```
[VDI API DEBUG] Calling: GET https://10.62.0.79:6060/api/v1/resources_group
[VDI API DEBUG] Error response: status=400, body={"error_code":1000,"error_message":"[COMMON_INVALID_URI]"}
```
DATA_END

### Error Messages
DATA_START
- `"error_code":1000,"error_message":"[COMMON_INVALID_URI]"`
- "failed to list resource groups: API request failed with status 400"
DATA_END

### Timeline
DATA_START
2026-05-28 18:41:48 - 定时任务执行时发现同步失败
DATA_END

### Reproduction Steps
DATA_START
1. VDI虚拟机数据同步定时任务执行
2. 调用 `/api/v1/resources_group` 端点获取资源组列表
3. 收到 400 错误和 [COMMON_INVALID_URI] 响应
DATA_END

## Current Focus

### Hypothesis
VDI同步代码使用了错误的API端点路径前缀。根据测试脚本验证：
- `/api/v1/resources_group` 返回 [COMMON_INVALID_URI]
- `/v1/servers` 端点可用（返回200）
- VDI API 路径前缀应该是 `/v1/` 而不是 `/api/v1/`

### Next Action
检查同步代码中所有VDI API端点路径，修复使用错误前缀的端点。

### Evidence
- 2026-05-28T18:45: 测试脚本 `vdi_probe.go` 验证 `/v1/servers` 端点可用
- 2026-05-28T18:45: 测试脚本验证 `/api/v1/resources_group` 不可用
- 2026-05-28T18:41: 同步日志显示调用 `/api/v1/resources_group` 失败

### Eliminated
- 不是认证问题（token有效，长度454）
- 不是网络问题（能连接到VDI服务器）
- 不是权限问题（端点不存在而非权限拒绝）
