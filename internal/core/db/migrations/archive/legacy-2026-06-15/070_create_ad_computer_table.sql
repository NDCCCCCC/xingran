-- =============================================
-- AD域电脑表
-- 迁移版本: 070
-- 描述: 创建AD域电脑信息表，用于存储从AD域同步的计算机设备信息
-- =============================================

-- 创建 AD 域电脑表
CREATE TABLE IF NOT EXISTS sys_ad_computer (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ad_config_id uuid NOT NULL,
    computer_name VARCHAR(255) NOT NULL,
    distinguished_name VARCHAR(500) NOT NULL,
    last_logon TIMESTAMP WITH TIME ZONE,
    password_last_set TIMESTAMP WITH TIME ZONE,
    logon_count INTEGER DEFAULT 0,
    ou_dn VARCHAR(500),
    status INTEGER DEFAULT 0,
    original_description TEXT,
    ip_address VARCHAR(50),
    mac_address VARCHAR(50),
    managed_by VARCHAR(255),
    operating_system VARCHAR(255),
    os_version VARCHAR(255),
    cpu_model VARCHAR(255),
    architecture VARCHAR(50),
    memory_capacity VARCHAR(50),
    hard_disk_capacity VARCHAR(50),
    last_online_time TIMESTAMP WITH TIME ZONE,
    serial_number VARCHAR(255),
    system_info TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0
);

-- 创建索引
CREATE UNIQUE INDEX IF NOT EXISTS uk_ad_computer_name ON sys_ad_computer(computer_name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ad_computer_config ON sys_ad_computer(ad_config_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ad_computer_name ON sys_ad_computer(computer_name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ad_computer_ou ON sys_ad_computer(ou_dn) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ad_computer_status ON sys_ad_computer(status) WHERE deleted_at IS NULL;

-- 添加注释
COMMENT ON TABLE sys_ad_computer IS 'AD域计算机设备信息表';
COMMENT ON COLUMN sys_ad_computer.computer_name IS '计算机名称';
COMMENT ON COLUMN sys_ad_computer.distinguished_name IS 'AD域专有名称(DN)';
COMMENT ON COLUMN sys_ad_computer.ou_dn IS '所属OU的DN';
COMMENT ON COLUMN sys_ad_computer.status IS '状态: 0=在线, 1=离线';
COMMENT ON COLUMN sys_ad_computer.ip_address IS 'IP地址';
COMMENT ON COLUMN sys_ad_computer.mac_address IS 'MAC地址';
COMMENT ON COLUMN sys_ad_computer.managed_by IS '管理者';
COMMENT ON COLUMN sys_ad_computer.operating_system IS '操作系统';
COMMENT ON COLUMN sys_ad_computer.os_version IS '操作系统版本';
COMMENT ON COLUMN sys_ad_computer.cpu_model IS 'CPU型号';
COMMENT ON COLUMN sys_ad_computer.architecture IS '架构(32/64位)';
COMMENT ON COLUMN sys_ad_computer.memory_capacity IS '内存容量';
COMMENT ON COLUMN sys_ad_computer.hard_disk_capacity IS '硬盘容量';
