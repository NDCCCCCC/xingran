-- 通知渠道配置数据库迁移
-- 文件: 006_create_notification_channel_configs.sql
-- 说明: 创建邮箱服务器配置、API通知配置和通知渠道关联表，支持多渠道通知发送

-- ============================================
-- 1. 创建邮箱服务器配置表
-- ============================================

CREATE TABLE IF NOT EXISTS sys_email_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(100) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INT NOT NULL DEFAULT 587,
    username VARCHAR(255) NOT NULL,
    password VARCHAR(500) NOT NULL,
    from_name VARCHAR(100),
    from_email VARCHAR(255),
    use_ssl BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    status INT DEFAULT 0,
    remark VARCHAR(500),
    created_by VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    del_flag INT DEFAULT 0
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_email_config_default ON sys_email_config(is_default, status, del_flag);
CREATE INDEX IF NOT EXISTS idx_email_config_status ON sys_email_config(status, del_flag);

-- 添加表和字段注释
COMMENT ON TABLE sys_email_config IS '邮箱服务器配置表';
COMMENT ON COLUMN sys_email_config.id IS '配置ID';
COMMENT ON COLUMN sys_email_config.config_name IS '配置名称';
COMMENT ON COLUMN sys_email_config.host IS 'SMTP服务器地址';
COMMENT ON COLUMN sys_email_config.port IS 'SMTP端口';
COMMENT ON COLUMN sys_email_config.username IS '发件人邮箱账号';
COMMENT ON COLUMN sys_email_config.password IS '邮箱密码或授权码（加密存储）';
COMMENT ON COLUMN sys_email_config.from_name IS '发件人名称';
COMMENT ON COLUMN sys_email_config.from_email IS '发件人邮箱地址';
COMMENT ON COLUMN sys_email_config.use_ssl IS '是否使用SSL';
COMMENT ON COLUMN sys_email_config.is_default IS '是否为默认配置';
COMMENT ON COLUMN sys_email_config.status IS '状态：0=正常 1=停用';
COMMENT ON COLUMN sys_email_config.remark IS '备注';
COMMENT ON COLUMN sys_email_config.created_by IS '创建者';
COMMENT ON COLUMN sys_email_config.created_at IS '创建时间';
COMMENT ON COLUMN sys_email_config.updated_by IS '更新者';
COMMENT ON COLUMN sys_email_config.updated_at IS '更新时间';
COMMENT ON COLUMN sys_email_config.del_flag IS '删除标志：0=正常 1=删除';

-- ============================================
-- 2. 创建API通知配置表（短信、Webhook等）
-- ============================================

CREATE TABLE IF NOT EXISTS sys_api_notification_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(100) NOT NULL,
    config_type VARCHAR(50) NOT NULL,
    api_url VARCHAR(500) NOT NULL,
    api_method VARCHAR(10) DEFAULT 'POST',
    headers TEXT,
    template_body TEXT,
    auth_type VARCHAR(50),
    auth_config TEXT,
    retry_count INT DEFAULT 3,
    timeout INT DEFAULT 30,
    is_default BOOLEAN DEFAULT FALSE,
    status INT DEFAULT 0,
    remark VARCHAR(500),
    created_by VARCHAR(64),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    del_flag INT DEFAULT 0
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_api_config_type ON sys_api_notification_config(config_type, status, del_flag);
CREATE INDEX IF NOT EXISTS idx_api_config_default ON sys_api_notification_config(is_default, status, del_flag);

-- 添加表和字段注释
COMMENT ON TABLE sys_api_notification_config IS 'API通知配置表';
COMMENT ON COLUMN sys_api_notification_config.id IS '配置ID';
COMMENT ON COLUMN sys_api_notification_config.config_name IS '配置名称';
COMMENT ON COLUMN sys_api_notification_config.config_type IS '配置类型：sms=短信 webhook=Webhook push=推送';
COMMENT ON COLUMN sys_api_notification_config.api_url IS 'API地址';
COMMENT ON COLUMN sys_api_notification_config.api_method IS '请求方式：GET/POST';
COMMENT ON COLUMN sys_api_notification_config.headers IS '请求头（JSON格式）';
COMMENT ON COLUMN sys_api_notification_config.template_body IS '请求体模板（JSON格式，支持变量替换）';
COMMENT ON COLUMN sys_api_notification_config.auth_type IS '认证方式：none=无 basic=BasicAuth bearer=BearerToken apikey=APIKey';
COMMENT ON COLUMN sys_api_notification_config.auth_config IS '认证配置（JSON格式）';
COMMENT ON COLUMN sys_api_notification_config.retry_count IS '重试次数';
COMMENT ON COLUMN sys_api_notification_config.timeout IS '超时时间（秒）';
COMMENT ON COLUMN sys_api_notification_config.is_default IS '是否为默认配置';
COMMENT ON COLUMN sys_api_notification_config.status IS '状态：0=正常 1=停用';
COMMENT ON COLUMN sys_api_notification_config.remark IS '备注';
COMMENT ON COLUMN sys_api_notification_config.created_by IS '创建者';
COMMENT ON COLUMN sys_api_notification_config.created_at IS '创建时间';
COMMENT ON COLUMN sys_api_notification_config.updated_by IS '更新者';
COMMENT ON COLUMN sys_api_notification_config.updated_at IS '更新时间';
COMMENT ON COLUMN sys_api_notification_config.del_flag IS '删除标志：0=正常 1=删除';

-- ============================================
-- 3. 创建通知渠道关联表
-- ============================================

CREATE TABLE IF NOT EXISTS sys_notification_channel (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notice_id UUID NOT NULL,
    channel_type VARCHAR(20) NOT NULL,
    email_config_id UUID,
    api_config_id UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_notification_channel_notice FOREIGN KEY (notice_id)
        REFERENCES sys_notice(id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_channel_email FOREIGN KEY (email_config_id)
        REFERENCES sys_email_config(id) ON DELETE SET NULL,
    CONSTRAINT fk_notification_channel_api FOREIGN KEY (api_config_id)
        REFERENCES sys_api_notification_config(id) ON DELETE SET NULL
);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_notification_channel_notice_id ON sys_notification_channel(notice_id);
CREATE INDEX IF NOT EXISTS idx_notification_channel_type ON sys_notification_channel(channel_type);

-- 添加表和字段注释
COMMENT ON TABLE sys_notification_channel IS '通知渠道关联表';
COMMENT ON COLUMN sys_notification_channel.id IS '主键ID';
COMMENT ON COLUMN sys_notification_channel.notice_id IS '通知ID';
COMMENT ON COLUMN sys_notification_channel.channel_type IS '渠道类型：web=站内信 email=邮件 sms=短信 api=API';
COMMENT ON COLUMN sys_notification_channel.email_config_id IS '邮箱配置ID';
COMMENT ON COLUMN sys_notification_channel.api_config_id IS 'API配置ID';
COMMENT ON COLUMN sys_notification_channel.created_at IS '创建时间';

-- ============================================
-- 4. 创建触发器自动更新 updated_at
-- ============================================

-- sys_email_config 更新时间触发器
CREATE OR REPLACE FUNCTION update_sys_email_config_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_sys_email_config_updated_at ON sys_email_config;
CREATE TRIGGER trigger_update_sys_email_config_updated_at
    BEFORE UPDATE ON sys_email_config
    FOR EACH ROW
    EXECUTE FUNCTION update_sys_email_config_updated_at();

-- sys_api_notification_config 更新时间触发器
CREATE OR REPLACE FUNCTION update_sys_api_notification_config_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_sys_api_notification_config_updated_at ON sys_api_notification_config;
CREATE TRIGGER trigger_update_sys_api_notification_config_updated_at
    BEFORE UPDATE ON sys_api_notification_config
    FOR EACH ROW
    EXECUTE FUNCTION update_sys_api_notification_config_updated_at();

-- ============================================
-- 迁移完成
-- ============================================

-- 验证迁移
SELECT '006_create_notification_channel_configs.sql migration completed' AS status;

-- 检查新表
SELECT table_name, (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) AS column_count
FROM information_schema.tables t
WHERE table_schema = 'public'
AND table_name IN ('sys_email_config', 'sys_api_notification_config', 'sys_notification_channel');
