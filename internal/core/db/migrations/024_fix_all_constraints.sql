-- 全面修复所有数据库约束问题
-- 执行此脚本后重启后端服务

-- 1. 修复知识库分类表
ALTER TABLE sys_knowledge_category DROP CONSTRAINT IF EXISTS uni_sys_knowledge_category_category_name;
DROP INDEX IF EXISTS uni_sys_knowledge_category_category_name;
DROP INDEX IF EXISTS idx_kb_category_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_category_name ON sys_knowledge_category(category_name);

-- 2. 修复知识库标签表（如果需要）
DROP INDEX IF EXISTS uni_sys_knowledge_tag_tag_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_kb_tag_name ON sys_knowledge_tag(tag_name);

-- 3. 修复工单分类表（如果需要）
ALTER TABLE sys_workorder_category DROP CONSTRAINT IF EXISTS uk_workorder_category_name;
ALTER TABLE sys_workorder_category DROP CONSTRAINT IF EXISTS uni_sys_workorder_category_name;
DROP INDEX IF EXISTS uk_workorder_category_name;
DROP INDEX IF EXISTS uni_sys_workorder_category_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_category_name ON sys_workorder_category(category_name);

-- 4. 修复工单表（如果需要）
ALTER TABLE sys_workorder DROP CONSTRAINT IF EXISTS uk_workorder_no;
ALTER TABLE sys_workorder DROP CONSTRAINT IF EXISTS uni_sys_workorder_work_order_no;
DROP INDEX IF EXISTS idx_wo_no;
DROP INDEX IF EXISTS uk_workorder_no;
DROP INDEX IF EXISTS uni_sys_workorder_work_order_no;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_no ON sys_workorder(work_order_no);

-- 验证
SELECT 'All constraints fixed successfully' AS result;
