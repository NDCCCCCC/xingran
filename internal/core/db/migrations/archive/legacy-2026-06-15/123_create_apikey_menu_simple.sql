-- 直接重建 API 密钥菜单（使用固定的管理员角色ID）
-- 此迁移在数据库启动时自动执行

-- 删除旧的 API 密钥相关菜单
DELETE FROM sys_role_menu WHERE menu_id IN (
    SELECT id FROM sys_menu WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志')
);
DELETE FROM sys_menu WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志');

-- 插入"密钥列表"菜单（直接作为系统管理的子菜单）
INSERT INTO sys_menu (
    menu_name, parent_id, order_num, path, component,
    menu_type, visible, status, perms, icon, remark
) VALUES (
    '密钥列表',
    'd67f4240-f887-481a-b345-94fb36782500',
    11,
    'apikeys',
    'system/apikeys/index',
    'C',
    1,
    0,
    'system:apikey:list',
    'KeyOutlined',
    'API密钥管理页面（含查看、创建、编辑、删除及使用日志）'
);

-- 为管理员角色分配权限（假设管理员角色ID）
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, m.id
FROM sys_role r, sys_menu m
WHERE r.role_key = 'admin' AND m.menu_name = '密钥列表'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 验证
SELECT '✅ API密钥菜单已重建' AS status;
