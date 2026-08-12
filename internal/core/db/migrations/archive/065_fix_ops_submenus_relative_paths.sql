-- ============================================
-- 修复运维管理子菜单：使用相对path，与ops拼接
-- ============================================
-- 目标路由：
--   - 工单管理: /ops/workorder/orders
--   - 楼宇管理: /ops/buildings
--   - 楼宇空间: /ops/building-spaces
--   - 知识库文章: /ops/knowledge/articles
--
-- 配置规则：
--   - path: 相对路径（不包含/），与父路径ops拼接
--   - component: 绝对路径（以/开头），指向实际组件文件
-- 日期：2026-01-26
-- ============================================

-- 1. 修复工单管理子菜单
-- path使用相对路径，最终路由为 /ops/workorder/orders
UPDATE sys_menu
SET path = 'workorder/orders',
    component = '/workorder/orders/index'
WHERE menu_name = '工单管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'workorder/categories',
    component = '/workorder/categories/index'
WHERE menu_name = '工单分类'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'workorder/statistics',
    component = '/workorder/statistics/index'
WHERE menu_name = '工单统计'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'workorder/periodic/templates',
    component = '/workorder/periodic/templates/index'
WHERE menu_name = '周期性工单'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

-- 2. 修复知识库子菜单
-- path使用相对路径，最终路由为 /ops/knowledge/articles
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

-- 3. 修复楼宇空间子菜单
-- path使用相对路径，最终路由为 /ops/building-spaces
UPDATE sys_menu
SET path = 'building-spaces',
    component = '/operations/building-spaces/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间';

UPDATE sys_menu
SET path = 'building-spaces-3d',
    component = '/operations/building-spaces-3d/index',
    parent_id = (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL)
WHERE menu_name = '楼宇空间3D';

-- 4. 修复其他运维子菜单
-- path使用相对路径，最终路由为 /ops/buildings, /ops/floors 等
UPDATE sys_menu
SET path = 'buildings',
    component = '/operations/buildings/index'
WHERE menu_name = '楼宇管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'floors',
    component = '/operations/floors/index'
WHERE menu_name = '楼层管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'workstations',
    component = '/operations/workstations/index'
WHERE menu_name = '工位管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'server-rooms',
    component = '/operations/server-rooms/index'
WHERE menu_name = '机房管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'dedicated-lines',
    component = '/operations/dedicated-lines/index'
WHERE menu_name = '专线管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

UPDATE sys_menu
SET path = 'room-devices',
    component = '/operations/room-devices/index'
WHERE menu_name = '机房设备管理'
  AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL);

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
