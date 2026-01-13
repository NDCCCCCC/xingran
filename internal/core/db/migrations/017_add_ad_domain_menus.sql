-- =============================================
-- AD域控管理功能 - 菜单数据迁移
-- 迁移版本: 017
-- 描述: 添加AD域管理相关菜单到sys_menu表
-- =============================================

-- ================================
-- 1. AD域管理（父菜单）
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
    'AD域管理',
    id,
    8,
    'ad-domain',
    NULL,
    'M',
    0,
    0,
    '',
    'ApiOutlined',
    'AD域控管理功能',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = 'AD域管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- ================================
-- 2. AD配置管理菜单
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
    'AD配置管理',
    id,
    1,
    'configs',
    '/ad-domain/configs',
    'C',
    0,
    0,
    'ops:ad:config:list',
    'SettingOutlined',
    'AD域连接配置管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = 'AD域管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = 'AD配置管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
                    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
            )
    )
LIMIT 1;

-- ================================
-- 3. AD配置管理按钮权限
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
    '新增AD配置',
    m.id,
    1,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:config:add',
    '',
    '新增AD配置权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'AD配置管理'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '新增AD配置' AND parent_id = m.id
    )
LIMIT 1;

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
    '编辑AD配置',
    m.id,
    2,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:config:edit',
    '',
    '编辑AD配置权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'AD配置管理'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '编辑AD配置' AND parent_id = m.id
    )
LIMIT 1;

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
    '删除AD配置',
    m.id,
    3,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:config:delete',
    '',
    '删除AD配置权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'AD配置管理'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '删除AD配置' AND parent_id = m.id
    )
LIMIT 1;

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
    '测试连接',
    m.id,
    4,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:config:test',
    '',
    '测试AD连接权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'AD配置管理'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '测试连接' AND parent_id = m.id
    )
LIMIT 1;

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
    '同步数据',
    m.id,
    5,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:config:sync',
    '',
    '同步AD数据权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'AD配置管理'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '同步数据' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 4. OU组织单位管理菜单
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
    'OU组织单位',
    id,
    2,
    'ous',
    '/ad-domain/ous',
    'C',
    0,
    0,
    'ops:ad:ou:view',
    'ApartmentOutlined',
    'AD域OU组织单位管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = 'AD域管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = 'OU组织单位'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
                    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
            )
    )
LIMIT 1;

-- ================================
-- 5. 用户组管理菜单
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
    '用户组管理',
    id,
    3,
    'groups',
    '/ad-domain/groups',
    'C',
    0,
    0,
    'ops:ad:group:view',
    'TeamOutlined',
    'AD域用户组管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = 'AD域管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '用户组管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
                    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
            )
    )
LIMIT 1;

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
    '编辑用户组',
    m.id,
    1,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:group:edit',
    '',
    '编辑用户组权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '用户组管理'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '编辑用户组' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 6. AD用户管理菜单
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
    'AD用户管理',
    id,
    4,
    'users',
    '/ad-domain/users',
    'C',
    0,
    0,
    'ops:ad:user:view',
    'UserOutlined',
    'AD域用户管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = 'AD域管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = 'AD用户管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
                    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
            )
    )
LIMIT 1;

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
    '编辑用户',
    m.id,
    1,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:user:edit',
    '',
    '编辑用户权限',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'AD用户管理'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '编辑用户' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 7. 同步日志菜单
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
    '同步日志',
    id,
    5,
    'logs',
    '/ad-domain/logs',
    'C',
    0,
    0,
    'ops:ad:log:view',
    'FileTextOutlined',
    'AD域同步日志',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = 'AD域管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '同步日志'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
                    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
            )
    )
LIMIT 1;

-- ================================
-- 8. 为所有启用角色分配AD域管理菜单权限
-- ================================
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.status = 0
    AND (
        m.menu_name IN ('AD域管理', 'AD配置管理', 'OU组织单位', '用户组管理', 'AD用户管理', '同步日志')
        OR m.menu_name IN ('新增AD配置', '编辑AD配置', '删除AD配置', '测试连接', '同步数据',
                           '编辑用户组', '编辑用户')
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- ================================
-- 验证迁移结果
-- ================================

-- 查看运维管理下的所有子菜单
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
WHERE parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
ORDER BY order_num;

-- ================================
-- 迁移完成
-- ================================
SELECT '017_add_ad_domain_menus.sql migration completed' AS status;
