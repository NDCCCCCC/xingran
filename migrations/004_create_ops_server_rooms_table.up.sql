-- 004_create_ops_server_rooms_table.up.sql
CREATE TABLE IF NOT EXISTS ops_server_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE,
    floor_id UUID NOT NULL,
    status INTEGER DEFAULT 0,
    remark TEXT,

    FOREIGN KEY (floor_id) REFERENCES ops_floors(id)
);

-- 创建索引
CREATE INDEX idx_ops_server_rooms_floor_id ON ops_server_rooms(floor_id);
CREATE INDEX idx_ops_server_rooms_status ON ops_server_rooms(status);
CREATE INDEX idx_ops_server_rooms_deleted_at ON ops_server_rooms(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_server_rooms IS '机房表';
COMMENT ON COLUMN ops_server_rooms.status IS '状态: 0=正常, 1=停用';
