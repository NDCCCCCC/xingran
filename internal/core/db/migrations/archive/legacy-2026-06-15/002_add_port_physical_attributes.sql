-- ==========================================
-- 端口状态表添加物理属性字段
-- 用于存储从 show interfaces status 获取的 VLAN、Duplex、Speed、Type
-- ==========================================

-- 添加新字段
ALTER TABLE sys_device_port_status
ADD COLUMN IF NOT EXISTS vlan INTEGER,
ADD COLUMN IF NOT EXISTS duplex VARCHAR(20),
ADD COLUMN IF NOT EXISTS speed VARCHAR(20),
ADD COLUMN IF NOT EXISTS port_type VARCHAR(50);

-- 添加字段注释
COMMENT ON COLUMN sys_device_port_status.vlan IS 'VLAN ID';
COMMENT ON COLUMN sys_device_port_status.duplex IS '双工模式 (Full/Half/Unknown)';
COMMENT ON COLUMN sys_device_port_status.speed IS '速率 (100M/1G/10G等)';
COMMENT ON COLUMN sys_device_port_status.port_type IS '端口类型 (copper/fiber等)';
