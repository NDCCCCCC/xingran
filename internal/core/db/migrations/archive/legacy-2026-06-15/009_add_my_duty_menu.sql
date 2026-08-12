-- ============================================
-- 009_add_my_duty_menu.sql
-- 说明: 添加"我的值班"菜单
-- ============================================

-- 步骤1: 创建"我的值班"菜单
-- 使用 WHERE NOT EXISTS 避免重复插入
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
    icon,
    remark,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '我的值班',
    id,
    5,
    'duty/my-duty',
    NULL,
    'C',
    1,
    0,
    'CalendarOutlined',
    '查看个人值班记录和统计',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '我的值班'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤2: 为所有启用角色分配菜单权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.status = 0
    AND m.menu_name = '我的值班'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- ============================================
-- 验证迁移结果
-- ============================================

-- 查看新创建的菜单
SELECT
    id,
    menu_name,
    parent_id,
    order_num,
    path,
    menu_type,
    visible,
    status,
    icon,
    remark
FROM sys_menu
WHERE menu_name = '我的值班';

-- 验证角色菜单关联
SELECT
    r.id as role_id,
    r.role_name,
    r.role_key,
    m.id as menu_id,
    m.menu_name
FROM sys_role_menu rm
JOIN sys_role r ON rm.role_id = r.id
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = '我的值班';
