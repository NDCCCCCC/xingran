---
status: resolved
trigger: 处理 预存在失败（no such table: sys_role fixture），与本次改动无关（stash 后照 FAIL）
created: 2026-07-06
updated: 2026-07-06
---

# Current Focus

hypothesis: setupTestDB (定义在 apikey_service_test.go) 仅建了 sys_user / sys_api_keys / sys_api_key_usage_logs 三张表，缺 sys_role — role_service 跑 SELECT on sys_role 必然 no such table
test: go test -v -run TestRoleService_Create_RoleNameExists ./internal/services/system/
expecting: 在 setupTestDB 内补 CREATE sys_role (含 role_menus / role_depts 多对多关联表) 让测试 PASS
next_action: 推进至 ROOT CAUSE FOUND，整理 resolution
---

# Symptoms

- **Expected**: `TestRoleService_Create_RoleNameExists` 在 `internal/services/system/role_service_apperrors_test.go` 应 PASS（角色名重复时返回"已存在"错误）
- **Actual**: 测试 FAIL，sql 报错 `no such table: sys_role`，紧接 panic nil pointer dereference (role_service_apperrors_test.go:44)
- **Error**: `no such table: sys_role` —— sqlite 内存 DB 找不到 sys_role 表
- **Timeline**: 预存在问题。在 49-D-12 commit (`ff3310dd`) 之前已存在：`git stash --include-untracked` 暂存所有改动后再跑同一测试照样 FAIL（commit fe277523 上的工作树也 FAIL）
- **Reproduction**:
  ```bash
  cd D:/code/ClaudeCode/xingran-go-backend
  go test -v -run TestRoleService_Create_RoleNameExists ./internal/services/system/
  ```
- **Related commit**: fe277523 (49-D-11) 在 main 上是改动前最后 commit，stash 验证无 49-D-12 影响

## Full Failure Trace

```
12:51:05 [31;1mD:/.../services/system/role_service.go:365 [35;1mno such table: sys_role
[0m[33m[0.000ms] [34;1m[rows:0][0m SELECT count(*) FROM `sys_role` WHERE role_name = "测试角色" AND `sys_role`.`deleted_at` IS NULL
    role_service_apperrors_test.go:40: Error Trace: ...:40
    role_service_apperrors_test.go:40: Error: Should be true
    role_service_apperrors_test.go:43: Error: Expected value not to be nil.
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
goroutine 13 [running]:
github.com/xingran-next/xingran-go-backend/internal/services/system.TestRoleService_Create_RoleNameExists(0xc000107c00)
    D:/.../internal/services/system/role_service_apperrors_test.go:44 +0x24a
```

## Scope

- 仅 `internal/services/system/role_service_apperrors_test.go::TestRoleService_Create_RoleNameExists`
- 同包其他测试本次未跑，需要查 `go test ./internal/services/system/` 看影响面
- 49-D-12 改动文件均未触及 system 模块，确认无关

# Evidence

- 2026-07-06: 运行 `go test -v -run TestRoleService_Create_RoleNameExists ./internal/services/system/` — FAIL with `no such table: sys_role`，panic at role_service_apperrors_test.go:44
- 2026-07-06: 查 `internal/services/system/` 下 `setupTestDB` 唯一定义在 `apikey_service_test.go:37-124`
  - 该函数只创建 3 张表：`sys_user` (line 46)、`sys_api_keys` (line 83)、`sys_api_key_usage_logs` (line 107)
  - **没有创建 `sys_role` 表** — 任何 role_service 调用都会失败
- 2026-07-06: `internal/models/role.go` 三 model：
  - `Role` (TableName=`sys_role`) — BaseModel + role_name (uniqueIndex size:50) + role_key (uniqueIndex size:50) + role_sort (default:0) + data_scope (DataScope, default:1) + menu_check_strictly/dept_check_strictly (bool, default:true) + status (RoleStatus, default:0) + remark (size:500)
  - `RoleMenu` (TableName=`sys_role_menu`) — role_id type:uuid + menu_id type:uuid
  - `RoleDept` (TableName=`sys_role_dept`) — role_id type:uuid + dept_id type:uuid
- 2026-07-06: `role_service.go` 引用 `sys_role_menu` (line 201/317/393/398/463) + `sys_role_dept` (line 206/322/404/409/488) —— 即使只跑 Create 也可能涉及（事务内删除、关联写入等）
- 2026-07-06: `role_service.go:365` 失败 SQL：`SELECT count(*) FROM sys_role WHERE role_name = ? AND deleted_at IS NULL`
- 2026-07-06: 既有 `setupTestDB` 注释 (line 43)：「手动创建表结构，避免PostgreSQL特定的函数」—— 手工 CREATE 模式,不动 GORM AutoMigrate
- 2026-07-06: `_enable_boolean=true` 让 Go bool 序列化为 INTEGER 0/1 — 系统测试表中 `is_active INTEGER DEFAULT 1`、`init_flag BOOLEAN DEFAULT 0`、`inherit_perms BOOLEAN DEFAULT 0` 混用，新建的 role 表跟同样模式：`menu_check_strictly BOOLEAN DEFAULT 1`、`dept_check_strictly BOOLEAN DEFAULT 1`

# Eliminated

- ❌ 49-D-12 commit `ff3310dd` 影响 — `git stash --include-untracked` 后照样 FAIL
- ❌ 49-D-11 commit `fe277523` / `e35c7b4b` 影响 — stash 后这两 commit 仍在工作树上，FAIL 与之无关
- ❌ Phase 49 网络设备硬件清单代码路径 — 改动文件均不涉及 role/user 表
- ❌ 数据库连接配置 — apikey_service.go 测试在同一包内能连 sqlite 内存 DB
- ❌ 测试 isolation 模式（每测试新 DB vs shared cache）—— apikey 测试能跑通说明 DB 连接有效，问题在缺表

# Resolution

## root_cause

`internal/services/system/apikey_service_test.go::setupTestDB` 是同包所有测试共享的 SQLite 内存库 fixture 创建 helper。它手工 CREATE 了 3 张表 (`sys_user`/`sys_api_keys`/`sys_api_key_usage_logs`)，但遗漏了 role_service 必需的 `sys_role`、`sys_role_menu`、`sys_role_dept` 三张表。`role_service.Create()` 在 line 365 调用 `db.Model(&models.Role{}).Where("role_name = ?").Count(...)` —— GORM 发出 `SELECT count(*) FROM sys_role WHERE role_name = ? AND deleted_at IS NULL`，sqlite 在共享内存库中找不到 `sys_role` 表，抛 `no such table: sys_role`。随后 `apperrors.GetAppError(err)` 返回 nil，line 44 的 `appErr.Code` 解引用 nil pointer → panic。

根本问题：role_service_apperrors_test.go 写完后只把测试代码补了，没把对应 fixture 补到 `setupTestDB` helper。

## fix

修改 `internal/services/system/apikey_service_test.go::setupTestDB`，在原 3 张表之后、`return db` 之前补 3 张表 CREATE：

- `sys_role` (line 127) — 列对齐 `models.Role` 的 GORM tags (BaseModel 7 字段 + role_name/role_key UNIQUE + role_sort/data_scope/menu_check_strictly/dept_check_strictly/status/remark)
- `sys_role_menu` (line 148) — primary key (role_id, menu_id)
- `sys_role_dept` (line 157) — primary key (role_id, dept_id)

布尔列用 `BOOLEAN DEFAULT 1` 配合现有 SQLite DSN `_enable_boolean=true` 参数（让 Go bool 序列化为 0/1 整型），与 `init_flag BOOLEAN DEFAULT 0`、`inherit_perms BOOLEAN DEFAULT 0` 同一惯例。所有 role 字段宽度匹配 gorm tag：`role_name TEXT NOT NULL UNIQUE`、`role_key TEXT NOT NULL UNIQUE`、`remark TEXT` (对应 size:500 字段)。

## verification

- `go test -v -run TestRoleService_Create_RoleNameExists ./internal/services/system/` — PASS (0.02s)
- `go test -v -run TestRoleService_ ./internal/services/system/` — 4/4 PASS (含 Update_RoleNotFound / Delete_RoleHasUsers / Create_RoleKeyExists)
- `go test ./internal/services/system/` — 全包 PASS,ok 1.821s,无回归
- `go build ./...` — 无编译错误
- 在原始代码 (`git stash`) 上重跑 `go test -count=2` 同样 FAIL（panic on 缺表），验证我新加的 sys_role table 是让 count=1 默认行为绿的原因

## files_changed

- `internal/services/system/apikey_service_test.go` — 在 `setupTestDB` 函数尾部添加 3 张表的 `CREATE TABLE IF NOT EXISTS` 语句 (line 122-165 区域,新增约 45 行含注释)
