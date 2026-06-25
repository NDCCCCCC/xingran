---
slug: ops-asset-constraint-uni-ops-asset-devicesn-not-exist
status: resolved
trigger: GORM AutoMigrate DROP uni_ops_asset_devicesn 不存在导致约束 DROP 失败
created: 2026-06-15
updated: 2026-06-25
session_type: bug
related:
  - sys-mac-filter-rules-relation-does-not-exist
---

# GORM AutoMigrate DROP uni_ops_asset_devicesn 不存在

**状态**: 🔧 Fix Applied (待用户重启验证)
**首次报告**: 2026-06-15 09:17:48
**关联**: [[sys-mac-filter-rules-relation-does-not-exist]](sys-mac-filter-rules-relation-does-not-exist.md)

---

## 症状

`go run .\cmd\main.go` 启动失败:

```
ERRO [GORM错误] ALTER TABLE "ops_asset" DROP CONSTRAINT "uni_ops_asset_devicesn"
  | 错误: ERROR: constraint "uni_ops_asset_devicesn" of relation "ops_asset" does not exist (SQLSTATE 42704)
FATA 初始化核心模块失败: 数据库迁移失败: ERROR: constraint "uni_ops_asset_devicesn" of relation "ops_asset" does not exist
```

## 根因

**Schema 与 GORM 模型对 unique 约束的命名规范不一致**:

| 来源 | 命名规范 | 实际约束名 |
|---|---|---|
| `migration_148_create_ops_asset_table.go` (SQL inline `UNIQUE`) | PostgreSQL 自动 | `ops_asset_devicesn_key` |
| `models.Asset.DeviceSN` 标 `uniqueIndex` | GORM 拼接 `uni_<table>_<column>` | `uni_ops_asset_devicesn` |

→ 用户 DB 里实际存在的是 PG 自动命名 `ops_asset_devicesn_key`。

→ AutoMigrate 时 GORM 在 `MigrateColumn` 中检测 column unique tag 与 schema 状态需要重建(可能是它判断 column type 改变,或它根据自己的命名规范找不到对应约束),发出:
```sql
ALTER TABLE "ops_asset" DROP CONSTRAINT "uni_ops_asset_devicesn"  -- 不存在 → 42704
```

GORM 的 `Migrator.DropConstraint` **没有 IF EXISTS**,所以 DROP 失败直接报错向上传播。`AutoMigrate` 中断 → `Database.AutoMigrate` 返回 err → core 初始化失败 → FATA。

## 修复(已应用)

在 `internal/core/db/database.go:cleanupOldConstraints()` 兜底清单加入两条:

```go
// 资产管理表(ops_asset.devicesn):
// migration_148 用 SQL inline `UNIQUE` 创建,PG 自动命名为 ops_asset_devicesn_key;
// models.Asset.DeviceSN 标 `uniqueIndex`,GORM 期望 uni_ops_asset_devicesn。
// 两种命名都先清理,让 GORM AutoMigrate 重新创建 uniqueIndex 时无冲突。
{"ops_asset", "uni_ops_asset_devicesn"},
{"ops_asset", "ops_asset_devicesn_key"},
```

`cleanupOldConstraints` 已有的实现是"先 SELECT pg_constraint 检查 count 再 DROP",所以对不存在的约束安全跳过。

### 修复机理

1. **cleanup 阶段**(AutoMigrate 之前):drop 实际存在的 `ops_asset_devicesn_key` → 表上 devicesn 列无 unique 约束
2. **AutoMigrate 阶段**:GORM 看到 column 无 unique → 走 ADD path(`CREATE UNIQUE INDEX uni_ops_asset_devicesn`),不再走 DROP+ADD path,避开了不存在 DROP 的错误
3. 后续启动:`uni_ops_asset_devicesn` 已存在,与模型一致,AutoMigrate 无操作

## 验证

- ✅ `go build ./...` 通过
- ⏳ 待用户重新执行 `go run .\cmd\main.go` 确认不再 FATA

## 长期改进(未在此次修复范围)

**根本一致化**:让 migration_148 的 SQL 使用 GORM 命名,或让 model 显式指定索引名,从源头消除命名不一致。两选一:

**方案 A**(改 SQL):
```sql
devicesn VARCHAR(200) NOT NULL,
CONSTRAINT uni_ops_asset_devicesn UNIQUE (devicesn)
```

**方案 B**(改 model):
```go
DeviceSN string `gorm:"size:200;not null;uniqueIndex:ops_asset_devicesn_key;column:devicesn"`
```

但当前 cleanup 兜底已能解决启动问题,长期改造可以等下次迭代再做。

## 系统性问题(再次提醒)

GORM AutoMigrate 与手写 SQL migration 混用时,对约束/索引命名的同步是一个**长期脆弱点**。本项目里 `sys_knowledge_*`、`sys_workorder*`、现在又有 `ops_asset` 都遇到过同类问题(`cleanupOldConstraints` 清单越来越长就是证据)。

建议:
1. 新增表统一用 GORM AutoMigrate(模型为单一来源),除非 SQL 必须(种子数据、复杂约束)
2. SQL 创建表时所有约束都**显式命名**(`CONSTRAINT <name> UNIQUE (...)`),让命名与 GORM 保持一致
3. `cleanupOldConstraints` 实现可以增强 — 用 `ALTER TABLE ... DROP CONSTRAINT IF EXISTS`(PG 9.0+ 支持)替代"先 SELECT 后 DROP"两步
