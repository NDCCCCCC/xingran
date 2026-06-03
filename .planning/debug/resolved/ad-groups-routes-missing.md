---
slug: ad-groups-routes-missing
status: resolved
trigger: AD域用户组页面返回404错误
created: 2026-05-27
updated: 2026-05-27
---

## 问题描述
域用户组页面调用以下API时返回404:
- `POST /api/v1/ad-domain/groups/list`
- `POST /api/v1/ad-domain/groups/sync-status`

## 症状

### 期望行为
- `/api/v1/ad-domain/groups/list` 应该返回AD用户组列表
- `/api/v1/ad-domain/groups/sync-status` 应该返回同步状态

### 实际行为
- 两个端点都返回 404 Not Found
- 日志显示: `WARN[2026-05-27 19:41:33] Client error ... status_code=404`

### 错误信息
```
POST http://10.62.10.33:9000/api/v1/ad-domain/groups/list 404 (Not Found)
POST http://10.62.10.33:9000/api/v1/ad-domain/groups/sync-status 404 (Not Found)
```

### 时间线
- 在删除独立的 group-mapping 页面并整合功能后出现
- 之前修改了 `ad_domain_router.go` 调用 `SetupGroupSyncRouter`

### 复现步骤
1. 打开域用户组页面
2. 选择AD配置
3. 页面自动调用 `/api/v1/ad-domain/groups/list`

## Current Focus

**假设**: ✅ 确认 - 路由缺失导致404

**下一步**: ✅ 已修复 - 添加缺失的路由

## Evidence
- timestamp: 2026-05-27 19:41:33
  path: ad_domain_router.go:18
  observation: 调用了 `addomainAPI.SetupGroupSyncRouter(r, service)` 但该函数注册的路由路径与前端期望不匹配
  evidence_type: code_review

- timestamp: 2026-05-27 19:45:00
  path: ad_domain_router.go:47-60
  observation: 添加了缺失的 `/groups` 路由组，注册 `list` 和 `sync-status` 端点
  evidence_type: fix_applied

- timestamp: 2026-05-27 19:46:00
  observation: `go build ./internal/api/v1/system` 编译成功，无错误
  evidence_type: build_verification

## Eliminated

## Resolution

**root_cause**: 
`SetupGroupSyncRouter` 注册的路由路径与前端API调用不匹配。前端调用 `POST /groups/list` 和 `POST /groups/sync-status`，但这些路由没有在 `ad_domain_router.go` 中注册。

**fix**:
在 `ad_domain_router.go` 中添加了用户组管理路由组，注册以下端点：
- `POST /groups/list` → `handler.ListGroups`
- `POST /groups/sync-status` → `handler.GetGroupSyncStatus`
- `POST /groups/:id` → `handler.GetGroupDetail`
- `POST /groups/:id/update` → `handler.UpdateGroup`
- `POST /groups/:id/members` → `handler.GetGroupMembers`
- `POST /groups/sync-single` → `handler.SyncSingleGroup`

**verification**:
- ✅ Go 代码编译成功
- ⏳ 需要重启服务器并验证前端页面不再返回 404

**files_changed**:
- `internal/api/v1/system/ad_domain_router.go` (添加了 /groups 路由组)
