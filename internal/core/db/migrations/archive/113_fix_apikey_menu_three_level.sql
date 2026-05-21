-- 修复 API 密钥管理菜单 - 完全仿照工单系统的三级菜单结构
-- 版本: 113
-- 模式: 系统管理(M) > API密钥管理(M) > 密钥列表(C)/使用日志(C)

-- 步骤1: 清理现有的API密钥相关菜单
DELETE FROM sys_role_menu
WHERE menu_id IN (
    SELECT id FROM sys_menu
    WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志')
);

DELETE FROM sys_menu
WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志');

-- 步骤2: 创建"API密钥管理"目录菜单（二级，仿照"工单系统"）
-- 关键：menu_type='M' 使其成为可展开的目录
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
    icon,
    remark,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    'API密钥管理',
    id,
    11,
    'apikeys',
    NULL,
    'M',
    1,
    0,
    'KeyOutlined',
    'API密钥管理目录',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM sys_menu
WHERE menu_name = '系统管理' AND parent_id IS NULL
LIMIT 1;

-- 步骤3: 创建"密钥列表"子菜单（三级，仿照"工单管理"）
WITH apikey_dir AS (
    SELECT id FROM sys_menu
    WHERE menu_name = 'API密钥管理'
      AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '系统管理' AND parent_id IS NULL)
    LIMIT 1
)
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
)
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
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM apikey_dir;

-- 步骤4: 创建"使用日志"子菜单（三级，仿照"工单分类"）
WITH apikey_dir AS (
    SELECT id FROM sys_menu
    WHERE menu_name = 'API密钥管理'
      AND parent_id IN (SELECT id FROM sys_menu WHERE menu_name = '系统管理' AND parent_id IS NULL)
    LIMIT 1
)
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
)
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
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM apikey_dir;

-- 步骤5: 为管理员角色分配权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT r.id, m.id
FROM sys_role r
CROSS JOIN sys_menu m
WHERE r.role_key = 'admin'
  AND m.menu_name IN ('API密钥管理', '密钥列表', '使用日志')
ON CONFLICT (role_id, menu_id) DO NOTHING;

-- 验证结果：查看API密钥管理菜单的完整结构
SELECT
    m1.menu_name as "一级菜单",
    m1.menu_type as "一级类型",
    m2.menu_name as "二级菜单",
    m2.menu_type as "二级类型",
    m3.menu_name as "三级菜单",
    m3.menu_type as "三级类型",
    CASE
        WHEN m3.id IS NOT NULL THEN m3.id
        WHEN m2.id IS NOT NULL THEN m2.id
        ELSE m1.id
    END as "菜单ID",
    CASE
        WHEN m3.id IS NOT NULL THEN m3.parent_id
        WHEN m2.id IS NOT NULL THEN m2.parent_id
        ELSE m1.parent_id
    END as "父菜单ID"
FROM sys_menu m1
LEFT JOIN sys_menu m2 ON m2.parent_id = m1.id
LEFT JOIN sys_menu m3 ON m3.parent_id = m2.id
WHERE m1.menu_name = '系统管理'
  AND (m2.menu_name = 'API密钥管理' OR m2.menu_name = '工单系统')
ORDER BY m2.order_num, m3.order_num;

SELECT 'API密钥管理菜单修复完成！三级菜单结构：系统管理 > API密钥管理 > 密钥列表/使用日志' AS status;
