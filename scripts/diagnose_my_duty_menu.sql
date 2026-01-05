-- ============================================
-- 深度诊断"我的值班"菜单问题
-- ============================================

-- ========== 第一部分：检查菜单创建情况 ==========

-- 1. 检查"运维管理"父菜单
SELECT '=== 运维管理父菜单 ===' as section;
SELECT
    id,
    menu_name,
    menu_type,
    visible,
    status,
    path,
    icon
FROM sys_menu
WHERE menu_name = '运维管理' AND menu_type = 'M';

-- 2. 检查"运维管理"下的所有子菜单
SELECT '=== 运维管理下的所有子菜单 ===' as section;
SELECT
    id,
    menu_name,
    menu_type,
    visible,
    status,
    path,
    perms,
    order_num
FROM sys_menu
WHERE parent_id IN (
    SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND menu_type = 'M'
)
ORDER BY order_num;

-- 3. 特别检查"我的值班"菜单
SELECT '=== 我的值班菜单详细信息 ===' as section;
SELECT
    m.id,
    m.menu_name,
    m.menu_type,
    m.visible,
    m.status,
    m.path,
    m.component,
    m.perms,
    m.parent_id,
    p.menu_name as parent_menu_name,
    p.menu_type as parent_menu_type,
    p.visible as parent_visible,
    p.status as parent_status
FROM sys_menu m
LEFT JOIN sys_menu p ON m.parent_id = p.id
WHERE m.menu_name = '我的值班';

-- ========== 第二部分：检查角色权限分配 ==========

-- 4. 检查所有启用角色
SELECT '=== 所有启用的角色 ===' as section;
SELECT id, role_name, role_key, status FROM sys_role WHERE status = 0;

-- 5. 检查"运维管理"的角色权限分配
SELECT '=== 运维管理菜单的角色权限 ===' as section;
SELECT
    r.id as role_id,
    r.role_name,
    r.role_key,
    r.status as role_status,
    m.id as menu_id,
    m.menu_name
FROM sys_role r
CROSS JOIN sys_menu m
WHERE m.menu_name = '运维管理' AND m.menu_type = 'M'
ORDER BY r.role_name;

-- 6. 检查"运维管理"的实际角色权限关联
SELECT '=== 运维管理菜单的实际角色关联（有权限） ===' as section;
SELECT
    r.id as role_id,
    r.role_name,
    m.id as menu_id,
    m.menu_name
FROM sys_role_menu rm
JOIN sys_role r ON rm.role_id = r.id
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = '运维管理' AND m.menu_type = 'M'
ORDER BY r.role_name;

-- 7. 检查"我的值班"的角色权限分配
SELECT '=== 我的值班菜单的角色权限 ===' as section;
SELECT
    r.id as role_id,
    r.role_name,
    r.status as role_status,
    m.id as menu_id,
    m.menu_name,
    m.visible,
    m.menu_type
FROM sys_role r
CROSS JOIN sys_menu m
WHERE m.menu_name = '我的值班'
ORDER BY r.role_name;

-- 8. 检查"我的值班"的实际角色权限关联
SELECT '=== 我的值班菜单的实际角色关联（有权限） ===' as section;
SELECT
    r.id as role_id,
    r.role_name,
    m.id as menu_id,
    m.menu_name,
    m.visible,
    m.menu_type
FROM sys_role_menu rm
JOIN sys_role r ON rm.role_id = r.id
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = '我的值班'
ORDER BY r.role_name;

-- ========== 第三部分：检查用户角色关联 ==========

-- 9. 检查用户角色关联（假设有一个admin用户）
SELECT '=== 用户角色关联（前5个用户） ===' as section;
SELECT
    u.id as user_id,
    u.username,
    u.nick_name,
    u.status as user_status,
    r.id as role_id,
    r.role_name,
    r.role_key,
    r.status as role_status
FROM sys_user_role ur
JOIN sys_user u ON ur.user_id = u.id
JOIN sys_role r ON ur.role_id = r.id
ORDER BY u.username
LIMIT 5;

-- ========== 第四部分：模拟GetUserMenus查询 ==========

-- 10. 模拟GetUserMenus的SQL查询（以admin用户为例）
SELECT '=== 模拟GetUserMenus查询（admin用户） ===' as section;
SELECT DISTINCT m.*
FROM sys_menu m
INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
INNER JOIN sys_user_role ur ON rm.role_id = ur.role_id
WHERE ur.user_id = (SELECT id FROM sys_user WHERE username = 'admin' LIMIT 1)
AND m.visible = 1
AND m.menu_type IN ('M', 'C')
ORDER BY m.order_num;

-- ========== 第五部分：找出缺失的权限 ==========

-- 11. 找出运维管理相关菜单中，缺失角色权限的
SELECT '=== 缺失角色权限的运维管理相关菜单 ===' as section;
SELECT
    m.id as menu_id,
    m.menu_name,
    m.menu_type,
    m.visible,
    m.status,
    m.order_num,
    m.path,
    '缺少角色权限' as issue
FROM sys_menu m
WHERE m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND menu_type = 'M')
   OR m.id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND menu_type = 'M')
AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.menu_id = m.id
)
ORDER BY m.order_num;

-- 12. 找出有权限但角色被禁用的菜单
SELECT '=== 有权限但角色被禁用的菜单 ===' as section;
SELECT
    m.id as menu_id,
    m.menu_name,
    r.id as role_id,
    r.role_name,
    r.status as role_status,
    '角色被禁用' as issue
FROM sys_role_menu rm
JOIN sys_menu m ON rm.menu_id = m.id
JOIN sys_role r ON rm.role_id = r.id
WHERE (m.parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND menu_type = 'M')
   OR m.id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND menu_type = 'M'))
AND r.status != 0;
