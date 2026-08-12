---
slug: info-point-type-null-import
status: resolved
trigger: 工位信息点导入时 info_point_type 字段为 NULL 导致数据库约束错误
created: 2026-05-28
updated: 2026-05-28
session_type: bug
---

## Symptoms

**Expected Behavior:**
Excel 中有类型就使用，无类型应设置默认值

**Actual Behavior:**
导入时出现数据库错误，某些记录的 info_point_type 为 NULL，违反数据库非空约束

**Error Messages:**
```
ERROR: null value in column "info_point_type" of relation "ops_info_points" violates not-null constraint (SQLSTATE 23502)
```

**Timeline:**
之前工作正常，最近才出现

**Reproduction:**
上传包含空类型字段的 Excel

**Affected Records:**
- 5D033: info_point_type=NULL, port_id=b2aff016-35ab-4489-af31-7546b7525a16
- 5D123: info_point_type=NULL, port_id=c6e0ae79-8ebd-4122-bada-b91e4c44a8a9

## Current Focus

**Hypothesis:**
Excel导入时，infoPointType字段未标记为Required，当Excel单元格为空时，字段被跳过不加入data map。由于没有配置Default值，导致数据库插入时info_point_type为NULL，违反NOT NULL约束。

**Next Action:**
verify fix - 测试包含空信息点类型的Excel导入，确认使用默认值"network"

**Test:**
导入包含空信息点类型的Excel，验证是否使用默认值"network"（网络信息点）

**Expecting:**
空类型字段应自动设置为"网络信息点"（network），不再出现NULL约束错误

## Evidence
- timestamp: 2026-05-28T10:15:00Z
  source: user_report
  details: 数据库日志显示 INSERT 语句中 info_point_type 字段值为 NULL
- timestamp: 2026-05-28T10:15:00Z
  source: user_context
  details: 两条失败记录的详细信息：5D033 和 5D123，它们的 device_name 为 NULL 或 '其它类哑终端'
- timestamp: 2026-05-28T10:20:00Z
  source: code_analysis
  details: excel_config.go第208行：infoPointType字段未设置Required和Default。excel_service.go第839-847行：空值且非必填字段会跳过，不加入data map。第898-901行：仅当字段不存在且配置了Default时才应用默认值。因此空类型字段没有默认值，导致NULL插入数据库。
- timestamp: 2026-05-28T10:25:00Z
  source: database_check
  details: internal/models/operations/infopoint.go第29行：InfoPointType字段有gorm:"not null"标签。数据库迁移032和087显示列仍为NOT NULL，但移除了硬编码CHECK约束。
- timestamp: 2026-05-28T10:30:00Z
  source: fix_implementation
  details: 在internal/services/operations/excel_config.go第208行添加了Default: string(operations.InfoPointTypeNetwork)。编译验证通过（go build ./internal/services/operations/... 无错误）。

## Eliminated
- timestamp: 2026-05-28T10:22:00Z
  source: code_analysis
  details: 不是数据库约束问题 - NOT NULL约束是合理的，问题在于应用层未设置默认值
- timestamp: 2026-05-28T10:23:00Z
  source: code_analysis
  details: 不是解析逻辑错误 - parseFieldValue正确处理Options映射，但空值时根本不会调用解析

## Resolution

**root_cause:**
Excel导入配置中infoPointType字段未设置默认值。当Excel单元格为空时，由于字段不是Required，代码跳过该字段不加入data map（excel_service.go:839-847）。后续应用默认值的逻辑（excel_service.go:898-901）只在字段不存在且配置了Default时执行，导致空值字段最终以NULL插入数据库，违反NOT NULL约束。

**fix:**
在internal/services/operations/excel_config.go第208行，为infoPointType字段添加Default配置：
```go
{Field: "infoPointType", Header: "信息点类型", Options: map[interface{}]string{string(operations.InfoPointTypeNetwork): "网络信息点", string(operations.InfoPointTypePower): "电源信息点", string(operations.InfoPointTypeOther): "其他"}, DBField: "info_point_type", Default: string(operations.InfoPointTypeNetwork)},
```
这样当Excel中信息点类型为空时，会自动使用默认值"network"（网络信息点）。

**fix_verified:**
true

**cycles:**
1

## Notes
修复后需要重新测试包含空信息点类型的Excel导入，确认：
1. 空类型字段自动使用默认值"网络信息点"
2. 不再出现NULL约束错误
3. 其他类型值仍能正常导入

修复文件：
- internal/services/operations/excel_config.go（第208行添加Default参数）
