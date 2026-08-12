-- ============================================
-- 修复运维管理下所有子菜单：使用绝对路径以移除ops前缀
-- ============================================
-- 原理：
--   - path包含'/'的将被视为绝对路径，不与父路径拼接
--   - 工单管理应该路由到 /workorder/orders 而不是 /ops/orders
--   - 知识库应该路由到 /knowledge/articles 而不是 /ops/articles
--   - 楼宇管理应该路由到 /operations/buildings 而不是 /ops/buildings
-- 日期：2026-01-26
-- ============================================

-- 获取运维管理父菜单ID
WITH ops_parent AS (
    SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
),

-- 1. 修复工单管理子菜单（使用绝对路径，路由为 /workorder/xxx）
workorder_fixes AS (
    UPDATE sys_menu
    SET path = 'workorder/orders',
        component = '/workorder/orders/index'
    WHERE menu_name = '工单管理'
      AND parent_id IN (SELECT id FROM ops_parent)

    RETURNING 1
)
UPDATE sys_menu
SET path = 'workorder/categories',
    component = '/workorder/categories/index'
WHERE menu_name = '工单分类'
  AND parent_id IN (SELECT id FROM ops_parent);

UPDATE sys_menu
SET path = 'workorder/statistics',
    component = '/workorder/statistics/index'
WHERE menu_name = '工单统计'
  AND parent_id IN (SELECT id FROM ops_parent);

UPDATE sys_menu
SET path = 'workorder/periodic/templates',
    component = '/workorder/periodic/templates/index'
WHERE menu_name = '周期性工单'
  AND parent_id IN (SELECT id FROM ops_parent);

-- 2. 修复知识库子菜单（使用绝对路径，路由为 /knowledge/xxx）
UPDATE sys_menu
SET path = 'knowledge/articles',
    component = '/knowledge/articles/index'
WHERE menu_name = '知识库文章'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'knowledge/view',
    component = '/knowledge/view/index'
WHERE menu_name = '知识库查看'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 3. 修复楼宇空间子菜单（使用绝对路径，路由为 /operations/xxx）
UPDATE sys_menu
SET path = 'operations/building-spaces',
    component = '/operations/building-spaces/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间';

UPDATE sys_menu
SET path = 'operations/building-spaces-3d',
    component = '/operations/building-spaces-3d/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间3D';

-- 4. 修复其他运维子菜单（使用绝对路径，路由为 /operations/xxx）
UPDATE sys_menu
SET path = 'operations/buildings',
    component = '/operations/buildings/index'
WHERE menu_name = '楼宇管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'operations/floors',
    component = '/operations/floors/index'
WHERE menu_name = '楼层管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'operations/workstations',
    component = '/operations/workstations/index'
WHERE menu_name = '工位管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'operations/server-rooms',
    component = '/operations/server-rooms/index'
WHERE menu_name = '机房管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'operations/dedicated-lines',
    component = '/operations/dedicated-lines/index'
WHERE menu_name = '专线管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'operations/room-devices',
    component = '/operations/room-devices/index'
WHERE menu_name = '机房设备管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'operations/info-points',
    component = '/operations/info-points/index'
WHERE menu_name = '信息点管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- ============================================
-- 验证修复结果
-- ============================================
SELECT
    menu_name,
    path,
    component,
    CASE
        WHEN path LIKE '%/%' THEN '绝对路径（不拼接ops）'
        ELSE '相对路径（会拼接ops）'
    END as path_type
FROM sys_menu
WHERE parent_id IN (
    SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
)
ORDER BY order_num;
