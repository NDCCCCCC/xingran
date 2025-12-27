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
