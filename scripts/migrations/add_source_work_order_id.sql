-- 为知识库文章表添加来源工单ID字段
-- 用于记录文章是从哪个工单转换而来的

ALTER TABLE sys_knowledge_article
ADD COLUMN IF NOT EXISTS source_work_order_id UUID;

-- 添加注释
COMMENT ON COLUMN sys_knowledge_article.source_work_order_id IS '来源工单ID';
