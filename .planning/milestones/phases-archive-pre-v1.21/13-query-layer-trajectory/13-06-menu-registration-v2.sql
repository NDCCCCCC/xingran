-- Phase 13-06: MAC轨迹查询菜单注册脚本 (版本2)
-- 执行时间: 2026-06-13
-- 说明: 将MAC轨迹查询页面注册到现有的"网络设备-MAC地址"菜单下

-- 步骤1: 查询现有的"网络设备-MAC地址"菜单信息
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
  icon
FROM sys_menu
WHERE menu_name LIKE '%MAC%' OR path LIKE '%mac%' OR path LIKE '%device%'
ORDER BY parent_id, order_num;

-- 步骤2: 注册 "MAC轨迹查询" 子菜单到现有父菜单
-- 注意: 需要根据步骤1的查询结果确认父菜单ID
-- 假设父菜单ID为 X (请根据实际查询结果替换)

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
  -- TODO: 替换为实际的父菜单ID (从步骤1查询结果获取)
  (SELECT menu_id FROM sys_menu WHERE path = 'network/mac/address' LIMIT 1),
  10,
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
WHERE path = 'network/mac/trajectory';

-- 预期结果: 1行记录 (MAC轨迹查询子菜单)
