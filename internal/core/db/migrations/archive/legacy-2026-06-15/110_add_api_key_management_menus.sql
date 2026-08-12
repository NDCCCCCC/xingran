-- API密钥管理菜单配置
-- 版本: 110
-- 添加系统管理下的API密钥管理二级菜单及其子菜单

-- 插入API密钥管理二级目录菜单
-- 使用子查询获取系统管理菜单的UUID
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    'API密钥管理',
    sm.id,
    11,
    NULL,
    NULL,
    'M',
    '1',
    '0',
    NULL,
    'KeyOutlined',
    'API密钥管理目录',
    NOW(),
    NOW()
FROM sys_menu sm
WHERE sm.menu_name = '系统管理' AND sm.parent_id IS NULL
ON CONFLICT (id) DO NOTHING;

-- 获取刚插入的API密钥管理菜单ID（作为父菜单ID）
DO $$
DECLARE
    v_apikey_menu_id UUID;
    v_system_menu_id UUID;
BEGIN
    -- 获取系统管理菜单ID
    SELECT id INTO v_system_menu_id FROM sys_menu WHERE menu_name = '系统管理' AND parent_id IS NULL LIMIT 1;

    -- 获取API密钥管理菜单ID
    SELECT id INTO v_apikey_menu_id FROM sys_menu WHERE menu_name = 'API密钥管理' AND parent_id = v_system_menu_id LIMIT 1;

    IF v_apikey_menu_id IS NOT NULL THEN
        -- 插入密钥列表子菜单
        INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
        VALUES (
            gen_random_uuid(),
            '密钥列表',
            v_apikey_menu_id,
            1,
            'system/apikeys',
            'pages/system/apikeys',
            'C',
            '1',
            '0',
            'system:apikey:list',
            NULL,
            'API密钥列表',
            NOW(),
            NOW()
        ) ON CONFLICT (id) DO NOTHING;

        -- 插入使用日志子菜单
        INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
        VALUES (
            gen_random_uuid(),
            '使用日志',
            v_apikey_menu_id,
            2,
            'system/apikeys/logs',
            'pages/system/apikeys/LogsModal',
            'C',
            '1',
            '0',
            'system:apikey:logs',
            NULL,
            'API密钥使用日志',
            NOW(),
            NOW()
        ) ON CONFLICT (id) DO NOTHING;

        -- 为管理员角色分配API密钥管理权限
        INSERT INTO sys_role_menu (role_id, menu_id)
        SELECT r.id, m.id
        FROM sys_role r
        CROSS JOIN sys_menu m
        WHERE r.role_key = 'admin'
          AND m.menu_name IN ('API密钥管理', '密钥列表', '使用日志')
        ON CONFLICT (role_id, menu_id) DO NOTHING;
    END IF;
END $$;

-- 记录此迁移
INSERT INTO schema_migrations (version, description, applied_at)
VALUES ('110', 'add_api_key_management_menus', NOW())
ON CONFLICT (version) DO NOTHING;

SELECT '110_add_api_key_management_menus.sql migration completed' AS status;
