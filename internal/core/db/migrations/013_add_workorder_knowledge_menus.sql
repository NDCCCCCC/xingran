-- ============================================
-- 013_add_workorder_knowledge_menus.sql
-- 说明: 添加运维工单和知识库管理菜单
--       所有子菜单直接放在运维管理下
-- ============================================

-- ============================================
-- 第一部分：运维工单管理菜单（直接放在运维管理下）
-- ============================================

-- 步骤1: 创建"工单管理"子菜单
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
    '工单管理',
    id,
    1,
    'orders',
    '/workorder/orders',
    'C',
    1,
    0,
    'FormOutlined',
    '工单列表、创建、分配和处理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单管理'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤2: 创建"工单分类"子菜单
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
    '工单分类',
    id,
    2,
    'categories',
    '/workorder/categories',
    'C',
    1,
    0,
    'FolderOutlined',
    '工单分类管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单分类'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤3: 创建"工单统计"子菜单
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
    '工单统计',
    id,
    3,
    'statistics',
    '/workorder/statistics',
    'C',
    1,
    0,
    'FundOutlined',
    '工单数据统计分析',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单统计'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤4: 创建"周期性工单"子菜单
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
    '周期性工单',
    id,
    4,
    'periodic/templates',
    '/workorder/periodic/templates',
    'C',
    1,
    0,
    'ClockCircleOutlined',
    '周期性工单模板管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '周期性工单'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤5: 为工单管理添加按钮权限
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
    '工单查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '工单查询权限',
    'ops:workorder:list',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单查询' AND parent_id = m.id
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
    '工单新增',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '工单新增权限',
    'ops:workorder:add',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单新增' AND parent_id = m.id
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
    '工单编辑',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '工单编辑权限',
    'ops:workorder:edit',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单编辑' AND parent_id = m.id
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
    '工单删除',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '工单删除权限',
    'ops:workorder:delete',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单删除' AND parent_id = m.id
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
    '工单分配',
    m.id,
    5,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '工单分配权限',
    'ops:workorder:assign',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单管理'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单分配' AND parent_id = m.id
    )
LIMIT 1;

-- 步骤6: 为工单分类添加按钮权限
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
    '分类查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '分类查询权限',
    'ops:workorder:category:list',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单分类'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '分类查询' AND parent_id = m.id
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
    '分类新增',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '分类新增权限',
    'ops:workorder:category:add',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单分类'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '分类新增' AND parent_id = m.id
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
    '分类编辑',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '分类编辑权限',
    'ops:workorder:category:edit',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单分类'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '分类编辑' AND parent_id = m.id
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
    '分类删除',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '分类删除权限',
    'ops:workorder:category:delete',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '工单分类'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '分类删除' AND parent_id = m.id
    )
LIMIT 1;

-- 步骤7: 为周期性工单添加按钮权限
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
    '周期工单查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '周期工单查询权限',
    'ops:workorder:periodic:list',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '周期性工单'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '周期工单查询' AND parent_id = m.id
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
    '周期工单新增',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '周期工单新增权限',
    'ops:workorder:periodic:add',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '周期性工单'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '周期工单新增' AND parent_id = m.id
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
    '周期工单编辑',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '周期工单编辑权限',
    'ops:workorder:periodic:edit',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '周期性工单'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '周期工单编辑' AND parent_id = m.id
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
    '周期工单删除',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '周期工单删除权限',
    'ops:workorder:periodic:delete',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '周期性工单'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '周期工单删除' AND parent_id = m.id
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
    '周期工单启用',
    m.id,
    5,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '周期工单启用/禁用权限',
    'ops:workorder:periodic:enable',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '周期性工单'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '周期工单启用' AND parent_id = m.id
    )
LIMIT 1;

-- ============================================
-- 第二部分：知识库管理菜单（直接放在运维管理下）
-- ============================================

-- 步骤8: 创建"知识库文章"子菜单
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
    '知识库文章',
    id,
    5,
    'articles',
    '/knowledge/articles',
    'C',
    1,
    0,
    'FileTextOutlined',
    '知识库文章管理',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '知识库文章'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤9: 创建"知识库查看"子菜单（所有用户可见）
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
    '知识库查看',
    id,
    6,
    'view',
    '/knowledge/view',
    'C',
    1,
    0,
    'EyeOutlined',
    '知识库文章查看（所有用户）',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '知识库查看'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- 步骤10: 为知识库文章添加按钮权限
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
    '文章查询',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '文章查询权限',
    'ops:knowledge:article:list',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '知识库文章'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '文章查询' AND parent_id = m.id
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
    '文章新增',
    m.id,
    2,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '文章新增权限',
    'ops:knowledge:article:add',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '知识库文章'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '文章新增' AND parent_id = m.id
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
    '文章编辑',
    m.id,
    3,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '文章编辑权限',
    'ops:knowledge:article:edit',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '知识库文章'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '文章编辑' AND parent_id = m.id
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
    '文章删除',
    m.id,
    4,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '文章删除权限',
    'ops:knowledge:article:delete',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '知识库文章'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '文章删除' AND parent_id = m.id
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
    '工单转知识库',
    m.id,
    5,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '工单转知识库权限',
    'ops:knowledge:convert',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '知识库文章'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '工单转知识库' AND parent_id = m.id
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
    '文章发布',
    m.id,
    6,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '文章发布权限',
    'ops:knowledge:article:publish',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '知识库文章'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '文章发布' AND parent_id = m.id
    )
LIMIT 1;

-- 步骤11: 为知识库查看添加权限（所有用户）
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
    '文章查看',
    m.id,
    1,
    '',
    NULL,
    'F',
    1,
    0,
    '',
    '文章查看权限（所有用户）',
    'ops:knowledge:article:view',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu m
WHERE m.menu_name = '知识库查看'
    AND m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '文章查看' AND parent_id = m.id
    )
LIMIT 1;

-- ============================================
-- 第三部分：为管理员角色分配菜单权限
-- ============================================

-- 获取管理员角色ID并分配所有新菜单
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.role_key = 'admin'
    AND (
        m.menu_name IN ('工单管理', '工单分类', '工单统计', '周期性工单',
                        '知识库文章', '知识库查看')
        OR m.menu_name IN ('工单查询', '工单新增', '工单编辑', '工单删除', '工单分配',
                           '分类查询', '分类新增', '分类编辑', '分类删除',
                           '周期工单查询', '周期工单新增', '周期工单编辑', '周期工单删除', '周期工单启用',
                           '文章查询', '文章新增', '文章编辑', '文章删除', '工单转知识库', '文章发布', '文章查看')
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
    AND m.menu_name IN ('工单管理', '工单分类', '工单统计', '周期性工单', '知识库文章', '知识库查看')
ORDER BY m.order_num;

-- ============================================
-- 迁移完成
-- ============================================

SELECT '013_add_workorder_knowledge_menus.sql migration completed' AS status;
