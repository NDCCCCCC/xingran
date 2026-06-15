-- Migration: 105_add_rpa_advanced_features.sql
-- Description: 添加 RPA 高级功能相关表 - 流程控制、错误处理、子流程
-- Created: 2025-02-25

-- 子流程定义表
CREATE TABLE IF NOT EXISTS sys_rpa_subprocesses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parameters JSONB DEFAULT '[]'::jsonb,     -- 参数定义列表
    actions JSONB NOT NULL,                    -- 子流程动作列表
    return_values JSONB DEFAULT '{}'::jsonb,   -- 返回值定义
    version INTEGER DEFAULT 1,
    status SMALLINT DEFAULT 0,                 -- 0=enabled 1=disabled
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64)
);

-- 流程执行记录表（支持条件、循环等）
CREATE TABLE IF NOT EXISTS sys_rpa_flow_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_execution_id UUID,                   -- 父执行ID
    subprocess_id UUID,                         -- 子流程ID（如果调用子流程）
    execution_id UUID NOT NULL,                 -- 关联的执行记录ID

    -- 流程信息
    flow_type VARCHAR(50) NOT NULL,             -- condition, loop, subprocess
    step_index INTEGER NOT NULL,                -- 步骤索引
    flow_name VARCHAR(255),                     -- 流程名称

    -- 执行状态
    status SMALLINT DEFAULT 0,                  -- 0=pending 1=running 2=completed 3=failed
    result JSONB,                               -- 执行结果

    -- 循环相关
    loop_type VARCHAR(50),
    loop_index INTEGER,
    loop_total INTEGER,

    -- 条件相关
    condition_expression TEXT,
    condition_result BOOLEAN,

    -- 错误处理
    error_strategy VARCHAR(50),
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 错误处理记录表
CREATE TABLE IF NOT EXISTS sys_rpa_error_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL,
    flow_execution_id UUID,
    step_index INTEGER,

    -- 错误信息
    error_type VARCHAR(100),
    error_message TEXT NOT NULL,
    error_stack TEXT,

    -- 处理信息
    handling_strategy VARCHAR(50),
    handling_result VARCHAR(50),
    recovery_action TEXT,

    -- 重试信息
    retry_attempt INTEGER,
    retry_delay INTEGER,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 数据映射模板表
CREATE TABLE IF NOT EXISTS sys_rpa_mapping_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    mapping_rules JSONB NOT NULL,              -- 映射规则
    mode VARCHAR(20) DEFAULT 'lenient',        -- strict/lenient

    -- 预览数据（用于测试）
    sample_input JSONB,
    sample_output JSONB,

    status SMALLINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_sys_rpa_subprocesses_name ON sys_rpa_subprocesses(name);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_subprocesses_status ON sys_rpa_subprocesses(status);

CREATE INDEX IF NOT EXISTS idx_sys_rpa_flow_executions_parent ON sys_rpa_flow_executions(parent_execution_id);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_flow_executions_execution ON sys_rpa_flow_executions(execution_id);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_flow_executions_type ON sys_rpa_flow_executions(flow_type);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_flow_executions_status ON sys_rpa_flow_executions(status);

CREATE INDEX IF NOT EXISTS idx_sys_rpa_error_logs_execution ON sys_rpa_error_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_error_logs_type ON sys_rpa_error_logs(error_type);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_error_logs_created ON sys_rpa_error_logs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sys_rpa_mapping_templates_name ON sys_rpa_mapping_templates(name);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_mapping_templates_status ON sys_rpa_mapping_templates(status);

-- 外键约束
ALTER TABLE sys_rpa_flow_executions
ADD CONSTRAINT IF NOT EXISTS fk_flow_parent_execution
FOREIGN KEY (parent_execution_id) REFERENCES sys_rpa_flow_executions(id) ON DELETE CASCADE;

ALTER TABLE sys_rpa_flow_executions
ADD CONSTRAINT IF NOT EXISTS fk_flow_execution
FOREIGN KEY (execution_id) REFERENCES sys_rpa_executions(id) ON DELETE CASCADE;

ALTER TABLE sys_rpa_flow_executions
ADD CONSTRAINT IF NOT EXISTS fk_flow_subprocess
FOREIGN KEY (subprocess_id) REFERENCES sys_rpa_subprocesses(id) ON DELETE SET NULL;

ALTER TABLE sys_rpa_error_logs
ADD CONSTRAINT IF NOT EXISTS fk_error_execution
FOREIGN KEY (execution_id) REFERENCES sys_rpa_executions(id) ON DELETE CASCADE;

-- 注释
COMMENT ON TABLE sys_rpa_subprocesses IS 'RPA 子流程定义表';
COMMENT ON TABLE sys_rpa_flow_executions IS 'RPA 流程执行记录表（支持条件、循环、子流程）';
COMMENT ON TABLE sys_rpa_error_logs IS 'RPA 错误处理记录表';
COMMENT ON TABLE sys_rpa_mapping_templates IS 'RPA 数据映射模板表';

COMMENT ON COLUMN sys_rpa_flow_executions.flow_type IS '流程类型：condition=条件分支, loop=循环, subprocess=子流程';
COMMENT ON COLUMN sys_rpa_flow_executions.loop_type IS '循环类型：count=计数, while=条件, forEach=遍历, until=直到';
COMMENT ON COLUMN sys_rpa_error_logs.handling_strategy IS '处理策略：ignore=忽略, retry=重试, rollback=回滚, skip=跳过, abort=中止, fallback=降级';
