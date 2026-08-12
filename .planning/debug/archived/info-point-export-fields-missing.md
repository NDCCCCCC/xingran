---
slug: info-point-export-fields-missing
status: root_cause_found
trigger: 信息点导出功能，经过多次修改，设备名称齐全了，但是所属工位名称不完整，端口名称完全没有
created: 2026-06-30
updated: 2026-06-30
session_type: observation
goal: find_and_fix
---

# Debug Session: 信息点 Excel 导出 工位名称不完整 + 端口名称空

## Symptoms

### Expected Behavior
- 信息点(info-points) Excel 导出
- 所属工位名称(来自 sys_workstation.workstation_name): 应显示完整工位名(和设备名称一样)
- 端口名称(来自 sys_device_port_status.interface_name): 应显示端口完整名(如 GE5/44)

### Actual Behavior
- ✅ 设备名称齐全(sys_network_device.device_name Join 工作正常)
- ❌ 所属工位名称 不完整(部分行有,部分行空)
- ❌ 端口名称 完全没有任何行显示(sys_device_port_status.interface_name 整列为空)

### Reference Context
- WIP commit d73c0984 (workstation-physical-link-zero paused) 修改了:
  - excel_config.go: portName 加 DependsOn deviceName
  - excel_export_config.go: JoinConfig 加 RightCast 字段 + 修正 device_name/interface_name 字段名
  - excel_export_service.go: IN 子句支持 ::text 类型转换
- 当时修改后未做导出验证,直接 PAUSED 等待后端重启 + 验证

## Root Cause Analysis

### 第一根因(端口完全空)

`internal/services/operations/excel_export_service.go:154` 的 `resolveAssociations` 对所有 JoinConfig **无条件**加 `WHERE deleted_at IS NULL` 过滤:

```go
rows, err := s.db.WithContext(ctx).
    Table(join.Table).
    Select(join.RightField+", "+join.SelectField).
    Where(rightFieldExpr+" IN ?", ids).
    Where("deleted_at IS NULL").  // ← 对 sys_device_port_status 报错 SQLSTATE 42703
    Rows()

if err != nil {
    logger.Warnf("批量查询关联数据失败: %v", err)
    continue  // ← 错误被吞,直接 continue 到下一个 join,portName 永远空
}
```

**触发条件** — JoinConfig.Table 的目标表是否有 `deleted_at` 列:

| 表 | 模型 | DeletedAt | 列存在? | 后果 |
|---|------|-----------|--------|------|
| sys_workstation | `Workstation` 含 `BaseModel` | 有 | ✅ | join 正常 |
| sys_network_device | `NetworkDevice` 含 `BaseModel` | 有 | ✅ | join 正常 |
| sys_device_port_status | `DevicePortStatus` 不含 `BaseModel` | **无** | ❌ | SQLSTATE 42703,结果集空 |

模型证据:
- `internal/models/workstation.go:37` — `type Workstation struct { BaseModel }`(含 DeletedAt)
- `internal/models/network_device.go:35` — `type NetworkDevice struct { BaseModel }`(含 DeletedAt)
- `internal/models/device_port_status.go:30-56` — `type DevicePortStatus struct` 无 BaseModel/无 DeletedAt 字段

auto-migrate 在 `internal/core/db/database.go:308` 用了 `&models.DevicePortStatus{}`,所以表无 deleted_at 列。

### 第二根因(工位不完整)

次根因,优先级较低:`ops_info_points.workstation_id` 与 `sys_workstation.id::text` 格式匹配失败的部分行。
可能:
- 部分行 workstation_id 为空字符串(模型是 `not null` 但实际数据可能违规)
- 部分行 workstation_id 找不到对应的 sys_workstation(工位被硬删除/UUID 漂移)

需要导出查询结果后才能精确定位(不阻塞本次修复)。

### 失理论证 — 为何 RightCast 修复没救

WIP commit 加了 `RightCast: "text"` 让 IN 子句做 UUID→text 强制转换,前提是 SQL **能跑通**。当 `WHERE deleted_at IS NULL` 报 42703 时,整个 query 失败,RightCast 没机会生效。同样不匹配的 UUID 也没机会匹配上。

## Fix Plan

### 方案 A (推荐,最小侵入)

在 `JoinConfig` 加 `SkipSoftDelete bool` 字段,目标表无 deleted_at 列时显式跳过,并在 portName join 启用:

```go
type JoinConfig struct {
    Table          string
    LeftField      string
    RightField     string
    SelectField    string
    As             string
    RightCast      string
    SkipSoftDelete bool // 选填: true 时跳过 `deleted_at IS NULL` 过滤(用于无 deleted_at 列的表,如 sys_device_port_status)
}
```

`resolveAssociations` 调整:
```go
if !join.SkipSoftDelete {
    queryBuilder = queryBuilder.Where("deleted_at IS NULL")
}
```

`excel_export_config.go` portName JoinConfig 加 `SkipSoftDelete: true`,其他 Join 不动。

### 方案 B (替代,更通用)

在 resolveAssociations 内动态检查 join.Table 是否有 deleted_at 列(查 information_schema.columns)。优点:对所有未来类似表自动适配;缺点:增加 DB round-trip。

### 选择 A 的理由
- 最小改动(1 个新字段 + 2 行实现代码 + 1 处配置)
- 配置即文档(每个 join 显式声明,审计清晰)
- 不引入额外 DB 查询

## Resolution

root_cause: `internal/services/operations/excel_export_service.go::resolveAssociations` 对所有 JoinConfig 无条件追加 `WHERE deleted_at IS NULL`,但 `sys_device_port_status` 模型不嵌入 BaseModel 因此无 `deleted_at` 列,PG 抛 SQLSTATE 42703 被吞掉,portName 整列空。次根因(工位不完整)部分行 workstation_id 无法匹配 sys_workstation 行,本次修复暂不覆盖。

fix: (1) `excel_export_config.go::JoinConfig` 增加 `SkipSoftDelete bool` 字段;(2) `excel_export_service.go::resolveAssociations` 在该字段为 true 时跳过 `deleted_at IS NULL` 过滤;(3) `excel_export_config.go::infoPoint.portName` JoinConfig 加 `SkipSoftDelete: true`。

verification:
- `go build ./...` 退出码 0 ✅
- `go vet ./internal/services/operations/` 退出码 0 ✅
- `go test ./internal/services/operations/ -run "TestResolve|TestExport|TestInfoPoint"` PASS ✅
- UAT 待用户在浏览器跑(后端已加此修复后导出 info-points 应显示完整 interface_name)

files_changed:
- internal/services/operations/excel_export_config.go(JoinConfig 新增 SkipSoftDelete 字段 + portName JoinConfig 加 SkipSoftDelete:true)
- internal/services/operations/excel_export_service.go(resolveAssociations 重构为 queryBuilder 链式调用 + 条件加 deleted_at IS NULL)

status: resolved (fix applied, build+vet+test green; browser UAT 待用户验证。Workstation 部分缺失为次根因,本次未覆盖,留作 follow-up。)
## Evidence

- timestamp: 2026-06-30
  checked: WIP commit d73c0984 实际 diff
  found: excel_export_config.go 修改了 deviceName SelectField 从 "name" → "device_name",portName 从 "port_name" → "interface_name"。两处均为 SELECT 字段名修正。
  implication: WIP commit 的初衷是修正字段名 + 加 RightCast,但**没注意 sys_device_port_status 是无 deleted_at 表**,RightCast 在 deleted_at 错误面前根本到不了。
- timestamp: 2026-06-30
  checked: `internal/models/device_port_status.go` 模型定义
  found: `type DevicePortStatus struct` 无 `BaseModel` 嵌入,无 `gorm.DeletedAt` 字段。
  implication: 自动迁移创建的 sys_device_port_status 表**无 deleted_at 列**。
- timestamp: 2026-06-30
  checked: `internal/core/db/database.go::AutoMigrate` 注册的模型
  found: `&models.DevicePortStatus{}` 在 AutoMigrate 列表(line 308)
  implication: 表结构由 GORM 从 model 推导,**Confirmed 无 deleted_at 列**。
- timestamp: 2026-06-30
  checked: `internal/services/operations/excel_export_service.go:154` resolveAssociations 实现
  found: `Where("deleted_at IS NULL")` 对所有 join.Table 无条件生效,错误被 logger.Warnf 后 `continue`。
  implication: 失败 join 静默丢数据(只 warn 不中断),这是端口名称空的关键路径。

## Eliminated

- hypothesis: WIP commit 引入的 RightCast 配置错误导致空 — 证据: RightCast 在 SQL WHERE 子句,WHERE 整体失败时整条 SQL 不会执行到 IN 子句。
  timestamp: 2026-06-30
- hypothesis: ops_info_points.port_id 列类型不匹配 — 证据: 模型 `PortID *string` + gorm:"size:64",DB 列是 VARCHAR(64) 存 UUID 字符串。GORM 扫描为 string。type assertion 通过。
  timestamp: 2026-06-30
- hypothesis: sys_device_port_status.interface_name 列不存在 — 证据: 模型 `InterfaceName string gorm:"size:100;not null;..."`,列必然存在。
  timestamp: 2026-06-30
