-- 修复 VDI 菜单路径配置
-- 问题：二级菜单路径包含父路径前缀，导致 React Router 路径匹配失败
-- 解决：将二级菜单路径改为相对路径（不包含父路径）

-- 更新虚拟机列表菜单路径
UPDATE sys_menu
SET path = 'vm'
WHERE id = '770e8400-e29b-41d4-a716-446655440002'
  AND menu_name = '虚拟机列表'
  AND path = 'vdi/vm';

-- 更新VDI服务器配置菜单路径
UPDATE sys_menu
SET path = 'servers'
WHERE id = '770e8400-e29b-41d4-a716-446655440018'
  AND menu_name = 'VDI服务器配置'
  AND path = 'vdi/servers';

-- 更新虚拟机详情菜单路径（隐藏菜单）
UPDATE sys_menu
SET path = 'vm/:id'
WHERE id = '770e8400-e29b-41d4-a716-446655440012'
  AND menu_name = '虚拟机详情'
  AND path = 'vdi/vm/:id';

-- 验证更新结果
SELECT id, menu_name, parent_id, path, component, menu_type
FROM sys_menu
WHERE id LIKE '770e8400-e29b-41d4-a716-446655440%'
ORDER BY menu_name;
