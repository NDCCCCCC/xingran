-- Phase 13-06: MAC轨迹查询菜单注册脚本
-- 执行时间: 2026-06-13
-- 说明: 注册MAC地址轨迹查询菜单到 sys_menu 表

-- 步骤1: 创建父菜单 "网络MAC管理"（如果不存在）
INSERT INTO sys_menu (
  menu_name,
  menu_type,
  path,
  component,
  parent_id,
  order_num,
  visible,
  status,
  perms,
  icon,
  create_by,
  update_by,
  created_at,
  updated_at,
  remark
) VALUES (
  '网络MAC管理',
  'M',
  'network/mac',
  NULL,
  (SELECT id FROM (SELECT id FROM sys_menu WHERE menu_name = '网络管理' LIMIT 1) AS parent),
  10,
  1,
  0,
  NULL,
  'wifi',
  'admin',
  'admin',
  NOW(),
  NOW(),
  'MAC地址历史管理和轨迹查询'
)
ON CONFLICT (path) DO NOTHING;

-- 步骤2: 注册 "MAC轨迹查询" 子菜单
INSERT INTO sys_menu (
  menu_name,
  menu_type,
  path,
  component,
  parent_id,
  order_num,
  visible,
  status,
  perms,
  icon,
  create_by,
  update_by,
  created_at,
  updated_at,
  remark
) VALUES (
  'MAC轨迹查询',
  'C',
  'network/mac/trajectory',
  'pages/network/mac/trajectory',
  (SELECT id FROM sys_menu WHERE path = 'network/mac' LIMIT 1),
  5,
  1,
  0,
  'network:mac:trajectory:list',
  'line-chart',
  'admin',
  'admin',
  NOW(),
  NOW(),
  'MAC地址历史轨迹查询页面'
)
ON CONFLICT (path) DO NOTHING;

-- 步骤3: 验证菜单注册
SELECT
  menu_id,
  menu_name,
  menu_type,
  path,
  component,
  parent_id,
  order_num,
  visible,
  status,
  perms,
  icon,
  created_at
FROM sys_menu
WHERE path IN ('network/mac', 'network/mac/trajectory')
ORDER BY parent_id, order_num;

-- 预期结果: 2行记录（父菜单 + 子菜单）
-- 如无结果，请检查网络管理父菜单是否存在
