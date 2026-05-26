-- Migration 085: Create VDI tables
-- Description: 深信服VDI集成相关数据表

-- VDI服务器配置表
CREATE TABLE IF NOT EXISTS vdi_server (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    endpoint VARCHAR(500) NOT NULL,
    username VARCHAR(100) NOT NULL,
    password_encrypted VARCHAR(500) NOT NULL,
    tenant_id INTEGER DEFAULT 0,
    auth_token VARCHAR(1000),
    token_expiry TIMESTAMP,
    status INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- 虚拟机表
CREATE TABLE IF NOT EXISTS vdi_vm (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    vm_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    resource_id VARCHAR(100),
    status INTEGER DEFAULT 0,
    power_state VARCHAR(50),
    ip_address VARCHAR(50),
    mac_address VARCHAR(50),
    os_type VARCHAR(50),
    cpu INTEGER DEFAULT 0,
    memory INTEGER DEFAULT 0,
    disk INTEGER DEFAULT 0,
    bound_user_id VARCHAR(100),
    bound_user_name VARCHAR(200),
    policy_group_id VARCHAR(100),
    last_sync_at TIMESTAMP,
    vdi_server_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_vdi_vm_server FOREIGN KEY (vdi_server_id)
        REFERENCES vdi_server(id) ON DELETE CASCADE
);

-- VDI资源组表
CREATE TABLE IF NOT EXISTS vdi_resource_group (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(200) NOT NULL,
    vdi_server_id VARCHAR(100) NOT NULL,
    description TEXT,
    total_vms INTEGER DEFAULT 0,
    used_vms INTEGER DEFAULT 0,
    status INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- 外键约束
    CONSTRAINT fk_vdi_rg_server FOREIGN KEY (vdi_server_id)
        REFERENCES vdi_server(id) ON DELETE CASCADE
);

-- VDI用户绑定表
CREATE TABLE IF NOT EXISTS vdi_user_binding (
    id VARCHAR(100) PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) NOT NULL,
    user_name VARCHAR(200) NOT NULL,
    vm_id VARCHAR(100) NOT NULL,
    vm_name VARCHAR(200),
    vdi_server_id VARCHAR(100),
    is_primary BOOLEAN DEFAULT false,
    status INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_vdi_vm_resource_id ON vdi_vm(resource_id);
CREATE INDEX IF NOT EXISTS idx_vdi_vm_server_id ON vdi_vm(vdi_server_id);
CREATE INDEX IF NOT EXISTS idx_vdi_vm_bound_user_id ON vdi_vm(bound_user_id);
CREATE INDEX IF NOT EXISTS idx_vdi_rg_server_id ON vdi_resource_group(vdi_server_id);
CREATE INDEX IF NOT EXISTS idx_vdi_user_binding_user_id ON vdi_user_binding(user_id);
CREATE INDEX IF NOT EXISTS idx_vdi_user_binding_vm_id ON vdi_user_binding(vm_id);
CREATE INDEX IF NOT EXISTS idx_vdi_user_binding_server_id ON vdi_user_binding(vdi_server_id);

-- 添加注释
COMMENT ON TABLE vdi_server IS 'VDI服务器配置表';
COMMENT ON TABLE vdi_vm IS '虚拟机信息表';
COMMENT ON TABLE vdi_resource_group IS 'VDI资源组表';
COMMENT ON TABLE vdi_user_binding IS 'VDI用户绑定表';

COMMENT ON COLUMN vdi_vm.status IS '0=正常, 1=停用';
COMMENT ON COLUMN vdi_server.status IS '0=正常, 1=停用';
COMMENT ON COLUMN vdi_resource_group.status IS '0=正常, 1=停用';
COMMENT ON COLUMN vdi_user_binding.status IS '0=正常, 1=停用';
