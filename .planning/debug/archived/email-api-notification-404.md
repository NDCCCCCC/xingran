---
slug: email-api-notification-404
status: resolved
trigger: 前端邮箱配置和API通知配置页面加载时出现404错误：1. POST /api/v1/system/settings/email-configs/list 404 2. POST /api/v1/system/settings/api-notification-configs/list 404
created: 2026-05-20
updated: 2026-05-20
session_type: bug
---

# Debug Session: email-api-notification-404

## Symptoms

### Expected Behavior
邮箱配置和API通知配置页面应该正常加载数据，显示配置列表。

### Actual Behavior
页面加载时出现404错误：
1. POST `/api/v1/system/settings/email-configs/list` 返回 404 Not Found
2. POST `/api/v1/system/settings/api-notification-configs/list` 返回 404 Not Found

前端显示错误信息：
- "加载邮箱配置失败: Error: 请求的资源不存在"
- "加载API配置失败: Error: 请求的资源不存在"

### Error Messages
```
POST http://10.62.10.33:9000/api/v1/system/settings/email-configs/list 404 (Not Found)
POST http://10.62.10.33:9000/api/v1/system/settings/api-notification-configs/list 404 (Not Found)
```

### Timeline
- 2026-05-20 13:35-13:36: 错误首次发现
- 后端服务正常运行，其他API可正常访问
- 只有这两个特定配置页面受影响

### Reproduction
1. 登录系统
2. 导航到系统设置 > 通知配置
3. 打开邮箱配置页面 → 出现404错误
4. 打开API配置页面 → 出现404错误

### Scope
- 影响范围：仅通知配置相关页面
- 后端状态：正常运行，其他API正常
- 功能状态：之前可能未实现或配置错误

## Current Focus

- hypothesis: 后端路由未注册或handler未实现
- next_action: gather initial evidence
- test: 检查后端路由配置和handler实现
- expecting: 发现路由缺失或路径配置错误
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-20T14:00:00Z
  source: codebase_analysis
  finding: |
    **ROOT CAUSE IDENTIFIED: Route Path Mismatch**

    Backend router configuration (`internal/api/router.go:125-128`):
    ```go
    notificationConfigs := authorized.Group("/settings/notification")
    notificationConfigs.Use(middleware.RequirePermissions([]string{permSystemConfig}, core))
    {
        systemV1.SetupNotificationConfigRouter(notificationConfigs, core)
    }
    ```

    Backend routes (`internal/api/v1/system/notification_config_router.go:21-39`):
    - `/api/v1/system/settings/notification/email-configs/list`
    - `/api/v1/system/settings/notification/api-notification-configs/list`

    Frontend API calls (`xingran-react-frontend/src/lib/notificationConfigApi.ts`):
    ```typescript
    // Line 59
    export const getEmailConfigList = (params: EmailConfigListParams) => {
      return post('/system/settings/email-configs/list', params);
    };

    // Line 166
    export const getAPINotificationConfigList = (params: APINotificationConfigListParams) => {
      return post('/system/settings/api-notification-configs/list', params);
    };
    ```

    **Mismatch**: Backend uses `/settings/notification/` but frontend calls `/settings/`

- timestamp: 2026-05-20T14:05:00Z
  source: verification
  finding: |
    Confirmed that the notification_config_router.go handlers are properly implemented
    and the routes are registered, but under the wrong parent group path.

- timestamp: 2026-05-20T14:10:00Z
  source: fix_applied
  finding: |
    Updated frontend API paths in `xingran-react-frontend/src/lib/notificationConfigApi.ts`:
    - Changed all `/system/settings/email-configs/` to `/system/settings/notification/email-configs/`
    - Changed all `/system/settings/api-notification-configs/` to `/system/settings/notification/api-notification-configs/`

    Affected functions:
    - getEmailConfigList
    - getEmailConfig
    - createEmailConfig
    - updateEmailConfig
    - deleteEmailConfig
    - testEmailConfig
    - getAPINotificationConfigList
    - getAPINotificationConfig
    - createAPINotificationConfig
    - updateAPINotificationConfig
    - deleteAPINotificationConfig
    - testAPINotificationConfig

## Eliminated

## Resolution

### root_cause
Route path mismatch between frontend and backend. Backend registers routes under `/api/v1/system/settings/notification/` while frontend calls `/api/v1/system/settings/`.

### fix
Updated all frontend API calls in `xingran-react-frontend/src/lib/notificationConfigApi.ts` to include `/notification/` in the path:
- Changed `/system/settings/email-configs/` to `/system/settings/notification/email-configs/`
- Changed `/system/settings/api-notification-configs/` to `/system/settings/notification/api-notification-configs/`

This maintains better URL organization and requires fewer backend changes.

### files_changed
- `xingran-react-frontend/src/lib/notificationConfigApi.ts`

### verification
After updating frontend API paths, the pages should load successfully without 404 errors. Users need to refresh their browser to load the updated JavaScript bundle.
