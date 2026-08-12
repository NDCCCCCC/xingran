# Legacy SQL Migrations Archive (2026-06-15)

本目录归档了 `internal/core/db/migrations/` 顶层的孤立 .sql 文件 + `scripts/migrate/` 整个一次性运维脚本目录。

## 归档原因

项目从未实现 .sql 文件自动加载器(`grep filepath.Walk|os.ReadDir|.sql` 在 `internal/core/db/` 无任何结果)。
顶层 ~125 个 .sql 文件分三种情况:

1. **被 Go migration 函数读取的**(`migration_*.go` 用 `os.ReadFile`)— 保留在顶层
2. **被一次性运维脚本读取的**(`scripts/migrate/*.go`,带 `//go:build ignore`)— 全部归档,因为这些脚本已事实废弃
3. **完全孤立的 .sql 文件** — 历史上手工 `psql -f xxx.sql` 执行,对应表/字段早已存在于生产 DB,
   现在只是历史档案 → **本次归档到这里**

## 顶层保留的 active .sql 清单(共 3 个)

| 文件 | 被谁读取 |
|---|---|
| `136_add_group_mapping_menu.sql` | `migration_136_group_mapping_menu.go:32` |
| `150_add_workstation_device_ip_address.sql` | `migration_150_add_workstation_device_ip_address.go:21` |
| `153_mac_heatmap_menu.sql` | `migration_153_mac_heatmap_menu.go:37` |

## scripts/migrate/ 已整目录归档

原 `scripts/migrate/` 目录(20+ 个一次性运维 Go 脚本,全部带 `//go:build ignore`)已 git mv 到:

```
scripts/.archive-migrate-2026-06-15/
```

依据:
- `scripts/README.md` 早已把 migrate/ 标注为"已执行的迁移脚本（归档）"
- 其中 `074_run_migration.go` 和 `113_fix_screenshots_column.go` 引用的 SQL **本就在 archive/** — 说明这套脚本已事实部分废弃
- 全部 `//go:build ignore` 标签 — 正常编译不会包含

恢复方法: `git mv scripts/.archive-migrate-2026-06-15 scripts/migrate`

## 已识别的真重复(归档时一并处理)

| 主题 | 文件 | 处理 |
|---|---|---|
| 创建 VDI 表 | `085_create_vdi_tables.sql` + `128_create_vdi_tables.sql` | 两份都归档;128 是较新版,与生产 schema 一致(表名带 sys_ 前缀, UUID 主键);085 是早期实验 |
| 移除 info_point.code 列 | `039_remove_info_point_code_column.sql` + `111_remove_info_point_code_column.sql` | 两份都归档;111 应是后续重做(多了 DROP INDEX) |

## 已识别的主题反复迭代(归档时保留全部历史)

不是真重复,是同一概念的多次修复/演化。归档不合并,以便追溯。

- **APIKey 菜单**(9 次):018 / 110 / 110_*_simple / 115_*_final / 116_remove / 119_simplify / 122_recreate / 123_*_simple
- **VDI 菜单**(3 次):129 / 130 / 131
- **ad_dn 列名 bug**(4 次):116_fix / 137_drop / 138_fix / 139_add
- **工位字段**(7+ 次):030 / 030_enhance / 034 / 035 / 036 / 090 / 101 / 034_remove
- **专线字段**(6 次):040 / 046 / 047 / 049 / 085_split / 115_migrate
- **建筑坐标**(2 次):029 / 031

## 如何恢复某个归档文件

```bash
git mv internal/core/db/migrations/archive/legacy-2026-06-15/<file>.sql \
       internal/core/db/migrations/<file>.sql
```

或者直接从 git 历史查看:
```bash
git log --all -- internal/core/db/migrations/<file>.sql
git show <commit>:internal/core/db/migrations/<file>.sql
```

## 注意事项

- **本目录的 .sql 文件不再被任何代码读取** — 删除它们也不会影响项目运行
- 保留是为了历史可追溯性(知道每个 schema 变更的原始 SQL)
- 如果以后某个 .sql 需要在新部署中被执行,应该:
  1. 写一个 Go migration `migration_NNN_*.go`,把 SQL 内嵌或 `os.ReadFile` 读取
  2. 在 `database.go` 的 `AutoMigrate()` 末尾追加调用
  3. 参考 `migration_117_create_mac_filter_rules.go` / `migration_148_create_ops_asset_table.go` 模式
- 同时按 `docs/开发规范.md §3.4` 唯一约束命名规范,显式命名 `CONSTRAINT uni_<table>_<col> UNIQUE`


## 已识别的真重复(归档时一并处理)

| 主题 | 文件 | 处理 |
|---|---|---|
| 创建 VDI 表 | `085_create_vdi_tables.sql` + `128_create_vdi_tables.sql` | 两份都归档;128 是较新版,与生产 schema 一致 |
| 移除 info_point.code 列 | `039_remove_info_point_code_column.sql` + `111_remove_info_point_code_column.sql` | 两份都归档;111 应是后续重做 |

## 已识别的主题反复迭代(归档时保留全部历史)

不是真重复,是同一概念的多次修复/演化。归档不合并,以便追溯。

- **APIKey 菜单**(9 次):018 / 110 / 110_*_simple / 115_*_final / 116_remove / 119_simplify / 122_recreate / 123_*_simple
- **VDI 菜单**(3 次):129 / 130 / 131
- **ad_dn 列名 bug**(4 次):116_fix / 137_drop / 138_fix / 139_add
- **工位字段**(7+ 次):030 / 030_enhance / 034 / 035 / 036 / 090 / 101 / 034_remove
- **专线字段**(6 次):040 / 046 / 047 / 049 / 085_split / 115_migrate
- **建筑坐标**(2 次):029 / 031

## 如何恢复某个归档文件

```bash
git mv internal/core/db/migrations/archive/legacy-2026-06-15/<file>.sql \
       internal/core/db/migrations/<file>.sql
```

或者直接从 git 历史查看:
```bash
git log --all -- internal/core/db/migrations/<file>.sql
git show <commit>:internal/core/db/migrations/<file>.sql
```

## 注意事项

- **本目录的 .sql 文件不再被任何代码读取** — 删除它们也不会影响项目运行
- 保留是为了历史可追溯性(知道每个 schema 变更的原始 SQL)
- 如果以后某个 .sql 需要在新部署中被执行,应该:
  1. 写一个 Go migration `migration_NNN_*.go`,把 SQL 内嵌或 `os.ReadFile` 读取
  2. 在 `database.go` 的 `AutoMigrate()` 末尾追加调用
  3. 参考 `migration_117_create_mac_filter_rules.go` / `migration_148_create_ops_asset_table.go` 模式
