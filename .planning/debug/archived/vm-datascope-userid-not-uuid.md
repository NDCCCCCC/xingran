---
status: resolved
deferred_to: v1.16-tech-debt
trigger: 虚拟机 bound_user_id=ninedrunk 但用户 ninedrunk 登录后看不到该虚拟机
slug: vm-datascope-userid-not-uuid
created: "2026-06-04T06:00:00Z"
updated: 2026-06-25
phase: "25"
---

# Debug Session: VM DataScope Filtering Issue

## Symptoms

### Expected Behavior
用户 ninedrunk (DataScope=5 本人权限) 登录后应该能看到 bound_user_id=ninedrunk 的虚拟机

### Actual Behavior
用户 ninedrunk 登录后看不到 bound_user_id=ninedrunk 的虚拟机

### Error Messages
无错误信息，只是虚拟机列表为空

### Timeline
- Phase 25 代码已修改完成（Handler 层从 Gin Context 读取 user_id/data_scope）
- 测试数据准备：虚拟机 bound_user_id 设置为 "ninedrunk"
- 用户登录后查询虚拟机列表：返回空

### Reproduction
1. 创建虚拟机，设置 bound_user_id = "ninedrunk"
2. 使用用户 ninedrunk 登录（角色 test，DataScope=5）
3. 访问虚拟机列表页面
4. 观察到虚拟机列表为空

### Key Evidence
```sql
-- 数据库中的虚拟机记录
SELECT id, name, bound_user_id, bound_user_name FROM sys_vdi_vm WHERE bound_user_id = 'ninedrunk';

-- 结果：
-- id: 1d74f643-ffae-412f-8ac3-2791fed25f44
-- name: 虚拟机0002
-- bound_user_id: ninedrunk (字符串！)
-- bound_user_name: ninedrunk (ninedrunk)
```

**关键观察**: `bound_user_id` 字段存储的是**用户名** (`ninedrunk`)，而不是用户的 **UUID**。

---

## Current Focus

hypothesis: "bound_user_id 存储的是用户名 (ninedrunk)，但过滤逻辑使用用户 UUID 进行比较，导致匹配失败"
test: "检查 ApplyVMDataScopeFilter 函数中 bound_user_id 的比较逻辑"
expecting: "发现过滤逻辑使用 uuid_filter(userID) 与 bound_user_id 字段比较"
next_action: "Read vm_data_scope_filter.go to verify the comparison logic"
reasoning_checkpoint: |
  bound_user_id 字段类型可能是 varchar/text
  DataScope=5 (本人权限) 过滤条件应该是: bound_user_id = 当前用户ID

  问题：
  1. 如果存储的是用户名 "ninedrunk"
  2. 但过滤时传入的是用户 UUID
  3. 则 WHERE bound_user_id = 'uuid-string' 永远匹配不到 'ninedrunk'

  需要检查：
  - bound_user_id 字段定义
  - ApplyVMDataScopeFilter 的过滤逻辑
  - Handler 传递给 Service 的 userID 值是什么（UUID 还是 用户名？）

---

## Evidence

- timestamp: "2026-06-04T06:00:00Z"
  source: "User report + database query"
  observation: "bound_user_id = 'ninedrunk' (varchar)，不是 UUID 格式"
  sql_output: "1d74f643-ffae-412f-8ac3-2791fed25f44 | 虚拟机0002 | ninedrunk | ninedrunk (ninedrunk)"

- timestamp: "2026-06-04T06:05:00Z"
  source: "Code analysis of authentication flow"
  observation: "JWT token stores user.ID (UUID) in claims.UserID"
  file: "internal/core/security/jwt.go:89"
  details: "accessClaims.UserID = userID (passed as user.ID from login)"

- timestamp: "2026-06-04T06:10:00Z"
  source: "Code analysis of auth middleware"
  observation: "Auth middleware sets c.Set('user_id', claims.UserID) which is the UUID"
  file: "pkg/middleware/auth.go:96"
  details: "JWT auth middleware extracts UUID from claims and stores in context"

- timestamp: "2026-06-04T06:15:00Z"
  source: "Code analysis of VM handler"
  observation: "Handler extracts UUID from Gin context and passes to service"
  file: "internal/api/v1/vdi/vm_handler.go:77-79"
  details: "userID := c.Get('user_id') which is the UUID from JWT"

- timestamp: "2026-06-04T06:20:00Z"
  source: "Code analysis of data scope filter"
  observation: "ApplyVMDataScopeFilter validates userID is UUID format, then compares with bound_user_id"
  file: "internal/services/vdi/vm_data_scope_filter.go:21-26, 93"
  details: |
    Line 23: if userID == '' || !isValidUUID(userID) { return query.Where('1=0') }
    Line 93: case DataScopeSelf: return query.Where('bound_user_id = ?', userID)

- timestamp: "2026-06-04T06:25:00Z"
  source: "Code analysis of BindUser service"
  observation: "BindUser stores username (not UUID) in bound_user_id field"
  file: "internal/services/vdi/vm_service_impl.go:814"
  details: "updates['bound_user_id'] = req.Username (stores 'ninedrunk', not UUID)"

---

## Root Cause

**ROOT CAUSE IDENTIFIED**:

Type mismatch between storage and comparison:

1. **BindUser service** (vm_service_impl.go:814) stores **username** in `bound_user_id`:
   ```go
   updates["bound_user_id"] = req.Username  // Stores "ninedrunk"
   ```

2. **Data scope filter** (vm_data_scope_filter.go:93) compares with **UUID**:
   ```go
   case DataScopeSelf:
       return query.Where("bound_user_id = ?", userID)  // userID is UUID like "550e8400-e29b-41d4-a716-446655440000"
   ```

3. **Result**: `WHERE bound_user_id = '550e8400-...'` never matches `'ninedrunk'`

---

## Resolution

root_cause: "bound_user_id 字段存储用户名，但数据范围过滤使用用户 UUID 进行比较，导致类型不匹配"
fix: "修改 BindUser 服务，将 bound_user_id 从存储用户名改为存储用户 UUID"
verification: "修复后，DataScope=5 用户应该能看到绑定给自己的虚拟机"
files_changed: []

## Phase 40 Closure (2026-06-25)

复测 `internal/services/vdi/vm_service_impl.go` 的 `BindUser` (line 810-833)：
- line 819-822 已先用 `username` 查 `sys_user` 拿到 `systemUser`
- line 832 `"bound_user_id": systemUser.ID` —— 写入的是 UUID（不是 username）

与 `vm_data_scope_filter.go` 的 `case DataScopeSelf: Where("bound_user_id = ?", userID)`
（userID 来自 JWT 也是 UUID）类型对齐，DataScope=5 用户可看到绑定给自己的 VM。
frontmatter 翻 `resolved`。

verification: `grep -n '"bound_user_id":' internal/services/vdi/vm_service_impl.go` 显示 `systemUser.ID`
files_changed: .planning/debug/vm-datascope-userid-not-uuid.md

---

## Eliminated

[none yet]
