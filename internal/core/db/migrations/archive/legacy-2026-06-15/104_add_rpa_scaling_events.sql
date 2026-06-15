-- Migration: 104_add_rpa_scaling_events.sql
-- Description: 添加 RPA 扩缩容事件记录表
-- Created: 2025-02-25

-- 扩缩容事件记录表
CREATE TABLE IF NOT EXISTS sys_rpa_scaling_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- 事件类型和状态
    event_type VARCHAR(20) NOT NULL,           -- scale_up, scale_down
    status VARCHAR(20) DEFAULT 'success',      -- success, failed

    -- 扩缩容数量
    from_count INTEGER NOT NULL,               -- 扩缩容前数量
    to_count INTEGER NOT NULL,                 -- 扩缩容后数量

    -- 触发原因
    trigger_reason TEXT,                       -- 触发原因描述

    -- 决策指标
    queue_length INTEGER DEFAULT 0,            -- 决策时队列长度
    active_workers INTEGER DEFAULT 0,          -- 决策时活跃 Worker 数
    worker_capacity INTEGER DEFAULT 0,         -- 决策时 Worker 总容量
    average_exec_time INTEGER DEFAULT 0,       -- 平均执行时间（毫秒）

    -- 容器信息
    container_ids TEXT,                        -- 涉及的容器 ID 列表（JSON 数组）

    -- 错误信息
    error_message TEXT                         -- 错误信息（失败时）
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_sys_rpa_scaling_events_type ON sys_rpa_scaling_events(event_type);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_scaling_events_status ON sys_rpa_scaling_events(status);
CREATE INDEX IF NOT EXISTS idx_sys_rpa_scaling_events_created_at ON sys_rpa_scaling_events(created_at DESC);

-- 注释
COMMENT ON TABLE sys_rpa_scaling_events IS 'RPA 扩缩容事件记录表';
COMMENT ON COLUMN sys_rpa_scaling_events.event_type IS '事件类型：scale_up=扩容, scale_down=缩容';
COMMENT ON COLUMN sys_rpa_scaling_events.from_count IS '扩缩容前的 Worker 数量';
COMMENT ON COLUMN sys_rpa_scaling_events.to_count IS '扩缩容后的 Worker 数量';
COMMENT ON COLUMN sys_rpa_scaling_events.queue_length IS '决策时的待处理任务队列长度';
COMMENT ON COLUMN sys_rpa_scaling_events.active_workers IS '决策时的活跃 Worker 数量';
COMMENT ON COLUMN sys_rpa_scaling_events.worker_capacity IS '决策时的 Worker 总容量（并发数）';
COMMENT ON COLUMN sys_rpa_scaling_events.average_exec_time IS '平均执行时间（毫秒）';
COMMENT ON COLUMN sys_rpa_scaling_events.container_ids IS '涉及的容器 ID 列表（JSON 数组格式）';
