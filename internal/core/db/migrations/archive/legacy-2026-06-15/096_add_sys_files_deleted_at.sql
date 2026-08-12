-- Migration: 096_add_sys_files_deleted_at.sql
-- Description: 为 sys_files 表添加 GORM 软删除支持所需的 deleted_at 列

-- 添加 deleted_at 列（GORM 软删除标准列）
ALTER TABLE sys_files
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 添加 file_width 和 file_height 列
ALTER TABLE sys_files
ADD COLUMN IF NOT EXISTS file_width INTEGER;

ALTER TABLE sys_files
ADD COLUMN IF NOT EXISTS file_height INTEGER;

-- 添加 metadata 列（JSONB 类型）
ALTER TABLE sys_files
ADD COLUMN IF NOT EXISTS metadata JSONB;

-- 添加 created_by 和 updated_by 列
ALTER TABLE sys_files
ADD COLUMN IF NOT EXISTS created_by VARCHAR(64);

ALTER TABLE sys_files
ADD COLUMN IF NOT EXISTS updated_by VARCHAR(64);

-- 添加 version 列
ALTER TABLE sys_files
ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 0;

-- 添加字段注释
COMMENT ON COLUMN sys_files.deleted_at IS '删除时间（GORM软删除）';
COMMENT ON COLUMN sys_files.file_width IS '图片宽度（像素）';
COMMENT ON COLUMN sys_files.file_height IS '图片高度（像素）';
COMMENT ON COLUMN sys_files.metadata IS '元数据（JSON格式）';
COMMENT ON COLUMN sys_files.created_by IS '创建者ID';
COMMENT ON COLUMN sys_files.updated_by IS '更新者ID';
COMMENT ON COLUMN sys_files.version IS '版本号';

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_files_deleted_at
    ON sys_files(deleted_at);

-- 验证迁移
SELECT
    '096_add_sys_files_deleted_at.sql' AS migration,
    'deleted_at and other columns added to sys_files' AS status;
