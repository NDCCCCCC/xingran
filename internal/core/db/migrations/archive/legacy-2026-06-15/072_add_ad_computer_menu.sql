-- =============================================
-- AD域电脑设备管理 - 菜单数据迁移
-- 迁移版本: 072
-- 描述: 添加AD域电脑设备管理相关菜单到sys_menu表
-- =============================================

-- ================================
-- 1. 电脑设备管理菜单
-- ================================
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
)
SELECT
    gen_random_uuid(),
    '电脑设备管理',
    id,
    5,
    'computers',
    '/ad-domain/computers',
    'C',
    0,
    0,
    'ops:ad:computer:view',
    'DesktopOutlined',
    'AD域电脑设备管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = 'AD域管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '电脑设备管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
                    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
            )
    )
LIMIT 1;

-- ================================
-- 2. 为所有启用角色分配电脑设备管理菜单权限
-- ================================
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.status = 0
    AND m.menu_name = '电脑设备管理'
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- ================================
-- 验证迁移结果
-- ================================

SELECT
    '072_add_ad_computer_menu.sql migration completed' AS status;
