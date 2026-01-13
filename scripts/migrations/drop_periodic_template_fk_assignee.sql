-- 删除周期性工单模板表的外键约束 fk_periodic_wo_template_assignee
-- 因为 AssignTargetID 现在可以存储两种类型的ID：
-- 1. 用户ID（当 assignType 为 'manual' 时）
-- 2. 值班池ID（当 assignType 为 'duty_pool' 时）

-- 删除外键约束
ALTER TABLE sys_periodic_workorder_template
DROP CONSTRAINT IF EXISTS fk_periodic_wo_template_assignee;
