-- 添加通知结束时间字段，用于周期性通知的自动停止
-- 执行时间: 2026-01-19

-- 添加 end_date 字段到 sys_notice 表
ALTER TABLE sys_notice ADD COLUMN IF NOT EXISTS end_date TIMESTAMP;

-- 添加注释
COMMENT ON COLUMN sys_notice.end_date IS '周期性通知结束时间，超过此时间后自动停止定时任务';
