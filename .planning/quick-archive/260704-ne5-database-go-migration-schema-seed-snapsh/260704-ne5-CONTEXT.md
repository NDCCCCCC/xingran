# Quick Task 260704-ne5: database.go 启动期 migration 调用精简 + snapshot 脚本 - Context

**Gathered:** 2026-07-04
**Status:** Ready for planning

<domain>
## Task Boundary

启动期 `internal/core/db/database.go::AutoMigrate()` 当前跑 200+ 个 migration 函数调用，~30s 启动开销 + 250 行启动日志噪音。所有 migration 已在生产 DB 跑过，行为完全 idempotent / noop。新部署流程改为"pg_dump 导出 schema + seed → 一键导入新库"。

代码改动范围严格限定：
- 修改：`internal/core/db/database.go`
- 新增：`scripts/db/snapshot.sh`（生成 schema/seed snapshot 的工具）
- 不动：`internal/core/db/migrations/` 整个目录（保留作为开发演进参考）
- 不动：service 初始化（设备采集 worker / cron 注册 / 缓存预热）
</domain>

<decisions>
## Implementation Decisions

### 1. database.go 改造范围
- **删除**：`dropDependentMaterializedViews()` 调用（schema snapshot 已包含 MV 定义）
- **删除**：`migrateCredentialModel()` 调用（生产已迁移完）
- **删除**：~200 行 `migrations.MigrateNNN...(d.DB)` 调用（Migrate143 到 Migrate201 + 117 + 033）
- **保留**：`cleanupOldConstraints()` 调用（无害，~5ms）
- **保留**：GORM `AutoMigrate(model 列表)`（model 加新字段的安全网）
- **保留**：所有 200+ migration 函数定义本身（在 `internal/core/db/migrations/`）

### 2. 新部署流程
- 使用 `pg_dump --schema-only --no-owner --no-acl -t 'sys_*' -t 'ops_*' xingran_next > schema-snapshot.sql`
- 使用 `pg_dump --data-only --inserts --no-owner --no-acl -t sys_menu -t sys_role -t sys_role_menu -t sys_user -t sys_config -t sys_dict_type -t sys_dict_data -t sys_ad_service_accounts -t sys_post xingran_next > seed-snapshot.sql`
- 新部署：`psql -f schema-snapshot.sql && psql -f seed-snapshot.sql && ./xingran-backend`

### 3. 保留 service 层初始化
- 设备信息采集 worker 启动（每次启动必须）
- MAC 历史表分区检查（每次启动必须）
- 缓存预热 + 配置加载（每次启动必须）
- 调度任务注册（每次启动必须）
- 所有 cron 任务（已通过 service init 注册）

### 4. 不在范围内（明确不做）
- 不删 `internal/core/db/migrations/` 任何文件（保留 git 历史）
- 不动 GORM AutoMigrate(model)（保留作为安全网）
- 不新建 `sys_schema_migrations` 版本表（用户决策：直接用 schema snapshot 部署，不需要版本表）
- 不写迁移监控 / 健康检查脚本（任务边界外）

### 5. Claude's Discretion
- snapshot.sh 的输出文件名规范（决定用 `schema-{YYYY-MM-DD}.sql` + `seed-{YYYY-MM-DD}.sql`）
- 是否压缩（决定不压缩，便于人工 review）
</decisions>

<specifics>
## Specific Ideas

- **文件命名**：`scripts/db/snapshot.sh`（新建 `scripts/db/` 子目录，区别于现有 `scripts/migrations/` 历史归档）
- **snapshot 文件落点**：`docs/deployment/snapshots/`（gitignore 由项目级决定，但建议提交一份作为"最近一次稳定快照"参考）
- **README 更新**：在 `docs/deployment/` 简述新部署流程（用户手动做，executor 不动 docs 目录）

### 关键约束（来自用户决策）
- ✅ "新部署只需要一次性的建表和导数据就行了阿不需要依赖这么多迁移"
- ✅ "有些迁移是必须每次启动执行的吧，请先确认"——已确认结果：**所有 200+ migration 都是 schema 演进期，每次启动都是 noop**
- ✅ 已经在前面对话里做 A1+A2+C1 三项启动日志优化（A2 约束审计 DEBUG、A1 CACHE_CONFIG 重复加载去除、C1 worker 启动日志 DEBUG）——这些**保留不动**
</specifics>

<canonical_refs>
## Canonical References

- `internal/core/db/database.go` — 待修改文件
- `internal/core/db/migrations/` — 200+ migration 函数定义（保留不动）
- `internal/core/db/init_data.sql` — 现有 seed 文件，snapshot.sh 不动它
- `internal/core/db/create_monitor_tables.sql` — 现有部分建表，snapshot.sh 不动它
- 前面对话上下文：用户决策"schema snapshot 部署 + 启动跳过 migration"
</canonical_refs>