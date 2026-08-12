-- ============================================
-- 042_add_hidden_special_page_menus.sql
-- 说明: 添加用户中心父菜单及特殊页面隐藏菜单
--       用于 profile、settings、my-notices 等页面的标题显示
--       这些菜单不在导航栏显示（visible=0），但用于标签页标题
-- ============================================

DO $$
DECLARE
    v_user_center_id UUID;
    v_menu_id UUID;
    v_admin_role_id UUID;
    v_count INTEGER;
BEGIN
    -- 获取管理员角色ID
    SELECT id INTO v_admin_role_id
    FROM sys_role
    WHERE role_key = 'admin'
    LIMIT 1;

    IF v_admin_role_id IS NULL THEN
        RAISE EXCEPTION '管理员角色不存在';
    END IF;

    -- ========================================
    -- 0. 创建"用户中心"父菜单（隐藏）
    -- ========================================
    SELECT COUNT(*) INTO v_count
    FROM sys_menu
    WHERE menu_name = '用户中心' AND parent_id IS NULL;

    IF v_count = 0 THEN
        INSERT INTO sys_menu (
            id,
            menu_name,
            parent_id,
            order_num,
            path,
            component,
            menu_type,
            visible,
            status,
            perms,
            icon,
            remark,
            created_at,
            updated_at
        ) VALUES (
            gen_random_uuid(),
            '用户中心',
            NULL,
            100,
            'user-center',
            NULL,
            'M',   -- M=目录
            0,    -- visible=0 (隐藏)
            0,    -- status=0 (正常)
            NULL,
            'user',
            '用户中心模块（隐藏目录，包含个人中心、系统设置、我的通知等页面）',
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO v_user_center_id;

        -- 分配权限给管理员
        INSERT INTO sys_role_menu (role_id, menu_id)
        VALUES (v_admin_role_id, v_user_center_id)
        ON CONFLICT (role_id, menu_id) DO NOTHING;

        RAISE NOTICE '已创建"用户中心"隐藏父菜单';
    ELSE
        SELECT id INTO v_user_center_id
        FROM sys_menu
        WHERE menu_name = '用户中心' AND parent_id IS NULL
        LIMIT 1;
        RAISE NOTICE '"用户中心"父菜单已存在';
    END IF;

    -- ========================================
    -- 1. 个人中心菜单（隐藏）
    -- ========================================
    SELECT COUNT(*) INTO v_count
    FROM sys_menu
    WHERE path = 'profile' AND parent_id = v_user_center_id;

    IF v_count = 0 THEN
        INSERT INTO sys_menu (
            id,
            menu_name,
            parent_id,
            order_num,
            path,
            component,
            menu_type,
            visible,
            status,
            perms,
            icon,
            remark,
            created_at,
            updated_at
        ) VALUES (
            gen_random_uuid(),
            '个人中心',
            v_user_center_id,
            1,
            'profile',
            NULL,
            'C',   -- C=菜单
            0,    -- visible=0 (隐藏)
            0,    -- status=0 (正常)
            'system:user:profile',
            'user',
            '个人中心页面（隐藏菜单，用于标签标题）',
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO v_menu_id;

        -- 分配权限给管理员
        INSERT INTO sys_role_menu (role_id, menu_id)
        VALUES (v_admin_role_id, v_menu_id)
        ON CONFLICT (role_id, menu_id) DO NOTHING;

        RAISE NOTICE '已创建"个人中心"隐藏菜单';
    ELSE
        RAISE NOTICE '"个人中心"菜单已存在，跳过创建';
    END IF;

    -- ========================================
    -- 2. 系统设置菜单（隐藏）
    -- ========================================
    SELECT COUNT(*) INTO v_count
    FROM sys_menu
    WHERE path = 'settings' AND parent_id = v_user_center_id;

    IF v_count = 0 THEN
        INSERT INTO sys_menu (
            id,
            menu_name,
            parent_id,
            order_num,
            path,
            component,
            menu_type,
            visible,
            status,
            perms,
            icon,
            remark,
            created_at,
            updated_at
        ) VALUES (
            gen_random_uuid(),
            '系统设置',
            v_user_center_id,
            2,
            'settings',
            NULL,
            'C',   -- C=菜单
            0,    -- visible=0 (隐藏)
            0,    -- status=0 (正常)
            'system:user:settings',
            'setting',
            '系统设置页面（隐藏菜单，用于标签标题）',
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO v_menu_id;

        -- 分配权限给管理员
        INSERT INTO sys_role_menu (role_id, menu_id)
        VALUES (v_admin_role_id, v_menu_id)
        ON CONFLICT (role_id, menu_id) DO NOTHING;

        RAISE NOTICE '已创建"系统设置"隐藏菜单';
    ELSE
        RAISE NOTICE '"系统设置"菜单已存在，跳过创建';
    END IF;

    -- ========================================
    -- 3. 我的通知菜单（隐藏）
    -- ========================================
    SELECT COUNT(*) INTO v_count
    FROM sys_menu
    WHERE path = 'my-notices' AND parent_id = v_user_center_id;

    IF v_count = 0 THEN
        INSERT INTO sys_menu (
            id,
            menu_name,
            parent_id,
            order_num,
            path,
            component,
            menu_type,
            visible,
            status,
            perms,
            icon,
            remark,
            created_at,
            updated_at
        ) VALUES (
            gen_random_uuid(),
            '我的通知',
            v_user_center_id,
            3,
            'my-notices',
            NULL,
            'C',   -- C=菜单
            0,    -- visible=0 (隐藏)
            0,    -- status=0 (正常)
            'system:notice:my',
            'bell',
            '我的通知页面（隐藏菜单，用于标签标题）',
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        ) RETURNING id INTO v_menu_id;

        -- 分配权限给管理员
        INSERT INTO sys_role_menu (role_id, menu_id)
        VALUES (v_admin_role_id, v_menu_id)
        ON CONFLICT (role_id, menu_id) DO NOTHING;

        RAISE NOTICE '已创建"我的通知"隐藏菜单';
    ELSE
        RAISE NOTICE '"我的通知"菜单已存在，跳过创建';
    END IF;

    RAISE NOTICE '成功添加用户中心父菜单及 3 个特殊页面隐藏菜单';

END $$;

-- ============================================
-- 验证迁移结果
-- ============================================

-- 查看新创建的菜单结构
WITH RECURSIVE menu_tree AS (
    SELECT
        id,
        menu_name,
        parent_id,
        path,
        menu_type,
        visible,
        status,
        icon,
        remark,
        order_num,
        1 AS level
    FROM sys_menu
    WHERE id IN (
        SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL
    )

    UNION ALL

    SELECT
        m.id,
        m.menu_name,
        m.parent_id,
        m.path,
        m.menu_type,
        m.visible,
        m.status,
        m.icon,
        m.remark,
        m.order_num,
        mt.level + 1
    FROM sys_menu m
    INNER JOIN menu_tree mt ON m.parent_id = mt.id
)
SELECT
    id,
    repeat('  ', level - 1) || menu_name AS menu_tree,
    parent_id,
    path,
    menu_type,
    visible,
    status,
    icon
FROM menu_tree
ORDER BY order_num;

-- 验证角色菜单关联
SELECT
    rm.role_id,
    r.role_name,
    rm.menu_id,
    m.menu_name,
    m.path,
    m.visible
FROM sys_role_menu rm
JOIN sys_role r ON rm.role_id = r.id
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.parent_id IN (
    SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL
)
   OR m.id IN (
    SELECT id FROM sys_menu WHERE menu_name = '用户中心' AND parent_id IS NULL
)
ORDER BY m.order_num;
