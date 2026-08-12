-- Phase 13-06: MAC轨迹查询菜单注册脚本 (版本4 - 实际表结构)
-- 执行时间: 2026-06-13
-- 父菜单: MAC地址 (id: 0013f129-3ec0-4e55-8ffc-25d97b20c37b, path: mac)

-- 步骤1: 验证父菜单存在
SELECT
  id,
  menu_name,
  path,
  parent_id,
  order_num
FROM sys_menu
WHERE id = '0013f129-3ec0-4e55-8ffc-25d97b20c37b';

-- 步骤2: 注册 "MAC轨迹查询" 子菜单
INSERT INTO sys_menu (
  id,
  created_at,
  updated_at,
  created_by,
  updated_by,
  version,
  menu_name,
  parent_id,
  order_num,
  path,
  component,
  menu_type,
  visible,
  status,
  perms,
  icon,
  remark,
  meta
) VALUES (
  gen_random_uuid(),                    -- 使用PostgreSQL自动生成UUID
  NOW(),                                 -- created_at
  NOW(),                                 -- updated_at
  'admin',                               -- created_by
  'admin',                               -- updated_by
  0,                                     -- version
  'MAC轨迹查询',                         -- menu_name
  '0013f129-3ec0-4e55-8ffc-25d97b20c37b', -- parent_id (MAC地址菜单ID)
  10,                                    -- order_num (在父菜单中的排序)
  'mac/trajectory',                      -- path (相对于父菜单)
  'pages/network/mac/trajectory',       -- component (前端组件路径)
  'C',                                   -- menu_type (C=菜单项)
  1,                                     -- visible (1=显示)
  0,                                     -- status (0=正常)
  'network:mac:trajectory:list',        -- perms (权限标识)
  'line-chart',                          -- icon (图标)
  'MAC地址历史轨迹查询页面',             -- remark
  '{"icon": "line-chart", "affix": false, "title": "MAC轨迹查询", "hidden": false, "keepAlive": false}'::jsonb -- meta
);

-- 步骤3: 验证菜单注册结果
SELECT
  id,
  menu_name,
  path,
  component,
  parent_id,
  order_num,
  status,
  perms,
  created_at
FROM sys_menu
WHERE path = 'mac/trajectory'
   OR menu_name = 'MAC轨迹查询';

-- 预期结果: 1行记录，显示新注册的MAC轨迹查询菜单
-- 如果看到记录，说明注册成功，页面应该可以通过菜单访问
