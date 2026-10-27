---
phase: quick
plan: batch-sync
type: execute
wave: 1
---

# 批量同步优化（BatchSyncUsersToAD）

## 目标
导入后同步从 **~10 分钟**（逐用户串行 + 每用户新建连接）降到 **~10 秒**（复用单连接 + errgroup 并发）。2274 用户场景。

## 决策（CONTEXT，已与用户确认）
1. **并发模型**：单连接 + errgroup 信号量（MaxConcurrentADSync=3）。go-ldap Conn 每请求有 message ID 支持并发 Modify
2. **MoveUser**：复用连接做 ModifyDN（完整同步属性 + OU，导入改部门时自动移动）
3. **路径关系**：保留 `SyncUserUpdateToAD`（单用户编辑场景）+ 新增 `BatchSyncUsersToAD`（导入/全量场景）

## 参考
`SyncManagersToAD`（user_ad_sync_service.go:375）已实现相同模式（ExecuteWithFailover 闭包内 errgroup + 信号量）。BatchSyncUsersToAD 直接复用此结构。

## 任务

### T1: user_ad_sync_service.go 新增 BatchSyncUsersToAD
- `BatchSyncResult` 类型（Total/Synced/Skipped/Failed/Errors，同 ManagerSyncResult 结构）
- `BatchSyncUsersToAD(ctx, userIDs []string) (*BatchSyncResult, error)`：
  1. 查 ADConfig（无启用则返回空 result）
  2. 批量查用户（`id IN ? AND ad_dn IS NOT NULL`，一次 DB）
  3. `ExecuteWithFailover` 闭包内 errgroup + 信号量并发：
     - 每用户：构造 updateReq（从 user 的 deptId/email/phone/nickname）→ `moveUserToNewOU`（部门变）+ `syncUserAttributes`（复用 ldapClient）
  4. 单失败不中断（收集 Errors，g.Go 返 nil）
  5. 成功后 `updateSyncTimestamp`

### T2: user_import_handler.go triggerADSyncAfterImport 改调批量
- 移除逐用户 for 循环 + syncImportedUserWithRetry 重试
- 改调 `h.userADSyncService.BatchSyncUsersToAD(ctx, userIDs)`
- 记录批量结果日志

### T3: 测试
- BatchSyncUsersToAD 单测（参考 manager_sync_test，测试钩子绕过 FailoverClient）

## 文件
- `internal/services/addomain/user_ad_sync_service.go`（新增方法，追加末尾）
- `internal/api/v1/system/user_import_handler.go`（改 triggerADSyncAfterImport）

## 约束
- 不动 ldap_client.go / ad_authenticator.go / account_pool.go（Phase 38）
- 保留 SyncUserUpdateToAD（单用户路径）
