-- =============================================
-- AD域控管理功能 - 数据库表创建（仅数据表）
-- 迁移版本: 016
-- 描述: 创建AD域配置、OU、用户组、用户缓存及同步日志表
-- =============================================

-- ================================
-- 1. AD域配置表
-- ================================
CREATE TABLE IF NOT EXISTS sys_ad_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name VARCHAR(100) NOT NULL,
    server_address VARCHAR(255) NOT NULL,
    server_port INT DEFAULT 389,
    domain_name VARCHAR(255) NOT NULL,
    base_dn VARCHAR(500) NOT NULL,
    admin_username VARCHAR(255) NOT NULL,
    admin_password VARCHAR(500) NOT NULL,
    use_ssl BOOLEAN DEFAULT FALSE,
    use_tls BOOLEAN DEFAULT FALSE,
    sync_enabled BOOLEAN DEFAULT TRUE,
    sync_interval INT DEFAULT 3600,
    last_sync_at TIMESTAMP,
    status INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(64),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(64),
    version INT DEFAULT 0,
    deleted_at TIMESTAMP,
    CONSTRAINT uk_ad_config_name UNIQUE (config_name, deleted_at)
);

-- 表注释
COMMENT ON TABLE sys_ad_config IS 'AD域配置表';
COMMENT ON COLUMN sys_ad_config.config_name IS '配置名称';
COMMENT ON COLUMN sys_ad_config.server_address IS 'AD服务器地址';
COMMENT ON COLUMN sys_ad_config.server_port IS 'AD服务器端口(默认389,LDAPS使用636)';
COMMENT ON COLUMN sys_ad_config.domain_name IS '域名(如: example.com)';
COMMENT ON COLUMN sys_ad_config.base_dn IS '基础DN(如: DC=example,DC=com)';
COMMENT ON COLUMN sys_ad_config.admin_username IS '管理员用户名';
COMMENT ON COLUMN sys_ad_config.admin_password IS '管理员密码(加密存储)';
COMMENT ON COLUMN sys_ad_config.use_ssl IS '是否使用SSL(LDAPS)';
COMMENT ON COLUMN sys_ad_config.use_tls IS '是否使用TLS(StartTLS)';
COMMENT ON COLUMN sys_ad_config.sync_enabled IS '是否启用自动同步';
COMMENT ON COLUMN sys_ad_config.sync_interval IS '同步间隔(秒)';
COMMENT ON COLUMN sys_ad_config.last_sync_at IS '最后同步时间';
COMMENT ON COLUMN sys_ad_config.status IS '状态: 0=启用 1=停用';

-- 索引
CREATE INDEX idx_ad_config_status ON sys_ad_config(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_config_sync ON sys_ad_config(sync_enabled, status) WHERE deleted_at IS NULL;


-- ================================
-- 2. OU组织单位缓存表
-- ================================
CREATE TABLE IF NOT EXISTS sys_ad_ou (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_config_id UUID NOT NULL,
    ou_dn VARCHAR(500) NOT NULL,
    ou_name VARCHAR(255) NOT NULL,
    ou_path TEXT,
    parent_dn VARCHAR(500),
    description TEXT,
    user_count INT DEFAULT 0,
    group_count INT DEFAULT 0,
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT fk_ad_ou_config FOREIGN KEY (ad_config_id) REFERENCES sys_ad_config(id) ON DELETE CASCADE,
    CONSTRAINT uk_ad_ou_dn UNIQUE (ad_config_id, ou_dn)
);

-- 表注释
COMMENT ON TABLE sys_ad_ou IS 'AD域OU组织单位缓存表';
COMMENT ON COLUMN sys_ad_ou.ad_config_id IS 'AD配置ID';
COMMENT ON COLUMN sys_ad_ou.ou_dn IS 'OU的LDAP DN';
COMMENT ON COLUMN sys_ad_ou.ou_name IS 'OU名称';
COMMENT ON COLUMN sys_ad_ou.ou_path IS 'OU完整路径';
COMMENT ON COLUMN sys_ad_ou.parent_dn IS '父OU的DN';
COMMENT ON COLUMN sys_ad_ou.description IS 'OU描述';
COMMENT ON COLUMN sys_ad_ou.user_count IS '用户数量';
COMMENT ON COLUMN sys_ad_ou.group_count IS '用户组数量';
COMMENT ON COLUMN sys_ad_ou.last_sync_at IS '最后同步时间';

-- 索引
CREATE INDEX idx_ad_ou_config ON sys_ad_ou(ad_config_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_ou_parent ON sys_ad_ou(parent_dn) WHERE deleted_at IS NULL;


-- ================================
-- 3. AD用户组缓存表
-- ================================
CREATE TABLE IF NOT EXISTS sys_ad_group (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_config_id UUID NOT NULL,
    group_dn VARCHAR(500) NOT NULL,
    group_name VARCHAR(255) NOT NULL,
    group_scope VARCHAR(50),
    group_type INT,
    description TEXT,
    member_count INT DEFAULT 0,
    ou_dn VARCHAR(500),
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT fk_ad_group_config FOREIGN KEY (ad_config_id) REFERENCES sys_ad_config(id) ON DELETE CASCADE,
    CONSTRAINT uk_ad_group_dn UNIQUE (ad_config_id, group_dn)
);

-- 表注释
COMMENT ON TABLE sys_ad_group IS 'AD域用户组缓存表';
COMMENT ON COLUMN sys_ad_group.ad_config_id IS 'AD配置ID';
COMMENT ON COLUMN sys_ad_group.group_dn IS '组的LDAP DN';
COMMENT ON COLUMN sys_ad_group.group_name IS '组名称';
COMMENT ON COLUMN sys_ad_group.group_scope IS '组作用域(Global/Local/Universal)';
COMMENT ON COLUMN sys_ad_group.group_type IS '组类型(安全组/分发组)';
COMMENT ON COLUMN sys_ad_group.description IS '组描述';
COMMENT ON COLUMN sys_ad_group.member_count IS '成员数量';
COMMENT ON COLUMN sys_ad_group.ou_dn IS '所属OU的DN';
COMMENT ON COLUMN sys_ad_group.last_sync_at IS '最后同步时间';

-- 索引
CREATE INDEX idx_ad_group_config ON sys_ad_group(ad_config_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_group_ou ON sys_ad_group(ou_dn) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_group_name ON sys_ad_group(group_name) WHERE deleted_at IS NULL;


-- ================================
-- 4. AD用户缓存表
-- ================================
CREATE TABLE IF NOT EXISTS sys_ad_user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_config_id UUID NOT NULL,
    user_dn VARCHAR(500) NOT NULL,
    username VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    mobile VARCHAR(50),
    title VARCHAR(100),
    department VARCHAR(255),
    company VARCHAR(255),
    ou_dn VARCHAR(500),
    user_account_control INT,
    is_enabled BOOLEAN DEFAULT TRUE,
    is_locked BOOLEAN DEFAULT FALSE,
    password_expired BOOLEAN DEFAULT FALSE,
    last_logon TIMESTAMP,
    password_last_set TIMESTAMP,
    account_expires TIMESTAMP,
    description TEXT,
    member_of TEXT,
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT fk_ad_user_config FOREIGN KEY (ad_config_id) REFERENCES sys_ad_config(id) ON DELETE CASCADE,
    CONSTRAINT uk_ad_user_dn UNIQUE (ad_config_id, user_dn)
);

-- 表注释
COMMENT ON TABLE sys_ad_user IS 'AD域用户缓存表';
COMMENT ON COLUMN sys_ad_user.ad_config_id IS 'AD配置ID';
COMMENT ON COLUMN sys_ad_user.user_dn IS '用户的LDAP DN';
COMMENT ON COLUMN sys_ad_user.username IS '用户登录名(sAMAccountName)';
COMMENT ON COLUMN sys_ad_user.display_name IS '显示名称';
COMMENT ON COLUMN sys_ad_user.email IS '邮箱';
COMMENT ON COLUMN sys_ad_user.phone IS '办公电话';
COMMENT ON COLUMN sys_ad_user.mobile IS '手机号码';
COMMENT ON COLUMN sys_ad_user.title IS '职位';
COMMENT ON COLUMN sys_ad_user.department IS '部门';
COMMENT ON COLUMN sys_ad_user.company IS '公司';
COMMENT ON COLUMN sys_ad_user.ou_dn IS '所属OU的DN';
COMMENT ON COLUMN sys_ad_user.user_account_control IS '用户账户控制标志';
COMMENT ON COLUMN sys_ad_user.is_enabled IS '是否启用';
COMMENT ON COLUMN sys_ad_user.is_locked IS '是否锁定';
COMMENT ON COLUMN sys_ad_user.password_expired IS '密码是否过期';
COMMENT ON COLUMN sys_ad_user.last_logon IS '最后登录时间';
COMMENT ON COLUMN sys_ad_user.password_last_set IS '密码最后设置时间';
COMMENT ON COLUMN sys_ad_user.account_expires IS '账户过期时间';
COMMENT ON COLUMN sys_ad_user.description IS '用户描述';
COMMENT ON COLUMN sys_ad_user.member_of IS '所属用户组(逗号分隔DN)';
COMMENT ON COLUMN sys_ad_user.last_sync_at IS '最后同步时间';

-- 索引
CREATE INDEX idx_ad_user_config ON sys_ad_user(ad_config_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_user_ou ON sys_ad_user(ou_dn) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_user_username ON sys_ad_user(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_user_email ON sys_ad_user(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_ad_user_enabled ON sys_ad_user(is_enabled) WHERE deleted_at IS NULL;


-- ================================
-- 5. 用户组成员关系表
-- ================================
CREATE TABLE IF NOT EXISTS sys_ad_group_member (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_config_id UUID NOT NULL,
    group_dn VARCHAR(500) NOT NULL,
    user_dn VARCHAR(500) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ad_gm_config FOREIGN KEY (ad_config_id) REFERENCES sys_ad_config(id) ON DELETE CASCADE,
    CONSTRAINT uk_ad_group_member UNIQUE (ad_config_id, group_dn, user_dn)
);

-- 表注释
COMMENT ON TABLE sys_ad_group_member IS 'AD域用户组成员关系表';
COMMENT ON COLUMN sys_ad_group_member.ad_config_id IS 'AD配置ID';
COMMENT ON COLUMN sys_ad_group_member.group_dn IS '组的LDAP DN';
COMMENT ON COLUMN sys_ad_group_member.user_dn IS '用户的LDAP DN';

-- 索引
CREATE INDEX idx_ad_gm_config ON sys_ad_group_member(ad_config_id);
CREATE INDEX idx_ad_gm_group ON sys_ad_group_member(group_dn);
CREATE INDEX idx_ad_gm_user ON sys_ad_group_member(user_dn);


-- ================================
-- 6. AD同步日志表
-- ================================
CREATE TABLE IF NOT EXISTS sys_ad_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_config_id UUID NOT NULL,
    sync_type VARCHAR(50) NOT NULL,
    sync_status VARCHAR(20) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration INT,
    ou_count INT DEFAULT 0,
    group_count INT DEFAULT 0,
    user_count INT DEFAULT 0,
    error_count INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ad_sync_config FOREIGN KEY (ad_config_id) REFERENCES sys_ad_config(id) ON DELETE CASCADE
);

-- 表注释
COMMENT ON TABLE sys_ad_sync_log IS 'AD域同步日志表';
COMMENT ON COLUMN sys_ad_sync_log.ad_config_id IS 'AD配置ID';
COMMENT ON COLUMN sys_ad_sync_log.sync_type IS '同步类型(full/incremental)';
COMMENT ON COLUMN sys_ad_sync_log.sync_status IS '同步状态(running/success/failed)';
COMMENT ON COLUMN sys_ad_sync_log.start_time IS '开始时间';
COMMENT ON COLUMN sys_ad_sync_log.end_time IS '结束时间';
COMMENT ON COLUMN sys_ad_sync_log.duration IS '耗时(秒)';
COMMENT ON COLUMN sys_ad_sync_log.ou_count IS '同步OU数量';
COMMENT ON COLUMN sys_ad_sync_log.group_count IS '同步用户组数量';
COMMENT ON COLUMN sys_ad_sync_log.user_count IS '同步用户数量';
COMMENT ON COLUMN sys_ad_sync_log.error_count IS '错误数量';
COMMENT ON COLUMN sys_ad_sync_log.error_message IS '错误信息';

-- 索引
CREATE INDEX idx_ad_sync_log_config ON sys_ad_sync_log(ad_config_id);
CREATE INDEX idx_ad_sync_log_time ON sys_ad_sync_log(start_time DESC);
CREATE INDEX idx_ad_sync_log_status ON sys_ad_sync_log(sync_status);
