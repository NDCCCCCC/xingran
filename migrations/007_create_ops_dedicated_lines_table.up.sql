-- 007_create_ops_dedicated_lines_table.up.sql
CREATE TABLE IF NOT EXISTS ops_dedicated_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by VARCHAR(64),
    updated_by VARCHAR(64),
    version INTEGER DEFAULT 0,

    name VARCHAR(100) NOT NULL,
    line_code VARCHAR(50) NOT NULL UNIQUE,
    line_type VARCHAR(20) NOT NULL,
    bandwidth VARCHAR(20),
    isp VARCHAR(50),
    source_device_id UUID,
    source_port VARCHAR(50),
    dest_device_id UUID,
    dest_port VARCHAR(50),
    ip_address VARCHAR(50),
    subnet_mask VARCHAR(50),
    gateway VARCHAR(50),
    vlan VARCHAR(20),
    status INTEGER DEFAULT 0,
    activation_date TIMESTAMP,
    expiry_date TIMESTAMP,
    responsible_id UUID,
    contract_no VARCHAR(50),
    monthly_fee DECIMAL(10,2) DEFAULT 0,
    remark TEXT
);

-- 创建索引
CREATE INDEX idx_ops_dedicated_lines_source_device ON ops_dedicated_lines(source_device_id);
CREATE INDEX idx_ops_dedicated_lines_dest_device ON ops_dedicated_lines(dest_device_id);
CREATE INDEX idx_ops_dedicated_lines_status ON ops_dedicated_lines(status);
CREATE INDEX idx_ops_dedicated_lines_deleted_at ON ops_dedicated_lines(deleted_at);

-- 添加注释
COMMENT ON TABLE ops_dedicated_lines IS '专线表';
COMMENT ON COLUMN ops_dedicated_lines.line_type IS '专线类型: internet/mpls/leased_line/fiber';
COMMENT ON COLUMN ops_dedicated_lines.status IS '状态: 0=正常, 1=故障, 2=停用';
