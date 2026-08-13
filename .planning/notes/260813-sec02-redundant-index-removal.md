# SEC-02: 冗余索引 idx_api_keys_key 移除决策记录

**Phase**: 60-security-hardening-and-enable-decision (Plan 02 / Task 2)
**Requirement**: SEC-02（P3「冗余索引」性能债收敛）
**Date**: 2026-08-13
**锁定决策来源**: `.planning/phases/60-security-hardening-and-enable-decision/60-CONTEXT.md` D-10
**对照文件**: `internal/core/db/migrations/archive/applied/migration_085_api_keys.go`（带 `//go:build archive_skip`，不再编译执行）

---

## 1. 为什么（D-10）：手动运维 SQL vs Go migration

| 方案 | 可审计性 | 执行控制 | 与 archive_skip 模式一致性 | 决策 |
|------|----------|----------|---------------------------|------|
| **手动运维 SQL + notes + 验证查询** | 高（SQL 文件 + 决策记录 + 验证片段） | 运维窗口可控、可回滚 | 更清晰 | **✅ 选定** |
| 新建 Go migration（migration_086） | 中（代码即文档） | 随应用启动自动执行，运维不可控 | 与 migration_085 的 archive_skip 退役模式冲突 | ❌ |

**选手动 SQL 的理由**（用户偏离推荐的快路径决策）：

1. **决策快路径**：无需写 migration 注册逻辑 / 测 migration 幂等性 / 进 `SchemaMigrationRunner` 历史链；一个 SQL 文件 + 决策记录即可追溯。
2. **运维可控**：DROP INDEX 是 schema 级操作，应在运维窗口由 DBA 显式执行，而非应用启动时隐式触发——避免生产启动时意外锁表/阻塞。
3. **与 archive_skip 模式对齐**：migration_085 已标记 `archive_skip` 不再执行，继续往 `internal/core/db/migrations/` 加 Go migration 只会延长退役链条；手动 SQL 文件（`docs/operations/sql/`）是退役期 schema 运维的更清晰归属。

---

## 2. 怎么跑 SQL

**文件位置**: `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql`

**执行命令**（按 dialect）：

```bash
# PostgreSQL（生产）
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql

# SQLite（本地/测试）
sqlite3 <db_file> < docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql
```

**运维窗口选择**：低峰期执行。`DROP INDEX IF EXISTS` 是元数据操作（不重写表数据），几乎瞬时完成；唯一成本是持有 `sys_api_keys` 的 schema 锁，窗口选择为保险措施。

**幂等性**：`IF EXISTS` 关键字保证重复执行安全——索引不存在时静默跳过，不报错。

**回滚**：SQL 文件注释含回滚模板 `CREATE INDEX idx_api_keys_key ON sys_api_keys(key);`。DBA 执行前应先备份 schema。

---

## 3. 验证查询

DROP 后确认索引从 system catalog 消失。双 dialect 双片段：

### PostgreSQL

```sql
SELECT indexname FROM pg_indexes
WHERE tablename = 'sys_api_keys' AND indexname = 'idx_api_keys_key';
-- 预期: 0 行（DROP 成功）；非 0 行 = DROP 失败
```

### SQLite

```sql
SELECT name FROM sqlite_master
WHERE type = 'index' AND tbl_name = 'sys_api_keys' AND name = 'idx_api_keys_key';
-- 预期: 空（DROP 成功）；非空 = DROP 失败
```

---

## 4. AutoMigrate 行为说明

Phase 60 / SEC-01 后，`APIKey` struct 的 `Key string`（uniqueIndex）替换为 `KeyHash string`（uniqueIndex）：

| 阶段 | uniqueIndex 列 | 唯一索引名 | `idx_api_keys_key` 状态 |
|------|----------------|-----------|------------------------|
| SEC-01 前 | `key` | `sys_api_keys_key_key`（GORM 自动）+ `idx_api_keys_key`（migration_085 手动） | **冗余**（两个索引同列同约束） |
| SEC-01 后 | `key_hash` | `sys_api_keys_key_hash_key`（GORM 自动） | **残留**（`key` 列退役，该索引失去语义） |

**冗余根因**：migration_085 在 GORM uniqueIndex 之外又手动建了一个等价索引，INSERT/UPDATE 写路径需双索引维护（写放大）。SEC-02 收敛后，写路径仅维护 `sys_api_keys_key_hash_key` 一个索引。

**运维修剪窗口与代码部署解耦**：SEC-01 的 schema 变更（代码）与 SEC-02 的索引清理（运维）可独立执行——代码先部署（KeyHash 列启用），索引清理在后续运维窗口执行，两者无强时序依赖。

---

## 关联

- **SEC-01**（`.planning/notes/260813-sec01-hash-migration.md`）：Key 列退役是 SEC-02 索引收敛的前提。
- **migration_085**：`internal/core/db/migrations/archive/applied/migration_085_api_keys.go`（archive_skip，历史执行过，不再编译）。
