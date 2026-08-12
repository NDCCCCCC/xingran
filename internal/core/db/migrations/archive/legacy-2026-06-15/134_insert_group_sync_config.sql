-- 插入AD组同步配置参数
INSERT INTO sys_config (config_name, config_key, config_value, config_type, description, created_at, updated_at)
VALUES
  ('组同步开关', 'sys.ad.group.sync.enabled', 'false', 'boolean', '是否启用AD组自动同步功能', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('组同步Cron表达式', 'sys.ad.group.sync.cron', '0 */15 * * * *', 'string', '组同步的Cron表达式，默认每15分钟执行一次', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('MemberOU路径', 'sys.ad.group.member_ou', '', 'string', '指定部门组的目标OU路径，留空则使用AD配置的MemberOUDN', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('自动创建组', 'sys.ad.group.auto_create', 'true', 'boolean', '是否自动创建不存在的AD组', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('最大并发同步数', 'sys.ad.group.max_concurrent', '5', 'number', '同时执行的最大同步任务数，范围1-20', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('批量同步大小', 'sys.ad.group.sync.batch_size', '100', 'number', '每次批量处理的成员数量，范围10-1000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (config_key) DO NOTHING;