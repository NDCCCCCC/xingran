-- ============================================
-- 106_add_rpa_menus.sql
-- 说明: 添加 RPA 机器人管理菜单
--       放在运维管理下
-- ============================================

-- 步骤1: 创建"RPA 管理"子菜单
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
    'RPA 管理',
    id,
    7,
    'rpa',
    'pages/operations/rpa/index',
    'C',
    1,
    0,
    'RobotOutlined',
    'RPA 机器人自动化任务管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = 'RPA 管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤2: 为 RPA 管理添加按钮权限
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
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '任务查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA 任务查询权限',
    'ops:rpa:task:list',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '任务查询' AND parent_id = m.id
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
    icon,
    remark,
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '任务新增',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA 任务新增权限',
    'ops:rpa:task:add',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '任务新增' AND parent_id = m.id
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
    icon,
    remark,
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '任务编辑',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA 任务编辑权限',
    'ops:rpa:task:edit',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '任务编辑' AND parent_id = m.id
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
    icon,
    remark,
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '任务删除',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA 任务删除权限',
    'ops:rpa:task:delete',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '任务删除' AND parent_id = m.id
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
    icon,
    remark,
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '任务执行',
    m.id,
    5,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA 任务执行权限',
    'ops:rpa:task:execute',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '任务执行' AND parent_id = m.id
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
    icon,
    remark,
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '执行记录查询',
    m.id,
    6,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA 执行记录查询权限',
    'ops:rpa:execution:list',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '执行记录查询' AND parent_id = m.id
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
    icon,
    remark,
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    '执行取消',
    m.id,
    7,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA 执行取消权限',
    'ops:rpa:execution:cancel',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '执行取消' AND parent_id = m.id
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
    icon,
    remark,
    perms,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    'Worker 监控',
    m.id,
    8,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    'RPA Worker 监控权限',
    'ops:rpa:worker:monitor',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = 'RPA 管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = 'Worker 监控' AND parent_id = m.id
    )
LIMIT 1;

-- 步骤3: 为管理员角色分配菜单权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.role_key = 'admin'
    AND (
        m.menu_name = 'RPA 管理'
        OR m.menu_name IN ('任务查询', '任务新增', '任务编辑', '任务删除', '任务执行',
                           '执行记录查询', '执行取消', 'Worker 监控')
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- ============================================
-- 验证迁移结果
-- ============================================

-- 查看运维管理下的所有子菜单
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
WHERE parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
ORDER BY order_num;

-- 验证角色菜单关联（管理员）
SELECT
    r.id as role_id,
    r.role_name,
    r.role_key,
    m.id as menu_id,
    m.menu_name,
    m.menu_type
FROM sys_role_menu rm
JOIN sys_role r ON rm.role_id = r.id
JOIN sys_menu m ON rm.menu_id = m.id
WHERE r.role_key = 'admin'
    AND m.menu_name = 'RPA 管理'
ORDER BY m.order_num;

-- ============================================
-- 迁移完成
-- ============================================

SELECT '106_add_rpa_menus.sql migration completed' AS status;
