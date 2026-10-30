-- 监控模块数据库表结构创建脚本
-- 执行时间：2025-12-22
-- 说明：创建第三阶段监控模块所需的所有数据库表

-- 删除已存在的表（如果需要重新创建）
-- DROP TABLE IF EXISTS sys_cache_stats CASCADE;
-- DROP TABLE IF EXISTS sys_cache_info CASCADE;
-- DROP TABLE IF EXISTS sys_job_log CASCADE;
-- DROP TABLE IF EXISTS sys_job CASCADE;
-- DROP TABLE IF EXISTS sys_logininfor CASCADE;
-- DROP TABLE IF EXISTS sys_oper_log CASCADE;
-- DROP TABLE IF EXISTS sys_system_metrics CASCADE;
-- DROP TABLE IF EXISTS sys_server_info CASCADE;

-- 1. 服务器信息表
CREATE TABLE IF NOT EXISTS sys_server_info (
    id varchar(36) PRIMARY KEY DEFAULT (gen_random_uuid()::text),
    host_name varchar(128) NOT NULL,
    os varchar(64),
    arch varchar(32),
    cpu_count integer,
    total_memory bigint,
    available_memory bigint,
    disk_total bigint,
    disk_available bigint,
    status integer DEFAULT 0,
    last_active_at timestamp,
    create_time timestamp DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp DEFAULT CURRENT_TIMESTAMP,
    remark text
);

-- 2. 系统性能指标表
CREATE TABLE IF NOT EXISTS sys_system_metrics (
    id varchar(36) PRIMARY KEY DEFAULT (gen_random_uuid()::text),
    server_id varchar(36) NOT NULL,
    cpu_usage double precision,
    memory_usage double precision,
    disk_usage double precision,
    network_rx bigint,
    network_tx bigint,
    process_count integer,
    load_average double precision,
    timestamp timestamp NOT NULL,
    create_time timestamp DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp DEFAULT CURRENT_TIMESTAMP,
    remark text
);

-- 3. 操作日志表 (如果不存在)
CREATE TABLE IF NOT EXISTS sys_oper_log (
    id varchar(36) PRIMARY KEY DEFAULT (gen_random_uuid()::text),
    title varchar(50) DEFAULT '',
    business_type int DEFAULT 0,
    method varchar(100) DEFAULT '',
    request_method varchar(10) DEFAULT '',
    operator_type int DEFAULT 0,
    oper_name varchar(50) DEFAULT '',
    dept_name varchar(50) DEFAULT '',
    oper_url varchar(255) DEFAULT '',
    oper_ip varchar(128) DEFAULT '',
    oper_location varchar(255) DEFAULT '',
    oper_param text,
    json_result text,
    status int DEFAULT 0,
    error_msg varchar(2000) DEFAULT '',
    oper_time timestamp DEFAULT CURRENT_TIMESTAMP,
    cost_time bigint DEFAULT 0
);

-- 4. 登录日志表 (注意：日志表使用sys_logininfor名称)
CREATE TABLE IF NOT EXISTS sys_logininfor (
    info_id varchar(36) PRIMARY KEY DEFAULT (gen_random_uuid()::text),
    user_name varchar(50) DEFAULT '',
    ipaddr varchar(128) DEFAULT '',
    login_location varchar(255) DEFAULT '',
    browser varchar(50) DEFAULT '',
    os varchar(50) DEFAULT '',
    status int DEFAULT 0,
    msg varchar(255) DEFAULT '',
    login_time timestamp DEFAULT CURRENT_TIMESTAMP
);

-- 5. 定时任务表
CREATE TABLE IF NOT EXISTS sys_job (
    id varchar(36) PRIMARY KEY DEFAULT (gen_random_uuid()::text),
    job_name varchar(128) NOT NULL,
    job_group varchar(128) NOT NULL,
    invoke_target varchar(500) NOT NULL,
    cron_expression varchar(255),
    misfire_policy int DEFAULT 1,
    concurrent boolean DEFAULT false,
    status int DEFAULT 0,
    create_by varchar(64),
    update_by varchar(64),
    remark text,
    create_time timestamp DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp DEFAULT CURRENT_TIMESTAMP
);

-- 6. 定时任务日志表
CREATE TABLE IF NOT EXISTS sys_job_log (
    id varchar(36) PRIMARY KEY DEFAULT (gen_random_uuid()::text),
    job_name varchar(128) NOT NULL,
    job_group varchar(128) NOT NULL,
    invoke_target varchar(500) NOT NULL,
    job_message text,
    status int DEFAULT 0,
    exception_info text,
    start_time timestamp DEFAULT CURRENT_TIMESTAMP,
    end_time timestamp,
    duration bigint
);

-- 7. 缓存信息表
CREATE TABLE IF NOT EXISTS sys_cache_info (
    key varchar(255) PRIMARY KEY,
    value text,
    ttl bigint,
    size bigint,
    type varchar(32),
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);

-- 8. 缓存统计表
CREATE TABLE IF NOT EXISTS sys_cache_stats (
    id varchar(36) PRIMARY KEY DEFAULT (gen_random_uuid()::text),
    cache_type varchar(32) NOT NULL,
    hit_count bigint DEFAULT 0,
    miss_count bigint DEFAULT 0,
    hit_rate double precision DEFAULT 0.0,
    total_memory bigint DEFAULT 0,
    used_memory bigint DEFAULT 0,
    key_count bigint DEFAULT 0,
    expired_count bigint DEFAULT 0,
    collect_time timestamp DEFAULT CURRENT_TIMESTAMP,
    create_time timestamp DEFAULT CURRENT_TIMESTAMP,
    update_time timestamp DEFAULT CURRENT_TIMESTAMP,
    remark text
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_server_info_status ON sys_server_info(status);
CREATE INDEX IF NOT EXISTS idx_server_info_last_active ON sys_server_info(last_active_at);
CREATE INDEX IF NOT EXISTS idx_system_metrics_server_id ON sys_system_metrics(server_id);
CREATE INDEX IF NOT EXISTS idx_system_metrics_timestamp ON sys_system_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_oper_log_oper_time ON sys_oper_log(oper_time);
CREATE INDEX IF NOT EXISTS idx_oper_log_oper_name ON sys_oper_log(oper_name);
CREATE INDEX IF NOT EXISTS idx_oper_log_status ON sys_oper_log(status);
CREATE INDEX IF NOT EXISTS idx_logininfor_login_time ON sys_logininfor(login_time);
CREATE INDEX IF NOT EXISTS idx_logininfor_user_name ON sys_logininfor(user_name);
CREATE INDEX IF NOT EXISTS idx_logininfor_status ON sys_logininfor(status);
CREATE INDEX IF NOT EXISTS idx_job_name_group ON sys_job(job_name, job_group);
CREATE INDEX IF NOT EXISTS idx_job_status ON sys_job(status);
CREATE INDEX IF NOT EXISTS idx_job_log_start_time ON sys_job_log(start_time);
CREATE INDEX IF NOT EXISTS idx_job_log_job_name ON sys_job_log(job_name);
CREATE INDEX IF NOT EXISTS idx_job_log_status ON sys_job_log(status);
CREATE INDEX IF NOT EXISTS idx_cache_info_type ON sys_cache_info(type);
CREATE INDEX IF NOT EXISTS idx_cache_info_created_at ON sys_cache_info(created_at);
CREATE INDEX IF NOT EXISTS idx_cache_stats_cache_type ON sys_cache_stats(cache_type);
CREATE INDEX IF NOT EXISTS idx_cache_stats_collect_time ON sys_cache_stats(collect_time);

-- 插入示例数据
-- 注意：以下任务处理器需要先实现才能启用，暂时注释掉
-- INSERT INTO sys_job (job_name, job_group, invoke_target, cron_expression, status, remark) VALUES
-- ('系统监控任务', 'DEFAULT', 'systemMonitor.collectMetrics', '0 */5 * * * *', 0, '每5分钟收集系统指标'),
-- ('缓存清理任务', 'DEFAULT', 'cacheCleaner.cleanExpired', '0 0 2 * * *', 0, '每天凌晨2点清理过期缓存'),
-- ('日志归档任务', 'DEFAULT', 'logArchiver.archiveOldLogs', '0 0 3 * * 0', 0, '每周日凌晨3点归档旧日志'),
-- ('数据备份任务', 'DEFAULT', 'dataBackup.backupDatabase', '0 0 1 * * *', 0, '每天凌晨1点备份数据库')
-- ON CONFLICT DO NOTHING;

INSERT INTO sys_server_info (host_name, os, arch, cpu_count, total_memory, disk_total, status) VALUES
('localhost', 'Windows 10', 'x86_64', 8, 17179869184, 107374182400, 0)
ON CONFLICT DO NOTHING;

-- 输出成功信息
DO $$
BEGIN
    RAISE NOTICE '监控模块数据库表创建成功！';
    RAISE NOTICE '已创建的表: sys_server_info, sys_system_metrics, sys_oper_log, sys_logininfor, sys_job, sys_job_log, sys_cache_info, sys_cache_stats';
END $$;