-- 修复 API 密钥管理菜单的父子关系
-- 版本: 112
-- 问题: 子菜单的 parent_id 可能没有正确指向父菜单

-- 步骤1: 删除所有现有的API密钥相关菜单（确保清理）
DELETE FROM sys_role_menu
WHERE menu_id IN (
    SELECT id FROM sys_menu
    WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志')
);

DELETE FROM sys_menu
WHERE menu_name IN ('API密钥管理', '密钥列表', '使用日志');

-- 步骤2: 重新创建API密钥管理菜单（使用确定性方法）
-- 首先获取系统管理菜单ID
DO $$
DECLARE
    v_system_menu_id UUID;
    v_apikey_menu_id UUID;
    v_keylist_menu_id UUID;
    v_logs_menu_id UUID;
    v_admin_role_id UUID;
BEGIN
    -- 获取系统管理菜单ID（根级别菜单）
    SELECT id INTO v_system_menu_id
    FROM sys_menu
    WHERE menu_name = '系统管理'
      AND parent_id IS NULL
    LIMIT 1;

    IF v_system_menu_id IS NULL THEN
        RAISE EXCEPTION '系统管理菜单不存在';
    END IF;

    -- 插入API密钥管理菜单（二级目录）
    INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
    VALUES (
        gen_random_uuid(),
        'API密钥管理',
        v_system_menu_id,
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
    )
    RETURNING id INTO v_apikey_menu_id;

    -- 插入密钥列表子菜单
    INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
    VALUES (
        gen_random_uuid(),
        '密钥列表',
        v_apikey_menu_id,
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
    )
    RETURNING id INTO v_keylist_menu_id;

    -- 插入使用日志子菜单
    INSERT INTO sys_menu (id, menu_name, parent_id, order_num, path, component, menu_type, visible, status, perms, icon, remark, created_at, updated_at)
    VALUES (
        gen_random_uuid(),
        '使用日志',
        v_apikey_menu_id,
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
    )
    RETURNING id INTO v_logs_menu_id;

    -- 获取管理员角色ID
    SELECT id INTO v_admin_role_id
    FROM sys_role
    WHERE role_key = 'admin'
    LIMIT 1;

    IF v_admin_role_id IS NOT NULL THEN
        -- 为管理员角色分配菜单权限
        INSERT INTO sys_role_menu (role_id, menu_id) VALUES
            (v_admin_role_id, v_apikey_menu_id),
            (v_admin_role_id, v_keylist_menu_id),
            (v_admin_role_id, v_logs_menu_id)
        ON CONFLICT (role_id, menu_id) DO NOTHING;
    END IF;

    RAISE NOTICE 'API密钥管理菜单已成功创建，父菜单ID: %, 子菜单1: %, 子菜单2: %',
        v_apikey_menu_id, v_keylist_menu_id, v_logs_menu_id;
END $$;

-- 验证结果
SELECT
    m1.id as parent_id,
    m1.menu_name as parent_name,
    m1.menu_type as parent_type,
    m2.id as child_id,
    m2.menu_name as child_name,
    m2.menu_type as child_type,
    m2.parent_id as child_parent_id
FROM sys_menu m1
LEFT JOIN sys_menu m2 ON m2.parent_id = m1.id
WHERE m1.menu_name = 'API密钥管理'
ORDER BY m2.order_num;

SELECT 'API密钥管理菜单父子关系验证完成！' AS status;
