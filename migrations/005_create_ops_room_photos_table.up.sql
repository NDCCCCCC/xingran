-- 005_create_ops_room_photos_table.up.sql
CREATE TABLE IF NOT EXISTS ops_room_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    room_id UUID NOT NULL,
    file_id UUID NOT NULL,
    sort_order INTEGER DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,
    description TEXT,

    FOREIGN KEY (room_id) REFERENCES ops_server_rooms(id)
);

-- 创建索引
CREATE INDEX idx_ops_room_photos_room_id ON ops_room_photos(room_id);
CREATE INDEX idx_ops_room_photos_file_id ON ops_room_photos(file_id);
CREATE INDEX idx_ops_room_photos_deleted_at ON ops_room_photos(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_room_photos IS '机房照片关联表';
COMMENT ON COLUMN ops_room_photos.is_primary IS '是否为主图';
