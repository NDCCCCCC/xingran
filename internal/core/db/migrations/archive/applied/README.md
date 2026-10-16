# migrations/archive/applied/

**已应用迁移的历史快照（已不再启动期调用）**

## 用途

此目录存放生产环境已应用过的 schema 演进迁移。每个 Go 文件定义了 `MigrateNNN<Description>(db *gorm.DB) error` 函数，**但 `database.go` 启动期不再调用它们**。

完整启动期调用清单见 `internal/core/db/database.go` 的 `AutoMigrate()`：

| 函数 | 启动期调用 |
|---|---|
| `Migrate175ReconciliationPhysicalLink` | ✅ |
| `Migrate176ReconciliationPhysicalMV` | ✅ |
| `Migrate202PortWriteAudit` | ✅ |
| `Migrate203ConnectionPoolSysConfig` | ✅ |
| `Migrate204AddDot1xUserLimit` | ✅ |

其余 65 个迁移函数（033 ~ 201 区间）只用于以下场景：

1. **跨环境手动 replay** —— 手动执行 `migrations.MigrateNNN(d.DB)` 重放历史变更
2. **审计追溯** —— schema 演进的真实历史（Phase 标记、incident 复盘、设计决策）
3. **回滚参考** —— 当新版本出现 regression 时参考历史变更

## 为什么不在启动期调用？

- 生产 DB schema 已稳定，重复执行是 idempotent noop，但每次启动会累积 ~30s 启动开销 + 250 行日志噪音（参考 `.planning/quick/260704-ne5-database-go-migration-schema-seed-snapsh/`）
- 新部署流程采用 `pg_dump snapshot.sh`，不依赖历史迁移 replay
- 历史迁移的真实价值是**知识沉淀**而非**运行时必需**

## 何时可以删除？

- 当 schema 完全稳定（如 v2.0 之后不再有 hot fix）
- 当所有历史迁移的注释价值已被 ADR/迁移文档吸收

**当前建议：保留**——archive 目录不参与编译（Go 编译器只构建顶层 + 显式 import 的子包），但保留全部历史注释便于回溯。

## 关联目录

| 目录 | 用途 |
|---|---|
| `../` (顶层) | 启动期调用的 5 个迁移 + 共享 helpers |
| `./legacy-2026-06-15/` | 早期 SQL 迁移（2026-06-15 前），大多已被 Go migration 替代 |
| `./init/` | 早期初始数据 seed SQL，已被 `database.go::initData()` 替代 |