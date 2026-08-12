-- ============================================
-- 添加 deleted_at 列到工单和知识库相关表
-- 文件: 014_add_deleted_at_columns.sql
-- 说明: GORM 软删除需要 deleted_at 列
-- ============================================

-- 为 sys_workorder_category 添加 deleted_at
ALTER TABLE sys_workorder_category ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_workorder 添加 deleted_at
ALTER TABLE sys_workorder ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_workorder_comment 添加 deleted_at
ALTER TABLE sys_workorder_comment ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_workorder_history 添加 deleted_at
ALTER TABLE sys_workorder_history ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_workorder_rating 添加 deleted_at
ALTER TABLE sys_workorder_rating ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_periodic_workorder_template 添加 deleted_at
ALTER TABLE sys_periodic_workorder_template ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_periodic_workorder_log 添加 deleted_at
ALTER TABLE sys_periodic_workorder_log ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_knowledge_category 添加 deleted_at
ALTER TABLE sys_knowledge_category ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_knowledge_tag 添加 deleted_at
ALTER TABLE sys_knowledge_tag ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_knowledge_article_tag 添加 deleted_at
ALTER TABLE sys_knowledge_article_tag ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 为 sys_knowledge_article 添加 deleted_at
ALTER TABLE sys_knowledge_article ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

-- 创建索引（GORM 会为 deleted_at 创建索引）
CREATE INDEX IF NOT EXISTS idx_workorder_category_deleted ON sys_workorder_category(deleted_at);
CREATE INDEX IF NOT EXISTS idx_workorder_deleted ON sys_workorder(deleted_at);
CREATE INDEX IF NOT EXISTS idx_workorder_comment_deleted ON sys_workorder_comment(deleted_at);
CREATE INDEX IF NOT EXISTS idx_workorder_history_deleted ON sys_workorder_history(deleted_at);
CREATE INDEX IF NOT EXISTS idx_workorder_rating_deleted ON sys_workorder_rating(deleted_at);
CREATE INDEX IF NOT EXISTS idx_periodic_wo_template_deleted ON sys_periodic_workorder_template(deleted_at);
CREATE INDEX IF NOT EXISTS idx_periodic_wo_log_deleted ON sys_periodic_workorder_log(deleted_at);
CREATE INDEX IF NOT EXISTS idx_knowledge_category_deleted ON sys_knowledge_category(deleted_at);
CREATE INDEX IF NOT EXISTS idx_knowledge_tag_deleted ON sys_knowledge_tag(deleted_at);
CREATE INDEX IF NOT EXISTS idx_knowledge_article_tag_deleted ON sys_knowledge_article_tag(deleted_at);
CREATE INDEX IF NOT EXISTS idx_knowledge_article_deleted ON sys_knowledge_article(deleted_at);

-- ============================================
-- 迁移完成
-- ============================================

SELECT '014_add_deleted_at_columns.sql migration completed' AS status;
