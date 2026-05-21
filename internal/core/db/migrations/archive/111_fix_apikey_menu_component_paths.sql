-- 修复API密钥管理菜单配置
-- 版本: 111
-- 修复组件路径格式（移除pages/前缀）

-- 1. 删除所有现有的API密钥相关菜单
DELETE FROM sys_menu
WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志');

-- 2. 重新创建API密钥管理二级目录菜单（目录类型不需要component）
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
WHERE sm.menu_name = '系统管理' AND sm.parent_id IS NULL;

-- 3. 创建密钥列表子菜单（组件路径：system/apikeys/index）
WITH apikey_menu AS (
    SELECT id FROM sys_menu WHERE menu_name = 'API密钥管理' ORDER BY created_at DESC LIMIT 1
)
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '密钥列表',
    apikey_menu.id,
    1,
    'system/apikeys',
    'system/apikeys/index',
    'C',
    '1',
    '0',
    'system:apikey:list',
    NULL,
    'API密钥列表',
    NOW(),
    NOW()
FROM apikey_menu;

-- 4. 创建使用日志子菜单（组件路径：system/apikeys/LogsModal/index）
WITH apikey_menu AS (
    SELECT id FROM sys_menu WHERE menu_name = 'API密钥管理' ORDER BY created_at DESC LIMIT 1
)
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '使用日志',
    apikey_menu.id,
    2,
    'system/apikeys/logs',
    'system/apikeys/LogsModal/index',
    'C',
    '1',
    '0',
    'system:apikey:logs',
    NULL,
    'API密钥使用日志',
    NOW(),
    NOW()
FROM apikey_menu;

-- 5. 为管理员角色分配权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.role_key = 'admin'
  AND m.menu_name IN ('API密钥管理', '密钥列表', '使用日志')
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 验证配置
SELECT
    menu_name,
    menu_type,
    path,
    component,
    perms
FROM sys_menu
WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志')
ORDER BY menu_type, order_num;

-- 完成
SELECT 'API密钥管理菜单配置已修复！组件路径格式：system/apikeys/index' AS status;
