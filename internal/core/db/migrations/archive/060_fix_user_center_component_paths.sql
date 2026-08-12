-- Fix user center component paths
UPDATE sys_menu SET component = 'profile/index' WHERE menu_name = '个人中心' AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL);

UPDATE sys_menu SET component = 'settings/index' WHERE menu_name = '系统设置' AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL);

UPDATE sys_menu SET component = 'my-notices/index' WHERE menu_name = '我的通知' AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL);
