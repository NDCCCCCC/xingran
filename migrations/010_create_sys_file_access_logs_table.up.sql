-- 010_create_sys_file_access_logs_table.up.sql
CREATE TABLE IF NOT EXISTS sys_file_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    file_id UUID NOT NULL,
    action_type VARCHAR(20) NOT NULL,
    user_id UUID NOT NULL,
    user_name VARCHAR(100),
    ip_address VARCHAR(50),
    user_agent VARCHAR(500)
);

-- 创建索引
CREATE INDEX idx_sys_file_access_logs_file_id ON sys_file_access_logs(file_id);
CREATE INDEX idx_sys_file_access_logs_user_id ON sys_file_access_logs(user_id);
CREATE INDEX idx_sys_file_access_logs_action_type ON sys_file_access_logs(action_type);
CREATE INDEX idx_sys_file_access_logs_created_at ON sys_file_access_logs(created_at);

-- 添加注释
COMMENT ON TABLE sys_file_access_logs IS '文件访问日志表';
COMMENT ON COLUMN sys_file_access_logs.action_type IS '操作类型: upload/download/delete/view';
