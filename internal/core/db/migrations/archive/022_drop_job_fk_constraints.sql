-- 临时删除定时任务相关的外键约束
-- 这是为了让 GORM AutoMigrate 能够正常工作
-- 迁移完成后，GORM 会自动重新创建这些约束

-- 删除 sys_periodic_workorder_template 的 job_id 外键
ALTER TABLE sys_periodic_workorder_template DROP CONSTRAINT IF EXISTS fk_periodic_wo_template_job;

-- 删除 sys_periodic_workorder_log 的 job_id 外键
ALTER TABLE sys_periodic_workorder_log DROP CONSTRAINT IF EXISTS fk_periodic_wo_log_job;

-- 说明：
-- GORM 会根据模型中的关联关系自动重新创建这些外键约束
-- 执行此迁移后，重启服务让 GORM 完成自动迁移
