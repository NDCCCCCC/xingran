-- Phase 32 P2-A4 source-tracking:
--   Original commit: 41d1dd57
--   Created: 2026-01-09
--   Note: No filename conflict — listed for completeness. Runner uses Go code ordering.

-- ============================================
-- 值班管理子菜单合并迁移
-- 迁移版本: 028
-- 描述: 将排班管理、节假日管理、值班配置合并为统一的"值班管理"页面
-- 说明: 创建新的值班管理菜单，隐藏旧菜单，保留值班池管理和我的值班
-- ============================================

-- ================================
-- 1. 创建新的"值班管理"统一菜单
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
    '值班管理',
    id,
    2,
    'duty/management',
    'duty/management/index',
    'C',
    1,
    0,
    'ops:duty:management:view',
    'ControlOutlined',
    '值班管理统一页面（排班+节假日+配置）',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '运维管理' AND parent_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班管理'
            AND path = 'duty/management'
            AND parent_id IN (
                SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
            )
    )
LIMIT 1;

-- ================================
-- 2. 值班管理按钮权限
-- ================================

-- 排班查询
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '排班查询' AND parent_id = m.id
    )
LIMIT 1;

-- 生成排班
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '生成排班' AND parent_id = m.id
    )
LIMIT 1;

-- 调班操作
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '调班操作' AND parent_id = m.id
    )
LIMIT 1;

-- 删除排班
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '删除排班' AND parent_id = m.id
    )
LIMIT 1;

-- 节假日查询
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日查询',
    m.id,
    5,
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日查询' AND parent_id = m.id
    )
LIMIT 1;

-- 节假日新增
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日新增',
    m.id,
    6,
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日新增' AND parent_id = m.id
    )
LIMIT 1;

-- 节假日修改
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日修改',
    m.id,
    7,
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日修改' AND parent_id = m.id
    )
LIMIT 1;

-- 节假日删除
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '节假日删除',
    m.id,
    8,
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '节假日删除' AND parent_id = m.id
    )
LIMIT 1;

-- 值班配置修改
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '值班配置修改',
    m.id,
    9,
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
WHERE m.menu_name = '值班管理' AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_menu
        WHERE menu_name = '值班配置修改' AND parent_id = m.id
    )
LIMIT 1;

-- ================================
-- 3. 隐藏旧的值班子菜单
-- ================================

-- 隐藏排班管理
UPDATE sys_menu
SET visible = 0
WHERE menu_name = '排班管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 隐藏节假日管理
UPDATE sys_menu
SET visible = 0
WHERE menu_name = '节假日管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 隐藏值班配置
UPDATE sys_menu
SET visible = 0
WHERE menu_name = '值班配置'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- ================================
-- 4. 调整值班池管理和我的值班的排序
-- ================================

-- 值班池管理排序改为 1
UPDATE sys_menu
SET order_num = 1
WHERE menu_name = '值班池管理'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 我的值班排序改为 3
UPDATE sys_menu
SET order_num = 3
WHERE menu_name = '我的值班'
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- ================================
-- 5. 为所有启用角色分配新菜单权限
-- ================================
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.status = 0
    AND m.menu_name = '值班管理'
    AND m.path = 'duty/management'
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- 同时分配新菜单的按钮权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.status = 0
    AND m.parent_id IN (
        SELECT id FROM sys_menu WHERE menu_name = '值班管理' AND path = 'duty/management'
    )
    AND NOT EXISTS (
        SELECT 1 FROM sys_role_menu rm
        WHERE rm.role_id = r.id AND rm.menu_id = m.id
    );

-- ================================
-- 6. 验证迁移结果
-- ================================

-- 查看新的值班管理菜单结构
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
WHERE (menu_name = '值班管理' AND path = 'duty/management')
   OR parent_id IN (
       SELECT id FROM sys_menu WHERE menu_name = '值班管理' AND path = 'duty/management'
   )
ORDER BY order_num;

-- 查看旧菜单的可见状态
SELECT
    id,
    menu_name,
    order_num,
    visible,
    status
FROM sys_menu
WHERE menu_name IN ('排班管理', '节假日管理', '值班配置', '值班池管理', '我的值班')
    AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
ORDER BY order_num;

-- ================================
-- 迁移完成
-- ================================
SELECT '028_merge_duty_menus.sql - 值班管理菜单合并完成' AS status;
