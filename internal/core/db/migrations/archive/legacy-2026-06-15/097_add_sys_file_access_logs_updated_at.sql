-- Migration: 097_add_sys_file_access_logs_updated_at.sql
-- Description: 为 sys_file_access_logs 表添加 updated_at 列

-- 添加 updated_at 列
ALTER TABLE sys_file_access_logs
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- 添加字段注释
COMMENT ON COLUMN sys_file_access_logs.updated_at IS '更新时间';

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_file_logs_updated_at
    ON sys_file_access_logs(updated_at DESC);

-- 验证迁移
SELECT
    '097_add_sys_file_access_logs_updated_at.sql' AS migration,
    'updated_at column added to sys_file_access_logs' AS status;
