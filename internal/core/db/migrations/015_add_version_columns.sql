-- ============================================
-- 添加 version 列到工单相关表
-- 文件: 015_add_version_columns.sql
-- 说明: BaseModel 包含 version 字段，用于乐观锁
-- ============================================

-- 为 sys_workorder 添加 version
ALTER TABLE sys_workorder ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_workorder_category 添加 version
ALTER TABLE sys_workorder_category ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_workorder_comment 添加 version
ALTER TABLE sys_workorder_comment ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_workorder_history 添加 version
ALTER TABLE sys_workorder_history ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_workorder_rating 添加 version
ALTER TABLE sys_workorder_rating ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_periodic_workorder_template 添加 version
ALTER TABLE sys_periodic_workorder_template ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_periodic_workorder_log 添加 version
ALTER TABLE sys_periodic_workorder_log ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_knowledge_category 添加 version
ALTER TABLE sys_knowledge_category ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- 为 sys_knowledge_article 添加 version
ALTER TABLE sys_knowledge_article ADD COLUMN IF NOT EXISTS version BIGINT DEFAULT 0;

-- ============================================
-- 迁移完成
-- ============================================

SELECT '015_add_version_columns.sql migration completed' AS status;
