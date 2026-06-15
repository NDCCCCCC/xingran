-- ============================================
-- 008_add_system_settings_menu.sql
-- 说明: 添加系统设置菜单
-- ============================================

DO $$
DECLARE
    v_system_menu_id UUID;
    v_count INTEGER;
BEGIN
    -- 获取系统管理父菜单ID
    SELECT id INTO v_system_menu_id
    FROM sys_menu
    WHERE menu_name = '系统管理' AND parent_id IS NULL
    LIMIT 1;

    IF v_system_menu_id IS NULL THEN
        RAISE EXCEPTION '系统管理父菜单不存在';
    END IF;

    -- 创建"系统设置"菜单
    SELECT COUNT(*) INTO v_count
    FROM sys_menu
    WHERE menu_name = '系统设置' AND parent_id = v_system_menu_id;

    IF v_count = 0 THEN
        INSERT INTO sys_menu (
            id, menu_name, parent_id, order_num, path, component,
            menu_type, visible, status, perms, icon, remark,
            created_at, updated_at
        ) VALUES (
            gen_random_uuid(),
            '系统设置',
            v_system_menu_id,
            10,  -- 排序在工位管理之后
            'system/settings-page',
            'system/settings-page',
            'C',   -- C=菜单
            '1',   -- 1=显示
            '0',   -- 0=正常
            'system:config:list',
            'control',
            '通知渠道配置管理',
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP
        );

        RAISE NOTICE '已创建"系统设置"菜单';
    ELSE
        RAISE NOTICE '"系统设置"菜单已存在，跳过创建';
    END IF;

    -- 为超级管理员分配权限
    INSERT INTO sys_role_menu (role_id, menu_id)
    SELECT
        (SELECT id FROM sys_role WHERE role_key = 'admin' LIMIT 1),
        id
    FROM sys_menu
    WHERE menu_name = '系统设置' AND parent_id = v_system_menu_id
    ON CONFLICT (role_id, menu_id) DO NOTHING;

    RAISE NOTICE '已为超级管理员分配"系统设置"菜单权限';

END $$;

-- ============================================
-- 验证迁移结果
-- ============================================

-- 查看新创建的菜单
SELECT
    id,
    menu_name,
    parent_id,
    path,
    component,
    menu_type,
    visible,
    status,
    icon
FROM sys_menu
WHERE menu_name = '系统设置';

-- 验证角色菜单关联
SELECT
    rm.role_id,
    r.role_name,
    rm.menu_id,
    m.menu_name
FROM sys_role_menu rm
JOIN sys_role r ON rm.role_id = r.id
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = '系统设置';
