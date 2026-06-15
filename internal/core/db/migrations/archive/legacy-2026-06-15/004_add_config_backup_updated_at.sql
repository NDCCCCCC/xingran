-- 004_add_config_backup_updated_at.sql
-- 为配置备份表添加 updated_at 字段

-- 添加 updated_at 字段（允许 NULL 以便处理现有记录）
ALTER TABLE sys_config_backup ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- 为现有记录设置 updated_at 等于 created_at
UPDATE sys_config_backup SET updated_at = created_at WHERE updated_at IS NULL OR updated_at = '1970-01-01 00:00:00'::TIMESTAMP;

-- 将字段设置为 NOT NULL（所有现有记录都已填充）
ALTER TABLE sys_config_backup ALTER COLUMN updated_at SET NOT NULL;

-- 添加注释
COMMENT ON COLUMN sys_config_backup.updated_at IS '更新时间';
