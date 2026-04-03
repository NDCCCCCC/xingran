-- +migrate Up
-- RPA 凭证表 - 存储加密的登录凭证
CREATE TABLE IF NOT EXISTS sys_rpa_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- 基本信息
    name VARCHAR(100) NOT NULL,           -- 凭证名称（如：ERP系统账号）
    target_system VARCHAR(100) NOT NULL,  -- 目标系统（如：erp, crm, hr系统）
    target_url VARCHAR(500),              -- 目标系统URL

    -- 加密的凭证信息（使用国密SM4加密）
    username_encrypted BYTEA NOT NULL,    -- 加密后的用户名
    password_encrypted BYTEA NOT NULL,    -- 加密后的密码
    extra_data_encrypted BYTEA,           -- 其他加密信息（如：密钥答案、备用手机号）

    -- 归属和权限
    user_id UUID NOT NULL,                -- 所属用户
    dept_id UUID,                         -- 所属部门
    is_shared BOOLEAN DEFAULT FALSE,      -- 是否共享（部门内共享）

    -- 状态和元数据
    status INTEGER DEFAULT 0,             -- 状态: 0=正常, 1=禁用
    last_used_at TIMESTAMP,               -- 最后使用时间
    last_login_at TIMESTAMP,              -- 最后登录成功时间
    login_success_count INTEGER DEFAULT 0, -- 登录成功次数
    login_fail_count INTEGER DEFAULT 0,   -- 登录失败次数

    CONSTRAINT chk_rpa_cred_status CHECK (status IN (0, 1))
);

-- RPA 会话表 - 存储登录后的 token/cookie 信息
CREATE TABLE IF NOT EXISTS sys_rpa_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- 关联信息
    credential_id UUID NOT NULL,          -- 关联的凭证ID
    execution_id UUID,                    -- 关联的执行记录（可为空，表示持久化会话）

    -- 目标系统信息
    target_system VARCHAR(100) NOT NULL,  -- 目标系统
    target_url VARCHAR(500),              -- 目标系统URL

    -- 会话数据（加密存储）
    access_token_encrypted BYTEA,         -- 访问令牌
    refresh_token_encrypted BYTEA,        -- 刷新令牌
    cookies_encrypted BYTEA,              -- Cookie数据（JSON数组）
    session_data_encrypted BYTEA,         -- 其他会话数据（如：localStorage）

    -- 过期和状态
    expires_at TIMESTAMP,                 -- Token过期时间
    is_valid BOOLEAN DEFAULT TRUE,        -- 是否有效
    invalid_reason VARCHAR(200),          -- 失效原因

    CONSTRAINT fk_rpa_session_cred FOREIGN KEY (credential_id)
        REFERENCES sys_rpa_credentials(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_rpa_cred_user ON sys_rpa_credentials(user_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_rpa_cred_system ON sys_rpa_credentials(target_system, deleted_at);
CREATE INDEX IF NOT EXISTS idx_rpa_cred_shared ON sys_rpa_credentials(is_shared, dept_id, deleted_at);

CREATE INDEX IF NOT EXISTS idx_rpa_session_cred ON sys_rpa_sessions(credential_id, is_valid, deleted_at);
CREATE INDEX IF NOT EXISTS idx_rpa_session_system ON sys_rpa_sessions(target_system, is_valid, deleted_at);
CREATE INDEX IF NOT EXISTS idx_rpa_session_exec ON sys_rpa_sessions(execution_id);

-- 添加注释
COMMENT ON TABLE sys_rpa_credentials IS 'RPA 登录凭证表（使用国密SM4加密）';
COMMENT ON COLUMN sys_rpa_credentials.username_encrypted IS 'SM4加密后的用户名';
COMMENT ON COLUMN sys_rpa_credentials.password_encrypted IS 'SM4加密后的密码';
COMMENT ON COLUMN sys_rpa_credentials.is_shared IS '是否在部门内共享（允许团队成员使用）';

COMMENT ON TABLE sys_rpa_sessions IS 'RPA 会话管理表（存储token/cookie实现免登录）';
COMMENT ON COLUMN sys_rpa_sessions.access_token_encrypted IS 'SM4加密的访问令牌';
COMMENT ON COLUMN sys_rpa_sessions.cookies_encrypted IS 'SM4加密的Cookie数据';

-- +migrate Down
-- 删除索引
DROP INDEX IF EXISTS idx_rpa_session_exec;
DROP INDEX IF EXISTS idx_rpa_session_system;
DROP INDEX IF EXISTS idx_rpa_session_cred;
DROP INDEX IF EXISTS idx_rpa_cred_shared;
DROP INDEX IF EXISTS idx_rpa_cred_system;
DROP INDEX IF EXISTS idx_rpa_cred_user;

-- 删除外键约束
ALTER TABLE sys_rpa_sessions DROP CONSTRAINT IF EXISTS fk_rpa_session_cred;

-- 删除表
DROP TABLE IF EXISTS sys_rpa_sessions;
DROP TABLE IF EXISTS sys_rpa_credentials;
