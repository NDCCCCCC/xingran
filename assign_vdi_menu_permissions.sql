-- 修复 VDI 菜单插入 - 查找管理员角色 UUID
-- 查询管理员角色（通常是 role_key = 'admin'）
SELECT role_id, role_name, role_key
FROM sys_role
WHERE role_key = 'admin' OR role_name LIKE '%管理员%'
LIMIT 1;

-- 假设管理员角色 UUID 是查询结果中的 role_id，下面使用变量替换
-- 请手动将下面的 YOUR_ADMIN_ROLE_ID 替换为上面查询到的 role_id

-- 为管理员角色分配 VDI 菜单权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    (SELECT role_id FROM sys_role WHERE role_key = 'admin' LIMIT 1),
    id
FROM sys_menu
WHERE id::text LIKE '770e8400%'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 验证分配结果
SELECT
    r.role_name,
    m.menu_name,
    m.path
FROM sys_role_menu rm
INNER JOIN sys_role r ON rm.role_id = r.role_id
INNER JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.id::text LIKE '770e8400%'
ORDER BY m.order_num;
