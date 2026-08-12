-- 重建 API 密钥菜单
-- 确保"密钥列表"菜单存在并正确配置

-- 第一步：删除任何旧的API密钥相关菜单（避免冲突）
DELETE FROM sys_role_menu
WHERE menu_id IN (
    SELECT id FROM sys_menu WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志')
);

DELETE FROM sys_menu WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志');

-- 第二步：创建"密钥列表"菜单（直接在系统管理下）
INSERT INTO sys_menu (
    id,
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
    created_at,
    updated_at
) VALUES (
    gen_random_uuid(),
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
    'API密钥管理页面（含查看、创建、编辑、删除及使用日志）',
    NOW(),
    NOW()
);

-- 第三步：为管理员角色分配权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT id, menu_id
FROM (
    SELECT
        (SELECT id FROM sys_role WHERE role_key = 'admin' LIMIT 1) as role_id,
        (SELECT id FROM sys_menu WHERE menu_name = '密钥列表' LIMIT 1) as menu_id
) sq
WHERE role_id IS NOT NULL AND menu_id IS NOT NULL
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 验证结果
SELECT
    '重建后的菜单配置' as "说明",
    id as "菜单ID",
    menu_name as "菜单名称",
    parent_id as "父菜单ID",
    path as "路径",
    component as "组件路径",
    menu_type as "类型",
    visible as "可见",
    status as "状态",
    order_num as "排序"
FROM sys_menu
WHERE menu_name = '密钥列表';

-- 验证权限分配
SELECT
    '权限分配' as "说明",
    r.role_name,
    m.menu_name,
    rm.menu_id
FROM sys_role_menu rm
JOIN sys_role r ON r.id = rm.role_id
JOIN sys_menu m ON m.id = rm.menu_id
WHERE m.menu_name = '密钥列表';

SELECT '✅ API密钥菜单重建完成' AS status;
