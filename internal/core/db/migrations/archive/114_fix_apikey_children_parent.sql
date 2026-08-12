-- 修复 API 密钥管理子菜单关联问题
-- 紧急修复：确保子菜单正确关联到父菜单

-- 首先检查当前状态
SELECT
    m1.menu_name as "父菜单",
    m1.id as "父ID",
    m1.menu_type as "父类型",
    m2.menu_name as "子菜单",
    m2.id as "子ID",
    m2.parent_id as "子菜单parent_id"
FROM sys_menu m1
LEFT JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.menu_name IN ('API密钥管理', '用户中心')
ORDER BY m1.menu_name, m2.order_num;

-- 删除错误的子菜单（如果有的话）
DELETE FROM sys_menu
WHERE menu_name IN ('密钥列表', '使用日志')
  AND parent_id NOT IN (
      SELECT id FROM sys_menu WHERE menu_name = 'API密钥管理'
  );

-- 重新创建子菜单，确保正确的 parent_id
WITH apikey_dir AS (
    SELECT id FROM sys_menu
    WHERE menu_name = 'API密钥管理'
      AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '系统管理' AND parent_id IS NULL)
    LIMIT 1
)
-- 创建密钥列表
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '密钥列表',
    apikey_dir.id,
    1,
    'system/apikeys',
    'system/apikeys/index',
    'C',
    1,
    0,
    'system:apikey:list',
    NULL,
    'API密钥列表',
    NOW(),
    NOW()
FROM apikey_dir
WHERE NOT EXISTS (
    SELECT 1 FROM sys_menu
    WHERE menu_name = '密钥列表'
      AND parent_id = apikey_dir.id
);

-- 创建使用日志
WITH apikey_dir AS (
    SELECT id FROM sys_menu
    WHERE menu_name = 'API密钥管理'
      AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '系统管理' AND parent_id IS NULL)
    LIMIT 1
)
INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
SELECT
    gen_random_uuid(),
    '使用日志',
    apikey_dir.id,
    2,
    'system/apikeys/logs',
    'system/apikeys/LogsModal/index',
    'C',
    1,
    0,
    'system:apikey:logs',
    NULL,
    'API密钥使用日志',
    NOW(),
    NOW()
FROM apikey_dir
WHERE NOT EXISTS (
    SELECT 1 FROM sys_menu
    WHERE menu_name = '使用日志'
      AND parent_id = apikey_dir.id
);

-- 验证结果
SELECT
    m1.menu_name as "父菜单",
    m2.menu_name as "子菜单",
    m2.menu_type as "子类型",
    CASE WHEN m2.id IS NOT NULL THEN '✓ 已关联' ELSE '✗ 未关联' END as "状态"
FROM sys_menu m1
LEFT JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.menu_name = 'API密钥管理';

SELECT 'API密钥管理子菜单关联修复完成！' AS status;
