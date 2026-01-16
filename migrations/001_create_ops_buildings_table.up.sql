-- 001_create_ops_buildings_table.up.sql
CREATE TABLE IF NOT EXISTS ops_buildings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) NOT NULL UNIQUE,
    address VARCHAR(200),
    org_id UUID NOT NULL,
    status INTEGER DEFAULT 0,
    remark TEXT,
    order_num INTEGER DEFAULT 0
);

-- 创建索引
CREATE INDEX idx_ops_buildings_org_id ON ops_buildings(org_id);
CREATE INDEX idx_ops_buildings_status ON ops_buildings(status);
CREATE INDEX idx_ops_buildings_deleted_at ON ops_buildings(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_buildings IS '楼宇表';
COMMENT ON COLUMN ops_buildings.status IS '状态: 0=正常, 1=停用';
