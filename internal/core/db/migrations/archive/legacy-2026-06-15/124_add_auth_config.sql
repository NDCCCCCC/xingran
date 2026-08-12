-- Migration 124: 添加AD认证配置参数
-- Date: 2026-05-21
-- Description: 添加AD域控认证相关配置参数到sys_config表

-- 插入AD认证配置参数
-- config_type: 'Y' = system/built-in, 'S' = string, 'N' = non-system
-- is_system: 1 = system protected (cannot delete)
INSERT INTO sys_config (config_key, config_name, config_value, config_type, remark, created_by, is_system)
VALUES
  ('sys.auth.ad.enabled', 'AD认证启用', 'false', 'Y', '是否启用AD域控认证（true/false）', 'admin', 1),
  ('sys.auth.default.mode', '默认认证模式', 'local', 'Y', '默认认证模式：local=本地, ad=AD, hybrid=混合', 'admin', 1),
  ('sys.auth.ad.config_id', 'AD配置ID', '', 'Y', 'AD域控配置ID（为空则使用第一个启用的配置）', 'admin', 0),
  ('sys.auth.ad.default_role_id', 'AD用户默认角色', '', 'Y', 'AD用户首次登录时分配的默认角色ID', 'admin', 0),
  ('sys.auth.ad.default_dept_id', 'AD用户默认部门', '', 'Y', 'AD用户首次登录时分配的默认部门ID', 'admin', 0)
ON CONFLICT (config_key) DO NOTHING;

-- 创建索引（如果不存在）
CREATE INDEX IF NOT EXISTS idx_sys_config_key ON sys_config(config_key);
