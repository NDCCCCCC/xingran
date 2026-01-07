-- =============================================
-- 值班管理菜单统一迁移
-- 迁移版本: 018
-- 描述: 统一创建所有值班管理菜单，替代 009 和 createDutyManagementMenus 函数
-- 说明: 此脚本包含了所有值班管理菜单的创建，使用完整的防重复检查
-- =============================================

-- ================================
-- 1. 值班池管理菜单
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
    '值班池管理',
    id,
    1,
    'duty/pools',
    'duty/pools/index',
    'C',
    1,
    0,
    'ops:duty:pools:view',
    'TeamOutlined',
    '值班池管理页面',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班池管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 值班池管理按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '值班池查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:pool:list',
    '',
    '值班池查询',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '值班池管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班池查询' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '值班池新增',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:pool:add',
    '',
    '值班池新增',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '值班池管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班池新增' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '值班池修改',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:pool:edit',
    '',
    '值班池修改',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '值班池管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班池修改' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '值班池删除',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:pool:delete',
    '',
    '值班池删除',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '值班池管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班池删除' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 2. 排班管理菜单
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
    '排班管理',
    id,
    2,
    'duty/schedules',
    'duty/schedules/index',
    'C',
    1,
    0,
    'ops:duty:schedules:view',
    'ClockCircleOutlined',
    '排班管理页面',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '排班管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 排班管理按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '排班查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:schedule:list',
    '',
    '排班查询',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '排班管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '排班查询' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '生成排班',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:schedule:add',
    '',
    '生成排班',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '排班管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '生成排班' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '调班操作',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:schedule:edit',
    '',
    '调班操作',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '排班管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '调班操作' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '删除排班',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:schedule:delete',
    '',
    '删除排班',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '排班管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '删除排班' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 3. 节假日管理菜单
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
    '节假日管理',
    id,
    3,
    'duty/holidays',
    'duty/holidays/index',
    'C',
    1,
    0,
    'ops:duty:holidays:view',
    'HomeOutlined',
    '节假日管理页面',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 节假日管理按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:holiday:list',
    '',
    '节假日查询',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '节假日管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日查询' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日新增',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:holiday:add',
    '',
    '节假日新增',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '节假日管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日新增' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日修改',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:holiday:edit',
    '',
    '节假日修改',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '节假日管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日修改' AND parent_id = m.id
    )
LIMIT 1;

INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日删除',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:holiday:delete',
    '',
    '节假日删除',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '节假日管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日删除' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 4. 值班配置菜单
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
    '值班配置',
    id,
    4,
    'duty/config',
    'duty/config/index',
    'C',
    1,
    0,
    'ops:duty:config:view',
    'SettingOutlined',
    '值班配置页面',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班配置'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 值班配置按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '值班配置修改',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:config:edit',
    '',
    '值班配置修改',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '值班配置'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班配置修改' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 5. 我的值班菜单（替代 009 迁移）
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
    '我的值班',
    id,
    5,
    'duty/my-duty',
    'duty/my-duty/index',
    'C',
    1,
    0,
    'ops:duty:my:view',
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

-- 我的值班按钮权限
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '值班查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    'ops:duty:my:list',
    '',
    '值班查询',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '我的值班'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班查询' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 6. 为所有启用角色分配菜单权限
-- ================================
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.status = 0
    AND (
        m.menu_name IN ('值班池管理', '排班管理', '节假日管理', '值班配置', '我的值班')
        OR m.menu_name IN ('值班池查询', '值班池新增', '值班池修改', '值班池删除',
                           '排班查询', '生成排班', '调班操作', '删除排班',
                           '节假日查询', '节假日新增', '节假日修改', '节假日删除',
                           '值班配置修改', '值班查询')
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- ================================
-- 验证迁移结果
-- ================================

-- 查看值班管理相关菜单
SELECT
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
    remark
FROM sys_menu
WHERE menu_name IN ('值班池管理', '排班管理', '节假日管理', '值班配置', '我的值班')
   OR parent_id IN (
       SELECT id FROM sys_menu
       WHERE menu_name IN ('值班池管理', '排班管理', '节假日管理', '值班配置', '我的值班')
   )
ORDER BY
    CASE menu_name
        WHEN '值班池管理' THEN 1
        WHEN '排班管理' THEN 2
        WHEN '节假日管理' THEN 3
        WHEN '值班配置' THEN 4
        WHEN '我的值班' THEN 5
    END,
    order_num;

-- ================================
-- 迁移完成
-- ================================
SELECT '018_unify_duty_menus.sql migration completed' AS status;
