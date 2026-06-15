-- Migration: 150_add_workstation_device_ip_address.sql
-- Description: 为工位设备关联表添加 IP 地址字段
-- Date: 2026-06-12
-- Reason: 支持子表格展示与编辑手动添加设备的 IP 地址
-- Note 1: 使用 IF NOT EXISTS 保证幂等
-- Note 2: PG 不支持 AFTER column 语法 (已移除)
-- Warning: 此文件被 executeSQL() 按 `;` 分割执行, 注释内禁用分号

ALTER TABLE ops_workstation_device
    ADD COLUMN IF NOT EXISTS ip_address VARCHAR(64);

COMMENT ON COLUMN ops_workstation_device.ip_address IS 'IP地址';

CREATE INDEX IF NOT EXISTS idx_workstation_device_ip ON ops_workstation_device(ip_address);
