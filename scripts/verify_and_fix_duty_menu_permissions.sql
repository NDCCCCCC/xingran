-- ============================================
-- 验证和修复"我的值班"菜单权限
-- ============================================

-- 步骤1: 检查"运维管理"父菜单是否存在
SELECT
    id,
    menu_name,
    menu_type,
    visible,
    status,
    parent_id IS NULL as is_top_level
FROM sys_menu
WHERE menu_name = '运维管理' AND menu_type = 'M';

-- 步骤2: 检查"我的值班"菜单是否存在
SELECT
    id,
    menu_name,
    menu_type,
    visible,
    status,
    path,
    perms,
    parent_id
FROM sys_menu
WHERE menu_name = '我的值班';

-- 步骤3: 检查用户角色及其状态
SELECT
    id,
    role_name,
    role_key,
    status
FROM sys_role
WHERE status = 0; -- 0表示启用

-- 步骤4: 检查"运维管理"菜单的角色权限分配情况
SELECT
    r.id as role_id,
    r.role_name,
    m.id as menu_id,
    m.menu_name
FROM sys_role r
CROSS JOIN sys_menu m
WHERE m.menu_name = '运维管理'
  AND r.status = 0
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.role_id = r.id AND rm.menu_id = m.id
);

-- 步骤5: 检查"我的值班"菜单的角色权限分配情况（缺失权限的）
SELECT
    r.id as role_id,
    r.role_name,
    m.id as menu_id,
    m.menu_name
FROM sys_role r
CROSS JOIN sys_menu m
WHERE m.menu_name = '我的值班'
  AND r.status = 0
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.role_id = r.id AND rm.menu_id = m.id
  );

-- 步骤6: 修复缺失的权限 - 为所有启用角色分配"运维管理"菜单权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE m.menu_name = '运维管理'
  AND r.status = 0
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.role_id = r.id AND rm.menu_id = m.id
  );

-- 步骤7: 修复缺失的权限 - 为所有启用角色分配"我的值班"菜单权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT
    r.id,
    m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE m.menu_name = '我的值班'
  AND r.status = 0
  AND NOT EXISTS (
    SELECT 1 FROM sys_role_menu rm
    WHERE rm.role_id = r.id AND rm.menu_id = m.id
  );

-- 步骤8: 验证修复结果 - 显示运维管理相关的所有角色菜单关联
SELECT
    r.role_name,
    p.menu_name as parent_menu,
    m.menu_name as child_menu,
    m.menu_type,
    m.visible,
    m.status
FROM sys_role_menu rm
JOIN sys_role r ON rm.role_id = r.id
JOIN sys_menu m ON rm.menu_id = m.id
LEFT JOIN sys_menu p ON m.parent_id = p.id
WHERE p.menu_name = '运维管理' OR m.menu_name = '运维管理'
ORDER BY r.role_name, p.menu_name, m.order_num;

-- 说明：
-- - 步骤1-2: 检查菜单是否存在
-- - 步骤3: 检查有哪些启用的角色
-- - 步骤4-5: 查找缺失权限的角色和菜单组合
-- - 步骤6-7: 自动修复缺失的权限
-- - 步骤8: 验证修复结果
