-- ============================================
-- 修复运维管理子菜单路径
-- ============================================
-- 问题：运维管理子菜单的路径被设置成了 operations/xxx 形式
--       这些路径包含 "/" 被当作绝对路径，无法与父路径 ops 正确拼接
-- 解决：将路径改为相对路径，使最终路由为 ops/xxx
-- 日期：2026-01-26
-- ============================================

-- 首先获取运维管理父菜单的ID
DO $$
DECLARE
    ops_parent_id UUID;
BEGIN
    SELECT id INTO ops_parent_id
    FROM sys_menu
    WHERE menu_name = '运维管理' AND parent_id IS NULL;

    IF ops_parent_id IS NULL THEN
        RAISE EXCEPTION '找不到运维管理父菜单';
    END IF;

    RAISE NOTICE '运维管理父菜单ID: %', ops_parent_id;

    -- 1. 修复楼宇管理路径和组件
    UPDATE sys_menu
    SET path = 'buildings',
        component = 'operations/buildings/index'
    WHERE menu_name = '楼宇管理'
      AND parent_id = ops_parent_id;

    -- 2. 修复楼层管理路径和组件
    UPDATE sys_menu
    SET path = 'floors',
        component = 'operations/floors/index'
    WHERE menu_name = '楼层管理'
      AND parent_id = ops_parent_id;

    -- 3. 修复工位管理路径和组件
    UPDATE sys_menu
    SET path = 'workstations',
        component = 'operations/workstations/index'
    WHERE menu_name = '工位管理'
      AND parent_id = ops_parent_id;

    -- 4. 修复机房管理路径和组件
    UPDATE sys_menu
    SET path = 'server-rooms',
        component = 'operations/server-rooms/index'
    WHERE menu_name = '机房管理'
      AND parent_id = ops_parent_id;

    -- 5. 修复专线管理路径和组件
    UPDATE sys_menu
    SET path = 'dedicated-lines',
        component = 'operations/dedicated-lines/index'
    WHERE menu_name = '专线管理'
      AND parent_id = ops_parent_id;

    -- 6. 修复机房设备管理路径和组件
    UPDATE sys_menu
    SET path = 'room-devices',
        component = 'operations/room-devices/index'
    WHERE menu_name = '机房设备管理'
      AND parent_id = ops_parent_id;

    -- 7. 修复楼宇空间路径、组件和parent_id
    UPDATE sys_menu
    SET path = 'building-spaces',
        component = 'operations/building-spaces/index',
        parent_id = ops_parent_id
    WHERE menu_name = '楼宇空间';

    -- 8. 修复楼宇空间3D路径、组件和parent_id
    UPDATE sys_menu
    SET path = 'building-spaces-3d',
        component = 'operations/building-spaces-3d/index',
        parent_id = ops_parent_id
    WHERE menu_name = '楼宇空间3D';

    -- 9. 修复信息点管理路径和组件（如果菜单不存在则创建）
    -- 先尝试更新
    UPDATE sys_menu
    SET path = 'info-points',
        component = 'operations/info-points/index'
    WHERE menu_name = '信息点管理'
      AND parent_id = ops_parent_id;

    -- 10. 修复工单管理路径和组件（工单模块使用绝对路径，路由为 /workorder/orders）
    UPDATE sys_menu
    SET path = 'workorder/orders',
        component = 'workorder/orders/index'
    WHERE menu_name = '工单管理'
      AND parent_id = ops_parent_id;

    -- 11. 修复工单分类路径和组件
    UPDATE sys_menu
    SET path = 'workorder/categories',
        component = 'workorder/categories/index'
    WHERE menu_name = '工单分类'
      AND parent_id = ops_parent_id;

    -- 12. 修复工单统计路径和组件
    UPDATE sys_menu
    SET path = 'workorder/statistics',
        component = 'workorder/statistics/index'
    WHERE menu_name = '工单统计'
      AND parent_id = ops_parent_id;

    -- 13. 修复周期性工单路径和组件
    UPDATE sys_menu
    SET path = 'workorder/periodic/templates',
        component = 'workorder/periodic/templates/index'
    WHERE menu_name = '周期性工单'
      AND parent_id = ops_parent_id;

    -- 14. 修复知识库文章路径和组件（知识库模块使用绝对路径，路由为 /knowledge/articles）
    UPDATE sys_menu
    SET path = 'knowledge/articles',
        component = 'knowledge/articles/index'
    WHERE menu_name = '知识库文章'
      AND parent_id = ops_parent_id;

    -- 15. 修复知识库查看路径和组件
    UPDATE sys_menu
    SET path = 'knowledge/view',
        component = 'knowledge/view/index'
    WHERE menu_name = '知识库查看'
      AND parent_id = ops_parent_id;

    RAISE NOTICE '运维管理子菜单路径修复完成';
END $$;

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
