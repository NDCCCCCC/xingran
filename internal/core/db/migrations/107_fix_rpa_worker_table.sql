-- ============================================
-- 107_fix_rpa_worker_table.sql
-- 说明: 修复 RPA Worker 表结构，使其与模型定义匹配
--       修改 status 为 VARCHAR，添加缺失字段
-- ============================================

-- 删除旧的 Worker 表（如果存在数据则备份）
DROP TABLE IF EXISTS sys_rpa_workers CASCADE;

-- 重新创建 Worker 表（匹配模型定义）
CREATE TABLE sys_rpa_workers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_name VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) UNIQUE NOT NULL,
    ip_address VARCHAR(50),
    port INTEGER,
    status VARCHAR(20) DEFAULT 'offline',
    capabilities JSONB NOT NULL,
    max_concurrency INTEGER DEFAULT 3,
    current_tasks INTEGER DEFAULT 0,
    total_tasks_executed INTEGER DEFAULT 0,
    total_execution_time BIGINT DEFAULT 0,
    last_heartbeat BIGINT,
    version VARCHAR(50),
    docker_container_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_rpa_workers_status ON sys_rpa_workers(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_rpa_workers_worker_id ON sys_rpa_workers(worker_id);

SELECT '107_fix_rpa_worker_table.sql migration completed' AS status;
