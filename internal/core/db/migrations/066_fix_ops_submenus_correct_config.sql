-- ============================================
-- 修复运维管理子菜单配置（配合前端resolvePath修复）
-- ============================================
-- 前端resolvePath已修复：只有以'/'开头的path才是绝对路径
-- 包含'/'但不以'/'开头的path（如'workorder/orders'）会与父路径拼接
--
-- 目标路由：
--   - 工单管理: /ops/workorder/orders
--   - 楼宇管理: /ops/operations/buildings
--   - 知识库文章: /ops/knowledge/articles
--   - 楼宇空间: /ops/operations/building-spaces
--
-- 配置规则：
--   - path: 相对路径（可包含/但不能以/开头），与ops拼接
--   - component: 绝对路径（以/开头），指向实际组件文件
-- 日期：2026-01-26
-- ============================================

-- 1. 修复工单管理子菜单（已经正确，只需确认component路径）
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

-- 3. 修复楼宇空间子菜单（修复parent_id）
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

-- 4. 修复其他运维子菜单
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
    '/ops/' || path as expected_route,
    CASE
        WHEN path LIKE '/%' THEN '❌ 绝对路径（错误）'
        ELSE '✅ 相对路径（正确）'
    END as path_status
FROM sys_menu
WHERE parent_id IN (
    SELECT id FROM sys_menu WHERE menu_name = '运维管理' AND parent_id IS NULL
)
ORDER BY order_num;
