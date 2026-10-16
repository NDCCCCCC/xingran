-- Phase 60 / SEC-02: 移除 migration_085 遗留的冗余索引 idx_api_keys_key
-- 触发原因: sys_api_keys.key 列的 uniqueIndex 与 idx_api_keys_key（由 migration_085 创建）功能重复；Phase 60 / SEC-01 将 key 列替换为 KeyHash/Salt/KeyPrefix 后，GORM AutoMigrate 自动生成 sys_api_keys_key_hash_key（uniqueIndex），idx_api_keys_key 与之冗余
-- 风险: 无 — uniqueIndex（sys_api_keys_key_hash_key）保留，索引扫描性能不变，索引覆盖查询不变
-- 回滚: CREATE INDEX idx_api_keys_key ON sys_api_keys(key);
-- 幂等: IF EXISTS 关键字保证重复执行安全
-- 验证查询: 见 .planning/notes/260813-sec02-redundant-index-removal.md「验证查询」段（PostgreSQL pg_indexes + SQLite sqlite_master 双片段）
-- 执行模式: 手动运维 — 生产 DB 在运维窗口执行；不进 Go migration runner（D-10 决策）

DROP INDEX IF EXISTS idx_api_keys_key;
