-- ============================================
-- 修复运维管理子菜单的component路径（添加前导斜杠）
-- ============================================
-- 问题：运维管理子菜单的component路径缺少前导斜杠 /
--      导致resolveComponent函数无法正确处理
-- 解决：为所有运维管理子菜单的component添加前导斜杠
-- 日期：2026-01-26
-- ============================================

-- 1. 修复楼宇管理（path保持相对，component添加前导斜杠）
UPDATE sys_menu
SET path = 'buildings',
    component = '/operations/buildings/index'
WHERE menu_name = '楼宇管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 2. 修复楼层管理
UPDATE sys_menu
SET path = 'floors',
    component = '/operations/floors/index'
WHERE menu_name = '楼层管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 3. 修复工位管理
UPDATE sys_menu
SET path = 'workstations',
    component = '/operations/workstations/index'
WHERE menu_name = '工位管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 4. 修复机房管理
UPDATE sys_menu
SET path = 'server-rooms',
    component = '/operations/server-rooms/index'
WHERE menu_name = '机房管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 5. 修复专线管理
UPDATE sys_menu
SET path = 'dedicated-lines',
    component = '/operations/dedicated-lines/index'
WHERE menu_name = '专线管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 6. 修复机房设备管理
UPDATE sys_menu
SET path = 'room-devices',
    component = '/operations/room-devices/index'
WHERE menu_name = '机房设备管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 7. 修复楼宇空间（包括修复parent_id）
UPDATE sys_menu
SET path = 'building-spaces',
    component = '/operations/building-spaces/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间';

-- 8. 修复楼宇空间3D（包括修复parent_id）
UPDATE sys_menu
SET path = 'building-spaces-3d',
    component = '/operations/building-spaces-3d/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间3D';

-- 9. 修复信息点管理（如果存在）
UPDATE sys_menu
SET path = 'info-points',
    component = '/operations/info-points/index'
WHERE menu_name = '信息点管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- ============================================
-- 验证修复结果
-- ============================================
SELECT
    id,
    menu_name,
    parent_id,
    path,
    component,
    menu_type
FROM sys_menu
WHERE parent_id IN (
    SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
)
ORDER BY order_num;
