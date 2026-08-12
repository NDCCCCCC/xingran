---
slug: operlog-dept-name-null-scan
status: resolved
deferred_to: v1.16-tech-debt
trigger: 第三次 AD 同步链路崩溃 — operlog middleware 报 "converting NULL to string is unsupported",触发用户 sys_user.id='8bd62962-...' 的 dept_name 为 NULL
created: 2026-06-16
updated: 2026-06-25
session_type: bug
related:
  - ad-sync-500-nul-byte-in-error-msg          # 同一链路 bug 1
  - ad-sync-500-on-conflict-duplicate-row      # 同一链路 bug 2
---

# Debug Session: OperLog 写入 NULL dept_name 失败

## 关键结论(用户首读这一段)

`internal/utils/context_helper.go:60` 的 `GetDeptNameFromDB` 用 `var deptName string` 扫描
`SELECT dept_name FROM sys_user WHERE id = ?` 的结果。当某用户 `sys_user.dept_name IS NULL`
(典型:系统账号/超管/未分配部门的用户),PostgreSQL 返回 NULL,Go `database/sql` 无法把 NULL
转换为 Go 的 `string`,抛 "converting NULL to string is unsupported" 错误。

`GetDeptNameFromDB` 在 **operlog 热路径** 上被调用 — `pkg/middleware/oper_log.go:101` 和
`internal/utils/operlog/operlog.go:253`,**每次写请求的 middleware 都会触发**。
NULL 兼容缺失意味着:任何 dept_name 为 NULL 的用户触发的写操作都会因为 operlog 中间件报错而
返回 500。

修复:`var deptName string` → `var deptName sql.NullString`,Scan 后用 `.Valid` 判断。

**这是 AD 同步链路上**第三个**被前两个修复暴露的隐藏 bug**。

---

## Symptoms

### Error Message
```
ERRO[2026-06-16 21:08:20] [GORM错误] SELECT dept_name FROM "sys_user" 
WHERE id = '8bd62962-2e25-496a-b1c8-f9fad307c8db' 
| 耗时: 1.5546ms 
| 错误: sql: Scan error on column index 0, name "dept_name": converting NULL to string is unsupported
```

### Trigger Conditions
- 用户 `8bd62962-2e25-496a-b1c8-f9fad307c8db` 在 `sys_user.dept_name` 上是 NULL
- 该用户触发任何写操作(POST/PUT/DELETE)
- → operlog middleware 调用 `GetDeptNameFromDB`
- → SELECT 返回 NULL → Go Scan 失败 → 整个 middleware 链返回 500

---

## Root Cause

### 调用链

1. **`pkg/middleware/oper_log.go:101`** (每次写请求)
   ```go
   deptName := utils.GetDeptNameFromDB(c, core.GetDB())
   ```

2. **`internal/utils/context_helper.go:60`** (修复前)
   ```go
   var deptName string                                                // ← 不能接受 NULL
   if err := db.Table("sys_user").
       Select("dept_name").
       Where("id = ?", userID).
       Scan(&deptName).Error; err == nil && deptName != "" { ... }
   ```

3. **数据源**:`sys_user` 表的 `dept_name` 列在 `models/user.go:21` 定义为 `*string`(可空)。
   PostgreSQL 真实表里某些用户的 `dept_name` 是 NULL(系统账号/超管/未分配部门)。

4. **Go Scan 失败**:Go `database/sql` 的 Scan 把列值赋给目标变量时,如果列是 SQL NULL
   而目标是 `string`(非指针、非 `sql.NullString`),就会抛
   `converting NULL to string is unsupported`。

### 为什么这个 bug 之前没被发现
- `GetDeptNameFromDB` 是 *nullable* 设计(返回 `*string`),暗示它知道 NULL 是合法值
- 但函数实现却用 plain `string` Scan,自相矛盾
- 大多数用户的 `dept_name` 是非空,平时不触发
- 一旦用户(系统/超管/未分配)触发,operlog middleware 直接 500

### 与前两个 bug 的关系
| 修复 | 暴露的下一个 bug | 原因 |
|------|------------------|------|
| NUL 字节(sync.go error_message) | ON CONFLICT duplicate row | sync 不再在 updateSyncLog 阶段崩,走到 batchUpdate |
| ON CONFLICT duplicate row | **NULL → string**(本次) | sync 不再在 batchUpdate 阶段崩,完成后 operlog middleware 触发 |

每一次修复让代码真的跑到了崩溃点。这说明:
- 应用的 happy path 走得通,但只要碰到"边界数据"(NUL 字节 / 重名 / NULL 字段),
  都会暴露应用层的健壮性缺陷
- **operlog 是写操作的必经中间件**,任何数据异常都会在这里炸

---

## Fix Applied

### `internal/utils/context_helper.go:60-72`

```go
// 关键:必须能接受 NULL。sys_user.dept_name 是 *string(nullable),
// 某些用户(系统/超管/未分配部门的用户)的 dept_name 为 NULL。
var deptName sql.NullString
if err := db.Table("sys_user").
    Select("dept_name").
    Where("id = ?", userID).
    Scan(&deptName).Error; err == nil && deptName.Valid && deptName.String != "" {
    // .Valid 区分"SQL 返回 NULL"和"SQL 返回空字符串"
    // .String 为 "" 也视为没有值,避免上游出现 "未知部门" 空标签
    return &deptName.String
}
```

### 关键点
- `sql.NullString` 是标准库类型,无需额外依赖
- `.Valid` 字段在 SQL 返回 NULL 时为 `false`,非 NULL(包括空字符串)时为 `true`
- 同时检查 `.Valid && .String != ""` 兼顾 NULL 和空字符串(都视为"没有部门")

---

## Verification

| 步骤 | 命令 | 结果 |
|---|---|---|
| 编译 | `go build ./...` | PASS |
| 单元测试 | `go test -count=1 ./internal/utils/...` | PASS |
| operlog 测试 | `go test -count=1 ./internal/utils/operlog/...` | PASS |
| vet | `go vet ./internal/utils/...` | clean |

> 注: `internal/services/system/apikey_service_test.go:452` panic 是**预先存在**的
> 测试 bug,与本次修复无关(我没碰过那个文件)。

---

## Remaining Risks

1. **`GetClientIP` 等其他 util 函数**未来可能有同样 NULL 不兼容问题。
   建议对所有从 DB 读 string 字段的 helper 做一次 NULL 兼容审计。

2. **opera_log 历史表**:已写入的 NULL dept_name 操作记录(虽然正常情况下不应该有,
   但如果 NULL scan 修复前 operlog 失败,记录根本没写入)需要确认。
   实际上 NULL scan 失败会回滚整个事务,所以历史表不会有 NULL。

3. **operlog 对 nil deptName 的下游处理**:`oper_log_service.go:53` 直接用 `deptName *string`,
   nil 安全。本次修复保证 `GetDeptNameFromDB` 在 NULL 时返回 nil 而不是报错。

---

## Files Changed

| 文件 | 修改 |
|---|---|
| `internal/utils/context_helper.go` | +19 行 / 修改 1 处:`GetDeptNameFromDB` 用 `sql.NullString` 替换 `string`,NULL 安全扫描 |

## Status

**修复已应用并通过测试** — 等待用户重启后端 + 触发任意写请求验证 500 不再出现。

## Phase 40 Closure (2026-06-25)

复测 `internal/utils/context_helper.go`：
- `GetDeptNameFromDB` (line 98-) 已用 `var deptName sql.NullString` 接收 NULL，
  Scan 后用 `.Valid && .String != ""` 双重判断
- nickname 字段同样用 sql.NullString（line 67）

NULL→string 500 风险已消除。frontmatter 翻 `resolved`。

verification: `grep -n "sql.NullString" internal/utils/context_helper.go` 命中
files_changed: .planning/debug/operlog-dept-name-null-scan.md
