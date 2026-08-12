-- ============================================
-- 统一定时任务表时间戳字段命名
-- 迁移版本: 029
-- ============================================

-- 1. 删除旧的 create_time/update_time 字段（如果存在）
ALTER TABLE sys_job DROP COLUMN IF EXISTS create_time;
ALTER TABLE sys_job DROP COLUMN IF EXISTS update_time;

-- 2. 添加 next_run_time 字段（如果不存在）
ALTER TABLE sys_job ADD COLUMN IF NOT EXISTS next_run_time timestamp;

-- 3. 添加 prev_run_time 字段（如果不存在）
ALTER TABLE sys_job ADD COLUMN IF NOT EXISTS prev_run_time timestamp;

-- 4. 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_sys_job_next_run_time ON sys_job(next_run_time);
CREATE INDEX IF NOT EXISTS idx_sys_job_prev_run_time ON sys_job(prev_run_time);

-- 5. 添加注释
COMMENT ON COLUMN sys_job.created_at IS '创建时间（GORM管理）';
COMMENT ON COLUMN sys_job.updated_at IS '更新时间（GORM管理）';
COMMENT ON COLUMN sys_job.next_run_time IS '下次执行时间';
COMMENT ON COLUMN sys_job.prev_run_time IS '上次执行时间';
