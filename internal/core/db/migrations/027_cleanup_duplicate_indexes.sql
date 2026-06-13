-- Phase 32 P2-A4 source-tracking:
--   Original commit: 15cb483d
--   Created: 2026-01-09
--   Note: Conflicts with 027_create_user_column_config.sql — both share prefix 027. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- ============================================
-- 清理知识库和工单表的重复索引
-- 文件: 027_cleanup_duplicate_indexes.sql
-- 说明: 删除旧的重名约束和索引，保留新的索引
-- ============================================

-- ============================================
-- 1. 知识库分类表 - 清理重复的 category_name 唯一索引
-- ============================================
-- 删除旧的唯一约束（如果存在）
ALTER TABLE sys_knowledge_category DROP CONSTRAINT IF EXISTS uk_knowledge_category_name;

-- 删除旧的唯一索引（如果存在）
DROP INDEX IF EXISTS uk_knowledge_category_name;

-- 保留新的索引 idx_kb_category_name（已在 023 中创建）

-- ============================================
-- 2. 知识库标签表 - 清理重复的 tag_name 唯一索引
-- ============================================
-- 删除旧的唯一约束（如果存在）
ALTER TABLE sys_knowledge_tag DROP CONSTRAINT IF EXISTS uk_knowledge_tag_name;

-- 删除旧的唯一索引（如果存在）
DROP INDEX IF EXISTS uk_knowledge_tag_name;
DROP INDEX IF EXISTS uni_sys_knowledge_tag_tag_name;

-- 创建新的唯一索引（如果不存在）
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_tag_name ON sys_knowledge_tag(tag_name);

-- ============================================
-- 3. 工单分类表 - 清理重复的 category_name 唯一索引
-- ============================================
-- 删除旧的唯一约束（如果存在）
ALTER TABLE sys_workorder_category DROP CONSTRAINT IF EXISTS uk_workorder_category_name;
ALTER TABLE sys_workorder_category DROP CONSTRAINT IF EXISTS uni_sys_workorder_category_name;

-- 删除旧的唯一索引（如果存在）
DROP INDEX IF EXISTS uk_workorder_category_name;
DROP INDEX IF EXISTS uni_sys_workorder_category_name;

-- 创建新的唯一索引（如果不存在）
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_category_name ON sys_workorder_category(category_name);

-- ============================================
-- 4. 工单表 - 清理重复的 work_order_no 唯一索引
-- ============================================
-- 删除旧的唯一约束（如果存在）
ALTER TABLE sys_workorder DROP CONSTRAINT IF EXISTS uk_workorder_no;
ALTER TABLE sys_workorder DROP CONSTRAINT IF EXISTS uni_sys_workorder_work_order_no;

-- 删除旧的唯一索引（如果存在）
DROP INDEX IF EXISTS uk_workorder_no;
DROP INDEX IF EXISTS uni_sys_workorder_work_order_no;

-- 创建新的唯一索引（如果不存在）
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_no ON sys_workorder(work_order_no);

-- ============================================
-- 迁移完成
-- ============================================

SELECT '027_cleanup_duplicate_indexes.sql - 清理重复索引完成' AS status;
