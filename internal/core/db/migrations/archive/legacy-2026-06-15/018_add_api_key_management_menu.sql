-- API密钥管理菜单
-- 添加到系统管理菜单下（三级菜单结构）
-- 系统管理 > API密钥管理 > 密钥列表/使用日志

-- 添加API密钥管理菜单（二级菜单，目录类型）
INSERT INTO sys_menu (menu_id, menu_name, parent_id, order_num, path, component, query_param, is_frame, is_cache, menu_type, visible, status, perms, icon, created_by, updated_by, created_at, updated_at)
VALUES
(
    'api-key-management',
    'API密钥管理',
    '1',
    8,
    NULL,
    NULL,
    NULL,
    1,
    0,
    'M',
    '1',
    '0',
    NULL,
    'KeyOutlined',
    'system',
    'system',
    NOW(),
    NOW()
) ON CONFLICT (menu_id) DO UPDATE SET
    menu_name = EXCLUDED.menu_name,
    order_num = EXCLUDED.order_num,
    menu_type = 'M',
    updated_at = NOW();

-- 添加密钥列表子菜单（三级菜单）
INSERT INTO sys_menu (menu_id, menu_name, parent_id, order_num, path, component, query_param, is_frame, is_cache, menu_type, visible, status, perms, icon, created_by, updated_by, created_at, updated_at)
VALUES
(
    'api-key-list',
    '密钥列表',
    'api-key-management',
    1,
    'system/apikeys',
    'pages/system/apikeys',
    NULL,
    1,
    0,
    'C',
    '1',
    '0',
    'system:apikey:list',
    NULL,
    'system',
    'system',
    NOW(),
    NOW()
) ON CONFLICT (menu_id) DO UPDATE SET
    menu_name = EXCLUDED.menu_name,
    parent_id = EXCLUDED.parent_id,
    order_num = EXCLUDED.order_num,
    path = EXCLUDED.path,
    component = EXCLUDED.component,
    menu_type = 'C',
    updated_at = NOW();

-- 添加使用日志子菜单（三级菜单）
INSERT INTO sys_menu (menu_id, menu_name, parent_id, order_num, path, component, query_param, is_frame, is_cache, menu_type, visible, status, perms, icon, created_by, updated_by, created_at, updated_at)
VALUES
(
    'api-key-logs',
    '使用日志',
    'api-key-management',
    2,
    'system/apikeys/logs',
    'pages/system/apikeys/LogsModal',
    NULL,
    1,
    0,
    'C',
    '1',
    '0',
    'system:apikey:logs',
    NULL,
    'system',
    'system',
    NOW(),
    NOW()
) ON CONFLICT (menu_id) DO UPDATE SET
    menu_name = EXCLUDED.menu_name,
    parent_id = EXCLUDED.parent_id,
    order_num = EXCLUDED.order_num,
    path = EXCLUDED.path,
    component = EXCLUDED.component,
    menu_type = 'C',
    updated_at = NOW();
