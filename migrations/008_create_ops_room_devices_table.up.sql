-- 008_create_ops_room_devices_table.up.sql
CREATE TABLE IF NOT EXISTS ops_room_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    device_code VARCHAR(50) NOT NULL UNIQUE,
    device_type VARCHAR(20) NOT NULL,
    model VARCHAR(100),
    vendor VARCHAR(100),
    serial_number VARCHAR(100),
    room_id UUID NOT NULL,
    position_u INTEGER DEFAULT 0,
    position_desc VARCHAR(100),
    asset_number VARCHAR(50),
    purchase_date TIMESTAMP,
    warranty_date TIMESTAMP,
    status INTEGER DEFAULT 0,
    responsible_id UUID,
    power_consumption INTEGER DEFAULT 0,
    remark TEXT,

    FOREIGN KEY (room_id) REFERENCES ops_server_rooms(id)
);

-- 创建索引
CREATE INDEX idx_ops_room_devices_room_id ON ops_room_devices(room_id);
CREATE INDEX idx_ops_room_devices_responsible_id ON ops_room_devices(responsible_id);
CREATE INDEX idx_ops_room_devices_status ON ops_room_devices(status);
CREATE INDEX idx_ops_room_devices_deleted_at ON ops_room_devices(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_room_devices IS '机房设备表';
COMMENT ON COLUMN ops_room_devices.device_type IS '设备类型: server/storage/ups/pdu/firewall/kvm/cabinet/other';
COMMENT ON COLUMN ops_room_devices.status IS '状态: 0=正常, 1=故障, 2=报废';
COMMENT ON COLUMN ops_room_devices.power_consumption IS '功耗（瓦）';
