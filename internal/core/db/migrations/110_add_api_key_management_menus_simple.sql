-- API密钥管理菜单配置 - 分步执行版本
-- 版本: 110
-- 添加系统管理下的API密钥管理二级菜单及其子菜单
-- 此文件不使用DO块，可以分步执行

-- 步骤1: 插入API密钥管理二级目录菜单
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

-- 步骤2: 插入密钥列表子菜单（使用CTE获取父菜单ID）
WITH apikey_menu AS (
    SELECT m.id
    FROM sys_menu m
    INNER JOIN sys_menu parent ON parent.id = m.parent_id
    WHERE m.menu_name = 'API密钥管理'
      AND parent.menu_name = '系统管理'
      AND parent.parent_id IS NULL
    LIMIT 1
)
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '密钥列表',
    apikey_menu.id,
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
FROM apikey_menu
ON CONFLICT (id) DO NOTHING;

-- 步骤3: 插入使用日志子菜单
WITH apikey_menu AS (
    SELECT m.id
    FROM sys_menu m
    INNER JOIN sys_menu parent ON parent.id = m.parent_id
    WHERE m.menu_name = 'API密钥管理'
      AND parent.menu_name = '系统管理'
      AND parent.parent_id IS NULL
    LIMIT 1
)
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '使用日志',
    apikey_menu.id,
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
FROM apikey_menu
ON CONFLICT (id) DO NOTHING;

-- 步骤4: 为管理员角色分配API密钥管理权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, m.id
FROM sys_role r
CROSS JOIN sys_menu m
INNER JOIN sys_menu parent ON parent.id = m.parent_id
WHERE r.role_key = 'admin'
  AND m.menu_name IN ('API密钥管理', '密钥列表', '使用日志')
  AND parent.menu_name = '系统管理'
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 步骤5: 记录此迁移
INSERT INTO schema_migrations (version, description, applied_at)
VALUES ('110', 'add_api_key_management_menus', NOW())
ON CONFLICT (version) DO NOTHING;

-- 完成
SELECT 'API密钥管理菜单配置完成！' AS status;
