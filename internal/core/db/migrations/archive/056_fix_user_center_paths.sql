-- ============================================
-- 056_fix_user_center_paths.sql
-- 说明: 修复用户中心相关页面的路径配置
--       使个人中心、系统设置等页面可以直接访问
-- ============================================

DO $$
DECLARE
    v_user_center_id UUID;
    v_count INTEGER;
BEGIN
    -- 获取用户中心父菜单ID
    SELECT id INTO v_user_center_id
    FROM sys_menu
    WHERE menu_name = '用户中心' AND parent_id IS NULL
    LIMIT 1;

    IF v_user_center_id IS NULL THEN
        RAISE NOTICE '用户中心父菜单不存在，跳过修复';
        RETURN;
    END IF;

    -- ========================================
    -- 1. 修复个人中心路径
    --    从 'profile' 改为 'user/profile'
    --    这样可以直接通过 /user/profile 访问
    -- ========================================
    UPDATE sys_menu
    SET path = 'user/profile'
    WHERE menu_name = '个人中心'
      AND parent_id = v_user_center_id
      AND path = 'profile';

    RAISE NOTICE '已修复个人中心路径为 user/profile';

    -- ========================================
    -- 2. 修复系统设置路径
    --    从 'settings' 改为 'user/settings'
    --    这样可以直接通过 /user/settings 访问
    -- ========================================
    UPDATE sys_menu
    SET path = 'user/settings'
    WHERE menu_name = '系统设置'
      AND parent_id = v_user_center_id
      AND path = 'settings';

    RAISE NOTICE '已修复系统设置路径为 user/settings';

    -- ========================================
    -- 3. 修复"我的通知"路径
    --    从 'my-notices' 改为 'user/my-notices'
    -- ========================================
    UPDATE sys_menu
    SET path = 'user/my-notices'
    WHERE menu_name = '我的通知'
      AND parent_id = v_user_center_id
      AND path = 'my-notices';

    RAISE NOTICE '已修复我的通知路径为 user/my-notices';

    -- ========================================
    -- 4. 更新组件路径（如果需要）
    -- ========================================
    -- 个人中心组件路径应该是 'profile/index'
    UPDATE sys_menu
    SET component = 'profile/index'
    WHERE menu_name = '个人中心'
      AND component IS NULL;

    -- 系统设置组件路径应该是 'system/settings-page/index'
    UPDATE sys_menu
    SET component = 'system/settings-page/index'
    WHERE menu_name = '系统设置'
      AND parent_id = v_user_center_id
      AND component IS NULL;

    -- 我的通知组件路径
    UPDATE sys_menu
    SET component = 'user/my-notices/index'
    WHERE menu_name = '我的通知'
      AND parent_id = v_user_center_id
      AND component IS NULL;

    RAISE NOTICE '成功修复用户中心相关页面的路径配置';

END $$;

-- ============================================
-- 验证修复结果
-- ============================================
SELECT
    id,
    menu_name,
    path,
    component,
    parent_id,
    visible,
    menu_type
FROM sys_menu
WHERE parent_id = (
    SELECT id FROM sys_menu
    WHERE menu_name = '用户中心' AND parent_id IS NULL
    LIMIT 1
)
ORDER BY sort;
