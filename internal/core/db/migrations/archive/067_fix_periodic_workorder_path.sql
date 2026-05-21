-- ============================================
-- 修复周期性工单路径重复问题
-- ============================================
-- 问题：周期性工单路由是 /ops/workorder/workorder/periodic/templates（重复）
-- 原因：周期性工单的 path 是 'workorder/periodic/templates'，如果工单管理是目录类型（M）
--      则会与父路径 workorder/orders 拼接，产生重复
-- 解决：将所有工单和知识库子菜单的路径简化，避免重复
-- 日期：2026-01-26
-- ============================================

-- 首先检查当前菜单结构
-- （运行此查询可以看到 parent_id 和 menu_type）
SELECT
    m1.menu_name as 父菜单,
    m1.menu_type as 父类型,
    m2.menu_name as 子菜单,
    m2.path as 子路径,
    m2.parent_id as 子父ID,
    CASE m1.menu_type
        WHEN 'M' THEN '目录（会拼接路径）'
        WHEN 'C' THEN '菜单（不会拼接路径）'
        ELSE '其他'
    END as 父菜单类型说明
FROM sys_menu m1
INNER JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.menu_name IN ('运维管理', '工单管理')
ORDER BY m1.menu_name, m2.order_num;

-- ============================================
-- 修复方案：统一使用简化的路径，不包含模块名
-- ============================================

-- 1. 工单管理：使用简化的路径 'orders'
UPDATE sys_menu
SET path = 'orders',
    component = '/workorder/orders/index'
WHERE menu_name = '工单管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 2. 工单分类：使用简化的路径 'categories'
UPDATE sys_menu
SET path = 'categories',
    component = '/workorder/categories/index'
WHERE menu_name = '工单分类'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 3. 工单统计：使用简化的路径 'statistics'
UPDATE sys_menu
SET path = 'statistics',
    component = '/workorder/statistics/index'
WHERE menu_name = '工单统计'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 4. 周期性工单：使用简化的路径 'periodic'
-- （而不是 'workorder/periodic/templates' 或 'periodic/templates'）
UPDATE sys_menu
SET path = 'periodic',
    component = '/workorder/periodic/templates/index'
WHERE menu_name = '周期性工单'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 5. 知识库文章：使用简化的路径 'articles'
UPDATE sys_menu
SET path = 'articles',
    component = '/knowledge/articles/index'
WHERE menu_name = '知识库文章'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 6. 知识库查看：使用简化的路径 'knowledge-view'
UPDATE sys_menu
SET path = 'knowledge-view',
    component = '/knowledge/view/index'
WHERE menu_name = '知识库查看'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 7. 楼宇管理：使用简化的路径 'buildings'
UPDATE sys_menu
SET path = 'buildings',
    component = '/operations/buildings/index'
WHERE menu_name = '楼宇管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 8. 楼层管理：使用简化的路径 'floors'
UPDATE sys_menu
SET path = 'floors',
    component = '/operations/floors/index'
WHERE menu_name = '楼层管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 9. 工位管理：使用简化的路径 'workstations'
UPDATE sys_menu
SET path = 'workstations',
    component = '/operations/workstations/index'
WHERE menu_name = '工位管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 10. 机房管理：使用简化的路径 'server-rooms'
UPDATE sys_menu
SET path = 'server-rooms',
    component = '/operations/server-rooms/index'
WHERE menu_name = '机房管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 11. 专线管理：使用简化的路径 'dedicated-lines'
UPDATE sys_menu
SET path = 'dedicated-lines',
    component = '/operations/dedicated-lines/index'
WHERE menu_name = '专线管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 12. 机房设备管理：使用简化的路径 'room-devices'
UPDATE sys_menu
SET path = 'room-devices',
    component = '/operations/room-devices/index'
WHERE menu_name = '机房设备管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 13. 楼宇空间：使用简化的路径 'building-spaces'
UPDATE sys_menu
SET path = 'building-spaces',
    component = '/operations/building-spaces/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间';

-- 14. 楼宇空间3D：使用简化的路径 'building-spaces-3d'
UPDATE sys_menu
SET path = 'building-spaces-3d',
    component = '/operations/building-spaces-3d/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间3D';

-- 15. 信息点管理：使用简化的路径 'info-points'
UPDATE sys_menu
SET path = 'info-points',
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
    '/ops/' || path as expected_route
FROM sys_menu
WHERE parent_id IN (
    SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
)
ORDER BY order_num;
