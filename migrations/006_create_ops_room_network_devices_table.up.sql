-- 006_create_ops_room_network_devices_table.up.sql
CREATE TABLE IF NOT EXISTS ops_room_network_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    room_id UUID NOT NULL,
    device_id UUID NOT NULL,
    position_u INTEGER NOT NULL DEFAULT 0,
    position_desc VARCHAR(100),

    FOREIGN KEY (room_id) REFERENCES ops_server_rooms(id)
);

-- 创建索引
CREATE INDEX idx_ops_room_network_devices_room_id ON ops_room_network_devices(room_id);
CREATE INDEX idx_ops_room_network_devices_device_id ON ops_room_network_devices(device_id);
CREATE INDEX idx_ops_room_network_devices_deleted_at ON ops_room_network_devices(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_room_network_devices IS '机房-网络设备关联表';
COMMENT ON COLUMN ops_room_network_devices.position_u IS '机架位置（U）';
