---
quick_id: "260605-hf7"
slug: "fix-ou-mapping-query"
description: "修复OU组映射查询错误 - 移除deleted_at条件"
status: "complete"
date: "2026-06-05"
duration: "2min"
author: "Claude"
files_modified:
  - "internal/scheduler/dept_sync_tasks.go"
---

# Quick Task: 修复OU组映射查询错误

## 问题

执行AD组自动同步任务时出现错误：
```
查询OU-组映射失败: ERROR: column "deleted_at" does not exist
```

## 原因

查询条件中使用了 `deleted_at IS NULL`，但 `sys_ou_group_mapping` 表没有该列。

## 解决方案

移除查询条件中的 `deleted_at IS NULL` 部分：

**修改前**:
```go
Where("ad_config_id = ? AND mapping_status = ? AND sync_enabled = ? AND deleted_at IS NULL",
    adConfigID, models.OUGroupMappingStatusActive, true)
```

**修改后**:
```go
Where("ad_config_id = ? AND mapping_status = ? AND sync_enabled = ?",
    adConfigID, models.OUGroupMappingStatusActive, true)
```

## 验证

- [x] 编译通过
- [x] 提交完成
- [x] 查询条件修复

---
**完成时间**: 2026-06-05
**提交**: 7cc4680
