---
slug: sys-mac-filter-rules-relation-does-not-exist
status: resolved
trigger: sys_mac_filter_rules 表不存在 SQLSTATE 42P01
created: 2026-06-14
updated: 2026-06-25
session_type: bug
---

# sys_mac_filter_rules 表不存在 (SQLSTATE 42P01)

**状态**: ✅ Resolved
**首次报告**: 2026-06-14 02:00:02
**修复**: 2026-06-15

---

## 症状

MAC 采集任务执行时(`device_type=switch`),GORM 反复抛出错误:

```
ERROR: relation "sys_mac_filter_rules" does not exist (SQLSTATE 42P01)
```

涉及查询:
```sql
SELECT * FROM "sys_mac_filter_rules"
WHERE (device_type = 'switch' AND vendor = 'ruijie')
  AND "sys_mac_filter_rules"."deleted_at" IS NULL
ORDER BY priority DESC, "sys_mac_filter_rules"."id" LIMIT 1
```

虽然采集流程继续(回退到了默认阈值 10),但日志被错误污染,且过滤规则无法工作。

## 根因

**SQL 迁移文件未被执行 — 项目缺少 SQL 自动加载机制**。

证据链:

1. `internal/core/db/migrations/117_create_mac_filter_rules.sql` — 定义了完整建表 + 5 条系统默认规则
2. `internal/models/mac_filter_rule.go` — `MACFilterRule` 模型,`TableName()` → `sys_mac_filter_rules`
3. `internal/services/topology/filter_rules.go:230` `GetEffectiveRule()` — 通过 GORM 查询该模型
4. `internal/core/db/database.go:211-292` `AutoMigrate()` 模型列表 — **没有** `&models.MACFilterRule{}`
5. `database.go:299-337` Go 迁移调用清单 — **没有** `Migrate117*`
6. `grep -rn "filepath.Walk|os.ReadDir|.sql" internal/core/db/` — **没有** 任何 SQL 文件加载器

→ 117 是孤立 SQL 文件,模型未注册到 AutoMigrate,服务初次查询时表根本不存在。

## 修复

新增 Go 迁移并接入 AutoMigrate 链路,遵循项目对"含约束 + 种子数据"的表惯例(参考 `migration_148_create_ops_asset_table.go`):

**新增文件**: `internal/core/db/migrations/migration_117_create_mac_filter_rules.go`
- `Migrate117CreateMacFilterRules(db *gorm.DB) error`
- 使用 `db.Migrator().HasTable("sys_mac_filter_rules")` 做幂等
- 建表 SQL 与 117_create_mac_filter_rules.sql 等价(含 `chk_mac_threshold`、`chk_priority`、`uq_device_vendor` 约束 + 4 个索引)
- 5 条默认规则 INSERT,使用 `ON CONFLICT (device_type, vendor) DO NOTHING` 防重

**修改**: `internal/core/db/database.go` `AutoMigrate()` 末尾(在 150 之后)新增调用:
```go
if err := migrations.Migrate117CreateMacFilterRules(d.DB); err != nil {
    applogger.Errorf("MAC 过滤规则表迁移失败: %v", err)
}
```

## 验证

- `go build ./internal/core/db/...` ✅
- `go build ./internal/... ./pkg/... ./cmd/...` ✅
- 表创建后,`GetEffectiveRule()` 步骤 1(厂商 + 设备类型)未命中会回退到步骤 2(仅设备类型,vendor IS NULL),命中默认规则:
  - switch → 阈值 10,启用 LLDP 过滤
  - router → 阈值 500
  - firewall → 阈值 100
  - loadbalancer → 阈值 50
  - ap → 阈值 100

## 部署生效路径

- **新部署 / 重建数据库**: AutoMigrate 调用 117 → 表创建 + 种子。
- **现有部署(本环境)**: 重启服务后 117 触发 → `HasTable` 返回 false → 建表 + 插入种子。

## 关联

- 相关代码: `[[internal/services/topology/filter_rules.go]]` `GetEffectiveRule`
- 模型: `[[internal/models/mac_filter_rule.go]]`
- API: `[[internal/api/v1/network/topology_handler.go]]`

## 系统性问题(已识别但未在此次修复范围内)

`internal/core/db/migrations/` 目录里仍有大量 `.sql` 文件未被任何代码引用(grep `filepath.Walk` 无结果)。其中可能存在与 117 类似的"孤立 SQL"。建议后续做一次审计:
```bash
ls internal/core/db/migrations/*.sql | while read f; do
  num=$(basename "$f" | grep -oE '^[0-9]+')
  grep -l "Migrate${num}\|migration_${num}" internal/core/db/migrations/*.go || echo "孤立: $f"
done
```
若孤立 SQL 较多,建议添加统一的 SQL 文件加载器(按文件名排序执行 + 用 `sys_schema_migrations` 表幂等记录)。

## 题外副作用(非此 bug 引入,但 build 时会暴露)

`go build ./...` 会被以下预存在文件污染,与本次修复无关:
- 根目录残留 `test_params.go` / `check_ops_constraints.go` → `main redeclared`
- `internal/services/system/apikey_service_test.go` 与 `role_service_apperrors_test.go` → `setupTestDB redeclared`
