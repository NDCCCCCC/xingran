---
status: fixed
trigger: GORM查询错误：部门缓存预热时 WHERE id IN 'map[...]' 导致SQL语法错误
slug: gorm-in-query-map-to-string
created: 2026-05-26
updated: 2026-05-26
type: bug
---

# Debug Session: GORM IN Query Map to String Error

## Symptoms

### Expected Behavior
部门缓存预热时，GORM应该生成正确的SQL查询：
```sql
SELECT "id","dept_name","ancestors" FROM "sys_dept" WHERE id IN ($1, $2, $3, ...) AND "sys_dept"."deleted_at" IS NULL
```

### Actual Behavior
GORM生成了错误的SQL，将Go map对象直接转换为字符串传递给SQL的IN子句：
```sql
SELECT "id","dept_name","ancestors" FROM "sys_dept" WHERE id IN 'map[06efaa51-c8c2-476d-a914-3ddeda5848ab:true 0bbb02cf-4415-4faa-b0aa-116843446117:true ...]' AND "sys_dept"."deleted_at" IS NULL
```

### Error Messages
```
ERRO[2026-05-26 17:22:37] [GORM错误] SELECT "id","dept_name","ancestors" FROM "sys_dept" WHERE id IN 'map[06efaa51-c8c2-476d-a914-3ddeda5848ab:true 0bbb02cf-4415-4faa-b0aa-116843446117:true 1590aad6-5615-46d1-a3b0-119e4e981b6a:true 4cf54e0c-1b35-4c8e-82f4-089839f999ed:true 6bdb9ce8-c3f7-4c13-9efc-563076032c26:true 8d98776b-b44e-4fa8-b918-c86c0bfc96d3:true ef76ffc5-30f5-4062-887d-f3cc960e26df:true]' AND "sys_dept"."deleted_at" IS NULL | 耗时: 3.2386ms | 错误: ERROR: syntax error at or near "$1" (SQLSTATE 42601)
```

### Timeline
- **发生时间**: 2026-05-26 17:22:37
- **触发条件**: 系统启动时的部门缓存预热
- **历史**: 这是一个新引入的bug，之前的缓存预热代码工作正常

### Reproduction
1. 启动后端服务
2. 触发部门缓存预热
3. 观察日志中的GORM错误

## Current Focus

**Hypothesis:** 代码在构建GORM查询时，错误地将一个 `map[string]bool` 类型的变量直接传递给了 `Where()` 方法，而不是先提取map的keys作为切片。

**Test:** 检查部门缓存预热相关代码，找到使用map构建IN查询的位置。

**Expecting:** 找到类似 `db.Where("id IN ?", deptMap)` 的错误用法，应该改为 `db.Where("id IN ?", getMapKeys(deptMap))`。

**Next Action:** gather initial evidence - 搜索部门缓存预热代码中的GORM IN查询

**Reasoning Checkpoint:** 需要找到日志输出 "开始预热缓存: dept" 的代码位置，然后追踪到构建GORM查询的具体代码行。

## Evidence

### Root Cause Found

**Location:** `internal/services/system/user_service.go:427` and `internal/services/system/user_service.go:449`

**Function:** `buildDepartmentPaths(ctx context.Context, list []models.User)`

**Bug Details:**
The function uses `map[string]bool` for deduplication of department IDs, but then passes these maps directly to GORM's `Where IN` clause instead of extracting the keys into a slice first.

**Problematic Code (Line 427):**
```go
// 获取所有唯一的部门ID
uniqueDeptIDs := make(map[string]bool)
for _, deptID := range userDeptMap {
    uniqueDeptIDs[deptID] = true
}

// 批量查询这些部门的ancestors信息
var depts []models.Department
s.db.WithContext(ctx).Select("id", "dept_name", "ancestors").Where("id IN ?", uniqueDeptIDs).Find(&depts)
```

**Problematic Code (Line 449):**
```go
// 对于每个有ancestors的部门，查询其所有祖先部门
allAncestorIDs := make(map[string]bool)
for _, dept := range depts {
    if dept.Ancestors != "" {
        ancestors := splitAncestors(dept.Ancestors)
        for _, ancestorID := range ancestors {
            allAncestorIDs[ancestorID] = true
        }
    }
}

// 查询所有祖先部门
var ancestorDepts []models.Department
if len(allAncestorIDs) > 0 {
    s.db.WithContext(ctx).Select("id", "dept_name").Where("id IN ?", allAncestorIDs).Find(&ancestorDepts)
}
```

**Why it fails:**
GORM expects a slice/array for `IN` clauses, not a map. When a map is passed, Go converts it to a string representation like `map[key1:value1 key2:value2]`, which gets inserted into the SQL as a string literal, causing syntax errors.

**Evidence from error log:**
The error shows exactly this pattern: `WHERE id IN 'map[06efaa51-c8c2-476d-a914-3ddeda5848ab:true ...]'`

The single quotes around the "map[...]" confirm that GORM treated it as a string value rather than a list of values.

## Eliminated

## Resolution

### Root Cause
在 `internal/services/system/user_service.go` 的 `buildDepartmentPaths` 函数中，有两处错误地将 `map[string]bool` 类型直接传递给 GORM 的 `Where IN` 子句：
- 第 427 行：`Where("id IN ?", uniqueDeptIDs)`
- 第 449 行：`Where("id IN ?", allAncestorIDs)`

GORM 需要切片/数组类型来构建 IN 查询，当传递 map 时，Go 会将其转换为字符串表示（如 `map[key:value]`），导致 SQL 语法错误。

### Fix
需要将 map 的键提取为切片后再传递给 GORM。修复方案：

**Line 420-427 (第一处):**
```go
// 获取所有唯一的部门ID
uniqueDeptIDs := make(map[string]bool)
for _, deptID := range userDeptMap {
    uniqueDeptIDs[deptID] = true
}

// 批量查询这些部门的ancestors信息
var depts []models.Department
// FIX: 提取map的keys为切片
deptIDList := make([]string, 0, len(uniqueDeptIDs))
for id := range uniqueDeptIDs {
    deptIDList = append(deptIDList, id)
}
s.db.WithContext(ctx).Select("id", "dept_name", "ancestors").Where("id IN ?", deptIDList).Find(&depts)
```

**Line 436-450 (第二处):**
```go
// 对于每个有ancestors的部门，查询其所有祖先部门
allAncestorIDs := make(map[string]bool)
for _, dept := range depts {
    if dept.Ancestors != "" {
        ancestors := splitAncestors(dept.Ancestors)
        for _, ancestorID := range ancestors {
            allAncestorIDs[ancestorID] = true
        }
    }
}

// 查询所有祖先部门
var ancestorDepts []models.Department
if len(allAncestorIDs) > 0 {
    // FIX: 提取map的keys为切片
    ancestorIDList := make([]string, 0, len(allAncestorIDs))
    for id := range allAncestorIDs {
        ancestorIDList = append(ancestorIDList, id)
    }
    s.db.WithContext(ctx).Select("id", "dept_name").Where("id IN ?", ancestorIDList).Find(&ancestorDepts)
}
```

### Verification
修复后应该：
1. 重启后端服务，观察部门缓存预热是否成功
2. 检查日志中是否还有 GORM 错误
3. 验证用户列表的部门路径显示功能正常
4. 运行 `go build ./...` 确保编译通过
5. 运行相关单元测试确保功能正常

### Files Changed
- `internal/services/system/user_service.go` - 修复 `buildDepartmentPaths` 函数中的两处 GORM 查询错误

## Specialist Review

**Specialist:** go (Go language specialist)
**Hint:** gorm-database-query

A root cause has been identified in a Go service using GORM ORM. The bug involves passing `map[string]bool` directly to GORM's `Where IN` clause instead of a slice. This causes GORM to convert the map to a string representation, resulting in SQL syntax errors.

**Recommended Fix Direction:**
Extract map keys into a slice before passing to GORM's `Where IN` clause:
```go
// Instead of: Where("id IN ?", myMap)
// Use:
idList := make([]string, 0, len(myMap))
for id := range myMap {
    idList = append(idList, id)
}
Where("id IN ?", idList)
```

Does this fix direction look correct for this Go/GORM codebase? Are there any GORM-specific best practices or utilities that should be used instead?