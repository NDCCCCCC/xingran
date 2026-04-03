-- +migrate Up
-- 创建 RPA 人工干预事件表
CREATE TABLE IF NOT EXISTS sys_rpa_human_interventions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- 关联信息
    execution_id UUID NOT NULL,
    worker_id VARCHAR(100),

    -- 干预信息
    action VARCHAR(20) NOT NULL,  -- pause, resume, skip, abort
    message TEXT,
    input_data JSONB,             -- 用户输入的数据 (验证码、选项等)
    reason TEXT,

    -- 状态
    status VARCHAR(20) DEFAULT 'pending',  -- pending, processed, timeout
    processed_at TIMESTAMP,

    CONSTRAINT chk_rpa_hi_action CHECK (action IN ('pause', 'resume', 'skip', 'abort')),
    CONSTRAINT chk_rpa_hi_status CHECK (status IN ('pending', 'processed', 'timeout'))
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_rpa_hi_exec ON sys_rpa_human_interventions(execution_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_hi_worker ON sys_rpa_human_interventions(worker_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_hi_status ON sys_rpa_human_interventions(status, created_at) WHERE deleted_at IS NULL;

-- 添加注释
COMMENT ON TABLE sys_rpa_human_interventions IS 'RPA 人工干预事件表';
COMMENT ON COLUMN sys_rpa_human_interventions.execution_id IS '执行记录ID';
COMMENT ON COLUMN sys_rpa_human_interventions.worker_id IS 'Worker ID';
COMMENT ON COLUMN sys_rpa_human_interventions.action IS '干预类型: pause-暂停等待人工输入, resume-恢复执行, skip-跳过当前项, abort-中止整个任务';
COMMENT ON COLUMN sys_rpa_human_interventions.input_data IS '用户输入的数据，如验证码、选项等';

-- 为执行记录表添加批量报告字段
ALTER TABLE sys_rpa_executions
ADD COLUMN IF NOT EXISTS batch_report JSONB,
ADD COLUMN IF NOT EXISTS total_items INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS success_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS failed_count INTEGER DEFAULT 0;

COMMENT ON COLUMN sys_rpa_executions.batch_report IS '批量执行详细报告 (JSON格式)';
COMMENT ON COLUMN sys_rpa_executions.total_items IS '批量处理的总项目数';
COMMENT ON COLUMN sys_rpa_executions.success_count IS '成功处理的项目数';
COMMENT ON COLUMN sys_rpa_executions.failed_count IS '失败处理的项目数';

-- +migrate Down
-- 删除索引
DROP INDEX IF EXISTS idx_rpa_hi_status;
DROP INDEX IF EXISTS idx_rpa_hi_worker;
DROP INDEX IF EXISTS idx_rpa_hi_exec;

-- 删除表
DROP TABLE IF EXISTS sys_rpa_human_interventions;

-- 删除添加的字段
ALTER TABLE sys_rpa_executions
DROP COLUMN IF EXISTS failed_count,
DROP COLUMN IF EXISTS success_count,
DROP COLUMN IF EXISTS total_items,
DROP COLUMN IF EXISTS batch_report;
