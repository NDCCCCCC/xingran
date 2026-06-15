-- Migration: 032_create_info_point_table.sql
-- Description: 创建信息点表，用于关联工位和设备端口
-- Date: 2025-01-14

-- 创建信息点表
CREATE TABLE IF NOT EXISTS ops_info_points (
    id VARCHAR(64) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    -- 基本信息
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    info_point_type VARCHAR(50) NOT NULL,

    -- 关联信息
    workstation_id VARCHAR(64) NOT NULL,
    workstation_name VARCHAR(100),
    device_id VARCHAR(64),
    device_name VARCHAR(100),
    port_id VARCHAR(64),
    port_name VARCHAR(100),

    -- 状态
    status INTEGER NOT NULL DEFAULT 0,

    -- 备注
    remark VARCHAR(500),

    -- 约束
    CONSTRAINT chk_info_point_type CHECK (info_point_type IN ('network', 'power', 'other'))
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_ops_info_points_workstation_id ON ops_info_points(workstation_id);
CREATE INDEX IF NOT EXISTS idx_ops_info_points_device_id ON ops_info_points(device_id);
CREATE INDEX IF NOT EXISTS idx_ops_info_points_port_id ON ops_info_points(port_id);
CREATE INDEX IF NOT EXISTS idx_ops_info_points_type ON ops_info_points(info_point_type);
CREATE INDEX IF NOT EXISTS idx_ops_info_points_status ON ops_info_points(status);
CREATE INDEX IF NOT EXISTS idx_ops_info_points_deleted_at ON ops_info_points(deleted_at);

-- 添加唯一约束（编码）
CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_info_points_code ON ops_info_points(code) WHERE deleted_at IS NULL;
