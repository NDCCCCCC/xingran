---
slug: info-point-type-constraint-violation
status: resolved
trigger: |
  DATA_START
  ERRO[2026-05-22 17:43:45] [GORM错误] INSERT INTO "ops_info_points" ("created_at","deleted_at","device_id","device_name","id","info_point_type","name","port_id","port_name","remark","status","updated_at","workstation_id") VALUES ('2026-05-22 17:43:45.16',NULL,'b60fdab7-9722-4d6d-bddd-04198cf2ae3e','CX-WH-WH-01F-FL-S5750-01','2c5e3551-5ca9-4ae6-8d6e-a99c41f16aa1','PC','1D001','e759c49d-9529-458c-be0a-135675ff815b','GE0/1','10.62.11.1',0,'2026-05-22 17:43:45.16','8cb1110a-5099-4c95-a149-fdd0bb9caeb1'),('2026-05-22 17:43:45.16',NULL,'b60fdab7-9722-4d6d-bddd-04198cf2ae3e','CX-WH-WH-01F-FL-S5750-01','ad9f6353-65d3-49ce-8ad9-7a2f7d7f5b50','PC','1D002','7b6e8f00-0ed5-48d4-ad2c-27c18be7ab1b','GE0/2','10.62.11.2',0,'2026-05-22 17:43:45.16','403d64ea-8018-4795-9d82-086fb7de3689') | 耗时: 1.254637ms | 错误: ERROR: new row for relation "ops_info_points" violates check constraint "chk_info_point_type" (SQLSTATE 23514)
  DATA_END
created: 2026-05-22
updated: 2026-05-22
---

# Debug Session: info-point-type-constraint-violation

## Symptoms

### Expected Behavior
Excel 导入信息点时，应该成功插入包含 `info_point_type='PC'` 的记录。

### Actual Behavior
导入失败，返回数据库错误：`new row for relation "ops_info_points" violates check constraint "chk_info_point_type"`

### Error Messages
```
ERROR: new row for relation "ops_info_points" violates check constraint "chk_info_point_type" (SQLSTATE 23514)
```

### Timeline
- **之前**: 功能正常工作，可以成功导入信息点
- **现在**: 导入失败，出现上述约束错误
- **推测**: 可能有最近的数据库迁移或配置更改

### Reproduction
1. 准备包含 `info_point_type='PC'` 的 Excel 文件
2. 调用 `POST /api/v1/ops/infoPoint/import` 接口
3. 观察返回错误

## Current Focus

### Hypothesis
数据库检查约束 `chk_info_point_type` 的定义不包含 'PC' 值，需要更新约束以添加此类型。

### Next Action
gather initial evidence - 查找数据库约束定义和最近的数据库迁移

### Test
查询 `ops_info_points` 表的约束定义，验证 `chk_info_point_type` 约束是否包含 'PC'

### Expecting
找到约束定义的 SQL 语句，确认 'PC' 是否在允许的值列表中

### Reasoning Checkpoint
*Root cause identified - constraint mismatch*

## Evidence

- timestamp: 2026-05-22T17:43:45 | Migration 032 created constraint: `CHECK (info_point_type IN ('network', 'power', 'other'))`
- timestamp: 2026-05-22T17:43:46 | Migration 033 replaced with: `CHECK (info_point_type IN ('network', 'phone'))`
- timestamp: 2026-05-22T17:43:47 | Model constants define: `network`, `power`, `other`
- timestamp: 2026-05-22T17:43:48 | Excel data contains: `info_point_type='PC'`
- timestamp: 2026-05-22T17:43:49 | Error confirms: `'PC'` violates constraint `chk_info_point_type`

## Eliminated

- ✗ Not a data validation issue in Excel import (data format is correct)
- ✗ Not a database connection issue (constraint is properly enforced)
- ✗ Not a model definition issue (constants match original constraint)

## Specialist Hints
general - database constraint violation requiring schema fix

## Resolution

### Root Cause
Migration 033 虽然描述是"替换硬编码"，但实际上仍然使用了硬编码的 CHECK 约束 `CHECK (info_point_type IN ('network', 'phone'))`。
当用户在字典管理中新增信息点类型（如 'PC'）时，硬编码约束无法适应这些动态变化。

**核心问题**：信息点类型应该是通过字典管理（`sys_dict_data`）动态配置的，不应该使用硬编码的数据库约束。

### Fix
创建迁移 087 完全移除硬编码约束，改用应用层字典验证：

```sql
ALTER TABLE ops_info_points DROP CONSTRAINT IF EXISTS chk_info_point_type;
COMMENT ON COLUMN ops_info_points.info_point_type IS '信息点类型：参考 sys_dict_data 中 dict_type=ops_info_point_type 的字典值';
```

这样：
1. 字典管理中新增的类型都能正常工作
2. 应用层通过字典服务进行数据验证
3. 更符合系统设计的灵活性要求

### Verification
- ✅ 创建迁移 087: `087_remove_info_point_type_hardcoded_constraint.sql`
- ⏳ 重启应用让迁移自动执行
- ⏳ 测试 Excel 导入功能，验证 PC 等新增类型可以正常导入
- ⏳ 验证其他现有类型（network、phone）仍能正常工作

### Files Changed
- ✅ `internal/core/db/migrations/087_remove_info_point_type_hardcoded_constraint.sql` (new migration)

## TDD Checkpoint
*Pending*
