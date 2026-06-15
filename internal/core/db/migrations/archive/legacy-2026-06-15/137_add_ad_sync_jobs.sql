-- Migration 137: 添加AD域组成员同步定时任务
-- 将硬编码的AD组成员同步调度器改为数据库驱动的定时任务
-- 创建时间: 2026-05-27
--
-- 注意：AD域数据自动同步已有完整的控制机制（AD配置页面），
-- 不需要迁移到数据库定时任务系统。
-- 本迁移仅添加AD域组成员同步任务。

-- AD域组成员同步任务（每15分钟执行一次）
-- 使用 DO NOTHING 确保幂等性：如果任务已存在则跳过
INSERT INTO sys_job (
    id,
    job_name,
    job_group,
    invoke_target,
    cron_expression,
    misfire_policy,
    concurrent,
    status,
    remark
)
SELECT
    gen_random_uuid(),
    'AD域组成员同步',
    'DEFAULT',
    'dept_member_to_ad_group_sync',
    '0 */15 * * * *',
    1, -- 立即执行
    false,
    0, -- 正常状态
    '自动同步部门成员到对应的AD组（每15分钟）'
WHERE NOT EXISTS (
    SELECT 1 FROM sys_job
    WHERE invoke_target = 'dept_member_to_ad_group_sync'
    AND deleted_at IS NULL
);

-- 添加注释
COMMENT ON COLUMN sys_job.invoke_target IS '任务调用目标：dept_member_to_ad_group_sync=部门成员到AD组同步（系统部门成员自动同步到AD域组）';
