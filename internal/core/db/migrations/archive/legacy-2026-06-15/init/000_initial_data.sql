-- ========================================
-- XingRan-Next 初始数据
-- ========================================
-- 说明: 新环境部署所需的初始数据
-- 生成时间: 2026-05-21
-- 
-- 使用方法:
--   psql -U postgres -d xingran_next -f init/000_initial_data.sql
--
-- 注意: 
--   - 表结构由 GORM AutoMigrate 自动创建
--   - 此文件仅包含初始数据（菜单、字典、配置等）
--   - 执行前请确保数据库已创建
-- ========================================

-- 开始事务
BEGIN;

-- ========================================
-- 1. 网络设备类型字典
-- ========================================
-- ==========================================
-- 网络设备类型字典初始化
-- ==========================================

-- 插入字典类型
INSERT INTO "sys_dict_type" ("id", "dict_name", "dict_type", "status", "remark", "created_at", "updated_at")
VALUES
  ('device-type-001', '网络设备类型', 'network_device_type', 0, '网络设备类型映射', NOW(), NOW())
ON CONFLICT ("dict_type") DO NOTHING;

-- 插入字典数据
INSERT INTO "sys_dict_data" ("id", "dict_sort", "dict_label", "dict_value", "dict_type", "css_class", "list_class", "is_default", "status", "remark", "created_at", "updated_at")
VALUES
  ('device-data-001', 1, '路由器', 'router', 'network_device_type', NULL, 'default', false, 0, '路由器设备', NOW(), NOW()),
  ('device-data-002', 2, '交换机', 'switch', 'network_device_type', NULL, 'default', false, 0, '交换机设备', NOW(), NOW()),
  ('device-data-003', 3, '防火墙', 'firewall', 'network_device_type', NULL, 'default', false, 0, '防火墙设备', NOW(), NOW()),
  ('device-data-004', 4, '无线AP', 'ap', 'network_device_type', NULL, 'default', false, 0, '无线接入点', NOW(), NOW()),
  ('device-data-005', 5, '负载均衡器', 'loadbalancer', 'network_device_type', NULL, 'default', false, 0, '负载均衡设备', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- ========================================
-- 初始数据第一部分添加完成
