-- 003_add_network_concurrent_config.sql
-- 添加网络设备定时任务并发数配置参数

-- 设备监控并发数：同时处理多少个设备的状态检查/信息更新
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES (gen_random_uuid()::text, '网络设备监控并发数', 'network.device.monitor.concurrent', '10', 'Y', 1, '设备状态检查和信息更新的最大并发数，默认10', NOW(), NOW())
ON CONFLICT (config_key) DO NOTHING;

-- 端口采集并发数：同时采集多少个设备的端口状态
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES (gen_random_uuid()::text, '端口采集并发数', 'network.port.collection.concurrent', '10', 'Y', 1, '端口状态采集的最大并发数，默认10', NOW(), NOW())
ON CONFLICT (config_key) DO NOTHING;

-- MAC地址采集并发数：同时采集多少个设备的MAC地址表
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES (gen_random_uuid()::text, 'MAC地址采集并发数', 'network.mac.collection.concurrent', '10', 'Y', 1, 'MAC地址表采集的最大并发数，默认10', NOW(), NOW())
ON CONFLICT (config_key) DO NOTHING;

-- 配置备份并发数：同时备份多少个设备
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES (gen_random_uuid()::text, '配置备份并发数', 'network.config.backup.concurrent', '5', 'Y', 1, '配置备份的最大并发数，默认5（配置备份较耗时，建议较低并发）', NOW(), NOW())
ON CONFLICT (config_key) DO NOTHING;

-- 设备连接超时时间（秒）
INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark, created_at, updated_at)
VALUES (gen_random_uuid()::text, '设备连接超时时间', 'network.device.timeout', '30', 'Y', 1, '单个设备连接和操作的超时时间（秒），默认30秒', NOW(), NOW())
ON CONFLICT (config_key) DO NOTHING;
