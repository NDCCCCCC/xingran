-- Phase 14-03: MAC 历史查询菜单注册脚本
-- 执行时间: 2026-06-14
-- 执行人: Phase 14-03 executor (autonomous plan)
-- 父菜单: MAC地址 (id: 0013f129-3ec0-4e55-8ffc-25d97b20c37b, path: mac)
-- 沿用: 13-06 v4 SQL 模板的列序与命名规范
-- 职责边界: 本脚本仅注册菜单条目与权限点;network:mac:export 按钮的渲染与拦截由 14-04 实施

-- ============================================================================
-- 步骤1: 验证父菜单存在(预期返回 1 行,menu_name = 'MAC地址', path = 'mac')
-- ============================================================================
SELECT
  id,
  menu_name,
  path,
  parent_id,
  order_num
FROM sys_menu
WHERE id = '0013f129-3ec0-4e55-8ffc-25d97b20c37b';

-- ============================================================================
-- 步骤2: 幂等注册 "MAC 历史查询" 子菜单 (菜单项, menu_type = 'C')
-- 路径: path = 'mac/history' (相对父菜单 'mac')
-- 组件: component = 'pages/network/mac/history' (14-01 计划创建的页面)
-- 排序: order_num = 11 (排在 MAC轨迹查询 = 10 之后)
-- 权限: perms = 'network:mac:list' (主权限点,用于菜单可见性判断)
-- ============================================================================
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
  gen_random_uuid(),                              -- PostgreSQL 自动生成 UUID
  NOW(),                                           -- created_at
  NOW(),                                           -- updated_at
  'admin',                                         -- created_by
  'admin',                                         -- updated_by
  0,                                               -- version
  'MAC 历史查询',                                  -- menu_name
  '0013f129-3ec0-4e55-8ffc-25d97b20c37b',          -- parent_id (MAC地址菜单ID)
  11,                                              -- order_num (在父菜单中排在MAC轨迹查询=10之后)
  'mac/history',                                   -- path (相对父菜单)
  'pages/network/mac/history',                    -- component (前端组件路径,14-01 创建)
  'C',                                             -- menu_type (C = 菜单项)
  1,                                               -- visible (1 = 显示)
  0,                                               -- status (0 = 正常)
  'network:mac:list',                              -- perms (主权限点)
  'history',                                       -- icon (图标)
  'MAC地址历史数据查询、导出页面',                 -- remark
  '{"icon": "history", "affix": false, "title": "MAC 历史查询", "hidden": false, "keepAlive": false}'::jsonb -- meta
)
WHERE NOT EXISTS (
  SELECT 1 FROM sys_menu
  WHERE path = 'mac/history'
    AND parent_id = '0013f129-3ec0-4e55-8ffc-25d97b20c37b'
);

-- ============================================================================
-- 步骤3: 注册按钮权限点 "MAC历史查询-查询" (menu_type = 'F', 不可见)
-- 用途: 控制 history.tsx 工具栏"查询"按钮的可见性,UI-01 实施时调用
-- ============================================================================
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
  gen_random_uuid(),
  NOW(),
  NOW(),
  'admin',
  'admin',
  0,
  'MAC历史查询-查询',
  '0013f129-3ec0-4e55-8ffc-25d97b20c37b',
  12,
  '',
  '',
  'F',
  0,
  0,
  'network:mac:query',
  '',
  'MAC历史查询页面的查询按钮权限点',
  '{"icon": "", "affix": false, "title": "MAC历史查询-查询", "hidden": true, "keepAlive": false}'::jsonb
)
WHERE NOT EXISTS (
  SELECT 1 FROM sys_menu WHERE perms = 'network:mac:query'
);

-- ============================================================================
-- 步骤4: 注册按钮权限点 "MAC历史查询-导出" (menu_type = 'F', 不可见)
-- 用途: 由 14-04 在 history.tsx 工具栏"导出"按钮上做可见性判断
-- 归属: 本脚本仅注册权限点;实际按钮渲染/下载/错误处理由 14-04 实施
-- ============================================================================
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
  gen_random_uuid(),
  NOW(),
  NOW(),
  'admin',
  'admin',
  0,
  'MAC历史查询-导出',
  '0013f129-3ec0-4e55-8ffc-25d97b20c37b',
  13,
  '',
  '',
  'F',
  0,
  0,
  'network:mac:export',
  '',
  'MAC历史查询页面的导出按钮权限点(由14-04消费)',
  '{"icon": "", "affix": false, "title": "MAC历史查询-导出", "hidden": true, "keepAlive": false}'::jsonb
)
WHERE NOT EXISTS (
  SELECT 1 FROM sys_menu WHERE perms = 'network:mac:export'
);

-- ============================================================================
-- 步骤5: 验证注册结果
-- 预期返回: 1 行主菜单(menu_type='C') + 2 行按钮权限(menu_type='F') = 3 行
-- ============================================================================
SELECT
  id,
  menu_name,
  path,
  component,
  parent_id,
  order_num,
  menu_type,
  visible,
  status,
  perms,
  created_at
FROM sys_menu
WHERE perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export')
ORDER BY order_num;

-- ============================================================================
-- 回滚 SQL (如需撤销注册,执行以下语句;备份后再删)
-- ============================================================================
-- DELETE FROM sys_role_menu
--   WHERE menu_id IN (SELECT id FROM sys_menu
--                     WHERE perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export'));
-- DELETE FROM sys_menu
--   WHERE perms IN ('network:mac:list', 'network:mac:query', 'network:mac:export');