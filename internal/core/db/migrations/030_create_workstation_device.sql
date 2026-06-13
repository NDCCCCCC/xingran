-- Phase 32 P2-A4 source-tracking:
--   Original commit: 71c6fe21
--   Created: 2026-06-10
--   Note: Conflicts with 030_add_building_spaces_3d_menu.sql and 030_enhance_workstation_table.sql — three files share prefix 030. Runner uses Go code ordering, not filename sort; conflict is harmless.

-- Migration: 030_create_workstation_device.sql
-- Description: 创建工位设备关联表
-- Date: 2026-06-10
-- Phase: 28-01

CREATE TABLE ops_workstation_device (
    id VARCHAR(36) PRIMARY KEY,
    workstation_id VARCHAR(36) NOT NULL COMMENT '工位ID',
    device_source VARCHAR(20) NOT NULL COMMENT '设备来源: ad, asset, manual',
    device_serial VARCHAR(200) COMMENT '序列号',
    device_name VARCHAR(255) COMMENT '设备名称',
    device_model VARCHAR(200) COMMENT '型号',
    device_type VARCHAR(100) COMMENT '类型',
    mac_address VARCHAR(100) COMMENT 'MAC地址',
    asset_id VARCHAR(36) COMMENT '关联资产ID',
    ad_computer_id VARCHAR(36) COMMENT '关联AD设备ID',
    responsible_user VARCHAR(100) COMMENT '责任人',
    responsible_user_id VARCHAR(64) COMMENT '责任人ID',
    status INT DEFAULT 0 COMMENT '状态: 0=正常, 1=停用',
    is_primary BOOLEAN DEFAULT FALSE COMMENT '是否主设备',
    priority INT DEFAULT 0 COMMENT '优先级',
    description TEXT COMMENT '备注',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL COMMENT '软删除时间戳',
    version INT DEFAULT 0 COMMENT '乐观锁版本号',
    CONSTRAINT fk_workstation_device_workstation FOREIGN KEY (workstation_id)
        REFERENCES ops_workstations(id) ON DELETE CASCADE,
    CONSTRAINT fk_workstation_device_asset FOREIGN KEY (asset_id)
        REFERENCES ops_assets(id) ON DELETE SET NULL,
    CONSTRAINT fk_workstation_device_ad_computer FOREIGN KEY (ad_computer_id)
        REFERENCES sys_ad_computer(id) ON DELETE SET NULL,
    INDEX idx_workstation (workstation_id),
    INDEX idx_device_serial (device_serial),
    INDEX idx_asset (asset_id),
    INDEX idx_ad_computer (ad_computer_id),
    INDEX idx_source_status (device_source, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='工位设备关联表';
