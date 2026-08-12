-- 修复知识库分类表约束问题
-- 执行此脚本后重启后端服务

-- 1. 删除旧的唯一约束（如果存在）
ALTER TABLE sys_knowledge_category DROP CONSTRAINT IF EXISTS uni_sys_knowledge_category_category_name;

-- 2. 删除可能存在的旧索引（如果存在）
DROP INDEX IF EXISTS uni_sys_knowledge_category_category_name;

-- 3. 创建新的唯一索引（使用 GORM 期望的名称）
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_category_name ON sys_knowledge_category(category_name);

-- 验证
SELECT 'Knowledge category constraints fixed successfully' AS result;
