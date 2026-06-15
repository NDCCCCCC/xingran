-- Migration 088: 修复AD域组成员同步任务调用目标
-- 修复问题：数据库中存在旧任务记录，invoke_target 为 ad_group_member_sync
-- 但代码中注册的处理器名称为 dept_member_to_ad_group_sync，导致任务执行失败
-- 创建时间: 2026-05-27

-- 将旧的 invoke_target 更新为正确的名称
UPDATE sys_job
SET invoke_target = 'dept_member_to_ad_group_sync',
    remark = '自动同步部门成员到对应的AD组（每15分钟）'
WHERE invoke_target = 'ad_group_member_sync'
AND deleted_at IS NULL;

-- 添加注释
COMMENT ON COLUMN sys_job.invoke_target IS '任务调用目标：dept_member_to_ad_group_sync=部门成员到AD组同步（系统部门成员自动同步到AD域组）';
