-- =============================================
-- AD Group Mapping Menu - Menu Data Migration
-- Migration version: 136
-- Description: Add Department-Group Mapping menu and permissions under AD domain management
-- =============================================

-- ================================
-- 1. Add "部门-组映射" menu item under "AD域管理"
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
    '部门-组映射',
    m.id,
    6,
    'group-mapping',
    'ad-domain/group-mapping/index',
    'C',
    1,
    0,
    'ops:ad:group:mapping:view',
    'PartitionOutlined',
    '部门与AD组映射管理页面',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'AD域管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '部门-组映射'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
                    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
            )
    )
LIMIT 1;

-- ================================
-- 2. Add button permissions under "部门-组映射"
-- ================================

-- Add mapping button
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
    '映射添加',
    m.id,
    1,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:group:mapping:add',
    '#',
    '添加部门组映射',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '部门-组映射'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE perms = 'ops:ad:group:mapping:add'
    )
LIMIT 1;

-- Edit mapping button
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
    '映射修改',
    m.id,
    2,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:group:mapping:edit',
    '#',
    '修改部门组映射',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '部门-组映射'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE perms = 'ops:ad:group:mapping:edit'
    )
LIMIT 1;

-- Delete mapping button
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
    '映射删除',
    m.id,
    3,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:group:mapping:delete',
    '#',
    '删除部门组映射',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '部门-组映射'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE perms = 'ops:ad:group:mapping:delete'
    )
LIMIT 1;

-- Auto-map button
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
    '自动映射',
    m.id,
    4,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:group:mapping:automap',
    '#',
    '自动映射部门到AD组',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '部门-组映射'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE perms = 'ops:ad:group:mapping:automap'
    )
LIMIT 1;

-- Sync members button
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
    '成员同步',
    m.id,
    5,
    '',
    NULL,
    'F',
    0,
    0,
    'ops:ad:group:mapping:sync',
    '#',
    '同步部门成员到AD组',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '部门-组映射'
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = 'AD域管理'
            AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE perms = 'ops:ad:group:mapping:sync'
    )
LIMIT 1;

-- ================================
-- 3. Assign permissions to admin role (role_id with status=0)
-- ================================
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.status = 0
    AND (
        m.menu_name = '部门-组映射'
        OR m.perms IN (
            'ops:ad:group:mapping:view',
            'ops:ad:group:mapping:add',
            'ops:ad:group:mapping:edit',
            'ops:ad:group:mapping:delete',
            'ops:ad:group:mapping:automap',
            'ops:ad:group:mapping:sync'
        )
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- ================================
-- Verification
-- ================================
SELECT '136_add_group_mapping_menu.sql migration completed' AS status;
