-- Phase 13-06: MAC轨迹查询菜单注册脚本 (版本3 - 适配实际表结构)
-- 执行时间: 2026-06-13

-- 步骤1: 查看 sys_menu 表的实际列名
-- SELECT * FROM sys_menu LIMIT 1;

-- 步骤2: 查询MAC相关菜单 (使用正确的列名)
SELECT
  id,
  menu_name,
  path,
  parent_id,
  order_num
FROM sys_menu
WHERE menu_name LIKE '%MAC%' OR path LIKE '%mac%'
ORDER BY parent_id, order_num;

-- 如果上面的查询失败，请尝试以下查询查看表结构：
-- SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'sys_menu';

-- 步骤3: 根据步骤2的查询结果，获取父菜单ID后执行以下插入

-- 示例 (请替换实际的父菜单ID):
-- INSERT INTO sys_menu (
--   menu_name,
--   menu_type,
--   path,
--   component,
--   parent_id,
--   order_num,
--   visible,
--   status,
--   perms,
--   icon,
--   create_by,
--   update_by,
--   created_at,
--   updated_at,
--   remark
-- ) VALUES (
--   'MAC轨迹查询',
--   'C',
--   'network/mac/trajectory',
--   'pages/network/mac/trajectory',
--   <父菜单ID>,  -- 替换为实际的父菜单ID
--   10,
--   1,
--   0,
--   'network:mac:trajectory:list',
--   'line-chart',
--   'admin',
--   'admin',
--   NOW(),
--   NOW(),
--   'MAC地址历史轨迹查询页面'
-- );
