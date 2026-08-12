---
status: complete
slug: 260622-u8n-ad-manager-leader-ad
completed_at: 2026-06-22
---

# AD Manager 同步（从 sys_dept.leader）

## 目标
从 `sys_dept.leader` 批量同步用户 AD `manager` 属性，解决部门 leader 变更后无法批量同步到下属用户 AD manager 字段、只能手动逐条维护的问题。

## 实现摘要

### Task 1: Service 层 — `internal/services/addomain/user_ad_sync_service.go`
- `ManagerSyncResult` 结构（Total / Synced / Skipped / Failed / Errors）
- `splitAncestorIDs`：解析 ancestors 字段，过滤 `"0"` 根占位符 / 空值 / 自身，返回 nil（无匹配时）
- `resolveLeaderByAncestors`：当前部门 leader，无则递归祖先链（一次性批量查祖先部门 leader，避免 N+1；深度上限 `maxLeaderDepth=20`）
- `SyncManagersToAD(ctx, userIDs)`：
  - 全量同步所有 ad_dn 非空用户（userIDs 非空时仅限这些）
  - **复用单个 LDAP 连接**（一次 connect/bind，遍历所有 UpdateUserAttribute），区别于 `SyncUserUpdateToAD` 每用户一个连接
  - `errgroup` + `constants.MaxConcurrentADSync=3` 信号量限并发
  - 单失败不中断批量（收集到 Errors，g.Go 始终返回 nil）
  - 测试钩子 `updateUserAttributeFn` 闭包字段（非 nil 时绕过真实 AD，单测用 sqlite 内存 DB）

### Task 2: API 层
- 路由 `POST /system/users/sync-managers`（`user_router.go`，注册在 `/:id` 之前避免参数路由捕获）
- `UserHandler.SyncManagers`（`user_import_handler.go` 末尾）
- operlog：`OperTypeSync=14`，模块"用户管理"，记录聚合统计（不记 errors 明细，避免 username 泄露）

### Task 3: 前端 leader 显示 — **已满足，无需改动**
- `department_service.go::fillLeaderInfo` 已 JOIN sys_user 填充 `LeaderName` / `LeaderUsername`
- 部门列表 `columns.tsx:48` 已渲染 `${leaderName}（${leaderUsername}）`

## 偏差说明
PLAN artifacts 预期 `SyncManagers` 放 `user_handler.go`，实际放 `user_import_handler.go` 末尾 —— 该文件已聚合 AD 同步逻辑（`triggerADSyncAfterImport` + `syncImportedUserWithRetry`），复用已 import 的 `operlog` / `response` / `applogger` / `fmt`，避免新增文件与重复 import。

## 测试结果
12 个单元测试全过：
```
go test ./internal/services/addomain/ -run 'TestSplitAncestorIDs|TestResolveLeaderByAncestors|TestSyncManagersToAD'
```
覆盖：splitAncestorIDs（4 case）、resolveLeaderByAncestors（当前部门/父递归/全无）、SyncManagersToAD（基础同步/父递归/自指跳过/leader 无 ad_dn/失败不中断/无 ADConfig/userIDs 过滤/无部门跳过）。

整体验证：
- `go build github.com/xingran-next/xingran-go-backend/...` ✓（无连锁错误）
- `go test ./internal/services/addomain/` ✓（7.7s，全包无回归）

## Task 3 端到端验证清单（需真实 AD，用户手动）
- [ ] AD 服务账号凭据有效（当前 LDAP error 49 `data 775` 账号锁定，环境问题非代码 bug）
- [ ] leader 用户有 ad_dn（实测 0/8 leader 有 ad_dn，数据治理问题）
- [ ] 调 `POST /system/users/sync-managers`，验证返回 `synced > 0`
- [ ] AD 中抽检用户 `manager` 属性已更新为 leader 的 DN
