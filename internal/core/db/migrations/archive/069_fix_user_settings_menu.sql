-- ============================================
-- 069_fix_user_settings_menu.sql
-- 说明: 修复用户中心的"用户设置"菜单
--       1. 将用户中心下的"系统设置"改名为"用户设置"
--       2. 确保其指向正确的组件路径 settings/index
--       3. 确保系统管理下的"系统设置"不受影响
-- ============================================

DO $$
DECLARE
    v_user_center_id UUID;
    v_system_management_id UUID;
    v_user_settings_menu_id UUID;
    v_system_settings_menu_id UUID;
    v_count INTEGER;
BEGIN
    -- 获取用户中心父菜单ID
    SELECT id INTO v_user_center_id
    FROM sys_menu
    WHERE menu_name = '用户中心' AND parent_id IS NULL
    LIMIT 1;

    IF v_user_center_id IS NULL THEN
        RAISE EXCEPTION '用户中心父菜单不存在';
    END IF;

    -- 获取系统管理父菜单ID
    SELECT id INTO v_system_management_id
    FROM sys_menu
    WHERE menu_name = '系统管理' AND parent_id IS NULL
    LIMIT 1;

    -- ========================================
    -- 1. 修复用户中心下的"系统设置"菜单
    -- ========================================

    -- 检查用户中心下是否存在"系统设置"菜单
    SELECT COUNT(*) INTO v_count
    FROM sys_menu
    WHERE menu_name = '系统设置' AND parent_id = v_user_center_id;

    IF v_count > 0 THEN
        -- 获取菜单ID
        SELECT id INTO v_user_settings_menu_id
        FROM sys_menu
        WHERE menu_name = '系统设置' AND parent_id = v_user_center_id
        LIMIT 1;

        -- 更新菜单：改名为"用户设置"，修复组件路径
        UPDATE sys_menu
        SET
            menu_name = '用户设置',
            component = 'settings/index',
            remark = '用户设置页面（个人偏好设置：主题、布局等）',
            updated_at = CURRENT_TIMESTAMP
        WHERE id = v_user_settings_menu_id;

        RAISE NOTICE '已将用户中心下的"系统设置"改名为"用户设置"，并修复组件路径';
    ELSE
        -- 检查是否已存在"用户设置"菜单
        SELECT COUNT(*) INTO v_count
        FROM sys_menu
        WHERE menu_name = '用户设置' AND parent_id = v_user_center_id;

        IF v_count = 0 THEN
            RAISE EXCEPTION '用户中心下既没有"系统设置"也没有"用户设置"菜单';
        END IF;
    END IF;

    -- ========================================
    -- 2. 确保系统管理下的"系统设置"菜单正确
    -- ========================================

    IF v_system_management_id IS NOT NULL THEN
        -- 检查系统管理下是否存在"系统设置"菜单
        SELECT COUNT(*) INTO v_count
        FROM sys_menu
        WHERE menu_name = '系统设置' AND parent_id = v_system_management_id;

        IF v_count > 0 THEN
            -- 获取菜单ID
            SELECT id INTO v_system_settings_menu_id
            FROM sys_menu
            WHERE menu_name = '系统设置' AND parent_id = v_system_management_id
            LIMIT 1;

            -- 确保组件路径正确（系统管理下的系统设置）
            UPDATE sys_menu
            SET
                component = 'system/settings-page/index',
                remark = '系统设置页面（管理员配置：邮箱、API、验证码等）',
                updated_at = CURRENT_TIMESTAMP
            WHERE id = v_system_settings_menu_id;

            RAISE NOTICE '已确认系统管理下的"系统设置"菜单组件路径正确';
        END IF;
    END IF;

    RAISE NOTICE '成功修复用户设置菜单';

END $$;

-- ============================================
-- 验证迁移结果
-- ============================================

-- 查看用户中心和系统管理的菜单结构
WITH RECURSIVE menu_tree AS (
    SELECT
        id,
        menu_name,
        parent_id,
        path,
        component,
        menu_type,
        visible,
        status,
        order_num,
        1 AS level
    FROM sys_menu
    WHERE menu_name IN ('用户中心', '系统管理') AND parent_id IS NULL

    UNION ALL

    SELECT
        m.id,
        m.menu_name,
        m.parent_id,
        m.path,
        m.component,
        m.menu_type,
        m.visible,
        m.status,
        m.order_num,
        mt.level + 1
    FROM sys_menu m
    INNER JOIN menu_tree mt ON m.parent_id = mt.id
)
SELECT
    id,
    repeat('  ', level - 1) || menu_name AS menu_tree,
    path,
    component,
    visible,
    status
FROM menu_tree
ORDER BY
    CASE WHEN menu_name = '用户中心' THEN 1 ELSE 2 END,
    level,
    order_num;
