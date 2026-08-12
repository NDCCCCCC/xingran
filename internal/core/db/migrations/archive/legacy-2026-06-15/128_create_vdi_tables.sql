-- Migration: 128_create_vdi_tables.sql
-- Description: Create VDI (Virtual Desktop Infrastructure) tables for Sangfor VDI integration
-- Created: Phase 22-01 VDI基础集成

-- VDI服务器配置表
CREATE TABLE IF NOT EXISTS sys_vdi_server (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,                  -- 服务器名称
    endpoint VARCHAR(500) NOT NULL,              -- API端点
    username VARCHAR(100) NOT NULL,              -- API用户名
    password_encrypted VARCHAR(500) NOT NULL,    -- SM4加密密码
    tenant_id INT DEFAULT 0,                     -- 租户ID
    auth_token VARCHAR(1000),                    -- 缓存的认证token
    token_expiry TIMESTAMP,                      -- token过期时间
    status INT DEFAULT 0,                        -- 状态: 0=正常, 1=停用
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- VDI虚拟机表
-- 命名规约: unique 约束显式命名 uni_<table>_<col> 与 GORM `uniqueIndex` tag 对齐
CREATE TABLE IF NOT EXISTS sys_vdi_vm (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vm_id VARCHAR(100) NOT NULL,                -- 深信服VM ID
    name VARCHAR(200) NOT NULL,                 -- 虚拟机名称
    resource_id VARCHAR(100),                   -- 资源组ID
    status INT DEFAULT 0,                       -- 状态: 0=正常, 1=停用
    power_state VARCHAR(50),                    -- 电源状态: running/stopped/suspended
    ip_address VARCHAR(50),                     -- IP地址
    mac_address VARCHAR(50),                    -- MAC地址
    os_type VARCHAR(50),                        -- 操作系统类型
    cpu INT DEFAULT 0,                          -- CPU核心数
    memory INT DEFAULT 0,                       -- 内存(MB)
    disk INT DEFAULT 0,                         -- 磁盘(GB)
    bound_user_id VARCHAR(100),                 -- 绑定用户ID
    bound_user_name VARCHAR(200),               -- 绑定用户名
    policy_group_id VARCHAR(100),               -- 策略组ID
    last_sync_at TIMESTAMP,                     -- 最后同步时间
    vdi_server_id VARCHAR(100) NOT NULL,        -- VDI服务器ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INT DEFAULT 0,
    CONSTRAINT uni_sys_vdi_vm_vm_id UNIQUE (vm_id)
);

-- VDI资源组表
CREATE TABLE IF NOT EXISTS sys_vdi_resource_group (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_group_id VARCHAR(100) NOT NULL,    -- 资源组ID（深信服返回的ID）
    name VARCHAR(200) NOT NULL,                 -- 资源组名称
    vdi_server_id VARCHAR(100) NOT NULL,        -- VDI服务器ID
    type VARCHAR(50),                           -- 类型: 独享桌面/池桌面
    status INT DEFAULT 0,                       -- 状态: 0=正常, 1=停用
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INT DEFAULT 0,
    CONSTRAINT uni_sys_vdi_resource_group_resource_group_id UNIQUE (resource_group_id)
);

-- VDI用户绑定表
CREATE TABLE IF NOT EXISTS sys_vdi_user_binding (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) NOT NULL,              -- 用户ID
    user_name VARCHAR(200) NOT NULL,            -- 用户名
    vm_id VARCHAR(100) NOT NULL,                -- 虚拟机ID
    vdi_server_id VARCHAR(100) NOT NULL,        -- VDI服务器ID
    bound_at TIMESTAMP NOT NULL,                -- 绑定时间
    status INT DEFAULT 0,                       -- 状态: 0=正常, 1=停用
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INT DEFAULT 0
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_sys_vdi_vm_resource_id ON sys_vdi_vm(resource_id);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_vm_vdi_server_id ON sys_vdi_vm(vdi_server_id);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_vm_bound_user_id ON sys_vdi_vm(bound_user_id);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_vm_deleted_at ON sys_vdi_vm(deleted_at);

CREATE INDEX IF NOT EXISTS idx_sys_vdi_server_status ON sys_vdi_server(status);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_server_deleted_at ON sys_vdi_server(deleted_at);

CREATE INDEX IF NOT EXISTS idx_sys_vdi_resource_group_vdi_server_id ON sys_vdi_resource_group(vdi_server_id);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_resource_group_deleted_at ON sys_vdi_resource_group(deleted_at);

CREATE INDEX IF NOT EXISTS idx_sys_vdi_user_binding_user_id ON sys_vdi_user_binding(user_id);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_user_binding_vm_id ON sys_vdi_user_binding(vm_id);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_user_binding_vdi_server_id ON sys_vdi_user_binding(vdi_server_id);
CREATE INDEX IF NOT EXISTS idx_sys_vdi_user_binding_deleted_at ON sys_vdi_user_binding(deleted_at);

-- 添加外键约束（可选，根据项目需要）
-- ALTER TABLE sys_vdi_vm ADD CONSTRAINT fk_vdi_vm_server
--     FOREIGN KEY (vdi_server_id) REFERENCES sys_vdi_server(id) ON DELETE CASCADE;

-- ALTER TABLE sys_vdi_resource_group ADD CONSTRAINT fk_vdi_resource_group_server
--     FOREIGN KEY (vdi_server_id) REFERENCES sys_vdi_server(id) ON DELETE CASCADE;

-- ALTER TABLE sys_vdi_user_binding ADD CONSTRAINT fk_vdi_user_binding_server
--     FOREIGN KEY (vdi_server_id) REFERENCES sys_vdi_server(id) ON DELETE CASCADE;

-- 注释
COMMENT ON TABLE sys_vdi_server IS 'VDI服务器配置表';
COMMENT ON TABLE sys_vdi_vm IS 'VDI虚拟机信息表';
COMMENT ON TABLE sys_vdi_resource_group IS 'VDI资源组表';
COMMENT ON TABLE sys_vdi_user_binding IS 'VDI用户绑定关系表';

COMMENT ON COLUMN sys_vdi_server.password_encrypted IS 'SM4加密的密码';
COMMENT ON COLUMN sys_vdi_server.auth_token IS '缓存的认证token，避免频繁登录';
COMMENT ON COLUMN sys_vdi_vm.vm_id IS '深信服VDI返回的虚拟机唯一标识';
COMMENT ON COLUMN sys_vdi_vm.power_state IS '电源状态: running=运行中, stopped=已关机, suspended=已休眠';
COMMENT ON COLUMN sys_vdi_user_binding.bound_at IS '用户与虚拟机绑定的 时间';
