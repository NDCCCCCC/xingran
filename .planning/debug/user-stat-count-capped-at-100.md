---
slug: user-stat-count-capped-at-100
status: resolved
trigger: "用户管理页面总用户数、正常用户全部显示 100，但实际用户数有 1196（后扩展到通知/值班池同款统计卡片）"
created: 2026-06-20
updated: 2026-06-20
---

# Debug Session: 用户管理统计卡片显示 100（实际 1196）

## Symptoms

- **Expected**: 统计卡片「总用户数」应显示 ~1196，「正常用户」「禁用用户」显示实际 status 计数。
- **Actual**: 「总用户数」「正常用户」均显示 100（实际 1196）。
- **Reproduction**: 打开用户管理页，看顶部三个统计卡片。
- **Errors**: 无报错（纯数据错误）。

## Root Cause

统计数字由「加载全量用户列表后在内存里 `.length` 计数」得到，而后端把 pageSize 钳制到 **MaxPageSize=100**，导致只返回 100 条 → `allUsers.length` = 100。

**证据链（前端 → 后端）：**

1. **前端** `ruoyi-react-frontend/src/pages/system/user/hooks/useUserData.ts:53-63`
   ```ts
   const result = await post<PageResponse<User>>("/system/users/list", { current: 1, pageSize: 1000 });
   const allUsers = result.data?.list || [];          // ← 只拿到被钳制后的 100 条
   ...
   setStatistics({ total: allUsers.length, ... });     // ← 用 list 长度当总数，丢弃 result.data.total
   ```
   请求 `pageSize:1000` 期望「全量」，但用 `list.length` 当总数，且**丢弃了响应里正确的 `total` 字段**。

2. **Handler** `internal/api/v1/system/user_handler.go:101-103` 调用 `params.GetPagination()` 后覆盖 `params.PageSize`。

3. **GetPagination** `internal/models/system/requests/user_requests.go:44-49`
   ```go
   if pageSize > constants.MaxPageSize { pageSize = constants.MaxPageSize }  // 1000 → 100
   ```

4. **常量** `internal/constants/pagination.go:20`：`MaxPageSize = 100`

5. 结果：1196 > 100，列表只回 100 条 → 统计卡显示 100。

**附注（非本 bug 但相关）：** `internal/services/operations/pagination_helper.go:17` 的运维模块用独立常量 `MaxPageSize = 10000`，故运维模块不受影响——本问题仅限 system 模块的 100 上限。

## Fix (Applied — 3 modules, same pattern)

全部改为后端专用 COUNT 聚合端点，前端直接读取，不再用 list 长度充当总数。

### 1. 用户管理 `POST /system/users/statistics`
返回 `{total, active, inactive}`（status 0=正常/1=停用）。

### 2. 通知公告 `POST /system/notices/statistics`
返回 `{total, published, draft, scheduled}`。**顺带修正状态桶语义**：旧前端 `useNoticeStatistics` 把 `scheduled` 当成 `publishStatus=3`（实为「已撤回」）、把 `draft` 当成 `0||2`（吞掉「定时发布中」）。后端按模型真实语义 `0=草稿/1=已发布/2=定时发布中/3=已撤回` 实现。

### 3. 值班池 `POST /duty/pools/statistics`
返回 `{total, enabled, disabled, totalMembers}`。旧实现最糟——直接用**当前页**（~10 条）list 算统计。`totalMembers` 用子查询 `pool_id IN (SELECT id FROM sys_duty_pool WHERE deleted_at IS NULL)` 统计非软删除池的成员，避免硬编码表名。

## Verification

**已通过的代码级验证：**

- [x] `go build ./...` 通过
- [x] 前端 `npm run type-check` 通过
- [x] 前端 `npm run build` 通过
- [x] 3 个回归测试全绿（均 >100 场景 + 软删除排除）：
  - `internal/services/system/user_statistics_test.go::TestUserStatistics_CountExceedsPageSizeCap` — 150 用户 → total=150/active=100/inactive=50
  - `internal/services/notice_status_statistics_test.go::TestNoticeStatusStatistics_CountExceedsPageSizeCap` — 150 通知 → total=150/published=60/draft=40/scheduled=30（锁定正确状态桶）
  - `internal/services/duty_pool_statistics_test.go::TestDutyPoolStatistics_NotDerivedFromCurrentPage` — 120 池 → total=120/enabled=70/disabled=50/totalMembers=190（软删除池的 5 成员被排除）
- [x] `internal/services/`、`internal/api/v1/system/`、`internal/services/system/`（跳过预存坏测试后）全绿，无回归

**改动文件（共 14 个）：**

后端：
- `internal/services/system/user_service.go`、`internal/api/v1/system/user_handler.go`、`internal/api/v1/system/user_router.go`
- `internal/services/notice_service.go`、`internal/services/system/notice_cache_impl.go`、`internal/api/v1/system/notice_handler.go`、`internal/api/v1/system/notice_router.go`
- `internal/services/duty_pool_service.go`、`internal/services/duty_service.go`、`internal/services/duty/duty_cache_impl.go`、`internal/api/v1/duty/duty_handler.go`、`internal/api/v1/duty/duty_router.go`

前端：
- `ruoyi-react-frontend/src/pages/system/user/hooks/useUserData.ts`
- `ruoyi-react-frontend/src/pages/system/notice/hooks/useNoticeStatistics.ts`、`ruoyi-react-frontend/src/lib/noticeApi.ts`
- `ruoyi-react-frontend/src/pages/duty/pools/index.tsx`、`ruoyi-react-frontend/src/lib/dutyApi.ts`

测试：3 个新增 `*_statistics_test.go`。

**待 UAT（需真实数据环境）：**
- [ ] 1196 用户 / 大量通知 / 多页值班池环境下，三处统计卡片显示正确计数
- [ ] 增删改后卡片刷新正确（前端各 mutation 后已调统计接口）
- [ ] 通知卡片 draft/scheduled 数值变化符合预期（状态桶已修正，与旧版数值会有差异——这是修正，非回归）

**附注（预存问题，与本次无关）：** `internal/services/system/apikey_service_test.go`（TestUpdateAPIKey/TestListAPIKeys 等）与 `role_service_apperrors_test.go::TestRoleService_Create_RoleNameExists` 为 nil 指针 panic，属测试 setup 自身问题，本次未触碰这些文件。
