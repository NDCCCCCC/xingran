---
quick_id: 260701-iface-sort
slug: iface-sort
date: 2026-07-01
status: complete
description: 端口采集接口名称列按"速率前缀 + 板卡号/子卡号/接口号"数值排序
---

# Quick Task 260701-iface-sort: 完成

## 成果

端口状态列表（`/network/port/list`）点击"接口名称"列头排序时，改为按**速率前缀 + 板卡号/子卡号/接口号**数值排序，修正了字符串字典序错误。

- **修正前**（字典序）：`GE0/1, GE0/10, GE0/11, GE0/2`（错误：'1' < '2' 让 GE0/10 排在 GE0/2 前）
- **修正后**（数值）：`GE0/1, GE0/2, GE0/10, GE0/11, GE1/1, GE1/2, GE1/10, GE1/11`

## 改动文件

| 文件 | 类型 | 说明 |
|------|------|------|
| `internal/services/portcollection/query.go` | 修改 | 新增 `interfaceNameSortExpr(dialectName, direction)`；GetList 排序分支对 `interfaceName` 特殊处理 |
| `internal/services/portcollection/query_test.go` | 新建 | 表达式生成测试 + 纯 Go 排序语义测试（用户示例、混合速率、无数字） |

## 技术决策

### 1. 必须用 SQL ORDER BY，不能用 Go sort.Slice

`GetList` 是 DB 层分页（`Count` → `Offset().Limit().Find()`）。Go 层 `sort.Slice` 只排序当前页，**跨页顺序仍乱**（第1页会混入本应第2页的记录）。必须 DB 层 ORDER BY 按数值，分页才正确。原 gsd-quick 计划里的 Go sort.Slice 方案被否决。

### 2. PostgreSQL int 数组比较 = 元组比较

`regexp_matches(interface_name,'([0-9]+)','g')` 提取数字段为 int 数组，PG 数组逐元素比较天然等价 `(板卡号,子卡号,接口号)` 元组比较，两段（`GE0/1`）和三段（`GE0/0/1`）格式都自动正确。`COALESCE(..., ARRAY[]::int[])` 兜底无数字接口名（Vlan/NULL）为空数组。

### 3. dialect-aware（PG/SQLite 兼容）

`regexp_matches` 是 PG 专属。遵循项目惯例（`reconciliation_service.go` 用 `db.Dialector.Name()` 分支）：PG 走正则数组数值排序，SQLite/其他降级为字符串排序。端口采集主库为 PG，SQLite 仅开发/测试，降级可接受。

### 4. 三个排序键

1. **速率前缀**（`regexp_replace` 取首数字前字母）：GE/XGE/HGE/FOE/FE 分组，避免 FE0/2 夹在 GE0/1 与 GE0/3 间交替
2. **数字段 int 数组**：板卡/子卡/接口数值
3. **interface_name**：稳定 tiebreaker

### 5. 避开 pkg/normalize 未提交重构

`pkg/normalize/` 是别人正在进行、未提交的重构（归一化下沉共享包）。本任务**不依赖、不修改**它，逻辑自包含在 `query.go`，避免与未完成工作耦合。

## 验证

```
go build ./...                                    # exit 0
go vet ./internal/services/portcollection/        # 无问题
go test ./internal/services/portcollection/       # PASS (含现有 NormalizeInterfaceName 全部测试)
```

新增 4 个测试函数全绿：
- `TestInterfaceNameSortExpr`（5 子测试：postgres/sqlite/mysql × ASC/DESC 表达式生成）
- `TestInterfaceNameSortSemantics`（用户示例 ASC + DESC 严格逆序）
- `TestInterfaceNameSortSemantics_MixedRate`（FE/GE/XGE 分组不交替）
- `TestInterfaceNameSortSemantics_NoDigits`（Vlan/NULL 不崩溃，排末尾）

**注**：SQL 数值排序语义的正确性由纯 Go 测试镜像证明（`digitRe`/`prefixRe` 用与生产 SQL 相同的正则语义），不依赖真实 PostgreSQL 连接。

## 影响面

GetList 两个调用点都受益（改一处）：
- `port_handler.go:69` 端口列表（前端点击列头触发服务端排序）
- `batch_export_helper.go:440` 批量导出（导出 Excel 也按数值排序）

前端无需改动：`ports/index.tsx` 已配置 `createSorterMeta("interfaceName")` + `useServerSort`。

## 生产 PG 验证查询（建议上线后抽样核对）

```sql
-- 验证数值排序生效(应看到 GE0/2 在 GE0/10 之前)
SELECT interface_name FROM sys_device_port_status
WHERE device_id = '<某设备>'
ORDER BY
  regexp_replace(interface_name, '[0-9].*$', '') ASC,
  (SELECT COALESCE(array_agg(m[1]::int), ARRAY[]::int[]) FROM regexp_matches(interface_name, '([0-9]+)', 'g') AS m) ASC,
  interface_name ASC
LIMIT 50;
```

## scope 声明

本任务**未触碰**工作区已有的未提交改动（均为别人进行中的工作）：
- `internal/services/portcollection/utils.go` / `utils_test.go`（migration_187 乱码清理）
- `internal/core/db/migrations/migration_187_clean_garbled_interface_names.go`
- `pkg/normalize/`（归一化下沉重构，未跟踪）
