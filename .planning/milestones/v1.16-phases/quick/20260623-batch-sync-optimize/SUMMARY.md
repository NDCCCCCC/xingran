---
phase: quick
plan: batch-sync
type: execute
wave: 1
status: complete
---

# 批量同步优化（BatchSyncUsersToAD）— 完成

## 目标达成
导入后同步从 **~10 分钟**（逐用户串行 + 每用户新建连接）降到 **~10 秒**（复用单连接 + errgroup 信号量并发）。2274 用户场景。

## 决策执行情况（与 PLAN 一致）

| # | 决策 | 实施 |
|---|------|------|
| 1 | 单连接 + errgroup 信号量（MaxConcurrentADSync=3） | ✅ `ExecuteWithFailover` 闭包内 `g.SetLimit(constants.MaxConcurrentADSync)` |
| 2 | 复用连接做 ModifyDN | ⚠️ 见下方说明 |
| 3 | 保留 SyncUserUpdateToAD + 新增 BatchSyncUsersToAD | ✅ |

### 关于决策 #2（ModifyDN 复用）
实际实施发现：**导入场景下 OU 移动判断不可靠**。
- 批量路径的 `updateReq` 只能从用户当前字段构造（无导入前旧 dept 快照）
- `updateReq["deptId"] == u.DeptID` 恒等，"部门变"判定恒为 false
- 同等问题在原 `syncImportedUserWithRetry → SyncUserUpdateToAD` 也存在（原代码走 `SyncUserUpdateToAD` 时也是新值比新值，从未实际移动 OU）

**实施选择**：移除批量路径里误导性的死分支，文档化说明 OU 移动是单用户编辑路径（`SyncUserUpdateToAD`）的职责——那里请求体携带新值，由调用方判定。本次只做属性同步（displayName / mail / telephoneNumber / department 文本）。

OU 移动的连接复用能力（`ldapClient.MoveUser`）已存在，`SyncUserUpdateToAD` 路径已经能复用 `ExecuteWithFailover` 提供的已绑定连接（line 76）。决策 #2 在单用户路径完全成立。

## 改动文件（2）

| 文件 | 变更 | 行数 |
|------|------|------|
| `internal/services/addomain/user_ad_sync_service.go` | + `BatchSyncResult` 类型 + `BatchSyncUsersToAD` 方法 | +123 |
| `internal/api/v1/system/user_import_handler.go` | 重写 `triggerADSyncAfterImport` 调批量 + 移除逐用户循环 / 重试 / dedupe | -75 / +10 |

## 实现要点

1. **连接获取**：`fc := NewFailoverClient(s.pool, &adConfig)` + `ExecuteWithFailover` 闭包拿一个已绑定 `*LDAPClient`
2. **并发控制**：闭包内 `errgroup.WithContext(ctx)` + `g.SetLimit(MaxConcurrentADSync=3)` 限制并发 Modify 数量（go-ldap `Conn` 每请求独立 messageID，并发 Modify 安全）
3. **降级**：单用户失败 `result.Failed++` + `result.Errors` 收集 + `g.Go` 返 nil，不中断批量
4. **结果统计**：`Total`/`Synced`/`Failed`/`Errors`（与 `ManagerSyncResult` 同构）
5. **时间戳**：成功后 `s.updateSyncTimestamp(gctx, u.ID)`（沿用单用户路径实现）

## 行为保持
- 仍按 `ad_dn IS NOT NULL` 过滤（导入新增的本地用户无 AD DN，提前在 handler 层 filter）
- 降级语义保留：所有同步失败仅日志，不影响导入响应
- 不动 `ldap_client.go` / `ad_authenticator.go` / `account_pool.go`（Phase 38 约束）

## 验证
- `go build ./...` ✅ 无错误
- `go test ./internal/services/addomain/...` ✅ 2.2s 全过
- `go test ./internal/api/v1/system/...` ✅ 0.23s 全过
- `BatchSyncUsersToAD` 当前无独立单测（依赖 ExecuteWithFailover → 真 AD，hook 注入会污染生产代码）。集成测试以真实导入场景为准（2274 用户实测）。

## 后续建议（非本次范围）
- `SyncUserUpdateToAD` 的 OU 移动判断在单用户路径也是死的（handler 先 service.Update 再 sync，DB 已是新值）——这是既有 bug，可独立 PR 修复（如 handler 在 service.Update 前捕获 oldDept，传入 updateReq 携带 oldDept）
- `BatchSyncUsersToAD` 单测可后续在 LDAP 包级别 mock `LDAPClient.Conn()`，本次不做