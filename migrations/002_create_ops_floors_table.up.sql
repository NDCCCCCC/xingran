-- 002_create_ops_floors_table.up.sql
CREATE TABLE IF NOT EXISTS ops_floors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    floor_no VARCHAR(10) NOT NULL,
    building_id UUID NOT NULL,
    status INTEGER DEFAULT 0,
    remark TEXT,
    order_num INTEGER DEFAULT 0,

    FOREIGN KEY (building_id) REFERENCES ops_buildings(id)
);

-- 创建索引
CREATE INDEX idx_ops_floors_building_id ON ops_floors(building_id);
CREATE INDEX idx_ops_floors_status ON ops_floors(status);
CREATE INDEX idx_ops_floors_deleted_at ON ops_floors(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_floors IS '楼层表';
COMMENT ON COLUMN ops_floors.status IS '状态: 0=正常, 1=停用';
