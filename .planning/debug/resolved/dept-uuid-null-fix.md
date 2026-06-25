---
slug: "dept-uuid-null-fix"
status: fixing
trigger: "部门AD同步失败: UUID类型错误"
created: "2026-05-25T07:35:00.000Z"
updated: "2026-05-25T07:35:00.000Z"
tdd_mode: "false"
---

# Debug Session: 部门UUID类型错误

## Trigger

用户输入: 调试并修复部门AD同步中的UUID类型错误

**错误信息**:
```
ERROR: invalid input syntax for type uuid: "" (SQLSTATE 22P02)
```

**问题SQL**:
```sql
SELECT * FROM "sys_dept" WHERE (parent_id IS NULL OR parent_id = '') AND status = 0
```

## Symptoms

### Expected behavior
- 部门AD同步应该成功查询根部门（parent_id为NULL的部门）

### Actual behavior
- 查询失败，PostgreSQL报错：UUID类型不能与空字符串比较

### Error messages
```
ERROR: invalid input syntax for type uuid: "" (SQLSTATE 22P02)
ERRO[2026-05-25 15:43:11] [GORM错误] SELECT * FROM "sys_dept" WHERE (parent_id IS NULL OR parent_id = '') AND status = 0 AND "sys_dept"."deleted_at" IS NULL
ERRO[2026-05-25 15:43:11] 部门到AD同步失败: 获取根部门失败
```

### Timeline
- 刚刚在执行"部门到AD同步"任务时发现

### Reproduction
- 执行部门AD同步定时任务
- 触发 `dept_to_ad_sync` 任务

## Current Focus

- **hypothesis**: `getRootDepartments` 方法中的查询条件 `parent_id = ''` 对UUID类型字段无效
- **test**: 读取 `dept_sync_service.go` 并修改查询条件
- **expecting**: 移除 `parent_id = ''` 条件，只保留 `parent_id IS NULL`
- **next_action**: 应用修复并验证
- **reasoning_checkpoint**: UUID字段在PostgreSQL中只能存储有效的UUID值或NULL，不能与空字符串比较

## Evidence

### 2026-05-25T07:35:00.000Z
- **source**: user_diagnosis
- **evidence**: 用户已明确指出问题位置和修复方案
  - 文件: `dept_sync_service.go`
  - 方法: `getRootDepartments`
  - 问题: `parent_id = ''` 对UUID类型无效
  - 修复: 改为只检查 `parent_id IS NULL`

## Eliminated

None - diagnosis already provided

## Resolution

- **root_cause**: UUID类型字段不能与空字符串比较，只能与NULL或有效UUID值比较
- **fix**: 将查询条件从 `parent_id IS NULL OR parent_id = ''` 改为只使用 `parent_id IS NULL`
- **files_changed**: `internal/services/addomain/dept_sync_service.go`
- **verification**: ✅ 编译通过 (`go build ./internal/services/addomain/...`)
- **status**: fixed

## Verification Steps

1. ✅ 代码编译通过
2. ⏳ 需要用户重新测试部门AD同步功能
3. ⏳ 验证根部门查询正常工作

## TDD Checkpoint

Not applicable - straightforward bug fix
