---
slug: ad-ou-dn-column-missing
status: resolved
trigger: 修复 AD 用户登录时部门同步失败：数据库错误 "column ad_ou_dn of relation sys_user does not exist"
created: 2026-05-27
updated: 2026-05-27
session_type: bug
---

# Debug Session: AD User Department Sync Failure - Missing ad_ou_dn Column

## Symptoms

### Expected Behavior
AD 用户登录时部门信息应该成功同步到数据库，更新用户的 dept_id、ad_user_dn 和 ad_ou_dn 字段。

### Actual Behavior
部门信息更新失败，数据库报错缺少 ad_ou_dn 列。用户登录本身成功，但部门同步失败：
```
ERRO[2026-05-27 11:45:56] [GORM错误] UPDATE "sys_user" SET "ad_ou_dn"='...' WHERE id = '8bd62962-2e25-496a-b1c8-f9fad307c8db' AND "sys_user"."deleted_at" IS NULL
| 错误: ERROR: column "ad_ou_dn" of relation "sys_user" does not exist (SQLSTATE 42703)
```

虽然用户同步声称成功：
```
INFO[2026-05-27 11:45:56] [SyncADUser] 同步成功: userID=8bd62962-2e25-496a-b1c8-f9fad307c8db, username=chenchao-076
```
但后续的部门信息更新失败：
```
ERRO[2026-05-27 11:45:56] 更新用户 chenchao-076 部门信息失败: ERROR: column "ad_ou_dn" of relation "sys_user" does not exist
```

### Error Messages
```
ERROR: column "ad_ou_dn" of relation "sys_user" does not exist (SQLSTATE 42703)
```

### Timeline
- 2026-05-27 11:45:56: AD 用户 chenchao-076 登录，部门同步失败

### Reproduction
1. AD 用户尝试登录
2. 用户认证成功，AD 用户信息获取成功
3. 用户同步到 sys_user 表成功
4. 更新部门信息时失败（因为缺少 ad_ou_dn 列）

## Current Focus

**Hypothesis:** (待形成假设)
**Test:** (待设计测试)
**Expecting:** (待确定预期结果)
**Next Action:** gather initial evidence - 检查 User 模型定义和数据库迁移脚本
**Reasoning Checkpoint:** (待填写)

## Evidence

## Eliminated

## Resolution
**Root Cause:** User 模型缺少 `ad_ou_dn` 字段，且代码中错误使用了不存在的 `ad_user_dn` 字段（与现有的 `ad_dn` 重复）

**Fix:**
1. User 模型：添加 `AdOuDn` 和 `AdSyncedAt` 字段
2. user_ou_service.go：修改 `ad_user_dn` 为 `ad_dn`（使用现有字段）
3. 迁移脚本：添加 `ad_ou_dn` 和 `ad_synced_at` 列（不包括 `ad_user_dn`）
4. 测试文件：统一使用 `ad_dn` 字段名

**Verification:** ✅ 编译通过，`go build ./...` 无错误

**Files Changed:**
- `internal/models/user.go` - 添加 AdOuDn, AdSyncedAt 字段
- `internal/services/addomain/user_ou_service.go` - 使用 ad_dn 替代 ad_user_dn
- `internal/core/db/migrations/139_add_ad_ou_dn_user_fields.sql` - 添加列和索引
- `internal/services/addomain/user_ou_service_test.go` - 修复字段名
- `internal/services/addomain/member_sync_service_test.go` - 修复字段名

**Commits:**
- `4792f7c`: feat(ad): add AdOuDn, AdUserDn, AdSyncedAt fields to User model
- `14509fb`: fix(ad): 移除重复的 AdUserDn 字段，使用现有的 AdDn 字段
