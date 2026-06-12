-- Migration: 150_add_workstation_device_ip_address.sql
-- Description: 为工位设备关联表添加 IP 地址字段
-- Date: 2026-06-12
-- Reason: 支持子表格展示与编辑手动添加设备的 IP 地址

ALTER TABLE ops_workstation_device
    ADD COLUMN ip_address VARCHAR(64) COMMENT 'IP地址' AFTER mac_address;

CREATE INDEX idx_workstation_device_ip ON ops_workstation_device(ip_address);
