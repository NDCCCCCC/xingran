---
slug: vdi-sync-invalid-uri
status: resolved
trigger: VDI虚拟机同步失败 - 使用错误的API端点导致 [COMMON_INVALID_URI]
created: 2026-05-28T18:45:00+08:00
updated: 2026-06-26
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

## Phase 41 Closure (2026-06-26)

**复测:** VDI API 端点路径修复已落地。
- `internal/services/vdi/vdi_client_extended.go:347-359` `ListResourceGroups` 方法使用 `GET /v1/resources_group`(**正确路径**,而非 `/api/v1/resources_group`),端点注释明确"VDI API endpoint: GET /v1/resources_group"。
- `vdi_client_extended.go:534` `SyncVMsFromVDIByServer` 调用 `ListResourceGroups` 已就位。

**根因复述:** .md 报告的 `/api/v1/resources_group` 错误端点源自早期 VDI client 误用 v1 前缀,通过测试脚本 `vdi_probe.go` 确认正确前缀是 `/v1/`,已在生产代码中修正。

**Phase 41 验证:** `go build ./...` 退出 0(本 plan 未触发任何 .go 改动)。

### won't_fix_reason (D-02)
复测确认 VDI resources_group 端点路径已修正为 `/v1/resources_group`(vdi_client_extended.go:359),代码层修复完整,本 plan 复测证据即翻 resolved。
action: wontfix (D-02,复测发现已落地型)
verification: 复测 `internal/services/vdi/vdi_client_extended.go:347-359` ListResourceGroups 使用 `/v1/resources_group`,go build ./... 退出 0
