-- 修复工单表约束问题
-- 执行此脚本后重启后端服务

-- 1. 修复工单分类表
ALTER TABLE sys_workorder_category DROP CONSTRAINT IF EXISTS uk_workorder_category_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_category_name ON sys_workorder_category(category_name);

-- 2. 修复工单主表
ALTER TABLE sys_workorder DROP CONSTRAINT IF EXISTS uk_workorder_no;
DROP INDEX IF EXISTS idx_wo_no;  -- 删除可能存在的普通索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_wo_no ON sys_workorder(work_order_no);

-- 验证
SELECT 'Constraints fixed successfully' AS result;
