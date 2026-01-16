-- 003_create_ops_workstations_table.up.sql
CREATE TABLE IF NOT EXISTS ops_workstations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    code VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL,
    floor_id UUID NOT NULL,
    user_id UUID,
    dept_id UUID,
    phone_point VARCHAR(20),
    status INTEGER DEFAULT 0,
    type INTEGER DEFAULT 0,
    equipment TEXT,
    network_info VARCHAR(500),
    remark TEXT,
    position_x INTEGER,
    position_y INTEGER,

    FOREIGN KEY (floor_id) REFERENCES ops_floors(id)
);

-- 创建索引
CREATE INDEX idx_ops_workstations_floor_id ON ops_workstations(floor_id);
CREATE INDEX idx_ops_workstations_user_id ON ops_workstations(user_id);
CREATE INDEX idx_ops_workstations_dept_id ON ops_workstations(dept_id);
CREATE INDEX idx_ops_workstations_status ON ops_workstations(status);
CREATE INDEX idx_ops_workstations_deleted_at ON ops_workstations(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_workstations IS '工位表';
COMMENT ON COLUMN ops_workstations.status IS '状态: 0=空闲, 1=占用';
COMMENT ON COLUMN ops_workstations.type IS '类型: 0=固定, 1=共享';
