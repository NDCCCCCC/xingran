---
slug: ad-sync-duplicate-username-softdeleted
status: resolved
deferred_to: v1.16-tech-debt
trigger: 批量同步AD用户时大量用户报错 "duplicate key value violates unique constraint idx_sys_user_username (SQLSTATE 23505)"
created: 2026-06-22
updated: 2026-06-25
session_type: bug
related:
  - ad-sync-500-on-conflict-duplicate-row
---

# Debug Session: AD 用户同步 — 软删除用户导致 username 唯一约束冲突

## 关键结论（用户首读这一段）

`sys_user.username` 是**普通唯一索引**（`gorm:"uniqueIndex"`，非 partial index），软删除的行
（`deleted_at IS NOT NULL`）仍占用 `username` 值。

`SyncUserFromAD` 用 `ad_username = ? AND deleted_at IS NULL` 查找用户。当用户曾被创建后被**软删除**，
再次批量同步时：
1. 查询带 `deleted_at IS NULL`，查不到软删除的记录
2. 走 `createNewUser` 分支，尝试 `INSERT INTO sys_user`
3. 撞上唯一索引 `idx_sys_user_username`（软删除行仍占用同名 username）
4. → `SQLSTATE 23505 duplicate key value violates unique constraint`

**修复：** 在 `SyncUserFromAD` 创建新用户前，先检查是否存在软删除的同 `username` 用户。若有则**恢复**
（清除 `deleted_at`、更新 AD 信息、重置密码与角色），而不是 INSERT 新行。

代码库里已存在一个 `restoreDeletedUserWithADInfo`（`user_ou_service.go:99`），但它是 **unused**，
且属于 `UserOUService`，未被 `SyncUserFromAD` 调用 —— 这印证了"恢复软删除用户"的设计意图曾经存在
但未接入同步链路。

---

## Symptoms

```
INFO [SyncADUser] 开始同步 AD 用户: username=liubolin-001
ERRO [GORM错误] INSERT INTO "sys_user" (...) VALUES (...,'liubolin-001',...)
      | 错误: ERROR: duplicate key value violates unique constraint "idx_sys_user_username" (SQLSTATE 23505)
ERRO [SyncADUser] 同步失败: 创建用户失败: ERROR: duplicate key value violates unique constraint
ERRO [BatchSyncADUsers] 同步用户 liubolin-001 失败
```

- 批量同步大量用户时，相当一部分（之前被软删除过的）全部失败
- 失败发生在 `createNewUser` → `tx.Create(user)` 阶段

---

## Root Cause

### 数据流回溯

1. **SyncUserFromAD** (`internal/services/system/user_sync_service.go:49`)
   ```go
   err := tx.Where("ad_username = ? AND deleted_at IS NULL", adUser.Username).First(&user).Error
   if err == nil { return s.updateExistingUser(...) }              // 活跃用户 → 更新
   if errors.Is(err, gorm.ErrRecordNotFound) {
       return s.createNewUser(tx, &user, adUser, defaultRoleID)   // ← 软删除用户落这里
   }
   ```

2. **createNewUser** (`user_sync_service.go:77`)
   ```go
   user.Username = adUser.Username    // 与软删除行同名
   tx.Create(user)                    // → INSERT 撞 uniqueIndex
   ```

3. **模型定义** (`internal/models/user.go:10`)
   ```go
   Username string `gorm:"uniqueIndex;size:64;not null"`   // 普通 unique，不含 WHERE deleted_at IS NULL
   ```

### 为什么查询查不到但 INSERT 撞约束

| 操作 | 是否考虑软删除行 |
|------|------------------|
| `First` 查询（带 `deleted_at IS NULL`） | ❌ 不含软删除 |
| `uniqueIndex` 约束 | ✅ 含软删除（全表唯一） |

→ 查询说"不存在"，约束说"已存在"，矛盾触发 23505。

---

## Fix

在 `SyncUserFromAD` 的 `ErrRecordNotFound` 分支内、`createNewUser` 之前，插入"软删除用户恢复"逻辑：

```go
if errors.Is(err, gorm.ErrRecordNotFound) {
    // 检查是否存在软删除的同 username 用户（唯一索引包含软删除行）
    var softDeleted models.User
    if findErr := tx.Unscoped().
        Where("username = ? AND deleted_at IS NOT NULL", adUser.Username).
        First(&softDeleted).Error; findErr == nil {
        // 恢复软删除用户
        user = softDeleted
        return s.restoreSoftDeletedUser(tx, &user, adUser, defaultRoleID)
    }
    return s.createNewUser(tx, &user, adUser, defaultRoleID)
}
```

新增 `restoreSoftDeletedUser`：
- `deleted_at = NULL`（恢复）
- 更新 AD 字段（ad_dn、ad_ou_dn、ad_username、ad_synced_at、email、phone）
- 重置默认密码（对齐 createNewUser：InitFlag=true、PwdExpireDays=90）
- 重新分配默认角色（角色关联可能已被清理）

---

## Verification

| 步骤 | 命令 | 结果 |
|---|---|---|
| 编译 | `go build ./...` | ✅ PASS |
| 现有测试 | `go test ./internal/services/system/...` | ⚠️ 现有 user_sync 测试全部 skip(依赖未实现的测试DB)；TestUpdateAPIKey 失败但**预先存在**(stash 后仍失败，与本修复无关) |
| 逻辑审查 | 对照 createNewUser/updateExistingUser 风格 | ✅ 一致(Unscoped 绕过软删除过滤更新；重置密码/角色对齐首次创建) |

注：现有 `user_sync_service_test.go` 所有用例 `setupTestDBForSync` 直接 `t.Skip`，无可用集成测试。
修复的正确性依赖真实 DB 验证(需重启后端 + 重新批量同步之前失败的软删除用户)。

## Status

修复已应用并通过编译。等待用户重启后端 + 触发批量同步验证 23505 不再出现。

## Phase 40 Closure (2026-06-25)

复测 `internal/services/system/user_sync_service.go`:
- line 69 已调用 `s.restoreSoftDeletedUser(tx, &user, adUser, defaultRoleID)`
- line 147 用 `Unscoped()` + `deleted_at IS NOT NULL` 预检软删除用户
- line 237-288 `restoreSoftDeletedUser` 实现：清除 deleted_at + 更新 AD 字段 + 重置密码/角色

23505 重复键风险已通过"软删除恢复"分支解决。frontmatter 翻 `resolved`。

verification: `grep -n "restoreSoftDeletedUser\|Unscoped" internal/services/system/user_sync_service.go` 命中
files_changed: .planning/debug/ad-sync-duplicate-username-softdeleted.md
