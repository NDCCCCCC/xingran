-- RPA 系统数据库迁移
-- 创建日期: 2025-02-25
-- 版本: 1.0.0

-- ==================== 任务表 ====================
CREATE TABLE IF NOT EXISTS sys_rpa_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    script JSONB NOT NULL,
    trigger_type VARCHAR(20) NOT NULL DEFAULT 'manual',
    cron_expr VARCHAR(100),
    schedule_id UUID,
    status SMALLINT NOT NULL DEFAULT 0,
    priority SMALLINT DEFAULT 5,
    timeout_seconds INTEGER DEFAULT 300,
    retry_count SMALLINT DEFAULT 0,
    ai_generated BOOLEAN DEFAULT false,
    tags VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0
);

-- ==================== Worker 节点表 ====================
CREATE TABLE IF NOT EXISTS sys_rpa_workers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    node_type VARCHAR(20) NOT NULL DEFAULT 'server',
    capabilities JSONB NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    last_heartbeat TIMESTAMP,
    current_tasks INTEGER DEFAULT 0,
    total_tasks_executed INTEGER DEFAULT 0,
    total_execution_time BIGINT DEFAULT 0,
    ip_address VARCHAR(45),
    port INTEGER,
    version VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- ==================== 执行记录表 ====================
CREATE TABLE IF NOT EXISTS sys_rpa_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL,
    task_name VARCHAR(255),
    worker_id UUID,
    worker_name VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    duration INTEGER,
    progress_current INTEGER DEFAULT 0,
    progress_total INTEGER DEFAULT 0,
    screenshots TEXT[],
    logs TEXT,
    error_message TEXT,
    retry_count SMALLINT DEFAULT 0,
    triggered_by VARCHAR(64),
    trigger_type VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- ==================== 定时调度表 ====================
CREATE TABLE IF NOT EXISTS sys_rpa_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    task_id UUID NOT NULL,
    cron_expr VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
    start_date DATE,
    end_date DATE,
    status SMALLINT NOT NULL DEFAULT 0,
    next_run_time TIMESTAMP,
    last_run_time TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0
);

-- ==================== 全局变量表 ====================
-- 命名规约: unique 约束显式命名 uni_<table>_<col> 与 GORM `uniqueIndex` tag 对齐
CREATE TABLE IF NOT EXISTS sys_rpa_variables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    value TEXT NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'string',
    description TEXT,
    is_encrypted BOOLEAN DEFAULT false,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,
    CONSTRAINT uni_sys_rpa_variables_name UNIQUE (name)
);

-- ==================== 通知配置表 ====================
CREATE TABLE IF NOT EXISTS sys_rpa_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    task_id UUID,
    events VARCHAR(100) NOT NULL,
    channels JSONB NOT NULL,
    recipients JSONB,
    template TEXT,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0
);

-- ==================== 审计日志表 ====================
CREATE TABLE IF NOT EXISTS sys_rpa_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    user_id VARCHAR(64),
    user_name VARCHAR(100),
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ==================== 脚本模板表 ====================
CREATE TABLE IF NOT EXISTS sys_rpa_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    script JSONB NOT NULL,
    tags VARCHAR(500),
    is_public BOOLEAN DEFAULT true,
    usage_count INTEGER DEFAULT 0,
    rating DECIMAL(3,2),
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0
);

-- ==================== 索引 ====================
CREATE INDEX IF NOT EXISTS idx_rpa_tasks_status ON sys_rpa_tasks(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_tasks_trigger_type ON sys_rpa_tasks(trigger_type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_workers_status ON sys_rpa_workers(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_executions_task_id ON sys_rpa_executions(task_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_executions_status ON sys_rpa_executions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_schedules_task_id ON sys_rpa_schedules(task_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_schedules_next_run ON sys_rpa_schedules(next_run_time) WHERE deleted_at IS NULL AND status = 0;
CREATE INDEX IF NOT EXISTS idx_rpa_audit_logs_entity ON sys_rpa_audit_logs(entity_type, entity_id);
